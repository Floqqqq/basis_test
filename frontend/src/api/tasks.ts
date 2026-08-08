import type {
  CreateTaskPayload,
  Task,
  TaskComment,
  TaskFilters,
  TaskHistory,
  UpdateTaskPayload,
} from "../types/api";
import { apiRequest } from "./client";

function taskListQuery(teamId: number, filters: TaskFilters): string {
  const params = new URLSearchParams({
    team_id: String(teamId),
    limit: String(filters.limit),
    offset: String(filters.offset),
  });
  if (filters.status) {
    params.set("status", filters.status);
  }
  if (filters.assigneeId) {
    params.set("assignee_id", String(filters.assigneeId));
  }
  return params.toString();
}

export const tasksApi = {
  list(teamId: number, filters: TaskFilters): Promise<Task[]> {
    return apiRequest<Task[]>(`/tasks?${taskListQuery(teamId, filters)}`);
  },

  get(taskId: number): Promise<Task> {
    return apiRequest<Task>(`/tasks/${taskId}`);
  },

  create(payload: CreateTaskPayload): Promise<{ task_id: number }> {
    return apiRequest<{ task_id: number }>("/tasks", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  update(taskId: number, payload: UpdateTaskPayload): Promise<void> {
    return apiRequest<void>(`/tasks/${taskId}`, {
      method: "PUT",
      body: JSON.stringify(payload),
    });
  },

  history(taskId: number): Promise<TaskHistory[]> {
    return apiRequest<TaskHistory[]>(`/tasks/${taskId}/history`);
  },

  comments(taskId: number): Promise<TaskComment[]> {
    return apiRequest<TaskComment[]>(`/tasks/${taskId}/comments`);
  },

  addComment(taskId: number, comment: string): Promise<TaskComment> {
    return apiRequest<TaskComment>(`/tasks/${taskId}/comments`, {
      method: "POST",
      body: JSON.stringify({ comment }),
    });
  },
};
