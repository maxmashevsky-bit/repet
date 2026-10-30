import type { StudySession } from "../../shared/api/study";
import { parseLearningState } from "../exercise-engine/learning-schema.ts";

const object = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);
const uuid = (v: unknown) => typeof v === "string" && /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i.test(v);
const date = (v: unknown): v is string => typeof v === "string" && Number.isFinite(Date.parse(v));
const integer = (v: unknown, min: number, max: number) => Number.isSafeInteger(v) && (v as number) >= min && (v as number) <= max;
function fail(): never { throw new Error("Некорректное состояние самостоятельной работы."); }

export function parseStudySession(v: unknown): StudySession {
  if (!object(v) || Object.keys(v).some(k => !["id", "specification", "seed", "deadline_at", "server_time", "points", "max_points", "state"].includes(k)) || !uuid(v.id) || !object(v.specification)) return fail();
  const s = v.specification;
  if (Object.keys(s).some(k => !["id", "family_id", "version", "author_id", "author_name", "collection_id", "title", "direction", "year", "duration_seconds", "instructions", "slots", "topics", "exercise_count", "status"].includes(k)) || ![s.id, s.family_id, s.author_id, s.collection_id].every(uuid) || !integer(s.version, 1, 1000000) || !integer(s.year, 1, 9999) || !integer(s.duration_seconds, 60, 28800) || !integer(s.exercise_count, 1, 200) || !["oge", "ege_basic", "ege_profile"].includes(s.direction as string) || !["draft", "published", "archived"].includes(s.status as string) || typeof s.title !== "string" || !s.title.trim() || s.title.length > 200 || typeof s.author_name !== "string" || typeof s.instructions !== "string" || s.instructions.length > 10000 || !Array.isArray(s.topics) || s.topics.some(t => typeof t !== "string") || !Array.isArray(s.slots) || !s.slots.length || s.slots.length > 100) return fail();
  let count = 0;
  const numbers = new Set<number>();
  for (const slot of s.slots) {
    if (!object(slot) || Object.keys(slot).some(k => !["number", "count", "points"].includes(k)) || !integer(slot.number, 1, 100) || !integer(slot.count, 1, 100) || !integer(slot.points, 1, 100) || numbers.has(slot.number as number)) return fail();
    numbers.add(slot.number as number); count += slot.count as number;
  }
  if (count > 100 || !date(v.server_time) || typeof v.seed !== "string" || !/^[a-z0-9-]{1,64}$/i.test(v.seed) || !integer(v.points, 0, 10000) || !integer(v.max_points, 1, 10000) || (v.points as number) > (v.max_points as number)) return fail();
  const state = parseLearningState(v.state);
  if (state.attempt.assignment_id || !state.version.exercises.length || (state.attempt.mode === "exam" ? !date(v.deadline_at) : v.deadline_at !== null)) return fail();
  if (state.attempt.mode === "exam" && (Date.parse(v.deadline_at as string) <= Date.parse(state.attempt.started_at) || state.attempt.status === "in_progress" && v.points !== 0)) return fail();
  return { ...v, state } as unknown as StudySession;
}

export function remainingSeconds(deadline: string | null, serverTime: string, elapsedMilliseconds: number): number {
  if (!deadline) return 0;
  return Math.max(0, Math.ceil((Date.parse(deadline) - Date.parse(serverTime) - Math.max(0, elapsedMilliseconds)) / 1000));
}
export function clockLabel(seconds: number): string {
  const n = Math.max(0, Math.floor(seconds));
  return `${Math.floor(n / 3600).toString().padStart(2, "0")}:${Math.floor(n / 60) % 60 < 10 ? "0" : ""}${Math.floor(n / 60) % 60}:${(n % 60).toString().padStart(2, "0")}`;
}
export function repeatExerciseCount(session: StudySession): number {
  return session.state.version.exercises.filter(e => { const answer = session.state.answers.find(a => a.exercise_id === e.id); return !answer || answer.status === "incorrect"; }).length;
}
