"use client";
import Link from "next/link";
import { useEffect,useState } from "react";
import { analyticsApi,type AnalyticsReport } from "../../shared/api/analytics";
import { analyticsRange,metricSample,parseAnalyticsReport,percentage } from "../learning-analytics/model";
import { Icon } from "../ui/Icon";
import { ReviewError } from "./ReviewUI";
export function MasteryOverview({studentID,revision}:{studentID:string;revision:number}){
 const[data,setData]=useState<{key:string;report?:AnalyticsReport;error?:unknown}>({key:""}),key=`${studentID}:${revision}`;
 useEffect(()=>{const c=new AbortController();analyticsApi.report(studentID,analyticsRange(90),c.signal).then(parseAnalyticsReport).then(report=>{if(report.student.id!==studentID)throw new Error("Некорректный ученик в статистике.");if(!c.signal.aborted)setData({key,report});}).catch((error:unknown)=>{if(!c.signal.aborted)setData({key,error});});return()=>c.abort();},[studentID,key]);
 return <section className="review-mastery"><header><h2>Освоение тем</h2><Link className="inline-link" href="/tasks/progress">Вся статистика<Icon name="arrow" size={17}/></Link></header><p className="review-period">Последние 90 дней</p>{data.key!==key?<p role="status">Загружаем результаты…</p>:data.error?<ReviewError error={data.error}/>:!data.report?.topics.items.length?<p className="muted">Результаты появятся после первых решений.</p>:<div>{data.report.topics.items.slice(0,5).map(t=><article key={t.key}><div><h3>{t.topic}{t.subtopic?` · ${t.subtopic}`:""}</h3><span>{{insufficient:"Мало данных",practice:"Нужно повторить",developing:"Закрепляем",secure:"Уверенный результат"}[t.level]}</span></div><div><strong>{percentage(t.accuracy)}</strong><small>{metricSample(t)}{t.pending?` · На проверке: ${t.pending}`:""}</small></div></article>)}</div>}</section>;
}
