package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/Deonkar/Aria/aria/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversationRepo struct {
	pool *pgxpool.Pool
}

func NewConversationRepo(pool *pgxpool.Pool) *ConversationRepo {
	return &ConversationRepo{pool: pool}
}

func (r *ConversationRepo) CreateConversation(ctx context.Context, userID, sessionToken string) (*models.Conversation, error) {
	const q = `
INSERT INTO conversations (user_id, session_token, title, message_count, started_at)
VALUES ($1, $2, NULL, 0, NOW())
RETURNING id, user_id, session_token, title, message_count, started_at, last_message_at, ended_at
`
	var c models.Conversation
	err := r.pool.QueryRow(ctx, q, userID, sessionToken).Scan(
		&c.ID, &c.UserID, &c.SessionToken, &c.Title, &c.MessageCount, &c.StartedAt, &c.LastMessageAt, &c.EndedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return &c, nil
}

func (r *ConversationRepo) FindConversationBySession(ctx context.Context, sessionToken string) (*models.Conversation, error) {
	const q = `
SELECT id, user_id, session_token, title, message_count, started_at, last_message_at, ended_at
FROM conversations
WHERE session_token = $1 AND ended_at IS NULL
`
	var c models.Conversation
	err := r.pool.QueryRow(ctx, q, sessionToken).Scan(
		&c.ID, &c.UserID, &c.SessionToken, &c.Title, &c.MessageCount, &c.StartedAt, &c.LastMessageAt, &c.EndedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConversationRepo) FindConversationByID(ctx context.Context, id string) (*models.Conversation, error) {
	const q = `
SELECT id, user_id, session_token, title, message_count, started_at, last_message_at, ended_at
FROM conversations
WHERE id = $1
`
	var c models.Conversation
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.UserID, &c.SessionToken, &c.Title, &c.MessageCount, &c.StartedAt, &c.LastMessageAt, &c.EndedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConversationRepo) UpdateConversationLastMessage(ctx context.Context, conversationID string, messageDelta int) error {
	_, err := r.pool.Exec(ctx, `
UPDATE conversations
SET last_message_at = NOW(), message_count = message_count + $2
WHERE id = $1
`, conversationID, messageDelta)
	return err
}

func (r *ConversationRepo) EndConversation(ctx context.Context, conversationID, userID string) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE conversations SET ended_at = NOW() WHERE id = $1 AND user_id = $2 AND ended_at IS NULL
`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ConversationRepo) CreateMessage(ctx context.Context, m *models.ChatMessage) (*models.ChatMessage, error) {
	const q = `
INSERT INTO messages (
  conversation_id, user_id, role, content, generated_sql, sql_row_count, sql_execution_ms,
  token_count_input, token_count_output, model_used, was_cached, was_sql_corrected
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING id, created_at
`
	err := r.pool.QueryRow(ctx, q,
		m.ConversationID, m.UserID, m.Role, m.Content, m.GeneratedSQL, m.SQLRowCount, m.SQLExecutionMs,
		m.TokenCountInput, m.TokenCountOutput, m.ModelUsed, m.WasCached, m.WasSQLCorrected,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}
	return m, nil
}

func (r *ConversationRepo) ListConversations(ctx context.Context, userID string) ([]*models.Conversation, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, user_id, session_token, title, message_count, started_at, last_message_at, ended_at
FROM conversations
WHERE user_id = $1 AND ended_at IS NULL
ORDER BY COALESCE(last_message_at, started_at) DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Conversation
	for rows.Next() {
		var c models.Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.SessionToken, &c.Title, &c.MessageCount, &c.StartedAt, &c.LastMessageAt, &c.EndedAt); err != nil {
			return nil, err
		}
		cp := c
		out = append(out, &cp)
	}
	return out, rows.Err()
}

func (r *ConversationRepo) ListMessages(ctx context.Context, conversationID string) ([]*models.ChatMessage, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, conversation_id, user_id, role, content, generated_sql, sql_row_count, sql_execution_ms,
       token_count_input, token_count_output, model_used, was_cached, was_sql_corrected, created_at
FROM messages
WHERE conversation_id = $1
ORDER BY created_at ASC
`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.UserID, &m.Role, &m.Content, &m.GeneratedSQL, &m.SQLRowCount, &m.SQLExecutionMs,
			&m.TokenCountInput, &m.TokenCountOutput, &m.ModelUsed, &m.WasCached, &m.WasSQLCorrected, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		mp := m
		out = append(out, &mp)
	}
	return out, rows.Err()
}

func (r *ConversationRepo) UpdateAssistantMessage(ctx context.Context, id, content string, generatedSQL *string, rowCount, execMs, tokIn, tokOut *int, model *string, wasCached, wasCorrected bool) error {
	_, err := r.pool.Exec(ctx, `
UPDATE messages
SET content = $2, generated_sql = $3, sql_row_count = $4, sql_execution_ms = $5,
    token_count_input = $6, token_count_output = $7, model_used = $8, was_cached = $9, was_sql_corrected = $10
WHERE id = $1
`, id, content, generatedSQL, rowCount, execMs, tokIn, tokOut, model, wasCached, wasCorrected)
	return err
}

func (r *ConversationRepo) FindMessageByID(ctx context.Context, messageID string) (*models.ChatMessage, error) {
	const q = `
SELECT id, conversation_id, user_id, role, content, generated_sql, sql_row_count, sql_execution_ms,
       token_count_input, token_count_output, model_used, was_cached, was_sql_corrected, created_at
FROM messages WHERE id = $1
`
	var m models.ChatMessage
	err := r.pool.QueryRow(ctx, q, messageID).Scan(
		&m.ID, &m.ConversationID, &m.UserID, &m.Role, &m.Content, &m.GeneratedSQL, &m.SQLRowCount, &m.SQLExecutionMs,
		&m.TokenCountInput, &m.TokenCountOutput, &m.ModelUsed, &m.WasCached, &m.WasSQLCorrected, &m.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
