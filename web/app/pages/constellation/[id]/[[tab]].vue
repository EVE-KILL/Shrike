<script setup lang="ts">
const route = useRoute()
const id = Number(route.params.id)
const { isDomainMode } = useDomainConfig()

if (!Number.isInteger(id) || id < 1 || id > 2147483647) {
    throw createError({ statusCode: 404, statusMessage: 'Constellation not found' })
}

const killlistEndpoint = computed(() => isDomainMode.value ? `/api/custom/constellation/${id}/killlist` : `/api/constellation/${id}/killlist`)

const { data, pending, error } = await useApiFetch<any>(`/api/constellation/${id}`)

if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'Constellation not found',
    })
}

const constellation = computed(() => data.value?.constellation)
const systems = computed(() => data.value?.systems ?? [])
const stats = computed(() => data.value?.stats)
const sovDistribution = computed(() => data.value?.sovDistribution ?? [])

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

// Dominant sovereignty holder
const dominantHolder = computed(() => {
    const top = sovDistribution.value[0]
    if (!top) return null
    if (top.alliance_id) return {
        name: top.alliance_name,
        image: `/images/alliances/${top.alliance_id}/logo?size=256`,
        link: `/alliance/${top.alliance_id}`,
    }
    if (top.faction_id) return {
        name: top.faction_name,
        image: `/images/corporations/${top.faction_id}/logo?size=256`,
        link: null,
    }
    return null
})

useHead({ title: computed(() => constellation.value?.constellation_name || 'Constellation') })

useSeoMeta({
    description: computed(() => {
        const c = constellation.value
        if (!c?.constellation_name) return 'View constellation kill statistics on EVE-KILL.'
        const s = stats.value
        let desc = `${c.constellation_name} — ${c.region_name} — ${systems.value.length} systems`
        if (s) {
            desc += `. 7d: ${s.kills ?? 0} kills, ${s.pod_kills ?? 0} pod kills`
            if (s.total_value) desc += `, ${formatIsk(Number(s.total_value))} ISK destroyed`
        }
        desc += ' — EVE-KILL.'
        return desc
    }),
    ogTitle: computed(() => {
        const c = constellation.value
        if (!c) return 'Constellation — EVE-KILL'
        return `${c.constellation_name} — ${c.region_name}`
    }),
    ogDescription: computed(() => {
        const c = constellation.value
        if (!c) return 'View constellation stats on EVE-KILL.'
        return `${c.constellation_name} — ${c.region_name} — ${systems.value.length} systems — kills and activity in EVE Online.`
    }),
    ogType: 'website',
    twitterCard: 'summary',
    twitterTitle: computed(() => {
        const c = constellation.value
        if (!c) return 'Constellation — EVE-KILL'
        return `${c.constellation_name} — ${c.region_name} — EVE-KILL`
    }),
    twitterDescription: computed(() => {
        const c = constellation.value
        if (!c) return 'View constellation stats on EVE-KILL.'
        return `${c.constellation_name} — ${c.region_name} — ${systems.value.length} systems — kills and activity in EVE Online.`
    }),
    ogImage: `/images/constellations/${id}`,
    twitterImage: `/images/constellations/${id}`,
})

useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: (() => {
            const c = constellation.value
            if (!c) return [{ name: 'Home', item: '/' }]
            return [
                { name: 'Home', item: '/' },
                { name: c.region_name ?? 'Region', item: `/region/${c.region_id}` },
                { name: c.constellation_name, item: `/constellation/${id}` },
            ]
        })(),
    }))),
])

const tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
    { id: 'kills', label: 'Kills', icon: 'lucide:skull' },
    { id: 'battles', label: 'Battles', icon: 'lucide:swords' },
    { id: 'map', label: 'Map', icon: 'lucide:map' },
] as const

type TabId = typeof tabs[number]['id']
const tabIds = new Set(tabs.map(t => t.id))

useDefaultTab('constellation', `/constellation/${id}`, 'dashboard', tabIds)

definePageMeta({
    key: route => `/constellation/${route.params.id}`,
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
    const name = constellation.value?.constellation_name
    if (!name) return 'Constellation'
    return activeTabLabel.value ? `${name} (${activeTabLabel.value})` : name
}) })

const setTab = (tabId: string) => {
    if (!tabIds.has(tabId as TabId)) return
    useAnalytics().track('tab.change', { entity: 'constellation', tab: tabId })
    if (tabId === 'map' && constellation.value?.region_id) {
        navigateTo(`/map/region/${constellation.value.region_id}`)
        return
    }
    navigateTo(tabId === 'dashboard' ? `/constellation/${id}` : `/constellation/${id}/${tabId}`)
}

const dotlanUrl = computed(() => {
    if (!constellation.value?.constellation_name) return '#'
    return `https://evemaps.dotlan.net/map/${encodeURIComponent(constellation.value.region_name ?? '')}/${encodeURIComponent(constellation.value.constellation_name)}`
})
</script>

<template>
    <div>
        <EntityHeader v-if="pending" loading />

        <div v-else-if="constellation">
            <EntityHeader :background-image="`/images/constellations/${id}?size=1024`">
                <template v-if="dominantHolder" #image>
                    <EntityImageExpand :full-src="dominantHolder.image" :alt="dominantHolder.name">
                        <NuxtLink v-if="dominantHolder.link" :to="dominantHolder.link">
                            <img :src="dominantHolder.image" :alt="dominantHolder.name" class="w-32 h-32 md:w-40 md:h-40 rounded-lg shadow-lg" loading="eager">
                        </NuxtLink>
                        <img v-else :src="dominantHolder.image" :alt="dominantHolder.name" class="w-32 h-32 md:w-40 md:h-40 rounded-lg shadow-lg" loading="eager">
                    </EntityImageExpand>
                </template>

                <div class="flex items-start justify-between">
                    <div>
                        <div class="flex items-center gap-3 mb-2 flex-wrap">
                            <h1 class="text-2xl md:text-3xl font-bold text-white" :class="pochvenClass(constellation.region_id)">{{ constellation.constellation_name }}</h1>
                            <span v-if="dominantHolder" class="px-2.5 py-1 rounded-md text-sm font-bold text-purple-400" style="background: rgba(168,85,247,0.1)">
                                {{ dominantHolder.name }}
                            </span>
                        </div>
                        <div class="flex items-center gap-1.5 text-xs text-gray-500 mb-2">
                            <NuxtLink to="/map" class="text-gray-500 hover:text-blue-400 transition-colors">Map</NuxtLink>
                            <span class="text-gray-700">/</span>
                            <NuxtLink :to="`/region/${constellation.region_id}`" class="hover:text-blue-400 transition-colors" :class="pochvenClass(constellation.region_id)">{{ constellation.region_name }}</NuxtLink>
                            <span class="text-gray-700">/</span>
                            <span class="text-gray-400" :class="pochvenClass(constellation.region_id)">{{ constellation.constellation_name }}</span>
                        </div>
                    </div>
                    <!-- Desktop: external links inline -->
                    <div class="hidden md:flex flex-col gap-1.5 flex-shrink-0">
                        <a :href="dotlanUrl" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/dotlan.png" width="16" height="16" class="w-4 h-4" alt="" /> DOTLAN</a>
                        <a :href="`https://www.jita.space/constellation/${id}`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/jita-space.png" width="16" height="16" class="w-4 h-4" alt="" /> Jita.Space</a>
                        <a :href="`https://zkillboard.com/constellation/${id}/`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/zkillboard.png" width="16" height="16" class="w-4 h-4" alt="" /> zKillboard</a>
                    </div>
                </div>

                <!-- Mobile: external links next to image -->
                <template #right>
                    <div class="md:hidden flex flex-col gap-1.5 flex-shrink-0">
                        <a :href="dotlanUrl" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/dotlan.png" width="16" height="16" class="w-4 h-4" alt="" /> DOTLAN</a>
                        <a :href="`https://www.jita.space/constellation/${id}`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/jita-space.png" width="16" height="16" class="w-4 h-4" alt="" /> Jita.Space</a>
                        <a :href="`https://zkillboard.com/constellation/${id}/`" target="_blank" rel="noopener noreferrer" class="ext-link text-fine"><NuxtImg src="/remotes/zkillboard.png" width="16" height="16" class="w-4 h-4" alt="" /> zKillboard</a>
                    </div>
                </template>

                <template #stats>
                    <EntityStatGrid variant="inline" :columns="5">
                        <div v-if="stats">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Kills (7d)</div>
                            <div class="text-base font-bold text-green-400 tabular-nums">{{ Number(stats.kills).toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="stats">
                            <div class="text-fine uppercase tracking-wider text-gray-500">ISK Destroyed (7d)</div>
                            <div class="text-base font-bold text-yellow-400 tabular-nums">{{ formatIsk(Number(stats.total_value)) }}</div>
                        </div>
                        <div v-if="stats">
                            <div class="text-fine uppercase tracking-wider text-gray-500">NPC Kills (7d)</div>
                            <div class="text-base font-bold text-white tabular-nums">{{ Number(stats.npc_kills).toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="stats">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Pod Kills (7d)</div>
                            <div class="text-base font-bold text-red-400 tabular-nums">{{ Number(stats.pod_kills).toLocaleString('en-US') }}</div>
                        </div>
                        <div>
                            <div class="text-fine uppercase tracking-wider text-gray-500">Systems</div>
                            <div class="text-base font-bold text-blue-400 tabular-nums">{{ systems.length }}</div>
                        </div>
                    </EntityStatGrid>
                </template>
            </EntityHeader>

            <MostValuable :api-endpoint="`/api/constellation/${id}/most-valuable`" />

            <!-- TAB BAR -->
            <EntityTabBar :tabs="tabs" :active-id="activeTab" @select="setTab" />

            <!-- DASHBOARD TAB -->
            <div v-if="activeTab === 'dashboard'">
                <div class="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4">
                    <!-- LEFT column -->
                    <div class="space-y-4">
                        <!-- Sovereignty Distribution -->
                        <div v-if="sovDistribution.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:flag" class="w-4 h-4 text-purple-500" />
                                <span class="text-xs font-semibold text-gray-300">Sovereignty</span>
                            </div>
                            <div class="space-y-1.5">
                                <div v-for="(s, idx) in sovDistribution" :key="idx"
                                    class="flex items-center justify-between gap-3 px-1 py-1 rounded hover:bg-blue-500/[0.04] transition-colors">
                                    <div class="flex items-center gap-2 min-w-0">
                                        <img v-if="s.alliance_id" :src="`/images/alliances/${s.alliance_id}/logo?size=32`" class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                                        <img v-else-if="s.faction_id" :src="`/images/corporations/${s.faction_id}/logo?size=32`" class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                                        <NuxtLink v-if="s.alliance_id" :to="`/alliance/${s.alliance_id}`" class="text-fine text-gray-300 hover:text-blue-400 transition-colors truncate">{{ s.alliance_name }}</NuxtLink>
                                        <span v-else-if="s.faction_name" class="text-fine text-gray-300 truncate">{{ s.faction_name }}</span>
                                    </div>
                                    <span class="text-xs text-gray-500 tabular-nums flex-shrink-0">{{ s.system_count }} system{{ s.system_count !== 1 ? 's' : '' }}</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- RIGHT column -->
                    <div class="space-y-4">
                        <!-- Systems -->
                        <div v-if="systems.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:globe" class="w-4 h-4 text-blue-500" />
                                <span class="text-xs font-semibold text-gray-300">Systems</span>
                                <span class="text-xs text-gray-600">({{ systems.length }})</span>
                            </div>
                            <div class="space-y-1.5">
                                <NuxtLink v-for="sys in systems" :key="sys.solar_system_id"
                                    :to="`/system/${sys.solar_system_id}`"
                                    class="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-blue-500/[0.06] transition-colors">
                                    <img :src="`/images/systems/${sys.solar_system_id}?size=32`"
                                        class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                                    <span class="text-xs font-medium tabular-nums w-8" :class="securityColor(sys.security)">
                                        {{ secLabel(sys.security) }}
                                    </span>
                                    <span class="text-fine text-gray-300 truncate flex-1" :class="pochvenClass(constellation.region_id)">{{ sys.system_name }}</span>
                                    <div v-if="sys.alliance_id || sys.faction_id" class="flex items-center gap-1 flex-shrink-0">
                                        <img v-if="sys.alliance_id" :src="`/images/alliances/${sys.alliance_id}/logo?size=32`" class="w-4 h-4 rounded" loading="lazy">
                                        <img v-else-if="sys.faction_id" :src="`/images/corporations/${sys.faction_id}/logo?size=32`" class="w-4 h-4 rounded" loading="lazy">
                                        <span class="text-fine text-gray-500 truncate max-w-[100px]">{{ sys.alliance_name || sys.faction_name }}</span>
                                    </div>
                                </NuxtLink>
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
                <EntityBattles :constellation-id="id" />
            </div>
        </div>
    </div>
</template>
