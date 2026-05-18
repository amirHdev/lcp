package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/observability"
	"github.com/amirhdev/ebook-lcp-server/internal/ratelimit"
)

type contextKey string

const claimsContextKey contextKey = "auth.claims"

var (
	ErrMissingToken = errors.New("missing bearer token")
	ErrInvalidToken = errors.New("invalid bearer token")
	ErrExpiredToken = errors.New("expired bearer token")
	ErrForbidden    = errors.New("insufficient permissions")
)

type Claims struct {
	Subject  string   `json:"sub"`
	TenantID string   `json:"tenantId,omitempty"`
	Role     string   `json:"role"`
	Roles    []string `json:"roles,omitempty"`
	Exp      int64    `json:"exp,omitempty"`
}

type APIKeyResolver func(key string) (*Claims, bool)
type RateLimitResolver func(tenantID string) int

func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

type Middleware struct {
	secret       string
	admin2FACode string
	limiter      *ratelimit.Limiter
	apiKeys      APIKeyResolver
	tenantLimits RateLimitResolver
	tenantBucket map[string]*ratelimit.Limiter
	mu           sync.Mutex
}

func New(secret, admin2FACode string, limiter *ratelimit.Limiter) *Middleware {
	return &Middleware{
		secret:       strings.TrimSpace(secret),
		admin2FACode: strings.TrimSpace(admin2FACode),
		limiter:      limiter,
		tenantBucket: map[string]*ratelimit.Limiter{},
	}
}

func (m *Middleware) WithAPIKeys(resolver APIKeyResolver) *Middleware {
	m.apiKeys = resolver
	return m
}

func (m *Middleware) WithTenantRateLimits(resolver RateLimitResolver) *Middleware {
	m.tenantLimits = resolver
	return m
}

func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, role := range roles {
		allowed[strings.ToLower(role)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := m.parseRequest(r)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, err)
				return
			}
			if !roleAllowed(claims, allowed) {
				writeAuthError(w, http.StatusForbidden, ErrForbidden)
				return
			}
			if hasRole(claims, "admin") && m.admin2FACode != "" && requiresTwoFactor(r) && r.Header.Get("X-2FA-Code") != m.admin2FACode {
				writeAuthError(w, http.StatusForbidden, errors.New("invalid admin 2fa code"))
				return
			}
			if !m.allow(r, claims) {
				writeAuthError(w, http.StatusTooManyRequests, errors.New("rate limit exceeded"))
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

func rateLimitKey(r *http.Request, claims *Claims) string {
	if claims != nil && claims.Subject != "" {
		return claims.Subject
	}
	return r.RemoteAddr
}

func (m *Middleware) allow(r *http.Request, claims *Claims) bool {
	if claims != nil && claims.TenantID != "" && m.tenantLimits != nil {
		if limit := m.tenantLimits(claims.TenantID); limit > 0 {
			m.mu.Lock()
			limiter, ok := m.tenantBucket[claims.TenantID]
			if !ok {
				limiter = ratelimit.New(limit, time.Minute)
				m.tenantBucket[claims.TenantID] = limiter
			}
			m.mu.Unlock()
			return limiter.Allow(rateLimitKey(r, claims))
		}
	}
	return m.limiter.Allow(rateLimitKey(r, claims))
}

func (m *Middleware) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := m.parseRequest(r)
		if err == nil {
			r = r.WithContext(WithClaims(r.Context(), claims))
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) parseRequest(r *http.Request) (*Claims, error) {
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" && m.apiKeys != nil {
		if claims, ok := m.apiKeys(key); ok {
			return claims, nil
		}
		return nil, ErrInvalidToken
	}
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return nil, ErrMissingToken
	}
	return ParseBearerToken(strings.TrimSpace(header[7:]), m.secret)
}

func ParseBearerToken(token, secret string) (*Claims, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is not configured")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expected := sign(signingInput, secret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, ErrExpiredToken
	}
	if claims.Role == "" && len(claims.Roles) > 0 {
		claims.Role = claims.Roles[0]
	}
	return &claims, nil
}

func IssueBearerToken(secret string, claims Claims) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret is not configured")
	}
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return signingInput + "." + sign(signingInput, secret), nil
}

func roleAllowed(claims *Claims, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, role := range append(claims.Roles, claims.Role) {
		if _, ok := allowed[strings.ToLower(role)]; ok {
			return true
		}
	}
	return false
}

func hasRole(claims *Claims, role string) bool {
	return roleAllowed(claims, map[string]struct{}{strings.ToLower(role): {}})
}

func requiresTwoFactor(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/v1/admin/")
}

func sign(input, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func writeAuthError(w http.ResponseWriter, status int, err error) {
	observability.IncAuthFailed()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
