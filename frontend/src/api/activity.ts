import type { ActivityEvent } from "../types/api";
import { apiRequest } from "./client";

export const activityApi = {
  list(teamId: number, limit: number, offset: number): Promise<ActivityEvent[]> {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
    return apiRequest<ActivityEvent[]>(`/teams/${teamId}/activity?${params}`);
  },
};
