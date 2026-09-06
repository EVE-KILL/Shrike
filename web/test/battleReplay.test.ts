import { expect, test } from 'bun:test'
import { hasReplayPosition, replayPoint, replayBounds, replaySide, replayLossTotals, bigKillClass, AU, type ReplayKill } from '../app/utils/map/replay'
import { battleRegionTotals } from '../app/utils/map/battles'
const kill: ReplayKill = { killmail_id: 1, killmail_time: '2026-04-19T00:30:00Z', solar_system_id: 30000225, solar_system_name: 'H-5GUI', position_x: AU, position_y: 2 * AU, position_z: 3 * AU, victim_corporation_id: 1, victim_alliance_id: null, ship_name: 'Capsule', ship_type_id: 670, total_value: 10000 }
test('zero coordinates are valid, missing/nonfinite positions are not', () => {
    expect(hasReplayPosition({ ...kill, position_x: 0 })).toBe(true)
    expect(hasReplayPosition({ ...kill, position_y: null })).toBe(false)
    expect(hasReplayPosition({ ...kill, position_z: NaN })).toBe(false)
})
test('projects metres to AU without mixing axes', () => {
    expect(replayPoint(kill, 'xz')).toEqual({ x: 1, y: -3 })
    expect(replayPoint(kill, 'xy')).toEqual({ x: 1, y: -2 })
    expect(replayPoint(kill, 'yz')).toEqual({ x: 2, y: -3 })
})
test('fits large offsets and coincident kills with finite nonzero bounds', () => {
    for (const points of [[], [{ x: 24, y: 26 }], [{ x: -30, y: 1 }, { x: 200, y: 900 }]]) {
        const box = replayBounds(points)
        expect(box.width).toBeGreaterThan(0)
        expect(box.height / box.width).toBeCloseTo(0.6)
        for (const point of points) {
            expect(point.x).toBeGreaterThanOrEqual(box.x)
            expect(point.x).toBeLessThanOrEqual(box.x + box.width)
            expect(point.y).toBeGreaterThanOrEqual(box.y)
            expect(point.y).toBeLessThanOrEqual(box.y + box.height)
        }
    }
})
test('unknown victims remain unassigned; explicit corporation membership takes precedence', () => {
    expect(replaySide(kill, [{ corps: [1], alliances: [] }])).toBe(0)
    expect(replaySide(kill, [{ corps: [2], alliances: [] }])).toBe(-1)
    expect(replaySide({ ...kill, victim_alliance_id: 4 }, [{ corps: [1], alliances: [] }, { corps: [], alliances: [4] }])).toBe(0)
})
test('region totals aggregate all systems and skip unknown regions', () => {
    const base = { solar_system_id: 1, solar_system_name: 'A', region_name: 'Region', region_id: 10, battle_count: 51, kill_count: 500, total_isk_destroyed: 1e12 }
    const totals = battleRegionTotals([base, { ...base, solar_system_id: 2, battle_count: 2 }, { ...base, region_id: null }])
    expect(totals.size).toBe(1)
    expect(totals.get(10)).toEqual({ battle_count: 53, total_isk_destroyed: 2e12 })
})

test('running losses respect replay time and all teams, including unassigned victims', () => {
    const teams = [1, 2, 3, 4].map(id => ({ corps: [id], alliances: [] }))
    const kills = [1, 2, 3, 4, 5].map(id => ({ ...kill, killmail_id: id, victim_corporation_id: id }))
    kills.push({ ...kill, killmail_id: 6, killmail_time: '2026-04-19T00:31:00Z' })
    expect(replayLossTotals(kills, teams, Date.parse('2026-04-19T00:29:00Z')).teams.every(team => team.losses === 0)).toBe(true)
    const totals = replayLossTotals(kills, teams, Date.parse(kill.killmail_time))
    expect(totals.teams).toEqual(teams.map(() => ({ losses: 1, isk: 10000 })))
    expect(totals.unassigned).toEqual({ losses: 1, isk: 10000 })
    expect(replayLossTotals(kills, teams, Date.parse('2026-04-19T00:31:00Z')).teams[0]?.losses).toBe(2)
})

test('big kills are classified by ship group, never by value or ship name', () => {
    expect(bigKillClass({ ...kill, ship_group_id: 30 })).toBe('Titan')
    expect(bigKillClass({ ...kill, ship_group_id: 659 })).toBe('Supercarrier')
    expect(bigKillClass({ ...kill, ship_group_id: 547, total_value: 1e12 })).toBeNull()
    expect(bigKillClass({ ...kill, ship_name: 'Titan', ship_group_id: null })).toBeNull()
})
