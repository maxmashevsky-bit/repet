"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useMemo } from "react";
import { formatTime, todayISO } from "../../shared/lib/date";
import { useWorkspace } from "../app-shell/AppShell";

type ViewMode = "day" | "week" | "month";

const modes: ViewMode[] = ["day", "week", "month"];

export function CalendarPage() {
  const { dashboard } = useWorkspace();
  const search = useSearchParams();
  const router = useRouter();
  const view = normalizeView(search.get("view"));
  const date = search.get("date") ?? todayISO();
  const hours = Array.from({ length: 14 }, (_, index) => index + 8);
  const monthDays = Array.from({ length: 35 }, (_, index) => index + 1);

  const title = useMemo(() => new Intl.DateTimeFormat("ru-RU", { month: "long", year: "numeric" }).format(new Date(date)), [date]);

  function setView(next: ViewMode) {
    router.push(`/calendar?view=${next}&date=${date}`);
  }

  return (
    <section className="calendar-layout">
      <div className="page-title full"><h1>Календарь</h1><p>Занятия, задания и важные даты в одном месте.</p></div>
      <aside className="panel calendar-side">
        <button className="terracotta" type="button">Добавить занятие</button>
        <h2>{title}</h2>
        <div className="mini-calendar">{monthDays.slice(0, 31).map((day) => <span key={day} className={day === Number(date.slice(8, 10)) ? "active" : ""}>{day}</span>)}</div>
        {dashboard.user.role === "tutor" ? (
          <div className="filters"><h2>Ученики</h2>{dashboard.relations.map((relation, index) => <label key={relation.id}><input type="checkbox" defaultChecked /> Ученик {index + 1}</label>)}</div>
        ) : (
          <div className="filters"><h2>Мой календарь</h2><label><input type="checkbox" defaultChecked /> Занятия</label><label><input type="checkbox" defaultChecked /> Домашние задания</label></div>
        )}
      </aside>
      <section className="panel calendar-main">
        <div className="calendar-toolbar">
          <button type="button" className="secondary">Сегодня</button>
          <h2>{view === "day" ? "Среда, 22 июля 2026" : view === "week" ? "20-26 июля 2026" : title}</h2>
          <div className="segmented">
            {modes.map((mode) => <button key={mode} type="button" className={view === mode ? "active" : ""} onClick={() => setView(mode)}>{modeLabel(mode)}</button>)}
          </div>
        </div>
        {view === "month" ? (
          <div className="month-grid">
            {monthDays.map((day) => (
              <div key={day} className="month-cell">
                <strong>{day}</strong>
                {dashboard.lessons.filter((_, index) => index % 7 === day % 7).slice(0, 2).map((lesson) => (
                  <span className="event-pill" key={`${day}-${lesson.id}`}>{formatTime(lesson.starts_at)} {lesson.title}</span>
                ))}
              </div>
            ))}
          </div>
        ) : (
          <div className={view === "week" ? "time-grid week" : "time-grid"}>
            {hours.map((hour) => <span key={hour} className="hour">{hour}:00</span>)}
            {dashboard.lessons.map((lesson, index) => (
              <button key={lesson.id} className={`calendar-event tone-${index % 4}`} style={{ top: `${72 + index * 72}px` }} type="button">
                <strong>{formatTime(lesson.starts_at)} - {formatTime(lesson.ends_at)}</strong>
                <span>{lesson.title}{dashboard.user.role === "tutor" ? " · ученик" : ""}</span>
              </button>
            ))}
          </div>
        )}
        {dashboard.lessons.length === 0 ? <p className="empty">Событий пока нет. Создайте урок через API или примите приглашение.</p> : null}
      </section>
    </section>
  );
}

function normalizeView(value: string | null): ViewMode {
  return value === "day" || value === "month" ? value : "week";
}

function modeLabel(mode: ViewMode): string {
  return mode === "day" ? "День" : mode === "week" ? "Неделя" : "Месяц";
}

