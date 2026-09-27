<script setup lang="ts">
import type { RoamPilot, RoamReport, RoamShip } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
const view = ref<'pilots' | 'ships' | 'groups'>('pilots')
const search = ref('')
const groupFilter = ref<number | null>(null)
const lossOnly = ref(false)
const sortKey = ref('default')
const descending = ref(true)
const views = [
    { id: 'pilots' as const, label: 'Pilots', icon: 'lucide:users' },
    { id: 'ships' as const, label: 'By Ship', icon: 'lucide:rocket' },
    { id: 'groups' as const, label: 'By Group', icon: 'lucide:layers' },
]

type PilotRow = { pilot: RoamPilot, ship: RoamShip, iskLost: number }
type Aggregate = {
    id: number, name: string, groupId: number, groupName: string,
    pilotIds: Set<number>, observations: number, losses: number, iskLost: number, damage: number,
}
const lossValue = computed(() => {
    const values = new Map<string, number>()
    for (const kill of props.report.engagements.flatMap(engagement => engagement.killmails)) {
        if (kill.role !== 'loss') continue
        const key = `${kill.victim_character_id}:${kill.victim_ship_type_id}`
        values.set(key, (values.get(key) ?? 0) + kill.total_value)
    }
    return values
})
const pilotRows = computed<PilotRow[]>(() => props.report.roster.flatMap(pilot =>
    (pilot.ships ?? []).map(ship => ({
        pilot, ship,
        iskLost: lossValue.value.get(`${pilot.character_id}:${ship.ship_type_id}`) ?? 0,
    }))))
const observedPilots = computed(() => new Set(pilotRows.value.map(row => row.pilot.character_id)).size)
const hullTypes = computed(() => new Set(pilotRows.value.map(row => row.ship.ship_type_id)).size)
const groups = computed(() => [...new Map(pilotRows.value
    .filter(row => row.ship.ship_group_id)
    .map(row => [row.ship.ship_group_id, row.ship.ship_group_name || 'Unknown group'] as const)).entries()]
    .sort((a, b) => a[1].localeCompare(b[1])))
const searchTerm = computed(() => search.value.trim().toLowerCase())
const filteredPilots = computed(() => pilotRows.value.filter(row =>
    (!groupFilter.value || row.ship.ship_group_id === groupFilter.value) &&
    (!lossOnly.value || row.ship.losses > 0) &&
    (!searchTerm.value || [row.pilot.name, row.pilot.corporation_name, row.ship.ship_name, row.ship.ship_group_name]
        .some(value => value.toLowerCase().includes(searchTerm.value)))))

function aggregate(rows: PilotRow[], byGroup: boolean): Aggregate[] {
    const totals = new Map<string, Aggregate>()
    for (const row of rows) {
        const key = byGroup ? String(row.ship.ship_group_id || 'unknown') : String(row.ship.ship_type_id)
        const entry = totals.get(key) ?? {
            id: byGroup ? row.ship.ship_group_id : row.ship.ship_type_id,
            name: byGroup ? row.ship.ship_group_name || 'Unknown group' : row.ship.ship_name,
            groupId: row.ship.ship_group_id, groupName: row.ship.ship_group_name,
            pilotIds: new Set<number>(), observations: 0, losses: 0, iskLost: 0, damage: 0,
        }
        entry.pilotIds.add(row.pilot.character_id)
        entry.observations += row.ship.killmails
        entry.losses += row.ship.losses
        entry.iskLost += row.iskLost
        entry.damage += row.ship.damage_done
        totals.set(key, entry)
    }
    return [...totals.values()]
}
const shipRows = computed(() => aggregate(filteredPilots.value, false))
const groupRows = computed(() => aggregate(filteredPilots.value, true))

function compareNumbers(a: number, b: number) { return descending.value ? b - a : a - b }
function compareNames(a: string, b: string) { return descending.value ? b.localeCompare(a) : a.localeCompare(b) }
const sortedPilots = computed(() => [...filteredPilots.value].sort((a, b) => {
    switch (sortKey.value) {
        case 'ship': return compareNames(a.ship.ship_name, b.ship.ship_name)
        case 'pilot': return compareNames(a.pilot.name, b.pilot.name)
        case 'observations': return compareNumbers(a.ship.killmails, b.ship.killmails)
        case 'losses': return compareNumbers(a.ship.losses, b.ship.losses)
        case 'iskLost': return compareNumbers(a.iskLost, b.iskLost)
        case 'damage': return compareNumbers(a.ship.damage_done, b.ship.damage_done)
        default: return b.ship.killmails - a.ship.killmails || b.ship.damage_done - a.ship.damage_done || a.pilot.name.localeCompare(b.pilot.name)
    }
}))
function sortAggregates(rows: Aggregate[]) {
    return [...rows].sort((a, b) => {
        switch (sortKey.value) {
            case 'ship': return compareNames(a.name, b.name)
            case 'pilots': return compareNumbers(a.pilotIds.size, b.pilotIds.size)
            case 'observations': return compareNumbers(a.observations, b.observations)
            case 'losses': return compareNumbers(a.losses, b.losses)
            case 'iskLost': return compareNumbers(a.iskLost, b.iskLost)
            case 'damage': return compareNumbers(a.damage, b.damage)
            default: return b.pilotIds.size - a.pilotIds.size || b.observations - a.observations || a.name.localeCompare(b.name)
        }
    })
}
const sortedShips = computed(() => sortAggregates(shipRows.value))
const sortedGroups = computed(() => sortAggregates(groupRows.value))
function setView(next: typeof view.value) { view.value = next; sortKey.value = 'default'; descending.value = true }
function toggleSort(key: string) {
    if (sortKey.value === key) descending.value = !descending.value
    else { sortKey.value = key; descending.value = true }
}
function sortIcon(key: string) {
    return sortKey.value === key ? descending.value ? 'lucide:chevron-down' : 'lucide:chevron-up' : 'lucide:chevrons-up-down'
}
function resetFilters() { search.value = ''; groupFilter.value = null; lossOnly.value = false }
</script>

<template>
    <div>
        <div class="glass-panel mb-4 border-l-2 border-blue-500/20 p-4">
            <div class="mb-2 text-xs font-semibold text-blue-400">Fleet · recorded observations</div>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                <div><div class="font-mono text-lg text-gray-200">{{ formatNumber(observedPilots) }}</div><div class="text-[10px] text-gray-500">Observed pilots</div></div>
                <div><div class="font-mono text-lg text-gray-200">{{ formatNumber(hullTypes) }}</div><div class="text-[10px] text-gray-500">Hull types</div></div>
                <div><div class="font-mono text-lg text-gray-200">{{ formatNumber(report.summary.losses) }}</div><div class="text-[10px] text-gray-500">Recorded losses</div></div>
                <div><div class="font-mono text-lg text-amber-200">{{ formatIsk(report.summary.isk_lost) }}</div><div class="text-[10px] text-gray-500">ISK lost</div></div>
            </div>
        </div>
        <p class="mb-3 text-xs text-gray-500">Ships observed on the report's killmails, not a complete fleet inventory. Pilots can appear in multiple hulls. {{ report.roster.length - observedPilots }} listed {{ report.roster.length - observedPilots === 1 ? 'pilot has' : 'pilots have' }} no observed ship.</p>

        <div class="glass-panel mb-4 flex flex-wrap items-center gap-3 p-3">
            <label class="text-xs text-gray-400">Find <input v-model="search" type="search" placeholder="Ship, pilot or corporation" class="ml-2 rounded border border-white/10 bg-black/30 px-3 py-2 text-gray-200"></label>
            <label class="text-xs text-gray-400">Ship class <select v-model="groupFilter" class="ml-2 max-w-52 rounded bg-[#141414] p-2 text-gray-200"><option :value="null">All classes</option><option v-for="[id, name] in groups" :key="id" :value="id">{{ name }}</option></select></label>
            <label class="flex items-center gap-2 text-xs text-gray-400"><input v-model="lossOnly" type="checkbox" class="accent-red-400"> Losses only</label>
            <button v-if="search || groupFilter || lossOnly" type="button" class="ml-auto text-xs text-blue-400 hover:underline" @click="resetFilters">Reset</button>
        </div>
        <div class="mb-4 flex flex-wrap gap-2">
            <button v-for="option in views" :key="option.id" type="button" class="inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors"
                :class="view === option.id ? 'border-blue-500/30 bg-blue-500/10 text-blue-300' : 'border-white/10 text-gray-400 hover:text-gray-200'"
                @click="setView(option.id)"><Icon :name="option.icon" /> {{ option.label }}</button>
        </div>

        <div class="rounded-lg border border-blue-500/20 bg-white/[0.04] p-3">
            <div class="mb-3 flex items-center justify-between border-b border-white/[0.06] pb-2">
                <h2 class="text-sm font-bold text-blue-400">Fleet</h2>
                <span class="text-fine text-gray-500">{{ view === 'pilots' ? sortedPilots.length + ' pilot / ship observations' : view === 'ships' ? sortedShips.length + ' ship types' : sortedGroups.length + ' ship groups' }}</span>
            </div>
            <div class="overflow-x-auto">
                <table class="w-full min-w-[680px] table-fixed text-xs">
                    <thead class="text-left text-gray-500">
                        <tr class="border-b border-white/[0.06]">
                            <th class="w-[26%] py-1.5 pr-2 font-medium"><button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('ship')">{{ view === 'groups' ? 'Ship group' : 'Ship' }} <Icon :name="sortIcon('ship')" /></button></th>
                            <th class="w-[25%] py-1.5 px-2 font-medium"><button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort(view === 'pilots' ? 'pilot' : 'pilots')">{{ view === 'pilots' ? 'Pilot' : 'Pilots' }} <Icon :name="sortIcon(view === 'pilots' ? 'pilot' : 'pilots')" /></button></th>
                            <th class="py-1.5 px-2 text-right font-medium"><button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('observations')">Observed <Icon :name="sortIcon('observations')" /></button></th>
                            <th class="py-1.5 px-2 text-right font-medium"><button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('losses')">Lost <Icon :name="sortIcon('losses')" /></button></th>
                            <th class="py-1.5 px-2 text-right font-medium"><button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('iskLost')">ISK Lost <Icon :name="sortIcon('iskLost')" /></button></th>
                            <th class="py-1.5 pl-2 text-right font-medium"><button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('damage')">Dmg Done <Icon :name="sortIcon('damage')" /></button></th>
                        </tr>
                    </thead>
                    <tbody v-if="view === 'pilots'">
                        <tr v-for="row in sortedPilots" :key="`${row.pilot.character_id}-${row.ship.ship_type_id}`" class="border-b border-white/[0.03] text-gray-300 hover:bg-white/[0.02]">
                            <td class="py-1.5 pr-2"><div class="flex min-w-0 items-center gap-2"><img :src="`/images/types/${row.ship.ship_type_id}/render?size=32`" :alt="row.ship.ship_name" class="h-7 w-7 shrink-0 rounded bg-gray-900" loading="lazy"><div class="min-w-0"><NuxtLink :to="`/item/${row.ship.ship_type_id}`" class="block truncate hover:text-blue-400">{{ row.ship.ship_name }}</NuxtLink><span class="block truncate text-fine text-gray-500">{{ row.ship.ship_group_name }}</span></div></div></td>
                            <td class="px-2 py-1.5"><div class="flex min-w-0 items-center gap-2"><img :src="`/images/characters/${row.pilot.character_id}/portrait?size=32`" :alt="row.pilot.name" class="h-6 w-6 shrink-0 rounded bg-gray-900" loading="lazy"><div class="min-w-0"><NuxtLink :to="`/character/${row.pilot.character_id}`" class="block truncate hover:text-blue-400">{{ row.pilot.name }}</NuxtLink><span class="block truncate text-fine text-gray-500">{{ row.pilot.corporation_name }}</span></div></div></td>
                            <td class="px-2 py-1.5 text-right tabular-nums">{{ formatNumber(row.ship.killmails) }}</td>
                            <td class="px-2 py-1.5 text-right tabular-nums" :class="row.ship.losses ? 'text-red-400' : 'text-gray-600'">{{ row.ship.losses || '—' }}</td>
                            <td class="px-2 py-1.5 text-right tabular-nums" :class="row.iskLost ? 'text-red-400' : 'text-gray-600'">{{ row.iskLost ? formatIsk(row.iskLost) : '—' }}</td>
                            <td class="py-1.5 pl-2 text-right tabular-nums">{{ formatNumber(row.ship.damage_done) }}</td>
                        </tr>
                    </tbody>
                    <tbody v-else>
                        <tr v-for="row in (view === 'ships' ? sortedShips : sortedGroups)" :key="row.id || row.name" class="border-b border-white/[0.03] text-gray-300 hover:bg-white/[0.02]">
                            <td class="py-1.5 pr-2"><div class="flex min-w-0 items-center gap-2"><img v-if="view === 'ships' && row.id" :src="`/images/types/${row.id}/render?size=32`" :alt="row.name" class="h-7 w-7 shrink-0 rounded bg-gray-900" loading="lazy"><NuxtLink v-if="row.id" :to="view === 'ships' ? `/item/${row.id}` : `/group/${row.id}`" class="truncate hover:text-blue-400">{{ row.name }}</NuxtLink><span v-else class="truncate">{{ row.name }}</span></div></td>
                            <td class="px-2 py-1.5 text-gray-400">{{ formatNumber(row.pilotIds.size) }} {{ row.pilotIds.size === 1 ? 'pilot' : 'pilots' }}</td>
                            <td class="px-2 py-1.5 text-right tabular-nums">{{ formatNumber(row.observations) }}</td>
                            <td class="px-2 py-1.5 text-right tabular-nums" :class="row.losses ? 'text-red-400' : 'text-gray-600'">{{ row.losses || '—' }}</td>
                            <td class="px-2 py-1.5 text-right tabular-nums" :class="row.iskLost ? 'text-red-400' : 'text-gray-600'">{{ row.iskLost ? formatIsk(row.iskLost) : '—' }}</td>
                            <td class="py-1.5 pl-2 text-right tabular-nums">{{ formatNumber(row.damage) }}</td>
                        </tr>
                    </tbody>
                </table>
                <p v-if="view === 'pilots' ? !sortedPilots.length : view === 'ships' ? !sortedShips.length : !sortedGroups.length" class="py-8 text-center text-xs text-gray-600">No observed ships match these filters.</p>
            </div>
        </div>
    </div>
</template>
