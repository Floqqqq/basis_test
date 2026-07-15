package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"task-manager/internal/models"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task models.Task) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
			INSERT INTO tasks(title, description, status, assignee_id, completed_at, team_id, created_by)
			VALUES (?, ?, ?, ?, CASE WHEN ? = 'done' THEN CURRENT_TIMESTAMP ELSE NULL END, ?, ?)
		`,
		task.Title,
		task.Description,
		task.Status,
		task.AssigneeID,
		task.Status,
		task.TeamID,
		task.CreatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("create task: %w", err)
	}

	taskID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get created task id: %w", err)
	}

	return taskID, nil
}

func (r *TaskRepository) List(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int) ([]models.Task, error) {
	query := `
			SELECT id, title, description, status, assignee_id, completed_at, team_id, created_by, created_at, updated_at
			FROM tasks
			WHERE team_id = ?
	`

	args := []any{teamID}

	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}

	if assigneeID != nil {
		query += ` AND assignee_id = ?`
		args = append(args, *assigneeID)
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]models.Task, 0)

	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		tasks = append(tasks, *t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepository) GetByID(
	ctx context.Context,
	id int64,
) (*models.Task, error) {
	return r.getByIDWithQuerier(ctx, r.db, id)
}

type taskScanner interface {
	Scan(dest ...any) error
}

type queryer interface {
	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

func (r *TaskRepository) getByIDWithQuerier(
	ctx context.Context,
	q queryer,
	id int64,
) (*models.Task, error) {
	row := q.QueryRowContext(ctx, `
		SELECT id, title, description, status, assignee_id,
		       completed_at, team_id, created_by, created_at, updated_at
		FROM tasks
		WHERE id = ?
	`, id)

	return scanTask(row)
}

func scanTask(scanner taskScanner) (*models.Task, error) {
	var t models.Task
	var description sql.NullString
	var assigneeID sql.NullInt64
	var completedAt sql.NullTime

	if err := scanner.Scan(
		&t.ID,
		&t.Title,
		&description,
		&t.Status,
		&assigneeID,
		&completedAt,
		&t.TeamID,
		&t.CreatedBy,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if description.Valid {
		t.Description = description.String
	}
	if assigneeID.Valid {
		t.AssigneeID = &assigneeID.Int64
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}

	return &t, nil
}

func (r *TaskRepository) Update(ctx context.Context, userID int64, task models.Task) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update task tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
				err = fmt.Errorf("rollback update task tx: %w", rollbackErr)
			}
		}
	}()

	oldTask, err := r.getByIDWithQuerier(ctx, tx, task.ID)
	if err != nil {
		return fmt.Errorf("get task before update: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
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
	`,
		task.Title,
		task.Description,
		task.Status,
		task.AssigneeID,
		task.Status,
		oldTask.Status,
		task.Status,
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if oldTask.Title != task.Title {
		if err := insertTaskHistory(ctx, tx, task.ID, userID, "title", oldTask.Title, task.Title); err != nil {
			return err
		}
	}

	if oldTask.Description != task.Description {
		if err := insertTaskHistory(ctx, tx, task.ID, userID, "description", oldTask.Description, task.Description); err != nil {
			return err
		}
	}

	if oldTask.Status != task.Status {
		if err := insertTaskHistory(ctx, tx, task.ID, userID, "status", oldTask.Status, task.Status); err != nil {
			return err
		}
	}

	if nullableInt64Value(oldTask.AssigneeID) != nullableInt64Value(task.AssigneeID) {
		if err := insertTaskHistory(ctx, tx, task.ID, userID, "assignee_id", nullableInt64Value(oldTask.AssigneeID), nullableInt64Value(task.AssigneeID)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update task tx: %w", err)
	}
	committed = true

	return nil
}

func insertTaskHistory(ctx context.Context, tx *sql.Tx, taskID, userID int64, fieldName, oldValue, newValue string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO task_history(task_id, changed_by, field_name, old_value, new_value)
		VALUES (?, ?, ?, ?, ?)
	`, taskID, userID, fieldName, oldValue, newValue)
	if err != nil {
		return fmt.Errorf("insert task history for %s: %w", fieldName, err)
	}
	return nil
}

func nullableInt64Value(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func (r *TaskRepository) History(ctx context.Context, taskID int64) ([]models.TaskHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, task_id, changed_by, field_name, old_value, new_value, created_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY created_at DESC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task history: %w", err)
	}
	defer rows.Close()

	history := make([]models.TaskHistory, 0)

	for rows.Next() {
		var h models.TaskHistory
		var oldValue sql.NullString
		var newValue sql.NullString

		if err := rows.Scan(
			&h.ID,
			&h.TaskID,
			&h.ChangedBy,
			&h.FieldName,
			&oldValue,
			&newValue,
			&h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task history: %w", err)
		}
		if oldValue.Valid {
			h.OldValue = oldValue.String
		}
		if newValue.Valid {
			h.NewValue = newValue.String
		}

		history = append(history, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task history: %w", err)
	}

	return history, nil
}
