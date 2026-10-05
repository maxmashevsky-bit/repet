export function formatDateTime(value: string, timeZone = "Europe/Moscow"): string {
  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone
  }).format(new Date(value));
}

export function formatTime(value: string, timeZone = "Europe/Moscow"): string {
  return new Intl.DateTimeFormat("ru-RU", {
    hour: "2-digit",
    minute: "2-digit",
    timeZone
  }).format(new Date(value));
}

export function dateInTimeZone(value: string, timeZone: string): string {
  const parts = new Intl.DateTimeFormat("en-US", { timeZone, year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(new Date(value));
  const part = (type: string) => parts.find((item) => item.type === type)?.value ?? "";
  return `${part("year")}-${part("month")}-${part("day")}`;
}

export function timeInTimeZone(value: string, timeZone: string): { hour: number; minute: number } {
  const parts = new Intl.DateTimeFormat("en-US", { timeZone, hour: "2-digit", minute: "2-digit", hourCycle: "h23" }).formatToParts(new Date(value));
  const part = (type: string) => Number(parts.find((item) => item.type === type)?.value ?? 0);
  return { hour: part("hour"), minute: part("minute") };
}

export function todayISO(timeZone = "Europe/Moscow"): string {
  return dateInTimeZone(new Date().toISOString(), timeZone);
}
