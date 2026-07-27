<script setup lang="ts">
const props = defineProps<{ data: any }>()

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

const endpoints = computed(() => {
    if (!props.data?.endpoints) return []
    return Object.entries(props.data.endpoints as Record<string, any>)
        .map(([path, e]: [string, any]) => {
            // Group subscriptions by prefix (e.g. victim.123, victim.456 → victim: 2)
            const raw = Object.entries(e.subscriptions || {}) as [string, number][]
            const groups = new Map<string, number>()
            for (const [sub, count] of raw) {
                const prefix = (sub.includes('.') ? sub.split('.')[0] : sub) ?? sub
                groups.set(prefix, (groups.get(prefix) || 0) + count)
            }
            return {
                path,
                connections: e.connections || 0,
                subscriptions: [...groups.entries()].sort((a, b) => b[1] - a[1]),
            }
        })
        .sort((a, b) => b.connections - a.connections)
})

</script>

<template>
    <div v-if="data" class="glass-panel p-4">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">WebSocket Server</div>
            <div class="flex items-center gap-3 text-xs text-gray-500">
                <span class="flex items-center gap-1.5">
                    <span class="w-1.5 h-1.5 rounded-full" :class="data.redis ? 'bg-green-400' : 'bg-red-400'"></span>
                    Redis {{ data.redis ? 'connected' : 'disconnected' }}
                </span>
                <span class="text-gray-700">&middot;</span>
                <span class="tabular-nums">{{ formatUptime(data.uptime || 0) }} uptime</span>
            </div>
        </div>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-2 mb-3">
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Connections</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ data.connections ?? 0 }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Channels</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ data.channels?.length ?? 0 }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Endpoints</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ endpoints.length }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Status</div>
                <div class="text-sm font-medium" :class="data.status === 'ok' ? 'text-green-400' : 'text-red-400'">{{ data.status?.toUpperCase() }}</div>
            </div>
        </div>
        <div class="space-y-1.5">
            <div v-for="ep in endpoints" :key="ep.path"
                class="flex items-center justify-between text-sm">
                <span class="text-gray-300 font-mono text-xs">{{ ep.path }}</span>
                <div class="flex items-center gap-3 tabular-nums">
                    <span :class="ep.connections > 0 ? 'text-green-400' : 'text-gray-600'">{{ ep.connections }} <span class="text-fine text-gray-500">conn</span></span>
                    <span v-for="[sub, count] in ep.subscriptions" :key="sub" class="text-blue-400">
                        {{ count }} <span class="text-fine text-gray-500">{{ sub }}</span>
                    </span>
                </div>
            </div>
        </div>
    </div>
</template>
