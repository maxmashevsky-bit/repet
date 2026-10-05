"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useState } from "react";
import { productApi } from "../../shared/api/client";
import type { Lesson } from "../../shared/api/types";
import { dateInTimeZone, formatDateTime, formatTime, timeInTimeZone, todayISO } from "../../shared/lib/date";
import { useWorkspace } from "../app-shell/AppShell";
import { Icon } from "../ui/Icon";

type ViewMode = "day" | "week" | "month";
const modes: ViewMode[] = ["day", "week", "month"];
const weekdays = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];

function dateKey(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

export function CalendarPage() {
  const { dashboard, refresh } = useWorkspace();
  const search = useSearchParams();
  const router = useRouter();
  const view = normalizeView(search.get("view"));
  const requestedDate = search.get("date") ?? todayISO(dashboard.user.timezone);
  const parsedDate = new Date(`${requestedDate}T12:00:00`);
  const date = Number.isNaN(parsedDate.getTime()) ? new Date() : parsedDate;
  const [showForm, setShowForm] = useState(false);
  const [relationID, setRelationID] = useState(dashboard.relations[0]?.id ?? "");
  const [title, setTitle] = useState("");
  const [startsAt, setStartsAt] = useState("");
  const [endsAt, setEndsAt] = useState("");
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<Lesson | null>(null);
  const firstDay = new Date(date.getFullYear(), date.getMonth(), 1);
  const gridStart = new Date(firstDay);
  gridStart.setDate(firstDay.getDate() - (firstDay.getDay() + 6) % 7);
  const monthDays = Array.from({ length: 42 }, (_, index) => {
    const day = new Date(gridStart);
    day.setDate(gridStart.getDate() + index);
    return day;
  });
  const monday = new Date(date);
  monday.setDate(date.getDate() - (date.getDay() + 6) % 7);
  const weekDays = Array.from({ length: 7 }, (_, index) => {
    const day = new Date(monday);
    day.setDate(monday.getDate() + index);
    return day;
  });
  const visibleLessons = dashboard.lessons.filter((lesson) => {
    const lessonDate = dateInTimeZone(lesson.starts_at, dashboard.user.timezone);
    return view === "day" ? lessonDate === dateKey(date) : view === "week" ? weekDays.some((item) => dateKey(item) === lessonDate) : lessonDate.slice(0, 7) === dateKey(date).slice(0, 7);
  });
  const titleDate = new Intl.DateTimeFormat("ru-RU", view === "day" ? { day: "numeric", month: "long", year: "numeric" } : { month: "long", year: "numeric" }).format(date);
  const weekTitle = `${new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "short" }).format(weekDays[0])} – ${new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "short", year: "numeric" }).format(weekDays[6])}`;

  function navigate(nextDate: Date, nextView = view) {
    router.push(`/calendar?view=${nextView}&date=${dateKey(nextDate)}`);
  }

  function shift(direction: number) {
    const next = new Date(date);
    if (view === "month") next.setMonth(next.getMonth() + direction);
    else next.setDate(next.getDate() + direction * (view === "week" ? 7 : 1));
    navigate(next);
  }

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    try {
      const lesson = await productApi.createLesson({ relationID, title, startsAt: new Date(startsAt).toISOString(), endsAt: new Date(endsAt).toISOString() });
      await refresh();
      setShowForm(false);
      setTitle("");
      navigate(new Date(`${dateInTimeZone(lesson.starts_at, dashboard.user.timezone)}T12:00:00`), "day");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось создать занятие");
    }
  }

  return (
    <section className="calendar-layout">
      <div className="page-title full"><h1>Календарь</h1><p>Занятия и важные даты.</p></div>
      <aside className="panel calendar-side">
        {dashboard.user.role === "tutor" ? <button className="terracotta add-event" type="button" onClick={() => setShowForm((value) => !value)}><Icon name="plus" />Добавить занятие</button> : null}
        {showForm ? <form className="form-stack" onSubmit={create}>
          <label>Ученик<select value={relationID} onChange={(event) => setRelationID(event.target.value)} required>{dashboard.relations.map((relation) => <option key={relation.id} value={relation.id}>{dashboard.conversations.find((item) => item.student_id === relation.student_id)?.student_name ?? relation.student_id}</option>)}</select></label>
          <label>Тема<input value={title} onChange={(event) => setTitle(event.target.value)} required /></label>
          <label>Начало (время устройства)<input type="datetime-local" value={startsAt} onChange={(event) => setStartsAt(event.target.value)} required /></label>
          <label>Конец (время устройства)<input type="datetime-local" value={endsAt} onChange={(event) => setEndsAt(event.target.value)} required /></label>
          {error ? <p className="field-error">{error}</p> : null}
          <button type="submit" disabled={!relationID}>Создать</button>
        </form> : null}
        <h2>{new Intl.DateTimeFormat("ru-RU", { month: "long", year: "numeric" }).format(date)}</h2>
        <div className="mini-calendar"><div className="mini-weekdays">{weekdays.map((day) => <b key={day}>{day}</b>)}</div>{monthDays.map((day) => <button type="button" key={dateKey(day)} className={dateKey(day) === dateKey(date) ? "active" : ""} onClick={() => navigate(day, "day")}>{day.getMonth() === date.getMonth() ? day.getDate() : ""}</button>)}</div>
      </aside>
      <section className="panel calendar-main">
        <div className="calendar-toolbar">
          <div className="date-navigation"><button type="button" className="secondary" onClick={() => navigate(new Date(`${todayISO(dashboard.user.timezone)}T12:00:00`))}>Сегодня</button><button type="button" className="icon-button" aria-label="Предыдущий период" onClick={() => shift(-1)}><Icon name="chevron" className="back" /></button><button type="button" className="icon-button" aria-label="Следующий период" onClick={() => shift(1)}><Icon name="chevron" /></button></div>
          <h2>{view === "week" ? weekTitle : titleDate}</h2>
          <div className="segmented">{modes.map((mode) => <button key={mode} type="button" className={view === mode ? "active" : ""} onClick={() => navigate(date, mode)}>{modeLabel(mode)}</button>)}</div>
        </div>
        {view === "month" ? <div className="month-grid">{monthDays.map((day) => <div key={dateKey(day)} className="month-cell"><strong>{day.getDate()}</strong>{visibleLessons.filter((lesson) => dateInTimeZone(lesson.starts_at, dashboard.user.timezone) === dateKey(day)).map((lesson) => <button className="event-pill" type="button" key={lesson.id} onClick={() => setSelected(lesson)}>{formatTime(lesson.starts_at, dashboard.user.timezone)} {lesson.title}</button>)}</div>)}</div> :
          <div className={view === "week" ? "time-grid week" : "time-grid"}>
            {view === "week" ? <div className="week-header">{weekDays.map((day, index) => <strong key={dateKey(day)}>{weekdays[index]} {day.getDate()}</strong>)}</div> : null}
            {Array.from({ length: 24 }, (_, index) => index).map((hour) => <span key={hour} className="hour">{hour}:00</span>)}
            {visibleLessons.map((lesson, index) => {
              const lessonDate = new Date(`${dateInTimeZone(lesson.starts_at, dashboard.user.timezone)}T12:00:00`);
              const dayIndex = (lessonDate.getDay() + 6) % 7;
              const start = timeInTimeZone(lesson.starts_at, dashboard.user.timezone);
              const top = (view === "week" ? 62 : 0) + start.hour * 56 + Math.round(start.minute / 60 * 56);
              return <button key={lesson.id} className={`calendar-event tone-${index % 4}`} style={view === "week" ? { top, left: `calc(72px + ${dayIndex * (100 / 7)}% - ${dayIndex * (72 / 7)}px)`, width: "calc(14.285% - 14px)" } : { top }} type="button" onClick={() => setSelected(lesson)}><strong>{formatTime(lesson.starts_at, dashboard.user.timezone)} - {formatTime(lesson.ends_at, dashboard.user.timezone)}</strong><span>{lesson.title}</span></button>;
            })}
          </div>}
        {selected ? <div className="notice"><strong>{selected.title}</strong><p>{formatDateTime(selected.starts_at, dashboard.user.timezone)} – {formatDateTime(selected.ends_at, dashboard.user.timezone)}</p><button type="button" className="secondary" onClick={() => setSelected(null)}>Закрыть</button></div> : null}
        {visibleLessons.length === 0 ? <p className="empty">В этом периоде занятий нет.</p> : null}
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
