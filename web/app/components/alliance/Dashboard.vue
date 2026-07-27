<script setup lang="ts">
const props = defineProps<{
    allianceId: number
    customDescriptionHtml?: string | null
}>()

const parsedBio = computed(() => props.customDescriptionHtml ?? '')

const periods = [
    { value: 14, label: '14d' },
    { value: 30, label: '30d' },
    { value: 90, label: '90d' },
    { value: 0, label: 'All Time' },
]
const activePeriod = ref(90)

const statsParams = computed(() => ({ days: activePeriod.value }))
const { data: statsData, pending: statsPending } = await useApiFetch<any>(`/api/alliance/${props.allianceId}/stats`, {
    params: statsParams,
    watch: [activePeriod],
    getCachedData: cachedPayload,
})

const stats = computed(() => statsData.value?.stats)
const topShipsUsed = computed(() => (statsData.value?.topShipsUsed || []).map((s: any) => ({ id: s.ship_type_id, name: s.ship_name, count: s.count })))
const topShipsLost = computed(() => (statsData.value?.topShipsLost || []).map((s: any) => ({ id: s.ship_type_id, name: s.ship_name, count: s.count })))
const diesToCorps = computed(() => statsData.value?.diesToCorporations || [])
const diesToAlliances = computed(() => statsData.value?.diesToAlliances || [])

// Intel data (graph-powered, lazy loaded client-side)
const { data: intelData, pending: intelPending } = useApiFetch<any>(`/api/alliance/${props.allianceId}/intel`, {
    lazy: true,
    server: false,
    getCachedData: cachedPayload,
})

const activeMembers = computed(() => intelData.value?.activeMembers)
const census = computed(() => intelData.value?.census)
const allies = computed(() => (intelData.value?.allies || []).map((a: any) => ({ id: a.id, name: a.name, count: a.shared_enemy_kills })))
const enemies = computed(() => (intelData.value?.enemies || []).map((e: any) => ({ id: e.id, name: e.name, count: e.total })))
const huntingGrounds = computed(() => (intelData.value?.huntingGrounds || []).map((s: any) => ({ id: s.id, name: s.name, count: s.active_characters })))
const recentDepartures = computed(() => (intelData.value?.recentDepartures || []).map((d: any) => ({ id: d.id, name: d.name, count: d.left_at ? new Date(d.left_at).getTime() : 0 })))
const recentJoins = computed(() => (intelData.value?.recentJoins || []).map((j: any) => ({ id: j.id, name: j.name, count: j.joined_at ? new Date(j.joined_at).getTime() : 0 })))


const formatTimeAgo = (entry: { id: number; name: string; count: number }): string => {
    if (!entry.count) return ''
    const diff = Date.now() - entry.count
    const mins = Math.floor(diff / 60000)
    if (mins < 60) return `${mins}m ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    if (days < 30) return `${days}d ago`
    const months = Math.floor(days / 30)
    if (months < 12) return `${months}mo ago`
    return `${Math.floor(days / 365)}y ago`
}
</script>

<template>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- ===== LEFT: Bio ===== -->
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
                <div v-else class="text-fine text-gray-600 italic">No bio set. The executor corporation's CEO can set one from the settings page.</div>
            </div>
        </div>

        <!-- ===== RIGHT: Everything else ===== -->
        <div class="space-y-4">
            <!-- Active Members + Census -->
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Active Members</div>
                <div v-if="intelPending" class="flex items-center justify-center py-4">
                    <Icon name="lucide:loader-2" class="w-4 h-4 text-gray-500 animate-spin" />
                </div>
                <template v-else-if="activeMembers">
                    <div class="grid grid-cols-3 gap-3 mb-3">
                        <div class="text-center p-2 rounded bg-white/[0.03]">
                            <div class="text-lg font-bold text-blue-400 tabular-nums">{{ formatNumber(activeMembers.days_7) }}</div>
                            <div class="text-fine text-gray-500">7 days</div>
                        </div>
                        <div class="text-center p-2 rounded bg-white/[0.03]">
                            <div class="text-lg font-bold text-blue-400 tabular-nums">{{ formatNumber(activeMembers.days_30) }}</div>
                            <div class="text-fine text-gray-500">30 days</div>
                        </div>
                        <div class="text-center p-2 rounded bg-white/[0.03]">
                            <div class="text-lg font-bold text-blue-400 tabular-nums">{{ formatNumber(activeMembers.days_90) }}</div>
                            <div class="text-fine text-gray-500">90 days</div>
                        </div>
                    </div>
                    <div v-if="census && census.total > 0" class="space-y-1.5">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400 mb-1">Composition</div>
                        <div class="flex flex-wrap gap-1.5">
                            <span class="px-2 py-0.5 rounded text-fine font-medium bg-white/[0.06] text-gray-300">
                                {{ formatNumber(census.total) }} tracked
                            </span>
                            <span v-if="census.fcs > 0" class="px-2 py-0.5 rounded text-fine font-medium bg-yellow-500/15 text-yellow-400">
                                {{ census.fcs }} FCs
                            </span>
                            <span v-if="census.logis > 0" class="px-2 py-0.5 rounded text-fine font-medium bg-green-500/15 text-green-400">
                                {{ census.logis }} Logi
                            </span>
                            <span v-if="census.caps > 0" class="px-2 py-0.5 rounded text-fine font-medium bg-purple-500/15 text-purple-400">
                                {{ census.caps }} Capital Pilots
                            </span>
                        </div>
                    </div>
                </template>
            </div>

            <!-- Period Stats -->
            <div class="glass-panel p-4">
                <div class="flex items-center justify-between mb-4">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Stats</div>
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

                <div v-if="statsPending" class="flex items-center justify-center py-8">
                    <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                </div>

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
                        <span>NPC Losses</span>
                        <span class="text-fine text-white tabular-nums">{{ formatNumber(stats.npc_losses) }}</span>
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

            <!-- Most Used Ships + Most Lost Ships -->
            <div v-if="topShipsUsed.length > 0 || topShipsLost.length > 0" class="grid grid-cols-2 gap-3">
                <TopBox v-if="topShipsUsed.length > 0" title="Most Used Ships" data-type="ships" :entries="topShipsUsed" count-color="text-isk/70" />
                <TopBox v-if="topShipsLost.length > 0" title="Most Lost Ships" data-type="ships" :entries="topShipsLost" count-color="text-red-400/70" />
            </div>

            <!-- Allies + Enemies -->
            <div v-if="!intelPending && (allies.length > 0 || enemies.length > 0)" class="grid grid-cols-2 gap-3">
                <TopBox v-if="allies.length > 0" title="Allies" data-type="alliances" :entries="allies" count-color="text-green-400/70" hide-footer />
                <TopBox v-if="enemies.length > 0" title="Enemies" data-type="alliances" :entries="enemies" count-color="text-red-400/70" hide-footer />
            </div>

            <!-- Hunting Grounds (7d) -->
            <TopBox v-if="!intelPending && huntingGrounds.length > 0" title="Hunting Grounds (7d)" data-type="systems" :entries="huntingGrounds" count-color="text-blue-400/70" hide-footer />

            <!-- Dies To (Corps) + Dies To (Alliances) -->
            <div v-if="diesToCorps.length > 0 || diesToAlliances.length > 0" class="grid grid-cols-2 gap-3">
                <TopBox v-if="diesToCorps.length > 0" title="Dies To (Corps)" data-type="corporations" :entries="diesToCorps" count-color="text-red-400/70" />
                <TopBox v-if="diesToAlliances.length > 0" title="Dies To (Alliances)" data-type="alliances" :entries="diesToAlliances" count-color="text-red-400/70" />
            </div>

            <!-- Recent Departures + Recent Joins -->
            <div v-if="!intelPending && (recentDepartures.length > 0 || recentJoins.length > 0)" class="grid grid-cols-2 gap-3">
                <TopBox v-if="recentDepartures.length > 0" title="Recent Departures" data-type="characters" :entries="recentDepartures" count-color="text-orange-400/70" :format-count="formatTimeAgo" hide-footer />
                <TopBox v-if="recentJoins.length > 0" title="Recent Joins" data-type="characters" :entries="recentJoins" count-color="text-green-400/70" :format-count="formatTimeAgo" hide-footer />
            </div>
        </div>
    </div>
</template>
