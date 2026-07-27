/**
 * Shared plumbing for the character / corporation / alliance / faction entity pages.
 *
 * Owns everything the three pages have in common: id parsing + validation,
 * the entity fetch (with 404 via error.vue for missing entities), the tab
 * system (route-segment tabs, default-tab redirect, analytics on change),
 * killlist role/params, the lazy top-lists fetch, the palette-derived accent,
 * and the shared derived stats (age, efficiency bar widths, danger ratio).
 *
 * Entity-specific concerns stay in the pages: header markup, SEO/schema.org
 * strings, per-entity tab content and helpers.
 *
 * Structure note: experimental.asyncContext is on, so the Nuxt instance
 * survives an `await` and composables can register in whatever order reads
 * best. The entity is fetched and awaited up front, which means a missing one
 * 404s before the rest of the page's composables register at all — previously
 * everything had to be declared first and the await deferred to the bottom,
 * so a 404 still paid for the SEO meta and the top-lists registration.
 */

export type EntityPageKind = 'character' | 'corporation' | 'alliance' | 'faction'

export interface EntityPageTab {
    id: string
    label: string
    icon: string
}

const KIND_LABELS: Record<EntityPageKind, string> = {
    character: 'Character',
    corporation: 'Corporation',
    alliance: 'Alliance',
    faction: 'Faction',
}

export async function useEntityPage<T extends EntityPageTab>(
    kind: EntityPageKind,
    opts: {
        /** Tab definitions for EntityTabBar — set and order differ per entity */
        tabs: readonly T[]
        /** Base document title (without the tab suffix), or null while the entity is unresolved */
        titleBase: (entity: any) => string | null
        /** Tab rendered at the bare base URL (no tab segment). Default 'dashboard'. */
        defaultTab?: T['id']
        /** Inclusive valid id range. Default [1, 2147483647]; factions use [500000, 599999]. */
        idRange?: [number, number]
        /** Fetch /top-lists for the kill-tab sidebars. Default true; faction has none. */
        withTopLists?: boolean
        /** Derive the palette accent + theme-color meta. Default true; faction has no palette. */
        withAccent?: boolean
    },
) {
    type TabId = T['id']

    const route = useRoute()
    const label = KIND_LABELS[kind]
    const id = Number(route.params.id)
    const defaultTab = (opts.defaultTab ?? 'dashboard') as TabId
    const [idMin, idMax] = opts.idRange ?? [1, 2147483647]

    const tabIds = new Set<string>(opts.tabs.map(t => t.id))

    if (!Number.isInteger(id) || id < idMin || id > idMax) {
        throw createError({ statusCode: 404, statusMessage: `${label} not found` })
    }

    // Unknown tab segments 404 instead of silently rendering the default tab.
    // /character/123/<anything> used to answer 200 with a canonical pointing at
    // /character/123, which is an unbounded set of URLs collapsing onto one
    // canonical — exactly what Bing flags as "a large number of pages pointing
    // to the same canonical URL". Checked here rather than in the activeTab
    // computed so it runs before the entity fetch and only on a real navigation
    // (definePageMeta keys the page by id, so setup does not re-run per tab).
    const tabParam = route.params.tab
    if (typeof tabParam === 'string' && tabParam.length > 0 && !tabIds.has(tabParam)) {
        throw createError({ statusCode: 404, statusMessage: `${label} tab not found` })
    }

    // Entity fetch, awaited immediately. Missing entities get a proper 404 via
    // error.vue (same pattern as kill and battle pages) — not fatal, this is an
    // expected response for a nonexistent ID rather than an exception.
    const { data, pending, error } = await useApiFetch<any>(`/api/${kind}/${id}`)

    if (error.value) {
        throw createError({
            statusCode: error.value.statusCode || 404,
            statusMessage: (error.value.data as any)?.message || `${label} not found`,
        })
    }

    const entity = computed(() => data.value?.[kind])
    const stats = computed(() => data.value?.stats)
    const recentStats = computed(() => data.value?.recentStats)

    // Identity accent derived from the entity's ESI palette (null when grayscale/absent,
    // or always for kinds without a palette — withAccent: false)
    const accent = computed(() => opts.withAccent === false ? null : entityAccent(entity.value?.palette))

    // Discord (and other link previews) color the embed strip from theme-color
    if (opts.withAccent !== false) {
        useSeoMeta({ themeColor: computed(() => accent.value?.accent) })
    }

    // Tab system with route segments — /character/123/kills
    // Tab from route param — reactive, no remount
    const activeTab = computed<TabId>(() => {
        const param = route.params.tab as string
        if (param && tabIds.has(param)) return param as TabId
        return defaultTab
    })

    const activeTabLabel = computed(() => {
        if (activeTab.value === defaultTab) return null
        return opts.tabs.find(t => t.id === activeTab.value)?.label ?? null
    })

    useHead({ title: computed(() => {
        const base = opts.titleBase(entity.value)
        if (!base) return label
        return activeTabLabel.value ? `${base} (${activeTabLabel.value})` : base
    }) })

    const setTab = (tabId: TabId) => {
        useAnalytics().track('tab.change', { entity: kind, tab: tabId })
        const path = tabId === defaultTab ? `/${kind}/${id}` : `/${kind}/${id}/${tabId}`
        navigateTo(path)
    }

    // Map active tab to killlist role
    const killlistRole = computed<'any' | 'attacker' | 'victim'>(() => {
        if (activeTab.value === 'kills') return 'attacker'
        if (activeTab.value === 'losses') return 'victim'
        return 'any'
    })

    // Pass through `?ship_group=` from the URL so EntityShipClasses click-throughs
    // filter the killlist to that hull group.
    const killlistExtraParams = computed(() => {
        const sg = route.query.ship_group
        return sg ? { ship_group: Number(sg) } : undefined
    })

    // Top lists for kill tab sidebars — no await, loads client-side only (don't block SSR).
    // Skipped entirely for kinds without a top-lists sidebar (withTopLists: false).
    const topLists = opts.withTopLists === false
        ? ref<any>(null)
        : useApiFetch<any>(`/api/entity/${kind}/${id}/top-lists`, {
            lazy: true,
            server: false,
            getCachedData: cachedPayload,
        }).data

    // Format date
    const formatDate = (iso: string | null): string => {
        if (!iso) return 'Unknown'
        return new Date(iso).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
    }

    // Entity age in whole years (birthday for characters, founding date otherwise)
    const ageYears = computed(() => entityAgeYears(kind === 'character' ? entity.value?.birthday : entity.value?.date_founded))

    // Derived stats
    const effWidth = computed(() => `${stats.value?.efficiency || 0}%`)
    const iskEffWidth = computed(() => `${stats.value?.isk_efficiency || 0}%`)

    const dangerRatio = computed(() => {
        const s = stats.value
        if (!s || s.kills + s.losses === 0) return 0
        return Math.round((1 - (s.npc_losses / Math.max(s.losses, 1))) * 100)
    })

    // Redirect to the user's preferred default tab (only when no tab in URL).
    // The entity already resolved above, so a nonexistent one has 404'd before
    // any redirect can fire.
    useDefaultTab(kind, `/${kind}/${id}`, defaultTab, tabIds)

    return {
        id,
        data,
        pending,
        entity,
        stats,
        recentStats,
        accent,
        activeTab,
        activeTabLabel,
        setTab,
        killlistRole,
        killlistExtraParams,
        topLists,
        formatDate,
        ageYears,
        effWidth,
        iskEffWidth,
        dangerRatio,
    }
}
