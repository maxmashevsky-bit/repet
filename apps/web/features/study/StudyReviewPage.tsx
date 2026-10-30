"use client";
import { useEffect, useState, type FormEvent } from "react";
import { studyApi, type StudyReview } from "../../shared/api/study";
import { reviewApi } from "../../shared/api/reviews";
import type { CatalogPage } from "../../shared/api/catalog";
import { exerciseApi } from "../../shared/api/exercises";
import { parseAttempt } from "../exercise-engine/schema";
import { useWorkspace } from "../app-shell/AppShell";
import { TaskSectionNav } from "../tasks/TaskSectionNav";
import { LibraryLoading } from "../material-library/LibraryUI";
import { Icon } from "../ui/Icon";
import { StudyError } from "./StudyUI";

export function StudyReviewPage({source="study"}:{source?:"study"|"review"}) {
  const { dashboard } = useWorkspace();
  return <section className="study-reviews"><TaskSectionNav /><header className="material-page-heading"><h1>Решения на проверку</h1></header>{dashboard.user.role === "tutor" ? <ReviewList source={source}/> : <p>Раздел доступен преподавателю.</p>}</section>;
}
function ReviewList({source}:{source:"study"|"review"}) {
  const [offset, setOffset] = useState(0), [retry, setRetry] = useState(0), [data, setData] = useState<{ key: string; page?: CatalogPage<StudyReview>; error?: unknown }>({ key: "" });
  const key = `${source}:${offset}:${retry}`;
  useEffect(() => { const c = new AbortController(); (source==="review"?reviewApi.checks(offset,c.signal):studyApi.reviews(offset,c.signal)).then(page => { if (!c.signal.aborted) { const items = page.items.map(item => ({ ...item, attempt: parseAttempt(item.attempt) })); setData({ key, page: { ...page, items } }); } }).catch((error: unknown) => { if (!c.signal.aborted) setData({ key, error }); }); return () => c.abort(); }, [offset, key,source]);
  return data.key !== key ? <LibraryLoading /> : data.error ? <StudyError error={data.error} retry={() => setRetry(n => n + 1)} /> : <>{!data.page?.items.length ? <div className="material-empty"><Icon name="check" size={30} /><h2>Все решения проверены</h2></div> : data.page.items.map(item => <ReviewForm key={item.attempt.id} item={item} saved={() => setRetry(n => n + 1)} />)}{data.page && data.page.total > 20 ? <nav className="material-pagination" aria-label="Страницы решений"><button type="button" disabled={!offset} onClick={() => setOffset(n => n - 20)}>Назад</button><span>{offset / 20 + 1} / {Math.ceil(data.page.total / 20)}</span><button type="button" disabled={offset + 20 >= data.page.total} onClick={() => setOffset(n => n + 20)}>Далее</button></nav> : null}</>;
}
function ReviewForm({ item, saved }: { item: StudyReview; saved: () => void }) {
  const [status, setStatus] = useState<"correct" | "incorrect">("correct"), [feedback, setFeedback] = useState(""), [busy, setBusy] = useState(false), [error, setError] = useState<unknown>();
  async function review(e: FormEvent) { e.preventDefault(); if (busy) return; setBusy(true); setError(undefined); try { const result = parseAttempt(await exerciseApi.review(item.lesson_id, item.attempt.id, status, feedback)); if (result.id !== item.attempt.id || result.status !== status) throw new Error("Некорректный результат проверки."); saved(); } catch (error) { setError(error); } finally { setBusy(false); } }
  return <article className="study-review"><h2>{item.attempt.student_name}</h2><p className="material-prose">{item.attempt.prompt}</p><h3>Решение ученика</h3><p className="engine-written-solution">{"value" in item.attempt.answer ? item.attempt.answer.value : item.attempt.answer.selected.join(", ")}</p><form onSubmit={review}><fieldset disabled={busy} className="engine-verdict"><legend>Результат проверки</legend>{([ ["correct", "Верно"], ["incorrect", "Неверно"] ] as const).map(([value, label]) => <label key={value}><input type="radio" name={`verdict-${item.attempt.id}`} checked={status === value} onChange={() => setStatus(value)} />{label}</label>)}</fieldset><label>Комментарий преподавателя<textarea value={feedback} maxLength={10000} rows={3} onChange={e => setFeedback(e.target.value)} /></label>{error ? <StudyError error={error} /> : null}<button type="submit" disabled={busy}><Icon name="check" size={17} />{busy ? "Сохраняем…" : "Сохранить проверку"}</button></form></article>;
}
