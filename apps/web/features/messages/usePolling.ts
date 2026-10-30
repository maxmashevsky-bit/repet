"use client";
import { useCallback,useEffect,useRef,useState } from "react";
import { startPolling } from "./polling.ts";
export function usePolling(load:(signal:AbortSignal)=>Promise<void>,enabled=true,intervalMs=5000,onError?:(error:unknown)=>void){
 const callback=useRef(load),errorCallback=useRef(onError);
 const[revision,setRevision]=useState(0);
 useEffect(()=>{callback.current=load;errorCallback.current=onError;},[load,onError]);
 useEffect(()=>{if(!enabled)return;return startPolling(signal=>callback.current(signal),{
  visible:()=>document.visibilityState==="visible",
  subscribe:listener=>{document.addEventListener("visibilitychange",listener);return()=>document.removeEventListener("visibilitychange",listener);},
  delay:(run,ms)=>window.setTimeout(run,ms),cancel:timer=>window.clearTimeout(timer as number)
 },intervalMs,error=>errorCallback.current?.(error));},[enabled,intervalMs,revision]);
 return useCallback(()=>setRevision(n=>n+1),[]);
}
