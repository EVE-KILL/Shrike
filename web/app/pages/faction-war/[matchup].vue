<script setup lang="ts">
const route = useRoute()
const matchup = route.params.matchup as string

const validMatchups = ['caldari-vs-gallente', 'amarr-vs-minmatar']
if (!validMatchups.includes(matchup)) {
    throw createError({ statusCode: 404, statusMessage: 'Unknown faction matchup' })
}

const { data, pending, error } = await useApiFetch<any>(`/api/faction-war/${matchup}`)

if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'Faction war not found',
    })
}

const { data: statsData, pending: statsPending } = await useApiFetch<any>(`/api/faction-war/${matchup}`, { lazy: true })
const lbPeriod = ref<'yesterday' | 'last_week' | 'active_total'>('last_week')
const overviewParams = computed(() => ({ period: lbPeriod.value }))
const { data: overviewData } = await useApiFetch<any>(`/api/faction-war/${matchup}/overview`, {
    lazy: true,
    params: overviewParams,
    watch: [lbPeriod],
})

const side1 = computed(() => data.value?.side1)
const side2 = computed(() => data.value?.side2)
const topShips = computed(() => statsData.value?.topShips || [])

// Overview data
const fwStats1 = computed(() => overviewData.value?.factionStats?.side1)
const fwStats2 = computed(() => overviewData.value?.factionStats?.side2)
const warzone = computed(() => overviewData.value?.warzone)
const flipDays = computed(() => overviewData.value?.flipDays || [])
const charLb1 = computed(() => overviewData.value?.leaderboards?.characters?.side1 || [])
const charLb2 = computed(() => overviewData.value?.leaderboards?.characters?.side2 || [])
const corpLb1 = computed(() => overviewData.value?.leaderboards?.corporations?.side1 || [])
const corpLb2 = computed(() => overviewData.value?.leaderboards?.corporations?.side2 || [])

const warzoneControlPct = computed(() => {
    const wz = warzone.value
    if (!wz || wz.total_systems === 0) return 50
    return Math.round((wz.side1.total / wz.total_systems) * 100)
})

const FACTION_BAR_COLORS: Record<number, { bg: string, fill: string }> = {
    500001: { bg: 'bg-green-500/30', fill: 'bg-blue-500/80' },   // Caldari vs Gallente: blue bar on green bg
    500003: { bg: 'bg-red-500/30', fill: 'bg-amber-500/80' },    // Amarr vs Minmatar: amber bar on red bg
}
const barColors = computed(() => FACTION_BAR_COLORS[side1.value?.id] || { bg: 'bg-blue-500/30', fill: 'bg-red-500/80' })

const MONTH_SHORT = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

const formatFlipDay = (isoDay: string): string => {
    // isoDay is UTC YYYY-MM-DD from the API. Build a UTC reference date
    // so labels don't shift across the local/UTC boundary for users near
    // midnight.
    const [y, m, d] = isoDay.split('-').map(Number)
    const dayDate = new Date(Date.UTC(y, m - 1, d))
    const now = new Date()
    const todayUtc = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()))
    const diffDays = Math.round((todayUtc.getTime() - dayDate.getTime()) / 86400000)
    const label = `${d} ${MONTH_SHORT[m - 1]}`
    if (diffDays === 0) return `Today — ${label}`
    if (diffDays === 1) return `Yesterday — ${label}`
    return label
}

const formatFlipTime = (iso: string): string => {
    const t = new Date(iso)
    const hh = String(t.getUTCHours()).padStart(2, '0')
    const mm = String(t.getUTCMinutes()).padStart(2, '0')
    return `${hh}:${mm}`
}

const pageTitle = computed(() => side1.value && side2.value
    ? `${side1.value.name} vs ${side2.value.name} — Faction Warfare`
    : 'Faction War')

const seoDescription = computed(() => {
    const s1 = side1.value
    const s2 = side2.value
    if (!s1 || !s2) return 'Faction warfare statistics on EVE-KILL.'
    const parts: string[] = [`${s1.name} vs ${s2.name} — faction warfare in EVE Online`]
    const wz = warzone.value
    if (wz) parts.push(`${s1.name} holds ${wz.side1.total} systems, ${s2.name} holds ${wz.side2.total} of ${wz.total_systems}`)
    const fs1 = fwStats1.value
    const fs2 = fwStats2.value
    if (fs1 && fs2) parts.push(`${(fs1.pilots + fs2.pilots).toLocaleString('en-US')} enlisted pilots`)
    const totalKills = (s1.kills ?? 0) + (s2.kills ?? 0)
    const totalIsk = (s1.isk_destroyed ?? 0) + (s2.isk_destroyed ?? 0)
    if (totalKills) parts.push(`${totalKills.toLocaleString('en-US')} kills, ${formatIsk(totalIsk)} ISK destroyed (30d)`)
    parts.push('Warzone map, system control, leaderboards on EVE-KILL.')
    return parts.join('. ')
})

useHead({ title: pageTitle })
useSeoMeta({
    description: seoDescription,
    ogTitle: computed(() => side1.value && side2.value
        ? `${side1.value.name} vs ${side2.value.name} — Faction Warfare — EVE-KILL`
        : 'Faction War — EVE-KILL'),
    ogDescription: seoDescription,
    ogType: 'website',
    twitterCard: 'summary',
    twitterTitle: computed(() => side1.value && side2.value
        ? `${side1.value.name} vs ${side2.value.name} — Faction Warfare — EVE-KILL`
        : 'Faction War — EVE-KILL'),
    twitterDescription: seoDescription,
    ogImage: computed(() => side1.value
        ? `/images/corporations/${side1.value.corpId}/logo?size=256`
        : 'https://eve-kill.com/og-default.png'),
    twitterImage: computed(() => side1.value
        ? `/images/corporations/${side1.value.corpId}/logo?size=256`
        : 'https://eve-kill.com/og-default.png'),
})

useSchemaOrg([
    defineBreadcrumb({
        itemListElement: computed(() => [
            { name: 'Home', item: '/' },
            { name: 'Wars', item: '/wars' },
            { name: side1.value && side2.value ? `${side1.value.name} vs ${side2.value.name}` : 'Faction War', item: `/faction-war/${matchup}` },
        ]),
    }),
    {
        '@type': 'Event',
        'name': computed(() => side1.value && side2.value
            ? `${side1.value.name} vs ${side2.value.name} — Faction Warfare`
            : 'Faction Warfare'),
        'url': `https://eve-kill.com/faction-war/${matchup}`,
        'eventStatus': 'https://schema.org/EventScheduled',
        'eventAttendanceMode': 'https://schema.org/OnlineEventAttendanceMode',
        'description': seoDescription,
        'location': {
            '@type': 'VirtualLocation',
            'url': 'https://www.eveonline.com',
        },
        'organizer': computed(() => {
            if (!side1.value || !side2.value) return undefined
            return [
                { '@type': 'Organization', 'name': side1.value.name, 'url': `https://eve-kill.com/faction/${side1.value.id}` },
                { '@type': 'Organization', 'name': side2.value.name, 'url': `https://eve-kill.com/faction/${side2.value.id}` },
            ]
        }),
    },
])

// Warzone systems data (for map + table)
const { data: warzoneData } = await useApiFetch<any>(`/api/faction-war/${matchup}/systems`, { lazy: true })
const warzoneSystems = computed(() => warzoneData.value?.systems || [])
const warzoneJumps = computed(() => warzoneData.value?.jumps || [])
const warzoneCelestials = computed(() => warzoneData.value?.celestials || [])

const lbPeriodLabel: Record<string, string> = { yesterday: 'Yesterday', last_week: 'This Week', active_total: 'All Time' }

const activeTab = ref<'combined' | 'side1' | 'side2' | 'map' | 'systems' | 'members' | 'intel'>('combined')


const side1Efficiency = computed(() => {
    if (!side1.value || !side2.value) return 50
    const total = (side1.value.isk_destroyed || 0) + (side2.value.isk_destroyed || 0)
    if (total === 0) return 50
    return Math.round((side1.value.isk_destroyed / total) * 100)
})

// Victim factions for killlist filtering per tab
const killlistVictimFactions = computed(() => {
    if (!side1.value || !side2.value) return ''
    if (activeTab.value === 'side1') return String(side2.value.id) // Side1's kills = side2 victims
    if (activeTab.value === 'side2') return String(side1.value.id) // Side2's kills = side1 victims
    return `${side1.value.id},${side2.value.id}` // Combined: both factions as victims
})

// Sidebar stats per tab — from ESI leaderboards
const currentChars = computed(() => {
    if (activeTab.value === 'side1') return charLb1.value
    if (activeTab.value === 'side2') return charLb2.value
    return [...charLb1.value, ...charLb2.value]
        .sort((a: any, b: any) => b.kills - a.kills).slice(0, 10)
})

const currentCorps = computed(() => {
    if (activeTab.value === 'side1') return corpLb1.value
    if (activeTab.value === 'side2') return corpLb2.value
    return [...corpLb1.value, ...corpLb2.value]
        .sort((a: any, b: any) => b.kills - a.kills).slice(0, 10)
})
</script>

<template>
    <div>
        <div v-if="pending" class="h-64 rounded-lg bg-white/[0.04] animate-pulse"></div>

        <div v-else-if="side1 && side2">
            <h1 class="sr-only">{{ side1.name }} vs {{ side2.name }} — Faction War</h1>
            <!-- Header -->
            <div class="glass-panel p-6 mb-6">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-amber-400/80 mb-4 text-center">Faction Warfare — Last 30 Days</div>

                <div class="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] gap-6 items-center">
                    <!-- Side 1 -->
                    <div class="text-center p-4 rounded-lg bg-red-500/[0.04] border border-red-500/20">
                        <img :src="`/images/corporations/${side1.corpId}/logo?size=128`" class="w-20 h-20 mx-auto rounded-lg mb-2" loading="eager">
                        <div class="text-xl font-bold text-white mb-2">{{ side1.name }}</div>
                        <div v-if="fwStats1" class="flex items-center justify-center gap-3 text-fine text-gray-500 mb-2">
                            <span><span class="text-gray-300 tabular-nums">{{ formatNumber(fwStats1.pilots) }}</span> pilots</span>
                            <span><span class="text-gray-300 tabular-nums">{{ formatNumber(fwStats1.systems_controlled) }}</span> systems</span>
                        </div>
                        <div class="grid grid-cols-2 gap-2 text-xs">
                            <div><span class="text-green-400 tabular-nums font-medium">{{ formatNumber(side1.kills) }}</span><div class="text-gray-500">kills</div></div>
                            <div><span class="text-red-400 tabular-nums font-medium">{{ formatNumber(side1.losses) }}</span><div class="text-gray-500">losses</div></div>
                            <div><span class="text-green-400 tabular-nums font-medium">{{ formatIsk(side1.isk_destroyed) }}</span><div class="text-gray-500">destroyed</div></div>
                            <div><span class="text-red-400 tabular-nums font-medium">{{ formatIsk(side1.isk_lost) }}</span><div class="text-gray-500">lost</div></div>
                        </div>
                        <div v-if="fwStats1" class="grid grid-cols-2 gap-2 text-xs mt-2 pt-2 border-t border-white/[0.06]">
                            <div><span class="text-amber-400 tabular-nums font-medium">{{ formatNumber(fwStats1.vp_last_week) }}</span><div class="text-gray-500">VP this week</div></div>
                            <div><span class="text-amber-400 tabular-nums font-medium">{{ formatNumber(fwStats1.kills_last_week) }}</span><div class="text-gray-500">kills this week</div></div>
                        </div>
                    </div>

                    <!-- VS + efficiency -->
                    <div class="flex flex-col items-center gap-2 px-4">
                        <div class="text-2xl font-bold text-gray-600">VS</div>
                        <div class="w-32 h-2 bg-blue-500/30 rounded-full overflow-hidden">
                            <div class="h-full bg-red-500 rounded-full" :style="{ width: `${side1Efficiency}%` }"></div>
                        </div>
                        <div class="text-fine text-gray-500 tabular-nums">{{ side1Efficiency }}% — {{ 100 - side1Efficiency }}%</div>
                    </div>

                    <!-- Side 2 -->
                    <div class="text-center p-4 rounded-lg bg-blue-500/[0.04] border border-blue-500/20">
                        <img :src="`/images/corporations/${side2.corpId}/logo?size=128`" class="w-20 h-20 mx-auto rounded-lg mb-2" loading="eager">
                        <div class="text-xl font-bold text-white mb-2">{{ side2.name }}</div>
                        <div v-if="fwStats2" class="flex items-center justify-center gap-3 text-fine text-gray-500 mb-2">
                            <span><span class="text-gray-300 tabular-nums">{{ formatNumber(fwStats2.pilots) }}</span> pilots</span>
                            <span><span class="text-gray-300 tabular-nums">{{ formatNumber(fwStats2.systems_controlled) }}</span> systems</span>
                        </div>
                        <div class="grid grid-cols-2 gap-2 text-xs">
                            <div><span class="text-green-400 tabular-nums font-medium">{{ formatNumber(side2.kills) }}</span><div class="text-gray-500">kills</div></div>
                            <div><span class="text-red-400 tabular-nums font-medium">{{ formatNumber(side2.losses) }}</span><div class="text-gray-500">losses</div></div>
                            <div><span class="text-green-400 tabular-nums font-medium">{{ formatIsk(side2.isk_destroyed) }}</span><div class="text-gray-500">destroyed</div></div>
                            <div><span class="text-red-400 tabular-nums font-medium">{{ formatIsk(side2.isk_lost) }}</span><div class="text-gray-500">lost</div></div>
                        </div>
                        <div v-if="fwStats2" class="grid grid-cols-2 gap-2 text-xs mt-2 pt-2 border-t border-white/[0.06]">
                            <div><span class="text-amber-400 tabular-nums font-medium">{{ formatNumber(fwStats2.vp_last_week) }}</span><div class="text-gray-500">VP this week</div></div>
                            <div><span class="text-amber-400 tabular-nums font-medium">{{ formatNumber(fwStats2.kills_last_week) }}</span><div class="text-gray-500">kills this week</div></div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Warzone Control -->
            <div v-if="warzone" class="glass-panel p-4 mb-4">
                <div class="flex items-center justify-between mb-2">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Warzone Control</div>
                    <div class="text-fine text-gray-500">{{ warzone.total_systems }} systems</div>
                </div>

                <!-- Control bar -->
                <div class="relative h-6 rounded-full overflow-hidden mb-3" :class="barColors.bg">
                    <div class="absolute inset-y-0 left-0 rounded-l-full transition-all duration-500" :class="barColors.fill"
                        :style="{ width: `${warzoneControlPct}%` }"></div>
                    <div class="absolute inset-0 flex items-center justify-center text-fine font-bold text-white">
                        {{ warzone.side1.total }} — {{ warzone.side2.total }}
                    </div>
                </div>

                <!-- Status breakdown per side -->
                <div class="grid grid-cols-2 gap-4 text-xs">
                    <div class="flex flex-wrap gap-x-3 gap-y-1">
                        <span class="text-gray-400">{{ side1.name }}:</span>
                        <span v-if="warzone.side1.uncontested > 0" class="text-green-400 tabular-nums">{{ warzone.side1.uncontested }} stable</span>
                        <span v-if="warzone.side1.contested > 0" class="text-yellow-400 tabular-nums">{{ warzone.side1.contested }} contested</span>
                        <span v-if="warzone.side1.vulnerable > 0" class="text-red-400 tabular-nums">{{ warzone.side1.vulnerable }} vulnerable</span>
                    </div>
                    <div class="flex flex-wrap gap-x-3 gap-y-1 justify-end">
                        <span class="text-gray-400">{{ side2.name }}:</span>
                        <span v-if="warzone.side2.uncontested > 0" class="text-green-400 tabular-nums">{{ warzone.side2.uncontested }} stable</span>
                        <span v-if="warzone.side2.contested > 0" class="text-yellow-400 tabular-nums">{{ warzone.side2.contested }} contested</span>
                        <span v-if="warzone.side2.vulnerable > 0" class="text-red-400 tabular-nums">{{ warzone.side2.vulnerable }} vulnerable</span>
                    </div>
                </div>

                <!-- Frontline ticker -->
                <div v-if="warzone.side1.vulnerable + warzone.side2.vulnerable > 0"
                    class="mt-3 pt-3 border-t border-white/[0.06] flex items-center gap-2 text-xs">
                    <Icon name="lucide:alert-triangle" class="text-red-400 flex-shrink-0" />
                    <span class="text-red-400 font-medium">Frontline:</span>
                    <span class="text-gray-400">
                        <template v-if="warzone.side1.vulnerable > 0">
                            <span class="text-red-400 tabular-nums font-medium">{{ warzone.side1.vulnerable }}</span> {{ side1.name }} system{{ warzone.side1.vulnerable > 1 ? 's' : '' }} vulnerable
                        </template>
                        <template v-if="warzone.side1.vulnerable > 0 && warzone.side2.vulnerable > 0"> · </template>
                        <template v-if="warzone.side2.vulnerable > 0">
                            <span class="text-red-400 tabular-nums font-medium">{{ warzone.side2.vulnerable }}</span> {{ side2.name }} system{{ warzone.side2.vulnerable > 1 ? 's' : '' }} vulnerable
                        </template>
                    </span>
                </div>
                <div v-else-if="warzone.side1.contested + warzone.side2.contested > 0"
                    class="mt-3 pt-3 border-t border-white/[0.06] flex items-center gap-2 text-xs">
                    <Icon name="lucide:swords" class="text-yellow-400 flex-shrink-0" />
                    <span class="text-yellow-400 font-medium">Active fighting:</span>
                    <span class="text-gray-400">
                        <span class="text-yellow-400 tabular-nums font-medium">{{ warzone.side1.contested + warzone.side2.contested }}</span> systems contested across the warzone
                    </span>
                </div>
            </div>

            <!-- Recent Flips -->
            <div v-if="flipDays.length > 0" class="glass-panel p-3 mb-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">Recent System Flips</div>
                <div class="space-y-3">
                    <div v-for="group in flipDays" :key="group.day">
                        <div class="text-fine font-semibold uppercase tracking-[0.1em] text-gray-400 mb-1">
                            {{ formatFlipDay(group.day) }}
                            <span class="text-gray-600 font-normal normal-case tracking-normal ml-1">· {{ group.items.length }}</span>
                        </div>
                        <div class="space-y-1">
                            <div v-for="flip in group.items" :key="`${flip.solar_system_id}-${flip.flipped_at}`"
                                class="flex items-center gap-2 text-xs py-1">
                                <NuxtLink :to="`/system/${flip.solar_system_id}`" class="text-blue-400 hover:text-blue-300 font-medium">
                                    {{ flip.system_name }}
                                </NuxtLink>
                                <span class="text-gray-600">flipped</span>
                                <span class="text-red-400">{{ flip.old_faction_name }}</span>
                                <Icon name="lucide:arrow-right" class="text-fine text-gray-600" />
                                <span class="text-green-400">{{ flip.new_faction_name }}</span>
                                <span class="ml-auto text-fine text-gray-600 tabular-nums">{{ formatFlipTime(flip.flipped_at) }}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Tab bar -->
            <div class="flex overflow-x-auto border-b border-white/[0.08] mb-4 scrollbar-hide">
                <button v-for="tab in [
                    { id: 'combined', label: 'Both Sides', icon: 'lucide:layers' },
                    { id: 'side1', label: side1.name, icon: 'lucide:swords', color: 'border-red-400' },
                    { id: 'side2', label: side2.name, icon: 'lucide:shield', color: 'border-blue-400' },
                    { id: 'members', label: 'Members', icon: 'lucide:users' },
                    { id: 'intel', label: 'Intel', icon: 'lucide:radar' },
                    { id: 'map', label: 'Warzone Map', icon: 'lucide:map' },
                    { id: 'systems', label: 'Systems', icon: 'lucide:list' },
                ]" :key="tab.id"
                    class="flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors border-b-2 whitespace-nowrap"
                    :class="activeTab === tab.id
                        ? `text-white ${tab.color || 'border-white'}`
                        : 'text-gray-500 border-transparent hover:text-blue-400'"
                    @click="activeTab = tab.id as any">
                    <Icon :name="tab.icon" class="text-base" />
                    <span class="truncate max-w-[150px]" :class="activeTab === tab.id ? '' : 'hidden md:inline'">{{ tab.label }}</span>
                </button>
            </div>

            <!-- Map tab -->
            <div v-if="activeTab === 'map'">
                <ClientOnly>
                    <FwWarzoneMap
                        v-if="warzoneSystems.length > 0"
                        :systems="warzoneSystems"
                        :jumps="warzoneJumps"
                        :celestials="warzoneCelestials"
                        :side1-id="side1.id"
                        :side2-id="side2.id"
                        :side1-name="side1.name"
                        :side2-name="side2.name"
                    />
                    <div v-else class="flex items-center justify-center py-20">
                        <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                    </div>
                </ClientOnly>
            </div>

            <!-- Systems table tab -->
            <div v-else-if="activeTab === 'systems'">
                <FwWarzoneTable v-if="warzoneSystems.length > 0" :systems="warzoneSystems" />
                <div v-else class="flex items-center justify-center py-20">
                    <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                </div>
            </div>

            <!-- Members view (last 30 days) -->
            <WarMembers v-else-if="activeTab === 'members'"
                :endpoint="`/api/faction-war/${matchup}/members`"
                :side1-label="side1.name"
                :side2-label="side2.name"
                empty-label="No faction pilots found in the last 30 days"
                :key="`fw-members-${matchup}`" />

            <!-- Intel view (last 30 days) -->
            <WarIntel v-else-if="activeTab === 'intel'"
                :endpoint="`/api/faction-war/${matchup}/intel`"
                :key="`fw-intel-${matchup}`" />

            <!-- Content: sidebar + killlist (for combined/side1/side2 tabs) -->
            <div v-else class="grid grid-cols-1 md:grid-cols-[250px_1fr] gap-4">
                <!-- Sidebar -->
                <div class="space-y-3">
                    <!-- Leaderboard period toggle -->
                    <div class="glass-panel flex gap-1 p-1">
                        <button v-for="p in (['yesterday', 'last_week', 'active_total'] as const)" :key="p"
                            class="flex-1 px-2 py-1 text-fine rounded-md font-medium transition-colors"
                            :class="lbPeriod === p ? 'bg-blue-500/15 text-blue-400' : 'text-gray-500 hover:text-gray-300'"
                            @click="lbPeriod = p">
                            {{ lbPeriodLabel[p] }}
                        </button>
                    </div>

                    <div v-if="currentChars.length > 0" class="glass-panel p-2">
                        <div class="px-1 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                            Top Pilots ({{ lbPeriodLabel[lbPeriod] }})
                        </div>
                        <div class="space-y-px">
                            <NuxtLink v-for="(e, idx) in currentChars" :key="e.id" :to="`/character/${e.id}`"
                                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(idx) + 1 }}</span>
                                <img :src="`/images/characters/${e.id}/portrait?size=64`" class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                                <div class="flex-1 min-w-0">
                                    <div class="text-xs truncate">{{ e.name }}</div>
                                    <div v-if="e.corporation_name" class="text-fine text-gray-600 truncate">{{ e.corporation_name }}</div>
                                </div>
                                <span class="text-xs text-gray-500 tabular-nums flex-shrink-0">{{ e.kills }}</span>
                            </NuxtLink>
                        </div>
                    </div>

                    <div v-if="currentCorps.length > 0" class="glass-panel p-2">
                        <div class="px-1 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                            Top Corps ({{ lbPeriodLabel[lbPeriod] }})
                        </div>
                        <div class="space-y-px">
                            <NuxtLink v-for="(e, idx) in currentCorps" :key="e.id" :to="`/corporation/${e.id}`"
                                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                                <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(idx) + 1 }}</span>
                                <img :src="`/images/corporations/${e.id}/logo?size=64`" class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                                <span class="flex-1 text-xs truncate">{{ e.name }}</span>
                                <span class="text-xs text-gray-500 tabular-nums">{{ e.kills }}</span>
                            </NuxtLink>
                        </div>
                    </div>

                    <div v-if="topShips.length > 0" class="glass-panel p-2">
                        <div class="px-1 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                            Top Ships Destroyed
                        </div>
                        <div class="space-y-px">
                            <div v-for="(ship, idx) in topShips" :key="ship.ship_type_id"
                                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400">
                                <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(idx) + 1 }}</span>
                                <img :src="`/images/types/${ship.ship_type_id}/icon?size=64`" class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                                <span class="flex-1 text-xs truncate">{{ ship.ship_name }}</span>
                                <span class="text-xs text-gray-500 tabular-nums">{{ ship.kills }}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Kill list -->
                <KillList
                    killlist-type="latest"
                    :victim-factions="killlistVictimFactions"
                    :key="`fw-${matchup}-${activeTab}`"
                />
            </div>
        </div>
    </div>
</template>
