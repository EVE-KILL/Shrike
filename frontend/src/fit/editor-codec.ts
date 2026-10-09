import type {Fit} from './types'
import type {SdeData} from '../../dogma/src/metadata'
import {encodeFitText} from './codec'
import {isEditorSlot,isEditorState} from './editor-model'

// Preserve the existing v3 CSV/gzip document format. Keep the shared encoder
// also used for killmail-to-editor handoffs; both paths emit identical bytes.
export function encodeEditorFit(fit:Fit):Promise<string> {
 let body = `ship,${fit.shipTypeId},${fit.name},${fit.description}\n`
 for(const module of fit.modules) body += `module,${module.slot.type},${module.slot.index},${module.typeId},${module.state},${module.charge?.typeId ?? ''}\n`
 for(const drone of fit.drones) body += `drone,${drone.typeId},${drone.states.Active},${drone.states.Passive}\n`
 for(const cargo of fit.cargo) body += `cargo,${cargo.typeId},${cargo.quantity}\n`
 return encodeFitText(body)
}
async function decompress(input:string):Promise<string> {
 const bytes = Uint8Array.from(atob(input.replace(/ /g,'+')),char=>char.charCodeAt(0)),reader = new Blob([bytes]).stream().pipeThrough(new DecompressionStream('gzip')).getReader(),chunks:Uint8Array[] = []
 let length = 0
 try {while(true) {const chunk = await reader.read();if(chunk.done) break;length += chunk.value.byteLength;if(length > 1_048_576) throw new Error('Fitting document is too large');chunks.push(chunk.value)}}finally {await reader.cancel().catch(()=>{})}
 const result = new Uint8Array(length);let offset = 0;for(const chunk of chunks) {result.set(chunk,offset);offset += chunk.byteLength}return new TextDecoder().decode(result)
}
export async function decodeEditorFit(input:string,sde:Pick<SdeData,'types'>,baseURL = 'http://localhost/'):Promise<Fit|null> {
 try {
  const value = input.includes('?') ? new URL(input,baseURL).searchParams.get('fit') : input
  if(!value?.startsWith('v3:') || value.length > 32768) return null
  const lines = (await decompress(value.slice(3))).trim().split('\n');if(lines.length > 500 || !lines[0]?.startsWith('ship,')) return null
  const header = lines[0].split(','),shipTypeId = Number(header[1]);if(!Number.isInteger(shipTypeId) || shipTypeId < 1 || !sde.types.has(shipTypeId)) return null
  const fit:Fit = {shipTypeId,name:header[2] || 'Imported Fit',description:header[3] || '',modules:[],drones:[],cargo:[]}
  for(const line of lines.slice(1)) {
   const parts = line.split(','),typeId = Number(parts[3])
   if(parts[0] === 'module' && isEditorSlot(parts[1]) && Number.isInteger(typeId) && sde.types.has(typeId)) {const index = Number(parts[2]);if(!Number.isInteger(index) || index < 1 || index > 8) continue;const chargeId = Number(parts[5]);fit.modules.push({slot:{type:parts[1],index},typeId,state:isEditorState(parts[4]) ? parts[4] : 'Active',...(chargeId > 0 && sde.types.has(chargeId) ? {charge:{typeId:chargeId}} : {})})}
   else if(parts[0] === 'drone') {const id = Number(parts[1]),active = Number(parts[2]),passive = Number(parts[3]);if(sde.types.has(id) && [active,passive].every(n=>Number.isInteger(n) && n >= 0 && n <= 1000)) fit.drones.push({typeId:id,states:{Active:active,Passive:passive}})}
   else if(parts[0] === 'cargo') {const id = Number(parts[1]),quantity = Number(parts[2]);if(sde.types.has(id) && Number.isInteger(quantity) && quantity > 0 && quantity <= 1_000_000) fit.cargo.push({typeId:id,quantity})}
  }
  return fit
 }catch{return null}
}
