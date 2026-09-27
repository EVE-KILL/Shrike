<script setup lang="ts">
import type { RoamPilot, RoamReport, RoamShip } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
type ObservedHull = { pilot: RoamPilot, ship: RoamShip }
const hulls = computed<ObservedHull[]>(() => props.report.roster.flatMap(pilot =>
    (pilot.ships ?? []).map(ship => ({ pilot, ship }))))
const commandHulls = computed(() => hulls.value.filter(({ ship }) =>
    ['Flag Cruiser', 'Command Ship', 'Command Destroyer'].includes(ship.ship_group_name)))
const logistics = computed(() => hulls.value.filter(({ ship }) =>
    ['Logistics', 'Logistics Frigate', 'Force Auxiliary'].includes(ship.ship_group_name)))
const capitals = computed(() => hulls.value.filter(({ ship }) =>
    ['Titan', 'Supercarrier', 'Lancer Dreadnought', 'Dreadnought', 'Carrier', 'Force Auxiliary'].includes(ship.ship_group_name)))
const specialistSections = computed(() => [
    { title: 'Command hulls', icon: 'lucide:crown', description: 'Command-class ships on matching killmails; this does not identify the FC.', rows: commandHulls.value },
    { title: 'Logistics', icon: 'lucide:heart-pulse', description: 'Logistics hulls observed on matching killmails.', rows: logistics.value },
    { title: 'Capitals', icon: 'lucide:ship', description: 'Capital hulls observed on matching killmails.', rows: capitals.value },
].filter(section => section.rows.length))
const topPilots = computed(() => [...props.report.roster]
    .filter(pilot => pilot.kill_participations || pilot.losses)
    .sort((a, b) => b.kill_participations - a.kill_participations || b.damage_done - a.damage_done)
    .slice(0, 12))
const destroyedShips = computed(() => {
    const ships = new Map<number, { id: number, name: string, count: number, isk: number }>()
    for (const kill of props.report.engagements.flatMap(engagement => engagement.killmails)) {
        if (kill.role !== 'kill' || !kill.victim_ship_type_id) continue
        const ship = ships.get(kill.victim_ship_type_id) ?? {
            id: kill.victim_ship_type_id, name: kill.victim_ship_name || 'Unknown ship', count: 0, isk: 0,
        }
        ship.count++
        ship.isk += kill.total_value
        ships.set(ship.id, ship)
    }
    return [...ships.values()].sort((a, b) => b.count - a.count || b.isk - a.isk).slice(0, 8)
})
const lossThreats = computed(() => {
    const pilots = new Map<string, { name: string, corporation: string, losses: number, isk: number }>()
    for (const kill of props.report.engagements.flatMap(engagement => engagement.killmails)) {
        if (kill.role !== 'loss' || kill.friendly_fire || !kill.final_blow_character_name) continue
        const key = `${kill.final_blow_character_name}\u0000${kill.final_blow_corporation_name}`
        const pilot = pilots.get(key) ?? {
            name: kill.final_blow_character_name, corporation: kill.final_blow_corporation_name, losses: 0, isk: 0,
        }
        pilot.losses++
        pilot.isk += kill.total_value
        pilots.set(key, pilot)
    }
    return [...pilots.values()].sort((a, b) => b.losses - a.losses || b.isk - a.isk).slice(0, 8)
})
</script>

<template>
    <div>
        <p class="mb-4 text-xs text-gray-500">Killmail observations for the listed fleet. Command hulls are a ship-class signal, not proof of command; the report cannot identify an FC or ships that never appeared on a killmail.</p>
        <div class="mb-4 rounded-lg border border-blue-500/20 bg-white/[0.04] p-3">
            <div class="mb-2 text-xs font-bold text-blue-400">Fleet · observed roles</div>
            <div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-xs">
                <div class="flex items-center gap-1.5"><Icon name="lucide:crown" class="text-yellow-400" /><span class="text-gray-400">Command hulls</span><strong class="tabular-nums text-gray-200">{{ commandHulls.length }}</strong></div>
                <div class="flex items-center gap-1.5"><Icon name="lucide:heart-pulse" class="text-emerald-400" /><span class="text-gray-400">Logistics</span><strong class="tabular-nums text-gray-200">{{ logistics.length }}</strong></div>
                <div class="flex items-center gap-1.5"><Icon name="lucide:ship" class="text-amber-400" /><span class="text-gray-400">Capitals</span><strong class="tabular-nums text-gray-200">{{ capitals.length }}</strong></div>
                <div class="flex items-center gap-1.5"><Icon name="lucide:users" class="text-blue-400" /><span class="text-gray-400">Active pilots</span><strong class="tabular-nums text-gray-200">{{ report.summary.active_pilots }} / {{ report.roster.length }}</strong></div>
            </div>
        </div>

        <div v-if="specialistSections.length" class="mb-4 grid gap-4 lg:grid-cols-2">
            <section v-for="section in specialistSections" :key="section.title" class="rounded-lg border border-blue-500/20 bg-white/[0.04] p-3">
                <div class="mb-3 flex items-center justify-between border-b border-white/[0.06] pb-2">
                    <h2 class="flex items-center gap-2 text-sm font-bold text-blue-400"><Icon :name="section.icon" /> {{ section.title }}</h2>
                    <span class="text-fine text-gray-500">{{ section.rows.length }} observed</span>
                </div>
                <p class="mb-3 text-fine text-gray-500">{{ section.description }}</p>
                <div class="overflow-x-auto">
                    <table class="w-full min-w-[480px] table-fixed text-xs">
                        <thead class="text-left text-gray-500"><tr class="border-b border-white/[0.06]"><th class="w-[38%] py-1.5 pr-2 font-medium">Pilot</th><th class="w-[32%] px-2 py-1.5 font-medium">Ship</th><th class="px-2 py-1.5 text-right font-medium">Damage</th><th class="py-1.5 pl-2 text-right font-medium">Status</th></tr></thead>
                        <tbody>
                            <tr v-for="item in section.rows" :key="`${item.pilot.character_id}-${item.ship.ship_type_id}`" class="border-b border-white/[0.03] text-gray-300 hover:bg-white/[0.02]">
                                <td class="py-1.5 pr-2"><div class="flex min-w-0 items-center gap-2"><img :src="`/images/characters/${item.pilot.character_id}/portrait?size=32`" :alt="item.pilot.name" class="h-6 w-6 shrink-0 rounded" loading="lazy"><div class="min-w-0"><NuxtLink :to="`/character/${item.pilot.character_id}`" class="block truncate hover:text-blue-400">{{ item.pilot.name }}</NuxtLink><span class="block truncate text-fine text-gray-500">{{ item.pilot.corporation_name }}</span></div></div></td>
                                <td class="px-2 py-1.5"><div class="flex min-w-0 items-center gap-2"><img :src="`/images/types/${item.ship.ship_type_id}/render?size=32`" :alt="item.ship.ship_name" class="h-6 w-6 shrink-0 rounded" loading="lazy"><NuxtLink :to="`/item/${item.ship.ship_type_id}`" class="truncate hover:text-blue-400">{{ item.ship.ship_name }}</NuxtLink></div></td>
                                <td class="px-2 py-1.5 text-right tabular-nums text-green-400">{{ formatNumber(item.ship.damage_done) }}</td>
                                <td class="py-1.5 pl-2 text-right"><span v-if="item.ship.losses" class="rounded bg-red-500/20 px-1.5 py-0.5 text-fine font-medium text-red-400">{{ item.ship.losses }} lost</span><span v-else class="text-fine text-gray-500">Observed</span></td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </section>
        </div>
        <div v-else class="mb-4 rounded-lg border border-white/[0.08] bg-white/[0.04] p-5 text-xs text-gray-500">No command, logistics, or capital hulls were observed in this report.</div>

        <div class="grid gap-4 lg:grid-cols-2">
            <section class="rounded-lg border border-blue-500/20 bg-white/[0.04] p-3">
                <div class="mb-3 flex items-center justify-between border-b border-white/[0.06] pb-2"><h2 class="text-sm font-bold text-blue-400">Fleet contributors</h2><span class="text-fine text-gray-500">{{ topPilots.length }} {{ topPilots.length === 1 ? 'pilot' : 'pilots' }}</span></div>
                <div v-if="topPilots.length" class="overflow-x-auto"><table class="w-full min-w-[470px] table-fixed text-xs"><thead class="text-left text-gray-500"><tr class="border-b border-white/[0.06]"><th class="w-[45%] py-1.5 pr-2 font-medium">Pilot</th><th class="px-2 py-1.5 text-right font-medium">Kills</th><th class="px-2 py-1.5 text-right font-medium">Final</th><th class="px-2 py-1.5 text-right font-medium">Losses</th><th class="py-1.5 pl-2 text-right font-medium">Damage</th></tr></thead><tbody><tr v-for="pilot in topPilots" :key="pilot.character_id" class="border-b border-white/[0.03] text-gray-300 hover:bg-white/[0.02]"><td class="py-1.5 pr-2"><div class="flex min-w-0 items-center gap-2"><img :src="`/images/characters/${pilot.character_id}/portrait?size=32`" :alt="pilot.name" class="h-6 w-6 shrink-0 rounded" loading="lazy"><NuxtLink :to="`/character/${pilot.character_id}`" class="truncate hover:text-blue-400">{{ pilot.name }}</NuxtLink></div></td><td class="px-2 py-1.5 text-right tabular-nums text-green-400">{{ pilot.kill_participations }}</td><td class="px-2 py-1.5 text-right tabular-nums">{{ pilot.final_blows }}</td><td class="px-2 py-1.5 text-right tabular-nums text-red-400">{{ pilot.losses }}</td><td class="py-1.5 pl-2 text-right tabular-nums">{{ formatNumber(pilot.damage_done) }}</td></tr></tbody></table></div>
                <p v-else class="py-5 text-center text-xs text-gray-600">No fleet combat recorded.</p>
            </section>
            <section class="rounded-lg border border-blue-500/20 bg-white/[0.04] p-3">
                <div class="mb-3 flex items-center justify-between border-b border-white/[0.06] pb-2"><h2 class="text-sm font-bold text-blue-400">Opposing corporations</h2><span class="text-fine text-gray-500">{{ report.targets.length }} {{ report.targets.length === 1 ? 'corp' : 'corps' }}</span></div>
                <div v-if="report.targets.length" class="overflow-x-auto"><table class="w-full min-w-[440px] table-fixed text-xs"><thead class="text-left text-gray-500"><tr class="border-b border-white/[0.06]"><th class="w-[55%] py-1.5 pr-2 font-medium">Corporation</th><th class="px-2 py-1.5 text-right font-medium">Kills</th><th class="py-1.5 pl-2 text-right font-medium">ISK destroyed</th></tr></thead><tbody><tr v-for="target in report.targets" :key="target.corporation_id" class="border-b border-white/[0.03] text-gray-300 hover:bg-white/[0.02]"><td class="py-1.5 pr-2"><div class="flex min-w-0 items-center gap-2"><img :src="`/images/corporations/${target.corporation_id}/logo?size=32`" :alt="target.corporation_name" class="h-6 w-6 shrink-0 rounded" loading="lazy"><div class="min-w-0"><NuxtLink :to="`/corporation/${target.corporation_id}`" class="block truncate hover:text-blue-400">{{ target.corporation_name }}</NuxtLink><span class="block truncate text-fine text-gray-500">{{ target.alliance_name }}</span></div></div></td><td class="px-2 py-1.5 text-right tabular-nums">{{ target.kills }}</td><td class="py-1.5 pl-2 text-right tabular-nums text-amber-200">{{ formatIsk(target.isk_destroyed) }}</td></tr></tbody></table></div>
                <p v-else class="py-5 text-center text-xs text-gray-600">No opposing corporations recorded.</p>
            </section>
            <section v-if="destroyedShips.length" class="rounded-lg border border-blue-500/20 bg-white/[0.04] p-3">
                <div class="mb-3 flex items-center justify-between border-b border-white/[0.06] pb-2"><h2 class="text-sm font-bold text-blue-400">Opposing ships destroyed</h2><span class="text-fine text-gray-500">{{ destroyedShips.length }} hull {{ destroyedShips.length === 1 ? 'type' : 'types' }}</span></div>
                <table class="w-full table-fixed text-xs"><thead class="text-left text-gray-500"><tr class="border-b border-white/[0.06]"><th class="w-[60%] py-1.5 pr-2 font-medium">Ship</th><th class="px-2 py-1.5 text-right font-medium">Kills</th><th class="py-1.5 pl-2 text-right font-medium">ISK</th></tr></thead><tbody><tr v-for="ship in destroyedShips" :key="ship.id" class="border-b border-white/[0.03] text-gray-300 hover:bg-white/[0.02]"><td class="py-1.5 pr-2"><div class="flex min-w-0 items-center gap-2"><img :src="`/images/types/${ship.id}/render?size=32`" :alt="ship.name" class="h-6 w-6 shrink-0 rounded" loading="lazy"><NuxtLink :to="`/item/${ship.id}`" class="truncate hover:text-blue-400">{{ ship.name }}</NuxtLink></div></td><td class="px-2 py-1.5 text-right tabular-nums">{{ ship.count }}</td><td class="py-1.5 pl-2 text-right tabular-nums text-amber-200">{{ formatIsk(ship.isk) }}</td></tr></tbody></table>
            </section>
            <section v-if="lossThreats.length" class="rounded-lg border border-blue-500/20 bg-white/[0.04] p-3">
                <div class="mb-3 flex items-center justify-between border-b border-white/[0.06] pb-2"><h2 class="text-sm font-bold text-blue-400">Losses to</h2><span class="text-fine text-gray-500">{{ lossThreats.length }} {{ lossThreats.length === 1 ? 'pilot' : 'pilots' }}</span></div>
                <table class="w-full table-fixed text-xs"><thead class="text-left text-gray-500"><tr class="border-b border-white/[0.06]"><th class="w-[55%] py-1.5 pr-2 font-medium">Pilot</th><th class="px-2 py-1.5 text-right font-medium">Losses</th><th class="py-1.5 pl-2 text-right font-medium">ISK lost</th></tr></thead><tbody><tr v-for="pilot in lossThreats" :key="`${pilot.name}-${pilot.corporation}`" class="border-b border-white/[0.03] text-gray-300 hover:bg-white/[0.02]"><td class="py-1.5 pr-2"><div class="truncate">{{ pilot.name }}</div><div class="truncate text-fine text-gray-500">{{ pilot.corporation }}</div></td><td class="px-2 py-1.5 text-right tabular-nums text-red-400">{{ pilot.losses }}</td><td class="py-1.5 pl-2 text-right tabular-nums">{{ formatIsk(pilot.isk) }}</td></tr></tbody></table>
            </section>
        </div>
    </div>
</template>
