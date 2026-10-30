"use client";

import { useEffect, useState } from "react";
import type { Exercise } from "../../shared/api/catalog";
import { allCatalogPages } from "../../shared/api/catalog";
import { exerciseApi, type ExerciseAttempt } from "../../shared/api/exercises";
import { LibraryError, LibraryLoading } from "../material-library/LibraryUI";
import { Icon } from "../ui/Icon";
import { ExerciseRenderer, type SubmitExercise } from "./ExerciseRenderer";
import { answerError, parseAttempt, parseAttemptsPage } from "./schema";

export function LessonExercises({ lessonID, versionID, exercises, mode }: { lessonID: string; versionID: string; exercises: Exercise[]; mode: "student" | "author" | "reader" }) {
  const [retry, setRetry] = useState(0);
  const key = `${lessonID}:${versionID}:${retry}:${mode}`;
  const [state, setState] = useState<{ key: string; items?: ExerciseAttempt[]; error?: unknown }>({ key: "" });
  useEffect(() => {
    if (mode === "reader") return;
    const controller = new AbortController();
    allCatalogPages(async (offset) => parseAttemptsPage(await exerciseApi.attempts(lessonID, offset, controller.signal))).then((items) => { if (!controller.signal.aborted) setState({ key, items }); }).catch((error: unknown) => { if (!controller.signal.aborted) setState({ key, error }); });
    return () => controller.abort();
  }, [lessonID, key, mode]);
  const submit: SubmitExercise = async (exercise, answer, requestID) => {
    const attempt = parseAttempt(await exerciseApi.submit(lessonID, versionID, exercise.id, answer, requestID));
    if (attempt.exercise_id !== exercise.id || attempt.version_id !== versionID || answerError(exercise, attempt.answer)) throw new Error("Unexpected exercise response");
    return attempt;
  };
  if (mode !== "reader" && state.key !== key) return <LibraryLoading />;
  if (mode !== "reader" && state.error) return <LibraryError error={state.error} retry={() => setRetry((value) => value + 1)} />;
  return <section className="material-exercises"><div className="material-results-heading"><h2>Упражнения <span>{exercises.length}</span></h2>{mode === "student" ? <button className="secondary" type="button" onClick={() => setRetry((value) => value + 1)}><Icon name="arrow" size={16} />Обновить результаты</button> : null}</div>
    {exercises.map((exercise, index) => <ExerciseRenderer key={exercise.id} exercise={exercise} index={index} submit={mode === "student" ? submit : undefined} initialAttempt={mode === "student" ? state.items?.find((attempt) => attempt.version_id === versionID && attempt.exercise_id === exercise.id) : undefined} />)}
    {mode === "author" ? <section className="engine-reviews"><h2>Решения учеников</h2><button type="button" className="secondary" onClick={() => setRetry((value) => value + 1)}><Icon name="arrow" size={16} />Обновить решения</button>{state.items?.filter((item) => item.kind === "long_text").length ? state.items.filter((item) => item.kind === "long_text").map((attempt) => <ReviewSolution key={`${attempt.id}:${attempt.status}`} lessonID={lessonID} attempt={attempt} reviewed={(updated) => setState((current) => ({ ...current, items: current.items?.map((item) => item.id === updated.id ? updated : item) }))} />) : <p className="muted">Решений на проверку пока нет.</p>}</section> : null}
  </section>;
}

function ReviewSolution({ lessonID, attempt, reviewed }: { lessonID: string; attempt: ExerciseAttempt; reviewed: (attempt: ExerciseAttempt) => void }) {
  const [verdict, setVerdict] = useState<"correct" | "incorrect">("correct");
  const [feedback, setFeedback] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  async function save() {
    if (busy) return;
    setBusy(true); setError(null);
    try { reviewed(parseAttempt(await exerciseApi.review(lessonID, attempt.id, verdict, feedback))); } catch (error) { setError(error); } finally { setBusy(false); }
  }
  if (attempt.status === "submitted") return <article className="engine-review"><h3>{attempt.student_name}</h3><p>{attempt.prompt}</p><p className="engine-written-solution">{attempt.answer.value}</p><p className="muted">Ожидает завершения контрольной</p></article>;
  return <article className="engine-review"><div className="material-exercise-caption"><h3>{attempt.student_name}</h3><span>Версия {attempt.version}</span></div><p>{attempt.prompt}</p><p className="engine-written-solution">{attempt.answer.value}</p>{attempt.status === "needs_review" ? <><fieldset disabled={busy} className="engine-verdict"><legend>Результат проверки</legend><label><input type="radio" name={`review-${attempt.id}`} checked={verdict === "correct"} onChange={() => setVerdict("correct")} />Верно</label><label><input type="radio" name={`review-${attempt.id}`} checked={verdict === "incorrect"} onChange={() => setVerdict("incorrect")} />Неверно</label></fieldset><label>Комментарий<textarea aria-label={`Комментарий для ${attempt.student_name}`} rows={3} maxLength={10000} value={feedback} onChange={(event) => setFeedback(event.target.value)} disabled={busy} /></label>{error ? <LibraryError error={error} /> : null}<button type="button" disabled={busy} onClick={() => void save()}><Icon name="check" size={17} />{busy ? "Сохраняем…" : "Сохранить проверку"}</button></> : <div className="engine-feedback"><strong>{attempt.status === "correct" ? "Верно" : "Неверно"}</strong>{attempt.feedback ? <p>{attempt.feedback}</p> : null}</div>}</article>;
}
