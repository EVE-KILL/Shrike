<script setup lang="ts">
defineProps<{ data: any }>()

const formatUptime = (seconds: number): string => {
    const d = Math.floor(seconds / 86400)
    const h = Math.floor((seconds % 86400) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const parts = []
    if (d) parts.push(`${d}d`)
    if (h) parts.push(`${h}h`)
    parts.push(`${m}m`)
    return parts.join(' ')
}
</script>

<template>
    <div class="glass-panel p-4">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">PostgreSQL</div>
            <span
                class="text-xs font-medium uppercase"
                :class="data.database?.role === 'primary' ? 'text-green-400' : 'text-amber-400'"
            >{{ data.database?.role || 'unknown' }}</span>
        </div>
        <div class="space-y-1.5 text-sm">
            <div class="flex justify-between"><span class="text-gray-300">Version</span><span class="text-white">{{ data.database?.version || '—' }}</span></div>
            <div class="flex justify-between"><span class="text-gray-300">Uptime</span><span class="text-white">{{ formatUptime(data.database?.uptime_seconds || 0) }}</span></div>
            <div class="flex justify-between"><span class="text-gray-300">Database Size</span><span class="text-white">{{ data.database?.size || '—' }}</span></div>
            <div class="flex justify-between">
                <span class="text-gray-300">Connections</span>
                <span class="text-white tabular-nums">
                    {{ data.database?.connections?.total || 0 }} / {{ data.database?.connections?.max || 0 }}
                    <span class="text-gray-500">({{ data.database?.connections?.active || 0 }} active)</span>
                </span>
            </div>
            <div class="flex justify-between"><span class="text-gray-300">Cache Hit Ratio</span><span class="text-green-400 tabular-nums">{{ (data.database?.cache_hit_ratio || 0).toFixed(2) }}%</span></div>
            <div class="flex justify-between">
                <span class="text-gray-300">Waiting Locks</span>
                <span class="tabular-nums" :class="(data.database?.waiting_locks || 0) > 0 ? 'text-red-400' : 'text-white'">{{ data.database?.waiting_locks || 0 }}</span>
            </div>
        </div>
    </div>
</template>
