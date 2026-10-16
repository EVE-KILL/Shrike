import {isRecord} from '../browser/shared/json'
import type {AdvancedEntity,AdvancedItemFilter} from '../../shared/api/types.gen'

export const advancedRoles=['victim','attacker','both'] as const
export type AdvancedRole=typeof advancedRoles[number]
export const advancedLocations=['system','region','constellation'] as const
export const advancedScalarFilters=['attackerCount','attackerType','iskValue','shipCategory','techLevel'] as const
export type AdvancedScalarFilter=typeof advancedScalarFilters[number]
export type AdvancedPickedEntity=AdvancedEntity&{name:string;role?:string}
export type AdvancedItem=AdvancedItemFilter&{name:string;slot:string;side:string}
export type AdvancedTime={preset:string|null;from?:never;to?:never}|{from:string;to?:string;preset?:never}
export type AdvancedLocation={securityTypes:string[];systemId:number|null;systemName:string|null;regionId:number|null;regionName:string|null;constellationId:number|null;constellationName:string|null}
export interface AdvancedFilters {
  label:string|null
  entities:Record<AdvancedRole,AdvancedPickedEntity[]>
  items:AdvancedItem[]
  location:AdvancedLocation
  timeRange:AdvancedTime
  attackerCount:string|null;attackerType:string|null;iskValue:string|null
  iskMin:string;iskMax:string;shipCategory:string|null;techLevel:string|null
  sort:{field:string;direction:string}
}
export interface AdvancedState {filters:AdvancedFilters;view:'kills'|'fits';dedup:'none'|'exact'|'family';hash:string;hashMode:'exact'|'family'|'';showResults?:boolean}
export interface AdvancedOption {value:string;label:string}
export interface AdvancedOptions {groups:Array<{key:string;options:AdvancedOption[]}>;labels:Array<{id:string;name:string}>;tech:AdvancedOption[];time:AdvancedOption[]}
export const isAdvancedScalar=(value:unknown):value is AdvancedScalarFilter=>advancedScalarFilters.some(key=>key===value)
export const isAdvancedRole=(value:unknown):value is AdvancedRole=>advancedRoles.some(role=>role===value)
export interface AdvancedHit {id:string;type:string;name:string;ticker?:string}
export function readAdvancedHits(raw:unknown):AdvancedHit[]{return isRecord(raw)&&Array.isArray(raw.hits)?raw.hits.flatMap(hit=>isRecord(hit)&&typeof hit.id==='string'&&/^[a-z]+_\d+$/.test(hit.id)&&typeof hit.name==='string'&&typeof hit.type==='string'?[{id:hit.id,type:hit.type,name:hit.name,...(typeof hit.ticker==='string'?{ticker:hit.ticker}:{})}]:[]):[]}
export function readAdvancedState(raw:unknown):AdvancedState {
  const source=isRecord(raw)?raw:{}
  return {filters:restoreAdvancedFilters(source.filters),view:source.view==='fits'?'fits':'kills',dedup:source.dedup==='none'||source.dedup==='exact'?source.dedup:'family',hash:typeof source.hash==='string'?source.hash:'',hashMode:source.hashMode==='exact'||source.hashMode==='family'?source.hashMode:'',showResults:source.showResults===true}
}
export function readAdvancedOptions(raw:unknown):AdvancedOptions {
  const source=isRecord(raw)?raw:{},options=(value:unknown):AdvancedOption[]=>Array.isArray(value)?value.flatMap(row=>isRecord(row)&&typeof row.value==='string'&&typeof row.label==='string'?[{value:row.value,label:row.label}]:[]):[]
  return {groups:Array.isArray(source.groups)?source.groups.flatMap(group=>isRecord(group)&&typeof group.key==='string'?[{key:group.key,options:options(group.options)}]:[]):[],labels:Array.isArray(source.labels)?source.labels.flatMap(label=>isRecord(label)&&typeof label.id==='string'&&typeof label.name==='string'?[{id:label.id,name:label.name}]:[]):[],tech:options(source.tech),time:options(source.time)}
}
export const defaultAdvancedFilters=():AdvancedFilters=>({label:null,entities:{victim:[],attacker:[],both:[]},items:[],location:{securityTypes:[],systemId:null,systemName:null,regionId:null,regionName:null,constellationId:null,constellationName:null},timeRange:{preset:'30d'},attackerCount:null,attackerType:null,iskValue:null,iskMin:'',iskMax:'',shipCategory:null,techLevel:null,sort:{field:'killmail_time',direction:'desc'}})
const positiveID=(value:unknown):value is number=>typeof value==='number'&&Number.isSafeInteger(value)&&value>0
const text=(value:unknown)=>typeof value==='string'?value:null
export function restoreAdvancedFilters(parsed:unknown):AdvancedFilters {
  const filters=defaultAdvancedFilters();if(!isRecord(parsed))return filters
  filters.label=text(parsed.label)
  const entities=parsed.entities
  if(isRecord(entities))for(const role of advancedRoles){const rows=entities[role];if(Array.isArray(rows))filters.entities[role]=rows.flatMap(row=>isRecord(row)&&positiveID(row.id)&&typeof row.type==='string'?[{id:row.id,type:row.type,name:text(row.name)??'',...(typeof row.role==='string'?{role:row.role}:{}),...(row.exclude===true?{exclude:true}:{})}]:[])}
  if(Array.isArray(parsed.items))filters.items=parsed.items.flatMap(item=>isRecord(item)&&(positiveID(item.typeId)||positiveID(item.groupId))?[{...(positiveID(item.typeId)?{typeId:item.typeId}:{}),...(positiveID(item.groupId)?{groupId:item.groupId}:{}),name:text(item.name)??'',slot:text(item.slot)??'any',side:text(item.side)??'victim'}]:[])
  const location=parsed.location
  if(isRecord(location)){if(Array.isArray(location.securityTypes))filters.location.securityTypes=location.securityTypes.filter((value):value is string=>typeof value==='string');for(const kind of advancedLocations){const id=location[`${kind}Id`];if(positiveID(id)){filters.location[`${kind}Id`]=id;filters.location[`${kind}Name`]=text(location[`${kind}Name`])}}}
  const time=parsed.timeRange
  if(isRecord(time)){if(typeof time.from==='string')filters.timeRange={from:time.from,...(typeof time.to==='string'?{to:time.to}:{})};else if(time.preset===null||typeof time.preset==='string')filters.timeRange={preset:time.preset}}
  for(const key of advancedScalarFilters)filters[key]=text(parsed[key])
  for(const key of ['iskMin','iskMax'] as const)if(typeof parsed[key]==='string'||typeof parsed[key]==='number')filters[key]=String(parsed[key])
  if(isRecord(parsed.sort)){filters.sort.field=text(parsed.sort.field)??'killmail_time';filters.sort.direction=text(parsed.sort.direction)??'desc'}
  return filters
}
export function parseAdvancedISK(value:unknown){
  if(!value)return null
  const match=String(value).trim().toLowerCase().match(/^([\d.]+)\s*([bmk])?$/);if(!match)return null
  const amount=parseFloat(match[1]);if(Number.isNaN(amount))return null
  const multipliers:Record<string,number>={b:1e9,m:1e6,k:1e3};return amount*(multipliers[match[2]]??1)
}
export function hasAdvancedFilters(filters:AdvancedFilters){return Boolean(filters.label||advancedRoles.some(role=>filters.entities[role].length)||filters.items.length||filters.location.securityTypes.length||advancedLocations.some(kind=>filters.location[`${kind}Id`])||advancedScalarFilters.some(key=>filters[key])||filters.iskMin!==''||filters.iskMax!=='')}
type ShareFilters=Partial<Omit<AdvancedFilters,'entities'|'location'>>&{entities?:Partial<AdvancedFilters['entities']>;location?:Partial<AdvancedLocation>}
type CommonFilters=Omit<ShareFilters,'iskMin'|'iskMax'>
function commonFilters(filters:AdvancedFilters,display:boolean):CommonFilters {
  const out:CommonFilters={},entities:Partial<AdvancedFilters['entities']>={}
  if(filters.label)out.label=filters.label
  for(const role of advancedRoles)if(filters.entities[role].length)entities[role]=display?filters.entities[role]:filters.entities[role].map(entity=>({id:entity.id,type:entity.type,name:entity.name,...(entity.exclude?{exclude:true}:{})}))
  if(Object.keys(entities).length)out.entities=entities
  if(filters.items.length)out.items=filters.items.map(item=>({...item}))
  const location:Partial<AdvancedLocation>={};if(filters.location.securityTypes.length)location.securityTypes=filters.location.securityTypes
  for(const kind of advancedLocations)if(filters.location[`${kind}Id`]){location[`${kind}Id`]=filters.location[`${kind}Id`];if(display)location[`${kind}Name`]=filters.location[`${kind}Name`]}
  if(Object.keys(location).length)out.location=location
  for(const key of advancedScalarFilters)if(filters[key])out[key]=filters[key]
  if(filters.sort.field!=='killmail_time'||filters.sort.direction!=='desc')out.sort=filters.sort
  return out
}
export function serializeAdvancedFilters(filters:AdvancedFilters){
  const shared=commonFilters(filters,false),out:Omit<ShareFilters,'iskMin'|'iskMax'>&{iskMin?:number;iskMax?:number}=shared
  out.timeRange=filters.timeRange
  const min=parseAdvancedISK(filters.iskMin),max=parseAdvancedISK(filters.iskMax)
  if(min!==null||max!==null){if(min!==null)out.iskMin=min;if(max!==null)out.iskMax=max;delete out.iskValue}
  return JSON.stringify(out)
}
export function advancedFiltersForURL(filters:AdvancedFilters){
  const out:ShareFilters=commonFilters(filters,true),time=filters.timeRange
  if(('preset' in time&&time.preset&&time.preset!=='30d')||'from' in time)out.timeRange=time
  if(filters.iskMin)out.iskMin=filters.iskMin;if(filters.iskMax)out.iskMax=filters.iskMax
  return Object.keys(out).length?JSON.stringify(out):null
}
export function advancedURL(state:AdvancedState){const query=new URLSearchParams(),filters=advancedFiltersForURL(state.filters);if(filters)query.set('q',filters);if(state.view!=='kills')query.set('view',state.view);if(state.view==='fits'&&state.dedup!=='family')query.set('dedup',state.dedup);if(state.hash&&state.hashMode){query.set('dh',state.hash);query.set('dm',state.hashMode)}return query}
export function advancedRoleColor(role:AdvancedRole,exclude:boolean|undefined){return exclude?'bg-red-900/30 text-red-300 border-red-500/40 line-through':role==='victim'?'bg-red-500/20 text-red-400 border-red-500/30':role==='attacker'?'bg-blue-500/20 text-link border-blue-500/30':'bg-green-500/20 text-green-400 border-green-500/30'}
export const advancedButtonClass=(active:boolean)=>active?'bg-blue-500/20 text-link border-blue-500/40':'bg-white/[0.04] text-gray-400 border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-link'
export const advancedEntityLabel=(role:AdvancedRole,entity:AdvancedPickedEntity)=>entity.exclude?'exclude':entity.type==='weapon'?'weapon':entity.type==='shipgroup'?'hull class':role
export function advancedEntityImage(type:string,id:number|string){if(!positiveID(Number(id)))return '';if(type==='character')return `https://images.evetech.net/characters/${id}/portrait?size=32`;if(type==='corporation'||type==='faction')return `https://images.evetech.net/corporations/${id}/logo?size=32`;if(type==='alliance')return `https://images.evetech.net/alliances/${id}/logo?size=32`;if(['item','ship','weapon'].includes(type))return `/images/types/${id}/icon?size=32`;return ''}
export function advancedLocationLabel(location:AdvancedLocation){for(const kind of ['system','constellation','region'] as const)if(location[`${kind}Name`])return kind[0].toUpperCase()+kind.slice(1)+': '+location[`${kind}Name`];return null}
export function advancedActiveChips(filters:AdvancedFilters,options:AdvancedOptions){
  const out:Array<{key:string;group:string;label:string}>=[],labelOf=(list:AdvancedOption[]|undefined,value:string)=>list?.find(option=>option.value===value)?.label??value,groups=new Map(options.groups.map(group=>[group.key,group.options]))
  if(filters.label)out.push({key:'label',group:'Label',label:options.labels.find(option=>option.id===filters.label)?.name??filters.label})
  for(const security of filters.location.securityTypes)out.push({key:'sec-'+security,group:'Space',label:labelOf(groups.get('security'),security)})
  for(const [key,group]of [['attackerCount','Attackers'],['attackerType','Type'],['iskValue','ISK']] as const){const value=filters[key];if(value)out.push({key,group,label:labelOf(groups.get(key),value)})}
  if(filters.iskMin||filters.iskMax)out.push({key:'iskRange',group:'ISK',label:`${filters.iskMin||'0'} – ${filters.iskMax||'∞'}`})
  if(filters.shipCategory)out.push({key:'shipCategory',group:'Ships',label:labelOf(groups.get('shipCategory'),filters.shipCategory)})
  if(filters.techLevel)out.push({key:'techLevel',group:'Tech',label:labelOf(options.tech,filters.techLevel)})
  const range=filters.timeRange;if('preset' in range&&range.preset)out.push({key:'timePreset',group:'Time',label:labelOf(options.time,range.preset)});else if('from' in range&&range.from)out.push({key:'timeRange',group:'Time',label:`${range.from.replace('T',' ')} → ${(range.to??'').replace('T',' ')} EVE`})
  return out
}
export function clearAdvancedChip(filters:AdvancedFilters,key:string){
  if(key.startsWith('sec-')){const security=key.slice(4),current=filters.location.securityTypes;filters.location.securityTypes=current.includes(security)&&current.length===1?[]:[security]}
  else if(key==='iskRange')filters.iskMin=filters.iskMax=''
  else if(key==='timePreset'||key==='timeRange')filters.timeRange={preset:null}
  else if(key==='label')filters.label=null
  else for(const field of advancedScalarFilters)if(field===key)filters[field]=null
}
