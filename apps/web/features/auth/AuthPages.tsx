"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useState, useEffect } from "react";
import { authApi, productApi, ApiError } from "../../shared/api/client";
import type { Role, User } from "../../shared/api/types";
import { Brand } from "../ui/Brand";
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
          <Link className="button secondary" href={actionHref}><span>{actionText} </span>{actionLabel}</Link>
        </div>
      </header>
      <div className="auth-shell">
        <section className="auth-card">{children}</section>
      </div>
      <footer className="auth-footer"><Link href="/privacy">Конфиденциальность</Link><Link href="/terms">Условия использования</Link></footer>
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
      const result = await authApi.register({ displayName, email, password, role, acceptTerms });
      router.push(result.user.email_verification_required && !result.user.email_verified ? `/verify-email?sent=${result.verification_email_sent ? "1" : "0"}` : authApi.safeReturnTo(search.get("returnTo")));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Регистрация не удалась");
    } finally {
      setLoading(false);
    }
  }

  return (
    <PublicShell actionText="Уже есть аккаунт?" actionLabel="Войти" actionHref="/login">
      <p className="eyebrow">Добро пожаловать</p>
      <h1>Создайте аккаунт</h1>
      <p className="muted">Для учеников и преподавателей</p>
      <form onSubmit={submit} className="form-stack">
        <fieldset className="role-grid">
          <legend>Кто вы?</legend>
          <button type="button" aria-pressed={role === "student"} className={role === "student" ? "role-card active" : "role-card"} onClick={() => setRole("student")}>
            <span className="role-icon"><Icon name="book" size={36} /></span>
            <strong>Я ученик</strong>
            {role === "student" ? <span className="role-check"><Icon name="check" size={16} /></span> : null}
          </button>
          <button type="button" aria-pressed={role === "tutor"} className={role === "tutor" ? "role-card active" : "role-card"} onClick={() => setRole("tutor")}>
            <span className="role-icon"><Icon name="monitor" size={36} /></span>
            <strong>Я учитель</strong>
            {role === "tutor" ? <span className="role-check"><Icon name="check" size={16} /></span> : null}
          </button>
        </fieldset>
        <label>Имя и фамилия<input autoComplete="name" value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="Как к вам обращаться" required /></label>
        <label>Электронная почта<input autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="name@example.com" type="email" required /></label>
        <label>
          Пароль
          <span className="password-row">
            <input autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Не менее 8 символов" type={showPassword ? "text" : "password"} minLength={8} required />
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
      </form>
      <p className="centered"><Link className="button secondary" href="/login">Уже есть аккаунт? Войти</Link></p>
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
      const result = await authApi.login({ email, password });
      router.push(result.user.email_verification_required && !result.user.email_verified ? "/verify-email" : authApi.safeReturnTo(search.get("returnTo")));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Проверьте email и пароль");
    } finally {
      setLoading(false);
    }
  }

  return (
    <PublicShell actionText="Нет аккаунта?" actionLabel="Создать" actionHref="/register">
      <p className="eyebrow">Вход</p>
      <h1>Вход в аккаунт</h1>
      <form onSubmit={submit} className="form-stack">
        <label>Электронная почта<input autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} type="email" required /></label>
        <label>Пароль<input autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} type="password" required /></label>
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
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    try {
      await authApi.forgotPassword(email);
      setDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отправить письмо");
    }
  }

  return (
    <PublicShell actionText="Вспомнили пароль?" actionLabel="Войти" actionHref="/login">
      <h1>Восстановление пароля</h1>
      {done ? <p className="notice">Если аккаунт существует, письмо с кодом отправлено. <Link href="/reset-password">Ввести код</Link></p> : null}
      <form onSubmit={submit} className="form-stack">
        <label>Электронная почта<input autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} type="email" required /></label>
        {error ? <p className="field-error">{error}</p> : null}
        <button type="submit">Отправить инструкцию</button>
      </form>
    </PublicShell>
  );
}

export function ResetPasswordPage() {
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    try {
      await authApi.resetPassword(token, password);
      setDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сменить пароль");
    }
  }

  return (
    <PublicShell actionText="Вернуться ко входу" actionLabel="Войти" actionHref="/login">
      <h1>Новый пароль</h1>
      {done ? <p className="notice">Пароль изменён. <Link href="/login">Войти</Link></p> : null}
      <form onSubmit={submit} className="form-stack">
        <label>Код восстановления<input autoComplete="one-time-code" value={token} onChange={(event) => setToken(event.target.value)} required /></label>
        <label>Новый пароль<input autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} type="password" minLength={8} required /></label>
        {error ? <p className="field-error">{error}</p> : null}
        <button type="submit">Сохранить пароль</button>
      </form>
    </PublicShell>
  );
}

export function VerifyEmailPage() {
  const router=useRouter(),search=useSearchParams();
  const [user,setUser]=useState<User|null>(null),[ready,setReady]=useState(false),[token,setToken]=useState(""),[busy,setBusy]=useState(false),[done,setDone]=useState(false),[error,setError]=useState(search.get("sent")==="0"?"Аккаунт создан, но письмо не отправилось. Попробуйте отправить его ещё раз.":""),[notice,setNotice]=useState(search.get("sent")==="1"?"Письмо отправлено. Проверьте входящие и папку «Спам».":""),[cooldown,setCooldown]=useState(0);
  useEffect(()=>{
    const value=new URLSearchParams(window.location.hash.slice(1)).get("token");
    if(value)history.replaceState(null,"","/verify-email");
    productApi.me().then(user=>{if(value)setToken(value);setUser(user);if(user.email_verified){setDone(true);setError("");}}).catch(err=>{if(!(err instanceof ApiError&&err.status===401))setError("Не удалось загрузить аккаунт. Обновите страницу.");}).finally(()=>setReady(true));
  },[]);
  useEffect(()=>{if(cooldown<=0)return;const timer=setTimeout(()=>setCooldown(value=>Math.max(0,value-1)),1000);return()=>clearTimeout(timer);},[cooldown]);
  async function confirm(event:FormEvent){event.preventDefault();setBusy(true);setError("");try{const result=await authApi.verifyEmail(token);setUser(result.user);setDone(true);setToken("");}catch{setError("Ссылка недействительна, истекла или относится к другому аккаунту. Запросите новое письмо.");}finally{setBusy(false);}}
  async function resend(){setBusy(true);setError("");setNotice("");try{const result=await authApi.resendVerification();if(result.status==="verified"){setDone(true);return;}setNotice("Письмо отправлено. Ссылка действует один час.");setCooldown(60);}catch(err){if(err instanceof ApiError&&err.status===409){setError("Письмо уже запрашивали. Подождите минуту перед повторной отправкой.");setCooldown(60);}else setError("Не удалось отправить письмо. Повторите позже; если ошибка остаётся, сообщите администратору.");}finally{setBusy(false);}}
  return <PublicShell actionText="Другой аккаунт?" actionLabel="Войти" actionHref="/login"><p className="eyebrow">Безопасность аккаунта</p><h1>{done?"Почта подтверждена":"Подтвердите почту"}</h1>{!ready?<p role="status">Загружаем...</p>:done?<><p>Теперь можно перейти к занятиям.</p><button onClick={()=>router.push("/")}>Открыть кабинет</button></>:!user?<p>Войдите в аккаунт, для которого пришло письмо, затем снова откройте ссылку из письма. <Link href="/login?returnTo=%2Fverify-email">Войти</Link></p>:<div className="form-stack"><p className="muted">Отправим ссылку на <strong>{user.email}</strong>. Откройте её в этом браузере и подтвердите адрес.</p>{notice?<p className="notice" role="status">{notice}</p>:null}{token?<form onSubmit={confirm}><button disabled={busy} type="submit">{busy?"Подтверждаем...":"Подтвердить электронную почту"}</button></form>:null}<button className="secondary" disabled={busy||cooldown>0} onClick={resend}>{cooldown>0?`Повторить через ${cooldown} с`:busy?"Отправляем...":"Отправить письмо ещё раз"}</button></div>}{error?<p role="alert" className="field-error">{error}</p>:null}</PublicShell>;
}
