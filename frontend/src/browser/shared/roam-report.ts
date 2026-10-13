import {isRecord} from './json'
import {eveInputToDate,eveInputToIso} from '../../utils/eveTime'
import type {RoamReportCreateData} from '../../../shared/api/types.gen'
import type {BattleMapSystem} from '../../utils/map/battles'
export type RoamDraft={names:string;start:string;end:string}
export type RoamSaved={id:string;hasNewerDraft:boolean;refresh?:Promise<void>}
export const roamTabs=['information','kills','systems','intel','composition'] as const
export type RoamTab=typeof roamTabs[number]
export const isRoamTab=(value:unknown):value is RoamTab=>roamTabs.some(tab=>tab===value)
export const isRoamID=(value:unknown):value is string=>typeof value==='string'&&/^[a-f0-9]{32}$/.test(value)
export const uniqueRoamPilots=(text:string)=>[...new Set(text.split(/\r?\n/).map(name=>name.trim().toLowerCase()).filter(Boolean))]
export function roamTimeError(draft:RoamDraft,now=Date.now()):string{const start=eveInputToDate(draft.start).getTime(),end=eveInputToDate(draft.end).getTime();return !Number.isFinite(start)||!Number.isFinite(end)?'Choose a start and end time.':end<=start?'End must be after start.':end-start>72*3600000?'Maximum window is 72 hours.':end>now+300000?'End cannot be in the future.':''}
export const roamBody=(draft:RoamDraft):RoamReportCreateData['body']=>({names_text:draft.names,start_time:eveInputToIso(draft.start),end_time:eveInputToIso(draft.end)})
export const sameRoamDraft=(a:RoamDraft,b:RoamDraft)=>a.names===b.names&&a.start===b.start&&a.end===b.end
export function readRoamDraft(raw:unknown):RoamDraft|undefined{return isRecord(raw)&&typeof raw.names==='string'&&typeof raw.start==='string'&&typeof raw.end==='string'?{names:raw.names,start:raw.start,end:raw.end}:undefined}
export function readRoamSavedID(raw:unknown):string{if(!isRecord(raw)||!isRoamID(raw.id))throw new Error('The saved report link is unavailable. Reload before submitting again.');return raw.id}
export function isRoamSaved(raw:unknown):raw is RoamSaved{return isRecord(raw)&&isRoamID(raw.id)&&typeof raw.hasNewerDraft==='boolean'&&(raw.refresh===undefined||raw.refresh instanceof Promise)}
export function readRoamMarkers(raw:unknown):BattleMapSystem[]{
 if(!Array.isArray(raw))throw new Error('Roam map systems are unavailable')
 const number=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:0,text=(value:unknown)=>typeof value==='string'?value:null
 return raw.flatMap(value=>isRecord(value)&&typeof value.solar_system_id==='number'&&Number.isSafeInteger(value.solar_system_id)&&value.solar_system_id>0?[{solar_system_id:value.solar_system_id,solar_system_name:text(value.solar_system_name),region_id:number(value.region_id)||null,region_name:text(value.region_name),battle_count:number(value.engagements),kill_count:number(value.kills)+number(value.losses),total_isk_destroyed:number(value.isk_destroyed)+number(value.isk_lost)}]:[])
}
