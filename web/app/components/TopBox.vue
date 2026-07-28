<script setup lang="ts">
const props = defineProps<{
    title: string
    dataType: string
    limit?: number
    days?: number
    apiEndpoint?: string
    killType?: string
    /** Pre-loaded entries — when provided, skips API fetch */
    entries?: { id: number; name: string; count: number; palette?: EntityPalette | null }[]
    /** Override color for the count text (e.g. 'text-isk/70', 'text-red-400/70') */
    countColor?: string
    /** Set to true for above-the-fold boxes to eager-load images */
    eager?: boolean
    /** Custom formatter for the count column. Defaults to `count.toLocaleString('en-US')`. */
    formatCount?: (entry: { id: number; name: string; count: number }) => string
    /** Hide the "Last N days" footer (use for lifetime/non-period lists). */
    hideFooter?: boolean
}>()

interface TopEntry {
    id: number
    name: string
    count: number
    type: string
    region_id?: number
    palette?: EntityPalette | null
}

const entityType = computed(() => {
    switch (props.dataType) {
        case 'characters': return 'character'
        case 'corporations': return 'corporation'
        case 'alliances': return 'alliance'
        case 'ships': return 'item'
        case 'systems': return 'system'
        case 'constellations': return 'constellation'
        case 'regions': return 'region'
        default: return null
    }
})

const getImageUrl = (entry: TopEntry): string | null => {
    if (!Number.isInteger(entry.id) || entry.id <= 0) return null
    switch (props.dataType) {
        case 'characters': return `/images/characters/${entry.id}/portrait?size=64`
        case 'corporations': return `/images/corporations/${entry.id}/logo?size=64`
        case 'alliances': return `/images/alliances/${entry.id}/logo?size=64`
        case 'ships': return `/images/types/${entry.id}/icon?size=64`
        case 'systems': return `/images/systems/${entry.id}?size=64`
        case 'regions': return `/images/regions/${entry.id}?size=64`
        default: return null
    }
}

const getEntityUrl = (entry: TopEntry): string => {
    return `/${entityType.value}/${entry.id}`
}

// API mode: fetch data when no entries prop is provided
const useApi = computed(() => !props.entries)

const endpoint = computed(() => props.apiEndpoint || '/api/stats')
const params = computed(() => {
    const p: Record<string, any> = {
        dataType: props.dataType,
        limit: props.limit || 10,
        days: props.days || 7,
    }
    if (props.killType) p.type = props.killType
    return p
})

const { data, pending } = await useApiFetch<{ entries: TopEntry[] }>(endpoint, {
    params,
    default: () => ({ entries: [] }),
    lazy: true,
    immediate: useApi.value,
})

const items = computed(() => {
    if (props.entries) return props.entries as TopEntry[]
    return data.value?.entries || []
})

const isLoading = computed(() => useApi.value && pending.value)

// Left accent strip for entries that carry a corp palette (corporation lists)
const accentStyles = computed(() => {
    const map: Record<number, { boxShadow: string }> = {}
    for (const e of items.value) {
        const a = entityAccent((e as TopEntry).palette)
        if (a) map[e.id] = { boxShadow: `inset 3px 0 0 0 ${a.accent}` }
    }
    return map
})
</script>

<template>
    <div v-if="isLoading || items.length > 0" class="glass-panel mb-4 p-2">
        <div class="px-1 pb-2 mb-1 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
            {{ title }}
        </div>

        <div v-if="isLoading" class="space-y-px">
            <div v-for="i in (limit || 10)" :key="i" class="flex items-center gap-2 px-2 py-1.5">
                <div class="w-6 h-6 rounded bg-white/[0.04] animate-pulse"></div>
                <div class="flex-1 h-3 rounded bg-white/[0.04] animate-pulse"></div>
                <div class="w-8 h-3 rounded bg-white/[0.04] animate-pulse"></div>
            </div>
        </div>

        <div v-else class="space-y-px">
            <NuxtLink
                v-for="(entry, idx) in items"
                :key="entry.id"
                :to="getEntityUrl(entry)"
                class="flex items-center gap-2 px-2 py-1 rounded-md text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors group"
                :style="accentStyles[entry.id]"
            >
                <span class="flex-shrink-0 w-4 text-fine text-gray-600 text-right">{{ idx + 1 }}</span>
                <div v-if="getImageUrl(entry)" class="flex-shrink-0 w-6 h-6 rounded overflow-hidden bg-white/[0.04]">
                    <EveImage :src="getImageUrl(entry)!" :size="32" :alt="entry.name" class="w-full h-full object-cover" :loading="props.eager ? 'eager' : 'lazy'" />
                </div>
                <div v-else class="flex-shrink-0 w-6 h-6 rounded bg-white/[0.04] flex items-center justify-center">
                    <Icon name="lucide:map-pin" class="text-fine text-gray-600" />
                </div>
                <span class="flex-1 text-xs text-gray-300 truncate" :class="pochvenClass(entry.type === 'region' ? entry.id : entry.region_id)">{{ entry.name }}</span>
                <span class="flex-shrink-0 text-fine tabular-nums" :class="countColor || 'text-gray-500'">{{ formatCount ? formatCount(entry) : entry.count.toLocaleString('en-US') }}</span>
            </NuxtLink>
        </div>

        <div v-if="useApi && !hideFooter" class="px-2 pt-1.5 text-fine text-gray-700">
            Last {{ days || 7 }} days
        </div>
    </div>
</template>
