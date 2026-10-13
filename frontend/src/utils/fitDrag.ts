type HullDropSlot = 'High' | 'Medium' | 'Low' | 'Rig' | 'SubSystem' | 'DroneBay'
type DragPayload = { getData: (type: string) => string }
const hullDropSlots = new Set(['High', 'Medium', 'Low', 'Rig', 'SubSystem', 'DroneBay'])

// HullDraggable's existing contract: the hull adds fresh sidebar items only.
// Fitted-slot moves and charges belong to their explicit drop targets.
export function readHullDrop(data: DragPayload | null | undefined): { typeId: number; slotType: HullDropSlot } | null {
    if (!data) return null
    const typeId = Number.parseInt(data.getData('application/esf-type-id'), 10)
    const slotIndex = Number.parseInt(data.getData('application/esf-slot-index'), 10)
    const slotType = data.getData('application/esf-slot-type')
    if (!Number.isInteger(typeId) || Number.isInteger(slotIndex) || !hullDropSlots.has(slotType)) return null
    return { typeId, slotType: slotType as HullDropSlot }
}
