package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Deonkar/Aria/aria/internal/ai"
	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/Deonkar/Aria/aria/internal/db"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/google/uuid"
)

// QueryHandler temporary POST /query endpoint (Phase 3 checkpoint).
type QueryHandler struct {
	UserRepo *db.UserRepo
	QuerySvc *ai.QueryService
}

type queryBody struct {
	Question  string `json:"question"`
	SessionID string `json:"session_id"`
}

func (h *QueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body queryBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(body.Question) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "question required")
		return
	}
	sid := strings.TrimSpace(body.SessionID)
	if sid == "" {
		sid = uuid.NewString()
	}

	user, err := h.UserRepo.FindByID(r.Context(), claims.Subject)
	if err != nil || user == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "user lookup failed")
		return
	}

	req := ai.QueryRequest{
		Question:  body.Question,
		SessionID: sid,
		AgentID:   claims.Subject,
		AgentRole: claims.Role,
		AgentName: user.FullName,
		Timezone:  user.Timezone,
	}

	res, err := h.QuerySvc.Query(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"answer":        res.Answer,
		"sql":           res.SQL,
		"row_count":     res.RowCount,
		"execution_ms":  res.ExecutionMs,
		"was_cached":    res.WasCached,
		"was_corrected": res.WasCorrected,
		"tokens_in":     res.TokensIn,
		"tokens_out":    res.TokensOut,
	})
}
