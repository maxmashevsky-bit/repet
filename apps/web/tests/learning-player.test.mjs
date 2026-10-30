import test from "node:test";
import assert from "node:assert/strict";
import {parseExample,playerSteps,canVisit,playerProgress} from "../features/learning-player/model.ts";
import {parseLearningState} from "../features/exercise-engine/learning-schema.ts";
const id="00000000-0000-4000-8000-000000000001",second="00000000-0000-4000-8000-000000000002",date="2026-10-20T00:00:00Z";
const example={title:"Two plus one",explanation:"Move right",min:0,max:10,steps:[{text:"Two",value:2},{text:"Three",value:3}]};
function fixture(){return {attempt:{id,lesson_id:id,version_id:id,number:1,status:"in_progress",started_at:date,completed_at:null,elapsed_seconds:0,total_exercises:2,answered_exercises:0,correct_exercises:0,pending_review:0,incorrect_attempts:0,mode:"practice",player_step:"theory",theory_seen:false,example_seen:false},version:{id,number:1,created_at:date,published_at:date,example,exercises:[id,second].map(id=>({id,kind:"numeric",prompt:"Seven",options:[],difficulty:1,position:0}))},answers:[],hints:[]};}
test("worked examples have bounded strict author data without fabricated steps",()=>{
 assert.deepEqual(parseExample(example),example);assert.equal(parseExample(null),undefined);
 for(const bad of [{...example,max:2000},{...example,steps:[]},{...example,steps:[{text:"A",value:2}]},{...example,answer:"secret"},{...example,steps:[{text:"A",value:99},{text:"B",value:1}]}])assert.throws(()=>parseExample(bad));
});
test("player navigation unlocks in order and progress reaches 100 only on completion",()=>{
 const s=fixture();assert.equal(playerSteps(s.version).length,5);assert.equal(canVisit(s,"example"),true);assert.equal(canVisit(s,`exercise:${id}`),false);assert.equal(canVisit(s,"results"),false);
 s.attempt.theory_seen=true;s.attempt.player_step="example";assert.equal(canVisit(s,`exercise:${id}`),true);assert.equal(canVisit(s,`exercise:${second}`),false);
 s.attempt.example_seen=true;s.answers=[{exercise_id:id}];assert.equal(canVisit(s,`exercise:${second}`),true);assert.equal(playerProgress(s).percent,60);
 s.answers.push({exercise_id:second});s.attempt.status="completed";assert.equal(playerProgress(s).percent,100);assert.equal(canVisit(s,"results"),true);
});
test("exam projections reject leaked correctness, feedback, counters and invalid steps",()=>{
 const state=fixture();state.attempt.mode="exam";state.attempt.answered_exercises=1;state.answers=[{id,exercise_id:id,version_id:id,version:1,student_id:id,student_name:"Student",kind:"numeric",prompt:"Seven",answer:{value:"7"},status:"submitted",feedback:"",created_at:date,reviewed_at:null,learning_attempt_id:id,attempt_number:1,hints_used:0,elapsed_seconds:0}];
 assert.equal(parseLearningState(state).answers[0].status,"submitted");
 for(const change of [s=>{s.answers[0].status="correct";s.attempt.correct_exercises=1;},s=>{s.answers[0].feedback="secret";},s=>{s.attempt.player_step="fake";}]){const copy=structuredClone(state);change(copy);assert.throws(()=>parseLearningState(copy));}
});
