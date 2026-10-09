/** Read catalogue metadata from the same FlatBuffer passed to Rust. */
import {ByteBuffer} from 'flatbuffers'
import {Sde} from './generated/eve'
import compatibility from '../../../engine/dogma/compat.json'

export interface SdeType {
 name:string;groupID:number;categoryID:number;published:boolean;
 marketGroupID:number|null;metaGroupID:number|null;factionID:number|null;raceID:number|null;
 capacity:number|null;mass:number|null;radius:number|null;volume:number|null;
}
export interface SdeTypeDogma {
 dogmaAttributes:Array<{attributeID:number;value:number}>;
 dogmaEffects:Array<{effectID:number;isDefault:boolean}>;
}
export interface SdeAttribute {
 name:string;displayName?:string|null;defaultValue:number;highIsGood:boolean;stackable:boolean;published:boolean;unitID?:number;
}
export interface SdeEffect {
 name:string;displayName:string|null;effectCategory:number;
 electronicChance:boolean;isAssistance:boolean;isOffensive:boolean;isWarpSafe:boolean;propulsionChance:boolean;rangeChance:boolean;
 dischargeAttributeID:number;durationAttributeID:number;rangeAttributeID:number;falloffAttributeID:number;trackingSpeedAttributeID:number;
 modifierInfo:Array<{domain:number;func:number;operation:number;modifiedAttributeID:number;modifyingAttributeID:number;groupID:number;skillTypeID:number}>;
}
export interface SdeData {
 types:Map<number,SdeType>;typeDogma:Map<number,SdeTypeDogma>;dogmaAttributes:Map<number,SdeAttribute>;dogmaEffects:Map<number,SdeEffect>;
 categories:Map<number,{name:string|null;published:boolean}>;groups:Map<number,{name:string;categoryID:number;published:boolean}>;marketGroups:Map<number,{name:string;parentGroupID:number|null}>;
 attributeNameToId:Map<string,number>;typeNameToId:Map<string,number>;groupNameToId:Map<string,number>;marketGroupNameToId:Map<string,number>;
 bytes:Uint8Array;buildNumber:number;majorVersion:number;
}
export function decodeSdeFromBytes(input:Uint8Array|ArrayBuffer):SdeData {
 const bytes=input instanceof Uint8Array?input:new Uint8Array(input)
 const buffer=new ByteBuffer(bytes)
 if(!Sde.bufferHasIdentifier(buffer))throw new Error('Invalid fitting SDE identifier')
 const root=Sde.getRootAsSde(buffer)
 if(root.majorVersion()<12)throw new Error('Fitting SDE is older than version 12')
 const sde:SdeData={bytes,buildNumber:root.buildNumber(),majorVersion:root.majorVersion(),types:new Map(),typeDogma:new Map(),dogmaAttributes:new Map(),dogmaEffects:new Map(),categories:new Map(),groups:new Map(),marketGroups:new Map(),attributeNameToId:new Map(),typeNameToId:new Map(),groupNameToId:new Map(),marketGroupNameToId:new Map()}
 for(let i=0;i<root.typesLength();i++){
  const t=root.types(i)!;const id=t.id();const name=t.name()!
  sde.types.set(id,{name,groupID:t.groupId(),categoryID:t.categoryId(),published:t.published(),marketGroupID:t.marketGroupId()||null,metaGroupID:t.metaGroupId()||null,factionID:t.factionId()||null,raceID:t.raceId()||null,capacity:t.capacity(),mass:t.mass(),radius:t.radius(),volume:t.volume()})
  sde.typeNameToId.set(name,id)
  const dogmaAttributes=Array.from({length:t.dogmaAttributesLength()},(_,j)=>{const a=t.dogmaAttributes(j)!;return{attributeID:a.attributeId(),value:a.value()}})
  const dogmaEffects=Array.from({length:t.dogmaEffectsLength()},(_,j)=>{const e=t.dogmaEffects(j)!;return{effectID:e.effectId(),isDefault:e.isDefault()}})
  sde.typeDogma.set(id,{dogmaAttributes,dogmaEffects})
 }
 for(let i=0;i<root.dogmaAttributesLength();i++){
  const a=root.dogmaAttributes(i)!;const id=a.id();const name=a.name()!
  sde.dogmaAttributes.set(id,{name,displayName:a.displayName(),defaultValue:a.defaultValue(),highIsGood:a.highIsGood(),stackable:a.stackable(),published:a.published(),unitID:a.unitId()})
  sde.attributeNameToId.set(name,id)
 }
 for(let i=0;i<root.dogmaEffectsLength();i++){
  const e=root.dogmaEffects(i)!
  const modifierInfo=Array.from({length:e.modifiersLength()},(_,j)=>{const m=e.modifiers(j)!;return{domain:m.domain(),func:m.func(),operation:m.operation(),modifiedAttributeID:m.modifiedAttributeId(),modifyingAttributeID:m.modifyingAttributeId(),groupID:m.groupId(),skillTypeID:m.skillTypeId()}})
  sde.dogmaEffects.set(e.id(),{name:e.name()!,displayName:e.displayName(),effectCategory:e.effectCategory(),electronicChance:e.electronicChance(),isAssistance:e.isAssistance(),isOffensive:e.isOffensive(),isWarpSafe:e.isWarpSafe(),propulsionChance:e.propulsionChance(),rangeChance:e.rangeChance(),dischargeAttributeID:e.dischargeAttributeId(),durationAttributeID:e.durationAttributeId(),rangeAttributeID:e.rangeAttributeId(),falloffAttributeID:e.falloffAttributeId(),trackingSpeedAttributeID:e.trackingSpeedAttributeId(),modifierInfo})
 }
 for(let i=0;i<root.categoriesLength();i++){const c=root.categories(i)!;sde.categories.set(c.id(),{name:c.name(),published:c.published()})}
 for(let i=0;i<root.groupsLength();i++){const g=root.groups(i)!;const name=g.name()!;sde.groups.set(g.id(),{name,categoryID:g.categoryId(),published:g.published()});sde.groupNameToId.set(name,g.id())}
 for(let i=0;i<root.marketGroupsLength();i++){const g=root.marketGroups(i)!;const name=g.name()!;sde.marketGroups.set(g.id(),{name,parentGroupID:g.parentGroupId()||null});sde.marketGroupNameToId.set(name,g.id())}
 // Preserve the application's names without copying upstream calculation rules.
 for(const [oldName,newName]of Object.entries(compatibility.aliases)){const id=sde.attributeNameToId.get(newName);if(id!==undefined)sde.attributeNameToId.set(oldName,id)}
 for(const a of compatibility.customAttributes){sde.attributeNameToId.set(a.name,a.id);sde.dogmaAttributes.set(a.id,{name:a.name,defaultValue:0,highIsGood:true,stackable:true,published:true})}
 return sde
}
