import {isRecord} from './json'
import type {EsiLogRow} from '../../utils/esiLog'
import type {EsiVolumeHour} from '../../utils/esiVolume'

export interface EsiLogData {rows:EsiLogRow[];sources:string[];total:number;pages:number}
function json(text:string|undefined):unknown{try{return JSON.parse(text??'null')}catch{return null}}
const number=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)
const nullableNumber=(value:unknown)=>value===null||number(value)
const optionalNumber=(value:unknown)=>value===undefined||nullableNumber(value)
const optionalString=(value:unknown)=>value===undefined||value===null||typeof value==='string'
const optionalNumbers=(value:unknown)=>value===undefined||value===null||(Array.isArray(value)&&value.every(number))
function row(value:unknown):value is EsiLogRow{
  return isRecord(value)&&number(value.id)&&typeof value.created_at==='string'&&typeof value.endpoint==='string'&&
    typeof value.endpoint_type==='string'&&typeof value.endpoint_action==='string'&&typeof value.method==='string'&&typeof value.source==='string'&&
    nullableNumber(value.status_code)&&typeof value.success==='boolean'&&(value.error_message===null||typeof value.error_message==='string')&&
    nullableNumber(value.request_duration_ms)&&nullableNumber(value.items_returned)&&nullableNumber(value.new_items)&&
    optionalNumber(value.endpoint_entity_id)&&optionalString(value.endpoint_entity_name)&&optionalNumber(value.character_id)&&optionalString(value.character_name)&&
    optionalNumbers(value.new_item_ids)&&optionalNumbers(value.stored_item_ids)
}
export function parseEsiLogData(value:unknown):EsiLogData{
  const data=isRecord(value)?value:{}
  return {rows:Array.isArray(data.rows)?data.rows.filter(row):[],sources:Array.isArray(data.sources)?data.sources.filter((source):source is string=>typeof source==='string'):[],
    total:typeof data.total==='number'?data.total:0,pages:typeof data.pages==='number'?data.pages:0}
}
export function readEsiLogSeed(text:string|undefined){return parseEsiLogData(json(text))}
export function readEsiVolumeSeed(text:string|undefined):EsiVolumeHour[]{
  const hours=json(text)
  return Array.isArray(hours)?hours.filter((hour):hour is EsiVolumeHour=>isRecord(hour)&&typeof hour.hour==='string'&&number(hour.total)&&number(hour.errors)&&(hour.new_items===undefined||number(hour.new_items))):[]
}
