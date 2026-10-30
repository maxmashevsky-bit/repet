import test from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir,writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
const require=createRequire(import.meta.url);
const {chromium,expect:baseExpect}=require(process.env.REPET_PLAYWRIGHT_MODULE||"playwright/test");
const expect=baseExpect.configure({timeout:30000}),baseURL=process.env.REPET_E2E_URL||"http://127.0.0.1:3015";
const output=fileURLToPath(new URL("../../../tmp/frontend-polish/",import.meta.url));
const id=n=>`00000000-0000-4000-8000-${String(n).padStart(12,"0")}`;
const longName="Александра Константиновна Великопольская-Заречная";
const title="Системы уравнений и нестандартные задачи с параметрами для самостоятельной подготовки";
const word="ОченьДлинноеНазваниеБезПробеловДляПроверкиПереноса".repeat(3);
const stamp=new Date().toISOString(),tomorrow=new Date(Date.now()+86400000).toISOString();
const pageOf=items=>({items,total:items.length,limit:100,offset:0});
const version={id:id(5),number:1,topic_id:id(3),title,summary:"Разбираем методы решения, исследуем условия и сравниваем несколько подходов.",content:`Условие с длинным обозначением: ${word}`,track:"oge",grade_min:5,grade_max:11,difficulty:3,created_at:stamp,published_at:stamp,exercises:[{id:id(6),kind:"numeric",prompt:`Вычислите значение выражения. Дополнительное условие: ${word}`,options:[],config:{},difficulty:2,position:0,hint_count:1},{id:id(7),kind:"text",prompt:"Объясните, какие значения переменной подходят условию.",options:[],config:{},difficulty:2,position:1,hint_count:0}]};
const course={id:id(2),author_id:id(1),author_name:longName,title,description:"Математика",track:"oge",grade_min:5,grade_max:11,difficulty:3,visibility:"linked_students",status:"published"};
const topic={id:id(3),course_id:id(2),title:"Числовые выражения и преобразования",description:"",parent_id:null,position:0,status:"published"};
const lesson={id:id(4),course_id:id(2),author_id:id(1),author_name:longName,status:"published",has_unpublished_changes:false,version};
const entries=version.exercises.map((exercise,i)=>({id:id(40+i),author_id:id(1),author_name:longName,lesson_id:id(4),version_id:id(5),subject:"math",exercise,grade:9,direction:"oge",topic:topic.title,subtopic:"Действия с обыкновенными дробями",exam_number:7,source:`Собственный демонстрационный материал ${word}`,attribution:longName,rights_status:"own",license:"Разрешено использовать на индивидуальных занятиях",created_at:stamp}));
const collection={id:id(50),lesson_id:id(4),version_id:id(5),title,exercise_ids:version.exercises.map(e=>e.id),provenance:entries,version,created_at:stamp};
const learning={attempt:{id:id(60),lesson_id:id(4),version_id:id(5),number:1,mode:"practice",player_step:`exercise:${id(6)}`,theory_seen:true,example_seen:true,status:"in_progress",started_at:stamp,completed_at:null,elapsed_seconds:20,total_exercises:2,answered_exercises:0,correct_exercises:0,pending_review:0,incorrect_attempts:0},version,answers:[],hints:[]};
function dashboard(role){return {user:{id:role==="tutor"?id(1):id(10),organization_id:id(99),display_name:longName,email:"alexandra.very.long.email.address@example.test",role,timezone:"Europe/Moscow",locale:"ru",created_at:stamp},relations:[{id:id(12),tutor_id:id(1),student_id:id(10),status:"active",created_at:stamp}],lessons:[{id:id(15),relation_id:id(12),tutor_id:id(1),student_id:id(10),title,starts_at:tomorrow,ends_at:new Date(Date.now()+90000000).toISOString(),status:"scheduled",created_at:stamp}],assignments:["assigned","submitted","done"].map((status,i)=>({id:id(20+i),kind:"text",tutor_id:id(1),student_id:id(10),student_name:longName,title:i===1?word:title,subject:"Математика",topic:topic.title,body:`Подробное условие: ${word}`,status,due_at:tomorrow,max_score:100,feedback:"",answer:"Развёрнутое решение ученика с подробным объяснением каждого шага.",created_at:stamp})),conversations:[{id:id(30),tutor_id:id(1),student_id:id(10),tutor_name:longName,student_name:longName,last_message:`Сообщение с длинной ссылкой https://example.test/${word}`,created_at:stamp}],notifications:[{id:id(31),title,body:word,href:"/tasks",created_at:stamp}]};}
async function fixture(context,role){
 await context.addCookies([{name:"repet_session",value:"layout-fixture-only",url:baseURL}]);
 await context.route("**/api/v1/**",async route=>{
  const p=new URL(route.request().url()).pathname;let data,status=200;
  if(p.endsWith("/dashboard"))data=dashboard(role);
  else if(p.endsWith("/me"))data=dashboard(role).user;
  else if(p.includes("/messages/conversations/"))data={messages:[{id:id(33),conversation_id:id(30),author_id:id(10),body:`Сообщение с длинным словом: ${word}\nhttps://example.test/${word}`,created_at:stamp},{id:id(34),conversation_id:id(30),author_id:id(1),body:"Хорошо, обсудим решение на следующем занятии.",created_at:stamp}]};
  else if(p.endsWith("/messages/conversations"))data={conversations:dashboard(role).conversations};
  else if(p.endsWith("/tutor-planner"))data={revision:1,students:[{id:id(10),name:longName,subject:title,price_kopecks:150000,duration:60,contact:"@layout_demo"}],lessons:[{id:id(80),student_id:id(10),weekday:1,start_minute:600,duration:60,paused:false}],windows:[],calculator:{goal_kopecks:10000000,price_kopecks:150000,lessons_per_student:2,duration:60,weeks:4.33,people:1,preparation_hours:2}};
  else if(p.includes("/pdf-source")){status=404;data={error:{message:"No PDF source"}};}
  else if(p.includes("/learning-attempts/"))data=learning;
  else if(p.endsWith("/answer-keys"))data={items:version.exercises.map(e=>({exercise_id:e.id,answer:"7",explanation:"Авторское решение.",hints:["Подсказка"]}))};
  else if(p.endsWith("/attempts"))data=pageOf([]);
  else if(p.endsWith("/topics"))data=pageOf([topic]);
  else if(p.includes("/catalog/topics/"))data=topic;
  else if(p.endsWith("/catalog/courses"))data=pageOf([course]);
  else if(p.includes("/catalog/courses/"))data=course;
  else if(p.endsWith("/versions"))data=pageOf([version]);
  else if(p.endsWith("/catalog/lessons"))data=pageOf([lesson,{...lesson,id:id(44),version:{...version,title:word}}]);
  else if(p.includes("/catalog/lessons/"))data=lesson;
  else if(p.endsWith("/exercise-bank/facets"))data={topics:[topic.title],subtopics:["Действия с обыкновенными дробями"]};
  else if(p.endsWith("/exercise-bank"))data=pageOf(entries);
  else if(p.endsWith("/bank-collections"))data=pageOf([collection]);
  else if(p.includes("/bank-collections/"))data=collection;
  else if(p.endsWith("/exercise-sets"))data=pageOf([]);
  else if(p.endsWith("/html-lessons")||p.endsWith("/pdf-imports"))data=pageOf([]);
  else {status=404;data={error:{message:`Layout fixture unavailable: ${p}`}};}
  await route.fulfill({status,contentType:"application/json",body:JSON.stringify(data)});
 });
}
async function layoutIssues(page){return page.evaluate(()=>{
 const issues=[];if(document.documentElement.scrollWidth>innerWidth+1)issues.push(`page overflow ${document.documentElement.scrollWidth}/${innerWidth}`);
 for(const el of document.querySelectorAll("button,.button,h1,h2,h3,.profile-copy,.composer,.calendar-toolbar,.summary-heading")){
  const rect=el.getBoundingClientRect(),style=getComputedStyle(el);if(!rect.width||!rect.height||style.visibility==="hidden")continue;
  let clipped=false;for(let p=el.parentElement;p;p=p.parentElement){if(["auto","scroll","hidden","clip"].includes(getComputedStyle(p).overflowX)&&p.scrollWidth>p.clientWidth+1){clipped=true;break;}}
  if(!clipped&&(rect.right>innerWidth+1||rect.left< -1))issues.push(`${el.tagName}.${el.className}: outside viewport`);
  if(el.matches("button,.button")&&el.scrollWidth>el.clientWidth+2&&getComputedStyle(el).textOverflow!=="ellipsis")issues.push(`control overflow: ${el.textContent?.trim().slice(0,70)}`);
 }
 const composer=document.querySelector(".composer"),area=composer?.querySelector("textarea"),send=composer?.querySelector("button");
 if(area&&send&&area.getBoundingClientRect().right>send.getBoundingClientRect().left)issues.push("composer fields overlap");
 return issues;
});}
async function resize(page,width){await page.setViewportSize({width,height:900});await page.evaluate(()=>scrollTo({top:0,left:0,behavior:"instant"}));await page.waitForTimeout(80);}
test("responsive product typography and controls with long Russian text",{timeout:1200000},async(t)=>{
 await mkdir(output,{recursive:true});const browser=await chromium.launch({headless:true,...(process.env.REPET_CHROME_PATH?{executablePath:process.env.REPET_CHROME_PATH}:{})});t.after(()=>browser.close());
 const failures=[],errors=[],summary=[];
 t.after(()=>writeFile(output+"layout-results.json",JSON.stringify({summary,failures,errors},null,2)));
 // The legacy workshop has server-side authentication and is covered by workshop-lessons.e2e.mjs.
 const routes={tutor:["/","/messages","/calendar","/tasks","/settings","/calendar/planner","/tasks/library","/tasks/library/new",`/tasks/library/${id(4)}/edit`,"/tasks/library/html","/tasks/library/html/import","/tasks/library/pdf","/tasks/library/bank",`/tasks/library/bank/collections/${id(50)}`],student:["/","/messages","/tasks","/settings",`/tasks/library/${id(4)}`]};
 const widths=(process.env.REPET_LAYOUT_WIDTHS||"1440,1024,768,375,320").split(",").map(Number);
 for(const role of ["tutor","student"]){const context=await browser.newContext({viewport:{width:1440,height:1000},timezoneId:"Europe/Moscow"});await fixture(context,role);const page=await context.newPage();page.on("pageerror",e=>errors.push(`${role}: ${e.message}`));
  for(const path of routes[role]){await page.goto(baseURL+path,{waitUntil:"networkidle",timeout:180000});await expect(page.locator("#content")).toBeVisible();await page.waitForTimeout(200);for(const width of widths){await resize(page,width);const issues=await layoutIssues(page);summary.push({role,path,width,issues});if(issues.length)failures.push({role,path,width,issues});if((width===1440||width===375)&&(["/","/messages","/calendar","/tasks","/tasks/library","/tasks/library/bank","/tasks/library/new","/settings"].includes(path)||path.endsWith("/edit")))await page.screenshot({path:output+`${role}-${path==="/"?"dashboard":path.slice(1).replaceAll("/","-")}-${width}.png`,fullPage:true});}
   if(path==="/tasks/library"||path==="/tasks/library/bank"){await page.setViewportSize({width:375,height:900});const toggle=page.getByRole("button",{name:"Фильтры",exact:true});await toggle.click();await expect(toggle).toHaveAttribute("aria-expanded","true");const issues=await layoutIssues(page);if(issues.length)failures.push({role,path:"open-filters",width:375,issues});await toggle.click();await expect(toggle).toHaveAttribute("aria-expanded","false");}
   if(path==="/tasks/library/new"){await page.setViewportSize({width:320,height:900});await page.getByRole("button",{name:"Новый курс",exact:true}).click();await expect(page.getByRole("dialog")).toBeVisible();const issues=await layoutIssues(page);if(issues.length)failures.push({role,path:"course-dialog",width:320,issues});await page.screenshot({path:output+"course-dialog-320.png"});await page.getByRole("button",{name:"Закрыть окно",exact:true}).click();}
   if(path==="/calendar"){await page.setViewportSize({width:375,height:900});await page.getByRole("button",{name:"День",exact:true}).click();await expect(page.locator(".time-grid:not(.week)")).toBeVisible();assert.equal(await page.locator(".time-grid").evaluate(el=>el.scrollWidth>el.clientWidth+1),false,"day calendar must fit the phone");await page.screenshot({path:output+"calendar-day-375.png",fullPage:true});await page.getByRole("button",{name:"Месяц",exact:true}).click();await expect(page.locator(".month-grid")).toBeVisible();await page.getByRole("button",{name:"Добавить занятие",exact:true}).click();const issues=await layoutIssues(page);if(issues.length)failures.push({role,path:"calendar-month-and-form",width:375,issues});}
   if(path==="/"){await page.getByRole("button",{name:"Открыть уведомления",exact:true}).click();await expect(page.getByRole("region",{name:"Уведомления",exact:true})).toBeVisible();for(const width of [375,320]){await page.setViewportSize({width,height:900});const issues=await layoutIssues(page);if(issues.length)failures.push({role,path:"notifications",width,issues});if(width===375)await page.screenshot({path:output+`${role}-notifications-375.png`});}await page.getByRole("button",{name:"Открыть уведомления",exact:true}).click();await page.getByRole("button",{name:"Открыть меню",exact:true}).click();await expect(page.getByRole("dialog",{name:"Мобильное меню"})).toBeVisible();const issues=await layoutIssues(page);if(issues.length)failures.push({role,path:"mobile-menu",width:320,issues});await page.getByRole("button",{name:"Закрыть меню",exact:true}).click();}
   console.log(`Layout checked ${role} ${path}`);
  }await context.close();
 }
 const context=await browser.newContext();const page=await context.newPage();page.on("pageerror",e=>errors.push(e.message));for(const path of ["/register","/login","/forgot-password","/reset-password","/privacy","/terms"]){await page.goto(baseURL+path,{waitUntil:"networkidle",timeout:180000});for(const width of widths){await resize(page,width);const issues=await layoutIssues(page);summary.push({role:"public",path,width,issues});if(issues.length)failures.push({role:"public",path,width,issues});if(path==="/register"&&(width===375||width===1440))await page.screenshot({path:output+`register-${width}.png`,fullPage:true});}}await context.close();
 assert.deepEqual(errors,[]);assert.deepEqual(failures,[]);console.log(`Layout PASS ${summary.length} viewport/page combinations`);
});
