import {HTMLLessonPage} from "../../../../../../features/html-lessons/HTMLLessonPage";
export default async function Page({params}:{params:Promise<{htmlId:string}>}){return <HTMLLessonPage lessonID={(await params).htmlId}/>;}
