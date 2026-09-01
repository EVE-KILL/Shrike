<script setup lang="ts">
// Tab system with route segments — /corporation/123/kills
const tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
    { id: 'combined', label: 'Combined', icon: 'lucide:layers' },
    { id: 'kills', label: 'Kills', icon: 'lucide:trophy' },
    { id: 'losses', label: 'Losses', icon: 'lucide:skull' },
    { id: 'top', label: 'Top', icon: 'lucide:trending-up' },
    { id: 'battles', label: 'Battles', icon: 'lucide:swords' },
    { id: 'wars', label: 'Wars', icon: 'lucide:flag' },
    { id: 'campaigns', label: 'Campaigns', icon: 'lucide:target' },
    { id: 'members', label: 'Members', icon: 'lucide:users' },
    { id: 'history', label: 'History', icon: 'lucide:history' },
] as const

// Keep the same page instance when only the tab param changes
definePageMeta({
    key: route => `/corporation/${route.params.id}`,
})

const {
    id, data, pending,
    entity: corp, stats, recentStats, accent,
    activeTab, setTab, killlistRole, killlistExtraParams, topLists,
    formatDate, ageYears: corpAge, effWidth, iskEffWidth, dangerRatio,
} = await useEntityPage('corporation', {
    tabs,
    titleBase: c => c?.name ? `${c.name} [${c.ticker}]` : null,
})

const allianceHistory = computed(() => data.value?.allianceHistory || [])
const allTimeRanking = computed(() => data.value?.rankings?.all_time || null)

const paletteSwatches = computed(() => {
    const p = corp.value?.palette
    if (!p) return []
    return [p.main_color, p.secondary_color, p.tertiary_color].filter(Boolean) as string[]
})

useSeoMeta({
    description: computed(() => {
        const c = corp.value
        if (!c?.name) return 'View corporation kill statistics on EVE-KILL.'
        const s = stats.value
        let desc = `${c.name} [${c.ticker}]`
        if (c.alliance_name) desc += ` — member of ${c.alliance_name}`
        if (s?.kills || s?.losses) desc += `. ${s.kills ?? 0} kills, ${s.losses ?? 0} losses`
        desc += ' — EVE Online corporation stats on EVE-KILL.'
        return desc
    }),
    ogTitle: computed(() => {
        const c = corp.value
        if (!c?.name) return 'Corporation — EVE-KILL'
        let t = `${c.name} [${c.ticker}]`
        if (c.alliance_name) t += ` | ${c.alliance_name} [${c.alliance_ticker}]`
        return t
    }),
    ogDescription: computed(() => {
        const c = corp.value
        if (!c?.name) return 'View corporation stats on EVE-KILL.'
        return `${c.name} [${c.ticker}] — kills, losses, and combat stats in EVE Online.`
    }),
    ogImage: computed(() => corp.value ? `/images/corporations/${corp.value.corporation_id}/logo?size=256` : ''),
    ogType: 'website',
    twitterCard: 'summary',
    twitterTitle: computed(() => {
        const c = corp.value
        if (!c?.name) return 'Corporation — EVE-KILL'
        return `${c.name} [${c.ticker}] — EVE-KILL`
    }),
    twitterDescription: computed(() => {
        const c = corp.value
        if (!c?.name) return 'View corporation stats on EVE-KILL.'
        return `${c.name} [${c.ticker}] — kills, losses, and combat stats in EVE Online.`
    }),
    twitterImage: computed(() => corp.value ? `/images/corporations/${corp.value.corporation_id}/logo?size=256` : ''),
})

useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: (() => {
            const crumbs: { name: string; item: string }[] = [{ name: 'Home', item: '/' }]
            const c = corp.value
            if (c?.alliance_id && c.alliance_name) {
                crumbs.push({ name: `${c.alliance_name}`, item: `/alliance/${c.alliance_id}` })
            }
            crumbs.push({ name: c?.name ? `${c.name} [${c.ticker}]` : 'Corporation', item: `/corporation/${id}` })
            return crumbs
        })(),
    }))),
    {
        '@type': 'Organization',
        'name': computed(() => corp.value ? `${corp.value.name} [${corp.value.ticker}]` : 'Corporation'),
        'url': `https://eve-kill.com/corporation/${id}`,
        'logo': computed(() => corp.value ? `/images/corporations/${corp.value.corporation_id}/logo?size=256` : ''),
        'memberOf': computed(() => {
            const c = corp.value
            if (c?.alliance_name) return { '@type': 'Organization', name: c.alliance_name, url: `https://eve-kill.com/alliance/${c.alliance_id}` }
            return undefined
        }),
    },
])

// Alliance history with durations
const allianceHistoryWithDuration = computed(() => {
    const h = allianceHistory.value
    return h.map((entry: any, idx: number) => {
        const start = new Date(entry.start_date)
        const end = idx > 0 ? new Date(h[idx - 1].start_date) : new Date()
        const totalDays = Math.floor((end.getTime() - start.getTime()) / 86400000)
        const years = Math.floor(totalDays / 365)
        const months = Math.floor((totalDays % 365) / 30)
        let duration = ''
        if (years > 0) duration += `${years}y `
        if (months > 0) duration += `${months}m `
        if (years === 0 && months === 0) duration = `${totalDays}d`
        return { ...entry, duration: duration.trim(), totalDays, isCurrent: idx === 0 }
    })
})
</script>

<template>
    <div>
        <!-- Loading -->
        <EntityHeader v-if="pending" loading />

        <div v-else-if="corp">
            <!-- ===== CORPORATION HEADER — MOBILE ===== -->
            <div class="glass-panel md:hidden overflow-hidden mb-6"
                :style="accent ? { backgroundImage: `linear-gradient(to bottom, ${accent.soft}, transparent 60%)`, boxShadow: `inset 0 2px 0 0 ${accent.accent}` } : undefined">
                <div class="p-4">
                    <div class="flex gap-3">
                        <EntityImageExpand :full-src="`/images/corporations/${corp.corporation_id}/logo?size=256`" :alt="corp.name">
                            <img :src="`/images/corporations/${corp.corporation_id}/logo?size=256`"
                                :alt="corp.name" class="w-20 h-20 flex-shrink-0 rounded-lg shadow-lg" loading="eager">
                        </EntityImageExpand>
                        <div class="flex-1 min-w-0">
                            <h1 class="text-lg font-bold text-white leading-tight mb-1">
                                {{ corp.name }}
                                <span class="text-sm text-gray-500 font-normal">[{{ corp.ticker }}]</span>
                                <span v-if="corp.state === 'closed'" class="text-fine uppercase tracking-wider px-1.5 py-0.5 rounded bg-red-500/15 text-red-400 font-medium align-middle ml-1">Closed</span>
                                <span v-if="paletteSwatches.length" class="inline-flex items-center gap-1 ml-1.5 align-middle">
                                    <span v-for="c in paletteSwatches" :key="c" class="w-2 h-2 rounded-full border border-white/20"
                                        :style="{ backgroundColor: c }" v-tooltip="c" />
                                </span>
                            </h1>
                            <div class="space-y-0.5">
                                <NuxtLink v-if="corp.alliance_id" :to="`/alliance/${corp.alliance_id}`"
                                    class="flex items-center gap-1.5 text-gray-300 hover:text-blue-400 transition-colors">
                                    <img :src="`/images/alliances/${corp.alliance_id}/logo?size=64`" class="w-4 h-4 rounded" loading="lazy">
                                    <span class="text-fine truncate">{{ corp.alliance_name || 'Unknown' }}</span>
                                    <span v-if="corp.alliance_ticker" class="text-fine text-gray-600">[{{ corp.alliance_ticker }}]</span>
                                </NuxtLink>
                                <NuxtLink v-if="corp.ceo_id" :to="`/character/${corp.ceo_id}`"
                                    class="flex items-center gap-1.5 text-gray-400 hover:text-blue-400 transition-colors">
                                    <Icon name="lucide:crown" class="w-4 h-4" />
                                    <span class="text-fine">CEO: {{ corp.ceo_name || 'Unknown' }}</span>
                                </NuxtLink>
                                <NuxtLink v-if="corp.faction_id" :to="`/faction/${corp.faction_id}`" class="flex items-center gap-1.5 text-gray-400 hover:text-blue-400 transition-colors">
                                    <Icon name="lucide:flag" class="w-4 h-4" />
                                    <span class="text-fine">{{ corp.faction_name }}</span>
                                </NuxtLink>
                            </div>
                        </div>
                    </div>
                    <div class="flex flex-wrap gap-x-4 gap-y-1 mt-3 text-fine">
                        <div class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:users" class="w-3 h-3" />
                            <span class="text-gray-300">{{ corp.member_count.toLocaleString('en-US') }} members</span>
                        </div>
                        <div v-if="corp.date_founded" class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:calendar" class="w-3 h-3" />
                            <span class="text-gray-300">{{ formatDate(corp.date_founded) }}</span>
                            <span v-if="corpAge !== null" class="text-gray-600">({{ corpAge }}y)</span>
                        </div>
                        <div class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:percent" class="w-3 h-3" />
                            <span class="text-gray-300">{{ (corp.tax_rate * 100).toFixed(0) }}% tax</span>
                            <span v-if="corp.lp_tax_rate > 0" class="text-gray-500">· {{ (corp.lp_tax_rate * 100).toFixed(1).replace(/\.0$/, '') }}% LP</span>
                        </div>
                        <div v-if="corp.war_eligible" class="flex items-center gap-1 text-red-400">
                            <Icon name="lucide:swords" class="w-3 h-3" />
                            <span>War Eligible</span>
                        </div>
                    </div>
                </div>
                <div v-if="stats" class="px-4 pb-4 pt-2 border-t border-white/[0.04]">
                    <EntityStatGrid variant="boxed">
                        <EntityStatBox icon="lucide:swords" icon-color="text-red-500" title="Combat">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Kills</span><span class="text-fine text-green-400 tabular-nums">{{ stats.kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ stats.losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Efficiency</span><span class="text-fine text-white tabular-nums">{{ stats.efficiency }}%</span></div>
                            <div class="h-1 bg-red-500/20 rounded-full overflow-hidden"><div class="h-full bg-green-500 rounded-full" :style="{ width: effWidth }"></div></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Danger Ratio</span><span class="text-fine tabular-nums" :class="dangerRatio >= 50 ? 'text-red-400' : 'text-yellow-400'">{{ dangerRatio }}%</span></div>
                        </EntityStatBox>
                        <EntityStatBox icon="lucide:coins" icon-color="text-yellow-500" title="ISK">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Destroyed</span><span class="text-fine text-green-400 tabular-nums">{{ formatIsk(stats.isk_destroyed) }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(stats.isk_lost) }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Eff.</span><span class="text-fine text-white tabular-nums">{{ stats.isk_efficiency }}%</span></div>
                            <div class="h-1 bg-red-500/20 rounded-full overflow-hidden"><div class="h-full bg-green-500 rounded-full" :style="{ width: iskEffWidth }"></div></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Balance</span><span class="text-fine tabular-nums" :class="stats.isk_destroyed > stats.isk_lost ? 'text-green-400' : 'text-red-400'">{{ stats.isk_destroyed > stats.isk_lost ? '+' : '' }}{{ formatIsk(stats.isk_destroyed - stats.isk_lost) }}</span></div>
                        </EntityStatBox>
                        <EntityStatBox icon="lucide:user" icon-color="text-blue-500" title="Activity">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Solo Kills</span><span class="text-fine text-green-400 tabular-nums">{{ stats.solo_kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Final Blows</span><span class="text-fine text-white tabular-nums">{{ stats.final_blows.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Combat Points</span><span class="text-fine text-white tabular-nums">{{ stats.points.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Members</span><span class="text-fine text-white tabular-nums">{{ corp.member_count.toLocaleString('en-US') }}</span></div>
                        </EntityStatBox>
                        <EntityStatBox v-if="recentStats" icon="lucide:clock" icon-color="text-purple-500" title="Last 90 Days">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Kills</span><span class="text-fine text-green-400 tabular-nums">{{ recentStats.kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ recentStats.losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Destroyed</span><span class="text-fine text-green-400 tabular-nums">{{ formatIsk(recentStats.isk_destroyed) }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(recentStats.isk_lost) }}</span></div>
                        </EntityStatBox>
                    </EntityStatGrid>
                </div>
            </div>

            <!-- ===== CORPORATION HEADER — DESKTOP ===== -->
            <div class="hidden md:block">
            <EntityHeader :accent="accent?.accent">
                <template #image>
                    <EntityImageExpand :full-src="`/images/corporations/${corp.corporation_id}/logo?size=256`" :alt="corp.name">
                        <img
                            :src="`/images/corporations/${corp.corporation_id}/logo?size=256`"
                            :alt="corp.name"
                            class="w-40 h-40 rounded-lg shadow-lg"
                            loading="eager"
                        >
                    </EntityImageExpand>
                </template>

                <div class="flex flex-row gap-6">
                    <div class="flex-1 min-w-0">
                        <h1 class="text-3xl font-bold text-white mb-1">
                            {{ corp.name }}
                            <span class="text-lg text-gray-500 font-normal">[{{ corp.ticker }}]</span>
                            <span v-if="corp.state === 'closed'" class="text-xs uppercase tracking-wider px-2 py-0.5 rounded bg-red-500/15 text-red-400 font-medium align-middle ml-1.5">Closed</span>
                            <span v-if="paletteSwatches.length" class="inline-flex items-center gap-1.5 ml-2 align-middle">
                                <span v-for="c in paletteSwatches" :key="c" class="w-3 h-3 rounded-full border border-white/20"
                                    :style="{ backgroundColor: c }" v-tooltip="c" />
                            </span>
                        </h1>

                        <!-- Affiliations -->
                        <div class="space-y-1.5 mb-4">
                            <NuxtLink v-if="corp.alliance_id" :to="`/alliance/${corp.alliance_id}`"
                                class="flex items-center gap-2 text-gray-300 hover:text-blue-400 transition-colors">
                                <img :src="`/images/alliances/${corp.alliance_id}/logo?size=64`"
                                    class="w-5 h-5 rounded" loading="lazy">
                                <span class="text-xs">{{ corp.alliance_name || 'Unknown' }}</span>
                                <span v-if="corp.alliance_ticker" class="text-fine text-gray-600">[{{ corp.alliance_ticker }}]</span>
                            </NuxtLink>

                            <NuxtLink v-if="corp.faction_id" :to="`/faction/${corp.faction_id}`" class="flex items-center gap-2 text-gray-400 hover:text-blue-400 transition-colors">
                                <Icon name="lucide:flag" class="w-5 h-5" />
                                <span class="text-xs">{{ corp.faction_name }}</span>
                            </NuxtLink>

                            <NuxtLink v-if="corp.ceo_id" :to="`/character/${corp.ceo_id}`"
                                class="flex items-center gap-2 text-gray-400 hover:text-blue-400 transition-colors">
                                <Icon name="lucide:crown" class="w-5 h-5" />
                                <span class="text-xs">CEO: {{ corp.ceo_name || 'Unknown' }}</span>
                            </NuxtLink>
                        </div>
                    </div>

                    <!-- Meta info -->
                    <div class="flex-shrink-0 min-w-[200px] space-y-2.5 text-xs">
                        <div>
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs">
                                <Icon name="lucide:users" class="w-3.5 h-3.5" />
                                Members
                            </div>
                            <div class="text-fine text-gray-300 ml-5">{{ corp.member_count.toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="corp.date_founded">
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs">
                                <Icon name="lucide:calendar" class="w-3.5 h-3.5" />
                                Founded
                            </div>
                            <div class="text-fine text-gray-300 ml-5">
                                {{ formatDate(corp.date_founded) }}
                                <span v-if="corpAge !== null" class="text-xs text-gray-500">({{ corpAge }} years)</span>
                            </div>
                        </div>
                        <div>
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs">
                                <Icon name="lucide:percent" class="w-3.5 h-3.5" />
                                Tax Rate
                            </div>
                            <div class="text-fine text-gray-300 ml-5">
                                {{ (corp.tax_rate * 100).toFixed(0) }}%
                                <span v-if="corp.lp_tax_rate > 0" class="text-gray-500">· {{ (corp.lp_tax_rate * 100).toFixed(1).replace(/\.0$/, '') }}% LP</span>
                            </div>
                        </div>
                        <div v-if="corp.war_eligible" class="flex items-center gap-1.5 text-red-400 text-xs ml-0">
                            <Icon name="lucide:swords" class="w-3.5 h-3.5" />
                            War Eligible
                        </div>
                    </div>
                </div>

                <template #right>
                    <div class="flex gap-3 items-start">
                        <EntityRankingBadge :ranking="allTimeRanking" />
                        <NuxtLink v-if="corp.alliance_id" :to="`/alliance/${corp.alliance_id}`"
                            class="block hover:opacity-80 transition-opacity">
                            <img :src="`/images/alliances/${corp.alliance_id}/logo?size=128`"
                                :alt="corp.alliance_name" class="w-20 h-20 rounded-lg shadow-md">
                        </NuxtLink>
                    </div>
                </template>

                <template v-if="stats" #stats>
                    <EntityStatGrid variant="boxed">
                        <EntityStatBox icon="lucide:swords" icon-color="text-red-500" title="Combat">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Kills</span><span class="text-fine text-green-400 tabular-nums">{{ stats.kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ stats.losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Efficiency</span><span class="text-fine text-white tabular-nums">{{ stats.efficiency }}%</span></div>
                            <div class="h-1 bg-red-500/20 rounded-full overflow-hidden"><div class="h-full bg-green-500 rounded-full" :style="{ width: effWidth }"></div></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Danger Ratio</span><span class="text-fine tabular-nums" :class="dangerRatio >= 50 ? 'text-red-400' : 'text-yellow-400'">{{ dangerRatio }}%</span></div>
                        </EntityStatBox>

                        <EntityStatBox icon="lucide:coins" icon-color="text-yellow-500" title="ISK">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Destroyed</span><span class="text-fine text-green-400 tabular-nums">{{ formatIsk(stats.isk_destroyed) }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(stats.isk_lost) }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Efficiency</span><span class="text-fine text-white tabular-nums">{{ stats.isk_efficiency }}%</span></div>
                            <div class="h-1 bg-red-500/20 rounded-full overflow-hidden"><div class="h-full bg-green-500 rounded-full" :style="{ width: iskEffWidth }"></div></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Balance</span><span class="text-fine tabular-nums" :class="stats.isk_destroyed > stats.isk_lost ? 'text-green-400' : 'text-red-400'">{{ stats.isk_destroyed > stats.isk_lost ? '+' : '' }}{{ formatIsk(stats.isk_destroyed - stats.isk_lost) }}</span></div>
                        </EntityStatBox>

                        <EntityStatBox icon="lucide:user" icon-color="text-blue-500" title="Activity">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Solo Kills</span><span class="text-fine text-green-400 tabular-nums">{{ stats.solo_kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Final Blows</span><span class="text-fine text-white tabular-nums">{{ stats.final_blows.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Combat Points</span><span class="text-fine text-white tabular-nums">{{ stats.points.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Members</span><span class="text-fine text-white tabular-nums">{{ corp.member_count.toLocaleString('en-US') }}</span></div>
                        </EntityStatBox>

                        <EntityStatBox v-if="recentStats" icon="lucide:clock" icon-color="text-purple-500" title="Last 90 Days">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Kills</span><span class="text-fine text-green-400 tabular-nums">{{ recentStats.kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ recentStats.losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Destroyed</span><span class="text-fine text-green-400 tabular-nums">{{ formatIsk(recentStats.isk_destroyed) }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(recentStats.isk_lost) }}</span></div>
                        </EntityStatBox>
                    </EntityStatGrid>
                </template>
            </EntityHeader>
            </div>

            <MostValuable :api-endpoint="`/api/entity/corporation/${id}/most-valuable`" />

            <EntityShipClasses entity-type="corporation" :entity-id="id" />

            <!-- ===== TAB BAR ===== -->
            <EntityTabBar :tabs="tabs" :active-id="activeTab" :accent="accent" @select="setTab" />

            <!-- ===== DASHBOARD TAB ===== -->
            <div v-if="activeTab === 'dashboard'">
                <CorporationDashboard
                    :corporation-id="id"
                    :description="corp.description"
                    :custom-description-html="corp.custom_description_html"
                />
            </div>

            <!-- ===== TOP TAB ===== -->
            <div v-if="activeTab === 'top'">
                <EntityTop entity-type="corporation" :entity-id="id" />
            </div>

            <!-- ===== MEMBERS TAB ===== -->
            <div v-if="activeTab === 'members'">
                <CorporationMembers :corporation-id="id" />
            </div>

            <!-- ===== ALLIANCE HISTORY TAB ===== -->
            <div v-if="activeTab === 'history'">
                <div v-if="allianceHistoryWithDuration.length > 0" class="space-y-2">
                    <NuxtLink v-for="entry in allianceHistoryWithDuration" :key="entry.start_date"
                        :to="entry.alliance_id ? `/alliance/${entry.alliance_id}` : '#'"
                        class="flex items-center gap-4 p-3 rounded-lg border transition-colors hover:bg-blue-500/[0.04]"
                        :class="entry.isCurrent ? 'border-blue-500/20 bg-blue-500/[0.04]' : 'border-white/[0.08]'">
                        <div class="flex-shrink-0">
                            <img v-if="entry.alliance_id"
                                :src="`/images/alliances/${entry.alliance_id}/logo?size=128`"
                                class="w-12 h-12 rounded-lg" loading="lazy">
                            <div v-else class="w-12 h-12 rounded-lg bg-white/[0.04] flex items-center justify-center">
                                <Icon name="lucide:circle-off" class="w-5 h-5 text-gray-600" />
                            </div>
                        </div>
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center gap-2">
                                <span class="text-xs text-gray-300 truncate">{{ entry.alliance_name || 'No Alliance' }}</span>
                                <span v-if="entry.alliance_ticker" class="text-fine text-gray-600">[{{ entry.alliance_ticker }}]</span>
                                <span v-if="entry.isCurrent" class="text-fine uppercase tracking-wider px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 font-medium">Current</span>
                            </div>
                            <div class="text-fine text-gray-500 mt-0.5">{{ formatDate(entry.start_date) }}</div>
                        </div>
                        <div class="flex-shrink-0 text-right">
                            <div class="text-fine text-gray-400 tabular-nums">{{ entry.duration }}</div>
                            <div class="text-fine text-gray-600">{{ entry.totalDays.toLocaleString('en-US') }} days</div>
                        </div>
                    </NuxtLink>
                </div>
                <div v-else class="text-sm text-gray-600 text-center py-8">No alliance history available</div>
            </div>

            <!-- ===== KILL TABS (Combined / Kills / Losses) ===== -->
            <EntityKillTabsLayout v-if="['combined', 'kills', 'losses'].includes(activeTab)"
                kind="corporation" :entity-id="id" :active-tab="activeTab"
                :top-lists="topLists" :killlist-role="killlistRole" :extra-params="killlistExtraParams" />

            <div v-if="activeTab === 'battles'">
                <EntityBattles :corporation-id="id" />
            </div>

            <div v-if="activeTab === 'wars'">
                <EntityWars :corporation-id="id" />
            </div>

            <div v-if="activeTab === 'campaigns'">
                <EntityCampaigns :corporation-id="id" />
            </div>
        </div>
    </div>
</template>
