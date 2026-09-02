<script setup lang="ts">
// Item Fittings tab — renders the most popular fits for a ship over the last
// 90 days. Data comes from /api/item/:id/fittings which groups by family_hash
// (T1/T2/meta variants of the same fit collapse into one row).
//
// See backend/src/services/FittingExtractor.ts for how the slot_group bucketing
// works. Visual order is high → med → low → rig → subsystem.
import { killmailFitToEditorUrl } from "~/composables/fit/killmailToFit"

const props = defineProps<{
    shipTypeId: number
    hullGroupName?: string | null
}>()
const route = useRoute()
const router = useRouter()

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

interface AllianceUsage {
    alliance_id: number
    name: string | null
    uses: number
    pct_of_alliance_losses: number
}

interface FittingFamily {
    family_hash: string
    canonical_fit_hash: string
    total_uses: number
    canonical_uses: number
    variant_count: number
    last_used: string
    fit_cost: number
    modules: FittingModule[]
    drones: FittingDrone[]
    top_alliances: AllianceUsage[]
    context?: FitFamilyContext
    stats?: {
        ehp: number | null
        dps: number | null
        alpha: number | null
        speed: number | null
        align: number | null
        repair: number | null
        npc_profile: string
        npc_ehp: number | null
    }
}

interface FittingsResponse {
    ship_type_id: number
    window_days: number
    is_rare_hull: boolean
    hull_cost: number | null
    families: FittingFamily[]
}

type FitFilterMetric = 'ehp' | 'dps' | 'alpha' | 'speed' | 'align' | 'repair' | 'shield_repair' | 'armor_repair' | 'hull_repair' | 'passive_shield' | 'npc_ehp'
type FitSort = 'observed' | 'recent' | 'cheapest' | 'expensive' | 'ehp' | 'dps' | 'alpha' | 'repair' | 'speed' | 'align'
interface FitFilter { metric: FitFilterMetric; min: number | null; max: number | null; npcProfile?: string }
const fitFilterOptions: Array<{ value: FitFilterMetric; label: string }> = [
    { value: 'ehp', label: 'Effective hitpoints' }, { value: 'dps', label: 'Damage / second' },
    { value: 'alpha', label: 'Alpha damage' }, { value: 'speed', label: 'Maximum velocity' },
    { value: 'align', label: 'Align time' }, { value: 'repair', label: 'Strongest local repair' }, { value: 'shield_repair', label: 'Active shield repair' },
    { value: 'armor_repair', label: 'Active armor repair' }, { value: 'hull_repair', label: 'Active hull repair' },
    { value: 'passive_shield', label: 'Passive shield regeneration' }, { value: 'npc_ehp', label: 'EHP against NPC faction' },
]
const npcProfiles = [
    ['omni', 'Omnidamage'], ['angels', 'Angel Cartel'], ['blood-raiders', 'Blood Raiders'],
    ['guristas', 'Guristas Pirates'], ['mordus', "Mordu's Legion"], ['sansha', "Sansha's Nation"],
    ['serpentis', 'Serpentis'], ['triglavian', 'Triglavian Collective'], ['amarr', 'Amarr Empire'],
    ['caldari', 'Caldari State'], ['gallente', 'Gallente Federation'], ['minmatar', 'Minmatar Republic'],
] as const
const filtersOpen = ref(false)
const metricDropdownOpen = ref(false)
const npcDropdownOpen = ref(false)
const sortDropdownOpen = ref(false)
const activeFilters = ref<FitFilter[]>([])
const moduleGroupFilters = ref<number[]>([])
const draftMetric = ref<FitFilterMetric>('ehp')
const draftMin = ref('')
const draftMax = ref('')
const draftNPCProfile = ref('guristas')
const fitSort = ref<FitSort>('observed')
const fitSortOptions: Array<{ value: FitSort; label: string }> = [
    { value: 'observed', label: 'Most observed' }, { value: 'recent', label: 'Recently seen' },
    { value: 'cheapest', label: 'Cheapest' }, { value: 'expensive', label: 'Most expensive' },
    { value: 'ehp', label: 'Highest EHP' }, { value: 'dps', label: 'Highest DPS' },
    { value: 'alpha', label: 'Highest alpha' }, { value: 'repair', label: 'Highest repair' },
    { value: 'speed', label: 'Highest speed' }, { value: 'align', label: 'Fastest align' },
]
const selectedSortLabel = computed(() => fitSortOptions.find(option => option.value === fitSort.value)?.label ?? 'Most observed')
const fitFilterMetrics = new Set<FitFilterMetric>(fitFilterOptions.map(option => option.value))
const npcProfileIDs = new Set<string>(npcProfiles.map(profile => profile[0]))
function queryValue(value: string | null | Array<string | null> | undefined): string | undefined {
    return Array.isArray(value) ? value.find(item => item !== null) ?? undefined : value ?? undefined
}
function queryNumber(value: string | null | Array<string | null> | undefined): number | null {
    const parsed = Number(queryValue(value))
    return Number.isFinite(parsed) ? parsed : null
}
function loadFiltersFromRoute() {
    const filters: FitFilter[] = []
    for (const metric of fitFilterMetrics) {
        const min = queryNumber(route.query[`min_${metric}`])
        const max = queryNumber(route.query[`max_${metric}`])
        if (min === null && max === null) continue
        const requestedProfile = queryValue(route.query.npc_profile)
        const npcProfile = metric === 'npc_ehp' && requestedProfile && npcProfileIDs.has(requestedProfile)
            ? requestedProfile
            : metric === 'npc_ehp' ? 'guristas' : undefined
        filters.push({ metric, min, max, npcProfile })
    }
    activeFilters.value = filters
    const groups = (queryValue(route.query.groups) ?? '').split(',')
        .map(Number)
        .filter((groupID): groupID is number => Number.isInteger(groupID) && groupID > 0)
    moduleGroupFilters.value = [...new Set(groups)]
    const requestedSort = queryValue(route.query.sort) as FitSort | undefined
    if (requestedSort && fitSortOptions.some(option => option.value === requestedSort)) fitSort.value = requestedSort
    filtersOpen.value = filters.length > 0 || groups.length > 0
}
loadFiltersFromRoute()
function parseFitFilterValue(raw: string): number | null {
    const cleaned = raw.trim().toLowerCase().replaceAll(',', '')
    if (!cleaned) return null
    const match = cleaned.match(/^([0-9]*\.?[0-9]+)\s*([kmb])?$/)
    if (!match) return null
    const multipliers: Record<string, number> = { k: 1_000, m: 1_000_000, b: 1_000_000_000 }
    return Number(match[1]) * (match[2] ? multipliers[match[2]]! : 1)
}
function addFitFilter() {
    const min = parseFitFilterValue(draftMin.value)
    const max = parseFitFilterValue(draftMax.value)
    if (min === null && max === null) return
    activeFilters.value = activeFilters.value.filter(item => item.metric !== draftMetric.value)
    activeFilters.value.push({ metric: draftMetric.value, min, max, npcProfile: draftMetric.value === 'npc_ehp' ? draftNPCProfile.value : undefined })
    draftMin.value = ''; draftMax.value = ''
}
function setFitMetricFilter(metric: FitFilterMetric, min: number | null, max: number | null, npcProfile?: string) {
    activeFilters.value = activeFilters.value.filter(item => item.metric !== metric)
    activeFilters.value.push({ metric, min, max, npcProfile })
}
function applyTrendBucket(metric: DistributionMetric, bucket: DistributionBucket) {
    const mapped = ({ ehp: 'ehp', dps: 'dps', alpha: 'alpha', repair: 'repair', speed: 'speed', align: 'align' } as Partial<Record<string, FitFilterMetric>>)[metric.metric]
    if (mapped) setFitMetricFilter(mapped, bucket.lower_bound, bucket.upper_bound)
}
function removeFitFilter(index: number) { activeFilters.value.splice(index, 1) }
function clearFitFilters() {
    activeFilters.value = []
    moduleGroupFilters.value = []
}
function toggleModuleGroup(groupID: number) {
    moduleGroupFilters.value = moduleGroupFilters.value.includes(groupID)
        ? moduleGroupFilters.value.filter(id => id !== groupID)
        : [...moduleGroupFilters.value, groupID]
}
function filterLabel(filter: FitFilter): string {
    const metric = fitFilterOptions.find(option => option.value === filter.metric)?.label ?? filter.metric
    const profile = filter.metric === 'npc_ehp' ? ` · ${npcProfiles.find(item => item[0] === filter.npcProfile)?.[1] ?? filter.npcProfile}` : ''
    const range = filter.min != null && filter.max != null ? `${filter.min.toLocaleString()}–${filter.max.toLocaleString()}` : filter.min != null ? `≥ ${filter.min.toLocaleString()}` : `≤ ${filter.max?.toLocaleString()}`
    return `${metric}${profile} ${range}`
}
function activeFilterParams(): URLSearchParams {
    const params = new URLSearchParams()
    for (const filter of activeFilters.value) {
        if (filter.min != null) params.set(`min_${filter.metric}`, String(filter.min))
        if (filter.max != null) params.set(`max_${filter.metric}`, String(filter.max))
        if (filter.metric === 'npc_ehp' && filter.npcProfile) params.set('npc_profile', filter.npcProfile)
    }
    if (moduleGroupFilters.value.length) params.set('groups', moduleGroupFilters.value.join(','))
    if (fitSort.value !== 'observed') params.set('sort', fitSort.value)
    return params
}
const managedFilterQueryKeys = new Set([
    ...fitFilterOptions.flatMap(option => [`min_${option.value}`, `max_${option.value}`]),
    'npc_profile', 'groups', 'sort',
])
watch([activeFilters, moduleGroupFilters, fitSort], async () => {
    const query = { ...route.query }
    for (const key of managedFilterQueryKeys) delete query[key]
    for (const [key, value] of activeFilterParams()) query[key] = value
    await router.replace({ query })
}, { deep: true })
const fittingsURL = computed(() => {
    const params = activeFilterParams()
    params.delete('sort')
    const query = params.toString()
    return `/api/item/${props.shipTypeId}/fittings${query ? `?${query}` : ''}`
})

const { data, pending, error } = await useApiFetch<FittingsResponse>(
    fittingsURL,
)

// Module group popularity — "97% use prop mods, 82% webs" etc.
interface FitMetaGroup {
    group_id: number
    name: string
    kill_count: number
    pct: number
}
interface FitMetaResponse {
    ship_type_id: number
    window_days: number
    total_kills: number
    groups: FitMetaGroup[]
}
const { data: metaData } = useApiFetch<FitMetaResponse>(
    computed(() => {
        const params = activeFilterParams()
        params.delete('sort')
        const query = params.toString()
        return `/api/item/${props.shipTypeId}/fit-meta${query ? `?${query}` : ''}`
    }),
    { lazy: true, server: false },
)
const moduleGroupName = (groupID: number) => metaData.value?.groups.find(group => group.group_id === groupID)?.name ?? `Module group ${groupID}`

interface DistributionBucket {
    bucket: number
    lower_bound: number
    upper_bound: number
    fit_count: number
    observation_count: number
}
interface DistributionMetric {
    metric: string
    fit_count: number
    observation_count: number
    minimum: number
    maximum: number
    p10: number
    p25: number
    median: number
    p75: number
    p90: number
    buckets: DistributionBucket[]
}
interface DistributionResponse {
    ship_type_id: number
    window_days: number
    metrics: DistributionMetric[]
}
const distributionURL = computed(() => {
    const params = activeFilterParams(); params.delete('sort'); params.set('days', '90')
    return `/api/item/${props.shipTypeId}/fit-distributions?${params.toString()}`
})
const { data: distributionData } = useApiFetch<DistributionResponse>(
    distributionURL,
    { lazy: true, server: false },
)
const visibleDistributions = computed(() => {
    const wanted = ['ehp', 'dps', 'repair', 'speed']
    return wanted.map(metric => distributionData.value?.metrics.find(row => row.metric === metric)).filter((row): row is DistributionMetric => Boolean(row))
})
const distributionLabels: Record<string, string> = { ehp: 'Effective Hitpoints', dps: 'Damage / second', repair: 'Effective repair / second', speed: 'Maximum velocity' }
function formatDistributionValue(metric: string, value: number): string {
    if (metric === 'speed') return `${Math.round(value).toLocaleString('en-US')} m/s`
    if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}m`
    if (value >= 1_000) return `${(value / 1_000).toFixed(value >= 100_000 ? 0 : 1)}k`
    return value.toFixed(value >= 100 ? 0 : 1)
}
function bucketHeight(metric: DistributionMetric, bucket: DistributionBucket): number {
    if (bucket.observation_count === 0) return 0
    const maximum = Math.max(...metric.buckets.map(item => item.observation_count), 1)
    return Math.max(3, (bucket.observation_count / maximum) * 100)
}
function fitStatParts(stats: FittingFamily['stats']): string[] {
    if (!stats) return []
    const parts: string[] = []
    if (stats.ehp != null) parts.push(`${formatDistributionValue('ehp', stats.ehp)} EHP`)
    if (stats.dps != null) parts.push(`${formatDistributionValue('dps', stats.dps)} DPS`)
    if (stats.repair != null && stats.repair > 0) parts.push(`${formatDistributionValue('repair', stats.repair)} EHP/s`)
    if (stats.speed != null) parts.push(`${formatDistributionValue('speed', stats.speed)}`)
    if (stats.npc_ehp != null && stats.npc_profile) parts.push(`${formatDistributionValue('ehp', stats.npc_ehp)} vs ${npcProfiles.find(item => item[0] === stats.npc_profile)?.[1] ?? stats.npc_profile}`)
    return parts
}

const expandedFamily = ref<string | null>(null)

const sortedFamilies = computed(() => {
    const families = [...(data.value?.families ?? [])]
    switch (fitSort.value) {
        case 'recent':
            return families.sort((a, b) => new Date(b.last_used).getTime() - new Date(a.last_used).getTime())
        case 'cheapest':
            return families.sort((a, b) => a.fit_cost - b.fit_cost)
        case 'expensive':
            return families.sort((a, b) => b.fit_cost - a.fit_cost)
        case 'ehp': case 'dps': case 'alpha': case 'repair': case 'speed': {
            const metric = fitSort.value
            return families.sort((a, b) => (b.stats?.[metric] ?? -1) - (a.stats?.[metric] ?? -1))
        }
        case 'align':
            return families.sort((a, b) => (a.stats?.align ?? Number.POSITIVE_INFINITY) - (b.stats?.align ?? Number.POSITIVE_INFINITY))
        default:
            return families.sort((a, b) => b.total_uses - a.total_uses)
    }
})

const toggleFamily = (familyHash: string) => {
    expandedFamily.value = expandedFamily.value === familyHash ? null : familyHash
}

const lastObserved = (iso: string): string => {
    const days = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000))
    if (days === 0) return 'today'
    if (days === 1) return 'yesterday'
    return `${days}d ago`
}

// Slot group metadata — matches the extractor's numbering.
const SLOT_LABELS: Record<number, string> = {
    1: 'High',
    2: 'Med',
    3: 'Low',
    4: 'Rig',
    5: 'Subsystem',
}
const SLOT_ORDER = [1, 2, 3, 4, 5] as const

function groupBySlot(modules: FittingModule[]): Record<number, FittingModule[]> {
    const result: Record<number, FittingModule[]> = {}
    for (const m of modules) {
        if (!result[m.slot_group]) result[m.slot_group] = []
        result[m.slot_group]!.push(m)
    }
    return result
}

/**
 * Convert the killmail-derived family into a UI Fit, encode it, and
 * navigate to the editor. Heavy lifting lives in killmailFitToEditorUrl
 * so this function only adds the editor-friendly description.
 */
async function loadIntoEditor(family: FittingFamily) {
    const url = await killmailFitToEditorUrl({
        shipTypeId: props.shipTypeId,
        modules: family.modules,
        drones: family.drones,
        name: 'Community Fit',
        description: `Loaded from /fits — ${family.total_uses} recorded use${family.total_uses === 1 ? '' : 's'} in the last ${data.value?.window_days ?? 90} days.`,
    })
    await navigateTo(url)
}
</script>

<template>
    <div>
        <div v-if="pending" class="py-12 text-center text-gray-500 text-sm">
            Loading fits…
        </div>

        <div v-else-if="error" class="py-12 text-center text-red-400 text-sm">
            Failed to load fittings.
        </div>

        <div v-else-if="data">
            <!-- Rare hull callout -->
            <div v-if="data.is_rare_hull"
                class="mb-4 rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-xs text-amber-300 flex items-start gap-2">
                <Icon name="lucide:sparkles" class="w-4 h-4 flex-shrink-0 mt-0.5" />
                <div>
                    <div class="font-semibold mb-0.5">Rare hull</div>
                    <div class="text-amber-200/80">
                        Sample size is small, so we're showing every fit we've seen — even the one-offs.
                    </div>
                </div>
            </div>

            <!-- Module meta breakdown -->
            <div v-if="metaData && metaData.groups.length > 0" class="mb-4">
                <div class="mb-2 flex items-baseline justify-between">
                    <span class="text-xs font-semibold uppercase tracking-wide text-gray-400">Module Usage</span>
                    <span class="text-fine text-gray-600">Click to require a module group · {{ metaData.total_kills.toLocaleString('en-US') }} losses</span>
                </div>
                <div class="flex flex-wrap gap-1.5">
                    <button v-for="g in metaData.groups" :key="g.group_id" type="button"
                        class="group inline-flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-xs transition-colors"
                        :class="moduleGroupFilters.includes(g.group_id) ? 'border-blue-400/35 bg-blue-500/15 text-blue-200' : 'border-white/[0.07] bg-white/[0.02] text-gray-400 hover:border-blue-400/25 hover:bg-blue-500/[0.07] hover:text-gray-200'"
                        @click="toggleModuleGroup(g.group_id)">
                        <span class="max-w-40 truncate">{{ g.name }}</span>
                        <span class="text-fine tabular-nums" :class="moduleGroupFilters.includes(g.group_id) ? 'text-blue-300' : 'text-gray-600'">{{ g.pct }}%</span>
                    </button>
                </div>
            </div>

            <section v-if="visibleDistributions.length" class="mb-5">
                <div class="mb-2 flex items-end justify-between gap-3">
                    <div>
                        <div class="text-xs font-semibold uppercase tracking-wide text-gray-400">Observed Fit Profiles</div>
                        <div class="mt-0.5 text-fine text-gray-600">Distribution of calculated all-V statistics across recorded losses</div>
                    </div>
                    <span class="text-fine text-gray-600">{{ distributionData?.window_days ?? 90 }} days</span>
                </div>
                <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
                    <div v-for="metric in visibleDistributions" :key="metric.metric"
                        class="rounded-lg border border-white/[0.08] bg-gradient-to-br from-blue-500/[0.045] to-transparent p-3">
                        <div class="flex items-start justify-between gap-2">
                            <div>
                                <div class="text-xs font-semibold text-gray-300">{{ distributionLabels[metric.metric] }}</div>
                                <div class="mt-1 text-lg font-bold tabular-nums text-white">{{ formatDistributionValue(metric.metric, metric.median) }}</div>
                                <div class="text-fine text-gray-600">median · {{ metric.observation_count.toLocaleString('en-US') }} losses</div>
                            </div>
                            <div class="rounded border border-blue-400/15 bg-blue-400/[0.06] px-1.5 py-1 text-right text-fine tabular-nums text-blue-200/80">
                                <div>P10 {{ formatDistributionValue(metric.metric, metric.p10) }}</div>
                                <div>P90 {{ formatDistributionValue(metric.metric, metric.p90) }}</div>
                            </div>
                        </div>
                        <div class="mt-3 flex h-14 items-end gap-px border-b border-white/[0.08]" :aria-label="`${distributionLabels[metric.metric]} histogram`">
                            <button v-for="bucket in metric.buckets" :key="bucket.bucket" type="button"
                                class="group relative min-w-0 flex-1 rounded-t-[1px] bg-blue-400/45 transition-all hover:bg-blue-300/90 hover:brightness-125"
                                :style="{ height: `${bucketHeight(metric, bucket)}%` }" @click="applyTrendBucket(metric, bucket)">
                                <div class="pointer-events-none absolute bottom-full left-1/2 z-10 mb-1 hidden -translate-x-1/2 whitespace-nowrap rounded border border-white/10 bg-gray-950 px-2 py-1 text-fine text-gray-300 shadow-xl group-hover:block">
                                    {{ formatDistributionValue(metric.metric, bucket.lower_bound) }}–{{ formatDistributionValue(metric.metric, bucket.upper_bound) }} · {{ bucket.observation_count.toLocaleString('en-US') }}
                                </div>
                            </button>
                        </div>
                        <div class="mt-1 flex justify-between text-fine tabular-nums text-gray-700">
                            <span>{{ formatDistributionValue(metric.metric, metric.p10) }}</span>
                            <span>{{ formatDistributionValue(metric.metric, metric.p90) }}</span>
                        </div>
                    </div>
                </div>
            </section>

            <div class="mb-3 rounded-lg border border-white/[0.07] bg-white/[0.018]">
                <div class="flex flex-wrap items-center gap-2 px-3 py-2">
                    <button type="button" class="inline-flex items-center gap-1.5 rounded-md border border-blue-500/25 bg-blue-500/[0.08] px-2.5 py-1.5 text-xs font-medium text-blue-300 hover:bg-blue-500/[0.14]"
                        @click="filtersOpen = !filtersOpen">
                        <Icon name="lucide:list-filter" class="h-3.5 w-3.5" />
                        Filter fits
                        <span v-if="activeFilters.length || moduleGroupFilters.length" class="rounded bg-blue-400/15 px-1.5 text-fine">{{ activeFilters.length + moduleGroupFilters.length }}</span>
                    </button>
                    <button v-for="(filter, index) in activeFilters" :key="`${filter.metric}-${index}`" type="button"
                        class="inline-flex items-center gap-1.5 rounded-full border border-white/[0.08] bg-black/25 px-2.5 py-1 text-fine text-gray-300 hover:border-red-400/25 hover:text-red-300"
                        :title="`Remove ${filterLabel(filter)} filter`" @click="removeFitFilter(index)">
                        {{ filterLabel(filter) }} <Icon name="lucide:x" class="h-3 w-3" />
                    </button>
                    <button v-for="groupID in moduleGroupFilters" :key="`group-${groupID}`" type="button"
                        class="inline-flex items-center gap-1.5 rounded-full border border-white/[0.08] bg-black/25 px-2.5 py-1 text-fine text-gray-300 hover:border-red-400/25 hover:text-red-300"
                        :title="`Remove required ${moduleGroupName(groupID)} module group`" @click="toggleModuleGroup(groupID)">
                        Requires {{ moduleGroupName(groupID) }} <Icon name="lucide:x" class="h-3 w-3" />
                    </button>
                    <span v-if="!activeFilters.length && !moduleGroupFilters.length" class="text-fine text-gray-600">EHP, damage, repair, mobility, or NPC-specific tank</span>
                    <button v-if="activeFilters.length || moduleGroupFilters.length" type="button"
                        class="ml-auto inline-flex items-center gap-1 text-fine text-gray-500 transition-colors hover:text-red-300"
                        @click="clearFitFilters">
                        <Icon name="lucide:rotate-ccw" class="h-3 w-3" /> Clear all
                    </button>
                </div>
                <form v-if="filtersOpen" class="grid gap-2 border-t border-white/[0.06] px-3 py-3 sm:grid-cols-2 lg:grid-cols-[minmax(170px,1.5fr)_minmax(150px,1.2fr)_1fr_1fr_auto]"
                    @submit.prevent="addFitFilter">
                    <label>
                        <span class="mb-1 block text-fine uppercase tracking-wide text-gray-600">Statistic</span>
                        <Dropdown v-model="metricDropdownOpen" align="left" class="w-full">
                            <template #trigger>
                                <button type="button" class="flex w-full items-center justify-between rounded-md border border-white/[0.08] bg-black/35 px-2.5 py-2 text-left text-xs text-gray-300 hover:border-blue-500/30">
                                    {{ fitFilterOptions.find(option => option.value === draftMetric)?.label }}
                                    <Icon name="lucide:chevron-down" class="h-3.5 w-3.5 text-gray-600" />
                                </button>
                            </template>
                            <template #default="{ close }">
                                <button v-for="option in fitFilterOptions" :key="option.value" type="button"
                                    class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-xs transition-colors hover:bg-blue-500/[0.08] hover:text-blue-300"
                                    :class="draftMetric === option.value ? 'text-blue-300' : 'text-gray-400'"
                                    @click="draftMetric = option.value; close()">
                                    <Icon name="lucide:check" class="h-3.5 w-3.5" :class="draftMetric === option.value ? 'opacity-100' : 'opacity-0'" />
                                    {{ option.label }}
                                </button>
                            </template>
                        </Dropdown>
                    </label>
                    <label v-if="draftMetric === 'npc_ehp'">
                        <span class="mb-1 block text-fine uppercase tracking-wide text-gray-600">NPC damage</span>
                        <Dropdown v-model="npcDropdownOpen" align="left" class="w-full">
                            <template #trigger>
                                <button type="button" class="flex w-full items-center justify-between rounded-md border border-white/[0.08] bg-black/35 px-2.5 py-2 text-left text-xs text-gray-300 hover:border-blue-500/30">
                                    {{ npcProfiles.find(profile => profile[0] === draftNPCProfile)?.[1] }}
                                    <Icon name="lucide:chevron-down" class="h-3.5 w-3.5 text-gray-600" />
                                </button>
                            </template>
                            <template #default="{ close }">
                                <button v-for="profile in npcProfiles" :key="profile[0]" type="button"
                                    class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-xs transition-colors hover:bg-blue-500/[0.08] hover:text-blue-300"
                                    :class="draftNPCProfile === profile[0] ? 'text-blue-300' : 'text-gray-400'"
                                    @click="draftNPCProfile = profile[0]; close()">
                                    <Icon name="lucide:check" class="h-3.5 w-3.5" :class="draftNPCProfile === profile[0] ? 'opacity-100' : 'opacity-0'" />
                                    {{ profile[1] }}
                                </button>
                            </template>
                        </Dropdown>
                    </label>
                    <div v-else class="hidden lg:block" />
                    <label>
                        <span class="mb-1 block text-fine uppercase tracking-wide text-gray-600">Minimum</span>
                        <input v-model="draftMin" inputmode="decimal" placeholder="e.g. 50k" class="w-full rounded-md border border-white/[0.08] bg-black/35 px-2.5 py-2 text-xs tabular-nums text-gray-300 outline-none placeholder:text-gray-700 focus:border-blue-500/40">
                    </label>
                    <label>
                        <span class="mb-1 block text-fine uppercase tracking-wide text-gray-600">Maximum</span>
                        <input v-model="draftMax" inputmode="decimal" placeholder="optional" class="w-full rounded-md border border-white/[0.08] bg-black/35 px-2.5 py-2 text-xs tabular-nums text-gray-300 outline-none placeholder:text-gray-700 focus:border-blue-500/40">
                    </label>
                    <button type="submit" class="self-end rounded-md border border-blue-500/30 bg-blue-500/15 px-3 py-2 text-xs font-semibold text-blue-300 hover:bg-blue-500/25">Add filter</button>
                </form>
            </div>

            <div v-if="data.families.length === 0" class="rounded-lg border border-white/[0.07] bg-white/[0.015] py-12 text-center">
                <Icon name="lucide:wrench" class="mx-auto mb-3 h-10 w-10 text-gray-700" />
                <p class="text-sm text-gray-500">
                    {{ activeFilters.length || moduleGroupFilters.length
                        ? 'No observed fits match these filters.'
                        : `No fits recorded for this ship in the last ${data.window_days} days.` }}
                </p>
                <p class="mt-2 text-xs text-gray-600">
                    {{ activeFilters.length || moduleGroupFilters.length
                        ? 'Remove a filter or clear everything to see the full set again.'
                        : "We extract fits from killmails — if a ship isn't losing any, we have nothing to learn from." }}
                </p>
                <button v-if="activeFilters.length || moduleGroupFilters.length" type="button"
                    class="mt-4 inline-flex items-center gap-1.5 rounded-md border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-xs font-semibold text-blue-300 hover:bg-blue-500/20"
                    @click="clearFitFilters">
                    <Icon name="lucide:rotate-ccw" class="h-3.5 w-3.5" /> Reset filters
                </button>
            </div>

            <!-- Header row -->
            <div v-if="data.families.length > 0" class="flex items-center justify-between gap-3 mb-3">
                <h2 class="text-sm font-semibold text-gray-300">
                    Popular Fits
                    <span class="text-gray-600 font-normal ml-2">(last {{ data.window_days }} days)</span>
                </h2>
                <div class="flex items-center gap-2">
                    <span class="text-xs text-gray-600 hidden sm:inline">
                        {{ data.families.length }} {{ data.families.length === 1 ? 'family' : 'families' }}
                    </span>
                    <Dropdown v-model="sortDropdownOpen" align="right">
                        <template #trigger>
                            <button type="button" class="inline-flex items-center gap-2 rounded-md border border-white/[0.08] bg-black/30 px-3 py-1.5 text-xs text-gray-300 transition-colors hover:border-blue-500/30">
                                <span class="text-gray-600">Sort by</span> {{ selectedSortLabel }}
                                <Icon name="lucide:chevron-down" class="h-3.5 w-3.5 text-gray-600" />
                            </button>
                        </template>
                        <template #default="{ close }">
                            <button v-for="option in fitSortOptions" :key="option.value" type="button"
                                class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-xs transition-colors hover:bg-blue-500/[0.08] hover:text-blue-300"
                                :class="fitSort === option.value ? 'text-blue-300' : 'text-gray-400'"
                                @click="fitSort = option.value; close()">
                                <Icon name="lucide:check" class="h-3.5 w-3.5" :class="fitSort === option.value ? 'opacity-100' : 'opacity-0'" />
                                {{ option.label }}
                            </button>
                        </template>
                    </Dropdown>
                </div>
            </div>

            <!-- Fit cards -->
            <div v-if="data.families.length > 0" class="space-y-3">
                <div v-for="(family, index) in sortedFamilies" :key="family.family_hash"
                    class="rounded-lg border bg-white/[0.025] overflow-hidden transition-colors"
                    :class="expandedFamily === family.family_hash ? 'border-blue-500/25' : 'border-white/[0.08] hover:border-white/[0.14]'">
                    <!-- Family header -->
                    <div class="flex items-stretch border-b border-white/[0.06] bg-white/[0.02]">
                        <button type="button"
                            class="flex min-w-0 flex-1 items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-blue-500/[0.035] sm:px-4"
                            :aria-expanded="expandedFamily === family.family_hash"
                            @click="toggleFamily(family.family_hash)">
                            <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-white/[0.07] bg-black/20 text-fine font-bold tabular-nums text-gray-600">
                                {{ String(index + 1).padStart(2, '0') }}
                            </span>
                            <Icon name="lucide:chevron-right"
                                class="h-4 w-4 shrink-0 text-gray-600 transition-transform duration-200"
                                :class="expandedFamily === family.family_hash ? 'rotate-90 text-blue-400' : ''" />
                            <div class="min-w-0 flex-1">
                                <div class="truncate text-sm font-semibold text-gray-200">
                                    {{ classifyFitFamily(family.modules, family.drones, { hullGroupName: props.hullGroupName }) }}
                                </div>
                                <div class="mt-0.5 flex flex-wrap items-center gap-x-5 gap-y-1">
                                    <div>
                                        <span class="text-base font-bold text-white tabular-nums">{{ family.total_uses }}</span>
                                        <span class="ml-1.5 text-xs text-gray-500">{{ family.total_uses === 1 ? 'loss' : 'losses' }}</span>
                                    </div>
                                    <div class="text-xs text-gray-500">
                                        <span class="font-medium text-gray-300 tabular-nums">{{ family.variant_count }}</span>
                                        variant{{ family.variant_count === 1 ? '' : 's' }}
                                    </div>
                                    <div class="hidden text-xs text-gray-500 md:block">
                                        Seen <span class="text-gray-300">{{ lastObserved(family.last_used) }}</span>
                                    </div>
                                    <div v-if="family.fit_cost > 0" class="text-xs text-gray-500 tabular-nums">
                                        <span class="font-semibold text-yellow-400">{{ formatIsk(family.fit_cost + (data?.hull_cost ?? 0)) }}</span>
                                        ISK total
                                    </div>
                                </div>
                                <div v-if="fitFamilyContextParts(family.context).length"
                                    class="mt-0.5 w-full truncate text-fine text-gray-600"
                                    :title="fitFamilyContextParts(family.context).join(' · ')">
                                    {{ fitFamilyContextParts(family.context).join(' · ') }}
                                </div>
                                <div v-if="fitStatParts(family.stats).length"
                                    class="mt-1 flex flex-wrap gap-x-3 text-fine font-medium tabular-nums text-blue-300/65">
                                    <span v-for="part in fitStatParts(family.stats)" :key="part">{{ part }}</span>
                                </div>
                            </div>
                        </button>
                        <button type="button"
                            class="m-2 ml-0 inline-flex shrink-0 items-center gap-1.5 rounded-md border border-blue-500/30 bg-blue-500/15 px-2.5 text-fine font-bold uppercase tracking-[0.1em] text-blue-400 transition-colors hover:bg-blue-500/25 sm:px-3"
                            @click="loadIntoEditor(family)">
                            <Icon name="lucide:square-pen" class="w-3.5 h-3.5" />
                            <span class="hidden sm:inline">Load in Editor</span>
                        </button>
                    </div>

                    <!-- Slot grid -->
                    <div v-if="expandedFamily === family.family_hash"
                        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-0 divide-y md:divide-y-0 md:divide-x divide-white/[0.06]">
                        <div v-for="slot in SLOT_ORDER" :key="slot" class="p-3">
                            <div class="text-fine uppercase tracking-wider text-gray-600 mb-2 flex items-center gap-1.5">
                                {{ SLOT_LABELS[slot] }}
                                <span v-if="groupBySlot(family.modules)[slot]?.length"
                                    class="text-gray-700">·&nbsp;{{ groupBySlot(family.modules)[slot]!.length }}</span>
                            </div>
                            <ul v-if="groupBySlot(family.modules)[slot]?.length" class="space-y-1">
                                <li v-for="mod in groupBySlot(family.modules)[slot]"
                                    :key="`${slot}-${mod.ordinal}`"
                                    class="text-xs">
                                    <div class="flex items-center gap-2">
                                        <img :src="`/images/types/${mod.type_id}/icon?size=32`"
                                            :alt="mod.name ?? ''"
                                            class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                                        <NuxtLink :to="`/item/${mod.type_id}`"
                                            class="text-gray-300 hover:text-blue-400 transition-colors truncate">
                                            {{ mod.name ?? `Type ${mod.type_id}` }}
                                        </NuxtLink>
                                    </div>
                                    <!-- Charge / loaded ammo, when paired by the extractor.
                                         Indented under the parent module. -->
                                    <div v-if="mod.charge_type_id"
                                        class="flex items-center gap-2 mt-0.5 ml-7 text-fine text-gray-500">
                                        <img :src="`/images/types/${mod.charge_type_id}/icon?size=32`"
                                            :alt="mod.charge_name ?? ''"
                                            class="w-3.5 h-3.5 rounded flex-shrink-0" loading="lazy">
                                        <NuxtLink :to="`/item/${mod.charge_type_id}`"
                                            class="hover:text-blue-400 transition-colors truncate">
                                            {{ mod.charge_name ?? `Type ${mod.charge_type_id}` }}
                                        </NuxtLink>
                                    </div>
                                </li>
                            </ul>
                            <div v-else class="text-fine text-gray-700 italic">empty</div>
                        </div>
                    </div>

                    <!-- Drone bay row — only shown when the canonical fit had drones. -->
                    <div v-if="expandedFamily === family.family_hash && family.drones.length > 0"
                        class="flex items-center gap-3 flex-wrap px-4 py-2 border-t border-white/[0.06] bg-white/[0.01]">
                        <div class="flex items-center gap-1.5 flex-shrink-0">
                            <Icon name="lucide:radar" class="w-3.5 h-3.5 text-blue-400/70" />
                            <span class="text-fine uppercase tracking-wider text-gray-600">Drone bay</span>
                        </div>
                        <div v-for="d in family.drones" :key="`d-${d.type_id}`"
                            class="flex items-center gap-1.5 px-2 py-0.5 rounded text-xs bg-white/[0.04]">
                            <img :src="`/images/types/${d.type_id}/icon?size=32`"
                                :alt="d.name ?? ''"
                                class="w-4 h-4 rounded flex-shrink-0" loading="lazy">
                            <NuxtLink :to="`/item/${d.type_id}`"
                                class="text-gray-300 hover:text-blue-400 transition-colors truncate">
                                {{ d.name ?? `Type ${d.type_id}` }}
                            </NuxtLink>
                            <span class="text-gray-500 tabular-nums">×{{ d.quantity }}</span>
                        </div>
                    </div>

                    <!-- Alliance usage footer -->
                    <div v-if="expandedFamily === family.family_hash && family.top_alliances.length > 0"
                        class="flex items-center gap-2 flex-wrap px-4 py-2 border-t border-white/[0.06] bg-white/[0.01]">
                        <Icon name="lucide:users" class="w-3.5 h-3.5 text-gray-600" />
                        <span class="text-fine uppercase tracking-wider text-gray-600 mr-1">Flown by</span>
                        <NuxtLink v-for="alliance in family.top_alliances" :key="alliance.alliance_id"
                            :to="`/alliance/${alliance.alliance_id}`"
                            class="flex items-center gap-1.5 px-2 py-0.5 rounded text-xs bg-white/[0.04] hover:bg-white/[0.08] transition-colors">
                            <img :src="`/images/alliances/${alliance.alliance_id}/logo?size=32`"
                                :alt="alliance.name ?? ''" class="w-4 h-4 rounded" loading="lazy">
                            <span class="text-gray-300">{{ alliance.name ?? `Alliance ${alliance.alliance_id}` }}</span>
                            <span class="text-gray-500 tabular-nums">{{ alliance.pct_of_alliance_losses }}%</span>
                        </NuxtLink>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
