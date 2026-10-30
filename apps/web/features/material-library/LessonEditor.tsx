"use client";
import { Select } from "../ui/Select";


import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState } from "react";
import { ApiError } from "../../shared/api/client";
import { catalogApi, loadLibraryStructure, allCatalogPages, type AnswerKey, type Course, type Exercise, type ExerciseInput, type LearningLesson, type LessonInput, type Topic } from "../../shared/api/catalog";
import { useWorkspace } from "../app-shell/AppShell";
import { TaskSectionNav } from "../tasks/TaskSectionNav";
import { Icon } from "../ui/Icon";
import { exerciseRegistry } from "../exercise-engine/registry";
import { validNumber, parseExercise } from "../exercise-engine/schema";
import { defaultVisualConfig, isVisual, parseVisualConfig, visualAnswerError } from "../exercise-engine/visual-schema";
import { ExerciseKeyFields } from "./ExerciseKeyFields";
import { ClassificationFields, CreateCourseDialog, CreateTopicDialog, EditorDialog } from "./EditorDialogs";
import { classLabel, difficultyLabels, LessonArticle, LibraryError, LibraryLoading, topicLabel } from "./LibraryUI";
import { ExampleEditor } from "../learning-player/ExampleEditor";
import { parseExample } from "../learning-player/model";
import { PDFSourcePanel } from "../pdf-import/PDFSourcePanel";

type EditorData = { courses: Course[]; topics: Topic[]; lesson?: LearningLesson; keys: AnswerKey[] };
type ExerciseDraft = { clientID: string; id?: string; saved?: string; kind: Exercise["kind"]; prompt: string; optionsText: string; difficulty: number; position: number; answer: string; explanation: string; configText: string; hintsText?: string; media?:Exercise["media"] };
type Dialog = { kind: "course" | "topic" } | { kind: "publish"; course: Course; chain: Topic[] };
const blank: LessonInput = { title: "", summary: "", content: "", topic_id: "", track: "school", grade_min: 1, grade_max: 11, difficulty: 1 };

function lessonInput(value: LessonInput): LessonInput {
  return { topic_id: value.topic_id, title: value.title.trim(), summary: value.summary, content: value.content, track: value.track, grade_min: value.grade_min, grade_max: value.grade_max, difficulty: value.difficulty,example:value.example };
}
function options(value: ExerciseDraft) { return value.kind === "single_choice" || value.kind === "multiple_choice" ? value.optionsText.split("\n").map((item) => item.trim()).filter(Boolean) : []; }
function exerciseInput(value: ExerciseDraft): ExerciseInput {
  let config: Record<string, unknown>;
  try { config = parseVisualConfig(value.kind, JSON.parse(value.configText)); } catch { throw new Error("Проверьте настройки JSON упражнения."); }
  const hints=(value.hintsText??"").split("\n").map(h=>h.trim()).filter(Boolean);
  if(hints.length>5||hints.some(h=>new TextEncoder().encode(h).length>4000))throw new Error("Допустимо до пяти подсказок, каждая до 4000 байт.");
  return { kind: value.kind, prompt: value.prompt.trim(), options: options(value), difficulty: value.difficulty, position: value.position, config, media:value.media??[], answer_key: { answer: value.answer.trim(), explanation: value.explanation, hints } };
}
function exerciseSignature(value: ExerciseDraft) { return JSON.stringify([value.kind, value.prompt, value.optionsText, value.difficulty, value.position, value.answer, value.explanation, value.configText, value.hintsText??"",value.media??[]]); }
function previewConfig(value: string) { try { return JSON.parse(value) as Record<string, unknown>; } catch { return undefined; } }
function topicChain(id: string, topics: Topic[]) {
  const chain: Topic[] = [];
  const seen = new Set<string>();
  let current: string | null = id;
  while (current) {
    const topic = topics.find((item) => item.id === current);
    if (!topic || seen.has(current)) throw new Error("Не удалось получить дерево тем. Обновите страницу.");
    seen.add(current); chain.unshift(topic); current = topic.parent_id;
  }
  return chain;
}

export function LessonEditorPage({ lessonID }: { lessonID?: string }) {
  const { dashboard } = useWorkspace();
  if (dashboard.user.role !== "tutor") return <><TaskSectionNav /><h1>Редактор недоступен</h1><p>Создавать материалы может преподаватель.</p><Link className="button secondary" href="/tasks/library">В библиотеку</Link></>;
  return <EditorLoader lessonID={lessonID} userID={dashboard.user.id} />;
}

function EditorLoader({ lessonID, userID }: { lessonID?: string; userID: string }) {
  const [retry, setRetry] = useState(0);
  const key = `${lessonID ?? "new"}:${userID}:${retry}`;
  const [state, setState] = useState<{ key: string; data?: EditorData; error?: unknown }>({ key: "" });
  useEffect(() => {
    const controller = new AbortController();
    async function load(): Promise<EditorData> {
      const [structure, lesson] = await Promise.all([loadLibraryStructure(controller.signal, userID), lessonID ? catalogApi.lesson(lessonID, controller.signal) : Promise.resolve(undefined)]);
      if (lesson && lesson.author_id !== userID) throw new ApiError("permission denied", 403);
      const keys = lesson ? (await catalogApi.keys(lesson.id, controller.signal)).items : [];
      if (lesson?.version.exercises.some((exercise) => !keys.some((item) => item.exercise_id === exercise.id))) throw new Error("Не удалось загрузить ключи упражнений. Повторите загрузку.");
      return { ...structure, lesson, keys };
    }
    load().then((data) => { if (!controller.signal.aborted) setState({ key, data }); }).catch((error: unknown) => { if (!controller.signal.aborted) setState({ key, error }); });
    return () => controller.abort();
  }, [lessonID, userID, key]);
  return <><TaskSectionNav />{state.key !== key ? <LibraryLoading /> : state.error ? <><Link className="inline-link material-back" href="/tasks/library">В библиотеку</Link><LibraryError error={state.error} retry={() => setRetry((n) => n + 1)} /></> : state.data ? <EditorSession key={key} initial={state.data} /> : null}</>;
}

function EditorSession({ initial }: { initial: EditorData }) {
  const router = useRouter();
  const form = useRef<HTMLFormElement>(null);
  const idRef = useRef(initial.lesson?.id);
  const [id, setID] = useState(initial.lesson?.id);
  const [courses, setCourses] = useState(initial.courses);
  const [topics, setTopics] = useState(initial.topics);
  const [courseID, setCourseID] = useState(initial.lesson?.course_id ?? "");
  const [fields, setFields] = useState<LessonInput>(lessonInput(initial.lesson?.version ?? blank));
  const [saved, setSaved] = useState(JSON.stringify(lessonInput(initial.lesson?.version ?? blank)));
  const [pendingPublication, setPendingPublication] = useState(initial.lesson?.has_unpublished_changes ?? false);
  const [exercises, setExercises] = useState<ExerciseDraft[]>(() => initial.lesson?.version.exercises.map((exercise) => {
    const key = initial.keys.find((item) => item.exercise_id === exercise.id)!;
    const draft: ExerciseDraft = { clientID: exercise.id, id: exercise.id, kind: exercise.kind, prompt: exercise.prompt, optionsText: exercise.options.join("\n"), difficulty: exercise.difficulty, position: exercise.position, answer: key.answer, explanation: key.explanation, configText: JSON.stringify(exercise.config ?? {}, null, 2) };
    draft.hintsText=(key.hints??[]).join("\n");
    draft.media=exercise.media??[];
    return { ...draft, saved: exerciseSignature(draft) };
  }) ?? []);
  const [mode, setMode] = useState<"edit" | "preview">("edit");
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [message, setMessage] = useState("");
  const course = courses.find((item) => item.id === courseID);
  const courseTopics = topics.filter((item) => item.course_id === courseID);
  const dirty = JSON.stringify(lessonInput(fields)) !== saved || exercises.some((item) => !item.id || item.saved !== exerciseSignature(item));

  useEffect(() => {
    if (!dirty) return;
    const beforeUnload = (event: BeforeUnloadEvent) => { event.preventDefault(); event.returnValue = ""; };
    const follow = (event: MouseEvent) => {
      if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || !(event.target instanceof Element)) return;
      const anchor = event.target.closest("a");
      if (!anchor || anchor.target === "_blank") return;
      const target = new URL(anchor.href, window.location.href);
      if (target.pathname === window.location.pathname && target.search === window.location.search) return;
      if (!window.confirm("Есть несохранённые изменения. Покинуть редактор?")) { event.preventDefault(); event.stopPropagation(); }
    };
    window.addEventListener("beforeunload", beforeUnload); document.addEventListener("click", follow, true);
    return () => { window.removeEventListener("beforeunload", beforeUnload); document.removeEventListener("click", follow, true); };
  }, [dirty]);

  function change(patch: Partial<LessonInput>) { setFields((value) => ({ ...value, ...patch })); setMessage(""); }
  function selectCourse(value: string) {
    const selected = courses.find((item) => item.id === value);
    setCourseID(value);
    change({ topic_id: "", ...(selected ? { track: selected.track, grade_min: selected.grade_min, grade_max: selected.grade_max, difficulty: selected.difficulty } : {}) });
  }
  function updateExercise(clientID: string, patch: Partial<ExerciseDraft>) { setExercises((items) => items.map((item) => item.clientID === clientID ? { ...item, ...patch } : item)); setMessage(""); }
  function validate() {
    parseExample(fields.example);
    if (!courseID || !fields.topic_id || !fields.title.trim()) throw new Error("Выберите курс и тему, заполните название урока.");
    if (fields.grade_min > fields.grade_max) throw new Error("Начальный класс не может быть больше конечного.");
    if (new TextEncoder().encode(fields.content).length > 250000) throw new Error("Текст урока слишком длинный: максимум 250 КБ.");
    for (let i = 0; i < exercises.length; i++) {
      const input = exerciseInput(exercises[i]);
      if (isVisual(input.kind)) {
        const { answer_key, ...publicFields } = input;
        const spec = parseExercise({ ...publicFields, id: exercises[i].id ?? exercises[i].clientID });
        let answer: unknown;
        try { answer = input.kind === "number_line" ? { value: answer_key.answer } : { selected: JSON.parse(answer_key.answer) }; } catch { throw new Error(`Укажите правильный ответ упражнения ${i + 1}.`); }
        const error = visualAnswerError(spec, answer); if (error) throw new Error(`Упражнение ${i + 1}: ${error}`);
      }
      if (!input.prompt || input.kind !== "long_text" && !input.answer_key.answer) throw new Error(`Заполните условие и закрытый ответ упражнения ${i + 1}.`);
      if (input.kind === "numeric" && !validNumber(input.answer_key.answer)) throw new Error(`В упражнении ${i + 1} укажите корректное число или дробь.`);
      if (input.kind === "multiple_choice") {
        let selected: unknown;
        try { selected = JSON.parse(input.answer_key.answer); } catch { throw new Error(`Выберите правильные ответы упражнения ${i + 1}.`); }
        if (input.options.length < 2 || input.options.length > 12 || new Set(input.options).size !== input.options.length || !Array.isArray(selected) || !selected.length || new Set(selected).size !== selected.length || selected.some((item) => !input.options.includes(item)) || input.answer_key.answer.length > 4096) throw new Error(`Проверьте варианты и правильные ответы упражнения ${i + 1}.`);
      }
      if (input.kind === "single_choice" && (input.options.length < 2 || input.options.length > 12 || new Set(input.options).size !== input.options.length || !input.options.includes(input.answer_key.answer))) throw new Error(`В упражнении ${i + 1} нужны 2–12 разных вариантов и ответ из этого списка.`);
    }
  }

  async function saveDraft() {
    validate();
    let currentID = idRef.current;
    const input = lessonInput(fields);
    if (!currentID) {
      const created = await catalogApi.createLesson(courseID, input);
      currentID = created.id; idRef.current = currentID; setID(currentID); setSaved(JSON.stringify(input)); setPendingPublication(true);
    } else if (JSON.stringify(input) !== saved) {
      await catalogApi.updateLesson(currentID, input); setSaved(JSON.stringify(input)); setPendingPublication(true);
    }
    const working = exercises.map((item) => ({ ...item }));
    try {
      for (const item of working) {
        const payload = exerciseInput(item);
        const signature = exerciseSignature(item);
        if (item.id && item.saved === signature) continue;
        const result = await catalogApi.saveExercise(currentID, payload, item.id);
        item.id = result.id; item.saved = signature; setPendingPublication(true);
      }
    } finally { setExercises(working); }
    return currentID;
  }

  async function save(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError(null); setMessage("");
    try {
      const result = await saveDraft(); setMessage("Черновик сохранён.");
      if (!initial.lesson) router.replace(`/tasks/library/${result}/edit`, { scroll: false });
    } catch (err) { setError(err); } finally { setBusy(false); }
  }

  async function preparePublication() {
    if (form.current && !form.current.reportValidity()) return;
    setError(null); setBusy(true); setMessage("");
    try {
      validate();
      if (!fields.content.trim() && !exercises.length) throw new Error("Добавьте текст урока или хотя бы одно упражнение.");
      const [freshCourse, freshTopics] = await Promise.all([catalogApi.course(courseID), allCatalogPages((offset) => catalogApi.topics(courseID, { limit: 100, offset }))]);
      setDialog({ kind: "publish", course: freshCourse, chain: topicChain(fields.topic_id, freshTopics) });
    } catch (err) { setError(err); } finally { setBusy(false); }
  }

  async function publish() {
    if (dialog?.kind !== "publish") return;
    setBusy(true); setError(null);
    try {
      const currentID = await saveDraft();
      if (dialog.course.status !== "published") await catalogApi.publishCourse(dialog.course.id);
      for (const topic of dialog.chain) if (topic.status !== "published") await catalogApi.publishTopic(topic.id);
      await catalogApi.publishLesson(currentID);
      setPendingPublication(false); setDialog(null); router.push(`/tasks/library/${currentID}`);
    } catch (err) { setDialog(null); setError(err); } finally { setBusy(false); }
  }

  const previewExercises: Exercise[] = exercises.map((item) => ({ id: item.id ?? item.clientID, kind: item.kind, prompt: item.prompt, options: options(item), difficulty: item.difficulty, position: item.position, config: previewConfig(item.configText),media:item.media }));
  const canPublish = !busy && (!id || dirty || pendingPublication || initial.lesson?.status !== "published");
  return <section className="material-editor">
    <Link href="/tasks/library" className="inline-link arrow-link material-back"><Icon name="arrow" className="back" size={17} />В библиотеку</Link>
    {id?<PDFSourcePanel lessonID={id} dirty={dirty||busy} refreshKey={saved+exercises.map(e=>e.saved??"").join("")} onExplanation={text=>change({content:fields.content?`${fields.content}\n\n${text}`:text})} onAdd={(prompt,kind,media)=>setExercises(items=>[...items,{clientID:crypto.randomUUID(),kind,prompt,media,configText:"{}",optionsText:"",difficulty:fields.difficulty,position:Math.min(1000000,Math.max(-1,...items.map(e=>e.position))+1),answer:"",explanation:""}])}/>:null}
    <form ref={form} onSubmit={save}>
      <header className="material-page-heading"><div><p className="material-kicker">{dirty ? "Есть несохранённые изменения" : "Черновик"}</p><h1>{id ? "Редактирование урока" : "Новый урок"}</h1></div><div className="actions"><button type="submit" className="secondary" disabled={busy}>{busy ? "Сохраняем…" : "Сохранить черновик"}</button><button type="button" disabled={!canPublish} onClick={preparePublication}>Опубликовать<Icon name="arrow" size={18} /></button></div></header>
      <div className="material-editor-tabs" role="tablist" aria-label="Режим редактора"><button type="button" role="tab" id="material-edit-tab" aria-selected={mode === "edit"} aria-controls="material-editor-panel" disabled={busy} onClick={() => setMode("edit")}>Редактирование</button><button type="button" role="tab" id="material-preview-tab" aria-selected={mode === "preview"} aria-controls="material-editor-panel" disabled={busy} onClick={() => setMode("preview")}><Icon name="eye" size={18} />Предпросмотр</button></div>
      {error ? <LibraryError error={error} /> : null}{message ? <p className="material-success" role="status"><Icon name="check" size={18} />{message}</p> : null}
      <div id="material-editor-panel" role="tabpanel" aria-labelledby={mode === "edit" ? "material-edit-tab" : "material-preview-tab"}>
        {mode === "preview" ? <div className="material-preview"><p className="material-kicker">{course?.title} · {classLabel(fields.grade_min, fields.grade_max)}</p><h2>{fields.title || "Без названия"}</h2><LessonArticle lesson={fields} exercises={previewExercises} /></div> : <fieldset disabled={busy} className="material-fieldset">
          <section className="material-editor-section"><h2>Основное</h2><div className="material-two-fields"><div><label>Курс<Select aria-label="Курс" value={courseID} onChange={(e) => selectCourse(e.target.value)} required disabled={Boolean(id)}><option value="">Выберите курс</option>{courses.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}</Select></label>{!id ? <button type="button" className="link-button material-add-link" onClick={() => setDialog({ kind: "course" })}><Icon name="plus" size={16} />Новый курс</button> : null}</div><div><label>Тема<Select aria-label="Тема" value={fields.topic_id} onChange={(e) => change({ topic_id: e.target.value })} required disabled={!courseID}><option value="">Выберите тему</option>{courseTopics.map((topic) => <option key={topic.id} value={topic.id}>{topicLabel(topic, topics)}</option>)}</Select></label><button type="button" className="link-button material-add-link" disabled={!course} onClick={() => setDialog({ kind: "topic" })}><Icon name="plus" size={16} />Новая тема</button></div></div>
            <label>Название урока<input aria-label="Название урока" value={fields.title} onChange={(e) => change({ title: e.target.value })} maxLength={200} required /></label><label>Краткое описание<textarea aria-label="Краткое описание" rows={2} value={fields.summary} onChange={(e) => change({ summary: e.target.value })} maxLength={10000} /></label><ClassificationFields value={fields} change={change} />
          </section>
          <section className="material-editor-section"><h2>Содержание урока</h2><label>Текст урока<textarea aria-label="Текст урока" rows={12} value={fields.content} onChange={(e) => change({ content: e.target.value })} maxLength={250000} /></label></section>
          <ExampleEditor value={fields.example} change={example=>change({example})}/>
          <section className="material-editor-section"><div className="material-results-heading"><h2>Упражнения <span>{exercises.length}</span></h2><button type="button" className="secondary" onClick={() => setExercises((items) => [...items, { clientID: crypto.randomUUID(), kind: "text", configText: "{}", prompt: "", optionsText: "", difficulty: 1, position: Math.min(1000000, Math.max(-1, ...items.map((item) => item.position)) + 1), answer: "", explanation: "" }])}><Icon name="plus" size={17} />Добавить упражнение</button></div>
            {!exercises.length ? <p className="muted">Упражнений пока нет.</p> : exercises.map((exercise, index) => <div className="material-exercise-editor" key={exercise.clientID}><div className="material-results-heading"><h3>Упражнение {index + 1}</h3>{!exercise.id ? <button type="button" className="icon-button" aria-label={`Удалить несохранённое упражнение ${index + 1}`} title="Удалить несохранённое упражнение" onClick={() => setExercises((items) => items.filter((item) => item.clientID !== exercise.clientID))}><Icon name="close" size={18} /></button> : null}</div><div className="material-two-fields"><label>Тип упражнения<Select aria-label="Тип упражнения" value={exercise.kind} onChange={(e) => updateExercise(exercise.clientID, { kind: e.target.value as Exercise["kind"], answer: "", configText: JSON.stringify(defaultVisualConfig(e.target.value), null, 2) })}>{Object.entries(exerciseRegistry).map(([kind, entry]) => <option key={kind} value={kind}>{entry.label}</option>)}</Select></label><label>Сложность упражнения<Select aria-label="Сложность упражнения" value={exercise.difficulty} onChange={(e) => updateExercise(exercise.clientID, { difficulty: Number(e.target.value) })}>{difficultyLabels.slice(1).map((label, i) => <option key={label} value={i + 1}>{label}</option>)}</Select></label></div><label>Условие упражнения<textarea aria-label="Условие упражнения" rows={3} value={exercise.prompt} onChange={(e) => updateExercise(exercise.clientID, { prompt: e.target.value })} maxLength={20000} required /></label>
              {exercise.media?.map((m,i)=><div className="pdf-attached-figure" key={m.asset_id}><img src={`/api/v1/pdf-assets/${m.asset_id}`} alt={m.alt}/><label>Описание рисунка<input aria-label={`Описание рисунка ${i+1}`} value={m.alt} maxLength={1000} required onChange={e=>updateExercise(exercise.clientID,{media:exercise.media?.map((v,n)=>n===i?{...v,alt:e.target.value}:v)})}/></label><button type="button" className="icon-button" aria-label={`Убрать рисунок ${i+1}`} title="Убрать рисунок" onClick={()=>updateExercise(exercise.clientID,{media:exercise.media?.filter(v=>v.asset_id!==m.asset_id)})}><Icon name="close" size={18}/></button></div>)}
              {isVisual(exercise.kind) ? <label>Настройки задания (JSON)<textarea aria-label="Настройки задания (JSON)" rows={8} value={exercise.configText} maxLength={32000} spellCheck={false} onChange={(event) => updateExercise(exercise.clientID, { configText: event.target.value, answer: "" })} required /></label> : null}
              {exercise.kind === "single_choice" || exercise.kind === "multiple_choice" ? <label>Варианты ответа, каждый с новой строки<textarea aria-label="Варианты ответа, каждый с новой строки" rows={4} value={exercise.optionsText} onChange={(e) => updateExercise(exercise.clientID, { optionsText: e.target.value })} required /></label> : null}
              <div className="material-private-key"><div className="material-kicker"><Icon name="lock" size={16} />Закрытый ключ</div><ExerciseKeyFields kind={exercise.kind} config={previewConfig(exercise.configText)} options={options(exercise)} answer={exercise.answer} change={(answer) => updateExercise(exercise.clientID, { answer })} /><label>Закрытое пояснение<textarea aria-label="Закрытое пояснение" rows={2} value={exercise.explanation} onChange={(e) => updateExercise(exercise.clientID, { explanation: e.target.value })} maxLength={10000} /></label><label>Подсказки (по одной на строку)<textarea aria-label="Подсказки" rows={3} value={exercise.hintsText??""} onChange={(e)=>updateExercise(exercise.clientID,{hintsText:e.target.value})} maxLength={20000} /></label></div>
            </div>)}
          </section>
        </fieldset>}
      </div>
    </form>
    {dialog?.kind === "course" ? <CreateCourseDialog close={() => setDialog(null)} created={(created) => { setCourses((items) => [...items, created]); setCourseID(created.id); change({ topic_id: "", track: created.track, grade_min: created.grade_min, grade_max: created.grade_max, difficulty: created.difficulty }); setDialog(null); }} /> : null}
    {dialog?.kind === "topic" && course ? <CreateTopicDialog course={course} topics={courseTopics} close={() => setDialog(null)} created={(created) => { setTopics((items) => [...items, created]); change({ topic_id: created.id }); setDialog(null); }} /> : null}
    {dialog?.kind === "publish" ? <EditorDialog title="Публикация урока" close={() => setDialog(null)} busy={busy}><p><strong>{fields.title}</strong></p><p>Доступ: {dialog.course.visibility === "organization" ? "вся организация" : "ваши связанные ученики"}.</p>{dialog.course.status !== "published" || dialog.chain.some((topic) => topic.status !== "published") ? <><p>Вместе с уроком будут опубликованы:</p><ul>{dialog.course.status !== "published" ? <li>Курс «{dialog.course.title}»</li> : null}{dialog.chain.filter((topic) => topic.status !== "published").map((topic) => <li key={topic.id}>Тема «{topic.title}»</li>)}</ul></> : null}{dialog.course.status === "archived" || dialog.chain.some((topic) => topic.status === "archived") ? <p className="material-notice">Восстановление архивной ветки также откроет другие опубликованные материалы в ней.</p> : null}<p className="muted">Ученики получат новую версию. Предыдущие опубликованные версии сохранятся.</p><div className="material-dialog-actions"><button className="secondary" type="button" disabled={busy} onClick={() => setDialog(null)}>Отмена</button><button type="button" disabled={busy} onClick={publish}>{busy ? "Публикуем…" : "Подтвердить публикацию"}</button></div></EditorDialog> : null}
  </section>;
}
