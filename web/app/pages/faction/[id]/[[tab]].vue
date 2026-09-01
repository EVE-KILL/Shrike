<script setup lang="ts">
// Tab system with route segments — /faction/500001/kills
const tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
    { id: 'combined', label: 'Combined', icon: 'lucide:layers' },
    { id: 'kills', label: 'Kills', icon: 'lucide:trophy' },
    { id: 'losses', label: 'Losses', icon: 'lucide:skull' },
] as const

// Keep the same page instance when only the tab param changes
definePageMeta({
    key: route => `/faction/${route.params.id}`,
})

const factionId = Number(useRoute().params.id)
if (!Number.isInteger(factionId) || factionId < 500000 || factionId > 599999) {
    throw createError({ statusCode: 404, statusMessage: 'Faction not found' })
}
const factionRequest = await useApiFetch<any>(`/api/faction/${factionId}`)

const {
    id, pending,
    entity: faction, stats, recentStats,
    activeTab, setTab, killlistRole,
} = useEntityPage('faction', {
    tabs,
    titleBase: f => f?.name || null,
    defaultTab: 'dashboard',
    idRange: [500000, 599999],
    withTopLists: false,
    withAccent: false,
}, factionRequest)

useSeoMeta({
    description: computed(() => {
        const f = faction.value
        if (!f?.name) return 'View faction kill statistics on EVE-KILL.'
        const s = stats.value
        let desc = `${f.name}`
        if (s?.losses) desc += ` — ${s.losses.toLocaleString('en-US')} losses`
        desc += ' — EVE Online faction stats on EVE-KILL.'
        return desc
    }),
    ogTitle: computed(() => faction.value?.name ? `${faction.value.name} — EVE-KILL` : 'Faction — EVE-KILL'),
    ogDescription: computed(() => faction.value?.name ? `${faction.value.name} — kills, losses, and combat stats in EVE Online.` : 'View faction stats on EVE-KILL.'),
    ogImage: computed(() => faction.value ? `/images/corporations/${faction.value.corporation_id || faction.value.faction_id}/logo?size=256` : ''),
    ogType: 'website',
    twitterCard: 'summary',
})

useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: [
            { name: 'Home', item: '/' },
            { name: faction.value?.name || 'Faction', item: `/faction/${id}` },
        ],
    }))),
])

const factionLogo = computed(() => faction.value
    ? `/images/corporations/${faction.value.corporation_id || faction.value.faction_id}/logo?size=256`
    : '')
</script>

<template>
    <div>
        <EntityHeader v-if="pending" loading />

        <div v-else-if="faction">
            <!-- ===== FACTION HEADER — MOBILE ===== -->
            <div class="md:hidden rounded-lg overflow-hidden mb-6 bg-white/[0.04] border border-white/[0.08]">
                <div class="p-4">
                    <div class="flex gap-3">
                        <EntityImageExpand :full-src="factionLogo" :alt="faction.name">
                            <img :src="factionLogo" :alt="faction.name" class="w-20 h-20 flex-shrink-0 rounded-lg shadow-lg" loading="eager">
                        </EntityImageExpand>
                        <div class="flex-1 min-w-0">
                            <h1 class="text-lg font-bold text-white leading-tight mb-1">{{ faction.name }}</h1>
                        </div>
                    </div>
                    <div class="flex flex-wrap gap-x-4 gap-y-1 mt-3 text-fine">
                        <div v-if="faction.station_count" class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:building" class="w-3 h-3" />
                            <span class="text-gray-300">{{ faction.station_count.toLocaleString('en-US') }} stations</span>
                        </div>
                        <div v-if="faction.station_system_count" class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:map-pin" class="w-3 h-3" />
                            <span class="text-gray-300">{{ faction.station_system_count.toLocaleString('en-US') }} systems</span>
                        </div>
                    </div>
                </div>
                <div v-if="stats" class="px-4 pb-4 pt-2 border-t border-white/[0.04]">
                    <EntityStatGrid variant="boxed">
                        <EntityStatBox icon="lucide:skull" icon-color="text-red-500" title="Losses (All Time)">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ stats.losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(stats.isk_lost) }}</span></div>
                        </EntityStatBox>
                        <EntityStatBox v-if="recentStats" icon="lucide:clock" icon-color="text-purple-500" title="Last 90 Days">
                            <div class="flex justify-between"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ recentStats.losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(recentStats.isk_lost) }}</span></div>
                        </EntityStatBox>
                    </EntityStatGrid>
                </div>
            </div>

            <!-- ===== FACTION HEADER — DESKTOP ===== -->
            <div class="hidden md:block">
                <EntityHeader>
                    <template #image>
                        <EntityImageExpand :full-src="factionLogo" :alt="faction.name">
                            <img :src="factionLogo" :alt="faction.name" class="w-40 h-40 rounded-lg shadow-lg" loading="eager">
                        </EntityImageExpand>
                    </template>

                    <div class="flex flex-row gap-6">
                        <div class="flex-1 min-w-0">
                            <!-- Description lives on the Dashboard tab, matching where the
                                 corporation and alliance pages keep theirs. Printing it here
                                 too meant the same paragraph twice on one screen. -->
                            <h1 class="text-3xl font-bold text-white mb-1">{{ faction.name }}</h1>
                        </div>

                        <div class="flex-shrink-0 min-w-[200px] space-y-2.5 text-xs">
                            <div v-if="faction.station_count">
                                <div class="flex items-center gap-1.5 text-gray-500 text-xs">
                                    <Icon name="lucide:building" class="w-3.5 h-3.5" />
                                    Stations
                                </div>
                                <div class="text-fine text-gray-300 ml-5">{{ faction.station_count.toLocaleString('en-US') }}</div>
                            </div>
                            <div v-if="faction.station_system_count">
                                <div class="flex items-center gap-1.5 text-gray-500 text-xs">
                                    <Icon name="lucide:map-pin" class="w-3.5 h-3.5" />
                                    Systems
                                </div>
                                <div class="text-fine text-gray-300 ml-5">{{ faction.station_system_count.toLocaleString('en-US') }}</div>
                            </div>
                        </div>
                    </div>

                    <template v-if="stats" #stats>
                        <EntityStatGrid variant="boxed">
                            <EntityStatBox icon="lucide:skull" icon-color="text-red-500" title="Losses (All Time)">
                                <div class="flex justify-between"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ stats.losses.toLocaleString('en-US') }}</span></div>
                                <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(stats.isk_lost) }}</span></div>
                            </EntityStatBox>
                            <EntityStatBox v-if="recentStats" icon="lucide:clock" icon-color="text-purple-500" title="Last 90 Days">
                                <div class="flex justify-between"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ recentStats.losses.toLocaleString('en-US') }}</span></div>
                                <div class="flex justify-between"><span class="text-fine text-gray-500">ISK Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(recentStats.isk_lost) }}</span></div>
                            </EntityStatBox>
                        </EntityStatGrid>
                    </template>
                </EntityHeader>
            </div>

            <!-- ===== TAB BAR ===== -->
            <EntityTabBar :tabs="tabs" :active-id="activeTab" @select="setTab" />

            <!-- ===== DASHBOARD ===== -->
            <div v-if="activeTab === 'dashboard'">
                <FactionDashboard :faction="faction" />
            </div>

            <!-- ===== KILL TABS ===== -->
            <div v-else>
                <KillList :entity-endpoint="`/api/entity/faction/${id}/killlist`"
                    victim-entity-type="faction" :victim-entity-id="id"
                    :entity-role="killlistRole"
                    :stream-topics="[`victim.${id}`, `attacker.${id}`]" />
            </div>
        </div>
    </div>
</template>
