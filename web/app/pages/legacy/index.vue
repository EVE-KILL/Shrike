<script setup lang="ts">
useHead({ title: 'Legacy Kills' })
useSeoMeta({
    description: 'Browse 436,000+ killmails from EVE Online\'s early days (2003–2007).',
    ogTitle: 'Legacy Kills — EVE-KILL',
})

const { data: stats } = await useApiFetch<{
    killmails: number
    characters: number
    corporations: number
    alliances: number
}>('/api/legacy/stats')

// Filters
const filterVictim = ref('')
const filterAttacker = ref('')
const filterCorp = ref('')
const filterAlliance = ref('')
const filterSystem = ref('')
const filterShip = ref('')
const filterFrom = ref('')
const filterTo = ref('')

// Applied filters (only update on search click to avoid constant refetching)
const appliedFilters = ref<Record<string, string>>({})

function applyFilters() {
    const f: Record<string, string> = {}
    if (filterVictim.value.trim()) f.victim = filterVictim.value.trim()
    if (filterAttacker.value.trim()) f.attacker = filterAttacker.value.trim()
    if (filterCorp.value.trim()) f.corp = filterCorp.value.trim()
    if (filterAlliance.value.trim()) f.alliance = filterAlliance.value.trim()
    if (filterSystem.value.trim()) f.system = filterSystem.value.trim()
    if (filterShip.value.trim()) f.ship = filterShip.value.trim()
    if (filterFrom.value) f.from = filterFrom.value
    if (filterTo.value) f.to = filterTo.value
    appliedFilters.value = f
}

function clearFilters() {
    filterVictim.value = ''
    filterAttacker.value = ''
    filterCorp.value = ''
    filterAlliance.value = ''
    filterSystem.value = ''
    filterShip.value = ''
    filterFrom.value = ''
    filterTo.value = ''
    appliedFilters.value = {}
}

const hasFilters = computed(() => Object.keys(appliedFilters.value).length > 0)

// Sort
const activeSort = ref('id_desc')
const sortOptions = [
    { value: 'id_desc', label: 'Newest' },
    { value: 'id_asc', label: 'Oldest' },
    { value: 'value_desc', label: 'Value (high)' },
    { value: 'value_asc', label: 'Value (low)' },
    { value: 'time_desc', label: 'Date (new)' },
    { value: 'time_asc', label: 'Date (old)' },
]

const killListParams = computed(() => {
    const p: Record<string, any> = { ...appliedFilters.value }
    if (activeSort.value !== 'id_desc') p.sort = activeSort.value
    return p
})

// Quick filter presets
const quickFilterGroups = [
    {
        label: 'Ships',
        filters: [
            { label: 'Capitals', fn: () => { filterShip.value = 'Avatar,Erebus,Ragnarok,Leviathan,Revelation,Moros,Naglfar,Phoenix,Archon,Thanatos,Nidhoggur,Chimera,Aeon,Hel,Nyx,Wyvern'; applyFilters() } },
            { label: 'Battleships', fn: () => { filterShip.value = 'Raven,Megathron,Apocalypse,Tempest,Dominix,Rokh,Maelstrom,Abaddon,Hyperion,Typhoon,Scorpion,Armageddon'; applyFilters() } },
            { label: 'Pods', fn: () => { filterShip.value = 'Capsule'; applyFilters() } },
        ],
    },
    {
        label: 'Space',
        filters: [
            { label: 'Jita', fn: () => { filterSystem.value = 'Jita'; applyFilters() } },
            { label: 'Rancer', fn: () => { filterSystem.value = 'Rancer'; applyFilters() } },
            { label: 'Amamake', fn: () => { filterSystem.value = 'Amamake'; applyFilters() } },
            { label: 'HED-GP', fn: () => { filterSystem.value = 'HED-GP'; applyFilters() } },
            { label: 'EC-P8R', fn: () => { filterSystem.value = 'EC-P8R'; applyFilters() } },
        ],
    },
    {
        label: 'Year',
        filters: [
            { label: '2003', fn: () => { filterFrom.value = '2003-01-01'; filterTo.value = '2003-12-31'; applyFilters() } },
            { label: '2004', fn: () => { filterFrom.value = '2004-01-01'; filterTo.value = '2004-12-31'; applyFilters() } },
            { label: '2005', fn: () => { filterFrom.value = '2005-01-01'; filterTo.value = '2005-12-31'; applyFilters() } },
            { label: '2006', fn: () => { filterFrom.value = '2006-01-01'; filterTo.value = '2006-12-31'; applyFilters() } },
            { label: '2007', fn: () => { filterFrom.value = '2007-01-01'; filterTo.value = '2007-12-31'; applyFilters() } },
        ],
    },
]

// Autocomplete
interface AcResult { name: string; id: number | null }
const acResults = ref<AcResult[]>([])
const acField = ref<string | null>(null)
let acTimeout: ReturnType<typeof setTimeout>

function acInput(field: string, value: string) {
    clearTimeout(acTimeout)
    if (!value || value.length < 2) { acResults.value = []; acField.value = null; return }
    acField.value = field
    acTimeout = setTimeout(async () => {
        try {
            const data = await apiFetch<AcResult[]>('/api/legacy/autocomplete', { params: { q: value, field } })
            // Only show if still the active field
            if (acField.value === field) acResults.value = data
        } catch { acResults.value = [] }
    }, 200)
}

function acSelect(field: string, name: string) {
    switch (field) {
        case 'victim': filterVictim.value = name; break
        case 'attacker': filterAttacker.value = name; break
        case 'corp': filterCorp.value = name; break
        case 'alliance': filterAlliance.value = name; break
        case 'system': filterSystem.value = name; break
    }
    acResults.value = []
    acField.value = null
}

function acBlur() {
    // Delay to allow click on dropdown
    setTimeout(() => { acResults.value = []; acField.value = null }, 150)
}

// Year selector for top lists
const selectedYear = ref<number | null>(null)
const years = [null, 2003, 2004, 2005, 2006, 2007] as const
const yearLabel = (y: number | null) => y ? String(y) : 'All'
const topParams = computed(() => {
    const p: Record<string, any> = { limit: 10 }
    if (selectedYear.value) p.year = selectedYear.value
    return p
})

const { data: topChars } = await useApiFetch<{ entries: any[] }>('/api/legacy/top', {
    params: computed(() => ({ ...topParams.value, dataType: 'characters' })),
    watch: [selectedYear], default: () => ({ entries: [] }), lazy: true,
})
const { data: topCorps } = await useApiFetch<{ entries: any[] }>('/api/legacy/top', {
    params: computed(() => ({ ...topParams.value, dataType: 'corporations' })),
    watch: [selectedYear], default: () => ({ entries: [] }), lazy: true,
})
const { data: topAllis } = await useApiFetch<{ entries: any[] }>('/api/legacy/top', {
    params: computed(() => ({ ...topParams.value, dataType: 'alliances' })),
    watch: [selectedYear], default: () => ({ entries: [] }), lazy: true,
})
const { data: topShips } = await useApiFetch<{ entries: any[] }>('/api/legacy/top', {
    params: computed(() => ({ ...topParams.value, dataType: 'ships' })),
    watch: [selectedYear], default: () => ({ entries: [] }), lazy: true,
})
const { data: topSystems } = await useApiFetch<{ entries: any[] }>('/api/legacy/top', {
    params: computed(() => ({ ...topParams.value, dataType: 'systems' })),
    watch: [selectedYear], default: () => ({ entries: [] }), lazy: true,
})

// Mobile tab state
const mobileTab = ref<'kills' | 'top'>('kills')
const setMobileTab = (tab: typeof mobileTab.value) => {
    mobileTab.value = tab
    const hash = tab === 'kills' ? '' : `#${tab}`
    window.history.pushState(null, '', hash || window.location.pathname)
}
if (import.meta.client) {
    const fromHash = window.location.hash.replace('#', '')
    if (fromHash === 'top') mobileTab.value = 'top'
    window.addEventListener('popstate', () => {
        mobileTab.value = window.location.hash === '#top' ? 'top' : 'kills'
    })
}

const inputClass = 'w-full rounded border border-white/[0.08] bg-white/[0.03] px-3 py-1.5 text-xs text-gray-200 placeholder-gray-500 focus:outline-none focus:border-blue-400/50'
</script>

<template>
    <div>
        <h1 class="text-xl font-bold text-white mb-4">Legacy Kills</h1>

        <!-- Nostalgia box -->
        <div class="rounded-lg border border-amber-500/30 bg-amber-500/[0.06] px-5 py-4 mb-6 space-y-2">
            <p class="text-sm text-white">Wanna experience the old days of EVE? Now you can — with this killmail dump from before XMLAPI / CREST / ESI, back when killmails were manually posted from the client.</p>
            <p class="text-xs text-gray-300">There are bound to be fakes and mails of dubious quality — but none of it is calculated into the stats on the rest of the site. It is completely isolated and provided as pure nostalgia.</p>
        </div>

        <!-- Stats -->
        <div v-if="stats" class="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
            <div class="rounded-lg border border-white/[0.06] bg-white/[0.02] p-4 text-center">
                <div class="text-xl font-bold text-white tabular-nums">{{ stats.killmails?.toLocaleString('en-US') }}</div>
                <div class="text-fine text-gray-500 uppercase tracking-wider mt-1">Killmails</div>
            </div>
            <div class="rounded-lg border border-white/[0.06] bg-white/[0.02] p-4 text-center">
                <div class="text-xl font-bold text-white tabular-nums">{{ stats.characters?.toLocaleString('en-US') }}</div>
                <div class="text-fine text-gray-500 uppercase tracking-wider mt-1">Characters</div>
            </div>
            <div class="rounded-lg border border-white/[0.06] bg-white/[0.02] p-4 text-center">
                <div class="text-xl font-bold text-white tabular-nums">{{ stats.corporations?.toLocaleString('en-US') }}</div>
                <div class="text-fine text-gray-500 uppercase tracking-wider mt-1">Corporations</div>
            </div>
            <div class="rounded-lg border border-white/[0.06] bg-white/[0.02] p-4 text-center">
                <div class="text-xl font-bold text-white tabular-nums">{{ stats.alliances?.toLocaleString('en-US') }}</div>
                <div class="text-fine text-gray-500 uppercase tracking-wider mt-1">Alliances</div>
            </div>
        </div>

        <!-- Quick filters -->
        <div class="flex flex-wrap items-center gap-x-4 gap-y-2 mb-3">
            <div v-for="group in quickFilterGroups" :key="group.label" class="flex flex-wrap items-center gap-1">
                <span class="text-fine text-gray-600 font-medium uppercase mr-0.5">{{ group.label }}</span>
                <button v-for="qf in group.filters" :key="qf.label" @click="qf.fn()"
                    class="px-2 py-0.5 text-fine font-medium rounded border transition-colors bg-white/[0.04] text-gray-500 border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400">
                    {{ qf.label }}
                </button>
            </div>
        </div>

        <!-- Search / Filter bar -->
        <div class="rounded-lg border border-white/[0.06] bg-white/[0.02] p-4 mb-6">
            <form @submit.prevent="applyFilters">
                <div class="grid grid-cols-2 md:grid-cols-4 gap-3 mb-3">
                    <!-- Victim (autocomplete) -->
                    <div class="relative">
                        <label class="text-fine text-gray-500 uppercase tracking-wider mb-1 block">Victim</label>
                        <input v-model="filterVictim" type="text" placeholder="Character name" :class="inputClass"
                            @input="acInput('victim', filterVictim)" @blur="acBlur">
                        <div v-if="acField === 'victim' && acResults.length" class="absolute z-50 top-full left-0 right-0 mt-1 rounded border border-white/[0.08] bg-gray-900 shadow-lg max-h-48 overflow-y-auto">
                            <button v-for="r in acResults" :key="r.name" type="button"
                                class="w-full px-3 py-1.5 text-xs text-left text-gray-300 hover:bg-blue-500/[0.15] hover:text-white transition-colors"
                                @mousedown.prevent="acSelect('victim', r.name)">
                                {{ r.name }}
                            </button>
                        </div>
                    </div>
                    <!-- Attacker (autocomplete) -->
                    <div class="relative">
                        <label class="text-fine text-gray-500 uppercase tracking-wider mb-1 block">Attacker</label>
                        <input v-model="filterAttacker" type="text" placeholder="Character name" :class="inputClass"
                            @input="acInput('attacker', filterAttacker)" @blur="acBlur">
                        <div v-if="acField === 'attacker' && acResults.length" class="absolute z-50 top-full left-0 right-0 mt-1 rounded border border-white/[0.08] bg-gray-900 shadow-lg max-h-48 overflow-y-auto">
                            <button v-for="r in acResults" :key="r.name" type="button"
                                class="w-full px-3 py-1.5 text-xs text-left text-gray-300 hover:bg-blue-500/[0.15] hover:text-white transition-colors"
                                @mousedown.prevent="acSelect('attacker', r.name)">
                                {{ r.name }}
                            </button>
                        </div>
                    </div>
                    <!-- Corporation (autocomplete) -->
                    <div class="relative">
                        <label class="text-fine text-gray-500 uppercase tracking-wider mb-1 block">Corporation</label>
                        <input v-model="filterCorp" type="text" placeholder="Corporation name" :class="inputClass"
                            @input="acInput('corp', filterCorp)" @blur="acBlur">
                        <div v-if="acField === 'corp' && acResults.length" class="absolute z-50 top-full left-0 right-0 mt-1 rounded border border-white/[0.08] bg-gray-900 shadow-lg max-h-48 overflow-y-auto">
                            <button v-for="r in acResults" :key="r.name" type="button"
                                class="w-full px-3 py-1.5 text-xs text-left text-gray-300 hover:bg-blue-500/[0.15] hover:text-white transition-colors"
                                @mousedown.prevent="acSelect('corp', r.name)">
                                {{ r.name }}
                            </button>
                        </div>
                    </div>
                    <!-- Alliance (autocomplete) -->
                    <div class="relative">
                        <label class="text-fine text-gray-500 uppercase tracking-wider mb-1 block">Alliance</label>
                        <input v-model="filterAlliance" type="text" placeholder="Alliance name" :class="inputClass"
                            @input="acInput('alliance', filterAlliance)" @blur="acBlur">
                        <div v-if="acField === 'alliance' && acResults.length" class="absolute z-50 top-full left-0 right-0 mt-1 rounded border border-white/[0.08] bg-gray-900 shadow-lg max-h-48 overflow-y-auto">
                            <button v-for="r in acResults" :key="r.name" type="button"
                                class="w-full px-3 py-1.5 text-xs text-left text-gray-300 hover:bg-blue-500/[0.15] hover:text-white transition-colors"
                                @mousedown.prevent="acSelect('alliance', r.name)">
                                {{ r.name }}
                            </button>
                        </div>
                    </div>
                    <!-- System (autocomplete) -->
                    <div class="relative">
                        <label class="text-fine text-gray-500 uppercase tracking-wider mb-1 block">System</label>
                        <input v-model="filterSystem" type="text" placeholder="System name" :class="inputClass"
                            @input="acInput('system', filterSystem)" @blur="acBlur">
                        <div v-if="acField === 'system' && acResults.length" class="absolute z-50 top-full left-0 right-0 mt-1 rounded border border-white/[0.08] bg-gray-900 shadow-lg max-h-48 overflow-y-auto">
                            <button v-for="r in acResults" :key="r.name" type="button"
                                class="w-full px-3 py-1.5 text-xs text-left text-gray-300 hover:bg-blue-500/[0.15] hover:text-white transition-colors"
                                @mousedown.prevent="acSelect('system', r.name)">
                                {{ r.name }}
                            </button>
                        </div>
                    </div>
                    <div>
                        <label class="text-fine text-gray-500 uppercase tracking-wider mb-1 block">Ship</label>
                        <input v-model="filterShip" type="text" placeholder="Ship type" :class="inputClass">
                    </div>
                    <div>
                        <label class="text-fine text-gray-500 uppercase tracking-wider mb-1 block">From</label>
                        <input v-model="filterFrom" type="date" min="2003-01-01" max="2007-12-31" :class="inputClass">
                    </div>
                    <div>
                        <label class="text-fine text-gray-500 uppercase tracking-wider mb-1 block">To</label>
                        <input v-model="filterTo" type="date" min="2003-01-01" max="2007-12-31" :class="inputClass">
                    </div>
                </div>
                <div class="flex gap-2">
                    <button type="submit" class="px-4 py-1.5 rounded bg-blue-500/20 text-blue-400 text-xs font-medium hover:bg-blue-500/30 transition-colors border border-blue-500/20">
                        Search
                    </button>
                    <button v-if="hasFilters" type="button" class="px-3 py-1.5 rounded border border-white/[0.08] text-xs text-gray-500 hover:text-white transition-colors" @click="clearFilters">
                        Clear
                    </button>
                </div>
            </form>
        </div>

        <!-- Desktop: sidebar + killlist -->
        <div class="hidden md:grid grid-cols-[250px_1fr] gap-4">
            <div>
                <!-- Year selector -->
                <div class="glass-panel flex gap-1 p-1 mb-4">
                    <button v-for="y in years" :key="y ?? 'all'"
                        class="flex-1 px-1.5 py-1 text-fine rounded-md font-medium transition-colors"
                        :class="selectedYear === y ? 'bg-blue-500/15 text-blue-400' : 'text-gray-500 hover:text-gray-300'"
                        @click="selectedYear = y">
                        {{ yearLabel(y) }}
                    </button>
                </div>

                <TopBox title="Top Victims" dataType="characters" :entries="topChars?.entries" hideFooter />
                <TopBox title="Top Corporations" dataType="corporations" :entries="topCorps?.entries" hideFooter />
                <TopBox title="Top Alliances" dataType="alliances" :entries="topAllis?.entries" hideFooter />
                <TopBox title="Top Ships Lost" dataType="ships" :entries="topShips?.entries" hideFooter />
                <TopBox title="Top Systems" dataType="systems" :entries="topSystems?.entries" hideFooter />
            </div>
            <div>
                <!-- Sort -->
                <div class="flex items-center gap-1 mb-3">
                    <span class="text-fine text-gray-600 font-medium uppercase mr-1">Sort</span>
                    <button v-for="opt in sortOptions" :key="opt.value" @click="activeSort = opt.value"
                        class="px-2 py-0.5 text-fine font-medium rounded border transition-colors"
                        :class="activeSort === opt.value ? 'bg-blue-500/15 text-blue-400 border-blue-500/20' : 'bg-white/[0.04] text-gray-500 border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400'">
                        {{ opt.label }}
                    </button>
                </div>
                <KillList api-endpoint="/api/legacy/kills" :stream-topics="null" :extra-params="killListParams" />
            </div>
        </div>

        <!-- Mobile: tabs -->
        <div class="md:hidden">
            <div class="flex border-b border-white/[0.08] mb-4">
                <button
                    class="flex-1 flex items-center justify-center gap-2 py-3 text-sm font-medium transition-colors border-b-2"
                    :class="mobileTab === 'kills' ? 'text-white border-blue-400' : 'text-gray-500 border-transparent hover:text-blue-400'"
                    @click="setMobileTab('kills')"
                >
                    <Icon name="lucide:swords" class="text-base" />
                    Kills
                </button>
                <button
                    class="flex-1 flex items-center justify-center gap-2 py-3 text-sm font-medium transition-colors border-b-2"
                    :class="mobileTab === 'top' ? 'text-white border-blue-400' : 'text-gray-500 border-transparent hover:text-blue-400'"
                    @click="setMobileTab('top')"
                >
                    <Icon name="lucide:list-ordered" class="text-base" />
                    Top Lists
                </button>
            </div>

            <div v-show="mobileTab === 'kills'">
                <div>
                <!-- Sort -->
                <div class="flex items-center gap-1 mb-3">
                    <span class="text-fine text-gray-600 font-medium uppercase mr-1">Sort</span>
                    <button v-for="opt in sortOptions" :key="opt.value" @click="activeSort = opt.value"
                        class="px-2 py-0.5 text-fine font-medium rounded border transition-colors"
                        :class="activeSort === opt.value ? 'bg-blue-500/15 text-blue-400 border-blue-500/20' : 'bg-white/[0.04] text-gray-500 border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400'">
                        {{ opt.label }}
                    </button>
                </div>
                <KillList api-endpoint="/api/legacy/kills" :stream-topics="null" :extra-params="killListParams" />
            </div>
            </div>

            <div v-show="mobileTab === 'top'" class="space-y-4">
                <!-- Year selector (mobile) -->
                <div class="glass-panel flex gap-1 p-1">
                    <button v-for="y in years" :key="y ?? 'all'"
                        class="flex-1 px-1.5 py-1 text-fine rounded-md font-medium transition-colors"
                        :class="selectedYear === y ? 'bg-blue-500/15 text-blue-400' : 'text-gray-500 hover:text-gray-300'"
                        @click="selectedYear = y">
                        {{ yearLabel(y) }}
                    </button>
                </div>

                <TopBox title="Top Victims" dataType="characters" :entries="topChars?.entries" hideFooter />
                <TopBox title="Top Corporations" dataType="corporations" :entries="topCorps?.entries" hideFooter />
                <TopBox title="Top Alliances" dataType="alliances" :entries="topAllis?.entries" hideFooter />
                <TopBox title="Top Ships Lost" dataType="ships" :entries="topShips?.entries" hideFooter />
                <TopBox title="Top Systems" dataType="systems" :entries="topSystems?.entries" hideFooter />
            </div>
        </div>
    </div>
</template>
