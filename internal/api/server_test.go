package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddlewareRequiresConfiguredToken(t *testing.T) {
	t.Setenv("DROGONCLAW_API_KEY", "")
	handler := AuthMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer any-value")
	res := httptest.NewRecorder()

	handler(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, res.Code)
	}
}

func TestAuthMiddlewareAcceptsOnlyConfiguredToken(t *testing.T) {
	t.Setenv("DROGONCLAW_API_KEY", "test-token")
	handler := AuthMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	valid := httptest.NewRequest(http.MethodGet, "/", nil)
	valid.Header.Set("Authorization", "Bearer test-token")
	validRes := httptest.NewRecorder()
	handler(validRes, valid)
	if validRes.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, validRes.Code)
	}

	invalid := httptest.NewRequest(http.MethodGet, "/", nil)
	invalid.Header.Set("Authorization", "Bearer wrong-token")
	invalidRes := httptest.NewRecorder()
	handler(invalidRes, invalid)
	if invalidRes.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, invalidRes.Code)
	}
}
