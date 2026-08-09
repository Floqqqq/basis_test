import { useState } from "react";
import { ChevronLeft, ChevronRight, ListChecks, RefreshCw, UserPlus } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { activityApi } from "../api/activity";
import { formatDate } from "../lib/format";
import { AsyncState } from "./AsyncState";

const PAGE_SIZE = 20;

export function ActivityFeed({ teamId }: { teamId: number }) {
  const [offset, setOffset] = useState(0);
  const activityQuery = useQuery({
    queryKey: ["team-activity", teamId, { limit: PAGE_SIZE, offset }],
    queryFn: () => activityApi.list(teamId, PAGE_SIZE, offset),
  });

  if (activityQuery.isLoading) return <AsyncState kind="loading" title="Загрузка активности" />;
  if (activityQuery.isError) return <AsyncState kind="error" title="Не удалось загрузить активность" message={activityQuery.error.message} />;

  return (
    <section className="section-stack">
      {activityQuery.data?.length === 0 && offset === 0 && (
        <AsyncState kind="empty" title="Событий пока нет" message="Новые действия команды появятся в этой ленте." />
      )}
      {activityQuery.data?.length === 0 && offset > 0 && (
        <AsyncState kind="empty" title="На этой странице нет событий" />
      )}
      {activityQuery.data && activityQuery.data.length > 0 && (
        <div className="activity-feed">
          {activityQuery.data.map((event) => (
            <article className="activity-row" key={event.id}>
              <span className="activity-row__icon" aria-hidden="true">
                {event.event_type === "team.member_added" ? <UserPlus size={17} /> : event.event_type === "task.created" ? <ListChecks size={17} /> : <RefreshCw size={17} />}
              </span>
              <div><p>{event.message}</p><time>{formatDate(event.created_at)}</time></div>
            </article>
          ))}
        </div>
      )}
      <div className="pagination">
        <span>Страница {Math.floor(offset / PAGE_SIZE) + 1}</span>
        <div>
          <button className="icon-button" type="button" onClick={() => setOffset((value) => Math.max(0, value - PAGE_SIZE))} disabled={offset === 0} aria-label="Предыдущая страница" title="Предыдущая страница"><ChevronLeft size={18} /></button>
          <button className="icon-button" type="button" onClick={() => setOffset((value) => value + PAGE_SIZE)} disabled={!activityQuery.data || activityQuery.data.length < PAGE_SIZE} aria-label="Следующая страница" title="Следующая страница"><ChevronRight size={18} /></button>
        </div>
      </div>
    </section>
  );
}
