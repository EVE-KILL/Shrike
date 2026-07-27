<script setup lang="ts">
/**
 * 24-hour activity for a solar system.
 *
 * This replaces a 24-row table of four numeric columns — readable, but you had
 * to scan 96 figures to notice that a system dies at 18:00 or that its traffic
 * never stops.
 *
 * Drawn as four separate rows rather than one combined chart, because the
 * series are not comparable: Jita runs 1,375–3,024 jumps an hour against 3–39
 * ship kills. On a shared axis the kills — the reason anyone is on a killboard
 * — would be a flat line at the bottom. Each row is therefore scaled to its
 * own peak, which is stated at the right so a tall bar is never mistaken for a
 * big number. Hovering any hour highlights that hour across all four.
 */
const props = defineProps<{
    history: { timestamp: string; ship_jumps?: number; ship_kills?: number; npc_kills?: number; pod_kills?: number }[]
    summary?: { ship_jumps?: number; ship_kills?: number; npc_kills?: number; pod_kills?: number } | null
}>()

const HOUR_MS = 3_600_000

/**
 * A fixed slot per hour for the last 24, oldest first.
 *
 * The payload arrives newest-first and is not guaranteed complete — QC-YX6
 * currently returns 23 records with 13:00 absent. A table could simply omit
 * the row, but a chart cannot: packing the remaining bars together would slide
 * every later hour one position left and quietly redraw the axis. Missing
 * hours therefore keep their slot and render empty, which also distinguishes
 * "no data collected" from "nothing happened".
 */
const hours = computed(() => {
    const byHour = new Map<number, typeof props.history[number]>()
    let latest = 0
    for (const rec of props.history) {
        const t = new Date(rec.timestamp).getTime()
        if (Number.isNaN(t)) continue
        const floored = Math.floor(t / HOUR_MS) * HOUR_MS
        byHour.set(floored, rec)
        if (floored > latest) latest = floored
    }
    if (!latest) return []

    return Array.from({ length: 24 }, (_, i) => {
        const ts = latest - (23 - i) * HOUR_MS
        return { ts, rec: byHour.get(ts) ?? null }
    })
})

const SERIES = [
    { key: 'ship_jumps', label: 'Jumps', bar: 'bg-cyan-400/70', text: 'text-cyan-400' },
    { key: 'ship_kills', label: 'Ship kills', bar: 'bg-red-400/70', text: 'text-red-400' },
    { key: 'npc_kills', label: 'NPC kills', bar: 'bg-purple-400/70', text: 'text-purple-400' },
    { key: 'pod_kills', label: 'Pod kills', bar: 'bg-orange-400/70', text: 'text-orange-400' },
] as const

const rows = computed(() => SERIES.map((s) => {
    // null = hour not collected, distinct from a collected zero.
    const values = hours.value.map(h => h.rec ? ((h.rec[s.key] as number) ?? 0) : null)
    const present = values.filter((v): v is number => v !== null)
    return {
        ...s,
        values,
        peak: Math.max(...present, 0),
        total: present.reduce((a, b) => a + b, 0),
    }
}).filter(r => r.total > 0))

/** EVE time — the payload is UTC and every other clock on the site is too. */
const hourLabel = (ts: number) => String(new Date(ts).getUTCHours()).padStart(2, '0')

const missingHours = computed(() => hours.value.filter(h => !h.rec).length)

const hovered = ref<number | null>(null)

/**
 * A zero hour still gets a hairline so the axis reads as continuous rather
 * than gapped — an empty hour and a missing hour look identical otherwise.
 */
const barHeight = (value: number, peak: number) =>
    peak <= 0 ? '2px' : `${Math.max(2, (value / peak) * 100)}%`

const fmt = (n: number) => n.toLocaleString('en-US')
</script>

<template>
    <div class="glass-panel p-3.5">
        <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
            <Icon name="lucide:activity" class="w-4 h-4 text-cyan-500" />
            <span class="text-xs font-semibold text-gray-300">Hourly Activity (24h)</span>
            <span class="ml-auto text-fine text-gray-600">EVE time</span>
        </div>

        <div v-if="!rows.length" class="py-6 text-center text-fine text-gray-600">
            No recorded activity in the last 24 hours
        </div>

        <template v-else>
            <div v-for="row in rows" :key="row.key" class="mb-2.5 last:mb-0">
                <div class="flex items-baseline gap-2 mb-1">
                    <span class="text-fine uppercase tracking-wider" :class="row.text">{{ row.label }}</span>
                    <span class="text-fine text-gray-600 tabular-nums">{{ fmt(row.total) }} total</span>
                    <span class="ml-auto text-fine text-gray-600 tabular-nums">peak {{ fmt(row.peak) }}</span>
                </div>
                <div class="flex items-end gap-px h-10">
                    <div v-for="(v, i) in row.values" :key="i"
                        class="flex-1 min-w-0 h-full flex items-end cursor-default"
                        @mouseenter="hovered = i" @mouseleave="hovered = null">
                        <!-- Hatched stub for an uncollected hour, so a hole in the
                             data never passes for a quiet hour -->
                        <div v-if="v === null" class="w-full h-0.5 bg-white/[0.06] rounded-t-sm" />
                        <div v-else class="w-full rounded-t-sm transition-[height,opacity] duration-150"
                            :class="[row.bar, hovered !== null && hovered !== i ? 'opacity-30' : '']"
                            :style="{ height: barHeight(v, row.peak) }" />
                    </div>
                </div>
            </div>

            <!-- Axis: every third hour, so the labels never collide -->
            <div class="flex gap-px mt-1">
                <div v-for="(h, i) in hours" :key="h.ts" class="flex-1 min-w-0 text-center">
                    <span v-if="i % 3 === 0" class="text-fine text-gray-600 tabular-nums">{{ hourLabel(h.ts) }}</span>
                </div>
            </div>

            <!-- Readout for the hovered hour, so exact figures survive the move to a chart -->
            <div class="mt-2 pt-2 border-t border-white/[0.04] flex flex-wrap items-center gap-x-4 gap-y-1 text-fine min-h-[1.25rem]">
                <template v-if="hovered !== null && hours[hovered]">
                    <span class="text-gray-300 tabular-nums font-medium">{{ hourLabel(hours[hovered]!.ts) }}:00</span>
                    <template v-if="hours[hovered]!.rec">
                        <span v-for="row in rows" :key="row.key" class="text-gray-500">
                            {{ row.label }} <span class="tabular-nums" :class="row.text">{{ fmt(row.values[hovered] as number) }}</span>
                        </span>
                    </template>
                    <span v-else class="text-gray-600 italic">not recorded</span>
                </template>
                <template v-else>
                    <span class="text-gray-600">Hover an hour for exact figures</span>
                    <span v-if="missingHours" class="ml-auto text-gray-700">{{ missingHours }}h not recorded</span>
                </template>
            </div>
        </template>
    </div>
</template>
