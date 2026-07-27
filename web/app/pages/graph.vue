<script setup lang="ts">
useHead({ title: 'Graph Intelligence' })
useSeoMeta({
    description: 'Explore EVE Online character connections, coalition structures, and rivalries using graph analysis.',
    ogTitle: 'Graph Intelligence — EVE-KILL',
})

type Tab = 'path_finder' | 'entity_intel' | 'coalitions' | 'rivalries' | 'hunting_grounds' | 'hot_zones' | 'migration' | 'spy_check' | 'census'

const route = useRoute()
const router = useRouter()
const activeTab = ref<Tab>((route.query.tab as Tab) || 'path_finder')

watch(activeTab, (val) => {
    router.replace({ query: { tab: val } })
})

// ─── Path Finder ────────────────────────────────────────
const fromQuery = ref('')
const toQuery = ref('')
const fromId = ref<number | null>(null)
const toId = ref<number | null>(null)
const fromResults = ref<any[]>([])
const toResults = ref<any[]>([])
let fromTimer: ReturnType<typeof setTimeout> | null = null
let toTimer: ReturnType<typeof setTimeout> | null = null

function parseEntityId(hit: any): number {
    // id is "character_12345" — extract numeric part
    return Number(String(hit.id).split('_').pop())
}

function searchCharacters(query: string, results: Ref<any[]>, timerRef: { current: ReturnType<typeof setTimeout> | null }) {
    if (timerRef.current) clearTimeout(timerRef.current)
    if (query.length < 3) { results.value = []; return }
    timerRef.current = setTimeout(async () => {
        try {
            const res = await apiFetch<{ hits: any[] }>('/api/search', { params: { q: query, type: 'character' } })
            results.value = (res.hits || []).slice(0, 8)
        } catch { results.value = [] }
    }, 300)
}

const fromTimerRef = { current: null as ReturnType<typeof setTimeout> | null }
const toTimerRef = { current: null as ReturnType<typeof setTimeout> | null }
function onFromInput() { fromId.value = null; searchCharacters(fromQuery.value, fromResults, fromTimerRef) }
function onToInput() { toId.value = null; searchCharacters(toQuery.value, toResults, toTimerRef) }
function selectFrom(r: any) { fromId.value = parseEntityId(r); fromQuery.value = r.name; fromResults.value = [] }
function selectTo(r: any) { toId.value = parseEntityId(r); toQuery.value = r.name; toResults.value = [] }

const pathData = ref<any>(null)
const pathPending = ref(false)

watch([fromId, toId], async ([from, to]) => {
    if (!from || !to) { pathData.value = null; return }
    pathPending.value = true
    try {
        pathData.value = await apiFetch('/api/graph', {
            params: { mode: 'path_finder', fromId: from, toId: to },
        })
    } catch {
        pathData.value = null
    } finally {
        pathPending.value = false
    }
})

// ─── Coalitions ─────────────────────────────────────────
const coalitionData = ref<any>(null)
const coalitionPending = ref(false)
let coalitionLoaded = false

watch(activeTab, async (tab) => {
    if (tab === 'coalitions' && !coalitionLoaded) {
        coalitionLoaded = true
        coalitionPending.value = true
        try {
            coalitionData.value = await apiFetch('/api/graph', { params: { mode: 'coalitions' } })
        } catch { coalitionData.value = null }
        finally { coalitionPending.value = false }
    }
}, { immediate: true })

// ─── Rivalries ──────────────────────────────────────────
const rivalryType = ref<'alliance' | 'character'>('alliance')
const rivalryData = ref<any>(null)
const rivalryPending = ref(false)
let lastRivalryType = ''

watch([activeTab, rivalryType], async ([tab, type]) => {
    if (tab !== 'rivalries') return
    if (type === lastRivalryType) return
    lastRivalryType = type
    rivalryPending.value = true
    try {
        rivalryData.value = await apiFetch('/api/graph', {
            params: { mode: 'rivalries', entityType: type, limit: 50 },
        })
    } catch { rivalryData.value = null }
    finally { rivalryPending.value = false }
}, { immediate: true })

// ─── Entity Intel ───────────────────────────────────────
const intelQuery = ref('')
const intelResults = ref<any[]>([])
const intelEntityType = ref<'alliance' | 'corporation'>('alliance')
const intelEntityId = ref<number | null>(null)
const intelData = ref<any>(null)
const intelPending = ref(false)
let intelTimer: ReturnType<typeof setTimeout> | null = null

function onIntelInput() {
    intelEntityId.value = null
    intelData.value = null
    if (intelTimer) clearTimeout(intelTimer)
    if (intelQuery.value.length < 3) { intelResults.value = []; return }
    intelTimer = setTimeout(async () => {
        try {
            const res = await apiFetch<{ hits: any[] }>('/api/search', {
                params: { q: intelQuery.value, type: intelEntityType.value },
            })
            intelResults.value = (res.hits || []).slice(0, 8)
        } catch { intelResults.value = [] }
    }, 300)
}

function selectIntelEntity(r: any) {
    intelEntityId.value = parseEntityId(r)
    intelQuery.value = r.name
    intelResults.value = []
}

watch([intelEntityId, intelEntityType], async ([id, type]) => {
    if (!id) { intelData.value = null; return }
    intelPending.value = true
    try {
        intelData.value = await apiFetch('/api/graph', {
            params: { mode: 'entity_intel', entityId: id, entityType: type },
        })
    } catch { intelData.value = null }
    finally { intelPending.value = false }
})

// ─── Hunting Grounds ────────────────────────────────────
const huntQuery = ref('')
const huntResults = ref<any[]>([])
const huntEntityType = ref<'alliance' | 'corporation'>('alliance')
const huntEntityId = ref<number | null>(null)
const huntData = ref<any>(null)
const huntPending = ref(false)
let huntTimer: ReturnType<typeof setTimeout> | null = null

function onHuntInput() {
    huntEntityId.value = null; huntData.value = null
    if (huntTimer) clearTimeout(huntTimer)
    if (huntQuery.value.length < 3) { huntResults.value = []; return }
    huntTimer = setTimeout(async () => {
        try {
            const res = await apiFetch<{ hits: any[] }>('/api/search', { params: { q: huntQuery.value, type: huntEntityType.value } })
            huntResults.value = (res.hits || []).slice(0, 8)
        } catch { huntResults.value = [] }
    }, 300)
}

function selectHuntEntity(r: any) { huntEntityId.value = parseEntityId(r); huntQuery.value = r.name; huntResults.value = [] }

watch([huntEntityId, huntEntityType], async ([id, type]) => {
    if (!id) { huntData.value = null; return }
    huntPending.value = true
    try { huntData.value = await apiFetch('/api/graph', { params: { mode: 'hunting_grounds', entityId: id, entityType: type } }) }
    catch { huntData.value = null }
    finally { huntPending.value = false }
})

// ─── Hot Zones ──────────────────────────────────────────
const hotZoneData = ref<any>(null)
const hotZonePending = ref(false)
let hotZoneLoaded = false

watch(activeTab, async (tab) => {
    if (tab === 'hot_zones' && !hotZoneLoaded) {
        hotZoneLoaded = true; hotZonePending.value = true
        try { hotZoneData.value = await apiFetch('/api/graph', { params: { mode: 'hot_zones' } }) }
        catch { hotZoneData.value = null }
        finally { hotZonePending.value = false }
    }
}, { immediate: true })

// ─── Migration ──────────────────────────────────────────
const migQuery = ref('')
const migResults = ref<any[]>([])
const migEntityId = ref<number | null>(null)
const migData = ref<any>(null)
const migPending = ref(false)
let migTimer: ReturnType<typeof setTimeout> | null = null

function onMigInput() {
    migEntityId.value = null; migData.value = null
    if (migTimer) clearTimeout(migTimer)
    if (migQuery.value.length < 3) { migResults.value = []; return }
    migTimer = setTimeout(async () => {
        try {
            const res = await apiFetch<{ hits: any[] }>('/api/search', { params: { q: migQuery.value, type: 'corporation' } })
            migResults.value = (res.hits || []).slice(0, 8)
        } catch { migResults.value = [] }
    }, 300)
}

function selectMigEntity(r: any) { migEntityId.value = parseEntityId(r); migQuery.value = r.name; migResults.value = [] }

watch(migEntityId, async (id) => {
    if (!id) { migData.value = null; return }
    migPending.value = true
    try { migData.value = await apiFetch('/api/graph', { params: { mode: 'migration', entityId: id } }) }
    catch { migData.value = null }
    finally { migPending.value = false }
})

// ─── Spy Check ──────────────────────────────────────────
const spyQuery = ref('')
const spyResults = ref<any[]>([])
const spyEntityType = ref<'alliance' | 'corporation'>('alliance')
const spyEntityId = ref<number | null>(null)
const spyData = ref<any>(null)
const spyPending = ref(false)
let spyTimer: ReturnType<typeof setTimeout> | null = null

function onSpyInput() {
    spyEntityId.value = null; spyData.value = null
    if (spyTimer) clearTimeout(spyTimer)
    if (spyQuery.value.length < 3) { spyResults.value = []; return }
    spyTimer = setTimeout(async () => {
        try {
            const res = await apiFetch<{ hits: any[] }>('/api/search', { params: { q: spyQuery.value, type: spyEntityType.value } })
            spyResults.value = (res.hits || []).slice(0, 8)
        } catch { spyResults.value = [] }
    }, 300)
}

function selectSpyEntity(r: any) { spyEntityId.value = parseEntityId(r); spyQuery.value = r.name; spyResults.value = [] }

watch([spyEntityId, spyEntityType], async ([id, type]) => {
    if (!id) { spyData.value = null; return }
    spyPending.value = true
    try { spyData.value = await apiFetch('/api/graph', { params: { mode: 'spy_check', entityId: id, entityType: type } }) }
    catch { spyData.value = null }
    finally { spyPending.value = false }
})

// ─── Census ─────────────────────────────────────────────
const censusQuery = ref('')
const censusResults = ref<any[]>([])
const censusEntityId = ref<number | null>(null)
const censusData = ref<any>(null)
const censusPending = ref(false)
let censusTimer: ReturnType<typeof setTimeout> | null = null

function onCensusInput() {
    censusEntityId.value = null; censusData.value = null
    if (censusTimer) clearTimeout(censusTimer)
    if (censusQuery.value.length < 3) { censusResults.value = []; return }
    censusTimer = setTimeout(async () => {
        try {
            const res = await apiFetch<{ hits: any[] }>('/api/search', { params: { q: censusQuery.value, type: 'alliance' } })
            censusResults.value = (res.hits || []).slice(0, 8)
        } catch { censusResults.value = [] }
    }, 300)
}

function selectCensusEntity(r: any) { censusEntityId.value = parseEntityId(r); censusQuery.value = r.name; censusResults.value = [] }

watch(censusEntityId, async (id) => {
    if (!id) { censusData.value = null; return }
    censusPending.value = true
    try { censusData.value = await apiFetch('/api/graph', { params: { mode: 'census', entityId: id } }) }
    catch { censusData.value = null }
    finally { censusPending.value = false }
})

// ─── Coalition expand/collapse ──────────────────────────
const expandedCoalitions = ref(new Set<number>())
function toggleCoalition(id: number) {
    const s = new Set(expandedCoalitions.value)
    if (s.has(id)) s.delete(id); else s.add(id)
    expandedCoalitions.value = s
}

// ─── Helpers ────────────────────────────────────────────
const tabs = [
    { id: 'path_finder' as Tab, label: 'Path Finder', subtitle: 'Six degrees of EVE' },
    { id: 'entity_intel' as Tab, label: 'Allies & Enemies', subtitle: 'Lookup any group' },
    { id: 'hunting_grounds' as Tab, label: 'Hunting Grounds', subtitle: 'Where to find them' },
    { id: 'hot_zones' as Tab, label: 'Hot Zones', subtitle: 'Active conflict areas' },
    { id: 'census' as Tab, label: 'Military Census', subtitle: 'FC / Logi / Cap count' },
    { id: 'spy_check' as Tab, label: 'Spy Check', subtitle: 'Cross-alliance activity' },
    { id: 'migration' as Tab, label: 'Migration', subtitle: 'Corp member movement' },
    { id: 'coalitions' as Tab, label: 'Coalitions', subtitle: 'Auto-detected blocs' },
    { id: 'rivalries' as Tab, label: 'Rivalries', subtitle: 'Who fights who' },
]
</script>

<template>
    <div>
        <!-- Header -->
        <div class="glass-panel overflow-hidden mb-4">
            <div class="p-4">
                <h1 class="text-xl font-bold text-white mb-1">Graph Intelligence <span class="text-xs font-medium px-1.5 py-0.5 rounded bg-yellow-500/20 text-yellow-400 border border-yellow-500/30 align-middle ml-1">BETA</span></h1>
                <p class="text-sm text-gray-500 mb-4">Relationship analysis powered by graph data from killmail activity — experimental features, subject to change</p>

                <!-- Tabs -->
                <div class="flex gap-2">
                    <button
                        v-for="tab in tabs"
                        :key="tab.id"
                        @click="activeTab = tab.id"
                        class="flex flex-col px-4 py-2 rounded-md text-sm transition-colors"
                        :class="activeTab === tab.id
                            ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                            : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white hover:bg-white/[0.08]'"
                    >
                        <span class="font-medium">{{ tab.label }}</span>
                        <span class="text-xs opacity-60">{{ tab.subtitle }}</span>
                    </button>
                </div>
            </div>
        </div>

        <!-- ═══ Path Finder ═══ -->
        <div v-if="activeTab === 'path_finder'">
            <!-- Search inputs -->
            <div class="glass-panel p-4 mb-4">
                <div class="flex flex-col sm:flex-row gap-3 items-start sm:items-center">
                    <!-- From -->
                    <div class="relative flex-1 w-full">
                        <label class="text-xs text-gray-500 mb-1 block">From</label>
                        <input
                            v-model="fromQuery"
                            @input="onFromInput"
                            placeholder="Search character..."
                            class="w-full bg-white/[0.06] border border-white/[0.1] rounded-md px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-blue-500/50"
                        />
                        <div v-if="fromResults.length > 0" class="absolute z-20 w-full mt-1 bg-gray-900 border border-white/[0.1] rounded-md shadow-xl max-h-48 overflow-y-auto">
                            <button v-for="r in fromResults" :key="r.id" @click="selectFrom(r)" class="w-full text-left px-3 py-2 text-sm hover:bg-white/[0.06] flex items-baseline gap-2">
                                <span class="text-white">{{ r.name }}</span>
                                <span v-if="r.corporation_name" class="text-xs text-gray-500">{{ r.corporation_name }}</span>
                                <span v-if="r.alliance_name" class="text-xs text-gray-600">[{{ r.alliance_ticker || r.alliance_name }}]</span>
                            </button>
                        </div>
                    </div>

                    <!-- Arrow -->
                    <div class="hidden sm:flex items-center pt-5 text-gray-600">
                        <Icon name="lucide:arrow-right" class="text-lg" />
                    </div>

                    <!-- To -->
                    <div class="relative flex-1 w-full">
                        <label class="text-xs text-gray-500 mb-1 block">To</label>
                        <input
                            v-model="toQuery"
                            @input="onToInput"
                            placeholder="Search character..."
                            class="w-full bg-white/[0.06] border border-white/[0.1] rounded-md px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-blue-500/50"
                        />
                        <div v-if="toResults.length > 0" class="absolute z-20 w-full mt-1 bg-gray-900 border border-white/[0.1] rounded-md shadow-xl max-h-48 overflow-y-auto">
                            <button v-for="r in toResults" :key="r.id" @click="selectTo(r)" class="w-full text-left px-3 py-2 text-sm hover:bg-white/[0.06] flex items-baseline gap-2">
                                <span class="text-white">{{ r.name }}</span>
                                <span v-if="r.corporation_name" class="text-xs text-gray-500">{{ r.corporation_name }}</span>
                                <span v-if="r.alliance_name" class="text-xs text-gray-600">[{{ r.alliance_ticker || r.alliance_name }}]</span>
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Path result -->
            <div v-if="pathPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center">
                <div class="text-gray-400 text-sm animate-pulse">Searching for connection...</div>
            </div>

            <div v-else-if="pathData?.path" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-4">
                <div class="flex items-center gap-2 mb-4">
                    <span class="text-sm font-medium text-green-400">Connected</span>
                    <span class="text-xs text-gray-500">{{ pathData.path.hops }} {{ pathData.path.hops === 1 ? 'hop' : 'hops' }}</span>
                </div>

                <!-- Chain -->
                <div class="flex flex-wrap items-center gap-1">
                    <template v-for="(node, i) in pathData.path.nodes" :key="node.id">
                        <!-- Node card -->
                        <div class="flex flex-col items-center px-3 py-2 rounded-lg bg-white/[0.04] border border-white/[0.06] min-w-[120px]">
                            <NuxtLink :to="`/character/${node.id}`" class="text-sm font-medium text-blue-400 hover:text-blue-300 text-center">
                                {{ node.name }}
                            </NuxtLink>
                            <NuxtLink v-if="node.corporation" :to="`/corporation/${node.corporation.id}`" class="text-xs text-gray-400 hover:text-gray-300 mt-0.5">
                                {{ node.corporation.name }}
                            </NuxtLink>
                            <NuxtLink v-if="node.alliance" :to="`/alliance/${node.alliance.id}`" class="text-xs text-gray-600 hover:text-gray-500">
                                [{{ node.alliance.name }}]
                            </NuxtLink>
                        </div>

                        <!-- Edge arrow -->
                        <div v-if="i < pathData.path.edges.length" class="flex flex-col items-center px-2 text-gray-600 shrink-0">
                            <div class="text-xs tabular-nums text-gray-500">&times;{{ formatCompact(pathData.path.edges[i].weight) }}</div>
                            <Icon name="lucide:arrow-right" class="text-base" />
                        </div>
                    </template>
                </div>
            </div>

            <div v-else-if="fromId && toId && !pathPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center">
                <div class="text-gray-500 text-sm">No connection found within 6 hops</div>
            </div>

            <div v-else-if="!fromId || !toId" class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]">
                <div class="text-gray-600 text-sm text-center">
                    Search for two characters to find the shortest FLEW_WITH chain between them
                </div>
            </div>
        </div>

        <!-- ═══ Hunting Grounds ═══ -->
        <div v-if="activeTab === 'hunting_grounds'">
            <div class="glass-panel p-4 mb-4">
                <div class="flex flex-col sm:flex-row gap-3 items-start sm:items-end">
                    <div class="flex gap-2">
                        <button @click="huntEntityType = 'alliance'; huntQuery = ''; huntEntityId = null" class="px-3 py-1.5 rounded-md text-sm transition-colors" :class="huntEntityType === 'alliance' ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white'">Alliance</button>
                        <button @click="huntEntityType = 'corporation'; huntQuery = ''; huntEntityId = null" class="px-3 py-1.5 rounded-md text-sm transition-colors" :class="huntEntityType === 'corporation' ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white'">Corporation</button>
                    </div>
                    <div class="relative flex-1 w-full">
                        <input v-model="huntQuery" @input="onHuntInput" :placeholder="`Search ${huntEntityType}...`" class="w-full bg-white/[0.06] border border-white/[0.1] rounded-md px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50" />
                        <div v-if="huntResults.length > 0" class="absolute z-20 w-full mt-1 bg-gray-900 border border-white/[0.1] rounded-md shadow-xl max-h-48 overflow-y-auto">
                            <button v-for="r in huntResults" :key="r.id" @click="selectHuntEntity(r)" class="w-full text-left px-3 py-2 text-sm hover:bg-white/[0.06] text-white">{{ r.name }} <span v-if="r.ticker" class="text-xs text-gray-500">[{{ r.ticker }}]</span></button>
                        </div>
                    </div>
                </div>
            </div>
            <div v-if="huntPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center"><div class="text-gray-400 text-sm animate-pulse">Scanning systems...</div></div>
            <div v-else-if="huntData?.systems?.length" class="rounded-lg overflow-hidden border border-white/[0.08] bg-white/[0.02]">
                <div class="px-4 py-2 bg-white/[0.04] border-b border-white/[0.06] text-xs text-gray-500">Activity in the last 24 hours</div>
                <table class="w-full text-sm">
                    <thead><tr class="text-left text-gray-500 border-b border-white/[0.06]"><th class="px-4 py-2 w-10">#</th><th class="px-4 py-2">System</th><th class="px-4 py-2 text-right">Active Characters</th></tr></thead>
                    <tbody>
                        <tr v-for="(sys, i) in huntData.systems" :key="sys.id" class="border-b border-white/[0.04] hover:bg-white/[0.04]">
                            <td class="px-4 py-2 text-gray-600 tabular-nums">{{ Number(i) + 1 }}</td>
                            <td class="px-4 py-2"><NuxtLink :to="`/system/${sys.id}`" class="text-blue-400 hover:text-blue-300">{{ sys.name }}</NuxtLink></td>
                            <td class="px-4 py-2 text-right tabular-nums text-white">{{ sys.active_characters }}</td>
                        </tr>
                    </tbody>
                </table>
            </div>
            <div v-else-if="huntEntityId && !huntPending" class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]"><div class="text-gray-600 text-sm">No recent activity found</div></div>
            <div v-else class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]"><div class="text-gray-600 text-sm">Search for an alliance or corporation to see where they're active</div></div>
        </div>

        <!-- ═══ Hot Zones ═══ -->
        <div v-if="activeTab === 'hot_zones'">
            <div v-if="hotZonePending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center"><div class="text-gray-400 text-sm animate-pulse">Scanning for hot zones...</div></div>
            <div v-else-if="hotZoneData?.systems?.length" class="rounded-lg overflow-hidden border border-white/[0.08] bg-white/[0.02]">
                <div class="px-4 py-2 bg-red-500/10 border-b border-white/[0.06] text-xs text-gray-500">Systems with cross-alliance conflict in the last 24 hours</div>
                <table class="w-full text-sm">
                    <thead><tr class="text-left text-gray-500 border-b border-white/[0.06]"><th class="px-4 py-2 w-10">#</th><th class="px-4 py-2">System</th><th class="px-4 py-2 text-right">Alliances</th><th class="px-4 py-2 text-right">Characters</th></tr></thead>
                    <tbody>
                        <tr v-for="(sys, i) in hotZoneData.systems" :key="sys.id" class="border-b border-white/[0.04] hover:bg-white/[0.04]">
                            <td class="px-4 py-2 text-gray-600 tabular-nums">{{ Number(i) + 1 }}</td>
                            <td class="px-4 py-2"><NuxtLink :to="`/system/${sys.id}`" class="text-blue-400 hover:text-blue-300">{{ sys.name }}</NuxtLink></td>
                            <td class="px-4 py-2 text-right tabular-nums text-yellow-400">{{ sys.alliances }}</td>
                            <td class="px-4 py-2 text-right tabular-nums text-white">{{ sys.characters }}</td>
                        </tr>
                    </tbody>
                </table>
            </div>
            <div v-else class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]"><div class="text-gray-600 text-sm">No hot zones detected in the last 24 hours</div></div>
        </div>

        <!-- ═══ Census ═══ -->
        <div v-if="activeTab === 'census'">
            <div class="glass-panel p-4 mb-4">
                <div class="relative">
                    <input v-model="censusQuery" @input="onCensusInput" placeholder="Search for an alliance..." class="w-full bg-white/[0.06] border border-white/[0.1] rounded-md px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50" />
                    <div v-if="censusResults.length > 0" class="absolute z-20 w-full mt-1 bg-gray-900 border border-white/[0.1] rounded-md shadow-xl max-h-48 overflow-y-auto">
                        <button v-for="r in censusResults" :key="r.id" @click="selectCensusEntity(r)" class="w-full text-left px-3 py-2 text-sm hover:bg-white/[0.06] text-white">{{ r.name }} <span v-if="r.ticker" class="text-xs text-gray-500">[{{ r.ticker }}]</span></button>
                    </div>
                </div>
            </div>
            <div v-if="censusPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center"><div class="text-gray-400 text-sm animate-pulse">Running census...</div></div>
            <div v-else-if="censusData?.corps?.length">
                <!-- Totals -->
                <div class="grid grid-cols-4 gap-3 mb-4">
                    <div class="glass-panel p-3 text-center">
                        <div class="text-lg font-bold text-white tabular-nums">{{ formatCompact(censusData.totals.characters) }}</div>
                        <div class="text-xs text-gray-500">Characters</div>
                    </div>
                    <div class="rounded-lg bg-blue-500/10 border border-blue-500/20 p-3 text-center">
                        <div class="text-lg font-bold text-blue-400 tabular-nums">{{ censusData.totals.fcs }}</div>
                        <div class="text-xs text-gray-500">FCs</div>
                    </div>
                    <div class="rounded-lg bg-green-500/10 border border-green-500/20 p-3 text-center">
                        <div class="text-lg font-bold text-green-400 tabular-nums">{{ formatCompact(censusData.totals.logis) }}</div>
                        <div class="text-xs text-gray-500">Logi Pilots</div>
                    </div>
                    <div class="rounded-lg bg-orange-500/10 border border-orange-500/20 p-3 text-center">
                        <div class="text-lg font-bold text-orange-400 tabular-nums">{{ formatCompact(censusData.totals.caps) }}</div>
                        <div class="text-xs text-gray-500">Cap Pilots</div>
                    </div>
                </div>
                <!-- Corps table -->
                <div class="rounded-lg overflow-hidden border border-white/[0.08] bg-white/[0.02]">
                    <table class="w-full text-sm">
                        <thead><tr class="text-left text-gray-500 border-b border-white/[0.06]"><th class="px-4 py-2">Corporation</th><th class="px-4 py-2 text-right">Chars</th><th class="px-4 py-2 text-right">FCs</th><th class="px-4 py-2 text-right">Logi</th><th class="px-4 py-2 text-right">Caps</th></tr></thead>
                        <tbody>
                            <tr v-for="corp in censusData.corps" :key="corp.id" class="border-b border-white/[0.04] hover:bg-white/[0.04]">
                                <td class="px-4 py-1.5"><NuxtLink :to="`/corporation/${corp.id}`" class="text-blue-400 hover:text-blue-300 text-sm">{{ corp.name }}</NuxtLink></td>
                                <td class="px-4 py-1.5 text-right tabular-nums text-white">{{ corp.total }}</td>
                                <td class="px-4 py-1.5 text-right tabular-nums text-blue-400">{{ corp.fcs || '-' }}</td>
                                <td class="px-4 py-1.5 text-right tabular-nums text-green-400">{{ corp.logis || '-' }}</td>
                                <td class="px-4 py-1.5 text-right tabular-nums text-orange-400">{{ corp.caps || '-' }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
            <div v-else-if="!censusEntityId" class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]"><div class="text-gray-600 text-sm">Search for an alliance to see its military composition</div></div>
        </div>

        <!-- ═══ Spy Check ═══ -->
        <div v-if="activeTab === 'spy_check'">
            <div class="glass-panel p-4 mb-4">
                <div class="flex flex-col sm:flex-row gap-3 items-start sm:items-end">
                    <div class="flex gap-2">
                        <button @click="spyEntityType = 'alliance'; spyQuery = ''; spyEntityId = null" class="px-3 py-1.5 rounded-md text-sm transition-colors" :class="spyEntityType === 'alliance' ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white'">Alliance</button>
                        <button @click="spyEntityType = 'corporation'; spyQuery = ''; spyEntityId = null" class="px-3 py-1.5 rounded-md text-sm transition-colors" :class="spyEntityType === 'corporation' ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white'">Corporation</button>
                    </div>
                    <div class="relative flex-1 w-full">
                        <input v-model="spyQuery" @input="onSpyInput" :placeholder="`Search ${spyEntityType}...`" class="w-full bg-white/[0.06] border border-white/[0.1] rounded-md px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50" />
                        <div v-if="spyResults.length > 0" class="absolute z-20 w-full mt-1 bg-gray-900 border border-white/[0.1] rounded-md shadow-xl max-h-48 overflow-y-auto">
                            <button v-for="r in spyResults" :key="r.id" @click="selectSpyEntity(r)" class="w-full text-left px-3 py-2 text-sm hover:bg-white/[0.06] text-white">{{ r.name }} <span v-if="r.ticker" class="text-xs text-gray-500">[{{ r.ticker }}]</span></button>
                        </div>
                    </div>
                </div>
            </div>
            <div v-if="spyPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center"><div class="text-gray-400 text-sm animate-pulse">Scanning for cross-alliance activity...</div></div>
            <div v-else-if="spyData?.suspects?.length" class="rounded-lg overflow-hidden border border-white/[0.08] bg-white/[0.02]">
                <div class="px-4 py-2 bg-yellow-500/10 border-b border-white/[0.06] text-xs text-gray-500">Characters with significant FLEW_WITH connections to outside groups</div>
                <div class="divide-y divide-white/[0.04]">
                    <div v-for="suspect in spyData.suspects" :key="suspect.id" class="px-4 py-2 hover:bg-white/[0.04]">
                        <div class="flex items-center gap-3">
                            <NuxtLink :to="`/character/${suspect.id}`" class="text-sm text-blue-400 hover:text-blue-300 font-medium">{{ suspect.name }}</NuxtLink>
                            <span v-if="suspect.corp_name" class="text-xs text-gray-500">{{ suspect.corp_name }}</span>
                            <span class="ml-auto text-xs text-yellow-400 tabular-nums">{{ formatCompact(suspect.total_flights) }} flights</span>
                        </div>
                        <div class="flex gap-2 mt-1 flex-wrap">
                            <span v-for="conn in suspect.connections" :key="conn.id" class="text-xs px-2 py-0.5 rounded bg-white/[0.06] text-gray-400">
                                {{ conn.name }} <span class="text-gray-600">&times;{{ formatCompact(conn.flights) }}</span>
                            </span>
                        </div>
                    </div>
                </div>
            </div>
            <div v-else-if="spyEntityId && !spyPending" class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]"><div class="text-gray-600 text-sm">No suspicious cross-alliance activity detected</div></div>
            <div v-else class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]"><div class="text-gray-600 text-sm">Search for an alliance or corporation to check for cross-group activity</div></div>
        </div>

        <!-- ═══ Migration ═══ -->
        <div v-if="activeTab === 'migration'">
            <div class="glass-panel p-4 mb-4">
                <div class="relative">
                    <input v-model="migQuery" @input="onMigInput" placeholder="Search for a corporation..." class="w-full bg-white/[0.06] border border-white/[0.1] rounded-md px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50" />
                    <div v-if="migResults.length > 0" class="absolute z-20 w-full mt-1 bg-gray-900 border border-white/[0.1] rounded-md shadow-xl max-h-48 overflow-y-auto">
                        <button v-for="r in migResults" :key="r.id" @click="selectMigEntity(r)" class="w-full text-left px-3 py-2 text-sm hover:bg-white/[0.06] text-white">{{ r.name }} <span v-if="r.ticker" class="text-xs text-gray-500">[{{ r.ticker }}]</span></button>
                    </div>
                </div>
            </div>
            <div v-if="migPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center"><div class="text-gray-400 text-sm animate-pulse">Tracking migrations...</div></div>
            <div v-else-if="migData" class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <!-- Departed -->
                <div class="rounded-lg border border-white/[0.08] bg-white/[0.02] overflow-hidden">
                    <div class="px-4 py-2 bg-red-500/10 border-b border-white/[0.06]">
                        <span class="text-sm font-medium text-red-400">Departed</span>
                        <span class="text-xs text-gray-500 ml-2">formerly in {{ migData.corp_name || 'this corp' }}</span>
                    </div>
                    <div v-if="migData.departed?.length" class="divide-y divide-white/[0.04]">
                        <div v-for="char in migData.departed" :key="char.id" class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04]">
                            <NuxtLink :to="`/character/${char.id}`" class="text-sm text-blue-400 hover:text-blue-300 flex-1 truncate">{{ char.name }}</NuxtLink>
                            <span v-if="char.current_corp?.name" class="text-xs text-gray-500 truncate max-w-[150px]">now in {{ char.current_corp.name }}</span>
                        </div>
                    </div>
                    <div v-else class="p-4 text-sm text-gray-600 text-center">No departures detected</div>
                </div>
                <!-- Joined -->
                <div class="rounded-lg border border-white/[0.08] bg-white/[0.02] overflow-hidden">
                    <div class="px-4 py-2 bg-green-500/10 border-b border-white/[0.06]">
                        <span class="text-sm font-medium text-green-400">Joined</span>
                        <span class="text-xs text-gray-500 ml-2">currently active, came from elsewhere</span>
                    </div>
                    <div v-if="migData.joined?.length" class="divide-y divide-white/[0.04]">
                        <div v-for="char in migData.joined" :key="char.id" class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04]">
                            <NuxtLink :to="`/character/${char.id}`" class="text-sm text-blue-400 hover:text-blue-300 flex-1 truncate">{{ char.name }}</NuxtLink>
                            <span v-if="char.previous_corp?.name" class="text-xs text-gray-500 truncate max-w-[150px]">from {{ char.previous_corp.name }}</span>
                        </div>
                    </div>
                    <div v-else class="p-4 text-sm text-gray-600 text-center">No recent joins from other corps</div>
                </div>
            </div>
            <div v-else-if="!migEntityId" class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]"><div class="text-gray-600 text-sm">Search for a corporation to see member movements</div></div>
        </div>

        <!-- ═══ Coalitions ═══ -->
        <div v-if="activeTab === 'coalitions'">
            <div v-if="coalitionPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center">
                <div class="text-gray-400 text-sm animate-pulse">Detecting coalitions...</div>
            </div>

            <div v-else-if="coalitionData?.coalitions?.length" class="space-y-3">
                <div class="text-xs text-gray-500 mb-2">
                    Detected {{ coalitionData.coalitions.length }} coalition{{ coalitionData.coalitions.length > 1 ? 's' : '' }} based on shared enemy analysis
                </div>

                <div
                    v-for="coalition in coalitionData.coalitions"
                    :key="coalition.id"
                    class="rounded-lg border border-white/[0.08] bg-white/[0.02] overflow-hidden"
                >
                    <button
                        @click="toggleCoalition(coalition.id)"
                        class="w-full px-4 py-3 bg-white/[0.04] flex items-center gap-3 hover:bg-white/[0.06] transition-colors text-left"
                    >
                        <Icon
                            :name="expandedCoalitions.has(coalition.id) ? 'lucide:chevron-down' : 'lucide:chevron-right'"
                            class="text-gray-500 text-sm shrink-0"
                        />
                        <span class="text-sm font-medium text-white">Coalition {{ coalition.id }}</span>
                        <span class="text-xs text-gray-500">{{ coalition.alliances.length }} alliances</span>
                        <span class="flex-1 text-xs text-gray-600 truncate ml-2">
                            {{ coalition.alliances.slice(0, 6).map((a: any) => a.name).join(', ') }}{{ coalition.alliances.length > 6 ? ', ...' : '' }}
                        </span>
                    </button>
                    <div v-if="expandedCoalitions.has(coalition.id)" class="divide-y divide-white/[0.04]">
                        <div
                            v-for="alliance in coalition.alliances"
                            :key="alliance.id"
                            class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04] transition-colors"
                        >
                            <NuxtLink :to="`/alliance/${alliance.id}`" class="text-sm text-blue-400 hover:text-blue-300 flex-1 truncate">
                                {{ alliance.name }}
                            </NuxtLink>
                            <span class="text-xs text-gray-500 tabular-nums shrink-0">
                                {{ formatCompact(alliance.cooperation) }}
                            </span>
                        </div>
                    </div>
                </div>
            </div>

            <div v-else class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]">
                <div class="text-gray-600 text-sm">No coalitions detected</div>
            </div>
        </div>

        <!-- ═══ Rivalries ═══ -->
        <div v-if="activeTab === 'rivalries'">
            <!-- Sub-tabs -->
            <div class="flex gap-2 mb-4">
                <button
                    @click="rivalryType = 'alliance'"
                    class="px-3 py-1.5 rounded-md text-sm transition-colors"
                    :class="rivalryType === 'alliance'
                        ? 'bg-red-500/20 text-red-400 border border-red-500/30'
                        : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white'"
                >
                    Alliances
                </button>
                <button
                    @click="rivalryType = 'character'"
                    class="px-3 py-1.5 rounded-md text-sm transition-colors"
                    :class="rivalryType === 'character'
                        ? 'bg-red-500/20 text-red-400 border border-red-500/30'
                        : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white'"
                >
                    Characters
                </button>
            </div>

            <div v-if="rivalryPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center">
                <div class="text-gray-400 text-sm animate-pulse">Loading rivalries...</div>
            </div>

            <div v-else-if="rivalryData?.items?.length" class="rounded-lg overflow-hidden border border-white/[0.08] bg-white/[0.02]">
                <table class="w-full text-sm">
                    <thead>
                        <tr class="text-left text-gray-500 border-b border-white/[0.06]">
                            <th class="px-4 py-2 w-10">#</th>
                            <th class="px-4 py-2">Side A</th>
                            <th class="px-4 py-2 text-center w-8"></th>
                            <th class="px-4 py-2">Side B</th>
                            <th class="px-4 py-2 text-right">Mutual Kills</th>
                            <th class="px-4 py-2 text-right">ISK Destroyed</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr
                            v-for="(item, i) in rivalryData.items"
                            :key="i"
                            class="border-b border-white/[0.04] hover:bg-white/[0.04] transition-colors"
                        >
                            <td class="px-4 py-2 text-gray-600 tabular-nums">{{ Number(i) + 1 }}</td>
                            <td class="px-4 py-2">
                                <NuxtLink
                                    :to="rivalryType === 'character' ? `/character/${item.entity_a.id}` : `/alliance/${item.entity_a.id}`"
                                    class="text-blue-400 hover:text-blue-300"
                                >
                                    {{ item.entity_a.name }}
                                </NuxtLink>
                                <div v-if="item.entity_a.corp_name" class="text-xs text-gray-500">{{ item.entity_a.corp_name }}</div>
                                <div v-if="item.entity_a.alliance_name" class="text-xs text-gray-600">[{{ item.entity_a.alliance_name }}]</div>
                            </td>
                            <td class="px-4 py-2 text-center text-red-500 text-xs font-bold">vs</td>
                            <td class="px-4 py-2">
                                <NuxtLink
                                    :to="rivalryType === 'character' ? `/character/${item.entity_b.id}` : `/alliance/${item.entity_b.id}`"
                                    class="text-blue-400 hover:text-blue-300"
                                >
                                    {{ item.entity_b.name }}
                                </NuxtLink>
                                <div v-if="item.entity_b.corp_name" class="text-xs text-gray-500">{{ item.entity_b.corp_name }}</div>
                                <div v-if="item.entity_b.alliance_name" class="text-xs text-gray-600">[{{ item.entity_b.alliance_name }}]</div>
                            </td>
                            <td class="px-4 py-2 text-right tabular-nums text-white">{{ formatCompact(item.mutual_kills) }}</td>
                            <td class="px-4 py-2 text-right tabular-nums text-gray-400">{{ formatIsk(item.total_isk) }}</td>
                        </tr>
                    </tbody>
                </table>
            </div>

            <div v-else class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]">
                <div class="text-gray-600 text-sm">No rivalry data available</div>
            </div>
        </div>

        <!-- ═══ Entity Intel ═══ -->
        <div v-if="activeTab === 'entity_intel'">
            <div class="glass-panel p-4 mb-4">
                <div class="flex flex-col sm:flex-row gap-3 items-start sm:items-end">
                    <!-- Type toggle -->
                    <div class="flex gap-2">
                        <button
                            @click="intelEntityType = 'alliance'; intelQuery = ''; intelEntityId = null; intelData = null"
                            class="px-3 py-1.5 rounded-md text-sm transition-colors"
                            :class="intelEntityType === 'alliance'
                                ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30'
                                : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white'"
                        >Alliance</button>
                        <button
                            @click="intelEntityType = 'corporation'; intelQuery = ''; intelEntityId = null; intelData = null"
                            class="px-3 py-1.5 rounded-md text-sm transition-colors"
                            :class="intelEntityType === 'corporation'
                                ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30'
                                : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white'"
                        >Corporation</button>
                    </div>

                    <!-- Search -->
                    <div class="relative flex-1 w-full">
                        <input
                            v-model="intelQuery"
                            @input="onIntelInput"
                            :placeholder="`Search for ${intelEntityType === 'alliance' ? 'an alliance' : 'a corporation'}...`"
                            class="w-full bg-white/[0.06] border border-white/[0.1] rounded-md px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
                        />
                        <div v-if="intelResults.length > 0" class="absolute z-20 w-full mt-1 bg-gray-900 border border-white/[0.1] rounded-md shadow-xl max-h-48 overflow-y-auto">
                            <button v-for="r in intelResults" :key="r.id" @click="selectIntelEntity(r)" class="w-full text-left px-3 py-2 text-sm hover:bg-white/[0.06] flex items-baseline gap-2">
                                <span class="text-white">{{ r.name }}</span>
                                <span v-if="r.ticker" class="text-xs text-gray-500">[{{ r.ticker }}]</span>
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Loading -->
            <div v-if="intelPending" class="rounded-lg border border-white/[0.08] bg-white/[0.02] p-8 flex items-center justify-center">
                <div class="text-gray-400 text-sm animate-pulse">Analyzing relationships...</div>
            </div>

            <!-- Results -->
            <div v-else-if="intelData" class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <!-- Allies -->
                <div class="rounded-lg border border-white/[0.08] bg-white/[0.02] overflow-hidden">
                    <div class="px-4 py-2 bg-green-500/10 border-b border-white/[0.06]">
                        <span class="text-sm font-medium text-green-400">Allies</span>
                        <span class="text-xs text-gray-500 ml-2">by shared enemy activity</span>
                    </div>
                    <div v-if="intelData.allies?.length" class="divide-y divide-white/[0.04]">
                        <div v-for="(ally, i) in intelData.allies" :key="ally.id" class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04]">
                            <span class="text-xs text-gray-600 tabular-nums w-5">{{ Number(i) + 1 }}</span>
                            <NuxtLink
                                :to="`/${intelEntityType === 'alliance' ? 'alliance' : 'corporation'}/${ally.id}`"
                                class="text-sm text-blue-400 hover:text-blue-300 flex-1 truncate"
                            >{{ ally.name }}</NuxtLink>
                            <span class="text-xs text-gray-500 tabular-nums shrink-0" v-tooltip="`Shared enemy kills: ${ally.shared_enemy_kills}, Mutual kills: ${ally.mutual_kills}`">
                                {{ formatCompact(ally.shared_enemy_kills) }} shared
                            </span>
                        </div>
                    </div>
                    <div v-else class="p-4 text-sm text-gray-600 text-center">No allies detected</div>
                </div>

                <!-- Enemies -->
                <div class="rounded-lg border border-white/[0.08] bg-white/[0.02] overflow-hidden">
                    <div class="px-4 py-2 bg-red-500/10 border-b border-white/[0.06]">
                        <span class="text-sm font-medium text-red-400">Enemies</span>
                        <span class="text-xs text-gray-500 ml-2">by mutual kills</span>
                    </div>
                    <div v-if="intelData.enemies?.length" class="divide-y divide-white/[0.04]">
                        <div v-for="(enemy, i) in intelData.enemies" :key="enemy.id" class="flex items-center gap-3 px-4 py-1.5 hover:bg-white/[0.04]">
                            <span class="text-xs text-gray-600 tabular-nums w-5">{{ Number(i) + 1 }}</span>
                            <NuxtLink
                                :to="`/${intelEntityType === 'alliance' ? 'alliance' : 'corporation'}/${enemy.id}`"
                                class="text-sm text-blue-400 hover:text-blue-300 flex-1 truncate"
                            >{{ enemy.name }}</NuxtLink>
                            <div class="flex gap-3 shrink-0">
                                <span class="text-xs text-green-400/70 tabular-nums" v-tooltip="'Kills given'">{{ formatCompact(enemy.kills_given) }}</span>
                                <span class="text-xs text-red-400/70 tabular-nums" v-tooltip="'Kills taken'">{{ formatCompact(enemy.kills_taken) }}</span>
                            </div>
                        </div>
                    </div>
                    <div v-else class="p-4 text-sm text-gray-600 text-center">No enemies detected</div>
                </div>
            </div>

            <!-- Empty state -->
            <div v-else-if="!intelEntityId" class="rounded-lg border border-white/[0.08] bg-[#0a0a0f] p-8 flex items-center justify-center min-h-[200px]">
                <div class="text-gray-600 text-sm text-center">
                    Search for an alliance or corporation to see their allies and enemies
                </div>
            </div>
        </div>
    </div>
</template>
