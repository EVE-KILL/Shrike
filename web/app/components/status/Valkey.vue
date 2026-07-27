<script setup lang="ts">
defineProps<{ data: any }>()

const formatUptime = (seconds: number): string => {
    const d = Math.floor(seconds / 86400)
    const h = Math.floor((seconds % 86400) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = Math.floor(seconds % 60)
    const parts = []
    if (d) parts.push(`${d}d`)
    if (h) parts.push(`${h}h`)
    if (m) parts.push(`${m}m`)
    parts.push(`${s}s`)
    return parts.join(' ')
}

const hitRatio = (r: any) => {
    const hits = r?.keyspace_hits || 0
    const misses = r?.keyspace_misses || 0
    if (hits + misses === 0) return 0
    return Math.round(hits / (hits + misses) * 100)
}
</script>

<template>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="glass-panel p-4">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Valkey <span class="text-gray-600 normal-case">(queues)</span></div>
            <div class="space-y-1.5 text-sm">
                <div class="flex justify-between"><span class="text-gray-300">Version</span><span class="text-white">{{ data.redis?.version }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Uptime</span><span class="text-white">{{ formatUptime(data.redis?.uptime_seconds || 0) }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Memory</span><span class="text-white">{{ data.redis?.used_memory }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Clients</span><span class="text-white tabular-nums">{{ data.redis?.connected_clients }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Commands</span><span class="text-white tabular-nums">{{ formatNumber(data.redis?.total_commands_processed || 0) }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Hit Ratio</span><span class="font-medium tabular-nums" :class="hitRatio(data.redis) > 50 ? 'text-green-400' : 'text-amber-400'">{{ hitRatio(data.redis) }}%</span></div>
            </div>
        </div>
        <div v-if="data.redis_cache" class="glass-panel p-4">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Valkey <span class="text-gray-600 normal-case">(cache)</span></div>
            <div class="space-y-1.5 text-sm">
                <div class="flex justify-between"><span class="text-gray-300">Version</span><span class="text-white">{{ data.redis_cache?.version }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Uptime</span><span class="text-white">{{ formatUptime(data.redis_cache?.uptime_seconds || 0) }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Memory</span><span class="text-white">{{ data.redis_cache?.used_memory }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Clients</span><span class="text-white tabular-nums">{{ data.redis_cache?.connected_clients }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Commands</span><span class="text-white tabular-nums">{{ formatNumber(data.redis_cache?.total_commands_processed || 0) }}</span></div>
                <div class="flex justify-between"><span class="text-gray-300">Hit Ratio</span><span class="font-medium tabular-nums" :class="hitRatio(data.redis_cache) > 50 ? 'text-green-400' : 'text-amber-400'">{{ hitRatio(data.redis_cache) }}%</span></div>
            </div>
        </div>
    </div>
</template>
