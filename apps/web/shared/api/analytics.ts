import { request } from "./client";
import type { CatalogPage } from "./catalog";
import type { Assignment } from "./types";
export type AnalyticsStudent = { id: string; display_name: string };
export type AnalyticsMetric = { attempts: number; graded: number; correct: number; incorrect: number; pending: number; mistakes: number; hints: number; hinted: number; accuracy: number | null; hint_rate: number | null };
export type AnalyticsTopic = AnalyticsMetric & { key: string; topic: string; subtopic: string; level: "insufficient" | "practice" | "developing" | "secure"; previous_graded: number; recent_graded: number; change: number | null };
export type AnalyticsDay = { day: string; attempts: number; graded: number; correct: number; accuracy: number | null };
export type AnalyticsExam = { id: string; title: string; started_at: string; completed_at: string | null; points: number | null; max_points: number; pending: number };
export type AnalyticsRecommendation = { topic_key: string; topic: string; subtopic: string; run_id: string; lesson_id: string; href: string; sample: number; errors: number; reason: "few_observations" | "weak_topic" | "recent_errors"; exercises: { exercise_id: string; prompt: string; attempts: number; mistakes: number }[] };
export type AnalyticsReport = { student: AnalyticsStudent; from: string; to: string; generated_at: string; summary: AnalyticsMetric; homework: { total: number; done: number; submitted: number; overdue: number }; topics: CatalogPage<AnalyticsTopic>; timeline: AnalyticsDay[]; exams: CatalogPage<AnalyticsExam>; recommendations: AnalyticsRecommendation[] };
export type AnalyticsQuery = { from: string; to: string; topic_offset: number; exam_offset: number };
export const analyticsApi = {
  students: (offset = 0, signal?: AbortSignal) => request<CatalogPage<AnalyticsStudent>>(`/api/v1/learning-analytics/students?limit=20&offset=${offset}`, { signal }),
  report: (studentID: string, query: AnalyticsQuery, signal?: AbortSignal) => request<unknown>(`/api/v1/learning-analytics/${encodeURIComponent(studentID)}?${new URLSearchParams(Object.entries(query).map(([key, value]) => [key, String(value)]))}`, { signal }),
  repeat: (studentID: string, input: { request_id: string; run_id: string; topic_key: string; due_at: string }) => request<Assignment>(`/api/v1/learning-analytics/${encodeURIComponent(studentID)}/repeat`, { method: "POST", body: JSON.stringify(input) })
};
