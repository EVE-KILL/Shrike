<script setup lang="ts">
interface SearchHit {
    id: string
    name: string
    ticker: string | null
    type: 'character' | 'corporation' | 'alliance' | 'faction' | 'system' | 'region' | 'constellation' | 'ship' | 'item'
    // Enriched by backend for characters
    corporation_id?: number | null
    corporation_name?: string | null
    corporation_ticker?: string | null
    alliance_id?: number | null
    alliance_name?: string | null
    alliance_ticker?: string | null
}

interface SearchResponse {
    hits: SearchHit[]
    query: string
    processingTimeMs: number
    total: number
    entityCounts: Record<string, number>
}

const route = useRoute()
const router = useRouter()

const query = ref((route.query.q as string) || '')
const activeFilter = ref<string>('all')
const results = ref<SearchHit[]>([])
const entityCounts = ref<Record<string, number>>({})
const isLoading = ref(false)
const hasSearched = ref(false)
const processingTime = ref(0)
const total = ref(0)

const searchInput = ref<HTMLInputElement>()

// Image URL for a search hit
const hitImage = (hit: SearchHit): string => {
    const numericId = hit.id.split('_').slice(1).join('_')
    switch (hit.type) {
        case 'character': return `/images/characters/${numericId}/portrait?size=512`
        case 'corporation': return `/images/corporations/${numericId}/logo?size=256`
        case 'alliance': return `/images/alliances/${numericId}/logo?size=256`
        case 'faction': return `/images/corporations/${numericId}/logo?size=256`
        case 'ship': return `/images/types/${numericId}/overlayrender?size=512`
        case 'item': return `/images/types/${numericId}/icon?size=64`
        case 'system': return `/images/systems/${numericId}?size=128`
        case 'region': return `/images/regions/${numericId}?size=128`
        case 'constellation': return `/images/constellations/${numericId}?size=128`
        default: return ''
    }
}

// URL for navigating to a result
const hitUrl = (hit: SearchHit): string => {
    const numericId = hit.id.split('_').slice(1).join('_')
    switch (hit.type) {
        case 'character': return `/character/${numericId}`
        case 'corporation': return `/corporation/${numericId}`
        case 'alliance': return `/alliance/${numericId}`
        case 'faction': return `/faction/${numericId}`
        case 'system': return `/system/${numericId}`
        case 'region': return `/region/${numericId}`
        case 'constellation': return `/constellation/${numericId}`
        case 'ship': return `/item/${numericId}`
        case 'item': return `/item/${numericId}`
        default: return '/'
    }
}

// Type label + icon + color mapping
const typeConfig: Record<string, { label: string; icon: string; color: string; bgColor: string }> = {
    character: { label: 'Characters', icon: 'lucide:user', color: 'text-blue-400', bgColor: 'bg-blue-500/20' },
    corporation: { label: 'Corporations', icon: 'lucide:building-2', color: 'text-emerald-400', bgColor: 'bg-emerald-500/20' },
    alliance: { label: 'Alliances', icon: 'lucide:shield', color: 'text-purple-400', bgColor: 'bg-purple-500/20' },
    faction: { label: 'Factions', icon: 'lucide:flag', color: 'text-amber-400', bgColor: 'bg-amber-500/20' },
    system: { label: 'Systems', icon: 'lucide:map-pin', color: 'text-cyan-400', bgColor: 'bg-cyan-500/20' },
    region: { label: 'Regions', icon: 'lucide:map', color: 'text-orange-400', bgColor: 'bg-orange-500/20' },
    constellation: { label: 'Constellations', icon: 'lucide:compass', color: 'text-teal-400', bgColor: 'bg-teal-500/20' },
    ship: { label: 'Ships', icon: 'lucide:rocket', color: 'text-red-400', bgColor: 'bg-red-500/20' },
    item: { label: 'Items', icon: 'lucide:box', color: 'text-gray-400', bgColor: 'bg-gray-500/20' },
}

// Whether this type has a big image (portrait/logo/render) or just an icon
const hasLargeImage = (type: string) => ['character', 'corporation', 'alliance', 'faction', 'ship', 'system', 'region', 'constellation'].includes(type)

// Filtered results
const filteredResults = computed(() => {
    if (activeFilter.value === 'all') return results.value
    return results.value.filter(h => h.type === activeFilter.value)
})

// Filter tabs with counts
const filterTabs = computed(() => {
    const tabs: { key: string; label: string; count: number; icon: string; color: string }[] = [
        { key: 'all', label: 'All', count: total.value, icon: 'lucide:layers', color: 'text-white' },
    ]
    for (const [type, count] of Object.entries(entityCounts.value)) {
        const config = typeConfig[type]
        if (config && count > 0) {
            tabs.push({ key: type, label: config.label, count, icon: config.icon, color: config.color })
        }
    }
    return tabs
})

// Perform search
let searchTimeout: ReturnType<typeof setTimeout>

const doSearch = () => {
    const q = query.value.trim()
    if (!q || q.length < 2) {
        results.value = []
        entityCounts.value = {}
        hasSearched.value = false
        return
    }

    clearTimeout(searchTimeout)
    searchTimeout = setTimeout(async () => {
        isLoading.value = true
        hasSearched.value = true
        activeFilter.value = 'all'
        try {
            const response = await apiFetch<SearchResponse>('/api/search', {
                params: { q },
            })
            results.value = response.hits || []
            entityCounts.value = response.entityCounts || {}
            processingTime.value = response.processingTimeMs || 0
            total.value = response.total || 0

            // Update URL without navigation
            router.replace({ query: { q } })
        } catch {
            results.value = []
            entityCounts.value = {}
        } finally {
            isLoading.value = false
        }
    }, 200)
}

// Search on initial load if ?q= is present
if (query.value) {
    doSearch()
}

// SEO
useHead({ title: computed(() => query.value ? `Search: ${query.value}` : 'Search') })
useSeoMeta({
    description: 'Search EVE-KILL — find pilots, corporations, alliances, ships, systems, and regions across New Eden.',
    ogTitle: 'Search — EVE-KILL',
    ogDescription: 'Search for pilots, corporations, alliances, ships, and more on EVE-KILL.',
})

useSchemaOrg([
    defineWebPage({
        '@type': 'SearchResultsPage',
        'name': 'Search — EVE-KILL',
    }),
])
</script>

<template>
    <div class="w-full">
        <h1 class="sr-only">Search — EVE-KILL</h1>
        <!-- Search header -->
        <div class="mb-6">
            <div class="relative">
                <Icon name="lucide:search" class="absolute left-4 top-1/2 -translate-y-1/2 text-gray-500 text-lg pointer-events-none" />
                <input
                    ref="searchInput"
                    v-model="query"
                    type="text"
                    placeholder="Search pilots, corporations, alliances, ships, systems..."
                    class="w-full pl-12 pr-4 py-4 rounded-xl bg-white/[0.04] border border-white/[0.08] text-white text-base
                           placeholder:text-gray-600 focus:outline-none focus:border-blue-500/40 focus:bg-white/[0.06]
                           transition-all backdrop-blur-sm"
                    @input="doSearch"
                    @keydown.enter.prevent
                >
                <!-- Loading spinner -->
                <div v-if="isLoading" class="absolute right-4 top-1/2 -translate-y-1/2">
                    <Icon name="lucide:loader-2" class="text-blue-400 text-lg animate-spin" />
                </div>
                <!-- Clear button -->
                <button
                    v-else-if="query"
                    class="absolute right-4 top-1/2 -translate-y-1/2 text-gray-500 hover:text-blue-400 transition-colors"
                    @click="query = ''; results = []; entityCounts = {}; hasSearched = false; router.replace({ query: {} })"
                >
                    <Icon name="lucide:x" class="text-lg" />
                </button>
            </div>

            <!-- Stats bar -->
            <div v-if="hasSearched && !isLoading" class="mt-2 flex items-center gap-3 text-xs text-gray-600 px-1">
                <span>{{ total.toLocaleString() }} result{{ total !== 1 ? 's' : '' }}</span>
                <span class="text-gray-700">&middot;</span>
                <span>{{ processingTime }}ms</span>
            </div>
        </div>

        <!-- Filter tabs -->
        <div v-if="hasSearched && filterTabs.length > 1" class="mb-5 flex gap-1.5 overflow-x-auto pb-1 scrollbar-none">
            <button
                v-for="tab in filterTabs"
                :key="tab.key"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-all"
                :class="activeFilter === tab.key
                    ? 'bg-blue-500/15 text-blue-400 border border-blue-500/30'
                    : 'bg-white/[0.03] text-gray-500 border border-transparent hover:bg-blue-500/[0.06] hover:text-blue-400'"
                @click="activeFilter = tab.key"
            >
                <Icon :name="tab.icon" class="text-sm" :class="activeFilter === tab.key ? tab.color : ''" />
                {{ tab.label }}
                <span class="ml-0.5 text-fine tabular-nums opacity-60">{{ tab.count }}</span>
            </button>
        </div>

        <!-- Results grid -->
        <div v-if="filteredResults.length" class="grid gap-3 grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
            <NuxtLink
                v-for="hit in filteredResults"
                :key="hit.id"
                :to="hitUrl(hit)"
                class="group relative rounded-xl border border-white/[0.08] overflow-hidden bg-white/[0.02]
                       hover:border-white/[0.15] hover:bg-blue-500/[0.04] transition-all duration-200
                       hover:shadow-lg hover:shadow-black/30 hover:-translate-y-0.5"
            >
                <!-- Image area -->
                <div class="relative aspect-square overflow-hidden" :class="hasLargeImage(hit.type) ? '' : 'bg-white/[0.03]'">
                    <!-- Large image types: character portrait / corp-alliance logo / ship render -->
                    <img
                        v-if="hasLargeImage(hit.type)"
                        :src="hitImage(hit)"
                        :alt="hit.name"
                        class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
                        loading="lazy"
                    >
                    <!-- Small image types: icon centered on dark bg -->
                    <div v-else class="w-full h-full flex items-center justify-center">
                        <img
                            v-if="hitImage(hit)"
                            :src="hitImage(hit)"
                            :alt="hit.name"
                            class="w-16 h-16 rounded-lg"
                            loading="lazy"
                        >
                        <Icon v-else :name="typeConfig[hit.type]?.icon || 'lucide:circle'" class="text-5xl text-gray-700" />
                    </div>

                    <!-- Gradient overlay -->
                    <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/20 to-transparent pointer-events-none"></div>

                    <!-- Type badge top-left -->
                    <div class="absolute top-2 left-2">
                        <span
                            class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-fine font-bold uppercase backdrop-blur-sm"
                            :class="[typeConfig[hit.type]?.bgColor || 'bg-gray-500/20', typeConfig[hit.type]?.color || 'text-gray-400']"
                        >
                            <Icon :name="typeConfig[hit.type]?.icon || 'lucide:circle'" class="text-fine" />
                            {{ hit.type }}
                        </span>
                    </div>

                    <!-- Corp/Alliance logos (characters + corps) -->
                    <div v-if="hit.corporation_id || (hit.type === 'corporation' && hit.alliance_id)" class="absolute top-2 right-2 flex gap-1 z-10">
                        <NuxtLink v-if="hit.corporation_id" :to="`/corporation/${hit.corporation_id}`"
                            class="w-7 h-7 rounded-md overflow-hidden bg-black/60 border border-white/[0.1] hover:border-blue-400/40 transition-all hover:scale-110"
                            @click.stop
                        >
                            <img :src="`/images/corporations/${hit.corporation_id}/logo?size=64`" class="w-full h-full" loading="lazy">
                        </NuxtLink>
                        <NuxtLink v-if="hit.alliance_id" :to="`/alliance/${hit.alliance_id}`"
                            class="w-7 h-7 rounded-md overflow-hidden bg-black/60 border border-white/[0.1] hover:border-blue-400/40 transition-all hover:scale-110"
                            @click.stop
                        >
                            <img :src="`/images/alliances/${hit.alliance_id}/logo?size=64`" class="w-full h-full" loading="lazy">
                        </NuxtLink>
                    </div>

                    <!-- Name + affiliation overlay bottom -->
                    <div class="absolute inset-x-0 bottom-0 p-3 space-y-0.5">
                        <div class="text-sm text-white font-semibold drop-shadow-lg truncate group-hover:text-blue-400 transition-colors">
                            {{ hit.name }}
                        </div>
                        <div v-if="hit.ticker" class="text-xs text-gray-400/90 drop-shadow truncate">
                            [{{ hit.ticker }}]
                        </div>
                        <!-- Character: show corp + alliance -->
                        <div v-if="hit.type === 'character' && hit.corporation_name" class="text-fine text-gray-400/80 drop-shadow truncate">
                            {{ hit.corporation_name }}<template v-if="hit.corporation_ticker"> [{{ hit.corporation_ticker }}]</template>
                        </div>
                        <div v-if="hit.type === 'character' && hit.alliance_name" class="text-fine text-gray-500/70 drop-shadow truncate">
                            {{ hit.alliance_name }}<template v-if="hit.alliance_ticker"> [{{ hit.alliance_ticker }}]</template>
                        </div>
                        <!-- Corporation: show alliance -->
                        <div v-if="hit.type === 'corporation' && hit.alliance_name" class="text-fine text-gray-400/80 drop-shadow truncate">
                            {{ hit.alliance_name }}<template v-if="hit.alliance_ticker"> [{{ hit.alliance_ticker }}]</template>
                        </div>
                    </div>
                </div>
            </NuxtLink>
        </div>

        <!-- Empty state -->
        <div v-else-if="hasSearched && !isLoading" class="flex flex-col items-center justify-center py-20">
            <Icon name="lucide:search-x" class="text-4xl text-gray-700 mb-4" />
            <p class="text-gray-500 text-sm">No results found for "{{ query }}"</p>
            <p class="text-gray-600 text-xs mt-1">Try a different search term or check your spelling</p>
        </div>

        <!-- Initial state (no search yet) -->
        <div v-else-if="!hasSearched && !isLoading" class="flex flex-col items-center justify-center py-20">
            <Icon name="lucide:compass" class="text-5xl text-gray-700/50 mb-5" />
            <p class="text-gray-500 text-sm mb-6">Search across all of New Eden</p>
            <div class="grid gap-2 grid-cols-2 md:grid-cols-3">
                <button
                    v-for="suggestion in ['Pandemic Legion', 'Goonswarm', 'Jita', 'Hel', 'Ishtar', 'Delve']"
                    :key="suggestion"
                    class="px-4 py-2 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-gray-500
                           hover:bg-blue-500/[0.06] hover:text-blue-400 hover:border-white/[0.1] transition-all"
                    @click="query = suggestion; doSearch()"
                >
                    {{ suggestion }}
                </button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.scrollbar-none::-webkit-scrollbar { display: none; }
.scrollbar-none { scrollbar-width: none; }
</style>
