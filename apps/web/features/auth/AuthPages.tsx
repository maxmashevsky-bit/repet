"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useState } from "react";
import { authApi } from "../../shared/api/client";
import type { Role } from "../../shared/api/types";

function PublicShell({ children }: { children: React.ReactNode }) {
  return (
    <main className="auth-page">
      <section className="auth-promo" aria-labelledby="auth-title">
        <Link href="/login" className="brand auth-brand">
          <span className="brand-mark" aria-hidden="true">P</span>
          Репет
        </Link>
        <h1 id="auth-title">Учиться проще вместе</h1>
        <p>Занятия, задания и общение в одном спокойном пространстве.</p>
        <ul className="benefits">
          <li>Все занятия под рукой</li>
          <li>Прямая связь с преподавателем</li>
          <li>Прогресс без лишнего стресса</li>
        </ul>
      </section>
      <section className="auth-card">{children}</section>
      <footer className="auth-footer">Нравится «Репет»? Поддержать создателя ♥</footer>
    </main>
  );
}

export function RegisterPage() {
  const router = useRouter();
  const search = useSearchParams();
  const [role, setRole] = useState<Role>("student");
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [acceptTerms, setAcceptTerms] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      await authApi.register({ displayName, email, password, role, acceptTerms });
      router.push(authApi.safeReturnTo(search.get("returnTo")));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Регистрация не удалась");
    } finally {
      setLoading(false);
    }
  }

  return (
    <PublicShell>
      <p className="eyebrow">Добро пожаловать</p>
      <h2>Создайте аккаунт</h2>
      <p className="muted">Сначала выберите, как вы будете пользоваться «Репет».</p>
      <form onSubmit={submit} className="form-stack">
        <fieldset className="role-grid">
          <legend>Кто вы?</legend>
          <button type="button" className={role === "student" ? "role-card active" : "role-card"} onClick={() => setRole("student")}>
            <strong>Я ученик</strong>
            <span>Учусь и выполняю задания</span>
          </button>
          <button type="button" className={role === "tutor" ? "role-card active" : "role-card"} onClick={() => setRole("tutor")}>
            <strong>Я учитель</strong>
            <span>Провожу занятия и проверяю работы</span>
          </button>
        </fieldset>
        <label>Имя и фамилия<input value={displayName} onChange={(event) => setDisplayName(event.target.value)} required /></label>
        <label>Электронная почта<input value={email} onChange={(event) => setEmail(event.target.value)} type="email" required /></label>
        <label>
          Пароль
          <span className="password-row">
            <input value={password} onChange={(event) => setPassword(event.target.value)} type={showPassword ? "text" : "password"} minLength={8} required />
            <button type="button" className="secondary" onClick={() => setShowPassword((value) => !value)}>
              {showPassword ? "Скрыть" : "Показать"}
            </button>
          </span>
        </label>
        <p className="hint">Не менее 8 символов. В production включается проверка сложности и утечек.</p>
        <label className="check-row">
          <input type="checkbox" checked={acceptTerms} onChange={(event) => setAcceptTerms(event.target.checked)} />
          Я принимаю условия использования и политику конфиденциальности
        </label>
        {error ? <p className="field-error">{error}</p> : null}
        <button type="submit" disabled={loading || !acceptTerms}>{loading ? "Создаём..." : "Создать аккаунт"}</button>
        <button type="button" className="oauth-button" disabled>Продолжить с Google</button>
      </form>
      <p className="centered">Уже есть аккаунт? <Link href="/login">Войти</Link></p>
    </PublicShell>
  );
}

export function LoginPage() {
  const router = useRouter();
  const search = useSearchParams();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      await authApi.login({ email, password });
      router.push(authApi.safeReturnTo(search.get("returnTo")));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Проверьте email и пароль");
    } finally {
      setLoading(false);
    }
  }

  return (
    <PublicShell>
      <p className="eyebrow">Вход</p>
      <h2>Вернитесь к занятиям</h2>
      <form onSubmit={submit} className="form-stack">
        <label>Email<input value={email} onChange={(event) => setEmail(event.target.value)} type="email" required /></label>
        <label>Пароль<input value={password} onChange={(event) => setPassword(event.target.value)} type="password" required /></label>
        {error ? <p className="field-error">{error}</p> : null}
        <button type="submit" disabled={loading}>{loading ? "Входим..." : "Войти"}</button>
      </form>
      <p className="centered"><Link href="/forgot-password">Забыли пароль?</Link></p>
      <p className="centered">Нет аккаунта? <Link href="/register">Создать</Link></p>
    </PublicShell>
  );
}

export function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [done, setDone] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await authApi.forgotPassword(email);
    setDone(true);
  }

  return (
    <PublicShell>
      <h2>Восстановление пароля</h2>
      {done ? <p className="notice">Если аккаунт существует, письмо с инструкциями будет отправлено.</p> : null}
      <form onSubmit={submit} className="form-stack">
        <label>Email<input value={email} onChange={(event) => setEmail(event.target.value)} type="email" required /></label>
        <button type="submit">Отправить инструкцию</button>
      </form>
    </PublicShell>
  );
}

export function ResetPasswordPage() {
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [done, setDone] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await authApi.resetPassword(token, password);
    setDone(true);
  }

  return (
    <PublicShell>
      <h2>Новый пароль</h2>
      {done ? <p className="notice">Reset flow принят. Production-реализация одноразовых токенов запланирована в identity-service.</p> : null}
      <form onSubmit={submit} className="form-stack">
        <label>Токен<input value={token} onChange={(event) => setToken(event.target.value)} required /></label>
        <label>Новый пароль<input value={password} onChange={(event) => setPassword(event.target.value)} type="password" minLength={8} required /></label>
        <button type="submit">Сохранить пароль</button>
      </form>
    </PublicShell>
  );
}

