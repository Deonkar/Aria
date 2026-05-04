package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Deonkar/Aria/aria/internal/ai"
	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/Deonkar/Aria/aria/internal/db"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/Deonkar/Aria/aria/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type ChatHandler struct {
	UserRepo         *db.UserRepo
	ConvRepo         *db.ConversationRepo
	QuerySvc         *ai.QueryService
	RDB              *redis.Client
}

type chatBody struct {
	Question       string `json:"question"`
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body chatBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	q, err := NormalizeQuestion(body.Question)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	sessionID := strings.TrimSpace(body.SessionID)
	convID := strings.TrimSpace(body.ConversationID)
	if convID != "" {
		if _, err := uuid.Parse(convID); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid conversation_id")
			return
		}
	}
	if sessionID != "" {
		if _, err := uuid.Parse(sessionID); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid session_id")
			return
		}
	}

	allowed, _, err := TryAIRateLimit(r.Context(), h.RDB, claims.Subject)
	if err != nil {
		log.Warn().Err(err).Msg("rate limit redis error, allowing request")
	}
	if !allowed {
		w.Header().Set("Retry-After", "60")
		httpx.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	user, err := h.UserRepo.FindByID(r.Context(), claims.Subject)
	if err != nil || user == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "user lookup failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	var conv *models.Conversation
	if convID != "" {
		conv, err = h.ConvRepo.FindConversationByID(ctx, convID)
		if err != nil || conv == nil || conv.UserID != claims.Subject || conv.EndedAt != nil {
			httpx.WriteError(w, http.StatusNotFound, "conversation not found")
			return
		}
	} else {
		if sessionID == "" {
			sessionID = uuid.NewString()
		}
		conv, err = h.ConvRepo.FindConversationBySession(ctx, sessionID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "conversation lookup failed")
			return
		}
		if conv == nil {
			conv, err = h.ConvRepo.CreateConversation(ctx, claims.Subject, sessionID)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "conversation create failed")
				return
			}
		}
	}

	userMsg := &models.ChatMessage{
		ConversationID: conv.ID,
		UserID:         claims.Subject,
		Role:           "user",
		Content:        q,
	}
	if _, err := h.ConvRepo.CreateMessage(ctx, userMsg); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to save message")
		return
	}
	_ = h.ConvRepo.UpdateConversationLastMessage(ctx, conv.ID, 1)

	assistantPlaceholder := &models.ChatMessage{
		ConversationID: conv.ID,
		UserID:         claims.Subject,
		Role:           "assistant",
		Content:        "",
	}
	if _, err := h.ConvRepo.CreateMessage(ctx, assistantPlaceholder); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to save assistant message")
		return
	}
	_ = h.ConvRepo.UpdateConversationLastMessage(ctx, conv.ID, 1)

	redisSession := conv.SessionToken

	req := ai.QueryRequest{
		Question:  q,
		SessionID: redisSession,
		AgentID:   claims.Subject,
		AgentRole: claims.Role,
		AgentName: user.FullName,
		Timezone:  user.Timezone,
	}

	streamRes, err := ai.StreamQuery(ctx, w, h.QuerySvc, req, assistantPlaceholder.ID)
	if err != nil {
		log.Error().Err(err).Str("user_id", claims.Subject).Msg("stream query failed")
	}

	finalText := streamRes.Answer
	if finalText == "" && streamRes.ErrorMessage != "" {
		finalText = streamRes.ErrorMessage
	}

	modelName := h.QuerySvc.Cfg().OpenAIChatModel
	sqlPtr := stringPtrOrNil(streamRes.SQL)
	rc := streamRes.RowCount
	em := int(streamRes.ExecutionMs)
	tin := streamRes.TokensIn
	tout := streamRes.TokensOut
	_ = h.ConvRepo.UpdateAssistantMessage(ctx, assistantPlaceholder.ID, finalText, sqlPtr, &rc, &em, &tin, &tout, &modelName, streamRes.WasCached, streamRes.WasCorrected)
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
