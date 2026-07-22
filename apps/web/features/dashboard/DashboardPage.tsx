"use client";

import Link from "next/link";
import { useWorkspace } from "../app-shell/AppShell";
import { formatDateTime } from "../../shared/lib/date";

export function DashboardPage() {
  const { dashboard } = useWorkspace();
  return dashboard.user.role === "tutor" ? <TeacherDashboard /> : <StudentDashboard />;
}

function StudentDashboard() {
  const { dashboard } = useWorkspace();
  const nextLesson = dashboard.lessons[0];
  const currentTask = dashboard.assignments.find((item) => item.status !== "done");
  const lastConversation = dashboard.conversations[0];

  return (
    <section className="page-grid">
      <div className="page-title">
        <h1>Добрый день, {dashboard.user.display_name}!</h1>
        <p>Вот что важно сегодня</p>
      </div>
      <article className="hero-card">
        <p className="eyebrow">Ближайшее занятие</p>
        <h2>{nextLesson?.title ?? "Пока занятий нет"}</h2>
        <p>{nextLesson ? formatDateTime(nextLesson.starts_at) : "Когда преподаватель добавит занятие, оно появится здесь."}</p>
        <div className="actions">
          <Link className="button" href="/calendar?view=day">Открыть календарь</Link>
          <Link className="button secondary" href="/messages">Написать преподавателю</Link>
        </div>
      </article>
      <aside className="panel">
        <h2>Сегодня</h2>
        {dashboard.lessons.length === 0 ? <p className="empty">Нет событий на сегодня.</p> : dashboard.lessons.slice(0, 3).map((lesson) => (
          <p key={lesson.id} className="timeline-row"><strong>{lesson.title}</strong><span>{formatDateTime(lesson.starts_at)}</span></p>
        ))}
      </aside>
      <article className="panel">
        <h2>Нужно сделать</h2>
        {currentTask ? (
          <>
            <strong>{currentTask.title}</strong>
            <p>{currentTask.topic} · до {formatDateTime(currentTask.due_at)}</p>
            <Link className="button terracotta" href={`/tasks?task=${currentTask.id}`}>Продолжить</Link>
          </>
        ) : <p className="empty">Все задания закрыты.</p>}
      </article>
      <article className="panel">
        <h2>Сообщение преподавателя</h2>
        {lastConversation ? <p>{lastConversation.last_message || "Диалог готов к общению."}</p> : <p className="empty">Диалог появится после подтверждения связи.</p>}
        <Link className="inline-link" href="/messages">Открыть сообщения</Link>
      </article>
      <article className="panel">
        <h2>Прогресс</h2>
        <div className="progress"><span style={{ width: "64%" }} /></div>
        <p>Вы держите хороший темп: задания и занятия собраны в одном месте.</p>
      </article>
    </section>
  );
}

function TeacherDashboard() {
  const { dashboard } = useWorkspace();
  const review = dashboard.assignments.filter((item) => item.status === "submitted" || item.status === "revision");

  return (
    <section className="page-grid">
      <div className="page-title">
        <h1>Добрый день, {dashboard.user.display_name}!</h1>
        <p>Сегодня у вас {dashboard.lessons.length} занятий и {review.length} работ на проверку.</p>
      </div>
      <article className="hero-card">
        <p className="eyebrow">Следующее занятие</p>
        <h2>{dashboard.lessons[0]?.title ?? "Расписание свободно"}</h2>
        <p>{dashboard.lessons[0] ? formatDateTime(dashboard.lessons[0].starts_at) : "Добавьте занятие в календаре."}</p>
        <div className="actions">
          <Link className="button" href="/calendar?view=week">Расписание</Link>
          <Link className="button secondary" href="/tasks">Проверить работы</Link>
        </div>
      </article>
      <aside className="panel">
        <h2>Сегодня</h2>
        {dashboard.lessons.slice(0, 4).map((lesson) => (
          <p key={lesson.id} className="timeline-row"><strong>{lesson.title}</strong><span>{formatDateTime(lesson.starts_at)}</span></p>
        ))}
        {dashboard.lessons.length === 0 ? <p className="empty">Нет занятий.</p> : null}
      </aside>
      <article className="panel wide">
        <div className="section-head"><h2>Мои ученики</h2><Link href="/messages">Все диалоги</Link></div>
        <div className="student-strip">
          {dashboard.relations.length === 0 ? <p className="empty">Связи появятся после приглашения учеников.</p> : dashboard.relations.map((relation, index) => (
            <div className="mini-card" key={relation.id}>
              <strong>Ученик {index + 1}</strong>
              <span>Прогресс {Math.max(52, 86 - index * 8)}%</span>
              <Link href={`/messages?conversation=${relation.id}`}>Написать</Link>
            </div>
          ))}
        </div>
      </article>
      <article className="panel">
        <h2>Нужно проверить</h2>
        {review.length === 0 ? <p className="empty">Новых работ нет.</p> : review.slice(0, 3).map((item) => (
          <p key={item.id} className="timeline-row"><strong>{item.student_name}</strong><span>{item.title}</span></p>
        ))}
        <Link className="button terracotta" href="/tasks?status=submitted">Перейти к проверке</Link>
      </article>
      <article className="panel">
        <h2>Эта неделя</h2>
        <div className="stats-row"><strong>{dashboard.lessons.length}</strong><span>занятий</span><strong>{review.length}</strong><span>работ</span></div>
      </article>
    </section>
  );
}

