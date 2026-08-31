<script setup lang="ts">
const route = useRoute()
const id = Number(route.params.id)

if (!Number.isInteger(id) || id < 1 || id > 2147483647) {
    throw createError({ statusCode: 404, statusMessage: 'Item not found' })
}

const { data, pending, error } = await useApiFetch<any>(`/api/item/${id}`)

if (error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'Item not found',
    })
}

// SSR-render a 5-row "Recent Destructions" preview so the dashboard tab
// carries unique per-item content (killmail titles, victim names, systems).
// Without this, thin SKIN/module items have ~700 bytes of shared nav chrome
// and Yandex flags DUPLICATE_CONTENT_ATTRS while Google parks them in
// "Crawled — currently not indexed".
const isShipForKilllist = computed(() => data.value?.item?.is_ship ?? false)
const { data: recentKillsData } = await useApiFetch<{ kills: any[] }>(
    () => `/api/${isShipForKilllist.value ? 'ship' : 'item'}/${id}/killlist?limit=5`,
    { key: `item-recent-${id}` },
)
const recentKills = computed(() => recentKillsData.value?.kills ?? [])

const item = computed(() => data.value?.item)
const isShip = computed(() => item.value?.is_ship ?? false)
const shipAttributes = computed(() => data.value?.shipAttributes ?? {})
const flatAttributes = computed(() => data.value?.attributes ?? [])
const requiredSkills = computed(() => data.value?.requiredSkills ?? [])
const materials = computed(() => data.value?.materials ?? [])
const marketBreadcrumb = computed(() => data.value?.marketBreadcrumb ?? [])
const variations = computed<Array<{ type_id: number, name: string | null, meta_group_id: number | null, meta_group_name: string | null }>>(() => data.value?.variations ?? [])
const pricing = computed(() => data.value?.pricing)
const priceSummary = computed(() => pricing.value?.summary)
const priceHistory = computed(() => pricing.value?.history ?? [])
const insurance = computed(() => pricing.value?.insurance ?? [])
const customSummary = computed(() => pricing.value?.customSummary)
const customHistory = computed(() => pricing.value?.customHistory ?? [])

const { convertEveHtml } = useEveHtmlParser()
const parsedDescription = computed(() => {
    if (!item.value?.description) return ''
    return convertEveHtml(item.value.description)
})

useHead({ title: computed(() => item.value?.name || 'Item') })

// Natural-language role phrase that appears in both the sr-only H2 and the
// leading clause of the meta description. Pulls meta_group (Tech II, Faction,
// etc.), group (Frigate, Railgun, Drone…), and category for the "what is this"
// signal searches hinge on.
const rolePhrase = computed(() => {
    const i = item.value
    if (!i?.group_name) return ''
    const meta = i.meta_group_name ? `${i.meta_group_name} ` : ''
    if (isShip.value) return `${meta}${i.group_name} — EVE Online ship`
    const cat = i.category_name && i.category_name !== i.group_name ? ` (${i.category_name})` : ''
    return `${meta}${i.group_name}${cat} — EVE Online item`
})

// Strip EVE HTML/markup from the CCP description for meta/OG use. Keeps the
// first sentence-ish (≈160 chars) so SERPs get a lore-flavored snippet.
const descriptionExcerpt = computed(() => {
    const raw = item.value?.description
    if (!raw) return ''
    const plain = String(raw).replace(/<[^>]+>/g, '').replace(/\s+/g, ' ').trim()
    if (plain.length <= 180) return plain
    return plain.slice(0, 177).replace(/\s+\S*$/, '') + '…'
})

useSeoMeta({
    description: computed(() => {
        const i = item.value
        if (!i?.name) return 'View item kill statistics on EVE-KILL.'
        const pieces: string[] = [`${i.name} — ${rolePhrase.value}`]
        if (priceSummary.value?.latest) pieces.push(`Jita ${formatIsk(priceSummary.value.latest)} ISK`)
        if (descriptionExcerpt.value) pieces.push(descriptionExcerpt.value)
        pieces.push(`Killmails, fittings, and market data on EVE-KILL.`)
        return pieces.join(' · ')
    }),
    ogTitle: computed(() => {
        const i = item.value
        if (!i) return 'Item — EVE-KILL'
        return `${i.name} (${i.group_name})`
    }),
    ogDescription: computed(() => {
        const i = item.value
        if (!i) return 'View item stats on EVE-KILL.'
        return `${i.name} — ${i.group_name} — killmails and stats in EVE Online.`
    }),
    ogImage: computed(() => {
        if (!item.value) return ''
        return isShip.value
            ? `/images/types/${item.value.type_id}/render?size=256`
            : `/images/types/${item.value.type_id}/icon?size=64`
    }),
    ogType: 'website',
    twitterCard: 'summary',
    twitterTitle: computed(() => {
        const i = item.value
        if (!i) return 'Item — EVE-KILL'
        return `${i.name} (${i.group_name}) — EVE-KILL`
    }),
    twitterDescription: computed(() => {
        const i = item.value
        if (!i) return 'View item stats on EVE-KILL.'
        return `${i.name} — ${i.group_name} — killmails and stats in EVE Online.`
    }),
    twitterImage: computed(() => {
        if (!item.value) return ''
        return isShip.value
            ? `/images/types/${item.value.type_id}/render?size=256`
            : `/images/types/${item.value.type_id}/icon?size=64`
    }),
})

useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: (() => {
            const i = item.value
            if (!i) return [{ name: 'Home', item: '/' }]
            return [
                { name: 'Home', item: '/' },
                { name: i.group_name ?? 'Group', item: i.group_id ? `/group/${i.group_id}` : '/market' },
                { name: i.name, item: `/item/${id}` },
            ]
        })(),
    }))),
    {
        '@type': 'Product',
        'name': computed(() => item.value?.name || 'Item'),
        'url': `https://eve-kill.com/item/${id}`,
        'image': computed(() => {
            if (!item.value) return undefined
            return isShip.value
                ? `/images/types/${item.value.type_id}/render?size=512`
                : `/images/types/${item.value.type_id}/icon?size=128`
        }),
        'category': computed(() => item.value?.group_name || undefined),
        'description': computed(() => {
            const desc = item.value?.description
            if (!desc) return undefined
            // Strip HTML/EVE markup tags for schema-friendly plain text
            return String(desc).replace(/<[^>]+>/g, '').replace(/\s+/g, ' ').trim().slice(0, 500)
        }),
        'offers': computed(() => {
            const latest = priceSummary.value?.latest
            if (!latest) return undefined
            return {
                '@type': 'Offer',
                'price': latest,
                'priceCurrency': 'ISK',
                'priceValidUntil': new Date(Date.now() + 86400000).toISOString().slice(0, 10),
                'availability': 'https://schema.org/InStock',
                'url': `https://eve-kill.com/item/${id}`,
            }
        }),
    },
])

type TabId = 'dashboard' | 'kills' | 'fittings'

// The fittings tab is only meaningful for ships — it's hidden entirely for
// modules, charges, structures, etc. The tabs list is computed because
// `isShip` only resolves after the initial /api/item/:id fetch completes.
const tabs = computed<ReadonlyArray<{ id: TabId; label: string; icon: string }>>(() => {
    const base: Array<{ id: TabId; label: string; icon: string }> = [
        { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
        { id: 'kills', label: 'Kills', icon: 'lucide:skull' },
    ]
    if (isShip.value) {
        base.push({ id: 'fittings', label: 'Fittings', icon: 'lucide:wrench' })
    }
    return base
})

const tabIds = computed(() => new Set(tabs.value.map(t => t.id)))

useDefaultTab('item', `/item/${id}`, 'dashboard', tabIds.value)

definePageMeta({
    key: route => `/item/${route.params.id}`,
})

const activeTab = computed<TabId>(() => {
    const param = route.params.tab as string
    if (param && tabIds.value.has(param as TabId)) return param as TabId
    return 'dashboard'
})

const activeTabLabel = computed(() => {
    if (activeTab.value === 'dashboard') return null
    return tabs.value.find(t => t.id === activeTab.value)?.label ?? null
})
useHead({ title: computed(() => {
    const name = item.value?.name
    if (!name) return 'Item'
    return activeTabLabel.value ? `${name} (${activeTabLabel.value})` : name
}) })

const setTab = (tabId: string) => {
    if (!tabIds.value.has(tabId as TabId)) return
    useAnalytics().track('tab.change', { entity: 'item', tab: tabId })
    navigateTo(tabId === 'dashboard' ? `/item/${id}` : `/item/${id}/${tabId}`)
}

const romanLevel = (level: number): string => {
    const map: Record<number, string> = { 1: 'I', 2: 'II', 3: 'III', 4: 'IV', 5: 'V' }
    return map[level] ?? String(level)
}

const metaGroupColor = (name: string | null): string => {
    if (!name) return 'text-gray-400'
    const n = name.toLowerCase()
    if (n.includes('tech ii') || n === 'tech ii') return 'text-yellow-400'
    if (n.includes('tech iii') || n === 'tech iii') return 'text-teal-400'
    if (n.includes('faction')) return 'text-green-400'
    if (n.includes('deadspace')) return 'text-blue-400'
    if (n.includes('officer')) return 'text-purple-400'
    return 'text-gray-400'
}

// Attribute names (for flat item attributes)
const ATTR_NAMES: Record<number, string> = {
    9: 'Structure HP', 263: 'Shield HP', 265: 'Armor HP',
    37: 'Max Velocity', 552: 'Signature Radius', 4: 'Mass',
    38: 'Capacity', 283: 'Drone Bandwidth', 284: 'Drone Capacity',
    482: 'Capacitor Capacity', 55: 'Capacitor Recharge',
    6: 'Activation Cost', 73: 'Cycle Time',
    30: 'Powergrid Usage', 50: 'CPU Usage', 1153: 'Calibration Cost',
    114: 'EM Damage', 118: 'Thermal Damage', 117: 'Kinetic Damage', 116: 'Explosive Damage',
    212: 'Damage Multiplier', 20: 'Velocity Modifier',
    554: 'Signature Radius Modifier', 68: 'Armor HP Bonus',
    128: 'Charge Size', 604: 'Charge Group',
    271: 'Shield EM Resist', 272: 'Shield Thermal Resist',
    273: 'Shield Kinetic Resist', 274: 'Shield Explosive Resist',
    267: 'Armor EM Resist', 268: 'Armor Thermal Resist',
    269: 'Armor Kinetic Resist', 270: 'Armor Explosive Resist',
    11: 'Powergrid Output', 48: 'CPU Output',
    12: 'High Slots', 13: 'Med Slots', 14: 'Low Slots',
    1137: 'Rig Slots', 1132: 'Calibration Output',
    1154: 'Launcher Hardpoints', 102: 'Turret Hardpoints',
    161: 'Warp Speed', 70: 'Inertia Modifier',
    564: 'Max Target Range', 192: 'Max Locked Targets',
    76: 'Radar Strength', 77: 'Ladar Strength',
    78: 'Magnetometric Strength', 79: 'Gravimetric Strength',
    479: 'Shield Recharge Time',
}

const resistIds = new Set([271, 272, 273, 274, 267, 268, 269, 270, 113, 110, 109, 111])

const formatAttrValue = (attrId: number, value: number): string => {
    if (resistIds.has(attrId)) return `${((1 - value) * 100).toFixed(1)}%`
    if ([73, 55, 479].includes(attrId)) return `${(value / 1000).toFixed(1)}s`
    if (attrId === 564) return value >= 1000 ? `${(value / 1000).toFixed(1)} km` : `${value.toFixed(0)} m`
    if (attrId === 6 || attrId === 482) return `${value.toFixed(1)} GJ`
    if (attrId === 30 || attrId === 11) return `${value.toFixed(0)} MW`
    if (attrId === 50 || attrId === 48) return `${value.toFixed(0)} tf`
    if (attrId === 37) return `${value.toFixed(0)} m/s`
    if (attrId === 552) return `${value.toFixed(0)} m`
    if (attrId === 70) return `${value.toFixed(3)}`
    if (attrId === 161) return `${value.toFixed(1)} AU/s`
    if (attrId === 600) return `${value.toFixed(0)} GJ`
    if (attrId === 4) return `${(value / 1000).toLocaleString('en-US', { maximumFractionDigits: 0 })} kg`
    if ([38, 284].includes(attrId)) return `${value.toLocaleString('en-US', { maximumFractionDigits: 0 })} m³`
    if (attrId === 283) return `${value.toFixed(0)} Mbit/s`
    if (attrId === 212) return `${value.toFixed(2)}x`
    if (attrId === 1153 || attrId === 1132) return `${value.toFixed(0)}`
    if ([20, 554, 68].includes(attrId)) {
        if (value <= 2 && value >= -2) return `${((value - 1) * 100).toFixed(1)}%`
        return `${value.toFixed(1)}%`
    }
    if (Number.isInteger(value)) return value.toLocaleString('en-US')
    return value.toFixed(2)
}

const resistColor = (attrId: number, value: number): string => {
    if (!resistIds.has(attrId)) return 'text-white'
    const pct = (1 - value) * 100
    if (pct >= 60) return 'text-green-400'
    if (pct >= 30) return 'text-yellow-400'
    return 'text-red-400'
}

const namedAttributes = computed(() => {
    return flatAttributes.value
        .filter((a: any) => ATTR_NAMES[a.id])
        .map((a: any) => ({ ...a, name: ATTR_NAMES[a.id] }))
})

// Ship attribute group config
const groupConfig: Record<string, { icon: string, color: string, label: string }> = {
    defense: { icon: 'lucide:shield', color: 'text-blue-500', label: 'Defense' },
    fitting: { icon: 'lucide:wrench', color: 'text-yellow-500', label: 'Fitting' },
    navigation: { icon: 'lucide:navigation', color: 'text-cyan-500', label: 'Navigation' },
    capacitor: { icon: 'lucide:zap', color: 'text-purple-500', label: 'Capacitor' },
    targeting: { icon: 'lucide:crosshair', color: 'text-red-500', label: 'Targeting' },
    drones: { icon: 'lucide:bug', color: 'text-green-500', label: 'Drones' },
}

// Ship header helpers
const getShipAttr = (attrId: number): number | null => {
    for (const group of Object.values(shipAttributes.value)) {
        const found = (group as any[]).find((a: any) => a.id === attrId)
        if (found) return found.value
    }
    return null
}
const ehp = computed(() => (getShipAttr(263) ?? 0) + (getShipAttr(265) ?? 0) + (getShipAttr(9) ?? 0))

// Breadcrumb: drop the leaf market group (usually race/faction), append item name
// The leaf market group (e.g. "Caldari", "Amarr") — stripped from breadcrumb, shown as a badge
const marketLeaf = computed(() => {
    const crumbs = marketBreadcrumb.value
    return crumbs.length >= 3 ? crumbs[crumbs.length - 1]?.name ?? null : null
})

const displayBreadcrumb = computed(() => {
    const raw = [...marketBreadcrumb.value] as { name: string, slug: string }[]
    // Keep the broad market taxonomy, then hand off to the gameplay group.
    // Narrow market categories remain available through the item's market badge.
    if (item.value?.group_id && raw.length > 2) raw.splice(2)
    else if (raw.length >= 3) raw.pop()
    // Build crumbs with cumulative paths
    const result: { name: string, path: string | null }[] = [{ name: 'Market', path: '/market' }]
    let slugPath = ''
    for (const c of raw) {
        slugPath += (slugPath ? '/' : '') + c.slug
        result.push({ name: c.name, path: `/market/${slugPath}` })
    }
    if (item.value?.group_id && item.value.group_name) {
        result.push({ name: item.value.group_name, path: `/group/${item.value.group_id}` })
    }
    result.push({ name: item.value?.name ?? '', path: null }) // current item, no link
    return result
})

const hasStats = computed(() =>
    customSummary.value?.latest || priceSummary.value?.latest || priceSummary.value?.average_90d
    || (isShip.value && ehp.value > 0) || (isShip.value && getShipAttr(37))
    || (isShip.value && getShipAttr(283) != null)
    || (!isShip.value && item.value?.volume) || item.value?.base_price
)
</script>

<template>
    <div>
        <EntityHeader v-if="pending" loading />

        <div v-else-if="item">
            <EntityHeader>
                <template #image>
                    <!-- Ship: large render. Item: small icon -->
                    <EntityImageExpand v-if="isShip" :full-src="`/images/types/${item.type_id}/render?size=512`" :alt="item.name">
                        <img
                            :src="`/images/types/${item.type_id}/render?size=256`"
                            :alt="item.name" class="w-32 h-32 md:w-40 md:h-40 rounded-lg shadow-lg" loading="eager">
                    </EntityImageExpand>
                    <EntityImageExpand v-else :full-src="`/images/types/${item.type_id}/icon?size=64`" :alt="item.name">
                        <img
                            :src="`/images/types/${item.type_id}/icon?size=64`"
                            :alt="item.name" class="w-16 h-16 rounded-lg shadow-lg" loading="eager">
                    </EntityImageExpand>
                </template>

                <div class="flex items-center gap-3 mb-1">
                    <h1 class="text-2xl md:text-3xl font-bold text-white">{{ item.name }}</h1>
                    <span v-if="item.meta_group_name" class="px-2 py-0.5 rounded text-fine font-bold uppercase tracking-wider"
                        :class="metaGroupColor(item.meta_group_name)"
                        style="background: rgba(255,255,255,0.06)">
                        {{ item.meta_group_name }}
                    </span>
                    <span v-if="marketLeaf" class="px-2 py-0.5 rounded text-fine font-bold uppercase tracking-wider text-gray-400"
                        style="background: rgba(255,255,255,0.06)">
                        {{ marketLeaf }}
                    </span>
                </div>
                <h2 v-if="rolePhrase" class="sr-only">{{ item.name }} — {{ rolePhrase }}</h2>

                <div class="flex items-center gap-1.5 text-xs mb-4 flex-wrap">
                    <template v-for="(crumb, idx) in displayBreadcrumb" :key="idx">
                        <span v-if="idx > 0" class="text-gray-700">/</span>
                        <NuxtLink v-if="crumb.path" :to="crumb.path" class="text-gray-500 hover:text-blue-400 transition-colors">{{ crumb.name }}</NuxtLink>
                        <span v-else class="text-gray-400">{{ crumb.name }}</span>
                    </template>
                </div>

                <div v-if="parsedDescription" class="item-description text-fine text-gray-400 leading-relaxed max-w-2xl" v-html="parsedDescription"></div>

                <template v-if="hasStats" #stats>
                    <EntityStatGrid variant="inline" :columns="6">
                        <div v-if="customSummary?.latest">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Est. Value</div>
                            <div class="text-base font-bold text-orange-400 tabular-nums">{{ formatIsk(customSummary.latest) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div v-if="priceSummary?.latest">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Jita Price</div>
                            <div class="text-base font-bold text-yellow-400 tabular-nums">{{ formatIsk(priceSummary.latest) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div v-if="priceSummary?.average_90d">
                            <div class="text-fine uppercase tracking-wider text-gray-500">90d Avg</div>
                            <div class="text-base font-bold text-white tabular-nums">{{ formatIsk(priceSummary.average_90d) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <!-- Ship-specific stats -->
                        <div v-if="isShip && ehp > 0">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Raw HP</div>
                            <div class="text-base font-bold text-blue-400 tabular-nums">{{ ehp.toLocaleString('en-US') }}</div>
                        </div>
                        <div v-if="isShip && getShipAttr(37)">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Max Speed</div>
                            <div class="text-base font-bold text-cyan-400 tabular-nums">{{ getShipAttr(37)!.toFixed(0) }} <span class="text-xs font-normal text-gray-600">m/s</span></div>
                        </div>
                        <div v-if="isShip && getShipAttr(283) != null">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Drone BW</div>
                            <div class="text-base font-bold text-green-400 tabular-nums">{{ getShipAttr(283)!.toFixed(0) }} <span class="text-xs font-normal text-gray-600">Mbit/s</span></div>
                        </div>
                        <!-- Non-ship stats -->
                        <div v-if="!isShip && item.volume">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Volume</div>
                            <div class="text-base font-bold text-gray-300 tabular-nums">{{ item.volume.toLocaleString('en-US') }} <span class="text-xs font-normal text-gray-600">m³</span></div>
                        </div>
                        <div v-if="item.base_price">
                            <div class="text-fine uppercase tracking-wider text-gray-500">Base Price</div>
                            <div class="text-base font-bold text-gray-300 tabular-nums">{{ formatIsk(item.base_price) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                    </EntityStatGrid>
                </template>
            </EntityHeader>

            <!-- TAB BAR -->
            <EntityTabBar :tabs="tabs" :active-id="activeTab" @select="setTab" />

            <!-- DASHBOARD TAB -->
            <div v-if="activeTab === 'dashboard'">
                <!-- Jita Market summary bar -->
                <div v-if="priceSummary" class="glass-panel p-4 mb-4">
                    <div class="flex items-center gap-2 mb-3">
                        <Icon name="lucide:trending-up" class="w-4 h-4 text-emerald-500" />
                        <span class="text-xs font-semibold text-gray-300">Jita Market (90d)</span>
                    </div>
                    <div class="grid grid-cols-2 md:grid-cols-5 gap-4">
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">Latest Price <span v-if="priceSummary.latest_date" class="text-gray-600">({{ priceSummary.latest_date }})</span></div>
                            <div class="text-base font-bold text-white tabular-nums">{{ formatIsk(priceSummary.latest) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">90d Average</div>
                            <div class="text-base font-bold text-yellow-400 tabular-nums">{{ formatIsk(priceSummary.average_90d) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">90d High</div>
                            <div class="text-base font-bold text-green-400 tabular-nums">{{ formatIsk(priceSummary.highest_90d) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">90d Low</div>
                            <div class="text-base font-bold text-red-400 tabular-nums">{{ formatIsk(priceSummary.lowest_90d) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">Avg Daily Volume</div>
                            <div class="text-base font-bold text-white tabular-nums">{{ priceSummary.avg_volume_90d.toLocaleString('en-US') }}</div>
                        </div>
                    </div>
                </div>

                <!-- Custom estimated value bar -->
                <div v-if="customSummary" class="rounded-lg border border-orange-500/20 bg-orange-500/[0.04] p-4 mb-4">
                    <div class="flex items-center gap-2 mb-3">
                        <Icon name="lucide:diamond" class="w-4 h-4 text-orange-500" />
                        <span class="text-xs font-semibold text-gray-300">Estimated Value (90d)</span>
                    </div>
                    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">Latest <span v-if="customSummary.latest_date" class="text-gray-600">({{ customSummary.latest_date }})</span></div>
                            <div class="text-base font-bold text-orange-400 tabular-nums">{{ formatIsk(customSummary.latest) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">90d Average</div>
                            <div class="text-base font-bold text-white tabular-nums">{{ formatIsk(customSummary.average_90d) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">90d High</div>
                            <div class="text-base font-bold text-green-400 tabular-nums">{{ formatIsk(customSummary.highest_90d) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                        <div>
                            <div class="text-xs text-gray-500 mb-0.5">90d Low</div>
                            <div class="text-base font-bold text-red-400 tabular-nums">{{ formatIsk(customSummary.lowest_90d) }} <span class="text-xs font-normal text-gray-600">ISK</span></div>
                        </div>
                    </div>
                </div>

                <!-- Variations (T1 root + all meta/faction/T2 variants) -->
                <div v-if="variations.length > 1" class="glass-panel p-3.5 mb-4">
                    <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                        <Icon name="lucide:layers" class="w-4 h-4 text-indigo-400" />
                        <span class="text-xs font-semibold text-gray-300">Variations</span>
                        <span class="text-fine text-gray-600">({{ variations.length }})</span>
                    </div>
                    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-2">
                        <template v-for="v in variations" :key="v.type_id">
                            <NuxtLink v-if="v.type_id !== id" :to="`/item/${v.type_id}`"
                                class="flex items-center gap-2 p-2 rounded-md hover:bg-blue-500/[0.08] transition-colors border border-transparent hover:border-blue-500/30">
                                <img :src="`/images/types/${v.type_id}/icon?size=32`"
                                    :alt="v.name ?? ''" class="w-8 h-8 rounded shrink-0" loading="lazy">
                                <div class="min-w-0 flex-1">
                                    <div class="text-xs text-gray-200 truncate">{{ v.name ?? `Type #${v.type_id}` }}</div>
                                    <div v-if="v.meta_group_name" class="text-fine uppercase tracking-wider truncate"
                                        :class="metaGroupColor(v.meta_group_name)">
                                        {{ v.meta_group_name }}
                                    </div>
                                </div>
                            </NuxtLink>
                            <div v-else
                                class="flex items-center gap-2 p-2 rounded-md bg-blue-500/[0.12] border border-blue-500/40">
                                <img :src="`/images/types/${v.type_id}/icon?size=32`"
                                    :alt="v.name ?? ''" class="w-8 h-8 rounded shrink-0" loading="lazy">
                                <div class="min-w-0 flex-1">
                                    <div class="text-xs text-white font-semibold truncate">{{ v.name ?? `Type #${v.type_id}` }}</div>
                                    <div v-if="v.meta_group_name" class="text-fine uppercase tracking-wider truncate"
                                        :class="metaGroupColor(v.meta_group_name)">
                                        {{ v.meta_group_name }}
                                    </div>
                                </div>
                            </div>
                        </template>
                    </div>
                </div>

                <!--
                    Main content: 2-column.

                    Recent Destructions sits inside the left column rather than
                    full-width above the grid. Full-width, it pushed the whole
                    right-hand rail down by its own height, so the price history
                    started level with Required Skills and ran on past the
                    bottom of the spec panels — a tall, narrow column of dates
                    alongside nothing. Starting the grid one row earlier lets
                    the prices climb up beside the kill list and the specs use
                    the width the rail was wasting.
                -->
                <div class="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4">
                    <!-- LEFT column -->
                    <div class="space-y-4">
                        <!-- Recent Destructions (SSR-rendered per-item content for SEO) -->
                        <div v-if="recentKills.length" class="glass-panel p-3.5">
                            <div class="flex items-center justify-between mb-3 pb-2 border-b border-white/[0.04]">
                                <div class="flex items-center gap-2">
                                    <Icon name="lucide:skull" class="w-4 h-4 text-red-400" />
                                    <span class="text-xs font-semibold text-gray-300">Recent Destructions</span>
                                </div>
                                <NuxtLink :to="`/item/${id}/kills`" class="text-fine text-gray-500 hover:text-blue-400 transition-colors">View all →</NuxtLink>
                            </div>
                            <div class="space-y-1">
                                <NuxtLink v-for="k in recentKills" :key="k.killmail_id" :to="`/kill/${k.killmail_id}`"
                                    class="flex items-center gap-3 text-sm hover:bg-blue-500/[0.04] rounded px-1.5 py-1 transition-colors">
                                    <img :src="`/images/types/${k.ship_type_id}/render?size=32`"
                                        :alt="k.ship_name ?? ''" class="w-7 h-7 rounded shrink-0" loading="lazy">
                                    <div class="min-w-0 flex-1 flex items-baseline gap-2">
                                        <span class="text-gray-200 truncate">{{ k.ship_name ?? 'Ship' }}</span>
                                        <span class="text-fine text-gray-500 truncate">· {{ k.victim_character_name ?? 'Unknown Pilot' }} · {{ k.solar_system_name ?? 'Unknown' }}</span>
                                    </div>
                                    <span class="text-xs text-orange-400 tabular-nums shrink-0 font-medium">{{ formatIsk(k.total_value ?? 0) }}</span>
                                    <span class="text-fine text-gray-600 tabular-nums shrink-0 hidden sm:inline">{{ new Date(k.killmail_time).toISOString().slice(0, 10) }}</span>
                                </NuxtLink>
                            </div>
                        </div>

                        <!-- Required Skills -->
                        <div v-if="requiredSkills.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:graduation-cap" class="w-4 h-4 text-yellow-500" />
                                <span class="text-xs font-semibold text-gray-300">Required Skills</span>
                            </div>
                            <div class="space-y-2">
                                <div v-for="skill in requiredSkills" :key="skill.type_id" class="flex items-center justify-between">
                                    <NuxtLink :to="`/item/${skill.type_id}`" class="text-sm text-gray-300 hover:text-blue-400 transition-colors">
                                        {{ skill.name ?? `Skill #${skill.type_id}` }}
                                    </NuxtLink>
                                    <span class="text-sm font-bold text-yellow-400">{{ romanLevel(skill.level) }}</span>
                                </div>
                            </div>
                        </div>

                        <!-- SHIP: Grouped attribute cards -->
                        <div v-if="isShip" class="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <template v-for="(group, key) in shipAttributes" :key="key">
                                <div v-if="groupConfig[key]" class="glass-panel p-3.5">
                                    <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                        <Icon :name="groupConfig[key].icon" class="w-4 h-4" :class="groupConfig[key].color" />
                                        <span class="text-xs font-semibold text-gray-300">{{ groupConfig[key].label }}</span>
                                    </div>
                                    <div class="space-y-2">
                                        <div v-for="attr in group" :key="attr.id" class="flex justify-between">
                                            <span class="text-xs text-gray-500">{{ attr.name }}</span>
                                            <span class="text-fine tabular-nums" :class="resistColor(attr.id, attr.value)">
                                                {{ formatAttrValue(attr.id, attr.value) }}
                                            </span>
                                        </div>
                                    </div>
                                </div>
                            </template>
                        </div>

                        <!-- ITEM: Flat attributes -->
                        <div v-if="!isShip && namedAttributes.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:sliders-horizontal" class="w-4 h-4 text-blue-500" />
                                <span class="text-xs font-semibold text-gray-300">Attributes</span>
                            </div>
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-2">
                                <div v-for="attr in namedAttributes" :key="attr.id" class="flex justify-between">
                                    <span class="text-xs text-gray-500">{{ attr.name }}</span>
                                    <span class="text-fine text-white tabular-nums">{{ formatAttrValue(attr.id, attr.value) }}</span>
                                </div>
                            </div>
                        </div>

                        <!-- Reprocessing Materials -->
                        <div v-if="materials.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:recycle" class="w-4 h-4 text-teal-500" />
                                <span class="text-xs font-semibold text-gray-300">Reprocessing Materials</span>
                            </div>
                            <div class="space-y-1.5">
                                <NuxtLink v-for="mat in materials" :key="mat.type_id"
                                    :to="`/item/${mat.type_id}`"
                                    class="flex items-center justify-between hover:bg-blue-500/[0.04] rounded px-1 py-0.5 transition-colors">
                                    <div class="flex items-center gap-2">
                                        <img :src="`/images/types/${mat.type_id}/icon?size=32`" class="w-5 h-5 rounded" loading="lazy">
                                        <span class="text-sm text-gray-300">{{ mat.name ?? `Type #${mat.type_id}` }}</span>
                                    </div>
                                    <span class="text-sm text-white tabular-nums font-medium">{{ mat.quantity.toLocaleString('en-US') }}</span>
                                </NuxtLink>
                            </div>
                        </div>

                        <!-- Properties -->
                        <div class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:box" class="w-4 h-4 text-gray-400" />
                                <span class="text-xs font-semibold text-gray-300">Properties</span>
                            </div>
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-2">
                                <div class="flex justify-between"><span class="text-xs text-gray-500">Type ID</span><span class="text-sm text-white tabular-nums">{{ item.type_id }}</span></div>
                                <div v-if="item.group_id" class="flex justify-between"><span class="text-xs text-gray-500">Group</span><NuxtLink :to="`/group/${item.group_id}`" class="text-sm text-blue-400 hover:text-blue-300 transition-colors">{{ item.group_name }} ({{ item.group_id }})</NuxtLink></div>
                                <div v-if="item.category_id" class="flex justify-between"><span class="text-xs text-gray-500">Category ID</span><span class="text-sm text-white tabular-nums">{{ item.category_id }}</span></div>
                                <div v-if="item.mass" class="flex justify-between"><span class="text-xs text-gray-500">Mass</span><span class="text-sm text-white tabular-nums">{{ item.mass.toLocaleString('en-US') }} kg</span></div>
                                <div v-if="item.volume" class="flex justify-between"><span class="text-xs text-gray-500">Volume</span><span class="text-sm text-white tabular-nums">{{ item.volume.toLocaleString('en-US') }} m³</span></div>
                                <div v-if="item.packaged_volume" class="flex justify-between"><span class="text-xs text-gray-500">Packaged Volume</span><span class="text-sm text-white tabular-nums">{{ item.packaged_volume.toLocaleString('en-US') }} m³</span></div>
                                <div v-if="item.capacity" class="flex justify-between"><span class="text-xs text-gray-500">Capacity</span><span class="text-sm text-white tabular-nums">{{ item.capacity.toLocaleString('en-US') }} m³</span></div>
                                <div v-if="item.portion_size" class="flex justify-between"><span class="text-xs text-gray-500">Portion Size</span><span class="text-sm text-white tabular-nums">{{ item.portion_size }}</span></div>
                                <div v-if="item.radius" class="flex justify-between"><span class="text-xs text-gray-500">Radius</span><span class="text-sm text-white tabular-nums">{{ item.radius.toLocaleString('en-US') }} m</span></div>
                                <div v-if="item.meta_level" class="flex justify-between"><span class="text-xs text-gray-500">Meta Level</span><span class="text-sm text-white tabular-nums">{{ item.meta_level }}</span></div>
                                <div v-if="item.tech_level" class="flex justify-between"><span class="text-xs text-gray-500">Tech Level</span><span class="text-sm text-white tabular-nums">{{ item.tech_level }}</span></div>
                                <div class="flex justify-between"><span class="text-xs text-gray-500">Published</span><span class="text-sm" :class="item.published ? 'text-green-400' : 'text-red-400'">{{ item.published ? 'Yes' : 'No' }}</span></div>
                            </div>
                        </div>
                    </div>

                    <!--
                        RIGHT column: prices + insurance.

                        Sticky, capped to the viewport and scrollable. A ship's
                        spec panels run far deeper than the rail — roughly
                        2,850px against 1,800px on a Loki — so the rail used to
                        run out and leave a thousand pixels of dead space beside
                        the reprocessing and properties tables. The cap matters:
                        thirty rows of price history are taller than a short
                        viewport, and without it the bottom rows would pin
                        permanently out of reach.
                    -->
                    <div class="space-y-4 lg:sticky lg:top-4 lg:max-h-[calc(100vh-2rem)] lg:overflow-y-auto lg:self-start">
                        <!-- Custom Price History -->
                        <div v-if="customHistory.length" class="rounded-lg border border-orange-500/20 bg-orange-500/[0.04] p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-orange-500/10">
                                <Icon name="lucide:diamond" class="w-4 h-4 text-orange-500" />
                                <span class="text-xs font-semibold text-gray-300">Estimated Value History</span>
                            </div>
                            <div class="grid grid-cols-[1fr_auto] gap-x-3 text-fine uppercase tracking-wider text-gray-600 mb-2">
                                <span>Date</span><span class="text-right">Price</span>
                            </div>
                            <div class="space-y-1.5">
                                <div v-for="p in customHistory.slice(0, 30)" :key="p.date" class="grid grid-cols-[1fr_auto] gap-x-3 text-xs items-center">
                                    <span class="text-gray-500 tabular-nums">{{ p.date }}</span>
                                    <span class="text-orange-400 tabular-nums font-medium text-right">{{ formatIsk(p.price) }}</span>
                                </div>
                            </div>
                        </div>

                        <!-- Recent Prices (Jita) -->
                        <div v-if="priceHistory.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:calendar" class="w-4 h-4 text-orange-500" />
                                <span class="text-xs font-semibold text-gray-300">Recent Prices (Jita)</span>
                            </div>
                            <div class="grid grid-cols-[1fr_auto_auto] gap-x-3 text-fine uppercase tracking-wider text-gray-600 mb-2">
                                <span>Date</span><span class="text-right">Price</span><span class="text-right w-14">Volume</span>
                            </div>
                            <div class="space-y-1.5">
                                <div v-for="p in priceHistory.slice(0, 30)" :key="p.date" class="grid grid-cols-[1fr_auto_auto] gap-x-3 text-xs items-center">
                                    <span class="text-gray-500 tabular-nums">{{ p.date }}</span>
                                    <span class="text-yellow-400 tabular-nums font-medium text-right">{{ formatIsk(p.average ?? 0) }}</span>
                                    <span class="text-gray-600 tabular-nums text-right w-14">{{ formatCompact(p.volume ?? 0) }}</span>
                                </div>
                            </div>
                        </div>

                        <!-- Insurance (ships only) -->
                        <div v-if="insurance.length" class="glass-panel p-3.5">
                            <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
                                <Icon name="lucide:shield-check" class="w-4 h-4 text-teal-500" />
                                <span class="text-xs font-semibold text-gray-300">Insurance</span>
                            </div>
                            <div class="space-y-2">
                                <div v-for="ins in insurance" :key="ins.level_name" class="flex justify-between items-center">
                                    <span class="text-xs text-gray-500">{{ ins.level_name }}</span>
                                    <div class="text-right">
                                        <span class="text-fine text-green-400 tabular-nums">{{ formatIsk(ins.payout) }}</span>
                                        <span class="text-xs text-gray-600 ml-1">/ {{ formatIsk(ins.cost) }}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Empty state -->
                <div v-if="!isShip && namedAttributes.length === 0 && !priceSummary && !customSummary && !requiredSkills.length && !materials.length" class="text-center py-12 text-gray-500 text-sm">
                    No data available for this item.
                </div>
            </div>

            <!-- KILLS TAB -->
            <div v-if="activeTab === 'kills'">
                <KillList :entity-endpoint="isShip ? `/api/ship/${id}/killlist` : `/api/item/${id}/killlist`" />
            </div>

            <!-- FITTINGS TAB (ships only — gated by the computed tabs list) -->
            <div v-if="activeTab === 'fittings' && isShip">
                <ItemFittings :ship-type-id="id" />
            </div>
        </div>
    </div>
</template>

<style>
.item-description a {
    color: rgb(96 165 250 / 0.8);
    text-decoration: underline;
    text-underline-offset: 2px;
}
.item-description a:hover {
    color: white;
}
</style>
