package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/Deonkar/Aria/aria/internal/db"
	"github.com/Deonkar/Aria/aria/internal/httpx"
	"github.com/Deonkar/Aria/aria/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func GoogleConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func HandleGoogleLogin(oauthCfg *oauth2.Config, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := randomState(32)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to start login")
			return
		}

		key := "oauth_state:" + state
		if err := rdb.Set(r.Context(), key, "1", 10*time.Minute).Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to start login")
			return
		}

		url := oauthCfg.AuthCodeURL(state)
		http.Redirect(w, r, url, http.StatusFound)
	}
}

func HandleGoogleCallback(
	oauthCfg *oauth2.Config,
	userRepo *db.UserRepo,
	rdb *redis.Client,
	cfg *config.Config,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		if code == "" || state == "" {
			httpx.WriteError(w, http.StatusBadRequest, "missing code/state")
			return
		}

		stateKey := "oauth_state:" + state
		ok, err := rdb.Exists(r.Context(), stateKey).Result()
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "oauth state check failed")
			return
		}
		if ok == 0 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid state")
			return
		}
		_ = rdb.Del(r.Context(), stateKey).Err()

		tok, err := oauthCfg.Exchange(r.Context(), code)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "oauth exchange failed")
			return
		}

		info, err := fetchGoogleUserInfo(r.Context(), tok)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "failed to fetch google user info")
			return
		}

		user, err := userRepo.FindByGoogleID(r.Context(), info.ID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "user lookup failed")
			return
		}

		if user == nil {
			avatar := strings.TrimSpace(info.Picture)
			var avatarPtr *string
			if avatar != "" {
				avatarPtr = &avatar
			}
			user = &models.User{
				GoogleID:  info.ID,
				Email:     info.Email,
				FullName:  info.Name,
				AvatarURL: avatarPtr,
				Role:      "agent",
				Timezone:  "Asia/Kolkata",
				IsActive:  true,
			}
			user, err = userRepo.Create(r.Context(), user)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "user create failed")
				return
			}
		}

		_ = userRepo.UpdateLastLogin(r.Context(), user.ID)

		jti := uuid.NewString()
		accessToken, err := SignToken(user.ID, user.Email, user.Role, jti, cfg.JWTSecret, 8*time.Hour)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to sign token")
			return
		}

		refreshToken := uuid.NewString()
		if err := rdb.Set(r.Context(), "refresh:"+refreshToken, user.ID, 30*24*time.Hour).Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to create refresh token")
			return
		}

		setRefreshCookie(w, refreshToken, shouldSecureCookie(cfg))

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

func HandleRefresh(userRepo *db.UserRepo, rdb *redis.Client, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt, err := r.Cookie("refresh_token")
		if err != nil || strings.TrimSpace(rt.Value) == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "missing refresh token")
			return
		}

		userID, err := rdb.Get(r.Context(), "refresh:"+rt.Value).Result()
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}

		user, err := userRepo.FindByID(r.Context(), userID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "user lookup failed")
			return
		}
		if user == nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}

		jti := uuid.NewString()
		accessToken, err := SignToken(user.ID, user.Email, user.Role, jti, cfg.JWTSecret, 8*time.Hour)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to sign token")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"access_token": accessToken})
	}
}

func HandleLogout(rdb *redis.Client, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims == nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ttl := time.Until(claims.ExpiresAt.Time)
		_ = BlacklistToken(r.Context(), rdb, claims.ID, ttl)

		if c, err := r.Cookie("refresh_token"); err == nil && strings.TrimSpace(c.Value) != "" {
			_ = rdb.Del(r.Context(), "refresh:"+c.Value).Err()
		}
		clearRefreshCookie(w, secureCookie)

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"message": "logged out"})
	}
}

func fetchGoogleUserInfo(ctx context.Context, tok *oauth2.Token) (*GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("userinfo status %d: %s", res.StatusCode, string(b))
	}
	var info GoogleUserInfo
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, err
	}
	if strings.TrimSpace(info.ID) == "" || strings.TrimSpace(info.Email) == "" {
		return nil, errors.New("missing id/email in google response")
	}
	return &info, nil
}

func randomState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setRefreshCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
	})
}

func clearRefreshCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		MaxAge:   -1,
	})
}

func shouldSecureCookie(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	// Secure cookies do not work over plain http://localhost; enable automatically when redirect is https.
	return strings.HasPrefix(strings.ToLower(cfg.GoogleRedirectURL), "https://")
}

