import type {Fit,FitSlotType} from '../../fit/types'
import type {FitStats,ComputedItem} from '../../../dogma/src/types'
import type {SdeData} from '../../../dogma/src/metadata'

export type BayKind = 'drones'|'cargo'
export type FitRenderAssets = {
 escape:(value:string)=>string;
 iconOf:(id:number,size?:number)=>string;
 nameOf:(id:number)=>string;
}
export type FitIcon = (name:string,size:number)=>string
export type FitFormatter = (value:number,digits?:number)=>string
export type FitRenderState = {
 fit:Fit|null;stats:FitStats|null;sde:SdeData;
}
export type HullStats = {hull:Pick<ComputedItem,'attributes'>}|null|undefined
export type CapacityFit = Pick<Fit,'shipTypeId'|'modules'|'drones'>|null|undefined
export type CapacitySde = Pick<SdeData,'attributeNameToId'|'typeDogma'|'types'>
export type SlotCount = (slot:FitSlotType)=>number
