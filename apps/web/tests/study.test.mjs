import test from "node:test";
import assert from "node:assert/strict";
import { parseStudySession,remainingSeconds,clockLabel,repeatExerciseCount } from "../features/study/model.ts";
const id="00000000-0000-4000-8000-000000000001",date="2026-10-30T00:00:00Z";
function fixture(){return {id,specification:{id,family_id:id,version:1,author_id:id,author_name:"Teacher",collection_id:id,title:"Own mock",direction:"oge",year:2026,duration_seconds:60,instructions:"",slots:[{number:7,count:1,points:2}],topics:["Numbers"],exercise_count:1,status:"published"},seed:"fixed-seed",deadline_at:"2026-10-30T00:01:00Z",server_time:date,points:0,max_points:2,state:{attempt:{id,lesson_id:id,version_id:id,number:1,status:"in_progress",started_at:date,completed_at:null,elapsed_seconds:0,total_exercises:1,answered_exercises:0,correct_exercises:0,pending_review:0,incorrect_attempts:0,mode:"exam",player_step:`exercise:${id}`,theory_seen:true,example_seen:true},version:{id,number:1,created_at:date,published_at:date,exercises:[{id,kind:"numeric",prompt:"Seven",options:[],difficulty:1,position:0,config:{}}]},answers:[],hints:[]}};}
test("study runtime validation rejects leaked scores, forged variants and invalid server clocks",()=>{
 assert.equal(parseStudySession(fixture()).state.attempt.mode,"exam");
 for(const mutate of [s=>s.points=1,s=>s.deadline_at=null,s=>s.server_time="bad",s=>s.max_points=-1,s=>s.answer_key="secret",s=>s.specification.slots.push(s.specification.slots[0]),s=>s.state.version.exercises[0].answer="7",s=>s.state.attempt.assignment_id=id]){const s=fixture();mutate(s);assert.throws(()=>parseStudySession(s));}
});
test("countdown uses a server clock and monotonic elapsed time with exact zero boundaries",()=>{
 const deadline="2026-10-30T00:01:00Z";assert.equal(remainingSeconds(deadline,date,0),60);assert.equal(remainingSeconds(deadline,date,59999),1);assert.equal(remainingSeconds(deadline,date,60000),0);assert.equal(remainingSeconds(deadline,date,999999),0);assert.equal(remainingSeconds(deadline,date,-1000),60);assert.equal(clockLabel(3661),"01:01:01");assert.equal(clockLabel(-1),"00:00:00");
});
test("repeat includes incorrect and blank questions but not ungraded manual solutions",()=>{
 const s=fixture();assert.equal(repeatExerciseCount(s),1);s.state.answers=[{exercise_id:id,status:"needs_review"}];assert.equal(repeatExerciseCount(s),0);s.state.answers[0].status="incorrect";assert.equal(repeatExerciseCount(s),1);s.state.answers[0].status="correct";assert.equal(repeatExerciseCount(s),0);
});
