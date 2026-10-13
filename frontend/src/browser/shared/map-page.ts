import type {MapScopeResponses,MapAiidResponses,MapSovereigntyResponses} from '../../../shared/api/types.gen'
import type {MapScopeData} from '../../utils/map/goFrontendCanvas'
import {isMapActivityLayer,isMapRenderBaseLayer,type MapActivityLayer,type MapRenderBaseLayer} from '../../utils/map/layers'
import type {MapFeedScope} from '../map-live'
import {parseLiveKill} from '../kill-row'
import {isRecord} from './json'
import {isMapScope,type MapScope} from './map-scope'

export type MapWindow = 1|6|24|168
export type MapToggle = 'routes'|'systems'|'labels'
export type MapPageState = {scope:MapScope;base:MapRenderBaseLayer;activity:MapActivityLayer;hours:MapWindow;routes:boolean;systems:boolean;labels:boolean;changes:boolean;watch:number[];nearAlarm:boolean;outerAlarm:boolean}
export const isMapWindow = (value:number):value is MapWindow=>[1,6,24,168].includes(value)
export const isMapToggle = (value:unknown):value is MapToggle=>value === 'routes' || value === 'systems' || value === 'labels'
export const isSpecialMapBase = (value:unknown)=>value === 'sovereignty' || value === 'live' || value === 'aiid'
export function parseMapWatch(value:string|null|undefined) {return [...new Set((value ?? '').split(',').map(Number).filter(id=>Number.isSafeInteger(id) && id > 0 && id <= 2147483647))].slice(0,8)}
export function mapPageState(query:URLSearchParams,previous?:MapPageState):MapPageState {
 const scope = query.get('scope'),base = query.get('base'),activity = query.get('activity'),layer = query.get('layer'),hours = Number(query.get('hours'))
 const state:MapPageState = {scope:isMapScope(scope) ? scope : 'new-eden',base:isMapRenderBaseLayer(base) ? base : layer === 'security' ? 'security' : 'geography',activity:isMapActivityLayer(activity) ? activity : isMapActivityLayer(layer) ? layer : 'none',hours:isMapWindow(hours) ? hours : 24,routes:query.get('routes') !== '0',systems:query.get('systems') !== '0',labels:query.get('labels') !== '0',changes:query.get('changes') === '1',watch:parseMapWatch(query.get('watch')),nearAlarm:previous?.nearAlarm ?? false,outerAlarm:previous?.outerAlarm ?? false}
 if(state.scope !== 'new-eden' && isSpecialMapBase(state.base))state.base = 'geography'
 return state
}

type SystemWire = MapScopeResponses[200]['systems'][number]
type AnchorWire = MapAiidResponses[200]['anchors'][number]
type ClaimWire = MapSovereigntyResponses[200]['sovereignty'][number]
export type MapPageData = MapFeedScope & {scope:string;snapshot_at?:string}
const id = (value:unknown):value is number=>typeof value === 'number' && Number.isSafeInteger(value) && value > 0 && value <= 2147483647
const finite = (value:unknown):value is number=>typeof value === 'number' && Number.isFinite(value)
function rows(value:unknown):unknown[] {if(value == null)return [];if(!Array.isArray(value))throw new Error('Invalid map data');return value}
function readSystem(value:unknown):Pick<SystemWire,'solar_system_id'|'system_name'|'region_id'|'security'|'x'|'z'|'distance'|'is_anchor'> {
 if(!isRecord(value) || !id(value.solar_system_id) || typeof value.system_name !== 'string' || !id(value.region_id) || !finite(value.security) || !finite(value.x) || !finite(value.z) || value.distance !== undefined && (!finite(value.distance) || value.distance < 0) || value.is_anchor !== undefined && typeof value.is_anchor !== 'boolean')throw new Error('Invalid map system')
 return {solar_system_id:value.solar_system_id,system_name:value.system_name,region_id:value.region_id,security:value.security,x:value.x,z:value.z,distance:typeof value.distance === 'number' ? value.distance : undefined,is_anchor:typeof value.is_anchor === 'boolean' ? value.is_anchor : undefined}
}
export function readMapPageData(value:unknown):MapPageData {
 if(!isRecord(value) || typeof value.scope !== 'string' || !isMapScope(value.scope) && value.scope !== 'aiid' && value.scope !== 'sovereignty')throw new Error('Invalid map scope')
 const systems = rows(value.systems).map(readSystem),regions = rows(value.regions).map((region):NonNullable<MapScopeData['regions']>[number]=>{if(!isRecord(region) || !id(region.region_id) || typeof region.name !== 'string')throw new Error('Invalid map region');return {region_id:region.region_id,name:region.name}})
 const jumps = rows(value.jumps).map((jump):NonNullable<MapScopeData['jumps']>[number]=>{if(!isRecord(jump) || !id(jump.from_solar_system_id) || !id(jump.to_solar_system_id))throw new Error('Invalid map jump');return {from_solar_system_id:jump.from_solar_system_id,to_solar_system_id:jump.to_solar_system_id}})
 const activity = rows(value.activity).map((row):NonNullable<MapScopeData['activity']>[number]=>{
  if(!isRecord(row) || !id(row.system_id))throw new Error('Invalid map activity')
  const metric = (key:'npc_kills'|'pod_kills'|'ship_jumps'|'ship_kills')=>{const amount = row[key] ?? 0;if(!finite(amount) || amount < 0)throw new Error('Invalid map activity');return amount}
  return {system_id:row.system_id,npc_kills:metric('npc_kills'),pod_kills:metric('pod_kills'),ship_jumps:metric('ship_jumps'),ship_kills:metric('ship_kills')}
 })
 const anchors = rows(value.anchors).map((anchor):Pick<AnchorWire,'solar_system_id'|'system_name'>=>{if(!isRecord(anchor) || !id(anchor.solar_system_id) || typeof anchor.system_name !== 'string')throw new Error('Invalid watched system');return {solar_system_id:anchor.solar_system_id,system_name:anchor.system_name}})
 const sovereignty = rows(value.sovereignty).map((claim):Pick<ClaimWire,'system_id'|'alliance_id'|'alliance_name'|'alliance_ticker'|'member_count'>=>{if(!isRecord(claim) || !id(claim.system_id) || !id(claim.alliance_id) || typeof claim.alliance_name !== 'string' || typeof claim.alliance_ticker !== 'string' || !finite(claim.member_count ?? 0) || Number(claim.member_count ?? 0) < 0)throw new Error('Invalid sovereignty claim');return {system_id:claim.system_id,alliance_id:claim.alliance_id,alliance_name:claim.alliance_name,alliance_ticker:claim.alliance_ticker,member_count:Number(claim.member_count ?? 0)}})
 const changes = rows(value.changes).map(change=>{if(!isRecord(change) || !id(change.system_id))throw new Error('Invalid sovereignty change');return {system_id:change.system_id}})
 if(value.snapshot_at != null && (typeof value.snapshot_at !== 'string' || !Number.isFinite(Date.parse(value.snapshot_at))))throw new Error('Invalid sovereignty snapshot')
 return {scope:value.scope,systems,regions,jumps,activity,anchors,sovereignty,changes,kills:rows(value.kills).flatMap(row=>{const kill = parseLiveKill(row);return kill ? [kill] : []}),snapshot_at:typeof value.snapshot_at === 'string' ? value.snapshot_at : undefined}
}
