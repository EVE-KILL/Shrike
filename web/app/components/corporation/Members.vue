<script setup lang="ts">
const props = defineProps<{
    corporationId: number
}>()

const page = ref(1)
const limit = 100
const activityFilter = ref(0) // 0 = all, 7/30/90
const sortBy = ref('name')

const sortOptions = [
    { value: 'name', label: 'Name' },
    { value: 'last_active', label: 'Last Active' },
    { value: 'security_status', label: 'Sec Status' },
]

const memberParams = computed(() => {
    const p: Record<string, any> = { page: page.value, limit, sort: sortBy.value }
    if (activityFilter.value) p.activity = activityFilter.value
    return p
})

const { data, pending, refresh } = await useApiFetch<any>(`/api/corporation/${props.corporationId}/members`, {
    params: memberParams,
    watch: [page, activityFilter, sortBy],
    getCachedData: cachedPayload,
})

const members = computed(() => data.value?.members || [])
const total = computed(() => data.value?.total || 0)
const totalPages = computed(() => Math.ceil(total.value / limit))

const goToPage = (p: number) => {
    if (p >= 1 && p <= totalPages.value) page.value = p
}

const setActivityFilter = (days: number) => {
    activityFilter.value = activityFilter.value === days ? 0 : days
    page.value = 1
}

const visiblePages = computed(() => {
    const pages: (number | string)[] = []
    const p = page.value
    const max = totalPages.value
    if (max <= 1) return []

    pages.push(1)
    if (p > 3) pages.push('...')
    const start = Math.max(2, p - 1)
    const end = Math.min(max - 1, p + 1)
    for (let i = start; i <= end; i++) pages.push(i)
    if (p < max - 2) pages.push('...')
    if (max > 1) pages.push(max)

    return pages
})

const secColor = (sec: number): string => {
    if (sec >= 5) return 'text-cyan-300'
    if (sec >= 3) return 'text-green-400'
    if (sec >= 1) return 'text-green-300'
    if (sec >= 0) return 'text-yellow-400'
    if (sec >= -2) return 'text-orange-400'
    if (sec >= -5) return 'text-red-400'
    return 'text-red-600'
}

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

const hasAnyRoles = computed(() => members.value.some((m: any) => m.is_fc || m.is_logi || m.is_capital_pilot))
</script>

<template>
    <div>
        <div v-if="pending && members.length === 0" class="flex items-center justify-center py-12">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <div v-else>
            <!-- Controls bar -->
            <div class="flex flex-wrap items-center gap-3 mb-4">
                <!-- Activity filter -->
                <div class="flex items-center gap-1.5">
                    <span class="text-fine text-gray-600">Active:</span>
                    <button v-for="days in [7, 30, 90]" :key="days"
                        class="px-2 py-1 text-fine rounded font-medium transition-colors"
                        :class="activityFilter === days
                            ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                            : 'text-gray-500 hover:text-blue-400 border border-transparent'"
                        @click="setActivityFilter(days)">
                        {{ days }}d
                    </button>
                    <button v-if="activityFilter > 0"
                        class="px-2 py-1 text-fine rounded text-gray-500 hover:text-gray-300 transition-colors"
                        @click="activityFilter = 0; page = 1">
                        <Icon name="lucide:x" class="w-3 h-3" />
                    </button>
                </div>

                <!-- Sort -->
                <div class="flex items-center gap-1.5 ml-auto">
                    <span class="text-fine text-gray-600">Sort:</span>
                    <button v-for="opt in sortOptions" :key="opt.value"
                        class="px-2 py-1 text-fine rounded font-medium transition-colors"
                        :class="sortBy === opt.value
                            ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                            : 'text-gray-500 hover:text-blue-400 border border-transparent'"
                        @click="sortBy = opt.value; page = 1">
                        {{ opt.label }}
                    </button>
                </div>
            </div>

            <!-- Header + top pagination -->
            <div class="flex items-center justify-between mb-3">
                <div class="text-xs text-gray-500">
                    {{ total.toLocaleString('en-US') }} members
                    <span v-if="activityFilter > 0" class="text-blue-400/70">active in {{ activityFilter }}d</span>
                    <span v-if="totalPages > 1" class="text-gray-600">&mdash; page {{ page }} of {{ totalPages }}</span>
                </div>

                <!-- Pagination -->
                <div v-if="totalPages > 1" class="flex items-center gap-1">
                    <button @click="goToPage(page - 1)" :disabled="page <= 1"
                        class="px-2 py-1 text-xs rounded transition-colors"
                        :class="page <= 1 ? 'text-gray-700 cursor-not-allowed' : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'">
                        <Icon name="lucide:chevron-left" class="w-3.5 h-3.5" />
                    </button>
                    <template v-for="p in visiblePages" :key="p">
                        <span v-if="p === '...'" class="px-1 text-xs text-gray-600">...</span>
                        <button v-else @click="goToPage(p as number)"
                            class="min-w-[28px] px-1.5 py-1 text-xs rounded transition-colors tabular-nums"
                            :class="p === page ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'">
                            {{ p }}
                        </button>
                    </template>
                    <button @click="goToPage(page + 1)" :disabled="page >= totalPages"
                        class="px-2 py-1 text-xs rounded transition-colors"
                        :class="page >= totalPages ? 'text-gray-700 cursor-not-allowed' : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'">
                        <Icon name="lucide:chevron-right" class="w-3.5 h-3.5" />
                    </button>
                </div>
            </div>

            <!-- Members grid -->
            <div v-if="members.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-1.5"
                :class="{ 'opacity-50': pending }">
                <NuxtLink v-for="m in members" :key="m.character_id"
                    :to="`/character/${m.character_id}`"
                    class="flex items-center gap-2.5 px-2.5 py-2 rounded-md border border-white/[0.04] hover:border-white/[0.1] hover:bg-blue-500/[0.04] transition-colors">
                    <img :src="`/images/characters/${m.character_id}/portrait?size=64`"
                        class="w-9 h-9 rounded" loading="lazy">
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-1.5">
                            <span class="text-fine text-gray-300 truncate">{{ m.name }}</span>
                            <!-- Role badges -->
                            <span v-if="m.is_fc" class="px-1 py-0.5 text-[10px] leading-none font-bold rounded bg-yellow-500/20 text-yellow-400 uppercase tracking-wider" v-tooltip="'Fleet Commander'">FC</span>
                            <span v-if="m.is_logi" class="px-1 py-0.5 text-[10px] leading-none font-bold rounded bg-green-500/20 text-green-400 uppercase tracking-wider" v-tooltip="'Logistics'">Logi</span>
                            <span v-if="m.is_capital_pilot" class="px-1 py-0.5 text-[10px] leading-none font-bold rounded bg-purple-500/20 text-purple-400 uppercase tracking-wider" v-tooltip="'Capital Pilot'">Cap</span>
                        </div>
                        <div class="flex items-center gap-2 text-fine text-gray-600">
                            <span :class="secColor(m.security_status)" class="font-mono">{{ m.security_status.toFixed(1) }}</span>
                            <span v-if="m.last_active">{{ timeAgo(m.last_active) }}</span>
                            <span v-if="m.kills_90d > 0 || m.losses_90d > 0" class="ml-auto tabular-nums">
                                <span class="text-green-400/70">{{ m.kills_90d }}</span>
                                <span class="text-gray-700">/</span>
                                <span class="text-red-400/70">{{ m.losses_90d }}</span>
                            </span>
                        </div>
                    </div>
                </NuxtLink>
            </div>

            <div v-else class="text-sm text-gray-600 text-center py-8">
                <template v-if="activityFilter > 0">No members active in the last {{ activityFilter }} days</template>
                <template v-else>No known members</template>
            </div>

            <!-- Bottom pagination -->
            <div v-if="totalPages > 1" class="flex justify-center mt-4">
                <div class="flex items-center gap-1">
                    <button @click="goToPage(page - 1)" :disabled="page <= 1"
                        class="px-2 py-1 text-xs rounded transition-colors"
                        :class="page <= 1 ? 'text-gray-700 cursor-not-allowed' : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'">
                        <Icon name="lucide:chevron-left" class="w-3.5 h-3.5" />
                    </button>
                    <template v-for="p in visiblePages" :key="p">
                        <span v-if="p === '...'" class="px-1 text-xs text-gray-600">...</span>
                        <button v-else @click="goToPage(p as number)"
                            class="min-w-[28px] px-1.5 py-1 text-xs rounded transition-colors tabular-nums"
                            :class="p === page ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'">
                            {{ p }}
                        </button>
                    </template>
                    <button @click="goToPage(page + 1)" :disabled="page >= totalPages"
                        class="px-2 py-1 text-xs rounded transition-colors"
                        :class="page >= totalPages ? 'text-gray-700 cursor-not-allowed' : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'">
                        <Icon name="lucide:chevron-right" class="w-3.5 h-3.5" />
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>
