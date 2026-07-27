import type { KilllistResponse } from '#shared/api'

/**
 * The shared REST/WebSocket kill-list row.
 *
 * The source shape is emitted by the Go Huma contract for /api/killlist. The
 * relay publishes the same hydrated object, so UI code has one generated wire
 * type for initial page data and live inserts.
 */
export type KilllistRow = KilllistResponse['kills'][number]

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
