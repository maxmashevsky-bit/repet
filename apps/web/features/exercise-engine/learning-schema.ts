import type { LearningState, LearningAttempt } from "../../shared/api/learning";
import { parseExercise, parseAttempt, answerError } from "./schema.ts";
import { parseExample,playerSteps } from "../learning-player/model.ts";

const object = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);
const uuid = (v: unknown): v is string => typeof v === "string" && /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i.test(v);
const date = (v: unknown) => typeof v === "string" && Number.isFinite(Date.parse(v));
function fail(): never { throw new Error("Некорректное состояние прохождения."); }
export function parseLearningAttempt(v: unknown): LearningAttempt {
 if (!object(v) || ![v.id,v.lesson_id,v.version_id].every(uuid) || !["in_progress","completed"].includes(v.status as string) || !date(v.started_at) || (v.status === "completed" ? !date(v.completed_at) : v.completed_at !== null)) return fail();
 const numbers = ["number","elapsed_seconds","total_exercises","answered_exercises","correct_exercises","pending_review","incorrect_attempts"];
 if(numbers.some((key)=>!Number.isSafeInteger(v[key]) || (v[key] as number)<(key==="number"?1:0)) || Object.keys(v).some((key)=>![...numbers,"id","lesson_id","version_id","status","started_at","completed_at","mode","player_step","theory_seen","example_seen","assignment_id"].includes(key)) || (v.answered_exercises as number)>(v.total_exercises as number) || (v.correct_exercises as number)+(v.pending_review as number)>(v.answered_exercises as number))return fail();
 if(v.assignment_id!==undefined&&!uuid(v.assignment_id))return fail();
 if(!["practice","exam"].includes(v.mode as string)||typeof v.player_step!=="string"||typeof v.theory_seen!=="boolean"||typeof v.example_seen!=="boolean")return fail();
 return v as unknown as LearningAttempt;
}
export function parseLearningState(v: unknown): LearningState {
 if(!object(v)||Object.keys(v).some((key)=>!["attempt","version","answers","hints"].includes(key))||!object(v.version)||!Array.isArray(v.version.exercises)||!Array.isArray(v.answers)||!Array.isArray(v.hints))return fail();
 const attempt=parseLearningAttempt(v.attempt);
 if(v.version.id!==attempt.version_id || !Number.isInteger(v.version.number) || (v.version.number as number)<1 || !date(v.version.created_at) || !date(v.version.published_at))return fail();
 const exercises=v.version.exercises.map(parseExercise);if(new Set(exercises.map(e=>e.id)).size!==exercises.length || exercises.length!==attempt.total_exercises)return fail();
 const answers=v.answers.map(parseAttempt);
 if(new Set(answers.map(a=>a.exercise_id)).size!==answers.length || answers.length!==attempt.answered_exercises)return fail();
 if(answers.filter(a=>a.status==="correct").length!==attempt.correct_exercises||answers.filter(a=>a.status==="needs_review").length!==attempt.pending_review)return fail();
 for(const answer of answers){const e=exercises.find(e=>e.id===answer.exercise_id);if(!e || answer.learning_attempt_id!==attempt.id || answer.version_id!==attempt.version_id || answerError(e,answer.answer))return fail();}
 const seen=new Set<string>();
 for(const h of v.hints){if(!object(h)||Object.keys(h).some(k=>!["exercise_id","number","text","revealed_at"].includes(k))||!uuid(h.exercise_id)||!Number.isInteger(h.number)|| (h.number as number)<1 || (h.number as number)>5 ||typeof h.text!=="string"||!h.text.trim()||h.text.length>4000||!date(h.revealed_at))return fail();const e=exercises.find(e=>e.id===h.exercise_id);const key=`${h.exercise_id}:${h.number}`;if(!e||(h.number as number)>(e.hint_count??0)||seen.has(key))return fail();seen.add(key);}
 const state={...v,attempt,version:{...v.version,exercises,example:parseExample(v.version.example)},answers} as unknown as LearningState;
 if(!playerSteps(state.version).some(s=>s.id===attempt.player_step))return fail();
 if(attempt.mode==="exam"&&attempt.status==="in_progress"&&(answers.some(a=>a.status!=="submitted")||attempt.correct_exercises||attempt.pending_review||attempt.incorrect_attempts||v.hints.length))return fail();
 return state;
}

// Delayed responses must not replace newer answers, revealed hints or a completed run.
export function newerLearningState(current:LearningState,next:LearningState):LearningState {
 if(current.attempt.id!==next.attempt.id)return next;
 if(current.attempt.completed_at&&!next.attempt.completed_at)return current;
 if(current.hints.some(h=>!next.hints.some(n=>n.exercise_id===h.exercise_id&&n.number===h.number)))return current;
 for(const previous of current.answers){const answer=next.answers.find(a=>a.exercise_id===previous.exercise_id);if(!answer||answer.attempt_number<previous.attempt_number||answer.id===previous.id&&previous.reviewed_at&&!answer.reviewed_at)return current;}
 return next;
}
