<script setup lang="ts">
const { domainConfig, isDomainMode } = useDomainConfig()
const siteName = computed(() => domainConfig.value?.siteName || 'EVE-KILL')

useHead({ title: 'EVE Online Wars' })
useSeoMeta({
    description: 'Browse active and historical wars in EVE Online — war declarations, faction warfare statistics, and conflict tracking on EVE-KILL.',
    ogTitle: 'Wars — EVE-KILL',
    ogDescription: 'Active and historical EVE Online wars — declarations, faction warfare, and conflict tracking.',
})

// Faction warfare stats
const fwRequest = useApiFetch<any>('/api/faction-wars', { lazy: true })
const warStatsRequest = useApiFetch<any>('/api/wars/stats', { lazy: true })

type WarsTab = 'wars' | 'finished' | 'upcoming' | 'eligible-corps' | 'eligible-alliances'

const activeTab = ref<WarsTab>('wars')
const page = ref(1)
const mutual = ref(false)
const hasActivity = ref(false)
const hasKills = ref(false)
const hasAllies = ref(false)
const sortBy = ref<'recent' | 'kills' | 'isk'>('recent')

const isUpcomingTab = computed(() => activeTab.value === 'upcoming')
const isFinishedTab = computed(() => activeTab.value === 'finished')
const isEligibleTab = computed(() => activeTab.value === 'eligible-corps' || activeTab.value === 'eligible-alliances')
const isWarListTab = computed(() => activeTab.value === 'wars' || activeTab.value === 'upcoming' || activeTab.value === 'finished')
const eligibleType = computed<'corporations' | 'alliances'>(() => activeTab.value === 'eligible-alliances' ? 'alliances' : 'corporations')

const fetchParams = computed(() => {
    if (isUpcomingTab.value) {
        return { page: page.value, limit: 50, upcoming: 'true' }
    }
    const statusParam = isFinishedTab.value ? { finished: 'true' } : { ongoing: 'true' }
    return {
        page: page.value,
        limit: 50,
        sort: sortBy.value,
        ...statusParam,
        ...(mutual.value ? { mutual: 'true' } : {}),
        ...(hasActivity.value ? { hasActivity: 'true' } : {}),
        ...(hasKills.value ? { hasKills: 'true' } : {}),
        ...(hasAllies.value ? { hasAllies: 'true' } : {}),
    }
})

const warsRequest = useApiFetch<any>('/api/conflicts/wars', {
    params: fetchParams,
    watch: [page, activeTab, mutual, hasActivity, hasKills, hasAllies, sortBy],
})

await Promise.all([fwRequest, warStatsRequest, warsRequest])
const { data: fwData } = fwRequest
const { data: warStats } = warStatsRequest
const { data, pending } = warsRequest

const wars = computed(() => data.value?.wars || [])

const eligibleFetchParams = computed(() => ({ type: eligibleType.value }))

const { data: eligibleData, pending: eligiblePending } = useApiFetch<any>('/api/wars/eligible', {
    params: eligibleFetchParams,
    lazy: true,
    server: false,
    // Only fetch while an eligible-* tab is active; switching away aborts
    // any in-flight request. The activeTab watch triggers the first fetch
    // when the user enters the tab (enabled flipping true doesn't fetch
    // on its own).
    enabled: () => isEligibleTab.value,
    watch: [activeTab],
})

const eligibleEntries = computed(() => eligibleData.value?.entries || [])

const setTab = (tab: WarsTab) => {
    if (activeTab.value === tab) return
    activeTab.value = tab
    page.value = 1
}

const resetFilters = () => {
    mutual.value = false
    hasActivity.value = false
    hasKills.value = false
    hasAllies.value = false
    sortBy.value = 'recent'
    page.value = 1
}

const toggleFilter = (filter: 'mutual' | 'hasActivity' | 'hasKills' | 'hasAllies') => {
    if (filter === 'mutual') mutual.value = !mutual.value
    else if (filter === 'hasActivity') hasActivity.value = !hasActivity.value
    else if (filter === 'hasKills') hasKills.value = !hasKills.value
    else if (filter === 'hasAllies') hasAllies.value = !hasAllies.value
    page.value = 1
}

const setSort = (sort: 'recent' | 'kills' | 'isk') => {
    sortBy.value = sort
    page.value = 1
}

const hasAnyFilter = computed(() => mutual.value || hasActivity.value || hasKills.value || hasAllies.value || sortBy.value !== 'recent')

const timeAgo = (iso: string | null): string => {
    if (!iso) return ''
    const diff = Date.now() - new Date(iso).getTime()
    const future = diff < 0
    const abs = Math.abs(diff)
    const mins = Math.floor(abs / 60000)
    const fmt = (val: number, unit: string) => future ? `in ${val}${unit}` : `${val}${unit} ago`
    if (mins < 60) return fmt(mins, 'm')
    const hours = Math.floor(mins / 60)
    if (hours < 24) return fmt(hours, 'h')
    const days = Math.floor(hours / 24)
    if (days < 30) return fmt(days, 'd')
    const months = Math.floor(days / 30)
    if (months < 12) return fmt(months, 'mo')
    return fmt(Math.floor(days / 365), 'y')
}

const entityImage = (entity: any): string => {
    if (entity.type === 'alliance') return `/images/alliances/${entity.id}/logo?size=64`
    return `/images/corporations/${entity.id}/logo?size=64`
}

const entityLink = (entity: any): string => `/${entity.type}/${entity.id}`

const isUpcoming = (w: any): boolean => {
    if (!w.started) return false
    return new Date(w.started).getTime() > Date.now()
}

const efficiencyWidth = (w: any): string => {
    const total = (w.aggressor.isk_destroyed || 0) + (w.defender.isk_destroyed || 0)
    if (total === 0) return '50%'
    return `${Math.round((w.aggressor.isk_destroyed / total) * 100)}%`
}
</script>

<template>
    <div>
        <PageHeader class="mb-4" title="Wars" eyebrow="Declared conflicts" icon="lucide:swords">
            <template #description>
                        <template v-if="isDomainMode">
                            Declared wars involving {{ siteName }}'s corporations and alliances — who is shooting whom,
                            and what it has cost so far.
                        </template>
                        <template v-else>
                            Declared wars in EVE Online — aggressors, defenders, allies and the ISK burned. Pick a
                            card below to switch between active, finished and upcoming wars.
                        </template>
                Explore individual engagements in
                <NuxtLink to="/battles" class="text-blue-400 hover:underline">battle reports</NuxtLink>, or follow a wider conflict through
                <NuxtLink to="/campaigns" class="text-blue-400 hover:underline">campaign scoreboards</NuxtLink>.
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

        <!-- Faction Warfare — cluster-wide, so global only -->
        <div v-if="fwData && !isDomainMode" class="mb-6">
            <div class="flex items-center gap-2.5 mb-2.5">
                <Icon name="lucide:flag" class="text-sm text-amber-400/70 flex-shrink-0" />
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400">Faction Warfare</h2>
                <span class="text-fine text-gray-600">Last 30 days</span>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- Caldari vs Gallente -->
            <NuxtLink to="/faction-war/caldari-vs-gallente" class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-4 hover:bg-blue-500/[0.06] transition-colors block">
                <div class="flex items-center gap-3">
                    <div class="flex-1 text-center">
                        <img :src="'/images/corporations/1000035/logo?size=64'" class="w-12 h-12 mx-auto rounded-lg mb-1.5" loading="lazy">
                        <div class="text-sm font-medium text-gray-200">Caldari</div>
                        <div class="text-xs text-green-400 tabular-nums">{{ formatIsk(fwData.caldariVsGallente.caldari.isk_destroyed) }}</div>
                        <div class="text-fine text-gray-500 tabular-nums">{{ fwData.caldariVsGallente.caldari.kills.toLocaleString('en-US') }} kills</div>
                        <div v-if="fwData.caldariVsGallente.caldari.systems_controlled" class="text-fine text-amber-400/80 tabular-nums mt-0.5">{{ fwData.caldariVsGallente.caldari.systems_controlled }} systems · {{ fwData.caldariVsGallente.caldari.pilots?.toLocaleString('en-US') }} pilots</div>
                    </div>
                    <div class="flex flex-col items-center gap-1.5">
                        <div class="text-xs font-bold text-gray-600">VS</div>
                        <div class="w-20 h-1.5 bg-blue-500/30 rounded-full overflow-hidden">
                            <div class="h-full bg-amber-500 rounded-full" :style="{ width: `${fwData.caldariVsGallente.caldari.kills + fwData.caldariVsGallente.gallente.kills > 0 ? Math.round(fwData.caldariVsGallente.caldari.kills / (fwData.caldariVsGallente.caldari.kills + fwData.caldariVsGallente.gallente.kills) * 100) : 50}%` }"></div>
                        </div>
                    </div>
                    <div class="flex-1 text-center">
                        <img :src="'/images/corporations/1000120/logo?size=64'" class="w-12 h-12 mx-auto rounded-lg mb-1.5" loading="lazy">
                        <div class="text-sm font-medium text-gray-200">Gallente</div>
                        <div class="text-xs text-green-400 tabular-nums">{{ formatIsk(fwData.caldariVsGallente.gallente.isk_destroyed) }}</div>
                        <div class="text-fine text-gray-500 tabular-nums">{{ fwData.caldariVsGallente.gallente.kills.toLocaleString('en-US') }} kills</div>
                        <div v-if="fwData.caldariVsGallente.gallente.systems_controlled" class="text-fine text-amber-400/80 tabular-nums mt-0.5">{{ fwData.caldariVsGallente.gallente.systems_controlled }} systems · {{ fwData.caldariVsGallente.gallente.pilots?.toLocaleString('en-US') }} pilots</div>
                    </div>
                </div>
            </NuxtLink>

            <!-- Amarr vs Minmatar -->
            <NuxtLink to="/faction-war/amarr-vs-minmatar" class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-4 hover:bg-blue-500/[0.06] transition-colors block">
                <div class="flex items-center gap-3">
                    <div class="flex-1 text-center">
                        <img :src="'/images/corporations/1000084/logo?size=64'" class="w-12 h-12 mx-auto rounded-lg mb-1.5" loading="lazy">
                        <div class="text-sm font-medium text-gray-200">Amarr</div>
                        <div class="text-xs text-green-400 tabular-nums">{{ formatIsk(fwData.amarrVsMinmatar.amarr.isk_destroyed) }}</div>
                        <div class="text-fine text-gray-500 tabular-nums">{{ fwData.amarrVsMinmatar.amarr.kills.toLocaleString('en-US') }} kills</div>
                        <div v-if="fwData.amarrVsMinmatar.amarr.systems_controlled" class="text-fine text-amber-400/80 tabular-nums mt-0.5">{{ fwData.amarrVsMinmatar.amarr.systems_controlled }} systems · {{ fwData.amarrVsMinmatar.amarr.pilots?.toLocaleString('en-US') }} pilots</div>
                    </div>
                    <div class="flex flex-col items-center gap-1.5">
                        <div class="text-xs font-bold text-gray-600">VS</div>
                        <div class="w-20 h-1.5 bg-red-500/30 rounded-full overflow-hidden">
                            <div class="h-full bg-amber-500 rounded-full" :style="{ width: `${fwData.amarrVsMinmatar.amarr.kills + fwData.amarrVsMinmatar.minmatar.kills > 0 ? Math.round(fwData.amarrVsMinmatar.amarr.kills / (fwData.amarrVsMinmatar.amarr.kills + fwData.amarrVsMinmatar.minmatar.kills) * 100) : 50}%` }"></div>
                        </div>
                    </div>
                    <div class="flex-1 text-center">
                        <img :src="'/images/corporations/1000051/logo?size=64'" class="w-12 h-12 mx-auto rounded-lg mb-1.5" loading="lazy">
                        <div class="text-sm font-medium text-gray-200">Minmatar</div>
                        <div class="text-xs text-green-400 tabular-nums">{{ formatIsk(fwData.amarrVsMinmatar.minmatar.isk_destroyed) }}</div>
                        <div class="text-fine text-gray-500 tabular-nums">{{ fwData.amarrVsMinmatar.minmatar.kills.toLocaleString('en-US') }} kills</div>
                        <div v-if="fwData.amarrVsMinmatar.minmatar.systems_controlled" class="text-fine text-amber-400/80 tabular-nums mt-0.5">{{ fwData.amarrVsMinmatar.minmatar.systems_controlled }} systems · {{ fwData.amarrVsMinmatar.minmatar.pilots?.toLocaleString('en-US') }} pilots</div>
                    </div>
                </div>
            </NuxtLink>
            </div>
        </div>

        <!-- War stat boxes -->
        <div v-if="warStats" class="grid grid-cols-2 md:grid-cols-5 gap-2 mb-6">
            <button type="button"
                class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-3 text-left hover:bg-green-500/[0.06] hover:border-green-500/30 transition-colors"
                :class="{ 'bg-green-500/[0.06] border-green-500/30': activeTab === 'wars' }"
                @click="setTab('wars')">
                <div class="flex items-center gap-1.5 text-fine font-bold uppercase tracking-wider text-green-400/80 mb-1">
                    <Icon name="lucide:swords" class="w-3 h-3" />
                    <span>Active Wars</span>
                </div>
                <div class="text-lg font-bold text-white tabular-nums">{{ warStats.activeWars.toLocaleString('en-US') }}</div>
            </button>
            <button type="button"
                class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-3 text-left hover:bg-gray-500/[0.08] hover:border-gray-500/30 transition-colors"
                :class="{ 'bg-gray-500/[0.08] border-gray-500/30': activeTab === 'finished' }"
                @click="setTab('finished')">
                <div class="flex items-center gap-1.5 text-fine font-bold uppercase tracking-wider text-gray-400 mb-1">
                    <Icon name="lucide:flag" class="w-3 h-3" />
                    <span>Finished Wars</span>
                </div>
                <div class="text-lg font-bold text-white tabular-nums">{{ warStats.finishedWars.toLocaleString('en-US') }}</div>
            </button>
            <button type="button"
                class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-3 text-left hover:bg-amber-500/[0.06] hover:border-amber-500/30 transition-colors"
                :class="{ 'bg-amber-500/[0.06] border-amber-500/30': activeTab === 'upcoming' }"
                @click="setTab('upcoming')">
                <div class="flex items-center gap-1.5 text-fine font-bold uppercase tracking-wider text-amber-400/80 mb-1">
                    <Icon name="lucide:clock" class="w-3 h-3" />
                    <span>Upcoming Wars</span>
                </div>
                <div class="text-lg font-bold text-white tabular-nums">{{ warStats.upcomingWars.toLocaleString('en-US') }}</div>
            </button>
            <button type="button"
                class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-3 text-left hover:bg-blue-500/[0.06] hover:border-blue-500/30 transition-colors"
                :class="{ 'bg-blue-500/[0.06] border-blue-500/30': activeTab === 'eligible-corps' }"
                @click="setTab('eligible-corps')">
                <div class="flex items-center gap-1.5 text-fine font-bold uppercase tracking-wider text-blue-400/80 mb-1">
                    <Icon name="lucide:building-2" class="w-3 h-3" />
                    <span>Eligible Corps</span>
                </div>
                <div class="text-lg font-bold text-white tabular-nums">{{ warStats.eligibleCorps.toLocaleString('en-US') }}</div>
            </button>
            <button type="button"
                class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-3 text-left hover:bg-purple-500/[0.06] hover:border-purple-500/30 transition-colors"
                :class="{ 'bg-purple-500/[0.06] border-purple-500/30': activeTab === 'eligible-alliances' }"
                @click="setTab('eligible-alliances')">
                <div class="flex items-center gap-1.5 text-fine font-bold uppercase tracking-wider text-purple-400/80 mb-1">
                    <Icon name="lucide:landmark" class="w-3 h-3" />
                    <span>Eligible Alliances</span>
                </div>
                <div class="text-lg font-bold text-white tabular-nums">{{ warStats.eligibleAlliances.toLocaleString('en-US') }}</div>
            </button>
        </div>

        <!-- Tab bar -->
        <div class="flex overflow-x-auto border-b border-white/[0.08] mb-4 scrollbar-hide">
            <button v-for="tab in [
                { id: 'wars', label: 'Wars', icon: 'lucide:swords' },
                { id: 'finished', label: 'Finished', icon: 'lucide:flag' },
                { id: 'upcoming', label: 'Upcoming', icon: 'lucide:clock' },
                { id: 'eligible-corps', label: 'War Eligible Corporations', icon: 'lucide:building-2' },
                { id: 'eligible-alliances', label: 'War Eligible Alliances', icon: 'lucide:landmark' },
            ]" :key="tab.id"
                class="flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors border-b-2 whitespace-nowrap"
                :class="activeTab === tab.id ? 'text-white border-white' : 'text-gray-500 border-transparent hover:text-blue-400'"
                @click="setTab(tab.id as any)">
                <Icon :name="tab.icon" class="text-base" />
                <span>{{ tab.label }}</span>
            </button>
        </div>

        <!-- Filters + Sort (Wars / Finished tabs) -->
        <div v-if="activeTab === 'wars' || activeTab === 'finished'" class="flex flex-wrap items-center gap-2 mb-4">
            <!-- Filter toggles -->
            <button
                v-for="f in [
                    { key: 'mutual', label: 'Mutual', active: mutual },
                    { key: 'hasActivity', label: 'Has ISK', active: hasActivity },
                    { key: 'hasKills', label: 'Has Kills', active: hasKills },
                    { key: 'hasAllies', label: 'Has Allies', active: hasAllies },
                ]" :key="f.key"
                class="px-3 py-1.5 text-xs rounded-md font-medium border transition-colors cursor-pointer"
                :class="f.active ? 'bg-blue-500/20 text-blue-400 border-blue-500/30' : 'text-gray-400 border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="toggleFilter(f.key as any)">
                {{ f.label }}
            </button>

            <div class="w-px h-5 bg-white/[0.08] mx-1"></div>

            <!-- Sort -->
            <span class="text-fine text-gray-500 uppercase tracking-wider">Sort</span>
            <div class="flex rounded-lg border border-white/[0.08] overflow-hidden">
                <button
                    v-for="s in [
                        { key: 'recent', label: 'Recent' },
                        { key: 'kills', label: 'Most Kills' },
                        { key: 'isk', label: 'Most ISK' },
                    ]" :key="s.key"
                    class="px-2.5 py-1.5 text-xs font-medium transition-colors cursor-pointer"
                    :class="sortBy === s.key ? 'bg-purple-500/20 text-purple-400' : 'bg-white/[0.02] text-gray-500 hover:bg-white/[0.05] hover:text-gray-300'"
                    @click="setSort(s.key as any)">
                    {{ s.label }}
                </button>
            </div>

            <button v-if="hasAnyFilter"
                class="px-2 py-1.5 text-xs text-gray-500 hover:text-blue-400 transition-colors"
                @click="resetFilters">
                <Icon name="lucide:x" class="w-3 h-3" /> Clear
            </button>
        </div>

        <!-- War list -->
        <template v-if="isWarListTab">
        <!-- Loading -->
        <div v-if="pending && wars.length === 0" class="flex items-center justify-center py-20">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <div v-else class="glass-panel" :class="{ 'opacity-60': pending }">
            <!-- Desktop header -->
            <div class="hidden md:grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_100px_100px] gap-3 px-4 py-2 text-fine font-bold uppercase tracking-wider text-gray-500 border-b border-white/[0.08]">
                <div>Aggressor</div>
                <div></div>
                <div>Defender</div>
                <div>{{ isUpcomingTab ? 'Starts' : 'Started' }}</div>
                <div class="text-right">Status</div>
            </div>

            <!-- Responsive rows -->
            <NuxtLink v-for="w in wars" :key="w.war_id" :to="`/war/${w.war_id}`"
                class="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_100px_100px] gap-3 px-4 py-3 border-b border-white/[0.04] hover:bg-blue-500/[0.04] transition-colors items-center">
                <div class="flex items-center gap-2.5 min-w-0">
                    <img :src="entityImage(w.aggressor)" class="w-7 h-7 md:w-8 md:h-8 rounded flex-shrink-0" loading="lazy">
                    <div class="min-w-0">
                        <div class="text-xs text-gray-200 truncate">{{ w.aggressor.name }}</div>
                        <div class="text-fine text-gray-500">
                            <span class="text-isk/70 tabular-nums">{{ formatIsk(w.aggressor.isk_destroyed) }}</span>
                            <span class="hidden md:inline text-gray-600 mx-1">·</span>
                            <span class="hidden md:inline tabular-nums">{{ w.aggressor.ships_killed }} ships</span>
                        </div>
                    </div>
                </div>
                <div class="flex items-center justify-center text-xs text-gray-600 font-bold px-2">VS</div>
                <div class="flex items-center gap-2.5 min-w-0">
                    <img :src="entityImage(w.defender)" class="w-7 h-7 md:w-8 md:h-8 rounded flex-shrink-0" loading="lazy">
                    <div class="min-w-0">
                        <div class="text-xs text-gray-200 truncate">{{ w.defender.name }}</div>
                        <div class="text-fine text-gray-500">
                            <span class="text-isk/70 tabular-nums">{{ formatIsk(w.defender.isk_destroyed) }}</span>
                            <span class="hidden md:inline text-gray-600 mx-1">·</span>
                            <span class="hidden md:inline tabular-nums">{{ w.defender.ships_killed }} ships</span>
                        </div>
                    </div>
                </div>
                <div class="col-span-2 md:col-span-1 text-fine md:text-xs text-gray-400">{{ timeAgo(w.started) }}</div>
                <div class="flex flex-wrap gap-1 justify-end">
                    <span v-if="isUpcoming(w)" class="px-1.5 py-0.5 text-fine rounded bg-amber-500/20 text-amber-400 font-medium">Upcoming</span>
                    <span v-else-if="!w.finished" class="px-1.5 py-0.5 text-fine rounded bg-green-500/20 text-green-400 font-medium">Active</span>
                    <span v-else class="px-1.5 py-0.5 text-fine rounded bg-gray-500/20 text-gray-400 font-medium">Finished</span>
                    <span v-if="w.mutual" class="px-1.5 py-0.5 text-fine rounded bg-amber-500/20 text-amber-400 font-medium">Mutual</span>
                    <span v-if="w.open_for_allies" class="px-1.5 py-0.5 text-fine rounded bg-blue-500/20 text-blue-400 font-medium">Allies</span>
                </div>
            </NuxtLink>

            <div v-if="wars.length === 0" class="py-12 text-center text-sm text-gray-500">{{ isUpcomingTab ? 'No upcoming wars' : isFinishedTab ? 'No finished wars found' : 'No active wars found' }}</div>
        </div>
        </template>

        <!-- Eligible entities list -->
        <template v-if="isEligibleTab">
            <div v-if="eligiblePending && eligibleEntries.length === 0" class="flex items-center justify-center py-20">
                <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
            </div>
            <WarsEligibleTable v-else-if="eligibleEntries.length > 0"
                :entries="eligibleEntries"
                :type="eligibleType"
                :class="{ 'opacity-60': eligiblePending }" />
            <div v-else class="glass-panel py-12 text-center text-sm text-gray-500">
                {{ eligibleType === 'alliances' ? 'No war-eligible alliances found' : 'No war-eligible corporations found' }}
            </div>
        </template>

        <!-- Pagination (wars list only) -->
        <div v-if="isWarListTab && wars.length >= 50" class="flex justify-center mt-4 gap-2">
            <button @click="page = Math.max(1, page - 1)" :disabled="page <= 1"
                class="px-3 py-1.5 text-xs rounded-md transition-colors"
                :class="page <= 1 ? 'text-gray-700' : 'text-gray-400 hover:text-blue-400 bg-white/[0.04] border border-white/[0.08]'">
                Previous
            </button>
            <span class="px-3 py-1.5 text-xs text-gray-500">Page {{ page }}</span>
            <button @click="page++"
                class="px-3 py-1.5 text-xs rounded-md text-gray-400 hover:text-blue-400 bg-white/[0.04] border border-white/[0.08] transition-colors">
                Next
            </button>
        </div>
    </div>
</template>
