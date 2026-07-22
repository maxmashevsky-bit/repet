"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useState } from "react";
import { authApi } from "../../shared/api/client";
import type { Role } from "../../shared/api/types";
import { Brand, BrandMark } from "../ui/Brand";
import { Icon } from "../ui/Icon";

function PublicShell({
  children,
  actionText,
  actionLabel,
  actionHref
}: {
  children: React.ReactNode;
  actionText: string;
  actionLabel: string;
  actionHref: string;
}) {
  return (
    <main className="auth-page">
      <header className="auth-topbar">
        <Brand href="/login" />
        <div className="auth-top-action">
          <span>{actionText}</span>
          <Link className="button secondary" href={actionHref}>{actionLabel}</Link>
        </div>
      </header>
      <div className="auth-shell">
        <section className="auth-promo" aria-labelledby="auth-title">
          <div className="auth-copy">
            <h1 id="auth-title">Учиться<br />проще вместе</h1>
            <p>Занятия, задания и общение —<br />в одном спокойном пространстве.</p>
          </div>
          <ul className="benefits">
            <li><span className="benefit-icon"><Icon name="calendar" /></span>Все занятия под рукой</li>
            <li><span className="benefit-icon"><Icon name="chat" /></span>Прямая связь с преподавателем</li>
            <li><span className="benefit-icon"><Icon name="chart" /></span>Прогресс без лишнего стресса</li>
          </ul>
          <BrandMark decorative />
        </section>
        <section className="auth-card">{children}</section>
      </div>
      <footer className="auth-footer">Нравится «Репет»? <span>Поддержать создателя ♥</span></footer>
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
    <PublicShell actionText="Уже есть аккаунт?" actionLabel="Войти" actionHref="/login">
      <p className="eyebrow">Добро пожаловать</p>
      <h2>Создайте аккаунт</h2>
      <p className="muted">Сначала выберите, как вы будете пользоваться «Репет»</p>
      <form onSubmit={submit} className="form-stack">
        <fieldset className="role-grid">
          <legend>Кто вы?</legend>
          <button type="button" className={role === "student" ? "role-card active" : "role-card"} onClick={() => setRole("student")}>
            <span className="role-icon"><Icon name="book" size={36} /></span>
            <strong>Я ученик</strong>
            <span>Учусь и выполняю задания</span>
            {role === "student" ? <span className="role-check"><Icon name="check" size={16} /></span> : null}
          </button>
          <button type="button" className={role === "tutor" ? "role-card active" : "role-card"} onClick={() => setRole("tutor")}>
            <span className="role-icon"><Icon name="monitor" size={36} /></span>
            <strong>Я учитель</strong>
            <span>Провожу занятия и проверяю работы</span>
            {role === "tutor" ? <span className="role-check"><Icon name="check" size={16} /></span> : null}
          </button>
        </fieldset>
        <label>Имя и фамилия<input value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="Как к вам обращаться" required /></label>
        <label>Электронная почта<input value={email} onChange={(event) => setEmail(event.target.value)} placeholder="name@example.com" type="email" required /></label>
        <label>
          Пароль
          <span className="password-row">
            <input value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Не менее 8 символов" type={showPassword ? "text" : "password"} minLength={8} required />
            <button type="button" className="password-toggle" aria-label={showPassword ? "Скрыть пароль" : "Показать пароль"} onClick={() => setShowPassword((value) => !value)}>
              <Icon name="eye" />
            </button>
          </span>
        </label>
        <label className="check-row">
          <input type="checkbox" checked={acceptTerms} onChange={(event) => setAcceptTerms(event.target.checked)} />
          <span>Я принимаю <Link href="/terms">условия использования</Link> и <Link href="/privacy">политику конфиденциальности</Link></span>
        </label>
        {error ? <p className="field-error">{error}</p> : null}
        <button type="submit" disabled={loading || !acceptTerms}>{loading ? "Создаём..." : "Создать аккаунт"}</button>
        <div className="auth-divider"><span>или</span></div>
        <button type="button" className="oauth-button" disabled><span className="google-mark">G</span>Продолжить с Google</button>
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
    <PublicShell actionText="Нет аккаунта?" actionLabel="Создать" actionHref="/register">
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
    <PublicShell actionText="Вспомнили пароль?" actionLabel="Войти" actionHref="/login">
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
    <PublicShell actionText="Вернуться ко входу" actionLabel="Войти" actionHref="/login">
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
