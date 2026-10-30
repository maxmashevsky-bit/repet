"use client";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { studyApi, type StudySession } from "../../shared/api/study";
import { learningApi } from "../../shared/api/learning";
import { ExerciseRenderer } from "../exercise-engine/ExerciseRenderer";
import { Icon } from "../ui/Icon";
import { parseStudySession, repeatExerciseCount } from "./model";
import { StudyError } from "./StudyUI";

export function StudyResults({ session, refresh }: { session: StudySession; refresh: () => void }) {
  const router = useRouter(), [busy, setBusy] = useState(false), [error, setError] = useState<unknown>();
  const [solutions, setSolutions] = useState<Record<string, string>>({});
  const nonce = useRef<string | undefined>(undefined), gate = useRef(false);
  const run = session.state.attempt, repeatCount = repeatExerciseCount(session);
  async function repeat() {
    if (gate.current) return;
    gate.current = true; setBusy(true); setError(undefined); nonce.current ??= crypto.randomUUID();
    try { const next = parseStudySession(await studyApi.start({ request_id: nonce.current, specification_id: session.specification.id, mode: "practice", seed: "", topic: "", count: repeatCount, repeat_session_id: session.id })); router.push(`/tasks/study/${next.id}`); }
    catch (e) { setError(e); } finally { gate.current = false; setBusy(false); }
  }
  async function solution(exerciseID: string) {
    setError(undefined);
    try { const value = await learningApi.solution(run.lesson_id, run.id, exerciseID); if (!value || typeof value !== "object" || !("text" in value) || typeof value.text !== "string" || !("exercise_id" in value) || value.exercise_id !== exerciseID) throw new Error("Некорректный разбор."); setSolutions(old => ({ ...old, [exerciseID]: value.text as string || "Разбор не добавлен." })); }
    catch (e) { setError(e); }
  }
  return <section className="study-results"><header className="material-results-heading"><h2>Работа завершена</h2><button type="button" className="secondary" onClick={refresh}>Обновить результат</button></header>
    <div className="study-result-score"><strong>{session.points} / {session.max_points}</strong><span>{run.pending_review ? "Предварительные баллы варианта" : "Баллы варианта"}</span></div>
    <dl className="learning-counts"><div><dt>Верно</dt><dd>{run.correct_exercises}</dd></div><div><dt>Неверно</dt><dd>{session.state.answers.filter(a => a.status === "incorrect").length}</dd></div><div><dt>На проверке</dt><dd>{run.pending_review}</dd></div><div><dt>Без ответа</dt><dd>{run.total_exercises - run.answered_exercises}</dd></div></dl>
    {error ? <StudyError error={error} /> : null}
    {repeatCount ? <button type="button" disabled={busy} onClick={() => void repeat()}><Icon name="arrow" size={17} />{busy ? "Создаём тренировку…" : `Повторить ошибки (${repeatCount})`}</button> : null}
    <div className="study-result-answers">{session.state.version.exercises.map((exercise, index) => { const answer = session.state.answers.find(a => a.exercise_id === exercise.id); return <details key={exercise.id}><summary>Задание {index + 1} · {answer ? { correct: "Верно", incorrect: "Неверно", needs_review: "На проверке", submitted: "Сохранено" }[answer.status] : "Без ответа"}</summary><ExerciseRenderer exercise={exercise} index={index} initialAttempt={answer} />{answer && answer.status !== "needs_review" ? <button type="button" className="secondary" onClick={() => void solution(exercise.id)}><Icon name="eye" size={16} />Разбор решения</button> : null}{solutions[exercise.id] ? <p className="material-prose">{solutions[exercise.id]}</p> : null}</details>; })}</div>
  </section>;
}
