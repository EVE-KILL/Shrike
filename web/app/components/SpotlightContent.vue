<script setup lang="ts">
import type { SearchHit } from './SpotlightSearch.vue'
import type { EveTool } from '#shared/utils/eveTools'

const props = defineProps<{
    searchQuery: string
    isLoading: boolean
    results: SearchHit[]
    entityCounts: Record<string, number>
    selectedIndex: number
    quickActions: { name: string; to: string; icon: string }[]
    eveTools: EveTool[]
    recentSearches: string[]
    mobile?: boolean
}>()

const emit = defineEmits<{
    selectRecent: [query: string]
    removeRecent: [index: number]
    navigate: [to: string]
    selectResult: [hit: SearchHit]
    close: []
}>()

// Category config
const categoryConfig: Record<string, { label: string; icon: string }> = {
    ship: { label: 'Ships', icon: 'lucide:rocket' },
    item: { label: 'Items', icon: 'lucide:package' },
    character: { label: 'Characters', icon: 'lucide:user' },
    corporation: { label: 'Corporations', icon: 'lucide:building' },
    alliance: { label: 'Alliances', icon: 'lucide:users' },
    faction: { label: 'Factions', icon: 'lucide:flag' },
    system: { label: 'Systems', icon: 'lucide:map-pin' },
    constellation: { label: 'Constellations', icon: 'lucide:hexagon' },
    region: { label: 'Regions', icon: 'lucide:globe' },
}

// Active type filter (local to spotlight); 'all' shows every category
const activeFilter = ref<string>('all')

// Reset filter whenever the query changes so new searches start unfiltered
watch(() => props.searchQuery, () => { activeFilter.value = 'all' })

// Group results by type, preserving API order within each group
const categorizedResults = computed(() => {
    const groups: Record<string, SearchHit[]> = {}
    const filtered = activeFilter.value === 'all'
        ? props.results
        : props.results.filter(h => h.type === activeFilter.value)
    for (const hit of filtered) {
        if (!groups[hit.type]) groups[hit.type] = []
        groups[hit.type]!.push(hit)
    }
    // Order categories by the entityCounts keys (API returns them ranked)
    return Object.entries(groups).map(([type, items]) => ({
        type,
        ...categoryConfig[type] || { label: type, icon: 'lucide:circle' },
        items,
    }))
})

// Track global index across all categories for keyboard nav
const getGlobalIndex = (categoryType: string, itemIndex: number): number => {
    let idx = 0
    for (const cat of categorizedResults.value) {
        if (cat.type === categoryType) return idx + itemIndex
        idx += cat.items.length
    }
    return -1
}

// EVE image URLs
const getImageUrl = (hit: SearchHit): string | null => {
    const numericId = hit.id.split('_').slice(1).join('_')
    switch (hit.type) {
        case 'character': return `/images/characters/${numericId}/portrait?size=64`
        case 'corporation': return `/images/corporations/${numericId}/logo?size=64`
        case 'alliance': return `/images/alliances/${numericId}/logo?size=64`
        case 'ship':
        case 'item': return `/images/types/${numericId}/icon?size=64`
        case 'faction': return `/images/corporations/${numericId}/logo?size=64`
        case 'system': return `/images/systems/${numericId}?size=64`
        case 'region': return `/images/regions/${numericId}?size=64`
        case 'constellation': return `/images/constellations/${numericId}?size=64`
        default: return null
    }
}
</script>

<template>
    <!-- Loading -->
    <div v-if="isLoading" class="flex items-center justify-center gap-2 py-12 text-gray-500">
        <Icon name="lucide:loader" class="text-lg animate-spin" />
        <span class="text-sm">Searching...</span>
    </div>

    <!-- Default view (no query) -->
    <div v-else-if="!searchQuery" class="p-5 space-y-5">
        <!-- Quick Actions -->
        <div>
            <div class="px-2 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                Quick Actions
            </div>
            <div :class="mobile ? 'grid grid-cols-2 gap-2' : 'grid grid-cols-4 gap-1'">
                <button
                    v-for="action in quickActions"
                    :key="action.to"
                    class="flex items-center gap-3 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors"
                    :class="mobile ? 'px-4 py-3' : 'flex-col gap-2 py-3 px-2'"
                    @click="emit('navigate', action.to)"
                >
                    <Icon :name="action.icon" :class="mobile ? 'text-lg' : 'text-xl'" />
                    <span :class="mobile ? 'text-sm' : 'text-xs'">{{ action.name }}</span>
                </button>
            </div>
        </div>

        <!-- Recent Searches + EVE Tools -->
        <div :class="mobile ? 'space-y-5' : 'flex gap-6'">
            <div v-if="recentSearches.length > 0" class="flex-1 min-w-0">
                <div class="px-2 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                    Recent Searches
                </div>
                <div class="space-y-px">
                    <div
                        v-for="(recent, index) in recentSearches"
                        :key="index"
                        class="flex items-center gap-2 w-full px-2 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors group cursor-pointer"
                        :class="mobile ? 'py-2.5 text-sm' : 'py-1.5 text-sm'"
                        @click="emit('selectRecent', recent)"
                    >
                        <Icon name="lucide:clock" class="text-sm opacity-40 flex-shrink-0" />
                        <span class="flex-1 text-left truncate">{{ recent }}</span>
                        <button
                            class="opacity-0 group-hover:opacity-100 text-gray-600 hover:text-blue-400 transition-opacity flex-shrink-0"
                            @click.stop="emit('removeRecent', index)"
                        >
                            <Icon name="lucide:x" class="text-xs" />
                        </button>
                    </div>
                </div>
            </div>

            <div class="flex-1 min-w-0">
                <div class="px-2 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                    EVE Tools
                </div>
                <div :class="mobile ? 'grid grid-cols-3 gap-2' : 'grid grid-cols-3 gap-1'">
                    <a
                        v-for="tool in eveTools"
                        :key="tool.name"
                        :href="tool.url"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="flex flex-col items-center gap-1.5 py-2 px-1 rounded-md text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors"
                        @click="emit('close')"
                    >
                        <NuxtImg :src="tool.icon" alt="" width="16" height="16" class="w-4 h-4 object-contain grayscale opacity-60" />
                        <span class="text-fine">{{ tool.label || tool.name }}</span>
                    </a>
                </div>
            </div>
        </div>
    </div>

    <!-- Min characters hint -->
    <div v-else-if="searchQuery.length > 0 && searchQuery.length < 3" class="flex flex-col items-center justify-center py-12 text-gray-600">
        <Icon name="lucide:type" class="text-2xl mb-3 opacity-40" />
        <p class="text-sm">Type at least 3 characters to search</p>
    </div>

    <!-- No results -->
    <div v-else-if="!isLoading && searchQuery.length >= 3 && results.length === 0" class="flex flex-col items-center justify-center py-12 text-gray-500">
        <Icon name="lucide:search-x" class="text-3xl mb-3 opacity-40" />
        <p class="text-sm font-medium">No results found</p>
        <p class="text-xs text-gray-600 mt-1">No results for "{{ searchQuery }}"</p>
    </div>

    <!-- ===== RESULTS ===== -->
    <div v-else-if="results.length > 0" class="p-3">
        <!-- Category filter pills (clickable) -->
        <div v-if="Object.keys(entityCounts).length > 1" class="flex flex-wrap gap-1.5 mb-3 px-1">
            <button
                type="button"
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-fine font-medium border transition-colors"
                :class="activeFilter === 'all'
                    ? 'text-blue-400 bg-blue-500/15 border-blue-500/30'
                    : 'text-gray-500 bg-white/[0.04] border-white/[0.08] hover:text-blue-400 hover:border-blue-500/20'"
                @click="activeFilter = 'all'"
            >
                <Icon name="lucide:layers" class="text-fine" />
                All {{ results.length }}
            </button>
            <button
                v-for="(count, type) in entityCounts"
                :key="type"
                type="button"
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-fine font-medium border transition-colors"
                :class="activeFilter === type
                    ? 'text-blue-400 bg-blue-500/15 border-blue-500/30'
                    : 'text-gray-500 bg-white/[0.04] border-white/[0.08] hover:text-blue-400 hover:border-blue-500/20'"
                @click="activeFilter = activeFilter === type ? 'all' : type"
            >
                <Icon :name="categoryConfig[type]?.icon || 'lucide:circle'" class="text-fine" />
                {{ categoryConfig[type]?.label || type }} {{ count }}
            </button>
        </div>

        <!-- Categorized results -->
        <div class="space-y-3">
            <div v-for="category in categorizedResults" :key="category.type">
                <!-- Category header -->
                <div class="flex items-center gap-1.5 px-2 pb-1.5 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                    <Icon :name="category.icon" class="text-fine" />
                    {{ category.label }}
                </div>

                <!-- Results -->
                <div class="space-y-px">
                    <div
                        v-for="(hit, idx) in category.items"
                        :key="hit.id"
                        class="flex items-center gap-3 px-2 py-1.5 rounded-md cursor-pointer transition-colors"
                        :class="[
                            getGlobalIndex(category.type, idx) === selectedIndex
                                ? 'bg-blue-500/15 text-white'
                                : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]',
                        ]"
                        @click="emit('selectResult', hit)"
                        @mouseenter="selectedIndex === -1 ? null : undefined"
                    >
                        <!-- Avatar/Icon -->
                        <div class="flex-shrink-0 w-8 h-8 rounded overflow-hidden bg-white/[0.04]">
                            <EveImage
                                v-if="getImageUrl(hit)"
                                :src="getImageUrl(hit)!"
                                :size="32"
                                :alt="hit.name"
                                class="w-full h-full object-cover"
                            />
                            <div v-else class="w-full h-full flex items-center justify-center text-gray-600">
                                <Icon :name="category.icon" class="text-sm" />
                            </div>
                        </div>

                        <!-- Name + meta -->
                        <div class="flex-1 min-w-0">
                            <div class="text-sm font-medium truncate">{{ hit.name }}</div>
                            <div v-if="hit.ticker" class="text-xs text-gray-600">[{{ hit.ticker }}]</div>
                        </div>

                        <!-- Type badge -->
                        <span class="flex-shrink-0 text-fine text-gray-600 capitalize">{{ hit.type }}</span>

                        <!-- Arrow -->
                        <Icon name="lucide:arrow-up-right" class="flex-shrink-0 text-xs opacity-30" />
                    </div>
                </div>
            </div>
        </div>

        <!-- View all results link -->
        <div class="mt-3 px-2">
            <button
                class="w-full flex items-center justify-center gap-2 py-2.5 rounded-lg text-xs font-medium
                       text-blue-400/80 bg-blue-500/[0.06] border border-blue-500/[0.1]
                       hover:bg-blue-500/[0.1] hover:text-blue-400 transition-colors"
                @click="emit('navigate', `/search?q=${encodeURIComponent(searchQuery)}`)"
            >
                <Icon name="lucide:arrow-right" class="text-xs" />
                View all results
            </button>
        </div>
    </div>
</template>
