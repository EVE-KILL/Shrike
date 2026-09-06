<script setup lang="ts">
const props = defineProps<{
    allianceId?: number
    corporationId?: number
    characterId?: number
}>()

const page = ref(1)
const sortBy = ref<'recent' | 'kills' | 'isk'>('recent')

const fetchParams = computed(() => {
    const p: Record<string, any> = { page: page.value, limit: 50, sort: sortBy.value }
    if (props.allianceId) p.allianceId = props.allianceId
    if (props.corporationId) p.corporationId = props.corporationId
    if (props.characterId) p.characterId = props.characterId
    return p
})

const { data, pending } = await useApiFetch<any>('/api/conflicts/wars', {
    params: fetchParams,
    watch: [page, sortBy],
})

const wars = computed(() => data.value?.wars || [])

const timeAgo = (iso: string | null): string => {
    if (!iso) return ''
    const diff = Date.now() - new Date(iso).getTime()
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

const entityImage = (entity: any): string => {
    if (entity.type === 'alliance') return `/images/alliances/${entity.id}/logo?size=64`
    return `/images/corporations/${entity.id}/logo?size=64`
}

const setSort = (s: 'recent' | 'kills' | 'isk') => {
    sortBy.value = s
    page.value = 1
}
</script>

<template>
    <div>
        <div class="flex flex-wrap items-center gap-2 mb-4">
            <span class="text-fine text-gray-500 uppercase tracking-wider">Sort:</span>
            <button
                v-for="s in [
                    { key: 'recent', label: 'Recent' },
                    { key: 'kills', label: 'Most Kills' },
                    { key: 'isk', label: 'Most ISK' },
                ]" :key="s.key"
                class="px-3 py-1.5 text-xs rounded-md font-medium transition-colors"
                :class="sortBy === s.key ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' : 'text-gray-400 border border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="setSort(s.key as any)">
                {{ s.label }}
            </button>
        </div>

        <div v-if="pending && wars.length === 0" class="flex items-center justify-center py-20">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <div v-else class="glass-panel" :class="{ 'opacity-60': pending }">
            <div class="hidden md:grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_100px_100px] gap-3 px-4 py-2 text-fine font-bold uppercase tracking-wider text-gray-500 border-b border-white/[0.08]">
                <div>Aggressor</div>
                <div></div>
                <div>Defender</div>
                <div>Started</div>
                <div class="text-right">Status</div>
            </div>

            <NuxtLink v-for="w in wars" :key="w.war_id" :to="`/war/${w.war_id}`"
                class="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_100px_100px] gap-3 px-4 py-3 border-b border-white/[0.04] hover:bg-blue-500/[0.04] transition-colors items-center">
                <div class="flex items-center gap-2.5 min-w-0">
                    <EveImage :src="entityImage(w.aggressor)" loading="lazy" :size="32" :alt="w.aggressor.name" class="w-7 h-7 md:w-8 md:h-8 rounded flex-shrink-0" />
                    <div class="min-w-0">
                        <div class="text-xs text-gray-200 truncate">{{ w.aggressor.name }}</div>
                        <div class="text-fine text-gray-500">
                            <span class="text-isk/70 tabular-nums">{{ formatIsk(w.aggressor.isk_destroyed) }}</span>
                            <span class="hidden md:inline text-gray-600 mx-1">·</span>
                            <span class="hidden md:inline tabular-nums">{{ w.aggressor.ships_killed }} ships</span>
                        </div>
                    </div>
                </div>
                <div class="flex items-center justify-center text-xs text-gray-600 font-bold px-2">VS</div>
                <div class="flex items-center gap-2.5 min-w-0">
                    <EveImage :src="entityImage(w.defender)" loading="lazy" :size="32" :alt="w.defender.name" class="w-7 h-7 md:w-8 md:h-8 rounded flex-shrink-0" />
                    <div class="min-w-0">
                        <div class="text-xs text-gray-200 truncate">{{ w.defender.name }}</div>
                        <div class="text-fine text-gray-500">
                            <span class="text-isk/70 tabular-nums">{{ formatIsk(w.defender.isk_destroyed) }}</span>
                            <span class="hidden md:inline text-gray-600 mx-1">·</span>
                            <span class="hidden md:inline tabular-nums">{{ w.defender.ships_killed }} ships</span>
                        </div>
                    </div>
                </div>
                <div class="col-span-2 md:col-span-1 text-fine md:text-xs text-gray-400">{{ timeAgo(w.started) }}</div>
                <div class="flex flex-wrap gap-1 justify-end">
                    <span v-if="!w.finished" class="px-1.5 py-0.5 text-fine rounded bg-green-500/20 text-green-400 font-medium">Active</span>
                    <span v-else class="px-1.5 py-0.5 text-fine rounded bg-gray-500/20 text-gray-400 font-medium">Finished</span>
                    <span v-if="w.mutual" class="px-1.5 py-0.5 text-fine rounded bg-amber-500/20 text-amber-400 font-medium">Mutual</span>
                    <span v-if="w.open_for_allies" class="px-1.5 py-0.5 text-fine rounded bg-blue-500/20 text-blue-400 font-medium">Allies</span>
                </div>
            </NuxtLink>

            <div v-if="wars.length === 0" class="py-12 text-center text-sm text-gray-500">No wars found</div>
        </div>

        <div v-if="wars.length >= 50 || page > 1" class="flex justify-center mt-4 gap-2">
            <button @click="page = Math.max(1, page - 1)" :disabled="page <= 1"
                class="px-3 py-1.5 text-xs rounded-md transition-colors"
                :class="page <= 1 ? 'text-gray-700' : 'text-gray-400 hover:text-blue-400 bg-white/[0.04] border border-white/[0.08]'">
                Previous
            </button>
            <span class="px-3 py-1.5 text-xs text-gray-500">Page {{ page }}</span>
            <button @click="page++" :disabled="wars.length < 50"
                class="px-3 py-1.5 text-xs rounded-md transition-colors"
                :class="wars.length < 50 ? 'text-gray-700' : 'text-gray-400 hover:text-blue-400 bg-white/[0.04] border border-white/[0.08]'">
                Next
            </button>
        </div>
    </div>
</template>
