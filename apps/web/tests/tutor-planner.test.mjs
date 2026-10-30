import test from "node:test";
import assert from "node:assert/strict";
import {clock,contactURL,incomeModel,overlaps,planSummary} from "../features/tutor-planner/calculations.ts";

const plan = () => ({revision:0,students:[{id:"a",name:"Student",subject:"Math",price_kopecks:120000,duration:60,contact:""}],lessons:[],windows:[],calculator:{goal_kopecks:10000000,price_kopecks:120000,lessons_per_student:2,duration:60,weeks:4,people:1,preparation_hours:0}});
test("weekly totals exclude pause; touching intervals do not overlap",()=>{
 const p=plan();const a={id:"1",student_id:"a",weekday:1,start_minute:540,duration:60,paused:false};
 p.lessons=[a,{...a,id:"2",start_minute:600},{...a,id:"3",start_minute:570},{...a,id:"4",start_minute:550,paused:true}];
 assert.equal(overlaps(a,p.lessons[1]),false);assert.equal(overlaps(a,p.lessons[2]),true);assert.equal(overlaps(a,p.lessons[3]),false);
 const r=planSummary(p);assert.equal(r.active.length,3);assert.equal(r.conflictIDs.size,3);assert.equal(r.revenue,360000);assert.equal(r.monthly,1440000);assert.equal(r.hours,3);assert.equal(r.students,1);assert.equal(r.daily[1].hours,0);assert.equal(r.hourly,120000);
});
test("capacity counts continuous windows separately per weekday and subtracts preparation",()=>{
 const p=plan();p.windows=[{weekday:1,start_minute:540},{weekday:1,start_minute:570},{weekday:1,start_minute:630},{weekday:2,start_minute:540}];
 let r=incomeModel(p);assert.equal(r.workingHours,2);assert.equal(r.capacity,1);assert.equal(r.groups,0);assert.equal(r.places,0);assert.equal(r.minimumSubscription,null);assert.equal(r.required,11);assert.equal(r.extraHours,21);
 p.calculator.duration=30;r=incomeModel(p);assert.equal(r.capacity,4);assert.equal(r.groups,2);assert.equal(r.places,2);
 p.calculator.preparation_hours=1.5;r=incomeModel(p);assert.equal(r.capacity,1);assert.equal(r.paidHours,0.5);
 p.calculator.preparation_hours=4;r=incomeModel(p);assert.equal(r.capacity,0);assert.equal(r.status,"Нет платных часов");assert.equal(r.hourly,null);
});
test("income matches workbook formats, fractional weeks, rounding and zero load",()=>{
 const p=plan();for(let i=0;i<8;i++)p.windows.push({weekday:1,start_minute:540+i*30});
 p.calculator.weeks=4.33;p.calculator.people=2;
 const r=incomeModel(p);assert.equal(r.capacity,4);assert.equal(r.groups,2);assert.equal(r.places,4);assert.equal(r.required,10);assert.equal(r.subscription,1039200);assert.equal(r.revenue,4156800);assert.equal(r.spare,-6);assert.equal(r.minimumSubscription,2500000);assert.deepEqual(r.formats.map(f=>f.places),[2,4,10,20,40]);
 p.windows=[];assert.equal(incomeModel(p).status,"Нет рабочих окон");assert.equal(planSummary(p).perStudent,null);assert.equal(planSummary(p).hourly,null);
 for(const key of ["weeks","price_kopecks","duration","lessons_per_student","people"]){const invalid=plan();invalid.calculator[key]=0;assert.equal(incomeModel(invalid),null);}
 const invalid=plan();invalid.calculator.weeks=NaN;assert.equal(incomeModel(invalid),null);
 const small=plan();small.calculator.goal_kopecks=1;small.calculator.price_kopecks=1000000000;small.calculator.weeks=6;small.calculator.lessons_per_student=21;assert.equal(incomeModel(small).required,1);
 small.calculator.goal_kopecks=0;assert.equal(incomeModel(small).required,0);
});
test("contacts are safe links; clock supports minute-level lessons",()=>{
 assert.equal(contactURL("@student_test"),"https://t.me/student_test");assert.equal(contactURL("https://example.test"),"https://example.test/");
 for(const v of ["javascript:alert(1)","data:text/html,x","https://user:pass@example.test","@a",""])assert.equal(contactURL(v),null);
 assert.equal(clock(545),"09:05");assert.equal(clock(1320),"22:00");
});
