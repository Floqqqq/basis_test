package handlers

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

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

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`)).
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

func TestAuthHandlerMe(t *testing.T) {
	handler, mock, _ := newAuthHandlerForTest(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, created_at FROM users WHERE id = $1`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "created_at"}).
			AddRow(int64(42), "user@example.com", now))

	rec := httptest.NewRecorder()
	handler.Me(rec, requestWithBody(http.MethodGet, "/me", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("response contains password field: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"email":"user@example.com"`) {
		t.Fatalf("response = %s, want user email", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAuthHandlerMeReturnsStorageError(t *testing.T) {
	handler, mock, _ := newAuthHandlerForTest(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, created_at FROM users WHERE id = $1`)).
		WithArgs(int64(42)).
		WillReturnError(errors.New("db failed"))

	rec := httptest.NewRecorder()
	handler.Me(rec, requestWithBody(http.MethodGet, "/me", ""))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rec.Body.String(), `"error":`) {
		t.Fatalf("response = %s, want JSON error", rec.Body.String())
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
