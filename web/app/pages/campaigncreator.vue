<script setup lang="ts">
import {
    CAMPAIGN_MAX_LOCATION_CONSTELLATIONS,
    CAMPAIGN_MAX_LOCATION_REGIONS,
    CAMPAIGN_MAX_LOCATION_SYSTEMS,
    CAMPAIGN_MAX_PRIVATE_WINDOW_DAYS,
    CAMPAIGN_MAX_PUBLIC_ONGOING_PER_USER,
    CAMPAIGN_MAX_PUBLIC_WINDOW_DAYS,
    CAMPAIGN_PRIZE_DEFAULT_SPLITS,
    CAMPAIGN_PRIZE_METRIC,
} from '#shared/utils/campaignPolicy'

/** What SearchPicker emits on select. */
interface PickedHit {
    type: 'alliance' | 'corporation' | 'character' | 'system' | 'constellation' | 'region'
    id: number
    name: string
    ticker: string | null
}

interface PickedEntity {
    id: number
    name: string
    type: 'character' | 'corporation' | 'alliance'
}

interface SideDraft {
    name: string
    entities: PickedEntity[]
}

interface PickedLocation {
    id: number
    name: string
    type: 'system' | 'constellation' | 'region'
}

interface PersonalWalletBalance {
    balance: string
    availableBalance: string
    reservedBalance: string
    totalBalance: string
}

const { isAuthenticated, user } = useAuth()
const {
    data: walletBalance,
    pending: walletBalancePending,
    refresh: refreshWalletBalance,
} = await useApiFetch<PersonalWalletBalance>('/api/user/wallet/balance', {
    immediate: isAuthenticated.value,
})
watch(isAuthenticated, (authenticated) => {
    if (authenticated) void refreshWalletBalance()
})

useHead({ title: 'Campaign Creator' })
useSeoMeta({
    description: 'Create a campaign on EVE-KILL: track a war between alliances, corporations or characters with live scoreboards, ISK stats and killmail feeds.',
    ogTitle: 'Campaign Creator — EVE-KILL',
    ogDescription: 'Bundle killmails by attackers, victims, systems or timeframe into a named campaign.',
})

useSchemaOrg([
    defineBreadcrumb({
        itemListElement: [
            { name: 'Home', item: '/' },
            { name: 'Campaign Creator' },
        ],
    }),
])

const MAX_SIDES = 4
const MAX_ENTITIES = 15

// ── Basics ───────────────────────────────────────────────────────────────────
const name = ref('')
const description = ref('')
const trackingMode = ref<'conflict' | 'single' | 'area'>('conflict')

// ── Visibility ───────────────────────────────────────────────────────────────
// 0 = public, 1 = private (unlisted, link only), 2 = killboard only.
const visibility = ref<0 | 1 | 2>(0)
const MAX_ALLOWED_ENTITIES = 10

// Killboard grants for area campaigns (no sides to infer an audience from):
// which entities' custom killboards may list this campaign.
const allowedEntities = ref<PickedEntity[]>([])

const addAllowedEntity = (p: PickedHit) => {
    if (allowedEntities.value.length >= MAX_ALLOWED_ENTITIES) return
    if (allowedEntities.value.some(e => e.type === p.type && e.id === p.id)) return
    allowedEntities.value.push({ id: p.id, name: p.name, type: p.type as PickedEntity['type'] })
}

const removeAllowedEntity = (e: PickedEntity) => {
    allowedEntities.value = allowedEntities.value.filter(x => !(x.type === e.type && x.id === e.id))
}

// ── Time window ──────────────────────────────────────────────────────────────
const defaultStart = () => {
    const d = new Date()
    d.setUTCHours(0, 0, 0, 0)
    return isoToEveInput(d)
}
const startTime = ref(defaultStart())
const ongoing = ref(true)
const endTime = ref('')

// ── Optional funded prize pool ───────────────────────────────────────────────
const prizeEnabled = ref(false)
const prizeMetric = ref<0 | 1 | 2 | 3>(CAMPAIGN_PRIZE_METRIC.KILLS)
const prizeWinnerCount = ref(3)
const prizePayoutPercentages = ref([...CAMPAIGN_PRIZE_DEFAULT_SPLITS[3]!])
const prizeInitialContribution = ref('')
const fundingRequestId = ref('')
const prizeMetricOptions = [
    { value: CAMPAIGN_PRIZE_METRIC.KILLS, label: 'Most kills', description: 'Each qualifying killmail counts once per pilot.' },
    { value: CAMPAIGN_PRIZE_METRIC.LOSSES, label: 'Most losses', description: 'Rewards the pilots with the most ship losses.' },
    { value: CAMPAIGN_PRIZE_METRIC.ISK_DESTROYED, label: 'Most ISK destroyed', description: 'Ranks pilots by the value of their qualifying kills.' },
    { value: CAMPAIGN_PRIZE_METRIC.ISK_LOST, label: 'Most ISK lost', description: 'Rewards the pilots who lose the most ship value.' },
] as const

const resetPrizeSplit = () => {
    prizePayoutPercentages.value = [...CAMPAIGN_PRIZE_DEFAULT_SPLITS[prizeWinnerCount.value]!]
}
watch(prizeWinnerCount, resetPrizeSplit)
watch(prizeEnabled, (enabled) => {
    if (enabled) ongoing.value = false
    else prizeInitialContribution.value = ''
})

const walletAvailableBalance = computed(() => {
    const value = Number(walletBalance.value?.availableBalance ?? walletBalance.value?.balance)
    return Number.isFinite(value) ? value : null
})
const normalizedInitialContribution = computed(() => prizeInitialContribution.value.trim())
const initialContributionValue = computed(() => Number(normalizedInitialContribution.value || 0))
const prizePercentageTotal = computed(() =>
    prizePayoutPercentages.value.reduce((sum, percentage) => sum + (Number(percentage) || 0), 0))
const prizeError = computed(() => {
    if (!prizeEnabled.value) return null
    if (ongoing.value || !endTime.value) return 'Prize campaigns require an end time'
    if (eveInputToDate(endTime.value).getTime() <= Date.now()) return 'Prize campaigns must end in the future'
    if (prizePayoutPercentages.value.some(percentage => !Number.isInteger(Number(percentage)) || Number(percentage) < 1)) {
        return 'Every winning place must receive at least 1%'
    }
    if (prizePercentageTotal.value !== 100) return 'Payout percentages must total exactly 100%'
    if (normalizedInitialContribution.value && !/^\d+(?:\.\d{1,2})?$/.test(normalizedInitialContribution.value)) {
        return 'Initial funding must be a positive ISK amount with at most two decimals'
    }
    if (initialContributionValue.value > 0 && walletBalancePending.value) {
        return 'Your EK Wallet balance is still loading'
    }
    if (
        initialContributionValue.value > 0
        && (walletAvailableBalance.value === null || initialContributionValue.value > walletAvailableBalance.value)
    ) {
        return 'Initial funding exceeds your available EK Wallet balance'
    }
    return null
})

const formatWalletIsk = (value: string | number | null | undefined) => {
    const amount = Number(value)
    if (!Number.isFinite(amount)) return '— ISK'
    return `${Math.round(amount).toLocaleString('en-US')} ISK`
}

const useMaximumWalletBalance = () => {
    const available = walletBalance.value?.availableBalance ?? walletBalance.value?.balance
    if (available) prizeInitialContribution.value = available
}

const timeError = computed(() => {
    if (!startTime.value) return 'Start time is required'
    if (!ongoing.value && !endTime.value) return 'Set an end time or mark the campaign as ongoing'
    if (!ongoing.value && endTime.value && eveInputToDate(endTime.value) <= eveInputToDate(startTime.value)) return 'End must be after start'
    const start = eveInputToDate(startTime.value)
    const end = ongoing.value ? new Date() : eveInputToDate(endTime.value)
    const maxDays = visibility.value === 0
        ? CAMPAIGN_MAX_PUBLIC_WINDOW_DAYS
        : CAMPAIGN_MAX_PRIVATE_WINDOW_DAYS
    if ((end.getTime() - start.getTime()) / 86_400_000 > maxDays) {
        return `${visibility.value === 0 ? 'Public' : 'Non-public'} campaigns may cover at most ${maxDays} days`
    }
    return null
})

// ── Location scope (optional) ────────────────────────────────────────────────
const locations = ref<PickedLocation[]>([])

// Mirrors the Go campaign policy's per-type caps so invalid choices can be
// rejected before submission as well as by the API.
const LOCATION_CAPS: Record<PickedLocation['type'], number> = {
    system: CAMPAIGN_MAX_LOCATION_SYSTEMS,
    constellation: CAMPAIGN_MAX_LOCATION_CONSTELLATIONS,
    region: CAMPAIGN_MAX_LOCATION_REGIONS,
}

const LOCATION_ICONS: Record<PickedLocation['type'], string> = {
    system: 'lucide:map-pin',
    constellation: 'lucide:compass',
    region: 'lucide:map',
}

const addLocation = (p: PickedHit) => {
    const type = p.type as PickedLocation['type']
    if (locations.value.some(l => l.type === type && l.id === p.id)) return
    if (locations.value.filter(l => l.type === type).length >= LOCATION_CAPS[type]) return
    locations.value.push({ id: p.id, name: p.name, type })
}

const locationIds = (type: PickedLocation['type']) =>
    locations.value.filter(l => l.type === type).map(l => l.id)

const removeLocation = (loc: PickedLocation) => {
    locations.value = locations.value.filter(l => !(l.type === loc.type && l.id === loc.id))
}

// ── Sides ────────────────────────────────────────────────────────────────────
const makeSide = (sideName: string): SideDraft => ({ name: sideName, entities: [] })

const sides = ref<SideDraft[]>([makeSide('Side 1'), makeSide('Side 2')])

// One-sided mode reuses the sides machinery but only ever shows (and
// submits) the first side — extra sides drafted in conflict mode are kept
// around in case the user switches back.
const visibleSides = computed(() => trackingMode.value === 'single' ? sides.value.slice(0, 1) : sides.value)

const SIDE_BORDERS = ['border-red-500/20', 'border-blue-500/20', 'border-emerald-500/20', 'border-orange-500/20']
const SIDE_TEXTS = ['text-red-400', 'text-blue-400', 'text-emerald-400', 'text-orange-400']

const addSide = () => {
    if (sides.value.length >= MAX_SIDES) return
    sides.value.push(makeSide(`Side ${sides.value.length + 1}`))
}
const removeSide = (i: number) => {
    if (sides.value.length <= 1) return
    sides.value.splice(i, 1)
}

const isPicked = (type: string, id: number) =>
    visibleSides.value.some(s => s.entities.some(e => e.type === type && e.id === id))

const addEntity = (i: number, p: PickedHit) => {
    const side = sides.value[i]!
    if (side.entities.length >= MAX_ENTITIES) return
    if (isPicked(p.type, p.id)) return
    side.entities.push({ id: p.id, name: p.name, type: p.type as PickedEntity['type'] })
}

const removeEntity = (i: number, e: PickedEntity) => {
    const side = sides.value[i]!
    side.entities = side.entities.filter(x => !(x.type === e.type && x.id === e.id))
}

const entityImage = (e: { type: string; id: number }): string => {
    if (e.type === 'alliance') return `/images/alliances/${e.id}/logo?size=64`
    if (e.type === 'corporation') return `/images/corporations/${e.id}/logo?size=64`
    return `/images/characters/${e.id}/portrait?size=64`
}

// ── Submit ───────────────────────────────────────────────────────────────────
const submitting = ref(false)
const submitError = ref<string | null>(null)

const hasName = computed(() => name.value.trim().length >= 3)
const hasParticipants = computed(() => trackingMode.value === 'area'
    ? locations.value.length > 0
    : visibleSides.value.length >= 1 && visibleSides.value.every(s => s.entities.length >= 1))

const canSubmit = computed(() =>
    hasName.value && !timeError.value && !prizeError.value && hasParticipants.value)

/** What is still missing, phrased as the thing to go do. */
const checklist = computed(() => {
    const items: Array<{ label: string; ok: boolean; detail?: string | null }> = [
        { label: 'Name the campaign', ok: hasName.value, detail: hasName.value ? null : 'At least 3 characters' },
    ]
    if (trackingMode.value === 'area') {
        items.push({
            label: 'Pick a battlefield',
            ok: locations.value.length > 0,
            detail: locations.value.length ? null : 'At least one region, constellation or system',
        })
    } else {
        items.push({
            label: visibleSides.value.length > 1 ? 'Fill every side' : 'Pick who to track',
            ok: hasParticipants.value,
            detail: hasParticipants.value ? null : 'Each side needs at least one entity',
        })
    }
    items.push({ label: 'Set the time range', ok: !timeError.value, detail: timeError.value })
    if (prizeEnabled.value) items.push({ label: 'Configure the prize pool', ok: !prizeError.value, detail: prizeError.value })
    return items
})

// Live preview uses the same card the listing renders, so what you build is
// literally what /campaigns will show.
const ENTITY_TYPE_ID: Record<PickedEntity['type'], number> = { character: 0, corporation: 1, alliance: 2 }

const previewCampaign = computed(() => ({
    campaign_id: 'preview',
    mode: (trackingMode.value === 'area' ? 'area' : 'conflict') as 'area' | 'conflict',
    name: name.value.trim() || 'Untitled campaign',
    description: description.value.trim() || null,
    status: 1,
    visibility: visibility.value,
    start_time: startTime.value ? eveInputToIso(startTime.value) : new Date().toISOString(),
    end_time: ongoing.value || !endTime.value ? null : eveInputToIso(endTime.value),
    location: {
        systemIds: locationIds('system'),
        constellationIds: locationIds('constellation'),
        regionIds: locationIds('region'),
    },
    creator: { character_id: user.value?.characterId ?? 0, name: user.value?.characterName ?? null },
    totals: null,
    sides: visibleSides.value.map((s, i) => ({
        side_index: i,
        name: s.name.trim() || `Side ${i + 1}`,
        kills: 0,
        losses: 0,
        isk_destroyed: 0,
        isk_lost: 0,
        palette: null,
        entities: s.entities.map(e => ({ entity_type: ENTITY_TYPE_ID[e.type], entity_id: e.id })),
    })),
}))

const submit = async () => {
    if (!canSubmit.value || submitting.value) return
    submitting.value = true
    submitError.value = null
    try {
        if (initialContributionValue.value > 0 && !fundingRequestId.value) {
            fundingRequestId.value = crypto.randomUUID()
        }
        const result = await apiFetch<{ campaign_id: string }>('/api/campaign/create', {
            method: 'POST',
            body: {
                name: name.value.trim(),
                description: description.value.trim() || null,
                startTime: eveInputToIso(startTime.value),
                endTime: ongoing.value || !endTime.value ? null : eveInputToIso(endTime.value),
                location: {
                    systemIds: locationIds('system'),
                    constellationIds: locationIds('constellation'),
                    regionIds: locationIds('region'),
                },
                sides: trackingMode.value === 'area' ? [] : visibleSides.value.map(s => ({
                    name: s.name.trim(),
                    entities: s.entities.map(e => ({ type: e.type, id: e.id })),
                })),
                visibility: visibility.value,
                allowedEntities: visibility.value === 2 && trackingMode.value === 'area'
                    ? allowedEntities.value.map(e => ({ type: e.type, id: e.id, name: e.name }))
                    : [],
                prizePool: prizeEnabled.value
                    ? {
                        enabled: true,
                        metric: prizeMetric.value,
                        winnerCount: prizeWinnerCount.value,
                        payoutPercentages: prizePayoutPercentages.value.map(Number),
                        initialContribution: normalizedInitialContribution.value || '0',
                        fundingRequestId: initialContributionValue.value > 0 ? fundingRequestId.value : null,
                    }
                    : { enabled: false },
            },
        })
        await navigateTo(`/campaign/${result.campaign_id}`)
    } catch (e: any) {
        submitError.value = e.data?.message || 'Failed to create campaign'
    } finally {
        submitting.value = false
    }
}
</script>
<template>
    <div>
        <!-- Header -->
        <div class="glass-panel p-5 mb-4">
            <div class="flex items-start justify-between gap-4 flex-wrap">
                <div class="min-w-0">
                    <h1 class="flex items-center gap-2.5 text-xl font-bold text-white">
                        <span class="w-8 h-8 rounded-lg bg-blue-500/15 flex items-center justify-center flex-shrink-0">
                            <Icon name="lucide:flag" class="text-base text-blue-400" />
                        </span>
                        Campaign Creator
                    </h1>
                    <p class="text-sm text-gray-500 mt-2 max-w-3xl">
                        Track a conflict between sides, a single group's war effort, or all activity across a
                        battlefield. Everything you set here is editable afterwards except the tracking mode.
                    </p>
                </div>
                <NuxtLink to="/campaigns"
                    class="inline-flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-white/[0.07] hover:text-gray-200 transition-colors flex-shrink-0">
                    <Icon name="lucide:arrow-left" class="text-xs" />
                    All campaigns
                </NuxtLink>
            </div>
        </div>

        <!-- Login gate -->
        <LoginGate v-if="!isAuthenticated" message="You must be logged in to create campaigns" />

        <div v-else class="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_340px] gap-4 items-start">
            <!-- ── Form column ────────────────────────────────────────────── -->
            <div class="space-y-4 min-w-0">
                <!-- 1 · Tracking mode -->
                <CampaignFormSection :step="1" title="Tracking mode" hint="Decides which killmails count — this cannot be changed later.">
                    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
                        <button v-for="mode in ([
                            { key: 'conflict', label: 'Conflict', description: 'Track named sides and compare their kills, losses and efficiency.', icon: 'lucide:swords' },
                            { key: 'single', label: 'One-sided', description: 'Follow a single group — every kill and loss for the entities you pick.', icon: 'lucide:crosshair' },
                            { key: 'area', label: 'Area activity', description: 'Track every killmail in selected regions, constellations or systems, regardless of participants.', icon: 'lucide:map-pinned' },
                        ] as const)" :key="mode.key" @click="trackingMode = mode.key"
                            class="flex items-start gap-3 rounded-lg border p-3 text-left transition-colors cursor-pointer"
                            :class="trackingMode === mode.key
                                ? 'border-blue-500/40 bg-blue-500/[0.08]'
                                : 'border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.05]'">
                            <Icon :name="mode.icon" class="text-lg mt-0.5 flex-shrink-0" :class="trackingMode === mode.key ? 'text-blue-400' : 'text-gray-500'" />
                            <div class="min-w-0">
                                <div class="text-sm font-medium" :class="trackingMode === mode.key ? 'text-white' : 'text-gray-300'">{{ mode.label }}</div>
                                <div class="text-fine text-gray-600 mt-0.5">{{ mode.description }}</div>
                            </div>
                        </button>
                    </div>
                </CampaignFormSection>

                <!-- 2 · Identity -->
                <CampaignFormSection :step="2" title="Name and description" hint="How the campaign shows up in listings and search." :done="hasName">
                    <div class="space-y-3">
                        <div>
                            <label class="text-fine text-gray-500 mb-1 block">Name</label>
                            <input v-model="name" type="text" maxlength="100" placeholder="e.g. The Battle for Vale — Imperium vs Fraternity"
                                class="w-full px-3 py-2.5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:border-blue-500/40 focus:outline-none" />
                            <p v-if="name.trim() && !hasName" class="text-fine text-amber-400 mt-1">At least 3 characters.</p>
                        </div>
                        <div>
                            <label class="text-fine text-gray-500 mb-1 block">Description <span class="text-gray-700">(optional)</span></label>
                            <textarea v-model="description" rows="2" maxlength="2000" placeholder="What is this war about?"
                                class="w-full px-3 py-2.5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-gray-300 placeholder-gray-600 focus:border-blue-500/40 focus:outline-none" />
                        </div>
                    </div>
                </CampaignFormSection>

                <!-- 3 · Participants (conflict / one-sided) -->
                <CampaignFormSection v-if="trackingMode !== 'area'" :step="3"
                    :title="trackingMode === 'single' ? 'Who to track' : 'The sides'"
                    :hint="trackingMode === 'single'
                        ? 'Every kill and loss for these alliances, corporations or characters.'
                        : 'Name each side and add the alliances, corporations or characters that fight for it.'"
                    :done="hasParticipants">
                    <template #aside>
                        <button v-if="trackingMode === 'conflict' && sides.length < MAX_SIDES" @click="addSide"
                            class="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-fine font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors cursor-pointer">
                            <Icon name="lucide:plus" class="text-fine" /> Add side ({{ sides.length }}/{{ MAX_SIDES }})
                        </button>
                    </template>

                    <div class="grid grid-cols-1 gap-3" :class="visibleSides.length > 1 ? 'md:grid-cols-2' : ''">
                        <div v-for="(side, i) in visibleSides" :key="i"
                            class="relative rounded-lg bg-white/[0.02] border p-3 pl-4 overflow-hidden" :class="SIDE_BORDERS[i]">
                            <div class="absolute inset-y-0 left-0 w-[3px]" :style="{ backgroundColor: CAMPAIGN_SIDE_FILL[i] }" />
                            <div class="flex items-center gap-2 mb-2.5">
                                <input v-model="side.name" type="text" maxlength="50"
                                    class="flex-1 min-w-0 px-2.5 py-1.5 rounded bg-white/[0.04] border border-white/[0.06] text-sm font-medium focus:border-blue-500/40 focus:outline-none"
                                    :class="SIDE_TEXTS[i]" />
                                <button v-if="trackingMode === 'conflict' && sides.length > 1" @click="removeSide(i)"
                                    class="px-2 py-1.5 rounded text-gray-600 hover:text-red-400 hover:bg-red-500/[0.08] transition-colors cursor-pointer flex-shrink-0" v-tooltip="'Remove side'">
                                    <Icon name="lucide:x" class="text-xs" />
                                </button>
                            </div>

                            <SearchPicker class="mb-2" :types="['alliance', 'corporation', 'character']"
                                :disabled="side.entities.length >= MAX_ENTITIES"
                                :placeholder="side.entities.length >= MAX_ENTITIES ? `Max ${MAX_ENTITIES} entities` : undefined"
                                :is-picked="isPicked"
                                @select="p => addEntity(i, p)" />

                            <div class="space-y-0.5 min-h-[60px]">
                                <div v-for="e in side.entities" :key="`${e.type}-${e.id}`"
                                    class="group flex items-center gap-2 px-1.5 py-1 rounded hover:bg-blue-500/[0.04] transition-colors">
                                    <img :src="entityImage(e)" class="w-5 h-5 rounded flex-shrink-0 bg-gray-900" loading="lazy" :alt="e.name">
                                    <span class="text-xs text-gray-300 truncate flex-1">{{ e.name }}</span>
                                    <span class="text-fine text-gray-600 capitalize flex-shrink-0">{{ e.type }}</span>
                                    <button @click="removeEntity(i, e)" class="opacity-0 group-hover:opacity-100 text-gray-500 hover:text-red-400 transition-opacity flex-shrink-0 cursor-pointer">
                                        <Icon name="lucide:x" class="text-fine" />
                                    </button>
                                </div>
                                <div v-if="!side.entities.length" class="text-center py-4 text-fine text-gray-600">
                                    Add up to {{ MAX_ENTITIES }} alliances, corporations or characters
                                </div>
                            </div>
                            <div class="text-fine text-gray-600 text-right mt-1 tabular-nums">{{ side.entities.length }}/{{ MAX_ENTITIES }}</div>
                        </div>
                    </div>
                </CampaignFormSection>

                <!-- 3/4 · Battlefield -->
                <CampaignFormSection :step="trackingMode === 'area' ? 3 : 4" title="Battlefield"
                    :optional="trackingMode !== 'area'"
                    :hint="trackingMode === 'area'
                        ? 'The regions, constellations or systems whose activity is tracked.'
                        : 'Limit the campaign to specific space. Leave empty to count kills everywhere.'"
                    :done="trackingMode === 'area' ? locations.length > 0 : undefined">
                    <SearchPicker :types="['region', 'constellation', 'system']"
                        :is-picked="(t, id) => locations.some(l => l.type === t && l.id === id)"
                        @select="addLocation" />
                    <div v-if="locations.length" class="flex flex-wrap gap-1.5 mt-3">
                        <div v-for="loc in locations" :key="`${loc.type}-${loc.id}`"
                            class="inline-flex items-center gap-1.5 px-2 py-1 rounded border border-amber-500/30 bg-amber-500/10 text-xs text-amber-400 font-medium">
                            <Icon :name="LOCATION_ICONS[loc.type]" class="text-fine" />
                            {{ loc.name }}
                            <button @click="removeLocation(loc)" class="ml-0.5 opacity-60 hover:opacity-100 cursor-pointer"><Icon name="lucide:x" class="text-fine" /></button>
                        </div>
                    </div>
                    <p v-else-if="trackingMode === 'area'" class="text-fine text-gray-600 mt-2">
                        Pick at least one region, constellation or system — up to
                        {{ CAMPAIGN_MAX_LOCATION_REGIONS }} regions,
                        {{ CAMPAIGN_MAX_LOCATION_CONSTELLATIONS }} constellations and
                        {{ CAMPAIGN_MAX_LOCATION_SYSTEMS }} systems.
                    </p>
                </CampaignFormSection>

                <!-- 5 · Time -->
                <CampaignFormSection :step="trackingMode === 'area' ? 4 : 5" title="Time range"
                    hint="EVE time (UTC). Kills outside this window are never counted."
                    :done="!timeError">
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                        <div>
                            <label class="text-fine text-gray-500 mb-1 block">Start</label>
                            <DateTimePicker v-model="startTime" :clearable="false" placeholder="Pick a start" />
                        </div>
                        <div>
                            <label class="text-fine text-gray-500 mb-1 block">End</label>
                            <div class="flex items-center gap-3">
                                <DateTimePicker v-model="endTime" :disabled="ongoing" class="flex-1 min-w-0" placeholder="Pick an end" />
                                <label class="flex items-center gap-1.5 text-xs text-gray-400 cursor-pointer whitespace-nowrap">
                                    <input v-model="ongoing" type="checkbox" class="accent-blue-500" /> Ongoing
                                </label>
                            </div>
                        </div>
                    </div>
                    <div v-if="timeError" class="mt-2 flex items-center gap-1.5 text-xs text-red-400">
                        <Icon name="lucide:alert-circle" class="text-xs flex-shrink-0" />{{ timeError }}
                    </div>
                    <div v-else-if="ongoing" class="mt-2 text-fine text-gray-600">
                        {{ trackingMode === 'area'
                            ? 'Ongoing area campaigns update hourly and remain active until manually archived.'
                            : 'Ongoing campaigns update hourly and auto-archive after 30 days without a matching kill.' }}
                        <span v-if="visibility === 0">
                            Each character may have {{ CAMPAIGN_MAX_PUBLIC_ONGOING_PER_USER }} active public ongoing campaigns.
                        </span>
                    </div>
                </CampaignFormSection>

                <!-- 6 · Visibility -->
                <CampaignFormSection :step="trackingMode === 'area' ? 5 : 6" title="Visibility" hint="Who can find and open this campaign.">
                    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
                        <button v-for="v in ([
                            { key: 0, label: 'Public', description: 'Listed on EVE-KILL for everyone.', icon: 'lucide:globe' },
                            { key: 1, label: 'Private', description: 'Only you, admins, and signed-in members of a custom domain you add it to can view it.', icon: 'lucide:lock' },
                            { key: 2, label: 'Killboard only', description: 'Unlisted on EVE-KILL; shown on the custom killboards of the entities involved.', icon: 'lucide:shield' },
                        ] as const)" :key="v.key" @click="visibility = v.key"
                            class="flex items-start gap-3 rounded-lg border p-3 text-left transition-colors cursor-pointer"
                            :class="visibility === v.key
                                ? 'border-blue-500/40 bg-blue-500/[0.08]'
                                : 'border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.05]'">
                            <Icon :name="v.icon" class="text-lg mt-0.5 flex-shrink-0" :class="visibility === v.key ? 'text-blue-400' : 'text-gray-500'" />
                            <div class="min-w-0">
                                <div class="text-sm font-medium" :class="visibility === v.key ? 'text-white' : 'text-gray-300'">{{ v.label }}</div>
                                <div class="text-fine text-gray-600 mt-0.5">{{ v.description }}</div>
                            </div>
                        </button>
                    </div>

                    <!-- Killboard grants — area campaigns have no sides to infer an audience from -->
                    <div v-if="visibility === 2 && trackingMode === 'area'" class="mt-4 pt-3 border-t border-white/[0.06]">
                        <p class="text-fine text-gray-600 mb-2">
                            Allow these entities to see it on their custom killboard — your own killboards always can.
                        </p>
                        <SearchPicker :types="['alliance', 'corporation', 'character']"
                            :disabled="allowedEntities.length >= MAX_ALLOWED_ENTITIES"
                            :placeholder="allowedEntities.length >= MAX_ALLOWED_ENTITIES ? `Max ${MAX_ALLOWED_ENTITIES} entities` : undefined"
                            :is-picked="(t, id) => allowedEntities.some(e => e.type === t && e.id === id)"
                            @select="addAllowedEntity" />
                        <div v-if="allowedEntities.length" class="flex flex-wrap gap-1.5 mt-3">
                            <div v-for="e in allowedEntities" :key="`${e.type}-${e.id}`"
                                class="inline-flex items-center gap-1.5 px-2 py-1 rounded border border-indigo-500/30 bg-indigo-500/10 text-xs text-indigo-300 font-medium">
                                <img :src="entityImage(e)" class="w-4 h-4 rounded" loading="lazy" alt="">
                                {{ e.name }}
                                <button @click="removeAllowedEntity(e)" class="ml-0.5 opacity-60 hover:opacity-100 cursor-pointer"><Icon name="lucide:x" class="text-fine" /></button>
                            </div>
                        </div>
                    </div>
                    <p v-else-if="visibility === 2" class="text-fine text-gray-600 mt-3">
                        Shown on custom killboards whose entities are on one of the sides, and always on your own killboards.
                    </p>
                </CampaignFormSection>

                <!-- 7 · Prize pool -->
                <CampaignFormSection :step="trackingMode === 'area' ? 6 : 7" title="Funded prize pool" optional
                    hint="Let pilots fund the campaign through the EVE-KILL.com corporation wallet and reward its top performers."
                    :done="prizeEnabled ? !prizeError : undefined">
                    <template #aside>
                        <label class="inline-flex items-center gap-2 text-xs text-gray-300 cursor-pointer whitespace-nowrap">
                            <input v-model="prizeEnabled" type="checkbox" class="accent-amber-500" />
                            Enable prizes
                        </label>
                    </template>

                    <div v-if="!prizeEnabled" class="flex items-center gap-2.5 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5 text-fine text-gray-600">
                        <Icon name="lucide:trophy" class="text-gray-700 flex-shrink-0" />
                        Prize campaigns need a fixed end time and lock their rules once funded.
                    </div>

                    <div v-else class="space-y-4">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                            <div>
                                <label class="text-fine text-gray-500 mb-1 block">Ranking metric</label>
                                <select v-model.number="prizeMetric"
                                    class="w-full px-3 py-2.5 rounded-lg bg-gray-900 border border-white/[0.08] text-sm text-gray-300 focus:border-amber-500/40 focus:outline-none">
                                    <option v-for="option in prizeMetricOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                                </select>
                                <p class="text-fine text-gray-600 mt-1">{{ prizeMetricOptions.find(option => option.value === prizeMetric)?.description }}</p>
                            </div>
                            <div>
                                <label class="text-fine text-gray-500 mb-1 block">Number of winners</label>
                                <select v-model.number="prizeWinnerCount"
                                    class="w-full px-3 py-2.5 rounded-lg bg-gray-900 border border-white/[0.08] text-sm text-gray-300 focus:border-amber-500/40 focus:outline-none">
                                    <option v-for="count in 8" :key="count + 2" :value="count + 2">Top {{ count + 2 }} pilots</option>
                                </select>
                                <p class="text-fine text-gray-600 mt-1">Only pilots with a positive score qualify.</p>
                            </div>
                        </div>

                        <div class="rounded-lg border border-amber-500/20 bg-amber-500/[0.06] p-3">
                            <div class="flex flex-col sm:flex-row sm:items-end gap-3">
                                <div class="flex-1 min-w-0">
                                    <div class="flex items-center justify-between gap-3 mb-1">
                                        <label class="text-fine text-gray-500">
                                            Fund from your EK Wallet at creation
                                            <span class="text-gray-700">(optional)</span>
                                        </label>
                                        <span class="text-fine text-gray-600">
                                            Available:
                                            <span class="text-amber-300">
                                                {{ formatWalletIsk(walletBalance?.availableBalance ?? walletBalance?.balance) }}
                                            </span>
                                        </span>
                                    </div>
                                    <div class="relative">
                                        <input v-model="prizeInitialContribution" type="text" inputmode="decimal" placeholder="0"
                                            class="w-full px-3 py-2.5 pr-12 rounded-lg bg-black/20 border border-white/[0.08] text-sm text-white tabular-nums placeholder-gray-700 focus:border-amber-500/40 focus:outline-none" />
                                        <span class="absolute right-3 top-2.5 text-xs text-gray-600">ISK</span>
                                    </div>
                                </div>
                                <button type="button" :disabled="walletAvailableBalance === null || walletAvailableBalance <= 0"
                                    class="px-3 py-2.5 rounded-lg bg-amber-500/15 text-xs font-medium text-amber-300 hover:bg-amber-500/25 disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
                                    @click="useMaximumWalletBalance">
                                    Use max
                                </button>
                            </div>
                            <p class="text-fine text-gray-600 mt-2">
                                Charged atomically with campaign creation. If creation fails, no ISK leaves your wallet. Funding immediately locks the prize rules.
                            </p>
                        </div>

                        <div>
                            <div class="flex items-center justify-between gap-3 mb-2">
                                <label class="text-fine text-gray-500">Payout split</label>
                                <button type="button" @click="resetPrizeSplit"
                                    class="text-fine text-blue-400 hover:text-blue-300 cursor-pointer">
                                    Reset to recommended
                                </button>
                            </div>
                            <div class="grid grid-cols-3 sm:grid-cols-5 lg:grid-cols-10 gap-2">
                                <label v-for="(_, index) in prizePayoutPercentages" :key="index" class="block">
                                    <span class="text-fine text-gray-600 mb-1 block">#{{ index + 1 }}</span>
                                    <div class="relative">
                                        <input v-model.number="prizePayoutPercentages[index]" type="number" min="1" max="100" step="1"
                                            class="w-full px-2 py-2 pr-5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-white tabular-nums focus:border-amber-500/40 focus:outline-none" />
                                        <span class="absolute right-1.5 top-2 text-xs text-gray-600">%</span>
                                    </div>
                                </label>
                            </div>
                            <div class="flex items-center justify-between mt-2 text-xs">
                                <span :class="prizePercentageTotal === 100 ? 'text-green-400' : 'text-red-400'">
                                    Total: {{ prizePercentageTotal }}%
                                </span>
                                <span class="text-gray-600">The split locks after the first matched contribution.</span>
                            </div>
                        </div>

                        <div v-if="prizeMetric === CAMPAIGN_PRIZE_METRIC.LOSSES || prizeMetric === CAMPAIGN_PRIZE_METRIC.ISK_LOST"
                            class="rounded-lg border border-amber-500/20 bg-amber-500/[0.08] px-3 py-2 text-xs text-amber-300">
                            This metric rewards losses and may encourage intentional welping. Campaign results are not guaranteed to be fraud-proof.
                        </div>
                        <div class="rounded-lg border border-blue-500/15 bg-blue-500/[0.06] px-3 py-2 text-fine text-gray-400">
                            Funding closes at the campaign end time. Official results are frozen 24 hours later so delayed killmails and wallet entries can catch up. Payouts are arranged manually through EVE-KILL Discord.
                        </div>
                        <div v-if="prizeError" class="flex items-center gap-1.5 text-xs text-red-400">
                            <Icon name="lucide:alert-circle" class="text-xs flex-shrink-0" />{{ prizeError }}
                        </div>
                    </div>
                </CampaignFormSection>
            </div>

            <!-- ── Sticky summary rail ────────────────────────────────────── -->
            <aside class="xl:sticky xl:top-4 space-y-3">
                <div class="glass-panel p-4">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-3">Preview</div>
                    <div class="pointer-events-none">
                        <CampaignCard :campaign="previewCampaign" />
                    </div>
                </div>

                <div class="glass-panel p-4">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-3">Before you create</div>
                    <ul class="space-y-2">
                        <li v-for="item in checklist" :key="item.label" class="flex items-start gap-2.5">
                            <Icon :name="item.ok ? 'lucide:check-circle-2' : 'lucide:circle'"
                                class="text-sm mt-px flex-shrink-0" :class="item.ok ? 'text-isk' : 'text-gray-600'" />
                            <div class="min-w-0">
                                <div class="text-xs" :class="item.ok ? 'text-gray-400' : 'text-gray-300'">{{ item.label }}</div>
                                <div v-if="item.detail" class="text-fine mt-0.5" :class="item.ok ? 'text-gray-600' : 'text-amber-400/80'">{{ item.detail }}</div>
                            </div>
                        </li>
                    </ul>

                    <div v-if="submitError" class="mt-3 flex items-start gap-2 text-xs text-red-400 bg-red-500/10 rounded-lg px-3 py-2">
                        <Icon name="lucide:alert-circle" class="text-sm mt-px flex-shrink-0" />
                        <span class="min-w-0">{{ submitError }}</span>
                    </div>

                    <button @click="submit" :disabled="!canSubmit || submitting"
                        class="w-full mt-4 py-2.5 rounded-lg text-sm font-medium transition-colors"
                        :class="canSubmit && !submitting
                            ? 'bg-blue-500 text-white hover:bg-blue-600 cursor-pointer'
                            : 'bg-white/[0.04] text-gray-600 cursor-not-allowed'">
                        <Icon v-if="submitting" name="lucide:loader-2" class="text-base animate-spin mr-2" />
                        {{ submitting ? 'Creating campaign…' : 'Create Campaign' }}
                    </button>
                    <p class="text-fine text-gray-600 text-center mt-2">
                        The workload is estimated before creation. Accepted campaigns compute in the background under hard size and runtime limits.
                    </p>
                </div>
            </aside>
        </div>
    </div>
</template>
