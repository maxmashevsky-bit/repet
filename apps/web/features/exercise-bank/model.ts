export const directionLabels = {school:"Школьная математика",olympiad:"Олимпиадная математика",oge:"ОГЭ",ege_basic:"ЕГЭ · базовая математика",ege_profile:"ЕГЭ · профильная математика"};
export function moveSelection<T>(items:T[],index:number,delta:number):T[]{
 const target=index+delta;if(!Number.isInteger(index)||!Number.isInteger(target)||index<0||index>=items.length||target<0||target>=items.length)return items;
 const result=[...items];[result[index],result[target]]=[result[target],result[index]];return result;
}
