<script setup lang="ts">
import { eveTools } from '#shared/utils/eveTools'

const { isOpen: isOpenReadonly, closeSearch, openSearch } = useSpotlightSearch()
const { track } = useAnalytics()

const isOpen = computed({
    get: () => isOpenReadonly.value,
    set: (v: boolean) => { if (v) openSearch(); else closeSearch() },
})

export interface SearchHit {
    id: string
    name: string
    ticker: string | null
    type: 'character' | 'corporation' | 'alliance' | 'faction' | 'system' | 'region' | 'constellation' | 'ship' | 'item'
}

interface SearchResponse {
    hits: SearchHit[]
    query: string
    processingTimeMs: number
    total: number
    entityCounts: Record<string, number>
}

const searchQuery = ref('')
const isLoading = ref(false)
const searchInput = ref<HTMLInputElement>()
const mobileSearchInput = ref<HTMLInputElement>()
const results = ref<SearchHit[]>([])
const entityCounts = ref<Record<string, number>>({})
const selectedIndex = ref(-1)

// An asynchronously opened dialog may mount its input after the open watcher.
const focusSearchInput = () => {
    if (!isOpen.value) return
    const input = [mobileSearchInput.value, searchInput.value].find(input => input?.getClientRects().length)
    input?.focus()
}
watch([searchInput, mobileSearchInput], focusSearchInput, { flush: 'post' })

// Recent searches
const recentSearches = ref<string[]>([])

onMounted(() => {
    if (import.meta.client) {
        try {
            const stored = localStorage.getItem('evekill-recent-searches')
            if (stored) {
                const parsed = JSON.parse(stored)
                if (Array.isArray(parsed)) recentSearches.value = parsed.slice(0, 5)
            }
        } catch { /* ignore */ }
    }
})

const saveRecentSearches = () => {
    if (import.meta.client) {
        try { localStorage.setItem('evekill-recent-searches', JSON.stringify(recentSearches.value)) }
        catch { /* ignore */ }
    }
}

const addToRecentSearches = (query: string) => {
    const trimmed = query.trim()
    if (trimmed && trimmed.length >= 2 && !recentSearches.value.includes(trimmed)) {
        recentSearches.value.unshift(trimmed)
        recentSearches.value = recentSearches.value.slice(0, 5)
        saveRecentSearches()
    }
}

const removeRecentSearch = (index: number) => {
    recentSearches.value.splice(index, 1)
    saveRecentSearches()
}

const selectRecentSearch = (query: string) => {
    searchQuery.value = query
    handleSearch()
}

const quickActions = [
    { name: 'Latest Kills', to: '/kills/latest', icon: 'lucide:zap' },
    { name: 'Big Kills', to: '/kills/big', icon: 'lucide:trending-up' },
    { name: 'Solo Kills', to: '/kills/solo', icon: 'lucide:swords' },
    { name: 'Battles', to: '/battles', icon: 'lucide:shield' },
    { name: 'Stats', to: '/stats', icon: 'lucide:bar-chart-3' },
    { name: 'Map', to: '/map', icon: 'lucide:map' },
    { name: 'Wars', to: '/wars', icon: 'lucide:flag' },
    { name: 'Advanced Search', to: '/advancedsearch', icon: 'lucide:search' },
]

// Search
let searchTimeout: ReturnType<typeof setTimeout>
let searchController: AbortController | null = null

const cancelSearch = () => {
    clearTimeout(searchTimeout)
    searchController?.abort()
    searchController = null
    isLoading.value = false
}
onScopeDispose(cancelSearch)

const handleSearch = () => {
    cancelSearch()
    const query = searchQuery.value.trim()
    if (!query || query.length < 3) {
        results.value = []
        entityCounts.value = {}
        return
    }
    searchTimeout = setTimeout(async () => {
        const controller = new AbortController()
        searchController = controller
        isLoading.value = true
        selectedIndex.value = -1
        try {
            const response = await apiFetch<SearchResponse>('/api/search', {
                params: { q: query },
                signal: controller.signal,
            })
            if (searchController !== controller) return
            results.value = response.hits || []
            entityCounts.value = response.entityCounts || {}
            addToRecentSearches(query)
            track('search.submit', { length: query.length, results: results.value.length })
        } catch {
            if (searchController !== controller) return
            results.value = []
            entityCounts.value = {}
        } finally {
            if (searchController === controller) {
                searchController = null
                isLoading.value = false
            }
        }
    }, 300)
}

// Generate URL for a search hit
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

// Navigate to a result
const selectResult = (hit: SearchHit) => {
    track('search.result_click', { type: hit.type })
    navigateTo(hitUrl(hit))
    closeAndReset()
}

// Keyboard navigation
const handleKeydown = (event: KeyboardEvent) => {
    const total = results.value.length
    if (event.key === 'ArrowDown') {
        event.preventDefault()
        selectedIndex.value = selectedIndex.value < total - 1 ? selectedIndex.value + 1 : 0
    } else if (event.key === 'ArrowUp') {
        event.preventDefault()
        selectedIndex.value = selectedIndex.value > 0 ? selectedIndex.value - 1 : total - 1
    } else if (event.key === 'Enter') {
        event.preventDefault()
        if (total === 1) {
            selectResult(results.value[0]!)
        } else if (selectedIndex.value >= 0 && results.value[selectedIndex.value]) {
            selectResult(results.value[selectedIndex.value]!)
        }
    }
}

watch(() => isOpen.value, (open) => {
    if (open) {
        nextTick(focusSearchInput)
    } else {
        searchQuery.value = ''
        results.value = []
        entityCounts.value = {}
        selectedIndex.value = -1
    }
}, { immediate: true })

const closeAndReset = () => { closeSearch() }

const navigateAndClose = (to: string) => {
    navigateTo(to)
    closeAndReset()
}

const spotlightContentProps = computed(() => ({
    searchQuery: searchQuery.value,
    isLoading: isLoading.value,
    results: results.value,
    entityCounts: entityCounts.value,
    selectedIndex: selectedIndex.value,
    quickActions,
    eveTools,
    recentSearches: recentSearches.value,
}))
</script>

<template>
    <!-- ===== DESKTOP: uses Modal ===== -->
    <Modal v-model="isOpen" max-width="max-w-2xl" no-padding top-align hide-below-md>
        <template #header>
            <div class="flex items-center gap-3 flex-1">
                <Icon name="lucide:search" class="text-lg text-gray-500 flex-shrink-0" />
                <input
                    ref="searchInput"
                    v-model="searchQuery"
                    type="text"
                    placeholder="Search characters, corporations, alliances, ships..."
                    class="flex-1 bg-transparent text-white text-sm outline-none placeholder-gray-600"
                    @input="handleSearch"
                    @keydown="handleKeydown"
                >
                <kbd class="flex-shrink-0 px-1.5 py-0.5 rounded text-fine font-medium text-gray-500 bg-white/[0.06] border border-white/[0.1]">ESC</kbd>
            </div>
        </template>

        <SpotlightContent
            v-bind="spotlightContentProps"
            @select-recent="selectRecentSearch"
            @remove-recent="removeRecentSearch"
            @navigate="navigateAndClose"
            @select-result="selectResult"
            @close="closeAndReset"
        />

        <template #footer>
            <div class="flex items-center justify-center gap-5">
                <div class="flex items-center gap-1.5">
                    <kbd class="inline-flex items-center justify-center w-5 h-5 rounded text-fine text-gray-500 bg-white/[0.06] border border-white/[0.1]">↑</kbd>
                    <kbd class="inline-flex items-center justify-center w-5 h-5 rounded text-fine text-gray-500 bg-white/[0.06] border border-white/[0.1]">↓</kbd>
                    <span class="text-xs text-gray-600 ml-0.5">Navigate</span>
                </div>
                <div class="flex items-center gap-1.5">
                    <kbd class="inline-flex items-center justify-center w-5 h-5 rounded text-fine text-gray-500 bg-white/[0.06] border border-white/[0.1]">↵</kbd>
                    <span class="text-xs text-gray-600 ml-0.5">Select</span>
                </div>
                <div class="flex items-center gap-1.5">
                    <kbd class="inline-flex items-center justify-center px-1 h-5 rounded text-fine text-gray-500 bg-white/[0.06] border border-white/[0.1]">ESC</kbd>
                    <span class="text-xs text-gray-600 ml-0.5">Close</span>
                </div>
            </div>
        </template>
    </Modal>

    <!-- ===== MOBILE: fullscreen takeover ===== -->
    <Teleport to="body">
        <Transition
            enter-active-class="transition duration-200 ease-out"
            enter-from-class="opacity-0"
            enter-to-class="opacity-100"
            leave-active-class="transition duration-150 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
        >
            <div v-if="isOpen" class="md:hidden fixed inset-0 z-[9999] bg-gray-950 flex flex-col">
                <div class="flex items-center gap-3 px-4 py-3 border-b border-white/[0.08]">
                    <button class="flex-shrink-0 text-gray-400" @click="closeAndReset">
                        <Icon name="lucide:arrow-left" class="text-xl" />
                    </button>
                    <div class="glass-panel flex-1 flex items-center gap-2 px-3 py-2">
                        <Icon name="lucide:search" class="text-base text-gray-500 flex-shrink-0" />
                        <input
                            ref="mobileSearchInput"
                            v-model="searchQuery"
                            type="text"
                            placeholder="Search..."
                            class="flex-1 bg-transparent text-white text-sm outline-none placeholder-gray-600"
                            @input="handleSearch"
                            @keydown="handleKeydown"
                        >
                        <button v-if="searchQuery" class="text-gray-500" aria-label="Clear search" @click="searchQuery = ''; handleSearch()">
                            <Icon name="lucide:x" class="text-sm" />
                        </button>
                    </div>
                </div>

                <div class="flex-1 overflow-y-auto">
                    <SpotlightContent
                        v-bind="spotlightContentProps"
                        mobile
                        @select-recent="selectRecentSearch"
                        @remove-recent="removeRecentSearch"
                        @navigate="navigateAndClose"
                        @select-result="selectResult"
                        @close="closeAndReset"
                    />
                </div>
            </div>
        </Transition>
    </Teleport>
</template>
