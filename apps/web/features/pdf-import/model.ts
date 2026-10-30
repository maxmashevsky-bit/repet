import type { Exercise } from "../../shared/api/catalog";

export function suggestPDFExercises(text:string):{prompt:string;kind:Exercise["kind"]}[]{
 const chunks=text.replaceAll("\f","").split(/\n(?=\s*\d{1,3}[.)]\s)/u).map(t=>t.trim()).filter(Boolean);
 return chunks.slice(0,50).map(prompt=>({prompt,kind:/докаж|объясн|обосну|prove|explain/iu.test(prompt)?"long_text":/найд|вычисл|сколько|числ|calculate|find/iu.test(prompt)?"numeric":"text"}));
}
