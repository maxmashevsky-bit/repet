export type PollEnvironment={visible:()=>boolean;subscribe:(onChange:()=>void)=>()=>void;delay:(run:()=>void,ms:number)=>unknown;cancel:(timer:unknown)=>void};
export function startPolling(load:(signal:AbortSignal)=>Promise<void>,env:PollEnvironment,intervalMs=5000,onError?:(error:unknown)=>void){
 if(!Number.isFinite(intervalMs)||intervalMs<1000)throw new Error("Invalid polling interval");
 let stopped=false,running=false,failures=0,timer:unknown,controller:AbortController|undefined;
 function clear(){if(timer!==undefined){env.cancel(timer);timer=undefined;}}
 function schedule(ms:number){clear();if(!stopped&&env.visible())timer=env.delay(()=>{timer=undefined;void run();},ms);}
 async function run(){
  if(stopped||running||!env.visible())return;
  running=true;const current=new AbortController();controller=current;
  try{await load(current.signal);if(!current.signal.aborted)failures=0;}
  catch(error){if(!stopped&&!current.signal.aborted){failures=Math.min(6,failures+1);onError?.(error);}}
  finally{running=false;if(controller===current)controller=undefined;schedule(current.signal.aborted?0:Math.min(60000,intervalMs*2**failures));}
 }
 const unsubscribe=env.subscribe(()=>{clear();if(!env.visible())controller?.abort();else if(!running)void run();});
 if(env.visible())void run();
 return()=>{stopped=true;clear();controller?.abort();unsubscribe();};
}
