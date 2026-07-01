package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"task-manager/internal/middleware"
	"task-manager/internal/repository"
	"task-manager/internal/service"
)

func TestTeamHandlerCreateRejectsInvalidJSON(t *testing.T) {
	handler, _, _ := newTeamHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := teamRequest(http.MethodPost, "/teams", "{")
	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTeamHandlerCreateRejectsMissingName(t *testing.T) {
	handler, _, _ := newTeamHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := teamRequest(http.MethodPost, "/teams", `{}`)
	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTeamHandlerInviteRejectsInvalidID(t *testing.T) {
	handler, _, _ := newTeamHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := teamRequest(http.MethodPost, "/teams/bad/invite", `{}`)
	req = withURLParam(req, "id", "bad")
	handler.Invite(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTeamHandlerInviteForbiddenWhenRoleMissing(t *testing.T) {
	handler, mock, _ := newTeamHandlerForTest(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`)).
		WithArgs(int64(5), int64(42)).
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	req := teamRequest(http.MethodPost, "/teams/5/invite", `{"user_id":7}`)
	req = withURLParam(req, "id", "5")
	handler.Invite(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newTeamHandlerForTest(t *testing.T) (*TeamHandler, sqlmock.Sqlmock, *sql.DB) {
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

	return NewTeamHandler(repository.NewTeamRepository(db), service.NewMockInviteSender()), mock, db
}

func teamRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	return req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, int64(42)))
}
