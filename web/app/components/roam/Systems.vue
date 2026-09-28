<script setup lang="ts">
import type { BattleMapSystem } from '~/utils/map/battles'
import type { RoamReport } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
const selectedSystem = ref<number | null>(null)
const mapScope = ref('new-eden')
const mapRegion = ref<number | null>(null)
const mapScopes = [
    { id: 'new-eden', label: 'New Eden' },
    { id: 'wormhole', label: 'Wormhole' },
    { id: 'zarzakh', label: 'Zarzakh' },
    { id: 'abyssal', label: 'Abyssal' },
    { id: 'proving', label: 'Proving' },
]
const orderedSystems = computed(() => [...props.report.systems].sort((a, b) =>
    a.first_seen.localeCompare(b.first_seen) || a.solar_system_id - b.solar_system_id))
const mapSystems = computed<BattleMapSystem[]>(() => orderedSystems.value.map(system => ({
    solar_system_id: system.solar_system_id,
    solar_system_name: system.solar_system_name,
    region_id: system.region_id || null,
    region_name: system.region_name,
    battle_count: system.engagements,
    kill_count: system.kills + system.losses,
    total_isk_destroyed: system.isk_destroyed + system.isk_lost,
})))
const selectedName = computed(() => orderedSystems.value.find(system => system.solar_system_id === selectedSystem.value)?.solar_system_name)
const visibleEngagements = computed(() => props.report.engagements.filter(engagement =>
    selectedSystem.value === null || engagement.solar_system_id === selectedSystem.value))
watch(() => props.report, () => {
    selectedSystem.value = null
    mapRegion.value = null
})

function selectMapSystem(system: BattleMapSystem) {
    selectedSystem.value = system.solar_system_id
    nextTick(() => document.getElementById('roam-system-order')?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}
</script>

<template>
    <div class="space-y-6">
        <p class="text-sm text-gray-400">Systems are ordered by their first recorded kill or loss. Repeated visits appear in the engagement trail below.</p>
        <div v-if="!orderedSystems.length" class="glass-panel p-10 text-center text-sm text-gray-400">No systems with recorded combat in this time window.</div>
        <template v-else>
            <div class="grid gap-6 xl:grid-cols-3">
                <section class="glass-panel overflow-hidden p-4 sm:p-5 xl:col-span-2" aria-label="Roam combat map">
                    <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
                        <div>
                            <h2 class="text-lg font-semibold text-white">Combat map</h2>
                            <p class="mt-1 text-xs text-gray-500">Markers show systems with recorded kills or losses.</p>
                        </div>
                        <select v-model="mapScope" aria-label="Map scope" class="rounded-lg border border-white/10 bg-[#141414] px-3 py-2 text-sm text-gray-200" @change="mapRegion = null">
                            <option v-for="scope in mapScopes" :key="scope.id" :value="scope.id">{{ scope.label }}</option>
                        </select>
                    </div>
                    <button v-if="mapRegion" type="button" class="mb-3 text-xs text-blue-300 hover:underline" @click="mapRegion = null">← All regions</button>
                    <ClientOnly>
                        <MapPixiScopeView
                            :key="mapScope"
                            :type="mapScope"
                            base-layer="geography"
                            activity-layer="none"
                            :hours="24"
                            :show-connections="true"
                            :show-systems="true"
                            :show-labels="true"
                            :battle-systems="mapSystems"
                            :battle-region-id="mapRegion"
                            marker-noun="engagements"
                            marker-value-label="ISK involved"
                            @battle-region="mapRegion = $event"
                            @battle-system="selectMapSystem"
                        />
                    </ClientOnly>
                </section>
                <section id="roam-system-order" class="glass-panel scroll-mt-6 p-5">
                    <h2 class="text-lg font-semibold text-white">Systems in combat order</h2>
                    <p class="mt-1 text-xs text-gray-500">First recorded killmail sets the order</p>
                    <div class="mt-4 max-h-[35rem] space-y-2 overflow-y-auto pr-1">
                        <button v-for="(system, index) in orderedSystems" :key="system.solar_system_id" type="button"
                            class="flex w-full items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-white/[0.04]"
                            :class="selectedSystem === system.solar_system_id ? 'border-blue-500/40 bg-blue-500/[0.08]' : 'border-white/[0.06]'"
                            @click="selectedSystem = selectedSystem === system.solar_system_id ? null : system.solar_system_id">
                            <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-blue-500/10 text-xs font-semibold text-blue-300">{{ index + 1 }}</span>
                            <span class="min-w-0 flex-1">
                                <span class="block truncate text-sm font-medium text-gray-100">{{ system.solar_system_name }}</span>
                                <span class="block truncate text-xs text-gray-500">{{ system.region_name }} · {{ formatEveDateTime(system.first_seen, true) }}</span>
                            </span>
                            <span class="shrink-0 text-xs tabular-nums text-gray-400">{{ system.kills }} K / {{ system.losses }} L</span>
                        </button>
                    </div>
                </section>
            </div>

            <section class="scroll-mt-6">
                <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
                    <div>
                        <h2 class="text-lg font-semibold text-white">Engagement trail</h2>
                        <p class="mt-1 text-sm text-gray-400">A new stretch begins after a system change or a 30-minute pause.</p>
                    </div>
                    <button v-if="selectedSystem !== null" type="button" class="text-sm text-blue-300 hover:underline" @click="selectedSystem = null">{{ selectedName }} · clear filter</button>
                </div>
                <div class="space-y-3">
                    <article v-for="engagement in visibleEngagements" :key="engagement.number" class="glass-panel flex flex-wrap items-center gap-4 p-5">
                        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-blue-500/10 text-sm font-bold text-blue-300">{{ engagement.number }}</span>
                        <div class="min-w-0 flex-1">
                            <NuxtLink :to="`/system/${engagement.solar_system_id}`" class="text-base font-semibold text-white hover:text-blue-300">{{ engagement.solar_system_name }}</NuxtLink>
                            <p class="mt-1 text-xs text-gray-500">{{ engagement.region_name }} · {{ formatEveDateTime(engagement.start_time, true) }}<template v-if="engagement.end_time !== engagement.start_time"> – {{ formatEveTime(engagement.end_time) }} EVE</template></p>
                        </div>
                        <div class="text-right text-sm tabular-nums">
                            <p><span class="text-emerald-300">{{ engagement.kills }} {{ engagement.kills === 1 ? 'kill' : 'kills' }}</span> · <span class="text-rose-300">{{ engagement.losses }} {{ engagement.losses === 1 ? 'loss' : 'losses' }}</span></p>
                            <p class="mt-1 text-xs text-gray-500">{{ formatIsk(engagement.isk_destroyed) }} destroyed · {{ engagement.pilot_ids.length }} {{ engagement.pilot_ids.length === 1 ? 'pilot' : 'pilots' }} observed</p>
                        </div>
                    </article>
                </div>
            </section>
        </template>
    </div>
</template>
