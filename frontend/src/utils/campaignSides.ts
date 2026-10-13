/**
 * Shared side palette for campaign UI — listing cards, detail scoreboard and
 * the creator preview all read from here so a side keeps the same identity
 * wherever it shows up.
 *
 * The fixed fills are the validated dark-surface set (red-500 / blue-500 /
 * emerald-600 / orange-600 — CVD-safe for adjacent pairs). Corp-palette
 * accents from campaignSideAccents() override them per side, but only when
 * every side resolves to a distinct hue (all-or-nothing, see entityAccent).
 */

export const CAMPAIGN_SIDE_TEXT = ['text-red-400', 'text-blue-400', 'text-emerald-400', 'text-orange-400']
export const CAMPAIGN_SIDE_BORDER = ['border-red-500/20', 'border-blue-500/20', 'border-emerald-500/20', 'border-orange-500/20']
export const CAMPAIGN_SIDE_FILL = ['#ef4444', '#3b82f6', '#059669', '#ea580c']

/** Neutral fill for sides beyond the palette, and for area campaigns. */
export const CAMPAIGN_NEUTRAL_FILL = '#6b7280'

/** Hex fill for a side — corp accent when one resolved, fixed palette otherwise. */
export function campaignSideFill(index: number, accent?: { accent: string } | null): string {
    return accent?.accent ?? CAMPAIGN_SIDE_FILL[index] ?? CAMPAIGN_NEUTRAL_FILL
}

/** ISK-destroyed share of the total, as a percentage. */
export function campaignEfficiency(side: { isk_destroyed: number; isk_lost: number }): number {
    const total = Number(side.isk_destroyed) + Number(side.isk_lost)
    return total > 0 ? (Number(side.isk_destroyed) / total) * 100 : 0
}
