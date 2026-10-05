"use client";

import { FormEvent, useState } from "react";
import { authApi, productApi } from "../../shared/api/client";
import { useWorkspace } from "../app-shell/AppShell";
import { Icon } from "../ui/Icon";

export function SettingsPage() {
  const { dashboard, refresh } = useWorkspace();
  const [displayName, setDisplayName] = useState(dashboard.user.display_name);
  const [timezone, setTimezone] = useState(dashboard.user.timezone);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    try {
      await productApi.updateProfile({ displayName, timezone, locale: dashboard.user.locale });
      await refresh();
      setMessage("Изменения сохранены");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить профиль");
    }
  }

  async function logout() {
    await authApi.logout();
    window.location.href = "/login";
  }

  return (
    <section className="settings-layout">
      <div className="page-title full"><h1>Личный кабинет</h1><p>Профиль и настройки.</p></div>
      <aside className="panel settings-nav" aria-label="Разделы настроек">
        <a className="active" href="#profile"><Icon name="user" />Профиль</a>
        <a href="#security"><Icon name="lock" />Безопасность</a>
        <a href="#privacy"><Icon name="shield" />Приватность</a>
        <button type="button" className="danger-link" onClick={logout}>Выйти</button>
      </aside>
      <form id="profile" className="panel form-stack" onSubmit={save}>
        <h2>Профиль</h2>
        <div className="profile-editor"><span className="avatar profile-avatar">{dashboard.user.display_name.slice(0, 1)}</span><span><strong>{dashboard.user.display_name}</strong><small>{dashboard.user.role === "tutor" ? "Преподаватель" : "Ученик"}</small></span></div>
        <label>Имя<input value={displayName} onChange={(event) => setDisplayName(event.target.value)} required /></label>
        <label>Email<input value={dashboard.user.email} disabled /></label>
        <label>Часовой пояс<input value={timezone} onChange={(event) => setTimezone(event.target.value)} required /></label>
        <button type="submit">Сохранить изменения</button>
        {error ? <p className="field-error">{error}</p> : null}
        {message ? <p className="status-ok">{message}</p> : null}
      </form>
      <section id="security" className="panel">
        <h2>Безопасность</h2>
        <p>Чтобы завершить сеанс на этом устройстве, выйдите из аккаунта.</p>
      </section>
      <section id="privacy" className="panel">
        <h2>Приватность</h2>
        <p>Роль нельзя изменить как обычную настройку. Для смены роли нужен отдельный проверяемый процесс.</p>
        <a href="/privacy">Политика конфиденциальности</a>
      </section>
    </section>
  );
}
