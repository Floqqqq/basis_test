package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type TeamStatsReport struct {
	TeamID             int64  `json:"team_id"`
	Name               string `json:"name"`
	MembersCount       int64  `json:"members_count"`
	DoneTasksLast7Days int64  `json:"done_tasks_last_7_days"`
}

type TopUserReport struct {
	TeamID     int64 `json:"team_id"`
	UserID     int64 `json:"user_id"`
	TasksCount int64 `json:"tasks_count"`
	Rank       int64 `json:"rank"`
}

type InvalidAssigneeReport struct {
	TaskID     int64  `json:"task_id"`
	Title      string `json:"title"`
	TeamID     int64  `json:"team_id"`
	AssigneeID int64  `json:"assignee_id"`
}

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) TeamStats(ctx context.Context, userID int64) ([]TeamStatsReport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			t.id,
			t.name,
			COUNT(DISTINCT tm.user_id) AS members_count,
			COUNT(DISTINCT CASE
				WHEN tasks.status = 'done'
				AND tasks.completed_at >= NOW() - INTERVAL 7 DAY
				THEN tasks.id
			END) AS done_tasks_last_7_days
		FROM teams t
		JOIN team_members current_user_tm
			ON current_user_tm.team_id = t.id
			AND current_user_tm.user_id = ?
		LEFT JOIN team_members tm ON tm.team_id = t.id
		LEFT JOIN tasks ON tasks.team_id = t.id
		GROUP BY t.id, t.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query team stats report: %w", err)
	}
	defer rows.Close()

	result := make([]TeamStatsReport, 0)
	for rows.Next() {
		var item TeamStatsReport
		if err := rows.Scan(&item.TeamID, &item.Name, &item.MembersCount, &item.DoneTasksLast7Days); err != nil {
			return nil, fmt.Errorf("scan team stats report: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team stats report: %w", err)
	}

	return result, nil
}

func (r *ReportRepository) TopUsers(ctx context.Context, userID int64) ([]TopUserReport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ranked.team_id,
			ranked.user_id,
			ranked.tasks_count,
			ranked.rn
		FROM (
			SELECT
				t.team_id,
				t.created_by AS user_id,
				COUNT(*) AS tasks_count,
				ROW_NUMBER() OVER (
					PARTITION BY t.team_id
					ORDER BY COUNT(*) DESC
				) AS rn
			FROM tasks t
			WHERE t.created_at >= DATE_FORMAT(CURRENT_DATE, '%Y-%m-01')
			GROUP BY t.team_id, t.created_by
		) ranked
		JOIN team_members current_user_tm
			ON current_user_tm.team_id = ranked.team_id
			AND current_user_tm.user_id = ?
		WHERE ranked.rn <= 3
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query top users report: %w", err)
	}
	defer rows.Close()

	result := make([]TopUserReport, 0)
	for rows.Next() {
		var item TopUserReport
		if err := rows.Scan(&item.TeamID, &item.UserID, &item.TasksCount, &item.Rank); err != nil {
			return nil, fmt.Errorf("scan top users report: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top users report: %w", err)
	}

	return result, nil
}

func (r *ReportRepository) InvalidAssignees(ctx context.Context, userID int64) ([]InvalidAssigneeReport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			tasks.id,
			tasks.title,
			tasks.team_id,
			tasks.assignee_id
		FROM tasks
		JOIN team_members current_user_tm
			ON current_user_tm.team_id = tasks.team_id
			AND current_user_tm.user_id = ?
		LEFT JOIN team_members tm
			ON tm.team_id = tasks.team_id
			AND tm.user_id = tasks.assignee_id
		WHERE tasks.assignee_id IS NOT NULL
		  AND tm.user_id IS NULL
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query invalid assignees report: %w", err)
	}
	defer rows.Close()

	result := make([]InvalidAssigneeReport, 0)
	for rows.Next() {
		var item InvalidAssigneeReport
		if err := rows.Scan(&item.TaskID, &item.Title, &item.TeamID, &item.AssigneeID); err != nil {
			return nil, fmt.Errorf("scan invalid assignees report: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invalid assignees report: %w", err)
	}

	return result, nil
}
