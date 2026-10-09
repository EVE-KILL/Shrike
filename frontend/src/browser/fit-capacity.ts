import type {FitSlotType} from '../fit/types'
import type {HullStats,CapacityFit,CapacitySde} from './shared/fit-render'

type Hardpoint = 'launcherSlotsLeft'|'turretSlotsLeft'
// Port of Nuxt useHullCapacity / useFitValidity. Strategic cruiser slot and
// hardpoint modifiers live on subsystems and may be absent from the hull result.
export const slotAttributes = { High: ['hiSlots', 'hiSlotModifier'], Medium: ['medSlots', 'medSlotModifier'], Low: ['lowSlots', 'lowSlotModifier'], Rig: ['rigSlots'], SubSystem: ['maxSubSystems'] } as const satisfies Record<FitSlotType,readonly [string,string?]>;
export function hullAttribute(stats:HullStats, sde:CapacitySde, name:string) {
  const id = sde.attributeNameToId.get(name);
  const attribute = id === undefined ? undefined : stats?.hull.attributes.get(id);
  return attribute?.value ?? attribute?.base_value ?? 0;
}
export function typeAttribute(sde:CapacitySde, typeId:number, name:string) {
  const id = sde.attributeNameToId.get(name);
  return sde.typeDogma.get(typeId)?.dogmaAttributes?.find(attribute => attribute.attributeID === id)?.value ?? 0;
}
export function assembledAttribute(fit:CapacityFit, sde:CapacitySde, base:string, modifier:string) {
  if (!fit) return 0;
  return typeAttribute(sde, fit.shipTypeId, base) + fit.modules.filter(module => module.slot.type === 'SubSystem').reduce((total, module) => total + typeAttribute(sde, module.typeId, modifier), 0);
}
export function hullSlotCount(fit:CapacityFit, sde:CapacitySde|null|undefined, stats:HullStats, slot:FitSlotType) {
  if (!fit || !sde) return 0;
  const [base, modifier]:readonly [string,string?] = slotAttributes[slot];
  const computed = hullAttribute(stats, sde, base) + (modifier ? hullAttribute(stats, sde, modifier) : 0);
  const fallback = modifier ? assembledAttribute(fit, sde, base, modifier) : stats ? computed : typeAttribute(sde, fit.shipTypeId, base);
  return Math.max(0, Math.min(slot === 'SubSystem' ? 4 : 8, Math.floor(Math.max(computed, fallback))));
}
export function hardpointAttribute(typeId:number, sde:CapacitySde, effectIds:ReadonlyMap<string,number>):Hardpoint|null {
  const effects = sde.typeDogma.get(typeId)?.dogmaEffects ?? [];
  if (effects.some(effect => effect.effectID === effectIds.get('launcherFitted'))) return 'launcherSlotsLeft';
  if (effects.some(effect => effect.effectID === effectIds.get('turretFitted'))) return 'turretSlotsLeft';
  return null;
}
export function hardpointLimit(fit:CapacityFit, sde:CapacitySde, attribute:Hardpoint) {
  return assembledAttribute(fit, sde, attribute, attribute === 'launcherSlotsLeft' ? 'launcherHardPointModifier' : 'turretHardPointModifier');
}
export function remainingHardpoints(fit:CapacityFit, sde:CapacitySde, stats:HullStats, effectIds:ReadonlyMap<string,number>, typeId:number) {
  const attribute = hardpointAttribute(typeId, sde, effectIds);
  if (!fit || !attribute) return 0;
  const fitted = fit.modules.filter(module => hardpointAttribute(module.typeId, sde, effectIds) === attribute).length;
  return Math.max(0, Math.floor(Math.max(hullAttribute(stats, sde, attribute), hardpointLimit(fit, sde, attribute) - fitted)));
}
export function fittingIssues(fit:CapacityFit, sde:CapacitySde, stats:HullStats, effectIds:ReadonlyMap<string,number>) {
  if (!fit?.shipTypeId || !stats) return [];
  const value = (name:string) => hullAttribute(stats, sde, name), result:string[] = [];
  const cpu = -value('cpuFree'), power = -value('powerFree');
  if (cpu > .01) result.push(`CPU exceeded by ${cpu.toFixed(1)} tf`);
  if (power > .01) result.push(`Powergrid exceeded by ${power.toFixed(1)} MW`);
  for (const slot of Object.keys(slotAttributes) as FitSlotType[]) {
    const used = fit.modules.filter(module => module.slot.type === slot).length, count = hullSlotCount(fit, sde, stats, slot);
    if (used > count) result.push(`${used - count} too many ${slot.toLowerCase()} modules`);
    if (slot === 'SubSystem' && count > 0 && used < count) result.push(`${count - used} subsystem slots empty`);
  }
  const droneVolume = fit.drones.reduce((total, drone) => total + (sde.types.get(drone.typeId)?.volume ?? 0) * (drone.states.Active + drone.states.Passive), 0);
  if (droneVolume > value('droneCapacity')) result.push('Drone bay capacity exceeded');
  if (value('droneBandwidthLoad') > value('droneBandwidth')) result.push('Drone bandwidth exceeded');
  for (const [attribute, label] of [['launcherSlotsLeft', 'Launcher'], ['turretSlotsLeft', 'Turret']] as const) {
    const fitted = fit.modules.filter(module => hardpointAttribute(module.typeId, sde, effectIds) === attribute).length;
    if (fitted > hardpointLimit(fit, sde, attribute)) result.push(`${label} hardpoints exceeded`);
  }
  if (value('upgradeCapacity') > 0 && value('upgradeLoad') > value('upgradeCapacity')) result.push('Calibration exceeded');
  return result;
}
