/** Filesystem adapter for browser engine regression tests. */
import {readFile} from 'node:fs/promises'
import {createDogma} from './core'
const base=new URL('../dist/upstream/',import.meta.url)
const dogma=createDogma({
 loadSdeBytes:()=>readFile(new URL('data/dist/sde.dat',base)),
 loadEngineBytes:()=>readFile(new URL('dogma-engine/esf_dogma_engine_bg.wasm',base)),
})
export const {calculateFit,loadSde,loadEngine,getEngine,getItemStat,getHullStat,getHullStats}=dogma
export * from './types'
export {fittingToEsfFit,type FittingItemRow,type FittingToEsfOptions} from './converter'
export type {SdeData,LoadedEngine,RawSdeBytes,DogmaIO,DogmaApi} from './core'
