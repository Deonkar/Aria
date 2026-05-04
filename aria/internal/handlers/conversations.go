package handlers

import (
	"net/http"
	"time"

	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/Deonkar/Aria/aria/internal/cache"
	"github.com/Deonkar/Aria/aria/internal/db"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type ConversationsHandler struct {
	ConvRepo *db.ConversationRepo
	RDB      *redis.Client
}

func (h *ConversationsHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	list, err := h.ConvRepo.ListConversations(r.Context(), claims.Subject)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		title := ""
		if c.Title != nil {
			title = *c.Title
		}
		var last *time.Time
		if c.LastMessageAt != nil {
			last = c.LastMessageAt
		} else {
			t := c.StartedAt
			last = &t
		}
		out = append(out, map[string]any{
			"id":              c.ID,
			"title":           title,
			"message_count":   c.MessageCount,
			"started_at":      c.StartedAt,
			"last_message_at": last,
			"session_token":   c.SessionToken,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *ConversationsHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	conv, err := h.ConvRepo.FindConversationByID(r.Context(), id)
	if err != nil || conv == nil || conv.UserID != claims.Subject {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	if conv.EndedAt != nil {
		httpx.WriteError(w, http.StatusNotFound, "conversation ended")
		return
	}
	msgs, err := h.ConvRepo.ListMessages(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "messages failed")
		return
	}
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		row := map[string]any{
			"id":         m.ID,
			"role":       m.Role,
			"content":    m.Content,
			"was_cached": m.WasCached,
			"created_at": m.CreatedAt,
		}
		if m.GeneratedSQL != nil {
			row["generated_sql"] = *m.GeneratedSQL
		}
		out = append(out, row)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *ConversationsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	conv, err := h.ConvRepo.FindConversationByID(r.Context(), id)
	if err != nil || conv == nil || conv.UserID != claims.Subject {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := h.ConvRepo.EndConversation(r.Context(), id, claims.Subject); err != nil {
		if err == pgx.ErrNoRows {
			httpx.WriteError(w, http.StatusNotFound, "not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	_ = cache.ClearSession(r.Context(), h.RDB, claims.Subject, conv.SessionToken)
	w.WriteHeader(http.StatusNoContent)
}
