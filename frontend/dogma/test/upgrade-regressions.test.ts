import {describe,expect,test} from 'vitest'
import {calculateFit,loadSde,getHullStat,getItemStat} from '../src/index'
import type {EsfFit,EsfModule} from '../src/types'
async function type(name:string){const id=(await loadSde()).typeNameToId.get(name);if(!id)throw new Error(`Missing type ${name}`);return id}
async function module(name:string,slot:'High'|'Medium'|'Low'|'Rig'|'SubSystem',index=1,charge?:string,state:EsfModule['state']='Active'):Promise<EsfModule>{return{type_id:await type(name),slot:{type:slot,index},state,charge:charge?{type_id:await type(charge)}:undefined}}
async function fit(ship:string,modules:EsfModule[]):Promise<EsfFit>{return{ship_type_id:await type(ship),modules,drones:[]}}
async function value(item:any,name:string){const result=await getItemStat(item,name);expect(result).not.toBeNull();return result!}

describe('retained additions and upstream fixes',()=>{
 test('hybrid ammunition optimal modifiers apply once',async()=>{
  const short=await calculateFit(await fit('Rifter',[await module('150mm Railgun II','High',1,'Antimatter Charge S')]))
  const long=await calculateFit(await fit('Rifter',[await module('150mm Railgun II','High',1,'Iron Charge S')]))
  const rangeRatio=await value(long.items[0],'maxRange')/await value(short.items[0],'maxRange')
  const bonusRatio=await value(long.items[0]!.charge,'weaponRangeMultiplier')/await value(short.items[0]!.charge,'weaponRangeMultiplier')
  expect(rangeRatio).toBeCloseTo(bonusRatio,5)
  expect(rangeRatio).toBeGreaterThan(2)
 })
 test('projectile ammunition falloff and tracking modifiers apply once',async()=>{
  const barrage=await calculateFit(await fit('Rifter',[await module('200mm AutoCannon II','High',1,'Barrage S')]))
  const hail=await calculateFit(await fit('Rifter',[await module('200mm AutoCannon II','High',1,'Hail S')]))
  for(const [target,bonus]of [['falloff','fallofMultiplier'],['trackingSpeed','trackingSpeedMultiplier']]){
   const ratio=await value(barrage.items[0],target!)/await value(hail.items[0],target!)
   const expected=await value(barrage.items[0]!.charge,bonus!)/await value(hail.items[0]!.charge,bonus!)
   expect(ratio).toBeCloseTo(expected,5)
  }
 })
 test('EWAR scripts change strength without multiplying module activation range',async()=>{
  const plain=await calculateFit(await fit('Rifter',[await module('Tracking Disruptor II','Medium')]))
  const scripted=await calculateFit(await fit('Rifter',[await module('Tracking Disruptor II','Medium',1,'Optimal Range Disruption Script')]))
  expect(await value(scripted.items[0],'maxRange')).toBeCloseTo(await value(plain.items[0],'maxRange'),5)
  expect(Math.abs(await value(scripted.items[0],'maxRangeBonus'))).toBeGreaterThan(Math.abs(await value(plain.items[0],'maxRangeBonus')))
 })
 test('ancillary shield repairs and charged armor repairs need no injected effects',async()=>{
  const shield=await calculateFit(await fit('Rifter',[await module('Small Ancillary Shield Booster','Medium',1,'Cap Booster 50')]))
  const boost=await getHullStat(shield,'shieldBoostRate')
  expect(boost).toBeGreaterThan(0)
  expect(boost).toBeCloseTo(await value(shield.items[0],'shieldBonus')/await value(shield.items[0],'duration')*1000,5)
  const empty=await calculateFit(await fit('Rifter',[await module('Small Ancillary Armor Repairer','Low')]))
  const loaded=await calculateFit(await fit('Rifter',[await module('Small Ancillary Armor Repairer','Low',1,'Nanite Repair Paste')]))
  const plain=await getHullStat(empty,'armorRepairRate'),charged=await getHullStat(loaded,'armorRepairRate')
  expect(plain).toBeGreaterThan(0)
  expect(charged!/plain!).toBeCloseTo(3,5)
 })
 test('remote totals exclude offline modules and preserve local repairs',async()=>{
  const stats=await calculateFit(await fit('Armageddon',[
   await module('Large Remote Armor Repairer II','High',1),
   await module('Large Remote Armor Repairer II','High',2,undefined,'Passive'),
   await module('Large Armor Repairer II','Low',1),
  ]))
  expect(await getHullStat(stats,'remoteArmorRepairRate')).toBeCloseTo(await value(stats.items[0],'armorRepairRate'),5)
  expect(await getHullStat(stats,'armorRepairRate')).toBeCloseTo(await value(stats.items[2],'armorRepairRate'),5)
 })
 test('remote capacitor, NOS and neutralizers remain distinct',async()=>{
  const stats=await calculateFit(await fit('Armageddon',[
   await module('Heavy Energy Neutralizer II','High',1),
   await module('Heavy Energy Nosferatu II','High',2),
   await module('Large Remote Capacitor Transmitter II','High',3),
  ]))
  for(const [index,name]of [[0,'energyNeutralizerRate'],[1,'energyNosferatuRate'],[2,'remoteCapTransferRate']]as const){
   const total=await getHullStat(stats,name)
   expect(total).toBeGreaterThan(0)
   expect(total).toBeCloseTo(await value(stats.items[index],'capacitorTransferRate'),5)
  }
 })
 test('legacy passive rigs remain fitted and apply their bonuses',async()=>{
  const weapon=await module('200mm AutoCannon II','High',1,'EMP S')
  const plain=await calculateFit(await fit('Rifter',[weapon]))
  const rigged=await calculateFit(await fit('Rifter',[weapon,await module('Small Projectile Collision Accelerator I','Rig',1,undefined,'Passive')]))
  expect(await getHullStat(rigged,'damageAlpha')).toBeGreaterThan((await getHullStat(plain,'damageAlpha'))!)
  expect(rigged.items[1]!.state).toBe('Passive')
 })
 test('subsystems remain online even when an old document calls them passive',async()=>{
  const names=['Tengu Core - Augmented Graviton Reactor','Tengu Defensive - Amplification Node','Tengu Offensive - Accelerated Ejection Bay','Tengu Propulsion - Interdiction Nullifier']
  const subsystems=await Promise.all(names.map((name,i)=>module(name,'SubSystem',i+1,undefined,'Passive')))
  const passive=await calculateFit(await fit('Tengu',subsystems))
  const online=await calculateFit(await fit('Tengu',subsystems.map(m=>({...m,state:'Online'}))))
  expect(passive.items.every(item=>item.state==='Online')).toBe(true)
  for(const stat of ['cpuOutput','powerOutput','hiSlots','medSlots','lowSlots'])expect(await getHullStat(passive,stat)).toBeCloseTo((await getHullStat(online,stat))!,5)
 })
})
