"use client";

import Link from "next/link";
import {useEffect,useState} from "react";
import {request,productApi} from "../../shared/api/client";
import type {Assignment} from "../../shared/api/types";
import {htmlApi,type HTMLLesson} from "../../shared/api/html-lessons";
import {HTMLFrame} from "./HTMLFrame";
import {LibraryError,LibraryLoading} from "../material-library/LibraryUI";
import {useWorkspace} from "../app-shell/AppShell";
import {MaterialAssignmentPage} from "../tasks/MaterialAssignmentPage";
import {formatDateTime} from "../../shared/lib/date";
import {TextAssignmentPage} from "../tasks/TextAssignmentPage";

export function AssignmentPage({assignmentID}:{assignmentID:string}){
 const [load,setLoad]=useState<{id:string;assignment?:Assignment;lesson?:HTMLLesson;error?:unknown}>({id:""}),[retry,setRetry]=useState(0);
 useEffect(()=>{const c=new AbortController();async function run(){const assignment=await request<Assignment>(`/api/v1/assignments/${encodeURIComponent(assignmentID)}`,{signal:c.signal});const lesson=assignment.html?await htmlApi.get(assignment.html.lesson_id,assignment.html.version_id,assignmentID,c.signal):undefined;if(!c.signal.aborted)setLoad({id:assignmentID,assignment,lesson});}run().catch(error=>{if(!c.signal.aborted)setLoad({id:assignmentID,error});});return()=>c.abort();},[assignmentID,retry]);
 if(load.id!==assignmentID)return <LibraryLoading/>;
 if(load.error)return <LibraryError error={load.error} retry={()=>setRetry(n=>n+1)}/>;
 return load.assignment?.html&&load.lesson?<HTMLAssigned assignment={load.assignment} lesson={load.lesson} changed={()=>setRetry(n=>n+1)}/>:load.assignment?.kind==="text"?<TextAssignmentPage key={`${load.assignment.id}:${load.assignment.status}:${load.assignment.score??""}`} assignment={load.assignment} changed={()=>setRetry(n=>n+1)}/>:<MaterialAssignmentPage assignmentID={assignmentID}/>;
}
function HTMLAssigned({assignment,lesson,changed}:{assignment:Assignment;lesson:HTMLLesson;changed:()=>void}){
 const {dashboard,refresh}=useWorkspace();const [answer,setAnswer]=useState(assignment.answer),[busy,setBusy]=useState(false),[error,setError]=useState<unknown>(null);
 async function submit(e:React.FormEvent){
  e.preventDefault();if(busy)return;setBusy(true);setError(null);
  try{await productApi.submitAssignment(assignment.id,answer);await refresh();changed();}catch(e){setError(e);}finally{setBusy(false);}
 }
 return <section className="html-reader">
  <Link className="inline-link material-back" href={`/tasks?task=${assignment.id}`}>К заданиям</Link>
  <header className="material-page-heading"><div><p className="material-kicker">Назначенный HTML-урок · Версия {lesson.version.number}</p><h1>{assignment.title}</h1><p>Сдать до {formatDateTime(assignment.due_at,dashboard.user.timezone)}</p></div></header>
  <HTMLFrame lessonID={lesson.id} versionID={lesson.version.id} assignmentID={assignment.id}/>
  {error?<LibraryError error={error}/>:null}
  {dashboard.user.role==="student"?<form className="form-stack html-assign" onSubmit={submit}>
   <h2>Ответ преподавателю</h2><label>Комментарий к выполнению<textarea value={answer} required maxLength={8000} rows={4} disabled={busy||!["assigned","revision"].includes(assignment.status)} onChange={e=>setAnswer(e.target.value)}/></label>
   {["assigned","revision"].includes(assignment.status)?<button type="submit" disabled={busy||!answer.trim()}>Отправить на ручную проверку</button>:<p className="notice">{assignment.status==="done"?"Проверено преподавателем":"Работа отправлена преподавателю"}</p>}
   {assignment.feedback?<p>Комментарий преподавателя: {assignment.feedback}</p>:null}{assignment.score!==undefined?<p>Оценка преподавателя: {assignment.score} / {assignment.max_score}</p>:null}
  </form>:<section className="html-assign"><h2>Ручная проверка</h2><p>{assignment.answer||"Ученик ещё не отправил работу."}</p><Link className="button secondary" href="/tasks?status=submitted">К проверке работ</Link></section>}
 </section>;
}
