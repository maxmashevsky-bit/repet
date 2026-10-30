"use client";
import { Select } from "../ui/Select";


import { FormEvent, ReactNode, useEffect, useId, useRef, useState } from "react";
import { catalogApi, type CatalogTrack, type Course, type CourseInput, type Topic } from "../../shared/api/catalog";
import { Icon } from "../ui/Icon";
import { difficultyLabels, LibraryError, topicLabel, trackLabels } from "./LibraryUI";

export function EditorDialog({ title, children, close, busy = false }: { title: string; children: ReactNode; close: () => void; busy?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null);
  const id = useId();
  useEffect(() => {
    const node = ref.current;
    if (!node?.open) node?.showModal();
    return () => node?.close();
  }, []);
  return <dialog ref={ref} className="material-dialog" aria-labelledby={id} onCancel={(event) => { if (busy) event.preventDefault(); else close(); }}><header><h2 id={id}>{title}</h2><button className="icon-button" type="button" aria-label="Закрыть окно" title="Закрыть" disabled={busy} onClick={close}><Icon name="close" /></button></header>{children}</dialog>;
}

export function ClassificationFields({ value, change }: { value: { track: CatalogTrack; grade_min: number; grade_max: number; difficulty: number }; change: (patch: Partial<typeof value>) => void }) {
  return <div className="material-classification">
    <label>Направление<Select aria-label="Направление" value={value.track} onChange={(e) => change({ track: e.target.value as CatalogTrack })}>{Object.entries(trackLabels).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</Select></label>
    <label>Класс от<Select aria-label="Класс от" value={value.grade_min} onChange={(e) => change({ grade_min: Number(e.target.value) })}>{Array.from({ length: 11 }, (_, i) => <option key={i} value={i + 1}>{i + 1}</option>)}</Select></label>
    <label>Класс до<Select aria-label="Класс до" value={value.grade_max} onChange={(e) => change({ grade_max: Number(e.target.value) })}>{Array.from({ length: 11 }, (_, i) => <option key={i} value={i + 1}>{i + 1}</option>)}</Select></label>
    <label>Сложность<Select aria-label="Сложность" value={value.difficulty} onChange={(e) => change({ difficulty: Number(e.target.value) })}>{difficultyLabels.slice(1).map((label, i) => <option key={label} value={i + 1}>{label}</option>)}</Select></label>
  </div>;
}

export function CreateCourseDialog({ close, created }: { close: () => void; created: (course: Course) => void }) {
  const [input, setInput] = useState<CourseInput>({ title: "", description: "", track: "school", grade_min: 1, grade_max: 11, difficulty: 1, visibility: "linked_students" });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (input.grade_min > input.grade_max) { setError(new Error("Начальный класс не может быть больше конечного.")); return; }
    setBusy(true); setError(null);
    try { created(await catalogApi.createCourse(input)); } catch (err) { setError(err); } finally { setBusy(false); }
  }
  return <EditorDialog title="Новый курс" close={close} busy={busy}><form onSubmit={submit} className="form-stack"><fieldset disabled={busy} className="material-fieldset">
    <label>Название курса<input aria-label="Название курса" value={input.title} onChange={(e) => setInput({ ...input, title: e.target.value })} maxLength={200} required /></label>
    <label>Описание курса<textarea aria-label="Описание курса" value={input.description} onChange={(e) => setInput({ ...input, description: e.target.value })} maxLength={10000} rows={3} /></label>
    <ClassificationFields value={input} change={(patch) => setInput({ ...input, ...patch })} />
    <label>Кому доступен курс<Select aria-label="Кому доступен курс" value={input.visibility} onChange={(e) => setInput({ ...input, visibility: e.target.value as CourseInput["visibility"] })}><option value="linked_students">Моим ученикам</option><option value="organization">Всей организации</option></Select></label>
    </fieldset>{error ? <LibraryError error={error} /> : null}<div className="material-dialog-actions"><button type="button" className="secondary" disabled={busy} onClick={close}>Отмена</button><button type="submit" disabled={busy}>{busy ? "Создаём…" : "Создать курс"}</button></div></form></EditorDialog>;
}

export function CreateTopicDialog({ course, topics, close, created }: { course: Course; topics: Topic[]; close: () => void; created: (topic: Topic) => void }) {
  const [title, setTitle] = useState("");
  const [parent, setParent] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  async function submit(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError(null);
    try { created(await catalogApi.createTopic(course.id, { title, parent_id: parent || null, description: "", position: Math.min(1000000, Math.max(-1, ...topics.map((topic) => topic.position)) + 1) })); } catch (err) { setError(err); } finally { setBusy(false); }
  }
  return <EditorDialog title="Новая тема" close={close} busy={busy}><form onSubmit={submit} className="form-stack"><p className="muted">{course.title}</p><fieldset disabled={busy} className="material-fieldset"><label>Название темы<input aria-label="Название темы" value={title} onChange={(e) => setTitle(e.target.value)} required maxLength={200} /></label><label>Родительская тема<Select aria-label="Родительская тема" value={parent} onChange={(e) => setParent(e.target.value)}><option value="">Без родительской темы</option>{topics.map((topic) => <option key={topic.id} value={topic.id}>{topicLabel(topic, topics)}</option>)}</Select></label></fieldset>{error ? <LibraryError error={error} /> : null}<div className="material-dialog-actions"><button type="button" className="secondary" disabled={busy} onClick={close}>Отмена</button><button type="submit" disabled={busy}>{busy ? "Создаём…" : "Создать тему"}</button></div></form></EditorDialog>;
}
