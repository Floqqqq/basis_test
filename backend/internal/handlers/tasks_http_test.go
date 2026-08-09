package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
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
			WHERE id = $1
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
			SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
		)
	`)).
		WithArgs(int64(1), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	redisMock.ExpectGet(cache.TaskListKey(1, "", nil, 20, 0)).RedisNil()
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE team_id = $1
	 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`)).
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

func TestTaskHandlerGetForbiddenForOutsider(t *testing.T) {
	handler, mock, _, _ := newTaskHandlerForTest(t)
	expectTaskByID(mock, 7)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
		)
	`)).
		WithArgs(int64(5), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodGet, "/tasks/7", "")
	req = withURLParam(req, "id", "7")
	handler.Get(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskHandlerGetForTeamMember(t *testing.T) {
	handler, mock, _, _ := newTaskHandlerForTest(t)
	expectTaskByID(mock, 7)
	expectTaskMembership(mock, 5, 42, true)

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodGet, "/tasks/7", "")
	req = withURLParam(req, "id", "7")
	handler.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"id":7`) {
		t.Fatalf("response = %s, want task id", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskHandlerGetUnknownTask(t *testing.T) {
	handler, mock, _, _ := newTaskHandlerForTest(t)
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE id = $1
	`)).
		WithArgs(int64(404)).
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodGet, "/tasks/404", "")
	req = withURLParam(req, "id", "404")
	handler.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskHandlerCreateComment(t *testing.T) {
	handler, mock, _, _ := newTaskHandlerForTest(t)
	now := time.Now()
	expectTaskByID(mock, 7)
	expectTaskMembership(mock, 5, 42, true)
	mock.ExpectQuery(regexp.QuoteMeta(`
		WITH inserted AS (
			INSERT INTO task_comments(task_id, user_id, comment)
			VALUES ($1, $2, $3)
			RETURNING id, task_id, user_id, comment, created_at
		)
		SELECT i.id, i.task_id, i.user_id, u.email, i.comment, i.created_at
		FROM inserted i
		JOIN users u ON u.id = i.user_id
	`)).
		WithArgs(int64(7), int64(42), "Check this case").
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "user_id", "email", "comment", "created_at"}).
			AddRow(int64(3), int64(7), int64(42), "user@example.com", "Check this case", now))

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodPost, "/tasks/7/comments", `{"comment":"Check this case"}`)
	req = withURLParam(req, "id", "7")
	handler.CreateComment(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if !strings.Contains(rec.Body.String(), `"email":"user@example.com"`) {
		t.Fatalf("response = %s, want comment author email", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskHandlerListComments(t *testing.T) {
	handler, mock, _, _ := newTaskHandlerForTest(t)
	now := time.Now()
	expectTaskByID(mock, 7)
	expectTaskMembership(mock, 5, 42, true)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT tc.id, tc.task_id, tc.user_id, u.email, tc.comment, tc.created_at
		FROM task_comments tc
		JOIN users u ON u.id = tc.user_id
		WHERE tc.task_id = $1
		ORDER BY tc.created_at, tc.id
	`)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "user_id", "email", "comment", "created_at"}).
			AddRow(int64(3), int64(7), int64(42), "user@example.com", "Check this case", now))

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodGet, "/tasks/7/comments", "")
	req = withURLParam(req, "id", "7")
	handler.ListComments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"text":"Check this case"`) {
		t.Fatalf("response = %s, want comment", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskHandlerCreateCommentRejectsEmptyComment(t *testing.T) {
	handler, mock, _, _ := newTaskHandlerForTest(t)

	rec := httptest.NewRecorder()
	req := requestWithBody(http.MethodPost, "/tasks/7/comments", `{"comment":"   "}`)
	req = withURLParam(req, "id", "7")
	handler.CreateComment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func expectTaskByID(mock sqlmock.Sqlmock, taskID int64) {
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE id = $1
	`)).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "assignee_id", "completed_at", "team_id", "created_by", "created_at", "updated_at",
		}).AddRow(taskID, "Task", "Description", "todo", nil, nil, int64(5), int64(42), now, now))
}

func expectTaskMembership(mock sqlmock.Sqlmock, teamID, userID int64, isMember bool) {
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
		)
	`)).
		WithArgs(teamID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(isMember))
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
	taskService := service.NewTaskService(
		repository.NewTaskRepository(db),
		service.NewTaskPolicy(teams),
		cache.NewTaskCache(redisClient, time.Minute),
	)
	return NewTaskHandler(
		taskService,
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
