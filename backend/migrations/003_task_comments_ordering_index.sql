CREATE INDEX idx_task_comments_task_created_at
    ON task_comments(task_id, created_at, id);
