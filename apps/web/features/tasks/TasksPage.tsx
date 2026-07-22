"use client";

import { FormEvent, useMemo, useState } from "react";
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
  const [selected, setSelected] = useState<Assignment | null>(dashboard.assignments[0] ?? null);
  const [answer, setAnswer] = useState(selected?.answer ?? "");
  const [error, setError] = useState("");

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
      <div className="page-title full"><h1>Задания</h1><p>Домашние работы, тесты и задания с занятий.</p></div>
      <div className="tasks-toolbar full">
        <div className="tabs"><button className="active">Нужно сделать <span>{dashboard.assignments.filter((item) => item.status !== "done").length}</span></button><button>В процессе</button><button>На проверке</button><button>Проверено</button><button>Все</button></div>
        <label className="search-field"><Icon name="search" size={19} /><input aria-label="Найти задание" placeholder="Найти задание" /></label>
      </div>
      <article className="hero-card task-hero full">
        <BrandMark decorative />
        <div className="hero-content">
          <p className="eyebrow">Продолжить задание</p>
          <h2>{selected?.title ?? "Заданий пока нет"}</h2>
          <p className="hero-meta">{selected ? <><Icon name="user" size={18} />{selected.topic}<span>·</span><Icon name="calendar" size={18} />Сдать до {formatDateTime(selected.due_at)}</> : "Когда преподаватель назначит работу, она появится здесь."}</p>
          {selected ? <><p className="hero-topic">Выполнено 6 из 10 заданий</p><div className="hero-progress"><span style={{ width: "60%" }} /><b>60%</b></div><div className="actions hero-actions"><a className="button hero-primary" href="#task-work">Продолжить <Icon name="arrow" size={18} /></a><button className="hero-secondary" type="button">Открыть конспект <Icon name="task" size={18} /></button></div></> : null}
        </div>
      </article>
      <section className="panel">
        <h2>Мои задания</h2>
        {dashboard.assignments.length === 0 ? <p className="empty">Нет назначенных заданий.</p> : dashboard.assignments.map((item) => (
          <button key={item.id} type="button" className={selected?.id === item.id ? "task-row active" : "task-row"} onClick={() => { setSelected(item); setAnswer(item.answer); }}>
            <span className={item.status === "overdue" ? "soft-icon terracotta-soft" : "soft-icon"}><Icon name="task" size={19} /></span>
            <span className="task-copy"><strong>{item.title}</strong><small>{item.status} · до {formatDateTime(item.due_at)}</small></span>
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
            <button type="button" onClick={submit}>Отправить на проверку</button>
          </>
        ) : <p className="empty">Выберите задание.</p>}
      </section>
    </section>
  );
}

function TeacherTasks() {
  const { dashboard, refresh } = useWorkspace();
  const [title, setTitle] = useState("Квадратные уравнения");
  const [body, setBody] = useState("Решите задачи 1-10 и приложите ход решения.");
  const [studentID, setStudentID] = useState(dashboard.relations[0]?.student_id ?? "");
  const [error, setError] = useState("");
  const review = useMemo(() => dashboard.assignments.filter((item) => item.status === "submitted" || item.status === "revision"), [dashboard.assignments]);

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    try {
      await productApi.createAssignment({
        studentID,
        title,
        subject: "Математика",
        topic: "Квадратные уравнения",
        body,
        dueAt: new Date(Date.now() + 48 * 3600_000).toISOString(),
        maxScore: 100
      });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось создать задание");
    }
  }

  async function grade(item: Assignment, revision: boolean) {
    await productApi.gradeAssignment(item.id, revision ? 60 : 92, revision ? "Нужно исправить вторую часть решения." : "Хорошая работа, принято.", revision);
    await refresh();
  }

  return (
    <section className="tasks-layout teacher">
      <div className="page-title full page-title-action"><div><h1>Задания</h1><p>Создавайте задания, проверяйте работы и следите за результатами учеников.</p></div><a className="button" href="#create-assignment"><Icon name="plus" />Создать задание</a></div>
      <div className="metric-row full">
        <div className="metric"><span className="soft-icon"><Icon name="task" /></span><span>На проверку<strong>{review.length}</strong></span></div>
        <div className="metric"><span className="soft-icon terracotta-soft"><Icon name="task" /></span><span>На исправлении<strong>{dashboard.assignments.filter((item) => item.status === "revision").length}</strong></span></div>
        <div className="metric"><span className="soft-icon danger-soft"><Icon name="clock" /></span><span>Просрочено<strong>{dashboard.assignments.filter((item) => item.status === "overdue").length}</strong></span></div>
        <div className="metric"><span className="soft-icon"><Icon name="chart" /></span><span>Проверено<strong>{dashboard.assignments.filter((item) => item.status === "done").length}</strong></span></div>
      </div>
      <section className="panel" id="create-assignment">
        <h2>Создать задание</h2>
        <form className="form-stack" onSubmit={create}>
          <label>Ученик<select value={studentID} onChange={(event) => setStudentID(event.target.value)} required>{dashboard.relations.map((relation, index) => <option key={relation.id} value={relation.student_id}>Ученик {index + 1}</option>)}</select></label>
          <label>Название<input value={title} onChange={(event) => setTitle(event.target.value)} required /></label>
          <label>Условие<textarea value={body} onChange={(event) => setBody(event.target.value)} rows={5} required /></label>
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
            <div className="actions">
              <button type="button" onClick={() => grade(item, false)}>Принять</button>
              <button type="button" className="secondary" onClick={() => grade(item, true)}>На исправление</button>
            </div>
          </div>
        ))}
      </section>
      <section className="panel full">
        <h2>Все задания</h2>
        <div className="tabs"><button className="active">Все</button><button>Черновики</button><button>Назначенные</button><button>На проверку</button><button>Архив</button></div>
        {dashboard.assignments.map((item) => <p className="task-row static" key={item.id}><strong>{item.title}</strong><span>{item.student_name} · {item.status} · {formatDateTime(item.due_at)}</span></p>)}
      </section>
    </section>
  );
}
