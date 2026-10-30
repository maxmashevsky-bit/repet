import Link from "next/link";
import {Icon,type IconName} from "../ui/Icon";

const tools:{href:string;label:string;icon:IconName}[]=[
 {href:"/calendar/planner",label:"Ученики и план",icon:"calendar"},
 {href:"/tasks/library",label:"Библиотека",icon:"book"},
 {href:"/tasks/library/pdf",label:"Импорт PDF",icon:"attach"},
 {href:"/tasks/library/bank",label:"Банк упражнений",icon:"task"},
 {href:"/tasks/study",label:"Пробники",icon:"clock"},
 {href:"/tasks/progress",label:"Успеваемость",icon:"chart"},
 {href:"/tasks/review",label:"Повторение",icon:"check"},
 {href:"/family",label:"Семья",icon:"user"}
];
export function TeacherTools(){return <nav className="teacher-tools" aria-label="Инструменты преподавателя">{tools.map(t=><Link key={t.href} href={t.href}><Icon name={t.icon}/><span>{t.label}</span><Icon name="chevron" size={16}/></Link>)}</nav>;}
