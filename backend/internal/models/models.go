package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Team struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamWithRole struct {
	Team
	Role string `json:"role"`
}

type TeamMember struct {
	ID       int64     `json:"id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type TeamLeaveRequest struct {
	ID          int64      `json:"id"`
	TeamID      int64      `json:"team_id"`
	UserID      int64      `json:"user_id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy  *int64     `json:"resolved_by,omitempty"`
}

type ActivityEvent struct {
	ID           int64           `json:"id"`
	TeamID       int64           `json:"team_id"`
	ActorID      int64           `json:"actor_id"`
	ActorEmail   string          `json:"actor_email"`
	EventType    string          `json:"event_type"`
	EntityType   string          `json:"entity_type"`
	EntityID     int64           `json:"entity_id"`
	Payload      json.RawMessage `json:"payload"`
	Message      string          `json:"message"`
	CreatedAt    time.Time       `json:"created_at"`
	RelatedEmail string          `json:"-"`
}

type Task struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	AssigneeID  *int64     `json:"assignee_id"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	TeamID      int64      `json:"team_id"`
	CreatedBy   int64      `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TaskHistory struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	ChangedBy int64     `json:"changed_by"`
	FieldName string    `json:"field_name"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskComment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskNotificationData struct {
	Title      string
	CreatorID  int64
	AssigneeID *int64
}
