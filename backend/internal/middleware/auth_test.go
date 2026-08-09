package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"task-manager/internal/service"
)

func TestAuthRejectsMissingToken(t *testing.T) {
	auth := service.NewAuthService("test-secret")
	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
}

func TestAuthSetsUserID(t *testing.T) {
	auth := service.NewAuthService("test-secret")
	token, err := auth.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userID := GetUserID(r); userID != 42 {
			t.Fatalf("userID = %d, want 42", userID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
