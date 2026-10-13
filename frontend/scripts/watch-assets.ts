import {watch} from 'node:fs'
import {spawn,type ChildProcess} from 'node:child_process'
import {resolve} from 'node:path'
const base=resolve(import.meta.dirname,'..')
let child:ChildProcess|undefined,timer:ReturnType<typeof setTimeout>|undefined,queued=false,closed=false
async function rebuild(){
  if(closed)return
  if(child){queued=true;return}
  console.log('[browser] Rebuilding source assets…')
  child=spawn(process.execPath,['--import','tsx','scripts/build-assets.ts','--dev'],{cwd:base,stdio:'inherit'})
  const code=await new Promise<number|null>(done=>{
    child!.once('error',()=>done(1));child!.once('exit',done)
  })
  child=undefined
  if(code!==0)console.error('[browser] Build failed; fix the source and save to retry.')
  if(queued){queued=false;await rebuild()}
}
const changed=()=>{if(timer)clearTimeout(timer);timer=setTimeout(()=>{timer=undefined;void rebuild()},250)}
const watchers=['src','shared','styles','scripts','dogma/src'].map(folder=>watch(resolve(base,folder),{recursive:true},changed))
watchers.push(watch(resolve(base,'../internal/frontend'),{recursive:true},(_event,file)=>{
  if(file&&/\.(templ|go)$/.test(file)&&!file.endsWith('_templ.go')&&!file.endsWith('_test.go')&&
    !file.startsWith('static/')&&!file.startsWith('data/'))changed()
}))
for(const file of ['package.json','tsconfig.json','dogma/dist/upstream/VERSION.json','../engine/dogma/compat.json'])watchers.push(watch(resolve(base,file),changed))
watchers.push(watch(resolve(base,'../internal/frontend/static/icons.svg'),changed))
const stop=()=>{closed=true;if(timer)clearTimeout(timer);for(const watcher of watchers)watcher.close();child?.kill('SIGTERM')}
process.once('SIGINT',stop);process.once('SIGTERM',stop)
if(!process.argv.includes('--skip-initial'))await rebuild()
console.log('[browser] Watching JavaScript, TypeScript, styles and generated data sources.')
