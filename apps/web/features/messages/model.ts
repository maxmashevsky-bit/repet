import type { Message } from "../../shared/api/types";
export function mergeMessages(current:Message[],incoming:Message[],conversationID:string):Message[]{
 const byID=new Map<string,Message>();
 for(const item of [...current,...incoming])if(item.conversation_id===conversationID)byID.set(item.id,item);
 return [...byID.values()].sort((a,b)=>typeof a.sequence==="number"&&typeof b.sequence==="number"?a.sequence-b.sequence:Date.parse(a.created_at)-Date.parse(b.created_at)||a.id.localeCompare(b.id));
}
