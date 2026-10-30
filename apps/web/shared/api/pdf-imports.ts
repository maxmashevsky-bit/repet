import { request } from "./client";
import type { CatalogPage, LearningLesson, LessonInput } from "./catalog";

export type PDFAsset = { id:string; kind:"page"|"figure"; page_number:number; ordinal:number; width:number; height:number; bytes:number };
export type PDFPage = { number:number; original_text:string; image_id:string; ocr_required:boolean; corrected_text:string|null; reviewed_at:string|null };
export type PDFImport = { id:string; filename:string; sha256:string; source_bytes:number; rights_status:string; license:string; source_label:string; status:"queued"|"extracting"|"needs_review"|"ocr_required"|"failed"; warnings:string[]; failure:string; created_at:string; pages:PDFPage[]; assets:PDFAsset[]; lesson_ids:string[] };
export type PDFSource = { source:PDFImport; revision:number; reviewed:boolean };
const base="/api/v1/pdf-imports";
function write<T>(path:string,body:object,method="POST"){return request<T>(path,{method,body:JSON.stringify(body)});}
export const pdfApi={
 list:(offset=0,signal?:AbortSignal)=>request<CatalogPage<PDFImport>>(`${base}?limit=20&offset=${offset}`,{signal}),
 get:(id:string,signal?:AbortSignal)=>request<PDFImport>(`${base}/${encodeURIComponent(id)}`,{signal}),
 upload:(input:{request_id:string;filename:string;data_base64:string;rights_status:string;license:string;source_label:string})=>write<PDFImport>(base,input),
 page:(id:string,number:number,text:string)=>write<PDFImport>(`${base}/${encodeURIComponent(id)}/pages/${number}`,{text},"PUT"),
 crop:(id:string,input:{page_number:number;x:number;y:number;width:number;height:number})=>write<PDFImport>(`${base}/${encodeURIComponent(id)}/figures`,input),
 lesson:(id:string,course_id:string,lesson:LessonInput)=>write<LearningLesson>(`${base}/${encodeURIComponent(id)}/lessons`,{course_id,lesson}),
 source:(lesson:string,signal?:AbortSignal)=>request<PDFSource>(`/api/v1/catalog/lessons/${encodeURIComponent(lesson)}/pdf-source`,{signal}),
 approve:(lesson:string,revision:number)=>write<PDFSource>(`/api/v1/catalog/lessons/${encodeURIComponent(lesson)}/pdf-review`,{revision,confirmed:true})
};
