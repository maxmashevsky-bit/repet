"use client";
import { Select } from "../ui/Select";


import type { Exercise } from "../../shared/api/catalog";
import { isVisual } from "../exercise-engine/visual-schema";
import { parseExercise } from "../exercise-engine/schema";
import { exerciseRegistry } from "../exercise-engine/registry";
import type { ExerciseAnswer } from "../../shared/api/exercises";

export function ExerciseKeyFields({ kind, options, answer, change, config }: { kind: Exercise["kind"]; options: string[]; answer: string; change: (answer: string) => void; config?: Record<string, unknown> }) {
  if (kind === "long_text") return null;
  if (isVisual(kind)) {
    let spec: Exercise;
    try { spec = parseExercise({ id: "00000000-0000-4000-8000-000000000001", kind, options: [], prompt: "Правильный ответ", difficulty: 1, position: 0, config }); } catch { return <p className="engine-error">Проверьте настройки JSON задания.</p>; }
    let value: ExerciseAnswer = kind === "number_line" ? { value: answer } : { selected: [] };
    if (kind !== "number_line") { try { const selected: unknown = JSON.parse(answer); if (Array.isArray(selected) && selected.every((item) => typeof item === "string")) value = { selected }; } catch { /* The author has not selected a key yet. */ } }
    const Input = exerciseRegistry[kind].Component;
    return <fieldset className="material-fieldset"><legend>Правильный ответ</legend><Input exercise={spec} value={value} onChange={(next) => change("selected" in next ? JSON.stringify(next.selected) : next.value)} disabled={false} invalid={false} errorID="" /></fieldset>;
  }
  if (kind === "multiple_choice") {
    let selected: string[] = [];
    try { const parsed: unknown = JSON.parse(answer); if (Array.isArray(parsed) && parsed.every((item) => typeof item === "string")) selected = parsed; } catch { /* A new key starts empty. */ }
    return <fieldset className="engine-options"><legend>Правильные ответы</legend>{options.map((option, index) => <label className="engine-option" key={index}><input type="checkbox" checked={selected.includes(option)} onChange={(event) => change(JSON.stringify(event.target.checked ? [...selected, option] : selected.filter((item) => item !== option)))} /><span>{option}</span></label>)}</fieldset>;
  }
  if (kind === "single_choice") return <label>Правильный ответ<Select aria-label="Правильный ответ" value={answer} onChange={(event) => change(event.target.value)} required><option value="">Выберите правильный ответ</option>{options.map((option, index) => <option key={index} value={option}>{option}</option>)}</Select></label>;
  return <label>Правильный ответ<input aria-label="Правильный ответ" value={answer} onChange={(event) => change(event.target.value)} maxLength={kind === "numeric" ? 128 : 1000} required /></label>;
}
