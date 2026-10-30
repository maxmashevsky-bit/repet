"use client";

import { useEffect,useId,useState } from "react";
import type { LearningExample } from "../../shared/api/catalog";
import { NumberLineInput } from "../exercise-engine/VisualInputs";
import { Icon } from "../ui/Icon";

export function AnimatedExample({example}:{example:LearningExample}){
 const [frame,setFrame]=useState(0),[playing,setPlaying]=useState(false);
 const [point,setPoint]=useState(String(example.steps[0].value));const id=useId();
 function select(index:number){setFrame(index);setPoint(String(example.steps[index].value));}
 useEffect(()=>{if(!playing)return;const timer=setTimeout(()=>{const next=frame+1;if(next>=example.steps.length){setPlaying(false);return;}setFrame(next);setPoint(String(example.steps[next].value));},1200);return()=>clearTimeout(timer);},[playing,frame,example.steps]);
 return <section className="player-example" aria-label="Интерактивный пример"><h2>{example.title}</h2><p className="material-prose">{example.explanation}</p>
  <NumberLineInput exercise={{id:"00000000-0000-4000-8000-000000000001",kind:"number_line",prompt:example.title,options:[],difficulty:1,position:0,config:{min:example.min,max:example.max,step:1}}} value={{value:point}} onChange={answer=>{setPlaying(false);setPoint(answer.value??"");}} disabled={false} errorID={id} invalid={false}/>
  <div className="player-example-caption" role="status"><strong>Шаг {frame+1} из {example.steps.length}</strong><p>{example.steps[frame].text}</p></div>
  <div className="player-example-controls"><button type="button" className="icon-button" title="Предыдущий шаг примера" aria-label="Предыдущий шаг примера" disabled={frame===0} onClick={()=>{setPlaying(false);select(frame-1);}}><Icon name="arrow" className="back" size={18}/></button><button type="button" className="secondary" onClick={()=>{if(playing)setPlaying(false);else{if(frame===example.steps.length-1)select(0);setPlaying(true);}}}><Icon name={playing?"close":"arrow"} size={17}/>{playing?"Остановить":"Запустить пример"}</button><button type="button" className="icon-button" title="Следующий шаг примера" aria-label="Следующий шаг примера" disabled={frame===example.steps.length-1} onClick={()=>{setPlaying(false);select(frame+1);}}><Icon name="arrow" size={18}/></button></div>
  <label>Шаг примера<input type="range" min={0} max={example.steps.length-1} step={1} value={frame} onChange={e=>{setPlaying(false);select(Number(e.target.value));}}/></label>
 </section>;
}
