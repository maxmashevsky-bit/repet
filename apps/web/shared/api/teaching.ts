import {request} from "./client";
export type TeachingPreferences={revision:number;subjects:string[];grades:number[];default_duration_minutes:number};
export const teachingApi={
 get:(signal?:AbortSignal)=>request<TeachingPreferences>("/api/v1/teaching-preferences",{signal}),
 save:(input:TeachingPreferences)=>request<TeachingPreferences>("/api/v1/teaching-preferences",{method:"PUT",body:JSON.stringify(input)})
};
