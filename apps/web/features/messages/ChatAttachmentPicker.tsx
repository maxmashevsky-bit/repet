"use client";
import {useEffect,useRef,useState,type FormEvent} from "react";
import {chatApi} from "../../shared/api/chat";
import {ApiError} from "../../shared/api/client";
import type {ChatAttachment} from "../../shared/api/types";
import {Icon} from "../ui/Icon";

export function ChatAttachmentPicker({conversation,onSelect,onClose}:{conversation:string;onSelect:(a:ChatAttachment)=>void;onClose:()=>void}){
 const[kind,setKind]=useState<"file"|"assignment"|"lesson">("file"),[busy,setBusy]=useState(false),[error,setError]=useState(""),[retryable,setRetryable]=useState(false);const upload=useRef<{file:File;nonce:string}|null>(null);
 async function sendFile(file:File){
  if(file.size<1||file.size>4*1024*1024||! /\.(png|jpe?g|txt)$/i.test(file.name)){setRetryable(false);setError("Выберите PNG, JPG или TXT размером до 4 МБ.");return;}
  if(upload.current?.file!==file)upload.current={file,nonce:crypto.randomUUID()};const nonce=upload.current.nonce;setBusy(true);setError("");
  try{const bytes=new Uint8Array(await file.arrayBuffer());let raw="";for(let i=0;i<bytes.length;i+=16384)raw+=String.fromCharCode(...bytes.subarray(i,i+16384));const saved=await chatApi.upload(conversation,file.name,btoa(raw),nonce);onSelect({kind:"file",id:saved.id,title:saved.filename,available:true,file:saved});}
  catch(e){setRetryable(true);setError(e instanceof ApiError&&e.status===400?"Не удалось проверить файл. Нужен корректный PNG, JPG или текст UTF-8 без управляющих символов.":e instanceof ApiError&&e.status===422?"Хранилище файлов заполнено или недоступно.":e instanceof Error?e.message:"Не удалось загрузить файл");}finally{setBusy(false);}
 }
 return <section className="chat-attachment-picker" aria-label="Добавить вложение"><div className="chat-picker-head"><div role="group" aria-label="Тип вложения">{([["file","Файл"],["assignment","Задание"],["lesson","Урок"]] as const).map(([value,label])=><button key={value} type="button" aria-pressed={kind===value} disabled={busy} onClick={()=>{setKind(value);setError("");}}>{label}</button>)}</div><button type="button" className="icon-button" title="Закрыть вложения" aria-label="Закрыть вложения" disabled={busy} onClick={onClose}><Icon name="close"/></button></div>
 {kind==="file"?<div className="chat-file-picker"><label>Файл<input type="file" accept=".png,.jpg,.jpeg,.txt" disabled={busy} onChange={e=>{const file=e.target.files?.[0];if(file)void sendFile(file);}}/></label><small>PNG, JPG, TXT · до 4 МБ</small>{busy?<p role="status">Проверяем и сохраняем файл…</p>:null}{error&&retryable?<button type="button" className="secondary" disabled={busy} onClick={()=>{if(upload.current)void sendFile(upload.current.file);}}>Повторить загрузку</button>:null}</div>:<ReferenceList key={kind} conversation={conversation} kind={kind} onSelect={onSelect}/>}
 {error?<p className="field-error" role="alert">{error}</p>:null}</section>;
}

function ReferenceList({conversation,kind,onSelect}:{conversation:string;kind:"assignment"|"lesson";onSelect:(a:ChatAttachment)=>void}){
 const[text,setText]=useState(""),[query,setQuery]=useState(""),[offset,setOffset]=useState(0),[items,setItems]=useState<ChatAttachment[]>([]),[loading,setLoading]=useState(true),[error,setError]=useState(""),[revision,setRevision]=useState(0);
 useEffect(()=>{const controller=new AbortController();async function load(){try{const page=await chatApi.references(conversation,kind,query,offset,controller.signal);if(!controller.signal.aborted){setItems(page.items);setError("");}}catch(e){if(!controller.signal.aborted)setError(e instanceof Error?e.message:"Не удалось загрузить список");}finally{if(!controller.signal.aborted)setLoading(false);}}void load();return()=>controller.abort();},[conversation,kind,query,offset,revision]);
 function search(e:FormEvent){e.preventDefault();setLoading(true);setQuery(text.trim());setOffset(0);setRevision(v=>v+1);}
 return <><form className="chat-reference-search" onSubmit={search}><input aria-label={kind==="lesson"?"Найти урок":"Найти задание"} value={text} maxLength={120} onChange={e=>setText(e.target.value)} placeholder="Название"/><button type="submit" className="icon-button" title="Поиск" aria-label="Найти вложение"><Icon name="search"/></button></form>{loading?<p role="status">Загружаем…</p>:error?<p className="field-error" role="alert">{error}</p>:items.length?<ul className="chat-reference-list">{items.map(a=><li key={a.id}><button type="button" onClick={()=>onSelect(a)}><Icon name={kind==="lesson"?"book":"task"}/><span>{a.title}</span><Icon name="plus" size={18}/></button></li>)}</ul>:<p className="empty">{kind==="lesson"?"Нет опубликованных уроков, доступных обоим участникам.":"Нет заданий для этой переписки."}</p>}<div className="chat-reference-pages"><button type="button" className="secondary" disabled={loading||offset===0} onClick={()=>{setLoading(true);setOffset(v=>Math.max(0,v-20));}}>Назад</button><button type="button" className="secondary" disabled={loading||items.length<20} onClick={()=>{setLoading(true);setOffset(v=>v+20);}}>Далее</button></div></>;
}
