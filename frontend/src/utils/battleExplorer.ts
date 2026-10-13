import { replaySide, bigKillClass, type ReplayKill } from './map/replay'
export type BattleSide = { corps: number[]; alliances: number[] }
export interface BattleLossFilter { side: number | null; group: number | null; minIsk: number; from: number | null; to: number | null }
export function filterBattleLosses(kills: ReplayKill[], teams: BattleSide[], filter: BattleLossFilter) {
    return kills.filter(kill => {
        const time = Date.parse(kill.killmail_time)
        return Number.isFinite(time) && (filter.side == null || replaySide(kill, teams) === filter.side)
            && (filter.group == null || kill.ship_group_id === filter.group) && kill.total_value >= filter.minIsk
            && (filter.from == null || time >= filter.from) && (filter.to == null || time < filter.to)
    })
}
export function battleLossBuckets(kills: ReplayKill[], teams: BattleSide[], start: number, end: number) {
    const minutes = [1, 5, 15, 30, 60, 180].find(step => (end - start) / 60000 / step <= 60) ?? 180
    const step = minutes * 60000
    const bins = Array.from({ length: Math.max(1, Math.ceil((end - start) / step)) }, (_, index) => ({
        start: start + index * step, end: index === Math.max(1, Math.ceil((end - start) / step)) - 1 ? end + 1 : start + (index + 1) * step,
        count: 0, isk: 0, big: 0, sides: Array.from({ length: teams.length + 1 }, () => ({ count: 0, isk: 0 })),
    }))
    for (const kill of kills) {
        const time = Date.parse(kill.killmail_time)
        if (!Number.isFinite(time) || time < start || time > end) continue
        const bin = bins[Math.min(bins.length - 1, Math.floor((time - start) / step))]!
        const side = replaySide(kill, teams)
        const total = bin.sides[side < 0 ? teams.length : side]!
        bin.count++; bin.isk += kill.total_value; total.count++; total.isk += kill.total_value
        if (bigKillClass(kill)) bin.big++
    }
    return bins
}
