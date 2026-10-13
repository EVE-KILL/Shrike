import {isRecord} from './json'
// Go seeds URL values as arrays. Keep only scalar strings accepted by URLSearchParams.
export function readQuerySeed(text:string|undefined):Record<string,string>{
  let value:unknown;try{value=JSON.parse(text??'{}')}catch{return {}}
  if(!isRecord(value))return {}
  return Object.fromEntries(Object.entries(value).flatMap(([key,item])=>Array.isArray(item)&&typeof item[0]==='string'?[[key,item[0]]]:typeof item==='string'?[[key,item]]:[]))
}
