<script setup lang="ts">
interface EligibleEntry {
    id: number
    name: string
    ticker: string
    member_count: number
    alliance_id?: number | null
    alliance_name?: string | null
    corporation_count?: number
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
}

const props = defineProps<{
    entries: EligibleEntry[]
    type: 'corporations' | 'alliances'
}>()

type SortKey = 'name' | 'members' | 'kills' | 'losses' | 'isk_destroyed' | 'isk_lost'
const sortKey = ref<SortKey>('members')
const sortDir = ref<'asc' | 'desc'>('desc')

function toggleSort(key: SortKey) {
    if (sortKey.value === key) sortDir.value = sortDir.value === 'desc' ? 'asc' : 'desc'
    else { sortKey.value = key; sortDir.value = 'desc' }
}

function sortIcon(key: SortKey): string {
    if (sortKey.value !== key) return 'lucide:chevrons-up-down'
    return sortDir.value === 'desc' ? 'lucide:chevron-down' : 'lucide:chevron-up'
}

const sorted = computed(() => {
    const arr = [...props.entries]
    const dir = sortDir.value === 'asc' ? 1 : -1
    arr.sort((a, b) => {
        let c = 0
        switch (sortKey.value) {
            case 'name': c = a.name.localeCompare(b.name); break
            case 'members': c = a.member_count - b.member_count; break
            case 'kills': c = a.kills - b.kills; break
            case 'losses': c = a.losses - b.losses; break
            case 'isk_destroyed': c = a.isk_destroyed - b.isk_destroyed; break
            case 'isk_lost': c = a.isk_lost - b.isk_lost; break
        }
        return dir * c
    })
    return arr
})

const entityLink = (e: EligibleEntry): string => {
    return props.type === 'alliances' ? `/alliance/${e.id}` : `/corporation/${e.id}`
}

const entityImage = (e: EligibleEntry): string => {
    return props.type === 'alliances'
        ? `/images/alliances/${e.id}/logo?size=64`
        : `/images/corporations/${e.id}/logo?size=64`
}

</script>

<template>
    <div class="glass-panel overflow-x-auto">
        <table class="w-full text-xs">
            <thead class="text-gray-500 text-left">
                <tr class="border-b border-white/[0.08]">
                    <th class="py-2 px-3 font-medium">
                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('name')">
                            {{ type === 'alliances' ? 'Alliance' : 'Corporation' }}
                            <Icon :name="sortIcon('name')" class="text-fine" />
                        </button>
                    </th>
                    <th v-if="type === 'alliances'" class="py-2 px-2 font-medium text-right w-[90px] hidden md:table-cell">Corps</th>
                    <th class="py-2 px-2 font-medium text-right w-[100px]">
                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('members')">
                            Members <Icon :name="sortIcon('members')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 px-2 font-medium text-right w-[80px]">
                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('kills')">
                            Kills <Icon :name="sortIcon('kills')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 px-2 font-medium text-right w-[80px]">
                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('losses')">
                            Losses <Icon :name="sortIcon('losses')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 px-2 font-medium text-right w-[120px] hidden sm:table-cell">
                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('isk_destroyed')">
                            ISK Destroyed <Icon :name="sortIcon('isk_destroyed')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 px-3 font-medium text-right w-[120px] hidden sm:table-cell">
                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('isk_lost')">
                            ISK Lost <Icon :name="sortIcon('isk_lost')" class="text-fine" />
                        </button>
                    </th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="e in sorted" :key="e.id"
                    class="border-b border-white/[0.03] hover:bg-white/[0.02]">
                    <td class="py-1.5 px-3">
                        <NuxtLink :to="entityLink(e)" class="flex items-center gap-2 min-w-0">
                            <img :src="entityImage(e)" class="w-6 h-6 rounded flex-shrink-0" loading="lazy">
                            <div class="min-w-0">
                                <div class="text-gray-300 hover:text-blue-400 truncate">
                                    <span class="font-medium">{{ e.name }}</span>
                                    <span v-if="e.ticker" class="text-gray-500 ml-1">[{{ e.ticker }}]</span>
                                </div>
                                <div v-if="e.alliance_name" class="text-fine text-gray-500 truncate">{{ e.alliance_name }}</div>
                            </div>
                        </NuxtLink>
                    </td>
                    <td v-if="type === 'alliances'" class="py-1.5 px-2 text-right text-gray-400 tabular-nums hidden md:table-cell">
                        {{ formatNumber(e.corporation_count || 0) }}
                    </td>
                    <td class="py-1.5 px-2 text-right text-gray-300 tabular-nums">
                        {{ formatNumber(e.member_count) }}
                    </td>
                    <td class="py-1.5 px-2 text-right tabular-nums" :class="e.kills > 0 ? 'text-green-400/80' : 'text-gray-700'">
                        {{ e.kills > 0 ? formatNumber(e.kills) : '—' }}
                    </td>
                    <td class="py-1.5 px-2 text-right tabular-nums" :class="e.losses > 0 ? 'text-red-400/80' : 'text-gray-700'">
                        {{ e.losses > 0 ? formatNumber(e.losses) : '—' }}
                    </td>
                    <td class="py-1.5 px-2 text-right tabular-nums hidden sm:table-cell" :class="e.isk_destroyed > 0 ? 'text-isk/70' : 'text-gray-700'">
                        {{ e.isk_destroyed > 0 ? formatIsk(e.isk_destroyed) : '—' }}
                    </td>
                    <td class="py-1.5 px-3 text-right tabular-nums hidden sm:table-cell" :class="e.isk_lost > 0 ? 'text-red-400/60' : 'text-gray-700'">
                        {{ e.isk_lost > 0 ? formatIsk(e.isk_lost) : '—' }}
                    </td>
                </tr>
            </tbody>
        </table>
    </div>
</template>
