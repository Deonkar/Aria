package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Deonkar/Aria/aria/internal/ai"
	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/Deonkar/Aria/aria/internal/cache"
	"github.com/Deonkar/Aria/aria/internal/db"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/Deonkar/Aria/aria/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go"
	"github.com/pgvector/pgvector-go"
	"github.com/redis/go-redis/v9"
)

type FeedbackHandler struct {
	Pool       *pgxpool.Pool
	ConvRepo   *db.ConversationRepo
	RDB        *redis.Client
	OA         openai.Client
	EmbedModel string
}

type feedbackBody struct {
	IsHelpful      bool   `json:"is_helpful"`
	CorrectionNote string `json:"correction_note"`
	CorrectedSQL   string `json:"corrected_sql"`
}

func (h *FeedbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	msgID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(msgID); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body feedbackBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	msg, err := h.ConvRepo.FindMessageByID(r.Context(), msgID)
	if err != nil || msg == nil {
		httpx.WriteError(w, http.StatusNotFound, "message not found")
		return
	}
	if msg.UserID != claims.Subject {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	var note any
	var corr any
	if strings.TrimSpace(body.CorrectionNote) != "" {
		note = body.CorrectionNote
	}
	if strings.TrimSpace(body.CorrectedSQL) != "" {
		corr = body.CorrectedSQL
	}

	var qID string
	err = h.Pool.QueryRow(r.Context(), `
INSERT INTO query_feedback (message_id, user_id, is_helpful, correction_note, corrected_sql)
VALUES ($1, $2, $3, $4, $5)
RETURNING id
`, msgID, claims.Subject, body.IsHelpful, note, corr).Scan(&qID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.WriteError(w, http.StatusConflict, "feedback already submitted")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "feedback save failed")
		return
	}

	userQ, err := previousUserQuestion(r.Context(), h.Pool, msg)
	if err == nil && strings.TrimSpace(userQ) != "" && !body.IsHelpful {
		_ = cache.InvalidateCachedQuery(r.Context(), h.RDB, claims.Subject, userQ)
	}

	if body.IsHelpful && msg.GeneratedSQL != nil && strings.TrimSpace(*msg.GeneratedSQL) != "" {
		_ = maybePromoteIntentExample(r.Context(), h.Pool, h.OA, h.EmbedModel, userQ, *msg.GeneratedSQL)
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": qID, "message_id": msgID, "is_helpful": body.IsHelpful})
}

func previousUserQuestion(ctx context.Context, pool *pgxpool.Pool, assistant *models.ChatMessage) (string, error) {
	var content string
	err := pool.QueryRow(ctx, `
SELECT m2.content
FROM messages m
JOIN messages m2 ON m2.conversation_id = m.conversation_id AND m2.role = 'user' AND m2.created_at <= m.created_at
WHERE m.id = $1
ORDER BY m2.created_at DESC
LIMIT 1
`, assistant.ID).Scan(&content)
	return content, err
}

func maybePromoteIntentExample(ctx context.Context, pool *pgxpool.Pool, client openai.Client, embedModel, question, generatedSQL string) error {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM query_feedback qf
JOIN messages m ON m.id = qf.message_id
WHERE qf.is_helpful = true AND m.generated_sql = $1 AND m.role = 'assistant'
`, generatedSQL).Scan(&n)
	if err != nil || n < 3 {
		return err
	}

	var exists int
	_ = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM intent_examples WHERE sql_template = $1 AND source = 'promoted_feedback'
`, generatedSQL).Scan(&exists)
	if exists > 0 {
		return nil
	}

	q := question
	if strings.TrimSpace(q) == "" {
		q = "promoted feedback example"
	}
	vec, _, _, err := ai.EmbedText(ctx, client, embedModel, q)
	if err != nil {
		return err
	}
	pgvec := pgvector.NewVector(vec)
	_, err = pool.Exec(ctx, `
INSERT INTO intent_examples (question_text, question_embedding, sql_template, tables_used, intent_category, source)
VALUES ($1, $2, $3, ARRAY[]::text[], 'promoted_feedback', 'promoted_feedback')
`, q, pgvec, generatedSQL)
	return err
}
