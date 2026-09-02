<script setup lang="ts">
/**
 * /fits/search — filtered fit explorer.
 *
 * Pick a ship, add role-count predicates ("2 autocannons", "1 stasis
 * web"), see canonical killmail-derived fits that match. The hosting
 * page state (ship + filters) is fully URL-synced so visitors can
 * share filtered queries.
 *
 * Backed by:
 *   /api/fits/roles  — static role taxonomy (cached, fetched once)
 *   /api/fits/search — query (cached per ship+filter combo for 2min)
 */

import { killmailFitToEditorUrl } from '~/composables/fit/killmailToFit'

useHead({ title: 'Search Fits' })
useSeoMeta({
    description:
        'Search EVE Online ship fittings by hull and modules. Filter by armor reps, missile launchers, tackle and more — see real fits flown over the last 90 days.',
    ogTitle: 'Search Fits — EVE-KILL',
    ogDescription:
        'Filter ship fits by exact module composition. Pick a hull, add role filters, see the most-used real fits from the last 90 days.',
})

// ---------- types ----------

interface RolePublic {
    id: string
    label: string
    icon: string
    description?: string
    category: 'tank' | 'weapon' | 'ewar' | 'tackle' | 'utility' | 'prop'
    typeCount: number
}

interface SearchHit {
    /** Composite identifier from /api/search — e.g. `ship_587`. Numeric
     *  type_id lives after the underscore. */
    id: string
    type: string
    name: string
    ticker?: string
}

function hitToTypeId(hit: SearchHit): number | null {
    const parts = hit.id.split('_')
    const n = Number(parts[parts.length - 1])
    return Number.isInteger(n) && n > 0 ? n : null
}

interface FittingModule {
    slot_group: number
    ordinal: number
    type_id: number
    name: string | null
    charge_type_id: number | null
    charge_name: string | null
}

interface FittingDrone {
    type_id: number
    name: string | null
    quantity: number
}

interface FitResult {
    fit_hash: string
    family_hash: string
    ship_type_id: number
    ship_name: string | null
    total_uses: number
    family_total_uses: number
    variant_count: number
    last_used: string
    fit_cost: number
    hull_cost: number | null
    modules: FittingModule[]
    drones: FittingDrone[]
    context?: FitFamilyContext
    stats?: {
        dps_with_reload: number | null
        alpha: number | null
        ehp: number | null
        shield_ehp: number | null
        armor_ehp: number | null
        shield_effective_boost: number | null
        armor_effective_repair: number | null
        passive_shield_effective: number | null
        cap_stable: boolean
        max_velocity: number | null
        align_time: number | null
    } | null
}

type FilterOp = '>=' | '<=' | '=' | '>' | '<'
type FilterPredicate =
    | { kind: 'role'; role_id: string; op: FilterOp; count: number }
    | { kind: 'type'; type_id: number; type_name?: string; op: FilterOp; count: number }

// ---------- state, URL-synced ----------

const route = useRoute()
const router = useRouter()

const selectedShip = ref<{ id: number; name: string } | null>(null)
const filters = ref<FilterPredicate[]>([])
const statSort = ref(typeof route.query.sort === 'string' ? route.query.sort : 'uses')
const minDps = ref(typeof route.query.min_dps === 'string' ? route.query.min_dps : '')
const minEhp = ref(typeof route.query.min_ehp === 'string' ? route.query.min_ehp : '')
const minRepair = ref(typeof route.query.min_repair === 'string' ? route.query.min_repair : '')
const capStable = ref(route.query.cap_stable === 'true')

// Hydrate from query string on first load.
{
    const shipId = Number(route.query.ship)
    const shipName = typeof route.query.ship_name === 'string' ? route.query.ship_name : ''
    if (Number.isInteger(shipId) && shipId > 0) {
        selectedShip.value = { id: shipId, name: shipName || `Type ${shipId}` }
    }
    if (typeof route.query.filters === 'string') {
        try {
            const parsed = JSON.parse(route.query.filters)
            if (Array.isArray(parsed)) {
                filters.value = parsed
                    .map((f): FilterPredicate | null => {
                        if (!f || typeof f.op !== 'string' || typeof f.count !== 'number') return null
                        if (typeof f.role_id === 'string') {
                            return { kind: 'role', role_id: f.role_id, op: f.op, count: f.count }
                        }
                        if (typeof f.type_id === 'number') {
                            return {
                                kind: 'type',
                                type_id: f.type_id,
                                type_name: typeof f.type_name === 'string' ? f.type_name : undefined,
                                op: f.op,
                                count: f.count,
                            }
                        }
                        return null
                    })
                    .filter((f): f is FilterPredicate => f !== null)
            }
        } catch {
            // ignore malformed query
        }
    }
}

const queryString = computed(() => {
    const params: Record<string, string> = {}
    if (selectedShip.value) {
        params.ship = String(selectedShip.value.id)
        if (selectedShip.value.name) params.ship_name = selectedShip.value.name
    }
    if (filters.value.length > 0) {
        params.filters = JSON.stringify(filters.value)
    }
    if (statSort.value !== 'uses') params.sort = statSort.value
    if (minDps.value) params.min_dps = minDps.value
    if (minEhp.value) params.min_ehp = minEhp.value
    if (minRepair.value) params.min_repair = minRepair.value
    if (capStable.value) params.cap_stable = 'true'
    return params
})

// Keep URL in sync — replace, not push, so back-button doesn't get
// polluted with every filter toggle.
watch(queryString, (q) => {
    router.replace({ path: '/fits/search', query: q })
}, { deep: true })

// ---------- roles ----------

const { data: rolesData } = await useApiFetch<{ roles: RolePublic[] }>('/api/fits/roles', { lazy: true })
const roles = computed<RolePublic[]>(() => rolesData.value?.roles ?? [])
const roleById = computed(() => {
    const m = new Map<string, RolePublic>()
    for (const r of roles.value) m.set(r.id, r)
    return m
})
const rolesByCategory = computed(() => {
    const m = new Map<RolePublic['category'], RolePublic[]>()
    for (const r of roles.value) {
        if (r.typeCount === 0) continue
        const list = m.get(r.category) ?? []
        list.push(r)
        m.set(r.category, list)
    }
    return m
})
const CATEGORY_ORDER: RolePublic['category'][] = ['weapon', 'tank', 'tackle', 'prop', 'ewar', 'utility']
const CATEGORY_LABEL: Record<RolePublic['category'], string> = {
    weapon: 'Weapons',
    tank: 'Tank',
    tackle: 'Tackle',
    prop: 'Propulsion',
    ewar: 'EWAR',
    utility: 'Utility',
}

// ---------- ship picker ----------

const shipQuery = ref('')
const shipResults = ref<SearchHit[]>([])
const shipSearchOpen = ref(false)
let shipSearchTimer: ReturnType<typeof setTimeout> | null = null

watch(shipQuery, (q) => {
    if (shipSearchTimer) clearTimeout(shipSearchTimer)
    const trimmed = q.trim()
    if (trimmed.length < 2) {
        shipResults.value = []
        shipSearchOpen.value = false
        return
    }
    shipSearchTimer = setTimeout(async () => {
        try {
            const data = await apiFetch<{ hits: SearchHit[] }>('/api/search', {
                params: { q: trimmed, type: 'ship' },
            })
            shipResults.value = (data.hits ?? []).filter((h) => h.type === 'ship').slice(0, 12)
            shipSearchOpen.value = shipResults.value.length > 0
        } catch {
            shipResults.value = []
            shipSearchOpen.value = false
        }
    }, 200)
})

function selectShip(hit: SearchHit) {
    const numId = hitToTypeId(hit)
    if (numId === null) return
    selectedShip.value = { id: numId, name: hit.name }
    shipQuery.value = ''
    shipResults.value = []
    shipSearchOpen.value = false
}

function clearShip() {
    selectedShip.value = null
    filters.value = []
    shipQuery.value = ''
}

// Popular ships fallback — fetched once so the picker doesn't start empty.
const { data: popularData } = await useApiFetch<{
    window_days: number
    ships: { ship_type_id: number; total_uses: number; fit_count: number; ship_name: string | null }[]
}>('/api/fits/popular-ships', { lazy: true })
const popularShips = computed(() => (popularData.value?.ships ?? []).slice(0, 12))

const formatCompactNumber = (value: number): string =>
    new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 1 }).format(value)

// ---------- filters ----------

const filterPickerOpen = ref(false)
const filterRoleSearch = ref('')

const visibleRolesForPicker = computed(() => {
    const q = filterRoleSearch.value.trim().toLowerCase()
    if (!q) return roles.value
    return roles.value.filter(
        (r) =>
            r.label.toLowerCase().includes(q) ||
            (r.description?.toLowerCase().includes(q) ?? false),
    )
})

// Specific-item search runs in parallel with the role-name filter so
// the dropdown can show both lists. We only hit /api/search once the
// query is non-trivial; otherwise it sits empty.
const itemResults = ref<SearchHit[]>([])
const itemSearchPending = ref(false)
let itemSearchTimer: ReturnType<typeof setTimeout> | null = null

watch(filterRoleSearch, (q) => {
    if (itemSearchTimer) clearTimeout(itemSearchTimer)
    const trimmed = q.trim()
    if (trimmed.length < 2) {
        itemResults.value = []
        return
    }
    itemSearchTimer = setTimeout(async () => {
        itemSearchPending.value = true
        try {
            const data = await apiFetch<{ hits: SearchHit[] }>('/api/search', {
                params: { q: trimmed, type: 'item' },
            })
            itemResults.value = (data.hits ?? []).filter((h) => h.type === 'item').slice(0, 12)
        } catch {
            itemResults.value = []
        } finally {
            itemSearchPending.value = false
        }
    }, 200)
})

function addRoleFilter(role: RolePublic) {
    if (filters.value.length >= 8) return
    if (filters.value.some((f) => f.kind === 'role' && f.role_id === role.id)) return
    filters.value.push({ kind: 'role', role_id: role.id, op: '>=', count: 1 })
    filterPickerOpen.value = false
    filterRoleSearch.value = ''
    itemResults.value = []
}

function addTypeFilter(hit: SearchHit) {
    const tid = hitToTypeId(hit)
    if (tid === null) return
    if (filters.value.length >= 8) return
    if (filters.value.some((f) => f.kind === 'type' && f.type_id === tid)) return
    filters.value.push({ kind: 'type', type_id: tid, type_name: hit.name, op: '>=', count: 1 })
    filterPickerOpen.value = false
    filterRoleSearch.value = ''
    itemResults.value = []
}

function removeFilter(idx: number) {
    filters.value.splice(idx, 1)
}

function bumpCount(idx: number, delta: number) {
    const f = filters.value[idx]
    if (!f) return
    const next = Math.max(0, Math.min(16, f.count + delta))
    filters.value[idx] = { ...f, count: next } as FilterPredicate
}

function setOp(idx: number, op: FilterOp) {
    const f = filters.value[idx]
    if (!f) return
    filters.value[idx] = { ...f, op } as FilterPredicate
}

function filterLabel(f: FilterPredicate): string {
    if (f.kind === 'role') return roleById.value.get(f.role_id)?.label ?? f.role_id
    return f.type_name ?? `Type ${f.type_id}`
}
function filterIcon(f: FilterPredicate): string {
    if (f.kind === 'role') return roleById.value.get(f.role_id)?.icon ?? 'lucide:circle'
    return 'lucide:package'
}
function filterImage(f: FilterPredicate): string | null {
    if (f.kind === 'type') return `/images/types/${f.type_id}/icon?size=32`
    return null
}

// ---------- results ----------

const offset = ref(0)
const limit = 24

interface SearchResponse {
    ship_type_id: number
    ship_name: string | null
    window_days: number
    total: number
    has_more: boolean
    offset: number
    limit: number
    filters_applied: FilterPredicate[]
    fits: FitResult[]
}

const resultsData = ref<SearchResponse | null>(null)
const resultsPending = ref(false)
let searchGeneration = 0

const total = computed(() => resultsData.value?.total ?? 0)
const fits = computed<FitResult[]>(() => resultsData.value?.fits ?? [])
const hasMore = computed(() => resultsData.value?.has_more ?? false)

async function runSearch() {
    const generation = ++searchGeneration
    const ship = selectedShip.value
    if (!ship) {
        resultsData.value = null
        resultsPending.value = false
        return
    }
    const requestedOffset = offset.value
    resultsPending.value = true
    try {
        const data = await apiFetch<SearchResponse>('/api/fits/search', {
            params: {
                ship: ship.id,
                filters: filters.value.length > 0 ? JSON.stringify(filters.value) : undefined,
                limit,
                offset: offset.value,
                sort: statSort.value,
                min_dps: minDps.value || undefined,
                min_ehp: minEhp.value || undefined,
                min_repair: minRepair.value || undefined,
                cap_stable: capStable.value || undefined,
            },
        })
        if (generation !== searchGeneration) return
        if (requestedOffset > 0 && resultsData.value?.ship_type_id === data.ship_type_id) {
            resultsData.value = {
                ...data,
                fits: [...resultsData.value.fits, ...data.fits],
            }
        } else {
            resultsData.value = data
        }
        if (data.ship_name && data.ship_name !== ship.name) {
            selectedShip.value = { id: ship.id, name: data.ship_name }
        }
        if (data.fits.length > 0) buildFitUrls(data.fits)
    } catch {
        if (generation === searchGeneration) {
            if (requestedOffset === 0) resultsData.value = null
            else offset.value = Math.max(0, requestedOffset - limit)
        }
    } finally {
        if (generation === searchGeneration) resultsPending.value = false
    }
}

// Re-fetch on ship/filter changes; loadMore drives offset and triggers
// its own re-fetch explicitly. No watch on `offset` means changing
// filters doesn't double-fetch when it resets offset.
watch(
    [() => selectedShip.value?.id, filters, statSort, minDps, minEhp, minRepair, capStable],
    () => {
        offset.value = 0
        fitUrls.value = new Map()
        runSearch()
    },
    { deep: true },
)

onMounted(() => {
    if (selectedShip.value) runSearch()
})

function loadMore() {
    if (resultsPending.value || !hasMore.value) return
    offset.value += limit
    runSearch()
}

// ---------- helpers ----------

const shipRenderUrl = (typeId: number) =>
    `/images/types/${typeId}/render?size=256`

const moduleIconUrl = (typeId: number) =>
    `/images/types/${typeId}/icon?size=32`

const SLOT_LABELS: Record<number, string> = {
    1: 'High',
    2: 'Mid',
    3: 'Low',
    4: 'Rigs',
    5: 'Subsystems',
}

function fittingGroups(fit: FitResult) {
    const groups = [1, 2, 3, 4, 5]
        .map(slot => ({
            slot,
            label: SLOT_LABELS[slot] ?? 'Other',
            modules: fit.modules.filter(module => module.slot_group === slot),
        }))
        .filter(group => group.modules.length > 0)
    if (fit.drones.length > 0) {
        groups.push({
            slot: 6,
            label: 'Drones',
            modules: fit.drones.map((drone, index) => ({
                slot_group: 6,
                ordinal: index,
                type_id: drone.type_id,
                name: drone.name,
                charge_type_id: null,
                charge_name: drone.quantity > 1 ? `${drone.quantity}×` : null,
            })),
        })
    }
    return groups
}

const fitName = (fit: FitResult): string => classifyFitFamily(fit.modules, fit.drones)
const fitInsights = (fit: FitResult): string[] => fitFamilyContextParts(fit.context)
const strongestRepair = (fit: FitResult): number => Math.max(
    fit.stats?.shield_effective_boost ?? 0,
    fit.stats?.armor_effective_repair ?? 0,
    fit.stats?.passive_shield_effective ?? 0,
)

const timeAgo = (iso: string | null): string => {
    if (!iso) return ''
    const diff = Date.now() - new Date(iso).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 60) return `${mins}m ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    if (days < 30) return `${days}d ago`
    return `${Math.floor(days / 30)}mo ago`
}


const fitUrls = ref(new Map<string, string>())

async function buildFitUrls(results: FitResult[]) {
    const entries = await Promise.all(
        results.map(async (f) => {
            const url = await killmailFitToEditorUrl({
                shipTypeId: f.ship_type_id,
                modules: f.modules,
                drones: f.drones,
                name: `${f.ship_name ?? 'Community'} Fit`,
                description: `Killmail-derived fit — ${f.total_uses} recorded uses in the last 90 days.`,
            })
            return [f.fit_hash, url] as const
        }),
    )
    const m = new Map(fitUrls.value)
    for (const [k, v] of entries) m.set(k, v)
    fitUrls.value = m
}
</script>

<template>
    <div>
        <PageHeader class="mb-6" title="Find killmail-derived fits" eyebrow="Fit Search" icon="lucide:search">
            <template #description>
                        Pick a ship, then add role filters like
                        <span class="text-blue-400/80">2 armor reps</span> or
                        <span class="text-blue-400/80">1 stasis webifier</span>. We surface the
                        most-flown matching fits from the last 90 days of killmails.
            </template>
            <template #actions>
                <NuxtLink
                    to="/fits"
                    class="inline-flex items-center gap-2 px-3 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md bg-white/[0.04] text-gray-400 border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors"
                >
                    <Icon name="lucide:arrow-left" class="w-3.5 h-3.5" />
                    Back to Fits
                </NuxtLink>
            </template>
        </PageHeader>

        <!-- ============================ Ship Picker ============================ -->
        <section class="mb-4">
            <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">
                Ship
            </label>

            <div v-if="selectedShip" class="relative overflow-hidden rounded-xl border border-blue-500/25 bg-black/35 p-4">
                <img
                    :src="shipRenderUrl(selectedShip.id)"
                    alt=""
                    class="pointer-events-none absolute -right-10 -top-20 h-56 w-56 opacity-20 blur-xl"
                />
                <div class="relative flex items-center gap-4">
                <img
                    :src="shipRenderUrl(selectedShip.id)"
                    :alt="selectedShip.name"
                    class="h-20 w-20 flex-shrink-0 object-contain drop-shadow-[0_8px_20px_rgba(59,130,246,0.2)]"
                    loading="lazy"
                />
                <div class="min-w-0 flex-1">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Selected hull</div>
                    <div class="mt-1 truncate text-xl font-bold text-white">{{ selectedShip.name }}</div>
                    <div class="mt-1 text-xs text-gray-500">Observed fitting families from the last 90 days</div>
                </div>
                <NuxtLink
                    :to="`/item/${selectedShip.id}/fittings`"
                    class="hidden items-center gap-1.5 rounded-md border border-white/[0.08] bg-white/[0.04] px-3 py-2 text-xs text-gray-400 transition-colors hover:border-blue-500/30 hover:text-blue-300 sm:inline-flex"
                >
                    <Icon name="lucide:layers-3" class="h-3.5 w-3.5" />
                    Hull fittings
                </NuxtLink>
                <button
                    type="button"
                    class="flex items-center justify-center w-8 h-8 rounded-md text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                    @click="clearShip"
                    aria-label="Change ship"
                >
                    <Icon name="lucide:x" class="w-4 h-4" />
                </button>
                </div>
            </div>

            <div v-else>
                <div class="relative">
                    <Icon
                        name="lucide:search"
                        class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 pointer-events-none"
                    />
                    <input
                        v-model="shipQuery"
                        type="text"
                        placeholder="Search for a hull — Rifter, Stabber, Vargur…"
                        class="w-full h-11 pl-10 pr-3 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder:text-gray-500 focus:outline-none focus:border-blue-500/50"
                        @focus="shipSearchOpen = shipResults.length > 0"
                    />
                    <div
                        v-if="shipSearchOpen && shipResults.length > 0"
                        class="absolute z-20 left-0 right-0 mt-1 rounded-lg bg-black/95 border border-white/[0.08] backdrop-blur-md overflow-hidden max-h-72 overflow-y-auto"
                    >
                        <button
                            v-for="hit in shipResults"
                            :key="hit.id"
                            type="button"
                            class="w-full flex items-center gap-3 px-3 py-2 hover:bg-blue-500/[0.08] transition-colors text-left"
                            @click="selectShip(hit)"
                        >
                            <img
                                :src="`/images/types/${hitToTypeId(hit) ?? 0}/icon?size=32`"
                                :alt="hit.name"
                                class="w-8 h-8 rounded bg-black/30 flex-shrink-0"
                                loading="lazy"
                            />
                            <div class="text-sm text-gray-200 flex-1 truncate">{{ hit.name }}</div>
                        </button>
                    </div>
                </div>

                <!-- Popular attacker hulls as quick-pick cards -->
                <div v-if="popularShips.length > 0" class="mt-3">
                    <div class="mb-3 flex items-end justify-between gap-3">
                        <div>
                            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">
                                Popular hulls
                            </div>
                            <div class="mt-1 text-sm text-gray-400">
                                Ships participating in the most kills over the last {{ popularData?.window_days ?? 30 }} days
                            </div>
                        </div>
                        <div class="hidden text-fine text-gray-600 sm:block">Click a hull to explore its fits</div>
                    </div>
                    <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
                        <button
                            v-for="(ship, index) in popularShips"
                            :key="ship.ship_type_id"
                            type="button"
                            class="group relative flex min-w-0 items-center gap-3 overflow-hidden rounded-lg border border-white/[0.08] bg-white/[0.025] p-3 text-left transition-all hover:-translate-y-0.5 hover:border-blue-500/30 hover:bg-blue-500/[0.06]"
                            @click="selectShip({ id: String(ship.ship_type_id), type: 'ship', name: ship.ship_name ?? `Type ${ship.ship_type_id}` })"
                        >
                            <span class="absolute right-2 top-1 text-[10px] font-bold tabular-nums text-white/[0.08]">
                                {{ String(index + 1).padStart(2, '0') }}
                            </span>
                            <img
                                :src="shipRenderUrl(ship.ship_type_id)"
                                :alt="ship.ship_name ?? ''"
                                class="h-12 w-12 flex-shrink-0 object-contain transition-transform duration-200 group-hover:scale-110"
                                loading="lazy"
                            />
                            <span class="min-w-0 flex-1">
                                <span class="block truncate text-sm font-semibold text-gray-200 group-hover:text-blue-300">
                                    {{ ship.ship_name ?? `Type ${ship.ship_type_id}` }}
                                </span>
                                <span class="mt-0.5 block text-fine text-gray-500">
                                    <span class="font-semibold tabular-nums text-gray-300">{{ formatCompactNumber(ship.total_uses) }}</span>
                                    kill participations
                                </span>
                                <span class="block text-[10px] text-gray-600">
                                    {{ ship.fit_count.toLocaleString('en-US') }} observed fit families
                                </span>
                            </span>
                        </button>
                    </div>
                </div>
            </div>
        </section>

        <!-- ============================ Filters ============================ -->
        <section v-if="selectedShip" class="mb-6">
            <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">
                Module Filters
            </label>

            <div class="glass-panel p-3 space-y-2">
                <div class="grid gap-2 border-b border-white/[0.06] pb-3 sm:grid-cols-2 lg:grid-cols-6">
                    <label class="space-y-1 lg:col-span-2">
                        <span class="text-fine font-bold uppercase tracking-wider text-gray-600">Sort by</span>
                        <select v-model="statSort" class="h-9 w-full rounded-md border border-white/[0.08] bg-black/30 px-2 text-xs text-gray-300 outline-none focus:border-blue-500/40">
                            <option value="uses">Most used</option><option value="dps">Highest DPS</option>
                            <option value="ehp">Highest EHP</option><option value="alpha">Highest alpha</option>
                            <option value="repair">Strongest local tank</option><option value="speed">Fastest</option>
                            <option value="align">Quickest align</option>
                        </select>
                    </label>
                    <label class="space-y-1"><span class="text-fine font-bold uppercase tracking-wider text-gray-600">Min DPS</span><input v-model="minDps" type="number" min="0" class="h-9 w-full rounded-md border border-white/[0.08] bg-black/30 px-2 text-xs text-gray-300 outline-none focus:border-blue-500/40" placeholder="Any" /></label>
                    <label class="space-y-1"><span class="text-fine font-bold uppercase tracking-wider text-gray-600">Min EHP</span><input v-model="minEhp" type="number" min="0" class="h-9 w-full rounded-md border border-white/[0.08] bg-black/30 px-2 text-xs text-gray-300 outline-none focus:border-blue-500/40" placeholder="Any" /></label>
                    <label class="space-y-1"><span class="text-fine font-bold uppercase tracking-wider text-gray-600">Min EHP/s</span><input v-model="minRepair" type="number" min="0" class="h-9 w-full rounded-md border border-white/[0.08] bg-black/30 px-2 text-xs text-gray-300 outline-none focus:border-blue-500/40" placeholder="Any" /></label>
                    <label class="flex h-9 self-end items-center gap-2 rounded-md border border-white/[0.08] bg-black/30 px-2 text-xs text-gray-400"><input v-model="capStable" type="checkbox" class="accent-blue-500" />Cap stable</label>
                </div>
                <!-- Existing filter pills -->
                <div v-if="filters.length === 0" class="text-fine text-gray-500 py-1">
                    No filters yet — leave empty to see the most-flown fits for this hull.
                </div>

                <div
                    v-for="(f, idx) in filters"
                    :key="`${f.kind}-${f.kind === 'role' ? f.role_id : f.type_id}-${idx}`"
                    class="flex items-center gap-2 flex-wrap"
                >
                    <div class="flex items-center gap-2 px-2.5 py-1.5 rounded-md bg-blue-500/[0.08] border border-blue-500/30 text-sm">
                        <img
                            v-if="filterImage(f)"
                            :src="filterImage(f) ?? ''"
                            class="w-4 h-4 rounded-sm"
                            loading="lazy"
                            alt=""
                        />
                        <Icon
                            v-else
                            :name="filterIcon(f)"
                            class="w-4 h-4 text-blue-400"
                        />
                        <span class="text-gray-200 font-medium">
                            {{ filterLabel(f) }}
                        </span>
                    </div>

                    <!-- Op selector -->
                    <div class="flex items-center rounded-md overflow-hidden border border-white/[0.08]">
                        <button
                            v-for="op in (['>=', '=', '<='] as const)"
                            :key="op"
                            type="button"
                            class="px-2 py-1 text-xs font-medium transition-colors"
                            :class="
                                f.op === op
                                    ? 'bg-blue-500/20 text-blue-300'
                                    : 'bg-white/[0.02] text-gray-500 hover:text-gray-300'
                            "
                            @click="setOp(idx, op)"
                        >
                            {{ op }}
                        </button>
                    </div>

                    <!-- Count stepper -->
                    <div class="flex items-center rounded-md overflow-hidden border border-white/[0.08]">
                        <button
                            type="button"
                            class="w-7 h-7 flex items-center justify-center text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors"
                            @click="bumpCount(idx, -1)"
                            aria-label="Decrement"
                        >
                            <Icon name="lucide:minus" class="w-3 h-3" />
                        </button>
                        <div class="px-2.5 text-sm tabular-nums text-white min-w-[2rem] text-center">
                            {{ f.count }}
                        </div>
                        <button
                            type="button"
                            class="w-7 h-7 flex items-center justify-center text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors"
                            @click="bumpCount(idx, +1)"
                            aria-label="Increment"
                        >
                            <Icon name="lucide:plus" class="w-3 h-3" />
                        </button>
                    </div>

                    <button
                        type="button"
                        class="flex items-center justify-center w-7 h-7 rounded-md text-gray-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                        @click="removeFilter(idx)"
                        aria-label="Remove filter"
                    >
                        <Icon name="lucide:x" class="w-3.5 h-3.5" />
                    </button>
                </div>

                <!-- Add filter trigger -->
                <div class="relative">
                    <button
                        type="button"
                        class="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-white/[0.04] border border-white/[0.08] text-xs font-medium text-gray-300 hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors"
                        :disabled="filters.length >= 8"
                        @click="filterPickerOpen = !filterPickerOpen"
                    >
                        <Icon name="lucide:plus" class="w-3.5 h-3.5" />
                        Add filter
                        <span v-if="filters.length > 0" class="text-fine text-gray-500">
                            ({{ filters.length }}/8)
                        </span>
                    </button>

                    <!-- Role + specific-item picker dropdown -->
                    <div
                        v-if="filterPickerOpen"
                        class="absolute z-20 mt-1 left-0 w-[min(36rem,calc(100vw-2rem))] rounded-lg bg-black/95 border border-white/[0.08] backdrop-blur-md p-3 max-h-[28rem] overflow-y-auto"
                    >
                        <input
                            v-model="filterRoleSearch"
                            type="text"
                            placeholder="Search role (pulse, rep, web) or specific item (Damage Control II)…"
                            class="w-full h-9 px-3 mb-3 rounded-md bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder:text-gray-500 focus:outline-none focus:border-blue-500/50"
                            autofocus
                        />
                        <div v-if="filterRoleSearch.trim().length > 0">
                            <!-- Role matches -->
                            <div v-if="visibleRolesForPicker.length > 0" class="mb-3">
                                <div class="text-fine font-bold uppercase tracking-wider text-blue-400/70 mb-1 px-1">
                                    Roles
                                </div>
                                <button
                                    v-for="r in visibleRolesForPicker"
                                    :key="r.id"
                                    type="button"
                                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-blue-500/[0.08] transition-colors text-left disabled:opacity-40 disabled:cursor-not-allowed"
                                    :disabled="filters.some((f) => f.kind === 'role' && f.role_id === r.id)"
                                    @click="addRoleFilter(r)"
                                >
                                    <Icon :name="r.icon" class="w-4 h-4 text-gray-400" />
                                    <span class="text-sm text-gray-200 flex-1">{{ r.label }}</span>
                                    <span class="text-fine text-gray-600 tabular-nums">{{ r.typeCount }}</span>
                                </button>
                            </div>

                            <!-- Specific item matches -->
                            <div v-if="itemResults.length > 0" class="mb-1">
                                <div class="text-fine font-bold uppercase tracking-wider text-amber-400/70 mb-1 px-1">
                                    Specific items
                                </div>
                                <button
                                    v-for="hit in itemResults"
                                    :key="hit.id"
                                    type="button"
                                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-amber-500/[0.08] transition-colors text-left disabled:opacity-40 disabled:cursor-not-allowed"
                                    :disabled="filters.some((f) => f.kind === 'type' && f.type_id === hitToTypeId(hit))"
                                    @click="addTypeFilter(hit)"
                                >
                                    <img
                                        :src="`/images/types/${hitToTypeId(hit) ?? 0}/icon?size=32`"
                                        :alt="hit.name"
                                        class="w-5 h-5 rounded-sm bg-black/30 flex-shrink-0"
                                        loading="lazy"
                                    />
                                    <span class="text-sm text-gray-200 flex-1 truncate">{{ hit.name }}</span>
                                </button>
                            </div>

                            <div
                                v-if="visibleRolesForPicker.length === 0 && itemResults.length === 0 && !itemSearchPending"
                                class="text-fine text-gray-500 px-2 py-2"
                            >
                                No matches — try a different word.
                            </div>
                            <div v-if="itemSearchPending" class="text-fine text-gray-500 px-2 py-1 flex items-center gap-2">
                                <Icon name="lucide:loader-2" class="w-3 h-3 animate-spin" />
                                Searching items…
                            </div>
                        </div>
                        <div v-else>
                            <template v-for="cat in CATEGORY_ORDER" :key="cat">
                                <div
                                    v-if="rolesByCategory.get(cat)?.length"
                                    class="mb-3 last:mb-0"
                                >
                                    <div class="text-fine font-bold uppercase tracking-wider text-blue-400/70 mb-1">
                                        {{ CATEGORY_LABEL[cat] }}
                                    </div>
                                    <div class="grid grid-cols-2 gap-x-2">
                                        <button
                                            v-for="r in rolesByCategory.get(cat)"
                                            :key="r.id"
                                            type="button"
                                            class="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-blue-500/[0.08] transition-colors text-left disabled:opacity-40 disabled:cursor-not-allowed"
                                            :disabled="filters.some((f) => f.kind === 'role' && f.role_id === r.id)"
                                            @click="addRoleFilter(r)"
                                        >
                                            <Icon :name="r.icon" class="w-4 h-4 text-gray-400 flex-shrink-0" />
                                            <span class="text-sm text-gray-200 flex-1 truncate">{{ r.label }}</span>
                                        </button>
                                    </div>
                                </div>
                            </template>
                            <div class="text-fine text-gray-500 px-1 pt-1 border-t border-white/[0.06]">
                                Tip: type the start of a module name to search for specific items
                                (e.g. <span class="text-amber-300/80">Damage Control II</span>).
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <!-- ============================ Results ============================ -->
        <section v-if="selectedShip">
            <div class="flex items-end justify-between mb-3">
                <div>
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1">
                        Last 90 Days
                    </div>
                    <h2 class="text-lg font-bold text-white">
                        {{ total.toLocaleString('en-US') }}
                        matching fit{{ total === 1 ? '' : 's' }}
                    </h2>
                </div>
                <div class="text-fine text-gray-500 hidden md:block">
                    Click a fit to open it in the editor
                </div>
            </div>

            <div
                v-if="resultsPending && fits.length === 0"
                class="glass-panel flex items-center justify-center py-20"
            >
                <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
            </div>

            <div
                v-else-if="fits.length === 0"
                class="glass-panel py-12 text-center text-sm text-gray-500"
            >
                No fits match — try loosening a filter.
            </div>

            <div v-else>
                <div class="space-y-2.5">
                    <article
                        v-for="(f, index) in fits"
                        :key="f.fit_hash"
                        class="group relative overflow-hidden rounded-xl border border-white/[0.08] bg-white/[0.025] transition-colors hover:border-blue-500/25 hover:bg-blue-500/[0.035]"
                    >
                        <div class="flex min-w-0 items-center gap-2.5 px-3.5 pb-0 pt-3">
                            <span class="text-[10px] font-bold tabular-nums text-white/20">
                                {{ String(index + 1).padStart(2, '0') }}
                            </span>
                            <h3 class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-100 transition-colors group-hover:text-blue-300">
                                {{ fitName(f) }}
                            </h3>
                            <div class="flex flex-shrink-0 items-center gap-1.5">
                                <NuxtLink
                                    :to="fitUrls.get(f.fit_hash) ?? '#'"
                                    class="inline-flex items-center justify-center gap-1.5 rounded-md border border-blue-500/25 bg-blue-500/[0.08] px-2.5 py-1.5 text-fine font-semibold text-blue-300 transition-colors hover:bg-blue-500/[0.15]"
                                >
                                    <Icon name="lucide:wrench" class="h-3 w-3" />
                                    <span class="hidden sm:inline">Open editor</span>
                                </NuxtLink>
                                <NuxtLink
                                    :to="`/item/${f.ship_type_id}/fittings`"
                                    class="inline-flex items-center justify-center gap-1.5 rounded-md px-2.5 py-1.5 text-fine text-gray-500 transition-colors hover:bg-white/[0.04] hover:text-gray-300"
                                >
                                    <Icon name="lucide:layers-3" class="h-3 w-3" />
                                    <span class="hidden sm:inline">All variants</span>
                                </NuxtLink>
                            </div>
                        </div>
                        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 px-3.5 pt-1.5 text-fine">
                            <span class="font-bold tabular-nums text-blue-400">
                                {{ f.total_uses.toLocaleString('en-US') }} exact uses
                            </span>
                            <span class="tabular-nums text-purple-300/80">
                                {{ f.family_total_uses.toLocaleString('en-US') }} family uses
                            </span>
                            <span class="text-gray-500">
                                {{ f.variant_count.toLocaleString('en-US') }} variant{{ f.variant_count === 1 ? '' : 's' }}
                            </span>
                            <template v-if="f.stats">
                                <span v-if="f.stats.dps_with_reload != null" class="tabular-nums text-red-300/80">{{ Math.round(f.stats.dps_with_reload) }} DPS</span>
                                <span v-if="f.stats.ehp != null" class="tabular-nums text-emerald-300/80">{{ formatCompactNumber(f.stats.ehp) }} EHP</span>
                                <span v-if="strongestRepair(f) > 0" class="tabular-nums text-cyan-300/80">{{ Math.round(strongestRepair(f)) }} EHP/s</span>
                                <span v-if="f.stats.cap_stable" class="text-blue-300/80">Cap stable</span>
                            </template>
                            <span
                                v-for="insight in fitInsights(f).slice(0, 3)"
                                :key="insight"
                                class="inline-flex items-center gap-1 text-gray-500"
                            >
                                <span class="text-white/10">·</span>
                                {{ insight }}
                            </span>
                        </div>
                        <div class="flex flex-col gap-3 p-3.5 lg:flex-row lg:items-stretch">
                            <div class="flex min-w-0 items-center gap-3 lg:w-72 lg:flex-shrink-0">
                                <div class="relative flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-lg bg-black/30">
                            <img
                                :src="shipRenderUrl(f.ship_type_id)"
                                :alt="f.ship_name ?? ''"
                                        class="h-14 w-14 object-contain"
                                loading="lazy"
                            />
                                </div>
                                <div class="min-w-0 flex-1">
                                    <div class="truncate text-xs font-medium text-gray-400">
                                        {{ f.ship_name ?? `Type ${f.ship_type_id}` }}
                                    </div>
                                    <div class="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-fine">
                                        <span class="tabular-nums text-yellow-400/80">
                                            {{ formatIsk((f.fit_cost ?? 0) + (f.hull_cost ?? 0)) }} ISK
                                        </span>
                                        <span class="text-gray-600">seen {{ timeAgo(f.last_used) }}</span>
                                    </div>
                                </div>
                            </div>

                            <div class="grid min-w-0 flex-1 grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-flow-col xl:auto-cols-fr xl:grid-cols-none">
                                <div
                                    v-for="group in fittingGroups(f)"
                                    :key="group.slot"
                                    class="min-w-0 rounded-md border border-white/[0.05] bg-black/15 px-2 py-1.5"
                                >
                                    <div class="mb-1 text-[9px] font-bold uppercase tracking-[0.12em] text-gray-600">
                                        {{ group.label }} · {{ group.modules.length }}
                                    </div>
                                    <div class="space-y-1">
                                        <div
                                            v-for="module in group.modules.slice(0, 3)"
                                            :key="`${module.type_id}-${module.ordinal}`"
                                            class="flex min-w-0 items-center gap-1.5"
                                            :title="module.name ?? `Type ${module.type_id}`"
                                        >
                                            <img
                                                :src="moduleIconUrl(module.type_id)"
                                                alt=""
                                                class="h-4 w-4 flex-shrink-0 rounded-sm"
                                                loading="lazy"
                                            />
                                            <span class="truncate text-[10px] text-gray-400">
                                                <span v-if="module.charge_name" class="mr-1 text-gray-600">{{ module.charge_name }}</span>
                                                {{ module.name ?? `Type ${module.type_id}` }}
                                            </span>
                                        </div>
                                        <div v-if="group.modules.length > 3" class="pl-5 text-[9px] text-gray-600">
                                            +{{ group.modules.length - 3 }} more
                                        </div>
                                    </div>
                                </div>
                            </div>

                        </div>
                    </article>
                </div>

                <div v-if="hasMore" class="mt-4 flex justify-center">
                    <button
                        type="button"
                        class="inline-flex items-center gap-2 px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md bg-white/[0.04] text-gray-300 border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors"
                        :disabled="resultsPending"
                        @click="loadMore"
                    >
                        <Icon v-if="resultsPending" name="lucide:loader-2" class="w-3.5 h-3.5 animate-spin" />
                        <Icon v-else name="lucide:plus" class="w-3.5 h-3.5" />
                        Show more fits
                    </button>
                </div>
            </div>
        </section>

        <!-- Empty-state when nothing has been picked yet -->
        <section v-else class="glass-panel py-16 text-center">
            <Icon name="lucide:ship" class="w-10 h-10 text-gray-600 mx-auto mb-3" />
            <div class="text-sm text-gray-300 mb-1">Pick a ship to start</div>
            <div class="text-fine text-gray-500">
                Once you pick a hull, you can layer on role filters.
            </div>
        </section>
    </div>
</template>
