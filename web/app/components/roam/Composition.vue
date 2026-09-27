<script setup lang="ts">
import type { RoamReport, RoamShip } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
const view = ref<'pilots' | 'ships' | 'groups'>('pilots')
const search = ref('')
const views = [
    { id: 'pilots' as const, label: 'By pilot', icon: 'lucide:users' },
    { id: 'ships' as const, label: 'By ship', icon: 'lucide:rocket' },
    { id: 'groups' as const, label: 'By group', icon: 'lucide:layers' },
]
interface ShipAggregate {
    id: number
    name: string
    groupName: string
    pilots: Set<number>
    observations: number
    losses: number
    damage: number
}
const observedPilots = computed(() => props.report.roster.filter(pilot => (pilot.ships ?? []).length).length)
const uniqueShips = computed(() => new Set(props.report.roster.flatMap(pilot => (pilot.ships ?? []).map(ship => ship.ship_type_id))).size)
const byShip = computed<ShipAggregate[]>(() => {
    const totals = new Map<number, ShipAggregate>()
    for (const pilot of props.report.roster) {
        for (const ship of pilot.ships ?? []) {
            const entry = totals.get(ship.ship_type_id) ?? {
                id: ship.ship_type_id, name: ship.ship_name, groupName: ship.ship_group_name || 'Unknown group',
                pilots: new Set<number>(), observations: 0, losses: 0, damage: 0,
            }
            entry.pilots.add(pilot.character_id)
            entry.observations += ship.killmails
            entry.losses += ship.losses
            entry.damage += ship.damage_done
            totals.set(ship.ship_type_id, entry)
        }
    }
    return [...totals.values()].sort((a, b) => b.pilots.size - a.pilots.size || b.observations - a.observations || a.name.localeCompare(b.name))
})
const byGroup = computed<ShipAggregate[]>(() => {
    const totals = new Map<string, ShipAggregate>()
    for (const pilot of props.report.roster) {
        for (const ship of pilot.ships ?? []) {
            const key = ship.ship_group_id ? String(ship.ship_group_id) : 'unknown'
            const entry = totals.get(key) ?? {
                id: ship.ship_group_id, name: ship.ship_group_name || 'Unknown group', groupName: 'Ship group',
                pilots: new Set<number>(), observations: 0, losses: 0, damage: 0,
            }
            entry.pilots.add(pilot.character_id)
            entry.observations += ship.killmails
            entry.losses += ship.losses
            entry.damage += ship.damage_done
            totals.set(key, entry)
        }
    }
    return [...totals.values()].sort((a, b) => b.pilots.size - a.pilots.size || b.observations - a.observations || a.name.localeCompare(b.name))
})
const query = computed(() => search.value.trim().toLowerCase())
const visiblePilots = computed(() => [...props.report.roster]
    .sort((a, b) => (b.ships?.length ?? 0) - (a.ships?.length ?? 0))
    .filter(pilot => !query.value || pilot.name.toLowerCase().includes(query.value) ||
        (pilot.ships ?? []).some(ship => ship.ship_name.toLowerCase().includes(query.value))))
const visibleShips = computed(() => byShip.value.filter(ship => !query.value || ship.name.toLowerCase().includes(query.value)))
const visibleGroups = computed(() => byGroup.value.filter(group => !query.value || group.name.toLowerCase().includes(query.value)))
function shipImage(ship: RoamShip) {
    return `/images/types/${ship.ship_type_id}/render?size=64`
}
</script>

<template>
    <div class="space-y-5">
        <div class="glass-panel flex flex-wrap items-center justify-between gap-4 p-5">
            <div>
                <h2 class="text-lg font-semibold text-white">Fleet composition</h2>
                <p class="mt-1 text-sm text-gray-400">{{ uniqueShips }} ship {{ uniqueShips === 1 ? 'type' : 'types' }} observed across {{ observedPilots }} {{ observedPilots === 1 ? 'pilot' : 'pilots' }}. A pilot may appear in several ships.</p>
            </div>
            <input v-model="search" type="search" placeholder="Search pilots or ships" aria-label="Search composition"
                class="w-full rounded-lg border border-white/10 bg-black/25 px-3 py-2 text-sm text-gray-100 outline-none placeholder:text-gray-600 focus:border-blue-500/50 sm:w-60">
        </div>
        <div class="flex flex-wrap gap-2">
            <button v-for="option in views" :key="option.id" type="button"
                class="inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors"
                :class="view === option.id ? 'border-blue-500/30 bg-blue-500/10 text-blue-300' : 'border-white/10 text-gray-400 hover:text-gray-200'"
                @click="view = option.id"><Icon :name="option.icon" /> {{ option.label }}</button>
        </div>

        <div v-if="view === 'pilots'" class="glass-panel divide-y divide-white/[0.06] overflow-hidden">
            <div v-for="pilot in visiblePilots" :key="pilot.character_id" class="grid gap-3 p-4 sm:grid-cols-[minmax(12rem,0.4fr)_minmax(0,1fr)] sm:items-center sm:px-5">
                <div class="flex min-w-0 items-center gap-3">
                    <img :src="`/images/characters/${pilot.character_id}/portrait?size=64`" :alt="pilot.name" class="h-10 w-10 rounded-full bg-gray-900" loading="lazy">
                    <div class="min-w-0">
                        <NuxtLink :to="`/character/${pilot.character_id}`" class="block truncate text-sm font-medium text-gray-100 hover:text-blue-300">{{ pilot.name }}</NuxtLink>
                        <p class="truncate text-xs text-gray-500">{{ pilot.corporation_name || 'Unknown corporation' }}<template v-if="pilot.alliance_name"> · {{ pilot.alliance_name }}</template></p>
                    </div>
                </div>
                <div v-if="pilot.ships?.length" class="flex flex-wrap gap-2">
                    <div v-for="ship in pilot.ships" :key="ship.ship_type_id" class="flex items-center gap-2 rounded-lg border border-white/[0.07] bg-white/[0.02] px-2 py-1.5">
                        <img :src="shipImage(ship)" :alt="ship.ship_name" class="h-8 w-8 rounded bg-gray-900" loading="lazy">
                        <div>
                            <NuxtLink :to="`/item/${ship.ship_type_id}`" class="block text-xs font-medium text-gray-200 hover:text-blue-300">{{ ship.ship_name }}</NuxtLink>
                            <p class="text-xs tabular-nums text-gray-500">{{ ship.killmails }} observed<template v-if="ship.losses"> · {{ ship.losses }} lost</template></p>
                        </div>
                    </div>
                </div>
                <p v-else class="text-xs text-gray-500">No ship observed on matching killmails</p>
            </div>
            <p v-if="!visiblePilots.length" class="p-8 text-center text-sm text-gray-500">No pilots match this search.</p>
        </div>
        <div v-else class="glass-panel divide-y divide-white/[0.06] overflow-hidden">
            <div v-for="entry in (view === 'ships' ? visibleShips : visibleGroups)" :key="entry.id || entry.name" class="flex items-center gap-3 p-4 sm:px-5">
                <img v-if="view === 'ships'" :src="`/images/types/${entry.id}/render?size=64`" :alt="entry.name" class="h-10 w-10 rounded bg-gray-900" loading="lazy">
                <Icon v-else name="lucide:layers" class="text-2xl text-blue-300" />
                <div class="min-w-0 flex-1">
                    <NuxtLink v-if="entry.id" :to="view === 'ships' ? `/item/${entry.id}` : `/group/${entry.id}`" class="block truncate text-sm font-medium text-gray-100 hover:text-blue-300">{{ entry.name }}</NuxtLink>
                    <p v-else class="truncate text-sm font-medium text-gray-100">{{ entry.name }}</p>
                    <p class="truncate text-xs text-gray-500">{{ entry.groupName }}</p>
                </div>
                <div class="shrink-0 text-right text-xs tabular-nums text-gray-400">
                    <p>{{ entry.pilots.size }} {{ entry.pilots.size === 1 ? 'pilot' : 'pilots' }} · {{ entry.observations }} observed</p>
                    <p class="mt-1">{{ entry.losses }} {{ entry.losses === 1 ? 'loss' : 'losses' }} · {{ formatNumber(entry.damage) }} damage</p>
                </div>
            </div>
            <p v-if="!(view === 'ships' ? visibleShips : visibleGroups).length" class="p-8 text-center text-sm text-gray-500">No observed ships match this search.</p>
        </div>
        <p class="text-xs leading-5 text-gray-500">Composition includes ships visible on matching killmails. It cannot show undocked ships or pilots who did not appear on a killmail.</p>
    </div>
</template>
