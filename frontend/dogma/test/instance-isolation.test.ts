import {expect,test} from 'vitest'
import {readFile} from 'node:fs/promises'
import {createDogma} from '../src/core'
test('separate factories retain their own WASM memory and loaded SDE',async()=>{
 const io={loadSdeBytes:()=>readFile(new URL('../dist/upstream/data/dist/sde.dat',import.meta.url)),loadEngineBytes:()=>readFile(new URL('../dist/upstream/dogma-engine/esf_dogma_engine_bg.wasm',import.meta.url))}
 const first=createDogma(io),second=createDogma(io)
 await Promise.all([first.getEngine(),second.getEngine()])
 const fit={ship_type_id:587,modules:[],drones:[]}
 const one=await first.calculateFit(fit),two=await second.calculateFit(fit),again=await first.calculateFit(fit)
 expect(await first.getHullStat(one,'ehp')).toBe(await second.getHullStat(two,'ehp'))
 expect(await first.getHullStat(again,'ehp')).toBe(await first.getHullStat(one,'ehp'))
})
