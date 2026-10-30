import { request } from "./client";
import type { CatalogPage } from "./catalog";
import type { LearningAttempt, LearningState } from "./learning";
import type { StudyReview } from "./study";
export type RewardPreference={enabled:boolean;enabled_from:string|null;updated_at:string|null};
export type RewardSummary={enabled:boolean;xp:number;level:number;level_progress:number;level_target:number;weekly_xp:number;weekly_achievements:number;awards:CatalogPage<{id:string;title:string;points:number;earned_at:string}>};
export type ReviewMode="due"|"errors"|"all";
export type ReviewItem={source_run_id:string;lesson_id:string;exercise_id:string;topic_key:string;topic:string;subtopic:string;prompt:string;status:"correct"|"incorrect"|"unanswered";clean_days:number;due_at:string};
export type ReviewPlan={server_time:string;due_total:number;mistake_total:number;unanswered:number;pending:number;weekly_completed:number;weekly_target:number;items:CatalogPage<ReviewItem>;active:CatalogPage<{id:string;title:string;attempt:LearningAttempt}>};
export type ReviewSession={id:string;state:LearningState};
export type ReviewInput={request_id:string;source_run_id:string;topic_key:string;mode:"quiz"|"errors"|"due";count:number};
export const reviewApi={
 plan:(studentID:string,mode:ReviewMode,offset=0,signal?:AbortSignal,activeOffset=0)=>request<unknown>(`/api/v1/learning-reviews/${encodeURIComponent(studentID)}?mode=${mode}&limit=20&offset=${offset}&active_offset=${activeOffset}`,{signal}),
 start:(input:ReviewInput)=>request<unknown>("/api/v1/learning-review-sessions",{method:"POST",body:JSON.stringify(input)}),
 session:(id:string,signal?:AbortSignal)=>request<unknown>(`/api/v1/learning-review-sessions/${encodeURIComponent(id)}`,{signal}),
 checks:(offset=0,signal?:AbortSignal)=>request<CatalogPage<StudyReview>>(`/api/v1/learning-review-checks?limit=20&offset=${offset}`,{signal}),
 rewards:(studentID:string,offset=0,signal?:AbortSignal)=>request<unknown>(`/api/v1/learning-rewards/${encodeURIComponent(studentID)}?limit=20&offset=${offset}`,{signal}),
 preference:(studentID:string,signal?:AbortSignal)=>request<unknown>(`/api/v1/learning-rewards/${encodeURIComponent(studentID)}/preference`,{signal}),
 savePreference:(studentID:string,enabled:boolean)=>request<unknown>(`/api/v1/learning-rewards/${encodeURIComponent(studentID)}/preference`,{method:"PUT",body:JSON.stringify({enabled})})
};
