import { request } from "./client";
import type { Assignment } from "./types";
import type { CatalogPage, CatalogQuery, CatalogStatus, CatalogTrack, LearningLesson } from "./catalog";

export type HTMLVersion = { id:string;number:number;topic_id:string;title:string;summary:string;track:CatalogTrack;grade_min:number;grade_max:number;difficulty:number;filename:string;sha256:string;metadata:{title:string;description:string;language:string;scripts:number;external_resources:number;bytes:number};rights_status:"own"|"permission"|"open_license";license:string;source_label:string;published_at:string|null };
export type HTMLLesson = {id:string;course_id:string;author_id:string;author_name:string;status:CatalogStatus;has_unpublished_changes:boolean;version:HTMLVersion};
export type HTMLInput = Pick<HTMLVersion,"topic_id"|"title"|"summary"|"track"|"grade_min"|"grade_max"|"difficulty"|"filename"|"rights_status"|"license"|"source_label"> & {course_id:string;source:string};
const base="/api/v1/html-lessons";
const write=<T>(path:string,body:object)=>request<T>(base+path,{method:"POST",body:JSON.stringify(body)});
const query=(values:object)=>{const params=new URLSearchParams();for(const [key,value]of Object.entries(values))if(value!==""&&value!==undefined)params.set(key,String(value));return params.size?"?"+params:"";};
export const htmlApi={
 list:(filter:CatalogQuery={},signal?:AbortSignal)=>request<CatalogPage<HTMLLesson>>(base+query(filter),{signal}),
 get:(id:string,version_id="",assignment_id="",signal?:AbortSignal)=>request<HTMLLesson>(`${base}/${encodeURIComponent(id)}${query({version_id,assignment_id})}`,{signal}),
 import:(input:HTMLInput,request_id:string,id="")=>write<HTMLLesson>(id?`/${encodeURIComponent(id)}/versions`:"",{...input,request_id}),
 publish:(id:string,version_id:string)=>write<HTMLLesson>(`/${encodeURIComponent(id)}/publish`,{version_id}),
 archive:(id:string)=>write<unknown>(`/${encodeURIComponent(id)}/archive`,{}),
 versions:(id:string,signal?:AbortSignal)=>request<{items:HTMLVersion[]}>(`${base}/${encodeURIComponent(id)}/versions`,{signal}),
 source:(id:string,version_id:string)=>request<{source:string}>(`${base}/${encodeURIComponent(id)}/source${query({version_id})}`),
 preview:(id:string,version_id:string,assignment_id="",signal?:AbortSignal)=>writePreview(id,version_id,assignment_id,signal),
 assign:(id:string,student_id:string,due_at:string,request_id:string)=>write<Assignment>(`/${encodeURIComponent(id)}/assign`,{student_id,due_at,request_id}),
 convert:(id:string,version_id:string)=>write<LearningLesson>(`/${encodeURIComponent(id)}/convert`,{version_id})
};
function writePreview(id:string,version_id:string,assignment_id:string,signal?:AbortSignal){return request<{url:string;expires_in:number}>(`${base}/${encodeURIComponent(id)}/preview`,{method:"POST",body:JSON.stringify({version_id,assignment_id}),signal});}
