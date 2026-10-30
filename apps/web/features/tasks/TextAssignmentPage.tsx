"use client";
import Link from "next/link";
import {useState,type FormEvent} from "react";
import {productApi} from "../../shared/api/client";
import type {Assignment} from "../../shared/api/types";
import {formatDateTime} from "../../shared/lib/date";
import {useWorkspace} from "../app-shell/AppShell";
import {LibraryError} from "../material-library/LibraryUI";
import {TaskSectionNav} from "./TaskSectionNav";

export function TextAssignmentPage({assignment:a,changed}:{assignment:Assignment;changed:()=>void}){
 const{dashboard,refresh}=useWorkspace(),teacher=dashboard.user.role==="tutor";
 const[answer,setAnswer]=useState(a.answer),[score,setScore]=useState(a.score??0),[feedback,setFeedback]=useState(a.feedback),[busy,setBusy]=useState(false),[error,setError]=useState<unknown>(null);
 async function act(review:boolean,revision=false){if(busy)return;setBusy(true);setError(null);try{if(review)await productApi.gradeAssignment(a.id,score,feedback,revision);else await productApi.submitAssignment(a.id,answer);changed();await refresh();}catch(e){setError(e);}finally{setBusy(false);}}
 return <section className="text-assignment-page"><TaskSectionNav/><Link className="inline-link material-back" href="/tasks">Все задания</Link><header className="material-page-heading"><div><p className="material-kicker">{teacher?a.student_name:"Домашнее задание"}</p><h1>{a.title}</h1><p>Сдать до {formatDateTime(a.due_at,dashboard.user.timezone)}</p></div></header><p className="material-prose">{a.body}</p>{error?<LibraryError error={error}/>:null}
 {teacher?<section className="form-stack"><h2>Ответ ученика</h2><p className="material-prose">{a.answer||"Ученик ещё не отправил работу."}</p>{a.status==="submitted"?<form className="form-stack" onSubmit={(e:FormEvent)=>{e.preventDefault();void act(true);}}><label>Баллы<input type="number" required min={0} max={a.max_score} disabled={busy} value={score} onChange={e=>setScore(Number(e.target.value))}/></label><label>Комментарий преподавателя<textarea maxLength={4000} disabled={busy} value={feedback} onChange={e=>setFeedback(e.target.value)}/></label><div className="actions"><button type="submit" disabled={busy}>Принять</button><button type="button" className="secondary" disabled={busy} onClick={()=>void act(true,true)}>На исправление</button></div></form>:null}</section>:<form className="form-stack" onSubmit={e=>{e.preventDefault();void act(false);}}><label>Ответ<textarea rows={6} maxLength={8000} disabled={busy||!["assigned","revision"].includes(a.status)} required value={answer} onChange={e=>setAnswer(e.target.value)}/></label>{["assigned","revision"].includes(a.status)?<button type="submit" disabled={busy||!answer.trim()}>Отправить на проверку</button>:<p className="notice">{a.status==="done"?"Проверено преподавателем":a.status==="archived"?"Работа в архиве":"Работа отправлена преподавателю"}</p>}</form>}
 {a.feedback?<p className="notice">Комментарий преподавателя: {a.feedback}</p>:null}{a.score!==undefined?<p className="notice">Оценка: {a.score} / {a.max_score}</p>:null}</section>;
}
