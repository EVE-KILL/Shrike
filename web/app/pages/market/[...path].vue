<script setup lang="ts">
interface TreeNode {
    id: number
    name: string
    slug: string
    parent_id: number | null
    has_types: boolean
    icon_id: number | null
    children: TreeNode[]
}

const route = useRoute()
const pathSegments = computed(() => {
    const raw = route.params.path
    if (!raw) return [] as string[]
    if (Array.isArray(raw)) return raw.filter(Boolean)
    return raw.split('/').filter(Boolean)
})

const pathString = computed(() => pathSegments.value.join('/'))

// Load the full tree once
const { data: treeData } = await useApiFetch<{ groups: TreeNode[] }>('/api/market/tree', {
    getCachedData: cachedPayload,
})
const tree = computed(() => treeData.value?.groups ?? [])

// Resolve path entirely from the client-side tree — no server round-trip
const resolvedPath = computed(() => {
    const segments = pathSegments.value
    const breadcrumb: { id: number, name: string, slug: string }[] = []
    let current: TreeNode[] = tree.value
    let node: TreeNode | null = null

    for (const seg of segments) {
        const match = current.find(n => n.slug === seg)
        if (!match) break
        breadcrumb.push({ id: match.id, name: match.name, slug: match.slug })
        node = match
        current = match.children
    }

    return { node, breadcrumb }
})

// A catch-all route answers 200 for any depth of nonsense, so /market/<junk>
// used to render an empty browser at its own self-canonical URL — unbounded
// thin content for crawlers to chew through. Anything that doesn't resolve to a
// real market group is a 404. Checked at setup (the tree fetch above is
// awaited) so SSR sends the real status rather than a soft 404.
if (pathSegments.value.length > 0 && resolvedPath.value.breadcrumb.length !== pathSegments.value.length) {
    throw createError({ statusCode: 404, statusMessage: 'Market group not found' })
}

const activeNode = computed(() => resolvedPath.value.node)
const activeGroupId = computed(() => activeNode.value?.id ?? null)
const breadcrumb = computed(() => resolvedPath.value.breadcrumb)

// Load items when on a leaf group
const itemsUrl = computed(() =>
    activeNode.value?.has_types ? `/api/market/groups/${activeGroupId.value}/items` : '',
)
const { data: itemsData, pending: itemsPending } = useApiFetch<{ items: any[] }>(itemsUrl, {
    watch: [activeGroupId],
    immediate: true,
}) as any
const items = computed(() => itemsData.value?.items ?? [])

// Child groups for non-leaf nodes
const childGroups = computed(() => activeNode.value?.children ?? [])

// Show top-level groups when at root
const isRoot = computed(() => pathSegments.value.length === 0)
const showItems = computed(() => activeNode.value?.has_types ?? false)
const showSubgroups = computed(() => !showItems.value && !isRoot.value && childGroups.value.length > 0)

definePageMeta({
    key: '/market',
})

useHead({ title: computed(() => {
    if (isRoot.value) return 'Market Browser'
    const last = breadcrumb.value[breadcrumb.value.length - 1]
    return last ? `${last.name} — Market` : 'Market Browser'
}) })

// Build slug path up to a breadcrumb index
const breadcrumbPath = (idx: number) => {
    return `/market/${breadcrumb.value.slice(0, idx + 1).map(c => c.slug).join('/')}`
}

const metaGroupColor = (mgId: number | null): string => {
    if (mgId === 2) return 'text-yellow-400'
    if (mgId === 3) return 'text-teal-400'
    if (mgId === 4) return 'text-green-400'
    if (mgId === 5) return 'text-purple-400'
    if (mgId === 6) return 'text-blue-400'
    if (mgId === 14) return 'text-teal-400'
    return ''
}

const metaGroupLabel = (mgId: number | null): string => {
    if (mgId === 1) return 'Tech I'
    if (mgId === 2) return 'Tech II'
    if (mgId === 3) return 'Storyline'
    if (mgId === 4) return 'Faction'
    if (mgId === 5) return 'Officer'
    if (mgId === 6) return 'Deadspace'
    if (mgId === 14) return 'Tech III'
    return ''
}

const sidebarOpen = ref(false)
</script>

<template>
    <div>
        <!-- Breadcrumb -->
        <div class="flex items-center gap-1.5 text-xs text-gray-500 mb-4 flex-wrap">
            <NuxtLink to="/market" class="hover:text-blue-400 transition-colors">Market</NuxtLink>
            <template v-for="(crumb, idx) in breadcrumb" :key="crumb.id">
                <span class="text-gray-700">/</span>
                <NuxtLink :to="breadcrumbPath(idx)" class="hover:text-blue-400 transition-colors">{{ crumb.name }}</NuxtLink>
            </template>
        </div>

        <!-- Mobile sidebar toggle -->
        <button class="glass-panel md:hidden mb-4 px-3 py-2 text-xs text-gray-400"
            @click="sidebarOpen = !sidebarOpen">
            <Icon name="lucide:menu" class="w-3.5 h-3.5 inline mr-1" />
            {{ sidebarOpen ? 'Hide' : 'Show' }} Categories
        </button>

        <!-- Two-column layout -->
        <div class="flex flex-col md:flex-row gap-4">
            <!-- Sidebar -->
            <div :class="sidebarOpen ? 'block w-full mb-4' : 'hidden'"
                class="glass-panel md:!block md:w-64 md:mb-0 flex-shrink-0 p-3 overflow-y-auto md:max-h-[calc(100vh-200px)] md:sticky md:top-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Market Groups</div>
                <MarketTreeSidebar
                    :nodes="tree"
                    :active-path="pathSegments"
                />
            </div>

            <!-- Content -->
            <div class="flex-1 min-w-0">
                <!-- Root: top-level groups -->
                <div v-if="isRoot" class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                    <NuxtLink v-for="group in tree" :key="group.id"
                        :to="`/market/${group.slug}`"
                        class="rounded-lg border border-white/[0.08] bg-white/[0.04] p-4 hover:bg-blue-500/[0.06] hover:border-blue-500/30 transition-all">
                        <div class="text-sm font-medium text-white mb-1">{{ group.name }}</div>
                        <div class="text-xs text-gray-500">{{ group.children.length }} subcategories</div>
                    </NuxtLink>
                </div>

                <!-- Subgroups -->
                <div v-else-if="showSubgroups" class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                    <NuxtLink v-for="child in childGroups" :key="child.id"
                        :to="`/market/${pathString}/${child.slug}`"
                        class="rounded-lg border border-white/[0.08] bg-white/[0.04] p-4 hover:bg-blue-500/[0.06] hover:border-blue-500/30 transition-all">
                        <div class="text-sm font-medium text-white mb-1">{{ child.name }}</div>
                        <div class="text-xs text-gray-500">
                            {{ child.has_types ? 'Items' : `${child.children.length} subcategories` }}
                        </div>
                    </NuxtLink>
                </div>

                <!-- Items grid -->
                <div v-else-if="showItems">
                    <div v-if="itemsPending" class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
                        <div v-for="i in 12" :key="i" class="h-24 rounded-lg bg-white/[0.04] animate-pulse"></div>
                    </div>
                    <div v-else-if="items.length" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
                        <NuxtLink v-for="item in items" :key="item.type_id"
                            :to="`/item/${item.type_id}`"
                            class="rounded-lg border border-white/[0.08] bg-white/[0.04] p-3 hover:bg-blue-500/[0.06] hover:border-blue-500/30 transition-all flex items-center gap-3">
                            <img :src="item.is_ship
                                    ? `/images/types/${item.type_id}/render?size=64`
                                    : `/images/types/${item.type_id}/icon?size=64`"
                                :alt="item.name"
                                class="w-10 h-10 rounded flex-shrink-0" loading="lazy">
                            <div class="min-w-0">
                                <div class="text-sm text-white truncate">{{ item.name }}</div>
                                <div v-if="metaGroupLabel(item.meta_group_id)" class="text-fine font-medium" :class="metaGroupColor(item.meta_group_id)">
                                    {{ metaGroupLabel(item.meta_group_id) }}
                                </div>
                            </div>
                        </NuxtLink>
                    </div>
                    <div v-else class="text-center py-12 text-gray-500 text-sm">
                        No items in this category.
                    </div>
                </div>

                <!-- Fallback -->
                <div v-else class="text-center py-12 text-gray-500 text-sm">
                    Select a category from the sidebar.
                </div>
            </div>
        </div>
    </div>
</template>
