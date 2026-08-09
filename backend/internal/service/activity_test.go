package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"task-manager/internal/events"
	"task-manager/internal/models"
)

type fakeActivityRepository struct {
	items  []models.ActivityEvent
	err    error
	limit  int
	offset int
}

func (r *fakeActivityRepository) List(ctx context.Context, teamID int64, limit, offset int) ([]models.ActivityEvent, error) {
	r.limit = limit
	r.offset = offset
	return r.items, r.err
}

func TestActivityServiceMembershipPermissions(t *testing.T) {
	repo := &fakeActivityRepository{}
	service := NewActivityService(repo, fakeTeamAccess{members: map[int64]bool{1: true}})

	if _, err := service.List(context.Background(), 1, 5, 20, 0); err != nil {
		t.Fatalf("member List() error = %v", err)
	}
	if repo.limit != 20 || repo.offset != 0 {
		t.Fatalf("pagination = %d/%d, want 20/0", repo.limit, repo.offset)
	}
	if _, err := service.List(context.Background(), 2, 5, 20, 0); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider List() error = %v, want ErrForbidden", err)
	}
}

func TestActivityMessageRepresentation(t *testing.T) {
	tests := []struct {
		name         string
		eventType    events.EventType
		payload      any
		relatedEmail string
		want         string
	}{
		{
			name:      "task created",
			eventType: events.TaskCreated,
			payload:   events.TaskPayload{TaskID: 10, TeamID: 5, Title: "Backend"},
			want:      "daniil@example.com создал задачу «Backend»",
		},
		{
			name:      "status changed",
			eventType: events.TaskStatusChanged,
			payload:   events.TaskPayload{TaskID: 10, TeamID: 5, Title: "Backend", Status: "done"},
			want:      "daniil@example.com изменил статус задачи «Backend» на «Готово»",
		},
		{
			name:         "member added",
			eventType:    events.TeamMemberAdded,
			payload:      events.TeamMemberPayload{TeamID: 5, UserID: 2, Role: "member"},
			relatedEmail: "ivan@example.com",
			want:         "daniil@example.com добавил ivan@example.com в команду",
		},
		{
			name:         "task assigned",
			eventType:    events.TaskAssigned,
			payload:      events.TaskPayload{TaskID: 10, TeamID: 5, Title: "Backend", AssigneeID: int64Ptr(2)},
			relatedEmail: "alex@example.com",
			want:         "daniil@example.com назначил alex@example.com на задачу «Backend»",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			got := activityMessage(models.ActivityEvent{
				ActorEmail:   "daniil@example.com",
				EventType:    string(tt.eventType),
				EntityID:     10,
				Payload:      payload,
				RelatedEmail: tt.relatedEmail,
				CreatedAt:    time.Now(),
			})
			if got != tt.want {
				t.Fatalf("activityMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
