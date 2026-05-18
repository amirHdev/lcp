package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/ratelimit"
)

func TestParseBearerToken(t *testing.T) {
	token := buildTestToken(t, Claims{
		Subject: "user-1",
		Role:    "admin",
		Exp:     time.Now().Add(time.Hour).Unix(),
	}, "secret")

	claims, err := ParseBearerToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseBearerToken returned error: %v", err)
	}
	if claims.Subject != "user-1" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestParseBearerTokenRejectsBadSignature(t *testing.T) {
	token := buildTestToken(t, Claims{
		Subject: "user-1",
		Role:    "admin",
		Exp:     time.Now().Add(time.Hour).Unix(),
	}, "secret")

	if _, err := ParseBearerToken(token, "other-secret"); err == nil {
		t.Fatal("expected invalid token error")
	}
}

func TestRequireRoleAllowsAdminWithoutTwoFactorOnNonAdminEndpoint(t *testing.T) {
	mw := New("secret", "123456", nil)
	token := buildTestToken(t, Claims{
		Subject: "user-1",
		Role:    "admin",
		Exp:     time.Now().Add(time.Hour).Unix(),
	}, "secret")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lcp/process", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler := mw.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rec.Code)
	}
}

func TestRequireRoleRequiresTwoFactorOnAdminEndpoint(t *testing.T) {
	mw := New("secret", "123456", nil)
	token := buildTestToken(t, Claims{
		Subject: "user-1",
		Role:    "admin",
		Exp:     time.Now().Add(time.Hour).Unix(),
	}, "secret")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler := mw.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
}

func TestRequireRoleAppliesRateLimit(t *testing.T) {
	mw := New("secret", "", ratelimit.New(1, time.Minute))
	token := buildTestToken(t, Claims{
		Subject: "publisher-1",
		Role:    "publisher",
		Exp:     time.Now().Add(time.Hour).Unix(),
	}, "secret")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lcp/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler := mw.RequireRole("publisher")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d", http.StatusTooManyRequests, second.Code)
	}
}

func TestRequireRoleAcceptsTenantAPIKey(t *testing.T) {
	mw := New("secret", "", nil).WithAPIKeys(func(key string) (*Claims, bool) {
		if key != "tenant-key" {
			return nil, false
		}
		return &Claims{Subject: "integration", TenantID: "tenant-a", Role: "publisher"}, true
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lcp/status", nil)
	req.Header.Set("X-API-Key", "tenant-key")
	rec := httptest.NewRecorder()

	handler := mw.RequireRole("publisher")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := FromContext(r.Context())
		if !ok || claims.TenantID != "tenant-a" {
			t.Fatal("expected tenant claims from api key")
		}
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rec.Code)
	}
}

func TestRequireRoleUsesTenantRateLimit(t *testing.T) {
	mw := New("secret", "", ratelimit.New(10, time.Minute)).
		WithTenantRateLimits(func(tenantID string) int {
			if tenantID == "tenant-a" {
				return 1
			}
			return 0
		})
	token := buildTestToken(t, Claims{
		Subject:  "publisher-1",
		TenantID: "tenant-a",
		Role:     "publisher",
		Exp:      time.Now().Add(time.Hour).Unix(),
	}, "secret")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lcp/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler := mw.RequireRole("publisher")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), req)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d", http.StatusTooManyRequests, second.Code)
	}
}

func buildTestToken(t *testing.T, claims Claims, secret string) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return signingInput + "." + sign(signingInput, secret)
}
