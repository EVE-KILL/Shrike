import type {SdeData} from '../../dogma/src/metadata'
import type {EsfFit,FitStats} from '../../dogma/src/types'
import type {FittingDetailLegacyResponses,FittingsShipFamiliesResponses} from '../../shared/api/types.gen'
import type {Fit,FitSlot,FitSlotType,FitState} from './types'
import type {BayKind} from '../browser/shared/fit-render'
import {isRecord} from '../browser/shared/json'
import {searchID} from '../browser/shared/search-model'

export const editorSlots = ['High','Medium','Low','Rig','SubSystem'] as const satisfies readonly FitSlotType[]
export const isEditorSlot = (value:unknown):value is FitSlotType=>editorSlots.some(slot=>slot === value)
export const isEditorState = (value:unknown):value is FitState=>['Passive','Online','Active','Overload'].some(state=>state === value)
export type CatalogSlot = FitSlotType|'DroneBay'|'Charge'
export type CatalogEntry = {id:number;name:string;group:string;slot:CatalogSlot|null;path:string[];meta:number}
export type ContextSlot = (FitSlot & {typeId:number;chargeTypeId:number|null;droneId?:never}) | {droneId:number;type?:never;index?:never;typeId?:never;chargeTypeId?:never}
type FamilyWire = FittingsShipFamiliesResponses[200]['families'][number]
export type EditorFamily = Pick<FamilyWire,'family_hash'|'total_uses'|'modules'|'drones'>
export type EditorFamilies = {window_days:number;families:EditorFamily[]}
type SavedWire = FittingDetailLegacyResponses[200]
export type SavedEditorFit = Omit<SavedWire,'fit'> & {fit:Pick<SavedWire['fit'],'description'> & {fit_id:string;ship_type_id:number;name:string;visibility:number}}
export type EditorWorkspace = {tab:'hulls'|'hardware';view:'ship'|'table';bay:BayKind|null;expanded:boolean}
export type EditorState = EditorWorkspace & {
 sde:SdeData;effectIds:Map<string,number>;fit:Fit|null;stats:FitStats|null;values:Record<string,number|null>|null;
 search:string;catalogSelection:number;catalogMode:'modules'|'charges';filterSlots:Set<CatalogSlot>;hullRestriction:boolean;catalogGroups:CatalogEntry[][];
 selected:FitSlot|null;undo:Fit[];redo:Fit[];generation:number;urlGeneration:number;priceGeneration:number;price:number|null;skillLevel:3|4|5;
 savedId:string|null;customFit:Fit|null;customStats:FitStats|null;customPrice:number|null;customGeneration:number;selectedFamily:string|null;
 observedSnapshot:Fit|null;showComparison:boolean;variantHull:number|null;variantsHTML:string|null;variantsData:EditorFamilies|null;variantsLoading:boolean;
 variantGeneration:number;previewGeneration:number;previewTarget:HTMLElement|null;columnWidths:Array<number|null>;statExpanded:Map<string,boolean>;
}
export function initialEditorState(sde:SdeData,query = new URLSearchParams()):EditorState {
 return {sde,effectIds:new Map([...sde.dogmaEffects].map(([id,effect])=>[effect.name,id])),fit:null,stats:null,values:null,tab:'hulls',view:query.get('view') === 'table' ? 'table' : 'ship',search:'',catalogSelection:0,catalogMode:'modules',filterSlots:new Set(),hullRestriction:true,catalogGroups:[],selected:null,bay:null,undo:[],redo:[],generation:0,urlGeneration:0,priceGeneration:0,price:null,skillLevel:5,expanded:false,savedId:null,customFit:null,customStats:null,customPrice:null,customGeneration:0,selectedFamily:null,observedSnapshot:null,showComparison:false,variantHull:null,variantsHTML:null,variantsData:null,variantsLoading:false,variantGeneration:0,previewGeneration:0,previewTarget:null,columnWidths:[null,55,58,46,46,60,120],statExpanded:new Map()}
}
export function editorToEngineFit(fit:Fit):EsfFit {
 return {ship_type_id:fit.shipTypeId,modules:fit.modules.map(module=>({type_id:module.typeId,slot:module.slot,state:module.state === 'Preview' ? 'Active' : module.state,charge:module.charge ? {type_id:module.charge.typeId} : null})),drones:fit.drones.flatMap(drone=>[...Array.from({length:drone.states.Active},()=>({type_id:drone.typeId,state:'Active' as const})),...Array.from({length:drone.states.Passive},()=>({type_id:drone.typeId,state:'Passive' as const}))])}
}
export function restoreEditorWorkspace(value:unknown,query:URLSearchParams,width:number):EditorWorkspace {
 const state:EditorWorkspace = {tab:'hulls',view:'ship',bay:null,expanded:false}
 if(isRecord(value)) {if(value.sidebarTab === 'hulls' || value.sidebarTab === 'hardware') state.tab = value.sidebarTab;if(value.fittingView === 'ship' || value.fittingView === 'table') state.view = value.fittingView;if(value.expandedBay === 'drones' || value.expandedBay === 'cargo') state.bay = value.expandedBay;if(value.workspaceExpanded && width >= 1024) state.expanded = true}
 if(query.get('view') === 'table') state.view = 'table'
 return state
}
const nonnegativeInteger = (value:unknown):value is number=>typeof value === 'number' && Number.isSafeInteger(value) && value >= 0
export function readSavedEditorFit(value:unknown):SavedEditorFit {
 if(!isRecord(value) || !isRecord(value.fit) || typeof value.fit.fit_id !== 'string' || !/^[a-zA-Z0-9]{14}$/.test(value.fit.fit_id) || !searchID(value.fit.ship_type_id) || typeof value.fit.name !== 'string' || value.fit.description != null && typeof value.fit.description !== 'string' || !nonnegativeInteger(value.fit.visibility) || value.items != null && !Array.isArray(value.items)) throw new Error('Invalid saved fitting')
 const viewer = value.viewer_rating ?? null;if(viewer !== null && (typeof viewer !== 'number' || !Number.isInteger(viewer) || viewer < 1 || viewer > 5)) throw new Error('Invalid saved fitting rating')
 const rawItems = value.items ?? []
 if(!Array.isArray(rawItems)) throw new Error('Invalid saved fitting items')
 const items = rawItems.map((item:unknown):SavedWire['items'][number]=>{
  if(!isRecord(item) || !searchID(item.type_id) || !nonnegativeInteger(item.slot_group) || item.slot_group < 1 || item.slot_group > 7 || !nonnegativeInteger(item.ordinal) || !nonnegativeInteger(item.state) || item.state > 3 || item.quantity !== undefined && !nonnegativeInteger(item.quantity) || item.charge_type_id != null && !searchID(item.charge_type_id)) throw new Error('Invalid saved fitting item')
  return {type_id:searchID(item.type_id),slot_group:item.slot_group,ordinal:item.ordinal,state:item.state,quantity:typeof item.quantity === 'number' ? item.quantity : 1,charge_type_id:item.charge_type_id == null ? null : searchID(item.charge_type_id)}
 })
 return {fit:{fit_id:value.fit.fit_id,ship_type_id:searchID(value.fit.ship_type_id),name:value.fit.name,description:typeof value.fit.description === 'string' ? value.fit.description : null,visibility:value.fit.visibility},items,viewer_rating:viewer}
}
export function readEditorFamilies(value:unknown):EditorFamilies {
 if(!isRecord(value) || !nonnegativeInteger(value.window_days) || value.families != null && !Array.isArray(value.families)) throw new Error('Invalid observed fittings')
 const rawFamilies = value.families ?? []
 if(!Array.isArray(rawFamilies)) throw new Error('Invalid observed fittings')
 const families = rawFamilies.map((family:unknown):EditorFamily=>{
  if(!isRecord(family) || typeof family.family_hash !== 'string' || !family.family_hash || !nonnegativeInteger(family.total_uses) || family.modules != null && !Array.isArray(family.modules) || family.drones != null && !Array.isArray(family.drones)) throw new Error('Invalid observed fitting family')
  const rawModules = family.modules ?? [],rawDrones = family.drones ?? []
  if(!Array.isArray(rawModules) || !Array.isArray(rawDrones)) throw new Error('Invalid observed fitting items')
  const modules = rawModules.map((module:unknown):EditorFamily['modules'][number]=>{if(!isRecord(module) || !searchID(module.type_id) || !nonnegativeInteger(module.slot_group) || module.slot_group < 1 || module.slot_group > 5 || !nonnegativeInteger(module.ordinal) || module.charge_type_id != null && module.charge_type_id !== 0 && !searchID(module.charge_type_id)) throw new Error('Invalid observed fitting module');return {type_id:searchID(module.type_id),slot_group:module.slot_group,ordinal:module.ordinal,name:typeof module.name === 'string' ? module.name : null,charge_type_id:module.charge_type_id == null || module.charge_type_id === 0 ? null : searchID(module.charge_type_id),charge_name:typeof module.charge_name === 'string' ? module.charge_name : null}})
  const drones = rawDrones.map((drone:unknown):EditorFamily['drones'][number]=>{if(!isRecord(drone) || !searchID(drone.type_id) || !nonnegativeInteger(drone.quantity)) throw new Error('Invalid observed fitting drone');return {type_id:searchID(drone.type_id),quantity:drone.quantity,name:typeof drone.name === 'string' ? drone.name : null}})
  return {family_hash:family.family_hash,total_uses:family.total_uses,modules,drones}
 })
 return {window_days:value.window_days,families}
}
export function readEditorPrices(value:unknown,ids:readonly number[]):Map<number,number> {
 if(!isRecord(value) || !isRecord(value.prices)) throw new Error('Invalid fitting prices')
 const prices = value.prices
 return new Map(ids.map(id=>{const price = Number(prices[id] ?? 0);if(!Number.isFinite(price) || price < 0) throw new Error('Invalid fitting price');return [id,price]}))
}
