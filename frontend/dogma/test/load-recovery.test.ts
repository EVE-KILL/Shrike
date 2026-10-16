import {readFile} from 'node:fs/promises'
import {resolve} from 'node:path'
import {expect,test,vi} from 'vitest'
import {createDogma} from '../src/core'

const dataPath=resolve(import.meta.dirname,'../dist/upstream')
test('catalogue and engine retries clear failed promises and retain successful shared loads',async()=>{
 const catalogue=vi.fn(()=>readFile(resolve(dataPath,'data/dist/sde.dat'))),wasm=vi.fn(()=>readFile(resolve(dataPath,'dogma-engine/esf_dogma_engine_bg.wasm')))
 catalogue.mockRejectedValueOnce(new Error('Catalogue offline'));wasm.mockRejectedValueOnce(new Error('Engine offline'))
 const dogma=createDogma({loadSdeBytes:catalogue,loadEngineBytes:wasm}),first=dogma.getEngine()
 expect(dogma.getEngine()).toBe(first);await expect(first).rejects.toThrow('Catalogue offline')
 const recovered=await dogma.getEngine();expect(recovered.sde.buildNumber).toBeGreaterThan(0);expect(catalogue).toHaveBeenCalledTimes(2);expect(wasm).toHaveBeenCalledTimes(2)
 expect(await dogma.getEngine()).toBe(recovered);expect(await dogma.loadSde()).toBe(recovered.sde);expect(await dogma.loadEngine()).toBe(recovered.engine)
 expect(catalogue).toHaveBeenCalledTimes(2);expect(wasm).toHaveBeenCalledTimes(2)
})
