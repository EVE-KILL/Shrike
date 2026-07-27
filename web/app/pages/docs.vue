<script setup lang="ts">
import type { ApiIndex, ApiCategory } from '~/composables/apiDocs'

useHead({ title: 'API Documentation' })
useSeoMeta({
    description: 'Browse and test the EVE-KILL public REST API. Self-describing, JSON, CORS-open. Includes killmails, characters, corporations, alliances, ships, battles, wars, leaderboards, search, and the full SDE.',
    ogTitle: 'API Documentation — EVE-KILL',
    ogDescription: 'Interactive explorer for the EVE-KILL public API.',
})

const config = useRuntimeConfig()
const baseUrl = ref<string>(config.public.publicApiUrl || 'https://eve-kill.com/api')

const index = ref<ApiIndex | null>(null)
const loading = ref(true)
const loadError = ref<string | null>(null)

const loadIndex = async () => {
    loading.value = true
    loadError.value = null
    try {
        const data = await apiFetch<ApiIndex>(`${baseUrl.value}/`)
        index.value = data
        // Trust the API's own self-reported baseUrl over our default — that
        // way switching the env override or pointing at localhost works
        // without code changes.
        if (data.baseUrl) baseUrl.value = data.baseUrl
    } catch (e: any) {
        loadError.value = e?.message ?? 'Failed to load API index'
    } finally {
        loading.value = false
    }
}

// Client-side only — we want devs to see the real public URLs in their
// network tab, not pre-rendered SSR markup.
onMounted(loadIndex)

const search = ref('')

const filteredCategories = computed<ApiCategory[]>(() => {
    if (!index.value) return []
    if (!search.value.trim()) return index.value.categories
    const q = search.value.toLowerCase()
    return index.value.categories
        .map(cat => ({
            ...cat,
            endpoints: cat.endpoints.filter(e =>
                e.path.toLowerCase().includes(q)
                || e.summary.toLowerCase().includes(q)
                || (e.description?.toLowerCase().includes(q) ?? false),
            ),
        }))
        .filter(cat => cat.endpoints.length > 0)
})

const totalEndpoints = computed(() =>
    filteredCategories.value.reduce((sum, c) => sum + c.endpoints.length, 0),
)

const scrollToCategory = (key: string) => {
    const el = document.getElementById(`cat-${key}`)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<template>
    <div class="max-w-6xl mx-auto py-4 px-2">
        <!-- Header -->
        <div class="glass-panel backdrop-blur-sm p-6 md:p-8 mb-6">
            <div class="flex items-start justify-between gap-4 flex-wrap">
                <div>
                    <h1 class="text-3xl md:text-4xl font-bold text-white">API Documentation</h1>
                    <p class="text-sm text-gray-400 mt-2 max-w-2xl">
                        Public read-only API powering eve-kill.com. JSON, CORS-open, no auth required.
                        Click any endpoint to expand it, fill in the parameters, and hit
                        <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20 text-[10px] font-medium align-middle">
                            <Icon name="lucide:play" class="text-[10px]" />
                            Try it
                        </span>
                        — requests fire straight against the live API.
                    </p>
                </div>
                <div v-if="!loading && index" class="text-xs text-gray-500 text-right">
                    <div>{{ index.name }}</div>
                    <div class="text-gray-600">v{{ index.version }}</div>
                    <a :href="baseUrl" target="_blank" rel="noopener" class="font-mono text-blue-400/80 hover:text-blue-300 transition-colors">
                        {{ baseUrl }}
                    </a>
                </div>
            </div>
        </div>

        <!-- Loading -->
        <div v-if="loading" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 text-center">
            <Icon name="lucide:loader-2" class="text-2xl text-blue-400 animate-spin" />
            <div class="text-sm text-gray-500 mt-3">Loading API index from {{ baseUrl }}…</div>
        </div>

        <!-- Error -->
        <div v-else-if="loadError" class="rounded-lg border border-red-500/[0.20] bg-red-500/[0.05] p-6">
            <div class="flex gap-3 items-start">
                <Icon name="lucide:alert-triangle" class="text-red-400 text-xl flex-shrink-0 mt-0.5" />
                <div class="flex-1">
                    <div class="text-sm text-red-300 font-medium">Failed to load API index</div>
                    <div class="text-xs text-red-300/70 mt-1">{{ loadError }}</div>
                    <button
                        class="mt-3 text-xs px-3 py-1.5 rounded bg-red-500/20 hover:bg-red-500/30 text-red-200 border border-red-500/30 transition-colors"
                        @click="loadIndex"
                    >
                        Retry
                    </button>
                </div>
            </div>
        </div>

        <!-- Loaded -->
        <div v-else-if="index" class="grid grid-cols-1 lg:grid-cols-[220px_1fr] gap-6">
            <!-- Sidebar nav (sticky) -->
            <aside class="lg:sticky lg:top-4 lg:self-start space-y-4">
                <!-- Search -->
                <div class="relative">
                    <Icon name="lucide:search" class="absolute left-2.5 top-1/2 -translate-y-1/2 text-xs text-gray-500 pointer-events-none" />
                    <input
                        v-model="search"
                        type="text"
                        placeholder="Filter endpoints…"
                        class="w-full bg-white/[0.04] border border-white/[0.10] rounded-md pl-7 pr-2.5 py-1.5 text-xs text-white placeholder-gray-600 outline-none focus:border-blue-500/40"
                    >
                </div>

                <!-- Category list -->
                <nav class="space-y-1">
                    <div class="text-[10px] uppercase tracking-wider text-gray-600 px-2 mb-1">
                        {{ totalEndpoints }} endpoints
                    </div>
                    <button
                        v-for="cat in filteredCategories"
                        :key="cat.key"
                        class="w-full flex items-center justify-between gap-2 px-2 py-1.5 rounded text-xs text-gray-400 hover:text-blue-300 hover:bg-blue-500/[0.06] transition-colors text-left"
                        @click="scrollToCategory(cat.key)"
                    >
                        <span class="truncate">{{ cat.label }}</span>
                        <span class="text-[10px] text-gray-600 font-mono flex-shrink-0">{{ cat.endpoints.length }}</span>
                    </button>
                    <div v-if="filteredCategories.length === 0" class="text-xs text-gray-600 px-2 py-3 text-center">
                        No endpoints match
                    </div>
                </nav>
            </aside>

            <!-- Main: categories + endpoints -->
            <main class="space-y-8 min-w-0">
                <section
                    v-for="cat in filteredCategories"
                    :id="`cat-${cat.key}`"
                    :key="cat.key"
                    class="scroll-mt-4"
                >
                    <div class="mb-3">
                        <h2 class="text-xl font-semibold text-white">{{ cat.label }}</h2>
                        <p v-if="cat.description" class="text-xs text-gray-500 mt-1">{{ cat.description }}</p>
                    </div>
                    <div class="space-y-2">
                        <ApiDocsEndpoint
                            v-for="ep in cat.endpoints"
                            :key="`${ep.method} ${ep.path}`"
                            :endpoint="ep"
                            :base-url="baseUrl"
                        />
                    </div>
                </section>
            </main>
        </div>
    </div>
</template>
