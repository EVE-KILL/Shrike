<script setup lang="ts">
import type { RoamReport } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
const filter = ref<'kills' | 'losses' | 'all'>('kills')
const options = computed(() => [
    { id: 'kills' as const, label: 'Kills', count: props.report.summary.kills },
    { id: 'losses' as const, label: 'Losses', count: props.report.summary.losses },
    { id: 'all' as const, label: 'All', count: props.report.summary.kills + props.report.summary.losses },
])
const endpoint = `/api/tools/roam-report/${props.report.id}/killlist`
const params = computed(() => ({ role: filter.value }))
const lossEntities = computed(() => ({ characterIds: props.report.roster.map(pilot => pilot.character_id) }))
</script>

<template>
    <div>
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
                <h2 class="text-lg font-semibold text-white">Killmails</h2>
                <p class="mt-1 text-sm text-gray-400">The report's recorded kills and losses in the regular killlist.</p>
            </div>
            <div class="flex rounded-lg border border-white/[0.08] bg-white/[0.04] p-1 text-xs">
                <button v-for="option in options" :key="option.id" type="button"
                    class="rounded-md px-3 py-1.5 font-medium transition-colors"
                    :class="filter === option.id ? 'bg-blue-500/15 text-blue-300' : 'text-gray-400 hover:text-gray-200'"
                    @click="filter = option.id">
                    {{ option.label }} <span class="tabular-nums text-gray-500">{{ option.count }}</span>
                </button>
            </div>
        </div>
        <Suspense>
            <LazyKillList :key="filter" :api-endpoint="endpoint" :extra-params="params" :loss-entities="lossEntities" :cache-payload="false" />
            <template #fallback><div role="status" class="glass-panel p-8 text-center text-gray-400">Loading killlist…</div></template>
        </Suspense>
    </div>
</template>
