"use client";

import { FormEvent, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { productApi } from "../../shared/api/client";
import type { Assignment } from "../../shared/api/types";
import { formatDateTime } from "../../shared/lib/date";
import { useWorkspace } from "../app-shell/AppShell";
import { BrandMark } from "../ui/Brand";
import { Icon } from "../ui/Icon";

export function TasksPage() {
  const { dashboard } = useWorkspace();
  return dashboard.user.role === "tutor" ? <TeacherTasks /> : <StudentTasks />;
}

function StudentTasks() {
  const { dashboard, refresh } = useWorkspace();
	const search = useSearchParams();
	const [selectedID, setSelectedID] = useState(search.get("task") ?? dashboard.assignments[0]?.id ?? "");
	const selected = dashboard.assignments.find((item) => item.id === selectedID) ?? dashboard.assignments[0] ?? null;
  const [answer, setAnswer] = useState(selected?.answer ?? "");
  const [error, setError] = useState("");
	const [filter, setFilter] = useState("Все");
	const [query, setQuery] = useState("");
	const visible = dashboard.assignments.filter((item) => (filter === "Все" || filter === "Нужно сделать" && ["assigned", "revision"].includes(item.status) || filter === "На проверке" && item.status === "submitted" || filter === "Проверено" && item.status === "done") && item.title.toLowerCase().includes(query.toLowerCase()));

  async function submit() {
    if (!selected) return;
    setError("");
    try {
      await productApi.submitAssignment(selected.id, answer);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отправить работу");
    }
  }

  return (
    <section className="tasks-layout">
      <div className="page-title full"><h1>Задания</h1><p>Домашние работы и обратная связь преподавателя.</p></div>
      <div className="tasks-toolbar full">
        <div className="tabs">{["Нужно сделать", "На проверке", "Проверено", "Все"].map((item) => <button key={item} type="button" className={filter === item ? "active" : ""} onClick={() => setFilter(item)}>{item}</button>)}</div>
        <label className="search-field"><Icon name="search" size={19} /><input aria-label="Найти задание" placeholder="Найти задание" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
      </div>
      <article className="hero-card task-hero full">
        <BrandMark decorative />
        <div className="hero-content">
          <p className="eyebrow">Продолжить задание</p>
          <h2>{selected?.title ?? "Заданий пока нет"}</h2>
          <p className="hero-meta">{selected ? <><Icon name="user" size={18} />{selected.topic}<span>·</span><Icon name="calendar" size={18} />Сдать до {formatDateTime(selected.due_at, dashboard.user.timezone)}</> : "Когда преподаватель назначит работу, она появится здесь."}</p>
          {selected ? <div className="actions hero-actions"><a className="button hero-primary" href="#task-work">Открыть задание <Icon name="arrow" size={18} /></a></div> : null}
        </div>
      </article>
      <section className="panel">
        <h2>Мои задания</h2>
        {visible.length === 0 ? <p className="empty">Заданий нет.</p> : visible.map((item) => (
          <button key={item.id} type="button" className={selected?.id === item.id ? "task-row active" : "task-row"} onClick={() => { setSelectedID(item.id); setAnswer(item.answer); }}>
            <span className={item.status === "assigned" && new Date(item.due_at) < new Date() ? "soft-icon terracotta-soft" : "soft-icon"}><Icon name="task" size={19} /></span>
            <span className="task-copy"><strong>{item.title}</strong><small>{item.status} · до {formatDateTime(item.due_at, dashboard.user.timezone)}</small></span>
            <Icon name="chevron" size={18} />
          </button>
        ))}
      </section>
      <section className="panel" id="task-work">
        <h2>Работа</h2>
        {selected ? (
          <>
            <p>{selected.body}</p>
            <label>Ответ<textarea value={answer} onChange={(event) => setAnswer(event.target.value)} rows={7} /></label>
            {selected.feedback ? <p className="notice">Комментарий преподавателя: {selected.feedback}</p> : null}
            {error ? <p className="field-error">{error}</p> : null}
            {selected.score != null ? <p className="notice">Оценка: {selected.score} / {selected.max_score}</p> : null}
            {selected.status === "assigned" || selected.status === "revision" ? <button type="button" onClick={submit} disabled={!answer.trim()}>Отправить на проверку</button> : null}
          </>
        ) : <p className="empty">Выберите задание.</p>}
      </section>
    </section>
  );
}

function TeacherTasks() {
  const { dashboard, refresh } = useWorkspace();
  const [title, setTitle] = useState("");
  const [subject, setSubject] = useState("");
  const [topic, setTopic] = useState("");
  const [body, setBody] = useState("");
  const [dueAt, setDueAt] = useState("");
  const [maxScore, setMaxScore] = useState(100);
  const [studentID, setStudentID] = useState(dashboard.relations[0]?.student_id ?? "");
  const [error, setError] = useState("");
  const [reviewID, setReviewID] = useState("");
  const [score, setScore] = useState(0);
  const [feedback, setFeedback] = useState("");
  const [filter, setFilter] = useState("Все");
  const review = useMemo(() => dashboard.assignments.filter((item) => item.status === "submitted"), [dashboard.assignments]);

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    try {
      await productApi.createAssignment({
        studentID,
        title,
        subject,
        topic,
        body,
        dueAt: new Date(dueAt).toISOString(),
        maxScore
      });
      await refresh();
      setTitle("");
      setBody("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось создать задание");
    }
  }

  async function grade(item: Assignment, revision: boolean) {
    setError("");
    try {
      await productApi.gradeAssignment(item.id, score, feedback, revision);
      await refresh();
      setReviewID("");
      setFeedback("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось проверить работу");
    }
  }

  return (
    <section className="tasks-layout teacher">
      <div className="page-title full page-title-action"><div><h1>Задания</h1><p>Создавайте задания, проверяйте работы и следите за результатами учеников.</p></div><a className="button" href="#create-assignment"><Icon name="plus" />Создать задание</a></div>
      <div className="metric-row full">
        <div className="metric"><span className="soft-icon"><Icon name="task" /></span><span>На проверку<strong>{review.length}</strong></span></div>
        <div className="metric"><span className="soft-icon terracotta-soft"><Icon name="task" /></span><span>На исправлении<strong>{dashboard.assignments.filter((item) => item.status === "revision").length}</strong></span></div>
        <div className="metric"><span className="soft-icon danger-soft"><Icon name="clock" /></span><span>Просрочено<strong>{dashboard.assignments.filter((item) => item.status === "assigned" && new Date(item.due_at) < new Date()).length}</strong></span></div>
        <div className="metric"><span className="soft-icon"><Icon name="chart" /></span><span>Проверено<strong>{dashboard.assignments.filter((item) => item.status === "done").length}</strong></span></div>
      </div>
      <section className="panel" id="create-assignment">
        <h2>Создать задание</h2>
        <form className="form-stack" onSubmit={create}>
          <label>Ученик<select value={studentID} onChange={(event) => setStudentID(event.target.value)} required>{dashboard.relations.map((relation) => <option key={relation.id} value={relation.student_id}>{dashboard.conversations.find((item) => item.student_id === relation.student_id)?.student_name ?? relation.student_id}</option>)}</select></label>
          <label>Название<input value={title} onChange={(event) => setTitle(event.target.value)} required /></label>
          <label>Предмет<input value={subject} onChange={(event) => setSubject(event.target.value)} /></label>
          <label>Тема<input value={topic} onChange={(event) => setTopic(event.target.value)} /></label>
          <label>Условие<textarea value={body} onChange={(event) => setBody(event.target.value)} rows={5} required /></label>
          <label>Срок сдачи<input type="datetime-local" value={dueAt} onChange={(event) => setDueAt(event.target.value)} required /></label>
          <label>Максимум баллов<input type="number" min={1} max={1000} value={maxScore} onChange={(event) => setMaxScore(Number(event.target.value))} required /></label>
          {error ? <p className="field-error">{error}</p> : null}
          <button type="submit" disabled={!studentID}>Опубликовать</button>
        </form>
      </section>
      <section className="panel">
        <h2>Нужно проверить</h2>
        {review.length === 0 ? <p className="empty">Работ на проверку нет.</p> : review.map((item) => (
          <div key={item.id} className="review-card">
            <strong>{item.student_name}</strong>
            <span>{item.title}</span>
            <p>{item.answer || "Ответ пока не сохранён."}</p>
            {reviewID === item.id ? <><label>Баллы<input type="number" min={0} max={item.max_score} value={score} onChange={(event) => setScore(Number(event.target.value))} /></label><label>Комментарий<textarea value={feedback} onChange={(event) => setFeedback(event.target.value)} /></label><div className="actions"><button type="button" onClick={() => grade(item, false)}>Принять</button><button type="button" className="secondary" onClick={() => grade(item, true)}>На исправление</button></div></> : <button type="button" onClick={() => { setReviewID(item.id); setScore(0); setFeedback(""); }}>Проверить</button>}
          </div>
        ))}
      </section>
      <section className="panel full">
        <h2>Все задания</h2>
        <div className="tabs">{["Все", "Назначенные", "На проверку", "Проверено"].map((item) => <button key={item} type="button" className={filter === item ? "active" : ""} onClick={() => setFilter(item)}>{item}</button>)}</div>
        {dashboard.assignments.filter((item) => filter === "Все" || filter === "Назначенные" && item.status === "assigned" || filter === "На проверку" && item.status === "submitted" || filter === "Проверено" && item.status === "done").map((item) => <p className="task-row static" key={item.id}><strong>{item.title}</strong><span>{item.student_name} · {item.status} · {formatDateTime(item.due_at, dashboard.user.timezone)}</span></p>)}
      </section>
    </section>
  );
}
