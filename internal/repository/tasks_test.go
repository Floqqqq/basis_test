package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"task-manager/internal/models"
)

func TestTaskRepositoryListBuildsFilteredQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewTaskRepository(db)
	assigneeID := int64(7)
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "title", "description", "status", "assignee_id", "completed_at", "team_id", "created_by", "created_at", "updated_at",
	}).AddRow(int64(1), "Task", "Desc", "todo", assigneeID, nil, int64(2), int64(3), now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE team_id = ?
	 AND status = ? AND assignee_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`)).
		WithArgs(int64(2), "todo", assigneeID, 20, 40).
		WillReturnRows(rows).
		RowsWillBeClosed()

	tasks, err := repo.List(context.Background(), 2, "todo", &assigneeID, 20, 40)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != 1 || tasks[0].AssigneeID == nil || *tasks[0].AssigneeID != assigneeID {
		t.Fatalf("tasks = %+v, want one task with assignee", tasks)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskRepositoryGetByIDNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewTaskRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE id = ?
	`)).
		WithArgs(int64(404)).
		WillReturnError(sql.ErrNoRows)

	task, err := repo.GetByID(context.Background(), 404)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetByID() error = %v, want sql.ErrNoRows", err)
	}
	if task != nil {
		t.Fatalf("task = %+v, want nil", task)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskRepositoryUpdateWritesHistoryInTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewTaskRepository(db)
	now := time.Now()

	oldRows := sqlmock.NewRows([]string{
		"id", "title", "description", "status", "assignee_id", "completed_at", "team_id", "created_by", "created_at", "updated_at",
	}).AddRow(int64(1), "Task", "Desc", "todo", nil, nil, int64(2), int64(3), now, now)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE id = ?
	`)).
		WithArgs(int64(1)).
		WillReturnRows(oldRows)
	mock.ExpectExec(regexp.QuoteMeta(`
	UPDATE tasks
		SET title = ?,
			description = ?,
			status = ?,
			assignee_id = ?,
			completed_at = CASE
				WHEN ? = 'done' AND ? <> 'done' THEN CURRENT_TIMESTAMP
				WHEN ? <> 'done' THEN NULL
				ELSE completed_at
			END
		WHERE id = ?
	`)).
		WithArgs("Task", "Desc", "done", nil, "done", "todo", "done", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO task_history(task_id, changed_by, field_name, old_value, new_value)
		VALUES (?, ?, ?, ?, ?)
	`)).
		WithArgs(int64(1), int64(9), "status", "todo", "done").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.Update(context.Background(), 9, models.Task{
		ID:          1,
		Title:       "Task",
		Description: "Desc",
		Status:      "done",
		TeamID:      2,
		CreatedBy:   3,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
