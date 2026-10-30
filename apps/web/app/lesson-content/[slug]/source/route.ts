import {readFile} from "node:fs/promises";
import path from "node:path";
import {findLesson} from "../../../../features/lesson-library/catalog";
import {teacherAccess} from "../../../../features/lesson-library/teacher-access";

export const runtime="nodejs";
export async function GET(_request:Request,{params}:{params:Promise<{slug:string}>}){
 const access=await teacherAccess();if(access!=="allowed")return Response.json({error:"Нет доступа"},{status:access==="unauthorized"?401:403});
 const lesson=findLesson((await params).slug);if(!lesson)return Response.json({error:"Урок не найден"},{status:404});
 const source=await readFile(path.join(process.cwd(),"lesson-content",lesson.file),"utf8");
 return Response.json({source,filename:lesson.file,title:lesson.title,summary:lesson.description},{headers:{"Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff"}});
}
