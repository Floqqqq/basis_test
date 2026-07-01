package service

import (
	"context"
	"database/sql"
	"errors"

	"task-manager/internal/models"
)

type TeamAccessRepository interface {
	GetUserRole(ctx context.Context, teamID, userID int64) (string, error)
	IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error)
}

type TaskPolicy struct {
	teams TeamAccessRepository
}

func NewTaskPolicy(teams TeamAccessRepository) *TaskPolicy {
	return &TaskPolicy{teams: teams}
}

func (p *TaskPolicy) IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error) {
	return p.teams.IsTeamMember(ctx, teamID, userID)
}

func (p *TaskPolicy) IsAssigneeValidForTeam(ctx context.Context, teamID int64, assigneeID *int64) (bool, error) {
	if assigneeID == nil {
		return true, nil
	}
	return p.teams.IsTeamMember(ctx, teamID, *assigneeID)
}

func (p *TaskPolicy) CanUpdateTask(ctx context.Context, task *models.Task, userID int64) (bool, error) {
	return p.canAccessTask(ctx, task, userID)
}

func (p *TaskPolicy) CanViewTaskHistory(ctx context.Context, task *models.Task, userID int64) (bool, error) {
	return p.canAccessTask(ctx, task, userID)
}

func (p *TaskPolicy) canAccessTask(ctx context.Context, task *models.Task, userID int64) (bool, error) {
	role, err := p.teams.GetUserRole(ctx, task.TeamID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if role == "owner" || role == "admin" {
		return true, nil
	}
	if task.CreatedBy == userID {
		return true, nil
	}
	if task.AssigneeID != nil && *task.AssigneeID == userID {
		return true, nil
	}

	return false, nil
}
