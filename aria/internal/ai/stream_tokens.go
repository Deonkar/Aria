package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

// StreamAnswerTokens streams the formatted natural-language answer token-by-token.
// It reuses the same prompt as the SSE path but emits tokens via the provided callback.
func (s *QueryService) StreamAnswerTokens(
	ctx context.Context,
	question string,
	executedSQL string,
	rows []map[string]any,
	onToken func(token string) error,
) (fullAnswer string, pt int, ct int, err error) {
	payload, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", 0, 0, err
	}

	system := `You are Aria. Summarize the query results in clear, concise natural language for a CRM agent.
Rules:
- Use ONLY the provided JSON rows; never invent rows or values.
- If the row array is empty, say clearly that no matching records were found.`
	user := fmt.Sprintf("Question: %s\nExecuted SQL: %s\nRows (JSON):\n%s", question, executedSQL, string(payload))

	stream := s.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(s.cfg.OpenAIChatModel),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
		Temperature: param.NewOpt(0.1),
	})

	var b strings.Builder
	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) == 0 {
			continue
		}
		piece := evt.Choices[0].Delta.Content
		if piece == "" {
			continue
		}
		b.WriteString(piece)
		if onToken != nil {
			if err := onToken(piece); err != nil {
				return b.String(), 0, 0, err
			}
		}
	}
	if err := stream.Err(); err != nil {
		return b.String(), 0, 0, err
	}

	// Token usage may not be available on all streaming responses; leave 0 if unknown.
	return b.String(), 0, 0, nil
}

