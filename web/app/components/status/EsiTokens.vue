<script setup lang="ts">
defineProps<{ data: any }>()
</script>

<template>
    <div class="glass-panel p-4">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">ESI Tokens</div>
            <div class="flex items-center gap-2 text-xs">
                <span class="tabular-nums text-green-400">{{ formatNumber(data.esi_tokens?.tokens?.active || 0) }}</span>
                <span class="text-gray-600">/</span>
                <span class="tabular-nums text-gray-500">{{ formatNumber(data.esi_tokens?.tokens?.total || 0) }} keys</span>
                <span v-if="(data.esi_tokens?.tokens?.revoked || 0) > 0" class="tabular-nums text-red-400">({{ formatNumber(data.esi_tokens?.tokens?.revoked || 0) }} rev)</span>
            </div>
        </div>
        <div class="grid grid-cols-2 gap-2">
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Char Fetch</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(data.esi_tokens?.tokens?.can_fetch_character || 0) }}</div>
                <div class="text-fine text-gray-600">keys with scope</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Corp Fetch</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(data.esi_tokens?.tokens?.can_fetch_corporation || 0) }}</div>
                <div class="text-fine text-gray-600">keys with scope</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">KMs Covered (24h)</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(data.esi_tokens?.fetches?.last_24h?.killmails_found || 0) }}</div>
                <div class="text-fine text-gray-600">last-24h kills with a keyed victim/attacker</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">New KMs (24h)</div>
                <div class="text-sm font-medium tabular-nums text-green-400">{{ formatNumber(data.esi_tokens?.fetches?.last_24h?.new_killmails || 0) }}</div>
            </div>
        </div>
        <div class="mt-2 grid grid-cols-2 gap-2 text-xs">
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-gray-500">Fetches (24h)</div>
                <div class="text-white tabular-nums">{{ formatNumber(data.esi_tokens?.fetches?.last_24h?.total || 0) }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-gray-500">Failed (24h)</div>
                <div class="tabular-nums" :class="(data.esi_tokens?.fetches?.last_24h?.failed || 0) > 0 ? 'text-red-400' : 'text-white'">{{ formatNumber(data.esi_tokens?.fetches?.last_24h?.failed || 0) }}</div>
            </div>
        </div>
        <div class="mt-3 pt-3 border-t border-white/[0.06] flex items-center justify-between text-xs">
            <span class="text-gray-500">All-Time</span>
            <div class="flex items-center gap-3 tabular-nums">
                <span class="text-white">{{ formatNumber(data.esi_tokens?.fetches?.all_time?.total || 0) }} <span class="text-gray-500">fetches</span></span>
                <span class="text-green-400">{{ formatNumber(data.esi_tokens?.fetches?.all_time?.new_killmails || 0) }} <span class="text-gray-500">new KMs</span></span>
            </div>
        </div>
    </div>
</template>
