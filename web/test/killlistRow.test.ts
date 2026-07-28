import { describe, expect, test } from 'bun:test'
import { isKilllistLoss } from '../shared/utils/killlistRow'

describe('killlist loss identity', () => {
    test('matches faction victims without treating faction kills as losses', () => {
        const faction = { factionIds: [500003] }

        expect(isKilllistLoss({
            victim_character_id: 90000001,
            victim_corporation_id: 98000001,
            victim_alliance_id: 99000001,
            victim_faction_id: 500003,
        }, faction)).toBe(true)

        expect(isKilllistLoss({
            victim_character_id: 90000002,
            victim_corporation_id: 98000002,
            victim_alliance_id: 99000002,
            victim_faction_id: 500002,
        }, faction)).toBe(false)
    })
})
