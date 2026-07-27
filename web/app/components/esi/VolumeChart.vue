<script setup lang="ts">
/**
 * 24-hour ESI request volume, as a stacked bar per hour: errors on top of
 * successes, both scaled against the busiest hour in the window.
 *
 * Lived as three near-identical copies — admin Overview, admin ESI and the
 * user-facing settings ESI tab — which had already drifted (different tooltip
 * wording, one had lost its `min-w-0`). Each copy also recomputed
 * `Math.max(...volumeByHour.map(x => x.total))` inline twice per bar, so 48
 * passes over the array on every render.
 *
 * All three also laid the rows out with `flex-1`, which spreads however many
 * rows the query returned evenly across the full width. The endpoint only
 * returns hours that actually logged something, so a quiet period did not read
 * as quiet — two rows fifteen hours apart rendered as two half-width blocks
 * under an axis labelled 18:00 → 09:00, looking like saturated traffic. The
 * timeline is now a fixed 24 slots and an hour with no requests is drawn empty,
 * so gaps look like gaps.
 */
interface VolumeHour {
    hour: string
    total: number
    errors: number
    new_items?: number
}

const props = withDefaults(defineProps<{
    hours: VolumeHour[]
    title?: string
    /** Mention new-item counts in the per-bar tooltip. */
    showNewItems?: boolean
}>(), {
    title: 'Request Volume (24h)',
    showNewItems: false,
})

const CHART_HEIGHT = 96
const SLOTS = 24
const HOUR_MS = 3_600_000

/**
 * Anchored on the newest hour in the payload rather than on the wall clock, so
 * the same props always draw the same chart — no server/client divergence, and
 * no dependency on how far behind the log happens to be.
 */
/**
 * Postgres hands these back as '2026-07-25 09:00:00+00'. V8 parses that via its
 * lenient non-ISO path, but Safari does not, so it needs normalising to real
 * ISO — which means both the 'T' *and* padding the offset, since '+00' alone is
 * not a valid ISO offset and makes the whole parse fail. Falls back to the raw
 * string so any other shape still goes through V8's lenient path.
 */
const toMs = (value: string): number => {
    const iso = value.replace(' ', 'T').replace(/([+-]\d{2})$/, '$1:00')
    const t = new Date(iso).getTime()
    return Number.isNaN(t) ? new Date(value).getTime() : t
}

const slots = computed(() => {
    if (!props.hours.length) return []

    const byHour = new Map<number, VolumeHour>()
    for (const h of props.hours) {
        const t = toMs(h.hour)
        if (!Number.isNaN(t)) byHour.set(Math.floor(t / HOUR_MS) * HOUR_MS, h)
    }
    if (!byHour.size) return []

    const latest = Math.max(...byHour.keys())
    const peak = Math.max(1, ...props.hours.map(h => h.total ?? 0))

    return Array.from({ length: SLOTS }, (_, i) => {
        const at = latest - (SLOTS - 1 - i) * HOUR_MS
        const rec = byHour.get(at)
        const errors = rec?.errors ?? 0
        const ok = Math.max(0, (rec?.total ?? 0) - errors)
        return {
            at,
            empty: !rec,
            errors,
            // A 2px floor keeps a quiet-but-nonzero hour visible instead of vanishing.
            errorHeight: Math.max(2, (errors / peak) * CHART_HEIGHT),
            okHeight: Math.max(2, (ok / peak) * CHART_HEIGHT),
            tooltip: rec
                ? (props.showNewItems
                    ? `${fmtHour(rec.hour)}: ${rec.total} requests, ${errors} errors, ${rec.new_items ?? 0} new items`
                    : `${fmtHour(rec.hour)}: ${rec.total} requests, ${errors} errors`)
                : `${String(new Date(at).getUTCHours()).padStart(2, '0')}:00 — no requests logged`,
        }
    })
})

const axisLabel = (at: number | undefined) =>
    at == null ? '—' : `${String(new Date(at).getUTCHours()).padStart(2, '0')}:00`
</script>

<template>
    <div v-if="slots.length" class="glass-panel p-5">
        <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">{{ title }}</h2>
        <div class="flex items-end gap-px" :style="{ height: `${CHART_HEIGHT}px` }">
            <div
                v-for="s in slots" :key="s.at"
                v-tooltip="s.tooltip"
                class="flex-1 flex flex-col justify-end gap-px min-w-0 h-full"
            >
                <template v-if="!s.empty">
                    <div
                        v-if="s.errors > 0"
                        class="bg-red-500/60 rounded-t-sm"
                        :style="{ height: `${s.errorHeight}px` }"
                    ></div>
                    <div class="bg-blue-500/40 rounded-t-sm" :style="{ height: `${s.okHeight}px` }"></div>
                </template>
                <!-- An hour that logged nothing: a hairline on the baseline, so
                     the gap is visibly a gap rather than a missing column. -->
                <div v-else class="bg-white/[0.06] rounded-t-sm" style="height: 1px"></div>
            </div>
        </div>
        <div class="flex justify-between text-fine text-gray-600 mt-1">
            <span>{{ axisLabel(slots[0]?.at) }}</span>
            <span class="flex items-center gap-3">
                <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-sm bg-blue-500/40"></span> OK</span>
                <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-sm bg-red-500/60"></span> Errors</span>
            </span>
            <span>{{ axisLabel(slots[slots.length - 1]?.at) }}</span>
        </div>
    </div>
</template>
