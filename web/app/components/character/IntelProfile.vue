<script setup lang="ts">
const props = defineProps<{
    characterId: number
    days: number
}>()

const fetchParams = computed(() => ({ days: props.days }))
const { data, pending } = await useApiFetch<any>(`/api/character/${props.characterId}/intel`, {
    params: fetchParams,
    watch: [() => props.days],
})

const playstyle = computed(() => data.value?.playstyle)
const fc = computed(() => data.value?.fc)
const shipsFlown = computed(() => data.value?.ships_flown || [])
const shipsLost = computed(() => data.value?.ships_lost || [])
const targets = computed(() => data.value?.targets || [])
const fleetPartners = computed(() => data.value?.fleet_partners || [])
const bridgeScore = computed(() => data.value?.bridge_score || 0)
const dominantStyle = computed(() => data.value?.dominant_style || 'Unknown')
const capitalPilot = computed(() => data.value?.capital_pilot || false)
const isLogi = computed(() => data.value?.is_logi || false)
const awoxKills = computed(() => data.value?.awox_kills || 0)
const cynoDeaths = computed(() => data.value?.cyno_deaths || 0)
const baitLevel = computed(() => data.value?.bait || 'None')
const baitCount = computed(() => data.value?.bait_count || 0)

const playstyleBars = computed(() => {
    if (!playstyle.value) return []
    return [
        { label: 'Solo', pct: playstyle.value.solo, color: 'bg-blue-500' },
        { label: 'Small', pct: playstyle.value.small_gang, color: 'bg-cyan-500' },
        { label: 'Mid', pct: playstyle.value.mid_gang, color: 'bg-green-500' },
        { label: 'Fleet', pct: playstyle.value.fleet, color: 'bg-yellow-500' },
        { label: 'Blob', pct: playstyle.value.blob, color: 'bg-red-500' },
    ].filter(b => b.pct > 0)
})

const fcColor = computed(() => {
    switch (fc.value?.likelihood) {
        case 'High': return 'text-red-400'
        case 'Medium': return 'text-yellow-400'
        case 'Low': return 'text-gray-400'
        default: return 'text-gray-600'
    }
})

</script>

<template>
    <div class="glass-panel p-4">
        <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">
            Intel Profile
        </div>

        <div v-if="pending" class="flex items-center justify-center py-6">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <template v-else-if="data">
            <!-- Playstyle bar -->
            <div class="mb-3">
                <div class="flex items-center justify-between mb-1.5">
                    <span class="text-xs text-gray-400">Playstyle</span>
                    <span class="text-xs text-white font-medium">{{ dominantStyle }}
                        <span class="text-gray-500 font-normal">({{ playstyle?.total_kills || 0 }} kills)</span>
                    </span>
                </div>
                <div class="flex h-2 rounded-full overflow-hidden gap-px">
                    <div
                        v-for="bar in playstyleBars"
                        :key="bar.label"
                        :class="bar.color"
                        :style="{ width: `${bar.pct}%` }"
                        class="rounded-sm transition-all"
                        v-tooltip="`${bar.label}: ${bar.pct}%`"
                    />
                </div>
                <div class="flex flex-wrap gap-x-3 gap-y-0.5 mt-1.5">
                    <span v-for="bar in playstyleBars" :key="bar.label" class="text-fine text-gray-500 flex items-center gap-1">
                        <span :class="bar.color" class="inline-block w-1.5 h-1.5 rounded-full"></span>
                        {{ bar.label }} {{ bar.pct }}%
                    </span>
                </div>
            </div>

            <!-- Role tags -->
            <div class="flex flex-wrap gap-1.5 mb-3">
                <span class="px-2 py-0.5 rounded text-fine font-medium bg-white/[0.06] text-gray-300">
                    Avg Fleet: {{ playstyle?.avg_fleet_size || '-' }}
                </span>
                <span v-if="fc?.likelihood !== 'None'" class="px-2 py-0.5 rounded text-fine font-medium"
                    :class="fc?.likelihood === 'High' ? 'bg-red-500/20 text-red-400' : 'bg-yellow-500/15 text-yellow-400'">
                    FC: {{ fc?.likelihood }}
                </span>
                <span v-if="capitalPilot" class="px-2 py-0.5 rounded text-fine font-medium bg-yellow-500/15 text-yellow-400">
                    Capital Pilot
                </span>
                <span v-if="isLogi" class="px-2 py-0.5 rounded text-fine font-medium bg-green-500/15 text-green-400">
                    Logi Pilot
                </span>
                <span v-if="cynoDeaths > 0" class="px-2 py-0.5 rounded text-fine font-medium bg-purple-500/15 text-purple-400">
                    Cyno Alt ({{ cynoDeaths }} deaths)
                </span>
                <span v-if="baitLevel !== 'None'" class="px-2 py-0.5 rounded text-fine font-medium"
                    :class="baitLevel === 'High' ? 'bg-red-500/20 text-red-400' : 'bg-orange-500/15 text-orange-400'">
                    Bait ({{ baitCount }}x)
                </span>
                <span v-if="awoxKills > 0" class="px-2 py-0.5 rounded text-fine font-medium bg-red-500/20 text-red-400">
                    Awox: {{ awoxKills }} kills
                </span>
            </div>

            <!-- Ships flown + lost -->
            <div v-if="shipsFlown.length > 0 || shipsLost.length > 0" class="grid grid-cols-2 gap-3 mb-3">
                <TopBox v-if="shipsFlown.length > 0" title="Typically Flies" data-type="ships" :entries="shipsFlown" count-color="text-isk/70" hide-footer />
                <TopBox v-if="shipsLost.length > 0" title="Typically Loses" data-type="ships" :entries="shipsLost" count-color="text-red-400/70" hide-footer />
            </div>

            <!--
                Targets (alliances).

                "Groups Flown With" used to sit beside this. It answered the same
                question as the dashboard's Flies With (Alliances) panel but from
                the graph rather than the attacker rows, so the two appeared a few
                hundred pixels apart naming the same alliances in a different
                order with different counts (337 against 53 for one pilot). The
                dashboard pair wins because it covers corporations as well and
                follows the period selector, which this did not.
            -->
            <div v-if="targets.length > 0" class="mb-3">
                <TopBox title="Targets (Alliances)" data-type="alliances" :entries="targets" count-color="text-red-400/70" hide-footer />
            </div>

            <!-- Fleet partners with corp/alliance info -->
            <div v-if="fleetPartners.length > 0" class="bg-white/[0.03] rounded-lg p-3">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-400 mb-2">Top Fleet Partners</div>
                <div class="space-y-1.5">
                    <div v-for="p in fleetPartners" :key="p.id" class="flex items-center gap-2 text-xs">
                        <img
                            :src="`/images/characters/${p.id}/portrait?size=64`"
                            class="w-6 h-6 rounded"
                            loading="lazy"
                        />
                        <NuxtLink :to="`/character/${p.id}`" class="text-blue-400 hover:text-blue-300 truncate flex-shrink-0" style="max-width: 120px">
                            {{ p.name }}
                        </NuxtLink>
                        <span v-if="p.corp_name || p.alliance_name" class="text-gray-600 truncate">
                            <span v-if="p.corp_name" class="text-gray-500">{{ p.corp_name }}</span>
                            <span v-if="p.alliance_name" class="text-gray-600"> [{{ p.alliance_name }}]</span>
                        </span>
                        <span class="ml-auto text-isk/70 tabular-nums flex-shrink-0">{{ p.count }}</span>
                    </div>
                </div>
            </div>
        </template>

        <div v-else class="text-fine text-gray-600 italic py-4 text-center">No intel data available</div>
    </div>
</template>
