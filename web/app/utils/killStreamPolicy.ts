export function streamPoolKey(topics: string[], wsUrl: string, background: boolean): string {
    return JSON.stringify([wsUrl, [...new Set(topics)].sort(), background])
}

export function matchesDomainKeys(keys: string[], entities: {
    characterIds: number[]; corporationIds: number[]; allianceIds: number[]
}, legacyKill?: {
    victim_character_id?: number | null; victim_corporation_id?: number | null; victim_alliance_id?: number | null
    final_blow_character_id?: number | null; final_blow_corporation_id?: number | null; final_blow_alliance_id?: number | null
}): boolean {
    // During a rolling deploy an older relay may omit the envelope keys.
    // Preserve its previous best-effort behavior until that socket reconnects.
    if (keys.length === 0 && legacyKill) {
        return entities.characterIds.some(id => id === legacyKill.victim_character_id || id === legacyKill.final_blow_character_id)
            || entities.corporationIds.some(id => id === legacyKill.victim_corporation_id || id === legacyKill.final_blow_corporation_id)
            || entities.allianceIds.some(id => id === legacyKill.victim_alliance_id || id === legacyKill.final_blow_alliance_id)
    }
    const routing = new Set(keys)
    return [...entities.characterIds, ...entities.corporationIds, ...entities.allianceIds]
        .some(id => routing.has(`victim.${id}`) || routing.has(`attacker.${id}`))
}

// Allow normal posting delays, but do not turn historical backfills into alarms.
export function createKillFreshnessGate(maxAgeMs = 15 * 60 * 1000) {
    const seen = new Map<number, number>()
    return (id: number, timestamp: string, now = Date.now()): boolean => {
        for (const [key, expires] of seen) if (expires <= now) seen.delete(key)
        const occurred = Date.parse(timestamp)
        if (!Number.isFinite(occurred) || occurred > now + 60_000 || now - occurred > maxAgeMs || seen.has(id)) return false
        seen.set(id, occurred + maxAgeMs + 60_000)
        return true
    }
}
