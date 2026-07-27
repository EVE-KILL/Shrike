<script setup lang="ts">
// Settings → Domains section. Extracted verbatim from pages/settings/[[tab]].vue.
// This component only mounts when the Domains tab is first opened (the parent
// keeps it alive afterwards via <KeepAlive>), so `immediate: true` below is
// equivalent to the old `immediate: activeSection === 'domains'`.
import { DEFAULT_NAVBAR } from '~/composables/useDomainConfig'

const route = useRoute()
const router = useRouter()

// ── Domains ─────────────────────────────────────────────────────────────────
const { data: domainsData, refresh: refreshDomains } = useApiFetch<{ domains: any[] }>('/api/user/domains', {
    immediate: true,
    lazy: true,
})
const domainsList = computed(() => domainsData.value?.domains || [])
const domainEditing = ref<any>(null) // null = list mode, {} = create, {id: ...} = edit
const domainEditBaseline = ref<Record<string, any> | null>(null)
const domainSaving = ref(false)
const domainError = ref('')

// Subdomain availability check
const subdomainStatus = ref<{ available: boolean; reason?: string } | null>(null)
const subdomainChecking = ref(false)
let subdomainCheckTimer: ReturnType<typeof setTimeout> | null = null
const subdomainCheckSeq = latestOnly()

const checkSubdomain = (value: string) => {
    const subdomain = value.toLowerCase().trim()
    if (subdomainCheckTimer) clearTimeout(subdomainCheckTimer)
    subdomainStatus.value = null

    if (!subdomain || subdomain.length < 3) {
        if (subdomain.length > 0) subdomainStatus.value = { available: false, reason: 'Subdomain must be 3-32 characters' }
        return
    }
    if (!/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/.test(subdomain)) {
        subdomainStatus.value = { available: false, reason: 'Lowercase alphanumeric and hyphens only' }
        return
    }

    subdomainChecking.value = true
    subdomainCheckTimer = setTimeout(async () => {
        const seq = subdomainCheckSeq.begin()
        try {
            const res = await apiFetch<{ available: boolean; reason?: string }>(
                '/api/user/domains/check-subdomain', { query: { subdomain } }
            )
            if (!subdomainCheckSeq.isCurrent(seq)) return
            subdomainStatus.value = res
        } catch {
            if (!subdomainCheckSeq.isCurrent(seq)) return
            subdomainStatus.value = null
        } finally {
            // Only the newest check owns the spinner — an older one resolving
            // late must not clear it while a newer request is still in flight.
            if (subdomainCheckSeq.isCurrent(seq)) subdomainChecking.value = false
        }
    }, 400)
}

// Entity search for domain editor
const entitySearchQuery = ref('')
const entitySearchResults = ref<any[]>([])
const entitySearching = ref(false)
let entitySearchTimer: ReturnType<typeof setTimeout> | null = null
const entitySearchSeq = latestOnly()

const searchEntities = (q: string) => {
    entitySearchQuery.value = q
    if (entitySearchTimer) clearTimeout(entitySearchTimer)
    if (!q || q.length < 2) { entitySearchResults.value = []; return }
    entitySearchTimer = setTimeout(async () => {
        entitySearching.value = true
        const seq = entitySearchSeq.begin()
        try {
            const data = await apiFetch<{ hits: any[] }>('/api/search', { params: { q, limit: 10 } })
            if (!entitySearchSeq.isCurrent(seq)) return
            entitySearchResults.value = (data.hits || []).filter((r: any) =>
                ['character', 'corporation', 'alliance'].includes(r.type) && r.id
            ).map((r: any) => ({ id: Number(r.id.split('_')[1]), name: r.name, type: r.type }))
        } catch {
            if (entitySearchSeq.isCurrent(seq)) entitySearchResults.value = []
        } finally {
            if (entitySearchSeq.isCurrent(seq)) entitySearching.value = false
        }
    }, 300)
}

const closeDomainEditor = () => {
    domainEditing.value = null
    domainEditBaseline.value = null
    subdomainStatus.value = null
    subdomainChecking.value = false
    const { edit, ...rest } = route.query
    router.replace({ query: Object.keys(rest).length > 0 ? rest : undefined })
}

const startCreateDomain = () => {
    domainEditing.value = {
        subdomain: '',
        entities: [],
        theme: { showLogoInBanner: true, showNameInBanner: true, showDescriptionInBanner: true },
        navbar_links: [],
        widgets: {
            top: [{ type: 'mostValuable', enabled: true }],
            left: [
                { type: 'topCharacters', enabled: true },
                { type: 'topCorporations', enabled: true },
                { type: 'topAlliances', enabled: true },
                { type: 'topShips', enabled: true },
                { type: 'topSystems', enabled: true },
                { type: 'topRegions', enabled: true },
            ],
            right: [{ type: 'killList', enabled: true, killlistType: 'latest' }],
            columnRatio: '250px_1fr',
        },
        site_name: '',
        site_description: '',
        campaign_policy: 0,
        campaigns: [],
        backgrounds: [],
    }
    domainError.value = ''
    domainEditBaseline.value = null
    domainEditTab.value = 'general'
    domainPresetId.value = 'default'
    router.replace({ query: { ...route.query, edit: 'new' } })
}

const startEditDomain = (domain: any) => {
    const theme = { ...(domain.theme || {}) }
    if (theme.showLogoInBanner === undefined) theme.showLogoInBanner = true
    if (theme.showNameInBanner === undefined) theme.showNameInBanner = true
    if (theme.showDescriptionInBanner === undefined) theme.showDescriptionInBanner = true
    const defaultWidgets = {
        top: [{ type: 'mostValuable', enabled: true }],
        left: [
            { type: 'topCharacters', enabled: true },
            { type: 'topCorporations', enabled: true },
            { type: 'topAlliances', enabled: true },
            { type: 'topShips', enabled: true },
            { type: 'topSystems', enabled: true },
            { type: 'topRegions', enabled: true },
        ],
        right: [{ type: 'killList', enabled: true, killlistType: 'latest' }],
        columnRatio: '250px_1fr',
    }
    domainEditing.value = {
        id: domain.id,
        subdomain: domain.subdomain,
        entities: [...(domain.entities || [])],
        theme,
        navbar_links: [...(domain.navbar_links || [])],
        widgets: domain.widgets ? JSON.parse(JSON.stringify(domain.widgets)) : defaultWidgets,
        site_name: domain.site_name || '',
        site_description: domain.site_description || '',
        campaign_policy: Number(domain.campaign_policy ?? 0),
        campaigns: [...(domain.campaigns || [])],
        backgrounds: [...(domain.backgrounds || [])],
        bannerAsset: domain.bannerAsset || null,
        logoAsset: domain.logoAsset || null,
    }
    domainEditBaseline.value = snapshotEditableDomain(domainEditing.value)
    domainError.value = ''
    domainEditTab.value = 'general'
    domainPresetId.value = theme.defaultThemePreset || 'default'
    router.replace({ query: { ...route.query, edit: String(domain.id) } })
}

const addEntityToDomain = (entity: any) => {
    if (!domainEditing.value) return
    const entities = domainEditing.value.entities
    if (entities.length >= 5) return
    if (entities.some((e: any) => e.type === entity.type && e.id === entity.id)) return
    entities.push({ type: entity.type, id: entity.id, name: entity.name })
    entitySearchQuery.value = ''
    entitySearchResults.value = []
}

const removeEntityFromDomain = (idx: number) => {
    if (!domainEditing.value) return
    domainEditing.value.entities.splice(idx, 1)
}

/**
 * Panel opacity as a percentage of the site default, where 100 is unchanged.
 *
 * Absent means "never touched", which has to keep meaning 100 rather than
 * being written on load — the theme patch only sends keys that actually
 * changed, and seeding it would push a no-op key on every save.
 */
const contentOpacity = computed({
    get: () => {
        const stored = Number(domainEditing.value?.theme?.contentOpacity)
        return Number.isFinite(stored) ? Math.min(100, Math.max(20, stored)) : 100
    },
    set: (value: number) => {
        if (!domainEditing.value) return
        domainEditing.value.theme.contentOpacity = Math.min(100, Math.max(20, Number(value) || 100))
    },
})

// Campaign selector for custom-domain curation.
const campaignSearchQuery = ref('')
const campaignSearchResults = ref<any[]>([])
const campaignSearching = ref(false)
let campaignSearchTimer: ReturnType<typeof setTimeout> | null = null
const campaignSearchSeq = latestOnly()

// Everything this user has made — the search endpoint returns it when given
// no query. Pre-shown under the box so their own campaigns (private ones
// especially, which are invisible everywhere else) are one click from being
// added, instead of only findable by typing a name they have to remember.
const ownCampaigns = ref<any[]>([])

const loadOwnCampaigns = async () => {
    const domainId = domainEditing.value?.id
    if (!domainId) return
    try {
        const data = await apiFetch<{ campaigns: any[] }>(
            `/api/user/domains/${domainId}/campaigns/search`,
        )
        if (domainEditing.value?.id !== domainId) return
        ownCampaigns.value = data.campaigns || []
    } catch {
        ownCampaigns.value = []
    }
}

watch(() => domainEditing.value?.id, (domainId) => {
    ownCampaigns.value = []
    campaignSearchQuery.value = ''
    campaignSearchResults.value = []
    if (domainId) void loadOwnCampaigns()
}, { immediate: true })

const selectedCampaignIds = computed(() =>
    new Set((domainEditing.value?.campaigns || []).map((campaign: any) => campaign.campaign_id)))

const campaignSearchActive = computed(() => campaignSearchQuery.value.trim().length >= 2)

const unselected = (list: any[]) =>
    list.filter((campaign: any) => !selectedCampaignIds.value.has(campaign.campaign_id))

const campaignSearchMatches = computed(() => unselected(campaignSearchResults.value))
const ownCampaignSuggestions = computed(() => unselected(ownCampaigns.value))

const searchDomainCampaigns = (q: string) => {
    campaignSearchQuery.value = q
    if (campaignSearchTimer) clearTimeout(campaignSearchTimer)
    if (!q || q.length < 2 || !domainEditing.value?.id) {
        campaignSearchResults.value = []
        return
    }
    campaignSearchTimer = setTimeout(async () => {
        campaignSearching.value = true
        const seq = campaignSearchSeq.begin()
        try {
            const data = await apiFetch<{ campaigns: any[] }>(
                `/api/user/domains/${domainEditing.value.id}/campaigns/search`,
                { params: { q } },
            )
            if (!campaignSearchSeq.isCurrent(seq)) return
            campaignSearchResults.value = data.campaigns || []
        } catch {
            if (campaignSearchSeq.isCurrent(seq)) campaignSearchResults.value = []
        } finally {
            if (campaignSearchSeq.isCurrent(seq)) campaignSearching.value = false
        }
    }, 300)
}

// Adding drops the row from both lists on its own — they filter on what is
// already selected.
const addDomainCampaign = (campaign: any) => {
    if (!domainEditing.value) return
    if ((domainEditing.value.campaigns || []).some((item: any) => item.campaign_id === campaign.campaign_id)) return
    domainEditing.value.campaigns.push(campaign)
}

const removeDomainCampaign = (campaignId: string) => {
    if (!domainEditing.value) return
    domainEditing.value.campaigns = (domainEditing.value.campaigns || [])
        .filter((campaign: any) => campaign.campaign_id !== campaignId)
}

// Who on this board may read a private campaign it has selected: its signed-in
// members only (default), or every visitor. Inert for public and killboard-only
// campaigns — those already show to anyone who can see the board.
const toggleCampaignAudience = (campaignId: string) => {
    const campaign = (domainEditing.value?.campaigns || [])
        .find((item: any) => item.campaign_id === campaignId)
    if (campaign) campaign.public_on_domain = !campaign.public_on_domain
}

const addNavbarItem = () => {
    if (!domainEditing.value) return
    domainEditing.value.navbar_links.push({ label: '', href: '/' })
}

const removeNavbarItem = (idx: number) => {
    if (!domainEditing.value) return
    domainEditing.value.navbar_links.splice(idx, 1)
}

const moveNavbarItem = (idx: number, dir: -1 | 1) => {
    if (!domainEditing.value) return
    const arr = domainEditing.value.navbar_links
    const newIdx = idx + dir
    if (newIdx < 0 || newIdx >= arr.length) return
    const temp = arr[idx]
    arr[idx] = arr[newIdx]
    arr[newIdx] = temp
}

const addNavbarGroup = (itemIdx: number) => {
    if (!domainEditing.value) return
    const item = domainEditing.value.navbar_links[itemIdx]
    if (!item.children) item.children = []
    item.children.push({ label: '', items: [] })
}

const removeNavbarGroup = (itemIdx: number, groupIdx: number) => {
    domainEditing.value?.navbar_links[itemIdx]?.children?.splice(groupIdx, 1)
}

const addNavbarChild = (itemIdx: number, groupIdx: number) => {
    const group = domainEditing.value?.navbar_links[itemIdx]?.children?.[groupIdx]
    if (group) group.items.push({ label: '', href: '/' })
}

const removeNavbarChild = (itemIdx: number, groupIdx: number, childIdx: number) => {
    domainEditing.value?.navbar_links[itemIdx]?.children?.[groupIdx]?.items.splice(childIdx, 1)
}

const navbarEditingIdx = ref<number | null>(null)

// Image upload state
const bannerUploading = ref(false)
const logoUploading = ref(false)
const bannerInput = useTemplateRef<HTMLInputElement>('bannerInput')
const logoInput = useTemplateRef<HTMLInputElement>('logoInput')

// bannerAsset / logoAsset on domainEditing is the latest non-approved
// submission for that slot. Status is 'pending' or 'rejected'; null when
// the slot is clean (either never uploaded or currently approved).
const bannerPending = computed(() => domainEditing.value?.bannerAsset?.status === 'pending')
const logoPending = computed(() => domainEditing.value?.logoAsset?.status === 'pending')
const bannerRejected = computed(() => domainEditing.value?.bannerAsset?.status === 'rejected')
const logoRejected = computed(() => domainEditing.value?.logoAsset?.status === 'rejected')

const uploadDomainImage = async (type: 'banner' | 'logo', file: File) => {
    if (!domainEditing.value?.id) {
        domainError.value = 'Save the domain first before uploading images'
        return
    }
    const uploading = type === 'banner' ? bannerUploading : logoUploading
    uploading.value = true
    domainError.value = ''
    try {
        const fd = new FormData()
        fd.append('type', type)
        fd.append('file', file)
        const result = await apiFetch<{ assetId: number; status: string }>(`/api/user/domains/${domainEditing.value.id}/upload`, {
            method: 'POST',
            body: fd,
        })
        // Mark as pending — the image won't show until admin approves.
        // Optimistic update so the UI flips before we refetch.
        if (domainEditing.value) {
            const asset = {
                id: result.assetId,
                status: result.status || 'pending',
                reject_reason: null,
                created_at: new Date().toISOString(),
            }
            if (type === 'banner') domainEditing.value.bannerAsset = asset
            else domainEditing.value.logoAsset = asset
        }
        await refreshDomains()
    } catch (err: any) {
        domainError.value = err?.data?.message || err?.message || `Failed to upload ${type}`
    } finally {
        uploading.value = false
    }
}

const removeDomainImage = async (type: 'banner' | 'logo') => {
    if (!domainEditing.value?.id) {
        if (type === 'banner') domainEditing.value!.theme.bannerUrl = ''
        else domainEditing.value!.theme.logoUrl = ''
        return
    }
    const uploading = type === 'banner' ? bannerUploading : logoUploading
    uploading.value = true
    domainError.value = ''
    try {
        await apiFetch(`/api/user/domains/${domainEditing.value.id}/upload`, {
            method: 'DELETE',
            body: { type },
        })
        if (type === 'banner') {
            domainEditing.value.theme.bannerUrl = ''
        } else {
            domainEditing.value.theme.logoUrl = ''
        }
        await refreshDomains()
    } catch (err: any) {
        domainError.value = err?.data?.message || err?.message || `Failed to remove ${type}`
    } finally {
        uploading.value = false
    }
}

const handleImageDrop = (type: 'banner' | 'logo', e: DragEvent) => {
    e.preventDefault()
    const file = e.dataTransfer?.files?.[0]
    if (file && file.type.startsWith('image/')) uploadDomainImage(type, file)
}

const handleImageSelect = (type: 'banner' | 'logo', e: Event) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (file) uploadDomainImage(type, file)
}

// ── Background uploads (up to 8 per domain) ─────────────────────────────────
const MAX_BACKGROUNDS = 8
const backgroundUploading = ref(false)
const backgroundInput = useTemplateRef<HTMLInputElement>('backgroundInput')

const uploadDomainBackground = async (file: File) => {
    if (!domainEditing.value?.id) {
        domainError.value = 'Save the domain first before uploading backgrounds'
        return
    }
    if ((domainEditing.value.backgrounds?.length ?? 0) >= MAX_BACKGROUNDS) {
        domainError.value = `Maximum ${MAX_BACKGROUNDS} backgrounds per domain`
        return
    }
    backgroundUploading.value = true
    domainError.value = ''
    try {
        const fd = new FormData()
        fd.append('type', 'background')
        fd.append('file', file)
        const res = await apiFetch<{ assetId: number; status: string }>(
            `/api/user/domains/${domainEditing.value.id}/upload`,
            { method: 'POST', body: fd },
        )
        // Optimistically add the new asset to the editor list as pending — the
        // navbar picker won't show it until admin approves.
        if (res?.assetId) {
            domainEditing.value.backgrounds = [
                ...(domainEditing.value.backgrounds || []),
                { id: res.assetId, status: res.status || 'pending', domain_id: domainEditing.value.id, created_at: new Date().toISOString() },
            ]
        }
        await refreshDomains()
    } catch (err: any) {
        domainError.value = err?.data?.message || err?.message || 'Failed to upload background'
    } finally {
        backgroundUploading.value = false
    }
}

const removeDomainBackground = async (assetId: number) => {
    if (!domainEditing.value?.id) return
    domainError.value = ''
    try {
        await apiFetch(`/api/user/domains/${domainEditing.value.id}/assets/${assetId}`, { method: 'DELETE' })
        domainEditing.value.backgrounds = (domainEditing.value.backgrounds || []).filter(
            (b: any) => b.id !== assetId,
        )
        await refreshDomains()
    } catch (err: any) {
        domainError.value = err?.data?.message || err?.message || 'Failed to remove background'
    }
}

const handleBackgroundDrop = (e: DragEvent) => {
    e.preventDefault()
    const file = e.dataTransfer?.files?.[0]
    if (file && file.type.startsWith('image/')) uploadDomainBackground(file)
}

const handleBackgroundSelect = (e: Event) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (file) uploadDomainBackground(file)
}

const domainSaved = ref(false)
const domainEditTab = ref<'general' | 'campaigns' | 'appearance' | 'navigation' | 'widgets'>('general')
const domainPresetId = ref('default')

const setDomainThemePreset = (presetId: string) => {
    if (!domainEditing.value) return
    domainPresetId.value = presetId
    const preset = THEME_PRESETS.find(p => p.id === presetId)
    if (preset) {
        domainEditing.value.theme.defaultThemePreset = presetId === 'default' ? undefined : presetId
        domainEditing.value.theme.defaultThemeOverrides = presetId === 'default' ? undefined : { ...preset.overrides } as Record<string, string>
    }
}

// ── Widget management ────────────────────────────────────────────────────────
const WIDGET_LABELS: Record<string, string> = {
    mostValuable: 'Most Valuable Kills',
    killList: 'Kill List',
    topCharacters: 'Top Characters',
    topCorporations: 'Top Corporations',
    topAlliances: 'Top Alliances',
    topShips: 'Top Ships',
    topSystems: 'Top Systems',
    topRegions: 'Top Regions',
    entityInfo: 'Entity Info',
    textBlock: 'Text Block',
    campaigns: 'Campaigns',
}

const WIDGET_ICONS: Record<string, string> = {
    mostValuable: 'lucide:gem',
    killList: 'lucide:swords',
    topCharacters: 'lucide:user',
    topCorporations: 'lucide:building-2',
    topAlliances: 'lucide:shield',
    topShips: 'lucide:rocket',
    topSystems: 'lucide:map-pin',
    topRegions: 'lucide:globe',
    entityInfo: 'lucide:info',
    textBlock: 'lucide:file-text',
    campaigns: 'lucide:target',
}

const ALL_WIDGET_TYPES = Object.keys(WIDGET_LABELS)

const COLUMN_RATIO_OPTIONS = [
    { value: '250px_1fr', label: '250px / auto' },
    { value: '300px_1fr', label: '300px / auto' },
    { value: '1fr_1fr', label: '50% / 50%' },
    { value: '1fr_2fr', label: '33% / 66%' },
    { value: '1fr_3fr', label: '25% / 75%' },
]

const moveWidget = (section: 'top' | 'left' | 'right', idx: number, dir: -1 | 1) => {
    if (!domainEditing.value) return
    const arr = domainEditing.value.widgets[section]
    const newIdx = idx + dir
    if (newIdx < 0 || newIdx >= arr.length) return
    const temp = arr[idx]
    arr[idx] = arr[newIdx]
    arr[newIdx] = temp
}

const removeWidget = (section: 'top' | 'left' | 'right', idx: number) => {
    domainEditing.value?.widgets[section].splice(idx, 1)
}

const addWidget = (section: 'top' | 'left' | 'right', type: string) => {
    if (!domainEditing.value) return
    const widget: any = { type, enabled: true }
    if (type === 'textBlock') widget.content = ''
    if (type === 'killList') widget.killlistType = 'latest'
    domainEditing.value.widgets[section].push(widget)
}

const availableWidgets = (section: 'top' | 'left' | 'right') => {
    if (!domainEditing.value) return []
    const existing = new Set(domainEditing.value.widgets[section].map((w: any) => w.type))
    return ALL_WIDGET_TYPES.filter(t => t === 'textBlock' || !existing.has(t))
}

const widgetAddOpen = ref<string | null>(null)

const SERVER_MANAGED_THEME_KEYS = new Set(['bannerUrl', 'logoUrl'])

const cloneEditableValue = <T>(value: T): T =>
    JSON.parse(JSON.stringify(value))

const snapshotEditableDomain = (domain: any) => ({
    entities: cloneEditableValue(domain.entities || []),
    theme: cloneEditableValue(domain.theme || {}),
    navbar_links: cloneEditableValue(
        domain.navbar_links.filter((link: any) => link.label && link.href),
    ),
    widgets: cloneEditableValue(domain.widgets || {}),
    site_name: domain.site_name || null,
    site_description: domain.site_description || null,
    campaign_policy: Number(domain.campaign_policy ?? 0),
    campaign_ids: (domain.campaigns || []).map((campaign: any) => campaign.campaign_id),
    campaign_public_ids: (domain.campaigns || [])
        .filter((campaign: any) => campaign.public_on_domain)
        .map((campaign: any) => campaign.campaign_id),
})

const valuesEqual = (left: unknown, right: unknown) =>
    JSON.stringify(left) === JSON.stringify(right)

const buildThemePatch = (
    current: Record<string, any>,
    previous: Record<string, any>,
) => {
    const patch: Record<string, any> = {}
    const keys = new Set([...Object.keys(previous || {}), ...Object.keys(current || {})])
    for (const key of keys) {
        if (SERVER_MANAGED_THEME_KEYS.has(key)) continue
        const currentHasKey = Object.prototype.hasOwnProperty.call(current, key)
        const currentValue = currentHasKey && current[key] !== undefined
            ? current[key]
            : null
        if (!valuesEqual(currentValue, previous?.[key] ?? null)) {
            patch[key] = currentValue
        }
    }
    return patch
}

const buildDomainPatch = (current: any, previous: Record<string, any>) => {
    const snapshot = snapshotEditableDomain(current)
    const patch: Record<string, any> = {}

    for (const key of [
        'entities',
        'navbar_links',
        'widgets',
        'site_name',
        'site_description',
        'campaign_policy',
        'campaign_ids',
        'campaign_public_ids',
    ]) {
        if (!valuesEqual(snapshot[key], previous[key])) {
            patch[key] = snapshot[key]
        }
    }

    // The server reads campaign_public_ids only alongside campaign_ids (it is a
    // per-selection flag, rewritten with the selection), so send them together
    // whenever either moved.
    if (patch.campaign_ids !== undefined || patch.campaign_public_ids !== undefined) {
        patch.campaign_ids = snapshot.campaign_ids
        patch.campaign_public_ids = snapshot.campaign_public_ids
    }

    const theme = buildThemePatch(snapshot.theme || {}, previous.theme || {})
    if (Object.keys(theme).length > 0) patch.theme = theme
    return patch
}

const saveDomain = async () => {
    if (!domainEditing.value) return
    domainSaving.value = true
    domainError.value = ''
    domainSaved.value = false
    try {
        const d = domainEditing.value
        const createBody = {
            subdomain: d.subdomain,
            ...snapshotEditableDomain(d),
        }
        const body = d.id
            ? buildDomainPatch(d, domainEditBaseline.value || {})
            : createBody
        if (d.id && Object.keys(body).length === 0) {
            domainSaved.value = true
            setTimeout(() => { domainSaved.value = false }, 2000)
            return
        }
        const res = d.id
            ? await apiFetch<{ domain: any }>(`/api/user/domains/${d.id}`, { method: 'PATCH', body })
            : await apiFetch<{ domain: any }>('/api/user/domains', { method: 'POST', body })
        if (!d.id && res?.domain?.id) {
            // Persist the new id so subsequent uploads work
            domainEditing.value.id = res.domain.id
            router.replace({ query: { ...route.query, edit: String(res.domain.id) } })
        }
        // Re-hydrate the local editing state from the server response so any
        // server-managed fields (e.g. theme.bannerUrl / theme.logoUrl set by
        // admin approval) become visible in the editor and survive the next
        // save round-trip. Only data fields are touched — UI state (active
        // tab, validation hints) is left alone.
        if (res?.domain && domainEditing.value) {
            const fresh = res.domain
            if (fresh.theme) domainEditing.value.theme = { ...fresh.theme }
            if (fresh.entities) domainEditing.value.entities = [...fresh.entities]
            if (fresh.navbar_links) domainEditing.value.navbar_links = [...fresh.navbar_links]
            if (fresh.widgets) domainEditing.value.widgets = JSON.parse(JSON.stringify(fresh.widgets))
            domainEditing.value.campaign_policy = Number(fresh.campaign_policy ?? domainEditing.value.campaign_policy ?? 0)
        }
        if (domainEditing.value?.id) {
            domainEditBaseline.value = snapshotEditableDomain(domainEditing.value)
        }
        domainSaved.value = true
        setTimeout(() => { domainSaved.value = false }, 2000)
        refreshDomains()
    } catch (err: any) {
        domainError.value = err?.data?.message || err?.message || 'Failed to save domain'
    } finally {
        domainSaving.value = false
    }
}

const deleteDomain = async (id: number) => {
    try {
        await apiFetch(`/api/user/domains/${id}`, { method: 'DELETE' })
        await refreshDomains()
    } catch (err: any) {
        domainError.value = err?.data?.message || err?.message || 'Failed to delete domain'
    }
}

// Restore domain editor state from ?edit= query param after mount
// Deferred to onMounted to avoid SSR/hydration mismatch
onMounted(() => {
    const tryRestore = () => {
        const data = domainsData.value
        if (!data?.domains || domainEditing.value) return
        const editParam = route.query.edit as string | undefined
        if (!editParam) return
        if (editParam === 'new') {
            startCreateDomain()
        } else {
            const id = Number(editParam)
            const domain = data.domains.find((d: any) => d.id === id)
            if (domain) startEditDomain(domain)
        }
    }
    // If data is already loaded, restore immediately (after hydration)
    if (domainsData.value) {
        tryRestore()
    } else {
        // Wait for data to arrive
        const stop = watch(() => domainsData.value, (data) => {
            if (data) { tryRestore(); stop() }
        })
    }
})
</script>

<template>
    <div class="space-y-4">
        <!-- Domain list -->
        <div v-if="!domainEditing" class="space-y-4">
            <div class="flex items-center justify-between">
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Custom Domains</h2>
                <button
                    class="px-3 py-1.5 rounded-md text-xs font-medium bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 transition-colors"
                    @click="startCreateDomain"
                >
                    + Create Domain
                </button>
            </div>

            <div v-if="domainsList.length === 0" class="glass-panel p-8 text-center">
                <Icon name="lucide:globe" class="text-3xl text-gray-600 mb-3" />
                <p class="text-sm text-gray-500">No custom domains yet.</p>
                <p class="text-xs text-gray-600 mt-1">Create a subdomain to get your own personalized killboard.</p>
            </div>

            <div v-for="domain in domainsList" :key="domain.id" class="glass-panel p-4">
                <div class="flex items-start justify-between gap-4">
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2 mb-1">
                            <a :href="`https://${domain.subdomain}.eve-kill.com`" target="_blank" rel="noopener" class="text-sm font-bold text-blue-400 hover:text-blue-300 transition-colors">
                                {{ domain.subdomain }}.eve-kill.com
                            </a>
                            <span class="px-1.5 py-0.5 rounded text-fine font-medium" :class="domain.active ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'">
                                {{ domain.active ? 'Active' : 'Inactive' }}
                            </span>
                        </div>
                        <div v-if="domain.site_name" class="text-xs text-gray-400 mb-2">{{ domain.site_name }}</div>
                        <div class="flex flex-wrap gap-1.5">
                            <span v-for="entity in (domain.entities || [])" :key="`${entity.type}-${entity.id}`"
                                class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-fine bg-white/[0.06] text-gray-300">
                                <img
                                    :src="`/images/${entity.type === 'character' ? 'characters' : entity.type === 'corporation' ? 'corporations' : 'alliances'}/${entity.id}/${entity.type === 'character' ? 'portrait' : 'logo'}?size=32`"
                                    class="w-4 h-4 rounded-full"
                                >
                                {{ entity.name }}
                            </span>
                        </div>
                    </div>
                    <div class="flex items-center gap-1 flex-shrink-0">
                        <button class="p-1.5 rounded text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors" @click="startEditDomain(domain)">
                            <Icon name="lucide:pencil" class="text-sm" />
                        </button>
                        <button class="p-1.5 rounded text-gray-500 hover:text-red-400 hover:bg-red-500/10 transition-colors" @click="deleteDomain(domain.id)">
                            <Icon name="lucide:trash-2" class="text-sm" />
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Domain editor (create/edit) -->
        <div v-else class="space-y-4">
            <!-- Header + tabs -->
            <div class="flex items-center gap-3 mb-2">
                <button class="p-1.5 rounded text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors" @click="closeDomainEditor()">
                    <Icon name="lucide:arrow-left" class="text-base" />
                </button>
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">
                    {{ domainEditing.id ? 'Edit Domain' : 'Create Domain' }}
                </h2>
            </div>

            <div class="flex gap-1 mb-2">
                <button
                    class="px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer"
                    :class="domainEditTab === 'general' ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    @click="domainEditTab = 'general'"
                >General</button>
                <button
                    class="px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer"
                    :class="domainEditTab === 'campaigns' ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    @click="domainEditTab = 'campaigns'"
                >Campaigns</button>
                <button
                    class="px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer"
                    :class="domainEditTab === 'appearance' ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    @click="domainEditTab = 'appearance'"
                >Appearance</button>
                <button
                    class="px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer"
                    :class="domainEditTab === 'widgets' ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    @click="domainEditTab = 'widgets'"
                >Widgets</button>
                <button
                    class="px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer"
                    :class="domainEditTab === 'navigation' ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                    @click="domainEditTab = 'navigation'"
                >Navigation</button>
            </div>

            <div v-if="domainError" class="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-sm text-red-400">
                {{ domainError }}
            </div>

            <!-- ═══ General Tab ═══ -->
            <template v-if="domainEditTab === 'general'">
                <!-- Subdomain -->
                <div class="glass-panel p-4">
                    <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">Subdomain</label>
                    <div class="flex items-center gap-0">
                        <input
                            v-model="domainEditing.subdomain"
                            type="text"
                            placeholder="mysite"
                            :disabled="!!domainEditing.id"
                            class="flex-1 px-3 py-2 rounded-l-md bg-white/[0.04] border text-sm text-white placeholder-gray-600 focus:outline-none disabled:opacity-50"
                            :class="!domainEditing.id && subdomainStatus
                                ? subdomainStatus.available
                                    ? 'border-green-500/50 focus:border-green-500/50'
                                    : 'border-red-500/50 focus:border-red-500/50'
                                : 'border-white/[0.08] focus:border-blue-500/50'"
                            @input="!domainEditing.id && checkSubdomain(($event.target as HTMLInputElement).value)"
                        >
                        <span class="px-3 py-2 rounded-r-md bg-white/[0.02] border border-l-0 border-white/[0.08] text-sm text-gray-500">.eve-kill.com</span>
                    </div>
                    <div v-if="!domainEditing.id && subdomainChecking" class="mt-1.5 text-xs text-gray-500">Checking availability...</div>
                    <div v-else-if="!domainEditing.id && subdomainStatus" class="mt-1.5 text-xs" :class="subdomainStatus.available ? 'text-green-400' : 'text-red-400'">
                        {{ subdomainStatus.available ? 'Subdomain is available' : subdomainStatus.reason }}
                    </div>
                </div>

                <!-- Entities -->
                <div class="glass-panel p-4">
                    <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">Entities ({{ domainEditing.entities.length }}/5)</label>
                    <div class="flex flex-wrap gap-1.5 mb-3">
                        <span v-for="(entity, idx) in domainEditing.entities" :key="`${entity.type}-${entity.id}`"
                            class="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-white/[0.06] text-xs text-gray-300">
                            <img
                                :src="`/images/${entity.type === 'character' ? 'characters' : entity.type === 'corporation' ? 'corporations' : 'alliances'}/${entity.id}/${entity.type === 'character' ? 'portrait' : 'logo'}?size=32`"
                                class="w-4 h-4 rounded-full"
                            >
                            {{ entity.name }}
                            <button class="ml-0.5 text-gray-500 hover:text-red-400" @click="removeEntityFromDomain(Number(idx))">
                                <Icon name="lucide:x" class="text-xs" />
                            </button>
                        </span>
                    </div>
                    <p v-if="domainEditing.entities.length >= 5" class="text-xs text-yellow-400/80">Maximum of 5 entities reached.</p>
                    <div v-if="domainEditing.entities.length < 5" class="relative">
                        <input
                            :value="entitySearchQuery"
                            @input="searchEntities(($event.target as HTMLInputElement).value)"
                            type="text"
                            placeholder="Search characters, corporations, alliances..."
                            class="w-full px-3 py-2 rounded-md bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50"
                        >
                        <div v-if="entitySearchResults.length > 0" class="absolute z-10 mt-1 w-full rounded-lg bg-gray-900 border border-white/[0.1] shadow-xl max-h-64 overflow-y-auto">
                            <button
                                v-for="result in entitySearchResults"
                                :key="`${result.type}-${result.id}`"
                                class="w-full flex items-center gap-3 px-3 py-2.5 text-sm text-gray-300 hover:bg-blue-500/[0.08] transition-colors"
                                @click="addEntityToDomain(result)"
                            >
                                <img
                                    :src="`/images/${result.type === 'character' ? 'characters' : result.type === 'corporation' ? 'corporations' : 'alliances'}/${result.id}/${result.type === 'character' ? 'portrait' : 'logo'}?size=64`"
                                    class="w-8 h-8 rounded-full flex-shrink-0"
                                >
                                <div class="flex-1 min-w-0 text-left">
                                    <div class="text-sm text-white truncate">{{ result.name }}</div>
                                    <div class="text-fine text-gray-500 capitalize">{{ result.type }}</div>
                                </div>
                            </button>
                        </div>
                    </div>
                </div>

                <!-- Site Info -->
                <div class="glass-panel p-4 space-y-3">
                    <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Site Info</label>
                    <input
                        v-model="domainEditing.site_name"
                        type="text"
                        placeholder="Site name (e.g. My Corp Killboard)"
                        class="w-full px-3 py-2 rounded-md bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50"
                    >
                    <textarea
                        v-model="domainEditing.site_description"
                        placeholder="Site description..."
                        rows="2"
                        class="w-full px-3 py-2 rounded-md bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 resize-none"
                    ></textarea>
                </div>

            </template>

            <!-- ═══ Campaigns Tab ═══ -->
            <template v-if="domainEditTab === 'campaigns'">
                <div class="glass-panel p-4 space-y-4">
                    <div>
                        <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">Campaign display policy</label>
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                            <button
                                class="rounded-lg border p-3 text-left transition-colors"
                                :class="domainEditing.campaign_policy === 0
                                    ? 'border-blue-500/40 bg-blue-500/[0.08]'
                                    : 'border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.05]'"
                                @click="domainEditing.campaign_policy = 0"
                            >
                                <div class="text-sm font-medium text-white">Automatic + selected</div>
                                <p class="text-fine text-gray-600 mt-1">Show campaigns involving this killboard’s entities, plus campaigns selected below.</p>
                            </button>
                            <button
                                class="rounded-lg border p-3 text-left transition-colors"
                                :class="domainEditing.campaign_policy === 1
                                    ? 'border-blue-500/40 bg-blue-500/[0.08]'
                                    : 'border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.05]'"
                                @click="domainEditing.campaign_policy = 1"
                            >
                                <div class="text-sm font-medium text-white">Selected only</div>
                                <p class="text-fine text-gray-600 mt-1">Reject every campaign from this killboard unless it is explicitly selected below.</p>
                            </button>
                        </div>
                    </div>

                    <div class="border-t border-white/[0.06] pt-4">
                        <div class="flex items-center justify-between mb-2">
                            <label class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">
                                Selected campaigns ({{ (domainEditing.campaigns || []).length }}/50)
                            </label>
                        </div>
                        <p class="text-fine text-gray-600 mb-3">
                            Your own campaigns are listed below — or paste a campaign ID / search by name to find others.
                            Your private campaigns become visible here only to signed-in members of this domain’s entities.
                        </p>

                        <div v-if="!domainEditing.id" class="rounded-lg border border-yellow-500/20 bg-yellow-500/[0.06] p-3 text-xs text-yellow-400">
                            Create the domain first, then reopen it to select campaigns.
                        </div>
                        <div v-else class="relative">
                            <div class="relative">
                                <Icon name="lucide:search" class="absolute left-3 top-2.5 text-sm text-gray-600" />
                                <input
                                    :value="campaignSearchQuery"
                                    type="text"
                                    placeholder="Campaign name or ID…"
                                    class="w-full pl-9 pr-3 py-2 rounded-md bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50"
                                    @input="searchDomainCampaigns(($event.target as HTMLInputElement).value)"
                                >
                                <Icon v-if="campaignSearching" name="lucide:loader-2" class="absolute right-3 top-2.5 text-sm text-gray-500 animate-spin" />
                            </div>
                            <div v-if="campaignSearchActive && campaignSearchMatches.length" class="absolute z-20 mt-1 w-full rounded-lg bg-gray-900 border border-white/[0.1] shadow-xl max-h-72 overflow-y-auto">
                                <button
                                    v-for="campaign in campaignSearchMatches"
                                    :key="campaign.campaign_id"
                                    class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-blue-500/[0.08] transition-colors"
                                    @click="addDomainCampaign(campaign)"
                                >
                                    <Icon
                                        :name="campaign.visibility === 1 ? 'lucide:lock' : campaign.visibility === 2 ? 'lucide:shield' : 'lucide:globe'"
                                        class="text-sm text-gray-500 flex-shrink-0"
                                    />
                                    <div class="flex-1 min-w-0">
                                        <div class="text-sm text-white truncate">{{ campaign.name }}</div>
                                        <div class="text-fine text-gray-600 font-mono">{{ campaign.campaign_id }}</div>
                                    </div>
                                    <span class="text-fine text-blue-400">Add</span>
                                </button>
                            </div>
                        </div>

                        <!-- Idle state of the picker: everything you made, ready to add. -->
                        <div v-if="domainEditing.id && !campaignSearchActive && ownCampaignSuggestions.length" class="mt-3">
                            <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-1.5">Your campaigns</div>
                            <div class="rounded-lg border border-white/[0.06] divide-y divide-white/[0.04] max-h-60 overflow-y-auto">
                                <button
                                    v-for="campaign in ownCampaignSuggestions"
                                    :key="campaign.campaign_id"
                                    class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-blue-500/[0.08] transition-colors"
                                    @click="addDomainCampaign(campaign)"
                                >
                                    <Icon
                                        :name="campaign.visibility === 1 ? 'lucide:lock' : campaign.visibility === 2 ? 'lucide:shield' : 'lucide:globe'"
                                        class="text-sm flex-shrink-0"
                                        :class="campaign.visibility === 1 ? 'text-yellow-400' : campaign.visibility === 2 ? 'text-blue-400' : 'text-green-400'"
                                    />
                                    <div class="flex-1 min-w-0">
                                        <div class="text-sm text-white truncate">{{ campaign.name }}</div>
                                        <div class="text-fine text-gray-600 font-mono">{{ campaign.campaign_id }}</div>
                                    </div>
                                    <span class="text-fine text-gray-600">
                                        {{ campaign.visibility === 1 ? 'Private' : campaign.visibility === 2 ? 'Killboard' : 'Public' }}
                                    </span>
                                    <span class="text-fine text-blue-400">Add</span>
                                </button>
                            </div>
                        </div>

                        <div v-if="!(domainEditing.campaigns || []).length" class="py-6 text-center text-xs text-gray-600">
                            No campaigns selected.
                        </div>
                        <div v-else class="space-y-1.5 mt-4">
                            <div
                                v-for="campaign in domainEditing.campaigns"
                                :key="campaign.campaign_id"
                                class="flex items-center gap-3 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5"
                            >
                                <Icon
                                    :name="campaign.visibility === 1 ? 'lucide:lock' : campaign.visibility === 2 ? 'lucide:shield' : 'lucide:globe'"
                                    class="text-sm"
                                    :class="campaign.visibility === 1 ? 'text-yellow-400' : campaign.visibility === 2 ? 'text-blue-400' : 'text-green-400'"
                                />
                                <div class="flex-1 min-w-0">
                                    <NuxtLink :to="`/campaign/${campaign.campaign_id}`" class="text-sm text-gray-200 hover:text-blue-400 truncate block">
                                        {{ campaign.name }}
                                    </NuxtLink>
                                    <span class="text-fine text-gray-600 font-mono">{{ campaign.campaign_id }}</span>
                                </div>
                                <!-- Audience only means anything for private campaigns; the
                                     other two are readable by any visitor already. -->
                                <button
                                    v-if="campaign.visibility === 1"
                                    class="inline-flex items-center gap-1.5 px-2 py-1 rounded border text-fine font-medium transition-colors cursor-pointer flex-shrink-0"
                                    :class="campaign.public_on_domain
                                        ? 'border-amber-500/30 bg-amber-500/10 text-amber-300 hover:bg-amber-500/20'
                                        : 'border-white/[0.08] bg-white/[0.04] text-gray-400 hover:bg-white/[0.07] hover:text-gray-200'"
                                    v-tooltip="campaign.public_on_domain
                                        ? 'Anyone visiting this killboard can read this private campaign — click for members only'
                                        : 'Only signed-in members of this killboard’s entities can read it — click to show everyone'"
                                    @click="toggleCampaignAudience(campaign.campaign_id)"
                                >
                                    <Icon :name="campaign.public_on_domain ? 'lucide:eye' : 'lucide:users'" class="text-fine" />
                                    {{ campaign.public_on_domain ? 'Everyone' : 'Members' }}
                                </button>
                                <span v-else class="text-fine text-gray-600">
                                    {{ campaign.visibility === 2 ? 'Killboard' : 'Public' }}
                                </span>
                                <button
                                    class="p-1.5 rounded text-gray-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                                    v-tooltip="'Remove campaign'"
                                    @click="removeDomainCampaign(campaign.campaign_id)"
                                >
                                    <Icon name="lucide:x" class="text-sm" />
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </template>

            <!-- ═══ Navigation Tab ═══ -->
            <template v-if="domainEditTab === 'navigation'">
                <div class="glass-panel p-4 space-y-3">
                    <div class="flex items-center justify-between">
                        <label class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Navigation Menu</label>
                        <div class="flex items-center gap-2">
                            <button
                                class="px-2 py-1 rounded text-fine text-gray-400 hover:text-yellow-400 hover:bg-yellow-500/10 transition-colors"
                                @click="domainEditing.navbar_links = JSON.parse(JSON.stringify(DEFAULT_NAVBAR))"
                            >Reset to Default</button>
                            <button
                                class="px-2 py-1 rounded text-fine text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                                @click="domainEditing.navbar_links = []"
                            >Clear All</button>
                            <button
                                v-if="domainEditing.navbar_links.length < 15"
                                class="px-2 py-1 rounded text-fine text-blue-400 hover:bg-blue-500/10 transition-colors"
                                @click="addNavbarItem"
                            >+ Add Item</button>
                        </div>
                    </div>
                    <p class="text-fine text-gray-600">Configure the navigation bar. Items can be plain links or dropdowns with grouped sub-items. Empty = default EVE-KILL navigation.</p>

                    <!-- Preview bar -->
                    <div class="flex items-center gap-1 p-2 rounded-lg bg-black/30 border border-white/[0.06] overflow-x-auto">
                        <span class="px-2 py-1 rounded text-xs text-white/80 bg-white/[0.06] flex-shrink-0">Home</span>
                        <template v-for="(item, idx) in domainEditing.navbar_links" :key="'preview-' + idx">
                            <span
                                class="px-2 py-1 rounded text-xs flex-shrink-0 cursor-pointer transition-colors"
                                :class="navbarEditingIdx === Number(idx) ? 'text-blue-400 bg-blue-500/10' : 'text-white/60 hover:text-white/80 hover:bg-white/[0.04]'"
                                @click="navbarEditingIdx = navbarEditingIdx === Number(idx) ? null : Number(idx)"
                            >
                                {{ item.label || '...' }}
                                <Icon v-if="item.children?.length" name="lucide:chevron-down" class="text-[9px] ml-0.5 opacity-50 inline" />
                            </span>
                        </template>
                    </div>

                    <div v-if="domainEditing.navbar_links.length === 0" class="py-6 text-center text-fine text-gray-600">
                        No custom navigation. The default menu will be shown.
                    </div>

                    <!-- Top-level items -->
                    <div v-for="(item, idx) in domainEditing.navbar_links" :key="idx" class="rounded-lg border overflow-hidden" :class="navbarEditingIdx === Number(idx) ? 'border-blue-500/30' : 'border-white/[0.06]'">
                        <!-- Item header -->
                        <div class="flex items-center gap-2 p-2.5 bg-white/[0.02]">
                            <button class="p-1 text-gray-500 hover:text-white disabled:opacity-30" :disabled="Number(idx) === 0" @click="moveNavbarItem(Number(idx), -1)">
                                <Icon name="lucide:chevron-up" class="text-sm" />
                            </button>
                            <button class="p-1 text-gray-500 hover:text-white disabled:opacity-30" :disabled="Number(idx) === domainEditing.navbar_links.length - 1" @click="moveNavbarItem(Number(idx), 1)">
                                <Icon name="lucide:chevron-down" class="text-sm" />
                            </button>
                            <input v-model="item.label" type="text" placeholder="Label" class="w-24 px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white placeholder-gray-600">
                            <input v-model="item.href" type="text" placeholder="/path or https://..." class="flex-1 px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white placeholder-gray-600">
                            <label class="flex items-center gap-1 text-fine text-gray-500 cursor-pointer">
                                <input v-model="item.external" type="checkbox" class="rounded border-white/[0.08]">
                                Ext
                            </label>
                            <button
                                class="p-1 transition-colors"
                                :class="navbarEditingIdx === Number(idx) ? 'text-blue-400' : 'text-gray-500 hover:text-blue-400'"
                                v-tooltip="navbarEditingIdx === Number(idx) ? 'Collapse' : 'Edit submenu'"
                                @click="navbarEditingIdx = navbarEditingIdx === Number(idx) ? null : Number(idx)"
                            >
                                <Icon :name="navbarEditingIdx === Number(idx) ? 'lucide:chevron-up' : 'lucide:layers'" class="text-sm" />
                            </button>
                            <button class="p-1 text-gray-500 hover:text-red-400" @click="removeNavbarItem(Number(idx))">
                                <Icon name="lucide:x" class="text-sm" />
                            </button>
                        </div>

                        <!-- Submenu editor (expanded) -->
                        <div v-if="navbarEditingIdx === Number(idx)" class="p-3 border-t border-white/[0.06] space-y-3 bg-white/[0.01]">
                            <div class="flex items-center justify-between">
                                <span class="text-fine text-gray-500">Dropdown groups ({{ item.children?.length || 0 }}) — multiple named groups create a mega menu</span>
                                <button class="px-2 py-1 rounded text-fine text-blue-400 hover:bg-blue-500/10 transition-colors" @click="addNavbarGroup(Number(idx))">+ Add Group</button>
                            </div>
                            <p v-if="!item.children?.length" class="text-fine text-gray-600 py-2">No dropdown — this item is a plain link. Add a group to make it a dropdown menu.</p>

                            <div v-for="(group, gIdx) in (item.children || [])" :key="gIdx" class="rounded-md border border-white/[0.06] p-2.5 space-y-2">
                                <div class="flex items-center gap-2">
                                    <input v-model="group.label" type="text" placeholder="Column header (optional — leave blank for simple dropdown)" class="flex-1 px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white placeholder-gray-600">
                                    <button class="p-1 text-gray-500 hover:text-red-400" v-tooltip="'Remove group'" @click="removeNavbarGroup(Number(idx), Number(gIdx))">
                                        <Icon name="lucide:x" class="text-sm" />
                                    </button>
                                </div>
                                <!-- Group items -->
                                <div v-for="(child, cIdx) in group.items" :key="cIdx" class="flex items-center gap-2 ml-3">
                                    <Icon name="lucide:corner-down-right" class="text-xs text-gray-600 flex-shrink-0" />
                                    <input v-model="child.label" type="text" placeholder="Label" class="w-24 px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white placeholder-gray-600">
                                    <input v-model="child.href" type="text" placeholder="/path" class="flex-1 px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white placeholder-gray-600">
                                    <button class="p-1 text-gray-500 hover:text-red-400" @click="removeNavbarChild(Number(idx), Number(gIdx), Number(cIdx))">
                                        <Icon name="lucide:x" class="text-xs" />
                                    </button>
                                </div>
                                <button class="ml-3 px-2 py-1 rounded text-fine text-blue-400 hover:bg-blue-500/10 transition-colors" @click="addNavbarChild(Number(idx), Number(gIdx))">+ Add Link</button>
                            </div>
                        </div>
                    </div>
                </div>
            </template>

            <!-- ═══ Appearance Tab ═══ -->
            <template v-if="domainEditTab === 'appearance'">
                <!-- Default Theme Preset -->
                <div class="glass-panel p-4 space-y-3">
                    <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Default Theme</label>
                    <p class="text-fine text-gray-600">The default color theme visitors will see. They can still override it with their own preference.</p>
                    <div class="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 gap-2">
                        <button
                            v-for="preset in THEME_PRESETS" :key="preset.id"
                            class="relative flex flex-col items-center gap-1.5 p-2.5 rounded-lg border transition-colors cursor-pointer"
                            :class="domainPresetId === preset.id
                                ? 'border-blue-500/50 bg-blue-500/10'
                                : 'border-white/[0.06] hover:border-white/[0.12] bg-white/[0.02]'"
                            @click="setDomainThemePreset(preset.id)"
                        >
                            <div class="flex gap-0.5">
                                <div class="w-4 h-4 rounded-full border border-white/10" :style="{ backgroundColor: preset.overrides.brandPrimary || '#3b82f6' }"></div>
                                <div class="w-4 h-4 rounded-full border border-white/10" :style="{ backgroundColor: preset.overrides.brandAccent || '#00aaff' }"></div>
                                <div class="w-4 h-4 rounded-full border border-white/10" :style="{ backgroundColor: preset.overrides.colorSuccess || '#10b981' }"></div>
                            </div>
                            <span class="text-fine text-gray-400 text-center leading-tight">{{ preset.name }}</span>
                            <div v-if="domainPresetId === preset.id" class="absolute top-1 right-1 w-2 h-2 rounded-full bg-blue-400"></div>
                        </button>
                    </div>
                </div>

                <!-- Banner Image -->
                <div class="glass-panel p-4 space-y-3">
                    <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Banner Image</label>
                    <!-- Pending state (preview the submitted image so the user sees what admin will review) -->
                    <div v-if="bannerPending" class="relative rounded-lg overflow-hidden border border-yellow-500/30">
                        <img :src="`/api/domains/preview/${domainEditing.bannerAsset.id}`" alt="Pending banner" class="w-full h-32 object-cover opacity-70">
                        <div class="absolute top-2 left-2 flex items-center gap-1.5 px-2 py-1 rounded bg-yellow-500/90 text-black text-fine font-bold uppercase tracking-wider">
                            <Icon name="lucide:clock" class="text-sm" />
                            Pending approval
                        </div>
                    </div>
                    <!-- Rejected state -->
                    <div v-else-if="bannerRejected" class="flex items-start gap-3 p-4 rounded-lg border border-red-500/30 bg-red-500/5">
                        <Icon name="lucide:x-circle" class="text-lg text-red-400 flex-shrink-0" />
                        <div class="min-w-0 flex-1">
                            <p class="text-sm text-red-300 font-medium">Banner rejected</p>
                            <p v-if="domainEditing.bannerAsset?.reject_reason" class="text-fine text-gray-400 mt-0.5">{{ domainEditing.bannerAsset.reject_reason }}</p>
                            <p class="text-fine text-gray-500 mt-1">Upload a new banner below to submit it for review.</p>
                        </div>
                    </div>
                    <!-- Existing approved banner -->
                    <div v-else-if="domainEditing.theme.bannerUrl" class="relative rounded-lg overflow-hidden border border-white/[0.08]">
                        <img :src="domainEditing.theme.bannerUrl" alt="Banner preview" class="w-full h-32 object-cover">
                        <button
                            class="absolute top-2 right-2 p-1.5 rounded-md bg-black/60 text-white hover:bg-red-500/80 transition-colors"
                            :disabled="bannerUploading"
                            @click="removeDomainImage('banner')"
                        >
                            <Icon name="lucide:trash-2" class="text-sm" />
                        </button>
                    </div>
                    <!-- Upload zone (hidden while a pending submission is in review) -->
                    <div
                        v-if="!bannerPending"
                        class="relative flex flex-col items-center justify-center gap-2 p-6 rounded-lg border-2 border-dashed border-white/[0.08] hover:border-blue-500/30 transition-colors cursor-pointer"
                        :class="{ 'opacity-50 pointer-events-none': bannerUploading || !domainEditing.id }"
                        @dragover.prevent
                        @drop="handleImageDrop('banner', $event)"
                        @click="bannerInput?.click()"
                    >
                        <Icon v-if="bannerUploading" name="lucide:loader-2" class="text-2xl text-blue-400 animate-spin" />
                        <Icon v-else name="lucide:image-plus" class="text-2xl text-gray-600" />
                        <span class="text-xs text-gray-500">{{ bannerUploading ? 'Uploading...' : (bannerRejected ? 'Upload a replacement' : (domainEditing.theme.bannerUrl ? 'Drop image to replace' : 'Drop image or click to upload')) }}</span>
                        <span class="text-fine text-gray-600">Max 4 MB &middot; JPEG, PNG, WebP, GIF</span>
                        <input ref="bannerInput" type="file" accept="image/jpeg,image/png,image/webp,image/gif" class="hidden" @change="handleImageSelect('banner', $event)">
                    </div>
                    <p v-if="!domainEditing.id" class="text-fine text-gray-600 mt-1">Save the domain first to enable image uploads</p>
                </div>

                <!-- Logo Image -->
                <div class="glass-panel p-4 space-y-3">
                    <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Logo</label>
                    <div class="flex items-start gap-3">
                        <!-- Pending state (preview submitted logo) -->
                        <div v-if="logoPending" class="relative w-16 h-16 flex-shrink-0">
                            <img :src="`/api/domains/preview/${domainEditing.logoAsset.id}`" alt="Pending logo" class="w-16 h-16 rounded-lg object-cover border border-yellow-500/40 opacity-70">
                            <div class="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-yellow-500/90 text-black flex items-center justify-center" v-tooltip="'Pending approval'">
                                <Icon name="lucide:clock" class="text-xs" />
                            </div>
                        </div>
                        <!-- Existing approved logo -->
                        <div v-else-if="domainEditing.theme.logoUrl" class="relative">
                            <img :src="domainEditing.theme.logoUrl" alt="Logo preview" class="w-16 h-16 rounded-lg object-cover border border-white/[0.08]">
                            <button
                                class="absolute -top-1.5 -right-1.5 p-0.5 rounded-full bg-black/60 text-white hover:bg-red-500/80 transition-colors"
                                :disabled="logoUploading"
                                @click="removeDomainImage('logo')"
                            >
                                <Icon name="lucide:x" class="text-xs" />
                            </button>
                        </div>
                        <!-- Upload zone (hidden while a submission is in review) -->
                        <div
                            v-else-if="!logoPending"
                            class="flex flex-col items-center justify-center gap-1 w-16 h-16 rounded-lg border-2 border-dashed border-white/[0.08] hover:border-blue-500/30 transition-colors cursor-pointer"
                            :class="{ 'opacity-50 pointer-events-none': logoUploading || !domainEditing.id }"
                            @dragover.prevent
                            @drop="handleImageDrop('logo', $event)"
                            @click="logoInput?.click()"
                        >
                            <Icon v-if="logoUploading" name="lucide:loader-2" class="text-lg text-blue-400 animate-spin" />
                            <Icon v-else name="lucide:plus" class="text-lg text-gray-600" />
                            <input ref="logoInput" type="file" accept="image/jpeg,image/png,image/webp,image/gif" class="hidden" @change="handleImageSelect('logo', $event)">
                        </div>
                        <div class="flex-1 min-w-0 space-y-1">
                            <template v-if="logoPending">
                                <p class="text-xs text-yellow-400 font-medium">Pending approval</p>
                                <p class="text-fine text-gray-500">Your logo is awaiting admin review.</p>
                            </template>
                            <template v-else-if="logoRejected">
                                <p class="text-xs text-red-300 font-medium">Logo rejected</p>
                                <p v-if="domainEditing.logoAsset?.reject_reason" class="text-fine text-gray-400">{{ domainEditing.logoAsset.reject_reason }}</p>
                                <p class="text-fine text-gray-500">Upload a replacement to submit it for review.</p>
                            </template>
                            <template v-else>
                                <p class="text-xs text-gray-500">Square image works best</p>
                                <p class="text-fine text-gray-600">Max 2 MB &middot; JPEG, PNG, WebP, GIF</p>
                                <p v-if="!domainEditing.id" class="text-fine text-gray-600 mt-1">Save first to enable uploads</p>
                            </template>
                        </div>
                    </div>
                </div>

                <!-- Background Images -->
                <div class="glass-panel p-4 space-y-3">
                    <div class="flex items-center justify-between">
                        <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Backgrounds</label>
                        <span class="text-fine text-gray-600">{{ (domainEditing.backgrounds?.length || 0) }} / 8</span>
                    </div>
                    <p class="text-fine text-gray-600">
                        Upload up to 8 backgrounds your visitors can pick from in the navbar background dropdown. The first approved one is the domain default.
                    </p>
                    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
                        <div
                            v-for="bg in (domainEditing.backgrounds || [])"
                            :key="bg.id"
                            class="relative aspect-video rounded-lg overflow-hidden border border-white/[0.08] bg-black/20 group"
                        >
                            <img
                                :src="`/api/domains/preview/${bg.id}`"
                                alt="Background preview"
                                class="w-full h-full object-cover"
                                :class="{ 'opacity-50': bg.status === 'pending' }"
                            >
                            <div v-if="bg.status === 'pending'" class="absolute top-1 left-1 px-1.5 py-0.5 rounded bg-yellow-500/80 text-fine font-medium text-black">
                                Pending
                            </div>
                            <button
                                class="absolute top-1 right-1 p-1 rounded bg-black/60 text-white opacity-0 group-hover:opacity-100 hover:bg-red-500/80 transition-all"
                                aria-label="Remove background"
                                @click="removeDomainBackground(bg.id)"
                            >
                                <Icon name="lucide:trash-2" class="text-xs" />
                            </button>
                        </div>
                        <div
                            v-if="(domainEditing.backgrounds?.length || 0) < 8"
                            class="relative flex flex-col items-center justify-center gap-1 aspect-video rounded-lg border-2 border-dashed border-white/[0.08] hover:border-blue-500/30 transition-colors cursor-pointer"
                            :class="{ 'opacity-50 pointer-events-none': backgroundUploading || !domainEditing.id }"
                            @dragover.prevent
                            @drop="handleBackgroundDrop"
                            @click="backgroundInput?.click()"
                        >
                            <Icon v-if="backgroundUploading" name="lucide:loader-2" class="text-xl text-blue-400 animate-spin" />
                            <Icon v-else name="lucide:image-plus" class="text-xl text-gray-600" />
                            <span class="text-fine text-gray-600">{{ backgroundUploading ? 'Uploading...' : 'Add background' }}</span>
                            <input ref="backgroundInput" type="file" accept="image/jpeg,image/png,image/webp,image/gif" class="hidden" @change="handleBackgroundSelect">
                        </div>
                    </div>
                    <p class="text-fine text-gray-600">Max 6 MB each &middot; JPEG, PNG, WebP, GIF &middot; pending uploads need admin approval before visitors see them</p>
                    <p v-if="!domainEditing.id" class="text-fine text-gray-600">Save the domain first to enable background uploads</p>
                </div>

                <!-- Content Opacity — named for what the number measures, so
                     100% reads as solid rather than as fully see-through. -->
                <div class="glass-panel p-4 space-y-3">
                    <div class="flex items-baseline justify-between gap-3">
                        <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Content Opacity</label>
                        <span class="text-sm text-gray-300 tabular-nums">{{ contentOpacity }}%</span>
                    </div>
                    <p class="text-fine text-gray-600">
                        How solid the content panels are. 100% is the standard look; lower lets more of your
                        background image through. Worth checking against a busy background — your text sits on these
                        panels, and it stops being readable well before the slider runs out.
                    </p>
                    <input
                        v-model.number="contentOpacity"
                        type="range" min="20" max="100" step="5"
                        class="w-full accent-blue-500 cursor-pointer"
                    >
                    <div class="flex justify-between text-fine text-gray-600">
                        <span>20% · see-through</span>
                        <button v-if="contentOpacity !== 100" class="text-blue-400 hover:text-blue-300 cursor-pointer"
                            @click="contentOpacity = 100">Reset to default</button>
                        <span>100% · solid</span>
                    </div>
                </div>

                <!-- Banner Display -->
                <div class="glass-panel p-4 space-y-3">
                    <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Banner Display</label>
                    <p class="text-fine text-gray-600">Choose what to show overlaid on the banner image.</p>
                    <div class="flex flex-col gap-2">
                        <label class="flex items-center gap-2.5 text-sm text-gray-300 cursor-pointer">
                            <input v-model="domainEditing.theme.showLogoInBanner" type="checkbox" class="rounded border-white/[0.08]">
                            Show logo
                        </label>
                        <label class="flex items-center gap-2.5 text-sm text-gray-300 cursor-pointer">
                            <input v-model="domainEditing.theme.showNameInBanner" type="checkbox" class="rounded border-white/[0.08]">
                            Show site name
                        </label>
                        <label class="flex items-center gap-2.5 text-sm text-gray-300 cursor-pointer">
                            <input v-model="domainEditing.theme.showDescriptionInBanner" type="checkbox" class="rounded border-white/[0.08]">
                            Show description
                        </label>
                    </div>

                    <div class="pt-3 border-t border-white/[0.06] space-y-2">
                        <label class="flex items-center gap-2.5 text-sm cursor-pointer"
                            :class="domainEditing.theme.bannerUrl ? 'text-gray-600 cursor-not-allowed' : 'text-gray-300'">
                            <input v-model="domainEditing.theme.transparentBanner" type="checkbox"
                                :disabled="!!domainEditing.theme.bannerUrl"
                                class="rounded border-white/[0.08] disabled:opacity-40">
                            Show transparent banner
                        </label>
                        <p class="text-fine text-gray-600">
                            {{ domainEditing.theme.bannerUrl
                                ? 'Not applicable while a banner image is set — remove the image to use it.'
                                : 'Keeps the header without an image: the same logo, name and description sit in a banner-shaped strip, with your background bleeding through and fading into the page.' }}
                        </p>
                    </div>
                </div>
            </template>

            <!-- ═══ Widgets Tab ═══ -->
            <template v-if="domainEditTab === 'widgets'">
                <!-- Column Ratio -->
                <div class="glass-panel p-4 space-y-3">
                    <label class="block text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Column Layout</label>
                    <p class="text-fine text-gray-600">Width ratio between the left sidebar and right main content.</p>
                    <select
                        v-model="domainEditing.widgets.columnRatio"
                        class="px-3 py-2 rounded-md bg-white/[0.04] border border-white/[0.08] text-sm text-white focus:outline-none focus:border-blue-500/50"
                    >
                        <option v-for="opt in COLUMN_RATIO_OPTIONS" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                    </select>
                </div>

                <!-- Widget sections -->
                <div v-for="section in (['top', 'left', 'right'] as const)" :key="section" class="glass-panel p-4 space-y-3">
                    <div class="flex items-center justify-between">
                        <label class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">
                            {{ section === 'top' ? 'Top Section (full width)' : section === 'left' ? 'Left Column (sidebar)' : 'Right Column (main)' }}
                        </label>
                        <div class="relative">
                            <button
                                class="px-2 py-1 rounded text-fine text-blue-400 hover:bg-blue-500/10 transition-colors"
                                @click="widgetAddOpen = widgetAddOpen === section ? null : section"
                            >+ Add Widget</button>
                            <div v-if="widgetAddOpen === section" class="absolute right-0 mt-1 z-10 w-48 rounded-lg bg-gray-900 border border-white/[0.1] shadow-xl overflow-hidden">
                                <button
                                    v-for="wt in availableWidgets(section)" :key="wt"
                                    class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-300 hover:bg-blue-500/[0.08] transition-colors text-left"
                                    @click="addWidget(section, wt); widgetAddOpen = null"
                                >
                                    <Icon :name="WIDGET_ICONS[wt] ?? 'lucide:box'" class="text-sm text-gray-500" />
                                    {{ WIDGET_LABELS[wt] }}
                                </button>
                                <p v-if="availableWidgets(section).length === 0" class="px-3 py-2 text-fine text-gray-600">All widgets added</p>
                            </div>
                        </div>
                    </div>

                    <div v-if="domainEditing.widgets[section].length === 0" class="py-4 text-center text-fine text-gray-600">
                        No widgets in this section
                    </div>

                    <div v-for="(widget, idx) in domainEditing.widgets[section]" :key="idx" class="flex items-center gap-2 p-2 rounded-lg" :class="widget.enabled ? 'bg-white/[0.02]' : 'bg-white/[0.01] opacity-50'">
                        <Icon :name="WIDGET_ICONS[widget.type] || 'lucide:box'" class="text-base text-gray-500 flex-shrink-0" />
                        <div class="flex-1 min-w-0">
                            <span class="text-sm text-white">{{ WIDGET_LABELS[widget.type] || widget.type }}</span>
                        </div>

                        <!-- TextBlock content editor -->
                        <div v-if="widget.type === 'textBlock' && widget.enabled" class="flex-1">
                            <textarea
                                v-model="widget.content"
                                placeholder="Enter text..."
                                rows="2"
                                class="w-full px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white placeholder-gray-600 resize-none"
                            ></textarea>
                        </div>

                        <!-- KillList type selector -->
                        <select
                            v-if="widget.type === 'killList'"
                            v-model="widget.killlistType"
                            class="px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white"
                        >
                            <option value="latest">Latest</option>
                            <option value="big">Big Kills</option>
                            <option value="solo">Solo Kills</option>
                            <option value="npc">NPC Kills</option>
                        </select>

                        <!-- Controls -->
                        <button
                            class="p-1 text-gray-500 hover:text-white disabled:opacity-30 transition-colors"
                            :disabled="Number(idx) === 0"
                            v-tooltip="'Move up'"
                            @click="moveWidget(section, Number(idx), -1)"
                        >
                            <Icon name="lucide:chevron-up" class="text-sm" />
                        </button>
                        <button
                            class="p-1 text-gray-500 hover:text-white disabled:opacity-30 transition-colors"
                            :disabled="Number(idx) === domainEditing.widgets[section].length - 1"
                            v-tooltip="'Move down'"
                            @click="moveWidget(section, Number(idx), 1)"
                        >
                            <Icon name="lucide:chevron-down" class="text-sm" />
                        </button>
                        <label class="flex items-center cursor-pointer" v-tooltip="widget.enabled ? 'Disable' : 'Enable'">
                            <input v-model="widget.enabled" type="checkbox" class="rounded border-white/[0.08]">
                        </label>
                        <button
                            class="p-1 text-gray-500 hover:text-red-400 transition-colors"
                            v-tooltip="'Remove'"
                            @click="removeWidget(section, Number(idx))"
                        >
                            <Icon name="lucide:x" class="text-sm" />
                        </button>
                    </div>
                </div>
            </template>

            <!-- Save (always visible) -->
            <div class="flex items-center gap-3">
                <button
                    class="px-4 py-2 rounded-md text-sm font-medium transition-colors disabled:opacity-50 cursor-pointer"
                    :class="domainSaved
                        ? 'bg-green-500 text-white'
                        : 'bg-blue-500 text-white hover:bg-blue-600'"
                    :disabled="domainSaving || !domainEditing.subdomain || domainEditing.entities.length === 0 || (!domainEditing.id && (!subdomainStatus || !subdomainStatus.available))"
                    @click="saveDomain"
                >
                    <template v-if="domainSaved"><Icon name="lucide:check" class="text-sm mr-1" />Saved</template>
                    <template v-else-if="domainSaving">Saving...</template>
                    <template v-else>{{ domainEditing.id ? 'Save Changes' : 'Create Domain' }}</template>
                </button>
                <button
                    class="px-4 py-2 rounded-md text-sm font-medium text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors"
                    @click="closeDomainEditor()"
                >
                    Cancel
                </button>
            </div>
        </div>
    </div>
</template>
