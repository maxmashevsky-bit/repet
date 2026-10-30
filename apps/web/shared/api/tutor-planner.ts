import { request } from "./client";

export type PlanStudent = { id: string; relation_id?:string; name: string; subject: string; price_kopecks: number; duration: number; contact: string };
export type PlanLesson = { id: string; student_id: string; weekday: number; start_minute: number; duration: number; paused: boolean };
export type PlanWindow = { weekday: number; start_minute: number };
export type PlanCalculator = { goal_kopecks: number; price_kopecks: number; lessons_per_student: number; duration: number; weeks: number; people: number; preparation_hours: number };
export type TutorPlan = { revision: number; students: PlanStudent[]; lessons: PlanLesson[]; windows: PlanWindow[]; calculator: PlanCalculator };
export type PlanCalendarResult={created:number;existing:number;unlinked:number;paused:number;from:string;to:string};

export const tutorPlannerApi = {
  get: () => request<TutorPlan>("/api/v1/tutor-planner"),
  save: (plan: TutorPlan) => request<TutorPlan>("/api/v1/tutor-planner", { method: "PUT", body: JSON.stringify(plan) }),
  publish:(revision:number,from:string,to:string,request_id:string)=>request<PlanCalendarResult>("/api/v1/tutor-planner/calendar",{method:"POST",body:JSON.stringify({revision,from,to,request_id})})
};
