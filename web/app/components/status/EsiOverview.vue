<script setup lang="ts">
const props = defineProps<{ coverage: any; tokens: any }>()

const pct = (num: number, den: number): string =>
    den > 0 ? `${((num / den) * 100).toFixed(1)}%` : '—'

const barWidth = (num: number, den: number): string =>
    den > 0 ? `${Math.min(100, (num / den) * 100).toFixed(1)}%` : '0%'

const coverageRows = computed(() => {
    const cov = props.coverage
    if (!cov) return []
    return [
        {
            label: 'Characters',
            covered: cov.characters.covered,
            active: cov.characters.active,
            detail: `${formatNumber(cov.characters.covered)} / ${formatNumber(cov.characters.active)} via own or corp key`,
        },
        {
            label: 'Corporations',
            covered: cov.corporations.covered,
            active: cov.corporations.active,
            detail: `${formatNumber(cov.corporations.covered)} / ${formatNumber(cov.corporations.active)} with corp key`,
        },
        {
            label: 'Alliances',
            covered: cov.alliances.corps_covered,
            active: cov.alliances.corps_active,
            detail: `${formatNumber(cov.alliances.corps_covered)} / ${formatNumber(cov.alliances.corps_active)} member corps · ${formatNumber(cov.alliances.active)} alliances`,
        },
    ]
})

const tokenStats = computed(() => [
    { label: 'Character keys', value: props.tokens?.tokens?.can_fetch_character || 0 },
    { label: 'Corporation keys', value: props.tokens?.tokens?.can_fetch_corporation || 0 },
    { label: 'Fetches (24h)', value: props.tokens?.fetches?.last_24h?.total || 0 },
    { label: 'Failed (24h)', value: props.tokens?.fetches?.last_24h?.failed || 0, warning: true },
    { label: 'Keyed KMs (24h)', value: props.tokens?.fetches?.last_24h?.killmails_found || 0 },
    { label: 'New KMs (24h)', value: props.tokens?.fetches?.last_24h?.new_killmails || 0, positive: true },
])
</script>

<template>
    <div v-if="coverage || tokens" class="glass-panel p-4">
        <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">ESI Coverage &amp; Tokens</div>
            <div v-if="tokens" class="flex items-center gap-2 text-xs tabular-nums">
                <span class="text-green-400">{{ formatNumber(tokens.tokens?.active || 0) }} active</span>
                <span class="text-gray-600">/</span>
                <span class="text-gray-500">{{ formatNumber(tokens.tokens?.total || 0) }} keys</span>
                <span v-if="(tokens.tokens?.revoked || 0) > 0" class="text-red-400">{{ formatNumber(tokens.tokens.revoked) }} revoked</span>
            </div>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 lg:gap-6">
            <section>
                <div class="flex items-center justify-between mb-2">
                    <div class="text-xs font-medium text-gray-400">Coverage</div>
                    <div v-if="coverage" class="text-fine text-gray-600">entities active last {{ coverage.window_days }}d</div>
                </div>
                <div v-if="coverageRows.length" class="space-y-2">
                    <div v-for="row in coverageRows" :key="row.label" class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                        <div class="flex items-baseline justify-between">
                            <div class="text-xs text-gray-400">{{ row.label }}</div>
                            <div class="text-sm text-white font-medium tabular-nums">{{ pct(row.covered, row.active) }}</div>
                        </div>
                        <div class="mt-1.5 h-1 rounded-full bg-white/[0.06] overflow-hidden">
                            <div class="h-full rounded-full bg-blue-400/70" :style="{ width: barWidth(row.covered, row.active) }"></div>
                        </div>
                        <div class="mt-1 text-fine text-gray-600 tabular-nums">{{ row.detail }}</div>
                    </div>
                </div>
                <div v-else class="text-xs text-gray-600 py-4">Coverage data unavailable</div>
            </section>

            <section>
                <div class="text-xs font-medium text-gray-400 mb-2">Token activity</div>
                <div v-if="tokens" class="grid grid-cols-2 gap-2">
                    <div v-for="stat in tokenStats" :key="stat.label" class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                        <div class="text-xs text-gray-500">{{ stat.label }}</div>
                        <div
                            class="text-sm font-medium tabular-nums"
                            :class="stat.warning && stat.value > 0 ? 'text-red-400' : stat.positive ? 'text-green-400' : 'text-white'"
                        >
                            {{ formatNumber(stat.value) }}
                        </div>
                    </div>
                </div>
                <div v-else class="text-xs text-gray-600 py-4">Token data unavailable</div>
                <div v-if="tokens" class="mt-3 pt-3 border-t border-white/[0.06] flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between text-xs">
                    <span class="text-gray-500">All time</span>
                    <div class="flex items-center gap-3 tabular-nums">
                        <span class="text-white">{{ formatNumber(tokens.fetches?.all_time?.total || 0) }} <span class="text-gray-500">fetches</span></span>
                        <span class="text-green-400">{{ formatNumber(tokens.fetches?.all_time?.new_killmails || 0) }} <span class="text-gray-500">new KMs</span></span>
                    </div>
                </div>
            </section>
        </div>
    </div>
</template>
