<script setup lang="ts">
const { isDomainMode } = useDomainConfig()

useHead({ title: 'Battles' })
useSeoMeta({
    description: 'Browse major battles in EVE Online — fleet engagements, battle reports, timelines, and ISK destroyed across New Eden on EVE-KILL.',
    ogTitle: 'Battles — EVE-KILL',
    ogDescription: 'Major EVE Online battles — fleet engagements, battle reports, and ISK destroyed.',
})

// Filters — initialized from URL query so back-button navigation restores them
const route = useRoute()
const router = useRouter()

const qNum = (v: unknown): number | null => {
    const n = Number(v)
    return Number.isFinite(n) && n > 0 ? n : null
}
const qStr = (v: unknown): string | null => (typeof v === 'string' && v ? v : null)

const page = ref(qNum(route.query.page) ?? 1)
const selectedYear = ref<number | null>(qNum(route.query.year))
const minKills = ref<number | null>(qNum(route.query.minKills))
const minIsk = ref<string | null>(qStr(route.query.minIsk))
const showCustom = ref(route.query.custom === '1')

// Entity search filter — only type+id live in the URL; name is resolved server-side
const entityFilter = ref<{ type: string; id: number; name: string } | null>(
    qStr(route.query.eType) && qNum(route.query.eId)
        ? { type: String(route.query.eType), id: Number(route.query.eId), name: '' }
        : null,
)

// Resolve the chip name when the filter is rehydrated from the URL.
if (entityFilter.value && !entityFilter.value.name) {
    const { data: resolved } = await useApiFetch<{ name: string }>('/api/entity/resolve', {
        params: { type: entityFilter.value.type, id: entityFilter.value.id },
    })
    if (resolved.value?.name && entityFilter.value) {
        entityFilter.value.name = resolved.value.name
    } else {
        // ID points to nothing — drop the filter so we don't fetch with a stale id
        entityFilter.value = null
    }
}

const searchQuery = ref('')
const searchResults = ref<any[]>([])
const searchOpen = ref(false)
const searchLoading = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null
const searchInputRef = ref<HTMLInputElement | null>(null)

const onSearchInput = (e: Event) => {
    const q = (e.target as HTMLInputElement).value
    searchQuery.value = q
    if (searchTimer) clearTimeout(searchTimer)

    if (q.length < 3) {
        searchResults.value = []
        searchOpen.value = false
        return
    }

    searchTimer = setTimeout(async () => {
        searchLoading.value = true
        try {
            const res = await apiFetch<any>('/api/search', {
                params: { q, limit: 10 },
            })
            // Filter to alliance, corporation, system, region only
            searchResults.value = (res?.hits || []).filter((h: any) =>
                ['alliance', 'corporation', 'system', 'region'].includes(h.type)
            )
            searchOpen.value = searchResults.value.length > 0
        } catch {
            searchResults.value = []
            searchOpen.value = false
        }
        searchLoading.value = false
    }, 300)
}

const selectEntity = (hit: any) => {
    const numericId = Number(hit.id.split('_').pop())
    entityFilter.value = { type: hit.type, id: numericId, name: hit.name }
    searchQuery.value = ''
    searchResults.value = []
    searchOpen.value = false
    page.value = 1
}

const clearEntity = () => {
    entityFilter.value = null
    page.value = 1
}

const onSearchBlur = () => {
    // Delay so click on result can fire first
    setTimeout(() => { searchOpen.value = false }, 200)
}

const entityIcon = (type: string): string => {
    if (type === 'alliance') return 'lucide:shield'
    if (type === 'corporation') return 'lucide:building-2'
    if (type === 'system') return 'lucide:globe'
    if (type === 'region') return 'lucide:map'
    return 'lucide:circle'
}

const entityImage = (hit: any): string | null => {
    const numericId = Number(hit.id.split('_').pop())
    if (hit.type === 'alliance') return `/images/alliances/${numericId}/logo?size=64`
    if (hit.type === 'corporation') return `/images/corporations/${numericId}/logo?size=64`
    return null
}

const entityTypeLabel = (type: string): string => {
    if (type === 'alliance') return 'Alliance'
    if (type === 'corporation') return 'Corporation'
    if (type === 'system') return 'System'
    if (type === 'region') return 'Region'
    return type
}

// Build fetch params
const fetchParams = computed(() => {
    const p: Record<string, any> = { page: page.value, limit: 50 }
    if (selectedYear.value) p.year = selectedYear.value
    if (minKills.value) p.minKills = minKills.value
    if (minIsk.value) p.minIsk = parseIskFilter(minIsk.value)
    if (showCustom.value) p.custom = 'true'

    if (entityFilter.value) {
        const { type, id } = entityFilter.value
        if (type === 'alliance') p.allianceId = id
        else if (type === 'corporation') p.corporationId = id
        else if (type === 'system') p.systemId = id
        else if (type === 'region') p.regionId = id
    }

    return p
})

const { data, pending } = await useApiFetch<any>('/api/conflicts/battles', {
    params: fetchParams,
    watch: [page, selectedYear, minKills, minIsk, showCustom, entityFilter],
})

// Mirror filter state into the URL so it survives back-button navigation.
// router.replace (not push) avoids polluting history on every filter tweak.
watch([page, selectedYear, minKills, minIsk, showCustom, entityFilter], () => {
    const q: Record<string, string> = {}
    if (page.value > 1) q.page = String(page.value)
    if (selectedYear.value) q.year = String(selectedYear.value)
    if (minKills.value) q.minKills = String(minKills.value)
    if (minIsk.value) q.minIsk = minIsk.value
    if (showCustom.value) q.custom = '1'
    if (entityFilter.value) {
        q.eType = entityFilter.value.type
        q.eId = String(entityFilter.value.id)
    }
    router.replace({ query: q })
})

const battles = computed(() => data.value?.battles || [])
const years = computed(() => data.value?.years || [])

const selectYear = (year: number | null) => {
    selectedYear.value = selectedYear.value === year ? null : year
    page.value = 1
}

const setMinKills = (v: number | null) => {
    minKills.value = minKills.value === v ? null : v
    page.value = 1
}

const setMinIsk = (v: string | null) => {
    minIsk.value = minIsk.value === v ? null : v
    page.value = 1
}

const hasAnyFilter = computed(() =>
    selectedYear.value || minKills.value || minIsk.value || showCustom.value || entityFilter.value
)

const resetFilters = () => {
    selectedYear.value = null
    minKills.value = null
    minIsk.value = null
    showCustom.value = false
    entityFilter.value = null
    page.value = 1
}

function parseIskFilter(v: string): number {
    if (v.endsWith('t')) return parseFloat(v) * 1e12
    if (v.endsWith('b')) return parseFloat(v) * 1e9
    if (v.endsWith('m')) return parseFloat(v) * 1e6
    return parseFloat(v) || 0
}

const formatDuration = (mins: number): string => {
    if (mins < 60) return `${mins}m`
    const h = Math.floor(mins / 60)
    const m = mins % 60
    return m > 0 ? `${h}h ${m}m` : `${h}h`
}

const securityColor = (sec: number | null): string => {
    if (sec == null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-green-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

const teamImage = (entry: any): string => {
    if (entry.alliance_id) return `/images/alliances/${entry.alliance_id}/logo?size=64`
    if (entry.corporation_id) return `/images/corporations/${entry.corporation_id}/logo?size=64`
    return ''
}

const teamName = (entry: any): string => {
    return entry.alliance_name || entry.corporation_name || 'Unknown'
}
</script>
<template>
    <div>
        <PageHeader class="mb-4" title="Battles" eyebrow="Fleet engagements" icon="lucide:swords"
            description="Fleet engagements detected across New Eden, with sides, timelines and ISK destroyed. Browse by year, or filter down to an alliance, corporation, system or region.">
            <template #actions>
                <div class="flex flex-wrap items-center justify-end gap-2">
                <span v-if="isDomainMode"
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-indigo-500/30 bg-indigo-500/10 text-fine font-medium text-indigo-300 flex-shrink-0"
                    v-tooltip="'Scoped to the entities this killboard tracks'">
                    <Icon name="lucide:shield" class="text-fine" />
                    This killboard only
                </span>
                <NuxtLink to="/battlegenerator"
                    class="inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium bg-blue-500 text-white hover:bg-blue-600 transition-colors flex-shrink-0">
                    <Icon name="lucide:plus" class="text-sm" />
                    Battle Report
                </NuxtLink>
                </div>
            </template>
        </PageHeader>

        <!-- Year browser -->
        <div class="glass-panel p-4 mb-4">
            <div class="flex items-center justify-between gap-3 mb-3">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Browse by year</div>
                <button v-if="selectedYear" class="text-fine text-blue-400 hover:text-blue-300 cursor-pointer" @click="selectYear(null)">
                    Clear year
                </button>
            </div>
            <div class="flex flex-wrap gap-1.5">
                <button v-for="y in years" :key="y.year"
                    class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs border transition-colors cursor-pointer"
                    :class="selectedYear === y.year
                        ? 'bg-blue-500/20 text-blue-400 border-blue-500/40'
                        : 'text-gray-400 border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    @click="selectYear(y.year)">
                    <span class="font-medium tabular-nums">{{ y.year }}</span>
                    <span class="tabular-nums text-fine" :class="selectedYear === y.year ? 'text-blue-400/60' : 'text-gray-600'">
                        {{ y.count.toLocaleString('en-US') }}
                    </span>
                </button>
            </div>
        </div>

        <!-- Toolbar -->
        <div class="flex flex-wrap items-center gap-2 mb-4">
            <!-- Custom toggle -->
            <button
                class="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md font-medium border transition-colors cursor-pointer"
                :class="showCustom
                    ? 'bg-amber-500/20 text-amber-400 border-amber-500/30'
                    : 'text-gray-400 border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="showCustom = !showCustom; page = 1">
                <Icon name="lucide:user-pen" class="text-xs" />
                Custom
            </button>

            <div class="w-px h-5 bg-white/[0.08] mx-1" />

            <span class="text-fine text-gray-500 uppercase tracking-wider">Kills</span>
            <div class="flex rounded-lg border border-white/[0.08] overflow-hidden">
                <button v-for="k in [10, 25, 50, 100, 200]" :key="k"
                    class="px-2.5 py-1.5 text-xs font-medium tabular-nums transition-colors cursor-pointer"
                    :class="minKills === k
                        ? 'bg-blue-500/20 text-blue-400'
                        : 'bg-white/[0.02] text-gray-500 hover:bg-white/[0.05] hover:text-gray-300'"
                    @click="setMinKills(k)">
                    {{ k }}+
                </button>
            </div>

            <span class="text-fine text-gray-500 uppercase tracking-wider ml-1">ISK</span>
            <div class="flex rounded-lg border border-white/[0.08] overflow-hidden">
                <button v-for="v in ['1b', '10b', '50b', '100b', '1t']" :key="v"
                    class="px-2.5 py-1.5 text-xs font-medium tabular-nums transition-colors cursor-pointer"
                    :class="minIsk === v
                        ? 'bg-purple-500/20 text-purple-400'
                        : 'bg-white/[0.02] text-gray-500 hover:bg-white/[0.05] hover:text-gray-300'"
                    @click="setMinIsk(v)">
                    {{ v }}+
                </button>
            </div>

            <!-- Entity search -->
            <div class="relative ml-auto w-full sm:w-72">
                <div class="glass-panel flex items-center gap-2 h-9 px-3 focus-within:border-white/[0.2] transition-colors">
                    <Icon name="lucide:search" class="text-sm text-gray-500 flex-shrink-0" />
                    <input
                        ref="searchInputRef"
                        type="text"
                        :value="searchQuery"
                        @input="onSearchInput"
                        @focus="searchResults.length > 0 && (searchOpen = true)"
                        @blur="onSearchBlur"
                        placeholder="Filter by alliance, corp, system…"
                        class="flex-1 min-w-0 bg-transparent text-sm text-white placeholder-gray-500 outline-none"
                    >
                    <Icon v-if="searchLoading" name="lucide:loader-2" class="text-sm text-gray-500 animate-spin flex-shrink-0" />
                </div>

                <!-- Dropdown -->
                <div v-if="searchOpen" class="absolute top-full left-0 right-0 mt-1 z-50 rounded-lg bg-[#141414]/95 backdrop-blur-xl border border-white/[0.1] shadow-xl overflow-hidden">
                    <button v-for="hit in searchResults" :key="hit.id"
                        class="flex items-center gap-3 w-full px-3 py-2 text-left hover:bg-blue-500/[0.06] transition-colors cursor-pointer"
                        @mousedown.prevent="selectEntity(hit)">
                        <img v-if="entityImage(hit)" :src="entityImage(hit)!" class="w-7 h-7 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                        <Icon v-else :name="entityIcon(hit.type)" class="w-7 h-7 text-gray-500 flex-shrink-0 p-1" />
                        <div class="min-w-0 flex-1">
                            <div class="text-sm text-gray-200 truncate">
                                {{ hit.name }}
                                <span v-if="hit.ticker" class="text-gray-500 text-xs">[{{ hit.ticker }}]</span>
                            </div>
                            <div class="text-fine text-gray-500">{{ entityTypeLabel(hit.type) }}</div>
                        </div>
                    </button>
                </div>
            </div>
        </div>

        <!-- Active filter chips -->
        <div v-if="hasAnyFilter" class="flex flex-wrap items-center gap-2 mb-4">
            <div v-if="entityFilter" class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-blue-500/15 border border-blue-500/30">
                <Icon :name="entityIcon(entityFilter.type)" class="text-sm text-blue-400" />
                <span class="text-xs text-blue-400 font-medium">{{ entityFilter.name }}</span>
                <button @click="clearEntity" class="text-blue-400/60 hover:text-blue-400 transition-colors cursor-pointer">
                    <Icon name="lucide:x" class="w-3.5 h-3.5" />
                </button>
            </div>
            <span v-if="selectedYear" class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs bg-white/[0.04] border border-white/[0.08] text-gray-400 tabular-nums">
                {{ selectedYear }}
                <button @click="selectYear(null)" class="text-gray-600 hover:text-gray-300 cursor-pointer"><Icon name="lucide:x" class="text-fine" /></button>
            </span>
            <span v-if="minKills" class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs bg-white/[0.04] border border-white/[0.08] text-gray-400 tabular-nums">
                {{ minKills }}+ kills
                <button @click="setMinKills(minKills)" class="text-gray-600 hover:text-gray-300 cursor-pointer"><Icon name="lucide:x" class="text-fine" /></button>
            </span>
            <span v-if="minIsk" class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs bg-white/[0.04] border border-white/[0.08] text-gray-400 tabular-nums">
                {{ minIsk }}+ ISK
                <button @click="setMinIsk(minIsk)" class="text-gray-600 hover:text-gray-300 cursor-pointer"><Icon name="lucide:x" class="text-fine" /></button>
            </span>
            <button class="px-2 py-1.5 text-xs text-gray-500 hover:text-blue-400 transition-colors cursor-pointer" @click="resetFilters">
                Clear all
            </button>
        </div>

        <!-- Loading -->
        <div v-if="pending && battles.length === 0" class="flex items-center justify-center py-20">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <!-- Battle cards -->
        <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3" :class="{ 'opacity-60': pending }">
            <NuxtLink v-for="b in battles" :key="b.battle_id" :to="`/battle/${b.battle_id}`"
                class="group relative block rounded-lg bg-white/[0.04] border border-white/[0.08] hover:border-blue-500/30 hover:bg-blue-500/[0.04] transition-colors overflow-hidden">

                <!-- Team identity rail -->
                <div class="absolute inset-y-0 left-0 w-[3px] opacity-80 group-hover:opacity-100 transition-opacity"
                    :style="b.teams.length >= 2
                        ? { backgroundImage: 'linear-gradient(to bottom, #ef4444 0%, #ef4444 50%, #3b82f6 50%, #3b82f6 100%)' }
                        : { backgroundColor: '#ef4444' }" />

                <!-- Header bar -->
                <div class="flex items-center justify-between pl-4 pr-3 py-2 border-b border-white/[0.06] bg-white/[0.02]">
                    <div class="flex items-center gap-2 text-xs min-w-0">
                        <img :src="`/images/systems/${b.solar_system_id}?size=32`" alt="" class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                        <span class="text-white font-medium truncate" :class="pochvenClass(b.region_id)">{{ b.solar_system_name }}</span>
                        <span class="tabular-nums flex-shrink-0" :class="securityColor(b.solar_system_security)">
                            {{ b.solar_system_security?.toFixed(1) }}
                        </span>
                        <span class="text-gray-600">·</span>
                        <span class="text-gray-500 truncate" :class="pochvenClass(b.region_id)">{{ b.region_name }}</span>
                    </div>
                    <div class="text-fine text-gray-500 tabular-nums flex-shrink-0 ml-2">{{ formatDate(b.start_time) }}</div>
                </div>

                <!-- Body -->
                <div class="pl-4 pr-3 py-3 space-y-2">
                    <!-- Team 1 -->
                    <div v-if="b.teams[0]" class="flex items-center gap-2">
                        <div class="w-1 self-stretch rounded-full bg-red-500/40 flex-shrink-0"></div>
                        <div class="flex -space-x-1.5 flex-shrink-0">
                            <img v-for="e in b.teams[0].top_alliances.slice(0, 3)" :key="e.alliance_id || e.corporation_id"
                                :src="teamImage(e)" v-tooltip="teamName(e)"
                                class="w-8 h-8 rounded border-2 border-[#151515] bg-gray-900" loading="lazy">
                        </div>
                        <div class="min-w-0 flex-1">
                            <div class="text-xs text-gray-200 truncate">
                                {{ b.teams[0].top_alliances[0] ? teamName(b.teams[0].top_alliances[0]) : 'Team 1' }}
                                <span v-if="b.teams[0].alliance_count > 1" class="text-gray-500">+{{ b.teams[0].alliance_count - 1 }}</span>
                            </div>
                            <div class="text-fine text-gray-500">
                                <span class="text-isk/70 tabular-nums">{{ b.teams[0].total_kills }} kills</span>
                                <span class="text-gray-600 mx-0.5">·</span>
                                <span class="text-isk/70 tabular-nums">{{ formatIsk(b.teams[0].total_isk_destroyed) }}</span>
                            </div>
                        </div>
                    </div>

                    <!-- Efficiency bar + stats -->
                    <div class="flex items-center gap-2 px-1">
                        <div v-if="b.teams.length === 2" class="flex-1 h-1.5 bg-blue-500/20 rounded-full overflow-hidden">
                            <div class="h-full bg-red-500 rounded-full"
                                :style="{ width: `${b.teams[0].total_isk_destroyed + b.teams[1].total_isk_destroyed > 0 ? Math.round(b.teams[0].total_isk_destroyed / (b.teams[0].total_isk_destroyed + b.teams[1].total_isk_destroyed) * 100) : 50}%` }">
                            </div>
                        </div>
                        <div class="flex items-center gap-1.5 text-fine text-gray-500 flex-shrink-0">
                            <span class="tabular-nums">{{ b.kill_count }} kills</span>
                            <span class="text-gray-600">·</span>
                            <span class="tabular-nums">{{ formatIsk(b.total_isk_destroyed) }}</span>
                            <span class="text-gray-600">·</span>
                            <span class="tabular-nums">{{ formatDuration(b.duration_minutes) }}</span>
                        </div>
                    </div>

                    <!-- Team 2 -->
                    <div v-if="b.teams[1]" class="flex items-center gap-2">
                        <div class="w-1 self-stretch rounded-full bg-blue-500/40 flex-shrink-0"></div>
                        <div class="flex -space-x-1.5 flex-shrink-0">
                            <img v-for="e in b.teams[1].top_alliances.slice(0, 3)" :key="e.alliance_id || e.corporation_id"
                                :src="teamImage(e)" v-tooltip="teamName(e)"
                                class="w-8 h-8 rounded border-2 border-[#151515] bg-gray-900" loading="lazy">
                        </div>
                        <div class="min-w-0 flex-1">
                            <div class="text-xs text-gray-200 truncate">
                                {{ b.teams[1].top_alliances[0] ? teamName(b.teams[1].top_alliances[0]) : 'Team 2' }}
                                <span v-if="b.teams[1].alliance_count > 1" class="text-gray-500">+{{ b.teams[1].alliance_count - 1 }}</span>
                            </div>
                            <div class="text-fine text-gray-500">
                                <span class="text-isk/70 tabular-nums">{{ b.teams[1].total_kills }} kills</span>
                                <span class="text-gray-600 mx-0.5">·</span>
                                <span class="text-isk/70 tabular-nums">{{ formatIsk(b.teams[1].total_isk_destroyed) }}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Footer badges -->
                <div v-if="b.is_multi_party || b.kill_count >= 100 || b.total_isk_destroyed >= 1e12" class="flex items-center gap-1.5 pl-4 pr-3 pb-2">
                    <span v-if="b.is_multi_party" class="px-1.5 py-0.5 text-fine rounded bg-amber-500/20 text-amber-400 font-medium">Multi-party</span>
                    <span v-if="b.kill_count >= 100" class="px-1.5 py-0.5 text-fine rounded bg-red-500/20 text-red-400 font-medium">Large Battle</span>
                    <span v-if="b.total_isk_destroyed >= 1e12" class="px-1.5 py-0.5 text-fine rounded bg-purple-500/20 text-purple-400 font-medium">1T+ ISK</span>
                </div>
            </NuxtLink>

            <div v-if="battles.length === 0" class="md:col-span-2 glass-panel p-12 text-center">
                <Icon name="lucide:swords" class="text-3xl text-gray-600 mb-3" />
                <p class="text-sm text-gray-500 mb-1">No battles match these filters</p>
                <button v-if="hasAnyFilter" class="text-sm text-blue-400 hover:text-blue-300 cursor-pointer" @click="resetFilters">
                    Clear filters
                </button>
            </div>
        </div>

        <!-- Pagination -->
        <div v-if="page > 1 || battles.length >= 50" class="flex items-center justify-center gap-2 mt-6">
            <button @click="page = Math.max(1, page - 1)" :disabled="page <= 1"
                class="px-3 py-1.5 rounded-lg text-xs font-medium bg-white/[0.04] text-gray-400 border border-white/[0.08] disabled:opacity-40 hover:bg-blue-500/[0.08] transition-colors cursor-pointer disabled:cursor-default">
                Previous
            </button>
            <span class="text-xs text-gray-500 tabular-nums px-2">Page {{ page }}</span>
            <button @click="page++" :disabled="battles.length < 50"
                class="px-3 py-1.5 rounded-lg text-xs font-medium bg-white/[0.04] text-gray-400 border border-white/[0.08] disabled:opacity-40 hover:bg-blue-500/[0.08] transition-colors cursor-pointer disabled:cursor-default">
                Next
            </button>
        </div>
    </div>
</template>
