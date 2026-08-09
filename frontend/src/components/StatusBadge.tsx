import type { TaskStatus } from "../types/api";
import { statusLabel } from "../lib/format";

export function StatusBadge({ status }: { status: TaskStatus }) {
  return <span className={`status-badge status-badge--${status}`}>{statusLabel(status)}</span>;
}
