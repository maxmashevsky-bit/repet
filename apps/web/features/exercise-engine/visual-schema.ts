import type { Exercise } from "../../shared/api/catalog";

export type VisualItem = { id: string; label: string };
export type NumberLineConfig = { min: number; max: number; step: number };
export type GridConfig = { rows: number; columns: number };
export type MatchingConfig = { left: VisualItem[]; right: VisualItem[] };
export type OrderingConfig = { items: VisualItem[] };
export type GeometryPoint = VisualItem & { x: number; y: number };
export type GeometryObject = VisualItem & { kind: "segment" | "polygon"; point_ids: string[] };
export type GeometryConfig = GridConfig & { mode: "point" | "connect" | "select"; points?: GeometryPoint[]; objects?: GeometryObject[] };
export const visualKinds = ["number_line", "grid_select", "matching", "ordering", "geometry"] as const;
export const isVisual = (kind: string) => visualKinds.some((value) => value === kind);
const object = (value: unknown): value is Record<string, unknown> => typeof value === "object" && value !== null && !Array.isArray(value) && [Object.prototype, null].includes(Object.getPrototypeOf(value));
const exact = (value: Record<string, unknown>, keys: string[]) => Object.keys(value).length === keys.length && keys.every((key) => Object.hasOwn(value, key));
const int = (value: unknown, min: number, max: number): value is number => typeof value === "number" && Number.isInteger(value) && value >= min && value <= max;
const label = (value: unknown): value is string => typeof value === "string" && value.trim().length > 0 && Array.from(value).length <= 80 && !value.includes("\0") && new TextDecoder().decode(new TextEncoder().encode(value)) === value;
const id = (value: unknown): value is string => typeof value === "string" && /^[A-Za-z][A-Za-z0-9_-]{0,15}$/.test(value);
const unique = (values: unknown[]) => new Set(values).size === values.length;
const items = (value: unknown, min: number, max: number): value is VisualItem[] => Array.isArray(value) && value.length >= min && value.length <= max && value.every((item) => object(item) && exact(item, ["id", "label"]) && id(item.id) && label(item.label)) && unique(value.map((item) => item.id)) && unique(value.map((item) => item.label));
const bad = (): never => { throw new Error("Некорректная конфигурация математического инструмента."); };

export function parseVisualConfig(kind: string, input: unknown = {}): Record<string, unknown> {
  if (!object(input) || JSON.stringify(input).length > 32000) return bad();
  switch (kind) {
    case "number_line":
      if (!exact(input, ["min", "max", "step"]) || !int(input.min, -1000, 1000) || !int(input.max, -1000, 1000) || !int(input.step, 1, 100) || input.max <= input.min || (input.max - input.min) % input.step !== 0 || (input.max - input.min) / input.step > 100) return bad();
      break;
    case "grid_select":
      if (!exact(input, ["rows", "columns"]) || !int(input.rows, 1, 8) || !int(input.columns, 1, 8)) return bad();
      break;
    case "matching":
      if (!exact(input, ["left", "right"]) || !items(input.left, 2, 8) || !items(input.right, 2, 8) || input.left.length !== input.right.length) return bad();
      break;
    case "ordering":
      if (!exact(input, ["items"]) || !items(input.items, 2, 12)) return bad();
      break;
    case "geometry": {
      if (!int(input.rows, 2, 8) || !int(input.columns, 2, 8)) return bad();
      if (input.mode === "point") { if (!exact(input, ["mode", "rows", "columns"])) return bad(); break; }
      if (input.mode !== "connect" && input.mode !== "select" || !exact(input, input.mode === "connect" ? ["mode", "rows", "columns", "points"] : ["mode", "rows", "columns", "points", "objects"])) return bad();
      const points = input.points;
      if (!Array.isArray(points) || points.length < 2 || points.length > 12 || !points.every((point) => object(point) && exact(point, ["id", "label", "x", "y"]) && id(point.id) && label(point.label) && int(point.x, 0, (input.columns as number) - 1) && int(point.y, 0, (input.rows as number) - 1)) || !unique(points.map((point) => point.id)) || !unique(points.map((point) => point.label)) || !unique(points.map((point) => `${point.x}:${point.y}`))) return bad();
      if (input.mode === "select") {
        const objects = input.objects;
        if (!Array.isArray(objects) || objects.length < 2 || objects.length > 12 || !unique(objects.map((item) => object(item) ? item.id : "")) || !unique(objects.map((item) => object(item) ? item.label : ""))) return bad();
        for (const item of objects) if (!object(item) || !exact(item, ["id", "label", "kind", "point_ids"]) || !id(item.id) || !label(item.label) || !["segment", "polygon"].includes(item.kind as string) || !Array.isArray(item.point_ids) || item.point_ids.length < (item.kind === "segment" ? 2 : 3) || item.point_ids.length > (item.kind === "segment" ? 2 : 12) || !unique(item.point_ids) || !item.point_ids.every((value) => points.some((point) => point.id === value))) return bad();
      }
      break;
    }
    default: if (!exact(input, [])) return bad();
  }
  return JSON.parse(JSON.stringify(input)) as Record<string, unknown>;
}

export function lineCoordinate(value: unknown): number | null {
  if (typeof value !== "string" || new TextEncoder().encode(value).length > 128) return null;
  const normalized = value.trim().replaceAll("−", "-");
  if (!/^[+-]?[0-9]+$/.test(normalized)) return null;
  const number = BigInt(normalized);
  return number < BigInt(-1000) || number > BigInt(1000) ? null : Number(number);
}
function cell(value: string, rows: number, columns: number) {
  const parts = value.split(":");
  return parts.length === 2 && parts.every((part) => /^(0|[1-7])$/.test(part)) && Number(parts[0]) < rows && Number(parts[1]) < columns;
}
export function visualAnswerError(exercise: Exercise, input: unknown): string | null {
  let config: Record<string, unknown>;
  try { config = parseVisualConfig(exercise.kind, exercise.config); } catch { return "Проверьте конфигурацию задания."; }
  if (!object(input)) return "Заполните ответ.";
  if (exercise.kind === "number_line") {
    const c = config as NumberLineConfig; const n = lineCoordinate(input.value);
    return exact(input, ["value"]) && n !== null && n >= c.min && n <= c.max && (n - c.min) % c.step === 0 ? null : "Поставьте точку на допустимую отметку прямой.";
  }
  if (!exact(input, ["selected"]) || !Array.isArray(input.selected) || !input.selected.length || input.selected.length > 66 || !unique(input.selected) || input.selected.some((value) => typeof value !== "string" || value.length > 40)) return "Завершите ответ без повторяющихся объектов.";
  const selected = input.selected as string[];
  let valid = false;
  switch (exercise.kind) {
    case "grid_select": { const c = config as GridConfig; valid = selected.every((value) => cell(value, c.rows, c.columns)); break; }
    case "matching": { const c = config as MatchingConfig; const pairs = selected.map((value) => value.split(":")); valid = pairs.length === c.left.length && unique(pairs.map((pair) => pair[0])) && unique(pairs.map((pair) => pair[1])) && pairs.every((pair) => pair.length === 2 && c.left.some((item) => item.id === pair[0]) && c.right.some((item) => item.id === pair[1])); break; }
    case "ordering": { const c = config as OrderingConfig; valid = selected.length === c.items.length && selected.every((value) => c.items.some((item) => item.id === value)); break; }
    case "geometry": {
      const c = config as GeometryConfig;
      if (c.mode === "point") valid = selected.length === 1 && cell(selected[0], c.columns, c.rows);
      if (c.mode === "select") valid = selected.length === 1 && c.objects!.some((item) => item.id === selected[0]);
      if (c.mode === "connect") valid = selected.every((edge) => { const pair = edge.split(":"); return pair.length === 2 && pair[0] < pair[1] && pair.every((value) => c.points!.some((point) => point.id === value)); });
    }
  }
  return valid ? null : "Завершите ответ, используя объекты задания.";
}

export function defaultVisualConfig(kind: string): Record<string, unknown> {
  switch (kind) {
    case "number_line": return { min: -5, max: 5, step: 1 };
    case "grid_select": return { rows: 4, columns: 4 };
    case "matching": return { left: [{ id: "a", label: "2 + 2" }, { id: "b", label: "3 + 3" }], right: [{ id: "x", label: "4" }, { id: "y", label: "6" }] };
    case "ordering": return { items: [{ id: "a", label: "3" }, { id: "b", label: "1" }, { id: "c", label: "2" }] };
    case "geometry": return { mode: "point", rows: 5, columns: 5 };
    default: return {};
  }
}
