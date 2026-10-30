import test from "node:test";
import assert from "node:assert/strict";
import {suggestPDFExercises} from "../features/pdf-import/model.ts";
import {parseExercise} from "../features/exercise-engine/schema.ts";
test("PDF suggestions retain numbering and numbers without inventing answers",()=>{
 const items=suggestPDFExercises("1. Calculate 5 + 2.\n2. Find 9 - 1.\n3. Explain why 2 + 3 = 3 + 2.\f");
 assert.equal(items.length,3);assert.deepEqual(items.map(i=>i.kind),["numeric","numeric","long_text"]);assert.equal(items[1].prompt,"2. Find 9 - 1.");assert.ok(items.every(i=>!Object.hasOwn(i,"answer")));
 assert.deepEqual(suggestPDFExercises(" \n\f"),[]);
 const long="1. "+"x".repeat(21000);assert.equal(suggestPDFExercises(long)[0].prompt,long);
});
test("exercise media allows private IDs and alt text, rejects URLs and answer fields",()=>{
 const id="10000000-0000-4000-8000-000000000001",base={id,kind:"numeric",prompt:"Find 2 + 3",options:[],difficulty:1,position:0};
 assert.deepEqual(parseExercise({...base,media:[{asset_id:id,alt:"Triangle"}]}).media,[{asset_id:id,alt:"Triangle"}]);
 for(const media of [[{asset_id:"https://evil.test/a.png",alt:"x"}],[{asset_id:id,alt:""}],[{asset_id:id,alt:"a",answer:"5"}],Array(5).fill({asset_id:id,alt:"x"}),[{asset_id:id,alt:"a"},{asset_id:id,alt:"b"}]])assert.throws(()=>parseExercise({...base,media}));
});
