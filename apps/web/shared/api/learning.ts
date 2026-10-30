import { request } from "./client";
import type { LessonVersion, CatalogPage } from "./catalog";
import type { ExerciseAttempt } from "./exercises";

export type LearningAttempt = {
 assignment_id?:string;
 mode:"practice"|"exam"; player_step:string; theory_seen:boolean; example_seen:boolean;
 id: string; lesson_id: string; version_id: string; number: number; status: "in_progress" | "completed";
 started_at: string; completed_at: string | null; elapsed_seconds: number; total_exercises: number;
 answered_exercises: number; correct_exercises: number; pending_review: number; incorrect_attempts: number;
};
export type LearningHint = { exercise_id: string; number: number; text: string; revealed_at: string };
export type LearningState = { attempt: LearningAttempt; version: LessonVersion; answers: ExerciseAttempt[]; hints: LearningHint[] };
const base = (lesson: string) => `/api/v1/catalog/lessons/${encodeURIComponent(lesson)}/learning-attempts`;
const write = (url: string, body: object) => request<unknown>(url, { method: "POST", body: JSON.stringify(body) });
export const learningApi = {
 start: (lesson: string, requestID: string, restart = false,mode?:"practice"|"exam") => write(base(lesson), { request_id: requestID, restart,...(mode?{mode}:{}) }),
 navigate:(lesson:string,id:string,step:string,requestID:string)=>write(`${base(lesson)}/${encodeURIComponent(id)}/navigate`,{request_id:requestID,step}),
 solution:(lesson:string,id:string,exercise:string,signal?:AbortSignal)=>request<unknown>(`${base(lesson)}/${encodeURIComponent(id)}/exercises/${encodeURIComponent(exercise)}/solution`,{signal}),
 state: (lesson: string, id: string, signal?: AbortSignal) => request<unknown>(`${base(lesson)}/${encodeURIComponent(id)}`, { signal }),
 results: (lesson: string, offset = 0, signal?: AbortSignal) => request<CatalogPage<LearningAttempt>>(`${base(lesson)}?limit=20&offset=${offset}`, { signal }),
 answers: (lesson: string, id: string, offset = 0, signal?: AbortSignal) => request<unknown>(`${base(lesson)}/${encodeURIComponent(id)}/answers?limit=20&offset=${offset}`, { signal }),
 complete: (lesson: string, id: string, requestID: string) => write(`${base(lesson)}/${encodeURIComponent(id)}/complete`, { request_id: requestID }),
 hint: (lesson: string, id: string, exercise: string, requestID: string) => write(`${base(lesson)}/${encodeURIComponent(id)}/exercises/${encodeURIComponent(exercise)}/hints`, { request_id: requestID })
};
