import { Navigate, Outlet, createBrowserRouter } from "react-router-dom";
import { tokenStore } from "../api/tokenStore";
import { AppLayout } from "../components/AppLayout";
import { LoginPage } from "../pages/LoginPage";
import { NotFoundPage } from "../pages/NotFoundPage";
import { RegisterPage } from "../pages/RegisterPage";
import { TaskPage } from "../pages/TaskPage";
import { TeamPage } from "../pages/TeamPage";
import { TeamsPage } from "../pages/TeamsPage";

function ProtectedRoute() {
  return tokenStore.has() ? <Outlet /> : <Navigate to="/login" replace />;
}

function GuestRoute() {
  return tokenStore.has() ? <Navigate to="/teams" replace /> : <Outlet />;
}

export const router = createBrowserRouter([
  {
    element: <GuestRoute />,
    children: [
      { path: "/login", element: <LoginPage /> },
      { path: "/register", element: <RegisterPage /> },
    ],
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { index: true, element: <Navigate to="/teams" replace /> },
          { path: "/teams", element: <TeamsPage /> },
          { path: "/teams/:teamId", element: <TeamPage /> },
          { path: "/tasks/:taskId", element: <TaskPage /> },
        ],
      },
    ],
  },
  { path: "*", element: <NotFoundPage /> },
]);
