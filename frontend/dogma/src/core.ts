/** Platform-independent facade over Shrike's shared Rust adapter. */
import * as engineBg from '../dist/upstream/dogma-engine/esf_dogma_engine_bg.js'
import version from '../dist/upstream/VERSION.json'
import {decodeSdeFromBytes,type SdeData} from './metadata'
import {allFive} from './skills'
import type {ComputedItem,EsfFit,FitStats,SkillMap} from './types'
export {decodeSdeFromBytes,type SdeData} from './metadata'
export type RawSdeBytes=Uint8Array|ArrayBuffer
export interface LoadedEngine {
 init:()=>void;load_sde:(bytes:Uint8Array)=>number;engine_version:()=>string;
 calculate:(fit:EsfFit,skills?:SkillMap,canonical?:boolean)=>FitStats;
}
export async function instantiateEngineFromBytes(bytes:Uint8Array|ArrayBuffer):Promise<LoadedEngine>{
 const owned=new Uint8Array(bytes.byteLength);owned.set(bytes instanceof Uint8Array?bytes:new Uint8Array(bytes))
 const module=await WebAssembly.compile(owned)
 const instance=await WebAssembly.instantiate(module,{'./esf_dogma_engine_bg.js':engineBg as unknown as WebAssembly.ModuleImports})
 const bind=()=>engineBg.__wbg_set_wasm(instance.exports)
 bind();(instance.exports as{__wbindgen_start?:()=>void}).__wbindgen_start?.()
 const invoke=<Args extends unknown[],Result>(fn:(...args:Args)=>Result)=>(...args:Args):Result=>{bind();return fn(...args)}
 const engine:LoadedEngine={init:invoke(engineBg.init),load_sde:invoke(engineBg.load_sde),engine_version:invoke(engineBg.engine_version),calculate:invoke((fit,skills,canonical=false)=>engineBg.calculate(fit,skills,canonical))}
 engine.init()
 if(engine.engine_version()!==version.dogma_engine)throw new Error('Fitting engine and data manifest versions differ')
 return engine
}
export interface DogmaIO {loadSdeBytes:()=>Promise<RawSdeBytes>;loadEngineBytes:()=>Promise<Uint8Array|ArrayBuffer>}
export interface DogmaApi {
 calculateFit:(fit:EsfFit,skills?:SkillMap)=>Promise<FitStats>;loadSde:()=>Promise<SdeData>;loadEngine:()=>Promise<LoadedEngine>;
 getEngine:()=>Promise<{sde:SdeData;engine:LoadedEngine}>;
 getItemStat:(item:ComputedItem,name:string)=>Promise<number|null>;
 getHullStat:(stats:FitStats,name:string)=>Promise<number|null>;
 getHullStats:(stats:FitStats,names:readonly string[])=>Promise<Record<string,number|null>>;
}
export function createDogma(io:DogmaIO):DogmaApi {
 let sdeCache:Promise<SdeData>|null=null,engineCache:Promise<LoadedEngine>|null=null,pair:Promise<{sde:SdeData;engine:LoadedEngine}>|null=null
 const loadSde=()=>sdeCache??=io.loadSdeBytes().then(decodeSdeFromBytes)
 const loadEngine=()=>engineCache??=io.loadEngineBytes().then(instantiateEngineFromBytes)
 const getEngine=()=>pair??=Promise.all([loadSde(),loadEngine()]).then(([sde,engine])=>{
  const build=engine.load_sde(sde.bytes)
  if(build!==sde.buildNumber)throw new Error('Fitting catalogue and engine SDE builds differ')
  return{sde,engine}
 })
 const calculateFit=async(fit:EsfFit,skills?:SkillMap)=>{const{sde,engine}=await getEngine();return engine.calculate(fit,skills??allFive(sde))}
 const itemStat=(sde:SdeData,item:ComputedItem,name:string)=>{const id=sde.attributeNameToId.get(name);const attr=id===undefined?undefined:item.attributes.get(id);return attr?.value??attr?.base_value??null}
 const getItemStat=async(item:ComputedItem,name:string)=>itemStat(await loadSde(),item,name)
 const getHullStat=(stats:FitStats,name:string)=>getItemStat(stats.hull,name)
 const getHullStats=async(stats:FitStats,names:readonly string[])=>{const sde=await loadSde();return Object.fromEntries(names.map(name=>[name,itemStat(sde,stats.hull,name)]))}
 return{calculateFit,loadSde,loadEngine,getEngine,getItemStat,getHullStat,getHullStats}
}
