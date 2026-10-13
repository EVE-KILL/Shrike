import type {AdvancedFilters} from '../../../shared/api/types.gen'
import type {KilllistRow} from '../../../shared/utils/killlistRow'
import {matchesMetaGroup} from '../../../shared/utils/killlistRow'

export const pickerKinds = ['region','system','ship'] as const
export type PickerKind = typeof pickerKinds[number]
export const securityKinds = ['highsec','lowsec','nullsec','wspace','pochven','abyssal'] as const
export type SecurityKind = typeof securityKinds[number]
export type QuickFilters = Record<PickerKind,{id:number;name:string}|null> & {security:Set<SecurityKind>;solo:boolean;npc:boolean;isk:'5b'|'10b'|null;tech:'t1'|'t2'|'t3'|null}
export const isPickerKind = (value:unknown):value is PickerKind=>pickerKinds.some(kind=>kind === value)
export const isSecurityKind = (value:unknown):value is SecurityKind=>securityKinds.some(kind=>kind === value)
export function quickFilters():QuickFilters {return {region:null,system:null,ship:null,security:new Set(),solo:false,npc:false,isk:null,tech:null}}
export function quickFilterCount(filters:QuickFilters):number {return Number(Boolean(filters.region || filters.system))+Number(Boolean(filters.ship))+filters.security.size+Number(filters.solo)+Number(filters.npc)+Number(Boolean(filters.isk))+Number(Boolean(filters.tech))}
export function quickFilterPayload(filters:QuickFilters):string {
  if(!quickFilterCount(filters)) return ''
  const payload:AdvancedFilters = {timeRange:{preset:'90d'}},location:NonNullable<AdvancedFilters['location']> = {}
  if(filters.system) location.systemId = filters.system.id;else if(filters.region) location.regionId = filters.region.id
  if(filters.security.size) location.securityTypes = [...filters.security]
  if(Object.keys(location).length) payload.location = location
  if(filters.ship) payload.entities = {victim:[{id:filters.ship.id,type:'ship'}]}
  if(filters.solo) payload.attackerCount = 'solo'
  if(filters.npc) payload.attackerType = 'npc'
  if(filters.isk) payload.iskValue = filters.isk
  if(filters.tech) payload.techLevel = filters.tech
  return JSON.stringify(payload)
}
export function matchesQuickFilters(kill:KilllistRow,filters:QuickFilters):boolean {
  if(filters.system && kill.solar_system_id !== filters.system.id || !filters.system && filters.region && kill.region_id !== filters.region.id || filters.ship && kill.ship_type_id !== filters.ship.id) return false
  if(filters.solo && kill.is_solo !== true || filters.npc && kill.is_npc !== true || filters.isk && Number(kill.total_value) < (filters.isk === '10b' ? 1e10 : 5e9)) return false
  if(filters.tech && !matchesMetaGroup(kill.meta_group_id,filters.tech)) return false
  if(filters.security.size) {
    const region = kill.region_id ?? 0,security = kill.solar_system_security
    const space:SecurityKind|undefined = region === 10000070 ? 'pochven' : region >= 12000001 && region <= 12000005 ? 'abyssal' : region >= 11000001 && region <= 11000033 ? 'wspace' : security == null ? undefined : security >= .45 ? 'highsec' : security > 0 ? 'lowsec' : 'nullsec'
    if(!space || !filters.security.has(space)) return false
  }
  return true
}
