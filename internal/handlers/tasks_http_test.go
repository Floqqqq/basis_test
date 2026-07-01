package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redismock/v9"

	"task-manager/internal/cache"
	"task-manager/internal/middleware"
	"task-manager/internal/repository"
	"task-manager/internal/service"
)

func TestTaskHandlerCreateRejectsInvalidJSON(t *testing.T) {
	handler, _, _, _ := newTaskHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodPost, "/tasks", "{")
	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTaskHandlerCreateRejectsMissingTitle(t *testing.T) {
	handler, _, _, _ := newTaskHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodPost, "/tasks", `{"team_id":1}`)
	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTaskHandlerListRejectsInvalidTeamID(t *testing.T) {
	handler, _, _, _ := newTaskHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodGet, "/tasks?team_id=bad", "")
	handler.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTaskHandlerUpdateReturnsNotFound(t *testing.T) {
	handler, mock, _, _ := newTaskHandlerForTest(t)

	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE id = ?
	`)).
		WithArgs(int64(404)).
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodPut, "/tasks/404", `{"status":"done"}`)
	req = withURLParam(req, "id", "404")
	handler.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskHandlerListReturnsStorageError(t *testing.T) {
	handler, mock, redisMock, _ := newTaskHandlerForTest(t)
	storageErr := errors.New("db failed")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM team_members WHERE team_id = ? AND user_id = ?
		)
	`)).
		WithArgs(int64(1), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	redisMock.ExpectGet(cache.TaskListKey(1, "", nil, 20, 0)).RedisNil()
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE team_id = ?
	 ORDER BY created_at DESC LIMIT ? OFFSET ?`)).
		WithArgs(int64(1), 20, 0).
		WillReturnError(storageErr)

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodGet, "/tasks?team_id=1", "")
	handler.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
	if err := redisMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}

func newTaskHandlerForTest(t *testing.T) (*TaskHandler, sqlmock.Sqlmock, redismock.ClientMock, *sql.DB) {
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

	redisClient, redisMock := redismock.NewClientMock()
	t.Cleanup(func() {
		if err := redisClient.Close(); err != nil {
			t.Logf("redis.Close() error = %v", err)
		}
	})

	teams := repository.NewTeamRepository(db)
	return NewTaskHandler(
		repository.NewTaskRepository(db),
		service.NewTaskPolicy(teams),
		cache.NewTaskCache(redisClient, time.Minute),
	), mock, redisMock, db
}

func requestWithBody(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	return req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, int64(42)))
}

func withURLParam(req *http.Request, name, value string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(name, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
	return req.WithContext(ctx)
}
