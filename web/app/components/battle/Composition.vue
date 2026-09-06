<script setup lang="ts">
interface Individual {
    character_id: number | null
    character_name: string | null
    corporation_id: number | null
    corporation_name: string | null
    alliance_id: number | null
    alliance_name: string | null
    ship_type_id: number | null
    ship_name: string | null
    ship_group_id: number | null
    ship_group_name: string | null
    isk_lost: number
    damage_done: number
    damage_taken: number
    deaths: number
    team_index: number
    rank: number
}

interface ShipAgg {
    key: string
    name: string
    ship_type_id?: number
    ship_group_id?: number
    count: number
    losses: number
    isk_lost: number
    damage_done: number
    damage_taken: number
    rank: number
}

interface Team {
    team_index: number
    individuals: Individual[]
    by_ship: ShipAgg[]
    by_group: ShipAgg[]
}

const props = defineProps<{
    apiEndpoint: string
    extraParams?: Record<string, any>
}>()

const view = ref<'individuals' | 'by_ship' | 'by_group'>('individuals')

const PAGE_SIZE = 500
const visibleCounts = ref<Record<number, number>>({ 0: PAGE_SIZE, 1: PAGE_SIZE })
const showMore = (teamIdx: number) => {
    visibleCounts.value[teamIdx] = (visibleCounts.value[teamIdx] || PAGE_SIZE) + PAGE_SIZE
}

const views = [
    { id: 'individuals' as const, label: 'Pilots', icon: 'lucide:users' },
    { id: 'by_ship' as const, label: 'By Ship', icon: 'lucide:rocket' },
    { id: 'by_group' as const, label: 'By Group', icon: 'lucide:layers' },
]

const { data, pending, error, refresh } = await useApiFetch<{ teams: Team[] }>(
    props.apiEndpoint,
    {
        params: props.extraParams,
        default: () => ({ teams: [] }),
    },
)

const teams = computed(() => data.value?.teams ?? [])
const search = ref('')
const groupFilter = ref<number | null>(Number(useRoute().query.group) || null)
const lossOnly = ref(false)
const groups = computed(() => [...new Map(teams.value.flatMap(team => team.by_group.map(group => [group.ship_group_id ?? Number(group.key), group.name] as const))).entries()].sort((a,b) => a[1].localeCompare(b[1])))
const stats = computed(() => teams.value.map(team => ({
    pilots: new Set(team.individuals.map(pilot => pilot.character_id).filter(Boolean)).size,
    hulls: team.by_ship.length,
    losses: team.by_ship.reduce((sum, ship) => sum + ship.losses, 0),
    isk: team.by_ship.reduce((sum, ship) => sum + ship.isk_lost, 0),
})))
const matches = (name: string | null | undefined) => !search.value || (name ?? '').toLowerCase().includes(search.value.toLowerCase())
function resetFilters() { search.value = ''; groupFilter.value = null; lossOnly.value = false }
watch([search, groupFilter, lossOnly], () => { visibleCounts.value = {0: PAGE_SIZE, 1: PAGE_SIZE} })


// Sorting state per view. key='default' uses the server-side rank-based sort.
type SortDir = 'asc' | 'desc'
type SortState = { key: string, dir: SortDir }
const sortInd = ref<SortState>({ key: 'default', dir: 'desc' })
const sortShip = ref<SortState>({ key: 'default', dir: 'desc' })
const sortGroup = ref<SortState>({ key: 'default', dir: 'desc' })

type SortTarget = 'ind' | 'ship' | 'group'
const sortRefs: Record<SortTarget, Ref<SortState>> = {
    ind: sortInd,
    ship: sortShip,
    group: sortGroup,
}
function toggleSort(target: SortTarget, key: string) {
    const state = sortRefs[target]
    if (state.value.key === key) {
        state.value = { key, dir: state.value.dir === 'desc' ? 'asc' : 'desc' }
    } else {
        state.value = { key, dir: 'desc' }
    }
}

function sortIcon(state: SortState, key: string): string {
    if (state.key !== key) return 'lucide:chevrons-up-down'
    return state.dir === 'desc' ? 'lucide:chevron-down' : 'lucide:chevron-up'
}

const cmpStr = (a: string | null | undefined, b: string | null | undefined) =>
    (a || '').localeCompare(b || '')
const cmpNum = (a: number, b: number) => a - b

function sortIndividuals(list: Individual[], state: SortState): Individual[] {
    const arr = list.slice()
    if (state.key === 'default') {
        arr.sort((a, b) =>
            (b.rank - a.rank)
            || (b.isk_lost - a.isk_lost)
            || (b.damage_done - a.damage_done)
            || cmpStr(a.character_name, b.character_name)
        )
        return arr
    }
    const sign = state.dir === 'asc' ? 1 : -1
    arr.sort((a, b) => {
        let c = 0
        switch (state.key) {
            case 'ship_name': c = cmpStr(a.ship_name, b.ship_name); break
            case 'character_name': c = cmpStr(a.character_name, b.character_name); break
            case 'isk_lost': c = cmpNum(a.isk_lost, b.isk_lost); break
            case 'damage_done': c = cmpNum(a.damage_done, b.damage_done); break
            case 'damage_taken': c = cmpNum(a.damage_taken, b.damage_taken); break
        }
        return sign * c
    })
    return arr
}

function sortAggs(list: ShipAgg[], state: SortState): ShipAgg[] {
    const arr = list.slice()
    if (state.key === 'default') {
        arr.sort((a, b) => (b.rank - a.rank) || (b.count - a.count) || (b.isk_lost - a.isk_lost))
        return arr
    }
    const sign = state.dir === 'asc' ? 1 : -1
    arr.sort((a, b) => {
        let c = 0
        switch (state.key) {
            case 'name': c = cmpStr(a.name, b.name); break
            case 'count': c = cmpNum(a.count, b.count); break
            case 'losses': c = cmpNum(a.losses, b.losses); break
            case 'isk_lost': c = cmpNum(a.isk_lost, b.isk_lost); break
            case 'damage_done': c = cmpNum(a.damage_done, b.damage_done); break
            case 'damage_taken': c = cmpNum(a.damage_taken, b.damage_taken); break
        }
        return sign * c
    })
    return arr
}

const teamA = computed(() => teams.value[0])
const teamB = computed(() => teams.value[1])

const sortedIndividuals = (team: Team | undefined) =>
    team ? sortIndividuals(team.individuals.filter(row => (!groupFilter.value || row.ship_group_id === groupFilter.value) && (!lossOnly.value || row.deaths > 0) && (matches(row.ship_name) || matches(row.character_name) || matches(row.corporation_name))), sortInd.value) : []
const sortedByShip = (team: Team | undefined) =>
    team ? sortAggs(team.by_ship.filter(row => (!groupFilter.value || row.ship_group_id === groupFilter.value) && (!lossOnly.value || row.losses > 0) && matches(row.name)), sortShip.value) : []
const sortedByGroup = (team: Team | undefined) =>
    team ? sortAggs(team.by_group.filter(row => (!groupFilter.value || row.ship_group_id === groupFilter.value) && (!lossOnly.value || row.losses > 0) && matches(row.name)), sortGroup.value) : []

const fmtNum = (v: number) => (v ?? 0).toLocaleString('en-US')

const teamLabel = (idx: number) => idx === 0 ? 'Team A' : 'Team B'
const teamBorder = (idx: number) => idx === 0 ? 'border-red-500/20' : 'border-blue-500/20'
const teamAccent = (idx: number) => idx === 0 ? 'text-red-400' : 'text-blue-400'

const corpLogo = (corpId: number | null) =>
    corpId ? `/images/corporations/${corpId}/logo?size=32` : null
const allianceLogo = (allianceId: number | null) =>
    allianceId ? `/images/alliances/${allianceId}/logo?size=32` : null
const shipRender = (typeId: number | null) =>
    typeId ? `/images/types/${typeId}/render?size=32` : null
</script>

<template>
    <div>
        <div class="mb-4 grid gap-3 sm:grid-cols-2">
            <div v-for="(team,index) in stats" :key="index" class="glass-panel border-l-2 p-4" :class="teamBorder(index)"><div class="mb-2 text-xs font-semibold" :class="teamAccent(index)">{{ teamLabel(index) }} · full-battle observations</div><div class="grid grid-cols-2 gap-3 lg:grid-cols-4"><div><div class="font-mono text-lg text-gray-200">{{ fmtNum(team.pilots) }}</div><div class="text-[10px] text-gray-500">Observed pilots</div></div><div><div class="font-mono text-lg text-gray-200">{{ fmtNum(team.hulls) }}</div><div class="text-[10px] text-gray-500">Hull types</div></div><div><div class="font-mono text-lg text-gray-200">{{ fmtNum(team.losses) }}</div><div class="text-[10px] text-gray-500">Recorded losses</div></div><div><div class="font-mono text-lg text-amber-200">{{ formatIsk(team.isk) }}</div><div class="text-[10px] text-gray-500">ISK lost</div></div></div></div>
        </div>
        <p class="mb-3 text-xs text-gray-500">Observed ships and pilots from the full battle’s killmails, not a complete fleet roster. Pilots can appear in multiple hulls.</p>
        <div class="glass-panel mb-4 flex flex-wrap items-center gap-3 p-3">
            <label class="text-xs text-gray-400">Find <input v-model="search" type="search" placeholder="Ship, pilot or corporation" class="ml-2 rounded border border-white/10 bg-black/30 px-3 py-2"></label>
            <label class="text-xs text-gray-400">Ship class <select v-model="groupFilter" class="ml-2 max-w-52 rounded bg-[#141414] p-2"><option :value="null">All classes</option><option v-for="[id,name] in groups" :key="id" :value="id">{{ name }}</option></select></label>
            <label class="flex items-center gap-2 text-xs text-gray-400"><input v-model="lossOnly" type="checkbox">With recorded losses</label>
            <button class="ml-auto text-xs text-sky-300" @click="resetFilters">Reset filters</button>
        </div>
        <!-- View switcher -->
        <div class="flex gap-1 mb-4">
            <button v-for="v in views" :key="v.id"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors cursor-pointer"
                :class="view === v.id ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="view = v.id">
                <Icon :name="v.icon" class="text-sm" />
                {{ v.label }}
            </button>
        </div>

        <div v-if="error" role="alert" class="glass-panel p-5 text-red-300">Unable to load composition. <button class="underline" @click="refresh()">Retry</button></div>
        <div v-else-if="pending" class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div v-for="i in 2" :key="i" class="h-96 rounded-lg bg-white/[0.04] animate-pulse"></div>
        </div>

        <div v-else-if="teams.length < 2" class="glass-panel p-8 text-center text-sm text-gray-500">
            Composition data is not available for this battle.
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <template v-for="(team, tIdx) in [teamA, teamB]" :key="tIdx">
                <div class="rounded-lg bg-white/[0.04] border p-3" :class="teamBorder(tIdx)">
                    <div class="flex items-center justify-between mb-3 pb-2 border-b border-white/[0.06]">
                        <div class="text-sm font-bold" :class="teamAccent(tIdx)">{{ teamLabel(tIdx) }}</div>
                        <div class="text-fine text-gray-500">
                            <template v-if="view === 'individuals'">{{ fmtNum(sortedIndividuals(team).length) }} pilots</template>
                            <template v-else-if="view === 'by_ship'">{{ fmtNum(sortedByShip(team).length) }} ship types</template>
                            <template v-else>{{ fmtNum(sortedByGroup(team).length) }} ship groups</template>
                        </div>
                    </div>

                    <!-- View 1: Individuals -->
                    <div v-if="view === 'individuals'">
                        <table class="w-full text-xs table-fixed">
                            <thead class="text-gray-500 text-left">
                                <tr class="border-b border-white/[0.06]">
                                    <th class="py-1.5 pr-2 font-medium">
                                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ind','ship_name')">
                                            Ship <Icon :name="sortIcon(sortInd, 'ship_name')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium">
                                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ind','character_name')">
                                            Pilot <Icon :name="sortIcon(sortInd, 'character_name')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right w-[90px]">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ind','isk_lost')">
                                            ISK Lost <Icon :name="sortIcon(sortInd, 'isk_lost')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 pl-2 font-medium text-right w-[90px]">
                                        <div class="flex flex-col items-end leading-tight">
                                            <button class="flex items-center gap-1 text-green-400/80 hover:text-green-400" @click="toggleSort('ind','damage_done')">
                                                Done <Icon :name="sortIcon(sortInd, 'damage_done')" class="text-fine" />
                                            </button>
                                            <button class="flex items-center gap-1 text-red-400/80 hover:text-red-400" @click="toggleSort('ind','damage_taken')">
                                                Taken <Icon :name="sortIcon(sortInd, 'damage_taken')" class="text-fine" />
                                            </button>
                                        </div>
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="(ind, i) in sortedIndividuals(team).slice(0, visibleCounts[tIdx] || 500)" :key="i"
                                    class="border-b border-white/[0.03] hover:bg-white/[0.02] align-top">
                                    <td class="py-1.5 pr-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <img v-if="shipRender(ind.ship_type_id)" :src="shipRender(ind.ship_type_id)!" class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                                            <div class="min-w-0 leading-tight">
                                                <NuxtLink v-if="ind.ship_type_id" :to="`/item/${ind.ship_type_id}`" class="block truncate hover:text-blue-400" :class="ind.deaths > 0 ? 'text-red-300' : 'text-gray-300'">{{ ind.ship_name || '?' }}</NuxtLink>
                                                <NuxtLink v-if="ind.ship_group_id" :to="`/group/${ind.ship_group_id}`" class="block truncate text-fine text-gray-500 hover:text-blue-400">{{ ind.ship_group_name || '?' }}</NuxtLink>
                                                <div v-else class="truncate text-fine text-gray-500">{{ ind.ship_group_name || '?' }}</div>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="py-1.5 px-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <img v-if="allianceLogo(ind.alliance_id) || corpLogo(ind.corporation_id)"
                                                :src="(allianceLogo(ind.alliance_id) || corpLogo(ind.corporation_id))!"
                                                class="w-8 h-8 rounded flex-shrink-0" loading="lazy">
                                            <div class="min-w-0 leading-tight">
                                                <NuxtLink v-if="ind.character_id" :to="`/character/${ind.character_id}`" class="block truncate text-gray-300 hover:text-blue-400">
                                                    {{ ind.character_name || 'Unknown' }}
                                                </NuxtLink>
                                                <span v-else class="block truncate text-gray-300">{{ ind.character_name || 'Unknown' }}</span>
                                                <div class="truncate text-fine text-gray-500">{{ ind.corporation_name || '' }}</div>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="py-1.5 px-2 text-right tabular-nums" :class="ind.isk_lost > 0 ? 'text-red-400' : 'text-gray-700'">
                                        {{ ind.isk_lost > 0 ? formatIsk(ind.isk_lost) : '—' }}
                                    </td>
                                    <td class="py-1.5 pl-2 text-right tabular-nums leading-tight">
                                        <div class="text-green-400">{{ fmtNum(ind.damage_done) }}</div>
                                        <div class="text-red-400 text-fine">{{ fmtNum(ind.damage_taken) }}</div>
                                    </td>
                                </tr>
                                <tr v-if="(sortedIndividuals(team).length) === 0">
                                    <td colspan="4" class="py-4 text-center text-gray-600">No pilots</td>
                                </tr>
                            </tbody>
                        </table>
                        <button v-if="(sortedIndividuals(team).length) > (visibleCounts[tIdx] || 500)"
                            @click="showMore(tIdx)"
                            class="w-full py-2.5 mt-2 rounded-lg text-xs font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors">
                            Show more ({{ fmtNum((sortedIndividuals(team).length) - (visibleCounts[tIdx] || 500)) }} remaining)
                        </button>
                    </div>

                    <!-- View 2: By Ship Type -->
                    <div v-else-if="view === 'by_ship'" class="overflow-x-auto">
                        <table class="w-full text-xs">
                            <thead class="text-gray-500 text-left">
                                <tr class="border-b border-white/[0.06]">
                                    <th class="py-1.5 pr-2 font-medium">
                                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ship','name')">
                                            Ship <Icon :name="sortIcon(sortShip, 'name')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ship','count')">
                                            Count <Icon :name="sortIcon(sortShip, 'count')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ship','losses')">
                                            Lost <Icon :name="sortIcon(sortShip, 'losses')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ship','isk_lost')">
                                            ISK Lost <Icon :name="sortIcon(sortShip, 'isk_lost')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ship','damage_done')">
                                            Dmg Done <Icon :name="sortIcon(sortShip, 'damage_done')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 pl-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ship','damage_taken')">
                                            Dmg Taken <Icon :name="sortIcon(sortShip, 'damage_taken')" class="text-fine" />
                                        </button>
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="row in sortedByShip(team)" :key="row.key"
                                    class="border-b border-white/[0.03] hover:bg-white/[0.02] text-gray-300">
                                    <td class="py-1 pr-2">
                                        <div class="flex items-center gap-1.5">
                                            <img v-if="shipRender(row.ship_type_id ?? null)" :src="shipRender(row.ship_type_id ?? null)!" class="w-6 h-6 rounded" loading="lazy">
                                            <NuxtLink v-if="row.ship_type_id" :to="`/item/${row.ship_type_id}`" class="hover:text-blue-400">{{ row.name }}</NuxtLink>
                                            <span v-else>{{ row.name }}</span>
                                        </div>
                                    </td>
                                    <td class="py-1 px-2 text-right tabular-nums">{{ fmtNum(row.count) }}</td>
                                    <td class="py-1 px-2 text-right tabular-nums" :class="row.losses > 0 ? 'text-red-400' : 'text-gray-600'">
                                        {{ row.losses > 0 ? fmtNum(row.losses) : '—' }}
                                    </td>
                                    <td class="py-1 px-2 text-right tabular-nums" :class="row.isk_lost > 0 ? 'text-red-400' : 'text-gray-600'">
                                        {{ row.isk_lost > 0 ? formatIsk(row.isk_lost) : '—' }}
                                    </td>
                                    <td class="py-1 px-2 text-right tabular-nums">{{ fmtNum(row.damage_done) }}</td>
                                    <td class="py-1 pl-2 text-right tabular-nums text-gray-500">{{ fmtNum(row.damage_taken) }}</td>
                                </tr>
                                <tr v-if="(sortedByShip(team).length) === 0">
                                    <td colspan="6" class="py-4 text-center text-gray-600">No ships</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>

                    <!-- View 3: By Ship Group -->
                    <div v-else class="overflow-x-auto">
                        <table class="w-full text-xs">
                            <thead class="text-gray-500 text-left">
                                <tr class="border-b border-white/[0.06]">
                                    <th class="py-1.5 pr-2 font-medium">
                                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('group','name')">
                                            Ship Group <Icon :name="sortIcon(sortGroup, 'name')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('group','count')">
                                            Count <Icon :name="sortIcon(sortGroup, 'count')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('group','losses')">
                                            Lost <Icon :name="sortIcon(sortGroup, 'losses')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('group','isk_lost')">
                                            ISK Lost <Icon :name="sortIcon(sortGroup, 'isk_lost')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 px-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('group','damage_done')">
                                            Dmg Done <Icon :name="sortIcon(sortGroup, 'damage_done')" class="text-fine" />
                                        </button>
                                    </th>
                                    <th class="py-1.5 pl-2 font-medium text-right">
                                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('group','damage_taken')">
                                            Dmg Taken <Icon :name="sortIcon(sortGroup, 'damage_taken')" class="text-fine" />
                                        </button>
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="row in sortedByGroup(team)" :key="row.key"
                                    class="border-b border-white/[0.03] hover:bg-white/[0.02] text-gray-300">
                                    <td class="py-1 pr-2"><NuxtLink v-if="row.ship_group_id" :to="`/group/${row.ship_group_id}`" class="hover:text-blue-400">{{ row.name }}</NuxtLink><span v-else>{{ row.name }}</span></td>
                                    <td class="py-1 px-2 text-right tabular-nums">{{ fmtNum(row.count) }}</td>
                                    <td class="py-1 px-2 text-right tabular-nums" :class="row.losses > 0 ? 'text-red-400' : 'text-gray-600'">
                                        {{ row.losses > 0 ? fmtNum(row.losses) : '—' }}
                                    </td>
                                    <td class="py-1 px-2 text-right tabular-nums" :class="row.isk_lost > 0 ? 'text-red-400' : 'text-gray-600'">
                                        {{ row.isk_lost > 0 ? formatIsk(row.isk_lost) : '—' }}
                                    </td>
                                    <td class="py-1 px-2 text-right tabular-nums">{{ fmtNum(row.damage_done) }}</td>
                                    <td class="py-1 pl-2 text-right tabular-nums text-gray-500">{{ fmtNum(row.damage_taken) }}</td>
                                </tr>
                                <tr v-if="(sortedByGroup(team).length) === 0">
                                    <td colspan="6" class="py-4 text-center text-gray-600">No ship groups</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </template>
        </div>
    </div>
</template>
