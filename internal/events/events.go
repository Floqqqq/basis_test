package events

import (
	"crypto/rand"
	"encoding/hex"
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

type Event struct {
	Metadata
	Payload any `json:"payload"`
}

type TaskPayload struct {
	TaskID             int64    `json:"task_id"`
	TeamID             int64    `json:"team_id"`
	Status             string   `json:"status,omitempty"`
	PreviousStatus     string   `json:"previous_status,omitempty"`
	AssigneeID         *int64   `json:"assignee_id"`
	PreviousAssigneeID *int64   `json:"previous_assignee_id"`
	ChangedFields      []string `json:"changed_fields,omitempty"`
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
