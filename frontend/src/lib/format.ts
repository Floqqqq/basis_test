import type { TaskStatus, TeamRole } from "../types/api";

export function formatDate(value: string): string {
  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

export function statusLabel(status: TaskStatus): string {
  return {
    todo: "К выполнению",
    in_progress: "В работе",
    done: "Готово",
  }[status];
}

export function roleLabel(role: TeamRole): string {
  return {
    owner: "Владелец",
    admin: "Администратор",
    member: "Участник",
  }[role];
}

export function initials(value: string): string {
  return value.slice(0, 2).toUpperCase();
}
