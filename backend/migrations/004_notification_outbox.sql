CREATE TABLE notification_outbox (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMPTZ,
    last_error TEXT,
    CONSTRAINT notification_outbox_status_check
        CHECK (status IN ('pending', 'processing', 'sent', 'failed'))
);

CREATE INDEX idx_notification_outbox_pending
    ON notification_outbox(next_attempt_at, created_at, id)
    WHERE status IN ('pending', 'failed');
