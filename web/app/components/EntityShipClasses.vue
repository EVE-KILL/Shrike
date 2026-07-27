<script setup lang="ts">
interface ShipGroup {
    group_id: number
    group_name: string
    losses: number
    isk_lost: number
}

const props = defineProps<{
    entityType: 'character' | 'corporation' | 'alliance'
    entityId: number
}>()

const endpoint = computed(() => `/api/entity/${props.entityType}/${props.entityId}/ship-classes`)

const { data, pending } = await useApiFetch<{ groups: ShipGroup[] }>(endpoint, {
    default: () => ({ groups: [] }),
    lazy: true,
})

// API returns groups sorted by losses DESC — keep that order for the summary
// so [0] is the most-lost class. Re-sort alphabetically for the grid so
// users can scan to find a specific class.
const byLosses = computed(() => data.value?.groups || [])
const alphabetical = computed(() => [...byLosses.value].sort((a, b) =>
    a.group_name.localeCompare(b.group_name),
))

const summary = computed(() => {
    const top = byLosses.value[0]
    if (!top) return null
    return {
        count: byLosses.value.length,
        topName: top.group_name,
        topLosses: top.losses,
    }
})

const expanded = ref(false)
const entityPath = computed(() => `/${props.entityType}/${props.entityId}`)
</script>

<template>
    <div v-if="pending || byLosses.length" class="glass-panel mb-6">
        <button
            class="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-white/[0.02] transition-colors"
            @click="expanded = !expanded"
        >
            <Icon name="lucide:layers" class="w-4 h-4 text-blue-400/80 flex-shrink-0" />
            <div class="flex-1 min-w-0">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">
                    Ship Classes
                </div>
                <div v-if="pending" class="text-xs text-gray-500 mt-0.5">Loading…</div>
                <div v-else-if="summary" class="text-xs text-gray-400 mt-0.5">
                    <span class="text-gray-500">{{ summary.count }} hull groups · top:</span>
                    <span class="text-gray-300 ml-1">{{ summary.topName }}</span>
                    <span class="text-red-400/80 ml-2">{{ summary.topLosses.toLocaleString('en-US') }} lost</span>
                </div>
            </div>
            <Icon
                :name="expanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                class="w-4 h-4 text-gray-500 flex-shrink-0"
            />
        </button>

        <div v-if="expanded && alphabetical.length" class="px-3 pb-3 border-t border-white/[0.04] pt-3">
            <div class="ship-class-columns">
                <div
                    v-for="g in alphabetical"
                    :key="g.group_id"
                    class="ship-class-row grid grid-cols-[1fr_auto] gap-1 items-center text-xs"
                >
                    <NuxtLink
                        :to="`${entityPath}/combined?ship_group=${g.group_id}`"
                        class="px-2 py-1 rounded text-gray-300 truncate hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors"
                    >
                        {{ g.group_name }}
                    </NuxtLink>
                    <NuxtLink
                        :to="`${entityPath}/losses?ship_group=${g.group_id}`"
                        class="px-2 py-1 rounded tabular-nums text-right min-w-[3rem] transition-colors"
                        :class="g.losses > 0
                            ? 'text-red-400/80 hover:bg-red-500/[0.10] hover:text-red-300'
                            : 'text-gray-700 pointer-events-none'"
                    >
                        {{ g.losses.toLocaleString('en-US') }}
                    </NuxtLink>
                </div>
            </div>
            <div class="px-2 pt-2 mt-2 border-t border-white/[0.04] flex items-center gap-4 text-fine text-gray-600">
                <div class="flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-sm bg-red-400/60"></span>
                    <span>lost → /losses</span>
                </div>
                <div class="ml-auto">click any cell to filter by hull</div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.ship-class-columns {
    column-count: 1;
    column-gap: 1.5rem;
}
@media (min-width: 640px) {
    .ship-class-columns { column-count: 2; }
}
@media (min-width: 1024px) {
    .ship-class-columns { column-count: 3; }
}
@media (min-width: 1280px) {
    .ship-class-columns { column-count: 4; }
}
.ship-class-row {
    break-inside: avoid;
}
</style>
