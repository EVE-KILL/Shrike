<script setup lang="ts">
const props = defineProps<{ data: any; wsStats: any }>()

const tqStatusColor = computed(() => {
    const s = props.data?.esi?.tq_status
    if (s === 'online') return 'text-green-400'
    if (s === 'vip') return 'text-amber-400'
    return 'text-red-400'
})

const totalQueued = computed(() => {
    if (!props.data?.queues) return 0
    return Object.values(props.data.queues as Record<string, any>).reduce((s: number, q: any) => s + (q.waiting || 0), 0)
})

const totalActive = computed(() => {
    if (!props.data?.queues) return 0
    return Object.values(props.data.queues as Record<string, any>).reduce((s: number, q: any) => s + (q.active || 0), 0)
})

const walletBalance = computed(() => {
    const balance = Number(props.data?.wallet?.total_balance)
    return Number.isFinite(balance) ? balance : null
})

const formatWalletBalance = (value: number): string => {
    const absolute = Math.abs(value)
    const units = [
        { value: 1e12, suffix: 't' },
        { value: 1e9, suffix: 'b' },
        { value: 1e6, suffix: 'm' },
        { value: 1e3, suffix: 'k' },
    ]
    const unit = units.find(candidate => absolute >= candidate.value)
    if (!unit) return Math.round(value).toLocaleString('en-US')
    return `${Math.round(value / unit.value)}${unit.suffix}`
}

const walletDisplay = computed(() =>
    walletBalance.value === null ? '— ISK' : `${formatWalletBalance(walletBalance.value)} ISK`,
)
</script>

<template>
    <div class="grid grid-cols-2 md:grid-cols-6 gap-3">
        <div class="glass-panel p-4 text-center">
            <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">Tranquility</div>
            <div class="text-base font-bold" :class="tqStatusColor">{{ data.esi?.tq_status?.toUpperCase() || 'UNKNOWN' }}</div>
            <div class="text-xs text-gray-500">{{ formatNumber(data.esi?.tq_players || 0) }} pilots</div>
        </div>
        <div class="glass-panel p-4 text-center">
            <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">EK Wallet</div>
            <div
                class="text-base font-bold text-amber-400 tabular-nums"
                v-tooltip="walletBalance === null ? 'Wallet balance unavailable' : `${formatNumber(walletBalance)} ISK`"
            >
                {{ walletDisplay }}
            </div>
            <div class="text-xs text-gray-500">corp balance</div>
        </div>
        <div class="glass-panel p-4 text-center">
            <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">Queued</div>
            <div class="text-base font-bold text-amber-400">{{ formatNumber(totalQueued) }}</div>
            <div class="text-xs text-gray-500">{{ totalActive }} active</div>
        </div>
        <div class="glass-panel p-4 text-center">
            <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">ESI Tokens</div>
            <div class="text-base font-bold text-purple-400">{{ data.esi_tokens?.tokens?.active || 0 }} <span class="text-sm text-gray-500">/ {{ data.esi_tokens?.tokens?.total || 0 }}</span></div>
            <div class="text-xs text-gray-500">active keys</div>
        </div>
        <div class="glass-panel p-4 text-center">
            <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">Database</div>
            <div class="text-base font-bold text-blue-400">{{ data.database?.size || '?' }}</div>
            <div class="text-xs text-gray-500">{{ Object.keys(data.database?.tables || {}).length }} tables</div>
        </div>
        <div class="glass-panel p-4 text-center">
            <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">WebSocket</div>
            <div class="text-base font-bold text-cyan-400">{{ wsStats?.connections ?? 0 }}</div>
            <div class="text-xs text-gray-500">connections</div>
        </div>
    </div>
</template>
