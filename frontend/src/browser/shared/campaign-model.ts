import {isRecord} from './json'
import type {PickedEntity} from '../browser-types'
export type CampaignEntity=PickedEntity&{type:'character'|'corporation'|'alliance'}
export type CampaignLocation=PickedEntity&{type:'system'|'constellation'|'region'}
export function isCampaignEntity(value:unknown):value is CampaignEntity{return isRecord(value)&&Number.isSafeInteger(value.id)&&Number(value.id)>0&&typeof value.name==='string'&&['character','corporation','alliance'].includes(String(value.type))}
export function isCampaignLocation(value:unknown):value is CampaignLocation{return isRecord(value)&&Number.isSafeInteger(value.id)&&Number(value.id)>0&&typeof value.name==='string'&&['system','constellation','region'].includes(String(value.type))}
export type CampaignSide={name:string;entities:CampaignEntity[]}
export interface CampaignEditView{Name:string;Description:string;EndTime:string;Mode:'area'|'single'|'conflict';Visibility:number;Ongoing:boolean;Locked:boolean;Sides:CampaignSide[];Allowed:CampaignEntity[]}
export function readCampaignEdit(raw:unknown):CampaignEditView{
 if(!isRecord(raw)||!['area','single','conflict'].includes(String(raw.Mode))||typeof raw.Name!=='string'||typeof raw.Description!=='string'||typeof raw.EndTime!=='string'||!Array.isArray(raw.Sides)||!Array.isArray(raw.Allowed))throw new Error('The campaign editor is unavailable')
 const sides=raw.Sides.map(side=>{if(!isRecord(side)||typeof side.name!=='string'||!Array.isArray(side.entities)||!side.entities.every(isCampaignEntity))throw new Error('The campaign participants are unavailable');return {name:side.name,entities:side.entities}})
 if(!raw.Allowed.every(isCampaignEntity))throw new Error('The campaign audience is unavailable')
 return {Name:raw.Name,Description:raw.Description,EndTime:raw.EndTime,Mode:raw.Mode==='area'?'area':raw.Mode==='single'?'single':'conflict',Visibility:Number(raw.Visibility)||0,Ongoing:raw.Ongoing===true,Locked:raw.Locked===true,Sides:sides,Allowed:raw.Allowed}
}
