package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Deonkar/Aria/aria/internal/cache"
	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/rs/zerolog/log"
)

func writeSSE(w http.ResponseWriter, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", b)
	return err
}

func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// UserFacingError maps infrastructure errors to safe SSE messages (Phase 5).
func UserFacingError(err error) string {
	if err == nil {
		return ""
	}
	if err == ErrQueryTimeout {
		return "Your query took too long. Try asking for a smaller date range."
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") || strings.Contains(msg, "503") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "quota") {
		return "AI service temporarily unavailable, please try again"
	}
	return "AI service temporarily unavailable, please try again"
}

func streamAnswerTokens(ctx context.Context, client openai.Client, cfg *config.Config, w http.ResponseWriter, question, executedSQL string, rows []map[string]any) (full string, pt, ct int, err error) {
	payload, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", 0, 0, err
	}
	system := `You are Aria. Summarize the query results in clear, concise natural language for a CRM agent.
Rules:
- Use ONLY the provided JSON rows; never invent rows or values.
- If the row array is empty, say clearly that no matching records were found.`
	user := fmt.Sprintf("Question: %s\nExecuted SQL: %s\nRows (JSON):\n%s", question, executedSQL, string(payload))

	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(cfg.OpenAIChatModel),
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
		_ = writeSSE(w, map[string]string{"type": "token", "text": piece})
		flush(w)
	}
	if err := stream.Err(); err != nil {
		return b.String(), 0, 0, err
	}
	// Token usage may not be available on all streaming responses; leave 0 if unknown.
	return b.String(), 0, 0, nil
}

func streamCachedAnswer(w http.ResponseWriter, answer string) error {
	const chunk = 20
	for i := 0; i < len(answer); i += chunk {
		j := i + chunk
		if j > len(answer) {
			j = len(answer)
		}
		if err := writeSSE(w, map[string]string{"type": "token", "text": answer[i:j]}); err != nil {
			return err
		}
		flush(w)
	}
	return nil
}

// StreamQuery implements TASK-028 SSE flow. assistantMessageID is included in the final done event when set.
func StreamQuery(ctx context.Context, w http.ResponseWriter, svc *QueryService, req QueryRequest, assistantMessageID string) (*StreamResult, error) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flush(w)

	res := &StreamResult{}

	if cr, ok, err := cache.GetCachedQuery(ctx, svc.rdb, req.AgentID, req.Question); err == nil && ok {
		res.WasCached = true
		res.Cached = true
		res.SQL = cr.SQL
		res.RowCount = cr.RowCount
		if err := streamCachedAnswer(w, cr.Answer); err != nil {
			return res, err
		}
		res.Answer = cr.Answer
		done := map[string]any{"type": "done", "cached": true, "row_count": cr.RowCount}
		if assistantMessageID != "" {
			done["message_id"] = assistantMessageID
		}
		_ = writeSSE(w, done)
		flush(w)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flush(w)
		return res, nil
	} else if err != nil {
		log.Warn().Err(err).Msg("redis query cache unavailable, skipping cache read")
	}

	sqlExec, rows, execMs, corrected, tin, tout, oosAns, err := svc.GenerateExecuteAndFormat(ctx, req)
	res.ExecutionMs = execMs
	res.WasCorrected = corrected
	res.TokensIn = tin
	res.TokensOut = tout

	if err == ErrOutOfScope {
		chunk := oosAns
		for i := 0; i < len(chunk); i += 20 {
			j := i + 20
			if j > len(chunk) {
				j = len(chunk)
			}
			_ = writeSSE(w, map[string]string{"type": "token", "text": chunk[i:j]})
			flush(w)
		}
		res.Answer = oosAns
		done := map[string]any{"type": "done", "cached": false, "row_count": 0, "out_of_scope": true}
		if assistantMessageID != "" {
			done["message_id"] = assistantMessageID
		}
		_ = writeSSE(w, done)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flush(w)
		return res, nil
	}

	if err != nil {
		msg := UserFacingError(err)
		_ = writeSSE(w, map[string]string{"type": "error", "message": msg})
		if assistantMessageID != "" {
			_ = writeSSE(w, map[string]any{"type": "done", "message_id": assistantMessageID, "cached": false, "row_count": 0, "error": true})
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flush(w)
		res.ErrorMessage = msg
		return res, nil
	}

	res.SQL = sqlExec
	res.RowCount = len(rows)
	if err := writeSSE(w, map[string]string{"type": "sql", "sql": sqlExec}); err != nil {
		return res, err
	}
	flush(w)

	answer, fpt, fct, err := streamAnswerTokens(ctx, svc.client, svc.cfg, w, req.Question, sqlExec, rows)
	res.TokensIn += fpt
	res.TokensOut += fct
	res.Answer = answer
	if err != nil {
		msg := UserFacingError(err)
		_ = writeSSE(w, map[string]string{"type": "error", "message": msg})
		if assistantMessageID != "" {
			_ = writeSSE(w, map[string]any{"type": "done", "message_id": assistantMessageID, "cached": false, "row_count": 0, "error": true})
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flush(w)
		res.ErrorMessage = msg
		return res, nil
	}

	cr := &cache.CachedResult{
		Answer:      answer,
		SQL:         sqlExec,
		RowCount:    len(rows),
		GeneratedAt: time.Now().UTC(),
	}
	if err := cache.SetCachedQuery(ctx, svc.rdb, req.AgentID, req.Question, cr, QueryCacheTTL); err != nil {
		log.Warn().Err(err).Msg("redis query cache set failed")
	}

	_ = cache.AppendMessage(ctx, svc.rdb, req.AgentID, req.SessionID, cache.SessionMessage{
		Role:    "user",
		Content: req.Question,
	}, SessionMaxMsgs, SessionTTL)
	_ = cache.AppendMessage(ctx, svc.rdb, req.AgentID, req.SessionID, cache.SessionMessage{
		Role:         "assistant",
		Content:      answer,
		GeneratedSQL: sqlExec,
	}, SessionMaxMsgs, SessionTTL)

	done := map[string]any{"type": "done", "cached": false, "row_count": len(rows)}
	if assistantMessageID != "" {
		done["message_id"] = assistantMessageID
	}
	_ = writeSSE(w, done)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flush(w)
	return res, nil
}
