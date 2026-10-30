"use client";

import { useEffect,useRef,useState } from "react";
import type { LessonVersion } from "../../shared/api/catalog";
import { learningApi,type LearningState } from "../../shared/api/learning";
import { exerciseApi } from "../../shared/api/exercises";
import { ExerciseRenderer,type SubmitExercise } from "../exercise-engine/ExerciseRenderer";
import { StudentLearning } from "../exercise-engine/StudentLearning";
import { parseAttempt,answerError } from "../exercise-engine/schema";
import { parseLearningState,newerLearningState } from "../exercise-engine/learning-schema";
import { Icon } from "../ui/Icon";
import { LibraryError } from "../material-library/LibraryUI";
import { AnimatedExample } from "./AnimatedExample";
import { playerSteps,canVisit,playerProgress } from "./model";

export function LearningPlayer({lessonID,version,initial,onCompleted,allowRestart=true}:{lessonID:string;version:LessonVersion;initial?:LearningState;onCompleted?:()=>void;allowRestart?:boolean}){
 const [state,setState]=useState(initial);
 const [mode,setMode]=useState<"practice"|"exam">(initial?.attempt.mode??"practice");
 const [busy,setBusy]=useState(false),[error,setError]=useState<unknown>(null);
 const [saving,setSaving]=useState<"saved"|"saving"|"failed"|"unsent">("saved");const [dirty,setDirty]=useState(false);
 const [solution,setSolution]=useState<{exerciseID:string;text:string;available:boolean}|null>(null);
 const gate=useRef(false),pending=useRef(new Map<string,string>()),heading=useRef<HTMLHeadingElement>(null);
 const run=state?.attempt;const lesson=state?.version??version;
 const steps=playerSteps(lesson),step=run?.player_step??"theory";
 const current=steps.findIndex(s=>s.id===step);
 const exercise=lesson.exercises.find(e=>step===`exercise:${e.id}`);
 const index=exercise?lesson.exercises.findIndex(e=>e.id===exercise.id):-1;
 const answer=state?.answers.find(a=>a.exercise_id===exercise?.id);
 const completed=run?.status==="completed";
 const exam=run?.mode==="exam";
 const hints=state?.hints.filter(h=>h.exercise_id===exercise?.id)??[];
 useEffect(()=>{if(!dirty)return;const warn=(e:BeforeUnloadEvent)=>{e.preventDefault();e.returnValue="";};window.addEventListener("beforeunload",warn);return()=>window.removeEventListener("beforeunload",warn);},[dirty]);
 function accept(value:unknown){const next=parseLearningState(value);if(next.attempt.lesson_id!==lessonID)throw new Error("Unexpected lesson response");setState(old=>old?newerLearningState(old,next):next);return next;}
 async function action(key:string,send:(id:string)=>Promise<unknown>,focus=true){
  if(gate.current)return;gate.current=true;setBusy(true);setError(null);setSaving("saving");
  const id=pending.current.get(key)??crypto.randomUUID();pending.current.set(key,id);
  try{accept(await send(id));pending.current.delete(key);const keepDraft=key.startsWith("hint:")&&dirty;setSaving(keepDraft?"unsent":"saved");if(!keepDraft)setDirty(false);setSolution(null);if(key==="complete")onCompleted?.();if(key.startsWith("restart:")){window.location.reload();return;}if(focus)requestAnimationFrame(()=>heading.current?.focus());}
  catch(e){setError(e);setSaving("failed");}finally{gate.current=false;setBusy(false);}
 }
 const navigate=(target:string)=>{if(!run||dirty&&!window.confirm("Ответ ещё не отправлен. Перейти без сохранения ответа?"))return;void action(`navigate:${target}`,id=>learningApi.navigate(lessonID,run.id,target,id));};
 const submit:SubmitExercise=async(e,value,requestID)=>{
  if(!run||gate.current)throw new Error("Saving in progress");gate.current=true;setBusy(true);setSaving("saving");setError(null);
  try{const result=parseAttempt(await exerciseApi.submit(lessonID,lesson.id,e.id,value,requestID,run.id));if(result.exercise_id!==e.id||result.learning_attempt_id!==run.id||result.version_id!==lesson.id||answerError(e,result.answer))throw new Error("Unexpected exercise response");accept(await learningApi.state(lessonID,run.id));setSaving("saved");setSolution(null);return result;}
  catch(e){setSaving("failed");throw e;}finally{gate.current=false;setBusy(false);}
 };
 async function reveal(){
  if(!run||!exercise||gate.current)return;gate.current=true;setBusy(true);setError(null);
  try{const value=await learningApi.solution(lessonID,run.id,exercise.id);if(!value||typeof value!=="object"||!("exercise_id"in value)||value.exercise_id!==exercise.id||!("text"in value)||typeof value.text!=="string"||!("available"in value)||typeof value.available!=="boolean")throw new Error("Некорректный разбор.");setSolution({exerciseID:exercise.id,text:value.text,available:value.available});}
  catch(e){setError(e);}finally{gate.current=false;setBusy(false);}
 }
 function continueLesson(){if(!run)return;if(completed){navigate("results");return;}const next=steps[current+1];if(!next||next.id==="results")void action("complete",id=>learningApi.complete(lessonID,run.id,id));else navigate(next.id);}
 const modeControl=<fieldset className="player-modes" disabled={busy}><legend>Режим урока</legend>{([ ["practice","Тренировка"],["exam","Контрольная"] ]as const).map(([value,label])=><label key={value} className={mode===value?"selected":""}><input type="radio" name={`learning-mode-${lessonID}`} checked={mode===value} onChange={()=>setMode(value)}/>{label}</label>)}</fieldset>;
 if(!state)return <section className="player-start"><p className="material-kicker">{lesson.exercises.length} {lesson.exercises.length===1?"задача":"задач"}</p><h2>Готовы начать?</h2>{lesson.summary?<p>{lesson.summary}</p>:null}{modeControl}<p className="player-policy">{mode==="exam"?"Одна отправка на задачу. Разборы и результаты после сдачи.":"Можно пробовать снова, брать подсказки и изучать разборы."}</p>{error?<LibraryError error={error}/>:null}<button type="button" disabled={busy} onClick={()=>void action(`start:${mode}`,id=>learningApi.start(lessonID,id,false,mode))}>{busy?"Начинаем…":"Начать урок"}<Icon name="arrow" size={18}/></button></section>;
 const progress=playerProgress(state);
 return <section className="learning-player" aria-label="Прохождение урока" data-learning-id={run!.id}>
  <header className="player-progress"><div><span className="player-mode">{exam?"Контрольная":"Тренировка"}</span><span className={`player-save player-save-${saving}`} role="status" aria-live="polite">{saving==="saving"?"Сохраняем…":saving==="failed"?"Не сохранено":saving==="unsent"?"Ответ ещё не отправлен": "Прогресс сохранён"}</span></div><progress max={progress.total} value={progress.completed} aria-label="Прогресс урока"/><p>{progress.percent}% · {completed?"Завершено":`Шаг ${current+1} из ${steps.length}`}</p></header>
  <div className="player-layout"><nav className="player-navigation" aria-label="Шаги урока">{steps.map((s,i)=>{const allowed=canVisit(state,s.id);return <button type="button" className={s.id===step?"active":""} key={s.id} disabled={busy||!allowed} aria-current={s.id===step?"step":undefined} onClick={()=>navigate(s.id)} title={allowed?s.label:"Сначала завершите предыдущий шаг"}><span className="player-step-number">{allowed?String(i+1).padStart(2,"0"):<Icon name="lock" size={15}/>}</span><span>{s.label}</span>{s.id.startsWith("exercise:")&&state.answers.some(a=>`exercise:${a.exercise_id}`===s.id)?<Icon name="check" size={16}/>:null}</button>;})}</nav>
   <div className="player-body"><h2 ref={heading} tabIndex={-1} className="player-step-heading">{steps[current]?.label??"Урок"}</h2>
    {error?<LibraryError error={error}/>:null}
    {step==="theory"?<section className="player-theory"><p className="material-kicker">Главная идея</p>{lesson.summary?<p className="material-summary">{lesson.summary}</p>:null}<div className="material-prose">{lesson.content||"Объяснение не добавлено."}</div></section>:null}
    {step==="example"&&lesson.example?<AnimatedExample key={lesson.id} example={lesson.example}/>:null}
    {exercise?<><ExerciseRenderer key={exercise.id} exercise={exercise} index={index} initialAttempt={answer} submit={completed||exam&&!!answer?undefined:submit} disabled={busy} submitLabel={exam?"Сохранить ответ":undefined} onDirtyChange={value=>{setDirty(value);if(value)setSaving("unsent");}}/>
     {exam&&!completed?<p className="player-policy">Результат появится после завершения контрольной.</p>:exercise.hint_count?<section className="learning-hints" aria-label="Подсказки">{hints.map(h=><p key={h.number}><strong>Подсказка {h.number}.</strong> {h.text}</p>)}<button type="button" className="secondary" disabled={busy||completed||hints.length>=exercise.hint_count} onClick={()=>void action(`hint:${exercise.id}`,id=>learningApi.hint(lessonID,run!.id,exercise.id,id),false)}><Icon name="book" size={16}/>Подсказка {hints.length} / {exercise.hint_count}</button></section>:null}
     <section className="player-solution"><button type="button" className="secondary" disabled={busy||!answer||exam&&!completed||answer.status==="needs_review"} onClick={()=>void reveal()}><Icon name="eye" size={17}/>Разбор решения</button>{answer?.status==="needs_review"?<p className="muted">Разбор после проверки преподавателем.</p>:null}{solution?.exerciseID===exercise.id?<div role="region" aria-label="Разбор решения"><h3>Разбор решения</h3><p className="material-prose">{solution.available?solution.text:"Разбор пока не добавлен."}</p></div>:null}</section>
    </>:null}
    {step==="results"?<><StudentLearning key={run!.id} initial={state} lessonID={lessonID} compactResults/>{allowRestart&&!run!.assignment_id?<>{modeControl}<button type="button" disabled={busy} onClick={()=>void action(`restart:${mode}`,id=>learningApi.start(lessonID,id,true,mode))}>Пройти ещё раз<Icon name="arrow" size={17}/></button></>:null}</>:<footer className="player-footer">{current>0?<button type="button" className="secondary" disabled={busy} onClick={()=>navigate(steps[current-1].id)}><Icon name="arrow" className="back" size={17}/>Назад</button>:<span/>}<button type="button" disabled={busy||dirty||!!exercise&&!answer} onClick={continueLesson}>{busy?"Сохраняем…":completed?"К результатам":current===steps.length-2?"Завершить урок":"Продолжить"}<Icon name="arrow" size={17}/></button></footer>}
   </div>
  </div>
 </section>;
}
