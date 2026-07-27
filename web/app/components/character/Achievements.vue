<script setup lang="ts">
import { getAchievementDef } from '~/utils/achievementRegistry'

const props = defineProps<{
    characterId: number
}>()

const { data, pending } = await useApiFetch<any>(`/api/character/${props.characterId}/achievements`, {
    getCachedData: cachedPayload,
})

interface Achievement {
    achievement_id: string
    current_count: number
    threshold: number
    completion_tiers: number
    is_completed: boolean
    points: number
    completed_at: string | null
    // Enriched from registry
    name: string
    description: string
    type: string
    rarity: string
    category: string
    pointsModifier: string
}

const achievements = computed<Achievement[]>(() => {
    const raw = data.value?.achievements || []
    return raw.map((a: any) => {
        const def = getAchievementDef(a.achievement_id)
        return { ...a, ...def }
    })
})

// Summary stats
const totalPoints = computed(() => achievements.value.reduce((s, a) => s + a.points, 0))
const completedCount = computed(() => achievements.value.filter(a => a.is_completed).length)
const totalCount = computed(() => achievements.value.length)

// Filters
const activeFilter = ref<'all' | 'completed' | 'in_progress'>('all')
const activeType = ref<string | null>(null)

const categories = computed(() => [...new Set(achievements.value.map(a => a.category))].sort())
const activeCategory = ref<string | null>(null)

const filtered = computed(() => {
    let list = achievements.value
    if (activeFilter.value === 'completed') list = list.filter(a => a.is_completed)
    else if (activeFilter.value === 'in_progress') list = list.filter(a => !a.is_completed && a.current_count > 0)
    if (activeType.value) list = list.filter(a => a.type === activeType.value)
    if (activeCategory.value) list = list.filter(a => a.category === activeCategory.value)

    // Sort: completed first, then by rarity (legendary→common), then by points
    const rarityOrder: Record<string, number> = { legendary: 0, epic: 1, rare: 2, uncommon: 3, common: 4 }
    return [...list].sort((a, b) => {
        if (a.is_completed !== b.is_completed) return a.is_completed ? -1 : 1
        const ra = rarityOrder[a.rarity] ?? 5
        const rb = rarityOrder[b.rarity] ?? 5
        if (ra !== rb) return ra - rb
        return b.points - a.points
    })
})

const progressPercent = (a: Achievement): number => {
    if (a.is_completed) return 100
    if (a.threshold <= 0) return 0
    return Math.min(100, Math.round(a.current_count / a.threshold * 100))
}

const rarityStyles: Record<string, { border: string, bg: string, text: string, badge: string }> = {
    legendary: { border: 'border-amber-500/30', bg: 'bg-amber-500/[0.06]', text: 'text-amber-400', badge: 'bg-amber-500/20 text-amber-400' },
    epic: { border: 'border-purple-500/30', bg: 'bg-purple-500/[0.06]', text: 'text-purple-400', badge: 'bg-purple-500/20 text-purple-400' },
    rare: { border: 'border-blue-500/30', bg: 'bg-blue-500/[0.06]', text: 'text-blue-400', badge: 'bg-blue-500/20 text-blue-400' },
    uncommon: { border: 'border-green-500/30', bg: 'bg-green-500/[0.06]', text: 'text-green-400', badge: 'bg-green-500/20 text-green-400' },
    common: { border: 'border-white/[0.08]', bg: 'bg-white/[0.02]', text: 'text-gray-400', badge: 'bg-white/[0.06] text-gray-400' },
}

const typeColors: Record<string, string> = {
    pvp: 'bg-red-500/20 text-red-400',
    pve: 'bg-blue-500/20 text-blue-400',
    exploration: 'bg-green-500/20 text-green-400',
    industry: 'bg-orange-500/20 text-orange-400',
    special: 'bg-purple-500/20 text-purple-400',
}

const rarityLabel = (r: string) => r.charAt(0).toUpperCase() + r.slice(1)

const progressBarColor = (a: Achievement): string => {
    if (a.pointsModifier === 'negative') return 'bg-red-500'
    const colors: Record<string, string> = { legendary: 'bg-amber-500', epic: 'bg-purple-500', rare: 'bg-blue-500', uncommon: 'bg-green-500', common: 'bg-gray-500' }
    return colors[a.rarity] || 'bg-gray-500'
}
</script>

<template>
    <div>
        <!-- Loading -->
        <div v-if="pending" class="flex items-center justify-center py-12">
            <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
        </div>

        <div v-else>
            <!-- Summary -->
            <div class="grid grid-cols-3 gap-4 mb-6">
                <div class="bg-white/[0.04] border border-amber-500/20 rounded-lg p-4 text-center">
                    <Icon name="lucide:trophy" class="w-5 h-5 text-amber-400 mx-auto mb-1" />
                    <div class="text-xl font-bold text-amber-400 tabular-nums">{{ totalPoints.toLocaleString('en-US') }}</div>
                    <div class="text-fine uppercase tracking-wider text-gray-500">Total Points</div>
                </div>
                <div class="bg-white/[0.04] border border-green-500/20 rounded-lg p-4 text-center">
                    <Icon name="lucide:check-circle" class="w-5 h-5 text-green-400 mx-auto mb-1" />
                    <div class="text-xl font-bold text-green-400 tabular-nums">{{ completedCount }} <span class="text-sm text-gray-600">/ {{ totalCount }}</span></div>
                    <div class="text-fine uppercase tracking-wider text-gray-500">Completed</div>
                </div>
                <div class="bg-white/[0.04] border border-blue-500/20 rounded-lg p-4">
                    <div class="text-fine uppercase tracking-wider text-gray-500 mb-1.5 text-center">Completion</div>
                    <div class="h-2 bg-white/[0.06] rounded-full overflow-hidden mb-1">
                        <div class="h-full bg-blue-500 rounded-full transition-all"
                            :style="{ width: totalCount > 0 ? `${Math.round(completedCount / totalCount * 100)}%` : '0%' }"></div>
                    </div>
                    <div class="text-center text-xs font-bold text-blue-400 tabular-nums">{{ totalCount > 0 ? Math.round(completedCount / totalCount * 100) : 0 }}%</div>
                </div>
            </div>

            <!-- Filters -->
            <div class="flex flex-wrap gap-2 mb-4">
                <!-- Status -->
                <div class="flex gap-1">
                    <button v-for="f in [{ v: 'all', l: 'All' }, { v: 'completed', l: 'Completed' }, { v: 'in_progress', l: 'In Progress' }]"
                        :key="f.v"
                        class="px-2.5 py-1 text-xs rounded font-medium transition-colors"
                        :class="activeFilter === f.v ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'text-gray-500 hover:text-blue-400 border border-transparent'"
                        @click="activeFilter = f.v as any">
                        {{ f.l }}
                    </button>
                </div>

                <div class="w-px h-6 bg-white/[0.06] self-center"></div>

                <!-- Category -->
                <div class="flex gap-1 flex-wrap">
                    <button class="px-2.5 py-1 text-xs rounded font-medium transition-colors"
                        :class="!activeCategory ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'text-gray-500 hover:text-blue-400 border border-transparent'"
                        @click="activeCategory = null">
                        All
                    </button>
                    <button v-for="cat in categories" :key="cat"
                        class="px-2.5 py-1 text-xs rounded font-medium transition-colors"
                        :class="activeCategory === cat ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'text-gray-500 hover:text-blue-400 border border-transparent'"
                        @click="activeCategory = activeCategory === cat ? null : cat">
                        {{ cat }}
                    </button>
                </div>
            </div>

            <!-- Achievement grid -->
            <div v-if="filtered.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                <div v-for="a in filtered" :key="a.achievement_id"
                    class="rounded-lg border p-3.5 transition-colors"
                    :class="[
                        rarityStyles[a.rarity]?.border || rarityStyles.common!.border,
                        rarityStyles[a.rarity]?.bg || rarityStyles.common!.bg,
                        !a.is_completed && a.current_count === 0 ? 'opacity-40' : '',
                    ]">
                    <!-- Header: name + points -->
                    <div class="flex items-start justify-between gap-2 mb-1.5">
                        <div class="flex-1 min-w-0">
                            <div class="text-xs font-medium text-white truncate">{{ a.name }}</div>
                            <div v-if="a.description" class="text-fine text-gray-500 mt-0.5 line-clamp-2">{{ a.description }}</div>
                        </div>
                        <div class="flex-shrink-0 text-right">
                            <div class="text-xs font-bold tabular-nums" :class="a.pointsModifier === 'negative' ? 'text-red-400' : rarityStyles[a.rarity]?.text || 'text-gray-400'">
                                {{ a.points > 0 ? (a.pointsModifier === 'negative' ? '' : '+') : '' }}{{ a.points.toLocaleString('en-US') }}
                            </div>
                            <div v-if="a.completion_tiers > 1" class="text-fine text-gray-600">x{{ a.completion_tiers }}</div>
                        </div>
                    </div>

                    <!-- Badges -->
                    <div class="flex items-center gap-1.5 mb-2">
                        <span class="text-fine uppercase tracking-wider px-1.5 py-0.5 rounded font-medium"
                            :class="rarityStyles[a.rarity]?.badge || rarityStyles.common!.badge">
                            {{ rarityLabel(a.rarity) }}
                        </span>
                        <span class="text-fine uppercase tracking-wider px-1.5 py-0.5 rounded font-medium"
                            :class="typeColors[a.type] || typeColors.pvp">
                            {{ a.type }}
                        </span>
                        <span v-if="a.is_completed" class="ml-auto">
                            <Icon name="lucide:check-circle" class="w-3.5 h-3.5 text-green-400" />
                        </span>
                        <span v-else-if="a.current_count > 0" class="ml-auto">
                            <Icon name="lucide:clock" class="w-3.5 h-3.5 text-blue-400" />
                        </span>
                    </div>

                    <!-- Progress -->
                    <div class="h-1.5 bg-black/30 rounded-full overflow-hidden">
                        <div class="h-full rounded-full transition-all" :class="progressBarColor(a)"
                            :style="{ width: `${progressPercent(a)}%` }">
                        </div>
                    </div>
                    <div class="flex justify-between mt-1 text-fine text-gray-600">
                        <span class="tabular-nums">{{ a.current_count.toLocaleString('en-US') }} / {{ a.threshold.toLocaleString('en-US') }}</span>
                        <span v-if="a.is_completed && a.completed_at">
                            {{ new Date(a.completed_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) }}
                        </span>
                        <span v-else class="tabular-nums">{{ progressPercent(a) }}%</span>
                    </div>
                </div>
            </div>

            <div v-else class="text-sm text-gray-600 text-center py-8">
                No achievements match the current filters
            </div>
        </div>
    </div>
</template>
