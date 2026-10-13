import {isRecord} from './json'
export interface RoamPilot {character_id:number;name:string;corporation_name:string}
export interface RoamShip {
  ship_type_id:number;ship_name:string;ship_group_id:number;ship_group_name:string
  killmails:number;losses:number;damage_done:number
}
export interface RoamPilotRow {pilot:RoamPilot;ship:RoamShip|null;iskLost:number}
export interface RoamAggregateRow {
  id:number;name:string;groupId:number;groupName:string;pilotIds:Set<number>
  observations:number;losses:number;iskLost:number;damage:number
}
export interface RoamCompositionOptions {
  view?:'pilots'|'groups'|'ships';search?:string;group?:number|null;losses?:boolean
  sort?:'default'|'ship'|'pilot'|'pilots'|'observations'|'losses'|'iskLost'|'damage';descending?:boolean
}
export function selectRoamCompositionRows(rows:RoamPilotRow[],options?:RoamCompositionOptions&{view?:'pilots'}):RoamPilotRow[]
export function selectRoamCompositionRows(rows:RoamPilotRow[],options:RoamCompositionOptions&{view:'ships'|'groups'}):RoamAggregateRow[]
export function selectRoamCompositionRows(rows:RoamPilotRow[],options:RoamCompositionOptions):Array<RoamPilotRow|RoamAggregateRow>
export function selectRoamCompositionRows(rows:RoamPilotRow[],options:RoamCompositionOptions={}):Array<RoamPilotRow|RoamAggregateRow> {
  const {view='pilots',search='',group=null,losses=false,sort='default',descending=true}=options
  const term=search.trim().toLowerCase()
  const filtered=rows.filter(row=>(!group||row.ship?.ship_group_id===group)
    &&(!losses||(row.ship?.losses??0)>0)
    &&(!term||[row.pilot.name,row.pilot.corporation_name,row.ship?.ship_name??'',row.ship?.ship_group_name??''].some(value=>value.toLowerCase().includes(term))))
  const number=(a:number,b:number)=>descending?b-a:a-b
  const names=(a:string,b:string)=>descending?b.localeCompare(a):a.localeCompare(b)
  if(view==='pilots')return filtered.sort((a,b)=>{
    switch(sort){
      case 'ship':return names(a.ship?.ship_name??'',b.ship?.ship_name??'')
      case 'pilot':return names(a.pilot.name,b.pilot.name)
      case 'observations':return number(a.ship?.killmails??0,b.ship?.killmails??0)
      case 'losses':return number(a.ship?.losses??0,b.ship?.losses??0)
      case 'iskLost':return number(a.iskLost,b.iskLost)
      case 'damage':return number(a.ship?.damage_done??0,b.ship?.damage_done??0)
      default:return (b.ship?.killmails??0)-(a.ship?.killmails??0)||(b.ship?.damage_done??0)-(a.ship?.damage_done??0)||a.pilot.name.localeCompare(b.pilot.name)
    }
  })
  const totals=new Map<string,RoamAggregateRow>()
  for(const row of filtered){
    if(!row.ship)continue
    const key=view==='groups'?String(row.ship.ship_group_id||'unknown'):String(row.ship.ship_type_id)
    const entry=totals.get(key)??{id:view==='groups'?row.ship.ship_group_id:row.ship.ship_type_id,
      name:view==='groups'?row.ship.ship_group_name||'Unknown group':row.ship.ship_name,
      groupId:row.ship.ship_group_id,groupName:row.ship.ship_group_name,pilotIds:new Set<number>(),observations:0,losses:0,iskLost:0,damage:0}
    entry.pilotIds.add(row.pilot.character_id);entry.observations+=row.ship.killmails;entry.losses+=row.ship.losses
    entry.iskLost+=row.iskLost;entry.damage+=row.ship.damage_done;totals.set(key,entry)
  }
  return [...totals.values()].sort((a,b)=>{
    switch(sort){
      case 'ship':return names(a.name,b.name)
      case 'pilots':return number(a.pilotIds.size,b.pilotIds.size)
      case 'observations':return number(a.observations,b.observations)
      case 'losses':return number(a.losses,b.losses)
      case 'iskLost':return number(a.iskLost,b.iskLost)
      case 'damage':return number(a.damage,b.damage)
      default:return b.pilotIds.size-a.pilotIds.size||b.observations-a.observations||a.name.localeCompare(b.name)
    }
  })
}

// The JSON script is an unknown transport boundary, including nullable hulls.
export function readRoamComposition(raw:unknown):{rows:RoamPilotRow[];pilots:number}{
 const number=(value:unknown)=>typeof value==='number'&&Number.isFinite(value)?value:0
 const text=(value:unknown)=>typeof value==='string'?value:''
 if(!isRecord(raw)||!Array.isArray(raw.rows))throw new Error('Roam composition is unavailable')
 return {pilots:number(raw.pilots),rows:raw.rows.flatMap(value=>{if(!isRecord(value)||!isRecord(value.pilot))return [];const pilot:RoamPilot={character_id:number(value.pilot.character_id),name:text(value.pilot.name),corporation_name:text(value.pilot.corporation_name)},hull=value.ship;const ship:RoamShip|null=isRecord(hull)?{ship_type_id:number(hull.ship_type_id),ship_name:text(hull.ship_name),ship_group_id:number(hull.ship_group_id),ship_group_name:text(hull.ship_group_name),killmails:number(hull.killmails),losses:number(hull.losses),damage_done:number(hull.damage_done)}:null;return [{pilot,ship,iskLost:number(value.iskLost)}]})}
}
