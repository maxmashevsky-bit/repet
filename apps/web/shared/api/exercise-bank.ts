import { request } from "./client";
import type { AnswerKey, CatalogPage, Exercise, LessonVersion } from "./catalog";
import type { Assignment } from "./types";

export type BankDirection = "school" | "olympiad" | "oge" | "ege_basic" | "ege_profile";
export type BankMetadata = { grade:number; direction:BankDirection; topic:string; subtopic:string; exam_number:number|null; source:string; attribution:string; rights_status:"own"|"permission"|"open_license"; license:string };
export type BankExercise = BankMetadata & { id:string; author_id:string; author_name:string; subject:"math"; lesson_id:string; version_id:string; exercise:Exercise; created_at:string };
export type BankQuery = { q?:string; direction?:string; grade?:string; exam_number?:string; topic?:string; subtopic?:string; difficulty?:string; kind?:string; limit?:number; offset?:number };
export type BankCollectionSummary = { id:string; title:string; lesson_id:string; version_id:string; exercise_ids:string[]; created_at:string };
export type BankCollection = BankCollectionSummary & { provenance:BankExercise[]; version:LessonVersion };
export type BankAssignResult = { student_id:string; assignment:Assignment|null; error?:string };
const query=(input:BankQuery)=>{const p=new URLSearchParams();Object.entries(input).forEach(([k,v])=>{if(v!==undefined&&v!=="")p.set(k,String(v));});return p.size?`?${p}`:"";};
const write=<T>(path:string,input:object)=>request<T>(path,{method:"POST",body:JSON.stringify(input)});
export const bankApi={
 list:(filter:BankQuery={},signal?:AbortSignal)=>request<CatalogPage<BankExercise>>("/api/v1/exercise-bank"+query(filter),{signal}),
 facets:(filter:BankQuery={},signal?:AbortSignal)=>request<{topics:string[];subtopics:string[]}>("/api/v1/exercise-bank/facets"+query(filter),{signal}),
 create:(input:BankMetadata&{lesson_id:string;exercise_id:string})=>write<BankExercise>("/api/v1/exercise-bank",input),
 key:(id:string)=>request<AnswerKey>(`/api/v1/exercise-bank/${encodeURIComponent(id)}/answer-key`),
 collections:(offset=0,signal?:AbortSignal)=>request<CatalogPage<BankCollectionSummary>>(`/api/v1/bank-collections?limit=20&offset=${offset}`,{signal}),
 collection:(id:string,signal?:AbortSignal)=>request<BankCollection>(`/api/v1/bank-collections/${encodeURIComponent(id)}`,{signal}),
 createCollection:(input:{request_id:string;title:string;course_id:string;topic_id:string;bank_ids:string[]})=>write<BankCollection>("/api/v1/bank-collections",input),
 assign:(id:string,input:{request_id:string;student_ids:string[];due_at:string;mode:"practice"|"exam"})=>write<BankAssignResult[]>(`/api/v1/bank-collections/${encodeURIComponent(id)}/assign`,input)
};
