<script setup lang="ts">
const route = useRoute()
const id = Number(route.params.id)
const { isDomainMode } = useDomainConfig()

if (!Number.isInteger(id) || id < 1 || id > 2147483647) {
    throw createError({ statusCode: 404, statusMessage: 'System not found' })
}

const killlistEndpoint = computed(() => isDomainMode.value ? `/api/custom/system/${id}/killlist` : `/api/system/${id}/killlist`)

const { data, pending, error } = await useApiFetch<any>(`/api/system/${id}`)

if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'System not found',
    })
}

const system = computed(() => data.value?.system)
const celestials = computed(() => data.value?.celestials)
const celestialList = computed(() => data.value?.celestialList ?? [])
const stations = computed(() => data.value?.stations ?? [])
const structures = computed(() => data.value?.structures ?? [])
const connections = computed(() => data.value?.connections ?? [])
const sov = computed(() => data.value?.sovereignty)
const sovHistory = computed(() => data.value?.sovereigntyHistory ?? [])
const stats = computed(() => data.value?.stats)
const activity = computed(() => data.value?.activity)
const latestActivity = computed(() => activity.value?.latest)
const activity24h = computed(() => activity.value?.summary_24h)
const activityHistory = computed(() => activity.value?.history ?? [])

const securityColor = (sec: number | null): string => {
    if (sec == null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-green-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

const secLabel = (sec: number | null): string => {
    if (sec == null) return '?'
    return sec.toFixed(1)
}

const secBadgeBg = (sec: number | null): string => {
    if (sec == null) return 'bg-gray-600'
    if (sec >= 0.5) return 'bg-green-500/20 text-green-400'
    if (sec > 0.0) return 'bg-amber-500/20 text-amber-400'
    return 'bg-red-500/20 text-red-400'
}

useHead({ title: computed(() => system.value?.system_name || 'System') })

useSeoMeta({
    description: computed(() => {
        const s = system.value
        if (!s?.system_name) return 'View solar system kill statistics on EVE-KILL.'
        const st = stats.value
        const a = latestActivity.value
        let desc = `${s.system_name} (${secLabel(s.security)}) — ${s.constellation_name}, ${s.region_name}`
        if (a) desc += `. Last hour: ${a.ship_kills ?? 0} ship kills, ${a.pod_kills ?? 0} pod kills, ${a.ship_jumps ?? 0} jumps`
        if (st?.kills || st?.losses) desc += `. 7d: ${st.kills ?? 0} kills, ${st.losses ?? 0} losses`
        desc += ' — EVE-KILL.'
        return desc
    }),
    ogTitle: computed(() => {
        const s = system.value
        if (!s) return 'System — EVE-KILL'
        return `${s.system_name} (${secLabel(s.security)}) — ${s.region_name}`
    }),
    ogDescription: computed(() => {
        const s = system.value
        if (!s) return 'View system stats on EVE-KILL.'
        return `${s.system_name} (${secLabel(s.security)}) — kills and activity in ${s.region_name}, EVE Online.`
    }),
    ogType: 'website',
    twitterCard: 'summary',
    twitterTitle: computed(() => {
        const s = system.value
        if (!s) return 'System — EVE-KILL'
        return `${s.system_name} (${secLabel(s.security)}) — EVE-KILL`
    }),
    twitterDescription: computed(() => {
        const s = system.value
        if (!s) return 'View system stats on EVE-KILL.'
        return `${s.system_name} — ${s.constellation_name}, ${s.region_name} — kills and activity in EVE Online.`
    }),
    ogImage: `/images/systems/${id}`,
    twitterImage: `/images/systems/${id}`,
})

useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: (() => {
            const s = system.value
            if (!s) return [{ name: 'Home', item: '/' }]
            return [
                { name: 'Home', item: '/' },
                { name: s.region_name ?? 'Region', item: `/region/${s.region_id}` },
                { name: s.constellation_name ?? 'Constellation', item: `/constellation/${s.constellation_id}` },
                { name: s.system_name, item: `/system/${id}` },
            ]
        })(),
    }))),
    {
        '@type': 'Place',
        'name': computed(() => system.value?.system_name || 'Solar System'),
        'url': `https://eve-kill.com/system/${id}`,
        'image': `/images/systems/${id}`,
        'containedInPlace': computed(() => {
            const s = system.value
            if (!s?.region_name) return undefined
            return {
                '@type': 'Place',
                'name': s.region_name,
                'url': `https://eve-kill.com/region/${s.region_id}`,
            }
        }),
        'additionalProperty': computed(() => {
            const s = system.value
            if (!s) return undefined
            const props: { '@type': string; name: string; value: any }[] = []
            if (s.security != null) props.push({ '@type': 'PropertyValue', name: 'Security Status', value: secLabel(s.security) })
            if (s.constellation_name) props.push({ '@type': 'PropertyValue', name: 'Constellation', value: s.constellation_name })
            if (s.region_name) props.push({ '@type': 'PropertyValue', name: 'Region', value: s.region_name })
            return props.length ? props : undefined
        }),
    },
])

const tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
    { id: 'kills', label: 'Kills', icon: 'lucide:skull' },
    { id: 'battles', label: 'Battles', icon: 'lucide:swords' },
] as const

type TabId = typeof tabs[number]['id']
const tabIds = new Set(tabs.map(t => t.id))

useDefaultTab('system', `/system/${id}`, 'dashboard', tabIds)

definePageMeta({
    key: route => `/system/${route.params.id}`,
})

const activeTab = computed<TabId>(() => {
    const param = route.params.tab as string
    if (param && tabIds.has(param as TabId)) return param as TabId
    return 'dashboard'
})

const activeTabLabel = computed(() => {
    if (activeTab.value === 'dashboard') return null
    return tabs.find(t => t.id === activeTab.value)?.label ?? null
})
useHead({ title: computed(() => {
    const name = system.value?.system_name
    if (!name) return 'System'
    return activeTabLabel.value ? `${name} (${activeTabLabel.value})` : name
}) })

const setTab = (tabId: string) => {
    if (!tabIds.has(tabId as TabId)) return
    useAnalytics().track('tab.change', { entity: 'system', tab: tabId })
    navigateTo(tabId === 'dashboard' ? `/system/${id}` : `/system/${id}/${tabId}`)
}

const dotlanUrl = computed(() => {
    if (!system.value?.system_name) return '#'
    return `https://evemaps.dotlan.net/system/${encodeURIComponent(system.value.system_name)}`
})

const planets = computed(() => celestialList.value.filter((c: any) => c.category === 'planet'))
const belts = computed(() => celestialList.value.filter((c: any) => c.category === 'belt'))
const star = computed(() => celestialList.value.find((c: any) => c.category === 'star'))

// Build sov history as "from → to" transitions
const sovTransitions = computed(() => {
    const hist = sovHistory.value
    if (hist.length === 0) return []
    const transitions: any[] = []
    for (let i = 0; i < hist.length; i++) {
        const current = hist[i]
        const previous = i + 1 < hist.length ? hist[i + 1] : null
        transitions.push({
            to_name: current.alliance_name || current.corporation_name || current.faction_name || 'Unclaimed',
            to_id: current.alliance_id || current.corporation_id || current.faction_id,
            to_type: current.alliance_id ? 'alliance' : current.corporation_id ? 'corporation' : current.faction_id ? 'faction' : null,
            from_name: previous ? (previous.alliance_name || previous.corporation_name || previous.faction_name || 'Unclaimed') : null,
            from_id: previous ? (previous.alliance_id || previous.corporation_id || previous.faction_id) : null,
            from_type: previous ? (previous.alliance_id ? 'alliance' : previous.corporation_id ? 'corporation' : previous.faction_id ? 'faction' : null) : null,
            date: current.date,
        })
    }
    /*
     * The payload is a series of daily holder snapshots, not transitions, so
     * pairing consecutive entries emits a row per snapshot even when nothing
     * changed hands. Every highsec system therefore listed twenty rows of
     * "CONCORD Assembly → CONCORD Assembly" under a heading promising history.
     * A holder that did not change is not a transition.
     */
    return transitions.filter(t => t.from_id === null || t.from_id !== t.to_id)
})

const entityImage = (type: string | null, entityId: number | null): string => {
    if (!type || !entityId) return ''
    if (type === 'alliance') return `/images/alliances/${entityId}/logo?size=32`
    if (type === 'corporation') return `/images/corporations/${entityId}/logo?size=32`
    if (type === 'faction') return `/images/corporations/${entityId}/logo?size=32`
    return ''
}

const entityLink = (type: string | null, entityId: number | null): string | null => {
    if (!type || !entityId) return null
    return `/${type}/${entityId}`
}
</script>

<template>
    <div>
        <EntityHeader v-if="pending" loading />

        <div v-else-if="system">
            <EntityHeader>
                <template v-if="sov && (sov.alliance_id || sov.faction_id)" #image>
                    <EntityImageExpand v-if="sov.alliance_id" :full-src="`/images/alliances/${sov.alliance_id}/logo?size=256`" :alt="sov.alliance_name">
                        <NuxtLink :to="`/alliance/${sov.alliance_id}`">
                            <img :src="`/images/alliances/${sov.alliance_id}/logo?size=256`"
                                :alt="sov.alliance_name" class="w-32 h-32 md:w-40 md:h-40 rounded-lg shadow-lg" loading="eager">
                        </NuxtLink>
                    </EntityImageExpand>
                    <EntityImageExpand v-else-if="sov.faction_id" :full-src="`/images/corporations/${sov.faction_id}/logo?size=256`" :alt="sov.faction_name">
                        <img
                            :src="`/images/corporations/${sov.faction_id}/logo?size=256`"
                            :alt="sov.faction_name" class="w-32 h-32 md:w-40 md:h-40 rounded-lg shadow-lg" loading="eager">
                    </EntityImageExpand>
                </template>

                <div class="flex items-start justify-between">
                    <div>
                        <div class="flex items-center gap-3 mb-2 flex-wrap">
                            <h1 class="text-2xl md:text-3xl font-bold text-white" :class="pochvenClass(system.region_id)">{{ system.system_name }}</h1>
                            <span class="px-2.5 py-1 rounded-md text-sm font-bold tabular-nums" :class="secBadgeBg(system.security)">
                                {{ secLabel(system.security) }}
                            </span>
                            <template v-if="sov">
                                <NuxtLink v-if="sov.alliance_id" :to="`/alliance/${sov.alliance_id}`"
                                    class="px-2.5 py-1 rounded-md text-sm font-bold text-purple-400 hover:text-blue-400 transition-colors" style="background: rgba(168,85,247,0.1)">
                                    {{ sov.alliance_name }}
                                </NuxtLink>
                                <span v-else-if="sov.faction_name" class="px-2.5 py-1 rounded-md text-sm font-bold text-purple-400" style="background: rgba(168,85,247,0.1)">
                                    {{ sov.faction_name }}
                                </span>
                                <NuxtLink v-else-if="sov.corporation_id" :to="`/corporation/${sov.corporation_id}`"
                                    class="px-2.5 py-1 rounded-md text-sm font-bold text-purple-400 hover:text-blue-400 transition-colors" style="background: rgba(168,85,247,0.1)">
                                    {{ sov.corporation_name }}
                                </NuxtLink>
                            </template>
                        </div>

                        <div class="flex items-center gap-1.5 text-xs text-gray-500 mb-2 flex-wrap">
                            <NuxtLink to="/map" class="hover:text-blue-400 transition-colors">Map</NuxtLink>
                            <span class="text-gray-700">/</span>
                            <NuxtLink :to="`/region/${system.region_id}`" class="hover:text-blue-400 transition-colors" :class="pochvenClass(system.region_id)">{{ system.region_name }}</NuxtLink>
                            <span class="text-gray-700">/</span>
                            <NuxtLink :to="`/constellation/${system.constellation_id}`" class="hover:text-blue-400 transition-colors" :class="pochvenClass(system.region_id)">{{ system.constellation_name }}</NuxtLink>
                            <span class="text-gray-700">/</span>
                            <span class="text-gray-400" :class="pochvenClass(system.region_id)">{{ system.system_name }}</span>
                        </div>
                        <div v-if="system.sun_type_name" class="text-xs text-gray-500 mt-1">
                            <Icon name="lucide:sun" class="w-3.5 h-3.5 inline text-yellow-500/60" />
                            {{ system.sun_type_name }}
                        </div>
                    </div>

                    <!-- Desktop: external links inline -->
                    <div class="hidden md:flex flex-col gap-1.5 flex-shrink-0">
                        <a :href="dotlanUrl" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/dotlan.png" width="16" height="16" class="w-4 h-4" alt="" /> DOTLAN</a>
                        <a :href="`https://eveeye.com/?m=${encodeURIComponent(system.region_name || '')}`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/eveeye.svg" width="16" height="16" class="w-4 h-4" alt="" /> EVEEye</a>
                        <a :href="`https://evemissioneer.com/s/${id}`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/evemissioneer.png" width="16" height="16" class="w-4 h-4" alt="" /> Missioneer</a>
                        <a :href="`https://www.jita.space/system/${id}`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/jita-space.png" width="16" height="16" class="w-4 h-4" alt="" /> Jita.Space</a>
                        <a :href="`https://zkillboard.com/system/${id}/`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/zkillboard.png" width="16" height="16" class="w-4 h-4" alt="" /> zKillboard</a>
                    </div>
                </div>

                <!-- Mobile: external links next to image -->
                <template #right>
                    <div class="md:hidden flex flex-col gap-1.5 flex-shrink-0">
                        <a :href="dotlanUrl" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/dotlan.png" width="16" height="16" class="w-4 h-4" alt="" /> DOTLAN</a>
                        <a :href="`https://eveeye.com/?m=${encodeURIComponent(system.region_name || '')}`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/eveeye.svg" width="16" height="16" class="w-4 h-4" alt="" /> EVEEye</a>
                        <a :href="`https://evemissioneer.com/s/${id}`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/evemissioneer.png" width="16" height="16" class="w-4 h-4" alt="" /> Missioneer</a>
                        <a :href="`https://www.jita.space/system/${id}`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/jita-space.png" width="16" height="16" class="w-4 h-4" alt="" /> Jita.Space</a>
                        <a :href="`https://zkillboard.com/system/${id}/`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/zkillboard.png" width="16" height="16" class="w-4 h-4" alt="" /> zKillboard</a>
                    </div>
                </template>

                <template #stats>
                    <EntityStatGrid variant="inline" :columns="8">
                        <!-- Live ESI stats (last hour) -->
                        <div v-if="latestActivity">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Jumps (1h)</div>
                            <div class="text-base font-bold text-cyan-400 tabular-nums">{{ latestActivity.ship_jumps.toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="latestActivity">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Ship Kills (1h)</div>
                            <div class="text-base font-bold text-green-400 tabular-nums">{{ latestActivity.ship_kills.toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="latestActivity">
                            <div class="text-fine uppercase tracking-wider text-gray-500">NPC Kills (1h)</div>
                            <div class="text-base font-bold text-white tabular-nums">{{ latestActivity.npc_kills.toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="latestActivity">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Pod Kills (1h)</div>
                            <div class="text-base font-bold text-red-400 tabular-nums">{{ latestActivity.pod_kills.toLocaleString('en-US') }}</div>
                        </div>
                        <!-- 7d stats -->
                        <div v-if="stats">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Kills (7d)</div>
                            <div class="text-base font-bold text-green-400 tabular-nums">{{ Number(stats.kills).toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="stats">
                            <div class="text-fine uppercase tracking-wider text-gray-500">ISK (7d)</div>
                            <div class="text-base font-bold text-yellow-400 tabular-nums">{{ formatIsk(Number(stats.total_value)) }}</div>
                        </div>
                        <!-- System info -->
                        <div v-if="celestials">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Planets</div>
                            <div class="text-base font-bold text-blue-400 tabular-nums">{{ celestials.planets }}</div>
                        </div>
                        <div>
                            <div class="text-fine uppercase tracking-wider text-gray-500">Gates</div>
                            <div class="text-base font-bold text-gray-300 tabular-nums">{{ connections.length }}</div>
                        </div>
                    </EntityStatGrid>
                </template>
            </EntityHeader>

            <MostValuable :api-endpoint="`/api/system/${id}/most-valuable`" />

            <!-- TAB BAR -->
            <EntityTabBar :tabs="tabs" :active-id="activeTab" @select="setTab" />

            <!-- DASHBOARD TAB -->
            <div v-if="activeTab === 'dashboard'">
                <div class="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4">
                    <!-- LEFT column -->
                    <div class="space-y-4">
                        <!-- Hourly Activity Breakdown -->
                        <SystemActivityChart v-if="activityHistory.length > 1"
                            :history="activityHistory" :summary="activity24h" />

                        <!-- Sovereignty History -->
                        <div v-if="sovTransitions.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:history" class="w-4 h-4 text-purple-400" />
                                <span class="text-xs font-semibold text-gray-300">Sovereignty History</span>
                            </div>
                            <div class="space-y-0">
                                <div v-for="(t, idx) in sovTransitions" :key="idx"
                                    class="flex items-center gap-3 py-2" :class="idx > 0 ? 'border-t border-white/[0.04]' : ''">
                                    <!-- From -->
                                    <div class="flex items-center gap-1.5 min-w-0 flex-1">
                                        <template v-if="t.from_id">
                                            <img :src="entityImage(t.from_type, t.from_id)" class="w-4 h-4 rounded flex-shrink-0" loading="lazy">
                                            <NuxtLink v-if="entityLink(t.from_type, t.from_id)" :to="entityLink(t.from_type, t.from_id)!" class="text-xs text-gray-500 truncate hover:text-blue-400 transition-colors">{{ t.from_name }}</NuxtLink>
                                            <span v-else class="text-xs text-gray-500 truncate">{{ t.from_name }}</span>
                                        </template>
                                        <span v-else class="text-xs text-gray-600 italic">initial</span>
                                    </div>
                                    <!-- Arrow -->
                                    <Icon name="lucide:arrow-right" class="w-3 h-3 text-gray-600 flex-shrink-0" />
                                    <!-- To -->
                                    <div class="flex items-center gap-1.5 min-w-0 flex-1">
                                        <img v-if="t.to_id" :src="entityImage(t.to_type, t.to_id)" class="w-4 h-4 rounded flex-shrink-0" loading="lazy">
                                        <NuxtLink v-if="entityLink(t.to_type, t.to_id)" :to="entityLink(t.to_type, t.to_id)!" class="text-xs text-gray-300 truncate hover:text-blue-400 transition-colors">{{ t.to_name }}</NuxtLink>
                                        <span v-else class="text-xs text-gray-300 truncate">{{ t.to_name }}</span>
                                    </div>
                                    <!-- Date -->
                                    <span class="text-fine text-gray-600 tabular-nums flex-shrink-0">{{ formatDate(t.date) }}</span>
                                </div>
                            </div>
                        </div>

                        <!-- Stations -->
                        <div v-if="stations.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:building" class="w-4 h-4 text-emerald-500" />
                                <span class="text-xs font-semibold text-gray-300">Stations</span>
                                <span class="text-xs text-gray-600">({{ stations.length }})</span>
                            </div>
                            <div class="space-y-2">
                                <div v-for="st in stations" :key="st.station_id" class="px-1">
                                    <div class="text-fine text-gray-300">{{ st.station_name }}</div>
                                    <div class="flex items-center gap-2 text-fine text-gray-600">
                                        <span v-if="st.operation_name">{{ st.operation_name }}</span>
                                        <NuxtLink v-if="st.corporation_id" :to="`/corporation/${st.corporation_id}`" class="hover:text-gray-400 transition-colors">
                                            {{ st.operation_name ? '·' : '' }} {{ st.corporation_name }}
                                        </NuxtLink>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- Player Structures -->
                        <div v-if="structures.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:landmark" class="w-4 h-4 text-orange-500" />
                                <span class="text-xs font-semibold text-gray-300">Structures</span>
                                <span class="text-xs text-gray-600">({{ structures.length }})</span>
                            </div>
                            <div class="space-y-2">
                                <div v-for="s in structures" :key="s.structure_id" class="px-1">
                                    <div class="flex items-center gap-2">
                                        <span class="text-fine text-gray-300 truncate">{{ s.name || 'Unknown Structure' }}</span>
                                        <span v-if="s.is_market" class="text-fine px-1.5 py-0.5 rounded bg-green-500/20 text-green-400">market</span>
                                    </div>
                                    <div class="flex items-center gap-2 text-fine text-gray-600">
                                        <span v-if="s.type_name">{{ s.type_name }}</span>
                                        <NuxtLink v-if="s.owner_id" :to="`/corporation/${s.owner_id}`" class="hover:text-gray-400 transition-colors">
                                            · {{ s.owner_name || `Corp #${s.owner_id}` }}
                                        </NuxtLink>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- RIGHT column -->
                    <div class="space-y-4">
                        <!-- Connected Systems -->
                        <div v-if="connections.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:git-branch" class="w-4 h-4 text-cyan-500" />
                                <span class="text-xs font-semibold text-gray-300">Connected Systems</span>
                                <span class="text-xs text-gray-600">({{ connections.length }})</span>
                            </div>
                            <div class="space-y-1">
                                <NuxtLink v-for="conn in connections" :key="conn.to_solar_system_id"
                                    :to="`/system/${conn.to_solar_system_id}`"
                                    class="flex items-center gap-2 px-2 py-1 rounded-md hover:bg-blue-500/[0.06] transition-colors">
                                    <span class="text-xs font-medium tabular-nums w-8" :class="securityColor(conn.security)">
                                        {{ secLabel(conn.security) }}
                                    </span>
                                    <span class="text-fine text-gray-300 truncate" :class="pochvenClass(conn.region_id)">{{ conn.system_name }}</span>
                                    <span v-if="conn.is_regional" class="ml-auto text-fine px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-400">regional</span>
                                </NuxtLink>
                            </div>
                        </div>

                        <!-- Celestials -->
                        <div v-if="planets.length || belts.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:globe" class="w-4 h-4 text-blue-500" />
                                <span class="text-xs font-semibold text-gray-300">Celestials</span>
                            </div>
                            <div v-if="star" class="flex items-center gap-2 mb-2 pb-2 border-b border-white/[0.04]">
                                <div class="w-2 h-2 rounded-full bg-yellow-500"></div>
                                <span class="text-xs text-yellow-400">{{ star.type_name || star.item_name || 'Star' }}</span>
                            </div>
                            <div class="space-y-1.5">
                                <div v-for="planet in planets" :key="planet.item_id" class="flex items-center gap-2">
                                    <div class="w-2 h-2 rounded-full bg-blue-400 flex-shrink-0"></div>
                                    <span class="text-xs text-gray-300">{{ planet.item_name }}</span>
                                    <span v-if="planet.type_name" class="text-fine text-gray-600">{{ planet.type_name }}</span>
                                </div>
                            </div>
                            <div v-if="belts.length" class="mt-2 pt-2 border-t border-white/[0.04]">
                                <div class="text-fine text-gray-600">{{ belts.length }} Asteroid Belt{{ belts.length !== 1 ? 's' : '' }}</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- KILLS TAB -->
            <div v-if="activeTab === 'kills'">
                <KillList :entity-endpoint="killlistEndpoint" />
            </div>

            <div v-if="activeTab === 'battles'">
                <EntityBattles :system-id="id" />
            </div>
        </div>
    </div>
</template>
