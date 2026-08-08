import type { Team, TeamMember, TeamRole } from "../types/api";
import { apiRequest } from "./client";

export const teamsApi = {
  list(): Promise<Team[]> {
    return apiRequest<Team[]>("/teams");
  },

  create(name: string): Promise<{ team_id: number }> {
    return apiRequest<{ team_id: number }>("/teams", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  },

  members(teamId: number): Promise<TeamMember[]> {
    return apiRequest<TeamMember[]>(`/teams/${teamId}/members`);
  },

  invite(teamId: number, userId: number, role: Exclude<TeamRole, "owner">): Promise<void> {
    return apiRequest<void>(`/teams/${teamId}/invite`, {
      method: "POST",
      body: JSON.stringify({ user_id: userId, role }),
    });
  },
};
