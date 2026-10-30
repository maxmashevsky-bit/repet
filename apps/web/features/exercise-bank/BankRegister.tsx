"use client";
import { Select } from "../ui/Select";

import Link from "next/link";
import { useRef, useState, type FormEvent } from "react";
import { bankApi, type BankDirection, type BankMetadata } from "../../shared/api/exercise-bank";
import type { LearningLesson } from "../../shared/api/catalog";
import { ApiError } from "../../shared/api/client";
import { directionLabels } from "./model";
import { LibraryError } from "../material-library/LibraryUI";
import { Icon } from "../ui/Icon";

export function BankRegister({lesson,topic}:{lesson:LearningLesson;topic:string}){
 const [busy,setBusy]=useState(false),[error,setError]=useState<unknown>(),[saved,setSaved]=useState(false),[direction,setDirection]=useState<BankDirection>(lesson.version.track==="ege"?"ege_profile":lesson.version.track);
 const guard=useRef(false);const exercises=lesson.version.exercises;
 if(lesson.status!=="published"||lesson.version.number<1||lesson.version.grade_max<5||!exercises.length)return null;
 async function submit(e:FormEvent<HTMLFormElement>){e.preventDefault();if(guard.current)return;guard.current=true;setBusy(true);setError(undefined);setSaved(false);const f=new FormData(e.currentTarget);
  try{await bankApi.create({lesson_id:lesson.id,exercise_id:String(f.get("exercise_id")),grade:Number(f.get("grade")),direction,topic:String(f.get("topic")).trim(),subtopic:String(f.get("subtopic")).trim(),exam_number:direction==="school"||direction==="olympiad"?null:Number(f.get("exam_number")),source:String(f.get("source")).trim(),attribution:String(f.get("attribution")).trim(),rights_status:String(f.get("rights_status")) as BankMetadata["rights_status"],license:String(f.get("license")).trim()});setSaved(true);}catch(e){setError(e instanceof ApiError&&e.status===409?new Error("Эта опубликованная версия упражнения уже есть в банке."):e);}finally{guard.current=false;setBusy(false);}
 }
 return <details className="bank-register"><summary>Добавить упражнение в банк</summary><form onSubmit={submit} className="bank-metadata-form"><fieldset disabled={busy}><legend>Опубликованное упражнение</legend>
 <label>Упражнение<Select name="exercise_id" required>{exercises.map((e,i)=><option key={e.id} value={e.id}>{i+1}. {e.prompt.slice(0,100)}</option>)}</Select></label>
 <label>Класс<Select name="grade" defaultValue={Math.max(5,lesson.version.grade_min)}>{Array.from({length:Math.min(11,lesson.version.grade_max)-Math.max(5,lesson.version.grade_min)+1},(_,i)=>{const g=i+Math.max(5,lesson.version.grade_min);return <option value={g} key={g}>{g}</option>;})}</Select></label>
 <label>Направление<Select value={direction} onChange={e=>setDirection(e.target.value as BankDirection)}>{Object.entries(directionLabels).map(([id,label])=><option value={id} key={id}>{label}</option>)}</Select></label>
 {direction!=="school"&&direction!=="olympiad"?<label>Номер экзаменационного задания<input name="exam_number" type="number" min={1} max={100} required/></label>:null}
 <label>Тема<input name="topic" defaultValue={topic} maxLength={200} required/></label><label>Подтема<input name="subtopic" maxLength={200}/></label>
 <label>Источник<input name="source" maxLength={1000} required/></label><label>Авторство / атрибуция<input name="attribution" defaultValue={lesson.author_name} maxLength={1000} required/></label>
 <label>Права использования<Select name="rights_status"><option value="own">Собственный материал</option><option value="permission">Есть разрешение</option><option value="open_license">Открытая лицензия</option></Select></label>
 <label>Лицензия / условия разрешения<textarea name="license" maxLength={1000} required/></label>
 </fieldset>{error?<LibraryError error={error}/>:null}<button disabled={busy} type="submit"><Icon name="plus" size={17}/>{busy?"Сохраняем…":"Добавить в банк"}</button>{saved?<p role="status">Упражнение добавлено. <Link href="/tasks/library/bank">Открыть банк</Link></p>:null}</form></details>;
}
