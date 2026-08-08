import { useState, type FormEvent } from "react";
import { ArrowRight, LockKeyhole, Mail } from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../api/auth";
import { tokenStore } from "../api/tokenStore";

interface AuthFormProps {
  mode: "login" | "register";
}

export function AuthForm({ mode }: AuthFormProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const navigate = useNavigate();
  const isLogin = mode === "login";

  const mutation = useMutation({
    mutationFn: () => (isLogin ? authApi.login({ email, password }) : authApi.register({ email, password })),
    onSuccess: (response) => {
      tokenStore.set(response.token);
      navigate("/teams", { replace: true });
    },
  });

  const submit = (event: FormEvent) => {
    event.preventDefault();
    mutation.mutate();
  };

  return (
    <div className="auth-page">
      <div className="auth-brand">
        <div className="brand-mark brand-mark--large">B</div>
        <strong>Рабочее пространство Basis</strong>
      </div>

      <section className="auth-panel">
        <header>
          <span className="eyebrow">Управление командами</span>
          <h1>{isLogin ? "С возвращением" : "Создание аккаунта"}</h1>
          <p>{isLogin ? "Войдите, чтобы продолжить работу с командами и задачами." : "Начните организовывать совместную работу."}</p>
        </header>

        <form onSubmit={submit} className="form-stack">
          <label className="field">
            <span>Электронная почта</span>
            <div className="input-with-icon">
              <Mail size={17} aria-hidden="true" />
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="user@example.com"
                autoComplete="email"
                required
              />
            </div>
          </label>

          <label className="field">
            <span>Пароль</span>
            <div className="input-with-icon">
              <LockKeyhole size={17} aria-hidden="true" />
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="Не менее 6 символов"
                autoComplete={isLogin ? "current-password" : "new-password"}
                minLength={6}
                required
              />
            </div>
          </label>

          {mutation.isError && <div className="form-error" role="alert">{mutation.error.message}</div>}

          <button className="button button--primary button--full" type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "Подождите..." : isLogin ? "Войти" : "Создать аккаунт"}
            {!mutation.isPending && <ArrowRight size={17} />}
          </button>
        </form>

        <footer>
          {isLogin ? "Впервые в Basis?" : "Уже есть аккаунт?"}{" "}
          <Link to={isLogin ? "/register" : "/login"}>{isLogin ? "Создать аккаунт" : "Войти"}</Link>
        </footer>
      </section>
    </div>
  );
}
