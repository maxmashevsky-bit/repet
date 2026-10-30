"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { studyApi, type StudySession } from "../../shared/api/study";
import { learningApi } from "../../shared/api/learning";
import { exerciseApi } from "../../shared/api/exercises";
import { ExerciseRenderer, type SubmitExercise } from "../exercise-engine/ExerciseRenderer";
import { parseAttempt } from "../exercise-engine/schema";
import { LearningPlayer } from "../learning-player/LearningPlayer";
import { LibraryLoading } from "../material-library/LibraryUI";
import { TaskSectionNav } from "../tasks/TaskSectionNav";
import { Icon } from "../ui/Icon";
import { clockLabel, parseStudySession, remainingSeconds } from "./model";
import { StudyError } from "./StudyUI";
import { StudyResults } from "./StudyResults";

export function StudySessionPage({ sessionID }: { sessionID: string }) {
  const [retry, setRetry] = useState(0);
  const key = `${sessionID}:${retry}`;
  const [data, setData] = useState<{ key: string; session?: StudySession; error?: unknown }>({ key: "" });
  useEffect(() => {
    const c = new AbortController();
    studyApi.session(sessionID, c.signal).then(parseStudySession).then(session => { if (!c.signal.aborted) setData({ key, session }); }).catch((error: unknown) => { if (!c.signal.aborted) setData({ key, error }); });
    return () => c.abort();
  }, [sessionID, key]);
  const session = data.session;
  return <section className="study-session"><TaskSectionNav /><Link className="inline-link material-back" href="/tasks/study"><Icon name="arrow" className="back" size={17} />Мои прохождения</Link>
    {data.key !== key ? <LibraryLoading /> : data.error ? <StudyError error={data.error} retry={() => setRetry(n => n + 1)} /> : session ? <>
      <header className="material-page-heading"><div><p className="material-kicker">{session.specification.year} · версия {session.specification.version} · вариант {session.seed}</p><h1>{session.specification.title}</h1></div></header>
      {session.state.attempt.status === "completed" ? <StudyResults session={session} refresh={() => setRetry(n => n + 1)} /> : session.state.attempt.mode === "practice" ? <LearningPlayer lessonID={session.state.attempt.lesson_id} version={session.state.version} initial={session.state} allowRestart={false} onCompleted={() => setRetry(n => n + 1)} /> : <ExamPlayer key={session.id} initial={session} completed={value => setData({ key, session: value })} />}
    </> : null}
  </section>;
}

function ExamPlayer({ initial, completed }: { initial: StudySession; completed: (session: StudySession) => void }) {
  const [session, setSession] = useState(initial), [busy, setBusy] = useState(false), [dirty, setDirty] = useState(false), [error, setError] = useState<unknown>();
  const [anchor, setAnchor] = useState(() => ({ server: initial.server_time, local: performance.now() })), [tick, setTick] = useState(0);
  const gate = useRef(false), nonces = useRef(new Map<string, string>()), revision = useRef(0);
  const run = session.state.attempt, exercises = session.state.version.exercises;
  const index = Math.max(0, exercises.findIndex(e => run.player_step === `exercise:${e.id}`));
  const exercise = exercises[index], answer = session.state.answers.find(a => a.exercise_id === exercise.id);
  const seconds = remainingSeconds(session.deadline_at, anchor.server, tick ? tick - anchor.local : 0);
  const refresh = useCallback(async () => {
    const token = ++revision.current;
    const value = parseStudySession(await studyApi.session(initial.id));
    if (value.id !== initial.id) throw new Error("Некорректный идентификатор прохождения.");
    if (token !== revision.current) return value;
    setSession(value); setAnchor({ server: value.server_time, local: performance.now() }); setTick(0);
    return value;
  }, [initial.id]);
  useEffect(() => { const timer = setInterval(() => setTick(performance.now()), 1000); return () => clearInterval(timer); }, []);
  useEffect(() => {
    const sync = () => { if (!gate.current) void refresh().catch(setError); };
    const timer = setInterval(sync, 15000);
    const visible = () => { if (document.visibilityState === "visible") sync(); };
    document.addEventListener("visibilitychange", visible);
    return () => { clearInterval(timer); document.removeEventListener("visibilitychange", visible); };
  }, [refresh]);
  useEffect(() => { if (!seconds && run.status !== "completed" && !gate.current) void refresh().catch(setError); }, [seconds, run.status, refresh]);
  useEffect(() => { if (run.status === "completed") completed(session); }, [run.status, session, completed]);
  useEffect(() => {
    if (!dirty) return;
    const warn = (e: BeforeUnloadEvent) => { e.preventDefault(); e.returnValue = ""; };
    window.addEventListener("beforeunload", warn); return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  async function action(key: string, send: (nonce: string) => Promise<unknown>) {
    if (gate.current) return;
    gate.current = true; setBusy(true); setError(undefined);
    const nonce = nonces.current.get(key) ?? crypto.randomUUID(); nonces.current.set(key, nonce);
    try { await send(nonce); await refresh(); nonces.current.delete(key); setDirty(false); }
    catch (e) { setError(e); } finally { gate.current = false; setBusy(false); }
  }
  function navigate(target: number) {
    if (target < 0 || target >= exercises.length || dirty && !window.confirm("Перейти без сохранения текущего ответа?")) return;
    void action(`navigate:${target}`, nonce => learningApi.navigate(run.lesson_id, run.id, `exercise:${exercises[target].id}`, nonce));
  }
  const submit: SubmitExercise = async (e, value, nonce) => {
    if (gate.current || !seconds) throw new Error("Сохранение недоступно");
    gate.current = true; setBusy(true); setError(undefined);
    try {
      const result = parseAttempt(await exerciseApi.submit(run.lesson_id, run.version_id, e.id, value, nonce, run.id));
      if (result.exercise_id !== e.id || result.learning_attempt_id !== run.id || result.version_id !== run.version_id || result.status !== "submitted") throw new Error("Некорректный ответ сервера.");
      await refresh(); setDirty(false); return result;
    } finally { gate.current = false; setBusy(false); }
  };
  return <div className="study-exam">
    <header className="study-exam-status"><div><strong>Пробный экзамен</strong><span role="status">{busy ? "Сохраняем…" : dirty ? "Есть несохранённый ответ" : "Ответы сохранены"}</span></div><div className={seconds < 60 ? "study-clock urgent" : "study-clock"} role="timer" aria-label="Осталось времени"><Icon name="clock" size={19} />{clockLabel(seconds)}</div></header>
    <p className="muted">Сохранено {run.answered_exercises} из {run.total_exercises}. Проверка и разборы после сдачи.</p>
    {session.specification.instructions ? <details className="study-instructions"><summary>Инструкция</summary><p className="material-prose">{session.specification.instructions}</p></details> : null}
    {error ? <StudyError error={error} retry={() => void refresh().catch(setError)} /> : null}
    <nav className="study-numbers" aria-label="Номера заданий">{exercises.map((e, i) => <button type="button" key={e.id} aria-label={`Задание ${i + 1}`} aria-current={i === index ? "step" : undefined} className={i === index ? "active" : session.state.answers.some(a => a.exercise_id === e.id) ? "answered" : ""} disabled={busy || !seconds} onClick={() => navigate(i)}>{i + 1}{session.state.answers.some(a => a.exercise_id === e.id) ? <Icon name="check" size={12} /> : null}</button>)}</nav>
    <ExerciseRenderer key={exercise.id} exercise={exercise} index={index} initialAttempt={answer} submit={submit} submitLabel="Сохранить ответ" disabled={busy || !seconds} onDirtyChange={setDirty} />
    <footer className="study-exam-footer"><div className="actions"><button type="button" className="secondary" disabled={busy || !index || !seconds} onClick={() => navigate(index - 1)}><Icon name="arrow" className="back" size={17} />Назад</button><button type="button" className="secondary" disabled={busy || index === exercises.length - 1 || !seconds} onClick={() => navigate(index + 1)}>Далее<Icon name="arrow" size={17} /></button></div><button type="button" disabled={busy || dirty || !seconds} onClick={() => { if (window.confirm(`Завершить работу? Без ответа: ${run.total_exercises - run.answered_exercises}.`)) void action("complete", nonce => learningApi.complete(run.lesson_id, run.id, nonce)); }}><Icon name="send" size={17} />Завершить и сдать</button></footer>
  </div>;
}
