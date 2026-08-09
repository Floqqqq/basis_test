DROP INDEX IF EXISTS idx_notification_outbox_pending;

CREATE INDEX idx_notification_outbox_pending
    ON notification_outbox(next_attempt_at, created_at, id)
    WHERE status IN ('pending', 'processing');
