import test from "node:test";
import assert from "node:assert/strict";
import { parseAnalyticsReport,analyticsRange,percentage,metricSample } from "../features/learning-analytics/model.ts";
const id="12345678-1234-4234-8234-123456789abc",key="a".repeat(32);
function fixture(){return {student:{id,display_name:"Ученица"},from:"2026-10-01",to:"2026-10-30",generated_at:"2026-10-30T12:00:00Z",summary:{attempts:22,graded:20,correct:15,incorrect:5,pending:2,mistakes:5,hints:3,hinted:2,accuracy:75,hint_rate:9.1},homework:{total:3,done:1,submitted:1,overdue:1},topics:{items:[],total:0,limit:20,offset:0},timeline:[{day:"2026-10-30",attempts:22,graded:20,correct:15,accuracy:75}],exams:{items:[{id,title:"Пробник",started_at:"2026-10-30T11:00:00Z",completed_at:null,points:null,max_points:7,pending:0}],total:1,limit:20,offset:0},recommendations:[{topic_key:key,topic:"Дроби",subtopic:"",run_id:id,lesson_id:id,href:`/tasks/${id}`,sample:10,errors:5,reason:"weak_topic",exercises:[{exercise_id:id,prompt:"Пример",attempts:1,mistakes:1}]}]};}
test("analytics runtime distinguishes graded and pending, denies leaks and unsafe links",()=>{
 const value=fixture();assert.deepEqual(parseAnalyticsReport(value),value);
 for(const mutate of [v=>v.answer_keys={},v=>v.summary.graded=22,v=>v.summary.accuracy=100,v=>v.summary.pending=-1,v=>v.exams.items[0].points=2,v=>v.recommendations[0].href="https://evil.test",v=>v.recommendations[0].errors=11,v=>v.timeline[0].graded=10,v=>v.homework.overdue=3]){const bad=fixture();mutate(bad);assert.throws(()=>parseAnalyticsReport(bad));}
 assert.equal(metricSample(value.summary),"15 из 20 оценённых ответов");assert.equal(percentage(null),"Нет оценок");
});
test("analytics date ranges use inclusive UTC dates across leap and year boundaries",()=>{
 assert.deepEqual(analyticsRange(30,new Date("2026-01-10T23:00:00Z")),{from:"2025-12-12",to:"2026-01-10",topic_offset:0,exam_offset:0});
 assert.equal(analyticsRange(2,new Date("2024-03-01T00:00:00Z")).from,"2024-02-29");
});
