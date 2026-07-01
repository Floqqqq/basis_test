CREATE INDEX idx_teams_created_by ON teams(created_by);
CREATE INDEX idx_tasks_team_created_at ON tasks(team_id, created_at);
CREATE INDEX idx_tasks_created_at_team_created_by ON tasks(created_at, team_id, created_by);
CREATE INDEX idx_task_history_changed_by ON task_history(changed_by);
CREATE INDEX idx_task_comments_task_id ON task_comments(task_id);
CREATE INDEX idx_task_comments_user_id ON task_comments(user_id);
