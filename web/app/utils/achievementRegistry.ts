/**
 * Achievement definitions — ported from Thessia's server/helpers/Achievements.ts.
 * This is the frontend's source of truth for achievement metadata (name, description,
 * type, rarity, category). Keep in sync with backend achievement calculations.
 * See /ACHIEVEMENTS.md for details.
 */

export interface AchievementDef {
    id: string
    name: string
    description: string
    type: 'pvp' | 'pve' | 'exploration' | 'industry' | 'special'
    rarity: 'common' | 'uncommon' | 'rare' | 'epic' | 'legendary'
    category: string
    pointsModifier: 'positive' | 'negative'
}

// Core achievements
const coreAchievements: AchievementDef[] = [
    // General kill milestones
    { id: 'babys_first_kill', name: "Baby's First Kill", description: 'Get your very first kill on an enemy player.', type: 'pvp', rarity: 'common', category: 'Combat', pointsModifier: 'positive' },
    { id: 'rookie_killer', name: 'Rookie Killer', description: 'Achieve 10 kills on enemy players.', type: 'pvp', rarity: 'common', category: 'Combat', pointsModifier: 'positive' },
    { id: 'experienced_killer', name: 'Experienced Killer', description: 'Achieve 100 kills on enemy players.', type: 'pvp', rarity: 'uncommon', category: 'Combat', pointsModifier: 'positive' },
    { id: 'veteran_killer', name: 'Veteran Killer', description: 'Achieve 1,000 kills on enemy players.', type: 'pvp', rarity: 'rare', category: 'Combat', pointsModifier: 'positive' },
    // Solo
    { id: 'solo_first_blood', name: 'Solo First Blood', description: 'Get your first solo kill on an enemy player.', type: 'pvp', rarity: 'common', category: 'Combat', pointsModifier: 'positive' },
    // High value
    { id: 'bling_hunter', name: 'Bling Hunter', description: 'Kill someone worth at least 1 billion ISK.', type: 'pvp', rarity: 'uncommon', category: 'High Value Targets', pointsModifier: 'positive' },
    // Locations
    { id: 'highsec_hunter', name: 'Highsec Hunter', description: 'Kill someone in high security space.', type: 'pvp', rarity: 'common', category: 'Locations', pointsModifier: 'positive' },
    { id: 'lowsec_prowler', name: 'Lowsec Prowler', description: 'Kill someone in low security space.', type: 'pvp', rarity: 'common', category: 'Locations', pointsModifier: 'positive' },
    { id: 'nullsec_warrior', name: 'Nullsec Warrior', description: 'Kill someone in null security space.', type: 'pvp', rarity: 'common', category: 'Locations', pointsModifier: 'positive' },
]

// Ship category definitions
const SHIP_CATEGORIES: Record<string, { name: string, thresholds: number[] }> = {
    frigates: { name: 'Frigates', thresholds: [10, 50, 100, 500, 1000] },
    destroyers: { name: 'Destroyers', thresholds: [10, 25, 50, 250, 500] },
    cruisers: { name: 'Cruisers', thresholds: [10, 25, 50, 200, 400] },
    battleships: { name: 'Battleships', thresholds: [5, 10, 25, 100, 200] },
    capitals: { name: 'Capital Ships', thresholds: [1, 5, 10, 25, 50] },
    industrials: { name: 'Industrial Ships', thresholds: [10, 50, 100, 500, 1000] },
}

const RARITY_BY_INDEX: AchievementDef['rarity'][] = ['common', 'uncommon', 'rare', 'epic', 'legendary']

// Generate ship kill/loss achievements
function generateShipAchievements(): AchievementDef[] {
    const out: AchievementDef[] = []
    for (const [key, cat] of Object.entries(SHIP_CATEGORIES)) {
        // Kill tiers
        for (let i = 0; i < cat.thresholds.length; i++) {
            const t = cat.thresholds[i]!
            out.push({
                id: `${key}_killer_${t}`,
                name: `${cat.name} Killer${t > 1 ? ` (${t.toLocaleString('en-US')})` : ''}`,
                description: `Destroy ${t.toLocaleString('en-US')} ${cat.name.toLowerCase()}.`,
                type: 'pvp',
                rarity: RARITY_BY_INDEX[i] ?? 'legendary',
                category: 'Ship Classes',
                pointsModifier: 'positive',
            })
        }
        // Loss tiers (first 2 only)
        for (let i = 0; i < Math.min(2, cat.thresholds.length); i++) {
            const t = cat.thresholds[i]!
            out.push({
                id: `${key}_loser_${t}`,
                name: `${cat.name} Sacrifice${t > 1 ? ` (${t.toLocaleString('en-US')})` : ''}`,
                description: `Lose ${t.toLocaleString('en-US')} ${cat.name.toLowerCase()}.`,
                type: 'pvp',
                rarity: i === 0 ? 'common' : 'uncommon',
                category: 'Ship Classes',
                pointsModifier: 'negative',
            })
        }
    }
    return out
}

// Build the lookup map
const allAchievements = [...coreAchievements, ...generateShipAchievements()]
const registry = new Map<string, AchievementDef>(allAchievements.map(a => [a.id, a]))

/**
 * Look up achievement metadata by ID.
 * Returns a fallback with parsed name if the ID isn't in the registry.
 */
export function getAchievementDef(id: string): AchievementDef {
    const def = registry.get(id)
    if (def) return def

    // Fallback: parse the ID into a display name
    const parts = id.split('_')
    const isLoss = id.includes('loser') || id.includes('sacrifice')
    const name = parts.map(p => p.charAt(0).toUpperCase() + p.slice(1)).join(' ')
    return {
        id,
        name,
        description: '',
        type: 'pvp',
        rarity: 'common',
        category: 'Other',
        pointsModifier: isLoss ? 'negative' : 'positive',
    }
}

export { allAchievements }
