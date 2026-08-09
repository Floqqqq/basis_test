package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
)

func TestRateLimitAllowsFirst100RequestsAndSetsTTL(t *testing.T) {
	client, mock := redismock.NewClientMock()
	defer client.Close()

	mock.ExpectIncr(rateLimitKey(42)).SetVal(1)
	mock.ExpectExpire(rateLimitKey(42), time.Minute).SetVal(true)
	for i := int64(2); i <= 100; i++ {
		mock.ExpectIncr(rateLimitKey(42)).SetVal(i)
	}
	mock.ExpectIncr(rateLimitKey(42)).SetVal(101)

	called := 0
	handler := RateLimit(client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 1; i <= 100; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithUserID())
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want %d", i, rec.Code, http.StatusNoContent)
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithUserID())
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request 101 status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if called != 100 {
		t.Fatalf("handler calls = %d, want 100", called)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}

func TestRateLimitFailsOpenWhenRedisUnavailable(t *testing.T) {
	client, mock := redismock.NewClientMock()
	defer client.Close()

	mock.ExpectIncr(rateLimitKey(42)).SetErr(errors.New("redis down"))

	called := false
	handler := RateLimit(client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithUserID())

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}

func requestWithUserID() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	return req.WithContext(context.WithValue(req.Context(), UserIDKey, int64(42)))
}
