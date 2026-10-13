import {readFile,writeFile} from 'node:fs/promises'
export async function writeChanged(path:string|URL,content:string){
  const previous=await readFile(path,'utf8').catch(()=>null)
  if(previous!==content)await writeFile(path,content)
}
