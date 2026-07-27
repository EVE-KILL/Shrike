<script setup lang="ts">
interface FwSystem {
    solar_system_id: number
    system_name: string
    security: number
    region_name: string
    constellation_name: string
    owner_faction_id: number
    occupier_faction_id: number
    contested: string
    victory_points: number
    victory_points_threshold: number
    kills_24h: number
    isk_24h: number
}

const FACTION_COLORS: Record<number, string> = {
    500001: 'text-blue-400',    // Caldari
    500002: 'text-red-400',     // Minmatar
    500003: 'text-amber-400',   // Amarr
    500004: 'text-green-400',   // Gallente
}
const FACTION_SHORT: Record<number, string> = {
    500001: 'Caldari', 500002: 'Minmatar', 500003: 'Amarr', 500004: 'Gallente',
}

const props = defineProps<{ systems: FwSystem[] }>()

type SortKey = 'name' | 'region' | 'status' | 'vp' | 'kills' | 'isk'
const sortKey = ref<SortKey>('kills')
const sortDir = ref<'asc' | 'desc'>('desc')

function toggleSort(key: SortKey) {
    if (sortKey.value === key) sortDir.value = sortDir.value === 'desc' ? 'asc' : 'desc'
    else { sortKey.value = key; sortDir.value = 'desc' }
}

function sortIcon(key: SortKey): string {
    if (sortKey.value !== key) return 'lucide:chevrons-up-down'
    return sortDir.value === 'desc' ? 'lucide:chevron-down' : 'lucide:chevron-up'
}

const STATUS_RANK: Record<string, number> = { vulnerable: 3, contested: 2, captured: 1, uncontested: 0 }

const sorted = computed(() => {
    const arr = [...props.systems]
    const dir = sortDir.value === 'asc' ? 1 : -1
    arr.sort((a, b) => {
        let c = 0
        switch (sortKey.value) {
            case 'name': c = a.system_name.localeCompare(b.system_name); break
            case 'region': c = (a.region_name || '').localeCompare(b.region_name || ''); break
            case 'status': c = (STATUS_RANK[a.contested] ?? 0) - (STATUS_RANK[b.contested] ?? 0); break
            case 'vp': c = (a.victory_points_threshold > 0 ? a.victory_points / a.victory_points_threshold : 0) - (b.victory_points_threshold > 0 ? b.victory_points / b.victory_points_threshold : 0); break
            case 'kills': c = a.kills_24h - b.kills_24h; break
            case 'isk': c = a.isk_24h - b.isk_24h; break
        }
        return dir * c
    })
    return arr
})

const statusClass = (s: string) => {
    if (s === 'vulnerable') return 'bg-red-500/20 text-red-400'
    if (s === 'contested') return 'bg-yellow-500/20 text-yellow-400'
    if (s === 'captured') return 'bg-purple-500/20 text-purple-400'
    return 'bg-green-500/10 text-green-400/70'
}

const vpPct = (sys: FwSystem) => sys.victory_points_threshold > 0
    ? Math.min(100, (sys.victory_points / sys.victory_points_threshold) * 100)
    : 0

const secColor = (sec: number) => sec >= 0.5 ? 'text-green-400' : sec > 0 ? 'text-amber-400' : 'text-red-400'
</script>

<template>
    <div class="overflow-x-auto">
        <table class="w-full text-xs">
            <thead class="text-gray-500 text-left">
                <tr class="border-b border-white/[0.06]">
                    <th class="py-2 pr-2 font-medium">
                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('name')">
                            System <Icon :name="sortIcon('name')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 px-2 font-medium hidden md:table-cell">
                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('region')">
                            Region <Icon :name="sortIcon('region')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 px-2 font-medium w-[80px]">Occupier</th>
                    <th class="py-2 px-2 font-medium w-[90px]">
                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('status')">
                            Status <Icon :name="sortIcon('status')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 px-2 font-medium w-[140px]">
                        <button class="flex items-center gap-1 hover:text-blue-400" @click="toggleSort('vp')">
                            VP Progress <Icon :name="sortIcon('vp')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 px-2 font-medium text-right w-[60px]">
                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('kills')">
                            Kills <Icon :name="sortIcon('kills')" class="text-fine" />
                        </button>
                    </th>
                    <th class="py-2 pl-2 font-medium text-right w-[80px] hidden sm:table-cell">
                        <button class="ml-auto flex items-center gap-1 hover:text-blue-400" @click="toggleSort('isk')">
                            ISK (24h) <Icon :name="sortIcon('isk')" class="text-fine" />
                        </button>
                    </th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="sys in sorted" :key="sys.solar_system_id"
                    class="border-b border-white/[0.03] hover:bg-white/[0.02]">
                    <td class="py-1.5 pr-2">
                        <NuxtLink :to="`/system/${sys.solar_system_id}`" class="text-gray-300 hover:text-blue-400 font-medium">
                            {{ sys.system_name }}
                        </NuxtLink>
                        <span class="ml-1 tabular-nums" :class="secColor(sys.security)">{{ sys.security.toFixed(1) }}</span>
                    </td>
                    <td class="py-1.5 px-2 text-gray-500 hidden md:table-cell">
                        {{ sys.region_name }}
                    </td>
                    <td class="py-1.5 px-2">
                        <span :class="FACTION_COLORS[sys.occupier_faction_id] || 'text-gray-400'" class="font-medium">
                            {{ FACTION_SHORT[sys.occupier_faction_id] || '?' }}
                        </span>
                    </td>
                    <td class="py-1.5 px-2">
                        <span class="px-1.5 py-0.5 rounded text-fine font-medium" :class="statusClass(sys.contested)">
                            {{ sys.contested }}
                        </span>
                    </td>
                    <td class="py-1.5 px-2">
                        <div class="flex items-center gap-2">
                            <div class="flex-1 h-1.5 bg-white/[0.06] rounded-full overflow-hidden">
                                <div class="h-full rounded-full transition-all"
                                    :class="sys.contested === 'vulnerable' ? 'bg-red-500' : sys.contested === 'contested' ? 'bg-yellow-500' : 'bg-white/20'"
                                    :style="{ width: `${vpPct(sys)}%` }"></div>
                            </div>
                            <span class="text-fine text-gray-600 tabular-nums w-[32px] text-right">{{ Math.round(vpPct(sys)) }}%</span>
                        </div>
                    </td>
                    <td class="py-1.5 px-2 text-right tabular-nums" :class="sys.kills_24h > 0 ? 'text-gray-300' : 'text-gray-700'">
                        {{ sys.kills_24h || '—' }}
                    </td>
                    <td class="py-1.5 pl-2 text-right tabular-nums hidden sm:table-cell" :class="sys.isk_24h > 0 ? 'text-isk/70' : 'text-gray-700'">
                        {{ sys.isk_24h > 0 ? formatIsk(sys.isk_24h) : '—' }}
                    </td>
                </tr>
            </tbody>
        </table>
    </div>
</template>
