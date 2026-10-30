import type { ReviewItem, ReviewPlan, ReviewSession, RewardPreference, RewardSummary } from "../../shared/api/reviews";
import { parseLearningAttempt, parseLearningState } from "../exercise-engine/learning-schema.ts";
function fail():never{throw new Error("Некорректные данные повторения.");}
function shape(v:unknown,keys:string){if(!v||typeof v!=="object"||Array.isArray(v))return fail();const o=v as Record<string,unknown>,list=keys.split(" ");if(Object.keys(o).some(k=>!list.includes(k))||list.some(k=>!(k in o)))fail();return o;}
function count(v:unknown):number{if(typeof v!=="number"||!Number.isSafeInteger(v)||v<0)return fail();return v;}
function text(v:unknown,max=20000){if(typeof v!=="string"||v.length>max)return fail();return v;}
function id(v:unknown){if(!/^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/i.test(text(v,36)))fail();}
function date(v:unknown){if(!Number.isFinite(Date.parse(text(v,80))))fail();}
function page(v:unknown,validate:(v:unknown)=>void){const o=shape(v,"items total limit offset");count(o.total);count(o.offset);if(o.limit!==20||!Array.isArray(o.items)||o.items.length>20||o.items.length>Number(o.total))fail();(o.items as unknown[]).forEach(validate);}
export function parseRewardPreference(v:unknown):RewardPreference{const o=shape(v,"enabled enabled_from updated_at");if(typeof o.enabled!=="boolean")fail();if(o.enabled_from!==null)date(o.enabled_from);if(o.updated_at!==null)date(o.updated_at);return v as RewardPreference;}
export function parseRewardSummary(v:unknown):RewardSummary{
 const o=shape(v,"enabled xp level level_progress level_target weekly_xp weekly_achievements awards");for(const key of["xp","level","level_progress","weekly_xp","weekly_achievements"])count(o[key]);if(typeof o.enabled!=="boolean"||o.level_target!==100||Number(o.level_progress)>99||Number(o.xp)%10||Number(o.weekly_xp)>Number(o.xp)||o.weekly_xp!==Number(o.weekly_achievements)*10)fail();
 if(o.enabled?(o.level!==1+Math.floor(Number(o.xp)/100)||o.level_progress!==Number(o.xp)%100):(o.xp!==0||o.level!==0||o.level_progress!==0||o.weekly_xp!==0||o.weekly_achievements!==0))fail();
 page(o.awards,v=>{const a=shape(v,"id title points earned_at");id(a.id);text(a.title,200);if(a.points!==10)fail();date(a.earned_at);});if(!o.enabled&&Number((o.awards as {total:number}).total))fail();return v as RewardSummary;
}
export function parseReviewPlan(v:unknown):ReviewPlan{
 const o=shape(v,"server_time due_total mistake_total unanswered pending weekly_completed weekly_target items active");date(o.server_time);for(const key of["due_total","mistake_total","unanswered","pending","weekly_completed"])count(o[key]);if(o.weekly_target!==3)fail();
 page(o.items,v=>{const i=shape(v,"source_run_id lesson_id exercise_id topic_key topic subtopic prompt status clean_days due_at");[i.source_run_id,i.lesson_id,i.exercise_id].forEach(id);if(!/^[a-f0-9]{32}$/.test(text(i.topic_key,32)))fail();[i.topic,i.subtopic,i.prompt].forEach(v=>text(v));if(!["correct","incorrect","unanswered"].includes(text(i.status,20)))fail();count(i.clean_days);date(i.due_at);});
 page(o.active,v=>{const a=shape(v,"id title attempt");id(a.id);text(a.title,200);const run=parseLearningAttempt(a.attempt);if(run.id!==a.id||run.mode!=="practice"||run.status!=="in_progress"||run.assignment_id)fail();});return v as ReviewPlan;
}
export function parseReviewSession(v:unknown):ReviewSession{const o=shape(v,"id state");id(o.id);const state=parseLearningState(o.state);if(state.attempt.id!==o.id||state.attempt.mode!=="practice"||state.attempt.assignment_id||!state.version.exercises.length||state.version.exercises.length>10)fail();return{id:o.id as string,state};}
export function reviewGroups(items:ReviewItem[]):ReviewItem[]{const seen=new Set<string>();return items.filter(i=>{const key=`${i.source_run_id}:${i.topic_key}`;if(seen.has(key))return false;seen.add(key);return true;});}
