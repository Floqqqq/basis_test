package repository

import (
	"context"
	"database/sql"
	"fmt"

	"task-manager/internal/models"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) GetTask(ctx context.Context, taskID int64) (models.TaskNotificationData, error) {
	var data models.TaskNotificationData
	var assigneeID sql.NullInt64
	if err := r.db.QueryRowContext(ctx, `
		SELECT title, created_by, assignee_id
		FROM tasks
		WHERE id = $1
	`, taskID).Scan(&data.Title, &data.CreatorID, &assigneeID); err != nil {
		return models.TaskNotificationData{}, fmt.Errorf("get task notification data: %w", err)
	}
	if assigneeID.Valid {
		data.AssigneeID = &assigneeID.Int64
	}
	return data, nil
}

func (r *NotificationRepository) GetTeamName(ctx context.Context, teamID int64) (string, error) {
	var name string
	if err := r.db.QueryRowContext(ctx, `SELECT name FROM teams WHERE id = $1`, teamID).Scan(&name); err != nil {
		return "", fmt.Errorf("get team notification data: %w", err)
	}
	return name, nil
}

func (r *NotificationRepository) GetUserEmail(ctx context.Context, userID int64) (string, error) {
	var email string
	if err := r.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		return "", fmt.Errorf("get notification recipient: %w", err)
	}
	return email, nil
}
