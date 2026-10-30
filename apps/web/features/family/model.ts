import type { FamilyQuery,FamilyReport,GuardianLink,GuardianNote } from "../../shared/api/family";
import type { CatalogPage } from "../../shared/api/catalog";
import { analyticsMetricKeys,validateAnalyticsMetric } from "../learning-analytics/model.ts";
const uuid=/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
function fail():never{throw new Error("Не удалось проверить данные семейного кабинета.");}
function shape(v:unknown,keys:string){if(!v||typeof v!=="object"||Array.isArray(v))return fail();const o=v as Record<string,unknown>,names=keys.split(" ");if(Object.keys(o).some(k=>!names.includes(k))||names.some(k=>!(k in o)))fail();return o;}
function text(v:unknown,max=2000){if(typeof v!=="string"||v.length>max)fail();return v as string;}
function id(v:unknown){if(!uuid.test(text(v)))fail();}
function count(v:unknown){if(typeof v!=="number"||!Number.isSafeInteger(v)||v<0)fail();}
function date(v:unknown){if(!Number.isFinite(Date.parse(text(v))))fail();}
function nullableDate(v:unknown){if(v!==null)date(v);}
function choice(v:unknown,values:string[]){if(!values.includes(text(v)))fail();}
function bool(v:unknown){if(typeof v!=="boolean")fail();}
function page<T>(v:unknown,check:(v:unknown)=>void):CatalogPage<T>{const o=shape(v,"items total limit offset");count(o.total);count(o.offset);if(o.limit!==20||!Array.isArray(o.items)||o.items.length>20||o.items.length>Number(o.total))fail();(o.items as unknown[]).forEach(check);return v as CatalogPage<T>;}
export function parseGuardianLink(v:unknown):GuardianLink{const o=shape(v,"id student_id student_name guardian_email guardian_id status created_at expires_at accepted_at revoked_at");id(o.id);id(o.student_id);text(o.student_name);text(o.guardian_email);if(o.guardian_id!==null)id(o.guardian_id);choice(o.status,["pending","active","revoked"]);date(o.created_at);date(o.expires_at);nullableDate(o.accepted_at);nullableDate(o.revoked_at);if(o.status==="active"&&(o.guardian_id===null||o.accepted_at===null)||o.status==="revoked"&&o.revoked_at===null)fail();return v as GuardianLink;}
export const parseGuardianLinks=(v:unknown)=>page<GuardianLink>(v,parseGuardianLink);
export function parseGuardianInvite(v:unknown){const o=shape(v,"link accept_url");const link=parseGuardianLink(o.link),url=text(o.accept_url);if(!/^\/family-invite#token=[A-Za-z0-9_-]{32,128}$/.test(url))fail();return{link,accept_url:url};}
export function parseGuardianNote(v:unknown):GuardianNote{const o=shape(v,"id tutor_name body created_at");id(o.id);text(o.tutor_name);text(o.body,4000);date(o.created_at);return v as GuardianNote;}
export const parseGuardianNotes=(v:unknown)=>page<GuardianNote>(v,parseGuardianNote);
export function familyRange(now=new Date()):FamilyQuery{const day=new Date(Date.UTC(now.getUTCFullYear(),now.getUTCMonth(),now.getUTCDate())),from=new Date(day),to=new Date(day);from.setUTCDate(from.getUTCDate()-29);to.setUTCDate(to.getUTCDate()+30);return{from:from.toISOString().slice(0,10),to:to.toISOString().slice(0,10),topic_offset:0,exam_offset:0,lesson_offset:0,assignment_offset:0,note_offset:0,event_offset:0,result_offset:0};}
export function parseFamilyReport(v:unknown):FamilyReport{
 const r=shape(v,"student from to generated_at summary topics timeline exams lessons assignments notes events results unread"),s=shape(r.student,"id display_name");id(s.id);text(s.display_name);date(r.from);date(r.to);date(r.generated_at);count(r.unread);validateAnalyticsMetric(shape(r.summary,analyticsMetricKeys));
 page(r.topics,v=>{const o=shape(v,`key topic subtopic ${analyticsMetricKeys} level previous_graded recent_graded change`);if(!/^[a-f0-9]{32}$/.test(text(o.key)))fail();text(o.topic);text(o.subtopic);validateAnalyticsMetric(o);choice(o.level,["insufficient","practice","developing","secure"]);count(o.previous_graded);count(o.recent_graded);if(o.change!==null)fail();});
 if(!Array.isArray(r.timeline)||r.timeline.length>366)fail();(r.timeline as unknown[]).forEach(v=>{const o=shape(v,"day attempts graded correct accuracy");date(o.day);count(o.attempts);count(o.graded);count(o.correct);if(Number(o.correct)>Number(o.graded)||Number(o.graded)>Number(o.attempts)||o.accuracy!==(o.graded?Math.round(Number(o.correct)*1000/Number(o.graded))/10:null))fail();});
 page(r.exams,v=>{const o=shape(v,"id title started_at completed_at points max_points pending");id(o.id);text(o.title);date(o.started_at);date(o.completed_at);count(o.points);count(o.max_points);count(o.pending);if(Number(o.points)>Number(o.max_points))fail();});
 page(r.lessons,v=>{const o=shape(v,"id title tutor_name starts_at ends_at status");id(o.id);text(o.title);text(o.tutor_name);date(o.starts_at);date(o.ends_at);choice(o.status,["scheduled","completed","cancelled"]);});
 page(r.assignments,v=>{const o=shape(v,"id title tutor_name subject topic status due_at score max_score");id(o.id);text(o.title);text(o.tutor_name);text(o.subject);text(o.topic);date(o.due_at);choice(o.status,["assigned","submitted","revision","done","overdue"]);count(o.max_score);if(o.score!==null){count(o.score);if(Number(o.score)>Number(o.max_score))fail();}});
 page(r.notes,parseGuardianNote);page(r.events,v=>{const o=shape(v,"id kind created_at unread");id(o.id);choice(o.kind,["lesson","assignment","submitted","graded","completed","reviewed","note"]);date(o.created_at);bool(o.unread);});
 page(r.results,v=>{const o=shape(v,"id lesson_title prompt status attempts created_at");id(o.id);text(o.lesson_title);text(o.prompt,20000);choice(o.status,["correct","incorrect","needs_review"]);count(o.attempts);if(!o.attempts)fail();date(o.created_at);});
 return v as FamilyReport;
}
