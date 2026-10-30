"use client";

import { useId, useRef, useState } from "react";
import type { Exercise } from "../../shared/api/catalog";
import type { ExerciseAnswer, ExerciseAttempt } from "../../shared/api/exercises";
import { Icon } from "../ui/Icon";
import { ApiError } from "../../shared/api/client";
import { parseAnswer, parseExercise, answerError } from "./schema";
import { exerciseRegistry } from "./registry";

export type SubmitExercise = (exercise: Exercise, answer: ExerciseAnswer, requestID: string) => Promise<ExerciseAttempt>;
type Props = { exercise: unknown; submit?: SubmitExercise; initialAttempt?: ExerciseAttempt; index?: number;onDirtyChange?:(dirty:boolean)=>void;submitLabel?:string;disabled?:boolean };
export function ExerciseRenderer(props: Props) {
  let exercise: Exercise;
  try { exercise = parseExercise(props.exercise); } catch { return <p className="engine-error" role="alert">Упражнение имеет неподдерживаемый формат.</p>; }
  if (props.initialAttempt && answerError(exercise, props.initialAttempt.answer)) return <p className="engine-error" role="alert">Сохранённый ответ не соответствует заданию.</p>;
  return <ExerciseSession key={`${exercise.id}:${props.initialAttempt?.id ?? "empty"}:${props.initialAttempt?.status ?? ""}`} {...props} exercise={exercise} />;
}

function ExerciseSession({ exercise, submit, initialAttempt, index,onDirtyChange,submitLabel,disabled=false }: Omit<Props, "exercise"> & { exercise: Exercise }) {
  const entry = exerciseRegistry[exercise.kind];
  const Input = entry.Component;
  const [answer, setAnswer] = useState<ExerciseAnswer>(initialAttempt?.answer ?? entry.initial(exercise));
  const [state, setState] = useState<"idle" | "checking" | "correct" | "incorrect" | "needs_review" | "submitted">(initialAttempt?.status ?? "idle");
  const [feedback, setFeedback] = useState(initialAttempt?.feedback ?? "");
  const [error, setError] = useState("");
  const requestID = useRef<string | undefined>(undefined);
  const checking = useRef(false);
  const errorID = useId();
  const titleID = useId();
  function change(value: ExerciseAnswer) {
    if(disabled)return;
    setAnswer(value); setState("idle"); setFeedback(""); setError(""); requestID.current = undefined;
    onDirtyChange?.(true);
  }
  async function check() {
    if (!submit || checking.current || disabled) return;
    const invalid = answerError(exercise, answer);
    if (invalid) { setError(invalid); return; }
    checking.current = true; setState("checking"); setError("");
    try {
      requestID.current ??= crypto.randomUUID();
      const result = await submit(exercise, parseAnswer(exercise, answer), requestID.current);
      setState(result.status); setFeedback(result.feedback);
      onDirtyChange?.(false);
    } catch (error) {
      setState("idle");
      setError(error instanceof ApiError && error.status === 409 ? "Версия урока изменилась. Обновите страницу перед отправкой." : error instanceof ApiError && error.status === 401 ? "Сессия истекла. Войдите заново." : "Не удалось отправить ответ. Повторите попытку.");
    } finally { checking.current = false; }
  }
  const messages = { idle: "", checking: "Проверяется…", correct: "Верно", incorrect: "Неверно", needs_review: "Требует проверки преподавателем",submitted:"Ответ сохранён" };
  return <section className={`engine-exercise engine-${state}`} aria-labelledby={titleID} data-exercise-kind={exercise.kind} onKeyDown={(event) => { if (event.key === "Enter" && event.target instanceof HTMLInputElement && event.target.type === "text") { event.preventDefault(); void check(); } }}>
    <div className="material-exercise-caption"><h3 id={titleID}>{index !== undefined ? `Упражнение ${index + 1}` : "Упражнение"}</h3><span>{entry.label}</span></div>
    <p className="engine-prompt">{exercise.prompt}</p>
    {exercise.media?.map(media=><figure className="exercise-media" key={media.asset_id}>{/* Private, authenticated images cannot use the public Next image optimizer. */}<img src={`/api/v1/pdf-assets/${encodeURIComponent(media.asset_id)}`} alt={media.alt} loading="lazy" /><figcaption>{media.alt}</figcaption></figure>)}
    <Input exercise={exercise} value={answer} onChange={change} disabled={disabled || !submit || state === "checking"} errorID={errorID} invalid={Boolean(error)} />
    {error ? <p className="engine-error" id={errorID} role="alert">{error}</p> : null}
    {submit || state !== "idle" ? <div className="engine-actions">{submit ? <button type="button" disabled={disabled || state === "checking"} onClick={() => void check()}>{state === "checking" ? "Проверяется…" : submitLabel??(exercise.kind === "long_text" ? "Отправить преподавателю" : "Проверить ответ")}<Icon name={exercise.kind === "long_text" ? "send" : "check"} size={17} /></button> : null}<p className="engine-status" role="status" aria-live="polite">{messages[state]}</p></div> : null}
    {feedback ? <div className="engine-feedback"><strong>Комментарий преподавателя</strong><p>{feedback}</p></div> : null}
  </section>;
}
