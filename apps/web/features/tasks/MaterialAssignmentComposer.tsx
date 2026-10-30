"use client";
import { Select } from "../ui/Select";


import Link from "next/link";
import { useEffect,useRef,useState,type FormEvent } from "react";
import { catalogApi,allCatalogPages,type LearningLesson,type LessonVersion } from "../../shared/api/catalog";
import { materialApi,type ExerciseSet } from "../../shared/api/material-assignments";
import { useWorkspace } from "../app-shell/AppShell";
import { LibraryError,classLabel,countLabel } from "../material-library/LibraryUI";
import { Icon } from "../ui/Icon";

export function MaterialAssignmentComposer(){
 const {dashboard,refresh}=useWorkspace();
 const [student,setStudent]=useState(dashboard.relations.find(r=>r.status==="active")?.student_id??"");
 const [query,setQuery]=useState(""),[page,setPage]=useState(0),[source,setSource]=useState<"library"|"sets">("library");
 const [list,setList]=useState<{key:string;items?:LearningLesson[];total?:number;error?:unknown}>({key:""});
 const [sets,setSets]=useState<ExerciseSet[]>([]),[setID,setSetID]=useState("");
 const [selected,setSelected]=useState<LearningLesson|null>(null),[version,setVersion]=useState<LessonVersion|null>(null);
 const [subset,setSubset]=useState(false),[ids,setIDs]=useState<string[]>([]),[setTitle,setSetTitle]=useState("");
 const [due,setDue]=useState(""),[calendar,setCalendar]=useState(""),[mode,setMode]=useState<"practice"|"exam">("practice");
 const [busy,setBusy]=useState(false),[error,setError]=useState<unknown>(null),[created,setCreated]=useState("");
 const gate=useRef(false),nonce=useRef<{body:string;id:string}|null>(null),selection=useRef(0);
 const key=`${query}:${page}`;
 useEffect(()=>{const c=new AbortController();const timer=setTimeout(()=>{catalogApi.lessons({q:query,status:"published",limit:12,offset:page*12},c.signal).then(data=>{if(!c.signal.aborted)setList({key,items:data.items,total:data.total});}).catch(e=>{if(!c.signal.aborted)setList({key,error:e});});},200);return()=>{clearTimeout(timer);c.abort();};},[query,page,key]);
 useEffect(()=>{const c=new AbortController();allCatalogPages<ExerciseSet>(offset=>materialApi.sets(offset,c.signal)).then(data=>{if(!c.signal.aborted)setSets(data);}).catch(e=>{if(!c.signal.aborted)setError(e);});return()=>c.abort();},[]);
 async function choose(lesson:LearningLesson){const n=++selection.current;setSelected(lesson);setVersion(null);setIDs([]);setError(null);try{const versions=await catalogApi.versions(lesson.id);if(!versions.items.length)throw new Error("Опубликованная версия не найдена.");const published=await catalogApi.version(lesson.id,versions.items[0].number);if(selection.current===n)setVersion(published.version);}catch(e){if(selection.current===n)setError(e);}}
 async function act(send:()=>Promise<void>){if(gate.current)return;gate.current=true;setBusy(true);setError(null);try{await send();}catch(e){setError(e);}finally{gate.current=false;setBusy(false);}}
 function submit(event:FormEvent){event.preventDefault();void act(async()=>{if(!student||source==="library"&&(!selected||!version||subset&&!ids.length)||source==="sets"&&!setID)throw new Error("Выберите ученика и материал.");const body={student_id:student,...(source==="sets"?{set_id:setID}:{lesson_id:selected!.id,...(subset?{exercise_ids:ids}:{})}),due_at:new Date(due).toISOString(),calendar_lesson_id:calendar,mode};const signature=JSON.stringify(body);if(nonce.current?.body!==signature)nonce.current={body:signature,id:crypto.randomUUID()};const result=await materialApi.create({...body,request_id:nonce.current.id});nonce.current=null;setCreated(result.id);await refresh();});}
 const activeSet=sets.find(s=>s.id===setID);
 return <section className="material-assignment-composer full" id="assign-material"><div className="section-head"><h2>Назначить из библиотеки</h2><Link className="inline-link" href="/tasks/library">Библиотека<Icon name="arrow" size={16}/></Link></div>
 <form className="form-stack" onSubmit={submit}><fieldset disabled={busy} className="assignment-fields">
 <label>Ученик назначения<Select aria-label="Ученик назначения" value={student} onChange={e=>{setStudent(e.target.value);setCalendar("");}} required><option value="">Выберите ученика</option>{dashboard.relations.filter(r=>r.status==="active").map(r=><option key={r.id} value={r.student_id}>{dashboard.conversations.find(c=>c.student_id===r.student_id)?.student_name??r.student_id}</option>)}</Select></label>
 <div className="tabs" role="tablist" aria-label="Источник заданий">{([ ["library","Библиотека"],["sets","Мои наборы"] ]as const).map(([value,label])=><button type="button" role="tab" aria-selected={source===value} className={source===value?"active":""} onClick={()=>setSource(value)} key={value}>{label}</button>)}</div>
 {source==="library"?<><label className="search-field"><Icon name="search" size={18}/><input aria-label="Найти материал для назначения" placeholder="Название материала" value={query} onChange={e=>{setQuery(e.target.value);setPage(0);}}/></label>
 {list.key!==key?<p className="empty" role="status">Загружаем материалы…</p>:list.error?<LibraryError error={list.error}/>:!list.items?.length?<p className="empty">Опубликованных материалов не найдено.</p>:<div className="assignment-material-options">{list.items.map(l=><button type="button" aria-pressed={selected?.id===l.id} className={selected?.id===l.id?"assignment-material-option selected":"assignment-material-option"} onClick={()=>void choose(l)} key={l.id}><span><strong>{l.version.title}</strong><small>{classLabel(l.version.grade_min,l.version.grade_max)}</small></span><Icon name={selected?.id===l.id?"check":"plus"} size={18}/></button>)}</div>}
 {(list.total??0)>12?<nav className="material-pagination" aria-label="Страницы материалов назначения"><button type="button" className="secondary" disabled={!page} onClick={()=>setPage(page-1)}>Назад</button><span>{page+1} / {Math.ceil((list.total??0)/12)}</span><button type="button" className="secondary" disabled={(page+1)*12>=(list.total??0)} onClick={()=>setPage(page+1)}>Далее</button></nav>:null}
 {selected&&!version?<p role="status">Загружаем опубликованную версию…</p>:version?<div className="assignment-selection">
 <h3>{version.title}</h3><p className="muted">Версия {version.number} · {countLabel(version.exercises.length)}</p>
 <label className="check-row"><input type="checkbox" checked={subset} onChange={e=>setSubset(e.target.checked)}/>Выбрать отдельные упражнения</label>
 {subset?<fieldset className="assignment-exercise-list"><legend>Упражнения для назначения</legend>{version.exercises.map((e,i)=><label className="check-row" key={e.id}><input type="checkbox" checked={ids.includes(e.id)} onChange={event=>setIDs(event.target.checked?[...ids,e.id]:ids.filter(id=>id!==e.id))}/><span>{i+1}. {e.prompt}</span></label>)}</fieldset>:null}
 <div className="assignment-save-set"><label>Название своего набора<input value={setTitle} maxLength={180} onChange={e=>setSetTitle(e.target.value)}/></label><button type="button" className="secondary" disabled={!setTitle.trim()||subset&&!ids.length} onClick={()=>void act(async()=>{const saved=await materialApi.saveSet(setTitle,selected!.id,subset?ids:[]);setSets(old=>[saved,...old]);setSetTitle("");})}><Icon name="plus" size={16}/>Сохранить набор</button></div>
 </div>:null}</>:<>
 <label>Свой набор<Select aria-label="Свой набор" value={setID} onChange={e=>setSetID(e.target.value)}><option value="">Выберите набор</option>{sets.map(s=><option key={s.id} value={s.id}>{s.title} · {countLabel(s.exercise_ids.length)}</option>)}</Select></label>
 {activeSet?<div className="actions"><span>Закреплена версия материала</span><button type="button" className="icon-button" title="Удалить набор" aria-label="Удалить набор" onClick={()=>{if(window.confirm("Удалить набор? Выданные задания сохранятся."))void act(async()=>{await materialApi.deleteSet(activeSet.id);setSets(old=>old.filter(s=>s.id!==activeSet.id));setSetID("");});}}><Icon name="close" size={18}/></button></div>:!sets.length?<p className="empty">Сохранённых наборов пока нет.</p>:null}</>}
 <div className="material-two-fields"><label>Срок интерактивного задания<input type="datetime-local" value={due} onChange={e=>setDue(e.target.value)} required/></label><label>Связать с занятием<Select aria-label="Связать с занятием" value={calendar} onChange={e=>setCalendar(e.target.value)}><option value="">Без занятия</option>{dashboard.lessons.filter(l=>l.student_id===student&&l.status!=="cancelled").map(l=><option key={l.id} value={l.id}>{l.title}</option>)}</Select></label></div>
 <fieldset className="player-modes"><legend>Режим назначения</legend>{([ ["practice","Тренировка"],["exam","Контрольная"] ]as const).map(([value,label])=><label key={value} className={mode===value?"selected":""}><input type="radio" name="assignment-mode" checked={mode===value} onChange={()=>setMode(value)}/>{label}</label>)}</fieldset>
 </fieldset>{error?<LibraryError error={error}/>:null}<button type="submit" disabled={busy||!student||source==="library"&&!version||source==="sets"&&!setID}><Icon name="plus" size={17}/>{busy?"Сохраняем…":"Назначить материал"}</button>{created?<p role="status" className="notice">Материал назначен. <Link href={`/tasks/${created}`}>Открыть назначение</Link></p>:null}</form></section>;
}
