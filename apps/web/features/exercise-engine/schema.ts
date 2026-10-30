import type { Exercise } from "../../shared/api/catalog";
import type { ExerciseAnswer, ExerciseAttempt } from "../../shared/api/exercises";
import { isVisual, parseVisualConfig, visualAnswerError, visualKinds } from "./visual-schema.ts";

export type ExerciseKind = Exercise["kind"];
export const exerciseKinds: readonly ExerciseKind[] = ["single_choice", "multiple_choice", "numeric", "text", "long_text", ...visualKinds];
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const decimal = /^[+-]?(?:[0-9]+(?:[.,][0-9]+)?|[.,][0-9]+)$/;
const integer = /^[+-]?[0-9]+$/;
const object = (value: unknown): value is Record<string, unknown> => typeof value === "object" && value !== null && !Array.isArray(value);
const text = (value: unknown, max: number, nonempty = true): value is string => typeof value === "string" && (!nonempty || value.trim().length > 0) && !value.includes("\0") && Array.from(value).length <= max && new TextDecoder().decode(new TextEncoder().encode(value)) === value;
const fail = (): never => { throw new Error("Некорректный формат упражнения или ответа сервера."); };
const exactKeys = (value: Record<string, unknown>, keys: string[]) => Object.keys(value).every((key) => keys.includes(key));

export function validNumber(input: string): boolean {
  if (new TextEncoder().encode(input).length > 128) return false;
  const value = input.trim().replaceAll("−", "-");
  const parts = value.split("/").map((item) => item.trim());
  if (parts.length === 2) return parts.every((item) => integer.test(item)) && BigInt(parts[1]) !== BigInt(0);
  return parts.length === 1 && decimal.test(value);
}

export function parseExercise(input: unknown): Exercise {
  if (!object(input) || !exactKeys(input, ["id", "kind", "prompt", "options", "difficulty", "position", "config", "hint_count", "media"])) return fail();
  if (input.media !== undefined && (!Array.isArray(input.media) || input.media.length > 4 || input.media.some(m => !object(m) || !exactKeys(m,["asset_id","alt"]) || typeof m.asset_id!=="string" || !uuid.test(m.asset_id) || !text(m.alt,1000)) || new Set(input.media.map(m=>m.asset_id)).size!==input.media.length)) return fail();
  if (input.hint_count !== undefined && (!Number.isInteger(input.hint_count) || (input.hint_count as number)<0 || (input.hint_count as number)>5)) return fail();
  const { id, kind, prompt, options, difficulty, position } = input;
  if (typeof id !== "string" || !uuid.test(id) || !exerciseKinds.includes(kind as ExerciseKind) || !text(prompt, 20000) || !Array.isArray(options) || options.some((option) => !text(option, 1000)) || new Set(options).size !== options.length || typeof difficulty !== "number" || !Number.isInteger(difficulty) || difficulty < 1 || difficulty > 5 || typeof position !== "number" || !Number.isInteger(position) || position < 0 || position > 1000000) return fail();
  const choice = kind === "single_choice" || kind === "multiple_choice";
  if (choice ? options.length < 2 || options.length > 12 : options.length !== 0) return fail();
  return { id, kind: kind as ExerciseKind, prompt, options: [...options] as string[], difficulty, position, config: parseVisualConfig(kind as string, input.config), hint_count: input.hint_count as number | undefined, ...(input.media!==undefined?{media:input.media as Exercise["media"]}:{}) };
}

export function answerError(exercise: Exercise, input: unknown): string | null {
  if (isVisual(exercise.kind)) return visualAnswerError(exercise, input);
  if (!object(input)) return "Заполните ответ.";
  if (exercise.kind === "single_choice" || exercise.kind === "multiple_choice") {
    if (!exactKeys(input, ["selected"]) || !Array.isArray(input.selected) || !input.selected.length) return "Выберите ответ.";
    if (input.selected.some((item) => typeof item !== "string" || !exercise.options.includes(item)) || new Set(input.selected).size !== input.selected.length || exercise.kind === "single_choice" && input.selected.length !== 1) return "Выберите допустимые варианты ответа.";
    return null;
  }
  if (!exactKeys(input, ["value"]) || !text(input.value, exercise.kind === "long_text" ? 20000 : exercise.kind === "numeric" ? 128 : 1000)) return "Введите непустой ответ допустимой длины.";
  if (exercise.kind === "numeric" && !validNumber(input.value)) return "Введите число или дробь с ненулевым знаменателем.";
  return null;
}

export function parseAnswer(exercise: Exercise, input: unknown): ExerciseAnswer {
  const error = answerError(exercise, input);
  if (error) throw new Error(error);
  const value = input as ExerciseAnswer;
  return "selected" in value ? { selected: [...value.selected!] } : { value: value.value };
}

export function parseAttempt(input: unknown): ExerciseAttempt {
  if (!object(input) || !exactKeys(input, ["id", "exercise_id", "version_id", "version", "student_id", "student_name", "kind", "prompt", "answer", "status", "feedback", "created_at", "reviewed_at", "learning_attempt_id", "attempt_number", "hints_used", "elapsed_seconds"])) return fail();
  if (input.learning_attempt_id!==null && (typeof input.learning_attempt_id!=="string" || !uuid.test(input.learning_attempt_id)) || !Number.isSafeInteger(input.attempt_number) || (input.attempt_number as number)<1 || !Number.isInteger(input.hints_used) || (input.hints_used as number)<0 || (input.hints_used as number)>5 || !Number.isSafeInteger(input.elapsed_seconds) || (input.elapsed_seconds as number)<0) return fail();
  if ([input.id, input.exercise_id, input.version_id, input.student_id].some((id) => typeof id !== "string" || !uuid.test(id)) || !Number.isInteger(input.version) || (input.version as number) < 1 || !exerciseKinds.includes(input.kind as ExerciseKind) || !text(input.student_name, 1000) || !text(input.prompt, 20000) || !text(input.feedback, 10000, false) || !["correct", "incorrect", "needs_review","submitted"].includes(input.status as string) || input.status === "needs_review" && input.kind !== "long_text" || typeof input.created_at !== "string" || !Number.isFinite(Date.parse(input.created_at)) || input.reviewed_at !== null && (typeof input.reviewed_at !== "string" || !Number.isFinite(Date.parse(input.reviewed_at)))) return fail();
  if(input.status==="submitted"&&(input.feedback!==""||input.reviewed_at!==null))return fail();
  const answer = input.answer;
  if (!object(answer)) return fail();
  if (input.kind === "single_choice" || input.kind === "multiple_choice" || isVisual(input.kind as string) && input.kind !== "number_line") {
    if (!exactKeys(answer, ["selected"]) || !Array.isArray(answer.selected) || !answer.selected.length || answer.selected.length > (isVisual(input.kind as string) ? 66 : 12) || answer.selected.some((item) => !text(item, 1000)) || new Set(answer.selected).size !== answer.selected.length || input.kind === "single_choice" && answer.selected.length !== 1) return fail();
  } else if (!exactKeys(answer, ["value"]) || !text(answer.value, input.kind === "long_text" ? 20000 : input.kind === "numeric" ? 128 : 1000) || input.kind === "numeric" && !validNumber(answer.value)) return fail();
  return input as unknown as ExerciseAttempt;
}

export function parseAttemptsPage(input: unknown) {
  if (!object(input) || !Array.isArray(input.items) || !Number.isInteger(input.total) || (input.total as number) < 0 || !Number.isInteger(input.offset) || (input.offset as number) < 0 || !Number.isInteger(input.limit) || (input.limit as number) < 1) return fail();
  return { items: input.items.map(parseAttempt), total: input.total as number, offset: input.offset as number, limit: input.limit as number };
}
