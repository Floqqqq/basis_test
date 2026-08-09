package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"task-manager/internal/events"
)

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type OutboxEntry struct {
	ID            int64
	EventID       string
	EventType     events.EventType
	Payload       json.RawMessage
	Status        string
	Attempts      int
	NextAttemptAt time.Time
	CreatedAt     time.Time
	ProcessedAt   *time.Time
	LastError     *string
}

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Create(ctx context.Context, q DBTX, event events.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal outbox event: %w", err)
	}

	_, err = q.ExecContext(ctx, `
		INSERT INTO notification_outbox(event_id, event_type, payload)
		VALUES ($1, $2, $3)
	`, event.EventID, event.EventType, payload)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

func (r *OutboxRepository) FetchPending(ctx context.Context, limit int) ([]OutboxEntry, error) {
	if limit <= 0 {
		return []OutboxEntry{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		WITH candidates AS (
			SELECT id
			FROM notification_outbox
			WHERE status IN ('pending', 'failed')
			  AND next_attempt_at <= CURRENT_TIMESTAMP
			ORDER BY next_attempt_at, created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE notification_outbox o
		SET status = 'processing'
		FROM candidates c
		WHERE o.id = c.id
		RETURNING o.id, o.event_id, o.event_type, o.payload, o.status,
		          o.attempts, o.next_attempt_at, o.created_at,
		          o.processed_at, o.last_error
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending outbox events: %w", err)
	}
	defer rows.Close()

	entries := make([]OutboxEntry, 0)
	for rows.Next() {
		var entry OutboxEntry
		var processedAt sql.NullTime
		var lastError sql.NullString
		if err := rows.Scan(
			&entry.ID,
			&entry.EventID,
			&entry.EventType,
			&entry.Payload,
			&entry.Status,
			&entry.Attempts,
			&entry.NextAttemptAt,
			&entry.CreatedAt,
			&processedAt,
			&lastError,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		if processedAt.Valid {
			entry.ProcessedAt = &processedAt.Time
		}
		if lastError.Valid {
			entry.LastError = &lastError.String
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", err)
	}
	return entries, nil
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE notification_outbox
		SET status = 'sent', processed_at = CURRENT_TIMESTAMP, last_error = NULL
		WHERE id = $1 AND status = 'processing'
	`, id)
	if err != nil {
		return fmt.Errorf("mark outbox event processed: %w", err)
	}
	return requireAffectedRow(result, "mark outbox event processed")
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id int64, eventErr error, retryAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE notification_outbox
		SET status = 'failed', attempts = attempts + 1,
			next_attempt_at = $2, last_error = $3
		WHERE id = $1 AND status = 'processing'
	`, id, retryAt, eventErr.Error())
	if err != nil {
		return fmt.Errorf("mark outbox event failed: %w", err)
	}
	return requireAffectedRow(result, "mark outbox event failed")
}

func requireAffectedRow(result sql.Result, operation string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows affected: %w", operation, err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
