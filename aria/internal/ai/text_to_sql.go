package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

func buildSQLSystemPrompt(req QueryRequest, schemaBlock, examplesBlock, historyBlock string) string {
	adminNote := ""
	if req.AgentRole == "admin" {
		adminNote = "This user is an admin — you may query across all agents when appropriate (omit assigned_agent_id filter when listing org-wide metrics).\n"
	} else {
		adminNote = "This user is an agent — every query that touches lead/task/booking style data MUST filter with assigned_agent_id = :agent_id (placeholder) or equivalent.\n"
	}
	return fmt.Sprintf(`You are Aria, an AI assistant for CRM agents at a student accommodation company.

AGENT CONTEXT:
- Agent ID: %s
- Agent name: %s
- Role: %s
- Timezone: %s

%s
DATA ACCESS RULES:
- Only generate read-only PostgreSQL SELECT queries.
- Use :agent_id as a placeholder for the current agent UUID where a filter is required (the server replaces it with a parameter).
- Never fabricate data. If a question cannot be answered from CRM tables, explain briefly.

%s

%s

CONVERSATION HISTORY:
%s

When the user asks a CRM question, you MUST call the tool query_crm_database with sql and explanation.
If the question is not about CRM data (e.g. weather, math trivia), respond with a short helpful explanation in plain text and do NOT call the tool.`,
		req.AgentID, req.AgentName, req.AgentRole, req.Timezone,
		adminNote,
		schemaBlock,
		examplesBlock,
		historyBlock,
	)
}

func toolParams() []openai.ChatCompletionToolParam {
	return []openai.ChatCompletionToolParam{
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "query_crm_database",
				Description: openai.String("Execute a read-only SQL SELECT against the CRM database"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"sql": map[string]any{
							"type":        "string",
							"description": "PostgreSQL SELECT only",
						},
						"explanation": map[string]any{
							"type":        "string",
							"description": "Plain English explanation of the query",
						},
					},
					"required": []string{"sql", "explanation"},
				},
			},
		},
	}
}

// GenerateSQL asks the model to emit SQL via tool calling.
func GenerateSQL(ctx context.Context, client openai.Client, cfg *config.Config, schemaText, examplesText, historyText string, req QueryRequest) (sql string, explanation string, promptTok, completionTok int, err error) {
	system := buildSQLSystemPrompt(req, schemaText, examplesText, historyText)
	user := "User question:\n" + req.Question

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(cfg.OpenAIChatModel),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
		Tools:       toolParams(),
		Temperature: param.NewOpt(0.1),
	}

	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", "", 0, 0, err
	}
	if len(completion.Choices) == 0 {
		return "", "", 0, 0, fmt.Errorf("no completion choices")
	}
	promptTok = int(completion.Usage.PromptTokens)
	completionTok = int(completion.Usage.CompletionTokens)

	msg := completion.Choices[0].Message
	if len(msg.ToolCalls) == 0 {
		content := strings.TrimSpace(msg.Content)
		return "", content, promptTok, completionTok, ErrOutOfScope
	}

	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "query_crm_database" {
			continue
		}
		var args struct {
			SQL         string `json:"sql"`
			Explanation string `json:"explanation"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", "", promptTok, completionTok, fmt.Errorf("tool args: %w", err)
		}
		return strings.TrimSpace(args.SQL), strings.TrimSpace(args.Explanation), promptTok, completionTok, nil
	}
	return "", strings.TrimSpace(msg.Content), promptTok, completionTok, fmt.Errorf("missing query_crm_database tool call")
}

// GenerateSQLWithRetry adds one correction round after a failed execution.
func GenerateSQLWithRetry(ctx context.Context, client openai.Client, cfg *config.Config, schemaText, examplesText, historyText string, req QueryRequest, failedSQL, dbError string) (sql string, explanation string, promptTok, completionTok int, err error) {
	system := buildSQLSystemPrompt(req, schemaText, examplesText, historyText)
	fix := fmt.Sprintf("Your previous SQL failed with this database error:\n%s\n\nFailed SQL:\n%s\n\nFix the SQL and call query_crm_database again.", dbError, failedSQL)

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(cfg.OpenAIChatModel),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage("User question:\n"+req.Question),
			openai.UserMessage(fix),
		},
		Tools:       toolParams(),
		Temperature: param.NewOpt(0.1),
	}

	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", "", 0, 0, err
	}
	if len(completion.Choices) == 0 {
		return "", "", 0, 0, fmt.Errorf("no completion choices")
	}
	promptTok = int(completion.Usage.PromptTokens)
	completionTok = int(completion.Usage.CompletionTokens)
	msg := completion.Choices[0].Message
	if len(msg.ToolCalls) == 0 {
		return "", strings.TrimSpace(msg.Content), promptTok, completionTok, fmt.Errorf("no tool call on retry")
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "query_crm_database" {
			continue
		}
		var args struct {
			SQL         string `json:"sql"`
			Explanation string `json:"explanation"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", "", promptTok, completionTok, fmt.Errorf("tool args: %w", err)
		}
		return strings.TrimSpace(args.SQL), strings.TrimSpace(args.Explanation), promptTok, completionTok, nil
	}
	return "", "", promptTok, completionTok, fmt.Errorf("missing tool on retry")
}
