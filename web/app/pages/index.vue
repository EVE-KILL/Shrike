<script setup lang="ts">
const { domainConfig, isDomainMode } = useDomainConfig()
const siteName = computed(() => domainConfig.value?.siteName || 'EVE-KILL')

// API endpoints — swap to /api/custom/* when in domain mode
const killlistEndpoint = computed(() => isDomainMode.value ? '/api/custom/killlist' : undefined)
const statsEndpoint = computed(() => isDomainMode.value ? '/api/custom/stats' : undefined)

// Homepage meta description carries two target phrases ("EVE killboard" and
// "EVE Online killboard") in the first 160 chars — the slice Google/Bing use
// for SERP snippets. We rank pos 3–6 on Bing for "eve killboard" with 300+
// daily impressions but 0–3% CTR; adding the bare phrase to the meta copy
// should lift the snippet match rate.
useSeoMeta({
    description: computed(() => domainConfig.value?.siteDescription || 'EVE Killboard — the community-driven EVE Online killboard. Real-time killmails, top pilots, corporations, alliances, and battle reports across New Eden.'),
    ogTitle: computed(() => `${siteName.value} — EVE Online Killboard`),
    ogDescription: computed(() => domainConfig.value?.siteDescription || 'The EVE Killboard — real-time kills, top pilots, and battle reports from New Eden.'),
    ogType: 'website',
})

if (!isDomainMode.value) {
    useSchemaOrg([
        defineWebSite({
            name: 'EVE-KILL',
            url: 'https://eve-kill.com',
            description: 'The EVE Killboard — community-driven killboard for EVE Online with real-time killmails, battle reports, and player statistics.',
            potentialAction: [
                {
                    '@type': 'SearchAction',
                    'target': {
                        '@type': 'EntryPoint',
                        'urlTemplate': 'https://eve-kill.com/search?q={search_term_string}',
                    },
                    'query-input': 'required name=search_term_string',
                },
            ],
        }),
        defineWebPage({
            '@type': 'CollectionPage',
            'name': 'EVE-KILL — EVE Online Killboard',
            'description': 'The EVE Killboard: real-time killmail feed, top pilots, corporations, and alliances in EVE Online.',
        }),
    ])
}

// Homepage: layout titleTemplate handles formatting.
//   EVE-KILL default → `Home - EVE-KILL.com`
//   Custom domain    → just the domain's site name (empty title hits the
//                      falsy branch in the titleTemplate)
useHead({
    title: () => isDomainMode.value ? '' : 'Home',
})

// ── Domain mode widget helpers ──────────────────────────────────────────────
const widgets = computed(() => domainConfig.value?.widgets)

const enabledWidgets = (section: 'top' | 'left' | 'right') =>
    (widgets.value?.[section] || []).filter(w => w.enabled)

const hasLeft = computed(() => enabledWidgets('left').length > 0)
const hasRight = computed(() => enabledWidgets('right').length > 0)
const hasColumns = computed(() => hasLeft.value || hasRight.value)

const columnGridTemplate = computed(() => {
    if (!hasLeft.value) return '1fr'
    if (!hasRight.value) return '1fr'
    const ratio = widgets.value?.columnRatio || '250px_1fr'
    return ratio.replace('_', ' ')
})
</script>

<template>
    <h1 class="sr-only">{{ siteName }} — EVE Online Killboard</h1>
    <h2 v-if="!isDomainMode" class="sr-only">The EVE Killboard: real-time killmail feed, top pilots, corporations, alliances, battle reports, and ship fitting analysis for EVE Online.</h2>

    <!-- One widget tree for both breakpoints; CSS handles column placement. -->
    <div v-if="isDomainMode" class="space-y-4">
        <div v-if="enabledWidgets('top').length" class="space-y-4">
            <HomepageWidget
                v-for="(w, i) in enabledWidgets('top')" :key="'top-' + i"
                :widget="w"
                :stats-endpoint="statsEndpoint!"
                :killlist-endpoint="killlistEndpoint!"
                :entities="domainConfig!.entities"
            />
        </div>
        <div v-if="hasColumns" class="grid grid-cols-1 gap-4 md:grid-cols-[var(--widget-columns)]"
            :style="{ '--widget-columns': columnGridTemplate }">
            <div v-if="hasRight" class="min-w-0 space-y-4" :class="hasLeft ? 'md:col-start-2 md:row-start-1' : ''">
                <HomepageWidget
                    v-for="(w, i) in enabledWidgets('right')" :key="'right-' + i"
                    :widget="w"
                    :stats-endpoint="statsEndpoint!"
                    :killlist-endpoint="killlistEndpoint!"
                    :entities="domainConfig!.entities"
                />
            </div>
            <div v-if="hasLeft" class="min-w-0 space-y-4 md:col-start-1 md:row-start-1 md:space-y-0">
                <HomepageWidget
                    v-for="(w, i) in enabledWidgets('left')" :key="'left-' + i"
                    :widget="w"
                    :stats-endpoint="statsEndpoint!"
                    :killlist-endpoint="killlistEndpoint!"
                    :entities="domainConfig!.entities"
                />
            </div>
        </div>
    </div>

    <div v-else class="space-y-4">
        <MostValuable :api-endpoint="statsEndpoint" />
        <div class="grid grid-cols-1 gap-4 md:grid-cols-[250px_minmax(0,1fr)]">
            <div class="min-w-0 md:col-start-2 md:row-start-1">
                <KillList killlist-type="latest" :api-endpoint="killlistEndpoint" :stream-topics="['all']" :quick-filter="true" />
            </div>
            <!-- SSR keeps the links and content; interactivity starts near the viewport. -->
            <div class="min-w-0 md:col-start-1 md:row-start-1">
                <DeferredTopBox title="Top Characters" dataType="characters" :limit="10" />
                <DeferredTopBox title="Top Corporations" dataType="corporations" :limit="10" />
                <DeferredTopBox title="Top Alliances" dataType="alliances" :limit="10" />
                <DeferredTopBox title="Top Factions" dataType="factions" :limit="6" />
                <DeferredTopBox title="Top Ships" dataType="ships" :limit="10" />
                <DeferredTopBox title="Top Systems" dataType="systems" :limit="10" />
                <DeferredTopBox title="Top Regions" dataType="regions" :limit="10" />
            </div>
        </div>
    </div>
</template>
