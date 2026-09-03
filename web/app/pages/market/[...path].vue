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

interface MarketGroupItem {
    type_id: number
    name: string
    group_id: number
    market_group_id: number
    category_id: number
    meta_group_id: number | null
    is_ship: boolean
    universe_average: number | null
    universe_volume: number | null
    jita_sell: number | null
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
const { data: itemsData, pending: itemsPending } = await useApiFetch<{ items: MarketGroupItem[] }>(itemsUrl, {
    watch: [activeGroupId],
    immediate: true,
})
const items = computed(() => itemsData.value?.items ?? [])

const singleItemTarget = (data = itemsData.value) => {
    const item = data?.items?.length === 1 ? data.items[0] : null
    if (!item || item.market_group_id !== activeGroupId.value) return null
    return `/market/item/${item.type_id}`
}

const initialSingleItemTarget = singleItemTarget()
if (initialSingleItemTarget) {
    await navigateTo(initialSingleItemTarget, { redirectCode: 302, replace: true })
}

watch(itemsData, (data) => {
    const target = singleItemTarget(data)
    if (target) navigateTo(target, { replace: true })
})

// Child groups for non-leaf nodes
const childGroups = computed(() => activeNode.value?.children ?? [])

// Show top-level groups when at root
const isRoot = computed(() => pathSegments.value.length === 0)
const showItems = computed(() => activeNode.value?.has_types ?? false)
const showSubgroups = computed(() => !showItems.value && !isRoot.value && childGroups.value.length > 0)
const visibleGroups = computed(() => isRoot.value ? tree.value : childGroups.value)
const groupPageTitle = computed(() => isRoot.value ? 'Market Explorer' : activeNode.value?.name ?? 'Market Group')
const groupPageDescription = computed(() => isRoot.value
    ? 'Browse the regional market by category, then compare current orders and traded history.'
    : `Choose one of ${childGroups.value.length} subcategories to narrow the market.`)

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

const groupPath = (group: TreeNode) => {
    return isRoot.value ? `/market/${group.slug}` : `/market/${pathString.value}/${group.slug}`
}

const childGroupPath = (group: TreeNode, child: TreeNode) => `${groupPath(group)}/${child.slug}`

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

const formatMarketVolume = (value: number | null): string => {
    if (value == null) return 'No recent volume'
    return `${value.toLocaleString('en-US')} traded latest day`
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
                <MarketItemSearch class="mb-3" />
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Market Groups</div>
                <MarketTreeSidebar
                    :nodes="tree"
                    :active-path="pathSegments"
                />
            </div>

            <!-- Content -->
            <div class="flex-1 min-w-0">
                <!-- Root and intermediary market groups -->
                <div v-if="isRoot || showSubgroups">
                    <div class="mb-4 rounded-lg border border-white/[0.08] bg-white/[0.025] p-5">
                        <div class="text-fine font-semibold uppercase tracking-[0.16em] text-blue-400/70">{{ isRoot ? 'Regional market' : 'Market group' }}</div>
                        <h1 class="mt-1 text-2xl font-semibold text-white">{{ groupPageTitle }}</h1>
                        <p class="mt-1 max-w-2xl text-xs leading-relaxed text-gray-500">{{ groupPageDescription }}</p>
                    </div>
                    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
                        <article v-for="group in visibleGroups" :key="group.id"
                            class="group flex min-h-32 flex-col rounded-lg border border-white/[0.08] bg-white/[0.035] p-4 transition-all hover:border-blue-500/30 hover:bg-blue-500/[0.04]">
                            <NuxtLink :to="groupPath(group)" class="flex items-start gap-3">
                                <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-md border border-white/[0.06] bg-black/20 text-blue-400/70">
                                    <Icon :name="group.has_types ? 'lucide:package-search' : 'lucide:folder-tree'" class="h-5 w-5" />
                                </div>
                                <div class="min-w-0 flex-1">
                                    <div class="text-sm font-semibold leading-snug text-white transition-colors group-hover:text-blue-300">{{ group.name }}</div>
                                    <div class="mt-1 text-fine uppercase tracking-wider text-gray-600">
                                        {{ group.has_types ? 'View market items' : `${group.children.length} subcategories` }}
                                    </div>
                                </div>
                                <Icon name="lucide:chevron-right" class="mt-1 h-4 w-4 flex-shrink-0 text-gray-700 transition-colors group-hover:text-blue-400" />
                            </NuxtLink>
                            <div v-if="group.children.length" class="mt-4 flex flex-wrap gap-1.5 border-t border-white/[0.05] pt-3">
                                <NuxtLink v-for="preview in group.children.slice(0, 4)" :key="preview.id"
                                    :to="childGroupPath(group, preview)"
                                    class="rounded border border-white/[0.06] bg-black/15 px-2 py-1 text-fine text-gray-500 transition-colors hover:border-blue-500/25 hover:text-blue-300">
                                    {{ preview.name }}
                                </NuxtLink>
                                <NuxtLink v-if="group.children.length > 4" :to="groupPath(group)" class="px-1 py-1 text-fine text-gray-700 hover:text-blue-400">
                                    +{{ group.children.length - 4 }} more
                                </NuxtLink>
                            </div>
                            <NuxtLink v-else :to="groupPath(group)" class="mt-auto border-t border-white/[0.05] pt-3 text-xs text-gray-600 transition-colors hover:text-blue-300">Open item listing →</NuxtLink>
                        </article>
                    </div>
                </div>

                <!-- Items grid -->
                <div v-else-if="showItems">
                    <div class="mb-4 rounded-lg border border-white/[0.08] bg-white/[0.025] p-4">
                        <div class="flex flex-wrap items-end justify-between gap-3">
                            <div>
                                <div class="text-fine font-semibold uppercase tracking-[0.16em] text-blue-400/70">Market group</div>
                                <h1 class="mt-1 text-xl font-semibold text-white">{{ activeNode?.name }}</h1>
                                <p class="mt-1 text-xs text-gray-500">Compare the latest universe-wide traded average with the current lowest sell order in Jita.</p>
                            </div>
                            <div v-if="!itemsPending" class="text-xs tabular-nums text-gray-500">{{ items.length.toLocaleString('en-US') }} items</div>
                        </div>
                    </div>
                    <div v-if="itemsPending" class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
                        <div v-for="i in 12" :key="i" class="h-24 rounded-lg bg-white/[0.04] animate-pulse"></div>
                    </div>
                    <div v-else-if="items.length" class="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
                        <NuxtLink v-for="item in items" :key="item.type_id"
                            :to="`/market/item/${item.type_id}`"
                            class="group rounded-lg border border-white/[0.08] bg-white/[0.035] p-4 transition-all hover:border-blue-500/30 hover:bg-blue-500/[0.06]">
                            <div class="flex items-center gap-3">
                                <img :src="item.is_ship
                                        ? `/images/types/${item.type_id}/render?size=64`
                                        : `/images/types/${item.type_id}/icon?size=64`"
                                    :alt="item.name"
                                    class="h-14 w-14 flex-shrink-0 rounded-md bg-black/20" loading="lazy">
                                <div class="min-w-0 flex-1">
                                    <div class="line-clamp-2 text-sm font-medium leading-snug text-white transition-colors group-hover:text-blue-300">{{ item.name }}</div>
                                    <div class="mt-1 flex items-center gap-2">
                                        <span v-if="metaGroupLabel(item.meta_group_id)" class="text-fine font-medium uppercase tracking-wider" :class="metaGroupColor(item.meta_group_id)">
                                            {{ metaGroupLabel(item.meta_group_id) }}
                                        </span>
                                        <span class="font-mono text-fine text-gray-700">#{{ item.type_id }}</span>
                                    </div>
                                </div>
                                <Icon name="lucide:chevron-right" class="h-4 w-4 flex-shrink-0 text-gray-700 transition-colors group-hover:text-blue-400" />
                            </div>
                            <div class="mt-3 grid grid-cols-2 gap-3 border-t border-white/[0.05] pt-3">
                                <div>
                                    <div class="text-fine font-semibold uppercase tracking-wider text-gray-600">Universe avg</div>
                                    <div class="mt-0.5 font-mono text-sm font-semibold tabular-nums text-white">{{ item.universe_average == null ? '—' : formatIsk(item.universe_average) }}</div>
                                    <div class="mt-0.5 text-fine text-gray-700">{{ formatMarketVolume(item.universe_volume) }}</div>
                                </div>
                                <div>
                                    <div class="text-fine font-semibold uppercase tracking-wider text-gray-600">Jita lowest sell</div>
                                    <div class="mt-0.5 font-mono text-sm font-semibold tabular-nums text-yellow-400">{{ item.jita_sell == null ? '—' : formatIsk(item.jita_sell) }}</div>
                                    <div class="mt-0.5 text-fine text-gray-700">Current order book</div>
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
