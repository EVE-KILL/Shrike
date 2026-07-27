<script setup lang="ts">
/**
 * Shared Top tab for character / corporation / alliance pages.
 *
 * Split into two independent slices, each with its own period selector:
 *   - Left  (`slice=left`)  — member widgets + lifetime widgets. Backed by
 *                             raw killmail queries; capped at 365d. Empty
 *                             for character pages.
 *   - Right (`slice=right`) — activity widgets (ships / systems /
 *                             constellations / regions / killed-corp /
 *                             killed-alliance). Backed by stats_breakdowns;
 *                             supports alltime. Constellations are derived
 *                             by rolling up the SYSTEM breakdown.
 *
 * On character pages the left slice has no widgets, so the right slice
 * spans the full width.
 */
const props = defineProps<{
    entityType: 'character' | 'corporation' | 'alliance'
    entityId: number
}>()

const isOrg = computed(() => props.entityType !== 'character')

const leftPeriods = [
    { label: '1h', value: 1 / 24 },
    { label: '6h', value: 6 / 24 },
    { label: '12h', value: 0.5 },
    { label: '1d', value: 1 },
    { label: '7d', value: 7 },
    { label: '14d', value: 14 },
    { label: '30d', value: 30 },
    { label: '90d', value: 90 },
    { label: '180d', value: 180 },
    { label: '365d', value: 365 },
] as const

const rightPeriods = [
    { label: '1h', value: 1 / 24 },
    { label: '6h', value: 6 / 24 },
    { label: '12h', value: 0.5 },
    { label: '1d', value: 1 },
    { label: '7d', value: 7 },
    { label: '14d', value: 14 },
    { label: '30d', value: 30 },
    { label: '90d', value: 90 },
    { label: '180d', value: 180 },
    { label: '365d', value: 365 },
    { label: 'All Time', value: 'alltime' as const },
] as const

const leftPeriod = ref<number>(7)
const rightPeriod = ref<number | 'alltime'>(7)


// Skip the left fetch entirely on character pages — no widgets render there.
const leftFetch = isOrg.value
    ? await useApiFetch<any>(
        `/api/${props.entityType}/${props.entityId}/top`,
        {
            query: computed(() => ({ slice: 'left', days: leftPeriod.value })),
            watch: [leftPeriod],
            getCachedData: cachedPayload,
        },
    )
    : null

const { data: rightData, pending: rightPending } = await useApiFetch<any>(
    `/api/${props.entityType}/${props.entityId}/top`,
    {
        query: computed(() => ({ slice: 'right', days: rightPeriod.value })),
        watch: [rightPeriod],
        getCachedData: cachedPayload,
    },
)

const leftData = computed(() => leftFetch?.data.value || null)
const leftPending = computed(() => leftFetch?.pending.value ?? false)
const left = computed(() => leftData.value || {})
const right = computed(() => rightData.value || {})

/** Format a unix timestamp as a relative "Xd ago" string for the recent-members list. */
const recentMembersFormatted = computed(() => {
    const rows: { id: number; name: string; count: number }[] = left.value?.recentMembers || []
    return rows
})
const formatTimeAgo = (entry: { count: number }): string => {
    const seconds = Math.max(0, Math.floor(Date.now() / 1000) - entry.count)
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    if (minutes < 60) return `${minutes}m`
    const hours = Math.floor(minutes / 60)
    if (hours < 24) return `${hours}h`
    const days = Math.floor(hours / 24)
    if (days < 30) return `${days}d`
    const months = Math.floor(days / 30)
    if (months < 12) return `${months}mo`
    const years = Math.floor(days / 365)
    return `${years}y`
}
</script>

<template>
    <div :class="isOrg ? 'grid grid-cols-1 xl:grid-cols-2 gap-4' : 'space-y-3'">
        <!-- ── LEFT slice — members & lifetime (corp/alliance only) ── -->
        <section v-if="isOrg" class="space-y-3">
            <div class="glass-panel p-3 flex items-center justify-between flex-wrap gap-2">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Members & Engagement</div>
                <div class="flex gap-1 flex-wrap">
                    <button
                        v-for="period in leftPeriods"
                        :key="period.value"
                        class="px-2.5 py-1 text-fine rounded font-medium transition-colors"
                        :class="leftPeriod === period.value
                            ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                            : 'text-gray-500 hover:text-blue-400 border border-transparent'"
                        @click="leftPeriod = period.value"
                    >
                        {{ period.label }}
                    </button>
                </div>
            </div>

            <div v-if="leftPending && !leftData" class="flex items-center justify-center py-12">
                <Icon name="lucide:loader-2" class="w-6 h-6 text-gray-500 animate-spin" />
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <TopBox
                    v-if="left.charactersByKills?.length"
                    title="Top Members (Kills)"
                    data-type="characters"
                    :entries="left.charactersByKills"
                    count-color="text-isk/70"
                    hide-footer
                />
                <TopBox
                    v-if="left.charactersByPoints?.length"
                    title="Top Members (Points)"
                    data-type="characters"
                    :entries="left.charactersByPoints"
                    count-color="text-isk/70"
                    hide-footer
                />
                <TopBox
                    v-if="left.charactersByIsk?.length"
                    title="Top Members (ISK Destroyed)"
                    data-type="characters"
                    :entries="left.charactersByIsk"
                    count-color="text-isk/70"
                    hide-footer
                    :format-count="(e) => formatIsk(e.count)"
                />
                <TopBox
                    v-if="left.soloKillers?.length"
                    title="Top Solo Killers"
                    data-type="characters"
                    :entries="left.soloKillers"
                    count-color="text-isk/70"
                    hide-footer
                />

                <TopBox
                    v-if="entityType === 'alliance' && left.corporationsByKills?.length"
                    title="Top Corporations (Kills)"
                    data-type="corporations"
                    :entries="left.corporationsByKills"
                    count-color="text-isk/70"
                    hide-footer
                />

                <!-- Lifetime widgets (period-independent) -->
                <TopBox
                    v-if="left.achievementPoints?.length"
                    title="Top Achievement Points (Lifetime)"
                    data-type="characters"
                    :entries="left.achievementPoints"
                    count-color="text-purple-400/70"
                    hide-footer
                />
                <TopBox
                    v-if="entityType === 'corporation' && recentMembersFormatted.length"
                    title="Latest Members to Join (Lifetime)"
                    data-type="characters"
                    :entries="recentMembersFormatted"
                    count-color="text-gray-400"
                    hide-footer
                    :format-count="formatTimeAgo"
                />
            </div>
        </section>

        <!-- ── RIGHT slice — activity widgets (ships / locations / killed-corp+alliance) ── -->
        <section class="space-y-3">
            <div class="glass-panel p-3 flex items-center justify-between flex-wrap gap-2">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Activity & Locations</div>
                <div class="flex gap-1 flex-wrap">
                    <button
                        v-for="period in rightPeriods"
                        :key="period.value"
                        class="px-2.5 py-1 text-fine rounded font-medium transition-colors"
                        :class="rightPeriod === period.value
                            ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                            : 'text-gray-500 hover:text-blue-400 border border-transparent'"
                        @click="rightPeriod = period.value"
                    >
                        {{ period.label }}
                    </button>
                </div>
            </div>

            <div v-if="rightPending && !rightData" class="flex items-center justify-center py-12">
                <Icon name="lucide:loader-2" class="w-6 h-6 text-gray-500 animate-spin" />
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <TopBox
                    v-if="right.shipsUsed?.length"
                    title="Most Used Ships"
                    data-type="ships"
                    :entries="right.shipsUsed"
                    count-color="text-isk/70"
                    hide-footer
                />
                <TopBox
                    v-if="right.systems?.length"
                    title="Most Fought-In Systems"
                    data-type="systems"
                    :entries="right.systems"
                    count-color="text-isk/70"
                    hide-footer
                />
                <TopBox
                    v-if="right.constellations?.length"
                    title="Most Fought-In Constellations"
                    data-type="constellations"
                    :entries="right.constellations"
                    count-color="text-isk/70"
                    hide-footer
                />
                <TopBox
                    v-if="right.regions?.length"
                    title="Most Fought-In Regions"
                    data-type="regions"
                    :entries="right.regions"
                    count-color="text-isk/70"
                    hide-footer
                />

                <TopBox
                    v-if="right.killedCorporations?.length"
                    title="Most Killed Corporations"
                    data-type="corporations"
                    :entries="right.killedCorporations"
                    count-color="text-isk/70"
                    hide-footer
                />
                <TopBox
                    v-if="right.killedAlliances?.length"
                    title="Most Killed Alliances"
                    data-type="alliances"
                    :entries="right.killedAlliances"
                    count-color="text-isk/70"
                    hide-footer
                />

                <TopBox
                    v-if="right.killedByCorporations?.length"
                    title="Most Killed By (Corporations)"
                    data-type="corporations"
                    :entries="right.killedByCorporations"
                    count-color="text-red-400/70"
                    hide-footer
                />
                <TopBox
                    v-if="right.killedByAlliances?.length"
                    title="Most Killed By (Alliances)"
                    data-type="alliances"
                    :entries="right.killedByAlliances"
                    count-color="text-red-400/70"
                    hide-footer
                />
            </div>
        </section>
    </div>
</template>
