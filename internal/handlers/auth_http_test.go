package handlers

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"task-manager/internal/repository"
	"task-manager/internal/service"
)

func TestAuthHandlerRegisterRejectsInvalidJSON(t *testing.T) {
	handler, _, _ := newAuthHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("{"))
	handler.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthHandlerRegisterRejectsMissingPassword(t *testing.T) {
	handler, _, _ := newAuthHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"user@example.com"}`))
	handler.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthHandlerLoginInvalidCredentials(t *testing.T) {
	handler, mock, _ := newAuthHandlerForTest(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`)).
		WithArgs("user@example.com").
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"user@example.com","password":"secret1"}`))
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newAuthHandlerForTest(t *testing.T) (*AuthHandler, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("db.Close() error = %v", err)
		}
	})

	return NewAuthHandler(repository.NewUserRepository(db), service.NewAuthService("test-secret")), mock, db
}
