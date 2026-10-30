"use client";

import { useRef, useState, type KeyboardEvent } from "react";
import { Icon } from "../ui/Icon";

export function GridBoard({ rows, columns, selected, pick, disabled, coordinates = false }: { rows: number; columns: number; selected: string[]; pick: (id: string) => void; disabled: boolean; coordinates?: boolean }) {
  const [focus, setFocus] = useState(0);
  const buttons = useRef<(HTMLButtonElement | null)[]>([]);
  function navigate(event: KeyboardEvent, index: number) {
    let next = index;
    switch (event.key) {
      case "ArrowLeft": next = index % columns > 0 ? index - 1 : index; break;
      case "ArrowRight": next = index % columns < columns - 1 ? index + 1 : index; break;
      case "ArrowUp": next = index >= columns ? index - columns : index; break;
      case "ArrowDown": next = index + columns < rows * columns ? index + columns : index; break;
      case "Home": next = event.ctrlKey ? 0 : Math.floor(index / columns) * columns; break;
      case "End": next = event.ctrlKey ? rows * columns - 1 : Math.floor(index / columns) * columns + columns - 1; break;
      default: return;
    }
    event.preventDefault(); setFocus(next); buttons.current[next]?.focus();
  }
  return <div className={`visual-grid ${coordinates ? "visual-coordinate-grid" : ""}`} role="grid" aria-label={coordinates ? "Координатная сетка" : "Поле клеток"} aria-rowcount={rows} aria-colcount={columns} aria-readonly={disabled}>
    {Array.from({ length: rows }, (_, row) => <div key={row} role="row" className="visual-grid-row" style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}>{Array.from({ length: columns }, (_, column) => {
      const index = row * columns + column;
      const value = coordinates ? `${column}:${rows - row - 1}` : `${row}:${column}`;
      const active = selected.includes(value);
      return <button key={column} ref={(node) => { buttons.current[index] = node; }} type="button" role="gridcell" aria-selected={active} aria-label={coordinates ? `Точка (${column}; ${rows - row - 1})` : `Клетка ${row + 1}, ${column + 1}`} className={active ? "visual-cell selected" : "visual-cell"} tabIndex={index === focus ? 0 : -1} disabled={disabled} onFocus={() => setFocus(index)} onKeyDown={(event) => navigate(event, index)} onClick={() => { setFocus(index); pick(value); }}>
        {coordinates ? <><span className="visual-cell-coordinate">{column}; {rows - row - 1}</span>{active ? <span className="visual-point-dot" aria-hidden="true" /> : null}</> : <>{active ? <Icon name="check" size={20} /> : null}<span className="visual-cell-coordinate">{row + 1}:{column + 1}</span></>}
      </button>;
    })}</div>)}
  </div>;
}
