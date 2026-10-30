"use client";
import {useState} from "react";
import {chatApi,type ChatAssignmentPreview} from "../../shared/api/chat";
import type {ChatAttachment} from "../../shared/api/types";
import {formatDateTime} from "../../shared/lib/date";
import {Icon} from "../ui/Icon";

export function ChatAttachmentView({attachment:a,conversation,timezone,pending=false}:{attachment:ChatAttachment;conversation:string;timezone:string;pending?:boolean}){
 const[preview,setPreview]=useState<ChatAssignmentPreview|null>(null),[open,setOpen]=useState(false),[busy,setBusy]=useState(false),[error,setError]=useState("");
 if(!a.available||!a.id)return <p className="muted">Вложение больше недоступно</p>;
 const id=a.id;
 async function show(){setOpen(v=>!v);if(preview||busy||open)return;setBusy(true);setError("");try{setPreview(await chatApi.assignment(conversation,id));}catch(e){setError(e instanceof Error?e.message:"Не удалось открыть задание");}finally{setBusy(false);}}
 async function download(){if(busy)return;setBusy(true);setError("");try{const res=await fetch(chatApi.download(conversation,id),{credentials:"include",cache:"no-store"});if(!res.ok)throw new Error("Файл недоступен. Обновите переписку.");const url=URL.createObjectURL(await res.blob()),link=document.createElement("a");link.href=url;link.download=a.file?.filename||"Вложение";document.body.append(link);link.click();link.remove();setTimeout(()=>URL.revokeObjectURL(url),1000);}catch(e){setError(e instanceof Error?e.message:"Не удалось скачать файл");}finally{setBusy(false);}}
 return <div className="chat-attachment"><div className="chat-attachment-title"><Icon name={a.kind==="file"?"attach":a.kind==="assignment"?"task":"book"}/><strong>{a.title}</strong></div>
 {a.kind==="file"?<><small>{a.file?`${Math.max(1,Math.ceil(a.file.size_bytes/1024))} КБ`:null}</small><button type="button" className="secondary" disabled={pending||busy} onClick={()=>void download()}>{busy?"Загрузка…":"Скачать файл"}</button></>:a.kind==="lesson"?<a href={`/tasks/library/${id}`} className="button secondary" aria-disabled={pending} onClick={e=>{if(pending)e.preventDefault();}}>Открыть урок <Icon name="arrow" size={16}/></a>:<><button type="button" className="secondary" disabled={pending} aria-expanded={open} onClick={()=>void show()}>{open?"Свернуть задание":"Посмотреть задание"}</button>{open?<div className="chat-assignment-preview">{busy?<p role="status">Загружаем задание…</p>:preview?<><p>{preview.body||preview.title}</p><small>Срок: {formatDateTime(preview.due_at,timezone)}</small><a className="button secondary" href={preview.kind==="material"?`/tasks/${id}`:`/tasks?task=${id}`}>Перейти к заданию <Icon name="arrow" size={16}/></a></>:null}</div>:null}</>}
 {error?<p className="field-error" role="alert">{error}</p>:null}</div>;
}
