import { useMemo, useState, type FormEvent } from "react";
import { ArrowLeft, Check, ChevronLeft, ChevronRight, LogOut, Plus, Trash2, UserPlus, X } from "lucide-react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { teamsApi } from "../api/teams";
import { tasksApi } from "../api/tasks";
import { AsyncState } from "../components/AsyncState";
import { Modal } from "../components/Modal";
import { StatusBadge } from "../components/StatusBadge";
import { formatDate, roleLabel } from "../lib/format";
import { useCurrentUser } from "../hooks/useCurrentUser";
import type { TaskFilters, TaskStatus, TeamLeaveRequest, TeamMember, TeamRole } from "../types/api";
import { teamsQueryKey } from "./TeamsPage";

const PAGE_SIZE = 10;

export function TeamPage() {
  const { teamId: teamIdParam } = useParams();
  const teamId = Number(teamIdParam);
  const validTeamId = Number.isInteger(teamId) && teamId > 0;
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = searchParams.get("tab") === "members" ? "members" : "tasks";
  const [status, setStatus] = useState<TaskStatus | "">("");
  const [assigneeId, setAssigneeId] = useState("");
  const [offset, setOffset] = useState(0);
  const [createOpen, setCreateOpen] = useState(false);
  const [inviteOpen, setInviteOpen] = useState(false);
  const queryClient = useQueryClient();
	const currentUserQuery = useCurrentUser();

  const teamsQuery = useQuery({ queryKey: teamsQueryKey, queryFn: teamsApi.list });
  const team = teamsQuery.data?.find((item) => item.id === teamId);
	const canManageMembers = team?.role === "owner" || team?.role === "admin";
  const membersQuery = useQuery({
    queryKey: ["team-members", teamId],
    queryFn: () => teamsApi.members(teamId),
    enabled: validTeamId,
  });

  const filters: TaskFilters = useMemo(() => ({
    status: status || undefined,
    assigneeId: assigneeId ? Number(assigneeId) : undefined,
    limit: PAGE_SIZE,
    offset,
  }), [status, assigneeId, offset]);

  const tasksQuery = useQuery({
    queryKey: ["tasks", teamId, filters],
    queryFn: () => tasksApi.list(teamId, filters),
    enabled: validTeamId && activeTab === "tasks",
  });
	const leaveRequestsQuery = useQuery({
		queryKey: ["team-leave-requests", teamId],
		queryFn: () => teamsApi.leaveRequests(teamId),
		enabled: validTeamId && canManageMembers,
		refetchInterval: 3000,
	});
	const ownLeaveRequestQuery = useQuery({
		queryKey: ["team-leave-request", teamId],
		queryFn: () => teamsApi.ownLeaveRequest(teamId),
		enabled: validTeamId && Boolean(team) && team?.role !== "owner",
		refetchInterval: 3000,
	});
	const requestLeaveMutation = useMutation({
		mutationFn: () => teamsApi.requestLeave(teamId),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ["team-leave-request", teamId] }),
				queryClient.invalidateQueries({ queryKey: ["team-leave-requests", teamId] }),
			]);
		},
	});

  if (!validTeamId) {
    return <AsyncState kind="error" title="Некорректная команда" message="Идентификатор команды указан неверно." />;
  }

  if (teamsQuery.isLoading) {
    return <AsyncState kind="loading" title="Загрузка команды" />;
  }

  if (teamsQuery.isError) {
    return <AsyncState kind="error" title="Не удалось загрузить команду" message={teamsQuery.error.message} />;
  }

  if (!team) {
    return <AsyncState kind="error" title="Команда не найдена" message="Возможно, у вас больше нет доступа к этой команде." />;
  }

  const canInvite = team.role === "owner" || team.role === "admin";
  const membersById = new Map(membersQuery.data?.map((member) => [member.id, member.email]));

  return (
    <div className="page-stack">
      <Link className="back-link" to="/teams"><ArrowLeft size={16} /> Все команды</Link>

      <header className="page-header page-header--team">
        <div>
          <div className="title-with-meta">
            <h1>{team.name}</h1>
            <span className={`role-pill role-pill--${team.role}`}>{roleLabel(team.role)}</span>
          </div>
          <p>Создана {formatDate(team.created_at)}</p>
        </div>
        {activeTab === "tasks" && (
          <button className="button button--primary" type="button" onClick={() => setCreateOpen(true)}>
            <Plus size={17} /> Новая задача
          </button>
        )}
		{activeTab === "members" && (
			<div className="page-actions">
				{team.role !== "owner" && (
					<button
						className="button button--secondary"
						type="button"
						onClick={() => requestLeaveMutation.mutate()}
						disabled={requestLeaveMutation.isPending || ownLeaveRequestQuery.isLoading || Boolean(ownLeaveRequestQuery.data)}
					>
						<LogOut size={17} /> {ownLeaveRequestQuery.data ? "Запрос отправлен" : requestLeaveMutation.isPending ? "Отправка..." : "Выйти из команды"}
					</button>
				)}
				{canInvite && (
					<button className="button button--primary" type="button" onClick={() => setInviteOpen(true)}>
						<UserPlus size={17} /> Добавить участника
					</button>
				)}
			</div>
		)}
      </header>

      <div className="tabs" role="tablist" aria-label="Разделы команды">
        <button className={activeTab === "tasks" ? "active" : ""} onClick={() => setSearchParams({ tab: "tasks" })} role="tab">Задачи</button>
        <button className={activeTab === "members" ? "active" : ""} onClick={() => setSearchParams({ tab: "members" })} role="tab">Участники</button>
      </div>

      {activeTab === "tasks" ? (
        <TasksSection
          tasks={tasksQuery.data}
          loading={tasksQuery.isLoading}
          error={tasksQuery.error}
          membersById={membersById}
          members={membersQuery.data ?? []}
          status={status}
          assigneeId={assigneeId}
          offset={offset}
          onStatusChange={(value) => { setStatus(value); setOffset(0); }}
          onAssigneeChange={(value) => { setAssigneeId(value); setOffset(0); }}
          onPrevious={() => setOffset((value) => Math.max(0, value - PAGE_SIZE))}
          onNext={() => setOffset((value) => value + PAGE_SIZE)}
        />
      ) : (
		<MembersSection
			teamId={teamId}
			currentUserId={currentUserQuery.data?.id}
			currentUserRole={team.role}
			membersQuery={membersQuery}
			leaveRequestsQuery={leaveRequestsQuery}
		/>
      )}

		{requestLeaveMutation.isError && <div className="form-error" role="alert">{requestLeaveMutation.error.message}</div>}

      <CreateTaskModal
        open={createOpen}
        teamId={teamId}
        members={membersQuery.data ?? []}
        onClose={() => setCreateOpen(false)}
        onCreated={async () => {
          await queryClient.invalidateQueries({ queryKey: ["tasks", teamId] });
          setCreateOpen(false);
        }}
      />

      <InviteMemberModal
        open={inviteOpen}
        teamId={teamId}
        onClose={() => setInviteOpen(false)}
        onInvited={async () => {
          await queryClient.invalidateQueries({ queryKey: ["team-members", teamId] });
          setInviteOpen(false);
        }}
      />
    </div>
  );
}

interface TasksSectionProps {
  tasks: Awaited<ReturnType<typeof tasksApi.list>> | undefined;
  loading: boolean;
  error: Error | null;
  membersById: Map<number, string>;
  members: Awaited<ReturnType<typeof teamsApi.members>>;
  status: TaskStatus | "";
  assigneeId: string;
  offset: number;
  onStatusChange: (value: TaskStatus | "") => void;
  onAssigneeChange: (value: string) => void;
  onPrevious: () => void;
  onNext: () => void;
}

function TasksSection(props: TasksSectionProps) {
  return (
    <section className="section-stack">
      <div className="filter-bar">
        <label className="field field--compact">
          <span>Статус</span>
          <select value={props.status} onChange={(event) => props.onStatusChange(event.target.value as TaskStatus | "")}>
            <option value="">Все статусы</option>
            <option value="todo">К выполнению</option>
            <option value="in_progress">В работе</option>
            <option value="done">Готово</option>
          </select>
        </label>
        <label className="field field--compact">
          <span>Исполнитель</span>
          <select value={props.assigneeId} onChange={(event) => props.onAssigneeChange(event.target.value)}>
            <option value="">Все исполнители</option>
            {props.members.map((member) => <option value={member.id} key={member.id}>{member.email}</option>)}
          </select>
        </label>
      </div>

      {props.loading && <AsyncState kind="loading" title="Загрузка задач" />}
      {props.error && <AsyncState kind="error" title="Не удалось загрузить задачи" message={props.error.message} />}
      {props.tasks?.length === 0 && <AsyncState kind="empty" title="Задачи не найдены" message="Создайте задачу или измените фильтры." />}

      {props.tasks && props.tasks.length > 0 && (
        <div className="data-table-wrap">
          <table className="data-table">
            <thead><tr><th>Задача</th><th>Статус</th><th>Исполнитель</th><th>Создана</th></tr></thead>
            <tbody>
              {props.tasks.map((task) => (
                <tr key={task.id}>
                  <td><Link className="table-primary" to={`/tasks/${task.id}`}>{task.title}</Link><span>#{task.id}</span></td>
                  <td><StatusBadge status={task.status} /></td>
                  <td>{task.assignee_id ? props.membersById.get(task.assignee_id) ?? `Пользователь #${task.assignee_id}` : <span className="muted">Не назначен</span>}</td>
                  <td>{formatDate(task.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="pagination">
        <span>Страница {Math.floor(props.offset / PAGE_SIZE) + 1}</span>
        <div>
          <button className="icon-button" type="button" onClick={props.onPrevious} disabled={props.offset === 0} aria-label="Предыдущая страница" title="Предыдущая страница"><ChevronLeft size={18} /></button>
          <button className="icon-button" type="button" onClick={props.onNext} disabled={!props.tasks || props.tasks.length < PAGE_SIZE} aria-label="Следующая страница" title="Следующая страница"><ChevronRight size={18} /></button>
        </div>
      </div>
    </section>
  );
}

interface MembersSectionProps {
	teamId: number;
	currentUserId?: number;
	currentUserRole: TeamRole;
	membersQuery: UseQueryResult<TeamMember[], Error>;
	leaveRequestsQuery: UseQueryResult<TeamLeaveRequest[], Error>;
}

function MembersSection({ teamId, currentUserId, currentUserRole, membersQuery, leaveRequestsQuery }: MembersSectionProps) {
	const queryClient = useQueryClient();
	const [removeCandidate, setRemoveCandidate] = useState<TeamMember | null>(null);
	const removeMutation = useMutation({
		mutationFn: (userId: number) => teamsApi.removeMember(teamId, userId),
		onSuccess: async () => {
			setRemoveCandidate(null);
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ["team-members", teamId] }),
				queryClient.invalidateQueries({ queryKey: ["team-leave-requests", teamId] }),
			]);
		},
	});
	const decisionMutation = useMutation({
		mutationFn: ({ requestId, decision }: { requestId: number; decision: "approve" | "reject" }) =>
			teamsApi.resolveLeaveRequest(teamId, requestId, decision),
		onSuccess: async () => {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: ["team-leave-requests", teamId] }),
				queryClient.invalidateQueries({ queryKey: ["team-members", teamId] }),
				queryClient.invalidateQueries({ queryKey: ["tasks", teamId] }),
			]);
		},
	});
	const canRemove = (member: TeamMember) =>
		member.id !== currentUserId &&
		member.role !== "owner" &&
		(currentUserRole === "owner" || (currentUserRole === "admin" && member.role === "member"));

  if (membersQuery.isLoading) return <AsyncState kind="loading" title="Загрузка участников" />;
  if (membersQuery.isError) return <AsyncState kind="error" title="Не удалось загрузить участников" message={membersQuery.error.message} />;
  if (!membersQuery.data?.length) return <AsyncState kind="empty" title="Участников пока нет" />;

  return (
	<div className="section-stack">
		{(currentUserRole === "owner" || currentUserRole === "admin") && (
			<section className="leave-requests">
				<div className="section-header">
					<div><h2>Запросы на выход</h2><p>Ожидают решения владельца или администратора</p></div>
					<span className="count-badge">{leaveRequestsQuery.data?.length ?? 0}</span>
				</div>
				{leaveRequestsQuery.isLoading && <AsyncState compact kind="loading" title="Загрузка запросов" />}
				{leaveRequestsQuery.isError && <AsyncState compact kind="error" title="Не удалось загрузить запросы" message={leaveRequestsQuery.error.message} />}
				{leaveRequestsQuery.data?.length === 0 && <p className="empty-inline">Нет ожидающих запросов</p>}
				{leaveRequestsQuery.data?.map((request) => (
					<div className="leave-request-row" key={request.id}>
						<div><strong>{request.email}</strong><span>{roleLabel(request.role)} · {formatDate(request.requested_at)}</span></div>
						<div className="row-actions">
							<button className="icon-button" type="button" onClick={() => decisionMutation.mutate({ requestId: request.id, decision: "reject" })} disabled={decisionMutation.isPending} aria-label="Отклонить запрос" title="Отклонить"><X size={18} /></button>
							<button className="icon-button icon-button--accent" type="button" onClick={() => decisionMutation.mutate({ requestId: request.id, decision: "approve" })} disabled={decisionMutation.isPending} aria-label="Одобрить запрос" title="Одобрить"><Check size={18} /></button>
						</div>
					</div>
				))}
				{decisionMutation.isError && <div className="form-error" role="alert">{decisionMutation.error.message}</div>}
			</section>
		)}

		<div className="member-list">
      {membersQuery.data.map((member) => (
        <div className="member-row" key={member.id}>
          <div className="avatar">{member.email.slice(0, 2).toUpperCase()}</div>
          <div><strong>{member.email}</strong><span>В команде с {formatDate(member.joined_at)}</span></div>
          <span className={`role-pill role-pill--${member.role}`}>{roleLabel(member.role)}</span>
			{canRemove(member) && <button className="icon-button icon-button--danger" type="button" onClick={() => setRemoveCandidate(member)} aria-label={`Удалить ${member.email}`} title="Удалить из команды"><Trash2 size={17} /></button>}
        </div>
      ))}
		</div>
		{removeMutation.isError && <div className="form-error" role="alert">{removeMutation.error.message}</div>}
		<Modal open={Boolean(removeCandidate)} title="Удалить участника" description={removeCandidate ? `${removeCandidate.email} потеряет доступ к команде и перестанет быть исполнителем ее задач.` : ""} onClose={() => setRemoveCandidate(null)}>
			<div className="modal__actions"><button className="button button--secondary" type="button" onClick={() => setRemoveCandidate(null)}>Отмена</button><button className="button button--danger" type="button" onClick={() => removeCandidate && removeMutation.mutate(removeCandidate.id)} disabled={removeMutation.isPending}>{removeMutation.isPending ? "Удаление..." : "Удалить"}</button></div>
		</Modal>
	</div>
  );
}

interface CreateTaskModalProps {
  open: boolean;
  teamId: number;
  members: Awaited<ReturnType<typeof teamsApi.members>>;
  onClose: () => void;
  onCreated: () => Promise<void>;
}

function CreateTaskModal({ open, teamId, members, onClose, onCreated }: CreateTaskModalProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<TaskStatus>("todo");
  const [assignee, setAssignee] = useState("");
  const mutation = useMutation({
    mutationFn: () => tasksApi.create({ team_id: teamId, title: title.trim(), description, status, assignee_id: assignee ? Number(assignee) : null }),
    onSuccess: async () => {
      setTitle(""); setDescription(""); setStatus("todo"); setAssignee("");
      await onCreated();
    },
  });

  const submit = (event: FormEvent) => { event.preventDefault(); if (title.trim()) mutation.mutate(); };

  return (
    <Modal open={open} title="Создание задачи" description="Добавьте задачу и при необходимости назначьте исполнителя." onClose={onClose}>
      <form className="form-stack" onSubmit={submit}>
        <label className="field"><span>Название</span><input value={title} onChange={(event) => setTitle(event.target.value)} maxLength={255} autoFocus required /></label>
        <label className="field"><span>Описание</span><textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={4} /></label>
        <div className="form-grid">
          <label className="field"><span>Статус</span><select value={status} onChange={(event) => setStatus(event.target.value as TaskStatus)}><option value="todo">К выполнению</option><option value="in_progress">В работе</option><option value="done">Готово</option></select></label>
          <label className="field"><span>Исполнитель</span><select value={assignee} onChange={(event) => setAssignee(event.target.value)}><option value="">Не назначен</option>{members.map((member) => <option value={member.id} key={member.id}>{member.email}</option>)}</select></label>
        </div>
        {mutation.isError && <div className="form-error" role="alert">{mutation.error.message}</div>}
        <div className="modal__actions"><button className="button button--secondary" type="button" onClick={onClose}>Отмена</button><button className="button button--primary" type="submit" disabled={mutation.isPending || !title.trim()}>{mutation.isPending ? "Создание..." : "Создать задачу"}</button></div>
      </form>
    </Modal>
  );
}

interface InviteMemberModalProps { open: boolean; teamId: number; onClose: () => void; onInvited: () => Promise<void>; }

function InviteMemberModal({ open, teamId, onClose, onInvited }: InviteMemberModalProps) {
  const [userId, setUserId] = useState("");
  const [role, setRole] = useState<Exclude<TeamRole, "owner">>("member");
  const mutation = useMutation({ mutationFn: () => teamsApi.invite(teamId, Number(userId), role), onSuccess: async () => { setUserId(""); setRole("member"); await onInvited(); } });
  const submit = (event: FormEvent) => { event.preventDefault(); if (Number(userId) > 0) mutation.mutate(); };

  return (
    <Modal open={open} title="Добавление участника" description="Добавьте существующего пользователя в команду по его ID." onClose={onClose}>
      <form className="form-stack" onSubmit={submit}>
        <div className="form-grid">
          <label className="field"><span>ID пользователя</span><input type="number" min="1" value={userId} onChange={(event) => setUserId(event.target.value)} autoFocus required /></label>
          <label className="field"><span>Роль</span><select value={role} onChange={(event) => setRole(event.target.value as Exclude<TeamRole, "owner">)}><option value="member">Участник</option><option value="admin">Администратор</option></select></label>
        </div>
        {mutation.isError && <div className="form-error" role="alert">{mutation.error.message}</div>}
        <div className="modal__actions"><button className="button button--secondary" type="button" onClick={onClose}>Отмена</button><button className="button button--primary" type="submit" disabled={mutation.isPending || Number(userId) <= 0}>{mutation.isPending ? "Добавление..." : "Добавить участника"}</button></div>
      </form>
    </Modal>
  );
}
