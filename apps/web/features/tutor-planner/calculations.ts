import type { PlanLesson, TutorPlan } from "../../shared/api/tutor-planner.ts";

export const days = ["Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"];
export const slots = Array.from({ length: 26 }, (_, i) => 540 + i * 30);
export function clock(minutes: number) { return `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`; }
export function money(kopecks: number | null) { return kopecks === null ? "—" : new Intl.NumberFormat("ru-RU", { style: "currency", currency: "RUB", maximumFractionDigits: kopecks % 100 ? 2 : 0 }).format(kopecks / 100); }
export function contactURL(contact: string) {
  if (/^@[a-zA-Z0-9_]{5,32}$/.test(contact)) return `https://t.me/${contact.slice(1)}`;
  try { const u = new URL(contact); return ["http:", "https:"].includes(u.protocol) && !u.username && !u.password ? u.href : null; } catch { return null; }
}
export function overlaps(a: PlanLesson, b: PlanLesson) {
  return a.id !== b.id && !a.paused && !b.paused && a.weekday === b.weekday && a.start_minute < b.start_minute + b.duration && b.start_minute < a.start_minute + a.duration;
}
export function planSummary(plan: TutorPlan) {
  const active = plan.lessons.filter(l => !l.paused);
  const conflictIDs = new Set(active.filter(a => active.some(b => overlaps(a, b))).map(l => l.id));
  const daily = days.map((_, i) => {
    const lessons = active.filter(l => l.weekday === i + 1);
    return { lessons: lessons.length, hours: lessons.reduce((n, l) => n + l.duration, 0) / 60, revenue: lessons.reduce((n, l) => n + (plan.students.find(s => s.id === l.student_id)?.price_kopecks ?? 0), 0) };
  });
  const hours = daily.reduce((n, d) => n + d.hours, 0), revenue = daily.reduce((n, d) => n + d.revenue, 0), students = new Set(active.map(l => l.student_id)).size;
  return { active, conflictIDs, daily, hours, revenue, students, monthly: revenue * plan.calculator.weeks, hourly: hours ? revenue / hours : null, perStudent: students ? revenue * plan.calculator.weeks / students : null };
}

export function incomeModel(plan: TutorPlan) {
  const c = plan.calculator;
  if (![c.weeks, c.price_kopecks, c.duration, c.lessons_per_student, c.people].every(v => Number.isFinite(v) && v > 0) || !Number.isFinite(c.goal_kopecks) || c.goal_kopecks < 0 || !Number.isFinite(c.preparation_hours) || c.preparation_hours < 0) return null;
  const dayCapacity = days.map((_, index) => {
    const selected = [...new Set(plan.windows.filter(w => w.weekday === index + 1).map(w => w.start_minute))].sort((a, b) => a - b);
    let run = 0, capacity = 0;
    for (let i = 0; i < selected.length; i++) {
      run += 30;
      if (selected[i + 1] !== selected[i] + 30) { capacity += Math.floor(run / c.duration); run = 0; }
    }
    return capacity;
  });
  const workingHours = plan.windows.length / 2, paidHours = Math.max(0, workingHours - c.preparation_hours);
  const capacity = Math.min(dayCapacity.reduce((a, b) => a + b, 0), Math.floor(paidHours * 60 / c.duration));
  const groups = Math.floor(capacity / c.lessons_per_student), places = groups * c.people;
  const subscription = c.price_kopecks * c.lessons_per_student * c.weeks;
  // Relative tolerance avoids rounding an exact whole pupil up, without erasing tiny positive goals.
  const ratio = c.goal_kopecks / subscription;
  const required = Math.ceil(ratio - Number.EPSILON * Math.abs(ratio) * 2);
  const hourly = paidHours ? c.goal_kopecks / (paidHours * c.weeks) : null;
  return {
    dayCapacity, workingHours, paidHours, capacity, groups, places, subscription, required,
    spare: places - required, revenue: places * subscription, gap: places * subscription - c.goal_kopecks,
    minimumSubscription: places ? Math.ceil(c.goal_kopecks / places / 100) * 100 : null,
    hourly, studentHourly: hourly === null ? null : hourly / c.people,
    load: groups ? Math.ceil(required / c.people) / groups : null,
    extraHours: Math.max(0, (Math.ceil(required / c.people) * c.lessons_per_student - capacity) * c.duration / 60),
    status: !workingHours ? "Нет рабочих окон" : !paidHours ? "Нет платных часов" : required <= places ? "Цель достижима" : "Нужно больше часов или выше цена",
    formats: [1, 2, 5, 10, 20].map(people => ({ people, groups, places: groups * people, hourly: hourly === null ? null : hourly / people, minimumSubscription: groups ? Math.ceil(c.goal_kopecks / (groups * people) / 100) * 100 : null, revenue: groups * people * subscription, gap: groups * people * subscription - c.goal_kopecks }))
  };
}
