<script setup lang="ts">
const { domainConfig, isDomainMode } = useDomainConfig()
const siteName = computed(() => domainConfig.value?.siteName || 'EVE-KILL')

/**
 * On a custom killboard the stats are scoped to that board's entities via
 * /api/custom/stats, which filters the same precomputed rollups by entity id.
 *
 * The "population" sections (largest / growth / newest / achievements /
 * security) are rankings across all of EVE — member counts, corp age, sec
 * status. Restricted to one board's handful of entities they'd be a list of
 * three rows ranked against nothing, so they're global-only.
 */
const statsApi = computed(() => isDomainMode.value ? '/api/custom/stats' : '/api/stats')

const activeFilter = ref<string>('all')

const PERIOD_SECTIONS = ['activity', 'isk', 'pvp', 'locations', 'ships'] as const
const periodSections = new Set<string>(PERIOD_SECTIONS)

const ALL_FILTERS = [
    { key: 'all', label: 'All' },
    { key: 'activity', label: 'Activity' },
    { key: 'isk', label: 'ISK' },
    { key: 'pvp', label: 'PvP' },
    { key: 'locations', label: 'Locations' },
    { key: 'ships', label: 'Ships', icon: 'lucide:rocket' },
    { key: 'largest', label: 'Largest' },
    { key: 'security', label: 'Security' },
    { key: 'growth', label: 'Growth' },
    { key: 'newest', label: 'Newest' },
    { key: 'achievements', label: 'Achievements' },
]

const filters = computed(() => isDomainMode.value
    ? ALL_FILTERS.filter(f => f.key === 'all' || periodSections.has(f.key))
    : ALL_FILTERS)

// A filter carried over in the URL/state that a domain can't serve would show
// an empty page — fall back to 'all'.
watch([isDomainMode, activeFilter], () => {
    if (isDomainMode.value && !(activeFilter.value === 'all' || periodSections.has(activeFilter.value))) {
        activeFilter.value = 'all'
    }
}, { immediate: true })

const show = (section: string) => {
    if (isDomainMode.value && !periodSections.has(section)) return false
    return activeFilter.value === 'all' || activeFilter.value === section
}

const showPeriod = computed(() => activeFilter.value === 'all' || periodSections.has(activeFilter.value))
const showNonPeriod = computed(() =>
    !isDomainMode.value && (activeFilter.value === 'all' || !periodSections.has(activeFilter.value)))

useHead({ title: () => isDomainMode.value ? `Statistics — ${siteName.value}` : 'EVE Online PvP Statistics' })
useSeoMeta({
    description: () => isDomainMode.value
        ? `Kill statistics for ${siteName.value} — top pilots, corporations, ships, systems and regions by kills and ISK destroyed.`
        : 'EVE Online kill statistics — top pilots, corporations, alliances, ships, systems, and regions ranked by kills, ISK destroyed, and PvP activity.',
    ogTitle: () => `Statistics — ${siteName.value}`,
    ogDescription: () => isDomainMode.value
        ? `Kill stats for ${siteName.value}.`
        : 'EVE Online kill stats — top pilots, corps, alliances, ships, and systems.',
    // A per-board stats page is not a distinct search result worth indexing.
    robots: () => isDomainMode.value ? 'noindex, follow' : 'index, follow',
})

// ─── Activity section ───
const periods = [
    { label: '1h', value: 1 / 24 },
    { label: '6h', value: 6 / 24 },
    { label: '12h', value: 0.5 },
    { label: '1d', value: 1 },
    { label: '2d', value: 2 },
    { label: '3d', value: 3 },
    { label: '4d', value: 4 },
    { label: '5d', value: 5 },
    { label: '6d', value: 6 },
    { label: '7d', value: 7 },
    { label: '14d', value: 14 },
    { label: '30d', value: 30 },
    { label: '45d', value: 45 },
    { label: '90d', value: 90 },
]
const activityPeriod = ref(7)

const SECTION_META: Record<string, { label: string; icon: string; blurb: string }> = {
    activity: { label: 'Activity', icon: 'lucide:crosshair', blurb: 'Who is killing the most' },
    isk: { label: 'ISK', icon: 'lucide:coins', blurb: 'Value destroyed and lost' },
    pvp: { label: 'PvP', icon: 'lucide:swords', blurb: 'Solo work and points earned' },
    locations: { label: 'Locations', icon: 'lucide:map-pinned', blurb: 'Where the fighting happens' },
    ships: { label: 'Ships', icon: 'lucide:rocket', blurb: 'What flies and what dies' },
}

// All period-based stats grouped by filter section
const statCards: { key: string; label: string; imgType: string | null; section: string; icon: string; unit?: string; format?: 'isk' | 'sec' }[] = [
    // Activity (kills)
    { key: 'characters', label: 'Top Killers', imgType: 'character', section: 'activity', icon: 'lucide:user-round' },
    { key: 'corporations', label: 'Top Corporations', imgType: 'corporation', section: 'activity', icon: 'lucide:building-2' },
    { key: 'alliances', label: 'Top Alliances', imgType: 'alliance', section: 'activity', icon: 'lucide:shield' },
    // ISK
    { key: 'isk_destroyers_chars', label: 'Top ISK Destroyers', imgType: 'character', section: 'isk', format: 'isk', icon: 'lucide:user-round' },
    { key: 'isk_destroyers_corps', label: 'Top ISK Corps', imgType: 'corporation', section: 'isk', format: 'isk', icon: 'lucide:building-2' },
    { key: 'isk_destroyers_alliances', label: 'Top ISK Alliances', imgType: 'alliance', section: 'isk', format: 'isk', icon: 'lucide:shield' },
    { key: 'biggest_losers', label: 'Biggest ISK Losers', imgType: 'character', section: 'isk', format: 'isk', icon: 'lucide:trending-down' },
    // PvP
    { key: 'solo_killers', label: 'Top Solo Killers', imgType: 'character', section: 'pvp', icon: 'lucide:user-round' },
    { key: 'top_points', label: 'Top Points Earners', imgType: 'character', section: 'pvp', icon: 'lucide:star' },
    // Locations
    { key: 'systems', label: 'Top Systems', imgType: null, section: 'locations', icon: 'lucide:map-pin' },
    { key: 'regions', label: 'Top Regions', imgType: null, section: 'locations', icon: 'lucide:map' },
    { key: 'dangerous_systems', label: 'Most Dangerous Systems', imgType: null, section: 'locations', format: 'isk', icon: 'lucide:skull' },
    { key: 'deadliest_regions', label: 'Deadliest Regions', imgType: null, section: 'locations', format: 'isk', icon: 'lucide:skull' },
    // Ships
    { key: 'ships', label: 'Top Ships (Kills)', imgType: 'ship', section: 'ships', icon: 'lucide:rocket' },
    { key: 'most_used_ships', label: 'Most Used Ships', imgType: 'ship', section: 'ships', icon: 'lucide:rocket' },
    { key: 'most_destroyed_ships', label: 'Most Destroyed Ships', imgType: 'ship', section: 'ships', icon: 'lucide:flame' },
]

const periodLabel = computed(() => periods.find(p => p.value === activityPeriod.value)?.label ?? `${activityPeriod.value}d`)

const POPULATION_META: Record<string, { label: string; icon: string; blurb: string }> = {
    largest: { label: 'Largest', icon: 'lucide:users', blurb: 'By member count' },
    security: { label: 'Security', icon: 'lucide:scale', blurb: 'Sec status extremes' },
    growth: { label: 'Growth', icon: 'lucide:trending-up', blurb: 'Gaining and shedding members' },
    newest: { label: 'Newest', icon: 'lucide:sparkles', blurb: 'Recently founded' },
    achievements: { label: 'Achievements', icon: 'lucide:trophy', blurb: 'Top achievement scores' },
}

/** Anchors for the jump bar, in page order, limited to what's rendered. */
const jumpTargets = computed(() => {
    const out: { key: string; label: string; icon: string }[] = []
    for (const section of PERIOD_SECTIONS) {
        if (show(section)) out.push({ key: section, ...SECTION_META[section]! })
    }
    if (!isDomainMode.value) {
        for (const [key, meta] of Object.entries(POPULATION_META)) {
            if (show(key)) out.push({ key, ...meta })
        }
    }
    return out
})

const jumpTo = (key: string) => {
    document.getElementById(`stats-${key}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// ISK and Locations carry four tables each; a 3-up grid leaves an awkward
// orphan on the second row, so those two go 4-up on wide screens.
const WIDE_SECTIONS = new Set(['isk', 'locations'])
const gridClassFor = (section: string) => WIDE_SECTIONS.has(section)
    ? 'grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3'
    : 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3'

/** Visible period cards bundled under their section heading. */
const cardGroups = computed(() => PERIOD_SECTIONS
    .filter(section => show(section))
    .map(section => ({
        section,
        ...SECTION_META[section]!,
        cards: statCards.filter(c => c.section === section),
    }))
    .filter(group => group.cards.length > 0))

// Start every request during setup so SSR can await it and hydration can reuse
// its payload. Template reads stay pure; they never start network requests.
const activityRequests = statCards.map(card => ({
    key: card.key,
    request: useApiFetch<{ entries: any[] }>(statsApi, {
        params: computed(() => ({ dataType: card.key, limit: 10, days: activityPeriod.value })),
        lazy: true,
    }),
}))

const securityRequests = isDomainMode.value ? [] : ['pirate', 'carebear'].map(rank => ({
    key: rank,
    request: useApiFetch<{ entries: any[] }>('/api/stats', {
        params: { dataType: `${rank}_characters`, limit: 10 },
        lazy: true,
    }),
}))

const rankingSpecs: { section: string; entityType: string; extra: Record<string, string> }[] = []
if (!isDomainMode.value) {
    for (const entityType of ['alliance', 'corporation']) {
        rankingSpecs.push({ section: 'largest', entityType, extra: {} })
        for (const rank of ['pirate', 'carebear']) rankingSpecs.push({ section: 'security', entityType, extra: { rank } })
        for (const direction of ['growing', 'shrinking']) rankingSpecs.push({ section: 'growth', entityType, extra: { direction } })
    }
    for (const entityType of ['alliance', 'corporation', 'character']) {
        for (const section of ['newest', 'achievements']) rankingSpecs.push({ section, entityType, extra: {} })
    }
}
const rankingKey = (section: string, entityType: string, extra: Record<string, string>) =>
    `${section}-${entityType}-${JSON.stringify(extra)}`
const rankingRequests = rankingSpecs.map(({ section, entityType, extra }) => ({
    key: rankingKey(section, entityType, extra),
    request: useApiFetch<{ entries: any[] }>('/api/stats/rankings', {
        params: { section, entityType, limit: 10, ...extra },
        lazy: true,
    }),
}))

await Promise.all([...activityRequests, ...securityRequests, ...rankingRequests].map(({ request }) => request))

const activityData = computed<Record<string, any[]>>(() => Object.fromEntries(
    activityRequests.map(({ key, request }) => [key, request.data.value?.entries || []]),
))
const activityLoading = computed(() => activityRequests.some(({ request }) => request.pending.value))
const securityChars = computed<Record<string, any[] | undefined>>(() => Object.fromEntries(
    securityRequests.map(({ key, request }) => [key, request.pending.value ? undefined : request.data.value?.entries || []]),
))
const rankingsData = computed<Record<string, any[] | undefined>>(() => Object.fromEntries(
    rankingRequests.map(({ key, request }) => [key, request.pending.value ? undefined : request.data.value?.entries || []]),
))
const rankings = (section: string, entityType: string, extra: Record<string, string> = {}) =>
    rankingsData.value[rankingKey(section, entityType, extra)]

// ─── Formatting ───

const formatDelta = (v: number): string => {
    if (v > 0) return `+${v.toLocaleString('en-US')}`
    return v.toLocaleString('en-US')
}

const deltaColor = (v: number): string => {
    if (v > 0) return 'text-green-400'
    if (v < 0) return 'text-red-400'
    return 'text-gray-600'
}

const timeAgo = (iso: string | null): string => {
    if (!iso) return ''
    const diff = Date.now() - new Date(iso).getTime()
    const days = Math.floor(diff / 86400000)
    if (days < 1) return 'Today'
    if (days < 30) return `${days}d ago`
    if (days < 365) return `${Math.floor(days / 30)}mo ago`
    return `${Math.floor(days / 365)}y ago`
}

const entityImage = (type: string, id: number): string => {
    if (type === 'character') return `/images/characters/${id}/portrait?size=64`
    if (type === 'corporation') return `/images/corporations/${id}/logo?size=64`
    if (type === 'alliance') return `/images/alliances/${id}/logo?size=64`
    if (type === 'ship') return `/images/types/${id}/icon?size=64`
    if (type === 'system') return `/images/systems/${id}?size=64`
    if (type === 'region') return `/images/regions/${id}?size=64`
    return ''
}

const entityLink = (type: string, id: number): string => {
    if (type === 'character') return `/character/${id}`
    if (type === 'corporation') return `/corporation/${id}`
    if (type === 'alliance') return `/alliance/${id}`
    if (type === 'ship') return `/item/${id}`
    if (type === 'system') return `/system/${id}`
    if (type === 'region') return `/region/${id}`
    return '#'
}

const secColor = (sec: number): string => {
    if (sec <= -5) return 'text-red-500'
    if (sec < 0) return 'text-red-400'
    if (sec === 0) return 'text-gray-400'
    if (sec < 3) return 'text-green-400'
    return 'text-green-500'
}
</script>

<template>
    <div>
        <PageHeader class="mb-4" title="Statistics" eyebrow="New Eden by the numbers" icon="lucide:chart-no-axes-column">
            <template #description>
                        <template v-if="isDomainMode">
                            Leaderboards for {{ siteName }} — kills, ISK and activity for this killboard's pilots,
                            corporations and alliances over the period you pick.
                        </template>
                        <template v-else>
                            EVE Online PvP leaderboards — who kills the most, who loses the most, and where. Period
                            tables refresh with the stats rollups; population rankings track the whole cluster.
                        </template>
                Read the <NuxtLink to="/faq" class="text-blue-400 hover:underline">FAQ on combat points and rankings</NuxtLink>
                for how scores are calculated.
            </template>
            <template v-if="isDomainMode" #actions>
                <span v-if="isDomainMode"
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-indigo-500/30 bg-indigo-500/10 text-fine font-medium text-indigo-300 flex-shrink-0"
                    v-tooltip="'Scoped to the entities this killboard tracks'">
                    <Icon name="lucide:shield" class="text-fine" />
                    This killboard only
                </span>
            </template>
        </PageHeader>

        <!-- Section filter -->
        <div class="flex flex-wrap items-center gap-1.5 mb-3">
            <button v-for="f in filters" :key="f.key"
                class="px-3 py-1.5 text-xs rounded-md font-medium border transition-colors cursor-pointer"
                :class="activeFilter === f.key
                    ? 'bg-blue-500/20 text-blue-400 border-blue-500/30'
                    : 'text-gray-400 border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="activeFilter = f.key">
                {{ f.label }}
            </button>
        </div>

        <!-- Jump to section -->
        <div v-if="jumpTargets.length > 1" class="flex flex-wrap items-center gap-1.5 mb-3">
            <span class="text-fine text-gray-600 font-medium uppercase tracking-wider mr-0.5">Jump to</span>
            <button v-for="t in jumpTargets" :key="t.key"
                class="inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-fine font-medium text-gray-500 bg-white/[0.02] border border-white/[0.06] hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors cursor-pointer"
                @click="jumpTo(t.key)">
                <Icon :name="t.icon" class="text-fine" />
                {{ t.label }}
            </button>
        </div>

        <!-- Period selector -->
        <div v-if="showPeriod" class="flex flex-wrap items-center gap-2 mb-4">
            <span class="text-fine text-gray-500 uppercase tracking-wider">Period</span>
            <div class="flex flex-wrap rounded-lg border border-white/[0.08] overflow-hidden">
                <button v-for="p in periods" :key="p.value"
                    class="px-2.5 py-1.5 text-xs font-medium tabular-nums transition-colors cursor-pointer"
                    :class="activityPeriod === p.value
                        ? 'bg-blue-500/20 text-blue-400'
                        : 'bg-white/[0.02] text-gray-500 hover:bg-white/[0.05] hover:text-gray-300'"
                    @click="activityPeriod = p.value">
                    {{ p.label }}
                </button>
            </div>
        </div>

        <!-- ===== PERIOD-BASED STATS, grouped by section ===== -->
        <template v-if="showPeriod">
            <section v-for="group in cardGroups" :key="group.section" :id="`stats-${group.section}`" class="mb-6 scroll-mt-4">
                <div class="flex items-center gap-2.5 mb-2.5">
                    <Icon :name="group.icon" class="text-sm text-blue-400/70 flex-shrink-0" />
                    <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400">{{ group.label }}</h2>
                    <span class="text-fine text-gray-600 truncate">{{ group.blurb }}</span>
                    <span class="ml-auto text-fine text-gray-600 tabular-nums flex-shrink-0">{{ periodLabel }}</span>
                </div>
                <div :class="gridClassFor(group.section)">
                    <div v-for="card in group.cards" :key="card.key" class="glass-panel p-3">
                        <div class="flex items-center gap-2 px-1 pb-2 mb-2 border-b border-white/[0.08]">
                            <Icon :name="card.icon" class="text-fine text-gray-600 flex-shrink-0" />
                            <span class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 truncate">{{ card.label }}</span>
                        </div>
                        <div v-if="activityLoading" class="space-y-1.5 py-1">
                            <div v-for="n in 5" :key="n" class="flex items-center gap-2 px-2">
                                <div class="w-6 h-6 rounded bg-white/[0.04] skeleton-animate flex-shrink-0" />
                                <div class="h-3 rounded bg-white/[0.04] skeleton-animate flex-1" />
                            </div>
                        </div>
                        <div v-else class="space-y-px">
                            <NuxtLink v-for="(e, idx) in activityData[card.key]" :key="e.id" :to="entityLink(e.type, e.id)"
                                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <span class="flex-shrink-0 w-4 text-fine text-right tabular-nums"
                                    :class="idx === 0 ? 'text-amber-300 font-bold' : idx < 3 ? 'text-gray-400' : 'text-gray-600'">{{ idx + 1 }}</span>
                                <img v-if="card.imgType" :src="entityImage(card.imgType, e.id)" class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                                <span class="flex-1 text-xs truncate">{{ e.name }}</span>
                                <span v-if="card.format === 'isk'" class="text-xs text-isk/70 tabular-nums">{{ formatIsk(e.isk || e.count) }}</span>
                                <span v-else-if="card.format === 'sec'" class="text-xs tabular-nums" :class="secColor(e.sec)">{{ e.sec }}</span>
                                <span v-else class="text-xs text-gray-500 tabular-nums">{{ formatNumber(e.count) }}</span>
                            </NuxtLink>
                            <div v-if="!activityData[card.key]?.length" class="py-4 text-center text-fine text-gray-600">
                                No activity in this period
                            </div>
                        </div>
                    </div>
                </div>
            </section>
        </template>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">

            <!-- ===== Separator between period-based and non-period sections ===== -->
            <div v-if="showPeriod && showNonPeriod" class="md:col-span-2 lg:col-span-3 border-t border-white/[0.06] my-2"></div>

            <!-- ===== LARGEST ===== -->
            <template v-if="show('largest')">
                <div :id="`stats-largest`" class="md:col-span-2 lg:col-span-3 flex items-center gap-2.5 mb-0.5 scroll-mt-4">
                    <Icon :name="POPULATION_META.largest!.icon" class="text-sm text-blue-400/70 flex-shrink-0" />
                    <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400">{{ POPULATION_META.largest!.label }}</h2>
                    <span class="text-fine text-gray-600 truncate">{{ POPULATION_META.largest!.blurb }}</span>
                </div>
                <div v-for="et in ['alliance', 'corporation']" :key="`largest-${et}`" class="glass-panel p-3">
                    <div class="px-1 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                        Largest {{ et === 'alliance' ? 'Alliances' : 'Corporations' }}
                    </div>
                    <div v-if="!rankings('largest', et)" class="flex items-center justify-center py-6">
                        <Icon name="lucide:loader-2" class="w-4 h-4 text-gray-500 animate-spin" />
                    </div>
                    <template v-else>
                        <div class="space-y-px">
                            <NuxtLink v-for="(e, idx) in rankings('largest', et)" :key="e.id" :to="entityLink(et, e.id)"
                                class="flex items-center gap-2 px-2 py-1.5 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ idx + 1 }}</span>
                                <img :src="entityImage(et, e.id)" class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                                <div class="flex-1 min-w-0">
                                    <div class="text-xs truncate">{{ e.name }}</div>
                                    <div class="text-fine text-gray-500 tabular-nums">{{ formatNumber(e.member_count) }} members</div>
                                </div>
                                <div class="flex gap-2 text-fine tabular-nums flex-shrink-0">
                                    <span :class="deltaColor(e.delta_1d)">{{ formatDelta(e.delta_1d) }}</span>
                                    <span :class="deltaColor(e.delta_7d)">{{ formatDelta(e.delta_7d) }}</span>
                                    <span :class="deltaColor(e.delta_30d)">{{ formatDelta(e.delta_30d) }}</span>
                                </div>
                            </NuxtLink>
                        </div>
                        <div class="flex justify-end gap-2 px-2 pt-1 text-fine text-gray-600">
                            <span>1d</span><span>7d</span><span>30d</span>
                        </div>
                    </template>
                </div>
            </template>

            <!-- ===== SECURITY ===== -->
            <template v-if="show('security')">
                <div :id="`stats-security`" class="md:col-span-2 lg:col-span-3 flex items-center gap-2.5 mb-0.5 scroll-mt-4">
                    <Icon :name="POPULATION_META.security!.icon" class="text-sm text-blue-400/70 flex-shrink-0" />
                    <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400">{{ POPULATION_META.security!.label }}</h2>
                    <span class="text-fine text-gray-600 truncate">{{ POPULATION_META.security!.blurb }}</span>
                </div>
                <div v-for="combo in [
                    { et: 'alliance', rank: 'pirate', label: 'Most Pirate Alliances', color: 'text-red-400/80' },
                    { et: 'corporation', rank: 'pirate', label: 'Most Pirate Corporations', color: 'text-red-400/80' },
                    { et: 'alliance', rank: 'carebear', label: 'Most Carebear Alliances', color: 'text-cyan-400/80' },
                    { et: 'corporation', rank: 'carebear', label: 'Most Carebear Corporations', color: 'text-cyan-400/80' },
                ]" :key="`sec-${combo.et}-${combo.rank}`" class="glass-panel p-3">
                    <div class="px-1 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] border-b border-white/[0.08]" :class="combo.color">
                        {{ combo.label }}
                    </div>
                    <div v-if="!rankings('security', combo.et, { rank: combo.rank })" class="flex items-center justify-center py-6">
                        <Icon name="lucide:loader-2" class="w-4 h-4 text-gray-500 animate-spin" />
                    </div>
                    <div v-else class="space-y-px">
                        <NuxtLink v-for="(e, idx) in rankings('security', combo.et, { rank: combo.rank })" :key="e.id" :to="entityLink(combo.et, e.id)"
                            class="flex items-center gap-2 px-2 py-1.5 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                            <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ idx + 1 }}</span>
                            <img :src="entityImage(combo.et, e.id)" class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                            <div class="flex-1 min-w-0">
                                <div class="text-xs truncate">{{ e.name }}</div>
                                <div class="text-fine text-gray-500 tabular-nums">{{ formatNumber(e.member_count) }} members</div>
                            </div>
                            <div class="flex-shrink-0 text-right">
                                <div class="text-xs tabular-nums" :class="secColor(e.avg_sec_status)">{{ e.avg_sec_status }}</div>
                                <div class="text-fine text-gray-600 tabular-nums">{{ e.weighted_score }}</div>
                            </div>
                        </NuxtLink>
                        <div v-if="rankings('security', combo.et, { rank: combo.rank })?.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                    </div>
                </div>

                <!-- Pirate/Carebear Characters -->
                <div v-for="rank in [
                    { key: 'pirate', label: 'Most Pirate Characters', color: 'text-red-400/80' },
                    { key: 'carebear', label: 'Most Carebear Characters', color: 'text-cyan-400/80' },
                ]" :key="`sec-char-${rank.key}`" class="glass-panel p-3">
                    <div class="px-1 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] border-b border-white/[0.08]" :class="rank.color">
                        {{ rank.label }}
                    </div>
                    <div v-if="!securityChars[rank.key]" class="flex items-center justify-center py-6">
                        <Icon name="lucide:loader-2" class="w-4 h-4 text-gray-500 animate-spin" />
                    </div>
                    <div v-else class="space-y-px">
                        <NuxtLink v-for="(e, idx) in securityChars[rank.key]" :key="e.id" :to="entityLink('character', e.id)"
                            class="flex items-center gap-2 px-2 py-1.5 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                            <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ idx + 1 }}</span>
                            <img :src="entityImage('character', e.id)" class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                            <div class="flex-1 min-w-0">
                                <div class="text-xs truncate">{{ e.name }}</div>
                                <div class="text-fine text-gray-600">&nbsp;</div>
                            </div>
                            <span class="text-xs tabular-nums" :class="secColor(e.sec)">{{ e.sec }}</span>
                        </NuxtLink>
                        <div v-if="securityChars[rank.key]?.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                    </div>
                </div>
            </template>

            <!-- ===== GROWTH ===== -->
            <template v-if="show('growth')">
                <div :id="`stats-growth`" class="md:col-span-2 lg:col-span-3 flex items-center gap-2.5 mb-0.5 scroll-mt-4">
                    <Icon :name="POPULATION_META.growth!.icon" class="text-sm text-blue-400/70 flex-shrink-0" />
                    <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400">{{ POPULATION_META.growth!.label }}</h2>
                    <span class="text-fine text-gray-600 truncate">{{ POPULATION_META.growth!.blurb }}</span>
                </div>
                <div v-for="combo in [
                    { et: 'alliance', dir: 'growing', label: 'Fastest Growing Alliances', color: 'text-isk/80' },
                    { et: 'corporation', dir: 'growing', label: 'Fastest Growing Corporations', color: 'text-isk/80' },
                    { et: 'alliance', dir: 'shrinking', label: 'Fastest Shrinking Alliances', color: 'text-red-400/80' },
                    { et: 'corporation', dir: 'shrinking', label: 'Fastest Shrinking Corporations', color: 'text-red-400/80' },
                ]" :key="`growth-${combo.et}-${combo.dir}`" class="glass-panel p-3">
                    <div class="px-1 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] border-b border-white/[0.08]" :class="combo.color">
                        {{ combo.label }} — 7d
                    </div>
                    <div v-if="!rankings('growth', combo.et, { direction: combo.dir })" class="flex items-center justify-center py-6">
                        <Icon name="lucide:loader-2" class="w-4 h-4 text-gray-500 animate-spin" />
                    </div>
                    <div v-else class="space-y-px">
                        <NuxtLink v-for="(e, idx) in rankings('growth', combo.et, { direction: combo.dir })" :key="e.id" :to="entityLink(combo.et, e.id)"
                            class="flex items-center gap-2 px-2 py-1.5 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                            <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ idx + 1 }}</span>
                            <img :src="entityImage(combo.et, e.id)" class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                            <div class="flex-1 min-w-0">
                                <div class="text-xs truncate">{{ e.name }}</div>
                                <div class="text-fine text-gray-500 tabular-nums">{{ formatNumber(e.member_count) }} members</div>
                            </div>
                            <span class="text-xs tabular-nums font-medium flex-shrink-0" :class="deltaColor(e.delta)">{{ formatDelta(e.delta) }}</span>
                        </NuxtLink>
                        <div v-if="rankings('growth', combo.et, { direction: combo.dir })?.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                    </div>
                </div>
            </template>

            <!-- ===== NEWEST ===== -->
            <template v-if="show('newest')">
                <div :id="`stats-newest`" class="md:col-span-2 lg:col-span-3 flex items-center gap-2.5 mb-0.5 scroll-mt-4">
                    <Icon :name="POPULATION_META.newest!.icon" class="text-sm text-blue-400/70 flex-shrink-0" />
                    <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400">{{ POPULATION_META.newest!.label }}</h2>
                    <span class="text-fine text-gray-600 truncate">{{ POPULATION_META.newest!.blurb }}</span>
                </div>
                <div v-for="et in ['alliance', 'corporation', 'character']" :key="`newest-${et}`" class="glass-panel p-3">
                    <div class="px-1 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-amber-400/80 border-b border-white/[0.08]">
                        Newest {{ et === 'alliance' ? 'Alliances' : et === 'corporation' ? 'Corporations' : 'Characters' }}
                    </div>
                    <div v-if="!rankings('newest', et)" class="flex items-center justify-center py-6">
                        <Icon name="lucide:loader-2" class="w-4 h-4 text-gray-500 animate-spin" />
                    </div>
                    <div v-else class="space-y-px">
                        <NuxtLink v-for="(e, idx) in rankings('newest', et)" :key="e.id" :to="entityLink(et, e.id)"
                            class="flex items-center gap-2 px-2 py-1.5 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                            <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ idx + 1 }}</span>
                            <img :src="entityImage(et, e.id)" class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                            <div class="flex-1 min-w-0">
                                <div class="text-xs truncate">{{ e.name }}</div>
                                <div v-if="e.member_count" class="text-fine text-gray-500 tabular-nums">{{ formatNumber(e.member_count) }} members</div>
                                <div v-else class="text-fine text-gray-600">&nbsp;</div>
                            </div>
                            <span class="text-fine text-gray-500 flex-shrink-0">{{ timeAgo(e.date_founded) }}</span>
                        </NuxtLink>
                        <div v-if="rankings('newest', et)?.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                    </div>
                </div>
            </template>

            <!-- ===== ACHIEVEMENTS ===== -->
            <template v-if="show('achievements')">
                <div :id="`stats-achievements`" class="md:col-span-2 lg:col-span-3 flex items-center gap-2.5 mb-0.5 scroll-mt-4">
                    <Icon :name="POPULATION_META.achievements!.icon" class="text-sm text-blue-400/70 flex-shrink-0" />
                    <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400">{{ POPULATION_META.achievements!.label }}</h2>
                    <span class="text-fine text-gray-600 truncate">{{ POPULATION_META.achievements!.blurb }}</span>
                </div>
                <div v-for="combo in [
                    { et: 'character', label: 'Top Achievement Characters' },
                    { et: 'corporation', label: 'Top Achievement Corporations' },
                    { et: 'alliance', label: 'Top Achievement Alliances' },
                ]" :key="`ach-${combo.et}`" class="glass-panel p-3">
                    <div class="px-1 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-purple-400/80 border-b border-white/[0.08]">
                        {{ combo.label }}
                    </div>
                    <div v-if="!rankings('achievements', combo.et)" class="flex items-center justify-center py-6">
                        <Icon name="lucide:loader-2" class="w-4 h-4 text-gray-500 animate-spin" />
                    </div>
                    <div v-else class="space-y-px">
                        <NuxtLink v-for="(e, idx) in rankings('achievements', combo.et)" :key="e.id" :to="entityLink(combo.et, e.id)"
                            class="flex items-center gap-2 px-2 py-1.5 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                            <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ idx + 1 }}</span>
                            <img :src="entityImage(combo.et, e.id)" class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                            <div class="flex-1 min-w-0">
                                <div class="text-xs truncate">{{ e.name }}</div>
                                <div v-if="combo.et !== 'character'" class="text-fine text-gray-500 tabular-nums">{{ e.avg_points }} avg/member</div>
                                <div v-else class="text-fine text-gray-600">&nbsp;</div>
                            </div>
                            <span class="text-xs text-purple-400 tabular-nums font-medium flex-shrink-0">{{ formatNumber(e.total_points) }}</span>
                        </NuxtLink>
                        <div v-if="rankings('achievements', combo.et)?.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                    </div>
                </div>
            </template>

        </div>
    </div>
</template>
