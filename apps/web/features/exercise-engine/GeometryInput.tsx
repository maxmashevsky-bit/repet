"use client";
import { Select } from "../ui/Select";


import { useId, useState } from "react";
import type { ExerciseInputProps } from "./registry";
import type { GeometryConfig } from "./visual-schema";
import { GridBoard } from "./GridBoard";
import { Icon } from "../ui/Icon";

export function GeometryInput({ exercise, value, onChange, disabled }: ExerciseInputProps) {
  const c = exercise.config as GeometryConfig; const selected = value.selected ?? [];
  const [active, setActive] = useState<string | null>(null);
  const [first, setFirst] = useState(c.points?.[0]?.id ?? ""); const [second, setSecond] = useState(c.points?.[1]?.id ?? "");
  const [announcement, setAnnouncement] = useState(""); const radioName = useId();
  if (c.mode === "point") return <GridBoard rows={c.rows} columns={c.columns} coordinates selected={selected} disabled={disabled} pick={(id) => onChange({ selected: [id] })} />;
  const points = c.points!;
  const xy = (id: string) => { const point = points.find((p) => p.id === id)!; return { x: 32 + point.x / (c.columns - 1) * 336, y: 288 - point.y / (c.rows - 1) * 256 }; };
  function add(a: string, b: string) {
    if (disabled || a === b || !points.some((point) => point.id === a) || !points.some((point) => point.id === b)) return;
    const edge = [a, b].sort().join(":"); if (!selected.includes(edge)) onChange({ selected: [...selected, edge] });
    setActive(null); setAnnouncement(`Отрезок ${a}, ${b} добавлен.`);
  }
  function pick(id: string) { if (disabled) return; if (!active || active === id) setActive(active === id ? null : id); else add(active, id); }
  return <div className="visual-geometry">
    <svg viewBox="0 0 400 320" className="visual-geometry-svg" role="group" aria-label={c.mode === "connect" ? "Соединение точек" : "Геометрические объекты"}>
      {Array.from({ length: c.columns }, (_, i) => <line key={`x${i}`} x1={32 + i / (c.columns - 1) * 336} x2={32 + i / (c.columns - 1) * 336} y1="32" y2="288" className="visual-grid-line" />)}
      {Array.from({ length: c.rows }, (_, i) => <line key={`y${i}`} y1={32 + i / (c.rows - 1) * 256} y2={32 + i / (c.rows - 1) * 256} x1="32" x2="368" className="visual-grid-line" />)}
      {c.mode === "connect" ? selected.map((edge) => { const [a, b] = edge.split(":"); const p = xy(a); const q = xy(b); return <line key={edge} x1={p.x} y1={p.y} x2={q.x} y2={q.y} className="visual-connection" />; }) : c.objects!.map((object) => {
        const positions = object.point_ids.map(xy); const on = selected.includes(object.id); const labelX = positions.reduce((sum, p) => sum + p.x, 0) / positions.length; const labelY = positions.reduce((sum, p) => sum + p.y, 0) / positions.length;
        return <g key={object.id} className={`visual-geometry-object ${on ? "selected" : ""}`} role="button" aria-label={object.label} aria-pressed={on} aria-disabled={disabled} tabIndex={disabled ? -1 : 0} onClick={() => { if (!disabled) onChange({ selected: [object.id] }); }} onKeyDown={(event) => { if (!disabled && (event.key === "Enter" || event.key === " ")) { event.preventDefault(); onChange({ selected: [object.id] }); } }}><title>{object.label}</title>{object.kind === "polygon" ? <polygon points={positions.map((p) => `${p.x},${p.y}`).join(" ")} /> : <><line x1={positions[0].x} y1={positions[0].y} x2={positions[1].x} y2={positions[1].y} className="visual-geometry-hit" /><line x1={positions[0].x} y1={positions[0].y} x2={positions[1].x} y2={positions[1].y} /></>}<text x={labelX} y={labelY - 8} textAnchor="middle">{object.id.slice(0, 4)}</text></g>;
      })}
      {points.map((point) => { const p = xy(point.id); return <g key={point.id} className={`visual-geometry-point ${active === point.id ? "active" : ""}`} role={c.mode === "connect" ? "button" : undefined} aria-label={`${point.label} (${point.x}; ${point.y})`} aria-pressed={c.mode === "connect" ? active === point.id : undefined} aria-disabled={c.mode === "connect" ? disabled : undefined} tabIndex={!disabled && c.mode === "connect" ? 0 : undefined} onClick={() => { if (c.mode === "connect") pick(point.id); }} onKeyDown={(event) => { if (c.mode === "connect" && (event.key === "Enter" || event.key === " ")) { event.preventDefault(); pick(point.id); } }}><title>{point.label}</title>{c.mode === "connect" ? <circle cx={p.x} cy={p.y} r="26" className="visual-geometry-point-hit" /> : null}<circle cx={p.x} cy={p.y} r="6" className="visual-geometry-dot" /><text x={p.x + (point.x === c.columns - 1 ? -10 : 10)} y={p.y - 10} textAnchor={point.x === c.columns - 1 ? "end" : "start"}>{point.label.length <= 4 ? point.label : point.id.slice(0, 4)}</text></g>; })}
    </svg>
    {c.mode === "connect" ? <><div className="visual-segment-controls"><label>Начало отрезка<Select aria-label="Начало отрезка" value={first} disabled={disabled} onChange={(event) => setFirst(event.target.value)}>{points.map((point) => <option key={point.id} value={point.id}>{point.label}</option>)}</Select></label><label>Конец отрезка<Select aria-label="Конец отрезка" value={second} disabled={disabled} onChange={(event) => setSecond(event.target.value)}>{points.map((point) => <option key={point.id} value={point.id}>{point.label}</option>)}</Select></label><button type="button" className="secondary" disabled={disabled || first === second} onClick={() => add(first, second)}><Icon name="plus" size={17} />Добавить отрезок</button></div><p className="visual-announcement" role="status">{announcement}</p><ul className="visual-edge-list">{selected.map((edge) => <li key={edge}><span>{edge.split(":").map((id) => points.find((p) => p.id === id)!.label).join(" · ")}</span><button type="button" className="icon-button" disabled={disabled} aria-label={`Удалить отрезок ${edge}`} title="Удалить отрезок" onClick={() => onChange({ selected: selected.filter((item) => item !== edge) })}><Icon name="close" size={16} /></button></li>)}</ul></> : <fieldset className="engine-options" disabled={disabled}><legend className="sr-only">Геометрический объект</legend>{c.objects!.map((object) => <label key={object.id} className={selected.includes(object.id) ? "engine-option selected" : "engine-option"}><input type="radio" name={radioName} checked={selected.includes(object.id)} onChange={() => onChange({ selected: [object.id] })} /><span>{object.label}</span></label>)}</fieldset>}
  </div>;
}
