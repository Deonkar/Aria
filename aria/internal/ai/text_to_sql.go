package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

var (
	selectFromTextRe   = regexp.MustCompile(`(?is)\bselect\b[\s\S]*?;`)
	fencedSelectPrefix = regexp.MustCompile(`(?i)^\s*select\b`)
)

func parseToolCallArgsJSON(arguments string) (sql, explanation string, err error) {
	var args struct {
		SQL         string `json:"sql"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(args.SQL), strings.TrimSpace(args.Explanation), nil
}

// tryParseAssistantSQL handles normal tool_calls, legacy function_call, and plain-text
// SELECT…; fallbacks (some OpenAI-compatible proxies omit tool_calls in the response).
func tryParseAssistantSQL(msg openai.ChatCompletionMessage) (sql, explanation string, ok bool) {
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "query_crm_database" {
			continue
		}
		s, e, err := parseToolCallArgsJSON(tc.Function.Arguments)
		if err != nil || s == "" {
			continue
		}
		return s, e, true
	}
	if msg.FunctionCall.Name == "query_crm_database" && msg.FunctionCall.Arguments != "" {
		if s, e, err := parseToolCallArgsJSON(msg.FunctionCall.Arguments); err == nil && s != "" {
			return s, e, true
		}
	}
	if s, ok2 := extractSelectStatementFromText(msg.Content); ok2 {
		e := strings.TrimSpace(strings.Replace(msg.Content, s, "", 1))
		return s, e, true
	}
	return "", "", false
}

func extractSelectStatementFromText(content string) (sql string, ok bool) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", false
	}
	if s := extractFencedSelect(content); s != "" {
		return s, true
	}
	m := strings.TrimSpace(selectFromTextRe.FindString(content))
	if m == "" {
		// Models often omit ';' before a blank line (e.g. "SELECT 1 WHERE false\n\nI can't help…").
		upper := strings.ToLower(content)
		idx := strings.Index(upper, "select")
		if idx < 0 {
			return "", false
		}
		rest := strings.TrimSpace(content[idx:])
		if cut := strings.Index(rest, "\n\n"); cut >= 0 {
			rest = strings.TrimSpace(rest[:cut])
		}
		if rest == "" || !fencedSelectPrefix.MatchString(rest) {
			return "", false
		}
		if !strings.HasSuffix(rest, ";") {
			rest += ";"
		}
		m = rest
	}
	return m, true
}

func extractFencedSelect(content string) string {
	idx := strings.Index(content, "```")
	if idx < 0 {
		return ""
	}
	rest := content[idx+3:]
	rest = strings.TrimLeft(rest, "\r\n")
	if strings.HasPrefix(strings.ToLower(rest), "sql") {
		rest = strings.TrimSpace(rest[3:])
	}
	end := strings.Index(rest, "```")
	if end < 0 {
		return ""
	}
	inner := strings.TrimSpace(rest[:end])
	if !fencedSelectPrefix.MatchString(inner) {
		return ""
	}
	if !strings.HasSuffix(strings.TrimSpace(inner), ";") {
		inner = strings.TrimSpace(inner) + ";"
	}
	return inner
}

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

You MUST call the tool query_crm_database on every turn (the API requires it).
For normal CRM questions: pass a real SELECT and explanation.
For questions outside CRM scope (weather, general knowledge, math trivia): use sql exactly "SELECT id FROM leads WHERE false AND assigned_agent_id = :agent_id" (returns 0 rows; keeps filters valid) and put the helpful refusal in the explanation field.`,
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
		Tools: toolParams(),
		ToolChoice: openai.ChatCompletionToolChoiceOptionParamOfChatCompletionNamedToolChoice(
			openai.ChatCompletionNamedToolChoiceFunctionParam{Name: "query_crm_database"},
		),
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
	if sql, expl, ok := tryParseAssistantSQL(msg); ok {
		return sql, expl, promptTok, completionTok, nil
	}
	content := strings.TrimSpace(msg.Content)
	if content != "" {
		return "", content, promptTok, completionTok, ErrOutOfScope
	}
	return "", "", promptTok, completionTok, fmt.Errorf("missing assistant sql/tool output")
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
		Tools: toolParams(),
		ToolChoice: openai.ChatCompletionToolChoiceOptionParamOfChatCompletionNamedToolChoice(
			openai.ChatCompletionNamedToolChoiceFunctionParam{Name: "query_crm_database"},
		),
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
	if sql, expl, ok := tryParseAssistantSQL(msg); ok {
		return sql, expl, promptTok, completionTok, nil
	}
	return "", strings.TrimSpace(msg.Content), promptTok, completionTok, fmt.Errorf("no tool call on retry")
}
