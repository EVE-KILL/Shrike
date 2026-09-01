<script setup lang="ts">
// Tab system with route segments — /alliance/123/kills
const tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
    { id: 'combined', label: 'Combined', icon: 'lucide:layers' },
    { id: 'kills', label: 'Kills', icon: 'lucide:trophy' },
    { id: 'losses', label: 'Losses', icon: 'lucide:skull' },
    { id: 'top', label: 'Top', icon: 'lucide:trending-up' },
    { id: 'battles', label: 'Battles', icon: 'lucide:swords' },
    { id: 'wars', label: 'Wars', icon: 'lucide:flag' },
    { id: 'campaigns', label: 'Campaigns', icon: 'lucide:target' },
    { id: 'corporations', label: 'Corporations', icon: 'lucide:building-2' },
    { id: 'members', label: 'Members', icon: 'lucide:users' },
] as const

// Keep the same page instance when only the tab param changes
definePageMeta({
    key: route => `/alliance/${route.params.id}`,
})

const {
    id, data, pending,
    entity: ally, stats, recentStats, accent,
    activeTab, setTab, killlistRole, killlistExtraParams, topLists,
    formatDate, ageYears: allianceAge, effWidth, iskEffWidth, dangerRatio,
} = await useEntityPage('alliance', {
    tabs,
    titleBase: a => a?.name ? `${a.name} [${a.ticker}]` : null,
})

const allTimeRanking = computed(() => data.value?.rankings?.all_time || null)

useSeoMeta({
    description: computed(() => {
        const a = ally.value
        if (!a?.name) return 'View alliance kill statistics on EVE-KILL.'
        const s = stats.value
        let desc = `${a.name} [${a.ticker}]`
        if (s?.kills || s?.losses) desc += `. ${s.kills ?? 0} kills, ${s.losses ?? 0} losses`
        desc += ' — EVE Online alliance stats on EVE-KILL.'
        return desc
    }),
    ogTitle: computed(() => ally.value ? `${ally.value.name} [${ally.value.ticker}]` : 'Alliance — EVE-KILL'),
    ogDescription: computed(() => {
        const a = ally.value
        if (!a?.name) return 'View alliance stats on EVE-KILL.'
        return `${a.name} [${a.ticker}] — kills, losses, and combat stats in EVE Online.`
    }),
    ogImage: computed(() => ally.value ? `/images/alliances/${ally.value.alliance_id}/logo?size=256` : ''),
    ogType: 'website',
    twitterCard: 'summary',
    twitterTitle: computed(() => ally.value ? `${ally.value.name} [${ally.value.ticker}] — EVE-KILL` : 'Alliance — EVE-KILL'),
    twitterDescription: computed(() => {
        const a = ally.value
        if (!a?.name) return 'View alliance stats on EVE-KILL.'
        return `${a.name} [${a.ticker}] — kills, losses, and combat stats in EVE Online.`
    }),
    twitterImage: computed(() => ally.value ? `/images/alliances/${ally.value.alliance_id}/logo?size=256` : ''),
})

useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: [
            { name: 'Home', item: '/' },
            { name: ally.value?.name ? `${ally.value.name} [${ally.value.ticker}]` : 'Alliance', item: `/alliance/${id}` },
        ],
    }))),
    {
        '@type': 'Organization',
        'name': computed(() => ally.value ? `${ally.value.name} [${ally.value.ticker}]` : 'Alliance'),
        'url': `https://eve-kill.com/alliance/${id}`,
        'logo': computed(() => ally.value ? `/images/alliances/${ally.value.alliance_id}/logo?size=256` : ''),
    },
])
</script>

<template>
    <div>
        <EntityHeader v-if="pending" loading />

        <div v-else-if="ally">
            <!-- ===== ALLIANCE HEADER — MOBILE ===== -->
            <div class="glass-panel md:hidden overflow-hidden mb-6"
                :style="accent ? { backgroundImage: `linear-gradient(to bottom, ${accent.soft}, transparent 60%)`, boxShadow: `inset 0 2px 0 0 ${accent.accent}` } : undefined">
                <div class="p-4">
                    <div class="flex gap-3">
                        <EntityImageExpand :full-src="`/images/alliances/${ally.alliance_id}/logo?size=256`" :alt="ally.name">
                            <img :src="`/images/alliances/${ally.alliance_id}/logo?size=256`"
                                :alt="ally.name" class="w-20 h-20 flex-shrink-0 rounded-lg shadow-lg" loading="eager">
                        </EntityImageExpand>
                        <div class="flex-1 min-w-0">
                            <h1 class="text-lg font-bold text-white leading-tight mb-1">
                                {{ ally.name }}
                                <span class="text-sm text-gray-500 font-normal">[{{ ally.ticker }}]</span>
                            </h1>
                            <div class="space-y-0.5">
                                <NuxtLink v-if="ally.executor_corporation_id" :to="`/corporation/${ally.executor_corporation_id}`"
                                    class="flex items-center gap-1.5 text-gray-300 hover:text-blue-400 transition-colors">
                                    <img :src="`/images/corporations/${ally.executor_corporation_id}/logo?size=64`" class="w-4 h-4 rounded" loading="lazy">
                                    <span class="text-fine truncate">Executor: {{ ally.executor_name || 'Unknown' }}</span>
                                </NuxtLink>
                                <NuxtLink v-if="ally.faction_id" :to="`/faction/${ally.faction_id}`" class="flex items-center gap-1.5 text-gray-400 hover:text-blue-400 transition-colors">
                                    <Icon name="lucide:flag" class="w-4 h-4" />
                                    <span class="text-fine">{{ ally.faction_name }}</span>
                                </NuxtLink>
                            </div>
                        </div>
                    </div>
                    <div class="flex flex-wrap gap-x-4 gap-y-1 mt-3 text-fine">
                        <div class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:building-2" class="w-3 h-3" />
                            <span class="text-gray-300">{{ ally.corporation_count.toLocaleString('en-US') }} corps</span>
                        </div>
                        <div class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:users" class="w-3 h-3" />
                            <span class="text-gray-300">{{ ally.member_count.toLocaleString('en-US') }} members</span>
                        </div>
                        <div v-if="ally.date_founded" class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:calendar" class="w-3 h-3" />
                            <span class="text-gray-300">{{ formatDate(ally.date_founded) }}</span>
                            <span v-if="allianceAge !== null" class="text-gray-600">({{ allianceAge }}y)</span>
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
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Corps / Members</span><span class="text-fine text-white tabular-nums">{{ ally.corporation_count }} / {{ ally.member_count.toLocaleString('en-US') }}</span></div>
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

            <!-- ===== ALLIANCE HEADER — DESKTOP ===== -->
            <div class="hidden md:block">
            <EntityHeader :accent="accent?.accent">
                <template #image>
                    <EntityImageExpand :full-src="`/images/alliances/${ally.alliance_id}/logo?size=256`" :alt="ally.name">
                        <img :src="`/images/alliances/${ally.alliance_id}/logo?size=256`"
                            :alt="ally.name" class="w-40 h-40 rounded-lg shadow-lg" loading="eager">
                    </EntityImageExpand>
                </template>

                <div class="flex flex-row gap-6">
                    <div class="flex-1 min-w-0">
                        <h1 class="text-3xl font-bold text-white mb-1">
                            {{ ally.name }}
                            <span class="text-lg text-gray-500 font-normal">[{{ ally.ticker }}]</span>
                        </h1>

                        <div class="space-y-1.5 mb-4">
                            <NuxtLink v-if="ally.executor_corporation_id" :to="`/corporation/${ally.executor_corporation_id}`"
                                class="flex items-center gap-2 text-gray-300 hover:text-blue-400 transition-colors">
                                <img :src="`/images/corporations/${ally.executor_corporation_id}/logo?size=64`"
                                    class="w-5 h-5 rounded" loading="lazy">
                                <span class="text-xs">Executor: {{ ally.executor_name || 'Unknown' }}</span>
                                <span v-if="ally.executor_ticker" class="text-fine text-gray-600">[{{ ally.executor_ticker }}]</span>
                            </NuxtLink>

                            <NuxtLink v-if="ally.faction_id" :to="`/faction/${ally.faction_id}`" class="flex items-center gap-2 text-gray-400 hover:text-blue-400 transition-colors">
                                <Icon name="lucide:flag" class="w-5 h-5" />
                                <span class="text-xs">{{ ally.faction_name }}</span>
                            </NuxtLink>
                        </div>
                    </div>

                    <div class="flex-shrink-0 min-w-[200px] space-y-2.5 text-xs">
                        <div>
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs">
                                <Icon name="lucide:building-2" class="w-3.5 h-3.5" />
                                Corporations
                            </div>
                            <div class="text-fine text-gray-300 ml-5">{{ ally.corporation_count.toLocaleString('en-US') }}</div>
                        </div>
                        <div>
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs">
                                <Icon name="lucide:users" class="w-3.5 h-3.5" />
                                Members
                            </div>
                            <div class="text-fine text-gray-300 ml-5">{{ ally.member_count.toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="ally.date_founded">
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs">
                                <Icon name="lucide:calendar" class="w-3.5 h-3.5" />
                                Founded
                            </div>
                            <div class="text-fine text-gray-300 ml-5">
                                {{ formatDate(ally.date_founded) }}
                                <span v-if="allianceAge !== null" class="text-xs text-gray-500">({{ allianceAge }} years)</span>
                            </div>
                        </div>
                    </div>
                </div>

                <template #right>
                    <div class="flex gap-3 items-start">
                        <EntityRankingBadge :ranking="allTimeRanking" />
                        <NuxtLink v-if="ally.executor_corporation_id" :to="`/corporation/${ally.executor_corporation_id}`"
                            class="block hover:opacity-80 transition-opacity">
                            <img :src="`/images/corporations/${ally.executor_corporation_id}/logo?size=128`"
                                class="w-20 h-20 rounded-lg shadow-md">
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
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Corps / Members</span><span class="text-fine text-white tabular-nums">{{ ally.corporation_count }} / {{ ally.member_count.toLocaleString('en-US') }}</span></div>
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

            <MostValuable :api-endpoint="`/api/entity/alliance/${id}/most-valuable`" />

            <EntityShipClasses entity-type="alliance" :entity-id="id" />

            <!-- ===== TAB BAR ===== -->
            <EntityTabBar :tabs="tabs" :active-id="activeTab" :accent="accent" @select="setTab" />

            <!-- Dashboard tab -->
            <div v-if="activeTab === 'dashboard'">
                <AllianceDashboard
                    :alliance-id="id"
                    :custom-description-html="ally.custom_description_html"
                />
            </div>

            <!-- Top tab -->
            <div v-if="activeTab === 'top'">
                <EntityTop entity-type="alliance" :entity-id="id" />
            </div>

            <!-- Corporations tab -->
            <div v-if="activeTab === 'corporations'">
                <AllianceCorporations :alliance-id="id" />
            </div>

            <!-- Members tab -->
            <div v-if="activeTab === 'members'">
                <AllianceMembers :alliance-id="id" />
            </div>

            <!-- Kill tabs -->
            <EntityKillTabsLayout v-if="['combined', 'kills', 'losses'].includes(activeTab)"
                kind="alliance" :entity-id="id" :active-tab="activeTab"
                :top-lists="topLists" :killlist-role="killlistRole" :extra-params="killlistExtraParams" />

            <div v-if="activeTab === 'battles'">
                <EntityBattles :alliance-id="id" />
            </div>

            <div v-if="activeTab === 'wars'">
                <EntityWars :alliance-id="id" />
            </div>

            <div v-if="activeTab === 'campaigns'">
                <EntityCampaigns :alliance-id="id" />
            </div>
        </div>
    </div>
</template>
