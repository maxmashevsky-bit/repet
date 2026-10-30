"use client";
import { useEffect,useRef,useState } from "react";
import { reviewApi,type RewardPreference,type RewardSummary } from "../../shared/api/reviews";
import { Icon } from "../ui/Icon";
import { parseRewardPreference,parseRewardSummary } from "./model";
import { ReviewError,ReviewPager } from "./ReviewUI";
export function RewardsPanel({studentID,tutor=false,revision=0}:{studentID:string;tutor?:boolean;revision?:number}){
 const [offset,setOffset]=useState(0),[retry,setRetry]=useState(0),[busy,setBusy]=useState(false),[error,setError]=useState<unknown>();const gate=useRef(false);
 const [data,setData]=useState<{key:string;rewards?:RewardSummary;preference?:RewardPreference;error?:unknown}>({key:""}),key=`${studentID}:${tutor}:${revision}:${retry}:${offset}`;
 useEffect(()=>{const c=new AbortController();Promise.all([reviewApi.rewards(studentID,offset,c.signal).then(parseRewardSummary),tutor?reviewApi.preference(studentID,c.signal).then(parseRewardPreference):Promise.resolve(undefined)]).then(([rewards,preference])=>{if(!c.signal.aborted)setData({key,rewards,preference});}).catch((error:unknown)=>{if(!c.signal.aborted)setData({key,error});});return()=>c.abort();},[key,studentID,tutor,offset]);
 async function toggle(enabled:boolean){
  if(gate.current)return;gate.current=true;setBusy(true);setError(undefined);
  const previous=data.preference;let saved:RewardPreference|undefined;
  setData(old=>({...old,preference:old.preference?{...old.preference,enabled}:undefined}));
  try{saved=parseRewardPreference(await reviewApi.savePreference(studentID,enabled));const rewards=parseRewardSummary(await reviewApi.rewards(studentID,offset));setData({key,rewards,preference:saved});}
  catch(e){setData(old=>({...old,preference:saved??previous}));setError(e);}finally{gate.current=false;setBusy(false);}
 }
 const rewards=data.rewards;
 if(data.key!==key)return <p role="status">Загружаем достижения…</p>;
 if(data.error)return <ReviewError error={data.error} retry={()=>setRetry(n=>n+1)}/>;
 if(!rewards)return null;if(!rewards.enabled&&!tutor)return null;
 return <section className="review-rewards"><header><h2>Достижения</h2>{tutor&&data.preference?<label className="review-reward-toggle"><input type="checkbox" checked={data.preference.enabled} disabled={busy} onChange={e=>void toggle(e.target.checked)}/><span>XP и награды</span></label>:null}</header>{error?<ReviewError error={error}/>:null}{rewards.enabled?<><div className="review-xp"><div className="review-level"><Icon name="shield" size={28}/><span>Уровень <strong>{rewards.level}</strong></span></div><div><div className="review-xp-label"><strong>{rewards.xp} XP</strong><span>{rewards.level_progress} / {rewards.level_target} до следующего уровня</span></div><progress max={rewards.level_target} value={rewards.level_progress} aria-label="Прогресс уровня"/></div><p>За неделю: <strong>{rewards.weekly_xp} XP</strong> · достижений: {rewards.weekly_achievements}</p></div>{rewards.awards.items.length?<><div className="review-awards">{rewards.awards.items.map(award=><article key={award.id}><span className="review-award-icon"><Icon name="check" size={22}/></span><div><h3>{award.title}</h3><p>{new Date(award.earned_at).toLocaleDateString("ru-RU")} · {award.points} XP</p></div></article>)}</div><ReviewPager page={rewards.awards} change={setOffset}/></>:<p className="muted">Достижений пока нет.</p>}</>:<p className="muted">XP и награды отключены.</p>}</section>;
}
