import type { Team, TeamLeaveRequest, TeamMember, TeamRole } from "../types/api";
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

  removeMember(teamId: number, userId: number): Promise<void> {
    return apiRequest<void>(`/teams/${teamId}/members/${userId}`, { method: "DELETE" });
  },

  requestLeave(teamId: number): Promise<TeamLeaveRequest> {
    return apiRequest<TeamLeaveRequest>(`/teams/${teamId}/leave-requests`, { method: "POST" });
  },

  ownLeaveRequest(teamId: number): Promise<TeamLeaveRequest | null> {
    return apiRequest<TeamLeaveRequest | null>(`/teams/${teamId}/leave-request`);
  },

  leaveRequests(teamId: number): Promise<TeamLeaveRequest[]> {
    return apiRequest<TeamLeaveRequest[]>(`/teams/${teamId}/leave-requests`);
  },

  resolveLeaveRequest(teamId: number, requestId: number, decision: "approve" | "reject"): Promise<void> {
    return apiRequest<void>(`/teams/${teamId}/leave-requests/${requestId}/decision`, {
      method: "POST",
      body: JSON.stringify({ decision }),
    });
  },
};
