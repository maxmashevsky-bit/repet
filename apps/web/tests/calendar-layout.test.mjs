import test from "node:test";
import assert from "node:assert/strict";
import {calendarPositions} from "../features/calendar/layout.ts";
test("calendar separates overlapping and short event labels without mutating data",()=>{
 const input=[{id:"a",day:"one",start:600,duration:60},{id:"b",day:"one",start:630,duration:30},{id:"c",day:"one",start:700,duration:15},{id:"d",day:"one",start:720,duration:15},{id:"e",day:"two",start:600,duration:60}],copy=structuredClone(input),out=calendarPositions(input);
 assert.deepEqual(input,copy);assert.equal(out.get("a").lanes,2);assert.notEqual(out.get("a").lane,out.get("b").lane);assert.notEqual(out.get("c").lane,out.get("d").lane);assert.equal(out.get("c").height,48);assert.equal(out.get("e").lanes,1);assert.equal(calendarPositions([]).size,0);
});
