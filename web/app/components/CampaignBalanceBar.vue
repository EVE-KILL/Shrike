<script setup lang="ts">
/**
 * "Who is winning" bar for a campaign — one stacked track split by each
 * side's share of the ISK destroyed. Used on the listing cards, the campaign
 * header and the creator preview so the same glance reads the same way
 * everywhere.
 *
 * A one-sided campaign has no opponent to split against, so it charts its own
 * ledger instead: ISK destroyed vs ISK lost (i.e. efficiency).
 */
import type { EntityAccent } from '#shared/utils/entityAccent'

interface BalanceSide {
    side_index: number
    name: string
    isk_destroyed: number
    isk_lost: number
}

const props = withDefaults(defineProps<{
    sides: BalanceSide[]
    accents?: (EntityAccent | null)[]
    /** Thinner track + no labels, for dense card layouts. */
    compact?: boolean
}>(), { accents: () => [], compact: false })

const oneSided = computed(() => props.sides.length === 1)

const segments = computed(() => {
    const sides = props.sides
    if (!sides.length) return []

    // One side: chart its own ledger so the bar still says something.
    if (oneSided.value) {
        const side = sides[0]!
        const destroyed = Number(side.isk_destroyed) || 0
        const lost = Number(side.isk_lost) || 0
        const total = destroyed + lost
        if (total <= 0) return []
        return [
            { key: 'destroyed', label: 'Destroyed', value: destroyed, pct: (destroyed / total) * 100, color: campaignSideFill(side.side_index, props.accents[side.side_index]) },
            { key: 'lost', label: 'Lost', value: lost, pct: (lost / total) * 100, color: '#ef4444' },
        ]
    }

    const total = sides.reduce((sum, s) => sum + (Number(s.isk_destroyed) || 0), 0)
    if (total <= 0) return []
    return sides.map(s => ({
        key: `side-${s.side_index}`,
        label: s.name,
        value: Number(s.isk_destroyed) || 0,
        pct: ((Number(s.isk_destroyed) || 0) / total) * 100,
        color: campaignSideFill(s.side_index, props.accents[s.side_index]),
    }))
})
</script>

<template>
    <div v-if="segments.length">
        <div class="flex w-full overflow-hidden rounded-full bg-white/[0.06]" :class="compact ? 'h-1.5 gap-px' : 'h-2 gap-0.5'">
            <div v-for="seg in segments" :key="seg.key"
                class="h-full first:rounded-l-full last:rounded-r-full transition-[width] duration-300"
                :style="{ width: `${seg.pct}%`, backgroundColor: seg.color }"
                v-tooltip="`${seg.label} — ${formatIsk(seg.value)} ISK (${seg.pct.toFixed(1)}%)`" />
        </div>
        <div v-if="!compact" class="flex items-center justify-between gap-3 mt-1.5 text-fine">
            <span v-for="(seg, i) in segments" :key="seg.key" class="flex items-center gap-1.5 min-w-0"
                :class="i === segments.length - 1 && segments.length > 1 ? 'text-right' : ''">
                <span class="w-1.5 h-1.5 rounded-sm flex-shrink-0" :style="{ backgroundColor: seg.color }" />
                <span class="text-gray-500 truncate">{{ seg.label }}</span>
                <span class="text-gray-400 tabular-nums flex-shrink-0">{{ seg.pct.toFixed(0) }}%</span>
            </span>
        </div>
    </div>
</template>
