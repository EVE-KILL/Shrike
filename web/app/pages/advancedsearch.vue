<script setup lang="ts">
const { track } = useAnalytics()

interface SearchHit {
    id: string
    name: string
    ticker: string | null
    type: 'character' | 'corporation' | 'alliance' | 'faction' | 'system' | 'region' | 'constellation' | 'ship' | 'item'
}

interface SearchResponse {
    hits: SearchHit[]
    query: string
    processingTimeMs: number
    total: number
    entityCounts: Record<string, number>
}

interface Entity {
    id: number
    type: string
    name: string
    role: 'victim' | 'attacker' | 'both'
    exclude?: boolean
}

type ItemSlot = 'fitted' | 'cargo' | 'any'
type ItemSide = 'victim' | 'attacker' | 'either'

interface ItemFilter {
    // Exactly one of typeId / groupId is set. typeId = specific item
    // (e.g. "Stasis Webifier II"). groupId = whole inv_group (e.g.
    // "Propulsion Module" = all MWDs + ABs + meta variants).
    typeId?: number
    groupId?: number
    name: string
    // Slot is victim-side only; for side='attacker' the backend ignores it
    // because attacker-side matches only weapon_type_id (damage-dealing).
    slot: ItemSlot
    side: ItemSide
}

interface Filters {
    entities: { victim: Entity[]; attacker: Entity[]; both: Entity[] }
    items: ItemFilter[]
    location: {
        securityTypes: string[]
        systemId: number | null
        systemName: string | null
        regionId: number | null
        regionName: string | null
        constellationId: number | null
        constellationName: string | null
    }
    timeRange: { preset: string | null } | { from: string; to: string }
    attackerCount: string | null
    attackerType: string | null
    iskValue: string | null
    iskMin: string
    iskMax: string
    shipCategory: string | null
    techLevel: string | null
    sort: { field: string; direction: 'asc' | 'desc' }
}

useHead({ title: 'Advanced Search' })
useSeoMeta({
    description: 'Advanced killmail search on EVE-KILL — filter by pilots, ships, location, ISK value, security status, time range, and more.',
    ogTitle: 'Advanced Search — EVE-KILL',
    ogDescription: 'Search EVE Online killmails with advanced filters — pilots, ships, locations, ISK value, and more.',
})

const route = useRoute()
const router = useRouter()

// Search
const searchQuery = ref('')
const searchResults = ref<SearchHit[]>([])
const isSearching = ref(false)
const showDropdown = ref(false)
let searchTimeout: ReturnType<typeof setTimeout>

const handleSearch = () => {
    const q = searchQuery.value.trim()
    if (!q || q.length < 3) {
        searchResults.value = []
        showDropdown.value = false
        return
    }
    clearTimeout(searchTimeout)
    searchTimeout = setTimeout(async () => {
        isSearching.value = true
        try {
            // Fetch in parallel: the default search (entities/items/ships/
            // locations) + a separate type=group query for inv_groups, which
            // the default search doesn't include to avoid polluting generic
            // results. Both sets are merged into one dropdown so users can
            // pick an item or a whole module group in one place.
            const [main, groups] = await Promise.all([
                apiFetch<SearchResponse>('/api/search', { params: { q } }),
                apiFetch<SearchResponse>('/api/search', { params: { q, type: 'group' } }),
            ])
            searchResults.value = [...(main.hits || []), ...(groups.hits || [])]
            showDropdown.value = searchResults.value.length > 0
        } catch {
            searchResults.value = []
        } finally {
            isSearching.value = false
        }
    }, 300)
}

// Filters
const defaultFilters = (): Filters => ({
    entities: { victim: [], attacker: [], both: [] },
    items: [],
    location: {
        securityTypes: [],
        systemId: null, systemName: null,
        regionId: null, regionName: null,
        constellationId: null, constellationName: null,
    },
    timeRange: { preset: '30d' },
    attackerCount: null,
    attackerType: null,
    iskValue: null,
    iskMin: '',
    iskMax: '',
    shipCategory: null,
    techLevel: null,
    sort: { field: 'killmail_time', direction: 'desc' },
})

const filters = ref<Filters>(defaultFilters())

const hasFilters = computed(() => {
    const f = filters.value
    return f.entities.victim.length > 0
        || f.entities.attacker.length > 0
        || f.entities.both.length > 0
        || f.items.length > 0
        || f.location.securityTypes.length > 0
        || f.location.systemId != null
        || f.location.regionId != null
        || f.location.constellationId != null
        || f.attackerCount != null
        || f.attackerType != null
        || f.iskValue != null
        || f.iskMin !== ''
        || f.iskMax !== ''
        || f.shipCategory != null
        || f.techLevel != null
})

// Serialize filters for the API (strips display-only fields)
const serializedFilters = computed(() => {
    const f = filters.value
    const out: Record<string, any> = {}

    const entities: Record<string, any> = {}
    const mapEntity = (e: Entity) => ({ id: e.id, type: e.type, name: e.name, ...(e.exclude ? { exclude: true } : {}) })
    if (f.entities.victim.length) entities.victim = f.entities.victim.map(mapEntity)
    if (f.entities.attacker.length) entities.attacker = f.entities.attacker.map(mapEntity)
    if (f.entities.both.length) entities.both = f.entities.both.map(mapEntity)
    if (Object.keys(entities).length) out.entities = entities

    if (f.items.length) out.items = f.items.map(i => ({
        typeId: i.typeId,
        groupId: i.groupId,
        name: i.name,
        slot: i.slot,
        side: i.side,
    }))

    const loc: Record<string, any> = {}
    if (f.location.securityTypes.length) loc.securityTypes = f.location.securityTypes
    if (f.location.systemId) loc.systemId = f.location.systemId
    if (f.location.regionId) loc.regionId = f.location.regionId
    if (f.location.constellationId) loc.constellationId = f.location.constellationId
    if (Object.keys(loc).length) out.location = loc

    out.timeRange = f.timeRange

    if (f.attackerCount) out.attackerCount = f.attackerCount
    if (f.attackerType) out.attackerType = f.attackerType

    // ISK: custom range takes precedence over preset
    const iskMinNum = f.iskMin ? parseIsk(f.iskMin) : null
    const iskMaxNum = f.iskMax ? parseIsk(f.iskMax) : null
    if (iskMinNum != null || iskMaxNum != null) {
        if (iskMinNum != null) out.iskMin = iskMinNum
        if (iskMaxNum != null) out.iskMax = iskMaxNum
    } else if (f.iskValue) {
        out.iskValue = f.iskValue
    }

    if (f.shipCategory) out.shipCategory = f.shipCategory
    if (f.techLevel) out.techLevel = f.techLevel

    if (f.sort.field !== 'killmail_time' || f.sort.direction !== 'desc') {
        out.sort = f.sort
    }

    return JSON.stringify(out)
})

// Custom time range inputs
const customFrom = ref('')
const customTo = ref('')

// Collapsible filter panel
const filtersCollapsed = ref(false)

// Copy URL
const copyStatus = ref<'idle' | 'copied'>('idle')

const copyUrl = async () => {
    try {
        await navigator.clipboard.writeText(window.location.href)
        copyStatus.value = 'copied'
        setTimeout(() => { copyStatus.value = 'idle' }, 2000)
    } catch { /* clipboard not available */ }
}

// Saved searches
interface SavedSearch {
    name: string
    filters: string
    date: string
}

const savedSearches = ref<SavedSearch[]>([])
const showSaveDialog = ref(false)
const saveName = ref('')
const showSavedList = ref(false)

const loadSavedSearches = () => {
    if (!import.meta.client) return
    try {
        const stored = localStorage.getItem('evekill-saved-searches')
        if (stored) savedSearches.value = JSON.parse(stored)
    } catch { /* ignore */ }
}

const persistSavedSearches = () => {
    if (!import.meta.client) return
    try {
        localStorage.setItem('evekill-saved-searches', JSON.stringify(savedSearches.value))
    } catch { /* ignore */ }
}

const saveCurrentSearch = () => {
    if (!saveName.value.trim()) return
    savedSearches.value.push({
        name: saveName.value.trim(),
        filters: filtersForUrl.value || '{}',
        date: new Date().toISOString(),
    })
    persistSavedSearches()
    saveName.value = ''
    showSaveDialog.value = false
}

const loadSavedSearch = (search: SavedSearch) => {
    const url = `/advancedsearch?q=${encodeURIComponent(search.filters)}`
    navigateTo(url)
    // Restore from the saved data directly
    try {
        const parsed = JSON.parse(search.filters)
        applyParsedFilters(parsed)
    } catch { /* ignore */ }
    showSavedList.value = false
}

const deleteSavedSearch = (index: number) => {
    savedSearches.value.splice(index, 1)
    persistSavedSearches()
}

// Results
const showResults = ref(false)

// View mode: kills (standard KillList) or fits (FitPreview cards)
type ViewMode = 'kills' | 'fits'
const viewMode = ref<ViewMode>('kills')

// Fits-specific state
interface FitEntry {
    killmail_id: number
    killmail_time: string
    victim_ship_type_id: number
    victim_ship_name: string
    total_value: number
    attacker_count: number
    modules: { slot_group: number; type_id: number; name: string; charge_type_id: number | null }[]
    drones: { type_id: number; name: string; quantity: number }[]
    count?: number
    hash?: string
    dedup_mode?: string
}
const fitsData = ref<FitEntry[]>([])
const fitsLoading = ref(false)
const fitsHasMore = ref(false)
const fitsCursor = ref<number | null>(null)

// Fits grouping: family (default — collapses T2/meta/faction variants),
// exact (identical module sets only), or none (every kill as its own card).
type DedupMode = 'none' | 'exact' | 'family'
const fitsDedup = ref<DedupMode>('family')

// Re-fetch when dedup mode changes
watch(fitsDedup, () => {
    if (showResults.value && viewMode.value === 'fits') fetchFits()
})

// Fit drill-down: show all kills for a specific fit hash
const drilldownHash = ref<string | null>(null)
const drilldownMode = ref<'exact' | 'family' | null>(null)
const drilldownShipName = ref('')

const handleShowKills = (hash: string, dedupMode: string, shipName: string) => {
    drilldownHash.value = hash
    drilldownMode.value = dedupMode as 'exact' | 'family'
    drilldownShipName.value = shipName
    viewMode.value = 'kills'
    appliedKillListParams.value = killListParams.value
}

const clearDrilldown = () => {
    drilldownHash.value = null
    drilldownMode.value = null
    drilldownShipName.value = ''
    viewMode.value = 'fits'
}

const killListParams = computed(() => {
    const params: Record<string, any> = { filters: serializedFilters.value }
    if (drilldownHash.value && drilldownMode.value) {
        if (drilldownMode.value === 'family') {
            params.familyHash = drilldownHash.value
        } else {
            params.fitHash = drilldownHash.value
        }
    }
    return params
})

// What KillList is actually querying with. Updated by the debounced watcher
// below rather than tracking killListParams directly: KillList refetches
// whenever its params change, so this ref is where the debounce has to bite.
const appliedKillListParams = ref<Record<string, any>>(killListParams.value)

const fetchFits = async (append = false) => {
    fitsLoading.value = true
    try {
        const params: Record<string, string> = {
            filters: serializedFilters.value,
            view: 'fits',
            limit: '50',
        }
        if (fitsDedup.value !== 'none') params.dedup = fitsDedup.value
        if (append && fitsCursor.value) params.after = String(fitsCursor.value)
        const res = await apiFetch<{ fits: FitEntry[]; hasMore: boolean; cursor: number | null }>(
            '/api/killlist/advanced', { params },
        )
        if (append) {
            fitsData.value = [...fitsData.value, ...res.fits]
        } else {
            fitsData.value = res.fits
        }
        fitsHasMore.value = res.hasMore
        fitsCursor.value = res.cursor
    } catch {
        fitsData.value = []
        fitsHasMore.value = false
    } finally {
        fitsLoading.value = false
    }
}

// Serialize filter state for URL (includes display names)
const filtersForUrl = computed(() => {
    const f = filters.value
    const out: Record<string, any> = {}

    const entities: Record<string, any> = {}
    if (f.entities.victim.length) entities.victim = f.entities.victim
    if (f.entities.attacker.length) entities.attacker = f.entities.attacker
    if (f.entities.both.length) entities.both = f.entities.both
    if (Object.keys(entities).length) out.entities = entities

    if (f.items.length) out.items = f.items

    const loc: Record<string, any> = {}
    if (f.location.securityTypes.length) loc.securityTypes = f.location.securityTypes
    if (f.location.systemId) { loc.systemId = f.location.systemId; loc.systemName = f.location.systemName }
    if (f.location.regionId) { loc.regionId = f.location.regionId; loc.regionName = f.location.regionName }
    if (f.location.constellationId) { loc.constellationId = f.location.constellationId; loc.constellationName = f.location.constellationName }
    if (Object.keys(loc).length) out.location = loc

    const tr = f.timeRange
    if ('preset' in tr && tr.preset && tr.preset !== '30d') out.timeRange = tr
    else if ('from' in tr) out.timeRange = tr

    if (f.attackerCount) out.attackerCount = f.attackerCount
    if (f.attackerType) out.attackerType = f.attackerType
    if (f.iskValue) out.iskValue = f.iskValue
    if (f.iskMin) out.iskMin = f.iskMin
    if (f.iskMax) out.iskMax = f.iskMax
    if (f.shipCategory) out.shipCategory = f.shipCategory
    if (f.techLevel) out.techLevel = f.techLevel
    if (f.sort.field !== 'killmail_time' || f.sort.direction !== 'desc') out.sort = f.sort

    return Object.keys(out).length ? JSON.stringify(out) : null
})

const applyParsedFilters = (parsed: any) => {
    const f = defaultFilters()
    if (parsed.entities) {
        if (parsed.entities.victim) f.entities.victim = parsed.entities.victim
        if (parsed.entities.attacker) f.entities.attacker = parsed.entities.attacker
        if (parsed.entities.both) f.entities.both = parsed.entities.both
    }
    if (parsed.items) {
        // Back-compat: saved searches from before the slot/side fields existed
        // had items as { typeId, name }. Default the new fields to match the
        // old behavior (any slot, victim-side).
        f.items = parsed.items.map((i: any) => ({
            typeId: i.typeId,
            groupId: i.groupId,
            name: i.name,
            slot: (i.slot as ItemSlot) ?? 'any',
            side: (i.side as ItemSide) ?? 'victim',
        }))
    }
    if (parsed.location) {
        if (parsed.location.securityTypes) f.location.securityTypes = parsed.location.securityTypes
        if (parsed.location.systemId) { f.location.systemId = parsed.location.systemId; f.location.systemName = parsed.location.systemName }
        if (parsed.location.regionId) { f.location.regionId = parsed.location.regionId; f.location.regionName = parsed.location.regionName }
        if (parsed.location.constellationId) { f.location.constellationId = parsed.location.constellationId; f.location.constellationName = parsed.location.constellationName }
    }
    if (parsed.timeRange) f.timeRange = parsed.timeRange
    if (parsed.attackerCount) f.attackerCount = parsed.attackerCount
    if (parsed.attackerType) f.attackerType = parsed.attackerType
    if (parsed.iskValue) f.iskValue = parsed.iskValue
    if (parsed.iskMin) f.iskMin = parsed.iskMin
    if (parsed.iskMax) f.iskMax = parsed.iskMax
    if (parsed.shipCategory) f.shipCategory = parsed.shipCategory
    if (parsed.techLevel) f.techLevel = parsed.techLevel
    if (parsed.sort) f.sort = parsed.sort

    filters.value = f

    if (parsed.timeRange && 'from' in parsed.timeRange) {
        customFrom.value = parsed.timeRange.from
        customTo.value = parsed.timeRange.to
    }
}

// Restore from URL
const restoreFromUrl = () => {
    const q = route.query.q as string
    if (!q) return
    try {
        const parsed = JSON.parse(q)
        applyParsedFilters(parsed)
        showResults.value = true

        // Restore view mode + dedup from URL
        const urlView = route.query.view as string
        const urlDh = route.query.dh as string
        const urlDm = route.query.dm as string

        if (urlDh && (urlDm === 'exact' || urlDm === 'family')) {
            // Restore fit drill-down → kills view filtered by hash
            drilldownHash.value = urlDh
            drilldownMode.value = urlDm
            viewMode.value = 'kills'
            appliedKillListParams.value = killListParams.value
        } else if (urlView === 'fits') {
            viewMode.value = 'fits'
            const urlDedup = route.query.dedup as string
            if (urlDedup === 'exact' || urlDedup === 'family' || urlDedup === 'none') {
                fitsDedup.value = urlDedup
            }
            fetchFits()
        } else {
            appliedKillListParams.value = killListParams.value
        }
    } catch { /* ignore */ }
}

// URL sync
let skipUrlSync = false

skipUrlSync = true
restoreFromUrl()
skipUrlSync = false

const syncUrl = () => {
    if (skipUrlSync) return
    if (!import.meta.client) return
    const encoded = filtersForUrl.value
    const q: Record<string, string> = {}
    if (encoded) q.q = encoded
    if (viewMode.value !== 'kills') q.view = viewMode.value
    if (viewMode.value === 'fits' && fitsDedup.value !== 'family') q.dedup = fitsDedup.value
    if (drilldownHash.value && drilldownMode.value) {
        q.dh = drilldownHash.value
        q.dm = drilldownMode.value
    }
    router.replace({ query: Object.keys(q).length ? q : {} })
}

watch(filtersForUrl, syncUrl, { flush: 'post' })
watch(viewMode, syncUrl, { flush: 'post' })
watch(fitsDedup, syncUrl, { flush: 'post' })
watch(drilldownHash, syncUrl, { flush: 'post' })

// Auto-refresh after a filter change. Typing into the ISK range needs a
// debounce so we don't fire a query per keystroke, but a chip click is a
// single discrete intent — making it wait 500ms with no visible feedback is
// what made the page feel locked up. Discrete changes go on the next tick.
const TYPED_DEBOUNCE_MS = 500
let typedEditPending = false
let debounceTimer: ReturnType<typeof setTimeout>
watch(() => serializedFilters.value, () => {
    clearTimeout(debounceTimer)
    // Clear drill-down when filters change — the new filter set may not intersect the hash
    if (drilldownHash.value) {
        drilldownHash.value = null
        drilldownMode.value = null
        drilldownShipName.value = ''
    }
    if (!hasFilters.value) {
        showResults.value = false
        fitsData.value = []
        return
    }
    const delay = typedEditPending ? TYPED_DEBOUNCE_MS : 0
    typedEditPending = false
    debounceTimer = setTimeout(() => {
        showResults.value = true
        track('filter.apply', {
            view: viewMode.value,
            categories: Object.keys(filtersForUrl.value || {}).join(',') || 'none',
        })
        if (viewMode.value === 'fits') {
            fetchFits()
        } else {
            appliedKillListParams.value = killListParams.value
        }
    }, delay)
})

// Re-fetch when view mode changes (if filters are active)
watch(viewMode, () => {
    // Clear drill-down when user switches back to fits
    if (viewMode.value === 'fits' && drilldownHash.value) {
        drilldownHash.value = null
        drilldownMode.value = null
        drilldownShipName.value = ''
    }
    if (!hasFilters.value || !showResults.value) return
    if (viewMode.value === 'fits') {
        fetchFits()
    } else {
        appliedKillListParams.value = killListParams.value
    }
})

// Entity helpers
const isEntityType = (type: string) => ['character', 'corporation', 'alliance', 'ship', 'shipgroup', 'faction'].includes(type)
const isItemType = (type: string) => type === 'item'
const isGroupType = (type: string) => type === 'group'
const isLocationType = (type: string) => ['system', 'region', 'constellation'].includes(type)

// Per-list cap for filter arrays — mirrors the POST /killmails/search
// server cap to keep query plans tight.
const MAX_PER_LIST = 15
const toast = useToast()

const addEntityToFilter = (result: SearchHit, role: 'victim' | 'attacker' | 'both', exclude = false) => {
    const numericId = Number(result.id.split('_').slice(1).join('_'))
    // Items used as weapons go as type 'weapon' on attacker side
    const entityType = (result.type === 'item' && role === 'attacker') ? 'weapon' : result.type
    const entity: Entity = { id: numericId, type: entityType, name: result.name, role, exclude }
    const arr = filters.value.entities[role]
    if (arr.some(e => e.id === numericId && e.type === entityType && e.exclude === exclude)) {
        showDropdown.value = false
        return
    }
    if (arr.length >= MAX_PER_LIST) {
        toast.error(`Max ${MAX_PER_LIST} ${role} entities per filter`)
        showDropdown.value = false
        return
    }
    arr.push(entity)
    showDropdown.value = false
}

const addItemToFilter = (result: SearchHit) => {
    const numericId = Number(result.id.split('_').slice(1).join('_'))
    if (filters.value.items.some(i => i.typeId === numericId)) {
        showDropdown.value = false
        return
    }
    if (filters.value.items.length >= MAX_PER_LIST) {
        toast.error(`Max ${MAX_PER_LIST} items per filter`)
        showDropdown.value = false
        return
    }
    filters.value.items.push({ typeId: numericId, name: result.name, slot: 'any', side: 'victim' })
    showDropdown.value = false
}

const addGroupToFilter = (result: SearchHit, side: ItemSide = 'victim') => {
    const numericId = Number(result.id.split('_').slice(1).join('_'))
    if (filters.value.items.some(i => i.groupId === numericId && i.side === side)) {
        showDropdown.value = false
        return
    }
    if (filters.value.items.length >= MAX_PER_LIST) {
        toast.error(`Max ${MAX_PER_LIST} items per filter`)
        showDropdown.value = false
        return
    }
    filters.value.items.push({ groupId: numericId, name: result.name, slot: 'any', side })
    showDropdown.value = false
}

// Cycle the slot through any → fitted → cargo → any on each click. The
// three-state toggle keeps the filter chip compact (no dropdown) while still
// exposing all options.
const cycleItemSlot = (idx: number) => {
    const item = filters.value.items[idx]
    if (!item) return
    const order: ItemSlot[] = ['any', 'fitted', 'cargo']
    const next = order[(order.indexOf(item.slot) + 1) % order.length]
    item.slot = next
}

// Cycle the side through victim → attacker → either → victim. Attacker-side
// only matches damage-dealing weapon modules (guns / missile launchers /
// smartbombs) because killmails don't record the full attacker fit.
const cycleItemSide = (idx: number) => {
    const item = filters.value.items[idx]
    if (!item) return
    const order: ItemSide[] = ['victim', 'attacker', 'either']
    const next = order[(order.indexOf(item.side) + 1) % order.length]
    item.side = next
}

const addLocationFilter = (result: SearchHit) => {
    const numericId = Number(result.id.split('_').slice(1).join('_'))
    filters.value.location.securityTypes = []
    filters.value.location.systemId = null; filters.value.location.systemName = null
    filters.value.location.regionId = null; filters.value.location.regionName = null
    filters.value.location.constellationId = null; filters.value.location.constellationName = null
    if (result.type === 'system') {
        filters.value.location.systemId = numericId
        filters.value.location.systemName = result.name
    } else if (result.type === 'region') {
        filters.value.location.regionId = numericId
        filters.value.location.regionName = result.name
    } else if (result.type === 'constellation') {
        filters.value.location.constellationId = numericId
        filters.value.location.constellationName = result.name
    }
    showDropdown.value = false
}

const removeEntity = (role: 'victim' | 'attacker' | 'both', index: number) => {
    filters.value.entities[role].splice(index, 1)
}

const removeItem = (index: number) => {
    filters.value.items.splice(index, 1)
}

const clearLocation = () => {
    filters.value.location.systemId = null; filters.value.location.systemName = null
    filters.value.location.regionId = null; filters.value.location.regionName = null
    filters.value.location.constellationId = null; filters.value.location.constellationName = null
}

const toggleSecurity = (sec: string) => {
    // Single-select, same as the other filter groups — clicking the active one
    // clears it. Kept as an array because the API and saved searches take a
    // list, but it never holds more than one entry.
    const current = filters.value.location.securityTypes
    filters.value.location.securityTypes = current.includes(sec) && current.length === 1 ? [] : [sec]
}

const toggleSingleFilter = (key: 'attackerCount' | 'attackerType' | 'iskValue' | 'shipCategory' | 'techLevel', value: string) => {
    (filters.value as any)[key] = (filters.value as any)[key] === value ? null : value
    // Clear ISK range when using preset
    if (key === 'iskValue') {
        filters.value.iskMin = ''
        filters.value.iskMax = ''
    }
}

const setTimePreset = (preset: string) => {
    const current = filters.value.timeRange
    if ('preset' in current && current.preset === preset) {
        filters.value.timeRange = { preset: null }
    } else {
        filters.value.timeRange = { preset }
    }
    customFrom.value = ''
    customTo.value = ''
}

const applyCustomTime = () => {
    if (customFrom.value && customTo.value) {
        filters.value.timeRange = { from: customFrom.value, to: customTo.value }
    }
}

const onIskRangeInput = () => {
    // Clear preset when using custom range
    filters.value.iskValue = null
    // Typing needs the debounce; a chip click does not — see the refresh watcher.
    typedEditPending = true
}

const clearAllFilters = () => {
    filters.value = defaultFilters()
    showResults.value = false
    customFrom.value = ''
    customTo.value = ''
}

// Close dropdown on outside click
const searchContainer = ref<HTMLElement>()
const onClickOutside = (e: MouseEvent) => {
    if (searchContainer.value && !searchContainer.value.contains(e.target as Node)) {
        showDropdown.value = false
    }
}

onMounted(() => {
    document.addEventListener('click', onClickOutside)
    loadSavedSearches()
})
onUnmounted(() => document.removeEventListener('click', onClickOutside))

// Parse ISK shorthand: "1b" → 1000000000, "500m" → 500000000, plain number as ISK
function parseIsk(val: string): number | null {
    if (!val) return null
    const s = val.trim().toLowerCase()
    const match = s.match(/^([\d.]+)\s*([bBmMkK])?$/)
    if (!match) return null
    const num = parseFloat(match[1]!)
    if (Number.isNaN(num)) return null
    switch (match[2]?.toLowerCase()) {
        case 'b': return num * 1e9
        case 'm': return num * 1e6
        case 'k': return num * 1e3
        default: return num
    }
}

// Filter option definitions
const securityOptions = [
    { value: 'highsec', label: 'Highsec' },
    { value: 'lowsec', label: 'Lowsec' },
    { value: 'nullsec', label: 'Nullsec' },
    { value: 'wspace', label: 'W-Space' },
    { value: 'abyssal', label: 'Abyssal' },
    { value: 'pochven', label: 'Pochven' },
]

const attackerCountOptions = [
    { value: 'solo', label: 'Solo' },
    { value: '2+', label: '2+' },
    { value: '5+', label: '5+' },
    { value: '10+', label: '10+' },
    { value: '20+', label: '20+' },
    { value: '50+', label: '50+' },
    { value: '100+', label: '100+' },
]

const attackerTypeOptions = [
    { value: 'ganked', label: 'Ganked' },
    { value: 'npc', label: 'NPC' },
]

const iskValueOptions = [
    { value: '1b', label: '1B+' },
    { value: '5b', label: '5B+' },
    { value: '10b', label: '10B+' },
    { value: '50b', label: '50B+' },
    { value: '100b', label: '100B+' },
]

const shipCategoryOptions = [
    { value: 'frigates', label: 'Frigates' },
    { value: 'destroyers', label: 'Destroyers' },
    { value: 'cruisers', label: 'Cruisers' },
    { value: 'battlecruisers', label: 'Battlecruisers' },
    { value: 'battleships', label: 'Battleships' },
    { value: 'capitals', label: 'Capitals' },
    { value: 'supercarriers', label: 'Supercarriers' },
    { value: 'titans', label: 'Titans' },
    { value: 'supercapitals', label: 'Super Capitals' },
    { value: 'freighters', label: 'Freighters' },
    { value: 'citadels', label: 'Structures' },
]

const techLevelOptions = [
    { value: 't1', label: 'T1' },
    { value: 't2', label: 'T2' },
    { value: 't3', label: 'T3' },
    { value: 'faction', label: 'Faction' },
]

const timePresets = [
    { value: 'today', label: 'Today' },
    { value: 'yesterday', label: 'Yesterday' },
    { value: '24h', label: '24h' },
    { value: '7d', label: '7 Days' },
    { value: '30d', label: '30 Days' },
    { value: '90d', label: '90 Days' },
    { value: 'thisWeek', label: 'This Week' },
    { value: 'thisMonth', label: 'This Month' },
]

const sortFieldOptions = [
    { value: 'killmail_time', label: 'Time' },
    { value: 'total_value', label: 'ISK Value' },
    { value: 'attacker_count', label: 'Attackers' },
]

/**
 * Everything the query has been narrowed by, as removable chips.
 *
 * Entities and items have their own tag row; this covers the option-list
 * filters, which previously could only be found by re-scanning all six
 * panels — and were invisible entirely once the panels were collapsed.
 */
const activeFilterChips = computed(() => {
    const f = filters.value
    const chips: Array<{ key: string; group: string; label: string; clear: () => void }> = []
    const labelOf = (opts: { value: string; label: string }[], value: string) =>
        opts.find(o => o.value === value)?.label ?? value

    for (const sec of f.location.securityTypes) {
        chips.push({
            key: `sec-${sec}`,
            group: 'Space',
            label: labelOf(securityOptions, sec),
            clear: () => toggleSecurity(sec),
        })
    }
    if (f.attackerCount) {
        chips.push({
            key: 'attackerCount',
            group: 'Attackers',
            label: labelOf(attackerCountOptions, f.attackerCount),
            clear: () => { filters.value.attackerCount = null },
        })
    }
    if (f.attackerType) {
        chips.push({
            key: 'attackerType',
            group: 'Type',
            label: labelOf(attackerTypeOptions, f.attackerType),
            clear: () => { filters.value.attackerType = null },
        })
    }
    if (f.iskValue) {
        chips.push({
            key: 'iskValue',
            group: 'ISK',
            label: labelOf(iskValueOptions, f.iskValue),
            clear: () => { filters.value.iskValue = null },
        })
    }
    if (f.iskMin || f.iskMax) {
        chips.push({
            key: 'iskRange',
            group: 'ISK',
            label: `${f.iskMin || '0'} – ${f.iskMax || '∞'}`,
            clear: () => { filters.value.iskMin = ''; filters.value.iskMax = '' },
        })
    }
    if (f.shipCategory) {
        chips.push({
            key: 'shipCategory',
            group: 'Ships',
            label: labelOf(shipCategoryOptions, f.shipCategory),
            clear: () => { filters.value.shipCategory = null },
        })
    }
    if (f.techLevel) {
        chips.push({
            key: 'techLevel',
            group: 'Tech',
            label: labelOf(techLevelOptions, f.techLevel),
            clear: () => { filters.value.techLevel = null },
        })
    }

    const range = f.timeRange
    if ('preset' in range && range.preset) {
        chips.push({
            key: 'timePreset',
            group: 'Time',
            label: labelOf(timePresets, range.preset),
            clear: () => setTimePreset(range.preset as string),
        })
    } else if ('from' in range && range.from) {
        chips.push({
            key: 'timeRange',
            group: 'Time',
            label: `${range.from.replace('T', ' ')} → ${range.to.replace('T', ' ')} EVE`,
            clear: () => {
                filters.value.timeRange = { preset: null }
                customFrom.value = ''
                customTo.value = ''
            },
        })
    }

    return chips
})

// Helpers
const btnClass = (active: boolean) =>
    active
        ? 'bg-blue-500/20 text-blue-400 border-blue-500/40'
        : 'bg-white/[0.04] text-gray-400 border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400'

const entityImage = (entity: Entity): string => {
    switch (entity.type) {
        case 'character': return `/images/characters/${entity.id}/portrait?size=32`
        case 'corporation': return `/images/corporations/${entity.id}/logo?size=32`
        case 'alliance': return `/images/alliances/${entity.id}/logo?size=32`
        case 'ship': return `/images/types/${entity.id}/icon?size=32`
        case 'shipgroup': return '' // no image for hull class groups
        case 'weapon': return `/images/types/${entity.id}/icon?size=32`
        case 'faction': return `/images/corporations/${entity.id}/logo?size=32`
        default: return ''
    }
}

const searchResultImage = (hit: SearchHit): string => {
    const numericId = hit.id.split('_').slice(1).join('_')
    switch (hit.type) {
        case 'character': return `/images/characters/${numericId}/portrait?size=32`
        case 'corporation': return `/images/corporations/${numericId}/logo?size=32`
        case 'alliance': return `/images/alliances/${numericId}/logo?size=32`
        case 'ship': return `/images/types/${numericId}/icon?size=32`
        case 'item': return `/images/types/${numericId}/icon?size=32`
        case 'faction': return `/images/corporations/${numericId}/logo?size=32`
        default: return ''
    }
}

const roleColor = (role: string, exclude?: boolean) => {
    if (exclude) return 'bg-red-900/30 text-red-300 border-red-500/40 line-through'
    switch (role) {
        case 'victim': return 'bg-red-500/20 text-red-400 border-red-500/30'
        case 'attacker': return 'bg-blue-500/20 text-blue-400 border-blue-500/30'
        case 'both': return 'bg-green-500/20 text-green-400 border-green-500/30'
        default: return ''
    }
}

const locationLabel = computed(() => {
    const loc = filters.value.location
    if (loc.systemName) return `System: ${loc.systemName}`
    if (loc.constellationName) return `Constellation: ${loc.constellationName}`
    if (loc.regionName) return `Region: ${loc.regionName}`
    return null
})
</script>

<template>
    <div>
        <!-- Header -->
        <div class="glass-panel p-5 mb-4">
            <div class="flex items-start justify-between gap-4 flex-wrap">
                <div class="min-w-0">
                    <h1 class="flex items-center gap-2.5 text-xl font-bold text-white">
                        <span class="w-8 h-8 rounded-lg bg-blue-500/15 flex items-center justify-center flex-shrink-0">
                            <Icon name="lucide:search-code" class="text-base text-blue-400" />
                        </span>
                        Advanced Search
                    </h1>
                    <p class="text-sm text-gray-500 mt-2 max-w-3xl">
                        Build a query from pilots, corporations, alliances, ships, items and space, then narrow it by
                        time, ISK and engagement size. Searches can be linked or saved.
                    </p>
                </div>
            <div class="flex items-center gap-2 flex-shrink-0">
                <!-- Copy URL -->
                <button
                    v-if="hasFilters"
                    @click="copyUrl"
                    class="px-2.5 py-1.5 text-xs font-medium rounded border transition-colors"
                    :class="copyStatus === 'copied' ? 'bg-green-500/20 text-green-400 border-green-500/40' : 'bg-white/[0.04] text-gray-400 border-white/[0.08] hover:bg-blue-500/[0.08]'"
                >
                    <Icon :name="copyStatus === 'copied' ? 'lucide:check' : 'lucide:link'" class="text-xs mr-1" />
                    {{ copyStatus === 'copied' ? 'Copied!' : 'Copy Link' }}
                </button>
                <!-- Save search -->
                <div class="relative">
                    <button
                        v-if="hasFilters"
                        @click="showSaveDialog = !showSaveDialog; showSavedList = false"
                        class="px-2.5 py-1.5 text-xs font-medium rounded border transition-colors bg-white/[0.04] text-gray-400 border-white/[0.08] hover:bg-blue-500/[0.08]"
                    >
                        <Icon name="lucide:bookmark" class="text-xs mr-1" />
                        Save
                    </button>
                    <div v-if="showSaveDialog" class="absolute right-0 top-full mt-1 z-50 w-64 rounded-lg bg-black/90 backdrop-blur-xl border border-white/[0.08] shadow-2xl p-3">
                        <input
                            v-model="saveName"
                            type="text"
                            placeholder="Search name..."
                            class="w-full px-2 py-1.5 text-xs bg-white/[0.04] border border-white/[0.08] rounded text-gray-300 focus:border-blue-500/40 focus:outline-none mb-2"
                            @keyup.enter="saveCurrentSearch"
                        />
                        <button @click="saveCurrentSearch" :disabled="!saveName.trim()" class="w-full px-2 py-1.5 text-xs font-medium rounded bg-blue-500 text-white hover:bg-blue-600 disabled:opacity-40">
                            Save Search
                        </button>
                    </div>
                </div>
                <!-- Load saved -->
                <div class="relative">
                    <button
                        v-if="savedSearches.length > 0"
                        @click="showSavedList = !showSavedList; showSaveDialog = false"
                        class="px-2.5 py-1.5 text-xs font-medium rounded border transition-colors bg-white/[0.04] text-gray-400 border-white/[0.08] hover:bg-blue-500/[0.08]"
                    >
                        <Icon name="lucide:folder-open" class="text-xs mr-1" />
                        Saved ({{ savedSearches.length }})
                    </button>
                    <div v-if="showSavedList" class="absolute right-0 top-full mt-1 z-50 w-72 rounded-lg bg-black/90 backdrop-blur-xl border border-white/[0.08] shadow-2xl max-h-60 overflow-y-auto">
                        <div v-for="(search, idx) in savedSearches" :key="idx"
                            class="flex items-center justify-between gap-2 px-3 py-2 border-b border-white/[0.04] last:border-b-0 hover:bg-blue-500/[0.04]">
                            <button @click="loadSavedSearch(search)" class="flex-1 text-left min-w-0">
                                <div class="text-xs text-gray-300 truncate">{{ search.name }}</div>
                                <div class="text-fine text-gray-600">{{ new Date(search.date).toLocaleDateString() }}</div>
                            </button>
                            <button @click.stop="deleteSavedSearch(idx)" class="text-gray-600 hover:text-red-400 flex-shrink-0">
                                <Icon name="lucide:trash-2" class="text-fine" />
                            </button>
                        </div>
                    </div>
                </div>
            </div>
            </div>
        </div>

        <!-- Search Bar -->
        <div ref="searchContainer" class="relative mb-4">
            <div class="glass-panel flex items-center gap-2 px-3 py-2.5 focus-within:border-blue-500/40 transition-colors">
                <Icon name="lucide:search" class="text-base text-gray-500 flex-shrink-0" />
                <input
                    v-model="searchQuery"
                    type="text"
                    placeholder="Search for pilots, corporations, alliances, ships, items, systems, regions..."
                    class="flex-1 bg-transparent text-white text-sm outline-none placeholder-gray-600"
                    @input="handleSearch"
                />
                <Icon v-if="isSearching" name="lucide:loader-2" class="text-base text-gray-500 animate-spin" />
            </div>

            <!-- Search dropdown -->
            <div v-if="showDropdown" class="absolute z-50 mt-1 w-full rounded-lg bg-black/90 backdrop-blur-xl border border-white/[0.08] shadow-2xl max-h-80 overflow-y-auto">
                <div v-for="hit in searchResults" :key="`${hit.type}-${hit.id}`" class="border-b border-white/[0.04] last:border-b-0">
                    <div class="px-3 py-2.5">
                        <div class="flex items-center justify-between gap-3">
                            <div class="flex items-center gap-2.5 min-w-0">
                                <img v-if="searchResultImage(hit)" :src="searchResultImage(hit)" class="w-7 h-7 rounded flex-shrink-0" loading="lazy" />
                                <div v-else class="w-7 h-7 rounded bg-white/[0.06] flex items-center justify-center flex-shrink-0">
                                    <Icon :name="isGroupType(hit.type) ? 'lucide:layers' : 'lucide:map-pin'" class="text-xs text-gray-500" />
                                </div>
                                <div class="min-w-0">
                                    <div class="text-sm text-gray-200 truncate">{{ hit.name }}</div>
                                    <div class="text-fine text-gray-500 capitalize">{{ hit.type }}
                                        <span v-if="hit.ticker" class="text-gray-600 font-mono ml-1">[{{ hit.ticker }}]</span>
                                    </div>
                                </div>
                            </div>

                            <!-- Action buttons -->
                            <div v-if="isEntityType(hit.type)" class="flex gap-1 flex-shrink-0">
                                <button @click.stop="addEntityToFilter(hit, 'victim')" class="px-2 py-1 text-fine font-medium rounded bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors">Victim</button>
                                <button @click.stop="addEntityToFilter(hit, 'both')" class="px-2 py-1 text-fine font-medium rounded bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors">Both</button>
                                <button @click.stop="addEntityToFilter(hit, 'attacker')" class="px-2 py-1 text-fine font-medium rounded bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 transition-colors">Attacker</button>
                                <button @click.stop="addEntityToFilter(hit, 'victim', true)" class="px-1.5 py-1 text-fine font-medium rounded bg-red-900/30 text-red-300 hover:bg-red-900/50 transition-colors" v-tooltip="'Exclude'">
                                    <Icon name="lucide:ban" class="text-fine" />
                                </button>
                            </div>
                            <div v-else-if="isItemType(hit.type)" class="flex gap-1 flex-shrink-0">
                                <button @click.stop="addItemToFilter(hit)" class="px-2 py-1 text-fine font-medium rounded bg-purple-500/20 text-purple-400 hover:bg-purple-500/30 transition-colors">Item</button>
                                <button @click.stop="addEntityToFilter(hit, 'attacker')" class="px-2 py-1 text-fine font-medium rounded bg-amber-500/20 text-amber-400 hover:bg-amber-500/30 transition-colors">Weapon</button>
                            </div>
                            <div v-else-if="isGroupType(hit.type)" class="flex gap-1 flex-shrink-0">
                                <button @click.stop="addGroupToFilter(hit, 'victim')" class="px-2 py-1 text-fine font-medium rounded bg-purple-500/20 text-purple-400 hover:bg-purple-500/30 transition-colors">Item</button>
                                <button @click.stop="addGroupToFilter(hit, 'attacker')" class="px-2 py-1 text-fine font-medium rounded bg-amber-500/20 text-amber-400 hover:bg-amber-500/30 transition-colors">Weapon</button>
                            </div>
                            <div v-else-if="isLocationType(hit.type)" class="flex-shrink-0">
                                <button @click.stop="addLocationFilter(hit)" class="px-2 py-1 text-fine font-medium rounded bg-amber-500/20 text-amber-400 hover:bg-amber-500/30 transition-colors">Set Location</button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Active Tags -->
        <div v-if="filters.entities.victim.length || filters.entities.attacker.length || filters.entities.both.length || filters.items.length || locationLabel" class="flex flex-wrap gap-1.5 mb-4">
            <template v-for="role in (['victim', 'attacker', 'both'] as const)" :key="role">
                <div v-for="(entity, idx) in filters.entities[role]" :key="`${role}-${entity.id}-${entity.exclude}`"
                    class="inline-flex items-center gap-1.5 px-2 py-1 rounded border text-xs font-medium"
                    :class="roleColor(role, entity.exclude)">
                    <img v-if="entityImage(entity)" :src="entityImage(entity)" class="w-4 h-4 rounded" loading="lazy" />
                    <Icon v-else name="lucide:ship" class="text-fine" />
                    <span :class="{ 'line-through': entity.exclude }">{{ entity.name }}</span>
                    <span class="opacity-60 capitalize text-fine">{{ entity.exclude ? 'exclude' : entity.type === 'weapon' ? 'weapon' : entity.type === 'shipgroup' ? 'hull class' : role }}</span>
                    <button @click="removeEntity(role, idx)" class="ml-0.5 opacity-60 hover:opacity-100">
                        <Icon name="lucide:x" class="text-fine" />
                    </button>
                </div>
            </template>

            <div v-for="(item, idx) in filters.items" :key="`item-${idx}`"
                class="inline-flex items-center gap-1 px-2 py-1 rounded border text-xs font-medium"
                :class="item.groupId != null
                    ? 'bg-cyan-500/20 text-cyan-400 border-cyan-500/30'
                    : 'bg-purple-500/20 text-purple-400 border-purple-500/30'">
                <img v-if="item.typeId != null" :src="`/images/types/${item.typeId}/icon?size=32`" class="w-4 h-4 rounded" loading="lazy" />
                <Icon v-else name="lucide:layers" class="text-fine ml-0.5" />
                <span class="ml-0.5">{{ item.name }}</span>
                <!-- Slot scope: any/fitted/cargo. Click to cycle. Ignored by
                     the backend when side='attacker' (no slot info on attackers). -->
                <button @click.stop="cycleItemSlot(idx)"
                    class="ml-1 px-1.5 py-0.5 rounded bg-black/25 text-fine opacity-80 hover:opacity-100 hover:bg-black/40 transition-colors"
                    v-tooltip="`Slot scope: ${item.slot}`">
                    {{ item.slot }}
                </button>
                <!-- Side: victim/attacker/either. Attacker only matches
                     damage-dealing weapons (see backend ItemSide docs). -->
                <button @click.stop="cycleItemSide(idx)"
                    class="px-1.5 py-0.5 rounded bg-black/25 text-fine opacity-80 hover:opacity-100 hover:bg-black/40 transition-colors"
                    v-tooltip="`Side: ${item.side}`">
                    {{ item.side }}
                </button>
                <button @click="removeItem(idx)" class="ml-0.5 opacity-60 hover:opacity-100">
                    <Icon name="lucide:x" class="text-fine" />
                </button>
            </div>

            <div v-if="locationLabel"
                class="inline-flex items-center gap-1.5 px-2 py-1 rounded border text-xs font-medium bg-amber-500/20 text-amber-400 border-amber-500/30">
                <Icon name="lucide:map-pin" class="text-fine" />
                <span>{{ locationLabel }}</span>
                <button @click="clearLocation" class="ml-0.5 opacity-60 hover:opacity-100">
                    <Icon name="lucide:x" class="text-fine" />
                </button>
            </div>
        </div>

        <!-- Applied filters — visible whether or not the panels are open -->
        <div v-if="activeFilterChips.length" class="flex flex-wrap items-center gap-1.5 mb-3">
            <span class="text-fine text-gray-600 font-medium uppercase tracking-wider mr-0.5">Applied</span>
            <button v-for="chip in activeFilterChips" :key="chip.key" @click="chip.clear()"
                class="group inline-flex items-center gap-1.5 pl-2 pr-1.5 py-1 rounded-md border border-blue-500/30 bg-blue-500/[0.12] text-xs text-blue-300 hover:bg-blue-500/20 transition-colors cursor-pointer"
                v-tooltip="`Remove ${chip.group}: ${chip.label}`">
                <span class="text-blue-400/60 text-fine uppercase tracking-wider">{{ chip.group }}</span>
                <span class="font-medium">{{ chip.label }}</span>
                <Icon name="lucide:x" class="text-fine text-blue-400/50 group-hover:text-blue-300 transition-colors" />
            </button>
        </div>

        <!-- Collapse toggle -->
        <div class="flex items-center justify-between gap-3 mb-2">
            <button @click="filtersCollapsed = !filtersCollapsed"
                class="flex items-center gap-1.5 text-fine font-bold uppercase tracking-wider text-gray-500 hover:text-blue-400 transition-colors cursor-pointer">
                <Icon :name="filtersCollapsed ? 'lucide:chevron-right' : 'lucide:chevron-down'" class="text-xs" />
                Filters
                <span v-if="filtersCollapsed && activeFilterChips.length" class="text-blue-400/70 normal-case tracking-normal">
                    ({{ activeFilterChips.length }} applied)
                </span>
            </button>
            <button v-if="hasFilters" @click="clearAllFilters"
                class="px-2.5 py-1 text-fine font-medium rounded border transition-colors bg-white/[0.04] text-gray-500 border-white/[0.08] hover:bg-blue-500/[0.08] cursor-pointer">
                Clear All
            </button>
        </div>

        <!-- Filter Controls Grid (collapsible) -->
        <div v-show="!filtersCollapsed" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 mb-4">
            <!-- Security Space -->
            <div class="glass-panel p-3">
                <div class="text-fine font-bold uppercase tracking-wider text-gray-600 mb-2">
                    Security Space
                </div>
                <div class="flex flex-wrap gap-1">
                    <button v-for="opt in securityOptions" :key="opt.value"
                        @click="toggleSecurity(opt.value)"
                        class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                        :class="btnClass(filters.location.securityTypes.includes(opt.value))">
                        {{ opt.label }}
                    </button>
                </div>
            </div>

            <!-- Attacker Count -->
            <div class="glass-panel p-3">
                <div class="text-fine font-bold uppercase tracking-wider text-gray-600 mb-2">Attacker Count</div>
                <div class="flex flex-wrap gap-1">
                    <button v-for="opt in attackerCountOptions" :key="opt.value"
                        @click="toggleSingleFilter('attackerCount', opt.value)"
                        class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                        :class="btnClass(filters.attackerCount === opt.value)">
                        {{ opt.label }}
                    </button>
                </div>
            </div>

            <!-- Attacker Type -->
            <div class="glass-panel p-3">
                <div class="text-fine font-bold uppercase tracking-wider text-gray-600 mb-2">Attacker Type</div>
                <div class="flex flex-wrap gap-1">
                    <button v-for="opt in attackerTypeOptions" :key="opt.value"
                        @click="toggleSingleFilter('attackerType', opt.value)"
                        class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                        :class="btnClass(filters.attackerType === opt.value)">
                        {{ opt.label }}
                    </button>
                </div>
            </div>

            <!-- ISK Value -->
            <div class="glass-panel p-3">
                <div class="text-fine font-bold uppercase tracking-wider text-gray-600 mb-2">ISK Value</div>
                <div class="flex flex-wrap gap-1 mb-2">
                    <button v-for="opt in iskValueOptions" :key="opt.value"
                        @click="toggleSingleFilter('iskValue', opt.value)"
                        class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                        :class="btnClass(filters.iskValue === opt.value)">
                        {{ opt.label }}
                    </button>
                </div>
                <div class="grid grid-cols-2 gap-2">
                    <div>
                        <label class="block text-fine text-gray-500 mb-1">Min</label>
                        <input v-model="filters.iskMin" type="text" placeholder="e.g. 500m"
                            class="w-full px-2 py-1 text-xs bg-white/[0.04] border border-white/[0.08] rounded text-gray-300 focus:border-blue-500/40 focus:outline-none"
                            @input="onIskRangeInput" />
                    </div>
                    <div>
                        <label class="block text-fine text-gray-500 mb-1">Max</label>
                        <input v-model="filters.iskMax" type="text" placeholder="e.g. 5b"
                            class="w-full px-2 py-1 text-xs bg-white/[0.04] border border-white/[0.08] rounded text-gray-300 focus:border-blue-500/40 focus:outline-none"
                            @input="onIskRangeInput" />
                    </div>
                </div>
            </div>

            <!-- Ship Category -->
            <div class="glass-panel p-3">
                <div class="text-fine font-bold uppercase tracking-wider text-gray-600 mb-2">Ship Category</div>
                <div class="flex flex-wrap gap-1">
                    <button v-for="opt in shipCategoryOptions" :key="opt.value"
                        @click="toggleSingleFilter('shipCategory', opt.value)"
                        class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                        :class="btnClass(filters.shipCategory === opt.value)">
                        {{ opt.label }}
                    </button>
                </div>
                <div class="text-fine font-bold uppercase tracking-wider text-gray-600 mt-3 mb-2">Tech Level</div>
                <div class="flex flex-wrap gap-1">
                    <button v-for="opt in techLevelOptions" :key="opt.value"
                        @click="toggleSingleFilter('techLevel', opt.value)"
                        class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                        :class="btnClass(filters.techLevel === opt.value)">
                        {{ opt.label }}
                    </button>
                </div>
            </div>

            <!-- Time Range -->
            <div class="glass-panel p-3">
                <div class="text-fine font-bold uppercase tracking-wider text-gray-600 mb-2">Time Range</div>
                <div class="flex flex-wrap gap-1 mb-2">
                    <button v-for="opt in timePresets" :key="opt.value"
                        @click="setTimePreset(opt.value)"
                        class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                        :class="btnClass('preset' in filters.timeRange && filters.timeRange.preset === opt.value)">
                        {{ opt.label }}
                    </button>
                </div>
                <div class="grid grid-cols-2 gap-2">
                    <div>
                        <label class="block text-fine text-gray-500 mb-1">From <span class="text-gray-700">(EVE)</span></label>
                        <DateTimePicker v-model="customFrom" size="sm" placeholder="From" />
                    </div>
                    <div>
                        <label class="block text-fine text-gray-500 mb-1">To <span class="text-gray-700">(EVE)</span></label>
                        <DateTimePicker v-model="customTo" size="sm" placeholder="To" @update:model-value="applyCustomTime" />
                    </div>
                </div>
            </div>
        </div>

        <!-- Sort controls + view mode toggle -->
        <div v-show="!filtersCollapsed" class="flex flex-wrap items-center gap-3 mb-4">
            <div class="flex flex-wrap items-center gap-1">
                <span class="text-fine text-gray-500 mr-1">Sort:</span>
                <button v-for="opt in sortFieldOptions" :key="opt.value"
                    @click="filters.sort.field = opt.value"
                    class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                    :class="btnClass(filters.sort.field === opt.value)">
                    {{ opt.label }}
                </button>
            </div>
            <div class="flex items-center gap-1">
                <button @click="filters.sort.direction = 'desc'"
                    class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                    :class="btnClass(filters.sort.direction === 'desc')">
                    <Icon name="lucide:arrow-down" class="text-xs" />
                </button>
                <button @click="filters.sort.direction = 'asc'"
                    class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                    :class="btnClass(filters.sort.direction === 'asc')">
                    <Icon name="lucide:arrow-up" class="text-xs" />
                </button>
            </div>
            <!-- View mode -->
            <div class="flex items-center gap-1 ml-auto">
                <button
                    class="px-3 py-1 text-xs font-medium rounded-l-lg border transition-colors"
                    :class="viewMode === 'kills'
                        ? 'bg-blue-500/20 text-blue-400 border-blue-400/40'
                        : 'bg-white/[0.04] text-gray-400 border-white/[0.08] hover:text-blue-400'"
                    @click="viewMode = 'kills'">
                    <Icon name="lucide:list" class="text-xs mr-1" />
                    Killmails
                </button>
                <button
                    class="px-3 py-1 text-xs font-medium rounded-r-lg border border-l-0 transition-colors"
                    :class="viewMode === 'fits'
                        ? 'bg-blue-500/20 text-blue-400 border-blue-400/40'
                        : 'bg-white/[0.04] text-gray-400 border-white/[0.08] hover:text-blue-400'"
                    @click="viewMode = 'fits'">
                    <Icon name="lucide:wrench" class="text-xs mr-1" />
                    Fits
                </button>
            </div>
        </div>

        <!-- Results: Killmails view -->
        <div v-if="showResults && viewMode === 'kills'">
            <!-- Drill-down banner: shown when viewing kills for a specific fit -->
            <div v-if="drilldownHash"
                 class="flex items-center gap-2 mb-3 px-3 py-2 rounded-lg bg-blue-500/10 border border-blue-500/20">
                <button @click="clearDrilldown"
                        class="inline-flex items-center gap-1 text-xs font-medium text-blue-400 hover:text-blue-300 transition-colors">
                    <Icon name="lucide:arrow-left" class="text-sm" />
                    Back to fits
                </button>
                <span class="text-xs text-gray-400">
                    All killmails with
                    <span class="text-white font-medium">{{ drilldownShipName }}</span>
                    {{ drilldownMode === 'family' ? 'family' : 'exact' }} fit
                </span>
            </div>
            <KillList
                api-endpoint="/api/killlist/advanced"
                :extra-params="appliedKillListParams"
            />
        </div>

        <!-- Results: Fits view -->
        <div v-else-if="showResults && viewMode === 'fits'">
            <!-- Grouping mode -->
            <div class="flex items-center gap-1 mb-3">
                <span class="text-fine text-gray-500 mr-1">Group:</span>
                <button @click="fitsDedup = 'family'"
                    class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                    :class="btnClass(fitsDedup === 'family')">
                    Family
                </button>
                <button @click="fitsDedup = 'exact'"
                    class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                    :class="btnClass(fitsDedup === 'exact')">
                    Exact
                </button>
                <button @click="fitsDedup = 'none'"
                    class="px-2 py-1 text-xs font-medium rounded border transition-colors"
                    :class="btnClass(fitsDedup === 'none')">
                    All
                </button>
                <span class="text-fine text-gray-600 ml-2">
                    {{ fitsDedup === 'family' ? 'T1/T2/meta/faction variants grouped' : fitsDedup === 'exact' ? 'Identical module sets grouped' : 'Every kill shown individually' }}
                </span>
            </div>

            <div v-if="fitsLoading && fitsData.length === 0" class="py-12 text-center">
                <Icon name="lucide:loader-2" class="text-2xl text-gray-500 animate-spin mb-2" />
                <div class="text-sm text-gray-500">Loading fits…</div>
            </div>
            <div v-else-if="fitsData.length === 0" class="glass-panel py-12 text-center">
                <Icon name="lucide:wrench" class="text-2xl text-gray-600 mb-2" />
                <div class="text-sm text-gray-500">No matching fits found</div>
            </div>
            <template v-else>
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2">
                    <FitPreview
                        v-for="fit in fitsData"
                        :key="fit.killmail_id"
                        :killmail-id="fit.killmail_id"
                        :killmail-time="fit.killmail_time"
                        :victim-ship-type-id="fit.victim_ship_type_id"
                        :victim-ship-name="fit.victim_ship_name"
                        :total-value="fit.total_value"
                        :attacker-count="fit.attacker_count"
                        :modules="fit.modules"
                        :drones="fit.drones"
                        :count="fit.count"
                        :hash="fit.hash"
                        :dedup-mode="fit.dedup_mode"
                        @show-kills="(hash: string, mode: string) => handleShowKills(hash, mode, fit.victim_ship_name)"
                    />
                </div>
                <div v-if="fitsHasMore" class="mt-4 text-center">
                    <button
                        class="px-4 py-2 text-sm font-medium rounded-lg bg-white/[0.04] text-gray-400 border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors"
                        :disabled="fitsLoading"
                        @click="fetchFits(true)">
                        <Icon v-if="fitsLoading" name="lucide:loader-2" class="text-sm animate-spin mr-1" />
                        Load more fits
                    </button>
                </div>
            </template>
        </div>

        <!-- Empty state -->
        <div v-else class="glass-panel py-16 text-center">
            <Icon name="lucide:search" class="text-3xl text-gray-600 mb-3" />
            <div class="text-sm text-gray-500">Add filters above to find killmails</div>
            <div class="text-xs text-gray-600 mt-1">Use the search bar to add pilots, corps, alliances, ships, or items</div>
        </div>
    </div>
</template>
