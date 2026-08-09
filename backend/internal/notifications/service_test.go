package notifications

import (
	"context"
	"testing"

	"task-manager/internal/events"
	"task-manager/internal/models"
)

type fakeDataRepository struct {
	task     models.TaskNotificationData
	teamName string
	emails   map[int64]string
}

func (r fakeDataRepository) GetTask(context.Context, int64) (models.TaskNotificationData, error) {
	return r.task, nil
}

func (r fakeDataRepository) GetTeamName(context.Context, int64) (string, error) {
	return r.teamName, nil
}

func (r fakeDataRepository) GetUserEmail(_ context.Context, userID int64) (string, error) {
	return r.emails[userID], nil
}

func TestNotificationBuilderTaskCreatedExcludesActor(t *testing.T) {
	assigneeID := int64(2)
	builder := NewNotificationBuilder(fakeDataRepository{
		task:   models.TaskNotificationData{Title: "Check API", CreatorID: 1, AssigneeID: &assigneeID},
		emails: map[int64]string{2: "assignee@example.com"},
	})
	event := events.New(events.TaskCreated, 2, events.TaskPayload{TaskID: 10, CreatorID: 1, AssigneeID: &assigneeID})

	message, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if message != nil {
		t.Fatalf("message = %+v, want nil when actor is assignee", message)
	}
}

func TestNotificationBuilderAssignmentAndTemplate(t *testing.T) {
	assigneeID := int64(2)
	builder := NewNotificationBuilder(fakeDataRepository{
		task:   models.TaskNotificationData{Title: "Check API", CreatorID: 1, AssigneeID: &assigneeID},
		emails: map[int64]string{2: "assignee@example.com"},
	})
	event := events.New(events.TaskAssigned, 1, events.TaskPayload{TaskID: 10, CreatorID: 1, AssigneeID: &assigneeID})

	message, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(message.To) != 1 || message.To[0] != "assignee@example.com" {
		t.Fatalf("recipients = %v, want assignee", message.To)
	}
	if message.Subject != "You were assigned to: Check API" {
		t.Fatalf("subject = %q", message.Subject)
	}
}

func TestNotificationBuilderUnassignmentDoesNotUseCurrentAssignee(t *testing.T) {
	currentAssigneeID := int64(2)
	builder := NewNotificationBuilder(fakeDataRepository{
		task:   models.TaskNotificationData{Title: "Check API", CreatorID: 1, AssigneeID: &currentAssigneeID},
		emails: map[int64]string{2: "assignee@example.com"},
	})
	event := events.New(events.TaskAssigned, 1, events.TaskPayload{
		TaskID:             10,
		CreatorID:          1,
		AssigneeID:         nil,
		RecipientsCaptured: true,
	})

	message, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if message != nil {
		t.Fatalf("message = %+v, want nil for unassignment", message)
	}
}

func TestNotificationBuilderUpdatedDeduplicatesAndExcludesActor(t *testing.T) {
	assigneeID := int64(2)
	builder := NewNotificationBuilder(fakeDataRepository{
		task: models.TaskNotificationData{Title: "Check API", CreatorID: 1, AssigneeID: &assigneeID},
		emails: map[int64]string{
			1: "same@example.com",
			2: "same@example.com",
		},
	})
	event := events.New(events.TaskUpdated, 3, events.TaskPayload{TaskID: 10, CreatorID: 1, AssigneeID: &assigneeID})

	message, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(message.To) != 1 || message.To[0] != "same@example.com" {
		t.Fatalf("recipients = %v, want one deduplicated email", message.To)
	}

	event.ActorID = 1
	builder = NewNotificationBuilder(fakeDataRepository{
		task:   models.TaskNotificationData{Title: "Check API", CreatorID: 1, AssigneeID: &assigneeID},
		emails: map[int64]string{1: "creator@example.com", 2: "assignee@example.com"},
	})
	message, err = builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("Build() with actor error = %v", err)
	}
	if len(message.To) != 1 || message.To[0] != "assignee@example.com" {
		t.Fatalf("recipients = %v, want actor excluded", message.To)
	}
}

func TestNotificationBuilderTeamMemberAdded(t *testing.T) {
	builder := NewNotificationBuilder(fakeDataRepository{
		teamName: "Backend",
		emails:   map[int64]string{2: "member@example.com"},
	})
	event := events.New(events.TeamMemberAdded, 1, events.TeamMemberPayload{TeamID: 5, UserID: 2, Role: "member"})

	message, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if message.Subject != "You were added to team: Backend" || len(message.To) != 1 || message.To[0] != "member@example.com" {
		t.Fatalf("message = %+v", message)
	}
}

type fakeEmailSender struct {
	messages []EmailMessage
	err      error
}

func (s *fakeEmailSender) Send(_ context.Context, message EmailMessage) error {
	s.messages = append(s.messages, message)
	return s.err
}
