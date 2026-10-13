import type {EntityPageMostValuableResponses,EntityPageShipClassesResponses} from '../../../shared/api/types.gen'
import {isRecord} from './json'
import {searchID} from './search-model'
import {formatCompact,formatNumber} from '../../utils/formatters'

export type EntityKind = 'character'|'corporation'|'alliance'|'faction'
export type EntityCategory = 'kills'|'ships'|'structures'
export const isEntityKind = (value:unknown):value is EntityKind=>value === 'character' || value === 'corporation' || value === 'alliance' || value === 'faction'
export const isEntityCategory = (value:unknown):value is EntityCategory=>value === 'kills' || value === 'ships' || value === 'structures'
export const entityPeriods = new Set([0,14,30,90,180,365])
export function entityIdentity(root:HTMLElement) {
  const kind = root.dataset.entityKind,id = searchID(root.dataset.entityId)
  if(!isEntityKind(kind) || !id) throw new Error('Invalid entity component')
  return {kind,id,path:`/${kind}/${id}`,api:`/api/entity/${kind}/${id}`,fragments:`/_fragments/entity`}
}
export function entityNode<K extends keyof HTMLElementTagNameMap>(tag:K,className='',text?:string|number):HTMLElementTagNameMap[K] {
  const element = document.createElement(tag);element.className = className;if(text !== undefined) element.textContent = String(text);return element
}
export function entityISK(value:number) {return value >= 1e6 ? formatCompact(value) : formatNumber(Math.round(value))}
type Valuable = Pick<EntityPageMostValuableResponses[200]['entries'][number],'killmail_id'|'ship_type_id'|'ship_name'|'total_value'|'victim_character_name'|'victim_corporation_name'>
type ShipClass = Pick<EntityPageShipClassesResponses[200]['groups'][number],'group_id'|'group_name'|'losses'>
const nullableText = (value:unknown)=>typeof value === 'string' ? value : null
export function readEntityValuable(value:unknown):Valuable[] {
  if(!isRecord(value) || !Array.isArray(value.entries)) throw new Error('Invalid valuable kills response')
  return value.entries.flatMap((entry:unknown):Valuable[]=>{
    if(!isRecord(entry) || typeof entry.killmail_id !== 'number' || !Number.isSafeInteger(entry.killmail_id) || entry.killmail_id < 1 || !searchID(entry.ship_type_id) || typeof entry.total_value !== 'number' || !Number.isFinite(entry.total_value) || entry.total_value < 0) return []
    return [{killmail_id:entry.killmail_id,ship_type_id:searchID(entry.ship_type_id),ship_name:nullableText(entry.ship_name) ?? '',total_value:entry.total_value,victim_character_name:nullableText(entry.victim_character_name),victim_corporation_name:nullableText(entry.victim_corporation_name)}]
  })
}
export function readEntityShipClasses(value:unknown):ShipClass[] {
  if(!isRecord(value) || !Array.isArray(value.groups)) throw new Error('Invalid ship classes response')
  return value.groups.flatMap((group:unknown):ShipClass[]=>isRecord(group) && searchID(group.group_id) && typeof group.group_name === 'string' && typeof group.losses === 'number' && Number.isSafeInteger(group.losses) && group.losses >= 0 ? [{group_id:searchID(group.group_id),group_name:group.group_name,losses:group.losses}] : [])
}
