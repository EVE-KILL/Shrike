<script setup lang="ts">
import { battleLossBuckets } from '~/utils/battleExplorer'
import type { BattleSide } from '~/utils/battleExplorer'
import { bigKillClass, replaySide } from '~/utils/map/replay'
import type { ReplayKill } from '~/utils/map/replay'
const props = defineProps<{ endpoint: string; mode: 'summary' | 'timeline' | 'kills'; startTime: string; endTime: string; teamEntities: BattleSide[] }>()
const { data, error, refresh } = await useApiFetch<{ kills: ReplayKill[] }>(props.endpoint)
const kills = computed(() => data.value?.kills ?? [])
const { filters, filtered, setFilter, clear, location, replayLocation } = useBattleExplorer(kills, () => props.teamEntities)
const start = computed(() => Date.parse(props.startTime)), end = computed(() => Date.parse(props.endTime))
const buckets = computed(() => battleLossBuckets(filtered.value, props.teamEntities, start.value, end.value))
const metric = ref<'count' | 'isk'>('count')
const peak = computed(() => Math.max(1, ...buckets.value.map(bin => bin[metric.value])))
const busiest = computed(() => buckets.value.reduce((best, bin) => bin.count > best.count ? bin : best, buckets.value[0]!))
const biggest = computed(() => [...filtered.value].sort((a, b) => b.total_value - a.total_value).slice(0, 6))
const superLosses = computed(() => filtered.value.filter(kill => bigKillClass(kill)))
const totalIsk = computed(() => filtered.value.reduce((total, kill) => total + kill.total_value, 0))
const sideName = (side: number) => side < 0 ? 'Unassigned' : `Team ${String.fromCharCode(65 + side)}`
const sideColor = (side: number) => ['#f87171', '#60a5fa', '#c084fc', '#fbbf24'][side] ?? '#94a3b8'
const eveTime = (time: number) => new Date(time).toISOString().slice(5, 16).replace('T', ' ') + ' EVE'
const groups = computed(() => [...new Map(kills.value.filter(kill => kill.ship_group_id).map(kill => [kill.ship_group_id!, kill.ship_group_name || `Class ${kill.ship_group_id}`])).entries()].sort((a,b) => a[1].localeCompare(b[1])))
const intervalRows = computed(() => buckets.value.filter(bin => bin.count).sort((a,b) => b.start - a.start))
const router = useRouter(), route = useRoute()
function chooseWindow(bin: typeof buckets.value[number]) { router.replace({ query: { ...route.query, from: String(bin.start), to: String(bin.end) } }) }
</script>
<template>
    <section class="space-y-4" :aria-label="`${mode} loss explorer`">
        <div v-if="error" role="alert" class="glass-panel p-5 text-red-300">Unable to load battle losses. <button class="underline" @click="refresh()">Retry</button></div>
        <template v-else>
            <div v-if="mode === 'summary' && (filters.side != null || filters.group != null || filters.minIsk || filters.from != null || filters.to != null)" class="glass-panel flex items-center gap-3 p-3 text-xs text-sky-300">Highlights for your selected losses. The team breakdown below covers the full battle.<button class="ml-auto underline" @click="clear">Full battle highlights</button></div>
            <div v-if="mode !== 'summary'" class="glass-panel flex flex-wrap items-center gap-3 p-3">
                <label class="text-xs text-gray-400">Victim side <select :value="filters.side ?? ''" class="ml-1 rounded bg-[#141414] p-2" @change="setFilter('side', ($event.target as HTMLSelectElement).value)"><option value="">All sides</option><option v-for="(_, i) in teamEntities" :key="i" :value="i">{{ sideName(i) }}</option><option value="-1">Unassigned</option></select></label>
                <label class="text-xs text-gray-400">Ship class <select :value="filters.group ?? ''" class="ml-1 max-w-52 rounded bg-[#141414] p-2" @change="setFilter('group', ($event.target as HTMLSelectElement).value)"><option value="">All classes</option><option v-for="[id,name] in groups" :key="id" :value="id">{{ name }}</option></select></label>
                <label class="text-xs text-gray-400">Minimum ISK <select :value="filters.minIsk" class="ml-1 rounded bg-[#141414] p-2" @change="setFilter('minIsk', ($event.target as HTMLSelectElement).value)"><option value="0">Any value</option><option v-for="value in [1e8,1e9,1e10,1e11]" :key="value" :value="value">{{ formatIsk(value) }}+</option></select></label>
                <button class="ml-auto text-xs text-blue-300" @click="clear">Reset filters</button>
                <div v-if="filters.from != null || filters.to != null" class="flex w-full items-center gap-2 text-xs text-sky-300"><Icon name="lucide:clock" />{{ eveTime(filters.from ?? start) }} → {{ eveTime(filters.to ?? end) }}<button class="ml-2 underline" @click="router.replace({ query: { ...route.query, from: undefined, to: undefined } })">Full battle</button></div>
            </div>
            <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
                <div class="glass-panel p-4"><div class="text-[10px] uppercase tracking-wider text-gray-500">Recorded losses</div><div class="mt-1 font-mono text-xl text-gray-100">{{ formatNumber(filtered.length) }}</div></div>
                <div class="glass-panel p-4"><div class="text-[10px] uppercase tracking-wider text-gray-500">ISK lost</div><div class="mt-1 font-mono text-xl text-amber-200">{{ formatIsk(totalIsk) }}</div></div>
                <NuxtLink :to="location('timeline', { group: undefined })" class="glass-panel p-4 hover:border-amber-400/30"><div class="text-[10px] uppercase tracking-wider text-gray-500">Titans / supercarriers lost</div><div class="mt-1 font-mono text-xl text-amber-300">{{ superLosses.length }}</div></NuxtLink>
                <NuxtLink v-if="busiest" :to="location('replay', { at: busiest.start, kill: undefined })" class="glass-panel p-4 hover:border-sky-400/30"><div class="text-[10px] uppercase tracking-wider text-gray-500">Busiest interval · replay ↗</div><div class="mt-1 text-sm text-sky-200">{{ eveTime(busiest.start) }}</div><div class="text-xs text-gray-500">{{ busiest.count }} losses</div></NuxtLink>
            </div>
            <p class="text-xs text-gray-500">Losses include capsules and unassigned pilots. All values describe recorded killmails.</p>
            <template v-if="mode === 'summary'">
                <div class="flex items-center justify-between"><h3 class="text-sm font-medium text-gray-300">Most expensive losses</h3><NuxtLink :to="location('timeline')" class="text-xs text-sky-300">Explore the timeline →</NuxtLink></div>
                <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                    <div v-for="kill in biggest" :key="kill.killmail_id" class="glass-panel flex items-center gap-3 p-3">
                        <img :src="`/images/types/${kill.ship_type_id}/icon?size=64`" alt="" class="h-10 w-10 rounded" loading="lazy">
                        <div class="min-w-0 flex-1"><NuxtLink :to="`/kill/${kill.killmail_id}`" class="block truncate text-sm text-gray-200">{{ kill.ship_name }}</NuxtLink><div class="text-xs text-amber-200">{{ formatIsk(kill.total_value) }} ISK</div><NuxtLink :to="replayLocation(kill)" class="text-xs text-sky-300">Show in replay →</NuxtLink></div>
                    </div>
                </div>
            </template>
            <template v-else-if="mode === 'timeline'">
                <div class="glass-panel p-4">
                    <div class="mb-4 flex flex-wrap items-center justify-between gap-3"><h3 class="text-sm text-gray-300">How the battle unfolded</h3><label class="text-xs text-gray-400">Measure <select v-model="metric" class="ml-1 rounded bg-[#141414] p-2"><option value="count">Losses</option><option value="isk">ISK lost</option></select></label></div>
                    <div class="mb-3 flex flex-wrap gap-4 text-xs"><span v-for="(_, i) in [...teamEntities, null]" :key="i" :style="{ color: sideColor(i === teamEntities.length ? -1 : i) }">● {{ sideName(i === teamEntities.length ? -1 : i) }}</span><span class="text-amber-300">┃ Titan / supercarrier loss</span></div>
                    <div class="flex h-36 items-end gap-1">
                        <button v-for="bin in buckets" :key="bin.start" class="relative flex h-full min-w-0 flex-1 flex-col justify-end hover:brightness-150" :title="`${eveTime(bin.start)} · ${bin.count} losses · ${formatIsk(bin.isk)} ISK`" :aria-label="`Filter interval ${eveTime(bin.start)}: ${bin.count} losses`" @click="chooseWindow(bin)">
                            <span v-if="bin.big" class="absolute inset-y-0 left-1/2 z-10 w-0.5 bg-amber-300" />
                            <span v-for="(side, i) in bin.sides" :key="i" class="block w-full" :style="{height: `${side[metric] / peak * 100}%`, background: sideColor(i === teamEntities.length ? -1 : i)}" />
                        </button>
                    </div>
                    <div class="mt-2 flex justify-between text-[10px] text-gray-500"><span>{{ eveTime(start) }}</span><span>Click an interval to filter</span><span>{{ eveTime(end) }}</span></div>
                </div>
                <div v-if="superLosses.length" class="glass-panel p-4"><h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-amber-300">Titan and supercarrier losses</h3><div class="max-h-64 space-y-2 overflow-y-auto"><div v-for="kill in superLosses" :key="kill.killmail_id" class="flex flex-wrap items-center gap-3 border-b border-white/5 pb-2 text-xs"><span class="font-mono text-gray-500">{{ eveTime(Date.parse(kill.killmail_time)) }}</span><NuxtLink :to="`/kill/${kill.killmail_id}`" class="text-amber-100">{{ kill.ship_name }} · {{ formatIsk(kill.total_value) }}</NuxtLink><span :style="{color: sideColor(replaySide(kill, teamEntities))}">{{ sideName(replaySide(kill, teamEntities)) }}</span><NuxtLink :to="replayLocation(kill)" class="ml-auto text-sky-300">Show in replay →</NuxtLink></div></div></div>
                <div class="space-y-2"><div v-for="bin in intervalRows" :key="bin.start" class="glass-panel flex flex-wrap items-center gap-4 px-4 py-3 text-xs"><span class="font-mono text-gray-400">{{ eveTime(bin.start) }}</span><span class="text-gray-200">{{ bin.count }} losses</span><span class="text-amber-200">{{ formatIsk(bin.isk) }} ISK</span><NuxtLink :to="location('kills', {from: bin.start, to: bin.end})" class="ml-auto text-sky-300">Inspect losses</NuxtLink><NuxtLink :to="location('replay', {at: bin.start, kill: undefined})" class="text-sky-300">Replay →</NuxtLink></div></div>
            </template>
            <template v-else>
                <slot name="killlist" :filters="filters" />
            </template>
            <p v-if="!filtered.length" class="glass-panel p-6 text-center text-sm text-gray-500">No losses match these filters.</p>
        </template>
    </section>
</template>
