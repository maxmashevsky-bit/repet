"use client";
import { Select } from "../ui/Select";


import Link from "next/link";
import { FormEvent, ReactNode, useEffect, useRef, useState } from "react";
import { ApiError } from "../../shared/api/client";
import { tutorPlannerApi, type PlanLesson, type PlanStudent, type TutorPlan } from "../../shared/api/tutor-planner";
import { useWorkspace } from "../app-shell/AppShell";
import { Icon } from "../ui/Icon";
import { clock, contactURL, days, incomeModel, money, planSummary, slots } from "./calculations";
import {todayISO} from "../../shared/lib/date";
import type {PlanCalendarResult} from "../../shared/api/tutor-planner";

type Tab = "students" | "week" | "windows" | "income";
const tabs: [Tab, string][] = [["students", "Ученики"], ["week", "Типовая неделя"], ["windows", "Рабочие окна"], ["income", "Доход"]];
type Editor = { kind: "student"; item: PlanStudent } | { kind: "lesson"; item: PlanLesson };

export function TutorPlanner() {
  const { dashboard,refresh } = useWorkspace();
  const [plan, setPlan] = useState<TutorPlan | null>(null);
  const [tab, setTab] = useState<Tab>("students");
  const [editor, setEditor] = useState<Editor | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [query, setQuery] = useState("");
  const [reload, setReload] = useState(0);
  const[from,setFrom]=useState(()=>todayISO(dashboard.user.timezone)),[to,setTo]=useState(()=>todayISO(dashboard.user.timezone)),[exporting,setExporting]=useState(false),[published,setPublished]=useState<PlanCalendarResult|null>(null);
  const exportNonce=useRef<{key:string;id:string}|null>(null);
  const busy = useRef(false);
  const tutor = dashboard.user.role === "tutor";
  useEffect(() => {
    if (!tutor) return;
    let current = true;
    tutorPlannerApi.get().then(p => { if (current) { setPlan(p); setError(""); setDirty(false); } }).catch(e => { if (current) setError(e instanceof Error ? e.message : "Не удалось загрузить план"); }).finally(() => { if (current) setLoading(false); });
    return () => { current = false; };
  }, [tutor, reload]);
  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => { event.preventDefault(); };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  if (!tutor) return <section className="full-state"><h1>Рабочий план преподавателя</h1><p>Этот раздел доступен только преподавателю.</p><Link href="/calendar">Вернуться в календарь</Link></section>;
  if (loading) return <div className="full-state" role="status">Загружаем рабочий план…</div>;
  if (!plan) return <section className="full-state"><p role="alert">{error}</p><button type="button" onClick={() => { setLoading(true); setReload(v => v + 1); }}>Повторить</button></section>;
  const p = plan;
  const accounts=dashboard.relations.filter(r=>r.status==="active").map(r=>({id:r.id,name:dashboard.conversations.find(c=>c.student_id===r.student_id)?.student_name??"Ученик"}));
  const summary = planSummary(p), model = incomeModel(p);

  async function save(next: TutorPlan) {
    if (busy.current) return false;
    busy.current = true; setSaving(true); setError("");
    try { setPlan(await tutorPlannerApi.save(next)); setDirty(false); return true; }
    catch (e) {
      setError(e instanceof ApiError && e.status === 409 ? "План изменён в другой вкладке. Обновите данные перед повторным редактированием." : e instanceof ApiError && e.status === 400 ? "Проверьте поля: имя должно быть уникальным, контакт — ссылкой или @ником, занятие — в пределах 09:00–22:00." : "Не удалось сохранить план. Ваши введённые данные остаются на экране.");
      return false;
    } finally { busy.current = false; setSaving(false); }
  }
  function newStudent() {
    setEditor({ kind: "student", item: { id: crypto.randomUUID(), name: "", subject: "", price_kopecks: p.calculator.price_kopecks, duration: p.calculator.duration, contact: "" } });
  }
  function newLesson(weekday = 1, minute = 540) {
    if (!p.students.length) { setTab("students"); newStudent(); return; }
    setEditor({ kind: "lesson", item: { id: crypto.randomUUID(), student_id: p.students[0].id, weekday, start_minute: minute, duration: Math.min(p.students[0].duration, 1320 - minute), paused: false } });
  }
  function editDraft(next: TutorPlan) { setPlan(next); setDirty(true); }
  async function publishCalendar(e:FormEvent){e.preventDefault();if(busy.current||exporting||dirty)return;busy.current=true;setExporting(true);setError("");setPublished(null);const key=`${p.revision}:${from}:${to}`;if(exportNonce.current?.key!==key)exportNonce.current={key,id:crypto.randomUUID()};try{const result=await tutorPlannerApi.publish(p.revision,from,to,exportNonce.current.id);setPublished(result);await refresh();}catch(e){setError(e instanceof ApiError&&e.status===409?"Есть пересечение с календарём или план изменён. Проверьте расписание и обновите данные.":e instanceof ApiError&&e.status===400?"Выберите период от сегодня до 12 недель, не дальше года вперёд.":"Не удалось перенести занятия. Проверьте связь учеников и повторите отправку.");}finally{busy.current=false;setExporting(false);}}
  function toggleWindow(weekday: number, minute: number) {
    const exists = p.windows.some(w => w.weekday === weekday && w.start_minute === minute);
    editDraft({ ...p, windows: exists ? p.windows.filter(w => w.weekday !== weekday || w.start_minute !== minute) : [...p.windows, { weekday, start_minute: minute }] });
  }
  async function deleteStudent(s: PlanStudent) {
    const count = p.lessons.filter(l => l.student_id === s.id).length;
    if (window.confirm(`Удалить «${s.name}» из рабочего плана${count ? ` и ${count} занятий типовой недели` : ""}? Аккаунт и календарные занятия не удаляются.`)) await save({ ...p, students: p.students.filter(x => x.id !== s.id), lessons: p.lessons.filter(l => l.student_id !== s.id) });
  }
  const sortedLessons = [...p.lessons].sort((a, b) => a.weekday - b.weekday || a.start_minute - b.start_minute || a.id.localeCompare(b.id));
  return <section className="tutor-planner">
    <div className="page-title page-title-action"><div><Link className="inline-link" href="/calendar">Календарь <Icon name="chevron" size={16} /></Link><h1>Рабочий план</h1><p>Типовая неделя · {dashboard.user.timezone}</p></div><div className="planner-save"><span role="status" aria-live="polite">{saving ? "Сохраняем…" : dirty ? "Есть несохранённые изменения" : "Все изменения сохранены"}</span><button type="button" disabled={saving || !dirty} onClick={() => void save(p)}><Icon name="check" size={18} />Сохранить</button></div></div>
    {error ? <div className="planner-error" role="alert"><p>{error}</p><button type="button" className="secondary" disabled={saving} onClick={() => { if (!dirty || window.confirm("Загрузить сохранённый план? Несохранённые изменения будут потеряны.")) { setLoading(true); setReload(v => v + 1); } }}>Обновить данные</button></div> : null}
    <dl className="planner-metrics"><div><dt>Учеников с занятиями</dt><dd>{summary.students}</dd></div><div><dt>Занятий в неделю</dt><dd>{summary.active.length}</dd></div><div><dt>Часов в неделю</dt><dd>{decimal(summary.hours)}</dd></div><div><dt>План в неделю</dt><dd>{money(summary.revenue)}</dd></div><div><dt>План за {p.calculator.weeks} нед.</dt><dd>{money(summary.monthly)}</dd></div></dl>
    <div className="planner-tabs" role="tablist" aria-label="Рабочий план">{tabs.map(([key, label]) => <button id={`planner-tab-${key}`} key={key} type="button" role="tab" aria-selected={tab === key} aria-controls="planner-content" tabIndex={tab === key ? 0 : -1} onClick={() => setTab(key)} onKeyDown={e => { if (["ArrowLeft", "ArrowRight", "Home", "End"].includes(e.key)) { e.preventDefault(); const i = tabs.findIndex(t => t[0] === tab); const next = e.key === "Home" ? 0 : e.key === "End" ? tabs.length - 1 : (i + (e.key === "ArrowRight" ? 1 : -1) + tabs.length) % tabs.length; setTab(tabs[next][0]); document.getElementById(`planner-tab-${tabs[next][0]}`)?.focus(); } }}>{label}{key === "students" ? <span>{p.students.length}</span> : key === "week" && summary.conflictIDs.size ? <span className="planner-conflict-count">{summary.conflictIDs.size}</span> : null}</button>)}</div>
    <div id="planner-content" role="tabpanel" aria-labelledby={`planner-tab-${tab}`}>
      {tab === "students" ? <>
        <div className="planner-section-head"><h2>Мои ученики</h2><div className="planner-actions"><label className="planner-search"><Icon name="search" size={18} /><input aria-label="Найти ученика" placeholder="Найти ученика" value={query} onChange={e => setQuery(e.target.value)} /></label><button type="button" disabled={saving} onClick={newStudent}><Icon name="plus" size={18} />Добавить ученика</button></div></div>
        {!p.students.length ? <div className="planner-empty"><Icon name="user" size={36} /><h3>Первый ученик</h3><p>Пока нет записей в рабочем плане.</p><button type="button" disabled={saving} onClick={newStudent}><Icon name="plus" />Добавить ученика</button></div> : <div className="planner-table-scroll"><table className="planner-table"><thead><tr><th>Ученик / предмет</th><th>Цена за урок</th><th>Длительность</th><th>Занятий в неделю</th><th>Контакт</th><th><span className="sr-only">Действия</span></th></tr></thead><tbody>{p.students.filter(s => `${s.name} ${s.subject}`.toLocaleLowerCase("ru").includes(query.toLocaleLowerCase("ru"))).map(s => <tr key={s.id}><th scope="row"><strong>{s.name}</strong><small>{s.subject || "—"}</small></th><td>{money(s.price_kopecks)}</td><td>{s.duration} мин</td><td>{summary.active.filter(l => l.student_id === s.id).length}</td><td>{contactURL(s.contact) ? <a className="inline-link" href={contactURL(s.contact)!} target="_blank" rel="noopener noreferrer">{s.contact.startsWith("@") ? s.contact : "Написать"}<Icon name="arrow" size={16} /></a> : "—"}</td><td><div className="planner-row-actions"><button className="secondary" type="button" disabled={saving} onClick={() => setEditor({ kind: "student", item: { ...s } })}>Изменить</button><button className="icon-button" type="button" title={`Удалить ${s.name}`} aria-label={`Удалить ${s.name}`} disabled={saving} onClick={() => void deleteStudent(s)}><Icon name="close" size={18} /></button></div></td></tr>)}</tbody></table>{p.students.length > 0 && !p.students.some(s => `${s.name} ${s.subject}`.toLocaleLowerCase("ru").includes(query.toLocaleLowerCase("ru"))) ? <p className="empty">Ученики не найдены.</p> : null}</div>}
      </> : null}
      {tab === "week" ? <>
        <div className="planner-section-head"><h2>Типовая неделя</h2><button type="button" disabled={saving} onClick={() => newLesson()}><Icon name="plus" size={18} />Добавить занятие</button></div>
        <form className="planner-calendar-publish" onSubmit={e=>void publishCalendar(e)}><h3>Занятия в календаре</h3><label>С даты<input type="date" aria-label="Перенести с даты" min={todayISO(dashboard.user.timezone)} required value={from} disabled={exporting} onChange={e=>setFrom(e.target.value)}/></label><label>По дату<input type="date" aria-label="Перенести по дату" min={from} required value={to} disabled={exporting} onChange={e=>setTo(e.target.value)}/></label><button type="submit" disabled={saving||exporting||dirty||!p.lessons.length}><Icon name="calendar" size={18}/>{exporting?"Переносим…":"Перенести в календарь"}</button>{dirty?<p className="muted">Сначала сохраните изменения плана.</p>:null}{published?<p className="notice" role="status">Создано: {published.created}. Уже в календаре: {published.existing}. Без привязки: {published.unlinked}. На паузе: {published.paused}. <Link href={`/calendar?view=week&date=${published.from}`}>Открыть календарь</Link></p>:null}</form>
        <div className="planner-legend"><span className="lesson">Занятие</span><span className="free">Свободное окно</span><span className="conflict">Пересечение</span></div>
        {summary.conflictIDs.size ? <p className="planner-conflict-notice" role="status">Занятий с пересечениями: {summary.conflictIDs.size}. Они включены в плановую выручку.</p> : null}
        <div className="planner-table-scroll"><table className="planner-week"><caption className="sr-only">Типовая неделя с 09:00 до 22:00, время профиля</caption><thead><tr><th>Время</th>{days.map(d => <th key={d}>{d}</th>)}</tr></thead><tbody>{slots.map(minute => <tr key={minute}><th scope="row">{clock(minute)}</th>{days.map((d, i) => {
          const lessons = summary.active.filter(l => l.weekday === i + 1 && l.start_minute < minute + 30 && l.start_minute + l.duration > minute);
          const free = p.windows.some(w => w.weekday === i + 1 && w.start_minute === minute);
          const conflict = lessons.some(l => summary.conflictIDs.has(l.id));
          return <td key={d} className={conflict ? "is-conflict" : lessons.length ? "has-lesson" : free ? "is-free" : ""}><button type="button" disabled={saving} aria-label={`${d}, ${clock(minute)}${lessons.length ? `, ${lessons.map(l => p.students.find(s => s.id === l.student_id)?.name).join(", ")}` : ", добавить занятие"}`} onClick={() => lessons.length ? setEditor({ kind: "lesson", item: { ...lessons[0] } }) : newLesson(i + 1, minute)}>{lessons.length ? lessons.map(l => <span key={l.id}><strong>{p.students.find(s => s.id === l.student_id)?.name}</strong><small>{clock(l.start_minute)}–{clock(l.start_minute + l.duration)}</small></span>) : free ? "Свободно" : <span aria-hidden="true">·</span>}</button></td>;
        })}</tr>)}</tbody><tfoot><tr><th>Занятий</th>{summary.daily.map((d, i) => <td key={i}>{d.lessons}</td>)}</tr><tr><th>Часов</th>{summary.daily.map((d, i) => <td key={i}>{decimal(d.hours)}</td>)}</tr><tr><th>Выручка</th>{summary.daily.map((d, i) => <td key={i}>{money(d.revenue)}</td>)}</tr></tfoot></table></div>
        <h3 className="planner-subheading">Все занятия</h3><div className="planner-table-scroll"><table className="planner-table"><thead><tr><th>Ученик</th><th>День / время</th><th>Цена</th><th>Статус</th><th>Действия</th></tr></thead><tbody>{sortedLessons.map(l => { const s = p.students.find(s => s.id === l.student_id); return <tr key={l.id}><th scope="row">{s?.name}</th><td>{days[l.weekday - 1]}<small>{clock(l.start_minute)}–{clock(l.start_minute + l.duration)}</small></td><td>{money(s?.price_kopecks ?? 0)}</td><td>{l.paused ? "Пауза" : summary.conflictIDs.has(l.id) ? <span className="field-error">Пересечение</span> : "Идёт"}</td><td><div className="planner-row-actions"><button type="button" className="secondary" disabled={saving} onClick={() => setEditor({ kind: "lesson", item: { ...l } })}>Изменить</button><button type="button" className="secondary" disabled={saving} aria-label={`${l.paused ? "Возобновить" : "Пауза"}: ${s?.name}, ${days[l.weekday - 1]}, ${clock(l.start_minute)}`} onClick={() => void save({ ...p, lessons: p.lessons.map(x => x.id === l.id ? { ...x, paused: !x.paused } : x) })}>{l.paused ? "Возобновить" : "Пауза"}</button><button type="button" className="icon-button" title="Удалить занятие" aria-label={`Удалить занятие ${s?.name}, ${days[l.weekday - 1]}, ${clock(l.start_minute)}`} disabled={saving} onClick={() => { if (window.confirm("Удалить занятие из типовой недели?")) void save({ ...p, lessons: p.lessons.filter(x => x.id !== l.id) }); }}><Icon name="close" size={18} /></button></div></td></tr>; })}</tbody></table>{!p.lessons.length ? <p className="empty">В типовой неделе пока нет занятий.</p> : null}</div>
      </> : null}
      {tab === "windows" ? <>
        <div className="planner-section-head"><h2>Рабочие окна</h2><div className="planner-actions"><button type="button" className="secondary" disabled={saving} onClick={() => { const windows = [...p.windows]; for (const l of summary.active) for (const minute of slots) if (l.start_minute < minute + 30 && l.start_minute + l.duration > minute && !windows.some(w => w.weekday === l.weekday && w.start_minute === minute)) windows.push({ weekday: l.weekday, start_minute: minute }); editDraft({ ...p, windows }); }}>По текущим занятиям</button><button type="button" disabled={saving || !dirty} onClick={() => void save(p)}><Icon name="check" size={18} />Сохранить окна</button></div></div>
        <div className="planner-table-scroll"><table className="planner-week planner-windows"><caption className="sr-only">Рабочие интервалы по 30 минут</caption><thead><tr><th>Время</th>{days.map((d, i) => <th key={d}>{d}<button className="planner-day-toggle" disabled={saving} type="button" onClick={() => { const all = slots.every(minute => p.windows.some(w => w.weekday === i + 1 && w.start_minute === minute)); editDraft({ ...p, windows: [...p.windows.filter(w => w.weekday !== i + 1), ...(all ? [] : slots.map(start_minute => ({ weekday: i + 1, start_minute })))] }); }}>{slots.every(minute => p.windows.some(w => w.weekday === i + 1 && w.start_minute === minute)) ? "Снять все" : "Выбрать все"}</button></th>)}</tr></thead><tbody>{slots.map(minute => <tr key={minute}><th scope="row">{clock(minute)}</th>{days.map((d, i) => <td key={d}><label><input type="checkbox" disabled={saving} checked={p.windows.some(w => w.weekday === i + 1 && w.start_minute === minute)} aria-label={`${d}, ${clock(minute)}–${clock(minute + 30)}`} onChange={() => toggleWindow(i + 1, minute)} /><span aria-hidden="true" /></label></td>)}</tr>)}</tbody><tfoot><tr><th>Часов</th>{days.map((_, i) => <td key={i}>{decimal(p.windows.filter(w => w.weekday === i + 1).length / 2)}</td>)}</tr><tr><th>Поместится уроков</th>{model?.dayCapacity.map((v, i) => <td key={i}>{v}</td>)}</tr></tfoot></table></div>
      </> : null}
      {tab === "income" ? <>
        <div className="planner-section-head"><h2>Плановая выручка</h2><span className="planner-muted">До расходов · расчётный месяц {p.calculator.weeks} нед.</span></div>
        <div className="planner-income"><form className="form-stack planner-calculator" onSubmit={e => { e.preventDefault(); void save(p); }}><h3>Параметры</h3>{([
          ["goal_kopecks", "Цель в месяц, ₽", 0, 1000000000, 0.01, 100], ["price_kopecks", "Цена с человека за урок, ₽", 0.01, 10000000, 0.01, 100], ["lessons_per_student", "Занятий с учеником в неделю", 1, 21, 1, 1], ["duration", "Длительность для расчёта, мин", 1, 780, 1, 1], ["weeks", "Недель в расчётном месяце", 1, 6, 0.01, 1], ["people", "Человек на занятии", 1, 100, 1, 1], ["preparation_hours", "Подготовка и общие занятия, ч/нед.", 0, 168, 0.5, 1]
        ] as const).map(([key, label, min, max, step, scale]) => <label key={key}>{label}<input required type="number" disabled={saving} min={min} max={max} step={step} value={p.calculator[key] / scale} onChange={e => editDraft({ ...p, calculator: { ...p.calculator, [key]: scale === 100 ? Math.round(Number(e.target.value) * scale) : Number(e.target.value) } })} /></label>)}<button type="submit" disabled={saving || !dirty || !model}><Icon name="check" size={18} />Сохранить параметры</button></form>
        {model ? <div className="planner-income-results"><div className={`planner-goal ${model.spare < 0 ? "needs-attention" : ""}`}><span>Цель: {money(p.calculator.goal_kopecks)}</span><h3>{model.status}</h3><strong>{model.required} / {model.places}</strong><p>нужно учеников / доступно мест</p></div><dl className="planner-result-list">{[
          ["План по расписанию в месяц", money(summary.monthly)], ["Учеников с занятиями", summary.students], ["Выручка на ученика в месяц", money(summary.perStudent)], ["Часов на ученика в неделю", summary.students ? decimal(summary.hours / summary.students) : "—"], ["Выручка за час сейчас", money(summary.hourly)], ["Абонемент при выбранной цене", money(model.subscription)], ["Рабочих часов по окнам", decimal(model.workingHours)], ["Часов на платные занятия", decimal(model.paidHours)], ["Уроков помещается в неделю", model.capacity], ["Полных групп при этой частоте", model.groups], ["Вместимость, учеников", model.places], ["Нужно учеников для цели", model.required], ["Запас / дефицит мест", model.spare], ["Выручка при полной загрузке", money(model.revenue)], ["Сверх цели / не хватает", money(model.gap)], ["Минимальный абонемент для цели", money(model.minimumSubscription)], ["Нужная выручка за час", money(model.hourly)], ["Нужная ставка ученика за час", money(model.studentHourly)], ["Загрузка групп для цели", model.load === null ? "—" : `${decimal(model.load * 100)}%`], ["Дополнительные часы для нужных уроков", decimal(model.extraHours)]
        ].map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl></div> : <p className="field-error">Введите корректные параметры расчёта.</p>}</div>
        {model ? <><h3 className="planner-subheading">Сравнение форматов</h3><div className="planner-table-scroll"><table className="planner-table"><thead><tr><th>Формат</th><th>Полных групп</th><th>Мест</th><th>Ставка за час для цели</th><th>Мин. абонемент</th><th>Выручка</th><th>Сверх цели / дефицит</th></tr></thead><tbody>{model.formats.map((f, i) => <tr key={f.people}><th scope="row">{["Один на один", "Пара", "Мини-группа", "Группа", "Большая группа"][i]}<small>{f.people} чел.</small></th><td>{f.groups}</td><td>{f.places}</td><td>{money(f.hourly)}</td><td>{money(f.minimumSubscription)}</td><td>{money(f.revenue)}</td><td>{money(f.gap)}</td></tr>)}</tbody></table></div></> : null}
      </> : null}
    </div>
    {editor ? <PlanDialog title={editor.kind === "student" ? p.students.some(s => s.id === editor.item.id) ? "Изменить ученика" : "Добавить ученика" : p.lessons.some(l => l.id === editor.item.id) ? "Изменить занятие" : "Добавить занятие"} onClose={() => { if (!saving) setEditor(null); }}>
      {editor.kind === "student" ? <StudentForm item={editor.item} saving={saving} error={error} accounts={accounts.filter(a=>!p.students.some(s=>s.id!==editor.item.id&&s.relation_id===a.id))} names={p.students.filter(s => s.id !== editor.item.id).map(s => s.name)} onSubmit={async item => { if (await save({ ...p, students: [...p.students.filter(s => s.id !== item.id), item] })) setEditor(null); }} /> : <LessonForm item={editor.item} students={p.students} saving={saving} error={error} onSubmit={async item => { if (await save({ ...p, lessons: [...p.lessons.filter(l => l.id !== item.id), item] })) setEditor(null); }} />}
    </PlanDialog> : null}
  </section>;
}

function decimal(n: number) { return new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 }).format(n); }
function PlanDialog({ title, children, onClose }: { title: string; children: ReactNode; onClose: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { ref.current?.showModal(); }, []);
  return <dialog className="planner-dialog" ref={ref} aria-labelledby="planner-dialog-title" onCancel={e => { e.preventDefault(); onClose(); }}><div className="planner-section-head"><h2 id="planner-dialog-title">{title}</h2><button className="icon-button" title="Закрыть" aria-label="Закрыть" type="button" onClick={onClose}><Icon name="close" /></button></div>{children}</dialog>;
}
function StudentForm({ item, saving, error, names, accounts,onSubmit }: { item: PlanStudent; saving: boolean; error: string; names: string[]; accounts:{id:string;name:string}[];onSubmit: (s: PlanStudent) => Promise<void> }) {
  const [draft, setDraft] = useState(item);
  const [localError, setLocalError] = useState("");
  function submit(e: FormEvent) { e.preventDefault(); const name = draft.name.trim(); if (names.some(n => n.toLocaleLowerCase("ru") === name.toLocaleLowerCase("ru"))) { setLocalError("Ученик с таким именем уже есть. Укажите фамилию."); return; } if (draft.contact.trim() && !contactURL(draft.contact.trim())) { setLocalError("Введите @ник Telegram (5–32 символа) или полную ссылку http(s)."); return; } void onSubmit({ ...draft, name, subject: draft.subject.trim(), contact: draft.contact.trim() }); }
  return <form className="form-stack" onSubmit={submit}><label>Аккаунт ученика<Select aria-label="Аккаунт ученика" disabled={saving} value={draft.relation_id??""} onChange={e=>setDraft({...draft,relation_id:e.target.value,name:draft.name||accounts.find(a=>a.id===e.target.value)?.name||""})}><option value="">Пока без аккаунта</option>{draft.relation_id&&!accounts.some(a=>a.id===draft.relation_id)?<option value={draft.relation_id}>Связь недоступна</option>:null}{accounts.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</Select></label><Link className="inline-link" href="/#invite-student">Пригласить нового ученика</Link><label>Имя и фамилия<input autoFocus required maxLength={120} value={draft.name} disabled={saving} onChange={e => setDraft({ ...draft, name: e.target.value })} /></label><label>Класс / предмет<input maxLength={180} value={draft.subject} disabled={saving} placeholder="Например, 7 класс · математика" onChange={e => setDraft({ ...draft, subject: e.target.value })} /></label><div className="planner-form-pair"><label>Цена за урок, ₽<input required type="number" min={0} max={10000000} step="0.01" value={draft.price_kopecks / 100} disabled={saving} onChange={e => setDraft({ ...draft, price_kopecks: Math.round(Number(e.target.value) * 100) })} /></label><label>Длительность, мин<input required type="number" min={1} max={780} step={1} value={draft.duration} disabled={saving} onChange={e => setDraft({ ...draft, duration: Number(e.target.value) })} /></label></div><label>Контакт<input maxLength={1000} value={draft.contact} disabled={saving} placeholder="@username или https://…" onChange={e => setDraft({ ...draft, contact: e.target.value })} /></label>{localError || error ? <p className="field-error" role="alert">{localError || error}</p> : null}<button type="submit" disabled={saving || !draft.name.trim()}>{saving ? "Сохраняем…" : "Сохранить ученика"}</button></form>;
}
function LessonForm({ item, students, saving, error, onSubmit }: { item: PlanLesson; students: PlanStudent[]; saving: boolean; error: string; onSubmit: (l: PlanLesson) => Promise<void> }) {
  const [draft, setDraft] = useState(item);
  const invalid = draft.start_minute < 540 || draft.start_minute + draft.duration > 1320;
  return <form className="form-stack" onSubmit={e => { e.preventDefault(); if (!invalid) void onSubmit(draft); }}><label>Ученик<Select autoFocus required disabled={saving} value={draft.student_id} onChange={e => setDraft({ ...draft, student_id: e.target.value, duration: students.find(s => s.id === e.target.value)?.duration ?? draft.duration })}>{students.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</Select></label><label>День недели<Select value={draft.weekday} disabled={saving} onChange={e => setDraft({ ...draft, weekday: Number(e.target.value) })}>{days.map((d, i) => <option key={d} value={i + 1}>{d}</option>)}</Select></label><div className="planner-form-pair"><label>Начало<input type="time" required min="09:00" max="21:59" step={60} value={clock(draft.start_minute)} disabled={saving} onChange={e => { const [h, m] = e.target.value.split(":").map(Number); setDraft({ ...draft, start_minute: h * 60 + m }); }} /></label><label>Длительность, мин<input type="number" required min={1} max={780} step={1} disabled={saving} value={draft.duration} onChange={e => setDraft({ ...draft, duration: Number(e.target.value) })} /></label></div><div className="planner-lesson-total"><span>Конец: <strong>{Number.isFinite(draft.start_minute) ? clock(draft.start_minute + draft.duration) : "—"}</strong></span><span>{money(students.find(s => s.id === draft.student_id)?.price_kopecks ?? 0)}</span></div><label className="planner-checkbox"><input type="checkbox" disabled={saving} checked={draft.paused} onChange={e => setDraft({ ...draft, paused: e.target.checked })} />Пауза занятия</label>{invalid ? <p className="field-error">Занятие должно быть в пределах 09:00–22:00.</p> : null}{error ? <p role="alert" className="field-error">{error}</p> : null}<button type="submit" disabled={saving || invalid}>{saving ? "Сохраняем…" : "Сохранить занятие"}</button></form>;
}
