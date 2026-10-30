import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { parseExercise, answerError, parseAnswer } from "../features/exercise-engine/schema.ts";
import { parseVisualConfig, defaultVisualConfig } from "../features/exercise-engine/visual-schema.ts";

const require = createRequire(import.meta.url);
const Ajv = createRequire(require.resolve("eslint/package.json"))("ajv");
const schema = JSON.parse(readFileSync(new URL("../features/exercise-engine/exercise.schema.json", import.meta.url)));
const validate = new Ajv({ allErrors: true }).compile(schema);
const exercise = (kind, config = defaultVisualConfig(kind)) => ({ id: "00000000-0000-4000-8000-000000000001", kind, prompt: "Задание", options: [], difficulty: 2, position: 0, config });
const points = [{ id: "a", label: "A", x: 0, y: 0 }, { id: "b", label: "B", x: 4, y: 0 }, { id: "c", label: "C", x: 2, y: 4 }];

test("all five visual types and three geometry modes have closed JSON schemas", () => {
  for (const kind of ["number_line", "grid_select", "matching", "ordering", "geometry"]) {
    const value = exercise(kind); assert.equal(validate(value), true, JSON.stringify(validate.errors)); assert.equal(parseExercise(value).kind, kind);
    for (const invalid of [{ ...value, answer_key: "secret" }, { ...value, correct: [] }, { ...value, config: { ...value.config, answer: "secret" } }, { ...value, config: null }, { ...value, config: undefined }]) {
      assert.equal(validate(invalid), false); assert.throws(() => parseExercise(invalid));
    }
  }
  for (const config of [{ mode: "connect", rows: 5, columns: 5, points }, { mode: "select", rows: 5, columns: 5, points, objects: [{ id: "seg", label: "Segment", kind: "segment", point_ids: ["a", "b"] }, { id: "tri", label: "Triangle", kind: "polygon", point_ids: ["a", "b", "c"] }] }]) { assert.equal(validate(exercise("geometry", config)), true); assert.doesNotThrow(() => parseExercise(exercise("geometry", config))); }
});

test("cross-field config constraints reject unsafe, duplicate and out-of-range objects", () => {
  for (const [kind, config] of [
    ["number_line", { min: 0, max: 10, step: 3 }], ["number_line", { min: 0, max: 1000, step: 1 }], ["number_line", { min: 10, max: 0, step: 1 }], ["number_line", { min: 0, max: 10, step: 0 }],
    ["grid_select", { rows: 9, columns: 2 }], ["grid_select", { rows: 2 }],
    ["matching", { left: [{ id: "a", label: "A" }, { id: "b", label: "B" }], right: [{ id: "x", label: "X" }] }],
    ["ordering", { items: [{ id: "a", label: "1" }, { id: "a", label: "2" }] }],
    ["geometry", { mode: "point", rows: 5, columns: 5, objects: [] }],
    ["geometry", { mode: "connect", rows: 5, columns: 5, points: [{ id: "a", label: "A", y: 0 }, points[1]] }],
    ["geometry", { mode: "connect", rows: 5, columns: 5, points: [points[0], { ...points[1], x: 0, y: 0 }] }],
    ["geometry", { mode: "connect", rows: 5, columns: 5, points: [points[0], { ...points[1], x: 5 }] }],
  ]) assert.throws(() => parseVisualConfig(kind, config), kind);
});

test("number line requires an integer on a visible tick", () => {
  const value = exercise("number_line", { min: -4, max: 4, step: 2 });
  assert.deepEqual(parseAnswer(value, { value: " −2 " }), { value: " −2 " });
  for (const input of ["-3", "5", "2.0", "4/2", "1e2", "NaN", "", "2 2"]) assert.notEqual(answerError(value, { value: input }), null);
});

test("grid, matching and ordering validate the answer, not an exposed key", () => {
  const grid = exercise("grid_select"); assert.equal(answerError(grid, { selected: ["0:0", "3:3"] }), null);
  for (const selected of [[], ["4:0"], ["0:4"], ["01:0"], ["0:0", "0:0"]]) assert.notEqual(answerError(grid, { selected }), null);
  const matching = exercise("matching");
  for (const selected of [["a:x", "b:y"], ["a:y", "b:x"]]) assert.equal(answerError(matching, { selected }), null);
  for (const selected of [["a:x"], ["a:x", "b:x"], ["a:x", "a:y"], ["a:x:y", "b:y"]]) assert.notEqual(answerError(matching, { selected }), null);
  const ordering = exercise("ordering");
  assert.equal(answerError(ordering, { selected: ["b", "c", "a"] }), null);
  for (const selected of [["a", "b"], ["a", "a", "b"], ["a", "b", "unknown"]]) assert.notEqual(answerError(ordering, { selected }), null);
});

test("geometry validates coordinates, canonical undirected edges and object IDs", () => {
  assert.equal(answerError(exercise("geometry"), { selected: ["2:3"] }), null);
  assert.notEqual(answerError(exercise("geometry"), { selected: ["5:0"] }), null);
  const edges = exercise("geometry", { mode: "connect", rows: 5, columns: 5, points });
  assert.equal(answerError(edges, { selected: ["a:b", "b:c"] }), null);
  for (const selected of [["b:a"], ["a:a"], ["a:unknown"], ["a:b", "a:b"]]) assert.notEqual(answerError(edges, { selected }), null);
  const objects = exercise("geometry", { mode: "select", rows: 5, columns: 5, points, objects: [{ id: "seg", label: "Segment", kind: "segment", point_ids: ["a", "b"] }, { id: "tri", label: "Triangle", kind: "polygon", point_ids: ["a", "b", "c"] }] });
  assert.equal(answerError(objects, { selected: ["tri"] }), null);
  assert.notEqual(answerError(objects, { selected: ["a"] }), null);
});
