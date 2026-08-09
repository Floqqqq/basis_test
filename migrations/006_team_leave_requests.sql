CREATE TABLE team_leave_requests (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMPTZ,
    resolved_by BIGINT REFERENCES users(id),
    CONSTRAINT team_leave_requests_status_check
        CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE UNIQUE INDEX idx_team_leave_requests_one_pending
    ON team_leave_requests(team_id, user_id)
    WHERE status = 'pending';

CREATE INDEX idx_team_leave_requests_pending_team
    ON team_leave_requests(team_id, requested_at, id)
    WHERE status = 'pending';
