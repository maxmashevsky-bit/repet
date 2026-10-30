import { request } from "./client";
import type { ChatAttachment,ChatFile } from "./types";

const base=(conversation:string)=>`/api/v1/messages/conversations/${encodeURIComponent(conversation)}`;
export type ChatAssignmentPreview={id:string;title:string;body:string;kind:"text"|"material";status:string;due_at:string};
export const chatApi={
 upload:(conversation:string,filename:string,data:string,request_id:string)=>request<ChatFile>(base(conversation)+"/files",{method:"POST",body:JSON.stringify({filename,data,request_id})}),
 remove:(conversation:string,id:string)=>request<{status:string}>(`${base(conversation)}/files/${encodeURIComponent(id)}`,{method:"DELETE"}),
 download:(conversation:string,id:string)=>`${base(conversation)}/files/${encodeURIComponent(id)}`,
 references:(conversation:string,kind:"assignment"|"lesson",q:string,offset:number,signal?:AbortSignal)=>request<{items:ChatAttachment[]}>(`${base(conversation)}/references?${new URLSearchParams({kind,q,offset:String(offset)})}`,{signal}),
 assignment:(conversation:string,id:string,signal?:AbortSignal)=>request<ChatAssignmentPreview>(`${base(conversation)}/assignments/${encodeURIComponent(id)}`,{signal})
};
