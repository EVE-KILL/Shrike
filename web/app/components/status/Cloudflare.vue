<script setup lang="ts">
defineProps<{ data: any }>()
</script>

<template>
    <div v-if="data" class="glass-panel p-4">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Cloudflare</div>
            <div class="text-xs text-gray-500 tabular-nums">Updated {{ new Date(data.last_updated).toLocaleTimeString() }}</div>
        </div>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-2 mb-3">
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Requests (24h)</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(data.last_24h.requests) }}</div>
                <div class="text-fine text-gray-600 tabular-nums">{{ formatNumber(data.last_1h.requests) }} last hour</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Bandwidth (24h)</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ data.last_24h.bandwidth_gb }} GB</div>
                <div class="text-fine text-gray-600 tabular-nums">{{ data.last_24h.bandwidth_cached_gb }} GB cached</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Cache Hit Rate</div>
                <div class="text-sm font-medium tabular-nums" :class="data.last_24h.cache_hit_rate > 50 ? 'text-green-400' : 'text-amber-400'">{{ data.last_24h.cache_hit_rate }}%</div>
                <div class="text-fine text-gray-600 tabular-nums">{{ data.last_1h.cache_hit_rate }}% last hour</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Threats Blocked (24h)</div>
                <div class="text-sm font-medium tabular-nums" :class="data.last_24h.threats_blocked > 0 ? 'text-red-400' : 'text-green-400'">{{ formatNumber(data.last_24h.threats_blocked) }}</div>
                <div class="text-fine text-gray-600 tabular-nums">{{ formatNumber(data.last_1h.threats_blocked) }} last hour</div>
            </div>
        </div>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-2">
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Page Views (24h)</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(data.last_24h.page_views) }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Unique Visitors (24h)</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(data.last_24h.unique_visitors) }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">API Requests (24h)</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber((data.last_24h.status_codes?.[200] || 0) + (data.last_24h.status_codes?.[304] || 0)) }}</div>
                <div class="text-fine text-gray-600">2xx + 304</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Errors (24h)</div>
                <div class="text-sm font-medium tabular-nums" :class="(data.last_24h.status_codes?.[500] || 0) > 0 ? 'text-red-400' : 'text-green-400'">{{ formatNumber((data.last_24h.status_codes?.[500] || 0) + (data.last_24h.status_codes?.[502] || 0) + (data.last_24h.status_codes?.[503] || 0) + (data.last_24h.status_codes?.[504] || 0)) }}</div>
                <div class="text-fine text-gray-600">5xx errors</div>
            </div>
        </div>
    </div>
</template>
