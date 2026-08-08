package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

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

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2`)).
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

func TestTeamHandlerMembersForbiddenForOutsider(t *testing.T) {
	handler, mock, _ := newTeamHandlerForTest(t)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
		)
	`)).
		WithArgs(int64(5), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	rec := httptest.NewRecorder()
	req := teamRequest(http.MethodGet, "/teams/5/members", "")
	req = withURLParam(req, "id", "5")
	handler.Members(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTeamHandlerMembersForTeamMember(t *testing.T) {
	handler, mock, _ := newTeamHandlerForTest(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
		)
	`)).
		WithArgs(int64(5), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT u.id, u.email, tm.role, tm.created_at
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id = $1
		ORDER BY tm.created_at, u.id
	`)).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "role", "created_at"}).
			AddRow(int64(42), "owner@example.com", "owner", now))

	rec := httptest.NewRecorder()
	req := teamRequest(http.MethodGet, "/teams/5/members", "")
	req = withURLParam(req, "id", "5")
	handler.Members(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"role":"owner"`) {
		t.Fatalf("response = %s, want owner role", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("response contains password field: %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTeamHandlerListIncludesCurrentUserRole(t *testing.T) {
	handler, mock, _ := newTeamHandlerForTest(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT t.id, t.name, t.created_by, t.created_at, tm.role
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		ORDER BY t.created_at DESC
	`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_by", "created_at", "role"}).
			AddRow(int64(5), "Backend", int64(42), now, "owner"))

	rec := httptest.NewRecorder()
	handler.List(rec, teamRequest(http.MethodGet, "/teams", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"role":"owner"`) {
		t.Fatalf("response = %s, want owner role", rec.Body.String())
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

	teamService := service.NewTeamService(repository.NewTeamRepository(db), service.NewMockInviteSender())
	return NewTeamHandler(teamService), mock, db
}

func teamRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	return req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, int64(42)))
}
