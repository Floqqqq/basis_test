package notifications

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"task-manager/internal/events"
	"task-manager/internal/models"
)

var tracer = otel.Tracer("task-manager/internal/notifications")

type DataRepository interface {
	GetTask(ctx context.Context, taskID int64) (models.TaskNotificationData, error)
	GetTeamName(ctx context.Context, teamID int64) (string, error)
	GetUserEmail(ctx context.Context, userID int64) (string, error)
}

type NotificationBuilder struct {
	data DataRepository
}

func NewNotificationBuilder(data DataRepository) *NotificationBuilder {
	return &NotificationBuilder{data: data}
}

func (b *NotificationBuilder) Build(ctx context.Context, event events.Event) (*EmailMessage, error) {
	switch payload := event.Payload.(type) {
	case events.TaskPayload:
		return b.buildTaskMessage(ctx, event, payload)
	case events.TeamMemberPayload:
		return b.buildTeamMessage(ctx, event, payload)
	default:
		return nil, fmt.Errorf("unsupported notification payload %T", event.Payload)
	}
}

func (b *NotificationBuilder) buildTaskMessage(ctx context.Context, event events.Event, payload events.TaskPayload) (*EmailMessage, error) {
	task, err := b.data.GetTask(ctx, payload.TaskID)
	if err != nil {
		return nil, err
	}
	if !payload.RecipientsCaptured {
		if payload.CreatorID == 0 {
			payload.CreatorID = task.CreatorID
		}
		if payload.AssigneeID == nil {
			payload.AssigneeID = task.AssigneeID
		}
	}

	var recipientIDs []int64
	switch event.EventType {
	case events.TaskCreated, events.TaskAssigned:
		if payload.AssigneeID != nil {
			recipientIDs = append(recipientIDs, *payload.AssigneeID)
		}
	case events.TaskUpdated, events.TaskStatusChanged:
		recipientIDs = append(recipientIDs, payload.CreatorID)
		if payload.AssigneeID != nil {
			recipientIDs = append(recipientIDs, *payload.AssigneeID)
		}
	default:
		return nil, fmt.Errorf("unsupported task event type %q", event.EventType)
	}

	recipients, err := b.resolveRecipients(ctx, recipientIDs, event.ActorID)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, nil
	}

	subject := "Task updated: " + task.Title
	body := fmt.Sprintf("Task %q was updated.", task.Title)
	switch event.EventType {
	case events.TaskCreated:
		subject = "New task: " + task.Title
		body = fmt.Sprintf("A new task %q was created and assigned to you.", task.Title)
	case events.TaskAssigned:
		subject = "You were assigned to: " + task.Title
		body = fmt.Sprintf("You were assigned to task %q.", task.Title)
	case events.TaskStatusChanged:
		body = fmt.Sprintf("Task %q changed status to %q.", task.Title, payload.Status)
	}

	return &EmailMessage{
		To:        recipients,
		Subject:   subject,
		Body:      body,
		EventID:   event.EventID,
		EventType: event.EventType,
	}, nil
}

func (b *NotificationBuilder) buildTeamMessage(ctx context.Context, event events.Event, payload events.TeamMemberPayload) (*EmailMessage, error) {
	if event.EventType != events.TeamMemberAdded {
		return nil, fmt.Errorf("unsupported team event type %q", event.EventType)
	}
	teamName, err := b.data.GetTeamName(ctx, payload.TeamID)
	if err != nil {
		return nil, err
	}
	recipients, err := b.resolveRecipients(ctx, []int64{payload.UserID}, 0)
	if err != nil {
		return nil, err
	}
	return &EmailMessage{
		To:        recipients,
		Subject:   "You were added to team: " + teamName,
		Body:      fmt.Sprintf("You were added to team %q with role %q.", teamName, payload.Role),
		EventID:   event.EventID,
		EventType: event.EventType,
	}, nil
}

func (b *NotificationBuilder) resolveRecipients(ctx context.Context, userIDs []int64, excludedUserID int64) ([]string, error) {
	seenUsers := make(map[int64]struct{}, len(userIDs))
	seenEmails := make(map[string]struct{}, len(userIDs))
	recipients := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID == 0 || userID == excludedUserID {
			continue
		}
		if _, exists := seenUsers[userID]; exists {
			continue
		}
		seenUsers[userID] = struct{}{}
		email, err := b.data.GetUserEmail(ctx, userID)
		if err != nil {
			return nil, err
		}
		normalized := strings.ToLower(email)
		if _, exists := seenEmails[normalized]; exists {
			continue
		}
		seenEmails[normalized] = struct{}{}
		recipients = append(recipients, email)
	}
	return recipients, nil
}

type NotificationService struct {
	builder *NotificationBuilder
	sender  EmailSender
}

func NewNotificationService(builder *NotificationBuilder, sender EmailSender) *NotificationService {
	return &NotificationService{builder: builder, sender: sender}
}

func (s *NotificationService) ProcessEvent(ctx context.Context, event events.Event) (err error) {
	ctx, span := tracer.Start(ctx, "NotificationService.ProcessEvent")
	defer func() { finishSpan(span, err) }()

	message, err := s.builder.Build(ctx, event)
	if err != nil {
		return err
	}
	if message == nil {
		return nil
	}
	if err := s.sender.Send(ctx, *message); err != nil {
		EmailFailed.WithLabelValues(string(event.EventType)).Inc()
		return fmt.Errorf("send notification email: %w", err)
	}
	EmailSent.WithLabelValues(string(event.EventType)).Inc()
	return nil
}

func finishSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "operation failed")
	}
	span.End()
}
