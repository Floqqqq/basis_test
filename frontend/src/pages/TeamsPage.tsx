import { useState, type FormEvent } from "react";
import { ArrowRight, Plus, UsersRound } from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { teamsApi } from "../api/teams";
import { AsyncState } from "../components/AsyncState";
import { Modal } from "../components/Modal";
import { formatDate, roleLabel } from "../lib/format";

export const teamsQueryKey = ["teams"] as const;

export function TeamsPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const teamsQuery = useQuery({ queryKey: teamsQueryKey, queryFn: teamsApi.list });

  const createMutation = useMutation({
    mutationFn: () => teamsApi.create(name.trim()),
    onSuccess: async (response) => {
      await queryClient.invalidateQueries({ queryKey: teamsQueryKey });
      setCreateOpen(false);
      setName("");
      navigate(`/teams/${response.team_id}`);
    },
  });

  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (name.trim()) createMutation.mutate();
  };

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <span className="eyebrow">Рабочее пространство</span>
          <h1>Команды</h1>
          <p>Выберите команду для управления задачами и участниками.</p>
        </div>
        <button className="button button--primary" type="button" onClick={() => setCreateOpen(true)}>
          <Plus size={17} /> Новая команда
        </button>
      </header>

      {teamsQuery.isLoading && <AsyncState kind="loading" title="Загрузка команд" />}
      {teamsQuery.isError && <AsyncState kind="error" title="Не удалось загрузить команды" message={teamsQuery.error.message} />}
      {teamsQuery.data?.length === 0 && (
        <AsyncState kind="empty" title="Создайте первую команду" message="Команда объединяет участников и их задачи в одном месте." />
      )}

      {teamsQuery.data && teamsQuery.data.length > 0 && (
        <div className="team-grid">
          {teamsQuery.data.map((team) => (
            <Link className="team-card" to={`/teams/${team.id}`} key={team.id}>
              <div className="team-card__top">
                <div className="team-icon"><UsersRound size={20} /></div>
                <span className={`role-pill role-pill--${team.role}`}>{roleLabel(team.role)}</span>
              </div>
              <div>
                <h2>{team.name}</h2>
                <p>Создана {formatDate(team.created_at)}</p>
              </div>
              <span className="team-card__link">Открыть команду <ArrowRight size={16} /></span>
            </Link>
          ))}
        </div>
      )}

      <Modal open={createOpen} title="Создание команды" description="Вы станете владельцем нового рабочего пространства." onClose={() => setCreateOpen(false)}>
        <form className="form-stack" onSubmit={submit}>
          <label className="field">
            <span>Название команды</span>
            <input value={name} onChange={(event) => setName(event.target.value)} placeholder="Команда разработки" maxLength={255} autoFocus required />
          </label>
          {createMutation.isError && <div className="form-error" role="alert">{createMutation.error.message}</div>}
          <div className="modal__actions">
            <button className="button button--secondary" type="button" onClick={() => setCreateOpen(false)}>Отмена</button>
            <button className="button button--primary" type="submit" disabled={createMutation.isPending || !name.trim()}>
              {createMutation.isPending ? "Создание..." : "Создать команду"}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
