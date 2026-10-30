import type { AnalyticsMetric, AnalyticsQuery, AnalyticsReport } from "../../shared/api/analytics";
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
function fail(): never { throw new Error("Не удалось проверить данные успеваемости."); }
function object(v: unknown): Record<string, unknown> { if (!v || typeof v !== "object" || Array.isArray(v)) return fail(); return v as Record<string, unknown>; }
function shape(v: unknown, keys: string) { const o = object(v), names = keys.split(" "); if (Object.keys(o).some(k => !names.includes(k)) || names.some(k => !(k in o))) fail(); return o; }
function count(v: unknown): number { if (typeof v !== "number" || !Number.isSafeInteger(v) || v < 0) return fail(); return v; }
function text(v: unknown): string { if (typeof v !== "string" || v.length > 2000) return fail(); return v; }
function id(v: unknown) { if (!uuid.test(text(v))) fail(); }
function key(v: unknown) { if (!/^[a-f0-9]{32}$/.test(text(v))) fail(); }
function date(v: unknown) { if (!Number.isFinite(Date.parse(text(v)))) fail(); }
function percent(v: unknown, signed = false) { if (v !== null && (typeof v !== "number" || !Number.isFinite(v) || v < (signed ? -100 : 0) || v > 100)) fail(); }
function array(v: unknown, max: number): unknown[] { if (!Array.isArray(v) || v.length > max) return fail(); return v; }
const metricKeys = "attempts graded correct incorrect pending mistakes hints hinted accuracy hint_rate";
function metric(o: Record<string, unknown>) {
  for (const k of metricKeys.split(" ").slice(0, 8)) count(o[k]); percent(o.accuracy); percent(o.hint_rate);
  if (Number(o.correct) + Number(o.incorrect) !== o.graded || Number(o.graded) + Number(o.pending) > Number(o.attempts) || Number(o.hinted) > Number(o.graded) + Number(o.pending)) fail();
  const expected = o.graded ? Math.round(Number(o.correct) * 1000 / Number(o.graded)) / 10 : null;
  if (o.accuracy !== expected) fail();
}
function page(v: unknown, check: (v: unknown) => void) { const o = shape(v, "items total limit offset"); count(o.total); count(o.offset); if (o.limit !== 20) fail(); const items = array(o.items, 20); items.forEach(check); if (items.length > Number(o.total)) fail(); }
export function parseAnalyticsReport(value: unknown): AnalyticsReport {
  const r = shape(value, "student from to generated_at summary homework topics timeline exams recommendations");
  const student = shape(r.student, "id display_name"); id(student.id); text(student.display_name); date(r.from); date(r.to); date(r.generated_at);
  metric(shape(r.summary, metricKeys));
  const hw = shape(r.homework, "total done submitted overdue"); Object.values(hw).forEach(count); if (Number(hw.done) + Number(hw.submitted) + Number(hw.overdue) > Number(hw.total)) fail();
  page(r.topics, v => { const o = shape(v, `key topic subtopic ${metricKeys} level previous_graded recent_graded change`); key(o.key); text(o.topic); text(o.subtopic); metric(o); count(o.previous_graded); count(o.recent_graded); percent(o.change, true); if (!["insufficient", "practice", "developing", "secure"].includes(text(o.level))) fail(); if ((Number(o.previous_graded) < 5 || Number(o.recent_graded) < 5) && o.change !== null) fail(); });
  array(r.timeline, 366).forEach(v => { const o = shape(v, "day attempts graded correct accuracy"); date(o.day); count(o.attempts); count(o.graded); count(o.correct); percent(o.accuracy); if (Number(o.correct) > Number(o.graded) || Number(o.graded) > Number(o.attempts)) fail(); });
  page(r.exams, v => { const o = shape(v, "id title started_at completed_at points max_points pending"); id(o.id); text(o.title); date(o.started_at); count(o.max_points); count(o.pending); if (o.completed_at === null) { if (o.points !== null || o.pending !== 0) fail(); } else { date(o.completed_at); count(o.points); if (Number(o.points) > Number(o.max_points)) fail(); } });
  array(r.recommendations, 5).forEach(v => { const o = shape(v, "topic_key topic subtopic run_id lesson_id href sample errors reason exercises"); key(o.topic_key); text(o.topic); text(o.subtopic); id(o.run_id); id(o.lesson_id); count(o.sample); count(o.errors); if (!Number(o.errors) || Number(o.errors) > Number(o.sample) || !/^\/tasks\/(?:library\/|study\/|review\/)?[a-f0-9-]{36}$/i.test(text(o.href)) || !["few_observations", "weak_topic", "recent_errors"].includes(text(o.reason))) fail(); array(o.exercises,20).forEach(v=>{const e=shape(v,"exercise_id prompt attempts mistakes");id(e.exercise_id);if(typeof e.prompt!=="string"||e.prompt.length>20000)fail();count(e.attempts);count(e.mistakes);if(Number(e.mistakes)>Number(e.attempts))fail();}); });
  return value as AnalyticsReport;
}
export function analyticsRange(days: number, now = new Date()): AnalyticsQuery {
  const to = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate())), from = new Date(to);
  from.setUTCDate(from.getUTCDate() - days + 1);
  return { from: from.toISOString().slice(0, 10), to: to.toISOString().slice(0, 10), topic_offset: 0, exam_offset: 0 };
}
export function percentage(value: number | null) { return value === null ? "Нет оценок" : `${value.toLocaleString("ru-RU", { maximumFractionDigits: 1 })}%`; }
export function metricSample(m: AnalyticsMetric) { return `${m.correct} из ${m.graded} оценённых ответов`; }
export { metric as validateAnalyticsMetric, metricKeys as analyticsMetricKeys };
