<script setup lang="ts">
const route = useRoute()
const id = Number(route.params.id)

if (!Number.isInteger(id) || id < 1 || id > 2147483647) throw createError({ statusCode: 404, statusMessage: 'War not found' })

const { data, pending, error } = await useApiFetch<any>(`/api/war/${id}`)

if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'War not found',
    })
}
// Stats load lazily — header + killlist render immediately, sidebar fills in after
const { data: statsData, pending: statsPending, refresh: loadStats } = useApiFetch<any>(`/api/war/${id}/stats`, {
    server: false,
    immediate: false,
    getCachedData: cachedPayload,
})
onMounted(() => { void loadStats() })

const war = computed(() => data.value?.war)
const warStats = computed(() => data.value?.stats)
const sidebarStats = computed(() => statsData.value)

// Derive sides from war data (available immediately) instead of waiting for heavy stats endpoint
const sides = computed(() => {
    const w = war.value
    if (!w) return null
    const aggressorCorps: number[] = []
    const aggressorAlliances: number[] = []
    const defenderCorps: number[] = []
    const defenderAlliances: number[] = []
    if (w.aggressor?.type === 'alliance') aggressorAlliances.push(w.aggressor.id)
    else if (w.aggressor?.type === 'corporation') aggressorCorps.push(w.aggressor.id)
    if (w.defender?.type === 'alliance') defenderAlliances.push(w.defender.id)
    else if (w.defender?.type === 'corporation') defenderCorps.push(w.defender.id)
    for (const ally of w.allies || []) {
        if (ally.type === 'alliance') defenderAlliances.push(ally.id)
        else if (ally.type === 'corporation') defenderCorps.push(ally.id)
    }
    return {
        aggressor: { corporations: aggressorCorps, alliances: aggressorAlliances },
        defender: { corporations: defenderCorps, alliances: defenderAlliances },
    }
})

// Single tab state — hash stores the active tab. 'combined' is the default and
// omits the hash. The three "side" tabs and the two view tabs share one bar.
type WarTab = 'combined' | 'aggressor' | 'defender' | 'members' | 'intel'
const validWarTabs = new Set<WarTab>(['combined', 'aggressor', 'defender', 'members', 'intel'])
const warTabFromHash = (hash: string): WarTab => {
    const t = hash.replace('#', '') as WarTab
    return validWarTabs.has(t) ? t : 'combined'
}
const activeTab = ref<WarTab>('combined')
const warTabLabels: Record<WarTab, string> = { combined: '', aggressor: 'Aggressor', defender: 'Defender', members: 'Members', intel: 'Intel' }

// For Overview side filtering, derive the side from the tab (falling back to
// combined for Members/Intel so the KillList sidebar never sees an invalid side).
const overviewSide = computed<'combined' | 'aggressor' | 'defender'>(() => {
    if (activeTab.value === 'aggressor' || activeTab.value === 'defender') return activeTab.value
    return 'combined'
})

useHead({ title: computed(() => {
    const w = war.value
    if (!w) return 'War'
    let t = w.aggressor?.name && w.defender?.name
        ? `${w.aggressor.name} vs ${w.defender.name} — War #${w.war_id}`
        : `War #${w.war_id}`
    const label = warTabLabels[activeTab.value]
    if (label) t += ` (${label})`
    return t
}) })
useSeoMeta({
    description: computed(() => {
        const w = war.value
        if (!w) return 'View war details on EVE-KILL.'
        let desc = `War #${w.war_id}`
        if (w.aggressor?.name && w.defender?.name) desc += ` — ${w.aggressor.name} vs ${w.defender.name}`
        if (w.mutual) desc += ' (mutual)'
        desc += w.finished ? '. Finished' : '. Active'
        if (w.started) desc += `, started ${new Date(w.started).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })}`
        const totalIsk = (w.aggressor?.isk_destroyed || 0) + (w.defender?.isk_destroyed || 0)
        const totalShips = (w.aggressor?.ships_killed || 0) + (w.defender?.ships_killed || 0)
        if (totalShips) desc += `. ${totalShips} ships destroyed`
        if (totalIsk) desc += `, ${formatIsk(totalIsk)} ISK destroyed`
        desc += ' — EVE-KILL.'
        return desc
    }),
    ogTitle: computed(() => {
        const w = war.value
        if (!w) return 'War — EVE-KILL'
        if (w.aggressor?.name && w.defender?.name) return `${w.aggressor.name} vs ${w.defender.name} — War #${w.war_id}`
        return `War #${w.war_id} — EVE-KILL`
    }),
    ogDescription: computed(() => {
        const w = war.value
        if (!w) return 'View war details on EVE-KILL.'
        let desc = `War #${w.war_id}`
        if (w.aggressor?.name && w.defender?.name) desc += ` — ${w.aggressor.name} vs ${w.defender.name}`
        desc += w.finished ? '. Finished' : '. Active'
        desc += ' — kills, ISK destroyed, and timeline on EVE-KILL.'
        return desc
    }),
    ogType: 'website',
    twitterCard: 'summary',
    twitterTitle: computed(() => {
        const w = war.value
        if (!w) return 'War — EVE-KILL'
        if (w.aggressor?.name && w.defender?.name) return `${w.aggressor.name} vs ${w.defender.name} — War #${w.war_id} — EVE-KILL`
        return `War #${w.war_id} — EVE-KILL`
    }),
    twitterDescription: computed(() => {
        const w = war.value
        if (!w) return 'View war details on EVE-KILL.'
        let desc = `War #${w.war_id}`
        if (w.aggressor?.name && w.defender?.name) desc += ` — ${w.aggressor.name} vs ${w.defender.name}`
        desc += w.finished ? '. Finished' : '. Active'
        desc += ' — kills, ISK destroyed, and timeline on EVE-KILL.'
        return desc
    }),
    ogImage: computed(() => {
        const a = war.value?.aggressor
        if (a?.type === 'alliance' && a.id) return `/images/alliances/${a.id}/logo?size=256`
        if (a?.type === 'corporation' && a.id) return `/images/corporations/${a.id}/logo?size=256`
        return 'https://eve-kill.com/og-default.png'
    }),
    twitterImage: computed(() => {
        const a = war.value?.aggressor
        if (a?.type === 'alliance' && a.id) return `/images/alliances/${a.id}/logo?size=256`
        if (a?.type === 'corporation' && a.id) return `/images/corporations/${a.id}/logo?size=256`
        return 'https://eve-kill.com/og-default.png'
    }),
})

useSchemaOrg([
    defineBreadcrumb({
        itemListElement: computed(() => [
            { name: 'Home', item: '/' },
            { name: 'Wars', item: '/wars' },
            { name: war.value ? `War #${war.value.war_id}` : 'War', item: `/war/${id}` },
        ]),
    }),
    {
        '@type': 'Event',
        'name': computed(() => {
            const w = war.value
            if (!w) return 'War'
            if (w.aggressor?.name && w.defender?.name) return `${w.aggressor.name} vs ${w.defender.name} — War #${w.war_id}`
            return `War #${w.war_id}`
        }),
        'url': `https://eve-kill.com/war/${id}`,
        'startDate': computed(() => war.value?.started || undefined),
        'endDate': computed(() => war.value?.finished || undefined),
        'eventStatus': computed(() => war.value?.finished
            ? 'https://schema.org/EventCancelled'
            : 'https://schema.org/EventScheduled'),
        'eventAttendanceMode': 'https://schema.org/OnlineEventAttendanceMode',
        'location': {
            '@type': 'VirtualLocation',
            'url': 'https://www.eveonline.com',
        },
        'organizer': computed(() => {
            const a = war.value?.aggressor
            if (!a?.name) return undefined
            const path = a.type === 'alliance' ? 'alliance' : 'corporation'
            return {
                '@type': 'Organization',
                'name': a.name,
                'url': `https://eve-kill.com/${path}/${a.id}`,
            }
        }),
        'attendee': computed(() => {
            const w = war.value
            if (!w) return undefined
            const list: { '@type': string; name: string; url: string }[] = []
            if (w.aggressor?.name) {
                const p = w.aggressor.type === 'alliance' ? 'alliance' : 'corporation'
                list.push({ '@type': 'Organization', name: w.aggressor.name, url: `https://eve-kill.com/${p}/${w.aggressor.id}` })
            }
            if (w.defender?.name) {
                const p = w.defender.type === 'alliance' ? 'alliance' : 'corporation'
                list.push({ '@type': 'Organization', name: w.defender.name, url: `https://eve-kill.com/${p}/${w.defender.id}` })
            }
            return list.length ? list : undefined
        }),
    },
])

// Read hash on client after hydration to avoid mismatch
if (import.meta.client) {
    onMounted(() => { activeTab.value = warTabFromHash(window.location.hash) })
}

const setWarTab = (tab: WarTab) => {
    if (activeTab.value === tab) return
    activeTab.value = tab
    const url = new URL(window.location.href)
    url.hash = tab === 'combined' ? '' : `#${tab}`
    window.history.pushState(null, '', url.toString())
}
if (import.meta.client) {
    window.addEventListener('popstate', () => {
        activeTab.value = warTabFromHash(window.location.hash)
    })
}

const sideColorFor = (entry: any): string => {
    if (overviewSide.value === 'aggressor') return 'text-red-400'
    if (overviewSide.value === 'defender') return 'text-blue-400'
    // Combined: use per-entry side
    if (entry.side === 'aggressor') return 'text-red-400'
    if (entry.side === 'defender') return 'text-blue-400'
    return 'text-gray-400'
}

const sideDotFor = (entry: any): string => {
    if (overviewSide.value === 'aggressor') return 'bg-red-500'
    if (overviewSide.value === 'defender') return 'bg-blue-500'
    if (entry.side === 'aggressor') return 'bg-red-500'
    if (entry.side === 'defender') return 'bg-blue-500'
    return 'bg-gray-500'
}

// Per-side stats — server returns separate lists for each side
const currentSideStats = computed(() => {
    if (!sidebarStats.value) return null
    if (overviewSide.value === 'aggressor') return sidebarStats.value.aggressor
    if (overviewSide.value === 'defender') return sidebarStats.value.defender
    return sidebarStats.value.combined
})

const filteredChars = computed(() => currentSideStats.value?.topCharacters || [])
const filteredCorps = computed(() => currentSideStats.value?.topCorporations || [])
const filteredAlliances = computed(() => currentSideStats.value?.topAlliances || [])
const filteredShips = computed(() => sidebarStats.value?.topShips || [])

// KillList side filter params — when on a specific side tab, filter kills where the VICTIM is the OTHER side
const killListSideCorps = computed(() => {
    if (!sides.value || overviewSide.value === 'combined') return ''
    // Aggressor tab = show kills where victim is defender (aggressor's kills)
    const victimSide = overviewSide.value === 'aggressor' ? sides.value.defender : sides.value.aggressor
    return victimSide.corporations.join(',')
})

const killListSideAlliances = computed(() => {
    if (!sides.value || overviewSide.value === 'combined') return ''
    const victimSide = overviewSide.value === 'aggressor' ? sides.value.defender : sides.value.aggressor
    return victimSide.alliances.join(',')
})

const entityImage = (entity: any, size = 128): string => {
    if (entity.type === 'alliance') return `/images/alliances/${entity.id}/logo?size=${size}`
    return `/images/corporations/${entity.id}/logo?size=${size}`
}

const entityLink = (entity: any): string => `/${entity.type}/${entity.id}`

const aggressorEfficiency = computed(() => {
    const w = war.value
    if (!w) return 50
    const total = (w.aggressor.isk_destroyed || 0) + (w.defender.isk_destroyed || 0)
    if (total === 0) return 50
    return Math.round((w.aggressor.isk_destroyed / total) * 100)
})

// Corp/alliance filter options for the Members view — derived from war parties + allies
const memberEntityOptions = computed(() => {
    const w = war.value
    if (!w) return [] as Array<{ id: number, name: string, type: 'corporation' | 'alliance', side: 'aggressor' | 'defender' }>
    const opts: Array<{ id: number, name: string, type: 'corporation' | 'alliance', side: 'aggressor' | 'defender' }> = []
    if (w.aggressor?.id && w.aggressor.type) {
        opts.push({ id: w.aggressor.id, name: w.aggressor.name, type: w.aggressor.type, side: 'aggressor' })
    }
    if (w.defender?.id && w.defender.type) {
        opts.push({ id: w.defender.id, name: w.defender.name, type: w.defender.type, side: 'defender' })
    }
    for (const ally of (w.allies || [])) {
        if (ally.id && ally.type) opts.push({ id: ally.id, name: ally.name, type: ally.type, side: 'defender' })
    }
    return opts
})
</script>

<template>
    <div>
        <div v-if="pending" class="h-64 rounded-lg bg-white/[0.04] animate-pulse"></div>

        <div v-else-if="war">
            <!-- Header -->
            <div class="glass-panel p-6 mb-6">
                <div class="flex items-center justify-between mb-4">
                    <div class="flex items-center gap-3">
                        <h1 class="text-2xl font-bold text-white">War #{{ war.war_id }}</h1>
                        <span v-if="!war.finished" class="px-2 py-0.5 text-xs rounded bg-green-500/20 text-green-400 font-medium">Active</span>
                        <span v-else class="px-2 py-0.5 text-xs rounded bg-gray-500/20 text-gray-400 font-medium">Finished</span>
                        <span v-if="war.mutual" class="px-2 py-0.5 text-xs rounded bg-amber-500/20 text-amber-400 font-medium">Mutual</span>
                        <span v-if="war.open_for_allies" class="px-2 py-0.5 text-xs rounded bg-blue-500/20 text-blue-400 font-medium">Open to Allies</span>
                    </div>
                </div>

                <div class="flex flex-wrap gap-6 text-xs text-gray-400 mb-6">
                    <div v-if="war.declared"><span class="text-gray-500">Declared:</span> {{ formatDate(war.declared) }}</div>
                    <div v-if="war.started"><span class="text-gray-500">Started:</span> {{ formatDate(war.started) }}</div>
                    <div v-if="war.finished"><span class="text-gray-500">Finished:</span> {{ formatDate(war.finished) }}</div>
                </div>

                <!-- Aggressor vs Defender -->
                <div class="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] gap-6 items-center">
                    <NuxtLink :to="entityLink(war.aggressor)"
                        class="flex items-center gap-4 p-4 rounded-lg bg-red-500/[0.04] border border-red-500/20 hover:bg-red-500/[0.08] transition-colors">
                        <img :src="entityImage(war.aggressor)" class="w-16 h-16 rounded-lg" loading="eager">
                        <div class="min-w-0">
                            <div class="text-fine uppercase tracking-wider text-red-400/80 mb-1">Aggressor</div>
                            <div class="text-base font-bold text-white truncate">{{ war.aggressor.name }}</div>
                            <div class="text-xs text-gray-400 mt-1">
                                <span class="text-green-400 tabular-nums">{{ formatIsk(war.aggressor.isk_destroyed) }}</span> destroyed
                                <span class="text-gray-600 mx-1">&middot;</span>
                                <span class="tabular-nums">{{ formatNumber(war.aggressor.ships_killed) }}</span> ships
                            </div>
                        </div>
                    </NuxtLink>

                    <div class="flex flex-col items-center gap-2 px-4">
                        <div class="text-base font-bold text-gray-600">VS</div>
                        <div class="w-32 h-2 bg-blue-500/30 rounded-full overflow-hidden">
                            <div class="h-full bg-red-500 rounded-full" :style="{ width: `${aggressorEfficiency}%` }"></div>
                        </div>
                        <div class="text-fine text-gray-500 tabular-nums">{{ aggressorEfficiency }}% — {{ 100 - aggressorEfficiency }}%</div>
                    </div>

                    <NuxtLink :to="entityLink(war.defender)"
                        class="flex items-center gap-4 p-4 rounded-lg bg-blue-500/[0.04] border border-blue-500/20 hover:bg-blue-500/[0.08] transition-colors">
                        <img :src="entityImage(war.defender)" class="w-16 h-16 rounded-lg" loading="eager">
                        <div class="min-w-0">
                            <div class="text-fine uppercase tracking-wider text-blue-400/80 mb-1">Defender</div>
                            <div class="text-base font-bold text-white truncate">{{ war.defender.name }}</div>
                            <div class="text-xs text-gray-400 mt-1">
                                <span class="text-green-400 tabular-nums">{{ formatIsk(war.defender.isk_destroyed) }}</span> destroyed
                                <span class="text-gray-600 mx-1">&middot;</span>
                                <span class="tabular-nums">{{ formatNumber(war.defender.ships_killed) }}</span> ships
                            </div>
                        </div>
                    </NuxtLink>
                </div>

                <!-- Stats summary -->
                <div v-if="warStats" class="grid grid-cols-3 gap-4 mt-6 pt-4 border-t border-white/[0.06]">
                    <div class="text-center">
                        <div class="text-base font-bold text-white tabular-nums">{{ formatNumber(warStats.total_kills) }}</div>
                        <div class="text-fine uppercase tracking-wider text-gray-500">Total Kills</div>
                    </div>
                    <div class="text-center">
                        <div class="text-base font-bold text-green-400 tabular-nums">{{ formatIsk(warStats.total_value) }}</div>
                        <div class="text-fine uppercase tracking-wider text-gray-500">ISK Destroyed</div>
                    </div>
                    <div class="text-center">
                        <div class="text-base font-bold text-white tabular-nums">{{ war.allies?.length || 0 }}</div>
                        <div class="text-fine uppercase tracking-wider text-gray-500">Allies</div>
                    </div>
                </div>
            </div>

            <!-- Allies -->
            <div v-if="war.allies?.length > 0" class="glass-panel p-4 mb-6">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Allies (defending) — {{ war.allies.length }}</div>
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                    <NuxtLink v-for="ally in war.allies" :key="`${ally.type}-${ally.id}`" :to="`/${ally.type}/${ally.id}`"
                        class="flex items-center gap-3 px-3 py-2 rounded-md bg-white/[0.03] border border-blue-500/20 hover:bg-blue-500/[0.06] transition-colors">
                        <img :src="ally.type === 'alliance' ? `/images/alliances/${ally.id}/logo?size=64` : `/images/corporations/${ally.id}/logo?size=64`"
                            class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                        <div class="min-w-0 flex-1">
                            <div class="text-xs text-blue-300 truncate">{{ ally.name }}</div>
                            <div class="text-fine text-gray-500 tabular-nums mt-0.5">
                                <span class="text-green-400/80">{{ formatNumber(ally.kills || 0) }} kills</span>
                                <span class="text-gray-600 mx-1">·</span>
                                <span class="text-red-400/80">{{ formatNumber(ally.losses || 0) }} losses</span>
                                <span v-if="ally.isk_destroyed > 0" class="text-gray-600 mx-1">·</span>
                                <span v-if="ally.isk_destroyed > 0" class="text-isk/70">{{ formatIsk(ally.isk_destroyed) }}</span>
                            </div>
                        </div>
                    </NuxtLink>
                </div>
            </div>

            <!-- Tab bar — sides + views share one row -->
            <div class="flex overflow-x-auto border-b border-white/[0.08] mb-4 scrollbar-hide">
                <button v-for="tab in [
                    { id: 'combined', label: 'Both Sides', icon: 'lucide:layers', color: '', border: 'border-white' },
                    { id: 'aggressor', label: war.aggressor.name, icon: 'lucide:swords', color: 'text-red-400', border: 'border-red-400' },
                    { id: 'defender', label: war.defender.name, icon: 'lucide:shield', color: 'text-blue-400', border: 'border-blue-400' },
                    { id: 'members', label: 'Members', icon: 'lucide:users', color: '', border: 'border-white' },
                    { id: 'intel', label: 'Intel', icon: 'lucide:radar', color: '', border: 'border-white' },
                ]" :key="tab.id"
                    class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 whitespace-nowrap"
                    :class="activeTab === tab.id
                        ? `text-white ${tab.border}`
                        : 'text-gray-500 border-transparent hover:text-blue-400'"
                    @click="setWarTab(tab.id as any)">
                    <Icon :name="tab.icon" class="text-base" :class="activeTab === tab.id ? (tab.color || '') : ''" />
                    <span class="truncate max-w-[150px]" :class="activeTab === tab.id || !tab.color ? '' : 'hidden md:inline'">{{ tab.label }}</span>
                </button>
            </div>

            <!-- Members view -->
            <WarMembers v-if="activeTab === 'members'"
                :endpoint="`/api/war/${id}/members`"
                :entity-options="memberEntityOptions"
                :key="`war-members-${id}`" />

            <!-- Intel view -->
            <WarIntel v-else-if="activeTab === 'intel'"
                :endpoint="`/api/war/${id}/intel`"
                :key="`war-intel-${id}`" />

            <!-- Overview view: sidebar + killlist -->
            <div v-else class="grid grid-cols-1 md:grid-cols-[250px_1fr] gap-4">
                <!-- Sidebar -->
                <div class="space-y-3">
                    <!-- Loading state -->
                    <div v-if="statsPending" class="flex flex-col items-center justify-center py-8 gap-2">
                        <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                        <span class="text-xs text-gray-600">Loading stats...</span>
                    </div>

                    <!-- Top Characters -->
                    <div v-if="!statsPending && filteredChars.length > 0" class="glass-panel p-2">
                        <div class="px-1 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                            Top Characters
                        </div>
                        <div class="space-y-px">
                            <NuxtLink v-for="(e, idx) in filteredChars" :key="e.id" :to="`/character/${e.id}`"
                                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <div class="w-1.5 h-1.5 rounded-full flex-shrink-0" :class="sideDotFor(e)"></div>
                                <img :src="`/images/characters/${e.id}/portrait?size=64`" class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                                <span class="flex-1 text-xs truncate">{{ e.name }}</span>
                                <span class="text-xs tabular-nums" :class="sideColorFor(e)">{{ e.kills }}</span>
                            </NuxtLink>
                        </div>
                    </div>

                    <!-- Top Corporations -->
                    <div v-if="!statsPending && filteredCorps.length > 0" class="glass-panel p-2">
                        <div class="px-1 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                            Top Corporations
                        </div>
                        <div class="space-y-px">
                            <NuxtLink v-for="(e, idx) in filteredCorps" :key="e.id" :to="`/corporation/${e.id}`"
                                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <div class="w-1.5 h-1.5 rounded-full flex-shrink-0" :class="sideDotFor(e)"></div>
                                <img :src="`/images/corporations/${e.id}/logo?size=64`" class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                                <span class="flex-1 text-xs truncate">{{ e.name }}</span>
                                <span class="text-xs tabular-nums" :class="sideColorFor(e)">{{ e.kills }}</span>
                            </NuxtLink>
                        </div>
                    </div>

                    <!-- Top Alliances -->
                    <div v-if="!statsPending && filteredAlliances.length > 0" class="glass-panel p-2">
                        <div class="px-1 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                            Top Alliances
                        </div>
                        <div class="space-y-px">
                            <NuxtLink v-for="(e, idx) in filteredAlliances" :key="e.id" :to="`/alliance/${e.id}`"
                                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <div class="w-1.5 h-1.5 rounded-full flex-shrink-0" :class="sideDotFor(e)"></div>
                                <img :src="`/images/alliances/${e.id}/logo?size=64`" class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                                <span class="flex-1 text-xs truncate">{{ e.name }}</span>
                                <span class="text-xs tabular-nums" :class="sideColorFor(e)">{{ e.kills }}</span>
                            </NuxtLink>
                        </div>
                    </div>

                    <!-- Top Ships -->
                    <div v-if="!statsPending && filteredShips.length > 0" class="glass-panel p-2">
                        <div class="px-1 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                            Top Ships Destroyed
                        </div>
                        <div class="space-y-px">
                            <div v-for="(ship, idx) in filteredShips" :key="ship.ship_type_id"
                                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400">
                                <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(idx) + 1 }}</span>
                                <img :src="`/images/types/${ship.ship_type_id}/icon?size=64`"
                                    class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                                <span class="flex-1 text-xs truncate">{{ ship.ship_name }}</span>
                                <span class="text-xs text-gray-500 tabular-nums">{{ ship.count }}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Kill list — filtered by tab, colored by side -->
                <KillList
                    :api-endpoint="`/api/war/${id}/killlist`"
                    :extra-params="{
                        warStart: war.started || war.declared,
                        warEnd: war.finished,
                        warSideCorps: killListSideCorps,
                        warSideAlliances: killListSideAlliances,
                    }"
                    :war-aggressor-corps="sides?.aggressor?.corporations"
                    :war-aggressor-alliances="sides?.aggressor?.alliances"
                    :war-defender-corps="sides?.defender?.corporations"
                    :war-defender-alliances="sides?.defender?.alliances"
                    :key="`war-${id}-${overviewSide}`"
                />
            </div>
        </div>
    </div>
</template>
