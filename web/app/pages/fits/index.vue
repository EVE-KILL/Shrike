<script setup lang="ts">
/**
 * /fits — ship fitting landing page / explorer.
 *
 * Sections (top to bottom):
 *   1. Hero          — title, tagline, "Create New Fit" CTA
 *   2. Quick Stats   — counter row across the top (osmium homage)
 *   3. Flavors of the Week — top canonical fits across all ships in the
 *                       last 30 days; click any card to load it into
 *                       the editor
 *   4. Top Alliance Doctrines — for the most-active alliances, their
 *                       signature fit; click to load it
 *   5. Popular Hulls — top 12 ships by recent fitting activity;
 *                       click drills into /item/:id/fittings
 *   6. Latest + Top Rated Community Fits — two-column grid with the
 *                       newest public user fits and the highest-rated
 *                       ones, both linking to /fit/:id
 *
 * No WASM on this page — pure consumption. The dogma engine only
 * loads on pages that actually calculate fits.
 */

import { killmailFitToEditorUrl } from '~/composables/fit/killmailToFit'
import FitRating from '~/components/fit/FitRating.vue'

useHead({ title: 'Ship Fittings' })
useSeoMeta({
    description:
        'Browse popular ship fittings from EVE Online killmails, share your own fits, and build new ones with the in-browser fitting tool.',
    ogTitle: 'Ship Fittings — EVE-KILL',
    ogDescription:
        'Popular fits from the last 30 days, alliance signature doctrines, community-shared builds, and a full in-browser fitting tool.',
})

// ---------- types ----------

interface QuickStats {
    fittings_known: number
    killmails_analyzed: number
    community_fits: number
    ratings_cast: number
}

interface FittingModule {
    slot_group: number
    ordinal: number
    type_id: number
    name: string | null
    charge_type_id: number | null
    charge_name: string | null
}

interface FittingDrone {
    type_id: number
    name: string | null
    quantity: number
}

interface FlavorFamily {
    family_hash: string
    ship_type_id: number
    ship_name: string | null
    canonical_fit_hash: string
    total_uses: number
    canonical_uses: number
    variant_count: number
    last_used: string
    fit_cost: number
    hull_cost: number | null
    modules: FittingModule[]
    drones: FittingDrone[]
}

interface AllianceDoctrine {
    alliance_id: number
    alliance_name: string | null
    total_losses: number
    family_hash: string
    ship_type_id: number
    ship_name: string | null
    canonical_fit_hash: string
    doctrine_uses: number
    doctrine_share: number
    last_used: string
    fit_cost: number
    hull_cost: number | null
    modules: FittingModule[]
    drones: FittingDrone[]
}

interface PopularShip {
    ship_type_id: number
    total_uses: number
    fit_count: number
    last_used: string
    ship_name: string | null
    group_id: number | null
}

interface CommunityFit {
    fit_id: string
    name: string
    description: string | null
    ship_type_id: number
    ship_name: string | null
    owner_character_id: number
    owner_name: string | null
    rating_avg: number | null
    rating_count: number
    created_at: string
    updated_at: string
    module_count: number
}

// ---------- fetches (all lazy, none block first paint) ----------

const { data: statsData } = await useApiFetch<QuickStats>('/api/fits/quick-stats', { lazy: true })
const { data: flavorsData, pending: flavorsPending } = await useApiFetch<{
    window_days: number
    families: FlavorFamily[]
}>('/api/fits/flavors-of-the-week', { lazy: true })
const { data: doctrinesData, pending: doctrinesPending } = await useApiFetch<{
    window_days: number
    doctrines: AllianceDoctrine[]
}>('/api/fits/top-alliance-doctrines', { lazy: true })
const { data: popularData, pending: popularPending } = await useApiFetch<{
    window_days: number
    ships: PopularShip[]
}>('/api/fits/popular-ships', { lazy: true })
const { data: communityData, pending: communityPending } = await useApiFetch<{
    fits: CommunityFit[]
}>('/api/fits/community-latest', { lazy: true })
const { data: topRatedData } = await useApiFetch<{
    fits: CommunityFit[]
}>('/api/fits/top-rated', { lazy: true })

const stats = computed(() => statsData.value)
const flavors = computed(() => flavorsData.value?.families ?? [])
const doctrines = computed(() => doctrinesData.value?.doctrines ?? [])
const popularShips = computed(() => popularData.value?.ships ?? [])
const communityFits = computed(() => communityData.value?.fits ?? [])
const topRatedFits = computed(() => topRatedData.value?.fits ?? [])

// ---------- helpers ----------

const timeAgo = (iso: string | null): string => {
    if (!iso) return ''
    const diff = Date.now() - new Date(iso).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 60) return `${mins}m ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    if (days < 30) return `${days}d ago`
    const months = Math.floor(days / 30)
    if (months < 12) return `${months}mo ago`
    return `${Math.floor(days / 365)}y ago`
}

const shipRenderUrl = (typeId: number) =>
    `/images/types/${typeId}/render?size=256`

const allianceLogoUrl = (allianceId: number) =>
    `/images/alliances/${allianceId}/logo?size=64`

const characterPortraitUrl = (characterId: number) =>
    `/images/characters/${characterId}/portrait?size=32`

const formatCount = (n: number | undefined | null): string => {
    if (n === null || n === undefined) return '0'
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
    return n.toLocaleString('en-US')
}


async function loadFlavor(family: FlavorFamily) {
    const url = await killmailFitToEditorUrl({
        shipTypeId: family.ship_type_id,
        modules: family.modules,
        drones: family.drones,
        name: `${family.ship_name ?? 'Community'} Fit`,
        description: `Flavor of the week — ${family.total_uses} recorded uses in the last ${flavorsData.value?.window_days ?? 30} days.`,
    })
    await navigateTo(url)
}

async function loadDoctrine(d: AllianceDoctrine) {
    const url = await killmailFitToEditorUrl({
        shipTypeId: d.ship_type_id,
        modules: d.modules,
        drones: d.drones,
        name: `${d.alliance_name ?? 'Alliance'} ${d.ship_name ?? ''}`.trim(),
        description: `${d.alliance_name ?? 'Alliance'} doctrine — ${d.doctrine_uses} losses (${d.doctrine_share}% of their activity) in the last ${doctrinesData.value?.window_days ?? 30} days.`,
    })
    await navigateTo(url)
}
</script>

<template>
    <div>
        <PageHeader class="mb-6" title="Browse, build, and share EVE Online fits" eyebrow="Ship Fittings" icon="lucide:wrench"
            description="Explore the most-flown fits from the last 30 days of killmails, see what top alliances are bringing to battle, and use our in-browser fitting tool to design and share your own — all with real dogma engine calculations.">
            <template #actions>
                <div class="flex flex-wrap items-center gap-2">
                    <NuxtLink
                        to="/fits/create"
                        class="inline-flex items-center gap-2 px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md bg-blue-500/20 text-blue-400 border border-blue-500/30 hover:bg-blue-500/30 transition-colors"
                    >
                        <Icon name="lucide:plus" class="w-3.5 h-3.5" />
                        Create New Fit
                    </NuxtLink>
                    <NuxtLink
                        to="/fits/search"
                        class="inline-flex items-center gap-2 px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md bg-white/[0.04] text-gray-200 border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 hover:border-blue-500/30 transition-colors"
                    >
                        <Icon name="lucide:search" class="w-3.5 h-3.5" />
                        Search Fits
                    </NuxtLink>
                </div>
            </template>
        </PageHeader>

        <!-- ============================ Quick Stats ============================ -->
        <section class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1">
                    Killmail Fits
                </div>
                <div class="text-2xl font-bold text-white tabular-nums">
                    {{ formatCount(stats?.fittings_known) }}
                </div>
                <div class="text-fine text-gray-500">unique families</div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1">
                    Killmails
                </div>
                <div class="text-2xl font-bold text-white tabular-nums">
                    {{ formatCount(stats?.killmails_analyzed) }}
                </div>
                <div class="text-fine text-gray-500">analyzed for fits</div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1">
                    Community Fits
                </div>
                <div class="text-2xl font-bold text-white tabular-nums">
                    {{ formatCount(stats?.community_fits) }}
                </div>
                <div class="text-fine text-gray-500">public user-saved</div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1">
                    Ratings
                </div>
                <div class="text-2xl font-bold text-white tabular-nums">
                    {{ formatCount(stats?.ratings_cast) }}
                </div>
                <div class="text-fine text-gray-500">cast by pilots</div>
            </div>
        </section>

        <!-- ============================ Flavors of the Week ============================ -->
        <section class="mb-6">
            <div class="flex items-end justify-between mb-3">
                <div>
                    <div
                        class="text-fine font-bold uppercase tracking-[0.15em] text-amber-400/80 mb-1"
                    >
                        Last 30 Days
                    </div>
                    <h2 class="text-lg font-bold text-white">Flavors of the Week</h2>
                </div>
                <div class="text-fine text-gray-500 hidden md:block">
                    Click any fit to load it into the editor
                </div>
            </div>

            <div
                v-if="flavorsPending && flavors.length === 0"
                class="glass-panel flex items-center justify-center py-20"
            >
                <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
            </div>

            <div
                v-else-if="flavors.length === 0"
                class="glass-panel py-12 text-center text-sm text-gray-500"
            >
                No recent fitting activity
            </div>

            <div
                v-else
                class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3"
            >
                <button
                    v-for="family in flavors"
                    :key="`${family.ship_type_id}-${family.family_hash}`"
                    type="button"
                    class="group rounded-lg bg-white/[0.04] border border-white/[0.08] overflow-hidden hover:bg-amber-500/[0.06] hover:border-amber-500/30 transition-colors text-left"
                    @click="loadFlavor(family)"
                >
                    <div class="aspect-square bg-black/40 relative">
                        <img
                            :src="shipRenderUrl(family.ship_type_id)"
                            :alt="family.ship_name ?? ''"
                            class="w-full h-full object-cover"
                            loading="lazy"
                        />
                        <div
                            class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/85 to-transparent p-2"
                        >
                            <div class="text-fine font-bold text-amber-300 tabular-nums">
                                {{ family.total_uses.toLocaleString('en-US') }} uses
                            </div>
                        </div>
                    </div>
                    <div class="p-2">
                        <div
                            class="text-xs text-gray-200 font-medium truncate group-hover:text-amber-300 transition-colors"
                        >
                            {{ family.ship_name ?? `Type ${family.ship_type_id}` }}
                        </div>
                        <div class="text-fine text-gray-500 tabular-nums mt-0.5">
                            {{ formatIsk((family.fit_cost ?? 0) + (family.hull_cost ?? 0)) }}
                            ISK
                        </div>
                    </div>
                </button>
            </div>
        </section>

        <!-- ============================ Top Alliance Doctrines ============================ -->
        <section class="mb-6">
            <div class="flex items-end justify-between mb-3">
                <div>
                    <div
                        class="text-fine font-bold uppercase tracking-[0.15em] text-purple-400/80 mb-1"
                    >
                        Last 30 Days
                    </div>
                    <h2 class="text-lg font-bold text-white">Popular Alliance Doctrines</h2>
                </div>
                <div class="text-fine text-gray-500 hidden md:block">
                    What the top alliances are flying
                </div>
            </div>

            <div
                v-if="doctrinesPending && doctrines.length === 0"
                class="glass-panel flex items-center justify-center py-16"
            >
                <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
            </div>

            <div
                v-else-if="doctrines.length === 0"
                class="glass-panel py-12 text-center text-sm text-gray-500"
            >
                No alliance activity in the window
            </div>

            <div
                v-else
                class="glass-panel overflow-hidden"
            >
                <button
                    v-for="d in doctrines"
                    :key="d.alliance_id"
                    type="button"
                    class="w-full flex items-center gap-3 px-4 py-3 border-b border-white/[0.04] last:border-b-0 hover:bg-purple-500/[0.05] transition-colors text-left"
                    @click="loadDoctrine(d)"
                >
                    <img
                        :src="allianceLogoUrl(d.alliance_id)"
                        :alt="d.alliance_name ?? ''"
                        class="w-10 h-10 rounded flex-shrink-0 bg-black/30"
                        loading="lazy"
                    />
                    <div class="min-w-0 flex-1 max-w-[200px]">
                        <div class="text-sm text-gray-200 font-medium truncate">
                            {{ d.alliance_name ?? `Alliance ${d.alliance_id}` }}
                        </div>
                        <div class="text-fine text-gray-500 tabular-nums">
                            {{ d.total_losses.toLocaleString('en-US') }} losses
                        </div>
                    </div>
                    <div class="w-px h-8 bg-white/[0.06] flex-shrink-0 hidden sm:block"></div>
                    <img
                        :src="`/images/types/${d.ship_type_id}/render?size=64`"
                        :alt="d.ship_name ?? ''"
                        class="w-10 h-10 rounded flex-shrink-0 bg-black/30 hidden sm:block"
                        loading="lazy"
                    />
                    <div class="min-w-0 flex-1 hidden sm:block">
                        <div class="text-sm text-purple-300 font-medium truncate">
                            {{ d.ship_name ?? `Type ${d.ship_type_id}` }}
                        </div>
                        <div class="text-fine text-gray-500 tabular-nums">
                            {{ d.modules.length }} modules ·
                            {{ formatIsk((d.fit_cost ?? 0) + (d.hull_cost ?? 0)) }} ISK
                        </div>
                    </div>
                    <div class="flex flex-col items-end flex-shrink-0">
                        <div class="text-sm font-bold text-purple-400 tabular-nums">
                            {{ d.doctrine_share }}%
                        </div>
                        <div class="text-fine text-gray-500">of losses</div>
                    </div>
                </button>
            </div>
        </section>

        <!-- ============================ Popular Hulls + Community Fits ============================ -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            <!-- Popular Hulls -->
            <section>
                <div class="flex items-end justify-between mb-3">
                    <div>
                        <div
                            class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1"
                        >
                            Last 30 Days
                        </div>
                        <h2 class="text-lg font-bold text-white">Popular Hulls</h2>
                    </div>
                    <div class="text-fine text-gray-500">Click for fit variants</div>
                </div>

                <div
                    v-if="popularPending && popularShips.length === 0"
                    class="glass-panel flex items-center justify-center py-20"
                >
                    <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                </div>

                <div
                    v-else-if="popularShips.length === 0"
                    class="glass-panel py-12 text-center text-sm text-gray-500"
                >
                    No recent fitting activity
                </div>

                <div
                    v-else
                    class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-3 gap-3"
                >
                    <NuxtLink
                        v-for="ship in popularShips"
                        :key="ship.ship_type_id"
                        :to="`/item/${ship.ship_type_id}/fittings`"
                        class="group rounded-lg bg-white/[0.04] border border-white/[0.08] overflow-hidden hover:bg-blue-500/[0.06] hover:border-blue-500/30 transition-colors block"
                    >
                        <div class="aspect-square bg-black/40 relative">
                            <img
                                :src="shipRenderUrl(ship.ship_type_id)"
                                :alt="ship.ship_name ?? ''"
                                class="w-full h-full object-cover"
                                loading="lazy"
                            />
                            <div
                                class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent p-2"
                            >
                                <div class="text-fine font-bold text-blue-400 tabular-nums">
                                    {{ ship.fit_count }} fit{{ ship.fit_count === 1 ? '' : 's' }}
                                </div>
                            </div>
                        </div>
                        <div class="p-2.5">
                            <div
                                class="text-xs text-gray-200 font-medium truncate group-hover:text-blue-400 transition-colors"
                            >
                                {{ ship.ship_name ?? `Type ${ship.ship_type_id}` }}
                            </div>
                            <div class="text-fine text-gray-500 tabular-nums mt-0.5">
                                {{ ship.total_uses.toLocaleString('en-US') }} killmails
                            </div>
                        </div>
                    </NuxtLink>
                </div>
            </section>

            <!-- Latest Community Fits -->
            <section>
                <div class="flex items-end justify-between mb-3">
                    <div>
                        <div
                            class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1"
                        >
                            Community
                        </div>
                        <h2 class="text-lg font-bold text-white">Latest Shared Fits</h2>
                    </div>
                    <div class="text-fine text-gray-500">Public user-saved</div>
                </div>

                <div
                    v-if="communityPending && communityFits.length === 0"
                    class="glass-panel flex items-center justify-center py-16"
                >
                    <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                </div>

                <div
                    v-else-if="communityFits.length === 0"
                    class="glass-panel p-8 text-center"
                >
                    <Icon name="lucide:ship" class="w-10 h-10 text-gray-600 mx-auto mb-3" />
                    <div class="text-sm text-gray-300 mb-1">Be the first to share a fit</div>
                    <div class="text-fine text-gray-500 mb-4">
                        Build a fit in the editor, save it as public, and it'll show up here.
                    </div>
                    <NuxtLink
                        to="/fits/create"
                        class="inline-flex items-center gap-2 px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md bg-blue-500/20 text-blue-400 border border-blue-500/30 hover:bg-blue-500/30 transition-colors"
                    >
                        <Icon name="lucide:plus" class="w-3.5 h-3.5" />
                        Create New Fit
                    </NuxtLink>
                </div>

                <div
                    v-else
                    class="glass-panel overflow-hidden"
                >
                    <NuxtLink
                        v-for="fit in communityFits"
                        :key="fit.fit_id"
                        :to="`/fit/${fit.fit_id}`"
                        class="flex items-center gap-3 px-4 py-3 border-b border-white/[0.04] last:border-b-0 hover:bg-blue-500/[0.04] transition-colors"
                    >
                        <img
                            :src="shipRenderUrl(fit.ship_type_id)"
                            :alt="fit.ship_name ?? ''"
                            class="w-12 h-12 rounded-md flex-shrink-0 bg-black/40"
                            loading="lazy"
                        />
                        <div class="min-w-0 flex-1">
                            <div class="text-sm text-gray-200 font-medium truncate">
                                {{ fit.name }}
                            </div>
                            <div
                                class="text-fine text-gray-500 flex items-center gap-1.5 mt-0.5"
                            >
                                <span class="text-blue-400/80">{{
                                    fit.ship_name ?? `Type ${fit.ship_type_id}`
                                }}</span>
                                <span class="text-gray-600">·</span>
                                <span class="tabular-nums">{{ fit.module_count }} modules</span>
                            </div>
                            <div class="mt-1">
                                <FitRating
                                    :fit-id="fit.fit_id"
                                    :rating-avg="fit.rating_avg"
                                    :rating-count="fit.rating_count"
                                    :viewer-rating="null"
                                    size="sm"
                                />
                            </div>
                        </div>
                        <div class="flex items-center gap-2 flex-shrink-0">
                            <img
                                :src="characterPortraitUrl(fit.owner_character_id)"
                                :alt="fit.owner_name ?? ''"
                                class="w-6 h-6 rounded-full"
                                loading="lazy"
                            />
                            <div class="text-right">
                                <div class="text-xs text-gray-300 truncate max-w-[120px]">
                                    {{ fit.owner_name ?? 'Unknown' }}
                                </div>
                                <div class="text-fine text-gray-500">
                                    {{ timeAgo(fit.created_at) }}
                                </div>
                            </div>
                        </div>
                    </NuxtLink>
                </div>
            </section>
        </div>

        <!-- ============================ Top Rated (only when non-empty) ============================ -->
        <section v-if="topRatedFits.length > 0" class="mb-6">
            <div class="flex items-end justify-between mb-3">
                <div>
                    <div
                        class="text-fine font-bold uppercase tracking-[0.15em] text-amber-400/80 mb-1"
                    >
                        Community Picks
                    </div>
                    <h2 class="text-lg font-bold text-white">Top Rated Fits</h2>
                </div>
                <div class="text-fine text-gray-500">Sorted by average rating</div>
            </div>

            <div class="glass-panel overflow-hidden">
                <NuxtLink
                    v-for="fit in topRatedFits"
                    :key="fit.fit_id"
                    :to="`/fit/${fit.fit_id}`"
                    class="flex items-center gap-3 px-4 py-3 border-b border-white/[0.04] last:border-b-0 hover:bg-amber-500/[0.04] transition-colors"
                >
                    <img
                        :src="shipRenderUrl(fit.ship_type_id)"
                        :alt="fit.ship_name ?? ''"
                        class="w-12 h-12 rounded-md flex-shrink-0 bg-black/40"
                        loading="lazy"
                    />
                    <div class="min-w-0 flex-1">
                        <div class="text-sm text-gray-200 font-medium truncate">
                            {{ fit.name }}
                        </div>
                        <div
                            class="text-fine text-gray-500 flex items-center gap-1.5 mt-0.5"
                        >
                            <span class="text-amber-300/80">{{
                                fit.ship_name ?? `Type ${fit.ship_type_id}`
                            }}</span>
                            <span class="text-gray-600">·</span>
                            <span class="tabular-nums">{{ fit.module_count }} modules</span>
                        </div>
                        <div class="mt-1">
                            <FitRating
                                :fit-id="fit.fit_id"
                                :rating-avg="fit.rating_avg"
                                :rating-count="fit.rating_count"
                                :viewer-rating="null"
                                size="sm"
                            />
                        </div>
                    </div>
                    <div class="flex items-center gap-2 flex-shrink-0">
                        <img
                            :src="characterPortraitUrl(fit.owner_character_id)"
                            :alt="fit.owner_name ?? ''"
                            class="w-6 h-6 rounded-full"
                            loading="lazy"
                        />
                        <div class="text-right">
                            <div class="text-xs text-gray-300 truncate max-w-[120px]">
                                {{ fit.owner_name ?? 'Unknown' }}
                            </div>
                            <div class="text-fine text-gray-500">{{ timeAgo(fit.created_at) }}</div>
                        </div>
                    </div>
                </NuxtLink>
            </div>
        </section>
    </div>
</template>
