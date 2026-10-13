import { describe, expect, test } from 'vitest'
import { mergeEsiLogRows } from '../src/utils/esiLog'

describe('ESI live batches', () => {
    test('deduplicates both the batch and current rows without mutating the response', () => {
        const data = { rows: [{ id: 4 }, { id: 3 }, { id: 2 }], total: 15 }
        const merged = mergeEsiLogRows(data, [{ id: 6 }, { id: 6 }, { id: 5 }, { id: 4 }], 4)
        expect(merged).toEqual({ rows: [{ id: 6 }, { id: 5 }, { id: 4 }, { id: 3 }], total: 17 })
        expect(data).toEqual({ rows: [{ id: 4 }, { id: 3 }, { id: 2 }], total: 15 })
        expect(merged.rows).not.toBe(data.rows)
    })

    test('caps visible rows while preserving the full count', () => {
        const data = { rows: [{ id: 1 }], total: 1 }
        const merged = mergeEsiLogRows(data, Array.from({ length: 60 }, (_, index) => ({ id: 61 - index })))
        expect(merged.rows).toHaveLength(50)
        expect(merged.total).toBe(61)
        expect(merged.rows[0]?.id).toBe(61)
    })
})
