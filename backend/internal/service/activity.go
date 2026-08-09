package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"task-manager/internal/events"
	"task-manager/internal/models"
)

type ActivityRepository interface {
	List(ctx context.Context, teamID int64, limit, offset int) ([]models.ActivityEvent, error)
}

type ActivityService struct {
	activity ActivityRepository
	teams    TeamAccessRepository
}

func NewActivityService(activity ActivityRepository, teams TeamAccessRepository) *ActivityService {
	return &ActivityService{activity: activity, teams: teams}
}

func (s *ActivityService) List(ctx context.Context, userID, teamID int64, limit, offset int) (result []models.ActivityEvent, err error) {
	ctx, span := startSpan(ctx, "ActivityService.List",
		attribute.Int64("user.id", userID),
		attribute.Int64("team.id", teamID),
	)
	defer func() { finishSpan(span, err) }()

	isMember, err := s.teams.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check activity team membership: %w", err)
	}
	if !isMember {
		return nil, ErrForbidden
	}

	result, err = s.activity.List(ctx, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list team activity: %w", err)
	}
	for i := range result {
		result[i].Message = activityMessage(result[i])
	}
	return result, nil
}

func activityMessage(activity models.ActivityEvent) string {
	actor := activity.ActorEmail
	switch events.EventType(activity.EventType) {
	case events.TaskCreated, events.TaskUpdated, events.TaskAssigned, events.TaskStatusChanged:
		var payload events.TaskPayload
		if err := json.Unmarshal(activity.Payload, &payload); err != nil {
			return fmt.Sprintf("%s выполнил действие с задачей #%d", actor, activity.EntityID)
		}
		title := payload.Title
		if title == "" {
			title = fmt.Sprintf("#%d", activity.EntityID)
		}
		switch events.EventType(activity.EventType) {
		case events.TaskCreated:
			return fmt.Sprintf("%s создал задачу «%s»", actor, title)
		case events.TaskUpdated:
			return fmt.Sprintf("%s обновил задачу «%s»", actor, title)
		case events.TaskStatusChanged:
			return fmt.Sprintf("%s изменил статус задачи «%s» на «%s»", actor, title, activityStatusLabel(payload.Status))
		case events.TaskAssigned:
			if payload.AssigneeID == nil {
				return fmt.Sprintf("%s снял исполнителя с задачи «%s»", actor, title)
			}
			assignee := activity.RelatedEmail
			if assignee == "" {
				assignee = fmt.Sprintf("пользователя #%d", *payload.AssigneeID)
			}
			return fmt.Sprintf("%s назначил %s на задачу «%s»", actor, assignee, title)
		}
	case events.TeamMemberAdded:
		member := activity.RelatedEmail
		if member == "" {
			member = fmt.Sprintf("пользователя #%d", activity.EntityID)
		}
		return fmt.Sprintf("%s добавил %s в команду", actor, member)
	}
	return fmt.Sprintf("%s выполнил действие %s", actor, activity.EventType)
}

func activityStatusLabel(status string) string {
	switch status {
	case "todo":
		return "К выполнению"
	case "in_progress":
		return "В работе"
	case "done":
		return "Готово"
	default:
		return status
	}
}
