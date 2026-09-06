<script setup lang="ts">
const { isDomainMode } = useDomainConfig()
const { isAuthenticated } = useAuth()

interface CampaignListEntity {
    entity_type: number
    entity_id: number
}

interface CampaignListSide {
    side_index: number
    name: string
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
    palette?: { main_color?: string; secondary_color?: string; tertiary_color?: string } | null
    entities: CampaignListEntity[]
}

interface CampaignListItem {
    campaign_id: string
    mode: 'conflict' | 'area'
    name: string
    description: string | null
    status: number
    visibility: number
    start_time: string
    end_time: string | null
    location: { systemIds?: number[]; constellationIds?: number[]; regionIds?: number[] } | null
    last_activity_at: string | null
    creator: { character_id: number; name: string | null }
    totals: { killCount: number; iskDestroyed: number } | null
    sides: CampaignListSide[]
}

interface CampaignListResponse {
    campaigns: CampaignListItem[]
    hasMore: boolean
    page: number
    total: number
    counts: { active: number; archived: number; private: number }
}

useHead({
    title: 'EVE Online Campaigns',
    link: [{ rel: 'canonical', href: 'https://eve-kill.com/campaigns' }],
})
useSeoMeta({
    description: 'Long-running conflict trackers on EVE-KILL — kills, losses and ISK destroyed for ongoing and historical wars in EVE Online.',
    ogTitle: 'EVE Online Campaigns — EVE-KILL',
    ogDescription: 'Track EVE Online wars and conflicts with campaign scoreboards on EVE-KILL.',
    ogType: 'website',
    ogUrl: 'https://eve-kill.com/campaigns',
    ogImage: 'https://eve-kill.com/og-default.png',
    twitterCard: 'summary',
    twitterTitle: 'EVE Online Campaigns — EVE-KILL',
    twitterDescription: 'Track active and historical EVE Online conflicts, sides, kills, losses, and ISK destroyed.',
    twitterImage: 'https://eve-kill.com/og-default.png',
})

useSchemaOrg([
    defineWebPage({
        '@type': 'CollectionPage',
        'name': 'EVE Online Campaigns',
        'description': 'Active and historical EVE Online campaign scoreboards with sides, kills, losses, and ISK destroyed.',
        'url': 'https://eve-kill.com/campaigns',
    }),
    defineBreadcrumb({
        itemListElement: [
            { name: 'Home', item: '/' },
            { name: 'Campaigns', item: '/campaigns' },
        ],
    }),
])

const route = useRoute()
const router = useRouter()

const PAGE_SIZE = 24

const STATUS_TABS = ['active', 'archived', 'private'] as const
type StatusTab = (typeof STATUS_TABS)[number]

const tab = computed<StatusTab>(() => {
    const status = String(route.query.status || '')
    return STATUS_TABS.includes(status as StatusTab) ? status as StatusTab : 'active'
})
const mode = computed(() => {
    const m = String(route.query.mode || '')
    return m === 'conflict' || m === 'area' ? m : 'all'
})
const search = computed(() => String(route.query.q || ''))
const page = computed(() => Math.max(1, Number(route.query.page) || 1))

// useApiFetch, not useApiFetch: the listing is auth- and host-sensitive (your own
// private campaigns, custom-domain scoping) and plain useApiFetch forwards neither
// cookie nor host on the SSR pass.
const { data, pending } = await useApiFetch<CampaignListResponse>('/api/campaigns', {
    query: computed(() => ({
        status: tab.value,
        page: page.value,
        ...(mode.value !== 'all' ? { mode: mode.value } : {}),
        ...(search.value ? { q: search.value } : {}),
    })),
    default: () => ({ campaigns: [], hasMore: false, page: 1, total: 0, counts: { active: 0, archived: 0, private: 0 } }),
})
const campaignData = computed<CampaignListResponse>(() => data.value ?? {
    campaigns: [],
    hasMore: false,
    page: 1,
    total: 0,
    counts: { active: 0, archived: 0, private: 0 },
})

/** Query patches always drop back to page 1 — the old offset means nothing. */
const patchQuery = (patch: Record<string, string | undefined>) => {
    const next: Record<string, string> = {}
    for (const [key, value] of Object.entries({ ...route.query, ...patch })) {
        if (value != null && value !== '') next[key] = String(value)
    }
    delete next.page
    router.push({ query: next })
}

const setTab = (t: string) => patchQuery({ status: t === 'active' ? undefined : t })
const setMode = (m: string) => patchQuery({ mode: m === 'all' ? undefined : m })
const setPage = (p: number) => router.push({ query: { ...route.query, page: p > 1 ? String(p) : undefined } })

// Local mirror so typing feels instant while the URL (and the fetch) lags
// behind by a debounce. Kept in sync when the query changes from elsewhere.
const searchInput = ref(search.value)
watch(search, (value) => { if (value !== searchInput.value) searchInput.value = value })

let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(searchInput, (value) => {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
        if (value.trim() !== search.value) patchQuery({ q: value.trim() || undefined })
    }, 300)
})
onUnmounted(() => { if (searchTimer) clearTimeout(searchTimer) })

const clearSearch = () => {
    searchInput.value = ''
    if (search.value) patchQuery({ q: undefined })
}

// Private — the viewer's own private campaigns, which nobody else can see — is
// rendered separately behind <ClientOnly>: auth only resolves after hydration,
// so folding it into this list would mismatch the server render.
const TABS = [
    { key: 'active', label: 'Active' },
    { key: 'archived', label: 'Archived' },
] as const

const TAB_CLASS = 'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors cursor-pointer'
const tabStateClass = (key: string) => tab.value === key
    ? 'bg-blue-500/20 text-blue-400'
    : 'bg-white/[0.02] text-gray-500 hover:bg-white/[0.05] hover:text-gray-300'

const MODES = [
    { key: 'all', label: 'All', icon: 'lucide:layers' },
    { key: 'conflict', label: 'Conflicts', icon: 'lucide:swords' },
    { key: 'area', label: 'Battlefields', icon: 'lucide:map-pinned' },
] as const

const tabCount = (key: string) =>
    key === 'archived' ? campaignData.value.counts.archived
    : key === 'private' ? campaignData.value.counts.private
    : campaignData.value.counts.active

const hasFilters = computed(() => mode.value !== 'all' || !!search.value)
const lastPage = computed(() => Math.max(1, Math.ceil((campaignData.value.total || 0) / PAGE_SIZE)))
</script>

<template>
    <div>
        <PageHeader class="mb-4" title="Campaigns" eyebrow="Tracked conflicts" icon="lucide:flag"
            >
            <template #description>
                Track EVE Online conflicts across multiple battles: kills, losses and ISK destroyed for opposing sides,
                one group's war effort, or a stretch of space. Scoreboards refresh hourly. For individual engagements,
                browse <NuxtLink to="/battles" class="text-blue-400 hover:underline">battle reports</NuxtLink>.
            </template>
            <template #actions>
                <div class="flex flex-wrap items-center justify-end gap-2">
                <span v-if="isDomainMode"
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-indigo-500/30 bg-indigo-500/10 text-fine font-medium text-indigo-300 flex-shrink-0"
                    v-tooltip="'Scoped to the entities this killboard tracks'">
                    <Icon name="lucide:shield" class="text-fine" />
                    This killboard only
                </span>
                <NuxtLink to="/campaigncreator"
                    class="inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium bg-blue-500 text-white hover:bg-blue-600 transition-colors flex-shrink-0">
                    <Icon name="lucide:plus" class="text-sm" />
                    New Campaign
                </NuxtLink>
                </div>
            </template>
        </PageHeader>

        <!-- Toolbar -->
        <div class="flex flex-wrap items-center gap-2 mb-4">
            <!-- Status -->
            <div class="flex rounded-lg border border-white/[0.08] overflow-hidden">
                <button v-for="t in TABS" :key="t.key" @click="setTab(t.key)"
                    :class="[TAB_CLASS, tabStateClass(t.key)]">
                    {{ t.label }}
                    <span class="tabular-nums" :class="tab === t.key ? 'text-blue-400/60' : 'text-gray-600'">{{ tabCount(t.key) }}</span>
                </button>
                <ClientOnly>
                    <button v-if="isAuthenticated" @click="setTab('private')"
                        :class="[TAB_CLASS, tabStateClass('private')]"
                        v-tooltip="'Campaigns you made that only you can see'">
                        <Icon name="lucide:lock" class="text-fine" />
                        Private
                        <span class="tabular-nums" :class="tab === 'private' ? 'text-blue-400/60' : 'text-gray-600'">{{ tabCount('private') }}</span>
                    </button>
                </ClientOnly>
            </div>

            <div class="w-px h-5 bg-white/[0.08] mx-1" />

            <!-- Mode -->
            <button v-for="m in MODES" :key="m.key" @click="setMode(m.key)"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border transition-colors cursor-pointer"
                :class="mode === m.key
                    ? 'bg-blue-500/15 text-blue-400 border-blue-500/30'
                    : 'text-gray-400 border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'">
                <Icon :name="m.icon" class="text-xs" />
                {{ m.label }}
            </button>

            <!-- Search -->
            <div class="relative ml-auto w-full sm:w-64">
                <div class="glass-panel flex items-center gap-2 h-9 px-3 focus-within:border-white/[0.2] transition-colors">
                    <Icon name="lucide:search" class="text-sm text-gray-500 flex-shrink-0" />
                    <input v-model="searchInput" type="text" placeholder="Search campaigns…"
                        class="flex-1 min-w-0 bg-transparent text-sm text-white placeholder-gray-500 outline-none">
                    <button v-if="searchInput" @click="clearSearch" class="text-gray-500 hover:text-gray-300 transition-colors flex-shrink-0">
                        <Icon name="lucide:x" class="text-sm" />
                    </button>
                </div>
            </div>
        </div>

        <!-- Loading -->
        <div v-if="pending && !campaignData.campaigns.length" class="flex items-center justify-center py-16">
            <Icon name="lucide:loader-2" class="text-2xl text-gray-500 animate-spin" />
        </div>

        <!-- Empty -->
        <div v-else-if="!campaignData.campaigns.length" class="glass-panel p-12 text-center">
            <Icon name="lucide:flag-off" class="text-3xl text-gray-600 mb-3" />
            <template v-if="hasFilters">
                <p class="text-sm text-gray-500 mb-1">No campaigns match these filters</p>
                <p class="text-fine text-gray-600 mb-4">Try a different search or clear the mode filter.</p>
                <button class="text-sm text-blue-400 hover:text-blue-300 cursor-pointer" @click="patchQuery({ q: undefined, mode: undefined })">
                    Clear filters
                </button>
            </template>
            <template v-else>
                <p class="text-sm text-gray-500 mb-4">No {{ tab }} campaigns yet</p>
                <NuxtLink to="/campaigncreator" class="text-sm text-blue-400 hover:text-blue-300">Create the first one →</NuxtLink>
            </template>
        </div>

        <!-- Cards -->
        <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-4" :class="{ 'opacity-60': pending }">
            <CampaignCard v-for="c in campaignData.campaigns" :key="c.campaign_id" :campaign="c" />
        </div>

        <!-- Pagination -->
        <div v-if="page > 1 || campaignData.hasMore" class="flex items-center justify-center gap-2 mt-6">
            <button :disabled="page <= 1" @click="setPage(page - 1)"
                class="px-3 py-1.5 rounded-lg text-xs font-medium bg-white/[0.04] text-gray-400 border border-white/[0.08] disabled:opacity-40 hover:bg-blue-500/[0.08] transition-colors cursor-pointer disabled:cursor-default">
                Previous
            </button>
            <span class="text-xs text-gray-500 tabular-nums px-2">Page {{ page }} of {{ lastPage }}</span>
            <button :disabled="!campaignData.hasMore" @click="setPage(page + 1)"
                class="px-3 py-1.5 rounded-lg text-xs font-medium bg-white/[0.04] text-gray-400 border border-white/[0.08] disabled:opacity-40 hover:bg-blue-500/[0.08] transition-colors cursor-pointer disabled:cursor-default">
                Next
            </button>
        </div>
    </div>
</template>
