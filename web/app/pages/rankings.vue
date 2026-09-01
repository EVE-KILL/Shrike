<script setup lang="ts">
const entityTypes = [
    ['character', 'Characters'], ['corporation', 'Corporations'], ['alliance', 'Alliances'],
    ['ship', 'Ships'], ['system', 'Systems'], ['region', 'Regions'],
] as const
const windows = [
    ['weekly', 'Weekly'], ['ninety_days', '90 Days'], ['all_time', 'All Time'],
] as const

const entityType = ref('character')
const rankingWindow = ref('all_time')
const { data, pending } = await useAsyncData(
    'eve-kill-ranking-explorer',
    () => $fetch('/api/stats/rankings', {
      query: {
        section: 'eve-kill', entityType: entityType.value,
        window: rankingWindow.value, limit: 50,
      },
    }),
    { watch: [entityType, rankingWindow] },
)
const entries = computed<any[]>(() => (data.value as any)?.entries || [])

function entityLink(entry: any) {
    const paths: Record<string, string> = {
        character: 'character', corporation: 'corporation', alliance: 'alliance',
        ship: 'item', system: 'system', region: 'region',
    }
    return `/${paths[entityType.value]}/${entry.entity_id}`
}

useSeoMeta({
    title: 'EVE-KILL Rankings',
    description: 'Explore weekly, 90-day, and all-time EVE-KILL combat rankings.',
})
</script>

<template>
    <main class="max-w-6xl mx-auto px-4 py-6 space-y-5">
        <div>
            <h1 class="text-2xl font-bold text-white">EVE-KILL Rankings</h1>
            <p class="text-sm text-gray-500 mt-1">Combat points are split by damage and participation. Character ratings add an achievement bonus worth up to 30% of the combined rating.</p>
        </div>

        <div class="flex flex-wrap gap-2 justify-between">
            <div class="flex flex-wrap gap-1.5">
                <button v-for="item in entityTypes" :key="item[0]" type="button"
                    class="px-3 py-1.5 rounded border text-xs transition-colors"
                    :class="entityType === item[0] ? 'border-blue-500/40 bg-blue-500/10 text-blue-300' : 'border-white/10 text-gray-400 hover:text-white'"
                    @click="entityType = item[0]">{{ item[1] }}</button>
            </div>
            <div class="flex gap-1.5">
                <button v-for="item in windows" :key="item[0]" type="button"
                    class="px-3 py-1.5 rounded border text-xs transition-colors"
                    :class="rankingWindow === item[0] ? 'border-purple-500/40 bg-purple-500/10 text-purple-300' : 'border-white/10 text-gray-400 hover:text-white'"
                    @click="rankingWindow = item[0]">{{ item[1] }}</button>
            </div>
        </div>

        <div class="rounded-lg border border-white/10 bg-black/20 overflow-hidden">
            <div class="grid grid-cols-[56px_1fr_100px_120px_100px] gap-3 px-4 py-2 border-b border-white/10 text-fine uppercase tracking-wider text-gray-600">
                <span>Rank</span><span>Entity</span><span class="text-right">Rating</span>
                <span class="text-right">Combat Points</span><span class="text-right">Achievements</span>
            </div>
            <div v-if="pending" class="p-8 text-center text-sm text-gray-500">Loading rankings…</div>
            <NuxtLink v-for="entry in entries" v-else :key="entry.entity_id" :to="entityLink(entry)"
                class="grid grid-cols-[56px_1fr_100px_120px_100px] gap-3 items-center px-4 py-2.5 border-b border-white/[0.05] last:border-0 hover:bg-white/[0.03] transition-colors">
                <span class="font-mono text-sm text-gray-500">#{{ entry.overall_rank }}</span>
                <span class="text-sm text-gray-200 truncate">{{ entry.name }}</span>
                <span class="text-right font-bold tabular-nums text-blue-300">{{ entry.eve_kill_rating.toLocaleString('en-US') }}</span>
                <span class="text-right text-xs tabular-nums text-gray-400">{{ entry.combat_points.toLocaleString('en-US') }}</span>
                <span class="text-right text-xs tabular-nums text-purple-400">{{ entry.achievement_points.toLocaleString('en-US') }}</span>
            </NuxtLink>
            <div v-if="!pending && entries.length === 0" class="p-8 text-center text-sm text-gray-500">Rankings are being generated.</div>
        </div>
    </main>
</template>
