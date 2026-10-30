"use client";

import { useId, type ComponentType } from "react";
import type { Exercise } from "../../shared/api/catalog";
import type { ExerciseAnswer } from "../../shared/api/exercises";
import type { ExerciseKind } from "./schema";
import { NumberLineInput, GridSelectInput, MatchingInput, OrderingInput } from "./VisualInputs";
import { GeometryInput } from "./GeometryInput";
import type { OrderingConfig } from "./visual-schema";

export type ExerciseInputProps = { exercise: Exercise; value: ExerciseAnswer; onChange: (answer: ExerciseAnswer) => void; disabled: boolean; errorID: string; invalid: boolean };

export function ChoiceInput({ exercise, value, onChange, disabled, errorID, invalid }: ExerciseInputProps) {
  const name = useId();
  const selected = value.selected ?? [];
  const multiple = exercise.kind === "multiple_choice";
  return <fieldset disabled={disabled} className="engine-options" aria-describedby={invalid ? errorID : undefined} aria-invalid={invalid}><legend className="sr-only">{multiple ? "Выберите несколько ответов" : "Выберите один ответ"}</legend>{exercise.options.map((option, index) => <label key={index} className={selected.includes(option) ? "engine-option selected" : "engine-option"}><input type={multiple ? "checkbox" : "radio"} name={name} checked={selected.includes(option)} onChange={(event) => onChange({ selected: multiple ? event.target.checked ? [...selected, option] : selected.filter((item) => item !== option) : [option] })} /><span>{option}</span></label>)}</fieldset>;
}

export function TextInput({ exercise, value, onChange, disabled, errorID, invalid }: ExerciseInputProps) {
  const id = useId();
  const numeric = exercise.kind === "numeric";
  return <label htmlFor={id}>{numeric ? "Число или дробь" : "Короткий ответ"}<input id={id} type="text" value={value.value ?? ""} onChange={(event) => onChange({ value: event.target.value })} maxLength={numeric ? 128 : 1000} disabled={disabled} autoComplete="off" spellCheck={!numeric} aria-invalid={invalid} aria-describedby={invalid ? errorID : undefined} /></label>;
}

export function SolutionInput({ value, onChange, disabled, errorID, invalid }: ExerciseInputProps) {
  const id = useId();
  return <label htmlFor={id}>Развёрнутое решение<textarea id={id} rows={6} value={value.value ?? ""} onChange={(event) => onChange({ value: event.target.value })} maxLength={20000} disabled={disabled} aria-invalid={invalid} aria-describedby={invalid ? errorID : undefined} /></label>;
}

type Entry = { label: string; Component: ComponentType<ExerciseInputProps>; initial: (exercise: Exercise) => ExerciseAnswer };
export const exerciseRegistry: Record<ExerciseKind, Entry> = {
  single_choice: { label: "Один ответ", Component: ChoiceInput, initial: () => ({ selected: [] }) },
  multiple_choice: { label: "Несколько ответов", Component: ChoiceInput, initial: () => ({ selected: [] }) },
  numeric: { label: "Число или дробь", Component: TextInput, initial: () => ({ value: "" }) },
  text: { label: "Короткий ответ", Component: TextInput, initial: () => ({ value: "" }) },
  long_text: { label: "Развёрнутое решение", Component: SolutionInput, initial: () => ({ value: "" }) },
  number_line: { label: "Числовая прямая", Component: NumberLineInput, initial: () => ({ value: "" }) },
  grid_select: { label: "Клетчатое поле", Component: GridSelectInput, initial: () => ({ selected: [] }) },
  matching: { label: "Соответствия", Component: MatchingInput, initial: () => ({ selected: [] }) },
  ordering: { label: "Порядок элементов", Component: OrderingInput, initial: (exercise) => ({ selected: (exercise.config as OrderingConfig).items.map((item) => item.id) }) },
  geometry: { label: "Геометрия", Component: GeometryInput, initial: () => ({ selected: [] }) }
};
