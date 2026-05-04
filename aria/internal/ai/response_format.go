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

func FormatAnswer(ctx context.Context, client openai.Client, cfg *config.Config, question, executedSQL string, rows []map[string]any) (string, int, int, error) {
	payload, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", 0, 0, err
	}
	system := `You are Aria. Summarize the query results in clear, concise natural language for a CRM agent.
Rules:
- Use ONLY the provided JSON rows; never invent rows or values.
- If the row array is empty, say clearly that no matching records were found.
- Do not show raw SQL unless asked.`

	user := fmt.Sprintf("Question: %s\nExecuted SQL: %s\nRows (JSON):\n%s", question, executedSQL, string(payload))

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(cfg.OpenAIChatModel),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
		Temperature: param.NewOpt(0.1),
	}

	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", 0, 0, err
	}
	if len(resp.Choices) == 0 {
		return "", 0, 0, fmt.Errorf("no choices")
	}
	text := strings.TrimSpace(resp.Choices[0].Message.Content)
	pt := int(resp.Usage.PromptTokens)
	ct := int(resp.Usage.CompletionTokens)
	return text, pt, ct, nil
}
