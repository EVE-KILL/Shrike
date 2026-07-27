<script setup lang="ts">
import { KILL_LIST_TYPES } from '#shared/utils/killListTypes'

const route = useRoute()
const type = computed(() => (route.params.type as string) || 'latest')
const { isDomainMode } = useDomainConfig()
const customKilllistEndpoint = computed(() => isDomainMode.value ? '/api/custom/killlist' : undefined)
const customTopEndpoint = computed(() => isDomainMode.value ? '/api/custom/kills/top' : '/api/kills/top')
const customMvEndpoint = computed(() => isDomainMode.value ? '/api/custom/kills/most-valuable' : undefined)

const validTypes = new Set<string>(KILL_LIST_TYPES)

if (!validTypes.has(type.value)) {
    throw createError({ statusCode: 404, statusMessage: 'Page not found' })
}

const typeLabels: Record<string, string> = {
    latest: 'Latest Kills', highsec: 'High Sec Kills', lowsec: 'Low Sec Kills',
    nullsec: 'Null Sec Kills', wspace: 'Wormhole Kills', abyssal: 'Abyssal Kills',
    pochven: 'Pochven Kills', jove: 'Jove Space Kills',
    big: 'Big Kills', solo: 'Solo Kills', npc: 'NPC Kills',
    '5b': '5B+ Kills', '10b': '10B+ Kills',
    frigates: 'Frigate Kills', destroyers: 'Destroyer Kills', cruisers: 'Cruiser Kills',
    battlecruisers: 'Battlecruiser Kills', battleships: 'Battleship Kills',
    capitals: 'Capital Kills', freighters: 'Freighter Kills',
    supercarriers: 'Supercarrier Kills', titans: 'Titan Kills',
    citadels: 'Structure Kills', t1: 'T1 Kills', t2: 'T2 Kills', t3: 'T3 Kills',
    faction: 'Faction Kills',
}

const title = computed(() => typeLabels[type.value] || 'Kills')

useHead({ title: computed(() => title.value) })
useSeoMeta({
    description: computed(() => `Browse ${title.value.toLowerCase()} in EVE Online — real-time killmail feed with ship fits, ISK values, and combat details on EVE-KILL.`),
    ogTitle: computed(() => `${title.value} — EVE-KILL`),
    ogDescription: computed(() => `Real-time ${title.value.toLowerCase()} feed from EVE Online on EVE-KILL.`),
})

// NPC kills show victim-based top lists (attackers are NPCs, not players)
const isVictimBased = computed(() => type.value === 'npc')
const topCharLabel = computed(() => isVictimBased.value ? 'Top Victim Characters' : 'Top Characters')
const topCorpLabel = computed(() => isVictimBased.value ? 'Top Victim Corporations' : 'Top Corporations')
const topAllyLabel = computed(() => isVictimBased.value ? 'Top Victim Alliances' : 'Top Alliances')

// Mobile tab state with history
const validMobileTabs = new Set(['kills', 'top', 'valuable'])
const mobileTabFromHash = (hash: string) => {
    const t = hash.replace('#', '')
    return validMobileTabs.has(t) ? t as 'kills' | 'top' | 'valuable' : 'kills'
}
const mobileTab = ref<'kills' | 'top' | 'valuable'>(
    import.meta.client ? mobileTabFromHash(window.location.hash) : 'kills',
)

const mobileTabLabels: Record<string, string> = { kills: '', top: 'Top Lists', valuable: 'Valuable' }
useHead({ title: computed(() => {
    const label = mobileTabLabels[mobileTab.value]
    return label ? `${title.value} (${label})` : title.value
}) })

const setMobileTab = (tab: typeof mobileTab.value) => {
    mobileTab.value = tab
    const hash = tab === 'kills' ? '' : `#${tab}`
    window.history.pushState(null, '', hash || window.location.pathname)
}
if (import.meta.client) {
    window.addEventListener('popstate', () => {
        mobileTab.value = mobileTabFromHash(window.location.hash)
    })
}
</script>

<template>
    <div>
        <!-- Page title -->
        <h1 class="text-xl font-bold text-white mb-4">{{ title }}</h1>

        <!-- Most Valuable — desktop only (mobile gets it as a tab) -->
        <MostValuableFiltered :kill-type="type" :api-endpoint="customMvEndpoint" class="mb-6 hidden md:block" />

        <!-- ===== DESKTOP: sidebar + killlist ===== -->
        <div class="hidden md:grid grid-cols-[250px_1fr] gap-4">
            <div>
                <TopBox :title="topCharLabel" dataType="characters" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
                <TopBox :title="topCorpLabel" dataType="corporations" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
                <TopBox :title="topAllyLabel" dataType="alliances" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
            </div>
            <KillList :killlist-type="type" :key="type" :api-endpoint="customKilllistEndpoint" :stream-topics="[type]" />
        </div>

        <!-- ===== MOBILE: tabs ===== -->
        <div class="md:hidden">
            <div class="flex border-b border-white/[0.08] mb-4">
                <button
                    class="flex-1 flex items-center justify-center gap-2 py-3 text-sm font-medium transition-colors border-b-2"
                    :class="mobileTab === 'kills' ? 'text-white border-blue-400' : 'text-gray-500 border-transparent hover:text-blue-400'"
                    @click="setMobileTab('kills')"
                >
                    <Icon name="lucide:swords" class="text-base" />
                    Kills
                </button>
                <button
                    class="flex-1 flex items-center justify-center gap-2 py-3 text-sm font-medium transition-colors border-b-2"
                    :class="mobileTab === 'top' ? 'text-white border-blue-400' : 'text-gray-500 border-transparent hover:text-blue-400'"
                    @click="setMobileTab('top')"
                >
                    <Icon name="lucide:list-ordered" class="text-base" />
                    Top Lists
                </button>
                <button
                    class="flex-1 flex items-center justify-center gap-2 py-3 text-sm font-medium transition-colors border-b-2"
                    :class="mobileTab === 'valuable' ? 'text-white border-blue-400' : 'text-gray-500 border-transparent hover:text-blue-400'"
                    @click="setMobileTab('valuable')"
                >
                    <Icon name="lucide:gem" class="text-base" />
                    Valuable
                </button>
            </div>

            <div v-show="mobileTab === 'kills'">
                <KillList :killlist-type="type" :key="type" :api-endpoint="customKilllistEndpoint" :stream-topics="[type]" />
            </div>

            <div v-show="mobileTab === 'top'" class="space-y-4">
                <TopBox :title="topCharLabel" dataType="characters" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
                <TopBox :title="topCorpLabel" dataType="corporations" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
                <TopBox :title="topAllyLabel" dataType="alliances" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
            </div>

            <div v-show="mobileTab === 'valuable'">
                <MostValuableFiltered :kill-type="type" :api-endpoint="customMvEndpoint" />
            </div>
        </div>
    </div>
</template>
