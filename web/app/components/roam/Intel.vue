<script setup lang="ts">
import type { RoamPilot, RoamReport, RoamShip } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
const topPilots = computed(() => [...props.report.roster]
    .filter(pilot => pilot.kill_participations || pilot.losses)
    .sort((a, b) => b.kill_participations - a.kill_participations || b.damage_done - a.damage_done)
    .slice(0, 12))
const topTarget = computed(() => props.report.targets[0])
const targetShare = computed(() => topTarget.value && props.report.summary.kills
    ? Math.round(topTarget.value.kills / props.report.summary.kills * 100) : 0)
const specialistShips = computed(() => {
    const found: Array<{ pilot: RoamPilot, ship: RoamShip, role: string }> = []
    for (const pilot of props.report.roster) {
        for (const ship of pilot.ships ?? []) {
            const group = ship.ship_group_name.toLowerCase()
            const role = /logistics|force auxiliary/.test(group) ? 'Logistics'
                : /carrier|dreadnought|titan/.test(group) ? 'Capital' : ''
            if (role) found.push({ pilot, ship, role })
        }
    }
    return found
})
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
    <div class="space-y-6">
        <p class="text-sm text-gray-400">Observed combat signals from the listed pilots and their opponents. Roles and fleet command cannot be confirmed from public killmails.</p>
        <div class="grid gap-4 md:grid-cols-3">
            <div class="glass-panel p-5">
                <p class="text-xs font-semibold uppercase tracking-wider text-blue-300">Coordination</p>
                <p class="mt-2 text-2xl font-semibold text-white tabular-nums">{{ report.summary.coordinated_kills }}</p>
                <p class="mt-1 text-sm text-gray-400">kills had two or more listed pilots</p>
            </div>
            <div class="glass-panel p-5">
                <p class="text-xs font-semibold uppercase tracking-wider text-amber-300">Target concentration</p>
                <p class="mt-2 text-2xl font-semibold text-white tabular-nums">{{ targetShare }}%</p>
                <p class="mt-1 text-sm text-gray-400">of kills were against {{ topTarget?.corporation_name || 'one corporation' }}</p>
            </div>
            <div class="glass-panel p-5">
                <p class="text-xs font-semibold uppercase tracking-wider text-emerald-300">ISK efficiency</p>
                <p class="mt-2 text-2xl font-semibold text-white tabular-nums">{{ report.summary.efficiency.toFixed(2) }}%</p>
                <p class="mt-1 text-sm text-gray-400">{{ formatIsk(report.summary.isk_destroyed) }} destroyed / {{ formatIsk(report.summary.isk_lost) }} lost</p>
            </div>
        </div>

        <div class="grid gap-6 xl:grid-cols-3">
            <section class="glass-panel overflow-hidden xl:col-span-2">
                <div class="border-b border-white/[0.07] p-5">
                    <h2 class="text-lg font-semibold text-white">Fleet contributors</h2>
                    <p class="mt-1 text-xs text-gray-500">Ranked by killmail participation in this report</p>
                </div>
                <div v-if="topPilots.length" class="divide-y divide-white/[0.05]">
                    <div v-for="(pilot, index) in topPilots" :key="pilot.character_id" class="flex items-center gap-3 px-5 py-3">
                        <span class="w-6 shrink-0 text-xs tabular-nums text-gray-500">{{ index + 1 }}</span>
                        <img :src="`/images/characters/${pilot.character_id}/portrait?size=64`" :alt="pilot.name" class="h-9 w-9 rounded-full bg-gray-900" loading="lazy">
                        <div class="min-w-0 flex-1">
                            <NuxtLink :to="`/character/${pilot.character_id}`" class="block truncate text-sm text-gray-100 hover:text-blue-300">{{ pilot.name }}</NuxtLink>
                            <p class="truncate text-xs text-gray-500">{{ pilot.corporation_name || 'Unknown corporation' }}<template v-if="pilot.alliance_name"> · {{ pilot.alliance_name }}</template></p>
                        </div>
                        <div class="shrink-0 text-right text-xs tabular-nums text-gray-400">
                            <p>{{ pilot.kill_participations }} K · {{ pilot.final_blows }} final blows · {{ pilot.losses }} L</p>
                            <p class="mt-1">{{ formatNumber(pilot.damage_done) }} damage · {{ pilot.engagements }} fights</p>
                        </div>
                    </div>
                </div>
                <p v-else class="p-6 text-sm text-gray-500">No pilots have recorded combat in this window.</p>
            </section>
            <section class="glass-panel overflow-hidden">
                <div class="border-b border-white/[0.07] p-5">
                    <h2 class="text-lg font-semibold text-white">Opposing corporations</h2>
                    <p class="mt-1 text-xs text-gray-500">Corporations of ships this fleet destroyed</p>
                </div>
                <div v-if="report.targets.length" class="divide-y divide-white/[0.05]">
                    <div v-for="target in report.targets" :key="target.corporation_id" class="flex items-center gap-3 px-5 py-3">
                        <img :src="`/images/corporations/${target.corporation_id}/logo?size=64`" :alt="target.corporation_name" class="h-9 w-9 rounded bg-gray-900" loading="lazy">
                        <div class="min-w-0 flex-1">
                            <NuxtLink :to="`/corporation/${target.corporation_id}`" class="block truncate text-sm text-gray-100 hover:text-blue-300">{{ target.corporation_name || `Corporation ${target.corporation_id}` }}</NuxtLink>
                            <p class="truncate text-xs text-gray-500">{{ target.alliance_name || 'No alliance recorded' }}</p>
                        </div>
                        <div class="shrink-0 text-right text-xs tabular-nums text-gray-400">
                            <p>{{ target.kills }} kills</p>
                            <p class="mt-1">{{ formatIsk(target.isk_destroyed) }}</p>
                        </div>
                    </div>
                </div>
                <p v-else class="p-6 text-sm text-gray-500">No opposing corporations recorded.</p>
            </section>
        </div>

        <div v-if="destroyedShips.length || lossThreats.length" class="grid gap-6 md:grid-cols-2">
            <section v-if="destroyedShips.length" class="glass-panel overflow-hidden">
                <div class="border-b border-white/[0.07] p-5">
                    <h2 class="text-lg font-semibold text-white">Opposing ships destroyed</h2>
                    <p class="mt-1 text-xs text-gray-500">Most common hulls on the fleet's kills</p>
                </div>
                <div class="divide-y divide-white/[0.05]">
                    <div v-for="ship in destroyedShips" :key="ship.id" class="flex items-center gap-3 px-5 py-2.5">
                        <img :src="`/images/types/${ship.id}/render?size=64`" :alt="ship.name" class="h-9 w-9 rounded bg-gray-900" loading="lazy">
                        <NuxtLink :to="`/item/${ship.id}`" class="min-w-0 flex-1 truncate text-sm text-gray-100 hover:text-blue-300">{{ ship.name }}</NuxtLink>
                        <span class="shrink-0 text-xs tabular-nums text-gray-400">{{ ship.count }} killed · {{ formatIsk(ship.isk) }}</span>
                    </div>
                </div>
            </section>
            <section v-if="lossThreats.length" class="glass-panel overflow-hidden">
                <div class="border-b border-white/[0.07] p-5">
                    <h2 class="text-lg font-semibold text-white">Losses to</h2>
                    <p class="mt-1 text-xs text-gray-500">Recorded final blows against the listed pilots</p>
                </div>
                <div class="divide-y divide-white/[0.05]">
                    <div v-for="pilot in lossThreats" :key="`${pilot.name}-${pilot.corporation}`" class="flex items-center gap-3 px-5 py-3">
                        <div class="min-w-0 flex-1">
                            <p class="truncate text-sm text-gray-100">{{ pilot.name }}</p>
                            <p class="truncate text-xs text-gray-500">{{ pilot.corporation || 'Unknown corporation' }}</p>
                        </div>
                        <span class="shrink-0 text-xs tabular-nums text-gray-400">{{ pilot.losses }} {{ pilot.losses === 1 ? 'loss' : 'losses' }} · {{ formatIsk(pilot.isk) }}</span>
                    </div>
                </div>
            </section>
        </div>

        <section v-if="specialistShips.length" class="glass-panel p-5">
            <h2 class="text-lg font-semibold text-white">Specialist hulls observed</h2>
            <p class="mt-1 text-xs text-gray-500">Ship groups visible on matching killmails; this does not establish a pilot's fleet role.</p>
            <div class="mt-4 flex flex-wrap gap-2">
                <span v-for="item in specialistShips" :key="`${item.pilot.character_id}-${item.ship.ship_type_id}`" class="rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2 text-xs text-gray-300">
                    <span class="font-medium text-blue-300">{{ item.role }}</span> · {{ item.pilot.name }} · {{ item.ship.ship_name }}
                </span>
            </div>
        </section>
    </div>
</template>
