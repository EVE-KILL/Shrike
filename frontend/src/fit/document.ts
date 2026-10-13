import type { Fit, FitCargo, FitDrone, FitState } from './types'
import type { SaveFitItem } from './toServer'

export { fitToSaveBody } from './toServer'

const slots = ['High', 'Medium', 'Low', 'Rig', 'SubSystem'] as const
const states: readonly FitState[] = ['Passive', 'Online', 'Active', 'Overload']

type SavedFitItem = Omit<SaveFitItem, 'charge_type_id' | 'quantity'> & {
  charge_type_id?: number | null
  quantity?: number
}

// Saved rows, editor drafts and forked fits share one slot/state/quantity contract.
export function savedFitToUI({ fit, items }: {
  fit: { ship_type_id: number; name: string; description?: string | null }
  items: readonly SavedFitItem[]
}): Fit {
  const drones = new Map<number, FitDrone>(), cargo = new Map<number, FitCargo>()
  for (const item of items) {
    const quantity = item.quantity ?? 1
    if (item.slot_group === 6) {
      let drone = drones.get(item.type_id)
      if (!drone) {
        drone = { typeId: item.type_id, states: { Active: 0, Passive: 0 } }
        drones.set(item.type_id, drone)
      }
      drone.states[item.state === 2 ? 'Active' : 'Passive'] += quantity
    } else if (item.slot_group === 7) {
      let stack = cargo.get(item.type_id)
      if (!stack) {
        stack = { typeId: item.type_id, quantity: 0 }
        cargo.set(item.type_id, stack)
      }
      stack.quantity += quantity
    }
  }
  return {
    shipTypeId: fit.ship_type_id, name: fit.name, description: fit.description ?? '',
    modules: items.filter(item => item.slot_group >= 1 && item.slot_group <= 5).map(item => ({
      typeId: item.type_id, slot: { type: slots[item.slot_group - 1], index: item.ordinal + 1 },
      state: states[item.state] ?? 'Active', ...(item.charge_type_id ? { charge: { typeId: item.charge_type_id } } : {}),
    })),
    drones: [...drones.values()], cargo: [...cargo.values()],
  }
}

export function enforcePropmodExclusivity(fit: Fit, sde: {
  types: ReadonlyMap<number, { groupID: number }>
}): void {
  let active = false
  for (const module of fit.modules) {
    if (!['Active', 'Overload'].includes(module.state) || sde.types.get(module.typeId)?.groupID !== 46) continue
    if (active) module.state = 'Online'; else active = true
  }
}
