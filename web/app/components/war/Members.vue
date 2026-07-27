<script setup lang="ts">
interface EntityOption {
    id: number
    name: string
    type: 'corporation' | 'alliance'
    side: 'aggressor' | 'defender'
}

const props = withDefaults(defineProps<{
    endpoint: string
    entityOptions?: EntityOption[]
    side1Label?: string
    side2Label?: string
    extraParams?: Record<string, string | number>
    emptyLabel?: string
}>(), {
    entityOptions: () => [],
    side1Label: 'Aggressor',
    side2Label: 'Defender',
    extraParams: () => ({}),
    emptyLabel: 'No members found',
})

type SortKey = 'activity' | 'kills' | 'losses' | 'isk'
type Side = 'combined' | 'aggressor' | 'defender'
const sortBy = ref<SortKey>('activity')
const sideFilter = ref<Side>('combined')
const corpFilter = ref<number | null>(null)
const allyFilter = ref<number | null>(null)

const fetchParams = computed(() => ({
    ...props.extraParams,
    side: sideFilter.value,
    sort: sortBy.value,
    ...(corpFilter.value ? { corporationId: corpFilter.value } : {}),
    ...(allyFilter.value ? { allianceId: allyFilter.value } : {}),
}))

const { data, pending } = await useApiFetch<any>(() => props.endpoint, {
    params: fetchParams,
    watch: [fetchParams],
    lazy: true,
})

const members = computed(() => data.value?.members || [])

const filteredOptions = computed(() => {
    if (sideFilter.value === 'combined') return props.entityOptions
    return props.entityOptions.filter(o => o.side === sideFilter.value)
})

const clearFilter = () => { corpFilter.value = null; allyFilter.value = null }

const setSide = (s: Side) => {
    if (sideFilter.value === s) return
    sideFilter.value = s
    // Drop entity filter if it no longer matches the selected side
    if (s !== 'combined') {
        const match = props.entityOptions.find(o => (corpFilter.value && o.type === 'corporation' && o.id === corpFilter.value) || (allyFilter.value && o.type === 'alliance' && o.id === allyFilter.value))
        if (match && match.side !== s) { corpFilter.value = null; allyFilter.value = null }
    }
}

const selectEntity = (o: EntityOption) => {
    if (o.type === 'corporation') {
        corpFilter.value = corpFilter.value === o.id ? null : o.id
        allyFilter.value = null
    } else {
        allyFilter.value = allyFilter.value === o.id ? null : o.id
        corpFilter.value = null
    }
}

const isSelected = (o: EntityOption): boolean =>
    (o.type === 'corporation' && corpFilter.value === o.id)
    || (o.type === 'alliance' && allyFilter.value === o.id)


const sideColor = (side: string): string =>
    side === 'aggressor' ? 'bg-red-500' : side === 'defender' ? 'bg-blue-500' : 'bg-gray-500'

const sideTextColor = (side: string): string =>
    side === 'aggressor' ? 'text-red-400' : side === 'defender' ? 'text-blue-400' : 'text-gray-400'
</script>

<template>
    <div>
        <!-- Filter bar -->
        <div class="flex flex-wrap items-center gap-2 mb-4">
            <span class="text-fine text-gray-500 uppercase tracking-wider">Side:</span>
            <button
                v-for="s in [
                    { key: 'combined', label: 'Both', activeClass: 'bg-white/10 text-white border border-white/20' },
                    { key: 'aggressor', label: side1Label, activeClass: 'bg-red-500/20 text-red-400 border border-red-500/30' },
                    { key: 'defender', label: side2Label, activeClass: 'bg-blue-500/20 text-blue-400 border border-blue-500/30' },
                ]" :key="s.key"
                class="px-2.5 py-1 text-fine rounded-md font-medium transition-colors"
                :class="sideFilter === s.key ? s.activeClass : 'text-gray-400 border border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="setSide(s.key as any)">
                {{ s.label }}
            </button>

            <template v-if="entityOptions.length > 0">
                <div class="w-px h-5 bg-white/[0.08] mx-1"></div>

                <span class="text-fine text-gray-500 uppercase tracking-wider">Filter:</span>
                <button
                    class="px-2.5 py-1 text-fine rounded-md font-medium transition-colors"
                    :class="(corpFilter === null && allyFilter === null)
                        ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                        : 'text-gray-400 border border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    @click="clearFilter">
                    All
                </button>

                <button v-for="o in filteredOptions" :key="`${o.type}-${o.id}`"
                    class="flex items-center gap-1.5 px-2.5 py-1 text-fine rounded-md font-medium transition-colors"
                    :class="isSelected(o)
                        ? (o.side === 'aggressor' ? 'bg-red-500/20 text-red-400 border border-red-500/30' : 'bg-blue-500/20 text-blue-400 border border-blue-500/30')
                        : 'text-gray-400 border border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    @click="selectEntity(o)">
                    <Icon :name="o.type === 'alliance' ? 'lucide:landmark' : 'lucide:building-2'" class="w-3 h-3" />
                    <span class="truncate max-w-[160px]">{{ o.name }}</span>
                </button>
            </template>

            <div class="w-px h-5 bg-white/[0.08] mx-1"></div>

            <span class="text-fine text-gray-500 uppercase tracking-wider">Sort:</span>
            <button
                v-for="s in [
                    { key: 'activity', label: 'Activity' },
                    { key: 'kills', label: 'Kills' },
                    { key: 'losses', label: 'Losses' },
                    { key: 'isk', label: 'ISK' },
                ]" :key="s.key"
                class="px-2.5 py-1 text-fine rounded-md font-medium transition-colors"
                :class="sortBy === s.key ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' : 'text-gray-400 border border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="sortBy = s.key as any">
                {{ s.label }}
            </button>
        </div>

        <!-- Loading -->
        <div v-if="pending && members.length === 0" class="flex items-center justify-center py-20">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <!-- Table -->
        <div v-else class="glass-panel overflow-x-auto"
             :class="{ 'opacity-60': pending }">
            <table class="w-full text-xs">
                <thead class="text-gray-500 text-left">
                    <tr class="border-b border-white/[0.08]">
                        <th class="py-2 px-3 font-medium">Character</th>
                        <th class="py-2 px-2 font-medium hidden md:table-cell">Corp / Alliance</th>
                        <th class="py-2 px-2 font-medium">Top Ship</th>
                        <th class="py-2 px-2 font-medium text-right w-[70px]">Kills</th>
                        <th class="py-2 px-2 font-medium text-right w-[70px]">Losses</th>
                        <th class="py-2 px-2 font-medium text-right w-[110px] hidden sm:table-cell">ISK Dest.</th>
                        <th class="py-2 px-3 font-medium text-right w-[110px] hidden sm:table-cell">ISK Lost</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="m in members" :key="`${m.character_id}-${m.side}`"
                        class="border-b border-white/[0.03] hover:bg-white/[0.02]">
                        <td class="py-1.5 px-3">
                            <NuxtLink :to="`/character/${m.character_id}`" class="flex items-center gap-2 min-w-0">
                                <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 mt-0.5" :class="sideColor(m.side)"></div>
                                <img :src="`/images/characters/${m.character_id}/portrait?size=64`"
                                    class="w-7 h-7 rounded flex-shrink-0" loading="lazy">
                                <div class="min-w-0">
                                    <div class="text-gray-200 hover:text-blue-400 truncate font-medium">{{ m.character_name }}</div>
                                    <div class="text-fine truncate md:hidden" :class="sideTextColor(m.side)">
                                        {{ m.alliance_name || m.corporation_name || '—' }}
                                    </div>
                                </div>
                            </NuxtLink>
                        </td>
                        <td class="py-1.5 px-2 hidden md:table-cell min-w-0">
                            <div class="flex flex-col min-w-0">
                                <NuxtLink v-if="m.corporation_id" :to="`/corporation/${m.corporation_id}`"
                                    class="text-gray-400 hover:text-blue-400 truncate">
                                    {{ m.corporation_name || `Corp ${m.corporation_id}` }}
                                </NuxtLink>
                                <NuxtLink v-if="m.alliance_id" :to="`/alliance/${m.alliance_id}`"
                                    class="text-fine text-gray-500 hover:text-blue-400 truncate">
                                    {{ m.alliance_name || `Alliance ${m.alliance_id}` }}
                                </NuxtLink>
                            </div>
                        </td>
                        <td class="py-1.5 px-2">
                            <div v-if="m.top_ship_type_id" class="flex items-center gap-1.5 min-w-0">
                                <img :src="`/images/types/${m.top_ship_type_id}/icon?size=32`"
                                    class="w-5 h-5 rounded flex-shrink-0" loading="lazy">
                                <span class="text-gray-400 truncate">{{ m.top_ship_name }}</span>
                                <span class="text-fine text-gray-600 tabular-nums">×{{ m.top_ship_count }}</span>
                            </div>
                            <span v-else class="text-gray-700">—</span>
                        </td>
                        <td class="py-1.5 px-2 text-right tabular-nums"
                            :class="m.kills > 0 ? 'text-green-400/80' : 'text-gray-700'">
                            {{ m.kills > 0 ? formatNumber(m.kills) : '—' }}
                        </td>
                        <td class="py-1.5 px-2 text-right tabular-nums"
                            :class="m.losses > 0 ? 'text-red-400/80' : 'text-gray-700'">
                            {{ m.losses > 0 ? formatNumber(m.losses) : '—' }}
                        </td>
                        <td class="py-1.5 px-2 text-right tabular-nums hidden sm:table-cell"
                            :class="m.isk_destroyed > 0 ? 'text-isk/70' : 'text-gray-700'">
                            {{ m.isk_destroyed > 0 ? formatIsk(m.isk_destroyed) : '—' }}
                        </td>
                        <td class="py-1.5 px-3 text-right tabular-nums hidden sm:table-cell"
                            :class="m.isk_lost > 0 ? 'text-red-400/60' : 'text-gray-700'">
                            {{ m.isk_lost > 0 ? formatIsk(m.isk_lost) : '—' }}
                        </td>
                    </tr>
                </tbody>
            </table>
            <div v-if="members.length === 0 && !pending"
                class="py-12 text-center text-sm text-gray-500">
                {{ emptyLabel }}
            </div>
            <div v-else-if="data?.count >= data?.limit"
                class="py-2 text-center text-fine text-gray-600">
                Showing top {{ data.limit }} — more participants exist
            </div>
        </div>
    </div>
</template>
