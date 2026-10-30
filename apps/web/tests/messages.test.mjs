import test from "node:test";
import assert from "node:assert/strict";
import {startPolling} from "../features/messages/polling.ts";
import {mergeMessages} from "../features/messages/model.ts";
const flush=()=>new Promise(resolve=>setImmediate(resolve));
function environment(){let visible=true,listener;const timers=new Map();let n=0;return{timers,env:{visible:()=>visible,subscribe:fn=>{listener=fn;return()=>{listener=undefined;};},delay:(fn,ms)=>{const id=++n;timers.set(id,{fn,ms});return id;},cancel:id=>timers.delete(id)},visibility:value=>{visible=value;listener?.();},tick:()=>{assert.equal(timers.size,1);const[id,t]=[...timers][0];timers.delete(id);t.fn();}};}
test("polling never overlaps, pauses/aborts hidden work, resumes and cleans up",async()=>{
 const e=environment(),calls=[];let settle;
 const stop=startPolling(signal=>{calls.push(signal);return new Promise(resolve=>{settle=resolve;});},e.env,5000);
 assert.equal(calls.length,1);assert.equal(e.timers.size,0);e.visibility(true);assert.equal(calls.length,1);
 e.visibility(false);assert.ok(calls[0].aborted);settle();await flush();assert.equal(e.timers.size,0);
 e.visibility(true);assert.equal(calls.length,2);settle();await flush();assert.equal([...e.timers.values()][0].ms,5000);e.tick();assert.equal(calls.length,3);stop();assert.ok(calls[2].aborted);settle();await flush();assert.equal(e.timers.size,0);e.visibility(true);assert.equal(calls.length,3);
});
test("polling backs off failures up to60s and resets after success",async()=>{
 const e=environment();let fails=true,errors=0;const stop=startPolling(async()=>{if(fails)throw new Error("offline");},e.env,5000,()=>errors++);
 for(const expected of[10000,20000,40000,60000]){await flush();assert.equal([...e.timers.values()][0].ms,expected);e.tick();}await flush();assert.equal(errors,5);fails=false;e.tick();await flush();assert.equal([...e.timers.values()][0].ms,5000);stop();assert.throws(()=>startPolling(async()=>{},e.env,0));
});
test("message merge deduplicates send acknowledgements and stale polls without crossing conversations",()=>{
 const item=(id,sequence,conversation_id="a")=>({id,sequence,conversation_id,author_id:"u",body:id,created_at:`2026-10-30T00:00:0${sequence}Z`});
 const current=[item("first",1),item("new",3)],incoming=[item("first",1),item("second",2),item("foreign",4,"b")];assert.deepEqual(mergeMessages(current,incoming,"a").map(m=>m.id),["first","second","new"]);
 assert.equal(mergeMessages([item("sent",3),item("sent",3)],[],"a").length,1);assert.deepEqual(current.map(m=>m.id),["first","new"]);
});
