package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

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

type TaskRepository interface {
	Create(ctx context.Context, task models.Task) (int64, error)
	List(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int) ([]models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
	Update(ctx context.Context, userID int64, task models.Task) error
	History(ctx context.Context, taskID int64) ([]models.TaskHistory, error)
}

type TaskCache interface {
	GetList(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int) ([]byte, bool, error)
	SetList(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int, body []byte) error
	InvalidateTeam(ctx context.Context, teamID int64) error
}

type TaskUpdate struct {
	Title       *string
	Description *string
	Status      *string
	AssigneeID  *int64
}

type TaskService struct {
	tasks     TaskRepository
	policy    *TaskPolicy
	taskCache TaskCache
}

func NewTaskService(tasks TaskRepository, policy *TaskPolicy, taskCache TaskCache) *TaskService {
	return &TaskService{
		tasks:     tasks,
		policy:    policy,
		taskCache: taskCache,
	}
}

func (s *TaskService) Create(ctx context.Context, userID int64, task models.Task) (int64, error) {
	isMember, err := s.policy.IsTeamMember(ctx, task.TeamID, userID)
	if err != nil {
		return 0, fmt.Errorf("check team membership: %w", err)
	}
	if !isMember {
		return 0, ErrForbidden
	}

	assigneeValid, err := s.policy.IsAssigneeValidForTeam(ctx, task.TeamID, task.AssigneeID)
	if err != nil {
		return 0, fmt.Errorf("check assignee: %w", err)
	}
	if !assigneeValid {
		return 0, fmt.Errorf("%w: assignee is not team member", ErrInvalidInput)
	}

	task.CreatedBy = userID
	taskID, err := s.tasks.Create(ctx, task)
	if err != nil {
		return 0, fmt.Errorf("create task: %w", err)
	}

	s.invalidateTeamCache(ctx, task.TeamID)
	return taskID, nil
}

func (s *TaskService) GetByID(ctx context.Context, taskID int64) (*models.Task, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return task, nil
}

func (s *TaskService) List(ctx context.Context, userID, teamID int64, status string, assigneeID *int64, limit, offset int) ([]models.Task, error) {
	isMember, err := s.policy.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check team membership: %w", err)
	}
	if !isMember {
		return nil, ErrForbidden
	}

	assigneeValid, err := s.policy.IsAssigneeValidForTeam(ctx, teamID, assigneeID)
	if err != nil {
		return nil, fmt.Errorf("check assignee: %w", err)
	}
	if !assigneeValid {
		return nil, fmt.Errorf("%w: assignee is not team member", ErrInvalidInput)
	}

	if s.taskCache != nil {
		cached, ok, err := s.taskCache.GetList(ctx, teamID, status, assigneeID, limit, offset)
		if err != nil {
			slog.Warn("task list cache get failed", "team_id", teamID, "error", err)
		}
		if ok {
			var tasks []models.Task
			if err := json.Unmarshal(cached, &tasks); err == nil {
				return tasks, nil
			}
			slog.Warn("task list cache decode failed", "team_id", teamID)
		}
	}

	tasks, err := s.tasks.List(ctx, teamID, status, assigneeID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	if s.taskCache != nil {
		body, err := json.Marshal(tasks)
		if err != nil {
			slog.Warn("task list cache encode failed", "team_id", teamID, "error", err)
		} else if err := s.taskCache.SetList(ctx, teamID, status, assigneeID, limit, offset, body); err != nil {
			slog.Warn("task list cache set failed", "team_id", teamID, "error", err)
		}
	}

	return tasks, nil
}

func (s *TaskService) Update(ctx context.Context, userID, taskID int64, update TaskUpdate) error {
	task, err := s.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	canUpdate, err := s.policy.CanUpdateTask(ctx, task, userID)
	if err != nil {
		return fmt.Errorf("check task permissions: %w", err)
	}
	if !canUpdate {
		return ErrForbidden
	}

	updated := *task
	if update.Title != nil {
		updated.Title = *update.Title
	}
	if update.Description != nil {
		updated.Description = *update.Description
	}
	if update.Status != nil {
		updated.Status = *update.Status
	}
	if update.AssigneeID != nil {
		assigneeValid, err := s.policy.IsAssigneeValidForTeam(ctx, task.TeamID, update.AssigneeID)
		if err != nil {
			return fmt.Errorf("check assignee: %w", err)
		}
		if !assigneeValid {
			return fmt.Errorf("%w: assignee is not team member", ErrInvalidInput)
		}
		updated.AssigneeID = update.AssigneeID
	}

	if err := s.tasks.Update(ctx, userID, updated); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	s.invalidateTeamCache(ctx, task.TeamID)
	return nil
}

func (s *TaskService) History(ctx context.Context, userID, taskID int64) ([]models.TaskHistory, error) {
	task, err := s.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	canView, err := s.policy.CanViewTaskHistory(ctx, task, userID)
	if err != nil {
		return nil, fmt.Errorf("check task history permissions: %w", err)
	}
	if !canView {
		return nil, ErrForbidden
	}

	history, err := s.tasks.History(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task history: %w", err)
	}
	return history, nil
}

func (s *TaskService) invalidateTeamCache(ctx context.Context, teamID int64) {
	if s.taskCache == nil {
		return
	}
	if err := s.taskCache.InvalidateTeam(ctx, teamID); err != nil {
		slog.Warn("task cache invalidation failed", "team_id", teamID, "error", err)
	}
}
