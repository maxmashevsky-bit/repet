import { request } from "./client";
import type { CatalogPage } from "./catalog";
import type { AnalyticsStudent,AnalyticsMetric,AnalyticsTopic,AnalyticsDay,AnalyticsExam } from "./analytics";
export type GuardianLink={id:string;student_id:string;student_name:string;guardian_email:string;guardian_id:string|null;status:"pending"|"active"|"revoked";created_at:string;expires_at:string;accepted_at:string|null;revoked_at:string|null};
export type GuardianNote={id:string;tutor_name:string;body:string;created_at:string};
export type FamilyLesson={id:string;title:string;tutor_name:string;starts_at:string;ends_at:string;status:"scheduled"|"cancelled"|"completed"};
export type FamilyAssignment={id:string;title:string;tutor_name:string;subject:string;topic:string;status:string;due_at:string;score:number|null;max_score:number};
export type FamilyEvent={id:string;kind:"lesson"|"assignment"|"submitted"|"graded"|"completed"|"reviewed"|"note";created_at:string;unread:boolean};
export type FamilyResult={id:string;lesson_title:string;prompt:string;status:"correct"|"incorrect"|"needs_review";attempts:number;created_at:string};
export type FamilyReport={student:AnalyticsStudent;from:string;to:string;generated_at:string;summary:AnalyticsMetric;topics:CatalogPage<AnalyticsTopic>;timeline:AnalyticsDay[];exams:CatalogPage<AnalyticsExam>;lessons:CatalogPage<FamilyLesson>;assignments:CatalogPage<FamilyAssignment>;notes:CatalogPage<GuardianNote>;events:CatalogPage<FamilyEvent>;results:CatalogPage<FamilyResult>;unread:number};
export type FamilyQuery={from:string;to:string;topic_offset:number;exam_offset:number;lesson_offset:number;assignment_offset:number;note_offset:number;event_offset:number;result_offset:number};
const post=(path:string,body:object={})=>request<unknown>(path,{method:"POST",body:JSON.stringify(body)});
export const familyApi={
 links:(offset=0,signal?:AbortSignal)=>request<unknown>(`/api/v1/family/links?limit=20&offset=${offset}`,{signal}),
 invite:(guardian_email:string)=>post("/api/v1/family/invitations",{guardian_email,confirm_share:true}),
 preview:(token:string)=>post("/api/v1/family/invitations/preview",{token}),
 accept:(token:string)=>post("/api/v1/family/invitations/accept",{token}),
 revoke:(id:string)=>post(`/api/v1/family/links/${encodeURIComponent(id)}/revoke`),
 report:(studentID:string,query:FamilyQuery,signal?:AbortSignal)=>request<unknown>(`/api/v1/family/children/${encodeURIComponent(studentID)}?${new URLSearchParams(Object.entries(query).map(([k,v])=>[k,String(v)]))}`,{signal}),
 read:(studentID:string)=>post(`/api/v1/family/children/${encodeURIComponent(studentID)}/read`),
 note:(input:{student_id:string;body:string;request_id:string})=>post("/api/v1/family/notes",input),
 notes:(studentID:string,offset=0,signal?:AbortSignal)=>request<unknown>(`/api/v1/family/students/${encodeURIComponent(studentID)}/notes?limit=20&offset=${offset}`,{signal}),
 withdraw:(id:string)=>post(`/api/v1/family/notes/${encodeURIComponent(id)}/withdraw`)
};
