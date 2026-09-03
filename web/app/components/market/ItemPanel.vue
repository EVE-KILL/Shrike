<script setup lang="ts">
const props = withDefaults(defineProps<{
    typeId: number
    embedded?: boolean
}>(), {
    embedded: false,
})

interface MarketItem {
    type_id: number
    name: string
    group_id: number | null
    market_group_id: number | null
}

interface MarketTreeNode {
    id: number
    name: string
    slug: string
    parent_id: number | null
    has_types: boolean
    icon_id: number | null
    children: MarketTreeNode[]
}

interface MarketOrder {
    order_id: number
    duration: number
    is_buy_order: boolean
    issued: string
    location_id: number
    min_volume: number
    price: number
    order_range: string
    system_id: number
    type_id: number
    volume_remain: number
    volume_total: number
    region_id: number
    constellation_id: number | null
    snapshot_at: string
    system_name: string | null
    security: number | null
    region_name: string | null
    location_name: string
    expires_at: string
}

interface MarketRegion {
    region_id: number
    name: string | null
    order_count: number
    lowest_sell: number | null
    highest_buy: number | null
}

interface OrdersResponse {
    item: MarketItem
    snapshot_at: string | null
    region_id: number
    security: string[]
    sellers: MarketOrder[]
    buyers: MarketOrder[]
    regions: MarketRegion[]
}

interface HistoryPoint {
    date: string
    average: number | null
    highest: number | null
    lowest: number | null
    order_count: number | null
    volume: number | null
    source_updated_at: string | null
}

interface HistoryResponse {
    type_id: number
    region_id: number
    days: number
    history: HistoryPoint[]
}

type OrderBookKey = 'sellers' | 'buyers'
type OrderSortKey = 'quantity' | 'price' | 'min_volume' | 'expires'
type SortDirection = 'asc' | 'desc'

interface OrderSort {
    key: OrderSortKey
    direction: SortDirection
}

const route = useRoute()
const router = useRouter()
const typeId = computed(() => props.typeId)
if (!Number.isInteger(props.typeId) || props.typeId <= 0) {
    throw createError({ statusCode: 404, statusMessage: 'Market item not found' })
}

const initialRegion = Number(route.query.region ?? 0)
const regionId = ref(Number.isInteger(initialRegion) && initialRegion > 0 ? initialRegion : 0)
const security = ref<string[]>(['high', 'low', 'null'])
const historyDays = ref(Number(route.query.days) === 90 ? 90 : 30)
const sidebarOpen = ref(false)

onMounted(() => {
    if (!props.embedded && 'tab' in route.query) {
        const query = { ...route.query }
        delete query.tab
        router.replace({ query })
    }
})

const { data: treeData } = await useApiFetch<{ groups: MarketTreeNode[] }>('/api/market/tree', {
    getCachedData: cachedPayload,
})
const marketTree = computed(() => treeData.value?.groups ?? [])

const ordersURL = computed(() => {
    const query = new URLSearchParams({ limit: '100', security: security.value.join(',') })
    if (regionId.value) query.set('region_id', String(regionId.value))
    return `/api/market/items/${typeId.value}/orders?${query}`
})
const historyURL = computed(() => {
    const query = new URLSearchParams({ days: String(historyDays.value) })
    if (regionId.value) query.set('region_id', String(regionId.value))
    return `/api/market/items/${typeId.value}/history?${query}`
})

const { data: orders, pending: ordersPending, error: ordersError } = await useApiFetch<OrdersResponse>(ordersURL, {
    watch: [ordersURL],
})
const { data: history, pending: historyPending } = await useApiFetch<HistoryResponse>(historyURL, {
    watch: [historyURL],
})

if (ordersError.value) {
    throw createError({ statusCode: 404, statusMessage: 'Market item not found' })
}

const item = computed(() => orders.value?.item)
const regions = computed(() => orders.value?.regions ?? [])
const selectedRegion = computed(() => regions.value.find(region => region.region_id === regionId.value))
const orderSort = reactive<Record<OrderBookKey, OrderSort>>({
    sellers: { key: 'price', direction: 'asc' },
    buyers: { key: 'price', direction: 'desc' },
})
const activeMarketPath = computed(() => {
    const wanted = item.value?.market_group_id
    if (!wanted) return [] as string[]
    const visit = (nodes: MarketTreeNode[], path: string[]): string[] | null => {
        for (const node of nodes) {
            const next = [...path, node.slug]
            if (node.id === wanted) return next
            const nested = visit(node.children, next)
            if (nested) return nested
        }
        return null
    }
    return visit(marketTree.value, []) ?? []
})

if (!props.embedded) {
    useSeoMeta({
        title: computed(() => item.value ? `${item.value.name} Market` : 'Market'),
        description: computed(() => item.value
            ? `Current regional buy and sell orders plus ${historyDays.value}-day market history for ${item.value.name}.`
            : 'Regional EVE Online market data.'),
    })
}

watch([regionId, historyDays], () => {
    const query: Record<string, string> = {}
    if (regionId.value) query.region = String(regionId.value)
    if (historyDays.value !== 30) query.days = String(historyDays.value)
    router.replace({ query })
})

function toggleSecurity(value: string) {
    if (security.value.includes(value)) {
        if (security.value.length === 1) return
        security.value = security.value.filter(entry => entry !== value)
    } else {
        security.value = [...security.value, value]
    }
}

function securityClass(value: number | null): string {
    if (value == null) return 'text-gray-500'
    if (value >= 0.5) return 'text-blue-400'
    if (value > 0) return 'text-orange-400'
    return 'text-red-400'
}

function securityLabel(value: number | null): string {
    return value == null ? '?' : Math.max(0, value).toFixed(1)
}

function formatPrice(value: number): string {
    return new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 }).format(value)
}

function expiresIn(value: string): string {
    const seconds = Math.max(0, Math.floor((new Date(value).getTime() - Date.now()) / 1000))
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    if (days > 0) return `${days}d ${hours}h`
    const minutes = Math.floor((seconds % 3600) / 60)
    return `${hours}h ${minutes}m`
}

function orderSortValue(order: MarketOrder, key: OrderSortKey): number {
    if (key === 'quantity') return order.volume_remain
    if (key === 'price') return order.price
    if (key === 'min_volume') return order.min_volume
    return new Date(order.expires_at).getTime()
}

function sortedOrders(rows: MarketOrder[], sort: OrderSort): MarketOrder[] {
    const direction = sort.direction === 'asc' ? 1 : -1
    return [...rows].sort((left, right) => {
        const difference = orderSortValue(left, sort.key) - orderSortValue(right, sort.key)
        return difference === 0 ? left.order_id - right.order_id : difference * direction
    })
}

function toggleOrderSort(book: OrderBookKey, key: OrderSortKey) {
    const current = orderSort[book]
    if (current.key === key) {
        current.direction = current.direction === 'asc' ? 'desc' : 'asc'
        return
    }
    current.key = key
    current.direction = key === 'quantity' || (key === 'price' && book === 'buyers') ? 'desc' : 'asc'
}

function orderSortIcon(book: OrderBookKey, key: OrderSortKey): string {
    const current = orderSort[book]
    if (current.key !== key) return 'lucide:chevrons-up-down'
    return current.direction === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'
}

function orderAriaSort(book: OrderBookKey, key: OrderSortKey): 'ascending' | 'descending' | 'none' {
    const current = orderSort[book]
    if (current.key !== key) return 'none'
    return current.direction === 'asc' ? 'ascending' : 'descending'
}

const orderBooks = computed(() => [
    {
        key: 'sellers' as const,
        label: 'Sellers',
        rows: sortedOrders(orders.value?.sellers ?? [], orderSort.sellers),
        color: 'text-orange-400',
    },
    {
        key: 'buyers' as const,
        label: 'Buyers',
        rows: sortedOrders(orders.value?.buyers ?? [], orderSort.buyers),
        color: 'text-green-400',
    },
])

const chart = computed(() => {
    const source = (history.value?.history ?? []).filter(point => point.average != null) as (HistoryPoint & { average: number })[]
    const width = 900
    const height = 205
    const left = 66
    const right = 18
    const top = 14
    const bottom = 34
    if (source.length === 0) return { points: '', area: '', bars: [], width, height, left, right, top, bottom, min: 0, max: 0, source }
    const values = source.map(point => point.average)
    let min = Math.min(...values)
    let max = Math.max(...values)
    const padding = Math.max((max - min) * 0.12, max * 0.015, 0.01)
    min = Math.max(0, min - padding)
    max += padding
    const plotWidth = width - left - right
    const plotHeight = height - top - bottom
    const x = (index: number) => left + ((index + 0.5) / source.length) * plotWidth
    const y = (value: number) => top + ((max - value) / Math.max(0.0001, max - min)) * (height - top - bottom)
    const points = source.map((point, index) => `${x(index)},${y(point.average)}`).join(' ')
    const area = `${x(0)},${height - bottom} ${points} ${x(source.length - 1)},${height - bottom}`
    const maxVolume = Math.max(...source.map(point => point.volume ?? 0), 1)
    const barWidth = Math.max(2, Math.min(16, (plotWidth / source.length) * 0.62))
    const bars = source.map((point, index) => {
        const barHeight = ((point.volume ?? 0) / maxVolume) * plotHeight * 0.42
        return {
            point,
            x: x(index) - barWidth / 2,
            y: height - bottom - barHeight,
            width: barWidth,
            height: barHeight,
        }
    })
    return { points, area, bars, width, height, left, right, top, bottom, min, max, source }
})

const chartTicks = computed(() => Array.from({ length: 5 }, (_, index) => {
    const ratio = index / 4
    return {
        y: chart.value.top + ratio * (chart.value.height - chart.value.top - chart.value.bottom),
        value: chart.value.max - ratio * (chart.value.max - chart.value.min),
    }
}))
</script>

<template>
    <div>
        <div v-if="!embedded" class="mb-4 flex items-center gap-1.5 text-xs text-gray-500">
            <NuxtLink to="/market" class="transition-colors hover:text-blue-400">Market</NuxtLink>
            <span class="text-gray-700">/</span>
            <span class="text-gray-300">{{ item?.name ?? `Type ${typeId}` }}</span>
        </div>

        <button v-if="!embedded" class="glass-panel mb-4 px-3 py-2 text-xs text-gray-400 md:hidden" @click="sidebarOpen = !sidebarOpen">
            <Icon name="lucide:menu" class="mr-1 inline h-3.5 w-3.5" />
            {{ sidebarOpen ? 'Hide' : 'Show' }} Market Browser
        </button>

        <div :class="embedded ? '' : 'flex flex-col gap-4 md:flex-row'">
            <aside v-if="!embedded" :class="sidebarOpen ? 'block' : 'hidden'" class="glass-panel shrink-0 p-3 md:sticky md:top-4 md:!block md:h-[calc(100vh-7rem)] md:w-72">
                <MarketItemSearch class="mb-3" />
                <div class="mb-3 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Browse</div>
                <div class="h-[calc(100%-4.5rem)] overflow-y-auto pr-1">
                    <MarketTreeSidebar :nodes="marketTree" :active-path="activeMarketPath" />
                    <NuxtLink to="/market" class="flex items-center gap-2 rounded-md px-2.5 py-2 text-xs text-gray-400 hover:bg-blue-500/[0.06] hover:text-blue-300">
                        <Icon name="lucide:layout-grid" class="h-4 w-4" />
                        Browse all categories
                    </NuxtLink>
                </div>
            </aside>

            <main class="min-w-0 flex-1 overflow-hidden rounded-lg border border-white/[0.08] bg-white/[0.025]">
                <header v-if="!embedded" class="flex flex-col gap-4 border-b border-white/[0.07] p-5 sm:flex-row sm:items-center sm:justify-between">
                    <div class="flex min-w-0 items-center gap-4">
                        <img :src="`/images/types/${typeId}/icon?size=64`" :alt="item?.name ?? ''" class="h-14 w-14 rounded-md bg-black/30">
                        <div class="min-w-0">
                            <div class="text-fine uppercase tracking-[0.16em] text-gray-600">Regional Market</div>
                            <h1 class="truncate text-2xl font-bold text-white">{{ item?.name ?? `Type ${typeId}` }}</h1>
                            <div class="mt-1 flex items-center gap-3 text-xs text-gray-500">
                                <span>Type ID: <span class="font-mono text-gray-400">{{ typeId }}</span></span>
                                <NuxtLink :to="`/item/${typeId}`" class="text-blue-400 hover:text-blue-300">Item details</NuxtLink>
                            </div>
                        </div>
                    </div>
                    <label class="flex items-center gap-2 text-xs text-gray-500">
                        Region
                        <select v-model.number="regionId" class="min-w-48 rounded-md border border-white/10 bg-[#101114] px-3 py-2 text-xs text-gray-200 outline-none focus:border-blue-500/40">
                            <option :value="0">All Regions</option>
                            <option v-for="region in regions" :key="region.region_id" :value="region.region_id">
                                {{ region.name ?? region.region_id }}
                            </option>
                        </select>
                    </label>
                </header>
                <div v-else class="flex items-center justify-between gap-3 border-b border-white/[0.07] px-4 py-3">
                    <div class="flex items-center gap-2 text-xs font-semibold text-gray-400">
                        <Icon name="lucide:shopping-cart" class="h-4 w-4 text-blue-400" />
                        Regional market
                    </div>
                    <label class="flex items-center gap-2 text-xs text-gray-500">
                        Region
                        <select v-model.number="regionId" class="min-w-48 rounded-md border border-white/10 bg-[#101114] px-3 py-2 text-xs text-gray-200 outline-none focus:border-blue-500/40">
                            <option :value="0">All Regions</option>
                            <option v-for="region in regions" :key="region.region_id" :value="region.region_id">
                                {{ region.name ?? region.region_id }}
                            </option>
                        </select>
                    </label>
                </div>

                <div class="space-y-6 p-4 sm:p-5">
                    <section>
                        <div class="mb-4 flex items-center justify-between gap-3">
                            <div>
                                <h2 class="text-sm font-semibold text-white">{{ selectedRegion?.name ?? 'All Regions' }} price history</h2>
                                <p class="mt-1 text-xs text-gray-600">Daily traded price, corrected when EVERef republishes historical data.</p>
                            </div>
                            <div class="inline-flex rounded-md bg-black/30 p-1">
                                <button v-for="days in [30, 90]" :key="days" type="button" class="rounded px-3 py-1.5 text-xs" :class="historyDays === days ? 'bg-blue-500/15 text-blue-300' : 'text-gray-600'" @click="historyDays = days">{{ days }}d</button>
                            </div>
                        </div>
                        <div class="overflow-hidden rounded-md border border-white/[0.06] bg-black/20">
                            <div v-if="historyPending && !history" class="py-16 text-center text-sm text-gray-500">Loading price history…</div>
                            <div v-else-if="chart.source.length" class="overflow-x-auto p-3">
                                <svg :viewBox="`0 0 ${chart.width} ${chart.height}`" class="min-w-[720px] w-full" role="img" :aria-label="`${historyDays}-day average price and traded volume`">
                                    <defs><linearGradient id="marketArea" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#3b82f6" stop-opacity="0.25"/><stop offset="1" stop-color="#3b82f6" stop-opacity="0"/></linearGradient></defs>
                                    <g v-for="tick in chartTicks" :key="tick.y">
                                        <line :x1="chart.left" :x2="chart.width - chart.right" :y1="tick.y" :y2="tick.y" stroke="rgba(255,255,255,0.06)" />
                                        <text :x="chart.left - 9" :y="tick.y + 4" text-anchor="end" fill="#62646b" font-size="11">{{ formatPrice(tick.value) }}</text>
                                    </g>
                                    <rect v-for="bar in chart.bars" :key="bar.point.date" :x="bar.x" :y="bar.y" :width="bar.width" :height="bar.height" rx="1" fill="#34d399" opacity="0.28">
                                        <title>{{ formatDate(bar.point.date) }} — {{ (bar.point.volume ?? 0).toLocaleString('en-US') }} units traded</title>
                                    </rect>
                                    <polygon :points="chart.area" fill="url(#marketArea)" />
                                    <polyline :points="chart.points" fill="none" stroke="#60a5fa" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
                                    <text :x="chart.left" :y="chart.height - 10" fill="#62646b" font-size="11">{{ formatDate(chart.source[0]?.date) }}</text>
                                    <text :x="chart.width - chart.right" :y="chart.height - 10" text-anchor="end" fill="#62646b" font-size="11">{{ formatDate(chart.source.at(-1)?.date) }}</text>
                                </svg>
                            </div>
                            <div v-else class="py-16 text-center text-sm text-gray-600">No history is available for this region and item yet.</div>
                            <div class="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-white/[0.06] px-3 py-2.5">
                                <div class="flex items-center gap-3 text-fine text-gray-600">
                                    <span class="flex items-center gap-1.5"><span class="h-0.5 w-4 bg-blue-400" />Average price</span>
                                    <span class="flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-sm bg-emerald-400/40" />Units traded</span>
                                </div>
                                <div class="flex flex-wrap items-center gap-2 sm:ml-auto">
                                    <span class="mr-1 text-fine font-bold uppercase tracking-[0.14em] text-gray-600">Order security</span>
                                    <button v-for="option in [{ id: 'high', label: 'High', color: 'bg-green-400' }, { id: 'low', label: 'Low', color: 'bg-orange-400' }, { id: 'null', label: 'Null', color: 'bg-red-400' }]" :key="option.id" type="button" class="flex items-center gap-1.5 rounded border px-2 py-1 text-fine transition-colors" :class="security.includes(option.id) ? 'border-white/10 bg-white/[0.06] text-gray-300' : 'border-white/[0.05] text-gray-700'" @click="toggleSecurity(option.id)">
                                        <span class="h-1.5 w-1.5 rounded-full" :class="option.color" />{{ option.label }}
                                    </button>
                                </div>
                                <span class="text-fine text-gray-700">Snapshot {{ formatDateTime(orders?.snapshot_at) }}</span>
                            </div>
                        </div>
                    </section>

                    <div v-if="ordersPending && !orders" class="py-20 text-center text-sm text-gray-500">Loading regional orders…</div>
                    <template v-else>
                        <section v-for="book in orderBooks" :key="book.key">
                            <div class="mb-2 flex items-center gap-2 text-xs font-bold uppercase tracking-[0.16em] text-gray-500">
                                <span class="h-3 w-0.5" :class="book.key === 'sellers' ? 'bg-orange-500' : 'bg-green-500'" />
                                {{ book.label }}
                                <span class="font-normal text-gray-700">({{ book.rows.length }})</span>
                            </div>
                            <div class="max-h-[25rem] overflow-auto rounded-md border border-white/[0.06] bg-black/15">
                                <table class="w-full min-w-[760px] text-xs">
                                    <thead class="sticky top-0 z-10 bg-[#111216] text-left text-fine uppercase tracking-wider text-gray-600">
                                        <tr>
                                            <th class="px-3 py-2" :aria-sort="orderAriaSort(book.key, 'quantity')">
                                                <button type="button" class="flex items-center gap-1 whitespace-nowrap hover:text-gray-300" @click="toggleOrderSort(book.key, 'quantity')">Quantity <Icon :name="orderSortIcon(book.key, 'quantity')" class="h-3 w-3" /></button>
                                            </th>
                                            <th class="px-3 py-2" :aria-sort="orderAriaSort(book.key, 'price')">
                                                <button type="button" class="flex items-center gap-1 whitespace-nowrap hover:text-gray-300" @click="toggleOrderSort(book.key, 'price')">Price (ISK) <Icon :name="orderSortIcon(book.key, 'price')" class="h-3 w-3" /></button>
                                            </th>
                                            <th class="px-3 py-2">Location</th>
                                            <th class="px-3 py-2" :aria-sort="orderAriaSort(book.key, 'min_volume')">
                                                <button type="button" class="ml-auto flex items-center gap-1 whitespace-nowrap hover:text-gray-300" @click="toggleOrderSort(book.key, 'min_volume')">Min. volume <Icon :name="orderSortIcon(book.key, 'min_volume')" class="h-3 w-3" /></button>
                                            </th>
                                            <th class="px-3 py-2" :aria-sort="orderAriaSort(book.key, 'expires')">
                                                <button type="button" class="ml-auto flex items-center gap-1 whitespace-nowrap hover:text-gray-300" @click="toggleOrderSort(book.key, 'expires')">Expires <Icon :name="orderSortIcon(book.key, 'expires')" class="h-3 w-3" /></button>
                                            </th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        <tr v-for="order in book.rows" :key="order.order_id" class="border-t border-white/[0.04] hover:bg-white/[0.025]">
                                            <td class="px-3 py-2 font-mono tabular-nums text-gray-300">{{ order.volume_remain.toLocaleString() }}</td>
                                            <td class="px-3 py-2 font-mono tabular-nums" :class="book.color">{{ formatPrice(order.price) }}</td>
                                            <td class="max-w-md truncate px-3 py-2 text-gray-200" :title="`${order.location_name}, ${order.region_name ?? ''}`">
                                                {{ order.location_name }}
                                                <NuxtLink :to="`/system/${order.system_id}`" class="ml-1 font-mono" :class="securityClass(order.security)">({{ securityLabel(order.security) }})</NuxtLink>
                                            </td>
                                            <td class="px-3 py-2 text-right font-mono tabular-nums text-gray-500">{{ order.min_volume.toLocaleString() }}</td>
                                            <td class="px-3 py-2 text-right font-mono tabular-nums text-gray-600">{{ expiresIn(order.expires_at) }}</td>
                                        </tr>
                                        <tr v-if="book.rows.length === 0"><td colspan="5" class="px-3 py-8 text-center text-gray-600">No matching {{ book.label.toLowerCase() }}.</td></tr>
                                    </tbody>
                                </table>
                            </div>
                        </section>
                    </template>
                </div>
            </main>
        </div>
    </div>
</template>
