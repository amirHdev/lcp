package rest

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/auth"
)

type AuthHandler struct {
	secret        string
	adminUser     string
	adminPass     string
	publisherUser string
	publisherPass string
	admin2FACode  string
}

type LoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	TwoFactor string `json:"twoFactor,omitempty"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	Role      string    `json:"role"`
	Subject   string    `json:"subject"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func NewAuthHandler(secret, adminUser, adminPass, publisherUser, publisherPass, admin2FACode string) *AuthHandler {
	return &AuthHandler{
		secret:        strings.TrimSpace(secret),
		adminUser:     strings.TrimSpace(adminUser),
		adminPass:     strings.TrimSpace(adminPass),
		publisherUser: strings.TrimSpace(publisherUser),
		publisherPass: strings.TrimSpace(publisherPass),
		admin2FACode:  strings.TrimSpace(admin2FACode),
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}
	if h.adminUser == "" || h.adminPass == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "login is not configured"})
		return
	}
	role := ""
	switch {
	case req.Username == h.adminUser && req.Password == h.adminPass:
		role = "admin"
		if h.admin2FACode != "" && req.TwoFactor != h.admin2FACode {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid admin 2fa code"})
			return
		}
	case h.publisherUser != "" && req.Username == h.publisherUser && req.Password == h.publisherPass:
		role = "publisher"
	default:
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid username or password"})
		return
	}
	claims := auth.Claims{
		Subject: req.Username,
		Role:    role,
		Roles:   []string{role},
		Exp:     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token, err := auth.IssueBearerToken(h.secret, claims)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     token,
		Role:      claims.Role,
		Subject:   claims.Subject,
		ExpiresAt: time.Unix(claims.Exp, 0).UTC(),
	})
}

func (h *AuthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
