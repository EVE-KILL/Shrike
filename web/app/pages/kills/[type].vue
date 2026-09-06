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
    'timezone-au': 'AU Timezone Kills', 'timezone-ru': 'RU Timezone Kills',
    'timezone-eu': 'EU Timezone Kills', 'timezone-us-east': 'US East Timezone Kills',
    'timezone-us-west': 'US West Timezone Kills',
    big: 'Big Kills', solo: 'Solo Kills', npc: 'NPC Kills',
    'attackers-1': '1-Attacker Non-solo Kills', 'attackers-2-4': '2–4 Attacker Kills',
    'attackers-5-9': '5–9 Attacker Kills', 'attackers-10-24': '10–24 Attacker Kills',
    'attackers-25-49': '25–49 Attacker Kills', 'attackers-50-99': '50–99 Attacker Kills',
    'attackers-100-999': '100–999 Attacker Kills', 'attackers-1000-plus': '1,000+ Attacker Kills',
    pvp: 'PvP Kills', ganked: 'Highsec Ganks',
    '5b': '5B+ Kills', '10b': '10B+ Kills',
    'under-1b': 'Under 1B ISK Kills', '1b-5b': '1B–5B ISK Kills',
    '5b-10b': '5B–10B ISK Kills', '10b-100b': '10B–100B ISK Kills',
    '100b-1t': '100B–1T ISK Kills', '1t-plus': '1T+ ISK Kills',
    'category-deployable': 'Deployable Kills', 'category-drone': 'Drone Kills',
    'category-fighter': 'Fighter Kills', 'category-orbital': 'Orbital Kills',
    'category-starbase': 'Starbase Kills', 'category-ship': 'Ship Kills',
    'category-sovereignty': 'Sovereignty Structure Kills',
    'category-structure': 'Structure Category Kills', 'category-infantry': 'Infantry Kills',
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
const mobileTab = ref<'kills' | 'top' | 'valuable'>('kills')

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
const syncMobileTab = () => { mobileTab.value = mobileTabFromHash(window.location.hash) }
onMounted(syncMobileTab)
if (import.meta.client) useEventListener(window, 'popstate', syncMobileTab)

</script>

<template>
    <div>
        <!-- Page title -->
        <h1 class="text-xl font-bold text-white mb-4">{{ title }}</h1>

        <!-- One set of widgets for both viewports; CSS controls mobile tabs. -->
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

        </div>
        <div :class="mobileTab === 'valuable' ? 'block' : 'hidden md:block'">
            <MostValuableFiltered :key="type" :kill-type="type" :api-endpoint="customMvEndpoint" />
        </div>
        <div class="grid grid-cols-1 md:grid-cols-[250px_minmax(0,1fr)] gap-4">
            <div :class="mobileTab === 'top' ? 'block' : 'hidden md:block'">
                <DeferredTopBox :title="topCharLabel" dataType="characters" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
                <DeferredTopBox :title="topCorpLabel" dataType="corporations" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
                <DeferredTopBox :title="topAllyLabel" dataType="alliances" :limit="10" :apiEndpoint="customTopEndpoint" :killType="type" />
            </div>
            <div class="min-w-0" :class="mobileTab === 'kills' ? 'block' : 'hidden md:block'">
                <KillList :killlist-type="type" :key="type" :api-endpoint="customKilllistEndpoint" :stream-topics="[type]" />
            </div>
        </div>
    </div>
</template>
