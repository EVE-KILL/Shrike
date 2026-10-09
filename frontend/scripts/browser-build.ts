import {build} from 'esbuild'
import {createHash} from 'node:crypto'
import {mkdir,readFile,readdir,unlink} from 'node:fs/promises'
import {basename,dirname,relative,resolve} from 'node:path'
import {writeChanged} from './io'
import {shellControllers} from './browser-entries'

const base=resolve(import.meta.dirname,'..')
const output=resolve(base,'../internal/frontend/static')
const manifestPath=resolve(base,'browser-artifacts.json')
type Artifact={path:string;bytes:number;sha256:string}
type Manifest={version:number;mode:'development'|'production';entries:string[];outputs:Artifact[]}

export async function buildBrowser(development=false){
  const folder=resolve(base,'src/browser')
  const sources=Object.fromEntries((await readdir(folder)).filter(name=>/\.(js|ts)$/.test(name)&&!name.endsWith('.d.ts'))
    .sort().map(name=>[basename(name).replace(/\.(js|ts)$/,''),resolve(folder,name)]))
  const entries=Object.fromEntries(Object.entries(sources).filter(([name])=>!shellControllers.has(name)))
  const {preferences,...modules}=entries
  if(!preferences)throw new Error('Missing synchronous preferences entry')
  const result=await build({
    entryPoints:modules,outdir:output,absWorkingDir:base,bundle:true,splitting:true,
    platform:'browser',format:'esm',target:'es2022',minify:!development,
    sourcemap:development?'linked':false,write:false,metafile:true,
    chunkNames:'chunks/[name]-[hash]',tsconfig:resolve(base,'tsconfig.json'),
    banner:{js:'// Generated browser asset. Edit frontend/src/browser and run npm --prefix frontend run build.'},
    plugins:[{name:'same-origin-module-imports',setup(builder){
      builder.onResolve({filter:/^\/assets\/.+\.js$/},args=>{
        const name=basename(args.path,'.js'),source=sources[name]
        if(!source)return {errors:[{text:`Unknown browser module: ${args.path}`}]}
        return {path:source}
      })
    }}],
  })
  // Appearance runs synchronously before paint; it must not import shared chunks.
  const appearance=await build({entryPoints:[preferences],outfile:resolve(output,'preferences.js'),
    absWorkingDir:base,bundle:true,platform:'browser',format:'iife',target:'es2022',
    minify:!development,sourcemap:development?'linked':false,write:false,metafile:true,
    banner:{js:'// Generated synchronous appearance bootstrap. Edit frontend/src/browser/preferences.ts.'},
  })
  const previous:Manifest=JSON.parse(await readFile(manifestPath,'utf8').catch(()=>'{"outputs":[]}'))
  const artifacts:Artifact[]=[]
  for(const file of [...result.outputFiles!,...appearance.outputFiles!]){
    await mkdir(dirname(file.path),{recursive:true})
    await writeChanged(file.path,file.text)
    artifacts.push({path:relative(output,file.path).replaceAll('\\','/'),bytes:file.contents.byteLength,
      sha256:createHash('sha256').update(file.contents).digest('hex')})
  }
  const retained=new Set(artifacts.map(file=>file.path))
  for(const old of previous.outputs){
    if(retained.has(old.path))continue
    const path=resolve(output,old.path)
    if(!path.startsWith(output+'/')||!/^.+\.js(?:\.map)?$/.test(path))throw new Error('Invalid generated artifact path')
    const data=await readFile(path).catch(()=>null)
    if(data&&createHash('sha256').update(data).digest('hex')!==old.sha256)throw new Error(`Generated asset was edited directly: ${old.path}`)
    await unlink(path).catch(error=>{if(error.code!=='ENOENT')throw error})
  }
  const manifest:Manifest={version:1,mode:development?'development':'production',entries:Object.keys(entries).sort(),outputs:artifacts.sort((a,b)=>a.path.localeCompare(b.path,'en'))}
  await writeChanged(manifestPath,JSON.stringify(manifest,null,2)+'\n')
  await mkdir(resolve(base,'../.data'),{recursive:true})
  await writeChanged(resolve(base,'../.data/browser-metafile.json'),JSON.stringify({inputs:{...result.metafile!.inputs,...appearance.metafile!.inputs},outputs:{...result.metafile!.outputs,...appearance.metafile!.outputs}},null,2)+'\n')
  const scripts=artifacts.filter(file=>file.path.endsWith('.js'))
  console.log(`Browser: ${manifest.entries.length} entries, ${scripts.length-manifest.entries.length} shared chunks, ${scripts.reduce((sum,a)=>sum+a.bytes,0).toLocaleString()} JS bytes (${manifest.mode})`)
}
