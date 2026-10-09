/** Browser entry for the Go-served fitting editor. */
import {createDogma} from './core'
async function bytes(name:string):Promise<ArrayBuffer>{
 const response=await fetch(`/assets/dogma/${name}`)
 if(!response.ok)throw new Error(`Could not load fitting data: ${name}`)
 return response.arrayBuffer()
}
const dogma=createDogma({loadSdeBytes:()=>bytes('sde.dat'),loadEngineBytes:()=>bytes('esf_dogma_engine_bg.wasm')})
export const {loadSde,calculateFit,getHullStats}=dogma
