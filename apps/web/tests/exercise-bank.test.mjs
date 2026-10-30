import test from "node:test";
import assert from "node:assert/strict";
import { moveSelection, directionLabels } from "../features/exercise-bank/model.ts";
test("bank selection order is immutable and bounded",()=>{
 const input=["a","b","c"];assert.deepEqual(moveSelection(input,0,1),["b","a","c"]);assert.deepEqual(moveSelection(input,2,-1),["a","c","b"]);assert.deepEqual(input,["a","b","c"]);
 for(const [i,d]of [[0,-1],[2,1],[-1,1],[3,-1],[0,.5]])assert.equal(moveSelection(input,i,d),input);
 assert.deepEqual(Object.keys(directionLabels),["school","olympiad","oge","ege_basic","ege_profile"]);
});
