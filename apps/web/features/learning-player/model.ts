import type { LearningExample,LessonVersion } from "../../shared/api/catalog";
import type { LearningState } from "../../shared/api/learning";

export function parseExample(v:unknown):LearningExample|undefined{
 if(v===undefined||v===null)return undefined;
 const object=(x:unknown):x is Record<string,unknown>=>!!x&&typeof x==="object"&&!Array.isArray(x);
 const text=(x:unknown,n:number)=>typeof x==="string"&&x.trim().length>0&&!x.includes("\0")&&new TextEncoder().encode(x).length<=n;
 if(!object(v)||Object.keys(v).some(k=>!["title","explanation","min","max","steps"].includes(k))||!text(v.title,200)||!text(v.explanation,10000)||!Number.isInteger(v.min)||!Number.isInteger(v.max)||(v.min as number)<-1000||(v.max as number)>1000||(v.max as number)<=(v.min as number)||(v.max as number)-(v.min as number)>100||!Array.isArray(v.steps)||v.steps.length<2||v.steps.length>12)throw new Error("Проверьте пример урока.");
 for(const s of v.steps)if(!object(s)||Object.keys(s).some(k=>!["text","value"].includes(k))||!text(s.text,2000)||!Number.isInteger(s.value)||(s.value as number)<(v.min as number)||(s.value as number)>(v.max as number))throw new Error("Проверьте шаг примера.");
 return v as unknown as LearningExample;
}
export function playerSteps(version:LessonVersion){return [{id:"theory",label:"Объяснение"},...(version.example?[{id:"example",label:"Пример"}]:[]),...version.exercises.map((e,i)=>({id:`exercise:${e.id}`,label:`Задача ${i+1}`})),{id:"results",label:"Результаты"}];}
export function canVisit(state:LearningState,id:string):boolean{
 if(id==="theory")return true;if(id==="results")return state.attempt.status==="completed";
 if(id==="example")return !!state.version.example&&(state.attempt.theory_seen||state.attempt.player_step==="theory");
 let ready=state.attempt.theory_seen||state.attempt.player_step==="theory";
 if(state.version.example)ready=state.attempt.theory_seen&&(state.attempt.example_seen||state.attempt.player_step==="example");
 if(!ready)return false;
 for(const e of state.version.exercises){if(id===`exercise:${e.id}`)return true;if(!state.answers.some(a=>a.exercise_id===e.id))return false;}
 return false;
}
export function playerProgress(state:LearningState){const total=playerSteps(state.version).length;const completed=Number(state.attempt.theory_seen)+Number(!!state.version.example&&state.attempt.example_seen)+state.answers.length+Number(state.attempt.status==="completed");return {completed,total,percent:Math.round(completed/total*100)};}
