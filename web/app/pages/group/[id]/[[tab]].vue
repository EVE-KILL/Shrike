<script setup lang="ts">
const route = useRoute()
const id = Number(route.params.id)

if (!Number.isInteger(id) || id < 1 || id > 2147483647) {
    throw createError({ statusCode: 404, statusMessage: 'Group not found' })
}

interface GroupType {
    type_id: number
    name: string | null
    description: string | null
    published: boolean
    meta_group_id: number | null
    meta_group_name: string | null
    volume: number | null
    mass: number | null
    base_price: number | null
}

interface GroupResponse {
    group: {
        group_id: number
        name: string | null
        category_id: number | null
        category_name: string | null
        published: boolean
        icon_id: number | null
        type_count: number
        published_type_count: number
    }
    types: GroupType[]
}

const { data, pending, error } = await useApiFetch<GroupResponse>(`/api/group/${id}`)
if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'Group not found',
    })
}

const group = computed(() => data.value?.group)
const types = computed(() => data.value?.types ?? [])
const publishedTypes = computed(() => types.value.filter(type => type.published))
const unpublishedTypes = computed(() => types.value.filter(type => !type.published))
const isShipGroup = computed(() => group.value?.category_id === 6)

const groupDescription = computed(() => {
    const g = group.value
    if (!g?.name) return ''
    const noun = isShipGroup.value ? 'ship hulls' : 'inventory types'
    return `${g.name} is an EVE Online ${g.category_name?.toLowerCase() ?? 'inventory'} group containing ${g.published_type_count.toLocaleString('en-US')} published ${noun}. Explore every type in the group and its recent combat losses.`
})

const metaGroups = computed(() => {
    const counts = new Map<string, number>()
    for (const type of publishedTypes.value) {
        const name = type.meta_group_name || 'Tech I'
        counts.set(name, (counts.get(name) ?? 0) + 1)
    }
    return [...counts.entries()]
        .map(([name, count]) => ({ name, count }))
        .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name))
})

type TabId = 'dashboard' | 'kills'
const tabs: ReadonlyArray<{ id: TabId; label: string; icon: string }> = [
    { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
    { id: 'kills', label: 'Kills', icon: 'lucide:skull' },
]
const tabIds = new Set<TabId>(tabs.map(tab => tab.id))

// Item and inventory-group pages share one preference: they are two levels of
// the same browsing flow and expose the same Dashboard/Kills choice.
useDefaultTab('item', `/group/${id}`, 'dashboard', tabIds)

definePageMeta({
    key: route => `/group/${route.params.id}`,
})

const activeTab = computed<TabId>(() => {
    const tab = route.params.tab as TabId | undefined
    return tab && tabIds.has(tab) ? tab : 'dashboard'
})

const setTab = (tab: string) => {
    if (!tabIds.has(tab as TabId)) return
    useAnalytics().track('tab.change', { entity: 'group', tab })
    navigateTo(tab === 'dashboard' ? `/group/${id}` : `/group/${id}/${tab}`)
}

const groupKillFilters = computed(() => JSON.stringify({
    entities: { victim: [{ id, type: 'shipgroup' }] },
    timeRange: { preset: '90d' },
}))

useHead({
    title: computed(() => {
        const name = group.value?.name || 'Item Group'
        return activeTab.value === 'kills' ? `${name} (Kills)` : name
    }),
})

useSeoMeta({
    description: groupDescription,
    ogTitle: computed(() => group.value?.name ? `${group.value.name} — EVE Online item group` : 'Item Group — EVE-KILL'),
    ogDescription: groupDescription,
    ogType: 'website',
})

useSchemaOrg([defineBreadcrumb(computed(() => ({
    itemListElement: [
        { name: 'Home', item: '/' },
        { name: group.value?.category_name ?? 'Items', item: '/market' },
        { name: group.value?.name ?? 'Group', item: `/group/${id}` },
    ],
})))])
</script>

<template>
    <div>
        <EntityHeader v-if="pending" loading />

        <div v-else-if="group">
            <EntityHeader :background-image="publishedTypes[0] ? `/images/types/${publishedTypes[0].type_id}/${isShipGroup ? 'render' : 'icon'}?size=1024` : null">
                <div class="flex items-center gap-3 mb-1 flex-wrap">
                    <h1 class="text-2xl md:text-3xl font-bold text-white">{{ group.name }}</h1>
                    <span v-if="group.category_name" class="px-2 py-0.5 rounded text-fine font-bold uppercase tracking-wider text-blue-400 bg-white/[0.06]">
                        {{ group.category_name }}
                    </span>
                    <span v-if="!group.published" class="px-2 py-0.5 rounded text-fine font-bold uppercase tracking-wider text-amber-400 bg-white/[0.06]">
                        Unpublished
                    </span>
                </div>

                <div class="flex items-center gap-1.5 text-xs text-gray-500 mb-4">
                    <NuxtLink to="/market" class="hover:text-blue-400 transition-colors">Market</NuxtLink>
                    <span class="text-gray-700">/</span>
                    <span>{{ group.category_name ?? 'Uncategorized' }}</span>
                    <span class="text-gray-700">/</span>
                    <span class="text-gray-400">{{ group.name }}</span>
                </div>

                <p class="text-fine text-gray-400 leading-relaxed max-w-3xl">{{ groupDescription }}</p>

                <template #stats>
                    <div class="group-hull-strip flex items-end gap-1.5 overflow-x-auto overflow-y-visible pt-7 pb-1 scrollbar-hide">
                        <NuxtLink v-for="type in publishedTypes" :key="type.type_id" :to="`/item/${type.type_id}`"
                            class="group-hull shrink-0 relative z-0 hover:z-20 focus:z-20" :title="type.name ?? `Type #${type.type_id}`">
                            <img :src="`/images/types/${type.type_id}/${isShipGroup ? 'render' : 'icon'}?size=64`"
                                :alt="type.name ?? ''" class="w-10 h-10 object-contain rounded-md bg-black/20 transition-transform duration-150 origin-bottom hover:scale-[1.8] focus:scale-[1.8]" loading="eager">
                        </NuxtLink>
                    </div>
                </template>
            </EntityHeader>

            <EntityTabBar :tabs="tabs" :active-id="activeTab" @select="setTab" />

            <div v-if="activeTab === 'dashboard'" class="space-y-4">
                <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div class="glass-panel p-4">
                        <div class="text-fine uppercase tracking-wider text-gray-500">Published Types</div>
                        <div class="text-xl font-bold text-white tabular-nums">{{ group.published_type_count.toLocaleString('en-US') }}</div>
                    </div>
                    <div class="glass-panel p-4">
                        <div class="text-fine uppercase tracking-wider text-gray-500">All Types</div>
                        <div class="text-xl font-bold text-gray-300 tabular-nums">{{ group.type_count.toLocaleString('en-US') }}</div>
                    </div>
                    <div class="glass-panel p-4">
                        <div class="text-fine uppercase tracking-wider text-gray-500">Technology Families</div>
                        <div class="text-xl font-bold text-indigo-400 tabular-nums">{{ metaGroups.length }}</div>
                    </div>
                    <div class="glass-panel p-4">
                        <div class="text-fine uppercase tracking-wider text-gray-500">Group ID</div>
                        <div class="text-xl font-bold text-blue-400 tabular-nums">{{ group.group_id }}</div>
                    </div>
                </div>

                <div v-if="metaGroups.length" class="glass-panel p-3.5">
                    <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                        <Icon name="lucide:layers" class="w-4 h-4 text-indigo-400" />
                        <h2 class="text-xs font-semibold text-gray-300">Technology Breakdown</h2>
                    </div>
                    <div class="flex flex-wrap gap-2">
                        <div v-for="meta in metaGroups" :key="meta.name" class="flex items-center gap-2 rounded-md border border-white/[0.06] bg-white/[0.025] px-3 py-2">
                            <span class="text-xs text-gray-300">{{ meta.name }}</span>
                            <span class="text-fine text-gray-600 tabular-nums">{{ meta.count }}</span>
                        </div>
                    </div>
                </div>

                <div class="glass-panel p-3.5">
                    <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                        <Icon name="lucide:boxes" class="w-4 h-4 text-blue-400" />
                        <h2 class="text-xs font-semibold text-gray-300">Published Types</h2>
                        <span class="text-fine text-gray-600">({{ publishedTypes.length }})</span>
                    </div>
                    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2">
                        <NuxtLink v-for="type in publishedTypes" :key="type.type_id" :to="`/item/${type.type_id}`"
                            class="flex items-center gap-3 p-2 rounded-md border border-white/[0.05] hover:border-blue-500/30 hover:bg-blue-500/[0.06] transition-colors min-w-0">
                            <img :src="`/images/types/${type.type_id}/${isShipGroup ? 'render' : 'icon'}?size=64`" :alt="type.name ?? ''"
                                class="w-12 h-12 object-contain rounded shrink-0" loading="lazy">
                            <div class="min-w-0">
                                <div class="text-sm text-gray-200 truncate">{{ type.name ?? `Type #${type.type_id}` }}</div>
                                <div class="text-fine text-gray-600 truncate">
                                    <span v-if="type.meta_group_name">{{ type.meta_group_name }} · </span>#{{ type.type_id }}
                                </div>
                            </div>
                        </NuxtLink>
                    </div>

                    <details v-if="unpublishedTypes.length" class="mt-4 pt-3 border-t border-white/[0.04]">
                        <summary class="text-xs text-gray-500 hover:text-gray-300 cursor-pointer">Show {{ unpublishedTypes.length }} unpublished types</summary>
                        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2 mt-3">
                            <NuxtLink v-for="type in unpublishedTypes" :key="type.type_id" :to="`/item/${type.type_id}`"
                                class="flex items-center gap-2 p-2 rounded-md text-xs text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04] transition-colors">
                                <img :src="`/images/types/${type.type_id}/icon?size=32`" :alt="type.name ?? ''" class="w-8 h-8 rounded opacity-60" loading="lazy">
                                <span class="truncate">{{ type.name ?? `Type #${type.type_id}` }}</span>
                            </NuxtLink>
                        </div>
                    </details>
                </div>
            </div>

            <div v-if="activeTab === 'kills'">
                <KillList entity-endpoint="/api/killlist/advanced" :extra-params="{ filters: groupKillFilters }" />
            </div>
        </div>
    </div>
</template>

<style scoped>
.group-hull-strip {
    min-height: 5rem;
}
</style>
