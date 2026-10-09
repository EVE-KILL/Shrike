import type {FittingsRolesLegacyResponses,FittingsSearchAvailabilityLegacyResponses} from '../../../shared/api/types.gen'
import {isRecord} from './json'
import {searchID} from './search-model'
import {isFitMetric} from './fit-filters'
import {readFitRequirements,isFitSort,type FitSearchState,type StatisticFilter} from '../fit-search-state'
export type FitRole = FittingsRolesLegacyResponses[200]['roles'][number]
export type FitAvailability = FittingsSearchAvailabilityLegacyResponses[200]
export type FitItemHit = {typeID:number;name:string}
export function readFitSearchSeed(value:unknown):FitSearchState {
 if(!isRecord(value)) throw new Error('Invalid fitting search state')
 const ship = isRecord(value.ship) && searchID(value.ship.id) ? {id:searchID(value.ship.id),name:typeof value.ship.name === 'string' ? value.ship.name : `Type ${searchID(value.ship.id)}`} : null
 const statistics:StatisticFilter[] = []
 if(Array.isArray(value.statistics)) for(const filter of value.statistics) {
  if(!isRecord(filter) || !isFitMetric(filter.metric)) continue
  const number = (value:unknown)=>typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : null
  statistics.push({metric:filter.metric,min:number(filter.min),max:number(filter.max),...(typeof filter.npcProfile === 'string' && filter.npcProfile ? {npcProfile:filter.npcProfile} : {})})
 }
 return {ship,filters:readFitRequirements(value.filters),statistics,sort:isFitSort(value.sort) ? value.sort : 'uses',capStable:value.capStable === true}
}
export function readFitRoles(value:unknown):FitRole[] {
 if(!Array.isArray(value)) throw new Error('Invalid fitting roles')
 return value.flatMap((role:unknown):FitRole[]=>!isRecord(role) || typeof role.id !== 'string' || !role.id || typeof role.label !== 'string' || typeof role.icon !== 'string' || typeof role.category !== 'string' || typeof role.typeCount !== 'number' || !Number.isSafeInteger(role.typeCount) || role.typeCount < 0 ? [] : [{id:role.id,label:role.label,icon:role.icon,category:role.category,typeCount:role.typeCount,...(typeof role.description === 'string' ? {description:role.description} : {})}])
}
export function readFitAvailability(value:unknown,ship:number):FitAvailability {
 if(!isRecord(value) || searchID(value.ship_type_id) !== ship || !isRecord(value.role_counts) || !isRecord(value.type_counts)) throw new Error('Invalid fitting availability')
 const counts = (input:Record<string,unknown>):Record<string,number>=>Object.fromEntries(Object.entries(input).filter((entry):entry is [string,number]=>typeof entry[1] === 'number' && Number.isSafeInteger(entry[1]) && entry[1] >= 0))
 return {ship_type_id:ship,role_counts:counts(value.role_counts),type_counts:counts(value.type_counts),window_days:typeof value.window_days === 'number' && Number.isFinite(value.window_days) && value.window_days >= 0 ? value.window_days : 0}
}
export function readFitResultState(value:unknown,offset:number) {
 if(!isRecord(value) || typeof value.more !== 'boolean' || value.offset !== offset || typeof value.total !== 'number' || !Number.isSafeInteger(value.total) || value.total < 0 || typeof value.shipName !== 'string') throw new Error('Invalid fitting result fragment')
 return {more:value.more,offset,total:value.total,shipName:value.shipName}
}
