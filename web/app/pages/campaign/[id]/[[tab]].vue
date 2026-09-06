<script setup lang="ts">
import { campaignSideAccents } from '#shared/utils/entityAccent'

interface CampaignEntity {
    entity_type: number
    entity_id: number
    name: string | null
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
}

interface CampaignSide {
    side_index: number
    name: string
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
    palette?: { main_color?: string; secondary_color?: string; tertiary_color?: string } | null
    entities: CampaignEntity[]
}

interface CampaignAllowedEntity {
    type: 'character' | 'corporation' | 'alliance'
    id: number
    name: string | null
}

interface CampaignEditEntity {
    type: 'character' | 'corporation' | 'alliance'
    id: number
    name: string | null
}

interface CampaignEditSide {
    name: string
    entities: CampaignEditEntity[]
}

interface CampaignPrizeResult {
    rank: number
    character_id: number
    character_name: string | null
    metric_value: string
    secondary_value: string
    payout_percentage: number
    payout_amount: number
    claimed: boolean
    paid: boolean
    can_claim: boolean
}

interface CampaignPrizeContribution {
    id: string
    source: 'ek_wallet' | 'eve_wallet'
    contributor_id: number | null
    contributor_name: string
    contributor_type: 'character' | 'corporation' | 'alliance' | 'unknown'
    amount: string
    contributed_at: string
}

interface CampaignPrizePool {
    metric: number
    metric_label: string
    winner_count: number
    payout_percentages: number[]
    status: number
    funded_total: string
    contribution_count: number
    rules_locked_at: string | null
    finalized_at: string | null
    funding_reference: string
    funding_closes_at: string
    settles_at: string
    last_wallet_sync: string | null
    discord_url: string
    projected_lead_percent: number | null
    results: CampaignPrizeResult[]
    contributions: CampaignPrizeContribution[]
}

interface PrizeWalletBalance {
    balance: string
    totalBalance: string
    reservedBalance: string
    availableBalance: string
}

interface CampaignDetail {
    campaign_id: string
    mode: 'conflict' | 'area'
    name: string
    description: string | null
    status: number
    visibility: number
    allowed_entities: CampaignAllowedEntity[]
    start_time: string
    end_time: string | null
    location_details: {
        systems: Array<{ id: number; name: string }>
        constellations: Array<{ id: number; name: string }>
        regions: Array<{ id: number; name: string }>
    }
    stats: any | null
    last_activity_at: string | null
    stats_updated_at: string | null
    processing?: {
        paused: boolean
        note: string | null
        estimated_killmails: number | null
        last_started_at: string | null
        last_duration_ms: number | null
        last_killmails: number | null
        last_error: string | null
    }
    prize_pool: CampaignPrizePool | null
    creator: { character_id: number; name: string | null }
    sides: CampaignSide[]
    daily: { granularity: 'day' | 'week'; rows: Array<{ period: string; side_index: number; kills: number; losses: number; isk_destroyed: number; isk_lost: number }> }
}

const route = useRoute()
const router = useRouter()
const campaignId = String(route.params.id)

// Key the page by campaign id only — switching tabs updates the URL but
// reuses the component instance (no remount, no refetch). Same pattern as
// the battle report page.
definePageMeta({
    key: route => `/campaign/${route.params.id}`,
})

// ── Tabs (route-driven: /campaign/:id and /campaign/:id/:tab) ────────────────
const TABS = [
    { key: 'overview', label: 'Overview', icon: 'lucide:layout-dashboard' },
    { key: 'intel', label: 'Intel', icon: 'lucide:radar' },
    { key: 'comments', label: 'Comments', icon: 'lucide:message-square' },
] as const

const COMMENT_TARGET_CAMPAIGN = 10

const { data: campaign, pending, refresh, error } = await useApiFetch<CampaignDetail>(`/api/campaign/${campaignId}`)

if (error.value) {
    throw createError({ statusCode: error.value.statusCode ?? 404, message: 'Campaign not found', fatal: true })
}

const isAreaCampaign = computed(() => campaign.value?.mode === 'area')
const isOneSided = computed(() => !isAreaCampaign.value && (campaign.value?.sides.length ?? 0) === 1)
const visibleTabs = computed(() => isAreaCampaign.value ? TABS.filter(t => t.key !== 'intel') : TABS)

// Unknown tab segments 404 instead of falling back to the overview tab. Public
// campaigns are indexable and every tab URL canonicals to /campaign/<id>, so
// the fallback made an unbounded set of URLs share one canonical. `campaign` is
// awaited above, so visibleTabs is already settled here.
const campaignTabParam = route.params.tab
if (typeof campaignTabParam === 'string' && campaignTabParam.length > 0
    && !visibleTabs.value.some(t => t.key === campaignTabParam)) {
    throw createError({ statusCode: 404, statusMessage: 'Campaign tab not found' })
}

const activeTab = computed(() => {
    const t = String(route.params.tab || 'overview')
    return visibleTabs.value.some(x => x.key === t) ? t : 'overview'
})
const setTab = (t: string) => {
    router.push(t === 'overview' ? `/campaign/${campaignId}` : `/campaign/${campaignId}/${t}`)
}

const seoNumber = (value: unknown): number => {
    const number = Number(value)
    return Number.isFinite(number) ? number : 0
}
const seoDate = (value: string | null): string | null => {
    if (!value) return null
    const date = new Date(value)
    if (!Number.isFinite(date.getTime())) return null
    return new Intl.DateTimeFormat('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        timeZone: 'UTC',
    }).format(date)
}
const seoEntityImagePath = (entity: CampaignEntity): string => {
    if (entity.entity_type === 2) return `alliances/${entity.entity_id}`
    if (entity.entity_type === 1) return `corporations/${entity.entity_id}`
    return `characters/${entity.entity_id}`
}
const campaignSeoImage = computed(() => {
    const entity = campaign.value?.sides[0]?.entities[0]
    if (!entity) return 'https://eve-kill.com/og-default.png'
    const path = seoEntityImagePath(entity)
    const imageType = entity.entity_type === 0 ? 'portrait' : 'logo'
    return `/images/${path}/${imageType}?size=512`
})
const campaignSideNames = computed(() =>
    (campaign.value?.sides ?? []).map(side => side.name).filter(Boolean))
const campaignSeoTitle = computed(() => {
    const current = campaign.value
    if (!current) return 'EVE Online Campaign'
    const sides = campaignSideNames.value
    if (sides.length >= 2) return `${sides.join(' vs ')} — EVE Online Campaign`
    return `${current.name} — EVE Online Campaign`
})
const campaignSeoDescription = computed(() => {
    const current = campaign.value
    if (!current) return 'Track EVE Online campaign kills, losses, sides, and ISK destroyed on EVE-KILL.'

    const parts: string[] = []
    const sideNames = campaignSideNames.value
    if (current.mode === 'area') {
        const places = [
            ...current.location_details.systems.map(location => location.name),
            ...current.location_details.constellations.map(location => location.name),
            ...current.location_details.regions.map(location => location.name),
        ].slice(0, 4)
        parts.push(places.length
            ? `${current.name} tracks battlefield activity in ${places.join(', ')}.`
            : `${current.name} tracks EVE Online battlefield activity.`)
    } else if (sideNames.length) {
        parts.push(`${sideNames.join(' vs ')}.`)
    } else {
        parts.push(`${current.name}.`)
    }

    const totals = current.stats?.totals
    const killCount = seoNumber(totals?.killCount)
    const iskDestroyed = seoNumber(totals?.iskDestroyed)
    if (killCount || iskDestroyed) {
        const activity: string[] = []
        if (killCount) activity.push(`${formatNumber(killCount)} killmail${killCount === 1 ? '' : 's'}`)
        if (iskDestroyed) activity.push(`${formatIsk(iskDestroyed)} ISK destroyed`)
        parts.push(`${activity.join(' and ')}.`)
    }

    if (current.sides.length) {
        const sideStats = current.sides.slice(0, 4).map((side) => {
            const values = [`${formatNumber(seoNumber(side.kills))} kills`]
            if (seoNumber(side.isk_destroyed)) values.push(`${formatIsk(seoNumber(side.isk_destroyed))} ISK`)
            return `${side.name}: ${values.join(', ')}`
        })
        parts.push(`${sideStats.join('; ')}.`)
    }

    if (!killCount && !iskDestroyed && current.description) {
        const description = current.description.replace(/\s+/g, ' ').trim()
        if (description) parts.push(description.endsWith('.') ? description : `${description}.`)
    }

    const fundedPrize = seoNumber(current.prize_pool?.funded_total)
    if (current.prize_pool && fundedPrize > 0) {
        parts.push(
            `Prize pool: ${formatIsk(fundedPrize)} ISK for the top ${current.prize_pool.winner_count} by ${current.prize_pool.metric_label.toLowerCase()}.`,
        )
    }

    const start = seoDate(current.start_time)
    const end = seoDate(current.end_time)
    if (start) parts.push(end ? `${start}–${end}.` : `Running since ${start}.`)
    return parts.join(' ')
})
const campaignSocialDescription = computed(() => {
    const current = campaign.value
    if (!current) return 'Track EVE Online campaigns on EVE-KILL.'
    const totals = current.stats?.totals
    const summary: string[] = []
    if (campaignSideNames.value.length) summary.push(campaignSideNames.value.join(' vs '))
    const killCount = seoNumber(totals?.killCount)
    const iskDestroyed = seoNumber(totals?.iskDestroyed)
    if (killCount) summary.push(`${formatNumber(killCount)} killmails`)
    if (iskDestroyed) summary.push(`${formatIsk(iskDestroyed)} ISK destroyed`)
    const fundedPrize = seoNumber(current.prize_pool?.funded_total)
    if (fundedPrize > 0) summary.push(`${formatIsk(fundedPrize)} ISK prize pool`)
    return `${summary.join(' · ') || current.name} — campaign scoreboard on EVE-KILL.`
})
const campaignCanonicalUrl = `https://eve-kill.com/campaign/${campaignId}`

useHead({
    title: () => campaignSeoTitle.value,
    link: [{ rel: 'canonical', href: campaignCanonicalUrl }],
})
useSeoMeta({
    description: () => campaignSeoDescription.value,
    ogTitle: () => campaignSeoTitle.value,
    ogDescription: () => campaignSocialDescription.value,
    ogType: 'website',
    ogUrl: campaignCanonicalUrl,
    ogImage: () => campaignSeoImage.value,
    ogImageAlt: () => `${campaign.value?.name ?? 'EVE Online campaign'} on EVE-KILL`,
    twitterCard: 'summary',
    twitterTitle: () => campaignSeoTitle.value,
    twitterDescription: () => campaignSocialDescription.value,
    twitterImage: () => campaignSeoImage.value,
    twitterImageAlt: () => `${campaign.value?.name ?? 'EVE Online campaign'} on EVE-KILL`,
    // Non-public campaigns are never for search-engine discovery.
    robots: () => campaign.value && campaign.value.visibility !== 0
        ? 'noindex, nofollow'
        : 'index, follow, max-image-preview:large',
})

useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: [
            { name: 'Home', item: '/' },
            { name: 'Campaigns', item: '/campaigns' },
            { name: campaign.value?.name ?? 'Campaign', item: `/campaign/${campaignId}` },
        ],
    }))),
    {
        '@type': 'Event',
        'name': computed(() => campaignSeoTitle.value),
        'description': computed(() => campaignSeoDescription.value),
        'url': campaignCanonicalUrl,
        'image': computed(() => campaignSeoImage.value),
        'startDate': computed(() => campaign.value?.start_time || undefined),
        'endDate': computed(() => campaign.value?.end_time || undefined),
        'eventStatus': computed(() => {
            const current = campaign.value
            if (!current) return 'https://schema.org/EventScheduled'
            const ended = current.status === 2
                || (!!current.end_time && new Date(current.end_time).getTime() <= Date.now())
            return ended
                ? 'https://schema.org/EventCompleted'
                : 'https://schema.org/EventScheduled'
        }),
        'eventAttendanceMode': 'https://schema.org/OnlineEventAttendanceMode',
        'location': computed(() => {
            const current = campaign.value
            const system = current?.location_details.systems[0]
            if (system) {
                return {
                    '@type': 'Place',
                    'name': system.name,
                    'url': `https://eve-kill.com/system/${system.id}`,
                }
            }
            const constellation = current?.location_details.constellations[0]
            if (constellation) {
                return {
                    '@type': 'Place',
                    'name': constellation.name,
                    'url': `https://eve-kill.com/constellation/${constellation.id}`,
                }
            }
            const region = current?.location_details.regions[0]
            if (region) {
                return {
                    '@type': 'Place',
                    'name': region.name,
                    'url': `https://eve-kill.com/region/${region.id}`,
                }
            }
            return {
                '@type': 'VirtualLocation',
                'url': 'https://www.eveonline.com',
            }
        }),
        'attendee': computed(() => {
            const sides = campaign.value?.sides ?? []
            return sides.length
                ? sides.map(side => ({
                    '@type': 'Organization',
                    'name': side.name,
                }))
                : undefined
        }),
    },
])

const { user, isAuthenticated, isAdmin, login } = useAuth()
const canEdit = computed(() =>
    isAuthenticated.value && campaign.value
    && (isAdmin.value || user.value?.characterId === campaign.value.creator.character_id))

const prizeCopied = ref(false)
const prizePanelOpen = ref(false)
const prizeClaiming = ref(false)
const prizeActionMessage = ref('')
const prizeActionError = ref('')
const prizeWalletBalance = ref<PrizeWalletBalance | null>(null)
const prizeWalletLoading = ref(false)
const prizeContributionAmount = ref('')
const prizeContributionRequestId = ref('')
const prizeContributing = ref(false)
const prizeFundingOpen = computed(() => {
    const prize = campaign.value?.prize_pool
    return !!prize
        && prize.status === 0
        && new Date(prize.funding_closes_at).getTime() > Date.now()
})
const prizeSettling = computed(() => {
    const prize = campaign.value?.prize_pool
    return !!prize && prize.status === 0 && !prizeFundingOpen.value
})
const prizeWalletAvailable = computed(() =>
    Number(prizeWalletBalance.value?.availableBalance ?? prizeWalletBalance.value?.balance ?? 0))
const loadPrizeWalletBalance = async () => {
    if (!isAuthenticated.value) {
        prizeWalletBalance.value = null
        return
    }
    prizeWalletLoading.value = true
    try {
        prizeWalletBalance.value = await apiFetch<PrizeWalletBalance>('/api/user/wallet/balance')
    } catch {
        prizeWalletBalance.value = null
    } finally {
        prizeWalletLoading.value = false
    }
}
watch([prizePanelOpen, isAuthenticated], ([open, authenticated]) => {
    if (open && authenticated) void loadPrizeWalletBalance()
})
watch(prizeContributionAmount, () => {
    if (!prizeContributing.value) prizeContributionRequestId.value = ''
})
const usePrizeWalletMaximum = () => {
    const available = prizeWalletBalance.value?.availableBalance ?? prizeWalletBalance.value?.balance
    if (available) prizeContributionAmount.value = available
}
const contributeFromPrizeWallet = async () => {
    const amount = prizeContributionAmount.value.trim()
    prizeActionMessage.value = ''
    prizeActionError.value = ''
    if (!/^\d+(?:\.\d{1,2})?$/.test(amount) || Number(amount) <= 0) {
        prizeActionError.value = 'Enter a positive ISK amount with at most two decimals.'
        return
    }
    if (Number(amount) > prizeWalletAvailable.value) {
        prizeActionError.value = 'That exceeds your available EK Wallet balance.'
        return
    }
    if (!prizeContributionRequestId.value) {
        prizeContributionRequestId.value = crypto.randomUUID()
    }

    prizeContributing.value = true
    try {
        const result = await apiFetch<{ contributed: string }>(
            `/api/campaign/${campaignId}/prize/contribute`,
            {
                method: 'POST',
                body: {
                    amount,
                    requestId: prizeContributionRequestId.value,
                },
            },
        )
        prizeContributionAmount.value = ''
        prizeContributionRequestId.value = ''
        await Promise.all([refresh(), loadPrizeWalletBalance()])
        prizeActionMessage.value = `${formatIsk(Number(result.contributed))} added to the prize pool.`
    } catch (e: any) {
        prizeActionError.value = e.data?.message || e.message || 'Could not fund the prize pool'
    } finally {
        prizeContributing.value = false
    }
}
const copyPrizeReference = async () => {
    const reference = campaign.value?.prize_pool?.funding_reference
    if (!reference) return
    await navigator.clipboard.writeText(reference)
    prizeCopied.value = true
    window.setTimeout(() => { prizeCopied.value = false }, 2000)
}
const claimPrize = async () => {
    prizeClaiming.value = true
    prizeActionMessage.value = ''
    prizeActionError.value = ''
    try {
        await apiFetch(`/api/campaign/${campaignId}/prize/claim`, { method: 'POST' })
        prizeActionMessage.value = 'Prize claimed. Join Discord to arrange the in-game payment.'
        await refresh()
    } catch (e: any) {
        prizeActionError.value = e.data?.message || e.message || 'Could not claim prize'
    } finally {
        prizeClaiming.value = false
    }
}
const formatPrizeScore = (result: CampaignPrizeResult) =>
    campaign.value?.prize_pool?.metric === 0 || campaign.value?.prize_pool?.metric === 1
        ? formatNumber(Number(result.metric_value))
        : formatIsk(Number(result.metric_value))
const prizeContributorImage = (contribution: CampaignPrizeContribution) => {
    if (!contribution.contributor_id) return null
    if (contribution.contributor_type === 'character') {
        return `/images/characters/${contribution.contributor_id}/portrait?size=64`
    }
    if (contribution.contributor_type === 'corporation') {
        return `/images/corporations/${contribution.contributor_id}/logo?size=64`
    }
    if (contribution.contributor_type === 'alliance') {
        return `/images/alliances/${contribution.contributor_id}/logo?size=64`
    }
    return null
}

// ── Pending stats polling ────────────────────────────────────────────────────
const isComputing = computed(() =>
    !!campaign.value
    && !campaign.value.processing?.paused
    && (campaign.value.status === 0 || !campaign.value.stats))
let pollTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
    if (!isComputing.value) return
    let attempts = 0
    pollTimer = setInterval(async () => {
        attempts++
        await refresh()
        if (!isComputing.value || attempts >= 40) {
            if (pollTimer) clearInterval(pollTimer)
            pollTimer = null
        }
    }, 8000)
})
onUnmounted(() => { if (pollTimer) clearInterval(pollTimer) })

// ── Formatting & colors ──────────────────────────────────────────────────────
const SIDE_TEXT = ['text-red-400', 'text-blue-400', 'text-emerald-400', 'text-orange-400']
const SIDE_BORDER = ['border-red-500/20', 'border-blue-500/20', 'border-emerald-500/20', 'border-orange-500/20']
const SIDE_BAR = ['bg-red-500/70', 'bg-blue-500/70', 'bg-emerald-600/70', 'bg-orange-600/70']

// Corp-palette identity accents per side (all-or-nothing with hue
// separation — see campaignSideAccents). Null per side → fixed palette.
const sideAccents = computed(() => campaignSideAccents((campaign.value?.sides ?? []).map(s => s.palette)))
const sideAccentHexes = computed(() => sideAccents.value.map(a => a?.accent ?? null))

// Most-valuable tile borders follow the side that LOST the ship — corp
// accent when active, fixed side palette otherwise, neutral default border.
const MV_SIDE_BORDERS = ['rgba(239, 68, 68, 0.45)', 'rgba(59, 130, 246, 0.45)', 'rgba(16, 185, 129, 0.45)', 'rgba(249, 115, 22, 0.45)']
const mvTileStyle = (mv: { victimSide?: number | null }): Record<string, string> | undefined => {
    if (mv.victimSide == null) return undefined
    const color = sideAccents.value[mv.victimSide]?.border ?? MV_SIDE_BORDERS[mv.victimSide]
    return color ? { borderColor: color } : undefined
}


const efficiency = (s: CampaignSide): number => {
    const total = s.isk_destroyed + s.isk_lost
    return total > 0 ? (s.isk_destroyed / total) * 100 : 0
}

const entityImage = (e: CampaignEntity): string => {
    if (e.entity_type === 2) return `/images/alliances/${e.entity_id}/logo?size=64`
    if (e.entity_type === 1) return `/images/corporations/${e.entity_id}/logo?size=64`
    return `/images/characters/${e.entity_id}/portrait?size=64`
}
const entityLink = (e: CampaignEntity): string => {
    if (e.entity_type === 2) return `/alliance/${e.entity_id}`
    if (e.entity_type === 1) return `/corporation/${e.entity_id}`
    return `/character/${e.entity_id}`
}

// ── Activity chart ───────────────────────────────────────────────────────────
// Stacked SVG columns of per-side losses. Fixed fills are the validated
// dark-surface set (red-500 / blue-500 / emerald-600 / orange-600 — CVD-safe
// adjacent pairs); corp-palette accents override per side when active.
const CHART_SIDE_FILLS = ['#ef4444', '#3b82f6', '#059669', '#ea580c']

// One-sided campaigns chart kills vs losses instead of losses-per-side —
// with a single side, "who lost what" needs both halves of the ledger.
// Pseudo side ids keep the segment plumbing untouched.
const SEG_KILLS = 100
const SEG_LOSSES = 101

const segColor = (side: number): string => {
    if (side === -1) return '#3b82f6'
    if (side === SEG_KILLS) return '#059669'
    if (side === SEG_LOSSES) return '#ef4444'
    return sideAccentHexes.value[side] ?? CHART_SIDE_FILLS[side] ?? '#6b7280'
}

const chartMetric = ref<'ships' | 'isk'>('ships')
const hoverIdx = ref<number | null>(null)

// Logical viewBox geometry — scales with the container.
const VB = { w: 960, h: 190, left: 48, right: 10, top: 10, bottom: 24 }

// Fine-grained ceiling so the axis hugs the peak (a 5.1k peak tops out at
// 6k, not 10k) — max wasted headroom ~25%.
const NICE_STEPS = [1, 1.2, 1.5, 2, 2.5, 3, 4, 5, 6, 8, 10]
const niceCeil = (v: number): number => {
    if (v <= 4) return 4
    const p = 10 ** Math.floor(Math.log10(v))
    const m = v / p
    return (NICE_STEPS.find(s => s >= m) ?? 10) * p
}

const fmtTick = (v: number): string => {
    if (chartMetric.value === 'isk') return formatIsk(v)
    if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(1)}m`
    if (v >= 1_000) return `${(v / 1_000).toFixed(v >= 10_000 ? 0 : 1)}k`
    return String(Math.round(v))
}

const chart = computed(() => {
    const plot = { x0: VB.left, y0: VB.top, x1: VB.w - VB.right, y1: VB.h - VB.bottom }
    const rows = campaign.value?.daily?.rows ?? []
    if (!rows.length) return { bars: [] as any[], ticks: [] as any[], xLabels: [] as any[], step: 0, plot }
    const metric = chartMetric.value

    const periods = new Map<string, Record<number, { ships: number; isk: number }>>()
    for (const r of rows) {
        const p = periods.get(r.period) ?? {}
        if (isOneSided.value) {
            const k = p[SEG_KILLS] ?? { ships: 0, isk: 0 }
            k.ships += r.kills
            k.isk += r.isk_destroyed
            p[SEG_KILLS] = k
            const l = p[SEG_LOSSES] ?? { ships: 0, isk: 0 }
            l.ships += r.losses
            l.isk += r.isk_lost
            p[SEG_LOSSES] = l
        } else {
            const e = p[r.side_index] ?? { ships: 0, isk: 0 }
            e.ships += isAreaCampaign.value ? r.kills : r.losses
            e.isk += isAreaCampaign.value ? r.isk_destroyed : r.isk_lost
            p[r.side_index] = e
        }
        periods.set(r.period, p)
    }

    const plotW = plot.x1 - plot.x0
    const plotH = plot.y1 - plot.y0
    const ordered = [...periods.entries()].sort((a, b) => a[0].localeCompare(b[0]))
    const n = ordered.length
    const step = plotW / n
    const barW = Math.max(Math.min(step * 0.72, 28), 1.5)
    const GAP = 2

    const metricValue = (e: { ships: number; isk: number }) => (metric === 'isk' ? e.isk : e.ships)
    const rawMax = Math.max(...ordered.map(([, bySide]) => Object.values(bySide).reduce((a, e) => a + metricValue(e), 0)), 1)
    const niceMax = niceCeil(rawMax)

    const bars = ordered.map(([period, bySide], i) => {
        const segsIn = Object.entries(bySide)
            .map(([side, e]) => ({ side: Number(side), ships: e.ships, isk: e.isk, value: metricValue(e) }))
            .filter(s => s.value > 0)
            .sort((a, b) => a.side - b.side)
        const total = segsIn.reduce((a, s) => a + s.value, 0)
        const stackH = (total / niceMax) * plotH
        const gaps = GAP * Math.max(segsIn.length - 1, 0)
        // Shrink segments proportionally so the 2px surface gaps between them
        // don't push the stack top above the true total.
        const scale = stackH > gaps ? (stackH - gaps) / stackH : 0
        let cursor = plot.y1
        const segments = segsIn.map((s, si) => {
            const h = Math.max((s.value / niceMax) * plotH * scale, 1)
            cursor -= h
            const y = cursor
            cursor -= GAP
            return { ...s, y, h, topmost: si === segsIn.length - 1 }
        })
        return {
            period,
            x: plot.x0 + i * step + (step - barW) / 2,
            w: barW,
            total,
            totalShips: segsIn.reduce((a, s) => a + s.ships, 0),
            totalIsk: segsIn.reduce((a, s) => a + s.isk, 0),
            segments,
        }
    })

    const ticks = [1, 2, 3, 4].map(t => ({
        y: plot.y1 - (t / 4) * plotH,
        label: fmtTick((niceMax * t) / 4),
    }))

    const spansYears = n > 1 && ordered[0]![0].slice(0, 4) !== ordered[n - 1]![0].slice(0, 4)
    const fmtX = (period: string) => new Date(`${period}T00:00:00Z`).toLocaleDateString('en-US',
        spansYears
            ? { month: 'short', day: 'numeric', year: '2-digit', timeZone: 'UTC' }
            : { month: 'short', day: 'numeric', timeZone: 'UTC' })
    const labelStep = Math.max(1, Math.ceil(n / 7))
    const xLabels = bars.filter((_, i) => i % labelStep === 0).map(b => ({ x: b.x + b.w / 2, label: fmtX(b.period) }))

    return { bars, ticks, xLabels, step, plot }
})

const hoverBar = computed(() => (hoverIdx.value !== null ? chart.value.bars[hoverIdx.value] ?? null : null))
const tooltipLeft = computed(() => {
    const b = hoverBar.value
    if (!b) return '0%'
    const pct = ((b.x + b.w / 2) / VB.w) * 100
    return `${Math.min(Math.max(pct, 12), 88)}%`
})
const sideNameOf = (side: number): string =>
    side === -1 ? 'All activity'
    : side === SEG_KILLS ? 'Kills'
    : side === SEG_LOSSES ? 'Losses'
    : campaign.value?.sides.find(s => s.side_index === side)?.name ?? `Side ${side + 1}`
const fmtPeriodFull = (period: string): string => {
    const d = new Date(`${period}T00:00:00Z`).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric', timeZone: 'UTC' })
    return campaign.value?.daily?.granularity === 'week' ? `Week of ${d}` : d
}
// Rounded top corners only — data ends round, baseline stays square.
const topRoundedPath = (x: number, y: number, w: number, h: number): string => {
    const r = Math.min(3, w / 2, h)
    return `M${x},${y + h} L${x},${y + r} Q${x},${y} ${x + r},${y} L${x + w - r},${y} Q${x + w},${y} ${x + w},${y + r} L${x + w},${y + h} Z`
}

// ── Stats blob accessors ─────────────────────────────────────────────────────
const stats = computed(() => campaign.value?.stats ?? null)
const sideKey = (i: number) => String(i)

// Affiliation ticker tags on top-killer/victim rows. Purple = corporation,
// indigo = alliance — same hues as the overview tiles. Blobs computed before
// the ticker fields existed simply don't render tags until a recompute.
const CORP_TAG = 'px-1 py-px rounded border border-purple-500/20 bg-purple-500/[0.08] text-fine text-purple-300/80 flex-shrink-0'
const ALLY_TAG = 'px-1 py-px rounded border border-indigo-500/20 bg-indigo-500/[0.08] text-fine text-indigo-300/80 flex-shrink-0'

// ── Intel (worker-computed, per side) ────────────────────────────────────────
const intelFor = (side: number) => stats.value?.intelBySide?.[String(side)] ?? null
const fmtDamage = (v: number): string => {
    if (v >= 1_000_000_000) return `${(v / 1_000_000_000).toFixed(1)}b`
    if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(1)}m`
    if (v >= 1_000) return `${(v / 1_000).toFixed(0)}k`
    return String(Math.round(v))
}

// ── Killfeed side coloring: rows are tinted by the VICTIM's side color, so
// each team's losses are visually distinct instead of everything being the
// generic red loss highlight. Indexed by side_index to match SIDE_* colors.
const killlistSides = computed(() =>
    (campaign.value?.sides ?? []).map((s) => {
        const out = { characterIds: [] as number[], corporationIds: [] as number[], allianceIds: [] as number[] }
        for (const e of s.entities) {
            if (e.entity_type === 0) out.characterIds.push(e.entity_id)
            else if (e.entity_type === 1) out.corporationIds.push(e.entity_id)
            else out.allianceIds.push(e.entity_id)
        }
        return out
    }))

// ── Killfeed tab: null = combined; N = that side's KILLS (they're the
// attackers, victims are the other sides/neutrals). Keyed remount on switch
// so KillList's cursor stack resets cleanly.
const killfeedTab = ref<number | null>(null)

// ── Owner / admin controls ───────────────────────────────────────────────────
const editing = ref(false)
const editName = ref('')
const editDescription = ref('')
const editEndTime = ref('')
const editOngoing = ref(true)
const editVisibility = ref(0)
const saving = ref(false)
const editError = ref<string | null>(null)
const reauthRequired = ref(false)
const deleteModalOpen = ref(false)
const deleting = ref(false)
const actionError = ref<string | null>(null)
const actionMessage = ref<string | null>(null)
const editSides = ref<CampaignEditSide[]>([])
const originalSidesJSON = ref('')

// Killboard grants editor (area campaigns only — sides imply the audience
// otherwise). SearchPicker handles the lookup; only list state lives here.
const MAX_ALLOWED_ENTITIES = 10
const editAllowed = ref<CampaignAllowedEntity[]>([])

const addEditAllowed = (p: { type: string; id: number; name: string }) => {
    if (editAllowed.value.length >= MAX_ALLOWED_ENTITIES) return
    if (editAllowed.value.some(e => e.type === p.type && e.id === p.id)) return
    editAllowed.value.push({ type: p.type as CampaignAllowedEntity['type'], id: p.id, name: p.name })
}

const removeEditAllowed = (e: CampaignAllowedEntity) => {
    editAllowed.value = editAllowed.value.filter(x => !(x.type === e.type && x.id === e.id))
}

const campaignEntityType = (type: number): CampaignEditEntity['type'] =>
    type === 0 ? 'character' : type === 1 ? 'corporation' : 'alliance'

const normalizedEditSides = () => editSides.value.map(side => ({
    name: side.name.trim(),
    entities: side.entities.map(entity => ({ type: entity.type, id: entity.id })),
}))

const editSidesDirty = computed(() => JSON.stringify(normalizedEditSides()) !== originalSidesJSON.value)
const editSidesValid = computed(() => isAreaCampaign.value || (
    editSides.value.length > 0
    && editSides.value.length <= 4
    && editSides.value.every(side => side.entities.length > 0 && side.entities.length <= 15)
))

const addEditParticipant = (sideIndex: number, picked: { type: string; id: number; name: string }) => {
    const type = picked.type as CampaignEditEntity['type']
    for (const side of editSides.value) {
        side.entities = side.entities.filter(entity => !(entity.type === type && entity.id === picked.id))
    }
    editSides.value[sideIndex]?.entities.push({ type, id: picked.id, name: picked.name })
}

const removeEditParticipant = (sideIndex: number, entity: CampaignEditEntity) => {
    const side = editSides.value[sideIndex]
    if (side) side.entities = side.entities.filter(item => !(item.type === entity.type && item.id === entity.id))
}

const addEditSide = () => {
    if (editSides.value.length < 4) editSides.value.push({ name: `Side ${editSides.value.length + 1}`, entities: [] })
}

const removeEditSide = (sideIndex: number) => {
    editSides.value.splice(sideIndex, 1)
}

const openEdit = () => {
    if (!campaign.value) return
    editName.value = campaign.value.name
    editDescription.value = campaign.value.description ?? ''
    editOngoing.value = !campaign.value.end_time
    editEndTime.value = isoToEveInput(campaign.value.end_time)
    editVisibility.value = campaign.value.visibility
    editAllowed.value = [...(campaign.value.allowed_entities ?? [])]
    editSides.value = campaign.value.sides.map(side => ({
        name: side.name,
        entities: side.entities.map(entity => ({
            type: campaignEntityType(entity.entity_type),
            id: entity.entity_id,
            name: entity.name,
        })),
    }))
    originalSidesJSON.value = JSON.stringify(normalizedEditSides())
    editError.value = null
    reauthRequired.value = false
    editing.value = true
}

const saveEdit = async () => {
	if (!editSidesValid.value) {
		editError.value = 'Every side must contain 1–15 participants.'
		return
	}
    saving.value = true
    editError.value = null
    try {
        await apiFetch(`/api/campaign/${campaignId}/update`, {
            method: 'POST',
            body: {
                name: editName.value,
                description: editDescription.value || null,
                endTime: editOngoing.value ? null : (editEndTime.value ? eveInputToIso(editEndTime.value) : null),
                visibility: editVisibility.value,
                allowedEntities: editAllowed.value,
                ...(editSidesDirty.value ? { sides: normalizedEditSides() } : {}),
            },
        })
        editing.value = false
        await refresh()
    } catch (e: any) {
        const status = e.statusCode ?? e.response?.status ?? e.data?.statusCode
        reauthRequired.value = status === 401
        editError.value = reauthRequired.value
            ? 'This browser is no longer signed in. Sign in again to save your changes.'
            : extractFetchError(e, 'Failed to save')
    } finally {
        saving.value = false
    }
}

const toggleArchive = async () => {
    if (!campaign.value) return
    saving.value = true
    actionError.value = null
    actionMessage.value = null
    try {
        await apiFetch(`/api/campaign/${campaignId}/update`, {
            method: 'POST',
            body: { archived: campaign.value.status !== 2 },
        })
        await refresh()
        actionMessage.value = campaign.value?.status === 2 ? 'Campaign archived.' : 'Campaign reactivated.'
    } catch (e: any) {
        actionError.value = extractFetchError(e, 'Failed to update campaign archive status')
    } finally {
        saving.value = false
    }
}

const resumeProcessing = async () => {
    saving.value = true
    try {
        await apiFetch(`/api/campaign/${campaignId}/update`, {
            method: 'POST',
            body: { resumeProcessing: true },
        })
        await refresh()
    } finally {
        saving.value = false
    }
}

const deleteCampaign = async () => {
    deleting.value = true
    actionError.value = null
    try {
        await apiFetch(`/api/campaign/${campaignId}`, { method: 'DELETE' })
        deleteModalOpen.value = false
        await navigateTo('/campaigns')
    } catch (e: any) {
        actionError.value = extractFetchError(e, 'Failed to delete campaign')
    } finally {
        deleting.value = false
    }
}
</script>

<template>
    <div v-if="campaign">
        <!-- Header -->
        <div class="hero-surface glass-panel p-5 mb-4">
            <div class="flex items-start justify-between gap-4">
                <div class="min-w-0">
                    <div class="flex items-center gap-2.5 flex-wrap">
                        <h1 class="text-xl font-bold text-white">{{ campaign.name }}</h1>
                        <span v-if="campaign.processing?.paused" class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-medium bg-red-500/15 text-red-400">
                            <Icon name="lucide:pause" class="text-fine" /> Processing paused
                        </span>
                        <span v-else-if="campaign.status === 0" class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-medium bg-amber-500/15 text-amber-400">
                            <Icon name="lucide:loader-2" class="text-fine animate-spin" /> Computing
                        </span>
                        <span v-else-if="campaign.status === 2" class="px-2 py-0.5 rounded text-fine font-medium bg-white/[0.06] text-gray-500">Archived</span>
                        <span v-else class="px-2 py-0.5 rounded text-fine font-medium bg-green-500/15 text-isk">Active</span>
                        <span v-if="campaign.visibility === 1" class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-medium bg-white/[0.06] text-gray-400" v-tooltip="'Visible only to its owner, admins, and members of custom domains where it is selected'">
                            <Icon name="lucide:lock" class="text-fine" /> Private
                        </span>
                        <span v-else-if="campaign.visibility === 2" class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-medium bg-indigo-500/15 text-indigo-300" v-tooltip="'Unlisted on EVE-KILL — shown on the custom killboards of involved or allowed entities'">
                            <Icon name="lucide:shield" class="text-fine" /> Killboard only
                        </span>
                    </div>
                    <div class="text-xs text-gray-500 mt-1">
                        {{ formatDate(campaign.start_time) }} — {{ campaign.end_time ? formatDate(campaign.end_time) : 'ongoing' }}
                        <template v-if="campaign.creator.name">
                            · created by
                            <NuxtLink :to="`/character/${campaign.creator.character_id}`" class="text-gray-400 hover:text-blue-400">{{ campaign.creator.name }}</NuxtLink>
                        </template>
                        <template v-if="campaign.stats_updated_at">· updated {{ formatDateTime(campaign.stats_updated_at) }}</template>
                        <template v-if="campaign.processing?.estimated_killmails != null">
                            · estimated {{ formatNumber(campaign.processing.estimated_killmails) }} killmails
                        </template>
                    </div>
                    <p v-if="campaign.description" class="text-sm text-gray-400 mt-3 whitespace-pre-line">{{ campaign.description }}</p>
                    <div v-if="campaign.location_details.systems.length || campaign.location_details.constellations.length || campaign.location_details.regions.length"
                        class="flex flex-wrap gap-1.5 mt-3">
                        <NuxtLink v-for="loc in campaign.location_details.systems" :key="`system-${loc.id}`" :to="`/system/${loc.id}`"
                            class="inline-flex items-center gap-1 px-2 py-1 rounded border border-amber-500/25 bg-amber-500/[0.08] text-fine text-amber-400 hover:bg-amber-500/[0.14] transition-colors">
                            <Icon name="lucide:map-pin" class="text-fine" /> {{ loc.name }}
                        </NuxtLink>
                        <NuxtLink v-for="loc in campaign.location_details.constellations" :key="`constellation-${loc.id}`" :to="`/constellation/${loc.id}`"
                            class="inline-flex items-center gap-1 px-2 py-1 rounded border border-amber-500/25 bg-amber-500/[0.08] text-fine text-amber-400 hover:bg-amber-500/[0.14] transition-colors">
                            <Icon name="lucide:map" class="text-fine" /> {{ loc.name }}
                        </NuxtLink>
                        <NuxtLink v-for="loc in campaign.location_details.regions" :key="`region-${loc.id}`" :to="`/region/${loc.id}`"
                            class="inline-flex items-center gap-1 px-2 py-1 rounded border border-amber-500/25 bg-amber-500/[0.08] text-fine text-amber-400 hover:bg-amber-500/[0.14] transition-colors">
                            <Icon name="lucide:map" class="text-fine" /> {{ loc.name }}
                        </NuxtLink>
                    </div>
                </div>

                <div class="flex items-center gap-1.5 flex-shrink-0">
                    <button v-if="campaign.prize_pool" type="button"
                        class="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-amber-500/25 bg-amber-500/[0.09] text-xs text-amber-300 hover:bg-amber-500/[0.16] hover:border-amber-500/40 transition-colors"
                        @click="prizePanelOpen = true">
                        <Icon name="lucide:trophy" />
                        <span class="font-medium">Prize pool</span>
                        <span class="pl-2 border-l border-amber-500/20 font-bold tabular-nums">
                            {{ formatIsk(Number(campaign.prize_pool.funded_total)) }}
                        </span>
                    </button>

                    <!-- Owner / admin controls -->
                    <ClientOnly>
                        <div v-if="canEdit" class="flex items-center gap-1.5">
                            <button @click="openEdit" class="px-2.5 py-1.5 rounded-lg text-xs bg-white/[0.04] text-gray-400 border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors" v-tooltip="'Edit'">
                                <Icon name="lucide:pencil" class="text-xs" />
                            </button>
                            <button v-if="!campaign.prize_pool" @click="toggleArchive" :disabled="saving" class="px-2.5 py-1.5 rounded-lg text-xs bg-white/[0.04] text-gray-400 border border-white/[0.08] hover:bg-amber-500/[0.08] hover:text-amber-400 transition-colors" v-tooltip="campaign.status === 2 ? 'Reactivate' : 'Archive'">
                                <Icon :name="campaign.status === 2 ? 'lucide:archive-restore' : 'lucide:archive'" class="text-xs" />
                            </button>
                            <button v-if="!campaign.prize_pool || Number(campaign.prize_pool.funded_total) === 0" @click="deleteModalOpen = true" class="px-2.5 py-1.5 rounded-lg text-xs border transition-colors bg-white/[0.04] text-gray-400 border-white/[0.08] hover:bg-red-500/[0.08] hover:text-red-400"
                                v-tooltip="'Delete'">
                                <Icon name="lucide:trash-2" class="text-xs" />
                            </button>
                        </div>
                    </ClientOnly>
                </div>
            </div>

            <!-- Who is ahead — the headline read, before the per-side detail below -->
            <div v-if="!isAreaCampaign && campaign.sides.length && !isComputing && stats" class="mt-4 pt-4 border-t border-white/[0.06]">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">
                    {{ isOneSided ? 'ISK destroyed vs lost' : 'Share of ISK destroyed' }}
                </div>
                <CampaignBalanceBar :sides="campaign.sides" :accents="sideAccents" />
            </div>

            <!-- Edit panel -->
            <div v-if="editing" class="mt-4 pt-4 border-t border-white/[0.06] space-y-3">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div>
                        <label class="text-fine text-gray-500 mb-1 block">Name</label>
                        <input v-model="editName" type="text" class="w-full px-3 py-2 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-white focus:border-blue-500/40 focus:outline-none" />
                    </div>
                    <div>
                        <label class="text-fine text-gray-500 mb-1 block">End time (UTC)</label>
                        <div class="flex items-center gap-2">
                            <DateTimePicker v-model="editEndTime" :disabled="editOngoing || !!campaign.prize_pool?.rules_locked_at" class="flex-1 min-w-0" placeholder="Pick an end" />
                            <label class="flex items-center gap-1.5 text-xs text-gray-400 cursor-pointer">
                                <input v-model="editOngoing" type="checkbox" :disabled="!!campaign.prize_pool" class="accent-blue-500 disabled:opacity-40" /> Ongoing
                            </label>
                        </div>
                    </div>
                </div>
                <div>
                    <label class="text-fine text-gray-500 mb-1 block">Description</label>
                    <textarea v-model="editDescription" rows="3" class="w-full px-3 py-2 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-gray-300 focus:border-blue-500/40 focus:outline-none" />
                </div>
                <div v-if="!isAreaCampaign" class="space-y-2">
                    <div class="flex items-center justify-between gap-3">
                        <div>
                            <label class="text-fine text-gray-500 block">Sides and participants</label>
                            <p class="text-fine text-gray-600 mt-0.5">Changing participants triggers a full campaign recompute.</p>
                        </div>
                        <button v-if="editSides.length < 4 && !campaign.prize_pool?.rules_locked_at" type="button" class="px-2 py-1 rounded bg-white/[0.05] text-fine text-gray-400 hover:text-blue-300" @click="addEditSide">Add side</button>
                    </div>
                    <div v-if="campaign.prize_pool?.rules_locked_at" class="rounded border border-amber-500/20 bg-amber-500/[0.06] px-3 py-2 text-xs text-amber-300">
                        Participants are locked because this prize campaign has received funding.
                    </div>
                    <div v-for="(side, sideIndex) in editSides" :key="sideIndex" class="rounded-lg border border-white/[0.08] bg-white/[0.025] p-3 space-y-2">
                        <div class="flex items-center gap-2">
                            <input v-model="side.name" type="text" maxlength="50" :disabled="!!campaign.prize_pool?.rules_locked_at" class="flex-1 min-w-0 px-2.5 py-1.5 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white focus:border-blue-500/40 focus:outline-none disabled:opacity-50" />
                            <span class="text-fine text-gray-600 tabular-nums">{{ side.entities.length }}/15</span>
                            <button v-if="editSides.length > 1 && !campaign.prize_pool?.rules_locked_at" type="button" class="text-gray-600 hover:text-red-400" @click="removeEditSide(sideIndex)"><Icon name="lucide:trash-2" class="text-xs" /></button>
                        </div>
                        <SearchPicker v-if="!campaign.prize_pool?.rules_locked_at" :types="['alliance', 'corporation', 'character']"
                            :disabled="side.entities.length >= 15"
                            :placeholder="side.entities.length >= 15 ? 'Maximum 15 participants' : 'Add a participant to this side'"
                            :is-picked="(type, id) => side.entities.some(entity => entity.type === type && entity.id === id)"
                            @select="addEditParticipant(sideIndex, $event)" />
                        <div class="flex flex-wrap gap-1.5">
                            <div v-for="entity in side.entities" :key="`${entity.type}-${entity.id}`" class="inline-flex items-center gap-1.5 px-2 py-1 rounded border border-white/[0.08] bg-white/[0.04] text-xs text-gray-300">
                                {{ entity.name ?? `#${entity.id}` }}
                                <button v-if="!campaign.prize_pool?.rules_locked_at" type="button" class="text-gray-600 hover:text-red-400" @click="removeEditParticipant(sideIndex, entity)"><Icon name="lucide:x" class="text-fine" /></button>
                            </div>
                            <span v-if="!side.entities.length" class="text-fine text-red-400">This side needs at least one participant.</span>
                        </div>
                    </div>
                </div>
                <div>
                    <label class="text-fine text-gray-500 mb-1 block">Visibility</label>
                    <select v-model.number="editVisibility"
                        class="w-full px-3 py-2 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-gray-300 focus:border-blue-500/40 focus:outline-none [&>option]:bg-gray-900">
                        <option :value="0">Public — listed on EVE-KILL for everyone</option>
                        <option :value="1">Private — owner and selected-domain members only</option>
                        <option :value="2">Killboard only — shown on involved or allowed custom killboards</option>
                    </select>
                </div>
                <div v-if="editVisibility === 2 && isAreaCampaign">
                    <label class="text-fine text-gray-500 mb-1 block">Allow these entities to see it on their custom killboard</label>
                    <SearchPicker :types="['alliance', 'corporation', 'character']"
                        :disabled="editAllowed.length >= MAX_ALLOWED_ENTITIES"
                        :placeholder="editAllowed.length >= MAX_ALLOWED_ENTITIES ? `Max ${MAX_ALLOWED_ENTITIES} entities` : undefined"
                        :is-picked="(t, id) => editAllowed.some(e => e.type === t && e.id === id)"
                        @select="addEditAllowed" />
                    <div v-if="editAllowed.length" class="flex flex-wrap gap-1.5 mt-2">
                        <div v-for="e in editAllowed" :key="`${e.type}-${e.id}`"
                            class="inline-flex items-center gap-1.5 px-2 py-1 rounded border border-indigo-500/30 bg-indigo-500/10 text-xs text-indigo-300 font-medium">
                            {{ e.name ?? `#${e.id}` }}
                            <button @click="removeEditAllowed(e)" class="ml-0.5 opacity-60 hover:opacity-100"><Icon name="lucide:x" class="text-fine" /></button>
                        </div>
                    </div>
                    <p v-else class="text-fine text-gray-600 mt-1.5">No grants yet — only your own killboards can list this campaign.</p>
                </div>
                <div v-if="editError" class="flex items-center gap-2 text-xs text-red-400">
                    <span>{{ editError }}</span>
                    <button v-if="reauthRequired" class="px-2 py-1 rounded bg-red-500/10 hover:bg-red-500/20 text-red-300 font-medium" @click="login({ redirect: route.fullPath })">Sign in again</button>
                </div>
                <div class="flex items-center gap-2">
                    <button @click="saveEdit" :disabled="saving || !editSidesValid" class="px-4 py-2 rounded-lg text-xs font-medium bg-blue-500 text-white hover:bg-blue-600 transition-colors disabled:opacity-40">
                        {{ saving ? 'Saving…' : 'Save changes' }}
                    </button>
                    <button @click="editing = false" class="glass-panel px-4 py-2 text-xs text-gray-400">Cancel</button>
                    <span class="text-fine text-gray-600">Changing the end time or participants triggers a full recompute</span>
                </div>
            </div>
            <div v-if="actionError || actionMessage" class="mt-3 text-xs" :class="actionError ? 'text-red-400' : 'text-isk'">
                {{ actionError ?? actionMessage }}
            </div>
        </div>

        <Modal v-model="deleteModalOpen" title="Delete campaign">
            <p class="text-sm text-gray-300">Delete <span class="font-semibold text-white">{{ campaign.name }}</span>? This permanently removes the campaign and its computed statistics.</p>
            <p class="text-xs text-gray-500 mt-2">This cannot be undone.</p>
            <p v-if="actionError" class="mt-3 text-xs text-red-400">{{ actionError }}</p>
            <template #footer>
                <div class="flex justify-end gap-2">
                    <button type="button" class="px-3 py-2 rounded-lg text-xs text-gray-400 bg-white/[0.04] hover:bg-white/[0.08]" :disabled="deleting" @click="deleteModalOpen = false">Cancel</button>
                    <button type="button" class="px-3 py-2 rounded-lg text-xs font-medium text-white bg-red-600 hover:bg-red-500 disabled:opacity-50" :disabled="deleting" @click="deleteCampaign">
                        {{ deleting ? 'Deleting…' : 'Delete campaign' }}
                    </button>
                </div>
            </template>
        </Modal>

        <!-- Prize pool details stay out of the campaign flow until requested. -->
        <Modal v-if="campaign.prize_pool" v-model="prizePanelOpen" max-width="max-w-4xl" no-padding>
            <template #header>
                <div class="flex items-center gap-3 min-w-0">
                    <div class="w-9 h-9 rounded-lg bg-amber-500/15 flex items-center justify-center flex-shrink-0">
                        <Icon name="lucide:trophy" class="text-lg text-amber-400" />
                    </div>
                    <div class="min-w-0">
                        <div class="flex items-center gap-2">
                            <h2 class="text-sm font-semibold text-white">Prize pool</h2>
                            <span class="px-2 py-0.5 rounded text-fine font-medium"
                                :class="campaign.prize_pool.status === 2
                                    ? 'bg-green-500/15 text-green-400'
                                    : campaign.prize_pool.status === 1
                                        ? 'bg-blue-500/15 text-blue-300'
                                        : prizeFundingOpen
                                            ? 'bg-amber-500/15 text-amber-300'
                                            : 'bg-purple-500/15 text-purple-300'">
                                {{ campaign.prize_pool.status === 2
                                    ? 'Paid'
                                    : campaign.prize_pool.status === 1
                                        ? 'Official results'
                                        : prizeFundingOpen ? 'Funding open' : 'Finalizing' }}
                            </span>
                        </div>
                        <p class="text-fine text-gray-600 truncate">
                            Top {{ campaign.prize_pool.winner_count }} by {{ campaign.prize_pool.metric_label.toLowerCase() }}
                        </p>
                    </div>
                </div>
            </template>

            <div class="grid grid-cols-1 lg:grid-cols-[minmax(0,1.25fr)_minmax(260px,0.75fr)]">
                <div class="p-5 lg:border-r border-white/[0.06] space-y-5">
                    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-3">
                        <div>
                            <div class="text-fine uppercase tracking-[0.15em] text-gray-600">Current pool</div>
                            <div class="text-3xl font-bold text-amber-300 tabular-nums mt-0.5">
                                {{ formatIsk(Number(campaign.prize_pool.funded_total)) }}
                            </div>
                            <div class="text-fine text-gray-600 mt-1">
                                {{ campaign.prize_pool.contribution_count }} confirmed contribution{{ campaign.prize_pool.contribution_count === 1 ? '' : 's' }}
                            </div>
                        </div>
                        <div class="flex flex-wrap gap-1.5">
                            <span v-for="(percentage, index) in campaign.prize_pool.payout_percentages" :key="index"
                                class="px-2 py-1 rounded-md border border-white/[0.06] bg-white/[0.025] text-fine text-gray-400">
                                #{{ index + 1 }} <strong class="text-gray-200">{{ percentage }}%</strong>
                            </span>
                        </div>
                    </div>

                    <div>
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">
                            {{ campaign.prize_pool.status === 0 ? 'Current leaders' : 'Official winners' }}
                        </div>
                        <div v-if="campaign.prize_pool.results.length" class="space-y-1.5">
                            <div v-for="result in campaign.prize_pool.results" :key="result.rank"
                                class="flex items-center gap-3 rounded-lg border border-white/[0.06] bg-white/[0.025] px-3 py-2.5">
                                <div class="w-7 text-center text-sm font-bold" :class="result.rank === 1 ? 'text-amber-300' : 'text-gray-500'">#{{ result.rank }}</div>
                                <img :src="`/images/characters/${result.character_id}/portrait?size=64`"
                                    class="w-8 h-8 rounded" :alt="result.character_name || ''" />
                                <NuxtLink :to="`/character/${result.character_id}`" class="min-w-0 flex-1 text-sm text-gray-200 hover:text-blue-300 truncate">
                                    {{ result.character_name || `Character ${result.character_id}` }}
                                </NuxtLink>
                                <div class="text-right">
                                    <div class="text-xs text-gray-300 tabular-nums">{{ formatPrizeScore(result) }}</div>
                                    <div class="text-fine text-gray-600">{{ result.payout_percentage }}% · {{ formatIsk(result.payout_amount) }}</div>
                                </div>
                                <div v-if="campaign.prize_pool.status !== 0" class="w-20 text-right">
                                    <span v-if="result.paid" class="text-fine text-green-400">Paid</span>
                                    <span v-else-if="result.claimed" class="text-fine text-blue-300">Claimed</span>
                                    <button v-else-if="result.can_claim" @click="claimPrize" :disabled="prizeClaiming"
                                        class="px-2.5 py-1 rounded bg-amber-500/15 text-fine font-medium text-amber-300 hover:bg-amber-500/25 disabled:opacity-50">
                                        Claim
                                    </button>
                                    <span v-else class="text-fine text-gray-600">Unclaimed</span>
                                </div>
                            </div>
                        </div>
                        <p v-else class="rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-4 text-xs text-gray-600">
                            Standings will appear after qualifying campaign activity is processed.
                        </p>
                    </div>

                    <div v-if="prizeFundingOpen" class="rounded-lg border border-amber-500/20 bg-amber-500/[0.06] p-4">
                        <div class="flex items-center gap-2 text-sm font-medium text-gray-200">
                            <Icon name="lucide:circle-plus" class="text-amber-400" />
                            Add to the prize pool
                        </div>

                        <div v-if="isAuthenticated" class="mt-3">
                            <div class="flex items-center justify-between gap-3 mb-1.5">
                                <label class="text-xs text-gray-400">Fund instantly from your EK Wallet</label>
                                <button type="button" class="text-fine text-amber-300 hover:text-amber-200 disabled:text-gray-600"
                                    :disabled="prizeWalletLoading || prizeWalletAvailable <= 0"
                                    @click="usePrizeWalletMaximum">
                                    {{ prizeWalletLoading ? 'Loading balance…' : `${formatIsk(prizeWalletAvailable)} available · use max` }}
                                </button>
                            </div>
                            <div class="flex items-center gap-2">
                                <div class="relative min-w-0 flex-1">
                                    <input v-model="prizeContributionAmount" type="text" inputmode="decimal"
                                        placeholder="Amount"
                                        class="w-full px-3 py-2 pr-10 rounded bg-black/35 border border-white/[0.08] text-xs text-white tabular-nums placeholder-gray-700 focus:border-amber-500/40 focus:outline-none"
                                        @keyup.enter="contributeFromPrizeWallet" />
                                    <span class="absolute right-3 top-2 text-fine text-gray-600">ISK</span>
                                </div>
                                <button type="button"
                                    class="px-3 py-2 rounded bg-amber-500/20 text-xs font-medium text-amber-200 hover:bg-amber-500/30 disabled:opacity-40 disabled:cursor-not-allowed"
                                    :disabled="prizeContributing || prizeWalletLoading || prizeWalletAvailable <= 0"
                                    @click="contributeFromPrizeWallet">
                                    {{ prizeContributing ? 'Adding…' : 'Add ISK' }}
                                </button>
                            </div>
                        </div>
                        <button v-else type="button"
                            class="mt-3 px-3 py-2 rounded bg-amber-500/15 text-xs font-medium text-amber-300 hover:bg-amber-500/25"
                            @click="login({ redirect: route.fullPath })">
                            Sign in to fund from your EK Wallet
                        </button>

                        <div class="flex items-center gap-3 my-3">
                            <div class="h-px flex-1 bg-white/[0.06]" />
                            <span class="text-fine uppercase tracking-wider text-gray-700">or send in-game</span>
                            <div class="h-px flex-1 bg-white/[0.06]" />
                        </div>
                        <p class="text-xs text-gray-500">
                            Send ISK to <strong class="text-gray-300">EVE-KILL.com</strong> with this exact reason:
                        </p>
                        <div class="flex items-center gap-2 mt-2">
                            <code class="min-w-0 flex-1 px-3 py-2 rounded bg-black/40 border border-white/[0.08] text-xs text-amber-300 select-all truncate">
                                {{ campaign.prize_pool.funding_reference }}
                            </code>
                            <button @click="copyPrizeReference" class="px-3 py-2 rounded bg-amber-500/15 text-xs font-medium text-amber-300 hover:bg-amber-500/25">
                                <Icon :name="prizeCopied ? 'lucide:check' : 'lucide:copy'" class="mr-1" />
                                {{ prizeCopied ? 'Copied' : 'Copy' }}
                            </button>
                        </div>
                        <div class="flex flex-wrap gap-x-4 gap-y-1 mt-2 text-fine text-gray-600">
                            <span>Closes {{ formatDateTime(campaign.prize_pool.funding_closes_at) }}</span>
                            <span>Wallet checked {{ campaign.prize_pool.last_wallet_sync ? formatDateTime(campaign.prize_pool.last_wallet_sync) : 'not yet' }}</span>
                        </div>
                    </div>

                    <div v-else-if="prizeSettling" class="rounded-lg border border-purple-500/20 bg-purple-500/[0.07] px-4 py-3 text-xs text-gray-300">
                        <div class="flex items-center gap-2">
                            <Icon name="lucide:hourglass" class="text-purple-300" />
                            Official winners will be announced {{ formatDateTime(campaign.prize_pool.settles_at) }} after the 24-hour killmail catch-up period.
                        </div>
                        <div v-if="campaign.prize_pool.projected_lead_percent != null && campaign.prize_pool.projected_lead_percent >= 25 && campaign.prize_pool.results[0]" class="mt-2 text-amber-300">
                            Projected winner: {{ campaign.prize_pool.results[0].character_name || campaign.prize_pool.results[0].character_id }}
                            leads second place by {{ Math.round(campaign.prize_pool.projected_lead_percent) }}%.
                        </div>
                    </div>

                    <div v-if="prizeActionMessage" class="text-xs text-green-400">{{ prizeActionMessage }}</div>
                    <div v-if="prizeActionError" class="text-xs text-red-400">{{ prizeActionError }}</div>
                </div>

                <div class="p-5 bg-black/15">
                    <div class="flex items-center justify-between gap-3 mb-3">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Contributors</div>
                        <span class="text-fine text-gray-700">{{ campaign.prize_pool.contribution_count }}</span>
                    </div>
                    <div v-if="campaign.prize_pool.contributions.length" class="space-y-1">
                        <div v-for="contribution in campaign.prize_pool.contributions" :key="contribution.id"
                            class="flex items-center gap-2.5 rounded-lg px-2 py-2 hover:bg-white/[0.025]">
                            <img v-if="prizeContributorImage(contribution)" :src="prizeContributorImage(contribution) || undefined"
                                class="w-8 h-8 rounded bg-gray-900 flex-shrink-0" :alt="contribution.contributor_name" />
                            <div v-else class="w-8 h-8 rounded bg-white/[0.04] flex items-center justify-center flex-shrink-0">
                                <Icon name="lucide:user-round" class="text-gray-600" />
                            </div>
                            <div class="min-w-0 flex-1">
                                <NuxtLink v-if="contribution.contributor_id && contribution.contributor_type !== 'unknown'"
                                    :to="`/${contribution.contributor_type}/${contribution.contributor_id}`"
                                    class="block text-xs text-gray-300 hover:text-blue-300 truncate">
                                    {{ contribution.contributor_name }}
                                </NuxtLink>
                                <div v-else class="text-xs text-gray-300 truncate">{{ contribution.contributor_name }}</div>
                                <div class="text-fine text-gray-700">
                                    {{ contribution.source === 'ek_wallet' ? 'EK Wallet' : 'EVE transfer' }}
                                    · {{ formatDateTime(contribution.contributed_at) }}
                                </div>
                            </div>
                            <div class="text-xs font-medium text-amber-300 tabular-nums">
                                {{ formatIsk(Number(contribution.amount)) }}
                            </div>
                        </div>
                        <p v-if="campaign.prize_pool.contribution_count > campaign.prize_pool.contributions.length"
                            class="text-fine text-gray-700 px-2 pt-2">
                            Showing the latest {{ campaign.prize_pool.contributions.length }} contributions.
                        </p>
                    </div>
                    <p v-else class="text-xs text-gray-600 py-3">No confirmed contributions yet.</p>
                </div>
            </div>

            <template #footer>
                <div class="text-fine text-gray-500">
                    Payouts are handled manually by EVE-KILL.com staff. Sign in with the winning character and
                    <a :href="campaign.prize_pool.discord_url" target="_blank" rel="noopener noreferrer" class="text-blue-400 hover:text-blue-300">join our Discord</a>
                    to arrange payment. Staff will never ask you to send ISK to release winnings.
                </div>
            </template>
        </Modal>

        <div v-if="campaign.processing?.paused" class="glass-panel border border-red-500/20 bg-red-500/[0.05] p-4 mb-4">
            <div class="flex items-start gap-3">
                <Icon name="lucide:shield-alert" class="text-lg text-red-400 mt-0.5" />
                <div class="flex-1 min-w-0">
                    <p class="text-sm font-medium text-red-300">Campaign processing was stopped by a workload safety limit.</p>
                    <p v-if="campaign.processing?.note" class="text-xs text-gray-400 mt-1">{{ campaign.processing.note }}</p>
                    <div v-if="canEdit" class="flex items-center gap-2 mt-3">
                        <button class="px-3 py-1.5 rounded-md text-xs bg-red-500/15 text-red-300 hover:bg-red-500/25" @click="openEdit">
                            Narrow campaign
                        </button>
                        <button class="px-3 py-1.5 rounded-md text-xs bg-white/[0.05] text-gray-300 hover:bg-white/[0.08]" :disabled="saving" @click="resumeProcessing">
                            Resume processing
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Computing state -->
        <div v-if="isComputing" class="glass-panel p-10 text-center mb-4">
            <Icon name="lucide:loader-2" class="text-3xl text-blue-400/60 animate-spin mb-3" />
            <p class="text-sm text-gray-400">Crunching killmails for this campaign…</p>
            <p class="text-fine text-gray-600 mt-1">Large campaigns can take a couple of minutes. This page refreshes automatically.</p>
        </div>

        <template v-if="!isComputing && stats">
            <!-- Overview tiles -->
            <div class="grid grid-cols-2 md:grid-cols-5 gap-3 mb-4">
                <div v-for="tile in [
                    { label: 'Killmails', icon: 'lucide:crosshair', value: formatNumber(stats.totals.killCount), color: 'text-white' },
                    { label: 'ISK Destroyed', icon: 'lucide:coins', value: formatIsk(stats.totals.iskDestroyed), color: 'text-yellow-400' },
                    { label: 'Characters', icon: 'lucide:user-round', value: formatNumber(stats.totals.charactersInvolved), color: 'text-blue-400' },
                    { label: 'Corporations', icon: 'lucide:building-2', value: formatNumber(stats.totals.corporationsInvolved), color: 'text-purple-400' },
                    { label: 'Alliances', icon: 'lucide:shield', value: formatNumber(stats.totals.alliancesInvolved), color: 'text-indigo-400' },
                ]" :key="tile.label" class="glass-panel p-3 text-center">
                    <div class="text-lg font-bold tabular-nums" :class="tile.color">{{ tile.value }}</div>
                    <div class="flex items-center justify-center gap-1.5 text-fine text-gray-500 uppercase tracking-wider mt-0.5">
                        <Icon :name="tile.icon" class="text-fine text-gray-600" />{{ tile.label }}
                    </div>
                </div>
            </div>

            <!-- Side scoreboard -->
            <div v-if="campaign.sides.length" class="grid gap-4 mb-4" :class="campaign.sides.length === 1 ? 'grid-cols-1' : campaign.sides.length === 2 ? 'grid-cols-1 md:grid-cols-2' : campaign.sides.length === 3 ? 'grid-cols-1 md:grid-cols-2 xl:grid-cols-3' : 'grid-cols-1 md:grid-cols-2 xl:grid-cols-4'">
                <div v-for="s in campaign.sides" :key="s.side_index"
                    class="rounded-lg bg-white/[0.04] border p-4" :class="SIDE_BORDER[s.side_index]"
                    :style="sideAccents[s.side_index] ? { borderColor: sideAccents[s.side_index]!.border } : undefined">
                    <div class="flex items-center justify-between mb-3">
                        <div class="text-sm font-bold" :class="SIDE_TEXT[s.side_index]"
                            :style="sideAccents[s.side_index] ? { color: sideAccents[s.side_index]!.accent } : undefined">{{ s.name }}</div>
                        <div class="text-fine text-gray-500 tabular-nums">{{ efficiency(s).toFixed(1) }}% efficiency</div>
                    </div>
                    <div class="grid grid-cols-4 gap-2 text-center mb-3">
                        <div><div class="text-sm font-bold text-isk tabular-nums">{{ formatNumber(s.kills) }}</div><div class="text-fine text-gray-600">Kills</div></div>
                        <div><div class="text-sm font-bold text-red-400 tabular-nums">{{ formatNumber(s.losses) }}</div><div class="text-fine text-gray-600">Losses</div></div>
                        <div><div class="text-sm font-bold text-yellow-400 tabular-nums">{{ formatIsk(s.isk_destroyed) }}</div><div class="text-fine text-gray-600">Destroyed</div></div>
                        <div><div class="text-sm font-bold text-orange-400 tabular-nums">{{ formatIsk(s.isk_lost) }}</div><div class="text-fine text-gray-600">Lost</div></div>
                    </div>
                    <!-- Efficiency bar -->
                    <div class="h-1 rounded-full bg-white/[0.06] overflow-hidden mb-3">
                        <div class="h-full bg-green-500/60" :style="{ width: `${efficiency(s)}%`, ...(sideAccents[s.side_index] ? { backgroundColor: sideAccents[s.side_index]!.accent } : {}) }" />
                    </div>
                    <!-- Entities -->
                    <div class="space-y-0.5">
                        <NuxtLink v-for="e in s.entities" :key="`${e.entity_type}-${e.entity_id}`" :to="entityLink(e)"
                            class="flex items-center gap-2 px-1.5 py-1 rounded hover:bg-blue-500/[0.06] transition-colors">
                            <img :src="entityImage(e)" class="w-5 h-5 rounded flex-shrink-0" loading="lazy" :alt="e.name ?? ''">
                            <span class="text-xs text-gray-300 truncate flex-1">{{ e.name ?? `#${e.entity_id}` }}</span>
                            <span class="text-fine tabular-nums flex-shrink-0">
                                <span class="text-isk">{{ formatNumber(e.kills) }}</span><span class="text-gray-600"> / </span><span class="text-red-400">{{ formatNumber(e.losses) }}</span>
                            </span>
                        </NuxtLink>
                    </div>
                </div>
            </div>

        </template>

        <!-- Tab bar — same underline style as battle reports -->
        <div class="flex overflow-x-auto border-b border-white/[0.08] mb-4 scrollbar-hide">
            <button v-for="t in visibleTabs" :key="t.key"
                class="flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors border-b-2 cursor-pointer"
                :class="activeTab === t.key
                    ? 'text-white border-white'
                    : 'text-gray-500 border-transparent hover:text-blue-400'"
                @click="setTab(t.key)">
                <Icon :name="t.icon" class="text-base" />
                {{ t.label }}
            </button>
        </div>

        <!-- ── Overview tab ─────────────────────────────────────────────── -->
        <template v-if="activeTab === 'overview'">
        <template v-if="!isComputing && stats">
            <!-- Most valuable kills — same horizontal tile strip as the rest of the site -->
            <div v-if="stats.mostValuable?.length" class="glass-panel p-3 mb-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-3 px-1">Most Valuable Kills</div>
                <div class="grid grid-cols-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-8 gap-2">
                    <NuxtLink
                        v-for="(mv, idx) in stats.mostValuable.slice(0, 8)"
                        :key="mv.killmailId"
                        :to="`/kill/${mv.killmailId}`"
                        class="group relative aspect-square rounded-lg overflow-hidden bg-black/40 border border-white/[0.04] hover:border-blue-500/30 transition-all"
                        :class="Number(idx) >= 6 ? 'hidden sm:block' : ''"
                        :style="mvTileStyle(mv)"
                    >
                        <img
                            v-if="mv.shipTypeId"
                            :src="`/images/types/${mv.shipTypeId}/overlayrender?size=256`"
                            :alt="mv.shipName ?? ''"
                            class="absolute inset-0 w-full h-full object-contain p-2 transition-transform duration-300 group-hover:scale-110"
                            loading="lazy"
                        >
                        <div class="absolute inset-x-0 bottom-0 h-2/3 bg-gradient-to-t from-black/90 via-black/50 to-transparent pointer-events-none"></div>
                        <div class="absolute top-1.5 right-1.5 px-1.5 py-0.5 rounded bg-black/60 backdrop-blur-sm text-fine font-semibold text-green-400">
                            {{ formatIsk(mv.value) }}
                        </div>
                        <div class="absolute inset-x-0 bottom-0 p-2.5 flex flex-col items-start">
                            <div class="text-xs font-semibold text-white leading-tight truncate w-full drop-shadow-lg">{{ mv.shipName ?? 'Unknown' }}</div>
                            <div v-if="mv.victimCharacterName" class="text-fine text-gray-300/90 truncate w-full mt-0.5 drop-shadow">{{ mv.victimCharacterName }}</div>
                            <div v-if="mv.victimCorporationName" class="text-fine text-gray-400/80 truncate w-full drop-shadow">{{ mv.victimCorporationName }}</div>
                        </div>
                    </NuxtLink>
                </div>
            </div>

            <!-- Activity chart -->
            <div v-if="chart.bars.length > 1" class="glass-panel p-4 mb-4">
                <div class="flex items-center justify-between mb-2">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">
                        Activity — {{ chartMetric === 'isk' ? 'ISK' : 'ships' }} {{ isAreaCampaign ? 'destroyed' : isOneSided ? 'destroyed vs lost' : 'lost per side' }} per {{ campaign.daily.granularity }}
                    </div>
                    <div class="flex rounded-lg border border-white/[0.08] overflow-hidden">
                        <button v-for="m in (['ships', 'isk'] as const)" :key="m" @click="chartMetric = m"
                            class="px-2.5 py-1 text-fine font-medium transition-colors cursor-pointer"
                            :class="chartMetric === m ? 'bg-blue-500/20 text-blue-400' : 'bg-white/[0.02] text-gray-500 hover:bg-white/[0.05]'">
                            {{ m === 'ships' ? 'Ships' : 'ISK' }}
                        </button>
                    </div>
                </div>

                <div class="relative" @mouseleave="hoverIdx = null">
                    <svg viewBox="0 0 960 190" class="w-full h-auto block select-none">
                        <!-- gridlines + y ticks -->
                        <g v-for="t in chart.ticks" :key="t.y">
                            <line :x1="chart.plot.x0" :x2="chart.plot.x1" :y1="t.y" :y2="t.y" stroke="rgba(255,255,255,0.05)" stroke-width="1" />
                            <text :x="chart.plot.x0 - 8" :y="t.y + 3" text-anchor="end" fill="#6b7280" font-size="10">{{ t.label }}</text>
                        </g>
                        <line :x1="chart.plot.x0" :x2="chart.plot.x1" :y1="chart.plot.y1" :y2="chart.plot.y1" stroke="rgba(255,255,255,0.10)" stroke-width="1" />

                        <!-- hover column highlight -->
                        <rect v-if="hoverIdx !== null && chart.bars[hoverIdx]"
                            :x="chart.plot.x0 + hoverIdx * chart.step" :y="chart.plot.y0"
                            :width="chart.step" :height="chart.plot.y1 - chart.plot.y0"
                            fill="rgba(255,255,255,0.04)" rx="3" />

                        <!-- stacked bars -->
                        <g v-for="(bar, i) in chart.bars" :key="bar.period"
                            class="transition-opacity duration-100"
                            :opacity="hoverIdx === null || hoverIdx === i ? 1 : 0.4">
                            <template v-for="seg in bar.segments" :key="seg.side">
                                <path v-if="seg.topmost" :d="topRoundedPath(bar.x, seg.y, bar.w, seg.h)" :fill="segColor(seg.side)" fill-opacity="0.9" />
                                <rect v-else :x="bar.x" :y="seg.y" :width="bar.w" :height="seg.h" :fill="segColor(seg.side)" fill-opacity="0.9" />
                            </template>
                        </g>

                        <!-- x labels -->
                        <text v-for="xl in chart.xLabels" :key="xl.x" :x="xl.x" :y="chart.plot.y1 + 16" text-anchor="middle" fill="#6b7280" font-size="10">{{ xl.label }}</text>

                        <!-- hover targets (full-height columns, bigger than the marks) -->
                        <rect v-for="(bar, i) in chart.bars" :key="`hover-${bar.period}`"
                            :x="chart.plot.x0 + i * chart.step" :y="chart.plot.y0"
                            :width="chart.step" :height="chart.plot.y1 - chart.plot.y0"
                            fill="transparent" @mouseenter="hoverIdx = i" />
                    </svg>

                    <!-- tooltip -->
                    <div v-if="hoverBar"
                        class="pointer-events-none absolute top-0 z-20 -translate-x-1/2 rounded-lg bg-black/90 backdrop-blur-xl border border-white/[0.10] px-3 py-2 shadow-2xl shadow-black/60 min-w-[200px]"
                        :style="{ left: tooltipLeft }">
                        <div class="text-fine font-medium text-gray-300 mb-1">{{ fmtPeriodFull(hoverBar.period) }}</div>
                        <div v-for="seg in [...hoverBar.segments].reverse()" :key="seg.side" class="flex items-center gap-1.5 py-0.5 text-fine">
                            <span class="w-2 h-2 rounded-sm flex-shrink-0" :style="{ backgroundColor: segColor(seg.side) }" />
                            <span class="text-gray-400 truncate flex-1">{{ sideNameOf(seg.side) }}</span>
                            <span class="text-gray-200 tabular-nums">{{ formatNumber(seg.ships) }}</span>
                            <span class="text-gray-500 tabular-nums w-14 text-right">{{ formatIsk(seg.isk) }}</span>
                        </div>
                        <div class="flex items-center gap-1.5 pt-1 mt-1 border-t border-white/[0.08] text-fine">
                            <span class="w-2 h-2 flex-shrink-0" />
                            <span class="text-gray-500 flex-1">Total</span>
                            <span class="text-gray-200 tabular-nums">{{ formatNumber(hoverBar.totalShips) }}</span>
                            <span class="text-yellow-400/80 tabular-nums w-14 text-right">{{ formatIsk(hoverBar.totalIsk) }}</span>
                        </div>
                    </div>
                </div>

                <!-- legend -->
                <div v-if="!isAreaCampaign" class="flex items-center justify-center gap-4 mt-2 text-fine text-gray-500">
                    <template v-if="isOneSided">
                        <span class="flex items-center gap-1.5">
                            <span class="w-2 h-2 rounded-sm" :style="{ backgroundColor: segColor(SEG_KILLS) }" /> Kills
                        </span>
                        <span class="flex items-center gap-1.5">
                            <span class="w-2 h-2 rounded-sm" :style="{ backgroundColor: segColor(SEG_LOSSES) }" /> Losses
                        </span>
                    </template>
                    <template v-else>
                        <span v-for="s in campaign.sides" :key="s.side_index" class="flex items-center gap-1.5">
                            <span class="w-2 h-2 rounded-sm" :style="{ backgroundColor: segColor(s.side_index) }" /> {{ s.name }}
                        </span>
                    </template>
                </div>
            </div>

            <!-- Global breakdowns for location-only campaigns -->
            <div v-if="isAreaCampaign" class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
                <div class="glass-panel p-4">
                    <div class="text-fine text-gray-500 uppercase tracking-wider mb-2">Top Killers</div>
                    <div class="space-y-0.5">
                        <NuxtLink v-for="k in (stats.topKillersOverall ?? []).slice(0, 10)" :key="k.characterId" :to="`/character/${k.characterId}`"
                            class="flex items-center gap-2 px-1 py-1 rounded hover:bg-blue-500/[0.06] transition-colors">
                            <img :src="`/images/characters/${k.characterId}/portrait?size=32`" class="w-5 h-5 rounded-full flex-shrink-0" loading="lazy" alt="">
                            <span class="text-xs text-gray-300 truncate flex-1">{{ k.name ?? `#${k.characterId}` }}</span>
                            <span v-if="k.corporationTicker" :class="CORP_TAG" v-tooltip="k.corporationName ?? undefined">{{ k.corporationTicker }}</span>
                            <span v-if="k.allianceTicker" :class="ALLY_TAG" v-tooltip="k.allianceName ?? undefined">{{ k.allianceTicker }}</span>
                            <span class="text-fine text-isk tabular-nums">{{ formatNumber(k.kills) }}</span>
                        </NuxtLink>
                        <div v-if="!(stats.topKillersOverall?.length)" class="text-fine text-gray-600 py-2">No kills yet</div>
                    </div>
                </div>
                <div class="glass-panel p-4">
                    <div class="text-fine text-gray-500 uppercase tracking-wider mb-2">Top Victims</div>
                    <div class="space-y-0.5">
                        <NuxtLink v-for="v in (stats.topVictimsOverall ?? []).slice(0, 10)" :key="v.characterId" :to="`/character/${v.characterId}`"
                            class="flex items-center gap-2 px-1 py-1 rounded hover:bg-blue-500/[0.06] transition-colors">
                            <img :src="`/images/characters/${v.characterId}/portrait?size=32`" class="w-5 h-5 rounded-full flex-shrink-0" loading="lazy" alt="">
                            <span class="text-xs text-gray-300 truncate flex-1">{{ v.name ?? `#${v.characterId}` }}</span>
                            <span v-if="v.corporationTicker" :class="CORP_TAG" v-tooltip="v.corporationName ?? undefined">{{ v.corporationTicker }}</span>
                            <span v-if="v.allianceTicker" :class="ALLY_TAG" v-tooltip="v.allianceName ?? undefined">{{ v.allianceTicker }}</span>
                            <span class="text-fine text-red-400 tabular-nums">{{ formatNumber(v.losses) }}</span>
                        </NuxtLink>
                        <div v-if="!(stats.topVictimsOverall?.length)" class="text-fine text-gray-600 py-2">No losses yet</div>
                    </div>
                </div>
                <div class="glass-panel p-4">
                    <div class="text-fine text-gray-500 uppercase tracking-wider mb-2">Ships Lost by Class</div>
                    <div class="space-y-0.5">
                        <div v-for="g in (stats.shipClassesOverall ?? []).slice(0, 10)" :key="g.groupId" class="flex items-center gap-2 px-1 py-1">
                            <span class="text-xs text-gray-300 truncate flex-1">{{ g.name ?? `Group ${g.groupId}` }}</span>
                            <span class="text-fine text-red-400 tabular-nums">{{ formatNumber(g.losses) }}</span>
                            <span class="text-fine text-gray-600 tabular-nums w-14 text-right">{{ formatIsk(g.iskLost) }}</span>
                        </div>
                        <div v-if="!(stats.shipClassesOverall?.length)" class="text-fine text-gray-600 py-2">No losses yet</div>
                    </div>
                </div>
            </div>

            <!-- Breakdowns per side -->
            <div v-if="!isAreaCampaign" class="grid grid-cols-1 gap-4 mb-4" :class="isOneSided ? '' : 'lg:grid-cols-2'">
                <div v-for="s in campaign.sides" :key="s.side_index" class="glass-panel p-4">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] mb-3" :class="SIDE_TEXT[s.side_index]"
                        :style="sideAccents[s.side_index] ? { color: sideAccents[s.side_index]!.accent } : undefined">{{ s.name }}</div>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <!-- Top killers -->
                        <div>
                            <div class="text-fine text-gray-500 uppercase tracking-wider mb-2">Top Killers</div>
                            <div class="space-y-0.5">
                                <NuxtLink v-for="k in (stats.topKillersBySide[sideKey(s.side_index)] ?? []).slice(0, 8)" :key="k.characterId"
                                    :to="`/character/${k.characterId}`"
                                    class="flex items-center gap-2 px-1 py-1 rounded hover:bg-blue-500/[0.06] transition-colors">
                                    <img :src="`/images/characters/${k.characterId}/portrait?size=32`" class="w-5 h-5 rounded-full flex-shrink-0" loading="lazy" alt="">
                                    <span class="text-xs text-gray-300 truncate flex-1">{{ k.name ?? `#${k.characterId}` }}</span>
                                    <span v-if="k.corporationTicker" :class="CORP_TAG" v-tooltip="k.corporationName ?? undefined">{{ k.corporationTicker }}</span>
                                    <span v-if="k.allianceTicker" :class="ALLY_TAG" v-tooltip="k.allianceName ?? undefined">{{ k.allianceTicker }}</span>
                                    <span class="text-fine text-isk tabular-nums">{{ formatNumber(k.kills) }}</span>
                                </NuxtLink>
                                <div v-if="!(stats.topKillersBySide[sideKey(s.side_index)]?.length)" class="text-fine text-gray-600 py-2">No kills yet</div>
                            </div>
                        </div>
                        <!-- Ship classes lost -->
                        <div>
                            <div class="text-fine text-gray-500 uppercase tracking-wider mb-2">Ships Lost by Class</div>
                            <div class="space-y-0.5">
                                <div v-for="g in (stats.shipClassesBySide[sideKey(s.side_index)] ?? []).slice(0, 8)" :key="g.groupId"
                                    class="flex items-center gap-2 px-1 py-1">
                                    <span class="text-xs text-gray-300 truncate flex-1">{{ g.name ?? `Group ${g.groupId}` }}</span>
                                    <span class="text-fine text-red-400 tabular-nums">{{ formatNumber(g.losses) }}</span>
                                    <span class="text-fine text-gray-600 tabular-nums w-14 text-right">{{ formatIsk(g.iskLost) }}</span>
                                </div>
                                <div v-if="!(stats.shipClassesBySide[sideKey(s.side_index)]?.length)" class="text-fine text-gray-600 py-2">No losses yet</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Hottest systems -->
            <div v-if="stats.topSystems?.length" class="glass-panel p-4 mb-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-3">Hottest Systems</div>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-0.5">
                    <NuxtLink v-for="sys in stats.topSystems" :key="sys.systemId" :to="`/system/${sys.systemId}`"
                        class="flex items-center gap-2.5 px-1.5 py-1.5 rounded hover:bg-blue-500/[0.06] transition-colors">
                        <img :src="`/images/systems/${sys.systemId}?size=64`" class="w-7 h-7 rounded flex-shrink-0 bg-gray-900" loading="lazy" :alt="sys.name ?? ''">
                        <div class="min-w-0 flex-1">
                            <span class="text-xs text-gray-200">{{ sys.name ?? `System ${sys.systemId}` }}</span>
                            <span class="text-fine text-gray-500 ml-1.5">{{ sys.regionName }}</span>
                        </div>
                        <span class="text-fine text-gray-400 tabular-nums">{{ formatNumber(sys.kills) }} kills</span>
                        <span class="text-fine text-yellow-400/80 tabular-nums w-16 text-right">{{ formatIsk(sys.iskDestroyed) }}</span>
                    </NuxtLink>
                </div>
            </div>
        </template>

        <!-- Killfeed -->
        <div class="glass-panel p-4">
            <div class="flex items-center justify-between mb-3 gap-3 flex-wrap">
                <div class="flex items-center gap-1.5">
                    <button
                        class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer border"
                        :class="killfeedTab === null
                            ? 'bg-blue-500/20 text-blue-400 border-blue-500/40'
                            : 'bg-white/[0.02] text-gray-500 border-white/[0.06] hover:bg-white/[0.05]'"
                        @click="killfeedTab = null">
                        {{ isAreaCampaign ? 'All activity' : 'Combined' }}
                    </button>
                    <button v-for="s in campaign.sides" :key="s.side_index"
                        class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer border"
                        :class="killfeedTab === s.side_index
                            ? 'bg-white/[0.06] border-white/[0.15]'
                            : 'bg-white/[0.02] text-gray-500 border-white/[0.06] hover:bg-white/[0.05]'"
                        :style="killfeedTab === s.side_index ? { color: segColor(s.side_index) } : undefined"
                        @click="killfeedTab = s.side_index">
                        {{ s.name }} kills
                    </button>
                </div>
                <div class="flex items-center gap-3 text-fine text-gray-500">
                    <span v-for="s in campaign.sides" :key="s.side_index" class="flex items-center gap-1">
                        <span class="w-2 h-2 rounded-sm" :style="{ backgroundColor: segColor(s.side_index) }" />
                        {{ s.name }} losses
                    </span>
                </div>
            </div>
            <LazyKillList
                :key="`killfeed-${killfeedTab ?? 'all'}`"
                :api-endpoint="`/api/campaign/${campaignId}/killlist`"
                :extra-params="killfeedTab !== null ? { side: killfeedTab } : undefined"
                :side-entities="killlistSides"
                :side-colors="sideAccentHexes"
                :stream-topics="null" />
        </div>
        </template>

        <!-- ── Intel tab ────────────────────────────────────────────────── -->
        <div v-else-if="activeTab === 'intel'" class="grid grid-cols-1 gap-4" :class="isOneSided ? '' : 'lg:grid-cols-2'">
            <div v-for="s in campaign.sides" :key="s.side_index" class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] mb-3" :class="SIDE_TEXT[s.side_index]"
                    :style="sideAccents[s.side_index] ? { color: sideAccents[s.side_index]!.accent } : undefined">{{ s.name }}</div>

                <template v-if="intelFor(s.side_index)">
                    <div v-for="section in [
                        { label: 'Fleet Commanders', hint: 'flew a Monitor', pilots: intelFor(s.side_index)!.fcs, total: intelFor(s.side_index)!.fcs.length, show: 20 },
                        { label: 'Logistics', hint: null, pilots: intelFor(s.side_index)!.logistics, total: intelFor(s.side_index)!.logisticsCount, show: 12 },
                        { label: 'Capital Pilots', hint: null, pilots: intelFor(s.side_index)!.capitals, total: intelFor(s.side_index)!.capitalsCount, show: 15 },
                    ]" :key="section.label" class="mb-4 last:mb-0">
                        <div class="flex items-center gap-2 mb-2">
                            <div class="text-fine text-gray-500 uppercase tracking-wider">{{ section.label }}</div>
                            <span class="text-fine text-gray-600 tabular-nums">{{ formatNumber(section.total) }}</span>
                            <span v-if="section.hint" class="text-fine text-gray-700">· {{ section.hint }}</span>
                        </div>
                        <div v-if="section.pilots.length" class="space-y-0.5">
                            <NuxtLink v-for="p in section.pilots.slice(0, section.show)" :key="p.characterId"
                                :to="`/character/${p.characterId}`"
                                class="flex items-center gap-2 px-1.5 py-1 rounded hover:bg-blue-500/[0.06] transition-colors">
                                <img :src="`/images/characters/${p.characterId}/portrait?size=32`"
                                    class="w-5 h-5 rounded-full flex-shrink-0" loading="lazy" alt="">
                                <div class="min-w-0 flex-1">
                                    <span class="text-xs text-gray-300">{{ p.name ?? `#${p.characterId}` }}</span>
                                    <span class="text-fine text-gray-600 ml-1.5">{{ p.shipName }}</span>
                                </div>
                                <Icon v-if="p.died" name="lucide:skull" class="text-fine text-red-400/70 flex-shrink-0" v-tooltip="'Died during the campaign'" />
                                <span class="text-fine text-gray-500 tabular-nums w-14 text-right flex-shrink-0">{{ fmtDamage(p.damage) }} dmg</span>
                            </NuxtLink>
                            <div v-if="section.total > section.show" class="text-fine text-gray-600 px-1.5 pt-1">
                                +{{ formatNumber(section.total - section.show) }} more
                            </div>
                        </div>
                        <div v-else class="text-fine text-gray-600 py-1 px-1.5">None spotted</div>
                    </div>
                </template>
                <div v-else class="text-fine text-gray-600 py-4 text-center">
                    Intel is generated with the next stats refresh.
                </div>
            </div>
        </div>

        <!-- ── Comments tab ─────────────────────────────────────────────── -->
        <div v-else-if="activeTab === 'comments'" class="glass-panel p-4">
            <LazyCommentsCommentList
                :target-type="COMMENT_TARGET_CAMPAIGN"
                :target-id="0"
                :target-slug="campaignId"
                :show-header="false"
            />
        </div>
    </div>

    <div v-else-if="pending" class="flex items-center justify-center py-20">
        <Icon name="lucide:loader-2" class="text-2xl text-gray-500 animate-spin" />
    </div>
</template>
