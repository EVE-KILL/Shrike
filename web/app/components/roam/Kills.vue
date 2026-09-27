<script setup lang="ts">
import type { RoamKillmail, RoamReport } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
const filter = ref<'kills' | 'losses' | 'all'>('kills')
const PAGE_SIZE = 100
const visibleCount = ref(PAGE_SIZE)
const allRows = computed<RoamKillmail[]>(() => props.report.engagements
    .flatMap(engagement => engagement.killmails)
    .sort((a, b) => b.killmail_time.localeCompare(a.killmail_time) || b.killmail_id - a.killmail_id))
const filters = computed(() => [
    { id: 'kills' as const, label: `Kills ${props.report.summary.kills}` },
    { id: 'losses' as const, label: `Losses ${props.report.summary.losses}` },
    { id: 'all' as const, label: `All ${allRows.value.length}` },
])
const filteredRows = computed(() => allRows.value.filter(kill => filter.value === 'all' || kill.role === (filter.value === 'kills' ? 'kill' : 'loss')))
const visibleRows = computed(() => filteredRows.value.slice(0, visibleCount.value))
watch(filter, () => { visibleCount.value = PAGE_SIZE })

function targetName(kill: RoamKillmail) {
    return kill.victim_name || kill.victim_corporation_name || 'Unknown target'
}
</script>

<template>
    <section class="glass-panel overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-white/[0.07] p-5">
            <div>
                <h2 class="text-lg font-semibold text-white">Killmails</h2>
                <p class="mt-1 text-sm text-gray-400">Every matching killmail in the selected window, newest first.</p>
            </div>
            <div class="flex rounded-lg border border-white/[0.08] bg-black/20 p-1 text-xs">
                <button v-for="option in filters" :key="option.id" type="button"
                    class="rounded-md px-3 py-1.5 font-medium transition-colors"
                    :class="filter === option.id ? 'bg-blue-500/15 text-blue-300' : 'text-gray-400 hover:text-gray-200'"
                    @click="filter = option.id">
                    {{ option.label }}
                </button>
            </div>
        </div>
        <div v-if="!filteredRows.length" class="p-10 text-center text-sm text-gray-400">No {{ filter === 'all' ? 'killmails' : filter }} recorded in this window.</div>
        <div v-else class="divide-y divide-white/[0.05]">
            <div v-for="kill in visibleRows" :key="kill.killmail_id" class="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-blue-500/[0.05] sm:px-5">
                <div class="relative h-11 w-11 shrink-0 overflow-hidden rounded-md bg-white/[0.04]">
                    <img v-if="kill.victim_ship_type_id" :src="`/images/types/${kill.victim_ship_type_id}/icon?size=64`" :alt="kill.victim_ship_name" class="h-full w-full object-cover" loading="lazy">
                    <Icon v-else name="lucide:rocket" class="m-3 text-gray-500" />
                </div>
                <div class="min-w-0 flex-1">
                    <NuxtLink :to="`/kill/${kill.killmail_id}`" class="block truncate text-sm font-medium text-gray-100 hover:text-blue-300">{{ targetName(kill) }}</NuxtLink>
                    <p class="truncate text-xs text-gray-500">{{ kill.victim_ship_name || 'Unknown ship' }} · {{ kill.victim_corporation_name || 'Unknown corporation' }}</p>
                    <p class="mt-0.5 truncate text-xs text-gray-500 lg:hidden">{{ kill.solar_system_name }} · {{ formatEveDateTime(kill.killmail_time, true) }}</p>
                </div>
                <div class="hidden w-36 min-w-0 shrink-0 lg:block">
                    <NuxtLink :to="`/system/${kill.solar_system_id}`" class="block truncate text-sm text-gray-300 hover:text-blue-300">{{ kill.solar_system_name }}</NuxtLink>
                    <p class="truncate text-xs text-gray-500">{{ kill.region_name }}</p>
                </div>
                <div class="hidden w-40 min-w-0 shrink-0 xl:block">
                    <p class="truncate text-xs text-gray-300">{{ kill.final_blow_character_name || 'Final blow unknown' }}</p>
                    <p class="truncate text-xs text-gray-500">{{ formatEveDateTime(kill.killmail_time, true) }}</p>
                </div>
                <div class="w-20 shrink-0 text-right sm:w-24">
                    <p class="text-sm font-medium tabular-nums text-gray-200">{{ formatIsk(kill.total_value) }}</p>
                    <p class="mt-1 text-xs font-semibold uppercase" :class="kill.role === 'kill' ? 'text-emerald-300' : 'text-rose-300'">{{ kill.role }}</p>
                    <p v-if="kill.friendly_fire" class="text-xs text-amber-300">Friendly fire</p>
                </div>
            </div>
        </div>
        <div v-if="filteredRows.length > visibleRows.length" class="border-t border-white/[0.07] p-4 text-center">
            <button type="button" class="rounded-lg border border-white/10 px-4 py-2 text-sm text-blue-300 hover:bg-white/5" @click="visibleCount += PAGE_SIZE">Show more · {{ visibleRows.length }} of {{ filteredRows.length }}</button>
        </div>
    </section>
</template>
