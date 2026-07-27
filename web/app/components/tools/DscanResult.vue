<script setup lang="ts">
const props = defineProps<{
    result: {
        grouped: Record<string, any>
        totalCount: number
        uniqueTypes: number
    }
}>()

const sortedCategories = computed(() => {
    if (!props.result?.grouped) return []
    const entries = Object.entries(props.result.grouped) as [string, any][]

    return entries
        .map(([catName, cat]) => {
            const groups = Object.entries(cat.groups) as [string, any][]
            const sortedGroups = groups
                .map(([grpName, grp]) => ({
                    name: grpName,
                    groupId: grp.groupId,
                    types: grp.types as { typeName: string, typeId: number | null, count: number }[],
                    totalCount: (grp.types as any[]).reduce((s: number, t: any) => s + t.count, 0),
                }))
                .sort((a, b) => b.totalCount - a.totalCount)

            return {
                name: catName,
                categoryId: cat.categoryId,
                groups: sortedGroups,
                totalCount: sortedGroups.reduce((s, g) => s + g.totalCount, 0),
            }
        })
        .sort((a, b) => {
            if (a.name === 'Ship' && b.name !== 'Ship') return -1
            if (b.name === 'Ship' && a.name !== 'Ship') return 1
            return b.totalCount - a.totalCount
        })
})

const collapsedGroups = ref<Record<string, boolean>>({})

function toggleGroup(key: string) {
    collapsedGroups.value[key] = !collapsedGroups.value[key]
}
</script>

<template>
    <div class="space-y-4">
        <!-- Summary bar -->
        <div class="glass-panel p-3">
            <div class="flex items-center justify-between">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Scan Results</div>
                <div class="flex gap-4">
                    <span class="text-xs text-gray-500">
                        <span class="text-gray-300 tabular-nums">{{ result.totalCount }}</span> objects
                    </span>
                    <span class="text-xs text-gray-500">
                        <span class="text-gray-300 tabular-nums">{{ result.uniqueTypes }}</span> types
                    </span>
                </div>
            </div>
        </div>

        <!-- Categories -->
        <div v-for="cat in sortedCategories" :key="cat.name" class="glass-panel overflow-hidden">
            <div class="px-3 py-2 border-b border-white/[0.08] flex items-center justify-between">
                <span class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">{{ cat.name }}</span>
                <span class="text-xs text-gray-500 tabular-nums">{{ cat.totalCount }}</span>
            </div>

            <div class="divide-y divide-white/[0.04]">
                <div v-for="grp in cat.groups" :key="grp.name">
                    <button
                        class="w-full px-3 py-2 flex items-center justify-between hover:bg-blue-500/[0.04] transition-colors"
                        @click="toggleGroup(`${cat.name}-${grp.name}`)"
                    >
                        <div class="flex items-center gap-2">
                            <Icon
                                :name="collapsedGroups[`${cat.name}-${grp.name}`] ? 'lucide:chevron-right' : 'lucide:chevron-down'"
                                class="text-gray-600 text-xs"
                            />
                            <span class="text-xs text-gray-300">{{ grp.name }}</span>
                        </div>
                        <span class="text-xs text-gray-600 tabular-nums">{{ grp.totalCount }}</span>
                    </button>

                    <div v-if="!collapsedGroups[`${cat.name}-${grp.name}`]">
                        <div
                            v-for="t in grp.types"
                            :key="t.typeName"
                            class="px-3 py-1 pl-9 flex items-center justify-between text-xs hover:bg-blue-500/[0.04] transition-colors"
                        >
                            <div class="flex items-center gap-2">
                                <img
                                    v-if="t.typeId"
                                    :src="`/images/types/${t.typeId}/icon?size=64`"
                                    :alt="t.typeName"
                                    class="w-5 h-5 rounded"
                                    loading="lazy"
                                />
                                <span class="text-gray-500">{{ t.typeName }}</span>
                            </div>
                            <span class="text-gray-600 tabular-nums">{{ t.count }}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
