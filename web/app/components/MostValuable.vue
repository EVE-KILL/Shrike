<script setup lang="ts">
interface MostValuableKill {
    killmail_id: number
    ship_type_id: number
    ship_name: string
    total_value: number
    victim_character_id: number | null
    victim_character_name: string | null
    victim_corporation_name: string | null
    victim_alliance_name: string | null
}

// withDefaults gives Vue the runtime prop type info it needs to coerce a
// bare `compact` attribute to true — without it the TS-only defineProps
// loses the Boolean type at runtime and the client hydrates with undefined,
// causing a class mismatch against the server-rendered compact layout.
const props = withDefaults(defineProps<{
    apiEndpoint?: string
    extraParams?: Record<string, any>
    // Compact = 4 cols × 2 rows (8 items, all visible). Designed to fit two
    // widgets side by side. Default = the full-width 8-cols-on-desktop layout.
    compact?: boolean
}>(), {
    compact: false,
})

const activeTab = ref<'kills' | 'ships' | 'structures'>('kills')

const tabs = [
    { key: 'kills' as const, label: 'Most Valuable Kills', shortLabel: 'Kills' },
    { key: 'ships' as const, label: 'Most Valuable Ships', shortLabel: 'Ships' },
    { key: 'structures' as const, label: 'Most Valuable Structures', shortLabel: 'Structures' },
]

const endpoint = computed(() => props.apiEndpoint || '/api/stats')
const mvParams = computed(() => ({
    dataType: `most_valuable_${activeTab.value}`,
    limit: 8,
    days: 7,
    ...(props.extraParams || {}),
}))
const { data, pending, refresh } = await useApiFetch<{ entries: MostValuableKill[] }>(endpoint, {
    params: mvParams,
    default: () => ({ entries: [] }),
    watch: [activeTab],
})

const items = computed(() => data.value?.entries || [])

// Match the grid's slot widths rather than treating every card as a 256px image.
const imageSizes = computed(() => props.compact
    ? '(min-width: 1360px) 144px, (min-width: 1024px) calc((100vw - 188px) / 8), (min-width: 768px) calc((100vw - 166px) / 4), (min-width: 640px) calc((100vw - 66px) / 4), calc((100vw - 50px) / 2)'
    : '(min-width: 1360px) 144px, (min-width: 1024px) calc((100vw - 198px) / 8), (min-width: 768px) calc((100vw - 166px) / 4), calc((100vw - 58px) / 3)')

/**
 * The window this widget covers, for the empty state only. "No data available"
 * and "nothing in the last 7 days" are very different claims, and an entity
 * quiet for a week was making the first while meaning the second.
 */
const windowLabel = computed(() => `${mvParams.value.days}d`)

</script>

<template>
    <div class="glass-panel p-3" :class="compact ? '' : 'mb-6'">
        <div class="flex gap-1 mb-3">
            <button
                v-for="tab in tabs"
                :key="tab.key"
                class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors"
                :class="activeTab === tab.key
                    ? 'bg-blue-500/10 text-blue-400'
                    : 'text-gray-400 hover:text-blue-300 hover:bg-blue-500/[0.04]'"
                @click="activeTab = tab.key"
            >
                <span class="sm:hidden">{{ tab.shortLabel }}</span>
                <span class="hidden sm:inline">{{ tab.label }}</span>
            </button>
        </div>

        <div v-if="pending" :class="compact ? 'grid grid-cols-2 sm:grid-cols-4 gap-2' : 'grid grid-cols-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-8 gap-2'">
            <div v-for="i in 8" :key="i" class="aspect-square rounded-lg bg-white/[0.02] animate-pulse" :class="!compact && i > 6 ? 'hidden sm:block' : ''"></div>
        </div>

        <div v-else-if="items.length === 0" class="py-8 text-center text-xs text-gray-600">
            Nothing in the last {{ windowLabel }}
        </div>

        <div v-else :class="compact ? 'grid grid-cols-2 sm:grid-cols-4 gap-2' : 'grid grid-cols-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-8 gap-2'">
            <NuxtLink
                v-for="(kill, idx) in items"
                :key="kill.killmail_id"
                :to="`/kill/${kill.killmail_id}`"
                class="group relative aspect-square rounded-lg overflow-hidden bg-black/40 border border-white/[0.04] hover:border-blue-500/30 transition-all"
                :class="!compact && idx >= 6 ? 'hidden sm:block' : ''"
            >
                <EveImage
                    :src="`/images/types/${kill.ship_type_id}/overlayrender?size=256`"
                    :alt="kill.ship_name"
                    :sizes="imageSizes"
                    class="absolute inset-0 w-full h-full object-contain p-2 transition-transform duration-300 group-hover:scale-110"
                    :loading="idx < 6 ? 'eager' : 'lazy'"
                    :fetchpriority="idx === 0 ? 'high' : undefined"
                />
                <div class="absolute inset-x-0 bottom-0 h-2/3 bg-gradient-to-t from-black/90 via-black/50 to-transparent pointer-events-none"></div>
                <div class="absolute top-1.5 right-1.5 px-1.5 py-0.5 rounded bg-black/60 backdrop-blur-sm text-fine font-semibold text-green-400">
                    {{ formatIsk(kill.total_value) }}
                </div>
                <div class="absolute inset-x-0 bottom-0 p-2.5 flex flex-col items-start">
                    <div class="text-xs font-semibold text-white leading-tight truncate w-full drop-shadow-lg">{{ kill.ship_name }}</div>
                    <div v-if="kill.victim_character_name" class="text-fine text-gray-300/90 truncate w-full mt-0.5 drop-shadow">{{ kill.victim_character_name }}</div>
                    <div v-if="kill.victim_corporation_name" class="text-fine text-gray-400/80 truncate w-full drop-shadow">{{ kill.victim_corporation_name }}</div>
                </div>
            </NuxtLink>
        </div>
    </div>
</template>
