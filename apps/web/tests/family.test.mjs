import test from "node:test";
import assert from "node:assert/strict";
import { familyRange,parseGuardianLink,parseGuardianInvite,parseFamilyReport } from "../features/family/model.ts";
const id="00000000-0000-4000-8000-000000000001",at="2026-10-30T00:00:00Z",page=()=>({items:[],total:0,limit:20,offset:0});
const link=()=>({id,student_id:id,student_name:"Ученик",guardian_email:"parent@example.test",guardian_id:null,status:"pending",created_at:at,expires_at:"2026-11-06T00:00:00Z",accepted_at:null,revoked_at:null});
const report=()=>({student:{id,display_name:"Ученик"},from:"2026-10-01",to:"2026-11-29",generated_at:at,summary:{attempts:0,graded:0,correct:0,incorrect:0,pending:0,mistakes:0,hints:0,hinted:0,accuracy:null,hint_rate:null},topics:page(),timeline:[],exams:page(),lessons:page(),assignments:page(),notes:page(),events:page(),results:page(),unread:0});
test("family consent DTOs are closed; secret invitation is a local fragment, not a redirect or query",()=>{
 assert.equal(parseGuardianLink(link()).status,"pending");assert.throws(()=>parseGuardianLink({...link(),token_hash:"private"}));assert.throws(()=>parseGuardianLink({...link(),status:"active"}));
 assert.equal(parseGuardianInvite({link:link(),accept_url:"/family-invite#token="+"a".repeat(43)}).link.id,id);
 for(const accept_url of["https://evil.test/#token="+"a".repeat(43),"//evil.test/#token="+"a".repeat(43),"/family-invite?token="+"a".repeat(43),"/family-invite#token=short"])assert.throws(()=>parseGuardianInvite({link:link(),accept_url}));
});
test("family report rejects private answers/messages, active exam scores and inconsistent grading",()=>{
 assert.equal(parseFamilyReport(report()).unread,0);for(const field of["messages","feedback","answer_keys","role"])assert.throws(()=>parseFamilyReport({...report(),[field]:[]}));
 const wrong=report();wrong.summary.correct=1;assert.throws(()=>parseFamilyReport(wrong));
 const active=report();active.exams.items=[{id,title:"Пробник",started_at:at,completed_at:null,points:1,max_points:1,pending:0}];active.exams.total=1;assert.throws(()=>parseFamilyReport(active));
 const solved=report();solved.results={items:[{id,lesson_title:"Урок",prompt:"3 + 4",status:"correct",attempts:1,created_at:at,answer:{value:"7"}}],total:1,limit:20,offset:0};assert.throws(()=>parseFamilyReport(solved));
 const result=familyRange(new Date("2027-01-02T23:00:00Z"));assert.equal(result.from,"2026-12-04");assert.equal(result.to,"2027-02-01");assert.equal(result.event_offset,0);
});
