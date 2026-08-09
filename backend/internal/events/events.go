package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type EventType string

const (
	TaskCreated       EventType = "task.created"
	TaskUpdated       EventType = "task.updated"
	TaskAssigned      EventType = "task.assigned"
	TaskStatusChanged EventType = "task.status_changed"
	TeamMemberAdded   EventType = "team.member_added"
)

type Metadata struct {
	EventID    string    `json:"event_id"`
	EventType  EventType `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
	ActorID    int64     `json:"actor_id"`
}

type storedEvent struct {
	Metadata
	Payload json.RawMessage `json:"payload"`
}

func Decode(data []byte) (Event, error) {
	var stored storedEvent
	if err := json.Unmarshal(data, &stored); err != nil {
		return Event{}, fmt.Errorf("decode event envelope: %w", err)
	}

	var payload any
	switch stored.EventType {
	case TaskCreated, TaskUpdated, TaskAssigned, TaskStatusChanged:
		var taskPayload TaskPayload
		if err := json.Unmarshal(stored.Payload, &taskPayload); err != nil {
			return Event{}, fmt.Errorf("decode task event payload: %w", err)
		}
		payload = taskPayload
	case TeamMemberAdded:
		var memberPayload TeamMemberPayload
		if err := json.Unmarshal(stored.Payload, &memberPayload); err != nil {
			return Event{}, fmt.Errorf("decode team member event payload: %w", err)
		}
		payload = memberPayload
	default:
		return Event{}, fmt.Errorf("unsupported event type %q", stored.EventType)
	}

	return Event{Metadata: stored.Metadata, Payload: payload}, nil
}

type Event struct {
	Metadata
	Payload any `json:"payload"`
}

type TaskPayload struct {
	TaskID             int64    `json:"task_id"`
	TeamID             int64    `json:"team_id"`
	Title              string   `json:"title,omitempty"`
	CreatorID          int64    `json:"creator_id,omitempty"`
	Status             string   `json:"status,omitempty"`
	PreviousStatus     string   `json:"previous_status,omitempty"`
	AssigneeID         *int64   `json:"assignee_id"`
	PreviousAssigneeID *int64   `json:"previous_assignee_id"`
	ChangedFields      []string `json:"changed_fields,omitempty"`
	RecipientsCaptured bool     `json:"recipients_captured,omitempty"`
}

func Route(event Event) (teamID int64, entityType string, entityID int64, err error) {
	switch payload := event.Payload.(type) {
	case TaskPayload:
		return payload.TeamID, "task", payload.TaskID, nil
	case TeamMemberPayload:
		return payload.TeamID, "team_member", payload.UserID, nil
	default:
		return 0, "", 0, fmt.Errorf("unsupported payload %T", event.Payload)
	}
}

type TeamMemberPayload struct {
	TeamID int64  `json:"team_id"`
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

func New(eventType EventType, actorID int64, payload any) Event {
	return Event{
		Metadata: Metadata{
			EventID:    newEventID(),
			EventType:  eventType,
			OccurredAt: time.Now().UTC(),
			ActorID:    actorID,
		},
		Payload: payload,
	}
}

func newEventID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic("generate event id: " + err.Error())
	}
	return hex.EncodeToString(value[:])
}
