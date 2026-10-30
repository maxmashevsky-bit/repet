"use client";
import { Select } from "../ui/Select";


import { useId, useRef, useState } from "react";
import type { ExerciseInputProps } from "./registry";
import type { GridConfig, MatchingConfig, NumberLineConfig, OrderingConfig } from "./visual-schema";
import { lineCoordinate } from "./visual-schema";
import { GridBoard } from "./GridBoard";
import { Icon } from "../ui/Icon";

export function NumberLineInput({ exercise, value, onChange, disabled, invalid, errorID }: ExerciseInputProps) {
  const c = exercise.config as NumberLineConfig; const steps = (c.max - c.min) / c.step;
  const n = lineCoordinate(value.value); const placed = n !== null && n >= c.min && n <= c.max && (n - c.min) % c.step === 0;
  const index = placed ? (n - c.min) / c.step : 0;
  function pick(event: React.PointerEvent<SVGSVGElement>) {
    if (disabled) return;
    const rect = event.currentTarget.getBoundingClientRect();
    const position = (event.clientX - rect.left) / rect.width * 600;
    const next = Math.max(0, Math.min(steps, Math.round((position - 36) / 528 * steps)));
    onChange({ value: String(c.min + next * c.step) });
  }
  return <div className="visual-number-line">
    <svg viewBox="0 0 600 120" className="visual-number-svg" role="img" aria-label={`Числовая прямая от ${c.min} до ${c.max}`} style={{ touchAction: disabled ? "auto" : "none" }} onPointerDown={(event) => { if (!disabled) { event.preventDefault(); event.currentTarget.setPointerCapture(event.pointerId); pick(event); } }} onPointerMove={(event) => { if (event.buttons === 1) pick(event); }}>
      <line x1="30" y1="64" x2="570" y2="64" className="visual-axis" />
      {Array.from({ length: steps + 1 }, (_, i) => <g key={i}><line x1={36 + i / steps * 528} x2={36 + i / steps * 528} y1="57" y2="71" className="visual-axis" />{steps <= 20 || i % Math.ceil(steps / 10) === 0 || i === steps ? <text x={36 + i / steps * 528} y="96" textAnchor="middle">{c.min + i * c.step}</text> : null}</g>)}
      {placed ? <g><circle cx={36 + index / steps * 528} cy="64" r="9" className="visual-dot" /><text x={36 + index / steps * 528} y="37" textAnchor="middle">{n}</text></g> : null}
    </svg>
    <label>Положение точки<input type="range" min={0} max={steps} step={1} value={index} disabled={disabled} aria-valuetext={placed ? String(n) : "Точка не поставлена"} onChange={(event) => onChange({ value: String(c.min + Number(event.target.value) * c.step) })} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); onChange({ value: String(c.min + Number(event.currentTarget.value) * c.step) }); } }} /></label>
    <label>Координата точки<input type="text" inputMode="numeric" aria-label="Координата точки" value={value.value ?? ""} maxLength={128} disabled={disabled} aria-invalid={invalid} aria-describedby={invalid ? errorID : undefined} onChange={(event) => onChange({ value: event.target.value })} /></label>
  </div>;
}

export function GridSelectInput({ exercise, value, onChange, disabled }: ExerciseInputProps) {
  const c = exercise.config as GridConfig; const selected = value.selected ?? [];
  return <GridBoard {...c} selected={selected} disabled={disabled} pick={(id) => onChange({ selected: selected.includes(id) ? selected.filter((item) => item !== id) : [...selected, id] })} />;
}

export function MatchingInput({ exercise, value, onChange, disabled }: ExerciseInputProps) {
  const c = exercise.config as MatchingConfig; const selected = value.selected ?? [];
  const [active, setActive] = useState<string | null>(null);
  function connect(left: string, right: string) {
    const next = selected.filter((pair) => { const [a, b] = pair.split(":"); return a !== left && (!right || b !== right); });
    if (right) next.push(`${left}:${right}`);
    onChange({ selected: next }); setActive(null);
  }
  return <div className="visual-matching">
    <div className="visual-matching-board">
      <svg viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">{selected.map((pair) => { const [left, right] = pair.split(":"); const a = c.left.findIndex((item) => item.id === left); const b = c.right.findIndex((item) => item.id === right); return a < 0 || b < 0 ? null : <line key={pair} x1="40" x2="60" y1={(a + .5) / c.left.length * 100} y2={(b + .5) / c.right.length * 100} />; })}</svg>
      <div className="visual-matching-column">{c.left.map((item) => <button type="button" key={item.id} aria-pressed={active === item.id} disabled={disabled} className={active === item.id ? "secondary visual-match-item active" : "secondary visual-match-item"} onClick={() => setActive(item.id)}>{item.label}</button>)}</div>
      <div className="visual-matching-column">{c.right.map((item) => <button type="button" key={item.id} disabled={disabled || !active} className="secondary visual-match-item" onClick={() => { if (active) connect(active, item.id); }}>{item.label}</button>)}</div>
    </div>
    <div className="visual-matching-alternative">{c.left.map((item) => <label key={item.id}>{item.label}<Select aria-label={`Соответствие для ${item.label}`} disabled={disabled} value={selected.find((pair) => pair.split(":")[0] === item.id)?.split(":")[1] ?? ""} onChange={(event) => connect(item.id, event.target.value)}><option value="">Не выбрано</option>{c.right.map((right) => <option key={right.id} value={right.id}>{right.label}</option>)}</Select></label>)}</div>
  </div>;
}

export function OrderingInput({ exercise, value, onChange, disabled }: ExerciseInputProps) {
  const c = exercise.config as OrderingConfig;
  const chosen = (value.selected ?? []).filter((id, index, values) => c.items.some((item) => item.id === id) && values.indexOf(id) === index);
  const order = [...chosen, ...c.items.filter((item) => !chosen.includes(item.id)).map((item) => item.id)];
  const dragged = useRef<string | null>(null); const list = useRef<HTMLOListElement>(null); const prefix = useId();
  function move(id: string, index: number, focus = false) {
    if (disabled) return;
    const next = order.filter((item) => item !== id); next.splice(Math.max(0, Math.min(order.length - 1, index)), 0, id); onChange({ selected: next });
    if (focus) queueMicrotask(() => list.current?.querySelector<HTMLElement>(`[data-order-id="${id}"]`)?.focus());
  }
  return <ol ref={list} className="visual-ordering" aria-label="Порядок элементов">{order.map((id, index) => {
    const item = c.items.find((entry) => entry.id === id)!;
    return <li key={id} data-order-id={id} tabIndex={disabled ? -1 : 0} draggable={!disabled} aria-label={`${index + 1}. ${item.label}`} onDragStart={(event) => { dragged.current = id; event.dataTransfer.effectAllowed = "move"; event.dataTransfer.setData("text/plain", id); }} onDragEnd={() => { dragged.current = null; }} onDragOver={(event) => { if (!disabled && dragged.current) event.preventDefault(); }} onDrop={(event) => { event.preventDefault(); if (dragged.current) { move(dragged.current, index); dragged.current = null; } }} onKeyDown={(event) => { if (event.key === "ArrowUp" || event.key === "ArrowDown") { event.preventDefault(); move(id, index + (event.key === "ArrowUp" ? -1 : 1), true); } }}>
      <span className="visual-order-number" aria-hidden="true">{index + 1}</span><span id={`${prefix}-${id}`} className="visual-order-label">{item.label}</span><div className="visual-order-controls"><button type="button" className="icon-button" disabled={disabled || index === 0} aria-label={`Переместить вверх: ${item.label}`} title="Переместить вверх" onClick={() => move(id, index - 1)}><Icon name="arrow" className="visual-up" size={18} /></button><button type="button" className="icon-button" disabled={disabled || index === order.length - 1} aria-label={`Переместить вниз: ${item.label}`} title="Переместить вниз" onClick={() => move(id, index + 1)}><Icon name="arrow" className="visual-down" size={18} /></button></div>
    </li>;
  })}</ol>;
}
