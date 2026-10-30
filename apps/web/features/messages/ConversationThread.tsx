"use client";
import { useEffect,useRef,useState,type FormEvent } from "react";
import { productApi } from "../../shared/api/client";
import type { ChatAttachment,Conversation,Message } from "../../shared/api/types";
import {chatApi} from "../../shared/api/chat";
import { formatTime } from "../../shared/lib/date";
import { useWorkspace } from "../app-shell/AppShell";
import { Icon } from "../ui/Icon";
import { usePolling } from "./usePolling";
import { mergeMessages } from "./model.ts";
import {ChatAttachmentPicker} from "./ChatAttachmentPicker";
import {ChatAttachmentView} from "./ChatAttachmentView";

export function ConversationThread({conversation,body,setBody,active}:{conversation:Conversation;body:string;setBody:(body:string)=>void;active:boolean}){
 const{dashboard,refreshConversations}=useWorkspace();const[messages,setMessages]=useState<Message[]>([]),[loaded,setLoaded]=useState(false),[hasOlder,setHasOlder]=useState(false),[error,setError]=useState(""),[busy,setBusy]=useState(false),[failed,setFailed]=useState<string[]>([]);
 const[attachment,setAttachment]=useState<ChatAttachment|null>(null),[picker,setPicker]=useState(false),[removing,setRemoving]=useState(false);
 const feed=useRef<HTMLDivElement>(null),scrollNext=useRef(true),initialized=useRef(false),readSequence=useRef(0),readBusy=useRef(false),sendGate=useRef(false),olderGate=useRef(false);
 const reload=usePolling(async signal=>{
  const items=await productApi.messages(conversation.id,0,signal);if(signal.aborted)return;
  const el=feed.current;scrollNext.current=!loaded||Boolean(el&&el.scrollHeight-el.scrollTop-el.clientHeight<80);
  setMessages(previous=>mergeMessages(previous,items,conversation.id));if(!initialized.current){setHasOlder(items.length===80);initialized.current=true;}setLoaded(true);setError("");
 },active,5000,e=>{setError(e instanceof Error?e.message:"Не удалось обновить чат");setLoaded(true);});

 async function markRead(){
  const last=messages.filter(m=>typeof m.sequence==="number").at(-1),el=feed.current;
  if(!active||!last?.sequence||last.sequence<=readSequence.current||readBusy.current||!document.hasFocus()||document.visibilityState!=="visible"||!el||el.scrollHeight-el.scrollTop-el.clientHeight>80)return;
  readBusy.current=true;
  try{await productApi.markMessagesRead(conversation.id,last.id);readSequence.current=last.sequence;await refreshConversations();}
  catch(e){setError(e instanceof Error?e.message:"Не удалось отметить сообщения прочитанными");}finally{readBusy.current=false;}
 }
 useEffect(()=>{
  if(!active)return;
  if(scrollNext.current&&feed.current){feed.current.scrollTop=feed.current.scrollHeight;scrollNext.current=false;}
  const frame=requestAnimationFrame(()=>void markRead()),focus=()=>void markRead();window.addEventListener("focus",focus);document.addEventListener("visibilitychange",focus);return()=>{cancelAnimationFrame(frame);window.removeEventListener("focus",focus);document.removeEventListener("visibilitychange",focus);};
 });

 async function loadOlder(){if(olderGate.current)return;olderGate.current=true;const el=feed.current,before=el?.scrollHeight??0;
  try{const items=await productApi.messages(conversation.id,messages.filter(m=>!m.id.startsWith("pending-")).length);setMessages(previous=>mergeMessages(items,previous,conversation.id));setHasOlder(items.length===80);requestAnimationFrame(()=>{if(el)el.scrollTop+=el.scrollHeight-before;});}
  catch(e){setError(e instanceof Error?e.message:"Не удалось загрузить историю");}finally{olderGate.current=false;}
 }
 async function send(text:string,nonce:string,attached?:ChatAttachment){
  if(sendGate.current)return;sendGate.current=true;setBusy(true);setError("");const pendingID=`pending-${nonce}`;setFailed(ids=>ids.filter(id=>id!==pendingID));scrollNext.current=true;
  setMessages(previous=>mergeMessages(previous,[{id:pendingID,conversation_id:conversation.id,author_id:dashboard.user.id,body:text,created_at:new Date().toISOString(),...(attached?{attachment:attached}:{})}],conversation.id));
  let sent=false;
  try{const fields=attached?.id?{[attached.kind==="file"?"file_id":attached.kind==="assignment"?"assignment_id":"lesson_id"]:attached.id}:{};const saved=await productApi.sendMessage(conversation.id,text,nonce,fields);sent=true;setMessages(previous=>mergeMessages(previous.filter(m=>m.id!==pendingID),[saved],conversation.id));await refreshConversations();}
  catch(e){if(!sent)setFailed(ids=>[...new Set([...ids,pendingID])]);setError(sent?"Сообщение отправлено, но не удалось обновить список диалогов.":e instanceof Error?e.message:"Сообщение не отправлено");}finally{sendGate.current=false;setBusy(false);}
 }
 function submit(e:FormEvent<HTMLFormElement>){e.preventDefault();if((!body.trim()&&!attachment)||sendGate.current||removing)return;const text=body.trim(),attached=attachment??undefined;setBody("");setAttachment(null);setPicker(false);void send(text,crypto.randomUUID(),attached);}
 async function removeAttachment(){if(removing||!attachment)return;setRemoving(true);try{if(attachment.kind==="file"&&attachment.id)await chatApi.remove(conversation.id,attachment.id);setAttachment(null);}catch(e){setError(e instanceof Error?e.message:"Не удалось убрать вложение");}finally{setRemoving(false);}}
 return <><header className="chat-head"><div className="chat-person"><span className="avatar">{(dashboard.user.role==="tutor"?conversation.student_name:conversation.tutor_name).slice(0,1)}</span><span><h2>{dashboard.user.role==="tutor"?conversation.student_name:conversation.tutor_name}</h2></span></div><div className="chat-actions"><a className="button secondary" href="/calendar?view=day">Открыть календарь</a></div></header>
 <div className="message-feed" ref={feed} onScroll={()=>void markRead()} aria-live="polite" aria-busy={!loaded}>{!loaded?<p className="empty">Загружаем сообщения…</p>:null}{hasOlder&&loaded?<button type="button" className="secondary" onClick={()=>void loadOlder()}>Загрузить предыдущие</button>:null}{!messages.length&&loaded?<p className="empty">История пуста. Напишите первое сообщение.</p>:null}{loaded?messages.map(m=><div key={m.id} className={m.author_id===dashboard.user.id?"bubble own":"bubble"}>{m.body?<p>{m.body}</p>:null}{m.attachment?<ChatAttachmentView attachment={m.attachment} conversation={conversation.id} timezone={dashboard.user.timezone} pending={m.id.startsWith("pending-")}/>:null}{m.id.startsWith("pending-")?failed.includes(m.id)?<><small>Не отправлено</small><button type="button" className="secondary message-retry" disabled={busy} onClick={()=>void send(m.body,m.id.slice(8),m.attachment)}>Повторить отправку</button></>:<small role="status">Отправляем…</small>:<small>{formatTime(m.created_at,dashboard.user.timezone)}</small>}</div>):null}</div>
 {error?<div className="message-error" role="alert"><p className="field-error">{error}</p><button type="button" className="secondary" onClick={reload}>Обновить переписку</button></div>:null}
 {picker?<ChatAttachmentPicker conversation={conversation.id} onClose={()=>setPicker(false)} onSelect={a=>{setAttachment(a);setPicker(false);}}/>:null}
 {attachment?<div className="chat-selected-attachment"><Icon name={attachment.kind==="file"?"attach":attachment.kind==="assignment"?"task":"book"}/><span>{attachment.title}</span><button type="button" className="icon-button" disabled={busy||removing} title="Убрать вложение" aria-label="Убрать вложение" onClick={()=>void removeAttachment()}><Icon name="close"/></button></div>:null}
 <form className="composer composer-with-attachment" onSubmit={submit}><button type="button" className="icon-button" title="Добавить вложение" aria-label="Добавить вложение" disabled={busy||Boolean(attachment)||removing} aria-expanded={picker} onClick={()=>setPicker(v=>!v)}><Icon name="attach"/></button><textarea aria-label="Сообщение" value={body} onChange={e=>setBody(e.target.value)} placeholder="Сообщение" rows={1} maxLength={4000}/><button className="send-button" type="submit" aria-label="Отправить сообщение" disabled={busy||removing||(!body.trim()&&!attachment)}><Icon name="send"/></button></form></>;
}
