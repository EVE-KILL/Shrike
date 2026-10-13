import type {KillmailEditorFitLegacyResponses,ShipMatchupResponses} from '../../../shared/api/types.gen'
import {isRecord} from './json'
import {searchID} from './search-model'

type EditorFit = KillmailEditorFitLegacyResponses[200]
export type Matchup = Pick<ShipMatchupResponses[200],'enough'|'mirror'|'sample'|'attacker_win_rate'|'attacker_wins'|'victim_wins'|'window_days'|'top_fits'>
const nonnegative = (value:unknown)=>typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : 0
export function readEditorFit(value:unknown):EditorFit {
  if(!isRecord(value) || !searchID(value.shipTypeId) || !Array.isArray(value.modules) || !Array.isArray(value.drones)) throw new Error('Could not read fitting')
  const modules:EditorFit['modules'] = value.modules.map((module:unknown)=>{if(!isRecord(module) || !searchID(module.type_id) || typeof module.slot_group !== 'number' || !Number.isSafeInteger(module.slot_group) || module.slot_group < 1 || module.slot_group > 5) throw new Error('Could not read fitting');return {type_id:searchID(module.type_id),slot_group:module.slot_group,ordinal:nonnegative(module.ordinal),name:typeof module.name === 'string' ? module.name : null,charge_type_id:searchID(module.charge_type_id) || null}})
  const drones:EditorFit['drones'] = value.drones.map((drone:unknown)=>{if(!isRecord(drone) || !searchID(drone.type_id) || typeof drone.quantity !== 'number' || !Number.isSafeInteger(drone.quantity) || drone.quantity < 0) throw new Error('Could not read fitting');return {type_id:searchID(drone.type_id),quantity:drone.quantity,name:typeof drone.name === 'string' ? drone.name : null}})
  return {shipTypeId:searchID(value.shipTypeId),name:typeof value.name === 'string' ? value.name : '',modules,drones}
}
export function killFitDocument(data:EditorFit,id:number):string {
  const slotNames = ['High','Medium','Low','Rig','SubSystem'],grouped = new Map<number,EditorFit['modules']>()
  for(const module of data.modules) {const group = grouped.get(module.slot_group) ?? [];group.push(module);grouped.set(module.slot_group,group)}
  let body = `ship,${data.shipTypeId},${data.name || 'Community Fit'},From killmail #${id}\n`
  for(let group=1;group<=5;group++) for(const [index,module] of (grouped.get(group) ?? []).entries()) body += `module,${slotNames[group-1]},${index+1},${module.type_id},Active,${module.charge_type_id ?? ''}\n`
  let budget = 5;for(const drone of data.drones) {const active = Math.min(drone.quantity,budget);budget -= active;body += `drone,${drone.type_id},${active},${drone.quantity-active}\n`}
  return body
}
export function readMatchup(value:unknown):Matchup {
  if(!isRecord(value) || typeof value.enough !== 'boolean') throw new Error('Invalid matchup')
  const top_fits:Matchup['top_fits'] = Array.isArray(value.top_fits) ? value.top_fits.flatMap((fit:unknown)=>{if(!isRecord(fit)) return [];const modules:Matchup['top_fits'][number]['modules'] = Array.isArray(fit.modules) ? fit.modules.flatMap((module:unknown)=>isRecord(module) && searchID(module.type_id) ? [{type_id:searchID(module.type_id),slot_group:nonnegative(module.slot_group),name:typeof module.name === 'string' ? module.name : null}] : []) : [];return [{family_hash:typeof fit.family_hash === 'string' ? fit.family_hash : '',modules,pct:nonnegative(fit.pct),uses:nonnegative(fit.uses)}]}) : []
  return {enough:value.enough,mirror:value.mirror === true,sample:nonnegative(value.sample),attacker_win_rate:Math.min(100,nonnegative(value.attacker_win_rate)),attacker_wins:nonnegative(value.attacker_wins),victim_wins:nonnegative(value.victim_wins),window_days:nonnegative(value.window_days) || 90,top_fits}
}
