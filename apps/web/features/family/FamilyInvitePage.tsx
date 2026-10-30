"use client";
import Link from "next/link";
import { useEffect,useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError } from "../../shared/api/client";
import { familyApi,type GuardianLink } from "../../shared/api/family";
import { Brand } from "../ui/Brand";
import { FamilyError } from "./FamilyUI";
import { parseGuardianLink } from "./model";
export function FamilyInvitePage(){
 const router=useRouter(),[retry,setRetry]=useState(0),[data,setData]=useState<{key:number;link?:GuardianLink;error?:unknown}>({key:-1}),[busy,setBusy]=useState(false),[error,setError]=useState<unknown>();
 useEffect(()=>{let alive=true;const token=new URLSearchParams(location.hash.slice(1)).get("token")??"";familyApi.preview(token).then(parseGuardianLink).then(link=>{if(alive)setData({key:retry,link});}).catch((error:unknown)=>{if(alive)setData({key:retry,error});});return()=>{alive=false;};},[retry]);
 async function accept(e:React.FormEvent<HTMLFormElement>){e.preventDefault();if(busy)return;setBusy(true);setError(undefined);try{const token=new URLSearchParams(location.hash.slice(1)).get("token")??"",link=parseGuardianLink(await familyApi.accept(token));history.replaceState(null,"","/family-invite");router.push(`/family/${link.student_id}`);}catch(e){setError(e);setBusy(false);}}
 return <main className="family-invitation"><Brand/><h1>Родительский доступ</h1>{data.key!==retry?<p role="status">Проверяем приглашение…</p>:data.error instanceof ApiError&&data.error.status===401?<><p>Нужен вход с приглашённым email.</p><div className="actions"><Link href="/login" target="_blank" rel="noopener" className="button">Войти</Link><Link href="/register" target="_blank" rel="noopener" className="button secondary">Создать аккаунт</Link></div><button type="button" className="secondary" onClick={()=>setRetry(n=>n+1)}>Проверить аккаунт</button></>:data.error?<FamilyError error={data.error} retry={()=>setRetry(n=>n+1)}/>:data.link?<form onSubmit={accept} className="form-stack"><h2>{data.link.student_name}</h2><p>{data.link.guardian_email}</p><label className="family-consent"><input type="checkbox" required/><span>Подтверждаю родительскую связь и получение доступа к учебным данным.</span></label><button type="submit" disabled={busy}>{busy?"Подтверждаем…":"Подтвердить доступ"}</button></form>:null}{error?<FamilyError error={error}/>:null}</main>;
}
