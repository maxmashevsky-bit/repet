import { request } from "./client";

export type ExerciseAnswer = { value: string; selected?: never } | { selected: string[]; value?: never };
export type AttemptStatus = "correct" | "incorrect" | "needs_review" | "submitted";
export type ExerciseAttempt = {
  id: string; exercise_id: string; version_id: string; version: number;
  student_id: string; student_name: string; kind: string; prompt: string;
  answer: ExerciseAnswer; status: AttemptStatus; feedback: string; created_at: string; reviewed_at: string | null;
  learning_attempt_id: string | null; attempt_number: number; hints_used: number; elapsed_seconds: number;
};

const base = (lessonID: string) => `/api/v1/catalog/lessons/${encodeURIComponent(lessonID)}`;
export const exerciseApi = {
  submit: (lessonID: string, versionID: string, exerciseID: string, answer: ExerciseAnswer, requestID: string, learningID?: string) => request<unknown>(`${base(lessonID)}/exercises/${encodeURIComponent(exerciseID)}/attempts`, { method: "POST", body: JSON.stringify({ version_id: versionID, request_id: requestID, answer, ...(learningID ? { learning_attempt_id: learningID } : {}) }) }),
  attempts: (lessonID: string, offset: number, signal?: AbortSignal) => request<unknown>(`${base(lessonID)}/attempts?limit=100&offset=${offset}`, { signal }),
  review: (lessonID: string, attemptID: string, status: "correct" | "incorrect", feedback: string) => request<unknown>(`${base(lessonID)}/attempts/${encodeURIComponent(attemptID)}/review`, { method: "PUT", body: JSON.stringify({ status, feedback }) })
};
