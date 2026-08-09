CREATE TABLE activity_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) NOT NULL UNIQUE,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    actor_id BIGINT NOT NULL REFERENCES users(id),
    event_type VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id BIGINT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_activity_events_team_feed
    ON activity_events(team_id, created_at DESC, id DESC);
