import type { KilllistResponse } from '#shared/api'

/**
 * The shared REST/WebSocket kill-list row.
 *
 * The source shape is emitted by the Go Huma contract for /api/killlist. The
 * relay publishes the same hydrated object, so UI code has one generated wire
 * type for initial page data and live inserts.
 */
export type KilllistRow = KilllistResponse['kills'][number]

// Source KillList's best-effort relay predicate, shared with Go's private feeds.
export function matchesStreamKillType(kill: { is_solo?: boolean; is_npc?: boolean; total_value?: number | null }, type?: string): boolean {
    switch (type) {
        case 'solo': return kill.is_solo === true
        case 'npc': return kill.is_npc === true
        case 'big': return (kill.total_value ?? 0) >= 1_000_000_000
        case '5b': return (kill.total_value ?? 0) >= 5_000_000_000
        case '10b': return (kill.total_value ?? 0) >= 10_000_000_000
        default: return true
    }
}

export interface KilllistLossEntities {
    characterIds?: number[]
    corporationIds?: number[]
    allianceIds?: number[]
    factionIds?: number[]
}

type KilllistVictimIdentity = Pick<
    KilllistRow,
    'victim_character_id' | 'victim_corporation_id' | 'victim_alliance_id' | 'victim_faction_id'
>

export function isKilllistLoss(
    kill: KilllistVictimIdentity,
    entities: KilllistLossEntities | null | undefined,
): boolean {
    if (!entities) return false
    if (kill.victim_character_id && entities.characterIds?.includes(kill.victim_character_id)) return true
    if (kill.victim_corporation_id && entities.corporationIds?.includes(kill.victim_corporation_id)) return true
    if (kill.victim_alliance_id && entities.allianceIds?.includes(kill.victim_alliance_id)) return true
    if (kill.victim_faction_id && entities.factionIds?.includes(kill.victim_faction_id)) return true
    return false
}

export const META_GROUP = {
    tech1: 1,
    tech2: 2,
    faction: 4,
    tech3: 14,
} as const

export function matchesMetaGroup(
    metaGroupId: number | null | undefined,
    level: string,
): boolean {
    switch (level) {
        case 't1':
            return (metaGroupId ?? META_GROUP.tech1) === META_GROUP.tech1
        case 't2':
            return metaGroupId === META_GROUP.tech2
        case 't3':
            return metaGroupId === META_GROUP.tech3
        case 'faction':
            return metaGroupId === META_GROUP.faction
        default:
            return true
    }
}
