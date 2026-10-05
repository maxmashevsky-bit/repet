"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { productApi } from "../../shared/api/client";
import { useWorkspace } from "../app-shell/AppShell";
import { dateInTimeZone, formatDateTime, formatTime, todayISO } from "../../shared/lib/date";
import { BrandMark } from "../ui/Brand";
import { Icon } from "../ui/Icon";

const weekDays = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];

export function DashboardPage() {
  const { dashboard } = useWorkspace();
  return dashboard.user.role === "tutor" ? <TeacherDashboard /> : <StudentDashboard />;
}

function LessonHero({ tutor = false }: { tutor?: boolean }) {
  const { dashboard } = useWorkspace();
  const now = new Date();
  const lesson = dashboard.lessons.filter((item) => new Date(item.ends_at) >= now).sort((a, b) => a.starts_at.localeCompare(b.starts_at))[0];

  return (
    <article className="hero-card dashboard-hero">
      <BrandMark decorative />
      <div className="hero-content">
        <p className="eyebrow">{tutor ? "Следующее занятие" : "Ближайшее занятие"}</p>
        <h2>{lesson?.title ?? "Расписание свободно"}</h2>
        {lesson ? (
          <>
            <p className="hero-meta"><Icon name="calendar" size={19} />{formatDateTime(lesson.starts_at, dashboard.user.timezone)}</p>
            <p className="hero-topic">{tutor ? "Ученик и тема доступны в календаре" : "Все детали занятия собраны в одном месте"}</p>
          </>
        ) : <p className="hero-topic">{tutor ? "Добавьте занятие в календаре." : "Новое занятие появится здесь сразу после назначения."}</p>}
        <div className="actions hero-actions">
          <Link className="button hero-primary" href={lesson ? `/calendar?view=day&date=${dateInTimeZone(lesson.starts_at, dashboard.user.timezone)}` : "/calendar?view=day"}>{tutor ? "Открыть расписание" : "Открыть занятие"}<Icon name="arrow" size={19} /></Link>
          <Link className="button hero-secondary" href={tutor ? "/tasks" : "/messages"}>{tutor ? "Проверить работы" : "Написать преподавателю"}</Link>
        </div>
      </div>
    </article>
  );
}

function TodayCard() {
  const { dashboard } = useWorkspace();
  const today = new Date(`${todayISO(dashboard.user.timezone)}T12:00:00`);
  const todayLessons = dashboard.lessons.filter((lesson) => dateInTimeZone(lesson.starts_at, dashboard.user.timezone) === todayISO(dashboard.user.timezone)).sort((a, b) => a.starts_at.localeCompare(b.starts_at));
  return (
    <aside className="panel today-card">
      <div className="section-head">
        <div><h2>Сегодня</h2><p className="muted">{new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "long", weekday: "long" }).format(today)}</p></div>
        <span className="soft-icon"><Icon name="calendar" /></span>
      </div>
      <div className="today-list">
        {todayLessons.length === 0 ? <p className="empty">На сегодня событий нет.</p> : todayLessons.slice(0, 3).map((lesson, index) => (
          <div key={lesson.id} className="today-row">
            <span className={index % 2 ? "event-dot orange" : "event-dot"} />
            <strong>{formatTime(lesson.starts_at, dashboard.user.timezone)}</strong>
            <span><b>{lesson.title}</b><small>{formatTime(lesson.ends_at, dashboard.user.timezone)} · занятие</small></span>
          </div>
        ))}
      </div>
      <Link className="inline-link arrow-link" href="/calendar">Открыть календарь <Icon name="arrow" size={18} /></Link>
    </aside>
  );
}

function WeekStrip() {
  const { dashboard } = useWorkspace();
  const monday = new Date(`${todayISO(dashboard.user.timezone)}T12:00:00`);
  monday.setDate(monday.getDate() - (monday.getDay() + 6) % 7);
  return (
    <article className="panel week-card">
      <div className="section-head"><h2>Расписание</h2><Link className="inline-link arrow-link" href="/calendar?view=week">Полный календарь <Icon name="arrow" size={18} /></Link></div>
      <div className="week-strip">
        {weekDays.map((day, index) => {
          const date = new Date(monday);
          date.setDate(monday.getDate() + index);
          const lesson = dashboard.lessons.find((item) => dateInTimeZone(item.starts_at, dashboard.user.timezone) === `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`);
          return (
            <div className={date.toDateString() === new Date(`${todayISO(dashboard.user.timezone)}T12:00:00`).toDateString() ? "week-day active" : "week-day"} key={day}>
              <span>{day}</span><strong>{date.getDate()}</strong>
              {lesson ? <small>{formatTime(lesson.starts_at, dashboard.user.timezone)}<b>{lesson.title}</b></small> : <small className="free">—</small>}
            </div>
          );
        })}
      </div>
    </article>
  );
}

function StudentDashboard() {
  const { dashboard } = useWorkspace();
  const currentTask = dashboard.assignments.find((item) => item.status === "assigned" || item.status === "revision");
  const lastConversation = dashboard.conversations[0];

  return (
    <section className="dashboard-page">
      <div className="page-title">
        <h1>Добрый день, {dashboard.user.display_name}!</h1>
        <p>Вот что важно сегодня</p>
      </div>
      <div className="dashboard-primary">
        <LessonHero />
        <TodayCard />
      </div>
      <WeekStrip />
      <div className="dashboard-cards">
        <article className="panel summary-card task-summary">
          <div className="summary-heading"><span className="soft-icon terracotta-soft"><Icon name="task" /></span><h2>Нужно сделать</h2></div>
          {currentTask ? (
            <>
              <strong className="summary-title">{currentTask.title}</strong>
              <p>{currentTask.topic}</p>
              <p className="deadline"><Icon name="clock" size={18} />До {formatDateTime(currentTask.due_at, dashboard.user.timezone)}</p>
              <Link className="button terracotta" href={`/tasks?task=${currentTask.id}`}>Продолжить <Icon name="arrow" size={18} /></Link>
            </>
          ) : <p className="empty">Все задания закрыты. Отличная работа.</p>}
        </article>
        <article className="panel summary-card">
          <div className="summary-heading"><span className="soft-icon"><Icon name="chat" /></span><h2>Сообщение преподавателя</h2></div>
          {lastConversation ? <><strong className="summary-title">{lastConversation.tutor_name}</strong><p>{lastConversation.last_message || "Диалог готов к общению."}</p></> : <p className="empty">Диалог появится после подтверждения связи.</p>}
          <Link className="inline-link arrow-link" href="/messages">Ответить <Icon name="arrow" size={18} /></Link>
        </article>
        <article className="panel summary-card">
          <div className="summary-heading"><span className="soft-icon"><Icon name="chart" /></span><h2>Сейчас изучаем</h2></div>
          <strong className="summary-title">{currentTask?.topic || "Учебный план"}</strong>
          <p className="muted">Актуальные задания доступны в разделе «Задания».</p>
        </article>
      </div>
    </section>
  );
}

function TeacherDashboard() {
  const { dashboard } = useWorkspace();
  const [studentEmail, setStudentEmail] = useState("");
  const [studentName, setStudentName] = useState("");
  const [inviteLink, setInviteLink] = useState("");
  const [inviteError, setInviteError] = useState("");
  async function invite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviteError("");
    try {
      const result = await productApi.createInvitation(studentEmail, studentName);
      setInviteLink(`${window.location.origin}${result.accept_url}`);
    } catch (err) {
      setInviteError(err instanceof Error ? err.message : "Не удалось создать приглашение");
    }
  }
  const review = dashboard.assignments.filter((item) => item.status === "submitted");
  const todayLessons = dashboard.lessons.filter((lesson) => dateInTimeZone(lesson.starts_at, dashboard.user.timezone) === todayISO(dashboard.user.timezone));
  const students = dashboard.relations.map((relation, index) => ({
    ...relation,
    name: dashboard.conversations.find((item) => item.student_id === relation.student_id)?.student_name || `Ученик ${index + 1}`
  }));

  return (
    <section className="dashboard-page teacher-dashboard">
      <div className="page-title">
        <h1>Добрый день, {dashboard.user.display_name}!</h1>
        <p>Сегодня у вас {todayLessons.length} занятий и {review.length} работ на проверку.</p>
      </div>
      <div className="dashboard-primary">
        <LessonHero tutor />
        <TodayCard />
      </div>
      <div className="teacher-overview">
        <article className="panel students-panel">
          <div className="section-head"><div className="heading-with-count"><h2>Мои ученики</h2><span>{students.length} активных</span></div><Link className="inline-link arrow-link" href="/messages">Все ученики <Icon name="arrow" size={18} /></Link></div>
          <form className="form-stack" onSubmit={invite}>
            <label>Email ученика<input type="email" value={studentEmail} onChange={(event) => setStudentEmail(event.target.value)} required /></label>
            <label>Имя ученика<input value={studentName} onChange={(event) => setStudentName(event.target.value)} /></label>
            <button type="submit">Пригласить ученика</button>
          </form>
          {inviteLink ? <p className="notice">Ссылка для ученика: <a href={inviteLink}>{inviteLink}</a></p> : null}
          {inviteError ? <p className="field-error">{inviteError}</p> : null}
          <div className="student-strip">
            {students.length === 0 ? <p className="empty">Связи появятся после приглашения учеников.</p> : students.slice(0, 4).map((student) => (
              <div className="mini-card student-card" key={student.id}>
                <div className="student-name"><span className="avatar small">{student.name.slice(0, 1)}</span><strong>{student.name}</strong></div>
                <Link className="inline-link" href={`/messages?conversation=${dashboard.conversations.find((item) => item.student_id === student.student_id)?.id ?? ""}`}><Icon name="chat" size={17} /> Написать</Link>
              </div>
            ))}
          </div>
        </article>
        <article className="panel review-panel">
          <div className="heading-with-count"><h2>Нужно проверить</h2><span className="count-badge">{review.length}</span></div>
          {review.length === 0 ? <p className="empty">Новых работ нет.</p> : review.slice(0, 3).map((item, index) => (
            <div key={item.id} className="review-line"><span className={index === 1 ? "soft-icon terracotta-soft" : "soft-icon"}><Icon name="task" size={19} /></span><span><strong>{item.student_name}</strong><small>{item.title}</small></span><em>{item.status === "revision" ? "Исправления" : "Новая работа"}</em></div>
          ))}
          <Link className="button terracotta" href="/tasks?status=submitted">Перейти к проверке <Icon name="arrow" size={18} /></Link>
        </article>
      </div>
      <div className="dashboard-cards teacher-stats">
        <article className="panel summary-card"><h2>Новые сообщения</h2>{dashboard.conversations.slice(0, 2).map((item) => <p className="compact-row" key={item.id}><strong>{item.student_name}</strong><span>{item.last_message || "Открыть диалог"}</span></p>)}<Link className="inline-link arrow-link" href="/messages">Все сообщения <Icon name="arrow" size={18} /></Link></article>
        <article className="panel summary-card"><h2>Эта неделя</h2><div className="big-stats"><span><strong>{dashboard.lessons.length}</strong>занятий</span><span><strong>{review.length}</strong>работ</span></div></article>
        <article className="panel summary-card attention-card"><h2>Требует внимания</h2><p>Проверьте просроченные работы и обратную связь ученикам.</p><Link className="button secondary" href="/tasks">Открыть задания <Icon name="arrow" size={18} /></Link></article>
      </div>
    </section>
  );
}
