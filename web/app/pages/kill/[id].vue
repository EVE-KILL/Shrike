<script setup lang="ts">
interface KillmailResponse {
    killmail_id: number
    killmail_hash: string
    killmail_time: string
    victim: {
        character_id: number | null
        character_name: string | null
        corporation_id: number | null
        corporation_name: string | null
        alliance_id: number | null
        alliance_name: string | null
        ship_type_id: number | null
        ship_name: string | null
        ship_group_id: number | null
        ship_group_name: string | null
        damage_taken: number
        ship_price: number
        ship_market_path?: string | null
    }
    solar_system_id: number
    solar_system_name: string | null
    solar_system_security: number | null
    constellation_id: number | null
    constellation_name: string | null
    region_id: number | null
    region_name: string | null
    position_x: number | null
    position_y: number | null
    position_z: number | null
    location: {
        item_id: number
        item_name: string | null
        type_id: number | null
        group_id: number | null
        distance: number
    } | null
    total_value: number
    fitted_value: number
    dropped_value: number
    destroyed_value: number
    points: number
    attacker_count: number
    is_npc: boolean
    is_solo: boolean
    total_damage: number
    attackers: {
        character_id: number | null
        character_name: string | null
        corporation_id: number | null
        corporation_name: string | null
        alliance_id: number | null
        alliance_name: string | null
        ship_type_id: number | null
        ship_name: string | null
        weapon_type_id: number | null
        weapon_name: string | null
        damage_done: number
        final_blow: boolean
        security_status: number | null
    }[]
    items: {
        item_index: number
        type_id: number
        type_name: string | null
        group_id: number | null
        category_id: number | null
        flag_id: number
        flag_name: string | null
        quantity_dropped: number
        quantity_destroyed: number
        singleton: boolean
        parent_index: number | null
        is_container: boolean
        slot: string
        price: number
        total_value: number
    }[]
}

interface Sibling {
    killmail_id: number
    ship_type_id: number | null
    ship_group_id: number | null
    ship_name: string | null
    total_value: number
    killmail_time: string
}

const route = useRoute()
const id = computed(() => Number(route.params.id))

// SSR renders both the desktop and mobile trees (CSS hides one); after
// hydration the inactive one unmounts — see useIsDesktop.
const isDesktop = useIsDesktop()

// Legacy killmail IDs are < 7,500,000 — redirect to the legacy kill page
if (id.value > 0 && id.value < 7_500_000) {
    await navigateTo(`/legacy/kill/${id.value}`, { replace: true, redirectCode: 301 })
}

const { data: kill, pending, error } = await useApiFetch<KillmailResponse>(`/api/killmail/${id.value}`)
const { data: siblingsData } = useApiFetch<{ siblings: Sibling[] }>(`/api/killmail/${id.value}/siblings`)
const siblings = computed<Sibling[]>(() => siblingsData.value?.siblings ?? [])

// Victim corp identity accent — info box ring + Discord link embed color
const victimAccent = computed(() => entityAccent((kill.value?.victim as any)?.corporation_palette))
useSeoMeta({ themeColor: computed(() => victimAccent.value?.accent) })

if (error.value) {
    throw createError({ statusCode: 404, statusMessage: 'Killmail not found' })
}

// Items fitted to the victim ship only (exclude items inside hangar ships)
const fittedItems = computed(() =>
    kill.value?.items?.filter((i: any) => i.slot !== 'container_item') ?? [],
)

const killTitle = computed(() => {
    if (!kill.value) return 'Killmail'
    const v = kill.value.victim
    const who = v.character_name || v.corporation_name || 'Unknown'
    const ship = v.ship_name || 'Unknown'
    const isk = formatIsk(kill.value.total_value || 0)
    return `${ship} | ${who} | ${isk} ISK`
})

const killDescription = computed(() => {
    if (!kill.value) return 'View killmail details on EVE-KILL.'
    const k = kill.value
    const v = k.victim
    const who = v.character_name || v.corporation_name || 'Unknown'
    const ship = v.ship_name || 'ship'
    const isk = formatIsk(k.total_value || 0)
    const system = k.solar_system_name || 'Unknown'
    const region = k.region_name || ''
    const sec = k.solar_system_security != null ? k.solar_system_security.toFixed(1) : null
    const attackerCount = k.attackers?.length || 0
    const fb = k.attackers?.find(a => a.final_blow)
    const fbName = fb?.character_name || fb?.corporation_name || null

    // "Victim (Corp / Alliance)" — only show corp when we actually have a
    // character; for structure/drone kills where the "victim" is already the
    // corp the repeat looks weird.
    const affiliation: string[] = []
    if (v.character_name && v.corporation_name) affiliation.push(v.corporation_name)
    if (v.alliance_name) affiliation.push(v.alliance_name)
    const affSuffix = affiliation.length ? ` (${affiliation.join(' / ')})` : ''

    const article = /^[aeiou]/i.test(ship) ? 'an' : 'a'

    const locBits: string[] = []
    if (sec) locBits.push(sec)
    if (region) locBits.push(region)
    const locSuffix = locBits.length ? ` (${locBits.join(', ')})` : ''

    const flags: string[] = []
    if (k.is_solo) flags.push('Solo')
    if (k.is_npc) flags.push('NPC')
    const flagSuffix = flags.length ? ` (${flags.join(', ')})` : ''

    let desc = `${who}${affSuffix} lost ${article} ${ship} in ${system}${locSuffix} worth ${isk} ISK.`
    desc += ` ${attackerCount} attacker${attackerCount !== 1 ? 's' : ''}`
    if (fbName) desc += `, final blow by ${fbName}${flagSuffix}`
    else if (flagSuffix) desc += flagSuffix
    desc += '.'
    return desc
})

// Social preview variant chooser — `?preview=<default|big|fancy|none>`:
//
//   default (or unset)  64×64 ship render, og:type=website,
//                       twitter:card=summary — small right-aligned thumbnail
//                       matching zKillboard's Discord embed style.
//   big                 512×512 ship render, og:type=article,
//                       twitter:card=summary_large_image — big ship card.
//   fancy               compound 550×200 social card from imageserver,
//                       og:type=article, twitter:card=summary_large_image —
//                       full card with victim + final blow + ISK.
//   none                no OG image / twitter:card emitted at all.
type PreviewMode = 'default' | 'big' | 'fancy' | 'none'
const previewMode = computed<PreviewMode>(() => {
    const p = route.query.preview
    if (p === 'big') return 'big'
    if (p === 'fancy') return 'fancy'
    if (p === 'none') return 'none'
    return 'default'
})

const killOgImage = computed(() => {
    if (previewMode.value === 'none') return undefined
    if (previewMode.value === 'fancy') {
        return `/images/killmail/${id.value}/social.png`
    }
    const shipId = kill.value?.victim?.ship_type_id
    if (!shipId) return undefined
    const size = previewMode.value === 'big' ? 512 : 64
    return `/images/types/${shipId}/overlayrender?size=${size}`
})

const killOgImageMeta = computed(() => {
    const url = killOgImage.value
    if (!url) return undefined
    if (previewMode.value === 'fancy') {
        return { url, width: 550, height: 200, type: 'image/png' as const }
    }
    const size = previewMode.value === 'big' ? 512 : 64
    return { url, width: size, height: size, type: 'image/png' as const }
})

useSeoMeta({
    description: killDescription,
    ogTitle: computed(() => {
        if (!kill.value) return 'Killmail — EVE-KILL'
        const v = kill.value.victim
        const who = v.character_name || v.corporation_name || 'Unknown'
        const ship = v.ship_name || 'Unknown'
        const isk = formatIsk(kill.value.total_value || 0)
        const system = kill.value.solar_system_name || 'Unknown'
        return `${ship} | ${who} | ${isk} ISK | ${system}`
    }),
    ogDescription: killDescription,
    ogImage: killOgImageMeta,
    // `website` drives the compact right-aligned thumbnail layout (default,
    // zKB-style). `article` switches to the big card layout used by `big`
    // and `fancy`.
    ogType: computed(() => previewMode.value === 'default' ? 'website' : 'article'),
    twitterCard: computed(() => {
        if (previewMode.value === 'none') return undefined
        if (previewMode.value === 'default') return 'summary'
        return 'summary_large_image'
    }),
    twitterTitle: computed(() => {
        if (!kill.value) return 'Killmail — EVE-KILL'
        const v = kill.value.victim
        const who = v.character_name || v.corporation_name || 'Unknown'
        const ship = v.ship_name || 'Unknown'
        const isk = formatIsk(kill.value.total_value || 0)
        return `${ship} | ${who} | ${isk} ISK — EVE-KILL`
    }),
    twitterDescription: killDescription,
    twitterImage: killOgImage,
})

useSchemaOrg([
    defineBreadcrumb({
        itemListElement: computed(() => {
            const crumbs: { name: string; item: string }[] = [
                { name: 'Home', item: '/' },
                { name: 'Kills', item: '/kills/latest' },
            ]
            crumbs.push({ name: killTitle.value, item: `/kill/${id.value}` })
            return crumbs
        }),
    }),
    defineArticle({
        headline: computed(() => killTitle.value),
        description: computed(() => killDescription.value),
        image: computed(() => killOgImage.value ?? ''),
        datePublished: computed(() => kill.value?.killmail_time || ''),
        dateModified: computed(() => kill.value?.killmail_time || ''),
        author: { '@type': 'Organization', name: 'EVE-KILL', url: 'https://eve-kill.com' },
    }),
])

/**
 * Merge items by type_id within a slot, keeping the dropped and destroyed
 * quantities on the same row.
 *
 * These used to be split into two rows per stack, which produced adjacent,
 * near-identical lines — "Caldari Navy Mjolnir Light Missile 150" twice, one
 * red and one green — where the only distinguishing mark was the tint. It also
 * broke the mobile table outright: that one renders `quantity_dropped` and
 * `quantity_destroyed` from the underlying record, so each half of a split pair
 * showed *both* badges and the stack appeared twice with identical numbers.
 */
const itemsBySlot = computed(() => {
    if (!kill.value?.items) return {}

    // First pass: merge by type_id per slot
    const merged: Record<string, Map<number, { type_id: number; type_name: string | null; group_id: number | null; flag_id: number; price: number; is_container: boolean; item_index: number; quantity_dropped: number; quantity_destroyed: number }>> = {}
    for (const item of kill.value.items) {
        if (item.slot === 'container_item') continue
        const slot = item.slot || 'other'
        if (!merged[slot]) merged[slot] = new Map()

        if (item.is_container) {
            // Containers stay individual
            merged[slot].set(-item.item_index, item)
        } else {
            const existing = merged[slot].get(item.type_id)
            if (existing) {
                existing.quantity_dropped += item.quantity_dropped
                existing.quantity_destroyed += item.quantity_destroyed
            } else {
                merged[slot].set(item.type_id, { ...item })
            }
        }
    }

    // Second pass: one row per stack, carrying both quantities
    const groups: Record<string, any[]> = {}
    for (const [slot, items] of Object.entries(merged)) {
        groups[slot] = []
        for (const item of items.values()) {
            if (item.is_container) {
                groups[slot].push({ ...item, _status: 'container', total_value: 0 })
                continue
            }
            groups[slot].push({
                ...item,
                _status: item.quantity_dropped > 0 && item.quantity_destroyed > 0
                    ? ('both' as const)
                    : item.quantity_dropped > 0 ? ('dropped' as const) : ('destroyed' as const),
                total_value: item.price * (item.quantity_dropped + item.quantity_destroyed),
            })
        }
    }
    return groups
})

// Build a map of parent_index → child items for containers (merged + split)
const childrenByParent = computed(() => {
    if (!kill.value?.items) return new Map()
    // First merge by type_id per parent
    const mergedMap = new Map<number, Map<number, any>>()
    for (const item of kill.value.items) {
        if (item.slot === 'container_item' && item.parent_index != null) {
            if (!mergedMap.has(item.parent_index)) mergedMap.set(item.parent_index, new Map())
            const children = mergedMap.get(item.parent_index)!
            const existing = children.get(item.type_id)
            if (existing) {
                existing.quantity_dropped += item.quantity_dropped
                existing.quantity_destroyed += item.quantity_destroyed
            } else {
                children.set(item.type_id, { ...item })
            }
        }
    }
    // One row per stack, same as the top-level table
    const result = new Map<number, any[]>()
    for (const [parentIdx, children] of mergedMap) {
        const rows: any[] = []
        for (const item of children.values()) {
            rows.push({
                ...item,
                total_value: (item.price ?? 0) * (item.quantity_dropped + item.quantity_destroyed),
            })
        }
        if (rows.length > 0) result.set(parentIdx, rows)
    }
    return result
})

const itemLink = (item: any): string => {
    return `/item/${item.type_id}`
}

const slotLabels: Record<string, string> = {
    high: 'High Slots', mid: 'Mid Slots', low: 'Low Slots', rig: 'Rig Slots',
    subsystem: 'Subsystems', drone: 'Drone Bay', cargo: 'Cargo', fuel: 'Fuel Bay',
    fighter: 'Fighters', fighter_bay: 'Fighter Bay', fleet: 'Fleet Hangar',
    ship_maintenance_bay: 'Ship Maintenance Bay',
    specialized: 'Specialized Hold', service: 'Service Slots',
    implant: 'Implants', booster: 'Boosters', other: 'Other',
}

const slotOrder = ['high', 'mid', 'low', 'rig', 'subsystem', 'service', 'drone', 'fighter', 'fighter_bay', 'cargo', 'fuel', 'fleet', 'ship_maintenance_bay', 'specialized', 'implant', 'booster', 'other']

// Items: collapsible sections + sorting
const collapsedSlots = ref<Set<string>>(new Set())
const toggleSlot = (slot: string) => {
    if (collapsedSlots.value.has(slot)) collapsedSlots.value.delete(slot)
    else collapsedSlots.value.add(slot)
}

// Sorting is global rather than per-section. It was previously scoped to one
// slot at a time, which meant every section header had to carry its own pair of
// sort controls — five repetitions of "QTY VALUE" down a table that already has
// a header row — and sorting one section silently left the others alone.
type SortCol = 'name' | 'dropped' | 'destroyed' | 'value'
const sortState = ref<{ col: SortCol; dir: 'asc' | 'desc' } | null>(null)
const toggleSort = (col: SortCol) => {
    if (sortState.value?.col === col) {
        sortState.value.dir = sortState.value.dir === 'asc' ? 'desc' : 'asc'
    } else {
        sortState.value = { col, dir: col === 'name' ? 'asc' : 'desc' }
    }
}

const sortedItems = (slot: string) => {
    const items = itemsBySlot.value[slot] || []
    if (!sortState.value) return items
    const { col, dir } = sortState.value
    return [...items].sort((a: any, b: any) => {
        let av: number, bv: number
        switch (col) {
            case 'name': return dir === 'asc' ? (a.type_name ?? '').localeCompare(b.type_name ?? '') : (b.type_name ?? '').localeCompare(a.type_name ?? '')
            case 'dropped': av = a.quantity_dropped ?? 0; bv = b.quantity_dropped ?? 0; break
            case 'destroyed': av = a.quantity_destroyed ?? 0; bv = b.quantity_destroyed ?? 0; break
            case 'value': av = a.total_value ?? 0; bv = b.total_value ?? 0; break
            default: return 0
        }
        return dir === 'asc' ? av - bv : bv - av
    })
}

/**
 * What a container is worth: the wrap itself is scrap, everything is in what
 * it holds. Without this a jump freighter's Cargo section reads as a few
 * hundred million while the plastic wraps listed inside it hold billions,
 * because container contents live under `container_item` and never appear in
 * the slot's own item list.
 */
const containerValue = (itemIndex: number) =>
    (childrenByParent.value.get(itemIndex) ?? []).reduce((sum: number, c: any) => sum + (c.total_value ?? 0), 0)

const rowValue = (i: any) => i._status === 'container' ? containerValue(i.item_index) : (i.total_value ?? 0)

const slotTotal = (slot: string) => {
    return (itemsBySlot.value[slot] || []).reduce((sum, i) => sum + rowValue(i), 0)
}

/**
 * The three buckets the loss divides into, summing to the table's grand total.
 *
 * These are derived from the item rows rather than the killmail's own
 * `fitted_value`/`total_value` aggregates, which are computed from a different
 * price snapshot and disagree with the rows by a fraction of a percent — a
 * footer whose parts visibly fail to add up to its own total is worse than one
 * that differs slightly from the header.
 */
const FITTING_SLOTS = ['high', 'mid', 'low', 'rig', 'subsystem', 'service']

const fittedItemsValue = computed(() =>
    FITTING_SLOTS.reduce((sum, slot) => sum + slotTotal(slot), 0))

const otherItemsValue = computed(() =>
    slotOrder.filter(s => !FITTING_SLOTS.includes(s)).reduce((sum, slot) => sum + slotTotal(slot), 0))

const itemsGrandTotal = computed(() =>
    (kill.value?.victim.ship_price ?? 0) + fittedItemsValue.value + otherItemsValue.value)

/** Section total split the way the rest of the page splits value. */
const slotSplit = (slot: string) => {
    let dropped = 0
    let destroyed = 0
    const add = (i: any) => {
        dropped += (i.price ?? 0) * (i.quantity_dropped ?? 0)
        destroyed += (i.price ?? 0) * (i.quantity_destroyed ?? 0)
    }
    for (const i of itemsBySlot.value[slot] || []) {
        add(i)
        if (i._status === 'container') for (const c of childrenByParent.value.get(i.item_index) ?? []) add(c)
    }
    return { dropped, destroyed }
}

// Featured attackers
const finalBlow = computed(() => kill.value?.attackers.find(a => a.final_blow) ?? null)

const topDamage = computed(() => {
    if (!kill.value?.attackers.length) return null
    const top = kill.value.attackers[0]! // already sorted by damage desc
    // Only show if different from final blow
    if (top.character_id && top.character_id === finalBlow.value?.character_id) return null
    if (!top.character_id && top.corporation_id === finalBlow.value?.corporation_id) return null
    return top
})

const validMobileTabs = new Set(['kill', 'tools'])
const mobileTabFromHash = (hash: string) => {
    const t = hash.replace('#', '')
    return validMobileTabs.has(t) ? t as 'kill' | 'tools' : 'kill'
}
const mobileTab = ref<'kill' | 'tools'>(
    import.meta.client ? mobileTabFromHash(window.location.hash) : 'kill',
)
const setMobileTab = (tab: typeof mobileTab.value) => {
    mobileTab.value = tab
    const hash = tab === 'kill' ? '' : `#${tab}`
    window.history.pushState(null, '', hash || window.location.pathname + window.location.search)
}
if (import.meta.client) {
    window.addEventListener('popstate', () => {
        mobileTab.value = mobileTabFromHash(window.location.hash)
    })
}

// Update page title with active mobile tab
const mobileTabLabels: Record<string, string> = { kill: '', tools: 'Tools' }
useHead({ title: computed(() => {
    const label = mobileTabLabels[mobileTab.value]
    return label ? `${killTitle.value} (${label})` : killTitle.value
}) })

// Attacker pagination — start at 10, then reveal in larger chunks.
const attackerLimit = ref(10)
const visibleAttackers = computed(() => kill.value?.attackers.slice(0, attackerLimit.value) || [])
const hasMoreAttackers = computed(() => (kill.value?.attackers.length || 0) > attackerLimit.value)
const showMoreAttackers = () => { attackerLimit.value += 25 }

// Always show the battle link. The /battle/<id>?killmail=<id> page auto-redirects
// to a saved battle when one exists, and renders an empty-state otherwise.
const battleLink = computed(() => `/battle/${id.value}?killmail=${id.value}`)

/** Everything the "Open in" menu needs — mirrors what KillNavbar used to take. */
const toolContext = computed(() => ({
    killmailId: kill.value!.killmail_id,
    killmailHash: kill.value!.killmail_hash,
    systemName: kill.value!.solar_system_name,
    systemId: kill.value!.solar_system_id,
    constellationId: kill.value!.constellation_id,
    constellationName: kill.value!.constellation_name,
    regionName: kill.value!.region_name,
    regionId: kill.value!.region_id,
    shipTypeId: kill.value!.victim.ship_type_id,
    shipGroupId: kill.value!.victim.ship_group_id,
    shipGroupName: kill.value!.victim.ship_group_name,
    shipName: kill.value!.victim.ship_name,
    characterId: kill.value!.victim.character_id,
    characterName: kill.value!.victim.character_name,
    corporationId: kill.value!.victim.corporation_id,
    corporationName: kill.value!.victim.corporation_name,
    allianceId: kill.value!.victim.alliance_id,
    allianceName: kill.value!.victim.alliance_name,
    finalBlowShipTypeId: finalBlow.value?.ship_type_id ?? null,
    finalBlowShipName: finalBlow.value?.ship_name ?? null,
    topDamageShipTypeId: topDamage.value?.ship_type_id ?? null,
    topDamageShipName: topDamage.value?.ship_name ?? null,
    siblings: siblings.value,
}))
</script>

<template>
    <!-- Loading -->
    <div v-if="pending" class="flex items-center justify-center py-20">
        <Icon name="lucide:loader" class="text-2xl text-gray-500 animate-spin" />
    </div>

    <div v-else-if="kill">
        <KillHeader
            :kill="kill"
            :final-blow="finalBlow"
            :battle-link="battleLink"
            :siblings="siblings"
            :tool-context="toolContext"
        />

        <!-- ===== DESKTOP: two columns, right spans full height ===== -->
        <div v-if="isDesktop !== false" class="hidden md:grid grid-cols-[1fr_minmax(0,374px)] gap-4">
            <!-- LEFT COLUMN: Fitting + Info + Items -->
            <div class="space-y-4">
                <!-- Fitting Wheel + Info Box -->
                <div class="flex gap-4 items-start">
                    <KillFittingPanel
                        class="flex-1 min-w-[350px] max-w-[650px]"
                        :killmail-id="kill.killmail_id"
                        :ship-type-id="kill.victim.ship_type_id!"
                        :ship-name="kill.victim.ship_name"
                        :items="fittedItems"
                    />

                    <!-- Info Box -->
                    <div class="flex-shrink-0 w-[252px]">
                        <div class="rounded-xl border border-white/[0.08] overflow-hidden"
                            :style="victimAccent ? { borderColor: victimAccent.border } : undefined">
                            <!-- Character portrait with overlay -->
                            <div class="relative">
                                <div v-if="kill.victim.character_id" class="w-full aspect-square bg-white/[0.04]">
                                    <img :src="`/images/characters/${kill.victim.character_id}/portrait?size=512`" :alt="kill.victim.character_name || 'Victim portrait'" class="w-full h-full object-cover">
                                </div>
                                <div v-else class="w-full aspect-square bg-white/[0.04] flex items-center justify-center">
                                    <Icon name="lucide:building" class="text-4xl text-gray-600" />
                                </div>
                                <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-transparent to-transparent pointer-events-none"></div>
                                <!-- Top badges -->
                                <div class="absolute top-2 left-2 flex gap-1">
                                    <span v-if="kill.is_solo" class="px-1.5 py-0.5 rounded text-fine font-bold bg-amber-500/20 text-amber-400 backdrop-blur-sm">SOLO</span>
                                    <span v-if="kill.is_npc" class="px-1.5 py-0.5 rounded text-fine font-bold bg-purple-500/20 text-purple-400 backdrop-blur-sm">NPC</span>
                                    <span v-if="kill.total_value >= 1e9" class="px-1.5 py-0.5 rounded text-fine font-bold bg-green-500/20 text-green-400 backdrop-blur-sm">{{ formatIsk(kill.total_value) }}</span>
                                </div>
                                <!-- Corp + Alliance logos bottom-right -->
                                <div class="absolute bottom-2 right-2 flex gap-1 z-10">
                                    <NuxtLink v-if="kill.victim.corporation_id" :to="`/corporation/${kill.victim.corporation_id}`"
                                        class="w-10 h-10 rounded-lg overflow-hidden bg-black/60 border border-white/[0.1] hover:border-blue-400/40 transition-all hover:scale-125 hover:z-20">
                                        <img :src="`/images/corporations/${kill.victim.corporation_id}/logo?size=64`" :alt="kill.victim.corporation_name || 'Corporation logo'" class="w-full h-full">
                                    </NuxtLink>
                                    <NuxtLink v-if="kill.victim.alliance_id" :to="`/alliance/${kill.victim.alliance_id}`"
                                        class="w-10 h-10 rounded-lg overflow-hidden bg-black/60 border border-white/[0.1] hover:border-blue-400/40 transition-all hover:scale-125 hover:z-20">
                                        <img :src="`/images/alliances/${kill.victim.alliance_id}/logo?size=64`" :alt="kill.victim.alliance_name || 'Alliance logo'" class="w-full h-full">
                                    </NuxtLink>
                                </div>
                                <!-- Name overlay bottom-left -->
                                <div class="absolute inset-x-0 bottom-0 p-3 pr-24 space-y-0.5 z-10 pointer-events-none [&_a]:pointer-events-auto">
                                    <NuxtLink v-if="kill.victim.character_id" :to="`/character/${kill.victim.character_id}`" class="text-sm text-white font-semibold drop-shadow-lg block truncate hover:text-blue-400 transition-colors">{{ kill.victim.character_name || 'Unknown' }}</NuxtLink>
                                    <div v-else class="text-sm text-white font-semibold drop-shadow-lg truncate">{{ kill.victim.corporation_name || 'Unknown' }}</div>
                                    <div v-if="kill.victim.character_name && kill.victim.corporation_name" class="text-xs text-gray-300/80 drop-shadow truncate">
                                        <NuxtLink :to="`/corporation/${kill.victim.corporation_id}`" class="hover:text-blue-400">{{ kill.victim.corporation_name }}</NuxtLink>
                                    </div>
                                    <div v-if="kill.victim.alliance_name" class="text-fine text-gray-400/80 drop-shadow truncate">
                                        <NuxtLink :to="`/alliance/${kill.victim.alliance_id}`" class="hover:text-blue-400">{{ kill.victim.alliance_name }}</NuxtLink>
                                    </div>
                                </div>
                            </div>
                            <!-- Stats + ISK values -->
                            <KillInfoStats :kill="kill" :fitted-items-value="fittedItemsValue" interactive />
                        </div>
                    </div>
                </div>

                <!-- Items table -->
            <div class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-2">
                <!-- Table header. Dropped and destroyed get named columns: the
                     split used to live only in the tint of a quantity pill, so
                     the reader had to already know the code to read the table. -->
                <div class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-1.5 text-fine font-bold uppercase tracking-wider border-b border-white/[0.08] select-none">
                    <div></div>
                    <div class="text-gray-600 cursor-pointer hover:text-gray-400 transition-colors" @click="toggleSort('name')">
                        Item <Icon v-if="sortState?.col === 'name'" :name="sortState.dir === 'asc' ? 'lucide:arrow-up' : 'lucide:arrow-down'" class="inline text-fine" />
                    </div>
                    <div class="text-center text-green-500/70 cursor-pointer hover:text-green-400 transition-colors" @click="toggleSort('dropped')">
                        Dropped <Icon v-if="sortState?.col === 'dropped'" :name="sortState.dir === 'asc' ? 'lucide:arrow-up' : 'lucide:arrow-down'" class="inline text-fine" />
                    </div>
                    <div class="text-center text-red-500/70 cursor-pointer hover:text-red-400 transition-colors" @click="toggleSort('destroyed')">
                        Destroyed <Icon v-if="sortState?.col === 'destroyed'" :name="sortState.dir === 'asc' ? 'lucide:arrow-up' : 'lucide:arrow-down'" class="inline text-fine" />
                    </div>
                    <div class="text-right text-gray-600 cursor-pointer hover:text-gray-400 transition-colors" @click="toggleSort('value')">
                        Value <Icon v-if="sortState?.col === 'value'" :name="sortState.dir === 'asc' ? 'lucide:arrow-up' : 'lucide:arrow-down'" class="inline text-fine" />
                    </div>
                </div>

                <!-- Ship hull row -->
                <NuxtLink :to="`/item/${kill.victim.ship_type_id}`" class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-1.5 items-center bg-red-500/[0.05] border-b border-white/[0.04] cursor-pointer hover:bg-red-500/[0.08] transition-colors">
                    <div class="flex items-center">
                        <div class="w-6 h-6 rounded overflow-hidden bg-white/[0.04] flex-shrink-0">
                            <img :src="`/images/types/${kill.victim.ship_type_id}/icon?size=64`" :alt="kill.victim.ship_name || 'Ship'" class="w-full h-full object-cover" loading="lazy">
                        </div>
                    </div>
                    <div class="text-xs text-red-300/80 font-medium truncate">
                        {{ kill.victim.ship_name }}
                        <span class="text-gray-500 font-normal">({{ kill.victim.ship_group_name }})</span>
                    </div>
                    <div></div>
                    <div class="text-center">
                        <span class="px-1.5 py-0.5 rounded text-fine font-medium bg-red-500/10 text-red-400">1</span>
                    </div>
                    <div class="text-right text-xs text-red-400/60 tabular-nums">
                        {{ formatIsk(kill.victim.ship_price) }}
                    </div>
                </NuxtLink>

                <template v-for="slot in slotOrder" :key="slot">
                    <template v-if="itemsBySlot[slot]?.length">
                        <!-- Section header. Carries its own totals, which used
                             to be a separate row at the foot of every section. -->
                        <div
                            class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-1.5 mt-1 bg-white/[0.03] border-b border-white/[0.04] cursor-pointer select-none hover:bg-blue-500/[0.05]"
                            @click="toggleSlot(slot)"
                        >
                            <div class="flex items-center justify-center">
                                <Icon :name="collapsedSlots.has(slot) ? 'lucide:chevron-right' : 'lucide:chevron-down'" class="text-fine text-gray-500" />
                            </div>
                            <div class="text-xs font-bold uppercase tracking-wider text-gray-400">
                                {{ slotLabels[slot] }}
                                <span class="text-gray-600 font-normal normal-case tracking-normal ml-1">({{ itemsBySlot[slot]!.length }})</span>
                            </div>
                            <!-- Right-aligned, unpilled: these are ISK subtotals
                                 sitting in columns whose item rows hold unit
                                 counts, so they must not read as quantities. -->
                            <div class="text-right text-fine tabular-nums text-green-400/60">
                                <template v-if="slotSplit(slot).dropped">{{ formatIsk(slotSplit(slot).dropped) }}</template>
                            </div>
                            <div class="text-right text-fine tabular-nums text-red-400/60">
                                <template v-if="slotSplit(slot).destroyed">{{ formatIsk(slotSplit(slot).destroyed) }}</template>
                            </div>
                            <div class="text-right text-xs text-gray-400 font-medium tabular-nums">{{ formatIsk(slotTotal(slot)) }}</div>
                        </div>

                        <!-- Items (collapsible) -->
                        <template v-if="!collapsedSlots.has(slot)">
                            <template v-for="(item, idx) in sortedItems(slot)" :key="`${item.type_id}-${idx}`">
                            <NuxtLink :to="itemLink(item)"
                                class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-1 items-center border-b border-white/[0.02] transition-colors cursor-pointer"
                                :class="[
                                    item._status === 'destroyed' ? 'bg-red-500/[0.05] hover:bg-red-500/[0.08]' : 'hover:bg-blue-500/[0.05]',
                                    item._status === 'dropped' ? 'bg-green-500/[0.02]' : '',
                                    item._status === 'container' ? 'bg-blue-500/[0.03]' : '',
                                ]"
                            >
                                <div class="flex items-center">
                                    <div class="w-6 h-6 rounded overflow-hidden bg-white/[0.04] flex-shrink-0">
                                        <img :src="`/images/types/${item.type_id}/icon?size=64`" :alt="item.type_name || 'Item'" class="w-full h-full object-cover" loading="lazy">
                                    </div>
                                </div>
                                <div class="text-xs truncate" :class="
                                    item._status === 'container' ? 'text-blue-300 font-medium'
                                    : item._status === 'destroyed' ? 'text-red-300/80'
                                    : 'text-gray-300'
                                ">
                                    {{ item.type_name || `Type ${item.type_id}` }}
                                    <span v-if="item._status === 'container' && childrenByParent.get(item.item_index)?.length" class="text-fine text-gray-500 ml-1">({{ childrenByParent.get(item.item_index)!.length }} items)</span>
                                </div>
                                <div class="text-center">
                                    <span v-if="item.quantity_dropped" class="px-1.5 py-0.5 rounded text-fine font-medium bg-green-500/10 text-green-400 tabular-nums">
                                        {{ item.quantity_dropped }}
                                    </span>
                                </div>
                                <div class="text-center">
                                    <span v-if="item.quantity_destroyed" class="px-1.5 py-0.5 rounded text-fine font-medium bg-red-500/10 text-red-400 tabular-nums">
                                        {{ item.quantity_destroyed }}
                                    </span>
                                </div>
                                <div class="text-right text-xs tabular-nums" :class="item._status === 'destroyed' ? 'text-red-400/60' : 'text-gray-500'">
                                    <template v-if="rowValue(item)">{{ formatIsk(rowValue(item)) }}</template>
                                </div>
                            </NuxtLink>
                            <!-- Nested children of containers -->
                            <template v-if="item._status === 'container' && childrenByParent.get(item.item_index)?.length">
                                <NuxtLink v-for="(child, ci) in childrenByParent.get(item.item_index)!" :key="`child-${child.type_id}-${ci}`"
                                    :to="itemLink(child)"
                                    class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-0.5 items-center border-b border-white/[0.02] cursor-pointer hover:bg-blue-500/[0.05] transition-colors bg-blue-500/[0.01]">
                                    <div class="flex items-center pl-3">
                                        <Icon name="lucide:corner-down-right" class="text-fine text-gray-600 mr-1" />
                                        <div class="w-5 h-5 rounded overflow-hidden bg-white/[0.04] flex-shrink-0">
                                            <img :src="`/images/types/${child.type_id}/icon?size=64`" :alt="child.type_name || 'Item'" class="w-full h-full object-cover" loading="lazy">
                                        </div>
                                    </div>
                                    <div class="text-xs truncate text-gray-400">{{ child.type_name || `Type ${child.type_id}` }}</div>
                                    <div class="text-center">
                                        <span v-if="child.quantity_dropped" class="px-1 py-0.5 rounded text-fine font-medium bg-green-500/10 text-green-400 tabular-nums">
                                            {{ child.quantity_dropped }}
                                        </span>
                                    </div>
                                    <div class="text-center">
                                        <span v-if="child.quantity_destroyed" class="px-1 py-0.5 rounded text-fine font-medium bg-red-500/10 text-red-400 tabular-nums">
                                            {{ child.quantity_destroyed }}
                                        </span>
                                    </div>
                                    <div class="text-right text-fine tabular-nums text-gray-500">
                                        <template v-if="child.total_value">{{ formatIsk(child.total_value) }}</template>
                                    </div>
                                </NuxtLink>
                            </template>
                            </template>
                        </template>
                    </template>
                </template>

                <!-- Grand total. Ship and fitting are called out because they
                     are the two figures people quote; the remainder keeps the
                     column honest so Total is the sum of what is above it. -->
                <div class="mt-1 pt-1.5 border-t border-white/[0.08]">
                    <div v-if="fittedItemsValue" class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-0.5">
                        <div></div>
                        <div class="text-xs text-gray-500 text-right">Fitted</div>
                        <div></div>
                        <div></div>
                        <div class="text-right text-xs text-gray-400 tabular-nums">{{ formatIsk(fittedItemsValue) }}</div>
                    </div>
                    <div v-if="kill.victim.ship_price" class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-0.5">
                        <div></div>
                        <div class="text-xs text-gray-500 text-right">Ship</div>
                        <div></div>
                        <div></div>
                        <div class="text-right text-xs text-gray-400 tabular-nums">{{ formatIsk(kill.victim.ship_price) }}</div>
                    </div>
                    <div v-if="otherItemsValue" class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-0.5">
                        <div></div>
                        <div class="text-xs text-gray-500 text-right">Cargo &amp; other</div>
                        <div></div>
                        <div></div>
                        <div class="text-right text-xs text-gray-400 tabular-nums">{{ formatIsk(otherItemsValue) }}</div>
                    </div>
                    <div class="grid grid-cols-[28px_1fr_64px_72px_90px] gap-2 px-2 py-1.5 mt-1 bg-white/[0.03] border-t border-white/[0.08]">
                        <div></div>
                        <div class="text-xs text-gray-300 font-medium text-right">Total</div>
                        <div></div>
                        <div></div>
                        <div class="text-right text-xs text-white font-semibold tabular-nums">{{ formatIsk(itemsGrandTotal) }}</div>
                    </div>
                </div>
            </div>
            </div>

            <!-- RIGHT COLUMN: Featured Attackers + All Attackers + Comments (spans full height) -->
            <div class="space-y-4">
            <!-- Solo 1v1 matchup stats — only for solo player kills with real hulls (no pods) -->
            <KillMatchupBox
                v-if="kill.is_solo && finalBlow?.character_id && finalBlow.ship_type_id && kill.victim.ship_type_id && kill.victim.ship_group_id !== 29"
                :attacker-ship-type-id="finalBlow.ship_type_id"
                :attacker-ship-name="finalBlow.ship_name"
                :victim-ship-type-id="kill.victim.ship_type_id"
                :victim-ship-name="kill.victim.ship_name"
            />
            <div class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-3">
                <KillFeaturedAttackers
                    :final-blow="finalBlow"
                    :top-damage="topDamage"
                    :total-damage="kill.total_damage"
                    interactive
                />

                <KillAttackerList
                    :attackers="visibleAttackers"
                    :attacker-count="kill.attacker_count"
                    :total-damage="kill.total_damage"
                    :has-more="hasMoreAttackers"
                    :remaining="kill.attacker_count - attackerLimit"
                    interactive
                    @show-more="showMoreAttackers"
                />
            </div>

            <!-- Comments (desktop: sits below attackers in the right column) -->
            <div class="hidden md:block rounded-lg bg-white/[0.04] border border-white/[0.08] p-3">
                <CommentsCommentList
                    :target-type="1"
                    :target-id="kill.killmail_id"
                />
            </div>
            </div>
        </div>

        <!-- ===== MOBILE ===== -->
        <div v-if="isDesktop !== true" class="md:hidden -mt-2">
            <div class="flex overflow-x-auto border-b border-white/[0.08] mb-4 scrollbar-hide">
                <button
                    v-for="tab in [
                        { key: 'kill', label: 'Kill', icon: 'lucide:swords' },
                        { key: 'tools', label: 'Tools', icon: 'lucide:external-link' },
                    ] as const"
                    :key="tab.key"
                    class="flex items-center justify-center gap-1.5 px-3 py-3 text-sm font-medium transition-colors border-b-2 whitespace-nowrap"
                    :class="mobileTab === tab.key ? 'text-white border-blue-400' : 'text-gray-500 border-transparent'"
                    @click="setMobileTab(tab.key)"
                >
                    <Icon :name="tab.icon" class="text-base" />
                    <span :class="mobileTab === tab.key ? '' : 'hidden'">{{ tab.label }}</span>
                </button>

                <!-- Battle Report link in tab bar -->
                <NuxtLink
                    v-if="battleLink"
                    :to="battleLink"
                    class="flex items-center justify-center gap-1.5 px-3 py-3 text-sm font-medium transition-colors border-b-2 whitespace-nowrap text-amber-400/70 border-transparent ml-auto"
                >
                    <Icon name="lucide:swords" class="text-base" />
                    <span>Battle</span>
                </NuxtLink>
            </div>

            <!-- Kill tab: Fitting Wheel -->
            <div v-show="mobileTab === 'kill'" class="mb-6">
                <KillFittingPanel
                    :killmail-id="kill.killmail_id"
                    :ship-type-id="kill.victim.ship_type_id!"
                    :ship-name="kill.victim.ship_name"
                    :items="fittedItems"
                    wheel-class="mb-4"
                />
            </div>

            <!-- Kill tab: Character information -->
            <div v-show="mobileTab === 'kill'" class="mb-6">
                <div class="rounded-xl border border-white/[0.08] overflow-hidden mb-4">
                    <!-- Compact header: portrait + names + corp/alliance logos -->
                    <div class="flex items-center gap-3 p-3 bg-white/[0.04]">
                        <NuxtLink v-if="kill.victim.character_id" :to="`/character/${kill.victim.character_id}`"
                            class="w-16 h-16 rounded-lg overflow-hidden bg-white/[0.04] flex-shrink-0">
                            <img :src="`/images/characters/${kill.victim.character_id}/portrait?size=128`" :alt="kill.victim.character_name || 'Victim portrait'" class="w-full h-full object-cover">
                        </NuxtLink>
                        <div v-else class="w-16 h-16 rounded-lg bg-white/[0.04] flex-shrink-0 flex items-center justify-center">
                            <Icon name="lucide:building" class="text-2xl text-gray-600" />
                        </div>
                        <div class="flex-1 min-w-0 space-y-0.5">
                            <div class="text-sm text-white font-semibold truncate">
                                <NuxtLink v-if="kill.victim.character_id" :to="`/character/${kill.victim.character_id}`" class="inline hover:text-blue-400">{{ kill.victim.character_name || 'Unknown' }}</NuxtLink>
                                <span v-else>{{ kill.victim.corporation_name || 'Unknown' }}</span>
                            </div>
                            <div v-if="kill.victim.character_name && kill.victim.corporation_name" class="text-xs text-gray-300/80 truncate">
                                <NuxtLink :to="`/corporation/${kill.victim.corporation_id}`" class="inline hover:text-blue-400">{{ kill.victim.corporation_name }}</NuxtLink>
                            </div>
                            <div v-if="kill.victim.alliance_name" class="text-fine text-gray-400/80 truncate">
                                <NuxtLink :to="`/alliance/${kill.victim.alliance_id}`" class="inline hover:text-blue-400">{{ kill.victim.alliance_name }}</NuxtLink>
                            </div>
                            <div class="flex gap-1 pt-0.5">
                                <span v-if="kill.is_solo" class="px-1.5 py-0.5 rounded text-fine font-bold bg-amber-500/20 text-amber-400">SOLO</span>
                                <span v-if="kill.is_npc" class="px-1.5 py-0.5 rounded text-fine font-bold bg-purple-500/20 text-purple-400">NPC</span>
                            </div>
                        </div>
                        <div class="flex flex-row gap-1 flex-shrink-0">
                            <NuxtLink v-if="kill.victim.corporation_id" :to="`/corporation/${kill.victim.corporation_id}`"
                                class="w-16 h-16 rounded-lg overflow-hidden bg-black/60 border border-white/[0.08]">
                                <img :src="`/images/corporations/${kill.victim.corporation_id}/logo?size=64`" :alt="kill.victim.corporation_name || 'Corporation logo'" class="w-full h-full">
                            </NuxtLink>
                            <NuxtLink v-if="kill.victim.alliance_id" :to="`/alliance/${kill.victim.alliance_id}`"
                                class="w-16 h-16 rounded-lg overflow-hidden bg-black/60 border border-white/[0.08]">
                                <img :src="`/images/alliances/${kill.victim.alliance_id}/logo?size=64`" :alt="kill.victim.alliance_name || 'Alliance logo'" class="w-full h-full">
                            </NuxtLink>
                        </div>
                    </div>
                    <!-- Stats + ISK values -->
                    <KillInfoStats :kill="kill" :fitted-items-value="fittedItemsValue" />
                </div>
            </div>

            <!-- Kill tab: Items -->
            <div v-show="mobileTab === 'kill'" class="mb-6">
                <!-- Mobile items table -->
                <template v-for="slot in slotOrder" :key="slot">
                    <template v-if="itemsBySlot[slot]?.length">
                        <div class="px-2 py-1.5 mt-2 bg-white/[0.02] border-b border-white/[0.04]">
                            <span class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ slotLabels[slot] }}</span>
                            <span class="text-fine text-gray-600 ml-1">({{ itemsBySlot[slot]!.length }})</span>
                        </div>
                        <div
                            v-for="(item, idx) in itemsBySlot[slot]"
                            :key="`m-${item.type_id}-${item.flag_id}-${idx}`"
                            class="flex items-center gap-2 px-2 py-1.5 border-b border-white/[0.02]"
                            :class="item.parent_index != null ? 'pl-8' : ''"
                        >
                            <div v-if="item.parent_index != null" class="text-gray-500"><Icon name="lucide:corner-down-right" class="text-fine" /></div>
                            <div class="w-6 h-6 rounded overflow-hidden bg-white/[0.04] flex-shrink-0">
                                <img :src="`/images/types/${item.type_id}/icon?size=64`" class="w-full h-full" loading="lazy">
                            </div>
                            <span class="flex-1 text-xs text-gray-300 truncate">{{ item.type_name || `Type ${item.type_id}` }}</span>
                            <span v-if="item.quantity_dropped" class="px-1.5 py-0.5 rounded text-fine bg-green-500/10 text-green-400">{{ item.quantity_dropped }}</span>
                            <span v-if="item.quantity_destroyed" class="px-1.5 py-0.5 rounded text-fine bg-red-500/10 text-red-400">{{ item.quantity_destroyed }}</span>
                        </div>
                    </template>
                </template>

                <!-- Same footer as the desktop table -->
                <div class="mt-2 pt-1.5 border-t border-white/[0.08] space-y-0.5">
                    <div v-if="fittedItemsValue" class="flex justify-between px-2">
                        <span class="text-xs text-gray-500">Fitted</span>
                        <span class="text-xs text-gray-400 tabular-nums">{{ formatIsk(fittedItemsValue) }}</span>
                    </div>
                    <div v-if="kill.victim.ship_price" class="flex justify-between px-2">
                        <span class="text-xs text-gray-500">Ship</span>
                        <span class="text-xs text-gray-400 tabular-nums">{{ formatIsk(kill.victim.ship_price) }}</span>
                    </div>
                    <div v-if="otherItemsValue" class="flex justify-between px-2">
                        <span class="text-xs text-gray-500">Cargo &amp; other</span>
                        <span class="text-xs text-gray-400 tabular-nums">{{ formatIsk(otherItemsValue) }}</span>
                    </div>
                    <div class="flex justify-between px-2 py-1.5 mt-1 bg-white/[0.03] border-t border-white/[0.08]">
                        <span class="text-xs text-gray-300 font-medium">Total</span>
                        <span class="text-xs text-white font-semibold tabular-nums">{{ formatIsk(itemsGrandTotal) }}</span>
                    </div>
                </div>
            </div>

            <!-- Kill tab: Solo 1v1 matchup stats (solo player kills, real hulls only) -->
            <div v-show="mobileTab === 'kill'" class="mb-3">
                <KillMatchupBox
                    v-if="kill.is_solo && finalBlow?.character_id && finalBlow.ship_type_id && kill.victim.ship_type_id && kill.victim.ship_group_id !== 29"
                    :attacker-ship-type-id="finalBlow.ship_type_id"
                    :attacker-ship-name="finalBlow.ship_name"
                    :victim-ship-type-id="kill.victim.ship_type_id"
                    :victim-ship-name="kill.victim.ship_name"
                />
            </div>

            <!-- Kill tab: Attackers -->
            <div v-show="mobileTab === 'kill'">
                <KillFeaturedAttackers
                    :final-blow="finalBlow"
                    :top-damage="topDamage"
                    :total-damage="kill.total_damage"
                />

                <KillAttackerList
                    :attackers="visibleAttackers"
                    :attacker-count="kill.attacker_count"
                    :total-damage="kill.total_damage"
                    :has-more="hasMoreAttackers"
                    :remaining="kill.attacker_count - attackerLimit"
                    @show-more="showMoreAttackers"
                />
            </div>

            <!-- Tools tab -->
            <div v-show="mobileTab === 'tools'">
                <!-- Battle Report + Siblings -->
                <div class="flex flex-wrap gap-2 mb-4">
                    <NuxtLink
                        v-if="battleLink"
                        :to="battleLink"
                        class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium bg-white/[0.04] text-gray-300 border border-white/[0.08]"
                    >
                        <Icon name="lucide:swords" class="text-sm" />
                        Battle Report
                    </NuxtLink>
                    <NuxtLink
                        v-for="sib in siblings"
                        :key="sib.killmail_id"
                        :to="`/kill/${sib.killmail_id}`"
                        class="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-white/[0.04] border border-white/[0.08]"
                        v-tooltip="sib.ship_name || `Kill #${sib.killmail_id}`"
                    >
                        <div class="w-5 h-5 rounded overflow-hidden bg-white/[0.04] flex-shrink-0">
                            <img :src="`/images/types/${sib.ship_type_id}/icon?size=64`" class="w-full h-full object-cover">
                        </div>
                        <span class="text-xs text-gray-300">{{ sib.ship_name || `#${sib.killmail_id}` }}</span>
                        <span class="text-xs text-isk/70 tabular-nums">{{ formatIsk(sib.total_value) }}</span>
                    </NuxtLink>
                </div>

                <!-- External tools — shown directly -->
                <div class="rounded-lg bg-white/[0.04] border border-white/[0.08] overflow-hidden">
                    <div v-for="tool in [
                        { name: 'DOTLAN', icon: '/remotes/dotlan.png', items: [
                            { label: 'System', url: `https://evemaps.dotlan.net/system/${encodeURIComponent(kill.solar_system_name || '')}`, ok: !!kill.solar_system_name },
                            { label: 'Region', url: `https://evemaps.dotlan.net/region/${encodeURIComponent(kill.region_name || '')}`, ok: !!kill.region_name },
                        ]},
                        { name: 'EVEEye', icon: '/remotes/eveeye.svg', items: [
                            { label: 'Region', url: `https://eveeye.com/?m=${encodeURIComponent(kill.region_name || '')}`, ok: !!kill.region_name },
                        ]},
                        { name: 'EveShip.fit', icon: '/remotes/eveship-fit.png', items: [
                            { label: 'Fitting', url: `https://eveship.fit/fit/${kill.killmail_id}`, ok: !!kill.victim.ship_type_id },
                        ]},
                        { name: 'EVERef', icon: '/remotes/everef.png', items: [
                            { label: 'Ship', url: `https://everef.net/type/${kill.victim.ship_type_id}`, ok: !!kill.victim.ship_type_id },
                        ]},
                        { name: 'Jita.Space', icon: '/remotes/jita-space.png', items: [
                            { label: 'System', url: `https://www.jita.space/system/${kill.solar_system_id}`, ok: true },
                            { label: 'Region', url: `https://www.jita.space/region/${kill.region_id}`, ok: !!kill.region_id },
                        ]},
                        { name: 'EVEWho', icon: '/remotes/evewho.png', items: [
                            { label: 'Character', url: `https://evewho.com/character/${kill.victim.character_id}`, ok: !!kill.victim.character_id },
                            { label: 'Corporation', url: `https://evewho.com/corporation/${kill.victim.corporation_id}`, ok: !!kill.victim.corporation_id },
                            { label: 'Alliance', url: `https://evewho.com/alliance/${kill.victim.alliance_id}`, ok: !!kill.victim.alliance_id },
                        ]},
                        { name: 'zKillboard', icon: '/remotes/zkillboard.png', items: [
                            { label: 'Killmail', url: `https://zkillboard.com/kill/${kill.killmail_id}/`, ok: true },
                            { label: 'System', url: `https://zkillboard.com/system/${kill.solar_system_id}/`, ok: true },
                            { label: 'Ship', url: `https://zkillboard.com/ship/${kill.victim.ship_type_id}/`, ok: !!kill.victim.ship_type_id },
                            { label: 'Character', url: `https://zkillboard.com/character/${kill.victim.character_id}/`, ok: !!kill.victim.character_id },
                        ]},
                        { name: 'kb.evetools.org', icon: '/remotes/evetools.png', items: [
                            { label: 'Killmail', url: `https://kb.evetools.org/kill/${kill.killmail_id}`, ok: true },
                            { label: 'Victim Ship', url: `https://kb.evetools.org/ship/${kill.victim.ship_type_id}`, ok: !!kill.victim.ship_type_id },
                            { label: 'Final Blow Ship', url: `https://kb.evetools.org/ship/${finalBlow?.ship_type_id}`, ok: !!finalBlow?.ship_type_id },
                            { label: 'Top Damage Ship', url: `https://kb.evetools.org/ship/${topDamage?.ship_type_id}`, ok: !!topDamage?.ship_type_id },
                            { label: 'Character', url: `https://kb.evetools.org/character/${kill.victim.character_id}`, ok: !!kill.victim.character_id },
                            { label: 'Corporation', url: `https://kb.evetools.org/corporation/${kill.victim.corporation_id}`, ok: !!kill.victim.corporation_id },
                            { label: 'Alliance', url: `https://kb.evetools.org/alliance/${kill.victim.alliance_id}`, ok: !!kill.victim.alliance_id },
                            { label: 'System', url: `https://kb.evetools.org/system/${kill.solar_system_id}`, ok: !!kill.solar_system_id },
                            { label: 'Region', url: `https://kb.evetools.org/region/${kill.region_id}`, ok: !!kill.region_id },
                        ]},
                        { name: 'ESI', icon: '', items: [
                            { label: 'Raw JSON', url: `https://esi.evetech.net/latest/killmails/${kill.killmail_id}/${kill.killmail_hash}/`, ok: true },
                        ]},
                    ]" :key="tool.name" class="border-b border-white/[0.04] last:border-b-0">
                        <div class="flex items-center gap-2 px-3 py-2 bg-white/[0.02]">
                            <NuxtImg v-if="tool.icon" :src="tool.icon" alt="" width="16" height="16" class="w-4 h-4 object-contain" />
                            <Icon v-else name="lucide:code" class="text-sm text-gray-500" />
                            <span class="text-xs font-medium text-gray-400">{{ tool.name }}</span>
                        </div>
                        <div class="flex flex-wrap gap-1.5 px-3 py-2">
                            <a
                                v-for="item in tool.items.filter(i => i.ok)"
                                :key="item.url"
                                :href="item.url"
                                target="_blank"
                                rel="noopener noreferrer"
                                class="flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-white/[0.04] border border-white/[0.06] text-xs text-gray-300 active:bg-white/[0.1] transition-colors"
                            >
                                {{ item.label }}
                                <Icon name="lucide:external-link" class="text-fine text-gray-600" />
                            </a>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Comments (mobile only — desktop shows them in the right column) -->
        <div v-if="isDesktop !== true" class="md:hidden mt-6 px-3">
            <CommentsCommentList
                :target-type="1"
                :target-id="kill.killmail_id"
            />
        </div>
    </div>
</template>
