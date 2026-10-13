import {isRecord} from './json'
import type {ReplayKill} from '../../utils/map/replay'
import type {BattleSide} from '../../utils/battleExplorer'
export const battleNumber=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:0
const text=(value:unknown)=>typeof value==='string'?value:''
const nullableNumber=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:null
export interface CompositionBase {rank:number;ship_group_id:number;ship_type_id?:number;isk_lost:number;damage_done?:number;damage_taken?:number}
export interface CompositionPilot extends CompositionBase {character_id:number;character_name:string;corporation_id?:number;corporation_name:string;alliance_id?:number;ship_name:string;ship_group_name?:string;deaths:number}
export interface CompositionAggregate extends CompositionBase {name:string;count:number;losses:number}
export type CompositionRow=CompositionPilot|CompositionAggregate
export type CompositionView='individuals'|'by_ship'|'by_group'
export type CompositionTeam={individuals?:CompositionPilot[];by_ship?:CompositionAggregate[];by_group?:CompositionAggregate[]}
export type CompositionFilters={search:string;group:number|null;losses:boolean}
export const compositionKeys=['default','ship_name','character_name','name','count','losses','isk_lost','damage_done','damage_taken'] as const
export type CompositionSortKey=typeof compositionKeys[number]
export type CompositionSort={key:CompositionSortKey;dir:'asc'|'desc'}
export function isCompositionView(value:unknown):value is CompositionView{return value==='individuals'||value==='by_ship'||value==='by_group'}
export function isCompositionSortKey(value:unknown):value is CompositionSortKey{return compositionKeys.some(key=>key===value)}
export function readComposition(raw:unknown):{teams:CompositionTeam[]}{
 const source=isRecord(raw)?raw:{},base=(row:Record<string,unknown>):CompositionBase=>({rank:battleNumber(row.rank),ship_group_id:battleNumber(row.ship_group_id),ship_type_id:battleNumber(row.ship_type_id),isk_lost:battleNumber(row.isk_lost),damage_done:battleNumber(row.damage_done),damage_taken:battleNumber(row.damage_taken)})
 const pilots=(values:unknown):CompositionPilot[]=>Array.isArray(values)?values.flatMap(row=>isRecord(row)?[{...base(row),character_id:battleNumber(row.character_id),character_name:text(row.character_name),corporation_id:battleNumber(row.corporation_id),corporation_name:text(row.corporation_name),alliance_id:battleNumber(row.alliance_id),ship_name:text(row.ship_name),ship_group_name:text(row.ship_group_name),deaths:battleNumber(row.deaths)}]:[]):[]
 const aggregates=(values:unknown):CompositionAggregate[]=>Array.isArray(values)?values.flatMap(row=>isRecord(row)?[{...base(row),name:text(row.name),count:battleNumber(row.count),losses:battleNumber(row.losses)}]:[]):[]
 return {teams:Array.isArray(source.teams)?source.teams.map(value=>{const team=isRecord(value)?value:{};return {individuals:pilots(team.individuals),by_ship:aggregates(team.by_ship),by_group:aggregates(team.by_group)}}):[]}
}
export function readBattleSides(raw:unknown):BattleSide[]{const ids=(value:unknown)=>Array.isArray(value)?value.filter((id):id is number=>typeof id==='number'&&Number.isSafeInteger(id)&&id>0):[];return Array.isArray(raw)?raw.map(value=>{const side=isRecord(value)?value:{};return {corps:ids(side.corps),alliances:ids(side.alliances)}}):[]}
export interface ReplayLandmark {id:number;name:string;solar_system_id:number;group_id:number;x:number;y:number;z:number}
export interface ReplaySource {kills:ReplayKill[];landmarks:ReplayLandmark[];team_entities:BattleSide[];start:number;end:number;time:number;selected:number}
export function readReplaySource(raw:unknown):ReplaySource {
 if(!isRecord(raw)||!Number.isFinite(raw.start)||!Number.isFinite(raw.end)||Number(raw.end)<=Number(raw.start)||!Number.isFinite(new Date(Number(raw.start)).getTime())||!Number.isFinite(new Date(Number(raw.end)).getTime()))throw new Error('Battle replay data is unavailable')
 const start=Number(raw.start),end=Number(raw.end),time=typeof raw.time==='number'&&Number.isFinite(raw.time)?Math.max(start,Math.min(end,raw.time)):start
 return {start,end,time,selected:battleNumber(raw.selected),team_entities:readBattleSides(raw.team_entities),kills:Array.isArray(raw.kills)?raw.kills.flatMap(k=>isRecord(k)&&typeof k.killmail_time==='string'&&Number.isFinite(Date.parse(k.killmail_time))&&battleNumber(k.killmail_id)>0?[{killmail_id:battleNumber(k.killmail_id),killmail_time:k.killmail_time,solar_system_id:battleNumber(k.solar_system_id),solar_system_name:text(k.solar_system_name),position_x:nullableNumber(k.position_x),position_y:nullableNumber(k.position_y),position_z:nullableNumber(k.position_z),victim_character_name:text(k.victim_character_name),victim_corporation_id:nullableNumber(k.victim_corporation_id),victim_alliance_id:nullableNumber(k.victim_alliance_id),ship_name:text(k.ship_name),ship_type_id:nullableNumber(k.ship_type_id),ship_group_id:nullableNumber(k.ship_group_id),ship_group_name:text(k.ship_group_name),total_value:battleNumber(k.total_value)}]:[]):[],landmarks:Array.isArray(raw.landmarks)?raw.landmarks.flatMap(l=>isRecord(l)&&[l.x,l.y,l.z].every(v=>typeof v==='number'&&Number.isFinite(v))?[{id:battleNumber(l.id),name:text(l.name),solar_system_id:battleNumber(l.solar_system_id),group_id:battleNumber(l.group_id),x:battleNumber(l.x),y:battleNumber(l.y),z:battleNumber(l.z)}]:[]):[]}
}
