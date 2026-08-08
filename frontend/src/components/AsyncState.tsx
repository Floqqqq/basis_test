import { AlertCircle, Inbox, LoaderCircle } from "lucide-react";

interface AsyncStateProps {
  kind: "loading" | "error" | "empty";
  title?: string;
  message?: string;
  compact?: boolean;
}

export function AsyncState({ kind, title, message, compact = false }: AsyncStateProps) {
  const Icon = kind === "loading" ? LoaderCircle : kind === "error" ? AlertCircle : Inbox;
  const defaultTitle = {
    loading: "Загрузка",
    error: "Что-то пошло не так",
    empty: "Здесь пока ничего нет",
  }[kind];

  return (
    <div className={`async-state ${compact ? "async-state--compact" : ""}`} role={kind === "error" ? "alert" : "status"}>
      <Icon className={kind === "loading" ? "spin" : ""} size={compact ? 20 : 26} aria-hidden="true" />
      <div>
        <strong>{title ?? defaultTitle}</strong>
        {message && <p>{message}</p>}
      </div>
    </div>
  );
}
