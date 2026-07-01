CREATE TABLE users (
                       id BIGINT PRIMARY KEY AUTO_INCREMENT,
                       email VARCHAR(255) NOT NULL UNIQUE,
                       password_hash VARCHAR(255) NOT NULL,
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE teams (
                       id BIGINT PRIMARY KEY AUTO_INCREMENT,
                       name VARCHAR(255) NOT NULL,
                       created_by BIGINT NOT NULL,
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE team_members (
                              user_id BIGINT NOT NULL,
                              team_id BIGINT NOT NULL,
                              role ENUM('owner', 'admin', 'member') NOT NULL DEFAULT 'member',
                              created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                              PRIMARY KEY (user_id, team_id),
                              FOREIGN KEY (user_id) REFERENCES users(id),
                              FOREIGN KEY (team_id) REFERENCES teams(id)
);

CREATE TABLE tasks (
                       id BIGINT PRIMARY KEY AUTO_INCREMENT,
	                       title VARCHAR(255) NOT NULL,
	                       description TEXT,
	                       status ENUM('todo', 'in_progress', 'done') NOT NULL DEFAULT 'todo',
	                       assignee_id BIGINT,
	                       completed_at TIMESTAMP NULL,
	                       team_id BIGINT NOT NULL,
	                       created_by BIGINT NOT NULL,
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                       FOREIGN KEY (assignee_id) REFERENCES users(id),
                       FOREIGN KEY (team_id) REFERENCES teams(id),
                       FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE task_history (
                              id BIGINT PRIMARY KEY AUTO_INCREMENT,
                              task_id BIGINT NOT NULL,
                              changed_by BIGINT NOT NULL,
                              field_name VARCHAR(100) NOT NULL,
                              old_value TEXT,
                              new_value TEXT,
                              created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                              FOREIGN KEY (task_id) REFERENCES tasks(id),
                              FOREIGN KEY (changed_by) REFERENCES users(id)
);

CREATE TABLE task_comments (
                               id BIGINT PRIMARY KEY AUTO_INCREMENT,
                               task_id BIGINT NOT NULL,
                               user_id BIGINT NOT NULL,
                               comment TEXT NOT NULL,
                               created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                               FOREIGN KEY (task_id) REFERENCES tasks(id),
                               FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_tasks_team_status ON tasks(team_id, status);
CREATE INDEX idx_tasks_team_status_assignee_created_at ON tasks(team_id, status, assignee_id, created_at);
CREATE INDEX idx_tasks_team_status_updated_at ON tasks(team_id, status, updated_at);
CREATE INDEX idx_tasks_team_status_completed_at ON tasks(team_id, status, completed_at);
CREATE INDEX idx_tasks_team_created_by_created_at ON tasks(team_id, created_by, created_at);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
CREATE INDEX idx_tasks_created_by ON tasks(created_by);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);
CREATE INDEX idx_task_history_task_id ON task_history(task_id);
CREATE INDEX idx_team_members_team_role ON team_members(team_id, role);
CREATE INDEX idx_team_members_user ON team_members(user_id);
