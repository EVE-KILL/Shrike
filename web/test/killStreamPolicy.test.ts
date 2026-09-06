import { describe, expect, test } from 'bun:test'
import { createKillFreshnessGate, matchesDomainKeys, streamPoolKey } from '../app/utils/killStreamPolicy'

describe('live kill consumer policies', () => {
    test('background watchers never share the visibility-paused pool', () => {
        expect(streamPoolKey(['all'], 'https://example.test', true)).not.toBe(streamPoolKey(['all'], 'https://example.test', false))
        expect(streamPoolKey(['b', 'a'], 'x', true)).toBe(streamPoolKey(['a', 'b', 'a'], 'x', true))
    })
    test('domain participation uses every attacker, not just the final blow', () => {
        const entities = { characterIds: [], corporationIds: [], allianceIds: [123] }
        expect(matchesDomainKeys(['all', 'attacker.123', 'attacker.456'], entities)).toBe(true)
        expect(matchesDomainKeys(['victim.123'], entities)).toBe(true)
        expect(matchesDomainKeys(['attacker.456'], entities)).toBe(false)
        expect(matchesDomainKeys([], entities, { final_blow_alliance_id: 123 })).toBe(true)
        expect(matchesDomainKeys(['attacker.456'], entities, { final_blow_alliance_id: 123 })).toBe(false)
    })
    test('alarms reject old, malformed, future and replayed kills', () => {
        const now = Date.parse('2026-09-06T12:00:00Z')
        const accept = createKillFreshnessGate()
        expect(accept(1, '2026-09-06T11:55:00Z', now)).toBe(true)
        expect(accept(1, '2026-09-06T11:55:00Z', now)).toBe(false)
        expect(accept(2, '2026-09-05T11:55:00Z', now)).toBe(false)
        expect(accept(3, 'invalid', now)).toBe(false)
        expect(accept(4, '2026-09-06T13:00:00Z', now)).toBe(false)
        expect(accept(5, '2026-09-06T11:59:00Z', now)).toBe(true)
    })
})
