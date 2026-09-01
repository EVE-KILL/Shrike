<script setup lang="ts">
interface KillmailLabel {
  id: string;
  name: string;
  description: string;
  category: string;
  count: number;
  view_url: string;
  search_filters: Record<string, unknown>;
}

interface CategoriesResponse {
  labels: KillmailLabel[];
}

useHead({ title: "Killmail Categories" });
useSeoMeta({
  description:
    "Browse the categories EVE-KILL applies to EVE Online killmails, with historical counts and direct search links.",
  ogTitle: "Killmail Categories — EVE-KILL",
  ogDescription:
    "Explore EVE Online killmail categories by space, engagement, involvement, faction warfare, value, hull class, and technology.",
});

const { data, pending, error } =
  await useApiFetch<CategoriesResponse>("/api/labels");
const query = ref("");
const totalKills = computed(() =>
  Math.max(0, ...(data.value?.labels ?? []).map((label) => label.count)),
);

const categoryVisuals: Record<string, { icon: string; image: string }> = {
  Space: { icon: "lucide:orbit", image: "/images/regions/10000002?size=64" },
  Timezone: { icon: "lucide:clock", image: "/images/types/19720/icon?size=64" },
  Engagement: {
    icon: "lucide:swords",
    image: "/images/types/638/icon?size=64",
  },
  "Killmail Type": {
    icon: "lucide:skull",
    image: "/images/types/670/icon?size=64",
  },
  Involvement: {
    icon: "lucide:users",
    image: "/images/types/11567/icon?size=64",
  },
  "Faction Warfare": {
    icon: "lucide:shield",
    image: "/images/corporations/1000180/logo?size=64",
  },
  Value: { icon: "lucide:coins", image: "/images/types/34/icon?size=64" },
  "Value Bands": { icon: "lucide:gem", image: "/images/types/34/icon?size=64" },
  "Victim Category": {
    icon: "lucide:boxes",
    image: "/images/types/35832/icon?size=64",
  },
  "Victim Hull": {
    icon: "lucide:rocket",
    image: "/images/types/24694/icon?size=64",
  },
  Technology: {
    icon: "lucide:settings",
    image: "/images/types/12005/icon?size=64",
  },
};

const categoryVisual = (category: string) =>
  categoryVisuals[category] ?? {
    icon: "lucide:tag",
    image: "/images/types/670/icon?size=64",
  };

const categoryOrder = [
  "Space",
  "Timezone",
  "Engagement",
  "Killmail Type",
  "Involvement",
  "Faction Warfare",
  "Value",
  "Value Bands",
  "Victim Category",
  "Victim Hull",
  "Technology",
];

const filteredLabels = computed(() => {
  const needle = query.value.trim().toLowerCase();
  const labels = data.value?.labels ?? [];
  if (!needle) return labels;
  return labels.filter(
    (label) =>
      label.name.toLowerCase().includes(needle) ||
      label.id.toLowerCase().includes(needle) ||
      label.description.toLowerCase().includes(needle) ||
      label.category.toLowerCase().includes(needle),
  );
});

const groups = computed(() => {
  const known = new Set(categoryOrder);
  const extra = [
    ...new Set(filteredLabels.value.map((label) => label.category)),
  ]
    .filter((category) => !known.has(category))
    .sort((a, b) => a.localeCompare(b));

  return [...categoryOrder, ...extra]
    .map((category) => ({
      category,
      labels: filteredLabels.value.filter(
        (label) => label.category === category,
      ),
    }))
    .filter((group) => group.labels.length > 0);
});

const searchTarget = (label: KillmailLabel) => ({
  path: "/advancedsearch",
  query: { q: JSON.stringify(label.search_filters) },
});

const canSearch = (label: KillmailLabel) =>
  Object.keys(label.search_filters ?? {}).length > 0;
const formatCount = (count: number) =>
  new Intl.NumberFormat("en-US").format(count);
</script>

<template>
  <div class="max-w-6xl mx-auto px-4 py-8">
    <PageHeader
      class="mb-6"
      title="Killmail Categories"
      eyebrow="Explore the killmail archive"
      icon="lucide:tags"
      image-src="/images/types/11567/overlayrender?size=1024"
      image-mode="background"
      description="Browse the killmail archive by activity, location, fleet size, value, ship class, and other useful classifications. A killmail can appear in more than one category."
    >
      <template #meta>
        <div class="flex flex-wrap gap-2 text-xs text-gray-500">
          <span
            class="rounded border border-white/[0.08] bg-black/20 px-2.5 py-1.5"
            ><strong class="text-gray-300">{{
              data?.labels?.length ?? 0
            }}</strong>
            categories</span
          >
          <span
            class="rounded border border-white/[0.08] bg-black/20 px-2.5 py-1.5"
            ><strong class="text-gray-300">{{ groups.length }}</strong>
            groups</span
          >
          <span
            class="rounded border border-white/[0.08] bg-black/20 px-2.5 py-1.5"
            ><strong class="text-gray-300">{{
              formatCount(totalKills)
            }}</strong>
            in the largest category</span
          >
        </div>
      </template>
    </PageHeader>

    <div class="relative mb-6 max-w-xl">
      <Icon
        name="lucide:search"
        class="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-gray-600"
      />
      <input
        v-model="query"
        type="search"
        placeholder="Filter categories..."
        class="w-full rounded-lg border border-white/[0.08] bg-white/[0.03] py-2.5 pl-9 pr-3 text-sm text-gray-200 placeholder:text-gray-600 focus:border-blue-500/40 focus:outline-none"
      />
    </div>

    <div
      v-if="pending"
      class="glass-panel p-8 text-center text-sm text-gray-500"
    >
      Loading classifications...
    </div>
    <div
      v-else-if="error"
      class="glass-panel p-8 text-center text-sm text-red-400"
    >
      Unable to load killmail categories.
    </div>
    <div
      v-else-if="groups.length === 0"
      class="glass-panel p-8 text-center text-sm text-gray-500"
    >
      No categories match “{{ query }}”.
    </div>
    <div v-else class="space-y-7">
      <section v-for="group in groups" :key="group.category">
        <div class="mb-3 flex items-center gap-3">
          <EveImage
            :src="categoryVisual(group.category).image"
            :size="64"
            :alt="group.category"
            class="h-10 w-10 rounded-lg bg-gray-900 object-cover ring-1 ring-white/10"
          />
          <div>
            <div class="flex items-center gap-2">
              <Icon
                :name="categoryVisual(group.category).icon"
                class="text-sm text-blue-400"
              />
              <h2
                class="text-sm font-bold uppercase tracking-[0.14em] text-gray-300"
              >
                {{ group.category }}
              </h2>
            </div>
            <span class="text-fine text-gray-600"
              >{{ group.labels.length }} categories</span
            >
          </div>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
          <article
            v-for="label in group.labels"
            :key="label.id"
            class="group rounded-lg border border-white/[0.08] bg-white/[0.02] p-4 flex flex-col gap-3 transition-colors hover:border-blue-400/20 hover:bg-blue-500/[0.03]"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h3
                  class="text-sm font-semibold text-white transition-colors group-hover:text-blue-300"
                >
                  {{ label.name }}
                </h3>
                <code class="text-fine text-blue-400/70">{{ label.id }}</code>
              </div>
              <span
                class="text-xs tabular-nums text-gray-400 whitespace-nowrap"
                :title="`${formatCount(label.count)} killmails`"
              >
                {{ formatCount(label.count) }}
              </span>
            </div>
            <p class="text-xs leading-relaxed text-gray-500 flex-1">
              {{ label.description }}
            </p>
            <div class="flex gap-2">
              <NuxtLink
                :to="label.view_url"
                class="px-2.5 py-1.5 rounded border border-white/[0.08] bg-white/[0.03] text-xs text-gray-300 hover:border-blue-500/30 hover:text-white transition-colors"
              >
                View kills
              </NuxtLink>
              <NuxtLink
                v-if="canSearch(label)"
                :to="searchTarget(label)"
                class="px-2.5 py-1.5 rounded border border-blue-500/20 bg-blue-500/[0.08] text-xs text-blue-300 hover:bg-blue-500/[0.14] transition-colors"
              >
                Advanced search
              </NuxtLink>
            </div>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>
