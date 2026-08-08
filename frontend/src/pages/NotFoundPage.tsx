import { ArrowLeft } from "lucide-react";
import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <div className="not-found">
      <span>404</span>
      <h1>Страница не найдена</h1>
      <p>Возможно, страница была перемещена или больше не существует.</p>
      <Link className="button button--primary" to="/teams">
        <ArrowLeft size={17} /> Вернуться к командам
      </Link>
    </div>
  );
}
