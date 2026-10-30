import test from "node:test";
import assert from "node:assert/strict";
import { parseLearningState, newerLearningState } from "../features/exercise-engine/learning-schema.ts";

const id="00000000-0000-4000-8000-000000000001";
const date="2026-10-20T10:00:00Z";
function fixture(){return {
 attempt:{id,lesson_id:id,version_id:id,number:1,status:"in_progress",started_at:date,completed_at:null,elapsed_seconds:10,total_exercises:1,answered_exercises:1,correct_exercises:0,pending_review:0,incorrect_attempts:1,mode:"practice",player_step:"theory",theory_seen:false,example_seen:false},
 version:{id,number:1,topic_id:id,title:"Lesson",summary:"",content:"",track:"school",grade_min:1,grade_max:11,difficulty:1,created_at:date,published_at:date,exercises:[{id,kind:"numeric",prompt:"One",options:[],difficulty:1,position:0,config:{},hint_count:1}]},
 answers:[{id,exercise_id:id,version_id:id,version:1,student_id:id,student_name:"Student",kind:"numeric",prompt:"One",answer:{value:"9"},status:"incorrect",feedback:"",created_at:date,reviewed_at:null,learning_attempt_id:id,attempt_number:1,hints_used:1,elapsed_seconds:10}],
 hints:[{exercise_id:id,number:1,text:"A revealed hint",revealed_at:date}]
};}
test("learning state validates run identity, version, counters and revealed hints",()=>{
 const value=fixture();assert.equal(parseLearningState(value).answers[0].attempt_number,1);
 const mutations=[v=>v.attempt.elapsed_seconds=-1,v=>v.attempt.number=0,v=>v.attempt.status="correct",v=>v.attempt.completed_at=date,v=>v.attempt.correct_exercises=2,v=>v.answers[0].learning_attempt_id=null,v=>v.answers[0].attempt_number=0,v=>v.answers[0].hints_used=6,v=>v.version.id="00000000-0000-4000-8000-000000000002",v=>v.version.exercises[0].answer_key="secret",v=>v.hints[0].number=2,v=>v.hints.push(v.hints[0]),v=>v.answers.push(v.answers[0])];
 for(const mutate of mutations){const v=fixture();mutate(v);assert.throws(()=>parseLearningState(v));}
});
test("late progress responses cannot erase answers, hints or completion",()=>{
 const current=parseLearningState(fixture());const stale=structuredClone(current);stale.answers=[];stale.attempt.answered_exercises=0;
 assert.equal(newerLearningState(current,stale),current);
 const noHints=structuredClone(current);noHints.hints=[];assert.equal(newerLearningState(current,noHints),current);
 const finished=structuredClone(current);finished.attempt.completed_at=date;finished.attempt.status="completed";
 assert.equal(newerLearningState(finished,current),finished);
 const fresh=structuredClone(current);fresh.answers[0].attempt_number=2;assert.equal(newerLearningState(current,fresh),fresh);
});
test("assigned progress validates its resource identity without changing ordinary progress",()=>{
 const v=fixture();v.attempt.assignment_id=id;
 assert.equal(parseLearningState(v).attempt.assignment_id,id);
 v.attempt.assignment_id="forged";assert.throws(()=>parseLearningState(v));
 const ordinary=fixture();assert.equal(parseLearningState(ordinary).attempt.assignment_id,undefined);
});
