<script setup lang="ts">
import type { ApiEndpoint } from '~/composables/apiDocs'
import { buildDocsModel, type DocsModel, type DocsTag } from '~/composables/openapiDocs'

useHead({ title: 'API Documentation' })
useSeoMeta({
    description: 'Browse and test the EVE-KILL API. Self-describing, JSON, CORS-open. Killmails, characters, corporations, alliances, ships, battles, wars, leaderboards, search, and the full SDE.',
    ogTitle: 'API Documentation — EVE-KILL',
    ogDescription: 'Interactive explorer for the EVE-KILL API.',
})

const config = useRuntimeConfig()
const baseUrl = ref<string>(config.public.publicApiUrl || '/api')

const model = ref<DocsModel | null>(null)
const loading = ref(true)
const loadError = ref<string | null>(null)

const loadSpec = async () => {
    loading.value = true
    loadError.value = null
    try {
        const doc = await $fetch<any>(`${baseUrl.value}/openapi.json`)
        model.value = buildDocsModel(doc)
    } catch (e: any) {
        loadError.value = e?.message ?? 'Failed to load the API document'
    } finally {
        loading.value = false
    }
}

// Client-side only, matching the previous page: developers should see the real
// request in their network tab rather than markup rendered on the server.
onMounted(loadSpec)

const search = ref('')
const activeTag = ref<string | null>(null)

/** Groups filtered by the search box. Empty tags and groups drop out. */
const groups = computed(() => {
    if (!model.value) return []
    const query = search.value.trim().toLowerCase()
    if (!query) return model.value.groups

    const matches = (endpoint: ApiEndpoint) =>
        endpoint.path.toLowerCase().includes(query)
        || endpoint.summary.toLowerCase().includes(query)
        || endpoint.method.toLowerCase() === query
        || (endpoint.description?.toLowerCase().includes(query) ?? false)

    return model.value.groups
        .map(group => ({
            ...group,
            tags: group.tags
                .map(tag => ({ ...tag, endpoints: tag.endpoints.filter(matches) }))
                .filter(tag => tag.endpoints.length > 0),
        }))
        .filter(group => group.tags.length > 0)
})

const visibleCount = computed(() =>
    groups.value.reduce(
        (sum, group) => sum + group.tags.reduce((n, tag) => n + tag.endpoints.length, 0),
        0,
    ),
)

// Collapsed by default: 441 endpoints expanded at once is not navigation, it
// is a wall. Searching opens everything that matched instead.
const openGroups = ref<Set<string>>(new Set())
const toggleGroup = (name: string) => {
    const next = new Set(openGroups.value)
    next.has(name) ? next.delete(name) : next.add(name)
    openGroups.value = next
}
const isOpen = (name: string) => Boolean(search.value.trim()) || openGroups.value.has(name)

const selectTag = (tag: DocsTag) => {
    activeTag.value = tag.name
    nextTick(() => {
        document.getElementById(`tag-${tag.name}`)?.scrollIntoView({
            behavior: 'smooth',
            block: 'start',
        })
    })
}

const tagSlug = (name: string) => name.replace(/\s+/g, '-')
</script>

<template>
    <div class="w-full">
        <PageHeader class="mb-6" title="API Documentation" eyebrow="Build with EVE-KILL" icon="lucide:braces">
            <template #description>
                        The API powering eve-kill.com. JSON, CORS-open, and read endpoints need no
                        authentication. Click any endpoint to expand it, fill in the parameters, and hit
                        <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20 text-[10px] font-medium align-middle">
                            <Icon name="lucide:play" class="text-[10px]" />
                            Try it
                        </span>
                        — requests fire straight against the live API.
            </template>
            <template v-if="!loading && model" #actions>
                <div v-if="!loading && model" class="text-xs text-gray-500 text-right">
                    <div>{{ model.title }}</div>
                    <div class="text-gray-600">v{{ model.version }}</div>
                    <a
                        :href="`${baseUrl}/openapi.json`"
                        target="_blank"
                        rel="noopener"
                        class="font-mono text-blue-400/80 hover:text-blue-300 transition-colors"
                    >
                        openapi.json
                    </a>
                </div>
            </template>
        </PageHeader>

        <!-- Loading -->
        <div v-if="loading" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 text-center">
            <Icon name="lucide:loader-2" class="text-2xl text-blue-400 animate-spin" />
            <div class="text-sm text-gray-500 mt-3">Loading the API document…</div>
        </div>

        <!-- Error -->
        <div v-else-if="loadError" class="rounded-lg border border-red-500/[0.20] bg-red-500/[0.05] p-6">
            <div class="flex gap-3 items-start">
                <Icon name="lucide:alert-triangle" class="text-red-400 text-xl flex-shrink-0 mt-0.5" />
                <div class="flex-1">
                    <div class="text-sm text-red-300 font-medium">Failed to load the API document</div>
                    <div class="text-xs text-red-300/70 mt-1">{{ loadError }}</div>
                    <button
                        class="mt-3 text-xs px-3 py-1.5 rounded bg-red-500/20 hover:bg-red-500/30 text-red-200 border border-red-500/30 transition-colors"
                        @click="loadSpec"
                    >
                        Retry
                    </button>
                </div>
            </div>
        </div>

        <!-- Loaded -->
        <div v-else-if="model" class="grid grid-cols-1 lg:grid-cols-[260px_1fr] gap-6">
            <!-- Sidebar -->
            <aside class="lg:sticky lg:top-4 lg:self-start space-y-4 lg:max-h-[calc(100vh-2rem)] lg:overflow-y-auto">
                <div class="relative">
                    <Icon name="lucide:search" class="absolute left-2.5 top-1/2 -translate-y-1/2 text-xs text-gray-500 pointer-events-none" />
                    <input
                        v-model="search"
                        type="text"
                        placeholder="Filter endpoints…"
                        class="w-full bg-white/[0.04] border border-white/[0.10] rounded-md pl-7 pr-2.5 py-1.5 text-xs text-white placeholder-gray-600 outline-none focus:border-blue-500/40"
                    >
                </div>

                <nav class="space-y-1">
                    <div class="text-[10px] uppercase tracking-wider text-gray-600 px-2 mb-1">
                        {{ visibleCount }} of {{ model.total }} endpoints
                    </div>

                    <div v-for="group in groups" :key="group.name">
                        <button
                            class="w-full flex items-center gap-1.5 px-2 py-1.5 rounded text-xs font-medium text-gray-300 hover:text-white hover:bg-white/[0.04] transition-colors text-left"
                            @click="toggleGroup(group.name)"
                        >
                            <Icon
                                :name="isOpen(group.name) ? 'lucide:chevron-down' : 'lucide:chevron-right'"
                                class="text-[10px] flex-shrink-0 text-gray-500"
                            />
                            <span class="truncate flex-1">{{ group.name }}</span>
                            <span class="text-[10px] text-gray-600 font-mono flex-shrink-0">
                                {{ group.tags.reduce((n, t) => n + t.endpoints.length, 0) }}
                            </span>
                        </button>

                        <div v-if="isOpen(group.name)" class="ml-3 pl-2 border-l border-white/[0.06] space-y-0.5 mt-0.5 mb-1">
                            <button
                                v-for="tag in group.tags"
                                :key="tag.name"
                                class="w-full flex items-center justify-between gap-2 px-2 py-1 rounded text-xs transition-colors text-left"
                                :class="activeTag === tag.name
                                    ? 'text-blue-300 bg-blue-500/[0.08]'
                                    : 'text-gray-400 hover:text-blue-300 hover:bg-blue-500/[0.06]'"
                                @click="selectTag(tag)"
                            >
                                <span class="truncate">{{ tag.name }}</span>
                                <span class="text-[10px] text-gray-600 font-mono flex-shrink-0">{{ tag.endpoints.length }}</span>
                            </button>
                        </div>
                    </div>

                    <div v-if="groups.length === 0" class="text-xs text-gray-600 px-2 py-3 text-center">
                        No endpoints match
                    </div>
                </nav>
            </aside>

            <!-- Main -->
            <main class="space-y-10 min-w-0">
                <section v-for="group in groups" :key="group.name" class="space-y-8">
                    <h2 class="text-2xl font-bold text-white border-b border-white/[0.08] pb-2">
                        {{ group.name }}
                    </h2>

                    <section
                        v-for="tag in group.tags"
                        :id="`tag-${tag.name}`"
                        :key="tag.name"
                        class="scroll-mt-4"
                    >
                        <div class="mb-3">
                            <h3 class="text-lg font-semibold text-white capitalize">{{ tag.name }}</h3>
                            <p v-if="tag.description" class="text-xs text-gray-500 mt-1">{{ tag.description }}</p>
                        </div>
                        <div class="space-y-2">
                            <ApiDocsEndpoint
                                v-for="ep in tag.endpoints"
                                :key="`${ep.method} ${ep.path}`"
                                :endpoint="ep"
                                :base-url="baseUrl"
                            />
                        </div>
                    </section>
                </section>
            </main>
        </div>
    </div>
</template>
