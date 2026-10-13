import type {KillmailSubmitResponses,KillmailDetailLegacyResponses} from '../../../shared/api/types.gen'
import {isRecord} from './json'
export type SubmissionResult = KillmailSubmitResponses[200]
type Detail = KillmailDetailLegacyResponses[200]
export type Arrival = {killmail_id:number;ship_type_id?:number|null;ship_name?:string|null;victim_character_name?:string|null;victim_corporation_name?:string|null;total_value?:Detail['total_value']|null}
export type ReadyNotice = Arrival & {uid:number;dismissAt:number}
export type SubmissionState = {startedAt:number;pending:number[];uncertain:number[];notices:ReadyNotice[]}
export const submissionStorageKey = 'shrike-kill-submissions-v1',submissionMaxAge = 2*60*60*1000
export const esiKillmailLink = /https?:\/\/esi\.evetech\.net\/(?:latest\/|v\d+\/)?killmails\/(\d+)\/([a-f0-9]{20,64})\/?/i
export function killID(value:unknown):number {return typeof value === 'number' && Number.isSafeInteger(value) && value > 0 ? value : 0}
const ids = (value:unknown):number[]=>Array.isArray(value) ? [...new Set(value.flatMap((item:unknown)=>killID(item) ? [killID(item)] : []))] : []
const text = (value:unknown)=>typeof value === 'string' ? value : null
export function submittedIDs(value:string):number[] {const found = new Set<number>();for(const match of value.matchAll(new RegExp(esiKillmailLink.source,'gi'))) {const id = killID(Number(match[1]));if(id) found.add(id)}return [...found]}
export function readSubmissionResult(value:unknown):SubmissionResult {
  if(!isRecord(value)) throw new Error('Invalid queue response')
  const count = (name:string)=>{const n = value[name];if(typeof n !== 'number' || !Number.isSafeInteger(n) || n < 0) throw new Error('Invalid queue response');return n}
  const total = count('total'),accepted = count('accepted'),existing = count('existing'),rejected = count('rejected'),killmails = ids(value.killmails),existingIds = ids(value.existingIds)
  if(accepted+existing+rejected !== total || !Array.isArray(value.killmails) || killmails.length !== value.killmails.length || !Array.isArray(value.existingIds) || existingIds.length !== value.existingIds.length || killmails.length !== total || existingIds.length !== existing || existingIds.some(id=>!killmails.includes(id))) throw new Error('Invalid queue response')
  return {total,accepted,existing,rejected,killmails,existingIds,message:text(value.message) ?? undefined}
}
export function readArrival(value:unknown):Arrival|null {
  if(!isRecord(value)) return null
  const id = killID(value.killmail_id);if(!id) return null
  return {killmail_id:id,ship_type_id:killID(value.ship_type_id) || null,ship_name:text(value.ship_name),victim_character_name:text(value.victim_character_name),victim_corporation_name:text(value.victim_corporation_name),total_value:typeof value.total_value === 'number' && Number.isFinite(value.total_value) && value.total_value >= 0 ? value.total_value : null}
}
export function detailArrival(id:number,value:unknown):Arrival {
  const detail = isRecord(value) ? value : {},victim = isRecord(detail.victim) ? detail.victim : {}
  return readArrival({killmail_id:id,ship_type_id:victim.ship_type_id,ship_name:victim.ship_name,victim_character_name:victim.character_name,victim_corporation_name:victim.corporation_name,total_value:detail.total_value}) ?? {killmail_id:id}
}
export function readSubmissionState(value:unknown,now=Date.now()):SubmissionState {
  if(!isRecord(value)) return {startedAt:0,pending:[],uncertain:[],notices:[]}
  const validStart = typeof value.startedAt === 'number' && Number.isFinite(value.startedAt) && value.startedAt > 0 && value.startedAt <= now && now-value.startedAt < submissionMaxAge
  const notices = Array.isArray(value.notices) ? value.notices.flatMap((item:unknown):ReadyNotice[]=>{if(!isRecord(item) || !killID(item.uid) || typeof item.dismissAt !== 'number' || !Number.isFinite(item.dismissAt) || item.dismissAt <= now) return [];const arrival = readArrival(item);return arrival ? [{...arrival,uid:killID(item.uid),dismissAt:item.dismissAt}] : []}) : []
  return {startedAt:validStart ? Number(value.startedAt) : 0,pending:validStart ? ids(value.pending) : [],uncertain:validStart ? ids(value.uncertain) : [],notices}
}
export function compactSubmissionValue(value:number):string {if(value >= 1e12) return (value/1e12).toFixed(2)+'t';if(value >= 1e9) return (value/1e9).toFixed(2)+'b';if(value >= 1e6) return (value/1e6).toFixed(2)+'m';if(value >= 1e3) return (value/1e3).toFixed(1)+'k';return String(Math.round(value))}
