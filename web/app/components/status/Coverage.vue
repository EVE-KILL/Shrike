<script setup lang="ts">
const props = defineProps<{ data: any }>()

const cov = computed(() => props.data ?? null)

const pct = (num: number, den: number): string =>
    den > 0 ? `${((num / den) * 100).toFixed(1)}%` : '—'
const barWidth = (num: number, den: number): string =>
    den > 0 ? `${Math.min(100, (num / den) * 100).toFixed(1)}%` : '0%'
</script>

<template>
    <div v-if="cov" class="glass-panel p-4">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">ESI Coverage</div>
            <div class="text-xs text-gray-500">entities active last {{ cov.window_days }}d</div>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-2">
            <!-- Characters: own key, or their corp's key (corp feed covers members) -->
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="flex items-baseline justify-between">
                    <div class="text-xs text-gray-400">Characters</div>
                    <div class="text-sm text-white font-medium tabular-nums">{{ pct(cov.characters.covered, cov.characters.active) }}</div>
                </div>
                <div class="mt-1.5 h-1 rounded-full bg-white/[0.06] overflow-hidden">
                    <div class="h-full rounded-full bg-blue-400/70" :style="{ width: barWidth(cov.characters.covered, cov.characters.active) }"></div>
                </div>
                <div class="mt-1 text-fine text-gray-600 tabular-nums">
                    {{ formatNumber(cov.characters.covered) }} / {{ formatNumber(cov.characters.active) }} via own or corp key
                </div>
            </div>

            <!-- Corporations: covered by a corp-scope key -->
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="flex items-baseline justify-between">
                    <div class="text-xs text-gray-400">Corporations</div>
                    <div class="text-sm text-white font-medium tabular-nums">{{ pct(cov.corporations.covered, cov.corporations.active) }}</div>
                </div>
                <div class="mt-1.5 h-1 rounded-full bg-white/[0.06] overflow-hidden">
                    <div class="h-full rounded-full bg-blue-400/70" :style="{ width: barWidth(cov.corporations.covered, cov.corporations.active) }"></div>
                </div>
                <div class="mt-1 text-fine text-gray-600 tabular-nums">
                    {{ formatNumber(cov.corporations.covered) }} / {{ formatNumber(cov.corporations.active) }} with corp key
                </div>
            </div>

            <!-- Alliances: corp-weighted — of the active member corps across
                 alliances, how many are covered by corp keys -->
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="flex items-baseline justify-between">
                    <div class="text-xs text-gray-400">Alliances</div>
                    <div class="text-sm text-white font-medium tabular-nums">{{ pct(cov.alliances.corps_covered, cov.alliances.corps_active) }}</div>
                </div>
                <div class="mt-1.5 h-1 rounded-full bg-white/[0.06] overflow-hidden">
                    <div class="h-full rounded-full bg-blue-400/70" :style="{ width: barWidth(cov.alliances.corps_covered, cov.alliances.corps_active) }"></div>
                </div>
                <div class="mt-1 text-fine text-gray-600 tabular-nums">
                    {{ formatNumber(cov.alliances.corps_covered) }} / {{ formatNumber(cov.alliances.corps_active) }} member corps · {{ formatNumber(cov.alliances.active) }} alliances active
                </div>
            </div>
        </div>
    </div>
</template>
