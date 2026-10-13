import { describe, expect, test } from 'vitest'
import { riverCanCancel, riverJSON, riverPending, riverStateClass, riverStates, type RiverQueue } from '../src/utils/adminRiver'

describe('River operations display contract', () => {
  test('only unfinished states contribute to pending work', () => {
    const queue = { depth: { available: 1, running: 2, scheduled: 3, retryable: 4, discarded: 100, completed: 200, cancelled: 300 } } as RiverQueue
    expect(riverPending(queue)).toBe(10)
    expect(riverCanCancel('running')).toBe(true)
    expect(riverCanCancel('completed')).toBe(false)
    expect(riverCanCancel('discarded')).toBe(false)
    expect(new Set(riverStates).size).toBe(7)
    expect(riverStates.every(state => riverStateClass(state) !== 'text-gray-400')).toBe(true)
    expect(riverStateClass('future-state')).toBe('text-gray-400')
  })
  test('inspector retains nested input and escaped-looking error text literally', () => {
    const input = { pilots: [4242, 4343], description: '<script>plain input</script>' }
    expect(JSON.parse(riverJSON(input))).toEqual(input)
    expect(riverJSON(null)).toBe('null')
    expect(riverJSON(undefined)).toBe('')
  })
})
