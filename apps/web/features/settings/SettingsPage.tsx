"use client";

import { FormEvent, useState } from "react";
import { authApi, productApi } from "../../shared/api/client";
import { useWorkspace } from "../app-shell/AppShell";
import { Icon } from "../ui/Icon";

export function SettingsPage() {
  const { dashboard, refresh } = useWorkspace();
  const [displayName, setDisplayName] = useState(dashboard.user.display_name);
  const [timezone, setTimezone] = useState("Europe/Moscow");
  const [locale, setLocale] = useState("ru");
  const [message, setMessage] = useState("");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await productApi.updateProfile({ displayName, timezone, locale });
    await refresh();
    setMessage("Изменения сохранены");
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
        <a href="#notifications"><Icon name="bell" />Уведомления</a>
        <a href="#privacy"><Icon name="shield" />Приватность</a>
        <a href="#appearance"><Icon name="palette" />Оформление</a>
        <a href="#language"><Icon name="globe" />Язык</a>
        <button type="button" className="danger-link" onClick={logout}>Выйти</button>
      </aside>
      <form id="profile" className="panel form-stack" onSubmit={save}>
        <h2>Профиль</h2>
        <div className="profile-editor"><span className="avatar profile-avatar">{dashboard.user.display_name.slice(0, 1)}</span><span><strong>{dashboard.user.display_name}</strong><small>{dashboard.user.role === "tutor" ? "Преподаватель" : "Ученик"}</small></span></div>
        <label>Имя<input value={displayName} onChange={(event) => setDisplayName(event.target.value)} required /></label>
        <label>Email<input value={dashboard.user.email} disabled /></label>
        <label>Часовой пояс<input value={timezone} onChange={(event) => setTimezone(event.target.value)} required /></label>
        <label>Язык<select value={locale} onChange={(event) => setLocale(event.target.value)}><option value="ru">Русский</option><option value="en">English</option></select></label>
        <button type="submit">Сохранить изменения</button>
        {message ? <p className="status-ok">{message}</p> : null}
      </form>
      <section id="security" className="panel">
        <h2>Безопасность</h2>
        <p className="settings-row"><span><Icon name="lock" />Изменить пароль</span><Icon name="chevron" /></p>
        <p className="settings-row"><span><Icon name="shield" />Двухэтапная проверка</span><b>Включена</b></p>
        <p className="settings-row"><span><Icon name="monitor" />Активные сессии</span><small>Этот компьютер · сейчас</small></p>
      </section>
      <section id="notifications" className="panel">
        <h2>Уведомления</h2>
        <label className="toggle-row">Сообщения преподавателя<input type="checkbox" defaultChecked /></label>
        <label className="toggle-row">Занятия за 15 минут<input type="checkbox" defaultChecked /></label>
        <label className="toggle-row">Задания и проверка<input type="checkbox" defaultChecked /></label>
      </section>
      <section id="privacy" className="panel">
        <h2>Приватность</h2>
        <p>Роль нельзя изменить как обычную настройку. Для смены роли нужен отдельный проверяемый процесс.</p>
        <button type="button" className="secondary">Запросить удаление аккаунта</button>
      </section>
    </section>
  );
}
