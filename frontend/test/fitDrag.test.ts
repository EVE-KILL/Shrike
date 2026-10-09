import { expect, test } from 'vitest'
import { readHullDrop } from '../src/utils/fitDrag'

const payload = (typeId: string, slotType: string, slotIndex = '') => {
    const values: Record<string, string> = {
        'application/esf-type-id': typeId,
        'application/esf-slot-type': slotType,
        'application/esf-slot-index': slotIndex,
    }
    return { getData: (key: string) => values[key] ?? '' }
}

test('fresh modules and drones can auto-fit through a hull drop', () => {
    for (const slotType of ['High', 'Medium', 'Low', 'Rig', 'SubSystem']) {
        expect(readHullDrop(payload('486', slotType))).toEqual({ typeId: 486, slotType })
    }
    expect(readHullDrop(payload('2454', 'DroneBay'))).toEqual({ typeId: 2454, slotType: 'DroneBay' })
})

test('dropping fitted slots on the hull cannot duplicate or move them', () => {
    for (const index of ['1', '8', '0', '-1']) expect(readHullDrop(payload('486', 'High', index))).toBeNull()
})

test('charges, cargo and external drag payloads do not modify a hull', () => {
    for (const type of ['Charge', 'CargoBay', '', 'text/plain']) expect(readHullDrop(payload('178', type))).toBeNull()
    for (const id of ['', 'not-an-item']) expect(readHullDrop(payload(id, 'High'))).toBeNull()
    expect(readHullDrop(null)).toBeNull()
    expect(readHullDrop(undefined)).toBeNull()
})
