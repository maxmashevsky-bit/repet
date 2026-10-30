export type CalendarInterval = { id: string; day: string; start: number; duration: number };
export type CalendarPosition = { lane: number; lanes: number; height: number };

export function calendarPositions(items: CalendarInterval[]): Map<string, CalendarPosition> {
  const result = new Map<string, CalendarPosition>();
  const days = new Map<string, CalendarInterval[]>();
  for (const item of items) {
    const day = days.get(item.day) ?? [];
    day.push(item);
    days.set(item.day, day);
  }
  for (const day of days.values()) {
    const ends: number[] = [];
    let group: { id: string; lane: number; height: number }[] = [], groupEnd = -1;
    function flush() {
      for (const item of group) result.set(item.id, { lane: item.lane, lanes: ends.length, height: item.height });
      group = [];
      ends.length = 0;
    }
    for (const item of [...day].sort((a, b) => a.start - b.start || a.id.localeCompare(b.id))) {
      if (item.start >= groupEnd) { flush(); groupEnd = -1; }
      // Short events still need two readable lines, so use their visual interval for collisions.
      const height = Math.max(48, item.duration * 56 / 60), end = item.start + height * 60 / 56;
      let lane = ends.findIndex(value => value <= item.start);
      if (lane < 0) lane = ends.length;
      ends[lane] = end;
      groupEnd = Math.max(groupEnd, end);
      group.push({ id: item.id, lane, height });
    }
    flush();
  }
  return result;
}
