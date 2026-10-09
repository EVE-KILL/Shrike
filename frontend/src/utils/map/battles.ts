export interface BattleMapSystem {
    solar_system_id: number
    solar_system_name: string | null
    region_id: number | null
    region_name: string | null
    battle_count: number
    kill_count: number
    total_isk_destroyed: number
}

export function battleRegionTotals(systems: BattleMapSystem[]) {
    const regions = new Map<number, { battle_count: number; total_isk_destroyed: number }>()
    for (const system of systems) {
        if (system.region_id == null) continue
        const total = regions.get(system.region_id) ?? { battle_count: 0, total_isk_destroyed: 0 }
        total.battle_count += system.battle_count
        total.total_isk_destroyed += system.total_isk_destroyed
        regions.set(system.region_id, total)
    }
    return regions
}
