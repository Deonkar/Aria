package logging

import (
	"net/http"
	"time"

	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/rs/zerolog/log"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		ev := log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", sw.status).
			Int64("duration_ms", time.Since(start).Milliseconds())

		if c, ok := auth.ClaimsFromContext(r.Context()); ok && c != nil {
			ev = ev.Str("user_id", c.Subject).Str("role", c.Role)
		}

		ev.Msg("http request")
	})
}

