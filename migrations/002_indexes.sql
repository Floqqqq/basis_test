CREATE INDEX idx_teams_created_by ON teams(created_by);

CREATE INDEX idx_team_members_team_user ON team_members(team_id, user_id);
CREATE INDEX idx_team_members_team_role ON team_members(team_id, role);

CREATE INDEX idx_tasks_team_created_at ON tasks(team_id, created_at DESC, id DESC);
CREATE INDEX idx_tasks_team_status_created_at ON tasks(team_id, status, created_at DESC, id DESC);
CREATE INDEX idx_tasks_team_assignee_created_at ON tasks(team_id, assignee_id, created_at DESC, id DESC);
CREATE INDEX idx_tasks_team_status_assignee_created_at ON tasks(team_id, status, assignee_id, created_at DESC, id DESC);
CREATE INDEX idx_tasks_team_created_by_created_at ON tasks(team_id, created_by, created_at DESC);
CREATE INDEX idx_tasks_done_completed_at ON tasks(team_id, completed_at DESC) WHERE status = 'done';
CREATE INDEX idx_tasks_invalid_assignee_lookup ON tasks(team_id, assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX idx_tasks_created_at_team_created_by ON tasks(created_at, team_id, created_by);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
CREATE INDEX idx_tasks_created_by ON tasks(created_by);

CREATE INDEX idx_task_history_task_created_at ON task_history(task_id, created_at DESC, id DESC);
CREATE INDEX idx_task_history_changed_by ON task_history(changed_by);

CREATE INDEX idx_task_comments_task_id ON task_comments(task_id);
CREATE INDEX idx_task_comments_user_id ON task_comments(user_id);
