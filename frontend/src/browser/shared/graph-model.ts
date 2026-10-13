import {isRecord} from './json'
import type {GraphData} from '../../../shared/api/types.gen'

export const graphModes = ['path_finder','entity_intel','hunting_grounds','hot_zones','census','spy_check','migration','coalitions','rivalries'] as const satisfies readonly NonNullable<GraphData['query']>['mode'][]
export type GraphMode = typeof graphModes[number]
export type GraphEntityType = 'alliance'|'corporation'|'character'
export type GraphSearchKey = 'from'|'to'|'entity'
export interface GraphEntity {id:number;name:string;corp_name:string;alliance_name:string}
export type GraphPathNode = GraphEntity & {corporation?:GraphEntity;alliance?:GraphEntity}
export type GraphAlly = GraphEntity & {shared_enemy_kills:number;mutual_kills:number}
export type GraphEnemy = GraphEntity & {kills_given:number;kills_taken:number}
export type GraphSystem = GraphEntity & {alliances:number;characters:number;active_characters:number}
export type GraphCorp = GraphEntity & {total:number;fcs:number;logis:number;caps:number}
export type GraphSuspect = GraphEntity & {total_flights:number;connections:Array<{name:string;flights:number}>}
export type GraphMigrant = GraphEntity & {current_corp?:GraphEntity;previous_corp?:GraphEntity}
export type GraphCoalition = {id:number;alliances:Array<GraphEntity & {cooperation:number}>}
export type GraphRivalry = {entity_a:GraphEntity;entity_b:GraphEntity;mutual_kills:number;total_isk:number}
export type GraphResults = {
  path_finder:{mode:'path_finder';path:{hops:number;nodes:GraphPathNode[];edges:Array<{weight:number}>}|null}
  entity_intel:{mode:'entity_intel';allies:GraphAlly[];enemies:GraphEnemy[]}
  hunting_grounds:{mode:'hunting_grounds';systems:GraphSystem[]}
  hot_zones:{mode:'hot_zones';systems:GraphSystem[]}
  census:{mode:'census';corps:GraphCorp[];totals:{characters:number;fcs:number;logis:number;caps:number}}
  spy_check:{mode:'spy_check';suspects:GraphSuspect[]}
  migration:{mode:'migration';corp_name:string;departed:GraphMigrant[];joined:GraphMigrant[]}
  coalitions:{mode:'coalitions';coalitions:GraphCoalition[]}
  rivalries:{mode:'rivalries';items:GraphRivalry[]}
}
export type GraphResult = GraphResults[GraphMode]
export type GraphHit = {id:number;name:string;ticker:string;corporation_name:string;alliance_name:string;alliance_ticker:string}
const text = (value:unknown)=>typeof value === 'string' ? value : ''
export function graphID(value:unknown):number {
  if(typeof value !== 'number' && typeof value !== 'string') return 0
  const n = Number(value);return Number.isSafeInteger(n) && n > 0 ? n : 0
}
const metric = (value:unknown)=>typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : 0
export function isGraphMode(value:unknown):value is GraphMode {return graphModes.some(mode=>mode === value)}
export function isGraphEntityType(value:unknown):value is GraphEntityType {return value === 'alliance' || value === 'corporation' || value === 'character'}
export function isGraphSearchKey(value:unknown):value is GraphSearchKey {return value === 'from' || value === 'to' || value === 'entity'}
export function graphSearchType(mode:GraphMode,type:GraphEntityType):GraphEntityType {return mode === 'path_finder' ? 'character' : mode === 'migration' ? 'corporation' : type}
function entity(value:unknown):GraphEntity|undefined {
  if(!isRecord(value) || !graphID(value.id)) return
  return {id:graphID(value.id),name:text(value.name),corp_name:text(value.corp_name),alliance_name:text(value.alliance_name)}
}
function rows(value:unknown):Record<string,unknown>[] {
  if(!Array.isArray(value)) throw new Error('Graph data is unavailable')
  return value.filter(isRecord)
}
function entities<T>(value:unknown,read:(row:Record<string,unknown>,base:GraphEntity)=>T):T[] {
  return rows(value).flatMap(row=>{const base = entity(row);return base ? [read(row,base)] : []})
}
function systems(value:unknown):GraphSystem[] {return entities(value,(row,base)=>({...base,alliances:metric(row.alliances),characters:metric(row.characters),active_characters:metric(row.active_characters)}))}
function migrants(value:unknown):GraphMigrant[] {return entities(value,(row,base)=>({...base,current_corp:entity(row.current_corp),previous_corp:entity(row.previous_corp)}))}

// These are projections of the actual Go graph_api.go payloads, including the
// nested affiliations/cooperation/characters fields omitted by the old schema.
export function readGraphData(mode:GraphMode,value:unknown):GraphResult {
  if(!isRecord(value) || value.mode !== undefined && value.mode !== mode) throw new Error('Graph data is unavailable')
  switch(mode) {
    case 'path_finder': {
      if(value.path === null) return {mode,path:null}
      if(!isRecord(value.path)) throw new Error('Graph data is unavailable')
      const path = value.path,nodes = entities(path.nodes,(row,base)=>({...base,corporation:entity(row.corporation),alliance:entity(row.alliance)})),edges = rows(path.edges).map(row=>({weight:metric(row.weight)}))
      if(!Array.isArray(path.nodes) || nodes.length !== path.nodes.length) throw new Error('Graph path is unavailable')
      return {mode,path:{nodes,edges,hops:metric(path.hops)}}
    }
    case 'entity_intel': return {mode,allies:entities(value.allies,(row,base)=>({...base,shared_enemy_kills:metric(row.shared_enemy_kills),mutual_kills:metric(row.mutual_kills)})),enemies:entities(value.enemies,(row,base)=>({...base,kills_given:metric(row.kills_given),kills_taken:metric(row.kills_taken)}))}
    case 'hunting_grounds': return {mode,systems:systems(value.systems)}
    case 'hot_zones': return {mode,systems:systems(value.systems)}
    case 'census': {const totals = isRecord(value.totals) ? value.totals : {};return {mode,corps:entities(value.corps,(row,base)=>({...base,total:metric(row.total),fcs:metric(row.fcs),logis:metric(row.logis),caps:metric(row.caps)})),totals:{characters:metric(totals.characters),fcs:metric(totals.fcs),logis:metric(totals.logis),caps:metric(totals.caps)}}}
    case 'spy_check': return {mode,suspects:entities(value.suspects,(row,base)=>({...base,total_flights:metric(row.total_flights),connections:Array.isArray(row.connections) ? row.connections.filter(isRecord).map(connection=>({name:text(connection.name),flights:metric(connection.flights)})) : []}))}
    case 'migration': return {mode,corp_name:text(value.corp_name),departed:migrants(value.departed),joined:migrants(value.joined)}
    case 'coalitions': return {mode,coalitions:rows(value.coalitions).map(row=>({id:metric(row.id),alliances:entities(row.alliances,(item,base)=>({...base,cooperation:metric(item.cooperation)}))}))}
    case 'rivalries': return {mode,items:rows(value.items).flatMap(row=>{const a = entity(row.entity_a),b = entity(row.entity_b);return a && b ? [{entity_a:a,entity_b:b,mutual_kills:metric(row.mutual_kills),total_isk:metric(row.total_isk)}] : []})}
  }
}
export function readGraphHits(value:unknown,type:GraphEntityType):GraphHit[] {
  if(!isRecord(value) || !Array.isArray(value.hits)) throw new Error('Search data is unavailable')
  return value.hits.flatMap((hit:unknown)=>{
    if(!isRecord(hit) || typeof hit.name !== 'string' || hit.type !== undefined && hit.type !== type) return []
    const raw = typeof hit.id === 'string' ? hit.id.replace(/^[a-z]+_/i,'') : hit.id,id = graphID(raw)
    return id ? [{id,name:hit.name,ticker:text(hit.ticker),corporation_name:text(hit.corporation_name),alliance_name:text(hit.alliance_name),alliance_ticker:text(hit.alliance_ticker)}] : []
  }).slice(0,8)
}
