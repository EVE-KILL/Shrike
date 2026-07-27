<script setup lang="ts">
useHead({ title: 'Map' })
useSeoMeta({
    description: 'Interactive map of New Eden and the rest of EVE Online — every region, system, and connection.',
    ogTitle: 'Map',
})

interface Tab { id: string; label: string; description: string }

// Tabs that render a MapScopeView (need scope id matching /api/map/scope?type=...)
const SCOPE_TABS: Tab[] = [
    { id: 'new-eden', label: 'New Eden', description: 'K-Space + Pochven + Exordium' },
    { id: 'zarzakh', label: 'Zarzakh', description: 'Deathless space' },
    { id: 'wormhole', label: 'Wormhole', description: 'J-Space' },
    { id: 'abyssal', label: 'Abyssal', description: 'Abyssal deadspace' },
    { id: 'proving', label: 'Proving', description: 'Arena regions' },
]

// Region-list tab uses the existing /api/map/regions payload.
const route = useRoute()
const router = useRouter()
const initialTab = (typeof route.query.tab === 'string' ? route.query.tab : null) ?? 'new-eden'
const validTabs = new Set([...SCOPE_TABS.map(t => t.id), 'regions'])
const activeTab = ref<string>(validTabs.has(initialTab) ? initialTab : 'new-eden')

watch(activeTab, (v) => {
    router.replace({ query: { ...route.query, tab: v } })
})

const isScopeTab = computed(() => SCOPE_TABS.some(t => t.id === activeTab.value))

// Region-list data only loaded when the user actually visits that tab.
const regionsData = ref<any | null>(null)
const regionsPending = ref(false)
async function ensureRegionsLoaded() {
    if (regionsData.value || regionsPending.value) return
    regionsPending.value = true
    try {
        regionsData.value = await apiFetch('/api/map/regions')
    } finally {
        regionsPending.value = false
    }
}

watchEffect(() => {
    if (activeTab.value === 'regions') ensureRegionsLoaded()
})

const sections = computed(() => {
    const d = regionsData.value
    if (!d) return []
    return [
        { id: 'kspace', label: 'New Eden', description: 'K-Space regions', regions: d.kspace },
        { id: 'pochven', label: 'Pochven', description: 'Triglavian space', regions: d.pochven },
        { id: 'zarzakh', label: 'Zarzakh', description: 'Deathless space', regions: d.zarzakh },
        { id: 'wormhole', label: 'Wormhole Space', description: 'J-Space regions', regions: d.wormhole },
        { id: 'abyssal', label: 'Abyssal Deadspace', description: 'Abyssal regions', regions: d.abyssal },
        { id: 'proving', label: 'Proving Grounds', description: 'Arena regions', regions: d.proving },
    ].filter(s => s.regions?.length > 0)
})
</script>

<template>
    <div>
        <div class="glass-panel overflow-hidden mb-4">
            <div class="px-4 pt-3 pb-0 flex items-baseline justify-between flex-wrap gap-2">
                <h1 class="text-xl font-bold text-white">Map</h1>
                <span class="text-xs text-gray-500">click a system to drill into its region</span>
            </div>
            <div class="px-2 pt-2 flex gap-1 overflow-x-auto">
                <button
                    v-for="t in SCOPE_TABS"
                    :key="t.id"
                    type="button"
                    class="px-3 py-1.5 text-sm rounded-t whitespace-nowrap transition-colors"
                    :class="activeTab === t.id
                        ? 'bg-[#0a0a0f] text-white border border-white/[0.08] border-b-transparent'
                        : 'text-gray-400 hover:text-white hover:bg-white/[0.04]'"
                    @click="activeTab = t.id"
                >
                    {{ t.label }}
                </button>
                <button
                    type="button"
                    class="px-3 py-1.5 text-sm rounded-t whitespace-nowrap transition-colors"
                    :class="activeTab === 'regions'
                        ? 'bg-[#0a0a0f] text-white border border-white/[0.08] border-b-transparent'
                        : 'text-gray-400 hover:text-white hover:bg-white/[0.04]'"
                    @click="activeTab = 'regions'"
                >
                    Regions
                </button>
            </div>
        </div>

        <!-- Scope map view: re-mounts on tab change so the fetch runs cleanly. -->
        <MapScopeView v-if="isScopeTab" :key="activeTab" :type="activeTab" />

        <!-- Region grid view -->
        <template v-else-if="activeTab === 'regions'">
            <div v-if="regionsPending && !regionsData" class="h-64 rounded-lg bg-white/[0.04] animate-pulse" />
            <template v-else>
                <div v-for="section in sections" :key="section.id" class="mb-8">
                    <div class="mb-3 flex items-baseline gap-3">
                        <h2 class="text-lg font-semibold text-white">{{ section.label }}</h2>
                        <span class="text-xs text-gray-500">{{ section.description }} · {{ section.regions.length }} regions</span>
                    </div>

                    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-2">
                        <NuxtLink
                            v-for="r in section.regions"
                            :key="r.region_id"
                            :to="`/map/region/${r.region_id}`"
                            class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-3 hover:bg-blue-500/[0.08] hover:border-white/[0.15] transition-all group"
                        >
                            <div class="text-sm font-medium text-white group-hover:text-blue-400 transition-colors truncate" :class="pochvenClass(r.region_id)">{{ r.name }}</div>
                            <div class="text-fine text-gray-500 mt-0.5">{{ r.system_count }} systems</div>
                        </NuxtLink>
                    </div>
                </div>
            </template>
        </template>
    </div>
</template>
