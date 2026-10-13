// Build the complete browser module graph; Go embeds the generated output.
import {buildBrowser} from './browser-build'
import groups from './browser-groups.json'
import {createHash} from 'node:crypto'
import {readFile} from 'node:fs/promises'
import {resolve} from 'node:path'
import {writeChanged} from './io'
const development=process.argv.includes('--dev')
const selected=process.argv.slice(2).filter(argument=>argument!=='--dev')
const javascriptGroups=new Set([...Object.values(groups),'browser'])
if(selected.some(name=>!javascriptGroups.has(name)&&name!=='css'&&name!=='icons'))throw new Error('Unknown browser asset group')
const all=selected.length===0
if(all||selected.includes('settings'))await import('./generate-settings-go-data')
if(all||selected.includes('account'))await import('./generate-tracker-go-data')
if(all||selected.includes('icons'))await import('./build-go-icons')
if(all||selected.includes('icons')||selected.some(name=>javascriptGroups.has(name)))await buildBrowser(development)
if(all||selected.includes('css'))await import('./build-go-css')
if(development){
  const base=resolve(import.meta.dirname,'..')
  const hash=createHash('sha256')
  for(const name of ['browser-artifacts.json','../internal/frontend/static/utilities.css','../internal/frontend/static/lucide-icons.json'])hash.update(await readFile(resolve(base,name)))
  await writeChanged(resolve(base,'../internal/frontend/browser-build.json'),JSON.stringify({revision:hash.digest('hex')})+'\n')
}
