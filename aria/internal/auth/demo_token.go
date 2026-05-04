package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/Deonkar/Aria/aria/internal/db"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/google/uuid"
)

// HandleDemoToken issues a JWT for a fixed DB user when ALLOW_DEMO_AUTH is enabled (local demos only).
func HandleDemoToken(userRepo *db.UserRepo, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !cfg.AllowDemoAuth {
			httpx.WriteError(w, http.StatusForbidden, "demo token endpoint disabled")
			return
		}

		userID := strings.TrimSpace(cfg.DemoUserID)
		if userID == "" {
			httpx.WriteError(w, http.StatusInternalServerError, "demo user id not configured")
			return
		}

		user, err := userRepo.FindByID(r.Context(), userID)
		if err != nil || user == nil {
			httpx.WriteError(w, http.StatusNotFound, "demo user not found in database")
			return
		}

		jti := uuid.NewString()
		accessToken, err := SignToken(user.ID, user.Email, user.Role, jti, cfg.JWTSecret, 24*time.Hour)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to sign token")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"access_token": accessToken,
			"user": map[string]any{
				"id":         user.ID,
				"email":      user.Email,
				"full_name":  user.FullName,
				"avatar_url": user.AvatarURL,
				"role":       user.Role,
			},
		})
	}
}
