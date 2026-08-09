import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { tokenStore } from "../api/tokenStore";
import type { RealtimeEvent } from "../types/api";

const reconnectDelays = [1000, 2000, 5000, 10000, 15000];

export function useTeamRealtime(teamId?: number) {
  const queryClient = useQueryClient();

  useEffect(() => {
    const token = tokenStore.get();
    if (!teamId || !token) return;

    let socket: WebSocket | null = null;
    let reconnectTimer: number | undefined;
    let reconnectAttempt = 0;
    let stopped = false;

    const invalidate = (event: RealtimeEvent) => {
      void queryClient.invalidateQueries({ queryKey: ["team-activity", teamId] });
      if (event.event_type.startsWith("task.")) {
        void queryClient.invalidateQueries({ queryKey: ["tasks", teamId] });
        void queryClient.invalidateQueries({ queryKey: ["task", event.entity_id] });
        void queryClient.invalidateQueries({ queryKey: ["task-history", event.entity_id] });
      }
      if (event.event_type === "team.member_added") {
        void queryClient.invalidateQueries({ queryKey: ["team-members", teamId] });
      }
    };

    const connect = () => {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      socket = new WebSocket(`${protocol}//${window.location.host}/api/v1/teams/${teamId}/ws`, ["access_token", token]);
      socket.onopen = () => { reconnectAttempt = 0; };
      socket.onmessage = (message) => {
        try {
          const event = JSON.parse(message.data) as RealtimeEvent;
          if (event.team_id === teamId) invalidate(event);
        } catch {
          // Некорректное сообщение не должно останавливать последующие обновления.
        }
      };
      socket.onclose = () => {
        if (stopped) return;
        const delay = reconnectDelays[Math.min(reconnectAttempt, reconnectDelays.length - 1)];
        reconnectAttempt += 1;
        reconnectTimer = window.setTimeout(connect, delay);
      };
    };

    connect();
    return () => {
      stopped = true;
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [queryClient, teamId]);
}
