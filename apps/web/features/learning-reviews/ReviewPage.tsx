"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect,useRef,useState } from "react";
import { reviewApi,type ReviewItem,type ReviewMode,type ReviewPlan } from "../../shared/api/reviews";
import { useWorkspace } from "../app-shell/AppShell";
import { TaskSectionNav } from "../tasks/TaskSectionNav";
import { Icon } from "../ui/Icon";
import { StudentPicker } from "../learning-analytics/AnalyticsPage";
import { parseReviewPlan,parseReviewSession,reviewGroups } from "./model";
import { ReviewError,ReviewPager } from "./ReviewUI";
import { RewardsPanel } from "./RewardsPanel";
import { MasteryOverview } from "./MasteryOverview";

export function ReviewPage(){const{dashboard}=useWorkspace(),[studentID,setStudentID]=useState("");const tutor=dashboard.user.role==="tutor";return <section className="review-page"><TaskSectionNav/><header className="material-page-heading"><div><p className="material-kicker">Регулярная практика</p><h1>{tutor?"Повторение учеников":"Повторение и мои достижения"}</h1></div>{tutor?<Link className="button secondary" href="/tasks/review/check"><Icon name="task" size={17}/>Решения на проверку</Link>:null}</header>{tutor?<StudentPicker value={studentID} change={setStudentID}/>:null}{tutor?studentID?<ReviewWorkspace key={studentID} studentID={studentID} tutor/>:null:dashboard.user.role==="student"?<ReviewWorkspace studentID={dashboard.user.id}/>:<p>Раздел недоступен для этой роли.</p>}</section>;}
function ReviewWorkspace({studentID,tutor=false}:{studentID:string;tutor?:boolean}){
 const router=useRouter();const[mode,setMode]=useState<ReviewMode>("due"),[offset,setOffset]=useState(0),[activeOffset,setActiveOffset]=useState(0),[revision,setRevision]=useState(0),[count,setCount]=useState(3),[busy,setBusy]=useState(false),[error,setError]=useState<unknown>();
 const[data,setData]=useState<{key:string;plan?:ReviewPlan;error?:unknown}>({key:""}),key=`${studentID}:${mode}:${offset}:${revision}:${activeOffset}`;const gate=useRef(false),pending=useRef<{body:string;nonce:string}|undefined>(undefined);
 useEffect(()=>{const c=new AbortController();reviewApi.plan(studentID,mode,offset,c.signal,activeOffset).then(parseReviewPlan).then(plan=>{if(!c.signal.aborted)setData({key,plan});}).catch((error:unknown)=>{if(!c.signal.aborted)setData({key,error});});return()=>c.abort();},[studentID,mode,offset,key,activeOffset]);
 async function start(item:ReviewItem){if(gate.current)return;gate.current=true;setBusy(true);setError(undefined);const body={source_run_id:item.source_run_id,topic_key:item.topic_key,mode:mode==="all"?"quiz"as const:mode,count},text=JSON.stringify(body);if(pending.current?.body!==text)pending.current={body:text,nonce:crypto.randomUUID()};try{const session=parseReviewSession(await reviewApi.start({...body,request_id:pending.current.nonce}));router.push(`/tasks/review/${session.id}`);}catch(e){setError(e);}finally{gate.current=false;setBusy(false);}}
 const plan=data.plan;
 const activePager=plan?<ReviewPager page={plan.active} label="Страницы незавершённых тренировок" change={setActiveOffset}/>:null;
 return <>
  <div className="review-toolbar">
   <div className="review-view-switch" role="group" aria-label="Вид повторения">{([["due","Сегодня"],["errors","Работа над ошибками"],["all","Все темы"]]as const).map(([value,label])=><button type="button" key={value} className={mode===value?"active":"secondary"} aria-pressed={mode===value} onClick={()=>{setMode(value);setOffset(0);}}>{label}</button>)}</div>
   <button className="icon-button" type="button" aria-label="Обновить повторение" title="Обновить" onClick={()=>setRevision(n=>n+1)}><Icon name="arrow" size={18}/></button>
  </div>
  {data.key!==key?<p role="status">Загружаем план повторения…</p>:data.error?<ReviewError error={data.error} retry={()=>setRevision(n=>n+1)}/>:plan?<>
   <dl className="review-counters"><div><dt>Повторить сегодня</dt><dd>{plan.due_total}</dd></div><div><dt>Ошибки</dt><dd>{plan.mistake_total}</dd></div><div><dt>Без ответа</dt><dd>{plan.unanswered}</dd></div><div><dt>На проверке</dt><dd>{plan.pending}</dd></div></dl>
   <section className="review-week"><div><h2>Практика за неделю</h2><p><strong>{plan.weekly_completed}</strong> / {plan.weekly_target} тренировок</p></div><progress value={Math.min(plan.weekly_completed,plan.weekly_target)} max={plan.weekly_target} aria-label="Цель на неделю"/></section>
   {plan.active.items.length?<section className="review-active"><h2>Продолжить тренировку</h2>{plan.active.items.map(item=>tutor?<p key={item.id}>{item.title} · {item.attempt.answered_exercises} из {item.attempt.total_exercises}</p>:<Link key={item.id} href={`/tasks/review/${item.id}`}><span>{item.title}<small>{item.attempt.answered_exercises} из {item.attempt.total_exercises}</small></span><Icon name="arrow" size={18}/></Link>)}</section>:null}
   {activePager}
   {!tutor?<label className="review-count">Максимум заданий<input type="number" min={1} max={10} value={count} onChange={e=>{const n=Number(e.target.value);setCount(Math.max(1,Math.min(10,Number.isFinite(n)?n:1)));}}/></label>:null}
   {error?<ReviewError error={error} retry={()=>{setError(undefined);setRevision(n=>n+1);}}/>:null}
   {!plan.items.items.length?<div className="review-empty"><Icon name="check" size={28}/><h2>{mode==="due"?"На сегодня всё повторено":mode==="errors"?"Нет ошибок для повторения":"Пока нет завершённых работ"}</h2><Link href="/tasks/library" className="inline-link">Открыть библиотеку<Icon name="arrow" size={17}/></Link></div>:<div className="review-card-grid">{reviewGroups(plan.items.items).map(item=><article className="review-topic-card" key={`${item.source_run_id}:${item.topic_key}`}>
    <div className="review-card-status"><Icon name={item.status==="correct"?"calendar":"book"} size={18}/><span>{item.status==="incorrect"?"Нужно разобрать ошибку":item.status==="unanswered"?"Осталось без ответа":new Date(item.due_at)<=new Date(plan.server_time)?"Пора повторить":`Повторение ${new Date(item.due_at).toLocaleDateString("ru-RU")}`}</span></div>
    <h2>{item.topic}</h2>{item.subtopic?<p className="muted">{item.subtopic}</p>:null}
    <details className="review-card-conditions"><summary>Условия задач</summary><ol>{plan.items.items.filter(e=>e.source_run_id===item.source_run_id&&e.topic_key===item.topic_key).map(e=><li key={e.exercise_id}><p>{e.prompt}</p><small>Дней самостоятельных решений: {e.clean_days}</small></li>)}</ol></details>
    {!tutor?<div className="review-card-footer"><button type="button" disabled={busy||plan.active.items.some(a=>a.attempt.lesson_id===item.lesson_id)} onClick={()=>void start(item)}>{busy?"Создаём…":mode==="all"?"Короткий квиз":"Повторить"}<Icon name="arrow" size={17}/></button></div>:null}
   </article>)}</div>}
   <ReviewPager page={plan.items} change={setOffset}/>
  </>:null}
  <MasteryOverview studentID={studentID} revision={revision}/>
  <RewardsPanel studentID={studentID} tutor={tutor} revision={revision}/>
 </>;
}
