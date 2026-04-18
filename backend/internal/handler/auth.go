package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"surajdrive/backend/internal/auth"
	"surajdrive/backend/internal/config"
	"surajdrive/backend/internal/storage"
)

type AuthHandler struct {
	cfg         *config.Config
	oauthConfig *oauth2.Config
	store       *storage.MinIOClient
}

func NewAuthHandler(cfg *config.Config, store *storage.MinIOClient) *AuthHandler {
	return &AuthHandler{
		cfg:   cfg,
		store: store,
		oauthConfig: auth.NewGoogleOAuthConfig(
			cfg.Google.ClientID,
			cfg.Google.ClientSecret,
			cfg.Google.RedirectURL,
		),
	}
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := auth.GenerateStateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("state generation failed: %w", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cfg.Server.IsProd,
		Path:     "/",
	})

	http.Redirect(w, r, h.oauthConfig.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || r.URL.Query().Get("state") != stateCookie.Value {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid state"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cfg.Server.IsProd,
	})

	userInfo, err := auth.GetUserInfo(r.Context(), h.oauthConfig, r.URL.Query().Get("code"))
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("google auth failed: %w", err))
		return
	}

	if allowedDomain := strings.TrimSpace(h.cfg.Google.AllowedDomain); allowedDomain != "" && !strings.HasSuffix(strings.ToLower(userInfo.Email), "@"+strings.ToLower(allowedDomain)) {
		writeError(w, http.StatusForbidden, fmt.Errorf("email domain not permitted"))
		return
	}

	bucket, err := h.store.BucketNameForSubject(userInfo.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := h.store.EnsureBucket(r.Context(), bucket); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("failed to provision user bucket: %w", err))
		return
	}

	jwtToken, err := auth.IssueJWT(
		h.cfg.JWT.Secret,
		userInfo.ID,
		userInfo.Email,
		userInfo.Name,
		userInfo.Picture,
		h.cfg.JWT.ExpiryHrs,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("token issuance failed: %w", err))
		return
	}

	setSessionCookie(w, jwtToken, time.Duration(h.cfg.JWT.ExpiryHrs)*time.Hour, h.cfg.Server.IsProd)

	frontendURL, err := url.Parse(h.cfg.Server.FrontendURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("invalid frontend url: %w", err))
		return
	}

	http.Redirect(w, r, frontendURL.String(), http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w, h.cfg.Server.IsProd)
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed_out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("missing auth claims"))
		return
	}

	bucket, err := h.store.BucketNameForSubject(claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"id":      claims.Subject,
		"email":   claims.Email,
		"name":    claims.Name,
		"picture": claims.Picture,
		"bucket":  bucket,
	})
}

func setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
