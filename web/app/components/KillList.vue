<script setup lang="ts">
import {
    type KilllistLossEntities,
    type KilllistRow,
    isKilllistLoss,
    matchesMetaGroup,
} from '#shared/utils/killlistRow'

const props = defineProps<{
    /** Kill list filter type (e.g. 'latest', 'big', 'solo') */
    killlistType?: string
    /** API endpoint override */
    apiEndpoint?: string
    /** Dedicated entity endpoint — when set, uses this with just role/page/limit */
    entityEndpoint?: string
    /** For combined kill/loss view: entity type being viewed */
    victimEntityType?: 'character' | 'corporation' | 'alliance' | 'faction'
    /** For combined kill/loss view: entity ID to match losses */
    victimEntityId?: number
    /** Multi-entity loss highlighting: victim matching any of these IDs shows as red */
    lossEntities?: KilllistLossEntities
    /** Campaign side coloring: array indexed by side, victim rows get their
     *  side's tint (0=red, 1=blue, 2=purple, 3=amber — matches the campaign
     *  page palette). Takes precedence over lossEntities/war coloring;
     *  victims on no side keep the normal zebra striping. */
    sideEntities?: Array<{ characterIds?: number[]; corporationIds?: number[]; allianceIds?: number[] }>
    /** Optional corp-palette accent hex per side (from campaignSideAccents).
     *  When a side has one, its rows use the accent for the border/tint
     *  instead of the fixed side class. Null entries fall back per side. */
    sideColors?: Array<string | null>
    /** Filter role: 'attacker' (kills), 'victim' (losses), 'any' (combined) */
    entityRole?: 'attacker' | 'victim' | 'any'
    /** Filter by war ID */
    warId?: number
    /** War side filter: victim corps (comma-separated) */
    warSideCorps?: string
    /** War side filter: victim alliances (comma-separated) */
    warSideAlliances?: string
    /** War date bounds for partition pruning */
    warStart?: string
    warEnd?: string | null
    /** Faction war: victim faction IDs (comma-separated) */
    victimFactions?: string
    /** War side coloring: aggressor entity IDs */
    warAggressorCorps?: number[]
    warAggressorAlliances?: number[]
    /** War side coloring: defender entity IDs */
    warDefenderCorps?: number[]
    warDefenderAlliances?: number[]
    /** Battle scoping */
    battleSystemId?: number
    battleStart?: string
    battleEnd?: string
    /** Extra query params passed through to the API (e.g. advanced search filters) */
    extraParams?: Record<string, any>
    /** WebSocket stream topics — null or omitted disables live updates */
    streamTopics?: string[] | null
    /** Total number of pages — only pass when the backing query is count-capable
     *  (entity-scoped lists, battles, wars, etc.). When provided the component
     *  renders numbered pagination; otherwise it falls back to a minimal
     *  prev / page-N / next control that matches what cursor-based endpoints
     *  can actually support. */
    totalPages?: number
    /** Opt-in: render a quick-filter button + pop-down panel (region/system/
     *  ship/security). Used on the homepage killlist. When a filter is active
     *  it routes queries through /api/killlist/advanced and disables the live
     *  feed. Defaults off, so every other KillList usage is unaffected. */
    quickFilter?: boolean
}>()

// Rows arrive from two producers — the REST killlist endpoints and the relay
// `killlist` event — which emit the same shape by construction. See KilllistRow.
type KillRow = KilllistRow

// ── Pagination state (cursor-based) ─────────────────────────────────────────

const limit = ref(50)
const limitOptions = [10, 25, 50, 100]

// Cursor stack: index 0 = page 1 (null cursor), index 1 = page 2 (after=X), etc.
// Only populated in cursor mode — rollup-backed types ignore it.
const cursors = ref<(number | null)[]>([null])
const pageIndex = ref(0)
const currentPage = computed(() => pageIndex.value + 1)

// Mirror of `data.value.totalPages` when the server returned one. `null` until
// the first response lands; its presence flips the component into numbered
// mode and changes how subsequent fetches are paginated (page=N vs after=X).
const apiTotalPages = ref<number | null>(null)

// ── Quick filter (opt-in, homepage) ──────────────────────────────────────────
interface QuickFilterState {
    regionId?: number
    regionName?: string
    systemId?: number
    systemName?: string
    shipId?: number
    shipName?: string
    securityTypes: string[]
    soloOnly?: boolean
    npcOnly?: boolean
    iskValue?: string   // '5b' | '10b'
    techLevel?: string  // 't1' | 't2' | 't3'
}
const quickFilterOpen = ref(false)
const activeFilters = ref<QuickFilterState>({ securityTypes: [] })

const hasActiveFilter = computed(() => !!(
    activeFilters.value.regionId || activeFilters.value.systemId
    || activeFilters.value.shipId || activeFilters.value.securityTypes.length
    || activeFilters.value.soloOnly || activeFilters.value.npcOnly
    || activeFilters.value.iskValue || activeFilters.value.techLevel
))
const isFiltering = computed(() => props.quickFilter === true && hasActiveFilter.value)
// Every quick filter is now verifiable straight off the stream payload, so the
// live feed keeps flowing regardless of which ones are active.
const liveActive = computed(() => !!props.streamTopics)
const activeFilterCount = computed(() => {
    let n = 0
    if (activeFilters.value.regionId || activeFilters.value.systemId) n++
    if (activeFilters.value.shipId) n++
    n += activeFilters.value.securityTypes.length
    if (activeFilters.value.soloOnly) n++
    if (activeFilters.value.npcOnly) n++
    if (activeFilters.value.iskValue) n++
    if (activeFilters.value.techLevel) n++
    return n
})
// Build the /api/killlist/advanced `filters` JSON from the quick-filter state.
// Returns '{}' when nothing is active so the watch below stays quiet for the
// unfiltered case and for every non-quick-filter KillList usage. Region/system
// are mutually exclusive (advanced prefers system); a 90d window keeps quiet
// regions from looking empty under advanced's 30d default.
const serializedFilters = computed(() => {
    if (!isFiltering.value) return '{}'
    const f: Record<string, any> = { timeRange: { preset: '90d' } }
    const loc: Record<string, any> = {}
    if (activeFilters.value.systemId) loc.systemId = activeFilters.value.systemId
    else if (activeFilters.value.regionId) loc.regionId = activeFilters.value.regionId
    if (activeFilters.value.securityTypes.length) loc.securityTypes = activeFilters.value.securityTypes
    if (Object.keys(loc).length) f.location = loc
    if (activeFilters.value.shipId) f.entities = { victim: [{ id: activeFilters.value.shipId, type: 'ship' }] }
    if (activeFilters.value.soloOnly) f.attackerCount = 'solo'
    if (activeFilters.value.npcOnly) f.attackerType = 'npc'
    if (activeFilters.value.iskValue) f.iskValue = activeFilters.value.iskValue
    if (activeFilters.value.techLevel) f.techLevel = activeFilters.value.techLevel
    return JSON.stringify(f)
})

// ── Data fetching ────────────────────────────────────────────────────────────

const endpoint = computed(() => {
    if (isFiltering.value) return '/api/killlist/advanced'
    return props.entityEndpoint || props.apiEndpoint || '/api/killlist'
})
const fetchParams = computed(() => {
    // Quick-filter active — route through the advanced endpoint with a
    // serialized filters payload. Cursor pagination only (no totalPages).
    if (isFiltering.value) {
        const p: Record<string, any> = { limit: limit.value, filters: serializedFilters.value }
        if (pageIndex.value > 0) {
            const cursor = cursors.value[pageIndex.value]
            if (cursor) p.after = cursor
        }
        return p
    }

    // Entity endpoint — simple params only. Only send `role` when the caller
    // explicitly passed entity-role: location killlists (system/const/region)
    // ignore it server-side, so omitting it keeps the URL honest and avoids
    // a misleading "combined" cache key.
    if (props.entityEndpoint) {
        const p: Record<string, any> = { limit: limit.value }
        if (props.entityRole) {
            const roleMap: Record<string, string> = { attacker: 'kills', victim: 'losses', any: 'combined' }
            p.role = roleMap[props.entityRole] || props.entityRole
        }
        const cursor = cursors.value[pageIndex.value]
        if (cursor) p.after = cursor
        if (props.extraParams) Object.assign(p, props.extraParams)
        return p
    }

    // Killlist endpoint
    const p: Record<string, any> = {
        type: props.killlistType || 'latest',
        limit: limit.value,
    }
    // Numbered mode (rollup-backed types) — send `page=N`. Cursor mode —
    // send `after=<cursor>`. Page 1 sends neither, lets the server return
    // the newest slice plus a `totalPages` hint that decides mode going forward.
    if (pageIndex.value > 0) {
        if (apiTotalPages.value !== null) {
            p.page = pageIndex.value + 1
        } else {
            const cursor = cursors.value[pageIndex.value]
            if (cursor) p.after = cursor
        }
    }

    if (props.victimEntityType && props.victimEntityId) {
        p.entityType = props.victimEntityType
        p.entityId = props.victimEntityId
        if (props.entityRole) p.entityRole = props.entityRole
    }
    if (props.warId) {
        p.warId = props.warId
        if (props.warSideCorps) p.warSideCorps = props.warSideCorps
        if (props.warSideAlliances) p.warSideAlliances = props.warSideAlliances
        if (props.warStart) p.warStart = props.warStart
        if (props.warEnd) p.warEnd = props.warEnd
    }
    if (props.victimFactions) p.victimFactions = props.victimFactions
    if (props.battleSystemId) {
        p.battleSystemId = props.battleSystemId
        if (props.battleStart) p.battleStart = props.battleStart
        if (props.battleEnd) p.battleEnd = props.battleEnd
    }
    if (props.extraParams) {
        Object.assign(p, props.extraParams)
    }
    return p
})


const { data, pending: loading, refresh } = await useApiFetch<{ kills: KillRow[]; hasMore?: boolean; cursor?: number | null; totalPages?: number }>(endpoint, {
    params: fetchParams,
    default: () => ({ kills: [], hasMore: false, cursor: null }),
    lazy: import.meta.client,
    getCachedData: cachedPayload,
})

const kills = computed({
    get: () => data.value?.kills || [],
    set: (v) => { data.value = { ...data.value, kills: v } },
})

const hasMore = computed(() => data.value?.hasMore ?? false)

// Capture server-side hints from each response: totalPages (unlocks numbered
// mode) and the next-page cursor (cursor mode only). Both are server-driven;
// the client never computes totalPages locally.
watch(data, (d) => {
    apiTotalPages.value = typeof d?.totalPages === 'number' ? d.totalPages : null
    if (d?.cursor && d.hasMore) {
        const nextIdx = pageIndex.value + 1
        if (cursors.value.length <= nextIdx) {
            cursors.value.push(d.cursor)
        } else {
            cursors.value[nextIdx] = d.cursor
        }
    }
}, { immediate: true })

// Re-fetch on pageIndex or limit change
watch([pageIndex, limit], () => refresh())

// Quick-filter changes reset pagination + drop numbered mode (advanced is
// cursor-only), then refetch. We refresh() explicitly (matching the pageIndex
// watch above) rather than relying on params auto-watch — when applying a
// filter from page 1 the page index doesn't change, so nothing else would.
watch(serializedFilters, () => {
    cursors.value = [null]
    pageIndex.value = 0
    apiTotalPages.value = null
    refresh()
})

// Same deal for filters owned by the PARENT (advanced search): reset the
// cursor stack so a filter change doesn't strand the user on a page that no
// longer exists. No refresh() here — extraParams feed fetchParams, so the
// params auto-watch already refetches; calling both fires two requests per
// change. Compared by content so a caller passing a fresh object with
// identical params doesn't reset the user's page for no reason.
watch(() => JSON.stringify(props.extraParams ?? {}), () => {
    cursors.value = [null]
    pageIndex.value = 0
    apiTotalPages.value = null
})

// ── Pagination controls ──────────────────────────────────────────────────────

// Numbered pagination. Takes priority from an explicit `totalPages` prop,
// then falls back to whatever the server attached to its response (rollup-
// backed /api/killlist types set this). Classic shape:
// [1] [...] [c-1] [c] [c+1] [...] [last]. Collapses at the edges.
type PageSlot = number | '...'
const effectiveTotalPages = computed<number | null>(() => {
    if (typeof props.totalPages === 'number' && props.totalPages > 0) return props.totalPages
    if (apiTotalPages.value !== null && apiTotalPages.value > 0) return apiTotalPages.value
    return null
})
const isNumberedMode = computed(() => effectiveTotalPages.value !== null)
const visiblePages = computed<PageSlot[]>(() => {
    const total = effectiveTotalPages.value ?? 0
    if (total <= 0) return []
    const current = pageIndex.value + 1
    const slots: PageSlot[] = []
    if (total <= 7) {
        for (let i = 1; i <= total; i++) slots.push(i)
        return slots
    }
    slots.push(1)
    if (current > 4) slots.push('...')
    const start = Math.max(2, current - 1)
    const end = Math.min(total - 1, current + 1)
    for (let i = start; i <= end; i++) slots.push(i)
    if (current < total - 3) slots.push('...')
    slots.push(total)
    return slots
})

// Compact page-number display for numbered pagination. The current page
// always shows its full value (with thousand separators) so the user can
// tell exactly where they are; neighbours and the "last" anchor get
// abbreviated because 92M-kill killlists produce page numbers in the
// millions that'd otherwise blow out the pagination bar.
const formatPageLabel = (p: number): string => {
    if (p === currentPage.value) return p.toLocaleString()
    if (p < 1000) return String(p)
    if (p < 1_000_000) {
        const k = p / 1000
        return `${k < 10 ? k.toFixed(1).replace(/\.0$/, "") : Math.round(k)}K`
    }
    const m = p / 1_000_000
    return `${m < 10 ? m.toFixed(1).replace(/\.0$/, "") : Math.round(m)}M`
}

const goNext = () => {
    if (isNumberedMode.value) {
        if (currentPage.value >= (effectiveTotalPages.value ?? 0)) return
    } else if (!hasMore.value) return
    pageIndex.value++
}

const goPrev = () => {
    if (pageIndex.value <= 0) return
    pageIndex.value--
}

const goToPage = (idx: number) => {
    if (idx < 0) return
    if (isNumberedMode.value) {
        if (idx >= (effectiveTotalPages.value ?? 0)) return
        pageIndex.value = idx
    } else if (idx < cursors.value.length) {
        pageIndex.value = idx
    }
}

const setLimit = (n: number) => {
    limit.value = n
    cursors.value = [null]
    pageIndex.value = 0
}

// ── Live kill stream (WebSocket) ─────────────────────────────────────────────

const stream = useKillStream(props.streamTopics ?? null)

// Every row renders in a desktop AND a mobile variant (CSS hides one).
// Post-hydration, unmount the inactive variant — with 50 rows × 2 layouts
// × ~6 imgs each, that halves this component's DOM weight.
const isDesktop = useIsDesktop()

// Pause stream inserts when not on page 1
watch(pageIndex, (idx) => {
    if (idx === 0) stream.resume()
    else stream.pause()
})

// Filter incoming stream kills by killlist type (client-side best-effort)
const matchesKillType = (kill: any): boolean => {
    switch (props.killlistType) {
        case 'solo': return kill.is_solo === true
        case 'npc': return kill.is_npc === true
        case 'big': return (kill.total_value ?? 0) >= 1_000_000_000
        case '5b': return (kill.total_value ?? 0) >= 5_000_000_000
        case '10b': return (kill.total_value ?? 0) >= 10_000_000_000
        case 'latest': return true
        default: return true // location/ship filters can't be checked client-side — allow and let next refresh correct
    }
}

// Client-side predicate for the quick filter — lets matching live kills keep
// flowing into a filtered view. Mirrors the server-side advanced filters; every
// field it reads is carried on the stream payload itself (see KilllistRow).
const killSecuritySpace = (kill: any): string => {
    const region = kill.region_id ?? 0
    if (region === 10000070) return 'pochven'
    if (region >= 12000001 && region <= 12000005) return 'abyssal'
    if (region >= 11000001 && region <= 11000033) return 'wspace'
    const sec = kill.solar_system_security
    if (sec == null) return ''
    if (sec >= 0.45) return 'highsec'
    if (sec > 0.0) return 'lowsec'
    return 'nullsec'
}
const matchesActiveFilter = (kill: any): boolean => {
    const af = activeFilters.value
    if (af.systemId) {
        if (kill.solar_system_id !== af.systemId) return false
    } else if (af.regionId && kill.region_id !== af.regionId) {
        return false
    }
    if (af.shipId && kill.ship_type_id !== af.shipId) return false
    if (af.soloOnly && kill.is_solo !== true) return false
    if (af.npcOnly && kill.is_npc !== true) return false
    if (af.iskValue) {
        const threshold = af.iskValue === '10b' ? 1e10 : 5e9
        if ((kill.total_value ?? 0) < threshold) return false
    }
    if (af.techLevel && !matchesMetaGroup(kill.meta_group_id, af.techLevel)) return false
    if (af.securityTypes.length && !af.securityTypes.includes(killSecuritySpace(kill))) return false
    return true
}

// Prepend live kills on page 1, trim to limit. Watch only the head of the
// stream — the array is mutated via unshift and holds up to 200 wide
// objects, so a deep watch re-traverses all of them on every incoming kill.
watch(() => stream.kills.value[0], (latest) => {
    if (pageIndex.value !== 0 || loading.value) return
    const topId = kills.value[0]?.killmail_id ?? 0
    if (!latest || latest.killmail_id <= topId) return
    if (!matchesKillType(latest)) return
    if (isFiltering.value && !matchesActiveFilter(latest)) return
    const updated = [latest as KillRow, ...kills.value]
    if (updated.length > limit.value) updated.length = limit.value
    kills.value = updated
})

// ── Live-updating timestamps ─────────────────────────────────────────────────

const now = ref(Date.now())
let tickTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
    tickTimer = setInterval(() => { now.value = Date.now() }, 30000)
})

onUnmounted(() => {
    if (tickTimer) clearInterval(tickTimer)
})

// ── Helpers ──────────────────────────────────────────────────────────────────

// Merge explicit lossEntities prop with domain entities (if in domain mode)
const { domainConfig, isDomainMode } = useDomainConfig()
const resolvedLossEntities = computed(() => {
    if (props.lossEntities) return props.lossEntities
    if (isDomainMode.value && domainConfig.value) return domainConfig.value.entityIds
    // Legacy single-entity fallback
    if (props.victimEntityType && props.victimEntityId) {
        const key = props.victimEntityType === 'character' ? 'characterIds'
            : props.victimEntityType === 'corporation' ? 'corporationIds'
                : props.victimEntityType === 'alliance' ? 'allianceIds'
                    : 'factionIds'
        return { [key]: [props.victimEntityId] }
    }
    return null
})

const isLoss = (kill: KillRow): boolean => isKilllistLoss(kill, resolvedLossEntities.value)

const warVictimSide = (kill: KillRow): 'aggressor' | 'defender' | null => {
    if (!props.warAggressorCorps && !props.warAggressorAlliances) return null
    const ac = new Set(props.warAggressorCorps || [])
    const aa = new Set(props.warAggressorAlliances || [])
    const dc = new Set(props.warDefenderCorps || [])
    const da = new Set(props.warDefenderAlliances || [])
    if (kill.victim_alliance_id && aa.has(kill.victim_alliance_id)) return 'aggressor'
    if (kill.victim_alliance_id && da.has(kill.victim_alliance_id)) return 'defender'
    if (kill.victim_corporation_id && ac.has(kill.victim_corporation_id)) return 'aggressor'
    if (kill.victim_corporation_id && dc.has(kill.victim_corporation_id)) return 'defender'
    return null
}

const warRowClass = (kill: KillRow): string => {
    const side = warVictimSide(kill)
    if (side === 'aggressor') return 'border-l-2 border-l-red-500/40 bg-red-500/[0.03]'
    if (side === 'defender') return 'border-l-2 border-l-blue-500/40 bg-blue-500/[0.03]'
    return ''
}

// Campaign side coloring — same visual language as war rows, but N sides.
const SIDE_ROW_CLASSES = [
    'border-l-2 border-l-red-500/40 bg-red-500/[0.03]',
    'border-l-2 border-l-blue-500/40 bg-blue-500/[0.03]',
    'border-l-2 border-l-emerald-500/40 bg-emerald-500/[0.03]',
    'border-l-2 border-l-orange-500/40 bg-orange-500/[0.03]',
]

// Victim → side lookup maps. Character beats corp beats alliance so a spy's
// pod dying counts for the side it was explicitly placed on.
const sideMaps = computed(() => {
    if (!props.sideEntities?.length) return null
    const chars = new Map<number, number>()
    const corps = new Map<number, number>()
    const allys = new Map<number, number>()
    props.sideEntities.forEach((s, i) => {
        s.characterIds?.forEach(id => chars.set(id, i))
        s.corporationIds?.forEach(id => corps.set(id, i))
        s.allianceIds?.forEach(id => allys.set(id, i))
    })
    return { chars, corps, allys }
})

const victimSideIndex = (kill: KillRow): number | undefined => {
    const m = sideMaps.value
    if (!m) return undefined
    if (kill.victim_character_id != null && m.chars.has(kill.victim_character_id)) return m.chars.get(kill.victim_character_id)
    if (kill.victim_corporation_id != null && m.corps.has(kill.victim_corporation_id)) return m.corps.get(kill.victim_corporation_id)
    if (kill.victim_alliance_id != null && m.allys.has(kill.victim_alliance_id)) return m.allys.get(kill.victim_alliance_id)
    return undefined
}

const sideRowClass = (kill: KillRow): string => {
    const idx = victimSideIndex(kill)
    if (idx === undefined) return ''
    // Corp-palette accent present → the inline style handles this row.
    if (props.sideColors?.[idx]) return ''
    return SIDE_ROW_CLASSES[idx] ?? ''
}

const hexToRgba = (hex: string, alpha: number): string => {
    const n = Number.parseInt(hex.replace('#', ''), 16)
    return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`
}

const sideRowStyle = (kill: KillRow): Record<string, string> | undefined => {
    const idx = victimSideIndex(kill)
    if (idx === undefined) return undefined
    const color = props.sideColors?.[idx]
    if (!color) return undefined
    return {
        borderLeft: `2px solid ${hexToRgba(color, 0.4)}`,
        backgroundColor: hexToRgba(color, 0.04),
    }
}

// English ordinal suffix — 1 → "1st", 2 → "2nd", 3 → "3rd", 21 → "21st", …
const ordinal = (n: number): string => {
    const v = n % 100
    if (v >= 11 && v <= 13) return `${n}th`
    switch (n % 10) {
        case 1: return `${n}st`
        case 2: return `${n}nd`
        case 3: return `${n}rd`
        default: return `${n}th`
    }
}

// Kill time — relative "Xm ago" within the last 30 minutes, otherwise
// the EVE-time clock (UTC HH:MM). Day context comes from the header and
// dividers, so the per-row slot only carries the hour/minute.
const killTime = (dateStr: string): string => {
    const diff = now.value - new Date(dateStr).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 30) return `${mins}m ago`
    return new Date(dateStr).toISOString().slice(11, 16)
}

// Friendly day label — e.g. "Today (15th April)" / "Yesterday (14th April)" /
// "13th April" (adds year only when it differs from the current UTC year).
const formatDayLabel = (utcDate: string): string => {
    const todayKey = new Date(now.value).toISOString().slice(0, 10)
    const yestKey = new Date(now.value - 86400000).toISOString().slice(0, 10)
    const d = new Date(`${utcDate}T00:00:00Z`)
    const day = d.getUTCDate()
    const month = d.toLocaleDateString('en-GB', { month: 'long', timeZone: 'UTC' })
    const currentYear = new Date(now.value).getUTCFullYear()
    const year = d.getUTCFullYear()
    const dateLabel = year === currentYear
        ? `${ordinal(day)} ${month}`
        : `${ordinal(day)} ${month} ${year}`
    if (utcDate === todayKey) return `Today (${dateLabel})`
    if (utcDate === yestKey) return `Yesterday (${dateLabel})`
    return dateLabel
}

// UTC YYYY-MM-DD — stable grouping key for date dividers.
const utcDateKey = (dateStr: string): string => dateStr.slice(0, 10)

// Interleaved rows: kill rows and date dividers at UTC day boundaries.
type RowItem =
    | { type: 'kill'; kill: KillRow; visualIdx: number }
    | { type: 'divider'; utcDate: string; label: string }

const rows = computed<RowItem[]>(() => {
    const out: RowItem[] = []
    let lastDate = ''
    let visualIdx = 0
    for (const kill of kills.value) {
        const d = utcDateKey(kill.killmail_time)
        if (d !== lastDate) {
            if (lastDate !== '') {
                out.push({ type: 'divider', utcDate: d, label: formatDayLabel(d) })
            }
            lastDate = d
        }
        out.push({ type: 'kill', kill, visualIdx })
        visualIdx++
    }
    return out
})

// Header date label — the UTC day of the newest kill on this page.
const headerDateLabel = computed(() => {
    const first = kills.value[0]
    return first ? formatDayLabel(utcDateKey(first.killmail_time)) : ''
})

const secColor = (sec: number | null): string => {
    if (sec === null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-blue-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

const secLabel = (sec: number | null): string => {
    if (sec === null) return '?'
    return sec.toFixed(1)
}
</script>

<template>
    <div class="rounded-lg bg-white/[0.04] border border-white/[0.08] p-2">
        <!-- Controls bar. Three equal columns so the centered date stays
             pinned to the viewport center even as the left (limit selector)
             or right (pagination) side changes width. -->
        <div class="grid grid-cols-3 items-center gap-2 mb-2">
            <div class="flex items-center gap-2 min-w-0">
                <!-- WebSocket status dot (only when stream is active) -->
                <div
                    v-if="liveActive"
                    class="w-2 h-2 rounded-full flex-shrink-0"
                    :class="stream.connected.value ? 'bg-green-500' : 'bg-red-500'"
                    v-tooltip="stream.connected.value ? 'Live — connected' : 'Disconnected'"
                ></div>

                <!-- New kills badge (when paused / not on page 1) -->
                <button
                    v-if="liveActive && stream.newCount.value > 0"
                    class="px-2 py-0.5 rounded-full text-fine font-medium bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 transition-colors"
                    @click="cursors = [null]; pageIndex = 0; stream.resume()"
                >
                    +{{ stream.newCount.value }} new
                </button>

                <!-- Limit selector -->
                <div class="flex items-center gap-1">
                    <button
                        v-for="opt in limitOptions"
                        :key="opt"
                        class="px-2 py-1 rounded text-xs font-medium transition-colors"
                        :class="limit === opt ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                        @click="setLimit(opt)"
                    >{{ opt }}</button>
                </div>

                <!-- Quick-filter toggle (opt-in) -->
                <button
                    v-if="props.quickFilter"
                    class="flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors"
                    :class="(quickFilterOpen || hasActiveFilter) ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    v-tooltip="quickFilterOpen ? 'Hide filters' : 'Filter killlist'"
                    @click="quickFilterOpen = !quickFilterOpen"
                >
                    <Icon name="lucide:list-filter" class="text-sm" />
                    <span v-if="activeFilterCount > 0" class="px-1 rounded-full bg-blue-500/30 text-blue-300 text-fine tabular-nums">{{ activeFilterCount }}</span>
                </button>
            </div>
            <div v-if="headerDateLabel" class="text-center text-xs text-gray-300 font-medium tabular-nums truncate" data-allow-mismatch="text">{{ headerDateLabel }}</div>
            <div v-else></div>
            <div class="flex items-center gap-0.5 text-xs justify-self-end">
                <!-- Cursor mode — minimal prev / page N / next. The cursor
                     backend can't random-jump, so we don't pretend otherwise. -->
                <template v-if="!isNumberedMode">
                    <button aria-label="Previous page" class="w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed" :disabled="pageIndex <= 0" @click="goPrev">
                        <Icon name="lucide:chevron-left" class="text-sm" />
                    </button>
                    <div class="px-2 text-xs text-gray-300 tabular-nums select-none flex items-center">
                        <template v-if="loading">
                            <span class="killlist-dot"></span>
                            <span class="killlist-dot"></span>
                            <span class="killlist-dot"></span>
                        </template>
                        <template v-else>Page {{ currentPage }}</template>
                    </div>
                    <button aria-label="Next page" class="w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed" :disabled="!hasMore" @click="goNext">
                        <Icon name="lucide:chevron-right" class="text-sm" />
                    </button>
                </template>
                <!-- Numbered mode — full page list, requires counting-capable backend. -->
                <template v-else>
                    <button aria-label="Previous page" class="w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed" :disabled="pageIndex <= 0" @click="goPrev">
                        <Icon name="lucide:chevron-left" class="text-sm" />
                    </button>
                    <template v-for="(p, idx) in visiblePages" :key="idx">
                        <span v-if="p === '...'" class="w-7 h-7 flex items-center justify-center text-gray-600">…</span>
                        <button
                            v-else
                            class="h-7 min-w-7 px-2 flex items-center justify-center rounded-md font-medium tabular-nums transition-colors"
                            :class="p === currentPage ? 'bg-blue-500/15 text-blue-400' : 'text-gray-500 hover:bg-blue-500/[0.06] hover:text-blue-400'"
                            @click="goToPage(p - 1)"
                        >{{ formatPageLabel(p) }}</button>
                    </template>
                    <button aria-label="Next page" class="w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed" :disabled="currentPage >= (effectiveTotalPages ?? 0)" @click="goNext">
                        <Icon name="lucide:chevron-right" class="text-sm" />
                    </button>
                </template>
            </div>
        </div>

        <!-- Quick-filter pop-down panel -->
        <KilllistQuickFilter
            v-if="props.quickFilter && quickFilterOpen"
            v-model="activeFilters"
        />

        <!-- Loading skeleton -->
        <div v-if="loading" class="space-y-px">
            <div v-for="i in limit" :key="i" class="flex items-center gap-3 px-2 py-2">
                <div class="w-10 h-10 rounded bg-white/[0.04] animate-pulse flex-shrink-0"></div>
                <div class="flex-1 space-y-1.5">
                    <div class="h-3 w-28 rounded bg-white/[0.04] animate-pulse"></div>
                    <div class="h-2 w-20 rounded bg-white/[0.03] animate-pulse"></div>
                </div>
            </div>
        </div>

        <!-- Empty -->
        <div v-else-if="kills.length === 0" class="py-12 text-center text-xs text-gray-600">No kills found</div>

        <!-- Kill rows -->
        <div v-else>
            <!-- Desktop header -->
            <div class="hidden md:grid grid-cols-[minmax(0,2fr)_minmax(0,2.5fr)_minmax(0,2.5fr)_minmax(0,1.2fr)_minmax(0,1fr)] gap-3 px-2 py-1.5 text-fine font-bold uppercase tracking-wider text-gray-600 border-b border-white/[0.08]">
                <div>Ship</div>
                <div>Victim</div>
                <div>Final Blow</div>
                <div>Location</div>
                <div class="text-right">Details</div>
            </div>

            <!-- Desktop rows — day divider separates kills across UTC date boundaries.
                 isDesktop unmounts the inactive variant post-hydration (see useIsDesktop);
                 the hidden/md: classes still handle SSR and pre-hydration. -->
            <template v-for="(row, rIdx) in (isDesktop !== false ? rows : [])" :key="row.type === 'kill' ? row.kill.killmail_id : `d-${row.utcDate}-${rIdx}`">
                <div
                    v-if="row.type === 'divider'"
                    class="hidden md:flex items-center gap-3 px-2 py-2 text-fine uppercase tracking-wider text-gray-400"
                >
                    <div class="flex-1 h-px bg-white/[0.08]"></div>
                    <span class="tabular-nums">{{ row.label }}</span>
                    <div class="flex-1 h-px bg-white/[0.08]"></div>
                </div>
                <div
                    v-else
                    class="relative hidden md:grid grid-cols-[minmax(0,2fr)_minmax(0,2.5fr)_minmax(0,2.5fr)_minmax(0,1.2fr)_minmax(0,1fr)] gap-3 px-2 py-1.5 items-center border-b border-white/[0.03] hover:bg-blue-500/[0.06] transition-colors group"
                    :class="sideRowClass(row.kill) || (isLoss(row.kill) ? 'loss-row loss-row-hover' : warRowClass(row.kill) || (row.visualIdx % 2 === 0 ? 'bg-white/[0.02]' : ''))"
                    :style="sideRowStyle(row.kill)"
                >
                    <!-- Stretched row link — covers whitespace so clicks anywhere that
                         miss an inner entity link navigate to the killmail. Inner
                         NuxtLinks use relative+z-10 to render above and capture clicks. -->
                    <NuxtLink
                        :to="`/kill/${row.kill.killmail_id}`"
                        class="absolute inset-0 z-0"
                        :aria-label="`Killmail: ${row.kill.ship_name || 'ship'} — ${formatIsk(row.kill.total_value)} ISK`"
                    />

                    <!-- Ship -->
                    <div class="relative z-10 flex items-center gap-2.5 min-w-0 pointer-events-none [&_a]:pointer-events-auto">
                        <NuxtLink v-if="row.kill.ship_type_id" :to="`/kill/${row.kill.killmail_id}`" class="flex-shrink-0 w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
                            <EveImage :src="`/images/types/${row.kill.ship_type_id}/icon?size=64`" :alt="row.kill.ship_name || ''" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                        </NuxtLink>
                        <div v-else class="flex-shrink-0 w-10 h-10 rounded bg-white/[0.04] flex items-center justify-center">
                            <Icon name="lucide:box" class="text-sm text-gray-500" />
                        </div>
                        <div class="min-w-0">
                            <NuxtLink v-if="row.kill.ship_type_id" :to="`/item/${row.kill.ship_type_id}`" class="block w-fit max-w-full text-xs text-gray-300 group-hover:text-blue-400 hover:text-blue-400 truncate">{{ row.kill.ship_name || 'Unknown' }}</NuxtLink>
                            <div v-else class="text-xs text-gray-300 truncate">{{ row.kill.ship_name || 'Unknown' }}</div>
                            <NuxtLink v-if="row.kill.ship_market_path && row.kill.ship_group_name" :to="row.kill.ship_market_path" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ row.kill.ship_group_name }}</NuxtLink>
                            <div v-else class="text-fine text-gray-400 truncate">{{ row.kill.ship_group_name }}</div>
                            <div class="text-fine text-isk/70 tabular-nums">{{ formatIsk(row.kill.total_value) }} ISK</div>
                        </div>
                    </div>

                    <!-- Victim -->
                    <div class="relative z-10 flex items-center gap-2 min-w-0 pointer-events-none [&_a]:pointer-events-auto">
                        <NuxtLink v-if="row.kill.victim_character_id" :to="`/character/${row.kill.victim_character_id}`" class="flex-shrink-0 w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
                            <EveImage :src="`/images/characters/${row.kill.victim_character_id}/portrait?size=64`" :alt="row.kill.victim_character_name || ''" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                        </NuxtLink>
                        <div v-else class="flex-shrink-0 w-10 h-10 rounded bg-white/[0.04] flex items-center justify-center">
                            <Icon name="lucide:building" class="text-sm text-gray-500" />
                        </div>
                        <div v-if="row.kill.victim_corporation_id || row.kill.victim_alliance_id" class="flex-shrink-0 flex flex-col gap-[2px]">
                            <NuxtLink v-if="row.kill.victim_corporation_id" :to="`/corporation/${row.kill.victim_corporation_id}`" class="w-[19px] h-[19px] rounded-sm overflow-hidden bg-white/[0.04]">
                                <EveImage :src="`/images/corporations/${row.kill.victim_corporation_id}/logo?size=32`" :alt="row.kill.victim_corporation_name || ''" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                            </NuxtLink>
                            <NuxtLink v-if="row.kill.victim_alliance_id" :to="`/alliance/${row.kill.victim_alliance_id}`" class="w-[19px] h-[19px] rounded-sm overflow-hidden bg-white/[0.04]">
                                <EveImage :src="`/images/alliances/${row.kill.victim_alliance_id}/logo?size=32`" :alt="row.kill.victim_alliance_name || ''" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                            </NuxtLink>
                        </div>
                        <div class="min-w-0">
                            <NuxtLink v-if="row.kill.victim_character_id" :to="`/character/${row.kill.victim_character_id}`" class="block w-fit max-w-full text-xs text-gray-300 hover:text-blue-400 truncate">{{ row.kill.victim_character_name || 'Unknown' }}</NuxtLink>
                            <NuxtLink v-else-if="row.kill.victim_corporation_id" :to="`/corporation/${row.kill.victim_corporation_id}`" class="block w-fit max-w-full text-xs text-gray-300 hover:text-blue-400 truncate">{{ row.kill.victim_corporation_name || 'Unknown' }}</NuxtLink>
                            <div v-else class="text-xs text-gray-300 truncate">Unknown</div>
                            <NuxtLink v-if="row.kill.victim_character_name && row.kill.victim_corporation_id" :to="`/corporation/${row.kill.victim_corporation_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ row.kill.victim_corporation_name }}</NuxtLink>
                            <NuxtLink v-if="row.kill.victim_alliance_id" :to="`/alliance/${row.kill.victim_alliance_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ row.kill.victim_alliance_name }}</NuxtLink>
                        </div>
                    </div>

                    <!-- Final Blow -->
                    <div class="relative z-10 flex items-center gap-2 min-w-0 pointer-events-none [&_a]:pointer-events-auto">
                        <NuxtLink v-if="row.kill.final_blow_character_id" :to="`/character/${row.kill.final_blow_character_id}`" class="flex-shrink-0 w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
                            <EveImage :src="`/images/characters/${row.kill.final_blow_character_id}/portrait?size=64`" :alt="row.kill.final_blow_character_name || ''" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                        </NuxtLink>
                        <NuxtLink v-else-if="row.kill.final_blow_ship_type_id" :to="`/item/${row.kill.final_blow_ship_type_id}`" class="flex-shrink-0 w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
                            <EveImage :src="`/images/types/${row.kill.final_blow_ship_type_id}/icon?size=64`" :alt="row.kill.final_blow_ship_name || 'NPC'" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                        </NuxtLink>
                        <div v-else class="flex-shrink-0 w-10 h-10 rounded bg-white/[0.04] flex items-center justify-center">
                            <Icon name="lucide:crosshair" class="text-sm text-gray-500" />
                        </div>
                        <div v-if="row.kill.final_blow_corporation_id || row.kill.final_blow_alliance_id" class="flex-shrink-0 flex flex-col gap-[2px]">
                            <NuxtLink v-if="row.kill.final_blow_corporation_id" :to="`/corporation/${row.kill.final_blow_corporation_id}`" class="w-[19px] h-[19px] rounded-sm overflow-hidden bg-white/[0.04]">
                                <EveImage :src="`/images/corporations/${row.kill.final_blow_corporation_id}/logo?size=32`" :alt="row.kill.final_blow_corporation_name || ''" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                            </NuxtLink>
                            <NuxtLink v-if="row.kill.final_blow_alliance_id" :to="`/alliance/${row.kill.final_blow_alliance_id}`" class="w-[19px] h-[19px] rounded-sm overflow-hidden bg-white/[0.04]">
                                <EveImage :src="`/images/alliances/${row.kill.final_blow_alliance_id}/logo?size=32`" :alt="row.kill.final_blow_alliance_name || ''" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                            </NuxtLink>
                        </div>
                        <div class="min-w-0">
                            <NuxtLink v-if="row.kill.final_blow_character_id" :to="`/character/${row.kill.final_blow_character_id}`" class="block w-fit max-w-full text-xs text-gray-300 hover:text-blue-400 truncate">{{ row.kill.final_blow_character_name || 'Unknown' }}</NuxtLink>
                            <NuxtLink v-else-if="row.kill.final_blow_corporation_id" :to="`/corporation/${row.kill.final_blow_corporation_id}`" class="block w-fit max-w-full text-xs text-gray-300 hover:text-blue-400 truncate">{{ row.kill.final_blow_corporation_name || 'Unknown' }}</NuxtLink>
                            <NuxtLink v-else-if="row.kill.final_blow_ship_type_id" :to="`/item/${row.kill.final_blow_ship_type_id}`" class="block w-fit max-w-full text-xs text-npc/80 hover:text-blue-400 truncate">{{ row.kill.final_blow_ship_name || 'NPC' }}</NuxtLink>
                            <div v-else class="text-xs text-npc/80 truncate">NPC</div>
                            <NuxtLink v-if="row.kill.final_blow_character_name && row.kill.final_blow_corporation_id" :to="`/corporation/${row.kill.final_blow_corporation_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ row.kill.final_blow_corporation_name }}</NuxtLink>
                            <NuxtLink v-if="row.kill.final_blow_alliance_id" :to="`/alliance/${row.kill.final_blow_alliance_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ row.kill.final_blow_alliance_name }}</NuxtLink>
                        </div>
                    </div>

                    <!-- Location -->
                    <div class="relative z-10 flex items-center gap-2 min-w-0 pointer-events-none [&_a]:pointer-events-auto">
                        <NuxtLink :to="`/system/${row.kill.solar_system_id}`" class="flex-shrink-0">
                            <EveImage :src="`/images/systems/${row.kill.solar_system_id}?size=32`" alt="" class="w-6 h-6 rounded" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                        </NuxtLink>
                        <div class="min-w-0">
                            <div class="text-fine truncate">
                                <span :class="secColor(row.kill.solar_system_security)" class="font-medium tabular-nums">{{ secLabel(row.kill.solar_system_security) }}</span>
                                <NuxtLink :to="`/system/${row.kill.solar_system_id}`" class="text-gray-300 hover:text-blue-400 ml-1" :class="pochvenClass(row.kill.region_id)">{{ row.kill.solar_system_name }}</NuxtLink>
                            </div>
                            <NuxtLink v-if="row.kill.region_id" :to="`/region/${row.kill.region_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate" :class="pochvenClass(row.kill.region_id)">{{ row.kill.region_name }}</NuxtLink>
                            <div v-else class="text-fine text-gray-400 truncate">{{ row.kill.region_name }}</div>
                        </div>
                    </div>

                    <!-- Details -->
                    <div class="relative z-10 text-right min-w-0 pointer-events-none [&_a]:pointer-events-auto">
                        <div
                            class="text-xs text-gray-300 truncate tabular-nums"
                            data-allow-mismatch="text"
                        >{{ killTime(row.kill.killmail_time) }}</div>
                        <div class="flex items-center justify-end gap-1 text-fine text-gray-400">
                            <Icon name="lucide:users" class="text-fine" />
                            <span class="tabular-nums">{{ row.kill.attacker_count }}</span>
                        </div>
                    </div>
                </div>
            </template>

            <!-- Mobile rows — day divider separates kills across UTC date boundaries -->
            <template v-for="(row, rIdx) in (isDesktop !== true ? rows : [])" :key="row.type === 'kill' ? `m-${row.kill.killmail_id}` : `md-${row.utcDate}-${rIdx}`">
                <div
                    v-if="row.type === 'divider'"
                    class="md:hidden flex items-center gap-2 px-2 py-1.5 text-fine uppercase tracking-wider text-gray-400"
                >
                    <div class="flex-1 h-px bg-white/[0.08]"></div>
                    <span class="tabular-nums">{{ row.label }}</span>
                    <div class="flex-1 h-px bg-white/[0.08]"></div>
                </div>
                <NuxtLink
                    v-else
                    :to="`/kill/${row.kill.killmail_id}`"
                    class="md:hidden flex gap-2.5 px-2 py-2 border-b border-white/[0.03] hover:bg-blue-500/[0.06] transition-colors"
                    :class="sideRowClass(row.kill) || (isLoss(row.kill) ? 'loss-row loss-row-hover' : row.visualIdx % 2 === 0 ? 'bg-white/[0.02]' : '')"
                    :style="sideRowStyle(row.kill)"
                >
                    <!-- Images stack -->
                    <div class="flex-shrink-0 flex flex-col gap-1">
                        <div v-if="row.kill.ship_type_id" class="w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
                            <EveImage :src="`/images/types/${row.kill.ship_type_id}/icon?size=64`" :alt="row.kill.ship_name || ''" class="w-full h-full object-cover" :loading="row.visualIdx < 20 ? 'eager' : 'lazy'" />
                        </div>
                        <div v-else class="w-10 h-10 rounded bg-white/[0.04] flex items-center justify-center">
                            <Icon name="lucide:box" class="text-sm text-gray-500" />
                        </div>
                    </div>

                    <!-- Info -->
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center justify-between gap-2 mb-0.5">
                            <span class="text-xs text-gray-200 font-medium truncate">{{ row.kill.ship_name || 'Unknown' }}</span>
                            <span class="flex-shrink-0 text-fine text-isk/80 tabular-nums">{{ formatIsk(row.kill.total_value) }}</span>
                        </div>
                        <div class="text-fine text-gray-400 truncate">
                            {{ row.kill.victim_character_name || row.kill.victim_corporation_name || 'Unknown' }}
                            <span v-if="row.kill.victim_character_name && row.kill.victim_corporation_name" class="text-gray-500"> &middot; {{ row.kill.victim_corporation_name }}</span>
                        </div>
                        <div class="flex items-center justify-between mt-0.5">
                            <span class="text-fine">
                                <span :class="secColor(row.kill.solar_system_security)" class="tabular-nums">{{ secLabel(row.kill.solar_system_security) }}</span>
                                <span class="text-gray-400 ml-0.5" :class="pochvenClass(row.kill.region_id)">{{ row.kill.solar_system_name }}</span>
                            </span>
                            <span class="text-fine text-gray-400 tabular-nums" data-allow-mismatch="text">{{ killTime(row.kill.killmail_time) }}</span>
                        </div>
                    </div>
                </NuxtLink>
            </template>
        </div>

        <!-- Bottom pagination -->
        <div v-if="kills.length > 0" class="flex items-center justify-end gap-0.5 mt-2 text-xs">
            <template v-if="!isNumberedMode">
                <button aria-label="Previous page" class="w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed" :disabled="pageIndex <= 0" @click="goPrev">
                    <Icon name="lucide:chevron-left" class="text-sm" />
                </button>
                <div class="px-2 text-xs text-gray-300 tabular-nums select-none">Page {{ currentPage }}</div>
                <button aria-label="Next page" class="w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed" :disabled="!hasMore" @click="goNext">
                    <Icon name="lucide:chevron-right" class="text-sm" />
                </button>
            </template>
            <template v-else>
                <button aria-label="Previous page" class="w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed" :disabled="pageIndex <= 0" @click="goPrev">
                    <Icon name="lucide:chevron-left" class="text-sm" />
                </button>
                <template v-for="(p, idx) in visiblePages" :key="`b-${idx}`">
                    <span v-if="p === '...'" class="w-7 h-7 flex items-center justify-center text-gray-600">…</span>
                    <button
                        v-else
                        class="h-7 min-w-7 px-2 flex items-center justify-center rounded-md font-medium tabular-nums transition-colors"
                        :class="p === currentPage ? 'bg-blue-500/15 text-blue-400' : 'text-gray-500 hover:bg-blue-500/[0.06] hover:text-blue-400'"
                        @click="goToPage(p - 1)"
                    >{{ formatPageLabel(p) }}</button>
                </template>
                <button aria-label="Next page" class="w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed" :disabled="currentPage >= (effectiveTotalPages ?? 0)" @click="goNext">
                    <Icon name="lucide:chevron-right" class="text-sm" />
                </button>
            </template>
        </div>
    </div>
</template>

<style scoped>
/* Three-dot loader that replaces the "Page N" readout while the killlist
   fetches. Bounces with a staggered delay so it reads left-to-right. */
.killlist-dot {
    display: inline-block;
    width: 4px;
    height: 4px;
    margin: 0 2px;
    background-color: currentColor;
    border-radius: 50%;
    opacity: 0.8;
    animation: killlist-dot-bounce 1.1s infinite ease-in-out both;
}
.killlist-dot:nth-child(1) { animation-delay: 0s; }
.killlist-dot:nth-child(2) { animation-delay: 0.15s; }
.killlist-dot:nth-child(3) { animation-delay: 0.3s; }

@keyframes killlist-dot-bounce {
    0%, 80%, 100% { transform: translateY(0); opacity: 0.35; }
    40% { transform: translateY(-4px); opacity: 1; }
}
</style>
