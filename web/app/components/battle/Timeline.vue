<script setup lang="ts">
/**
 * Battle timeline — losses over time, split by side.
 *
 * Two things the previous version was missing: the shape of the fight (which
 * minutes actually mattered) and any way to navigate a long one. A battle
 * spanning hours produced hundreds of identical 5-minute panels and a page
 * tens of thousands of pixels tall, with no map and no labels saying which
 * column belonged to which side.
 */
const props = defineProps<{
    battleId: number | null
    solarSystemId: number
    startTime: string
    endTime: string
    teams: any[]
    teamEntities: { corps: number[]; alliances: number[] }[]
    timelineEndpoint?: string
}>()

const endpoint = props.timelineEndpoint || `/api/battle/${props.battleId}/timeline`
const { data, pending } = await useApiFetch<any>(endpoint, {
    lazy: true,
    getCachedData: cachedPayload,
})

const kills = computed(() => data.value?.kills || [])

const teamName = (i: number) => props.teams?.[i]?.name || `Team ${i + 1}`

// ── Bucket size ─────────────────────────────────────────────────────────────
// Fixed 5-minute buckets are right for a skirmish and absurd for a 12-hour
// grind. Scale the interval so any battle lands in a readable number of rows.
const BUCKET_STEPS_MIN = [1, 5, 15, 30, 60, 180]
const TARGET_BUCKETS = 60

const durationMin = computed(() => {
    const start = new Date(props.startTime).getTime()
    const end = new Date(props.endTime).getTime()
    if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return 0
    return (end - start) / 60_000
})

const bucketMinutes = computed(() => {
    const span = durationMin.value
    if (span <= 0) return 5
    return BUCKET_STEPS_MIN.find(step => span / step <= TARGET_BUCKETS) ?? 180
})

const bucketLabel = computed(() => {
    const m = bucketMinutes.value
    return m >= 60 ? `${m / 60}h` : `${m}m`
})

// ── Side classification ─────────────────────────────────────────────────────
const team0Corps = computed(() => new Set(props.teamEntities?.[0]?.corps || []))
const team0Alliances = computed(() => new Set(props.teamEntities?.[0]?.alliances || []))
const team1Corps = computed(() => new Set(props.teamEntities?.[1]?.corps || []))
const team1Alliances = computed(() => new Set(props.teamEntities?.[1]?.alliances || []))

function classifyVictimSide(kill: any): 0 | 1 {
    if (kill.victim_alliance_id && team0Alliances.value.has(kill.victim_alliance_id)) return 0
    if (kill.victim_alliance_id && team1Alliances.value.has(kill.victim_alliance_id)) return 1
    if (kill.victim_corporation_id && team0Corps.value.has(kill.victim_corporation_id)) return 0
    if (kill.victim_corporation_id && team1Corps.value.has(kill.victim_corporation_id)) return 1
    return 1 // default to team 1
}

interface Bucket {
    time: string
    label: string
    team0: any[]
    team1: any[]
    isk0: number
    isk1: number
    total: number
    totalIsk: number
}

const buckets = computed<Bucket[]>(() => {
    if (kills.value.length === 0) return []
    const stepMs = bucketMinutes.value * 60_000
    const groups: Bucket[] = []
    const byKey = new Map<number, Bucket>()

    for (const k of kills.value) {
        const t = new Date(k.killmail_time).getTime()
        if (!Number.isFinite(t)) continue
        const key = Math.floor(t / stepMs) * stepMs

        let bucket = byKey.get(key)
        if (!bucket) {
            bucket = {
                time: new Date(key).toISOString(),
                label: formatEveTime(new Date(key)),
                team0: [], team1: [], isk0: 0, isk1: 0, total: 0, totalIsk: 0,
            }
            byKey.set(key, bucket)
            groups.push(bucket)
        }

        const value = Number(k.total_value) || 0
        if (classifyVictimSide(k) === 0) { bucket.team0.push(k); bucket.isk0 += value }
        else { bucket.team1.push(k); bucket.isk1 += value }
        bucket.total++
        bucket.totalIsk += value
    }

    groups.sort((a, b) => a.time.localeCompare(b.time))
    return groups
})

const totals = computed(() => buckets.value.reduce(
    (acc, b) => {
        acc.ships0 += b.team0.length; acc.ships1 += b.team1.length
        acc.isk0 += b.isk0; acc.isk1 += b.isk1
        return acc
    },
    { ships0: 0, ships1: 0, isk0: 0, isk1: 0 },
))

/** Busiest bucket by ships lost — the peak the overview is scaled against. */
const peak = computed(() => Math.max(1, ...buckets.value.map(b => b.total)))

const peakBucket = computed(() => buckets.value.reduce<Bucket | null>(
    (best, b) => (!best || b.total > best.total ? b : best), null))

// ── Pagination ──────────────────────────────────────────────────────────────
const PAGE_SIZE = 50
const visibleBuckets = ref(PAGE_SIZE)
const visible = computed(() => buckets.value.slice(0, visibleBuckets.value))
const hasMore = computed(() => buckets.value.length > visibleBuckets.value)
const showMore = () => { visibleBuckets.value += PAGE_SIZE }
const fmtNum = (v: number) => v.toLocaleString('en-US')

/** Overview bars double as navigation — reveal the row first if it's paged out. */
const jumpToBucket = async (index: number) => {
    if (index >= visibleBuckets.value) {
        visibleBuckets.value = Math.ceil((index + 1) / PAGE_SIZE) * PAGE_SIZE
        await nextTick()
    }
    const el = document.getElementById(`tl-${buckets.value[index]?.time}`)
    el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

const hoverIdx = ref<number | null>(null)

// Fixed side fills, NOT the theme-mapped red-500/blue-500 utilities. Custom
// domains re-map Tailwind's blue to their own brand colour (see the @theme
// chain in main.css), so on a red-branded killboard both sides rendered red
// and the two columns became indistinguishable. Same validated dark-surface
// pair the campaign charts use.
const SIDE_FILL = [CAMPAIGN_SIDE_FILL[0]!, CAMPAIGN_SIDE_FILL[1]!]

// A busy battle is thousands of rows — 1,318 kills rendered flat is a page
// 43,000px tall that nobody scrolls. Collapsed, a bucket shows its most
// expensive losses (the ones that decided anything); expanding restores the
// full chronological list for that interval.
const COLLAPSED_ROWS = 6
const expanded = ref<Set<string>>(new Set())
const isExpanded = (time: string) => expanded.value.has(time)
const toggleBucket = (time: string) => {
    const next = new Set(expanded.value)
    next.has(time) ? next.delete(time) : next.add(time)
    expanded.value = next
}

/** Collapsed: priciest first. Expanded: back to chronological. */
const rowsFor = (time: string, list: any[]) => {
    if (isExpanded(time)) return list
    if (list.length <= COLLAPSED_ROWS) return list
    return [...list]
        .sort((a, b) => (Number(b.total_value) || 0) - (Number(a.total_value) || 0))
        .slice(0, COLLAPSED_ROWS)
}
</script>

<template>
    <div>
        <div v-if="pending" class="flex items-center justify-center py-12">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <div v-else-if="buckets.length === 0" class="py-12 text-center text-sm text-gray-500">
            No kills found in this battle
        </div>

        <template v-else>
            <!-- Overview: the shape of the fight, and a way to move through it -->
            <div class="glass-panel p-4 mb-4">
                <div class="flex items-center justify-between gap-3 mb-3 flex-wrap">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">
                        Ships lost per {{ bucketLabel }}
                    </div>
                    <div class="flex items-center gap-3 text-fine">
                        <span class="flex items-center gap-1.5">
                            <span class="w-2 h-2 rounded-sm" :style="{ backgroundColor: SIDE_FILL[0] }" />
                            <span class="text-gray-400 truncate max-w-[12rem]">{{ teamName(0) }}</span>
                            <span class="text-gray-500 tabular-nums">{{ fmtNum(totals.ships0) }}</span>
                        </span>
                        <span class="flex items-center gap-1.5">
                            <span class="w-2 h-2 rounded-sm" :style="{ backgroundColor: SIDE_FILL[1] }" />
                            <span class="text-gray-400 truncate max-w-[12rem]">{{ teamName(1) }}</span>
                            <span class="text-gray-500 tabular-nums">{{ fmtNum(totals.ships1) }}</span>
                        </span>
                    </div>
                </div>

                <div class="flex items-end gap-px h-20" @mouseleave="hoverIdx = null">
                    <button v-for="(b, i) in buckets" :key="b.time" type="button"
                        class="group relative flex-1 min-w-[2px] flex flex-col justify-end h-full cursor-pointer"
                        v-tooltip="`${b.label} — ${b.total} ${b.total === 1 ? 'ship' : 'ships'}, ${formatIsk(b.totalIsk)} ISK`"
                        @mouseenter="hoverIdx = i" @click="jumpToBucket(i)">
                        <div class="w-full transition-opacity opacity-80 group-hover:opacity-100"
                            :style="{ height: `${(b.team0.length / peak) * 100}%`, backgroundColor: SIDE_FILL[0] }" />
                        <div class="w-full transition-opacity opacity-80 group-hover:opacity-100"
                            :style="{ height: `${(b.team1.length / peak) * 100}%`, backgroundColor: SIDE_FILL[1] }" />
                    </button>
                </div>

                <div class="flex items-center justify-between gap-3 mt-2 text-fine text-gray-600 tabular-nums">
                    <span>{{ buckets[0]?.label }}</span>
                    <span v-if="hoverIdx !== null && buckets[hoverIdx]" class="text-gray-300">
                        {{ buckets[hoverIdx]!.label }} —
                        <span :style="{ color: SIDE_FILL[0] }">{{ buckets[hoverIdx]!.team0.length }}</span> /
                        <span :style="{ color: SIDE_FILL[1] }">{{ buckets[hoverIdx]!.team1.length }}</span> ships ·
                        <span class="text-yellow-400/80">{{ formatIsk(buckets[hoverIdx]!.totalIsk) }}</span>
                    </span>
                    <span v-else-if="peakBucket" class="text-gray-500">
                        Heaviest {{ bucketLabel }}: {{ peakBucket.label }} · {{ peakBucket.total }} {{ peakBucket.total === 1 ? 'ship' : 'ships' }}
                    </span>
                    <span>{{ buckets[buckets.length - 1]?.label }}</span>
                </div>
            </div>

            <!-- Column key, so the two sides are named rather than inferred from hover colour -->
            <div class="hidden sm:grid grid-cols-2 gap-3 mb-2 px-1">
                <div class="flex items-center gap-2 min-w-0">
                    <span class="w-2 h-2 rounded-sm flex-shrink-0" :style="{ backgroundColor: SIDE_FILL[0] }" />
                    <span class="text-fine font-bold uppercase tracking-[0.15em] truncate" :style="{ color: SIDE_FILL[0] }">{{ teamName(0) }} losses</span>
                    <span class="text-fine text-gray-600 flex-shrink-0">· priciest first</span>
                </div>
                <div class="flex items-center gap-2 min-w-0">
                    <span class="w-2 h-2 rounded-sm flex-shrink-0" :style="{ backgroundColor: SIDE_FILL[1] }" />
                    <span class="text-fine font-bold uppercase tracking-[0.15em] truncate" :style="{ color: SIDE_FILL[1] }">{{ teamName(1) }} losses</span>
                </div>
            </div>

            <div class="space-y-3">
                <div v-for="bucket in visible" :key="bucket.time" :id="`tl-${bucket.time}`"
                    class="glass-panel overflow-hidden scroll-mt-4">
                    <!-- Time header with a per-bucket read of who was losing -->
                    <div class="px-4 py-2 bg-white/[0.02] border-b border-white/[0.06]">
                        <div class="flex items-center justify-between gap-3 flex-wrap">
                            <div class="flex items-center gap-2">
                                <Icon name="lucide:clock" class="text-fine text-gray-600" />
                                <span class="text-xs font-medium text-gray-300 tabular-nums">{{ bucket.label }}</span>
                                <span class="text-fine text-gray-600">EVE</span>
                            </div>
                            <div class="flex items-center gap-3 text-fine tabular-nums">
                                <span class="text-gray-500">{{ bucket.total }} {{ bucket.total === 1 ? 'ship' : 'ships' }}</span>
                                <span class="text-yellow-400/80">{{ formatIsk(bucket.totalIsk) }}</span>
                            </div>
                        </div>
                        <!-- Share of ISK lost this bucket -->
                        <div v-if="bucket.totalIsk > 0" class="flex h-1 mt-2 rounded-full overflow-hidden bg-white/[0.06]">
                            <div class="opacity-70" :style="{ width: `${(bucket.isk0 / bucket.totalIsk) * 100}%`, backgroundColor: SIDE_FILL[0] }" />
                            <div class="opacity-70" :style="{ width: `${(bucket.isk1 / bucket.totalIsk) * 100}%`, backgroundColor: SIDE_FILL[1] }" />
                        </div>
                    </div>

                    <div class="grid grid-cols-2 divide-x divide-white/[0.06]">
                        <div v-for="(side, sideIdx) in [
                            { kills: bucket.team0, isk: bucket.isk0, fill: SIDE_FILL[0] },
                            { kills: bucket.team1, isk: bucket.isk1, fill: SIDE_FILL[1] },
                        ]" :key="sideIdx" class="p-2 space-y-px">
                            <div v-if="side.kills.length === 0" class="py-3 text-center text-fine text-gray-700">
                                No losses
                            </div>
                            <NuxtLink v-for="k in rowsFor(bucket.time, side.kills)" :key="k.killmail_id" :to="`/kill/${k.killmail_id}`"
                                class="flex items-center gap-2 px-2 py-1.5 rounded-md transition-colors hover:bg-white/[0.05]">
                                <img :src="`/images/types/${k.ship_type_id}/icon?size=64`"
                                    class="w-7 h-7 rounded flex-shrink-0 bg-gray-900" loading="lazy" alt="">
                                <div class="min-w-0 flex-1">
                                    <div class="text-xs text-gray-300 truncate">{{ k.victim_character_name || k.victim_corporation_name || 'Unknown' }}</div>
                                    <div class="text-fine text-gray-500 truncate">{{ k.ship_name }}</div>
                                </div>
                                <div class="text-fine tabular-nums flex-shrink-0 opacity-80" :style="{ color: side.fill }">{{ formatIsk(k.total_value) }}</div>
                            </NuxtLink>
                            <div v-if="!isExpanded(bucket.time) && side.kills.length > COLLAPSED_ROWS"
                                class="px-2 pt-1 text-fine text-gray-600">
                                +{{ fmtNum(side.kills.length - COLLAPSED_ROWS) }} more
                            </div>
                        </div>
                    </div>

                    <button v-if="bucket.team0.length > COLLAPSED_ROWS || bucket.team1.length > COLLAPSED_ROWS"
                        type="button"
                        class="w-full py-1.5 text-fine font-medium text-gray-500 bg-white/[0.02] border-t border-white/[0.06] hover:text-blue-400 hover:bg-blue-500/[0.04] transition-colors cursor-pointer"
                        @click="toggleBucket(bucket.time)">
                        {{ isExpanded(bucket.time)
                            ? 'Show top losses only'
                            : `Show all ${fmtNum(bucket.total)} losses in this ${bucketLabel}` }}
                    </button>
                </div>

                <button v-if="hasMore" @click="showMore"
                    class="w-full py-2.5 rounded-lg text-xs font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors cursor-pointer">
                    Show more ({{ fmtNum(buckets.length - visibleBuckets) }} intervals remaining)
                </button>
            </div>
        </template>
    </div>
</template>
