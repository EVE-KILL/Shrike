<script setup lang="ts">
const props = withDefaults(defineProps<{
    endpoint: string
    extraParams?: Record<string, string | number>
}>(), {
    extraParams: () => ({}),
})

const fetchParams = computed(() => ({ ...props.extraParams }))

const { data, pending } = await useApiFetch<any>(() => props.endpoint, {
    params: fetchParams,
    watch: [fetchParams],
    lazy: true,
})

const intel = computed(() => data.value)


const securityClass = (sec: number | null): string => {
    if (sec == null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-green-400'
    if (sec > 0) return 'text-yellow-400'
    return 'text-red-400'
}

const formatSecurity = (sec: number | null): string => {
    if (sec == null) return '?'
    return sec.toFixed(1)
}

const secClassLabel = (sc: string): string => {
    if (sc === 'highsec') return 'High-sec'
    if (sc === 'lowsec') return 'Low-sec'
    if (sc === 'nullsec') return 'Null-sec'
    if (sc === 'wormhole') return 'Wormhole / Pochven'
    return 'Unknown'
}

const secClassColor = (sc: string): string => {
    if (sc === 'highsec') return 'text-green-400'
    if (sc === 'lowsec') return 'text-yellow-400'
    if (sc === 'nullsec') return 'text-red-400'
    if (sc === 'wormhole') return 'text-purple-400'
    return 'text-gray-400'
}
</script>

<template>
    <div v-if="pending && !intel" class="flex items-center justify-center py-20">
        <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
    </div>

    <div v-else-if="intel" class="space-y-4">
        <!-- Summary -->
        <div class="glass-panel p-4">
            <div class="grid grid-cols-2 md:grid-cols-5 gap-3">
                <div class="text-center">
                    <div class="text-lg font-bold text-white tabular-nums">{{ formatNumber(intel.summary.kills) }}</div>
                    <div class="text-fine uppercase tracking-wider text-gray-500">Kills</div>
                </div>
                <div class="text-center">
                    <div class="text-lg font-bold text-green-400 tabular-nums">{{ formatIsk(intel.summary.isk_destroyed) }}</div>
                    <div class="text-fine uppercase tracking-wider text-gray-500">ISK Destroyed</div>
                </div>
                <div class="text-center">
                    <div class="text-lg font-bold text-white tabular-nums">{{ formatNumber(intel.summary.systems) }}</div>
                    <div class="text-fine uppercase tracking-wider text-gray-500">Systems</div>
                </div>
                <div class="text-center">
                    <div class="text-lg font-bold text-white tabular-nums">{{ formatNumber(intel.summary.constellations) }}</div>
                    <div class="text-fine uppercase tracking-wider text-gray-500">Constellations</div>
                </div>
                <div class="text-center">
                    <div class="text-lg font-bold text-white tabular-nums">{{ formatNumber(intel.summary.regions) }}</div>
                    <div class="text-fine uppercase tracking-wider text-gray-500">Regions</div>
                </div>
            </div>
        </div>

        <!-- Security breakdown -->
        <div v-if="intel.security_breakdown?.length > 0"
            class="glass-panel p-4">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Fighting by Security</div>
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                <div v-for="b in intel.security_breakdown" :key="b.sec_class"
                    class="rounded-md bg-white/[0.03] border border-white/[0.04] p-3">
                    <div class="text-fine uppercase tracking-wider mb-1" :class="secClassColor(b.sec_class)">
                        {{ secClassLabel(b.sec_class) }}
                    </div>
                    <div class="text-base font-bold text-white tabular-nums">{{ formatNumber(b.kills) }} kills</div>
                    <div class="text-fine text-isk/70 tabular-nums">{{ formatIsk(b.isk_destroyed) }}</div>
                </div>
            </div>
        </div>

        <!-- Location panels -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <!-- Top Systems -->
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">
                    Top Systems <span class="text-gray-600">({{ intel.top_systems.length }})</span>
                </div>
                <div class="space-y-px">
                    <NuxtLink v-for="(s, i) in intel.top_systems" :key="s.system_id"
                        :to="`/system/${s.system_id}`"
                        class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                        <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(i) + 1 }}</span>
                        <div class="min-w-0 flex-1">
                            <div class="text-xs truncate">
                                <span class="tabular-nums mr-1" :class="securityClass(s.security)">{{ formatSecurity(s.security) }}</span>
                                {{ s.system_name }}
                            </div>
                            <div v-if="s.region_name" class="text-fine text-gray-600 truncate">{{ s.region_name }}</div>
                        </div>
                        <span class="text-xs tabular-nums text-gray-400">{{ formatNumber(s.kills) }}</span>
                    </NuxtLink>
                    <div v-if="intel.top_systems.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                </div>
            </div>

            <!-- Top Constellations -->
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">
                    Top Constellations <span class="text-gray-600">({{ intel.top_constellations.length }})</span>
                </div>
                <div class="space-y-px">
                    <div v-for="(c, i) in intel.top_constellations" :key="c.constellation_id"
                        class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400">
                        <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(i) + 1 }}</span>
                        <div class="min-w-0 flex-1">
                            <div class="text-xs truncate">{{ c.constellation_name }}</div>
                            <div v-if="c.region_name" class="text-fine text-gray-600 truncate">{{ c.region_name }}</div>
                        </div>
                        <span class="text-xs tabular-nums">{{ formatNumber(c.kills) }}</span>
                    </div>
                    <div v-if="intel.top_constellations.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                </div>
            </div>

            <!-- Top Regions -->
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">
                    Top Regions <span class="text-gray-600">({{ intel.top_regions.length }})</span>
                </div>
                <div class="space-y-px">
                    <NuxtLink v-for="(r, i) in intel.top_regions" :key="r.region_id"
                        :to="`/region/${r.region_id}`"
                        class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                        <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(i) + 1 }}</span>
                        <div class="min-w-0 flex-1">
                            <div class="text-xs truncate">{{ r.region_name }}</div>
                            <div class="text-fine text-isk/60 tabular-nums">{{ formatIsk(r.isk_destroyed) }}</div>
                        </div>
                        <span class="text-xs tabular-nums">{{ formatNumber(r.kills) }}</span>
                    </NuxtLink>
                    <div v-if="intel.top_regions.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                </div>
            </div>
        </div>

        <!-- Ship panels -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <!-- Ships Destroyed -->
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-red-400/80 mb-3">
                    Ships Destroyed <span class="text-gray-600">({{ intel.ships_destroyed.length }})</span>
                </div>
                <div class="space-y-px">
                    <div v-for="(s, i) in intel.ships_destroyed" :key="s.ship_type_id"
                        class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400">
                        <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(i) + 1 }}</span>
                        <img :src="`/images/types/${s.ship_type_id}/icon?size=32`"
                            class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                        <div class="min-w-0 flex-1">
                            <div class="text-xs truncate">{{ s.ship_name }}</div>
                            <div v-if="s.group_name" class="text-fine text-gray-600 truncate">{{ s.group_name }}</div>
                        </div>
                        <span class="text-xs tabular-nums">{{ formatNumber(s.count) }}</span>
                    </div>
                    <div v-if="intel.ships_destroyed.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                </div>
            </div>

            <!-- Ships Used -->
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-green-400/80 mb-3">
                    Ships Used by Attackers <span class="text-gray-600">({{ intel.ships_used.length }})</span>
                </div>
                <div class="space-y-px">
                    <div v-for="(s, i) in intel.ships_used" :key="s.ship_type_id"
                        class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400">
                        <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(i) + 1 }}</span>
                        <img :src="`/images/types/${s.ship_type_id}/icon?size=32`"
                            class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                        <div class="min-w-0 flex-1">
                            <div class="text-xs truncate">{{ s.ship_name }}</div>
                            <div v-if="s.group_name" class="text-fine text-gray-600 truncate">{{ s.group_name }}</div>
                        </div>
                        <span class="text-xs tabular-nums">{{ formatNumber(s.count) }}</span>
                    </div>
                    <div v-if="intel.ships_used.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                </div>
            </div>

            <!-- Ship Groups Destroyed -->
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-amber-400/80 mb-3">
                    Ship Classes Destroyed <span class="text-gray-600">({{ intel.ship_groups_destroyed.length }})</span>
                </div>
                <div class="space-y-px">
                    <div v-for="(g, i) in intel.ship_groups_destroyed" :key="g.group_id"
                        class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400">
                        <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ Number(i) + 1 }}</span>
                        <div class="min-w-0 flex-1">
                            <div class="text-xs truncate">{{ g.group_name }}</div>
                            <div class="text-fine text-isk/60 tabular-nums">{{ formatIsk(g.isk_destroyed) }}</div>
                        </div>
                        <span class="text-xs tabular-nums">{{ formatNumber(g.count) }}</span>
                    </div>
                    <div v-if="intel.ship_groups_destroyed.length === 0" class="py-4 text-center text-fine text-gray-600">No data</div>
                </div>
            </div>
        </div>
    </div>
</template>
