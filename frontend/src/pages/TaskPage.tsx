import { useEffect, useState, type FormEvent, type ReactNode } from "react";
import { ArrowLeft, Calendar, MessageSquare, Pencil, Send, UserRound } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { teamsApi } from "../api/teams";
import { tasksApi } from "../api/tasks";
import { AsyncState } from "../components/AsyncState";
import { Modal } from "../components/Modal";
import { StatusBadge } from "../components/StatusBadge";
import { useCurrentUser } from "../hooks/useCurrentUser";
import { formatDate } from "../lib/format";
import type { Task, TaskStatus, TeamMember } from "../types/api";
import { teamsQueryKey } from "./TeamsPage";

export function TaskPage() {
  const { taskId: taskIdParam } = useParams();
  const taskId = Number(taskIdParam);
  const validTaskId = Number.isInteger(taskId) && taskId > 0;
  const [editOpen, setEditOpen] = useState(false);
  const queryClient = useQueryClient();

  const userQuery = useCurrentUser();
  const taskQuery = useQuery({ queryKey: ["task", taskId], queryFn: () => tasksApi.get(taskId), enabled: validTaskId });
  const teamsQuery = useQuery({ queryKey: teamsQueryKey, queryFn: teamsApi.list });
  const team = teamsQuery.data?.find((item) => item.id === taskQuery.data?.team_id);
  const membersQuery = useQuery({
    queryKey: ["team-members", taskQuery.data?.team_id],
    queryFn: () => teamsApi.members(taskQuery.data!.team_id),
    enabled: Boolean(taskQuery.data?.team_id),
  });

  const canEdit = Boolean(
    taskQuery.data && userQuery.data && team && (
      team.role === "owner" ||
      team.role === "admin" ||
      taskQuery.data.created_by === userQuery.data.id ||
      taskQuery.data.assignee_id === userQuery.data.id
    ),
  );

  const historyQuery = useQuery({
    queryKey: ["task-history", taskId],
    queryFn: () => tasksApi.history(taskId),
    enabled: validTaskId && canEdit,
  });
  const commentsQuery = useQuery({
    queryKey: ["task-comments", taskId],
    queryFn: () => tasksApi.comments(taskId),
    enabled: validTaskId,
  });

  if (!validTaskId) return <AsyncState kind="error" title="Некорректная задача" message="Идентификатор задачи указан неверно." />;
  if (taskQuery.isLoading || teamsQuery.isLoading || userQuery.isLoading) return <AsyncState kind="loading" title="Загрузка задачи" />;
  if (taskQuery.isError) return <AsyncState kind="error" title="Не удалось загрузить задачу" message={taskQuery.error.message} />;
  if (teamsQuery.isError) return <AsyncState kind="error" title="Не удалось загрузить команду" message={teamsQuery.error.message} />;
  if (!taskQuery.data) return <AsyncState kind="error" title="Задача не найдена" />;

  const task = taskQuery.data;
  const members = membersQuery.data ?? [];
  const membersById = new Map(members.map((member) => [member.id, member.email]));
  const assignee = task.assignee_id ? membersById.get(task.assignee_id) ?? `Пользователь #${task.assignee_id}` : "Не назначен";
  const creator = membersById.get(task.created_by) ?? `Пользователь #${task.created_by}`;

  return (
    <div className="page-stack task-page">
      <Link className="back-link" to={`/teams/${task.team_id}`}><ArrowLeft size={16} /> {team?.name ?? "Вернуться к команде"}</Link>

      <header className="task-heading">
        <div>
          <div className="task-heading__meta"><span>Задача #{task.id}</span><StatusBadge status={task.status} /></div>
          <h1>{task.title}</h1>
        </div>
        {canEdit && <button className="button button--primary" type="button" onClick={() => setEditOpen(true)}><Pencil size={16} /> Редактировать</button>}
      </header>

      <div className="task-overview">
        <section className="task-description">
          <span className="section-label">Описание</span>
          <p className={task.description ? "" : "muted"}>{task.description || "Описание не добавлено."}</p>
        </section>
        <aside className="task-facts">
          <Fact icon={<UserRound size={17} />} label="Исполнитель" value={assignee} />
          <Fact icon={<UserRound size={17} />} label="Автор" value={creator} />
          <Fact icon={<Calendar size={17} />} label="Создана" value={formatDate(task.created_at)} />
          <Fact icon={<Calendar size={17} />} label="Обновлена" value={formatDate(task.updated_at)} />
        </aside>
      </div>

      <div className="task-columns">
        <section className="content-section">
          <header className="section-header"><div><span className="section-label">Обсуждение</span><h2>Комментарии</h2></div><span className="count-badge">{commentsQuery.data?.length ?? 0}</span></header>
          <CommentForm taskId={taskId} onCreated={() => queryClient.invalidateQueries({ queryKey: ["task-comments", taskId] })} />
          {commentsQuery.isLoading && <AsyncState kind="loading" compact title="Загрузка комментариев" />}
          {commentsQuery.isError && <AsyncState kind="error" compact title="Не удалось загрузить комментарии" message={commentsQuery.error.message} />}
          {commentsQuery.data?.length === 0 && <AsyncState kind="empty" compact title="Комментариев пока нет" message="Начните обсуждение этой задачи." />}
          {commentsQuery.data && commentsQuery.data.length > 0 && (
            <div className="comment-list">
              {commentsQuery.data.map((comment) => (
                <article className="comment" key={comment.id}>
                  <div className="avatar">{comment.email.slice(0, 2).toUpperCase()}</div>
                  <div><header><strong>{comment.email}</strong><time>{formatDate(comment.created_at)}</time></header><p>{comment.text}</p></div>
                </article>
              ))}
            </div>
          )}
        </section>

        <section className="content-section">
          <header className="section-header"><div><span className="section-label">Журнал изменений</span><h2>История</h2></div></header>
          {!canEdit && <AsyncState kind="empty" compact title="История недоступна" message="Её могут просматривать автор, исполнитель и администраторы команды." />}
          {canEdit && historyQuery.isLoading && <AsyncState kind="loading" compact title="Загрузка истории" />}
          {canEdit && historyQuery.isError && <AsyncState kind="error" compact title="Не удалось загрузить историю" message={historyQuery.error.message} />}
          {canEdit && historyQuery.data?.length === 0 && <AsyncState kind="empty" compact title="Изменений пока нет" />}
          {canEdit && historyQuery.data && historyQuery.data.length > 0 && (
            <div className="history-list">
              {historyQuery.data.map((entry) => (
                <div className="history-item" key={entry.id}>
                  <span className="history-dot" />
                  <div><strong>{historyFieldLabel(entry.field_name)}</strong><p><span>{entry.old_value || "Пусто"}</span> → <span>{entry.new_value || "Пусто"}</span></p><time>{formatDate(entry.created_at)}, изменил {membersById.get(entry.changed_by) ?? `пользователь #${entry.changed_by}`}</time></div>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>

      <EditTaskModal
        open={editOpen}
        task={task}
        members={members}
        onClose={() => setEditOpen(false)}
        onUpdated={async () => {
          await Promise.all([
            queryClient.invalidateQueries({ queryKey: ["task", taskId] }),
            queryClient.invalidateQueries({ queryKey: ["tasks", task.team_id] }),
            queryClient.invalidateQueries({ queryKey: ["task-history", taskId] }),
          ]);
          setEditOpen(false);
        }}
      />
    </div>
  );
}

function Fact({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return <div className="fact-row"><span className="fact-row__icon">{icon}</span><div><span>{label}</span><strong>{value}</strong></div></div>;
}

function historyFieldLabel(field: string): string {
  const labels: Record<string, string> = {
    title: "Название",
    description: "Описание",
    status: "Статус",
    assignee_id: "Исполнитель",
  };

  return labels[field] ?? field.replaceAll("_", " ");
}

function CommentForm({ taskId, onCreated }: { taskId: number; onCreated: () => Promise<unknown> }) {
  const [comment, setComment] = useState("");
  const mutation = useMutation({
    mutationFn: () => tasksApi.addComment(taskId, comment.trim()),
    onSuccess: async () => { setComment(""); await onCreated(); },
  });
  const submit = (event: FormEvent) => { event.preventDefault(); if (comment.trim()) mutation.mutate(); };

  return (
    <form className="comment-form" onSubmit={submit}>
      <MessageSquare size={18} aria-hidden="true" />
      <textarea value={comment} onChange={(event) => setComment(event.target.value)} placeholder="Добавить комментарий..." maxLength={4000} rows={2} />
      <button className="icon-button icon-button--accent" type="submit" disabled={mutation.isPending || !comment.trim()} aria-label="Отправить комментарий" title="Отправить комментарий"><Send size={17} /></button>
      {mutation.isError && <div className="form-error" role="alert">{mutation.error.message}</div>}
    </form>
  );
}

interface EditTaskModalProps { open: boolean; task: Task; members: TeamMember[]; onClose: () => void; onUpdated: () => Promise<void>; }

function EditTaskModal({ open, task, members, onClose, onUpdated }: EditTaskModalProps) {
  const [title, setTitle] = useState(task.title);
  const [description, setDescription] = useState(task.description);
  const [status, setStatus] = useState<TaskStatus>(task.status);
  const [assignee, setAssignee] = useState(task.assignee_id ? String(task.assignee_id) : "");

  useEffect(() => {
    if (!open) return;
    setTitle(task.title);
    setDescription(task.description);
    setStatus(task.status);
    setAssignee(task.assignee_id ? String(task.assignee_id) : "");
  }, [open, task]);

  const mutation = useMutation({
    mutationFn: () => tasksApi.update(task.id, { title: title.trim(), description, status, assignee_id: assignee ? Number(assignee) : null }),
    onSuccess: onUpdated,
  });
  const submit = (event: FormEvent) => { event.preventDefault(); if (title.trim()) mutation.mutate(); };

  return (
    <Modal open={open} title="Редактирование задачи" description="Измените данные задачи или снимите текущего исполнителя." onClose={onClose}>
      <form className="form-stack" onSubmit={submit}>
        <label className="field"><span>Название</span><input value={title} onChange={(event) => setTitle(event.target.value)} maxLength={255} autoFocus required /></label>
        <label className="field"><span>Описание</span><textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={4} /></label>
        <div className="form-grid">
          <label className="field"><span>Статус</span><select value={status} onChange={(event) => setStatus(event.target.value as TaskStatus)}><option value="todo">К выполнению</option><option value="in_progress">В работе</option><option value="done">Готово</option></select></label>
          <label className="field"><span>Исполнитель</span><select value={assignee} onChange={(event) => setAssignee(event.target.value)}><option value="">Не назначен</option>{members.map((member) => <option value={member.id} key={member.id}>{member.email}</option>)}</select></label>
        </div>
        {mutation.isError && <div className="form-error" role="alert">{mutation.error.message}</div>}
        <div className="modal__actions"><button className="button button--secondary" type="button" onClick={onClose}>Отмена</button><button className="button button--primary" type="submit" disabled={mutation.isPending || !title.trim()}>{mutation.isPending ? "Сохранение..." : "Сохранить"}</button></div>
      </form>
    </Modal>
  );
}
