package handlers

import (
	"net/http"

	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminHandler struct {
	Pool *pgxpool.Pool
}

func (h *AdminHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if claims.Role != "admin" {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	ctx := r.Context()

	var queriesToday int
	_ = h.Pool.QueryRow(ctx, `
SELECT COUNT(*) FROM messages WHERE role = 'assistant' AND created_at >= CURRENT_DATE
`).Scan(&queriesToday)

	var avgMs *float64
	_ = h.Pool.QueryRow(ctx, `
SELECT AVG(sql_execution_ms)::float8 FROM messages
WHERE role = 'assistant' AND created_at >= CURRENT_DATE AND sql_execution_ms IS NOT NULL
`).Scan(&avgMs)

	var cachedCnt, totalToday int
	_ = h.Pool.QueryRow(ctx, `
SELECT COALESCE(SUM(CASE WHEN was_cached THEN 1 ELSE 0 END), 0)::int,
       COUNT(*)::int
FROM messages WHERE role = 'assistant' AND created_at >= CURRENT_DATE
`).Scan(&cachedCnt, &totalToday)

	cacheRate := 0.0
	if totalToday > 0 {
		cacheRate = float64(cachedCnt) / float64(totalToday)
	}

	var helpful, totalFB int
	_ = h.Pool.QueryRow(ctx, `
SELECT COALESCE(SUM(CASE WHEN is_helpful THEN 1 ELSE 0 END), 0)::int, COUNT(*)::int
FROM query_feedback WHERE created_at >= CURRENT_DATE
`).Scan(&helpful, &totalFB)

	feedbackScore := 0.0
	if totalFB > 0 {
		feedbackScore = float64(helpful) / float64(totalFB)
	}

	var gaps int
	_ = h.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM intent_gaps WHERE is_resolved = false`).Scan(&gaps)

	avg := 0.0
	if avgMs != nil {
		avg = *avgMs
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"queries_today":           queriesToday,
		"avg_response_ms":         avg,
		"cache_hit_rate":          cacheRate,
		"top_question_types":      []string{},
		"feedback_score":          feedbackScore,
		"intent_gaps_unresolved":  gaps,
	})
}
