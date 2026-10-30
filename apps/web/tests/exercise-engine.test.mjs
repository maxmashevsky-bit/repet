import test from "node:test";
import assert from "node:assert/strict";
import { validNumber, parseExercise, parseAnswer, answerError, parseAttempt } from "../features/exercise-engine/schema.ts";

const id = "00000000-0000-4000-8000-000000000001";
const exercise = (kind, options = []) => ({ id, kind, prompt: "Задача", options, difficulty: 2, position: 0 });

test("numbers: signs, exact fractions, comma, whitespace and bounded input", () => {
  for (const value of ["1/2", " -2 / -4 ", "+.5", "−3 / 6", "-0", "0/7", "1 / -2", "9007199254740993.0", "\u00a0-1/4\u00a0", ".000000000000000001"]) assert.equal(validNumber(value), true, value);
  for (const value of ["", " ", "1/0", "0/-0", "1//2", "1/2/3", "1 2", "1 1/2", "1.5/2", "--2", "- 2", "+", "1e3", "NaN", "Infinity", "1+2", "1,2.3", "1.", "½", "2%", "0x10", "9".repeat(129), "1\0"]) assert.equal(validNumber(value), false, value);
});

test("runtime schema supports five types, rejects keys and invalid structures", () => {
  for (const kind of ["text", "numeric", "long_text", "single_choice", "multiple_choice"]) assert.equal(parseExercise(exercise(kind, kind.includes("choice") ? ["a", "b"] : [])).kind, kind);
  for (const input of [null, [], { ...exercise("numeric"), answer_key: "secret" }, { ...exercise("text"), correct: "secret" }, { ...exercise("text"), kind: "html" }, exercise("single_choice", ["a"]), exercise("multiple_choice", ["a", "a"]), exercise("text", ["a"]), { ...exercise("text"), difficulty: 0 }, { ...exercise("text"), position: -1 }, { ...exercise("text"), prompt: "\ud800" }]) assert.throws(() => parseExercise(input));
  const original = exercise("single_choice", ["a", "b"]);
  const parsed = parseExercise(original); original.options.push("c");
  assert.deepEqual(parsed.options, ["a", "b"]);
});

test("answer validation denies forged values, duplicates, blanks and oversized text", () => {
  const multi = exercise("multiple_choice", ["a", "b"]);
  assert.deepEqual(parseAnswer(multi, { selected: ["b", "a"] }), { selected: ["b", "a"] });
  for (const answer of [{ selected: [] }, { selected: ["a", "a"] }, { selected: ["c"] }, { selected: ["a"], value: "a" }, { selected: ["a"], status: "correct" }]) assert.notEqual(answerError(multi, answer), null);
  assert.notEqual(answerError(exercise("single_choice", ["a", "b"]), { selected: ["a", "b"] }), null);
  assert.equal(answerError(exercise("numeric"), { value: " −2 / 4 " }), null);
  for (const answer of [{}, { value: " \n\t " }, { value: "x\0" }, { value: "\ud800" }, { value: "я".repeat(1001) }, { value: "x", selected: [] }]) assert.notEqual(answerError(exercise("text"), answer), null);
  assert.equal(answerError(exercise("long_text"), { value: "я".repeat(20000) }), null);
  assert.notEqual(answerError(exercise("long_text"), { value: "я".repeat(20001) }), null);
});

test("server results have a closed runtime-validated shape without keys", () => {
  const attempt = { id, exercise_id: id, version_id: id, version: 1, student_id: id, student_name: "Ученик", kind: "numeric", prompt: "Задача", answer: { value: "1/2" }, status: "correct", feedback: "", created_at: "2026-10-19T00:00:00Z", reviewed_at: null, learning_attempt_id: null, attempt_number: 1, hints_used: 0, elapsed_seconds: 0 };
  assert.equal(parseAttempt(attempt).status, "correct");
  for (const value of [{ ...attempt, answer_key: "secret" }, { ...attempt, status: "needs_review" }, { ...attempt, answer: { value: "1/0" } }, { ...attempt, reviewed_at: "bad date" }]) assert.throws(() => parseAttempt(value));
});
