import test from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
const require=createRequire(import.meta.url);
const {chromium,request,expect:baseExpect}=require(process.env.REPET_PLAYWRIGHT_MODULE||"playwright/test");
const expect=baseExpect.configure({timeout:30000}),baseURL=process.env.REPET_E2E_URL||"http://127.0.0.1:3012";
const artifacts=fileURLToPath(new URL("../../../tmp/bank10/",import.meta.url));
test("teacher registers, filters and orders a cross-lesson bank collection, assigns two students; pinned player works on mobile",{timeout:360000},async(t)=>{
 await mkdir(artifacts,{recursive:true});
 const browser=await chromium.launch({headless:true,...(process.env.REPET_CHROME_PATH?{executablePath:process.env.REPET_CHROME_PATH}:{})});
 const teacher=await request.newContext({baseURL}),student=await request.newContext({baseURL}),peer=await request.newContext({baseURL});
 t.after(async()=>{await browser.close();await teacher.dispose();await student.dispose();await peer.dispose();});
 const call=async(api,method,url,data,status=200)=>{const r=await api.fetch(url,{method,...(data?{data}:{})});assert.equal(r.status(),status,`${url}: ${await r.text()}`);return r.json();};
 const unique=crypto.randomUUID();
 const names=["Преподаватель банка","Алексей Банк","Мария Банк"];
 const users=[];
 for(const [i,api]of[teacher,student,peer].entries()){const out=await call(api,"POST","/api/v1/auth/register",{email:`bank-${i}-${unique}@example.test`,display_name:names[i],password:"BankBrowser123!",role:i===0?"tutor":"student",accept_terms:true},201);users.push(out.user);}
 for(const [api,u]of[[student,users[1]],[peer,users[2]]]){const inv=await call(teacher,"POST","/api/v1/invitations",{student_email:u.email},201);await call(api,"POST",`/api/v1/invitations/${inv.token}/accept`,{},201);}
 const base="/api/v1/catalog",course=await call(teacher,"POST",base+"/courses",{title:`Банк ${unique.slice(0,8)}`,track:"school",grade_min:5,grade_max:11,difficulty:1,visibility:"linked_students"},201);
 const topic=await call(teacher,"POST",`${base}/courses/${course.id}/topics`,{title:"Числовая прямая",position:0},201);await call(teacher,"POST",`${base}/courses/${course.id}/publish`,{});await call(teacher,"POST",`${base}/topics/${topic.id}/publish`,{});
 const sources=[];
 for(let i=0;i<3;i++){
  const lesson=await call(teacher,"POST",`${base}/courses/${course.id}/lessons`,{topic_id:topic.id,title:`Собственный пример ${i+1}`,summary:"Демонстрационный материал автора",content:"",track:"school",grade_min:5,grade_max:11,difficulty:2},201);
  const exercise=await call(teacher,"POST",`${base}/lessons/${lesson.id}/exercises`,{kind:"numeric",prompt:`На числовой прямой: сколько будет ${i+3} + 4?`,options:[],difficulty:2,position:0,answer_key:{answer:String(i+7),explanation:"PRIVATE_BANK_BROWSER_SOLUTION"}},201);
  await call(teacher,"POST",`${base}/lessons/${lesson.id}/publish`,{});sources.push({lesson,exercise});
 }
 const tc=await browser.newContext({storageState:await teacher.storageState(),viewport:{width:1440,height:1000}}),tp=await tc.newPage(),errors=[];tp.on("pageerror",e=>errors.push(e.message));
 await tp.goto(`${baseURL}/tasks/library/${sources[0].lesson.id}`);const reg=tp.locator(".bank-register");await reg.locator("summary").click();
 await reg.getByRole("combobox",{name:"Класс",exact:true}).selectOption("9");await reg.getByRole("combobox",{name:"Направление",exact:true}).selectOption("oge");await reg.getByLabel("Номер экзаменационного задания",{exact:true}).fill("7");await reg.getByLabel("Подтема",{exact:true}).fill("Сложение");await reg.getByLabel("Источник",{exact:true}).fill("Собственный пример; не официальный экзамен");await reg.getByLabel("Лицензия / условия разрешения",{exact:true}).fill("Назначение моим ученикам разрешено");await reg.getByRole("button",{name:"Добавить в банк",exact:true}).click();await expect(reg.getByRole("status")).toContainText("Упражнение добавлено");
 for(const [i,s]of sources.entries()){if(i===0)continue;await call(teacher,"POST","/api/v1/exercise-bank",{lesson_id:s.lesson.id,exercise_id:s.exercise.id,grade:9,direction:i===2?"olympiad":"oge",topic:"Числовая прямая",subtopic:"Сложение",exam_number:i===2?null:7,source:"Собственный демонстрационный пример",attribution:names[0],rights_status:"own",license:"Назначение ученикам разрешено"},201);}
 await reg.getByRole("link",{name:"Открыть банк",exact:true}).click();await expect(tp.getByRole("heading",{name:"Банк упражнений",exact:true})).toBeVisible();
 await tp.getByRole("combobox",{name:"Направление",exact:true}).selectOption("oge");await tp.getByRole("combobox",{name:"Класс",exact:true}).selectOption("9");await tp.getByLabel("Номер экзаменационного задания",{exact:true}).fill("7");await tp.getByRole("combobox",{name:"Тема",exact:true}).selectOption("Числовая прямая");await expect(tp.locator(".bank-exercise-row")).toHaveCount(2);
 await tp.getByRole("textbox",{name:"Поиск упражнений"}).fill("прямой");await tp.getByRole("button",{name:"Найти",exact:true}).click();await expect(tp.locator(".bank-exercise-row")).toHaveCount(2);
 await tp.locator(".bank-exercise-row").first().locator("summary").click();await tp.getByRole("button",{name:"Показать решение",exact:true}).click();await expect(tp.locator(".bank-solution")).toContainText("PRIVATE_BANK_BROWSER_SOLUTION");
 await tp.locator(".bank-exercise-row").nth(0).getByRole("checkbox").check();await tp.locator(".bank-exercise-row").nth(1).getByRole("checkbox").check();await tp.getByRole("button",{name:"Поднять упражнение 2",exact:true}).click();
 await tp.setViewportSize({width:375,height:900});assert.equal(await tp.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);await tp.evaluate(()=>scrollTo(0,0));await tp.screenshot({path:artifacts+"bank-mobile.png",fullPage:true});await tp.setViewportSize({width:1440,height:1000});
 const selection=tp.locator(".bank-selection");await selection.getByLabel("Название подборки",{exact:true}).fill("ОГЭ №7 — моя подборка");await selection.getByRole("combobox",{name:"Курс подборки",exact:true}).selectOption(course.id);await selection.getByRole("combobox",{name:"Тема подборки",exact:true}).selectOption(topic.id);
 const nonces=[];tp.on("request",r=>{if(r.method()==="POST"&&new URL(r.url()).pathname==="/api/v1/bank-collections")nonces.push(r.postDataJSON().request_id);});
 await tp.route("**/api/v1/bank-collections",r=>r.request().method()==="POST"?r.fulfill({status:503,contentType:"application/json",body:JSON.stringify({error:{message:"Test retry"}})}):r.continue());await selection.getByRole("button",{name:"Сохранить подборку",exact:true}).click();await expect(selection.getByRole("alert")).toBeVisible();await tp.unroute("**/api/v1/bank-collections");await selection.getByRole("button",{name:"Сохранить подборку",exact:true}).click();await expect(tp).toHaveURL(/\/bank\/collections\/[a-f0-9-]{36}$/);assert.equal(nonces[0],nonces[1]);
 const cid=new URL(tp.url()).pathname.split("/").at(-1),collection=await call(teacher,"GET",`/api/v1/bank-collections/${cid}`);
 assert.equal(collection.version.exercises.length,2);assert.equal(collection.provenance[0].exercise.prompt,sources[0].exercise.prompt);
 await call(teacher,"PUT",`${base}/lessons/${sources[0].lesson.id}/exercises/${sources[0].exercise.id}`,{kind:"numeric",prompt:"ИЗМЕНЕННЫЙ ИСТОЧНИК",options:[],difficulty:2,position:0,answer_key:{answer:"99",explanation:"Different key"}});await call(teacher,"POST",`${base}/lessons/${sources[0].lesson.id}/publish`,{});
 await tp.getByRole("checkbox",{name:names[1],exact:true}).check();await tp.getByRole("checkbox",{name:names[2],exact:true}).check();await tp.getByLabel("Срок сдачи",{exact:true}).fill(new Date(Date.now()+2*86400000).toISOString().slice(0,16));
 const assigned=tp.waitForResponse(r=>r.url().endsWith(`/bank-collections/${cid}/assign`)&&r.request().method()==="POST");await tp.getByRole("button",{name:"Назначить подборку",exact:true}).click();const batch=await(await assigned).json();assert.equal(batch.length,2);assert.ok(batch.every(r=>r.assignment));await expect(tp.locator(".bank-batch-results").getByRole("link")).toHaveCount(2);await tp.evaluate(()=>scrollTo(0,0));await tp.screenshot({path:artifacts+"collection-desktop.png",fullPage:true});
 const own=batch.find(r=>r.student_id===users[1].id).assignment,other=batch.find(r=>r.student_id===users[2].id).assignment;
 const hidden=await student.get(`${base}/lessons/${collection.lesson_id}`);assert.equal(hidden.status(),404);assert.equal((await student.get(`/api/v1/exercise-bank/${collection.provenance[0].id}/answer-key`)).status(),403);assert.equal((await student.get(`/api/v1/assignments/${other.id}/learning-results`)).status(),404);
 const mc=await browser.newContext({storageState:await student.storageState(),viewport:{width:375,height:900},isMobile:true,hasTouch:true}),sp=await mc.newPage();sp.on("pageerror",e=>errors.push(e.message));const keyCalls=[];sp.on("request",r=>{if(r.url().includes("answer-key"))keyCalls.push(r.url());});
 await sp.goto(`${baseURL}/tasks/${own.id}`);await sp.getByRole("button",{name:"Начать выполнение",exact:true}).tap();await sp.getByRole("button",{name:"Продолжить",exact:true}).tap();
 for(let i=0;i<2;i++){await expect(sp.locator(".engine-prompt")).toHaveText(collection.version.exercises[i].prompt);const prompt=await sp.locator(".engine-prompt").innerText();await sp.getByRole("textbox",{name:"Число или дробь",exact:true}).fill(prompt.includes("3 + 4")?"7":"8");await sp.getByRole("button",{name:"Проверить ответ",exact:true}).tap();await expect(sp.locator(".engine-status")).toHaveText("Верно");await sp.getByRole("button",{name:i===1?"Завершить урок":"Продолжить",exact:true}).tap();}
 await expect(sp.getByRole("heading",{name:"Урок завершён",exact:true})).toBeVisible();assert.equal(await sp.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);await sp.evaluate(()=>scrollTo(0,0));await sp.screenshot({path:artifacts+"student-results-mobile.png",fullPage:true});
 const results=await call(teacher,"GET",`/api/v1/assignments/${own.id}/learning-results`);assert.equal(results.state.attempt.correct_exercises,2);assert.equal(results.assignment.status,"done");assert.equal(results.version.exercises[0].prompt,sources[0].exercise.prompt);assert.deepEqual(keyCalls,[]);assert.deepEqual(errors,[]);
});
