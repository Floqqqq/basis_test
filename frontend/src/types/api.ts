export type TeamRole = "owner" | "admin" | "member";
export type TaskStatus = "todo" | "in_progress" | "done";

export interface AuthResponse {
  user_id: number;
  token: string;
}

export interface User {
  id: number;
  email: string;
  created_at: string;
}

export interface Team {
  id: number;
  name: string;
  created_by: number;
  created_at: string;
  role: TeamRole;
}

export interface TeamMember {
  id: number;
  email: string;
  role: TeamRole;
  joined_at: string;
}

export interface TeamLeaveRequest {
  id: number;
  team_id: number;
  user_id: number;
  email: string;
  role: TeamRole;
  status: "pending";
  requested_at: string;
}

export interface Task {
  id: number;
  title: string;
  description: string;
  status: TaskStatus;
  assignee_id: number | null;
  completed_at?: string | null;
  team_id: number;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface TaskHistory {
  id: number;
  task_id: number;
  changed_by: number;
  field_name: string;
  old_value: string;
  new_value: string;
  created_at: string;
}

export interface TaskComment {
  id: number;
  task_id: number;
  user_id: number;
  email: string;
  text: string;
  created_at: string;
}

export interface TaskFilters {
  status?: TaskStatus;
  assigneeId?: number;
  limit: number;
  offset: number;
}

export interface CreateTaskPayload {
  team_id: number;
  title: string;
  description: string;
  status: TaskStatus;
  assignee_id: number | null;
}

export interface UpdateTaskPayload {
  title?: string;
  description?: string;
  status?: TaskStatus;
  assignee_id?: number | null;
}
