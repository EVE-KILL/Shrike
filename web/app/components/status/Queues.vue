<script setup lang="ts">
defineProps<{ data: any }>()
</script>

<template>
    <div class="glass-panel p-4">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Queues</div>
            <div class="flex items-center gap-3 text-xs">
                <span class="text-gray-500">ESI Budget</span>
                <span class="font-medium tabular-nums" :class="(data.esi?.error_budget || 0) > 50 ? 'text-green-400' : 'text-red-400'">{{ data.esi?.error_budget || 0 }}%</span>
                <span v-if="data.esi?.paused" class="px-1.5 py-0.5 rounded bg-red-500/20 text-red-400 font-medium">PAUSED</span>
            </div>
        </div>
        <div class="space-y-1.5">
            <div v-for="(q, name) in (data.queues as Record<string, any>)" :key="name"
                class="flex items-center justify-between text-sm">
                <span class="text-gray-300">{{ name }}</span>
                <div class="flex items-center gap-3 tabular-nums">
                    <span class="text-amber-400" :class="{ 'text-gray-600': !q.waiting }">{{ formatNumber(q.waiting) }} <span class="text-fine text-gray-500">wait</span></span>
                    <span class="text-green-400" :class="{ 'text-gray-600': !q.active }">{{ q.active }} <span class="text-fine text-gray-500">active</span></span>
                    <span v-if="q.failed" class="text-red-400">{{ q.failed }} <span class="text-fine text-gray-500">fail</span></span>
                </div>
            </div>
        </div>
    </div>
</template>
