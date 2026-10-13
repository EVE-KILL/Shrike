import { expect, test } from 'vitest';
import type {Fit,FitModule,FitSlotType} from '../src/fit/types';
import type {SdeData} from '../dogma/src/metadata';
import type {ComputedAttribute} from '../dogma/src/types';
import { hullSlotCount, fittingIssues, remainingHardpoints } from '../src/browser/fit-capacity';

function fixture() {
  const names = ['hiSlots','hiSlotModifier','medSlots','medSlotModifier','lowSlots','lowSlotModifier','rigSlots','maxSubSystems','turretSlotsLeft','turretHardPointModifier','launcherSlotsLeft','launcherHardPointModifier','cpuFree','powerFree','droneCapacity','droneBandwidth','droneBandwidthLoad','upgradeCapacity','upgradeLoad'];
  const ids = new Map(names.map((name,i)=>[name,i+1]));
  const attrs = (values: Record<string,number>) => Object.entries(values).map(([name,value])=>({attributeID:ids.get(name)!,value}));
  const sde:Pick<SdeData,'attributeNameToId'|'types'|'typeDogma'> = {attributeNameToId:ids,types:new Map([[2454,{name:"Hobgoblin",volume:5,groupID:100,categoryID:18,published:true,marketGroupID:null,metaGroupID:null,factionID:null,raceID:null,capacity:null,mass:null,radius:null}]]),typeDogma:new Map([
    [1,{dogmaEffects:[],dogmaAttributes:attrs({hiSlots:0,medSlots:0,lowSlots:0,rigSlots:3,maxSubSystems:5,turretSlotsLeft:0})}],
    [10,{dogmaEffects:[],dogmaAttributes:attrs({hiSlotModifier:3,medSlotModifier:2,lowSlotModifier:3,turretHardPointModifier:3})}],
    [11,{dogmaEffects:[],dogmaAttributes:attrs({hiSlotModifier:3,medSlotModifier:3,lowSlotModifier:2,turretHardPointModifier:3})}],
    [12,{dogmaEffects:[],dogmaAttributes:attrs({hiSlotModifier:2,lowSlotModifier:1})}],
    [13,{dogmaEffects:[],dogmaAttributes:[]}],
    [100,{dogmaAttributes:[],dogmaEffects:[{effectID:1000,isDefault:false}]}],
  ])};
  const fit:Fit={name:"Capacity fixture",description:"",shipTypeId:1,modules:[...Array.from({length:4},(_,i):FitModule=>({typeId:10+i,state:'Active',slot:{type:'SubSystem',index:i+1}})),...Array.from({length:2},(_,i):FitModule=>({typeId:100,state:'Active',slot:{type:'High',index:i+1}}))],drones:[{typeId:2454,states:{Active:3,Passive:2}}],cargo:[]};
  const stats={hull:{attributes:new Map<number,ComputedAttribute>(attrs({rigSlots:3,maxSubSystems:5,droneCapacity:50,droneBandwidth:25,droneBandwidthLoad:15,turretSlotsLeft:1,upgradeCapacity:400,upgradeLoad:300}).map(a=>[a.attributeID,{base_value:a.value,value:null,effects:[]}]))}};
  const effects=new Map([['turretFitted',1000]]);
  return {fit,sde,stats,effects,set(name:string,value:number){stats.hull.attributes.set(ids.get(name)!,{base_value:0,value,effects:[]})}};
}

test('strategic cruiser subsystem slots and hardpoints remain available when dogma omits their hull modifiers',()=>{
  const {fit,sde,stats,effects}=fixture();
  expect((['High','Medium','Low','Rig','SubSystem'] as FitSlotType[]).map(slot=>hullSlotCount(fit,sde,stats,slot))).toEqual([8,5,6,3,4]);
  expect(hullSlotCount(fit,sde,null,'High')).toBe(8);
  expect(remainingHardpoints(fit,sde,stats,effects,100)).toBe(4);
  expect(fittingIssues(fit,sde,stats,effects)).toEqual([]);
});

test('validity retains the source missing-subsystem, resources, bay, hardpoint and calibration checks',()=>{
  const {fit,sde,stats,effects,set}=fixture();
  fit.modules=fit.modules.filter(m=>m.typeId!==13);
  expect(fittingIssues(fit,sde,stats,effects)).toEqual(['1 subsystem slots empty']);
  set('cpuFree',-.02);set('powerFree',-3);set('droneCapacity',20);set('droneBandwidthLoad',30);set('upgradeLoad',500);
  fit.modules.push(...Array.from({length:5},(_,i):FitModule=>({typeId:100,state:'Active',slot:{type:'High',index:i+3}})));
  expect(fittingIssues(fit,sde,stats,effects)).toEqual(['CPU exceeded by 0.0 tf','Powergrid exceeded by 3.0 MW','1 subsystem slots empty','Drone bay capacity exceeded','Drone bandwidth exceeded','Turret hardpoints exceeded','Calibration exceeded']);
});
