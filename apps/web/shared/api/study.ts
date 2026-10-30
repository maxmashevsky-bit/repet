import { request } from "./client";
import type { CatalogPage } from "./catalog";
import type { LearningAttempt,LearningState } from "./learning";
import type { ExerciseAttempt } from "./exercises";
export type ExamSlot={number:number;count:number;points:number};
export type ExamSpecification={id:string;family_id:string;version:number;author_id:string;author_name:string;collection_id:string;title:string;direction:"oge"|"ege_basic"|"ege_profile";year:number;duration_seconds:number;instructions:string;slots:ExamSlot[];topics:string[];exercise_count:number;status:"draft"|"published"|"archived"};
export type SpecificationInput={request_id:string;parent_id:string;collection_id:string;title:string;direction:ExamSpecification["direction"];year:number;duration_seconds:number;instructions:string;slots:ExamSlot[]};
export type StudyInput={request_id:string;specification_id:string;mode:"practice"|"exam";seed:string;topic:string;count:number;repeat_session_id?:string};
export type StudySession={id:string;specification:ExamSpecification;seed:string;deadline_at:string|null;server_time:string;points:number;max_points:number;state:LearningState};
export type StudyHistoryEntry={id:string;title:string;attempt:LearningAttempt;deadline_at:string|null;seed:string};
export type StudyReview={session_id:string;lesson_id:string;attempt:ExerciseAttempt};
const write=<T>(url:string,input:object)=>request<T>(url,{method:"POST",body:JSON.stringify(input)});
export const studyApi={
 specifications:(offset=0,signal?:AbortSignal)=>request<CatalogPage<ExamSpecification>>(`/api/v1/exam-specifications?limit=20&offset=${offset}`,{signal}),
 specification:(id:string,signal?:AbortSignal)=>request<ExamSpecification>(`/api/v1/exam-specifications/${encodeURIComponent(id)}`,{signal}),
 createSpecification:(input:SpecificationInput)=>write<ExamSpecification>("/api/v1/exam-specifications",input),
 status:(id:string,status:"publish"|"archive")=>write<ExamSpecification>(`/api/v1/exam-specifications/${encodeURIComponent(id)}/${status}`,{}),
 start:(input:StudyInput)=>write<StudySession>("/api/v1/study-sessions",input),
 session:(id:string,signal?:AbortSignal)=>request<StudySession>(`/api/v1/study-sessions/${encodeURIComponent(id)}`,{signal}),
 history:(offset=0,signal?:AbortSignal)=>request<CatalogPage<StudyHistoryEntry>>(`/api/v1/study-sessions?limit=20&offset=${offset}`,{signal}),
 reviews:(offset=0,signal?:AbortSignal)=>request<CatalogPage<StudyReview>>(`/api/v1/study-reviews?limit=20&offset=${offset}`,{signal})
};
