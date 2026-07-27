<script setup lang="ts">
const props = defineProps<{
    characterId: number
    description: string | null
    customDescriptionHtml?: string | null
    /** ISO timestamp of the pilot's last killmail, for the empty-period notice. */
    lastActive?: string | null
    /** Lifetime kills + losses, used to tell "never flew" from "not lately". */
    lifetimeEvents?: number
}>()

const { convertEveHtml } = useEveHtmlParser()

// Prefer the server-rendered custom bio (already sanitized) when set.
// Fall back to parsing the ESI `description` client-side via the EVE HTML
// parser — which is the legacy path and stays untouched for non-customized
// characters.
const parsedBio = computed(() => {
    if (props.customDescriptionHtml) return props.customDescriptionHtml
    if (!props.description) return ''
    return convertEveHtml(props.description)
})

// Unified period selection (shared between Stats + Intel Profile)
const periods = [
    { value: 14, label: '14d' },
    { value: 30, label: '30d' },
    { value: 90, label: '90d' },
    { value: 180, label: '180d' },
    { value: 365, label: '1y' },
    // Characters topped out at a year while corporations already offered this.
    // A pilot who stopped flying in 2020 had no reachable period that showed
    // anything, so the whole dashboard read as zeroes with no way out.
    { value: 0, label: 'All Time' },
]
const activePeriod = ref(90)

const statsParams = computed(() => ({ days: activePeriod.value }))
const { data, pending } = await useApiFetch<any>(`/api/character/${props.characterId}/stats`, {
    params: statsParams,
    watch: [activePeriod],
    getCachedData: cachedPayload,
})

const stats = computed(() => data.value?.stats)
const heatMap = computed(() => data.value?.heatMap || {})

// Panels the endpoint has always returned and this dashboard never rendered.
const diesToCorps = computed(() => data.value?.diesToCorporations || [])
const diesToAlliances = computed(() => data.value?.diesToAlliances || [])

// Co-attackers: groups that turn up on the same killmails as this pilot.
// Counted in shared killmails, so the numbers read as "fleets together".
const fliesWithCorps = computed(() => data.value?.fliesWithCorporations || [])
const fliesWithAlliances = computed(() => data.value?.fliesWithAlliances || [])

/**
 * Nothing happened inside the selected window, but the pilot does have a
 * record. Without this the page contradicted itself — a header reading "2,414
 * kills" above a stats table reading "Kills 0", with nothing to say that the
 * table was scoped to ninety days and the pilot last undocked years ago.
 */
const hasPeriodActivity = computed(() =>
    !!stats.value && ((stats.value.kills ?? 0) > 0 || (stats.value.losses ?? 0) > 0))

const showEmptyPeriodNotice = computed(() =>
    !pending.value && activePeriod.value !== 0 && !hasPeriodActivity.value && (props.lifetimeEvents ?? 0) > 0)

const lastActiveLabel = computed(() => {
    if (!props.lastActive) return null
    const d = new Date(props.lastActive)
    if (Number.isNaN(d.getTime())) return null
    return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric', timeZone: 'UTC' })
})

const activePeriodLabel = computed(() =>
    periods.find(p => p.value === activePeriod.value)?.label ?? `${activePeriod.value}d`)

const heatMapHasData = computed(() =>
    Object.values(heatMap.value as Record<number, number>).some(v => v > 0))

// Heat map helpers
const heatMapMax = computed(() => Math.max(1, ...Object.values(heatMap.value as Record<number, number>)))
const heatMapColor = (count: number): string => {
    if (count === 0) return 'bg-white/[0.03]'
    const ratio = count / heatMapMax.value
    if (ratio > 0.75) return 'bg-purple-500/60'
    if (ratio > 0.5) return 'bg-indigo-500/50'
    if (ratio > 0.25) return 'bg-blue-500/40'
    return 'bg-blue-500/20'
}

</script>

<template>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- ===== LEFT HALF: Bio ===== -->
        <div>
            <!-- Sized to content, not stretched to the stats column. `h-full` made
                 this a bordered box as tall as everything opposite it -- 2,504px of
                 empty frame around the words "No bio available" on an alliance
                 with no bio. Sticky from the same breakpoint the two-column split starts at, so
                 the identity stays with you while the long stats column scrolls. Capped to the
                 viewport and scrollable, or a bio longer than the screen would
                 pin with its tail permanently out of reach. -->
            <div class="glass-panel p-4 md:sticky md:top-4 md:max-h-[calc(100vh-2rem)] md:overflow-y-auto">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Bio</div>
                <div v-if="parsedBio" class="entity-bio" v-html="parsedBio"></div>
                <div v-else class="text-fine text-gray-600 italic">No bio available</div>
            </div>
        </div>

        <!-- ===== RIGHT HALF: Stats ===== -->
        <div class="space-y-4">
            <!-- Period selector (shared) -->
            <div class="flex items-center justify-between">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">
                    Dashboard
                </div>
                <div class="flex gap-1">
                    <button
                        v-for="period in periods"
                        :key="period.value"
                        class="px-2.5 py-1 text-fine rounded font-medium transition-colors"
                        :class="activePeriod === period.value
                            ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                            : 'text-gray-500 hover:text-blue-400 border border-transparent'"
                        @click="activePeriod = period.value"
                    >
                        {{ period.label }}
                    </button>
                </div>
            </div>

            <!-- Period Stats -->
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-4">
                    Stats
                </div>

                <!-- Loading -->
                <div v-if="pending" class="flex items-center justify-center py-8">
                    <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                </div>

                <!-- Quiet window: say so, rather than printing a column of zeroes -->
                <div v-else-if="showEmptyPeriodNotice" class="py-6 text-center">
                    <Icon name="lucide:moon" class="text-2xl text-gray-600 mb-2" />
                    <p class="text-xs text-gray-400">No activity in the last {{ activePeriodLabel }}</p>
                    <p v-if="lastActiveLabel" class="text-fine text-gray-600 mt-1">Last seen {{ lastActiveLabel }}</p>
                    <button
                        class="mt-3 px-3 py-1.5 rounded-lg text-fine font-medium bg-blue-500/15 text-blue-400 border border-blue-500/30 hover:bg-blue-500/25 transition-colors cursor-pointer"
                        @click="activePeriod = 0"
                    >
                        Show all time
                    </button>
                </div>

                <!-- Stats table -->
                <div v-else-if="stats" class="space-y-1.5 text-xs">
                    <div class="flex justify-between text-gray-400">
                        <span>Kills</span>
                        <span class="text-fine text-green-400 tabular-nums">{{ formatNumber(stats.kills) }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>Losses</span>
                        <span class="text-fine text-red-400 tabular-nums">{{ formatNumber(stats.losses) }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>Efficiency</span>
                        <span class="text-fine text-white tabular-nums">{{ stats.efficiency }}%</span>
                    </div>
                    <div class="h-1 bg-red-500/20 rounded-full overflow-hidden my-1">
                        <div class="h-full bg-green-500 rounded-full" :style="{ width: `${stats.efficiency}%` }"></div>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>ISK Destroyed</span>
                        <span class="text-fine text-green-400 tabular-nums">{{ formatIsk(stats.isk_destroyed) }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>ISK Lost</span>
                        <span class="text-fine text-red-400 tabular-nums">{{ formatIsk(stats.isk_lost) }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>ISK Efficiency</span>
                        <span class="text-fine text-white tabular-nums">{{ stats.isk_efficiency }}%</span>
                    </div>
                    <div class="h-1 bg-red-500/20 rounded-full overflow-hidden my-1">
                        <div class="h-full bg-green-500 rounded-full" :style="{ width: `${stats.isk_efficiency}%` }"></div>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>Solo Kills</span>
                        <span class="text-fine text-white tabular-nums">{{ formatNumber(stats.solo_kills) }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>Solo Losses</span>
                        <span class="text-fine text-white tabular-nums">{{ formatNumber(stats.solo_losses) }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>NPC Losses</span>
                        <span class="text-fine text-white tabular-nums">{{ formatNumber(stats.npc_losses) }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>Blob Factor</span>
                        <span class="text-fine text-white tabular-nums">{{ stats.blob_factor }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>Active Timezone</span>
                        <span class="text-fine text-white">{{ stats.active_timezone }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>Final Blows</span>
                        <span class="text-fine text-white tabular-nums">{{ formatNumber(stats.final_blows) }}</span>
                    </div>
                    <div class="flex justify-between text-gray-400">
                        <span>Points</span>
                        <span class="text-fine text-white tabular-nums">{{ formatNumber(stats.points) }}</span>
                    </div>
                </div>
            </div>

            <!-- Heat Map. Hidden when flat: 24 cells all reading "0" is a
                 grid that looks like data and carries none. -->
            <div v-if="!pending && heatMapHasData" class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Activity Heat Map (EVE Time)</div>
                <div class="grid grid-cols-12 gap-1">
                    <div v-for="h in 24" :key="h - 1"
                        class="rounded p-1 text-center transition-colors"
                        :class="heatMapColor((heatMap as any)[h - 1] || 0)">
                        <div class="text-fine text-gray-500">{{ String(h - 1).padStart(2, '0') }}</div>
                        <div class="text-fine tabular-nums" :class="(heatMap as any)[h - 1] ? 'text-gray-300' : 'text-gray-700'">
                            {{ (heatMap as any)[h - 1] || 0 }}
                        </div>
                    </div>
                </div>
            </div>

            <!-- Who this pilot fleets with -->
            <div v-if="!pending && (fliesWithCorps.length > 0 || fliesWithAlliances.length > 0)" class="grid grid-cols-2 gap-3">
                <TopBox v-if="fliesWithCorps.length > 0" title="Flies With (Corps)" data-type="corporations" :entries="fliesWithCorps" count-color="text-isk/70" hide-footer />
                <TopBox v-if="fliesWithAlliances.length > 0" title="Flies With (Alliances)" data-type="alliances" :entries="fliesWithAlliances" count-color="text-isk/70" hide-footer />
            </div>

            <!-- Who kills this pilot. The endpoint has always returned these;
                 only the corporation dashboard was rendering them. -->
            <div v-if="!pending && (diesToCorps.length > 0 || diesToAlliances.length > 0)" class="grid grid-cols-2 gap-3">
                <TopBox v-if="diesToCorps.length > 0" title="Dies To (Corps)" data-type="corporations" :entries="diesToCorps" count-color="text-red-400/70" />
                <TopBox v-if="diesToAlliances.length > 0" title="Dies To (Alliances)" data-type="alliances" :entries="diesToAlliances" count-color="text-red-400/70" />
            </div>

            <!-- Intel Profile (graph-powered) -->
            <CharacterIntelProfile :character-id="props.characterId" :days="activePeriod" />
        </div>
    </div>
</template>
