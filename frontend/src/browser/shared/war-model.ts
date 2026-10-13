import {isRecord} from './json'
import type {FwSystem,FwCelestial} from '../../utils/fwWarzone'
export const warTabs=['wars','finished','upcoming','eligible-corps','eligible-alliances'] as const
export type WarsTab=typeof warTabs[number]
export const warSorts=['recent','kills','isk'] as const
export type WarsSort=typeof warSorts[number]
export const warFilterNames=['mutual','hasActivity','hasKills','hasAllies'] as const
export type WarFilter=typeof warFilterNames[number]
export const eligibleSorts=['name','members','kills','losses','isk_destroyed','isk_lost'] as const
export type EligibleSort=typeof eligibleSorts[number]
export const memberSorts=['activity','kills','losses','isk'] as const
export type MemberSort=typeof memberSorts[number]
export type WarSide={id:number;type:'alliance'|'corporation'|'';name:string;ships_killed:number;isk_destroyed:number}
export type WarRow={war_id:number;started:string;finished:string;mutual:boolean;open_for_allies:boolean;aggressor:WarSide;defender:WarSide}
export type EligibleEntry={id:number;name:string;ticker:string;alliance_name:string;member_count:number;corporation_count:number;kills:number;losses:number;isk_destroyed:number;isk_lost:number}
export type FwMapData={systems:FwSystem[];jumps:number[][];celestials:FwCelestial[]}
export const warNumber=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:0
const text=(value:unknown)=>typeof value==='string'?value:''
const id=(value:unknown)=>typeof value==='number'&&Number.isSafeInteger(value)&&value>0?value:0
export const isWarsTab=(value:unknown):value is WarsTab=>warTabs.some(tab=>tab===value)
export const isWarsSort=(value:unknown):value is WarsSort=>warSorts.some(sort=>sort===value)
export const isWarFilter=(value:unknown):value is WarFilter=>warFilterNames.some(filter=>filter===value)
export const isEligibleSort=(value:unknown):value is EligibleSort=>eligibleSorts.some(sort=>sort===value)
export const isMemberSort=(value:unknown):value is MemberSort=>memberSorts.some(sort=>sort===value)
function side(raw:unknown):WarSide{const value=isRecord(raw)?raw:{};return {id:id(value.id),type:value.type==='alliance'||value.type==='corporation'?value.type:'',name:text(value.name)||'Unknown',ships_killed:warNumber(value.ships_killed),isk_destroyed:warNumber(value.isk_destroyed)}}
export function readWars(raw:unknown):WarRow[]{if(!isRecord(raw)||!Array.isArray(raw.wars))throw new Error('War data is unavailable');return raw.wars.flatMap(value=>isRecord(value)&&id(value.war_id)?[{war_id:id(value.war_id),started:text(value.started),finished:text(value.finished),mutual:value.mutual===true,open_for_allies:value.open_for_allies===true,aggressor:side(value.aggressor),defender:side(value.defender)}]:[])}
export function readEligible(raw:unknown):EligibleEntry[]{if(!isRecord(raw)||!Array.isArray(raw.entries))throw new Error('Eligible entities are unavailable');return raw.entries.flatMap(value=>isRecord(value)&&id(value.id)?[{id:id(value.id),name:text(value.name)||'Unknown',ticker:text(value.ticker),alliance_name:text(value.alliance_name),member_count:warNumber(value.member_count),corporation_count:warNumber(value.corporation_count),kills:warNumber(value.kills),losses:warNumber(value.losses),isk_destroyed:warNumber(value.isk_destroyed),isk_lost:warNumber(value.isk_lost)}]:[])}
export function sortEligible(entries:EligibleEntry[],key:EligibleSort,direction:'asc'|'desc'):EligibleEntry[]{const dir=direction==='asc'?1:-1;return [...entries].sort((a,b)=>dir*(key==='name'?a.name.localeCompare(b.name):a[key==='members'?'member_count':key]-b[key==='members'?'member_count':key]))}
export function readFwSystems(raw:unknown):FwMapData{
 if(!isRecord(raw)||!Array.isArray(raw.systems)||!Array.isArray(raw.jumps))throw new Error('Faction war systems are unavailable')
 return {systems:raw.systems.flatMap(value=>isRecord(value)&&id(value.solar_system_id)&&[value.x,value.y].every(n=>typeof n==='number'&&Number.isFinite(n))?[{solar_system_id:id(value.solar_system_id),system_name:text(value.system_name)||`System ${value.solar_system_id}`,x:warNumber(value.x),y:warNumber(value.y),security:warNumber(value.security),region_id:id(value.region_id),region_name:text(value.region_name),constellation_name:text(value.constellation_name),owner_faction_id:id(value.owner_faction_id),occupier_faction_id:id(value.occupier_faction_id),contested:text(value.contested)||'uncontested',victory_points:warNumber(value.victory_points),victory_points_threshold:warNumber(value.victory_points_threshold),kills_24h:warNumber(value.kills_24h),isk_24h:warNumber(value.isk_24h)}]:[]),jumps:raw.jumps.flatMap(value=>Array.isArray(value)&&value.length===2&&value.every(id)?[[id(value[0]),id(value[1])]]:[]),celestials:Array.isArray(raw.celestials)?raw.celestials.flatMap(value=>isRecord(value)&&id(value.system_id)&&[value.x,value.z].every(n=>typeof n==='number'&&Number.isFinite(n))?[{system_id:id(value.system_id),group_id:id(value.group_id),x:warNumber(value.x),z:warNumber(value.z)}]:[]):[]}
}
const systemsCache=new WeakMap<HTMLElement,{source:string;data:FwMapData}>()
// Parent table and child map consume the same immutable JSON script.
export function readFwSystemsScript(script:HTMLElement){const source=script.textContent??'null',cached=systemsCache.get(script);if(cached?.source===source)return cached.data;const data=readFwSystems(JSON.parse(source));systemsCache.set(script,{source,data});return data}
