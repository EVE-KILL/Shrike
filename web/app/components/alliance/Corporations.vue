<script setup lang="ts">
const props = defineProps<{
    allianceId: number
}>()

const sortBy = ref<'members' | 'name'>('members')
const sortDir = ref<'desc' | 'asc'>('desc')

const corpParams = computed(() => ({ sort: sortBy.value, dir: sortDir.value }))
const { data, pending } = await useApiFetch<any>(`/api/alliance/${props.allianceId}/corporations`, {
    params: corpParams,
    watch: [sortBy, sortDir],
    getCachedData: cachedPayload,
})

const corps = computed(() => data.value?.corporations || [])
const total = computed(() => data.value?.total || 0)

// Left accent strip per row, derived from each corp's ESI palette
const accentStyles = computed(() => {
    const map: Record<number, { boxShadow: string }> = {}
    for (const c of corps.value) {
        const a = entityAccent(c.palette)
        if (a) map[c.corporation_id] = { boxShadow: `inset 3px 0 0 0 ${a.accent}` }
    }
    return map
})

const toggleSort = (col: 'members' | 'name') => {
    if (sortBy.value === col) {
        sortDir.value = sortDir.value === 'desc' ? 'asc' : 'desc'
    } else {
        sortBy.value = col
        sortDir.value = col === 'members' ? 'desc' : 'asc'
    }
}

const sortIcon = (col: string) => {
    if (sortBy.value !== col) return 'lucide:arrow-up-down'
    return sortDir.value === 'desc' ? 'lucide:arrow-down' : 'lucide:arrow-up'
}
</script>

<template>
    <div>
        <div v-if="pending && corps.length === 0" class="flex items-center justify-center py-12">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <div v-else>
            <div class="text-xs text-gray-500 mb-3">{{ total }} corporations</div>

            <div v-if="corps.length > 0">
                <!-- Header -->
                <div class="flex items-center gap-2 px-3 py-2 text-fine uppercase tracking-wider text-gray-500 border-b border-white/[0.08]">
                    <div class="w-8"></div>
                    <button class="flex-1 flex items-center gap-1 hover:text-blue-400 transition-colors" @click="toggleSort('name')">
                        Corporation <Icon :name="sortIcon('name')" class="w-3 h-3" />
                    </button>
                    <button class="w-24 text-right flex items-center justify-end gap-1 hover:text-blue-400 transition-colors" @click="toggleSort('members')">
                        Members <Icon :name="sortIcon('members')" class="w-3 h-3" />
                    </button>
                </div>

                <!-- Rows -->
                <div :class="{ 'opacity-50': pending }">
                    <NuxtLink v-for="c in corps" :key="c.corporation_id"
                        :to="`/corporation/${c.corporation_id}`"
                        class="flex items-center gap-2 px-3 py-2 border-b border-white/[0.03] hover:bg-blue-500/[0.04] transition-colors"
                        :style="accentStyles[c.corporation_id]">
                        <img :src="`/images/corporations/${c.corporation_id}/logo?size=64`"
                            class="w-8 h-8 rounded" loading="lazy">
                        <div class="flex-1 min-w-0">
                            <span class="text-fine text-gray-300 truncate">{{ c.name }}</span>
                            <span class="text-fine text-gray-600 ml-1.5">[{{ c.ticker }}]</span>
                        </div>
                        <div class="w-24 text-right text-fine text-gray-400 tabular-nums">{{ c.member_count.toLocaleString('en-US') }}</div>
                    </NuxtLink>
                </div>
            </div>

            <div v-else class="text-sm text-gray-600 text-center py-8">No corporations found</div>
        </div>
    </div>
</template>
