<script setup lang="ts">
/**
 * Campaign summary card — the single card design used by /campaigns and the
 * campaigns tab on entity pages, so a campaign looks the same wherever it is
 * listed.
 *
 * The reading order is deliberate: identity (name + state) → what it covers →
 * who is ahead (balance bar) → the roster → the totals footer.
 */
import { campaignSideAccents } from '#shared/utils/entityAccent'

interface CardEntity {
    entity_type: number
    entity_id: number
}

interface CardSide {
    side_index: number
    name: string
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
    palette?: { main_color?: string; secondary_color?: string; tertiary_color?: string } | null
    entities: CardEntity[]
}

interface CardCampaign {
    campaign_id: string
    mode?: 'conflict' | 'area'
    name: string
    description: string | null
    status: number
    visibility?: number
    start_time: string
    end_time: string | null
    location?: { systemIds?: number[]; constellationIds?: number[]; regionIds?: number[] } | null
    creator: { character_id: number; name: string | null }
    totals: { killCount: number; iskDestroyed: number } | null
    sides: CardSide[]
}

const props = defineProps<{ campaign: CardCampaign }>()

// Area campaigns have no sides — the API says so explicitly, but entity-page
// payloads predate the field, so fall back to the shape of the data.
const isArea = computed(() => (props.campaign.mode ?? (props.campaign.sides.length ? 'conflict' : 'area')) === 'area')
const isOneSided = computed(() => !isArea.value && props.campaign.sides.length === 1)

// Computed once per campaign rather than per side — the accent derivation
// walks OKLCH conversions for every entity palette.
const accents = computed(() => campaignSideAccents(props.campaign.sides.map(s => s.palette)))

const sideColor = (side: CardSide) => campaignSideFill(side.side_index, accents.value[side.side_index])

/** Hard-segmented identity rail: one band per side, amber for area campaigns. */
const railStyle = computed(() => {
    if (isArea.value) return { backgroundColor: '#f59e0b' }
    const colors = props.campaign.sides.map(s => sideColor(s))
    if (!colors.length) return { backgroundColor: CAMPAIGN_NEUTRAL_FILL }
    if (colors.length === 1) return { backgroundColor: colors[0]! }
    const stops = colors
        .flatMap((c, i) => [`${c} ${(i / colors.length) * 100}%`, `${c} ${((i + 1) / colors.length) * 100}%`])
        .join(', ')
    return { backgroundImage: `linear-gradient(to bottom, ${stops})` }
})

const timeRange = computed(() => {
    const start = formatDate(props.campaign.start_time)
    return props.campaign.end_time ? `${start} — ${formatDate(props.campaign.end_time)}` : `${start} — ongoing`
})

const areaLabel = computed(() => {
    const loc = props.campaign.location
    const systems = loc?.systemIds?.length ?? 0
    const constellations = loc?.constellationIds?.length ?? 0
    const regions = loc?.regionIds?.length ?? 0
    return [
        systems ? `${systems} system${systems === 1 ? '' : 's'}` : '',
        constellations ? `${constellations} constellation${constellations === 1 ? '' : 's'}` : '',
        regions ? `${regions} region${regions === 1 ? '' : 's'}` : '',
    ].filter(Boolean).join(' · ')
})

const entityImage = (e: CardEntity): string => {
    if (e.entity_type === 2) return `/images/alliances/${e.entity_id}/logo?size=64`
    if (e.entity_type === 1) return `/images/corporations/${e.entity_id}/logo?size=64`
    return `/images/characters/${e.entity_id}/portrait?size=64`
}
</script>

<template>
    <NuxtLink :to="`/campaign/${campaign.campaign_id}`"
        class="group relative flex flex-col h-full rounded-lg bg-white/[0.04] border border-white/[0.08] overflow-hidden hover:border-blue-500/30 hover:bg-blue-500/[0.03] transition-colors">
        <!-- Side identity rail -->
        <div class="absolute inset-y-0 left-0 w-[3px] opacity-80 group-hover:opacity-100 transition-opacity" :style="railStyle" />

        <!-- Title + state -->
        <div class="flex items-start justify-between gap-3 pl-5 pr-4 pt-3.5">
            <div class="min-w-0">
                <h3 class="text-sm font-bold text-white truncate group-hover:text-blue-300 transition-colors">{{ campaign.name }}</h3>
                <div class="flex items-center gap-1.5 mt-1 text-fine text-gray-500">
                    <Icon :name="isArea ? 'lucide:map-pinned' : isOneSided ? 'lucide:crosshair' : 'lucide:swords'" class="text-fine flex-shrink-0" />
                    <span class="truncate">{{ timeRange }}</span>
                </div>
            </div>

            <div class="flex items-center gap-1.5 flex-shrink-0">
                <span v-if="campaign.visibility === 2"
                    class="inline-flex items-center px-1.5 py-0.5 rounded text-fine font-medium bg-indigo-500/15 text-indigo-300"
                    v-tooltip="'Killboard only — not listed on EVE-KILL'">
                    <Icon name="lucide:shield" class="text-fine" />
                </span>
                <span v-else-if="campaign.visibility === 1"
                    class="inline-flex items-center px-1.5 py-0.5 rounded text-fine font-medium bg-white/[0.06] text-gray-400"
                    v-tooltip="'Private — owner and selected-domain members only'">
                    <Icon name="lucide:lock" class="text-fine" />
                </span>
                <span v-if="campaign.status === 0" class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-medium bg-amber-500/15 text-amber-400">
                    <Icon name="lucide:loader-2" class="text-fine animate-spin" /> Computing
                </span>
                <span v-else-if="campaign.status === 2" class="px-2 py-0.5 rounded text-fine font-medium bg-white/[0.06] text-gray-500">Archived</span>
                <span v-else class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-fine font-medium bg-green-500/15 text-isk">
                    <span class="w-1.5 h-1.5 rounded-full bg-isk animate-pulse" /> Active
                </span>
            </div>
        </div>

        <p v-if="campaign.description" class="pl-5 pr-4 mt-2 text-xs text-gray-500 line-clamp-2">{{ campaign.description }}</p>

        <!-- Battlefield scope (area) or side scoreboard (conflict) -->
        <div class="pl-5 pr-4 mt-3">
            <div v-if="isArea" class="flex items-center gap-2.5 rounded-lg border border-amber-500/20 bg-amber-500/[0.06] px-3 py-2.5">
                <Icon name="lucide:map-pinned" class="text-amber-400 flex-shrink-0" />
                <div class="min-w-0">
                    <div class="text-xs font-medium text-amber-300">All battlefield activity</div>
                    <div class="text-fine text-gray-500 truncate">{{ areaLabel || 'Scope pending' }}</div>
                </div>
            </div>

            <template v-else>
                <CampaignBalanceBar :sides="campaign.sides" :accents="accents" compact class="mb-2.5" />
                <div class="space-y-1">
                    <div v-for="s in campaign.sides" :key="s.side_index" class="flex items-center gap-2">
                        <div class="flex -space-x-1 flex-shrink-0">
                            <EveImage v-for="e in s.entities.slice(0, 4)" :key="`${e.entity_type}-${e.entity_id}`"
                                :src="entityImage(e)" :size="32" alt="" class="w-5 h-5 rounded ring-1 ring-black/40 bg-gray-900" />
                            <span v-if="s.entities.length > 4"
                                class="w-5 h-5 rounded ring-1 ring-black/40 bg-white/[0.06] text-fine text-gray-500 flex items-center justify-center tabular-nums">
                                +{{ s.entities.length - 4 }}
                            </span>
                        </div>
                        <span class="text-xs font-medium truncate flex-1" :style="{ color: sideColor(s) }">{{ s.name }}</span>
                        <span class="text-fine text-gray-500 tabular-nums flex-shrink-0">
                            <span class="text-isk">{{ formatNumber(s.kills) }}</span> kills ·
                            <span class="text-yellow-400">{{ formatIsk(s.isk_destroyed) }}</span>
                        </span>
                    </div>
                </div>
            </template>
        </div>

        <!-- Totals -->
        <div class="mt-auto flex items-center justify-between gap-3 pl-5 pr-4 py-2.5 mt-3 border-t border-white/[0.06] bg-black/15 text-fine">
            <span v-if="campaign.totals" class="flex items-center gap-3 min-w-0 text-gray-500">
                <span class="flex items-center gap-1.5">
                    <Icon name="lucide:crosshair" class="text-fine text-gray-600 flex-shrink-0" />
                    <span class="tabular-nums text-gray-400">{{ formatNumber(campaign.totals.killCount) }}</span> killmails
                </span>
                <span class="flex items-center gap-1.5">
                    <Icon name="lucide:coins" class="text-fine text-yellow-400/50 flex-shrink-0" />
                    <span class="tabular-nums text-yellow-400/90">{{ formatIsk(campaign.totals.iskDestroyed) }}</span> destroyed
                </span>
            </span>
            <span v-else class="text-gray-600">Stats pending…</span>
            <span v-if="campaign.creator.name" class="text-gray-600 truncate">by {{ campaign.creator.name }}</span>
        </div>
    </NuxtLink>
</template>
