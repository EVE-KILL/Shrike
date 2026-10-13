import {expect,test} from 'vitest'
import {access,readFile} from 'node:fs/promises'
import {execFileSync} from 'node:child_process'
import {fileURLToPath} from 'node:url'
import {getEngine,getHullStats} from '../src/index'
const binary=fileURLToPath(new URL('../../../bin/shrike-dogma',import.meta.url))
const present=await access(binary).then(()=>true,()=>false)
if(!present&&process.env.DOGMA_REQUIRE_NATIVE==='1')throw new Error('Native engine is required; run make build-dogma')
test.skipIf(!present)('native and browser WASM return the same 49 fields for the reference fits',async()=>{
 const folder=new URL('../../../engine/dogma/',import.meta.url)
 const requests=await readFile(new URL('testdata/requests.jsonl',folder),'utf8')
 const fieldsSource=await readFile(new URL('src/fields.rs',folder),'utf8')
 const fields=[...fieldsSource.matchAll(/\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*,\s*(true|false)/g)].map(m=>({key:m[1]!,name:m[2]!,resist:m[3]==='true'}))
 expect(fields).toHaveLength(49)
 const data=fileURLToPath(new URL('../dist/upstream/data/dist',import.meta.url))
 const native=execFileSync(binary,[data],{input:requests,encoding:'utf8',timeout:30000}).trim().split('\n').map(line=>JSON.parse(line))
 const {engine}=await getEngine()
 const lines=requests.trim().split('\n')
 expect(native).toHaveLength(lines.length)
 for(let i=0;i<lines.length;i++){
  const req=JSON.parse(lines[i]!),computed=engine.calculate(req.fit,req.skills,true)
  const hull=await getHullStats(computed,fields.map(f=>f.name))
  expect(native[i].error).toBeUndefined()
  for(const f of fields){
   const v=hull[f.name],browser=v==null?null:f.resist?1-v:v,server=native[i].hull[f.key]
   if(browser===null)expect(server,`case ${i+1} ${f.key}`).toBeNull()
   else expect(Math.abs(browser-server),`case ${i+1} ${f.key}`).toBeLessThanOrEqual(Math.max(1e-7,Math.abs(browser)*1e-9))
  }
 }
})
