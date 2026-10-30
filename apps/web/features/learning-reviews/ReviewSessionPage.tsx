"use client";
import Link from "next/link";
import { useEffect,useState } from "react";
import { reviewApi,type ReviewSession } from "../../shared/api/reviews";
import { LearningPlayer } from "../learning-player/LearningPlayer";
import { useWorkspace } from "../app-shell/AppShell";
import { TaskSectionNav } from "../tasks/TaskSectionNav";
import { Icon } from "../ui/Icon";
import { parseReviewSession } from "./model";
import { ReviewError } from "./ReviewUI";
import { RewardsPanel } from "./RewardsPanel";
export function ReviewSessionPage({sessionID}:{sessionID:string}){
 const {dashboard}=useWorkspace(),[retry,setRetry]=useState(0),[data,setData]=useState<{key:string;session?:ReviewSession;error?:unknown}>({key:""}),key=`${sessionID}:${retry}`;
 useEffect(()=>{const c=new AbortController();reviewApi.session(sessionID,c.signal).then(parseReviewSession).then(session=>{if(session.id!==sessionID)throw new Error("Некорректное прохождение.");if(!c.signal.aborted)setData({key,session});}).catch((error:unknown)=>{if(!c.signal.aborted)setData({key,error});});return()=>c.abort();},[sessionID,key]);
 const session=data.session,run=session?.state.attempt,completed=run?.status==="completed";
 return <section className="review-session"><TaskSectionNav/><Link href="/tasks/review" className="inline-link material-back"><Icon name="arrow" className="back" size={17}/>К повторению</Link>{data.key!==key?<p role="status">Загружаем тренировку…</p>:data.error?<ReviewError error={data.error} retry={()=>setRetry(n=>n+1)}/>:session?<><header className="material-page-heading"><div><p className="material-kicker">Короткая тренировка</p><h1>{session.state.version.title}</h1></div><button type="button" className="icon-button" title="Обновить" aria-label="Обновить тренировку" onClick={()=>setRetry(n=>n+1)}><Icon name="arrow" size={18}/></button></header>{completed?<section className="review-complete" role="status"><Icon name="check" size={28}/><div><h2>Тренировка завершена</h2><p>Верно: {run.correct_exercises} из {run.total_exercises}{run.pending_review?` · На проверке: ${run.pending_review}`:""}</p></div></section>:null}<LearningPlayer key={key} lessonID={session.state.attempt.lesson_id} version={session.state.version} initial={session.state} allowRestart={false} onCompleted={()=>setRetry(n=>n+1)}/>{completed?<RewardsPanel studentID={dashboard.user.id} revision={retry}/>:null}</>:null}</section>;
}
