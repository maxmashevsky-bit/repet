"use client";
import { Select } from "../ui/Select";


import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect,useRef,useState } from "react";
import { materialApi,type MaterialResults } from "../../shared/api/material-assignments";
import { exerciseApi,type ExerciseAttempt } from "../../shared/api/exercises";
import { parseLearningState } from "../exercise-engine/learning-schema";
import { parseAttemptsPage } from "../exercise-engine/schema";
import { useWorkspace } from "../app-shell/AppShell";
import { LibraryError,LibraryLoading } from "../material-library/LibraryUI";
import { LearningPlayer } from "../learning-player/LearningPlayer";
import { formatDateTime,dateInTimeZone } from "../../shared/lib/date";
import { Icon } from "../ui/Icon";
import { TaskSectionNav } from "./TaskSectionNav";

const statusLabel=(status:string)=>status==="correct"?"Верно":status==="incorrect"?"Неверно":status==="submitted"?"Ответ сохранён":"Требует проверки";
export function MaterialAssignmentPage({assignmentID}:{assignmentID:string}){
 const {dashboard,refresh}=useWorkspace();const router=useRouter();
 const [retry,setRetry]=useState(0),[page,setPage]=useState(0);
 const key=`${assignmentID}:${retry}:${page}:${dashboard.user.id}`;
 const [load,setLoad]=useState<{key:string;data?:MaterialResults;error?:unknown}>({key:""});
 const [busy,setBusy]=useState(false),[error,setError]=useState<unknown>(null),[due,setDue]=useState("");
 const pending=useRef(new Map<string,string>()),gate=useRef(false);
 useEffect(()=>{const c=new AbortController();materialApi.results(assignmentID,page*20,c.signal).then(data=>{if(data.state){data.state=parseLearningState(data.state);if(data.state.attempt.assignment_id!==assignmentID)throw new Error("Несогласованное назначение.");}data.history=parseAttemptsPage(data.history);if(!c.signal.aborted)setLoad({key,data});}).catch(e=>{if(!c.signal.aborted)setLoad({key,error:e});});return()=>c.abort();},[assignmentID,page,key]);
 async function action(name:string,send:(nonce:string)=>Promise<void>){if(gate.current)return;gate.current=true;setBusy(true);setError(null);const nonce=pending.current.get(name)??crypto.randomUUID();pending.current.set(name,nonce);try{await send(nonce);pending.current.delete(name);}catch(e){setError(e);}finally{gate.current=false;setBusy(false);}}
 const data=load.data;const teacher=dashboard.user.role==="tutor";
 return <section className="material-reader"><TaskSectionNav/><Link className="inline-link material-back" href="/tasks"><Icon name="arrow" className="back" size={16}/>К заданиям</Link>
 {load.key!==key?<LibraryLoading/>:load.error?<LibraryError error={load.error} retry={()=>setRetry(n=>n+1)}/>:data?<><header className="material-page-heading"><div><p className="material-kicker">{data.assignment.material?.repeat_of?"Повторение":"Интерактивное задание"}</p><h1>{data.assignment.title}</h1><p className="muted">До {formatDateTime(data.assignment.due_at,dashboard.user.timezone)} · {data.assignment.material?.mode==="exam"?"Контрольная":"Тренировка"}{teacher?` · ${data.assignment.student_name}`:""}</p></div><button type="button" className="secondary" disabled={busy} onClick={()=>setRetry(n=>n+1)}><Icon name="arrow" size={16}/>Обновить</button></header>
 {data.assignment.material?.calendar_lesson_id?(()=>{const calendar=dashboard.lessons.find(l=>l.id===data.assignment.material?.calendar_lesson_id);return calendar?<Link className="inline-link" href={`/calendar?view=day&date=${dateInTimeZone(calendar.starts_at,dashboard.user.timezone)}`}><Icon name="calendar" size={17}/>{calendar.title}</Link>:null;})():null}
 {error?<LibraryError error={error}/>:null}
 {!teacher?(data.state?<LearningPlayer key={data.state.attempt.id} lessonID={data.assignment.material!.lesson_id} version={data.version} initial={data.state} onCompleted={()=>void refresh()}/>:<section className="player-start"><h2>{data.version.title}</h2><p>{data.version.summary}</p><p className="muted">{data.version.exercises.length} упражнений · Версия {data.version.number}</p><button type="button" disabled={busy} onClick={()=>void action("start",async id=>{parseLearningState(await materialApi.start(assignmentID,id));setRetry(n=>n+1);})}>{busy?"Начинаем…":"Начать выполнение"}<Icon name="arrow" size={17}/></button></section>):<>
 <section className="assignment-results"><h2>Результат ученика</h2>{!data.state?<p className="empty">Ученик ещё не начал выполнение.</p>:<><dl className="learning-counts"><div><dt>Решено</dt><dd>{data.state.attempt.answered_exercises} / {data.state.attempt.total_exercises}</dd></div><div><dt>Верно</dt><dd>{data.state.attempt.correct_exercises}</dd></div><div><dt>Ошибок</dt><dd>{data.state.attempt.incorrect_attempts}</dd></div><div><dt>На проверке</dt><dd>{data.state.attempt.pending_review}</dd></div></dl>{data.state.attempt.mode==="exam"&&data.state.attempt.status!=="completed"?<p className="notice">Контрольная ещё не завершена. Результаты закрыты.</p>:null}<div className="assignment-exercise-results">{data.version.exercises.map((e,i)=>{const a=data.state!.answers.find(a=>a.exercise_id===e.id);const stat=data.exercise_stats.find(s=>s.exercise_id===e.id);return <article key={e.id}><h3>{i+1}. {e.prompt}</h3><p>{a?statusLabel(a.status):"Не отправлено"} · Попыток: {stat?.attempts??0} · Ошибок: {stat?.mistakes??0}</p>{a?<p className="engine-written-solution">{a.answer.value??a.answer.selected?.join(", ")}</p>:null}{a?.status==="needs_review"?<ManualReview attempt={a} lessonID={data.assignment.material!.lesson_id} saved={()=>{setRetry(n=>n+1);void refresh();}}/>:a?.feedback?<p className="notice">{a.feedback}</p>:null}</article>;})}</div><section className="learning-history"><h2>История отправок</h2>{data.history.items.map(a=><article className="learning-history-item" key={a.id}><strong>{a.prompt}</strong><p>{a.answer.value??a.answer.selected?.join(", ")}</p><p>{statusLabel(a.status)} · {formatDateTime(a.created_at,dashboard.user.timezone)}</p></article>)}{data.history.total>20?<nav className="material-pagination" aria-label="Страницы отправок"><button type="button" className="secondary" disabled={!page} onClick={()=>setPage(page-1)}>Назад</button><span>{page+1} / {Math.ceil(data.history.total/20)}</span><button type="button" className="secondary" disabled={(page+1)*20>=data.history.total} onClick={()=>setPage(page+1)}>Далее</button></nav>:null}</section></>}</section>
 <form className="assignment-repeat form-stack" onSubmit={event=>{event.preventDefault();void action(`repeat:${due}`,async id=>{const next=await materialApi.repeat(assignmentID,new Date(due).toISOString(),id);await refresh();router.push(`/tasks/${next.id}`);});}}><h2>Назначить повторение</h2><label>Новый срок сдачи<input type="datetime-local" value={due} onChange={e=>setDue(e.target.value)} required/></label><button type="submit" disabled={busy||!due}><Icon name="arrow" size={17}/>Отправить на повторение</button></form>
 </>}
 </>:null}</section>;
}

function ManualReview({attempt,lessonID,saved}:{attempt:ExerciseAttempt;lessonID:string;saved:()=>void}){
 const [status,setStatus]=useState<"correct"|"incorrect">("correct"),[feedback,setFeedback]=useState(""),[busy,setBusy]=useState(false),[error,setError]=useState<unknown>(null);
 async function submit(event:React.FormEvent){
  event.preventDefault();if(busy)return;setBusy(true);setError(null);
  try{await exerciseApi.review(lessonID,attempt.id,status,feedback);saved();}catch(e){setError(e);}finally{setBusy(false);}
 }
 return <form className="form-stack assignment-manual-review" onSubmit={submit}><label>Результат решения<Select value={status} disabled={busy} onChange={e=>setStatus(e.target.value as "correct"|"incorrect")}><option value="correct">Верно</option><option value="incorrect">Неверно</option></Select></label><label>Комментарий к решению<textarea value={feedback} maxLength={10000} disabled={busy} onChange={e=>setFeedback(e.target.value)}/></label>{error?<LibraryError error={error}/>:null}<button type="submit" disabled={busy}><Icon name="check" size={16}/>Сохранить проверку</button></form>;
}
