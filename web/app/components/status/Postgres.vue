<script setup lang="ts">
const props = defineProps<{ data: any }>()

const history = ref<number[]>([])

watch(
    () => props.data.database?.sampled_at,
    (sampledAt) => {
        if (!sampledAt) return
        const value = props.data.database?.statements?.queries_per_second
            ?? props.data.database?.workload?.transactions_per_second
            ?? 0
        history.value = [...history.value.slice(-23), Number(value)]
    },
    { immediate: true },
)

const sparkline = computed(() => {
    if (history.value.length < 2) return ''
    const maximum = Math.max(...history.value, 1)
    return history.value.map((value, index) => {
        const x = index * 100 / (history.value.length - 1)
        const y = 28 - (value / maximum * 26) - 1
        return `${x.toFixed(2)},${y.toFixed(2)}`
    }).join(' ')
})

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

const formatRate = (value: number): string => {
    if (value >= 1000) return `${(value / 1000).toFixed(1)}k/s`
    if (value >= 10) return `${value.toFixed(0)}/s`
    return `${value.toFixed(1)}/s`
}

const formatBytes = (bytes: number): string => {
    if (bytes >= 1024 ** 3) return `${(bytes / 1024 ** 3).toFixed(1)} GB/s`
    if (bytes >= 1024 ** 2) return `${(bytes / 1024 ** 2).toFixed(1)} MB/s`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB/s`
    return `${bytes.toFixed(0)} B/s`
}

const formatLag = (database: any): string => {
    const seconds = database?.cluster?.max_lag_seconds || 0
    const bytes = database?.cluster?.max_lag_bytes || 0
    if (seconds >= 1) return `${seconds.toFixed(1)}s`
    if (seconds > 0) return `${Math.round(seconds * 1000)}ms`
    if (bytes >= 1024 ** 2) return `${(bytes / 1024 ** 2).toFixed(1)} MB`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return bytes ? `${bytes} B` : '0'
}
</script>

<template>
    <div class="glass-panel p-4 flex flex-col">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">PostgreSQL</div>
            <span
                class="text-xs font-medium uppercase"
                :class="data.database?.role === 'primary' ? 'text-green-400' : 'text-amber-400'"
            >{{ data.database?.role || 'unknown' }}</span>
        </div>

        <div class="grid grid-cols-3 gap-2 mb-4">
            <div class="bg-white/[0.025] border border-white/[0.04] rounded px-2 py-2">
                <div class="text-[9px] uppercase tracking-wider text-gray-500">Role</div>
                <div class="text-sm font-medium text-green-400 capitalize">{{ data.database?.role || 'unknown' }}</div>
            </div>
            <div class="bg-white/[0.025] border border-white/[0.04] rounded px-2 py-2">
                <div class="text-[9px] uppercase tracking-wider text-gray-500">Replicas</div>
                <div v-if="data.database?.cluster?.replicas" class="text-sm font-medium text-white tabular-nums">
                    {{ data.database.cluster.streaming }} / {{ data.database.cluster.replicas }}
                    <span class="text-[10px] text-gray-500">streaming</span>
                </div>
                <div v-else class="text-sm font-medium text-gray-300">Standalone</div>
            </div>
            <div class="bg-white/[0.025] border border-white/[0.04] rounded px-2 py-2">
                <div class="text-[9px] uppercase tracking-wider text-gray-500">Max Lag</div>
                <div class="text-sm font-medium tabular-nums" :class="(data.database?.cluster?.max_lag_seconds || 0) > 5 ? 'text-red-400' : 'text-green-400'">
                    {{ formatLag(data.database) }}
                </div>
            </div>
        </div>

        <div class="space-y-1.5 text-sm">
            <div class="flex justify-between">
                <span class="text-gray-300">Connections</span>
                <span class="text-white tabular-nums">
                    {{ data.database?.connections?.total || 0 }} / {{ data.database?.connections?.max || 0 }}
                    <span class="text-gray-500">· {{ data.database?.connections?.active || 0 }} active · {{ data.database?.connections?.waiting || 0 }} waiting</span>
                </span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-300">Transactions</span>
                <span v-if="data.database?.workload" class="text-white tabular-nums">
                    {{ formatRate(data.database.workload.transactions_per_second) }}
                    <span class="text-gray-500">· {{ data.database.workload.rollback_percent.toFixed(2) }}% rollback</span>
                </span>
                <span v-else class="text-gray-500">collecting…</span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-300">Queries</span>
                <span v-if="data.database?.statements" class="text-white tabular-nums">
                    {{ formatRate(data.database.statements.queries_per_second) }}
                    <span class="text-gray-500">· {{ data.database.statements.average_latency_ms.toFixed(2) }} ms avg</span>
                </span>
                <span v-else class="text-gray-500">statistics unavailable</span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-300">I/O Throughput</span>
                <span v-if="data.database?.workload" class="text-white tabular-nums">
                    <span class="text-blue-400">↓ {{ formatBytes(data.database.workload.read_bytes_per_second) }}</span>
                    <span class="text-gray-600 mx-1">·</span>
                    <span class="text-amber-400">↑ {{ formatBytes(data.database.workload.write_bytes_per_second) }}</span>
                </span>
                <span v-else class="text-gray-500">collecting…</span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-300">WAL</span>
                <span v-if="data.database?.workload" class="text-white tabular-nums">{{ formatBytes(data.database.workload.wal_bytes_per_second) }}</span>
                <span v-else class="text-gray-500">collecting…</span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-300">Buffer Cache</span>
                <span class="tabular-nums" :class="(data.database?.cache_hit_ratio || 0) >= 95 ? 'text-green-400' : 'text-amber-400'">{{ (data.database?.cache_hit_ratio || 0).toFixed(2) }}%</span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-300">Contention</span>
                <span class="tabular-nums" :class="(data.database?.waiting_locks || 0) > 0 ? 'text-red-400' : 'text-white'">
                    {{ data.database?.waiting_locks || 0 }} locks
                    <span class="text-gray-500">· {{ data.database?.workload?.deadlocks || 0 }} deadlocks</span>
                </span>
            </div>
        </div>

        <div class="mt-4 pt-3 border-t border-white/[0.05]">
            <div class="flex justify-between items-center mb-1">
                <span class="text-[9px] uppercase tracking-wider text-gray-500">{{ data.database?.statements ? 'Query rate' : 'Transaction rate' }} · recent samples</span>
                <span class="text-[10px] text-gray-500 tabular-nums">{{ history.length }}/24</span>
            </div>
            <svg viewBox="0 0 100 30" preserveAspectRatio="none" class="w-full h-12" aria-hidden="true">
                <line x1="0" y1="29" x2="100" y2="29" stroke="rgba(255,255,255,0.06)" stroke-width="0.5" />
                <polyline v-if="sparkline" :points="sparkline" fill="none" stroke="rgb(52 211 153)" stroke-width="1.5" vector-effect="non-scaling-stroke" />
            </svg>
        </div>

        <div class="mt-auto pt-3 flex flex-wrap gap-x-3 gap-y-1 text-[10px] text-gray-500 border-t border-white/[0.05]">
            <span>v{{ data.database?.version || '—' }}</span>
            <span>{{ formatUptime(data.database?.uptime_seconds || 0) }} uptime</span>
            <span>{{ data.database?.size || '—' }}</span>
        </div>
    </div>
</template>
