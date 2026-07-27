<script setup lang="ts">
const props = defineProps<{
    systemId?: number
    constellationId?: number
    regionId?: number
    allianceId?: number
    corporationId?: number
    characterId?: number
}>()

const page = ref(1)

const fetchParams = computed(() => {
    const p: Record<string, any> = { page: page.value, limit: 50 }
    if (props.systemId) p.systemId = props.systemId
    if (props.constellationId) p.constellationId = props.constellationId
    if (props.regionId) p.regionId = props.regionId
    if (props.allianceId) p.allianceId = props.allianceId
    if (props.corporationId) p.corporationId = props.corporationId
    if (props.characterId) p.characterId = props.characterId
    return p
})

const { data, pending } = await useApiFetch<any>('/api/battles', {
    params: fetchParams,
    watch: [page],
})

const battles = computed(() => data.value?.battles || [])

const formatDuration = (mins: number): string => {
    if (mins < 60) return `${mins}m`
    const h = Math.floor(mins / 60)
    const m = mins % 60
    return m > 0 ? `${h}h ${m}m` : `${h}h`
}

const securityColor = (sec: number | null): string => {
    if (sec == null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-green-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

const teamImage = (entry: any): string => {
    if (entry.alliance_id) return `/images/alliances/${entry.alliance_id}/logo?size=64`
    if (entry.corporation_id) return `/images/corporations/${entry.corporation_id}/logo?size=64`
    return ''
}

const teamName = (entry: any): string =>
    entry.alliance_name || entry.corporation_name || 'Unknown'
</script>

<template>
    <div>
        <div v-if="pending && battles.length === 0" class="flex items-center justify-center py-20">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3" :class="{ 'opacity-60': pending }">
            <NuxtLink v-for="b in battles" :key="b.battle_id" :to="`/battle/${b.battle_id}`"
                class="block rounded-lg bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.06] transition-colors overflow-hidden">

                <div class="flex items-center justify-between px-3 py-2 border-b border-white/[0.06] bg-white/[0.02]">
                    <div class="flex items-center gap-2 text-xs min-w-0">
                        <EveImage :src="`/images/systems/${b.solar_system_id}?size=32`" :size="32" alt="" class="w-5 h-5 rounded flex-shrink-0" />
                        <span class="text-white font-medium truncate" :class="pochvenClass(b.region_id)">{{ b.solar_system_name }}</span>
                        <span class="tabular-nums flex-shrink-0" :class="securityColor(b.solar_system_security)">
                            {{ b.solar_system_security?.toFixed(1) }}
                        </span>
                        <span class="text-gray-600">·</span>
                        <span class="text-gray-500 truncate" :class="pochvenClass(b.region_id)">{{ b.region_name }}</span>
                    </div>
                    <div class="text-fine text-gray-500 tabular-nums flex-shrink-0 ml-2">{{ formatDate(b.start_time) }}</div>
                </div>

                <div class="p-3 space-y-2">
                    <div v-if="b.teams[0]" class="flex items-center gap-2">
                        <div class="w-1 self-stretch rounded-full bg-red-500/40 flex-shrink-0"></div>
                        <div class="flex -space-x-1.5 flex-shrink-0">
                            <EveImage v-for="e in b.teams[0].top_alliances.slice(0, 3)" :key="e.alliance_id || e.corporation_id"
                                :src="teamImage(e)" v-tooltip="teamName(e)"
                                :size="32"
                                :alt="teamName(e)"
                                class="w-8 h-8 rounded border-2 border-[#151515] bg-gray-900" />
                        </div>
                        <div class="min-w-0 flex-1">
                            <div class="text-xs text-gray-200 truncate">
                                {{ b.teams[0].top_alliances[0] ? teamName(b.teams[0].top_alliances[0]) : 'Team 1' }}
                                <span v-if="b.teams[0].alliance_count > 1" class="text-gray-500">+{{ b.teams[0].alliance_count - 1 }}</span>
                            </div>
                            <div class="text-fine text-gray-500">
                                <span class="text-isk/70 tabular-nums">{{ b.teams[0].total_kills }} kills</span>
                                <span class="text-gray-600 mx-0.5">·</span>
                                <span class="text-isk/70 tabular-nums">{{ formatIsk(b.teams[0].total_isk_destroyed) }}</span>
                            </div>
                        </div>
                    </div>

                    <div class="flex items-center gap-2 px-1">
                        <div v-if="b.teams.length === 2" class="flex-1 h-1.5 bg-blue-500/20 rounded-full overflow-hidden">
                            <div class="h-full bg-red-500 rounded-full"
                                :style="{ width: `${b.teams[0].total_isk_destroyed + b.teams[1].total_isk_destroyed > 0 ? Math.round(b.teams[0].total_isk_destroyed / (b.teams[0].total_isk_destroyed + b.teams[1].total_isk_destroyed) * 100) : 50}%` }">
                            </div>
                        </div>
                        <div class="flex items-center gap-1.5 text-fine text-gray-500 flex-shrink-0">
                            <span class="tabular-nums">{{ b.kill_count }} kills</span>
                            <span class="text-gray-600">·</span>
                            <span class="tabular-nums">{{ formatIsk(b.total_isk_destroyed) }}</span>
                            <span class="text-gray-600">·</span>
                            <span class="tabular-nums">{{ formatDuration(b.duration_minutes) }}</span>
                        </div>
                    </div>

                    <div v-if="b.teams[1]" class="flex items-center gap-2">
                        <div class="w-1 self-stretch rounded-full bg-blue-500/40 flex-shrink-0"></div>
                        <div class="flex -space-x-1.5 flex-shrink-0">
                            <EveImage v-for="e in b.teams[1].top_alliances.slice(0, 3)" :key="e.alliance_id || e.corporation_id"
                                :src="teamImage(e)" v-tooltip="teamName(e)"
                                :size="32"
                                :alt="teamName(e)"
                                class="w-8 h-8 rounded border-2 border-[#151515] bg-gray-900" />
                        </div>
                        <div class="min-w-0 flex-1">
                            <div class="text-xs text-gray-200 truncate">
                                {{ b.teams[1].top_alliances[0] ? teamName(b.teams[1].top_alliances[0]) : 'Team 2' }}
                                <span v-if="b.teams[1].alliance_count > 1" class="text-gray-500">+{{ b.teams[1].alliance_count - 1 }}</span>
                            </div>
                            <div class="text-fine text-gray-500">
                                <span class="text-isk/70 tabular-nums">{{ b.teams[1].total_kills }} kills</span>
                                <span class="text-gray-600 mx-0.5">·</span>
                                <span class="text-isk/70 tabular-nums">{{ formatIsk(b.teams[1].total_isk_destroyed) }}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <div v-if="b.is_multi_party || b.kill_count >= 100 || b.total_isk_destroyed >= 1e12" class="flex items-center gap-1.5 px-3 pb-2">
                    <span v-if="b.is_multi_party" class="px-1.5 py-0.5 text-fine rounded bg-amber-500/20 text-amber-400 font-medium">Multi-party</span>
                    <span v-if="b.kill_count >= 100" class="px-1.5 py-0.5 text-fine rounded bg-red-500/20 text-red-400 font-medium">Large Battle</span>
                    <span v-if="b.total_isk_destroyed >= 1e12" class="px-1.5 py-0.5 text-fine rounded bg-purple-500/20 text-purple-400 font-medium">1T+ ISK</span>
                </div>
            </NuxtLink>

            <div v-if="battles.length === 0" class="md:col-span-2 py-12 text-center text-sm text-gray-500">No battles found</div>
        </div>

        <div v-if="battles.length >= 50 || page > 1" class="flex justify-center mt-4 gap-2">
            <button @click="page = Math.max(1, page - 1)" :disabled="page <= 1"
                class="px-3 py-1.5 text-xs rounded-md transition-colors"
                :class="page <= 1 ? 'text-gray-700' : 'text-gray-400 hover:text-blue-400 bg-white/[0.04] border border-white/[0.08]'">
                Previous
            </button>
            <span class="px-3 py-1.5 text-xs text-gray-500">Page {{ page }}</span>
            <button @click="page++" :disabled="battles.length < 50"
                class="px-3 py-1.5 text-xs rounded-md transition-colors"
                :class="battles.length < 50 ? 'text-gray-700' : 'text-gray-400 hover:text-blue-400 bg-white/[0.04] border border-white/[0.08]'">
                Next
            </button>
        </div>
    </div>
</template>
