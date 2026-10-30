import Link from "next/link";
import { ApiError } from "../../shared/api/client";
import type { CatalogTrack, CatalogStatus, LessonInput, Exercise, Topic } from "../../shared/api/catalog";
import { BrandMark } from "../ui/Brand";
import { Icon } from "../ui/Icon";
import { ExerciseRenderer } from "../exercise-engine/ExerciseRenderer";
import type { ReactNode } from "react";
import { AnimatedExample } from "../learning-player/AnimatedExample";
import { parseExample } from "../learning-player/model";

export const trackLabels: Record<CatalogTrack, string> = { school: "Школьная математика", olympiad: "Олимпиады", oge: "ОГЭ", ege: "ЕГЭ" };
export const statusLabels: Record<CatalogStatus, string> = { draft: "Черновик", published: "Опубликован", archived: "В архиве" };
export const difficultyLabels = ["", "Начальный", "Базовый", "Средний", "Сложный", "Продвинутый"];
export function classLabel(min: number, max: number) { return min === max ? `${min} класс` : `${min}–${max} классы`; }
export function countLabel(count: number, forms = ["упражнение", "упражнения", "упражнений"]) {
  const mod = count % 100;
  return `${count} ${forms[mod >= 11 && mod <= 14 ? 2 : count % 10 === 1 ? 0 : count % 10 >= 2 && count % 10 <= 4 ? 1 : 2]}`;
}
export function topicLabel(topic: Topic, topics: Topic[]) {
  const names = [topic.title];
  const seen = new Set([topic.id]);
  let parent = topic.parent_id;
  while (parent && !seen.has(parent)) {
    seen.add(parent);
    const item = topics.find((candidate) => candidate.id === parent);
    if (!item) break;
    names.unshift(item.title); parent = item.parent_id;
  }
  return names.join(" › ");
}
export function libraryError(error: unknown) {
  if (error instanceof ApiError) {
    if (error.status === 401) return "Сессия истекла. Войдите в аккаунт заново.";
    if (error.status === 403) return "Недостаточно прав для этого действия.";
    if (error.status === 404) return "Материал недоступен или был удалён.";
    if (error.status === 400) return "Проверьте заполненные поля и выбранные значения.";
    if (error.status === 422) return "Публикация недоступна: проверьте курс, темы и изменения черновика.";
    return "Не удалось выполнить запрос. Попробуйте ещё раз.";
  }
  return error instanceof TypeError ? "Нет связи с сервером. Попробуйте ещё раз." : error instanceof Error ? error.message : "Не удалось загрузить библиотеку.";
}
export function LibraryError({ error, retry }: { error: unknown; retry?: () => void }) {
  return <div className="material-error" role="alert"><p>{libraryError(error)}</p>{error instanceof ApiError && error.status === 401 ? <Link className="button secondary" href="/login">Войти</Link> : retry ? <button type="button" className="secondary" onClick={retry}><Icon name="arrow" size={17} />Повторить</button> : null}</div>;
}
export function LibraryLoading() {
  return <div className="material-loading" aria-busy="true" role="status"><span>Загружаем материалы…</span><div className="material-grid">{[0, 1, 2].map((id) => <div key={id} className="material-skeleton" aria-hidden="true" />)}</div></div>;
}
export function LessonCover({ track, course }: { track: CatalogTrack; course: string }) {
  return <div className={`material-cover material-cover-${track}`}><span>{trackLabels[track]}</span><strong title={course}>{course}</strong><BrandMark decorative /></div>;
}
export function LessonArticle({ lesson, exercises, practice }: { lesson: LessonInput; exercises: Exercise[]; practice?: ReactNode }) {
  return <div className="material-article">
    {lesson.summary ? <p className="material-summary">{lesson.summary}</p> : null}
    {lesson.content ? <div className="material-prose">{lesson.content}</div> : null}
    {lesson.example?<ExamplePreview example={lesson.example}/>:null}
    {practice ?? (exercises.length ? <section className="material-exercises"><h2>Упражнения <span>{exercises.length}</span></h2>{exercises.map((exercise, index) => <ExerciseRenderer key={exercise.id} exercise={exercise} index={index} />)}</section> : null)}
  </div>;
}
function ExamplePreview({example}:{example:unknown}){let value;try{value=parseExample(example);}catch{return <p className="engine-error">Проверьте настройки примера.</p>;}return value?<AnimatedExample example={value}/>:null;}
