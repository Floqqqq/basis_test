package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"task-manager/internal/events"
	"task-manager/internal/models"
)

type ActivityRepository struct {
	db *sql.DB
}

func NewActivityRepository(db *sql.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (r *ActivityRepository) Create(ctx context.Context, q DBTX, event events.Event) error {
	teamID, entityType, entityID, err := events.Route(event)
	if err != nil {
		return fmt.Errorf("route activity event: %w", err)
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal activity payload: %w", err)
	}

	_, err = q.ExecContext(ctx, `
		INSERT INTO activity_events(
			event_id, team_id, actor_id, event_type, entity_type, entity_id, payload, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.EventID, teamID, event.ActorID, event.EventType, entityType, entityID, payload, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("insert activity event: %w", err)
	}
	return nil
}

func (r *ActivityRepository) List(ctx context.Context, teamID int64, limit, offset int) ([]models.ActivityEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ae.id, ae.team_id, ae.actor_id, actor.email, ae.event_type,
		       ae.entity_type, ae.entity_id, ae.payload, ae.created_at,
		       related.email
		FROM activity_events ae
		JOIN users actor ON actor.id = ae.actor_id
		LEFT JOIN users related ON related.id = CASE
			WHEN ae.event_type = 'team.member_added'
				THEN NULLIF(ae.payload->>'user_id', '')::BIGINT
			WHEN ae.event_type = 'task.assigned'
				THEN NULLIF(ae.payload->>'assignee_id', '')::BIGINT
			ELSE NULL
		END
		WHERE ae.team_id = $1
		ORDER BY ae.created_at DESC, ae.id DESC
		LIMIT $2 OFFSET $3
	`, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list activity events: %w", err)
	}
	defer rows.Close()

	result := make([]models.ActivityEvent, 0)
	for rows.Next() {
		var activity models.ActivityEvent
		var relatedEmail sql.NullString
		if err := rows.Scan(
			&activity.ID,
			&activity.TeamID,
			&activity.ActorID,
			&activity.ActorEmail,
			&activity.EventType,
			&activity.EntityType,
			&activity.EntityID,
			&activity.Payload,
			&activity.CreatedAt,
			&relatedEmail,
		); err != nil {
			return nil, fmt.Errorf("scan activity event: %w", err)
		}
		if relatedEmail.Valid {
			activity.RelatedEmail = relatedEmail.String
		}
		result = append(result, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity events: %w", err)
	}
	return result, nil
}
