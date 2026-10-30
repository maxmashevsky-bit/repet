import { request } from "./client";
import type { Assignment } from "./types";
import type { CatalogPage, LessonVersion } from "./catalog";
import type { LearningState } from "./learning";
import type { ExerciseAttempt } from "./exercises";

export type ExerciseSet = { id:string;title:string;lesson_id:string;version_id:string;exercise_ids:string[];created_at:string };
export type MaterialResults = { assignment:Assignment;version:LessonVersion;state:LearningState|null;history:CatalogPage<ExerciseAttempt>;exercise_stats:{exercise_id:string;attempts:number;mistakes:number}[] };
export type MaterialInput = { request_id:string;student_id:string;lesson_id?:string;set_id?:string;exercise_ids?:string[];due_at:string;calendar_lesson_id?:string;mode:"practice"|"exam" };
const base="/api/v1/assignments";
const write=<T>(path:string,body:object,method="POST")=>request<T>(path,{method,body:JSON.stringify(body)});
export const materialApi={
 create:(input:MaterialInput)=>write<Assignment>(base+"/material",input),
 results:(id:string,offset=0,signal?:AbortSignal)=>request<MaterialResults>(`${base}/${encodeURIComponent(id)}/learning-results?limit=20&offset=${offset}`,{signal}),
 start:(id:string,requestID:string)=>write<unknown>(`${base}/${encodeURIComponent(id)}/learning-attempt`,{request_id:requestID}),
 repeat:(id:string,dueAt:string,requestID:string)=>write<Assignment>(`${base}/${encodeURIComponent(id)}/repeat`,{request_id:requestID,due_at:dueAt}),
 sets:(offset=0,signal?:AbortSignal)=>request<CatalogPage<ExerciseSet>>(`/api/v1/exercise-sets?limit=100&offset=${offset}`,{signal}),
 saveSet:(title:string,lessonID:string,ids:string[])=>write<ExerciseSet>("/api/v1/exercise-sets",{title,lesson_id:lessonID,exercise_ids:ids}),
 deleteSet:(id:string)=>request<void>(`/api/v1/exercise-sets/${encodeURIComponent(id)}`,{method:"DELETE"})
};
