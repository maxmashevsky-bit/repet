"use client";

import {useEffect,useState} from "react";
import {htmlApi} from "../../shared/api/html-lessons";
import {LibraryError} from "../material-library/LibraryUI";
import {Icon} from "../ui/Icon";

export function HTMLFrame({lessonID,versionID,assignmentID=""}:{lessonID:string;versionID:string;assignmentID?:string}){
 const [retry,setRetry]=useState(0),[load,setLoad]=useState<{key:string;url?:string;error?:unknown}>({key:""});
 const key=`${lessonID}:${versionID}:${assignmentID}:${retry}`;
 useEffect(()=>{const c=new AbortController();htmlApi.preview(lessonID,versionID,assignmentID,c.signal).then(data=>{const url=new URL(data.url);if(!["http:","https:"].includes(url.protocol)||url.hostname===location.hostname||url.username||url.password||!url.pathname.startsWith("/view/"))throw new Error("Сервер просмотра настроен небезопасно.");if(!c.signal.aborted)setLoad({key,url:url.href});}).catch(error=>{if(!c.signal.aborted)setLoad({key,error});});return()=>c.abort();},[lessonID,versionID,assignmentID,key]);
 return <section className="html-preview"><div className="section-head"><h2>Просмотр HTML-урока</h2><button type="button" className="secondary" onClick={()=>setRetry(n=>n+1)}><Icon name="arrow" size={17}/>Обновить просмотр</button></div><p className="html-format-notice">Старый HTML-формат · Встроенные баллы не являются проверенным результатом. Внешние ресурсы и сохранение внутри HTML недоступны.</p>
 {load.key!==key?<p role="status">Подготавливаем изолированный просмотр…</p>:load.error?<LibraryError error={load.error} retry={()=>setRetry(n=>n+1)}/>:load.url?<iframe key={load.url} className="html-lesson-frame" title="Изолированный HTML-урок" src={load.url} sandbox="allow-scripts" referrerPolicy="no-referrer" allow="camera 'none'; microphone 'none'; geolocation 'none'; clipboard-read 'none'; clipboard-write 'none'"/>:null}</section>;
}
