import {readFile,readdir} from 'node:fs/promises'
import {createHash} from 'node:crypto'
import {expect,test} from 'vitest'
import artifacts from '../browser-artifacts.json'
import {shellControllers} from '../scripts/browser-entries'
import {Script} from 'node:vm'

test('all shipped browser modules are generated, current and connected to real files',async()=>{
  const root=new URL('../../internal/frontend/static/',import.meta.url)
  const known=new Set(artifacts.outputs.map(file=>file.path))
  const shipped=(await readdir(root,{recursive:true})).filter(name=>/\.js(?:\.map)?$/.test(name)).sort()
  expect(shipped).toEqual([...known].sort())
  for(const file of artifacts.outputs){
    const data=await readFile(new URL(file.path,root))
    expect(createHash('sha256').update(data).digest('hex'),file.path).toBe(file.sha256)
    expect(data.byteLength).toBe(file.bytes)
    if(!file.path.endsWith('.js'))continue
    for(const match of data.toString().matchAll(/(?:from\s*|import\s*\(|import\s*)["'](\.\.?\/[^"']+\.js)["']/g)){
      const dependency=new URL(match[1]!,new URL(file.path,root))
      const name=decodeURIComponent(dependency.pathname.slice(root.pathname.length))
      expect(known.has(name),`${file.path} imports ${name}`).toBe(true)
    }
  }
  const entries=(await readdir(new URL('../src/browser/',import.meta.url)))
    .filter(file=>/\.(js|ts)$/.test(file)&&!file.endsWith('.d.ts'))
    .map(file=>file.replace(/\.(js|ts)$/,'')).filter(name=>!shellControllers.has(name)).sort()
  expect(entries).toEqual(artifacts.entries)
})
test('appearance remains a standalone synchronous script while the common shell is an ESM entry',async()=>{
  const root=new URL('../../internal/frontend/static/',import.meta.url)
  const appearance=await readFile(new URL('preferences.js',root),'utf8')
  expect(()=>new Script(appearance)).not.toThrow()
  expect(artifacts.entries).toContain('site')
  for(const name of shellControllers)expect(artifacts.entries).not.toContain(name)
})
