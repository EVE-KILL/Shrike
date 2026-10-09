import {parseFitNumber,sortForFitMetric,type FitMetric} from './shared/fit-filters'
import {isRecord} from './shared/json'
import {searchID} from './shared/search-model'

// URL and filter logic ported from web/app/pages/fits/search.vue.
export const statisticOptions = [
  ['ehp','Effective hitpoints'], ['dps','Damage / second'], ['alpha','Alpha damage'], ['speed','Maximum velocity'], ['align','Align time'], ['repair','Strongest local repair'], ['shield_repair','Active shield repair'], ['armor_repair','Active armor repair'], ['hull_repair','Active hull repair'], ['passive_shield','Passive shield regeneration'], ['npc_ehp','EHP against NPC faction'],
] as const satisfies readonly (readonly [FitMetric,string])[]
export const npcProfiles = [
  ['omni','Omnidamage'], ['angels','Angel Cartel'], ['blood-raiders','Blood Raiders'], ['guristas','Guristas Pirates'], ['mordus',"Mordu's Legion"], ['sansha',"Sansha's Nation"], ['serpentis','Serpentis'], ['triglavian','Triglavian Collective'], ['amarr','Amarr Empire'], ['caldari','Caldari State'], ['gallente','Gallente Federation'], ['minmatar','Minmatar Republic'],
] as const
export const fitSorts = ['uses','ehp','dps','alpha','repair','speed','align','npc_ehp'] as const
export type FitSort = typeof fitSorts[number]
export type FitComparison = '>='|'<='|'='|'>'|'<'
export type FitRequirement = {op:FitComparison;count:number} & ({kind:'role';role_id:string}|{kind:'type';type_id:number;type_name?:string})
export type StatisticFilter = {metric:FitMetric;min:number|null;max:number|null;npcProfile?:string}
export type FitSearchState = {ship:{id:number;name:string}|null;filters:FitRequirement[];statistics:StatisticFilter[];sort:FitSort;capStable:boolean}
export const isFitSort = (value:unknown):value is FitSort=>fitSorts.some(sort=>sort === value)
export const isFitComparison = (value:unknown):value is FitComparison=>['>=','<=','=','>','<'].some(op=>op === value)
export function parseStatisticValue(raw:string) {return parseFitNumber(raw)}
export function sortForMetric(metric:FitMetric):FitSort {const sort = sortForFitMetric(metric);if(!isFitSort(sort)) throw new Error('Invalid fitting sort');return sort}
export function statisticLabel(filter:StatisticFilter) {
  const label = statisticOptions.find(option=>option[0] === filter.metric)?.[1] ?? filter.metric
  const profile = filter.metric === 'npc_ehp' ? ` · ${npcProfiles.find(option=>option[0] === filter.npcProfile)?.[1] ?? filter.npcProfile}` : ''
  if(filter.min === null && filter.max === null) return label+profile
  const range = filter.min !== null && filter.max !== null ? `${filter.min.toLocaleString('en-US')}–${filter.max.toLocaleString('en-US')}` : filter.min !== null ? `≥ ${filter.min.toLocaleString('en-US')}` : `≤ ${filter.max?.toLocaleString('en-US')}`
  return `${label}${profile} ${range}`
}
export function statisticParams(state:Pick<FitSearchState,'statistics'|'capStable'>) {
  const query = new URLSearchParams()
  for(const filter of state.statistics) {if(filter.min !== null) query.set(`min_${filter.metric}`,String(filter.min));if(filter.max !== null) query.set(`max_${filter.metric}`,String(filter.max));if(filter.metric === 'npc_ehp' && filter.npcProfile) query.set('npc_profile',filter.npcProfile)}
  if(state.capStable) query.set('cap_stable','true')
  return query
}
export function searchParams(state:FitSearchState,{sort = false,name = false}:{sort?:boolean;name?:boolean} = {}) {
  const query = new URLSearchParams()
  if(state.ship) {query.set('ship',String(state.ship.id));if(name && state.ship.name) query.set('ship_name',state.ship.name)}
  if(state.filters.length) query.set('filters',JSON.stringify(state.filters))
  if(sort || name && state.sort !== 'uses') query.set('sort',state.sort)
  for(const [key,value] of statisticParams(state)) query.set(key,value)
  return query
}
export function readFitRequirements(value:unknown):FitRequirement[] {
  if(!Array.isArray(value)) return []
  const result:FitRequirement[] = []
  for(const filter of value) {
    if(!isRecord(filter) || !isFitComparison(filter.op) || typeof filter.count !== 'number' || !Number.isInteger(filter.count) || filter.count < 0 || filter.count > 16) continue
    if(typeof filter.role_id === 'string' && filter.role_id) result.push({kind:'role',role_id:filter.role_id,op:filter.op,count:filter.count})
    else if(typeof filter.type_id === 'number' && searchID(filter.type_id)) result.push({kind:'type',type_id:filter.type_id,...(typeof filter.type_name === 'string' && filter.type_name ? {type_name:filter.type_name} : {}),op:filter.op,count:filter.count})
    if(result.length === 8) break
  }
  return result
}
export function stateFromURL(query:URLSearchParams):FitSearchState {
  const ship = searchID(query.get('ship')),shipName = query.get('ship_name'),sort = query.get('sort')
  const state:FitSearchState = {ship:ship ? {id:ship,name:shipName || `Type ${ship}`} : null,filters:[],statistics:[],sort:isFitSort(sort) ? sort : 'uses',capStable:query.get('cap_stable') === 'true'}
  try {const filters:unknown = JSON.parse(query.get('filters') || '[]');state.filters = readFitRequirements(filters)}catch{/* A malformed URL never grants a filter shape. */}
  const number = (key:string)=>{const raw = query.get(key),value = Number(raw);return raw?.trim() && Number.isFinite(value) && value >= 0 ? value : null}
  for(const [metric] of statisticOptions) {const min = number(`min_${metric}`),max = number(`max_${metric}`),npcProfile = metric === 'npc_ehp' ? query.get('npc_profile') : null;if(min !== null || max !== null || npcProfile) state.statistics.push({metric,min,max,...(npcProfile ? {npcProfile} : {})})}
  return state
}
export function setStatistic(state:FitSearchState,metric:FitMetric,min:number|null,max:number|null,npcProfile?:string) {state.statistics = state.statistics.filter(filter=>filter.metric !== metric);state.statistics.push({metric,min,max,...(npcProfile ? {npcProfile} : {})})}
