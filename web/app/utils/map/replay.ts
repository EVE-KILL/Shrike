export interface ReplayKill {
    killmail_id: number
    killmail_time: string
    solar_system_id: number
    solar_system_name: string | null
    position_x: number | null
    position_y: number | null
    position_z: number | null
    victim_character_name?: string | null
    victim_corporation_id: number | null
    victim_alliance_id: number | null
    ship_name: string | null
    ship_type_id: number | null
    ship_group_id?: number | null
    ship_group_name?: string | null
    total_value: number
}
export type ReplayPlane = 'xz' | 'xy' | 'yz'
export const AU = 149597870700
export function hasReplayPosition(kill: ReplayKill): boolean {
    return [kill.position_x, kill.position_y, kill.position_z].every(value => typeof value === 'number' && Number.isFinite(value))
}
export function replayPoint(kill: ReplayKill, plane: ReplayPlane): { x: number; y: number } {
    return { x: (plane === 'yz' ? kill.position_y! : kill.position_x!) / AU, y: -(plane === 'xy' ? kill.position_y! : kill.position_z!) / AU }
}
export function replaySide(kill: ReplayKill, teams: { corps: number[]; alliances: number[] }[]): number {
    const corporation = kill.victim_corporation_id == null ? -1 : teams.findIndex(team => team.corps?.includes(kill.victim_corporation_id!))
    if (corporation >= 0) return corporation
    return kill.victim_alliance_id == null ? -1 : teams.findIndex(team => team.alliances?.includes(kill.victim_alliance_id!))
}
export function replayBounds(points: { x: number; y: number }[]) {
    if (!points.length) return { x: -1, y: -0.6, width: 2, height: 1.2 }
    let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity
    for (const point of points) { minX = Math.min(minX, point.x); maxX = Math.max(maxX, point.x); minY = Math.min(minY, point.y); maxY = Math.max(maxY, point.y) }
    const width = Math.max(maxX - minX, (maxY - minY) / 0.6, 0.00001) * 1.2
    return { x: (minX + maxX - width) / 2, y: (minY + maxY - width * 0.6) / 2, width, height: width * 0.6 }
}

export function replayLossTotals(kills: ReplayKill[], teams: { corps: number[]; alliances: number[] }[], time: number) {
    const totals = teams.map(() => ({ losses: 0, isk: 0 }))
    const unassigned = { losses: 0, isk: 0 }
    for (const kill of kills) {
        const timestamp = Date.parse(kill.killmail_time)
        if (!Number.isFinite(timestamp) || timestamp > time) continue
        const side = replaySide(kill, teams)
        const total = side < 0 ? unassigned : totals[side]!
        total.losses++
        total.isk += kill.total_value ?? 0
    }
    return { teams: totals, unassigned }
}

export function bigKillClass(kill: ReplayKill): 'Titan' | 'Supercarrier' | null {
    if (kill.ship_group_id === 30) return 'Titan'
    if (kill.ship_group_id === 659) return 'Supercarrier'
    return null
}
