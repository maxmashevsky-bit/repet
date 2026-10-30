"use client";

import { useEffect, useRef, useState } from "react";
import { learningApi, type LearningState, type LearningAttempt } from "../../shared/api/learning";
import { exerciseApi, type ExerciseAttempt } from "../../shared/api/exercises";
import { LibraryError } from "../material-library/LibraryUI";
import { ExerciseRenderer, type SubmitExercise } from "./ExerciseRenderer";
import { parseAttempt, parseAttemptsPage, answerError } from "./schema";
import { parseLearningState, parseLearningAttempt, newerLearningState } from "./learning-schema";
import { Icon } from "../ui/Icon";

export function StudentLearning({ initial, lessonID,compactResults=false }: { initial: LearningState; lessonID: string;compactResults?:boolean }) {
 const [state,setState]=useState(initial);
 const [error,setError]=useState<unknown>(null);
 const [busy,setBusy]=useState(false);
 const [historyPage,setHistoryPage]=useState(0);
 const [history,setHistory]=useState<{items:ExerciseAttempt[];total:number}>({items:[],total:0});
 const [results,setResults]=useState<{items:LearningAttempt[];total:number}>({items:[],total:0});
 const [resultsPage,setResultsPage]=useState(0);
 const [historyError,setHistoryError]=useState<unknown>(null);
 const [refresh,setRefresh]=useState(0);
 const [pastID,setPastID]=useState<string|null>(null);
 const pending=useRef(new Map<string,string>());
 const run=state.attempt;const completed=run.status==="completed";
 function accept(value:unknown){const next=parseLearningState(value);setState(current=>newerLearningState(current,next));}
 useEffect(()=>{
  const controller=new AbortController();
  Promise.all([learningApi.answers(lessonID,run.id,historyPage*20,controller.signal),run.assignment_id?Promise.resolve({items:[run],total:1}):learningApi.results(lessonID,resultsPage*20,controller.signal)]).then(([answers,runs])=>{
   if(controller.signal.aborted)return;setHistoryError(null);setHistory(parseAttemptsPage(answers));
   if(!Array.isArray(runs.items)||!Number.isSafeInteger(runs.total)||runs.total<0)throw new Error("Некорректные результаты.");
   setResults({items:runs.items.map(parseLearningAttempt),total:runs.total});
  }).catch((e:unknown)=>{if(!controller.signal.aborted)setHistoryError(e);});
  return()=>controller.abort();
 },[lessonID,run,historyPage,resultsPage,refresh]);
 const submit:SubmitExercise=async(exercise,answer,requestID)=>{
  const result=parseAttempt(await exerciseApi.submit(lessonID,state.version.id,exercise.id,answer,requestID,run.id));
  if(result.exercise_id!==exercise.id||result.version_id!==state.version.id||result.learning_attempt_id!==run.id||answerError(exercise,result.answer))throw new Error("Unexpected exercise response");
  accept(await learningApi.state(lessonID,run.id));setRefresh(n=>n+1);return result;
 };
 async function action(key:string,send:(requestID:string)=>Promise<unknown>,restart=false){
  if(busy)return;setBusy(true);setError(null);
  const requestID=pending.current.get(key)??crypto.randomUUID();pending.current.set(key,requestID);
  try{accept(await send(requestID));pending.current.delete(key);setRefresh(n=>n+1);if(restart)window.location.reload();}
  catch(e){setError(e);}finally{setBusy(false);}
 }
 async function reload(){setBusy(true);setError(null);try{accept(await learningApi.state(lessonID,run.id));setRefresh(n=>n+1);}catch(e){setError(e);}finally{setBusy(false);}}
 return <section className="material-exercises">
  <header className="learning-progress" data-learning-id={run.id}>
   <div><h2>{completed?"Урок завершён":"Прохождение урока"}</h2><p className="muted">Прохождение {run.number} · Версия {state.version.number}</p></div>
   <dl className="learning-counts"><div><dt>Отправлено</dt><dd>{run.answered_exercises} / {run.total_exercises}</dd></div><div><dt>Верно</dt><dd>{run.correct_exercises}</dd></div><div><dt>На проверке</dt><dd>{run.pending_review}</dd></div><div><dt>Время с начала</dt><dd>{Math.floor(run.elapsed_seconds/60)} мин {run.elapsed_seconds%60} с</dd></div></dl>
   <div className="engine-actions"><button type="button" className="secondary" disabled={busy} onClick={()=>void reload()}><Icon name="arrow" size={16}/>Обновить результаты</button>{completed&&!compactResults?<button type="button" disabled={busy} onClick={()=>void action("restart",id=>learningApi.start(lessonID,id,true),true)}>Пройти заново<Icon name="arrow" size={16}/></button>:!completed?<button type="button" disabled={busy||run.answered_exercises!==run.total_exercises} onClick={()=>void action("complete",id=>learningApi.complete(lessonID,run.id,id))}>Завершить урок<Icon name="check" size={16}/></button>:null}</div>
   {error?<LibraryError error={error}/>:null}
  </header>
  {!compactResults?<><h2>Упражнения <span>{state.version.exercises.length}</span></h2>
  {state.version.exercises.map((exercise,index)=>{
   const answer=state.answers.find(a=>a.exercise_id===exercise.id);const hints=state.hints.filter(h=>h.exercise_id===exercise.id);
   return <div key={exercise.id} className="learning-exercise"><ExerciseRenderer exercise={exercise} index={index} initialAttempt={answer} submit={completed?undefined:submit}/>
    {exercise.hint_count?<section className="learning-hints" aria-label={`Подсказки к упражнению ${index+1}`}>{hints.map(h=><p key={h.number}><strong>Подсказка {h.number}.</strong> {h.text}</p>)}<button type="button" className="secondary" disabled={busy||completed||hints.length>=exercise.hint_count} onClick={()=>void action(`hint:${exercise.id}`,id=>learningApi.hint(lessonID,run.id,exercise.id,id))}><Icon name="book" size={16}/>Подсказка {hints.length} / {exercise.hint_count}</button></section>:null}
   </div>;
  })}
  </>:null}
  <section className="learning-history"><h2>История ответов</h2>{historyError?<LibraryError error={historyError} retry={()=>setRefresh(n=>n+1)}/>:null}
   {!history.items.length?<p className="muted">Ответов пока нет.</p>:history.items.map(a=><article key={a.id} className="learning-history-item"><h3>{a.prompt}</h3><p className="muted">Попытка {a.attempt_number} · Версия {a.version} · Подсказок: {a.hints_used} · {a.elapsed_seconds} с</p><p className="engine-written-solution">{a.answer.value??a.answer.selected?.join(", ")}</p><strong>{a.status==="correct"?"Верно":a.status==="incorrect"?"Неверно":"Требует проверки преподавателем"}</strong>{a.feedback?<p>{a.feedback}</p>:null}</article>)}
   <PageButtons page={historyPage} total={history.total} setPage={setHistoryPage}/>
  </section>
  <section className="learning-history"><h2>Мои прохождения</h2>{results.items.map(r=><article key={r.id} className="learning-history-item"><h3>Прохождение {r.number}</h3><p>{r.status==="completed"?"Завершено":"В процессе"} · Верно: {r.correct_exercises} / {r.total_exercises} · На проверке: {r.pending_review} · Ошибочных попыток: {r.incorrect_attempts}</p>{r.id!==run.id?<button type="button" className="secondary" onClick={()=>setPastID(r.id)}>История прохождения {r.number}<Icon name="book" size={16}/></button>:null}</article>)}<PageButtons page={resultsPage} total={results.total} setPage={setResultsPage}/>{pastID?<PastAnswers key={pastID} lessonID={lessonID} runID={pastID}/>:null}</section>
 </section>;
}
function PageButtons({page,total,setPage}:{page:number;total:number;setPage:(value:number)=>void}){return total>20?<nav className="material-pagination" aria-label="Страницы результатов"><button type="button" className="secondary" disabled={!page} onClick={()=>setPage(page-1)}>Назад</button><span>{page+1} / {Math.ceil(total/20)}</span><button type="button" className="secondary" disabled={(page+1)*20>=total} onClick={()=>setPage(page+1)}>Далее</button></nav>:null;}

function PastAnswers({lessonID,runID}:{lessonID:string;runID:string}){
 const [page,setPage]=useState(0),[retry,setRetry]=useState(0);
 const key=`${runID}:${page}:${retry}`;
 const [state,setState]=useState<{key:string;items?:ExerciseAttempt[];total?:number;error?:unknown}>({key:""});
 useEffect(()=>{const controller=new AbortController();learningApi.answers(lessonID,runID,page*20,controller.signal).then(data=>{if(!controller.signal.aborted)setState({key,...parseAttemptsPage(data)});}).catch((error:unknown)=>{if(!controller.signal.aborted)setState({key,error});});return()=>controller.abort();},[lessonID,runID,page,key]);
 return <section className="learning-past" aria-label="Ответы прошлого прохождения"><h3>Ответы прошлого прохождения</h3>{state.key!==key?<p role="status">Загружаем ответы…</p>:state.error?<LibraryError error={state.error} retry={()=>setRetry(n=>n+1)}/>:<>{state.items?.map(a=><article key={a.id} className="learning-history-item"><h4>{a.prompt}</h4><p>Попытка {a.attempt_number} · Версия {a.version}</p><p className="engine-written-solution">{a.answer.value??a.answer.selected?.join(", ")}</p><strong>{a.status==="correct"?"Верно":a.status==="incorrect"?"Неверно":"Требует проверки преподавателем"}</strong>{a.feedback?<p>{a.feedback}</p>:null}</article>)}<PageButtons page={page} total={state.total??0} setPage={setPage}/></>}</section>;
}
