import { useState } from "react";
import { LayoutGrid, LogOut, Menu, X } from "lucide-react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { tokenStore } from "../api/tokenStore";
import { useCurrentUser } from "../hooks/useCurrentUser";
import { initials } from "../lib/format";
import { AsyncState } from "./AsyncState";

export function AppLayout() {
  const [menuOpen, setMenuOpen] = useState(false);
  const userQuery = useCurrentUser();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const logout = () => {
    tokenStore.clear();
    queryClient.clear();
    navigate("/login", { replace: true });
  };

  return (
    <div className="app-shell">
      <aside className={`sidebar ${menuOpen ? "sidebar--open" : ""}`}>
        <div className="brand-block">
          <div className="brand-mark">B</div>
          <div>
            <strong>Basis</strong>
            <span>Рабочее пространство</span>
          </div>
          <button className="icon-button sidebar__close" type="button" onClick={() => setMenuOpen(false)} aria-label="Закрыть навигацию">
            <X size={18} />
          </button>
        </div>

        <nav className="main-nav" aria-label="Основная навигация">
          <NavLink to="/teams" onClick={() => setMenuOpen(false)}>
            <LayoutGrid size={18} />
            Команды
          </NavLink>
        </nav>

        <div className="sidebar__account">
          {userQuery.isLoading && <AsyncState kind="loading" compact title="Загрузка профиля" />}
          {userQuery.isError && <AsyncState kind="error" compact title="Профиль недоступен" />}
          {userQuery.data && (
            <div className="account-row">
              <div className="avatar">{initials(userQuery.data.email)}</div>
              <div className="account-row__copy">
                <strong>{userQuery.data.email}</strong>
                <span>Пользователь #{userQuery.data.id}</span>
              </div>
              <button className="icon-button" type="button" onClick={logout} aria-label="Выйти" title="Выйти">
                <LogOut size={18} />
              </button>
            </div>
          )}
        </div>
      </aside>

      {menuOpen && <button className="sidebar-scrim" type="button" onClick={() => setMenuOpen(false)} aria-label="Закрыть навигацию" />}

      <div className="app-main">
        <header className="mobile-header">
          <button className="icon-button" type="button" onClick={() => setMenuOpen(true)} aria-label="Открыть навигацию">
            <Menu size={20} />
          </button>
          <strong>Basis</strong>
          <button className="icon-button" type="button" onClick={logout} aria-label="Выйти">
            <LogOut size={18} />
          </button>
        </header>
        <main className="content-area">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
