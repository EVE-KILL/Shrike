import {isRecord} from './json'
import {isTrackerTargetType,type EntityTracker} from '../../models/trackers'

export function readTrackerSeed(text:string|undefined):EntityTracker[]{
  let rows:unknown
  try{rows=JSON.parse(text??'[]')}catch{return []}
  if(!Array.isArray(rows))return []
  return rows.filter((row):row is EntityTracker=>isRecord(row)&&
    typeof row.id==='number'&&Number.isSafeInteger(row.id)&&row.id>0&&
    typeof row.target_id==='number'&&Number.isSafeInteger(row.target_id)&&row.target_id>0&&
    isTrackerTargetType(row.target_type)&&typeof row.target_name==='string'&&
    (row.target_ticker===null||typeof row.target_ticker==='string')&&
    typeof row.enabled==='boolean'&&typeof row.notifications_enabled==='boolean'&&
    typeof row.created_at==='string'&&typeof row.updated_at==='string'&&
    typeof row.event_count==='number'&&typeof row.unread_notifications==='number'&&
    (row.last_event_at===null||typeof row.last_event_at==='string'))
}
