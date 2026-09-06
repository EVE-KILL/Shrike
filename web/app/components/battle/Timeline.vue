<script setup lang="ts">
const props = defineProps<{
    battleId: number | null
    solarSystemId: number
    startTime: string
    endTime: string
    teams: any[]
    teamEntities: { corps: number[]; alliances: number[] }[]
    timelineEndpoint?: string
}>()
const endpoint = computed(() => (props.timelineEndpoint || `/api/battle/${props.battleId}/timeline`).replace('/timeline', '/replay'))
</script>
<template>
    <Suspense>
        <BattleLossExplorer mode="timeline" :endpoint="endpoint" :start-time="startTime" :end-time="endTime" :team-entities="teamEntities" />
        <template #fallback><div role="status" class="glass-panel p-10 text-center text-gray-400">Loading battle losses…</div></template>
    </Suspense>
</template>
