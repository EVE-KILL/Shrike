import { expect, test } from 'bun:test'
import { battleLossBuckets, filterBattleLosses } from '../app/utils/battleExplorer'
import type { ReplayKill } from '../app/utils/map/replay'
const base: ReplayKill = { killmail_id: 1, killmail_time: '2026-04-06T16:00:00Z', solar_system_id: 1, solar_system_name: 'A', position_x: null, position_y: null, position_z: null, victim_corporation_id: 1, victim_alliance_id: 10, ship_type_id: 1, ship_name: 'Titan', ship_group_id: 30, total_value: 1e11 }
const teams = [{ corps: [1], alliances: [10] }, { corps: [2], alliances: [20] }]
const start = Date.parse(base.killmail_time), end = start + 3 * 60000
const kills = [base, { ...base, killmail_id: 2, killmail_time: new Date(start + 60000).toISOString(), victim_corporation_id: 3, victim_alliance_id: null, ship_group_id: 29, total_value: 10000 }, { ...base, killmail_id: 3, killmail_time: new Date(end).toISOString(), victim_corporation_id: 2, victim_alliance_id: 20 }]
test('timeline retains quiet periods, unassigned losses and final boundary kills', () => {
    const bins = battleLossBuckets(kills, teams, start, end)
    expect(bins).toHaveLength(3)
    expect(bins[1]?.sides[2]?.count).toBe(1)
    expect(bins[1]?.sides[1]?.count).toBe(0)
    expect(bins[2]?.big).toBe(1)
    const filtered = filterBattleLosses(kills, teams, {side:null,group:null,minIsk:0,from:bins[2]!.start,to:bins[2]!.end})
    expect(filtered.map(kill=>kill.killmail_id)).toEqual([3])
    expect(battleLossBuckets([base],teams,start,end)[1]?.count).toBe(0)
})
test('shared filters compose and adjacent time windows do not duplicate kills', () => {
    const filter = {side:null,group:null,minIsk:0,from:start,to:start+60000}
    expect(filterBattleLosses(kills,teams,filter).map(kill=>kill.killmail_id)).toEqual([1])
    expect(filterBattleLosses(kills,teams,{...filter,from:null,to:null,side:-1}).map(kill=>kill.killmail_id)).toEqual([2])
    expect(filterBattleLosses(kills,teams,{...filter,from:null,to:null,side:1,group:30,minIsk:1e10}).map(kill=>kill.killmail_id)).toEqual([3])
})
