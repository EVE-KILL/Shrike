// The WASM is checked in; reject a Rust edit without rebuilding its artifact.
import {readFile,readdir,writeFile} from 'node:fs/promises'
import {createHash} from 'node:crypto'
import {resolve,relative} from 'node:path'
const root=resolve(import.meta.dirname,'../..'),engine=resolve(root,'engine/dogma')
async function files(dir){
 const out=[]
 for(const e of await readdir(dir,{withFileTypes:true})){
  const path=resolve(dir,e.name)
  if(e.isDirectory()&&e.name!=='target')out.push(...await files(path))
  else if(e.isFile()&&(e.name.endsWith('.rs')||e.name==='Cargo.toml'||e.name==='Cargo.lock'||e.name==='compat.json'))out.push(path)
 }
 return out
}
const hash=createHash('sha256')
for(const file of(await files(engine)).filter(f=>!f.endsWith('/src/main.rs')&&!f.endsWith('/src/fields.rs')).sort()){
 hash.update(relative(engine,file)+'\0');hash.update(await readFile(file));hash.update('\0')
}
const artifactDir=resolve(root,'frontend/dogma/dist/upstream/dogma-engine')
const result={source_sha256:hash.digest('hex'),wasm_sha256:createHash('sha256').update(await readFile(resolve(artifactDir,'esf_dogma_engine_bg.wasm'))).digest('hex'),glue_sha256:createHash('sha256').update(await readFile(resolve(artifactDir,'esf_dogma_engine_bg.js'))).digest('hex')}
const path=resolve(artifactDir,'SOURCE.json')
if(process.argv.includes('--check')){
 const expected=JSON.parse(await readFile(path,'utf8'))
 for(const key of Object.keys(result))if(result[key]!==expected[key])throw new Error('Dogma WASM is stale; run make build-dogma-wasm')
 console.log('Dogma WASM source and artifact checksums match')
}else await writeFile(path,JSON.stringify(result,null,2)+'\n')
