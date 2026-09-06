<script setup lang="ts">
/**
 * /fits — ship fitting landing page / explorer.
 *
 * Sections (top to bottom):
 *   1. Hero          — title, tagline, "Create New Fit" CTA
 *   2. Quick Stats   — counter row across the top (osmium homage)
 *   3. Flavors of the Week — weekly hull rankings by kills, final blows,
 *                       or losses, paired with a representative observed fit
 *   4. Top Alliance Doctrines — for the most-active alliances, their
 *                       signature fit; click to load it
 *   5. Popular Hulls — top 12 ships by recent kill participation;
 *                       click drills into /item/:id/fittings
 *   6. Latest + Top Rated Community Fits — two-column grid with the
 *                       newest public user fits and the highest-rated
 *                       ones, both linking to /fit/:id
 *
 * No WASM on this page — pure consumption. The dogma engine only
 * loads on pages that actually calculate fits.
 */

import { killmailFitToEditorUrl } from "~/composables/fit/killmailToFit";
import FitRating from "~/components/fit/FitRating.vue";

useHead({ title: "Ship Fittings" });
useSeoMeta({
  description:
    "Browse popular ship fittings from EVE Online killmails, share your own fits, and build new ones with the in-browser fitting tool.",
  ogTitle: "Ship Fittings — EVE-KILL",
  ogDescription:
    "Popular fits from the last 30 days, alliance signature doctrines, community-shared builds, and a full in-browser fitting tool.",
});

// ---------- types ----------

interface QuickStats {
  fittings_known: number;
  killmails_analyzed: number;
  community_fits: number;
  ratings_cast: number;
}

interface FittingModule {
  slot_group: number;
  ordinal: number;
  type_id: number;
  name: string | null;
  charge_type_id: number | null;
  charge_name: string | null;
}

interface FittingDrone {
  type_id: number;
  name: string | null;
  quantity: number;
}

interface FlavorFamily {
  family_hash: string;
  ship_type_id: number;
  ship_name: string | null;
  canonical_fit_hash: string;
  total_uses: number;
  ranking_count: number;
  canonical_uses: number;
  variant_count: number;
  last_used: string;
  fit_cost: number;
  hull_cost: number | null;
  modules: FittingModule[];
  drones: FittingDrone[];
}

type FlavorMode = "kills" | "final_blows" | "losses";

const flavorModes: { id: FlavorMode; label: string; icon: string }[] = [
  { id: "kills", label: "Kills", icon: "lucide:swords" },
  { id: "final_blows", label: "Final Blows", icon: "lucide:crosshair" },
  { id: "losses", label: "Losses", icon: "lucide:skull" },
];
const route = useRoute();
const replaceQueryWithoutNavigation = (name: string, value?: string) => {
  if (!import.meta.client) return;
  const url = new URL(window.location.href);
  if (value) url.searchParams.set(name, value);
  else url.searchParams.delete(name);
  window.history.replaceState(window.history.state, "", url);
};
const requestedFlavorMode =
  typeof route.query.flavor === "string" ? route.query.flavor : "";
const flavorMode = ref<FlavorMode>(
  flavorModes.some((mode) => mode.id === requestedFlavorMode)
    ? (requestedFlavorMode as FlavorMode)
    : "kills",
);
watch(flavorMode, (mode) => {
  replaceQueryWithoutNavigation("flavor", mode === "kills" ? undefined : mode);
});

type DoctrineEntityType = "alliance" | "corporation";

const doctrineEntityTypes: {
  id: DoctrineEntityType;
  label: string;
  icon: string;
}[] = [
  { id: "alliance", label: "Alliances", icon: "lucide:shield" },
  { id: "corporation", label: "Corporations", icon: "lucide:building-2" },
];
const requestedDoctrineEntity =
  typeof route.query.doctrines === "string" ? route.query.doctrines : "";
const doctrineEntityType = ref<DoctrineEntityType>(
  doctrineEntityTypes.some((mode) => mode.id === requestedDoctrineEntity)
    ? (requestedDoctrineEntity as DoctrineEntityType)
    : "alliance",
);
watch(doctrineEntityType, (mode) => {
  replaceQueryWithoutNavigation(
    "doctrines",
    mode === "alliance" ? undefined : mode,
  );
});

interface EntityDoctrine {
  entity_id: number;
  entity_name: string | null;
  total_losses: number;
  family_hash: string;
  ship_type_id: number;
  ship_name: string | null;
  canonical_fit_hash: string;
  doctrine_uses: number;
  doctrine_share: number;
  last_used: string;
  fit_cost: number;
  hull_cost: number | null;
  modules: FittingModule[];
  drones: FittingDrone[];
}

interface PopularShip {
  ship_type_id: number;
  total_uses: number;
  fit_count: number;
  last_used: string;
  ship_name: string | null;
  group_id: number | null;
}

interface CommunityFit {
  fit_id: string;
  name: string;
  description: string | null;
  ship_type_id: number;
  ship_name: string | null;
  owner_character_id: number;
  owner_name: string | null;
  rating_avg: number | null;
  rating_count: number;
  created_at: string;
  updated_at: string;
  module_count: number;
}

// ---------- parallel fetches with hydration payload reuse ----------

const statsRequest = useApiFetch<QuickStats>(
  "/api/fits/quick-stats",
  { lazy: true },
);
const flavorsRequest = useApiFetch<{
  window_days: number;
  ranking_mode: FlavorMode;
  families: FlavorFamily[];
}>("/api/fits/flavors-of-the-week", {
  lazy: true,
  query: computed(() => ({ mode: flavorMode.value })),
});
const doctrinesRequest = useApiFetch<{
  window_days: number;
  entity_type: DoctrineEntityType;
  doctrines: EntityDoctrine[];
}>("/api/fits/top-alliance-doctrines", {
  lazy: true,
  query: computed(() => ({ entity_type: doctrineEntityType.value })),
});
const popularRequest = useApiFetch<{
  window_days: number;
  ships: PopularShip[];
}>("/api/fits/popular-ships", {
  lazy: true,
  query: { mode: "kills" },
});
const communityRequest = useApiFetch<{
  fits: CommunityFit[];
}>("/api/fits/community-latest", { lazy: true });
const topRatedRequest = useApiFetch<{
  fits: CommunityFit[];
}>("/api/fits/top-rated", { lazy: true });

// Start independent SSR requests together, then render from the shared payload.
await Promise.all([statsRequest, flavorsRequest, doctrinesRequest, popularRequest, communityRequest, topRatedRequest]);
const { data: statsData } = statsRequest;
const { data: flavorsData, pending: flavorsPending } = flavorsRequest;
const { data: doctrinesData, pending: doctrinesPending } = doctrinesRequest;
const { data: popularData, pending: popularPending } = popularRequest;
const { data: communityData, pending: communityPending } = communityRequest;
const { data: topRatedData } = topRatedRequest;

const stats = computed(() => statsData.value);
const flavors = computed(() => flavorsData.value?.families ?? []);
const doctrines = computed(() => doctrinesData.value?.doctrines ?? []);
const popularShips = computed(() => popularData.value?.ships ?? []);
const communityFits = computed(() => communityData.value?.fits ?? []);
const topRatedFits = computed(() => topRatedData.value?.fits ?? []);
const flavorMetricLabel = computed(() =>
  flavorMode.value === "final_blows" ? "final blows" : flavorMode.value,
);
const quickStats = computed(() => [
  {
    label: "Fit families",
    value: formatCount(stats.value?.fittings_known),
    detail: "from real losses",
    icon: "lucide:git-branch",
  },
  {
    label: "Killmails",
    value: formatCount(stats.value?.killmails_analyzed),
    detail: "analyzed for fits",
    icon: "lucide:scan-search",
  },
  {
    label: "Shared fits",
    value: formatCount(stats.value?.community_fits),
    detail: "published by pilots",
    icon: "lucide:users",
  },
  {
    label: "Ratings",
    value: formatCount(stats.value?.ratings_cast),
    detail: "from the community",
    icon: "lucide:star",
  },
]);

// ---------- helpers ----------

const timeAgo = (iso: string | null): string => {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(days / 365)}y ago`;
};

const shipRenderUrl = (typeId: number) =>
  `/images/types/${typeId}/render?size=256`;

const allianceLogoUrl = (allianceId: number) =>
  `/images/alliances/${allianceId}/logo?size=64`;

const doctrineEntityImageUrl = (entity: EntityDoctrine) =>
  doctrineEntityType.value === "corporation"
    ? `/images/corporations/${entity.entity_id}/logo?size=64`
    : allianceLogoUrl(entity.entity_id);

const characterPortraitUrl = (characterId: number) =>
  `/images/characters/${characterId}/portrait?size=32`;

const formatCount = (n: number | undefined | null): string => {
  if (n === null || n === undefined) return "0";
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toLocaleString("en-US");
};

async function loadFlavor(family: FlavorFamily) {
  const url = await killmailFitToEditorUrl({
    shipTypeId: family.ship_type_id,
    modules: family.modules,
    drones: family.drones,
    name: `${family.ship_name ?? "Community"} Fit`,
    description: `Flavor of the week — ${family.ranking_count} ${flavorMetricLabel.value} in the last ${flavorsData.value?.window_days ?? 7} days. Representative fit observed on ${family.total_uses} losses.`,
  });
  await navigateTo(url);
}

async function loadDoctrine(d: EntityDoctrine) {
  const url = await killmailFitToEditorUrl({
    shipTypeId: d.ship_type_id,
    modules: d.modules,
    drones: d.drones,
    name: `${d.entity_name ?? "Doctrine"} ${d.ship_name ?? ""}`.trim(),
    description: `${d.entity_name ?? "Entity"} doctrine — ${d.doctrine_uses} losses (${d.doctrine_share}% of their activity) in the last ${doctrinesData.value?.window_days ?? 30} days.`,
  });
  await navigateTo(url);
}
</script>

<template>
  <div>
    <PageHeader
      class="mb-6"
      title="Ship Fittings"
      eyebrow="Build from real combat data"
      icon="lucide:wrench"
      image-src="/images/types/24694/render?size=1024"
      image-mode="background"
      description="Discover fittings recovered from real losses, explore alliance doctrines, or build and share your own with live dogma calculations."
    >
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
      <template #meta>
        <div class="grid grid-cols-2 gap-x-5 gap-y-3 sm:grid-cols-4">
          <div
            v-for="stat in quickStats"
            :key="stat.label"
            class="flex min-w-0 items-center gap-2.5"
          >
            <span
              class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-white/[0.04] text-blue-400/70"
            >
              <Icon :name="stat.icon" class="h-4 w-4" />
            </span>
            <div class="min-w-0">
              <div class="flex items-baseline gap-1.5">
                <span class="text-lg font-bold tabular-nums text-white">{{
                  stat.value
                }}</span>
                <span
                  class="truncate text-fine font-bold uppercase tracking-wider text-gray-500"
                  >{{ stat.label }}</span
                >
              </div>
              <div class="truncate text-fine text-gray-600">
                {{ stat.detail }}
              </div>
            </div>
          </div>
        </div>
      </template>
    </PageHeader>

    <!-- ============================ Flavors of the Week ============================ -->
    <section class="mb-6">
      <div
        class="mb-3 flex items-end justify-between border-b border-white/[0.06] pb-3"
      >
        <div class="flex items-center gap-3">
          <span
            class="flex h-8 w-8 items-center justify-center rounded-lg border border-amber-400/15 bg-amber-400/[0.06] text-xs font-bold text-amber-300"
            >01</span
          >
          <div>
            <div
              class="text-fine font-bold uppercase tracking-[0.15em] text-amber-400/80 mb-1"
            >
              Last 7 Days
            </div>
            <h2 class="text-lg font-bold text-white">Flavors of the Week</h2>
          </div>
        </div>
        <div
          class="flex max-w-full gap-1 overflow-x-auto rounded-lg border border-white/[0.08] bg-black/25 p-1"
        >
          <button
            v-for="mode in flavorModes"
            :key="mode.id"
            type="button"
            class="inline-flex shrink-0 items-center gap-1.5 rounded-md px-2.5 py-1.5 text-fine font-bold uppercase tracking-wider transition-colors"
            :class="
              flavorMode === mode.id
                ? 'bg-amber-400/10 text-amber-300'
                : 'text-gray-500 hover:bg-white/[0.04] hover:text-gray-300'
            "
            @click="flavorMode = mode.id"
          >
            <Icon :name="mode.icon" class="h-3.5 w-3.5" />
            {{ mode.label }}
          </button>
        </div>
      </div>

      <div
        v-if="flavorsPending && flavors.length === 0"
        class="glass-panel flex items-center justify-center py-20"
      >
        <Icon
          name="lucide:loader-2"
          class="w-5 h-5 text-gray-500 animate-spin"
        />
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
        <div
          v-for="family in flavors"
          :key="`${family.ship_type_id}-${family.family_hash}`"
          class="group relative overflow-hidden rounded-xl border border-white/[0.08] bg-black/25 transition-all duration-200 hover:-translate-y-0.5 hover:border-amber-500/30 hover:bg-amber-500/[0.05] hover:shadow-lg hover:shadow-black/20"
        >
          <button
            type="button"
            class="block w-full text-left"
            @click="loadFlavor(family)"
          >
            <div class="aspect-square bg-black/40 relative">
              <img
                :src="shipRenderUrl(family.ship_type_id)"
                :alt="family.ship_name ?? ''"
                class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.04]"
                loading="lazy"
              />
              <div
                class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/85 to-transparent p-2"
              >
                <div class="text-fine font-bold text-amber-300 tabular-nums">
                  {{ family.ranking_count.toLocaleString("en-US") }}
                  {{ flavorMetricLabel }}
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
                {{
                  formatIsk((family.fit_cost ?? 0) + (family.hull_cost ?? 0))
                }}
                ISK
              </div>
            </div>
          </button>
          <NuxtLink
            :to="`/item/${family.ship_type_id}/fittings`"
            class="absolute right-2 top-2 z-10 rounded-md border border-white/10 bg-black/65 px-1.5 py-1 text-fine font-semibold tabular-nums text-gray-300 backdrop-blur-sm transition-colors hover:border-amber-400/35 hover:bg-amber-400/15 hover:text-amber-200"
            :aria-label="`View ${family.variant_count} ${family.ship_name ?? 'ship'} fitting variants`"
          >
            {{ family.variant_count.toLocaleString("en-US") }} variants
          </NuxtLink>
        </div>
      </div>
    </section>

    <!-- ============================ Popular Doctrines ============================ -->
    <section class="mb-6">
      <div
        class="mb-3 flex items-end justify-between border-b border-white/[0.06] pb-3"
      >
        <div class="flex items-center gap-3">
          <span
            class="flex h-8 w-8 items-center justify-center rounded-lg border border-purple-400/15 bg-purple-400/[0.06] text-xs font-bold text-purple-300"
            >02</span
          >
          <div>
            <div
              class="text-fine font-bold uppercase tracking-[0.15em] text-purple-400/80 mb-1"
            >
              Last 30 Days
            </div>
            <h2 class="text-lg font-bold text-white">Popular Doctrines</h2>
          </div>
        </div>
        <div
          class="flex items-center rounded-lg border border-white/[0.07] bg-black/20 p-1"
        >
          <button
            v-for="mode in doctrineEntityTypes"
            :key="mode.id"
            type="button"
            class="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-semibold transition-all"
            :class="
              doctrineEntityType === mode.id
                ? 'bg-purple-400/15 text-purple-200 shadow-sm'
                : 'text-gray-500 hover:bg-white/[0.04] hover:text-gray-300'
            "
            @click="doctrineEntityType = mode.id"
          >
            <Icon :name="mode.icon" class="h-3.5 w-3.5" />
            {{ mode.label }}
          </button>
        </div>
      </div>

      <div
        v-if="doctrinesPending && doctrines.length === 0"
        class="glass-panel flex items-center justify-center py-16"
      >
        <Icon
          name="lucide:loader-2"
          class="w-5 h-5 text-gray-500 animate-spin"
        />
      </div>

      <div
        v-else-if="doctrines.length === 0"
        class="glass-panel py-12 text-center text-sm text-gray-500"
      >
        No {{ doctrineEntityType }} activity in the window
      </div>

      <div v-else class="glass-panel overflow-hidden">
        <div
          class="hidden grid-cols-[2.5rem_minmax(0,1fr)_minmax(0,1fr)_7rem_5rem] items-center gap-3 border-b border-white/[0.06] bg-white/[0.015] px-4 py-2 text-fine font-bold uppercase tracking-[0.12em] text-gray-600 sm:grid"
        >
          <span>Rank</span>
          <span>{{ doctrineEntityType }}</span>
          <span>Signature fit</span>
          <span class="text-right">Observed</span>
          <span class="text-right">Share</span>
        </div>
        <button
          v-for="(d, index) in doctrines"
          :key="d.entity_id"
          type="button"
          class="group grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-3 border-b border-white/[0.04] px-4 py-3 text-left transition-all last:border-b-0 hover:bg-purple-500/[0.055] sm:grid-cols-[2.5rem_minmax(0,1fr)_minmax(0,1fr)_7rem_5rem]"
          @click="loadDoctrine(d)"
        >
          <div
            class="hidden h-7 w-7 items-center justify-center rounded-md border border-white/[0.06] bg-black/20 text-xs font-bold tabular-nums text-gray-600 transition-colors group-hover:border-purple-400/20 group-hover:text-purple-300 sm:flex"
          >
            {{ String(index + 1).padStart(2, "0") }}
          </div>
          <div class="flex min-w-0 items-center gap-3">
            <img
              :src="doctrineEntityImageUrl(d)"
              :alt="d.entity_name ?? ''"
              class="h-10 w-10 flex-shrink-0 rounded bg-black/30 ring-1 ring-white/[0.06]"
              loading="lazy"
            />
            <div class="min-w-0">
              <div class="truncate text-sm font-medium text-gray-200">
                {{
                  d.entity_name ??
                  `${doctrineEntityType === "alliance" ? "Alliance" : "Corporation"} ${d.entity_id}`
                }}
              </div>
              <div class="text-fine tabular-nums text-gray-500 sm:hidden">
                {{ d.total_losses.toLocaleString("en-US") }} total losses
              </div>
            </div>
          </div>
          <div class="hidden min-w-0 items-center gap-3 sm:flex">
            <img
              :src="`/images/types/${d.ship_type_id}/render?size=64`"
              :alt="d.ship_name ?? ''"
              class="h-10 w-10 flex-shrink-0 rounded bg-black/30 transition-transform duration-200 group-hover:scale-105"
              loading="lazy"
            />
            <div class="min-w-0">
              <div class="truncate text-sm font-medium text-purple-300">
                {{ d.ship_name ?? `Type ${d.ship_type_id}` }}
              </div>
              <div class="text-fine truncate tabular-nums text-gray-500">
                {{ d.modules.length }} modules ·
                {{ formatIsk((d.fit_cost ?? 0) + (d.hull_cost ?? 0)) }} ISK
              </div>
            </div>
          </div>
          <div
            class="hidden text-right text-sm font-semibold tabular-nums text-gray-300 sm:block"
          >
            {{ d.doctrine_uses.toLocaleString("en-US") }}
            <div class="text-fine font-normal text-gray-600">
              of {{ formatCount(d.total_losses) }}
            </div>
          </div>
          <div class="flex flex-shrink-0 flex-col items-end">
            <div class="text-sm font-bold tabular-nums text-purple-400">
              {{ d.doctrine_share }}%
            </div>
            <div class="text-fine text-gray-600 sm:hidden">of losses</div>
          </div>
        </button>
      </div>
    </section>

    <!-- ============================ Popular Hulls + Community Fits ============================ -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <!-- Popular Hulls -->
      <section>
        <div
          class="mb-3 flex items-end justify-between border-b border-white/[0.06] pb-3"
        >
          <div class="flex items-center gap-3">
            <span
              class="flex h-8 w-8 items-center justify-center rounded-lg border border-blue-400/15 bg-blue-400/[0.06] text-xs font-bold text-blue-300"
              >03</span
            >
            <div>
              <div
                class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1"
              >
                Last 30 Days
              </div>
              <h2 class="text-lg font-bold text-white">Popular Hulls</h2>
            </div>
          </div>
          <div class="text-fine text-gray-500">Click for fit variants</div>
        </div>

        <div
          v-if="popularPending && popularShips.length === 0"
          class="glass-panel flex items-center justify-center py-20"
        >
          <Icon
            name="lucide:loader-2"
            class="w-5 h-5 text-gray-500 animate-spin"
          />
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
            class="group block overflow-hidden rounded-xl border border-white/[0.08] bg-black/25 transition-all duration-200 hover:-translate-y-0.5 hover:border-blue-500/30 hover:bg-blue-500/[0.05] hover:shadow-lg hover:shadow-black/20"
          >
            <div class="aspect-square bg-black/40 relative">
              <img
                :src="shipRenderUrl(ship.ship_type_id)"
                :alt="ship.ship_name ?? ''"
                class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.04]"
                loading="lazy"
              />
              <div
                class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/85 to-transparent p-2"
              >
                <div class="text-fine font-bold text-blue-300 tabular-nums">
                  {{ ship.total_uses.toLocaleString("en-US") }} kills
                </div>
              </div>
              <div
                class="absolute right-2 top-2 rounded-md border border-white/10 bg-black/60 px-1.5 py-1 text-fine font-semibold text-gray-300 backdrop-blur-sm"
              >
                {{ ship.fit_count.toLocaleString("en-US") }} families
              </div>
            </div>
            <div class="p-2">
              <div
                class="text-xs text-gray-200 font-medium truncate group-hover:text-blue-400 transition-colors"
              >
                {{ ship.ship_name ?? `Type ${ship.ship_type_id}` }}
              </div>
              <div class="text-fine text-gray-500 tabular-nums mt-0.5">
                View observed fitting families
              </div>
            </div>
          </NuxtLink>
        </div>
      </section>

      <!-- Latest Community Fits -->
      <section>
        <div
          class="mb-3 flex items-end justify-between border-b border-white/[0.06] pb-3"
        >
          <div class="flex items-center gap-3">
            <span
              class="flex h-8 w-8 items-center justify-center rounded-lg border border-cyan-400/15 bg-cyan-400/[0.06] text-xs font-bold text-cyan-300"
              >04</span
            >
            <div>
              <div
                class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1"
              >
                Community
              </div>
              <h2 class="text-lg font-bold text-white">Latest Shared Fits</h2>
            </div>
          </div>
          <div class="text-fine text-gray-500">Public user-saved</div>
        </div>

        <div
          v-if="communityPending && communityFits.length === 0"
          class="glass-panel flex items-center justify-center py-16"
        >
          <Icon
            name="lucide:loader-2"
            class="w-5 h-5 text-gray-500 animate-spin"
          />
        </div>

        <div
          v-else-if="communityFits.length === 0"
          class="glass-panel p-8 text-center"
        >
          <Icon
            name="lucide:ship"
            class="w-10 h-10 text-gray-600 mx-auto mb-3"
          />
          <div class="text-sm text-gray-300 mb-1">
            Be the first to share a fit
          </div>
          <div class="text-fine text-gray-500 mb-4">
            Build a fit in the editor, save it as public, and it'll show up
            here.
          </div>
          <NuxtLink
            to="/fits/create"
            class="inline-flex items-center gap-2 px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md bg-blue-500/20 text-blue-400 border border-blue-500/30 hover:bg-blue-500/30 transition-colors"
          >
            <Icon name="lucide:plus" class="w-3.5 h-3.5" />
            Create New Fit
          </NuxtLink>
        </div>

        <div v-else class="glass-panel overflow-hidden">
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
                  {{ fit.owner_name ?? "Unknown" }}
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
      <div
        class="mb-3 flex items-end justify-between border-b border-white/[0.06] pb-3"
      >
        <div class="flex items-center gap-3">
          <span
            class="flex h-8 w-8 items-center justify-center rounded-lg border border-amber-400/15 bg-amber-400/[0.06] text-xs font-bold text-amber-300"
            >05</span
          >
          <div>
            <div
              class="text-fine font-bold uppercase tracking-[0.15em] text-amber-400/80 mb-1"
            >
              Community Picks
            </div>
            <h2 class="text-lg font-bold text-white">Top Rated Fits</h2>
          </div>
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
                {{ fit.owner_name ?? "Unknown" }}
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
</template>
