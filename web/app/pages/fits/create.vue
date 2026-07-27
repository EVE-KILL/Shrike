<script setup lang="ts">
/**
 * /fits/create — the ship fitting editor.
 *
 * Layout (two outer columns):
 *   left "editor stack" — 936px wide:
 *     row 1  [260px sidebar] [600px ring]
 *     row 2  ResourceBar (full 936px — spans sidebar + ring)
 *     row 3  expanded Drone Bay / Cargo drawer (optional)
 *   right "stats panel" — 300px wide, aligned to top
 *
 * Totalling 936 + 16 + 300 = 1252px, well within the site's 1360
 * content max. The expandable drawer pattern (ResourceBar cell →
 * drawer below) keeps the idle state compact while giving a big
 * drag-drop target when the user actually wants to edit bays.
 *
 * Fit loading priority on mount:
 *   1. `?fit=v3:...` query param (shareable links, "Load in Editor"
 *       buttons on the fittings tab, etc.)
 *   2. No fallback — the editor starts empty and the sidebar opens on
 *      the Hulls tab so the visitor picks a ship first.
 *
 * Everything that touches the dogma engine runs client-side — hence
 * the <ClientOnly> wrapper. The FitNavbar at the top stays SSR-safe.
 */

import ShipFit from "~/components/fit/ring/ShipFit.vue";
import FitTable from "~/components/fit/table/FitTable.vue";
import ShipStatistics from "~/components/fit/stats/ShipStatistics.vue";
import HullListing from "~/components/fit/sidebar/HullListing.vue";
import HardwareListing from "~/components/fit/sidebar/HardwareListing.vue";
import FitNavbar from "~/components/fit/FitNavbar.vue";
import ResourceBar from "~/components/fit/widgets/ResourceBar.vue";
import DroneBay from "~/components/fit/widgets/DroneBay.vue";
import CargoBay from "~/components/fit/widgets/CargoBay.vue";
import type { BayKey } from "~/components/fit/widgets/bayKey";
import { encodeFitV3 } from "~/composables/fit/encode";

useHead({ title: "Create Fit" });
useSeoMeta({
    description:
        "Build EVE Online ship fittings in your browser with real dogma engine calculations. Import from EFT, share via link, export to clipboard.",
    ogTitle: "Create a Fit — EVE-KILL",
    ogDescription:
        "Browser-based EVE Online fitting tool with real dogma calculations, EFT import, and shareable links.",
});

const { currentFit, loadFreshFit } = useCurrentFit();
const loader = useFitLoader();
const route = useRoute();
const router = useRouter();

// Default to "hulls" when the editor is empty — that's the natural next
// action for someone who just opened the page. As soon as a fit exists we
// flip to "hardware" so modules are one click away.
const sidebarTab = ref<"hulls" | "hardware">("hulls");

// Main fitting view: "ship" = ring wheel, "table" = Pyfa-style module list.
// Initialised from ?view= query param if present, otherwise defaults to "ship".
const fittingView = ref<"ship" | "table">(
    route.query.view === "table" ? "table" : "ship",
);

// Which bay drawer is expanded below the ResourceBar. Click a "Drone
// Bay" / "Cargo" cell in the bar to toggle; clicking the currently-
// expanded cell collapses. Starts collapsed.
const expandedBay = ref<BayKey>(null);

// Sidebar component refs for Cmd+F / Ctrl+F focus
const hullListingRef = ref<{ focusSearch: () => void } | null>(null);
const hardwareListingRef = ref<{ focusSearch: () => void } | null>(null);

function onKeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key === "f") {
        e.preventDefault();
        if (sidebarTab.value === "hulls") {
            hullListingRef.value?.focusSearch();
        } else {
            hardwareListingRef.value?.focusSearch();
        }
    }
}

onMounted(() => document.addEventListener("keydown", onKeydown));
onUnmounted(() => document.removeEventListener("keydown", onKeydown));

onMounted(async () => {
    // `?fit=v3:xxx` takes priority — it's what the "Load in Editor" button
    // on /item/{id}/fittings and /kill/{id} uses, and what shareable links
    // resolve to.
    const fitParam = route.query.fit;
    if (typeof fitParam === "string" && fitParam.length > 0) {
        const loaded = await loader.loadFromEncoded(fitParam);
        if (loaded) {
            sidebarTab.value = "hardware";
        }
        return;
    }

    // No ?fit= query — /fits/create is the "new draft" URL. Always clear
    // the shared fit state on mount so navigating here from anywhere
    // (landing page "Create new fit" button, /fit/:id back-button, an
    // unsaved draft left over from a previous visit) lands the visitor
    // in a blank editor with the hulls sidebar ready. If they wanted to
    // keep the old work, they should have saved it first.
    loadFreshFit(null);
    sidebarTab.value = "hulls";
});

// Once a fit exists (either loaded via query param or created from the
// hulls sidebar), auto-flip to hardware so the user can start fitting.
watch(
    () => currentFit.value?.shipTypeId,
    (shipId, prevShipId) => {
        if (shipId && !prevShipId) {
            sidebarTab.value = "hardware";
        }
    },
);

// Keep the URL's `?fit=` query in lockstep with the editor state so a
// refresh (or a copy of the address bar) preserves in-progress work.
// Every mutator on the ring goes through useFitManager.setFit which
// replaces currentFit with a cloned object — that's what this watcher
// observes. We use router.replace so the sync doesn't pollute history
// and avoid a replace when the encoded string already matches, so the
// initial ?fit= load doesn't bounce the URL for no reason.
watch(
    () => currentFit.value,
    async (fit) => {
        // Empty editor → strip any leftover ?fit= query. "New Fit" click
        // lands here with shipTypeId=0, as does an unrelated `null`.
        if (!fit || fit.shipTypeId === 0) {
            if (route.query.fit !== undefined) {
                const nextQuery = { ...route.query };
                delete nextQuery.fit;
                await router.replace({ query: nextQuery });
            }
            return;
        }
        const encoded = await encodeFitV3(fit);
        if (route.query.fit === encoded) return;
        await router.replace({ query: { ...route.query, fit: encoded } });
    },
    { deep: true },
);

// Keep ?view= in sync with the fitting view toggle.
watch(fittingView, async (view) => {
    const next = { ...route.query };
    if (view === "ship") {
        delete next.view;
    } else {
        next.view = view;
    }
    await router.replace({ query: next });
});
</script>

<template>
    <div class="py-4">
        <FitNavbar />

        <div class="flex justify-center">
            <ClientOnly>
                <!-- 2-column grid: [sidebar+center group] [stats].
                     Heights sync via items-stretch. min 650px. -->
                <div
                    class="grid gap-4 items-stretch"
                    style="grid-template-columns: 1fr 300px; min-height: 650px;"
                >
                    <!-- Left group: sidebar + center + resource bar -->
                    <div class="flex flex-col gap-2 min-h-0">
                        <!-- Top area: sidebar + ring/table side by side -->
                        <div class="flex gap-4 flex-1 min-h-0">
                            <!-- Sidebar -->
                            <div class="flex flex-col min-h-0" style="width: 260px;">
                                <div class="glass-panel flex gap-1 p-1 mb-2">
                                    <button
                                        type="button"
                                        :class="[
                                            'flex-1 px-3 py-1.5 text-[11px] font-bold uppercase tracking-[0.12em] rounded transition-colors',
                                            sidebarTab === 'hulls'
                                                ? 'bg-blue-500/[0.15] text-blue-400'
                                                : 'text-gray-500 hover:text-gray-300',
                                        ]"
                                        @click="sidebarTab = 'hulls'"
                                    >
                                        Hulls
                                    </button>
                                    <button
                                        type="button"
                                        :class="[
                                            'flex-1 px-3 py-1.5 text-[11px] font-bold uppercase tracking-[0.12em] rounded transition-colors',
                                            sidebarTab === 'hardware'
                                                ? 'bg-blue-500/[0.15] text-blue-400'
                                                : 'text-gray-500 hover:text-gray-300',
                                        ]"
                                        @click="sidebarTab = 'hardware'"
                                    >
                                        Hardware
                                    </button>
                                </div>
                                <div class="flex-1 min-h-0">
                                    <HullListing v-if="sidebarTab === 'hulls'" ref="hullListingRef" />
                                    <HardwareListing v-else ref="hardwareListingRef" />
                                </div>
                            </div>

                            <!-- Center: ring / table — min-w-0 prevents table content
                                 from pushing the flex item wider than its allocation -->
                            <div class="flex flex-col flex-1 min-h-0 min-w-0 overflow-hidden">
                                <div class="flex gap-1 mb-1">
                                    <button
                                        type="button"
                                        :class="[
                                            'px-3 py-1 text-[11px] font-bold uppercase tracking-[0.12em] rounded transition-colors',
                                            fittingView === 'ship'
                                                ? 'bg-blue-500/[0.15] text-blue-400'
                                                : 'text-gray-500 hover:text-gray-300',
                                        ]"
                                        @click="fittingView = 'ship'"
                                    >
                                        Ship
                                    </button>
                                    <button
                                        type="button"
                                        :class="[
                                            'px-3 py-1 text-[11px] font-bold uppercase tracking-[0.12em] rounded transition-colors',
                                            fittingView === 'table'
                                                ? 'bg-blue-500/[0.15] text-blue-400'
                                                : 'text-gray-500 hover:text-gray-300',
                                        ]"
                                        @click="fittingView = 'table'"
                                    >
                                        Table
                                    </button>
                                </div>

                                <!-- Ship ring view — pin to top so expanding the sidebar
                                     doesn't drag the ring downward to stay vertically centered. -->
                                <div
                                    v-show="fittingView === 'ship'"
                                    class="flex-1 flex items-start justify-center min-h-0"
                                >
                                <div class="relative" style="width: 600px; height: 600px;">
                                    <ShipFit with-stats />
                                    <div
                                        v-if="!currentFit"
                                        class="absolute inset-0 flex items-center justify-center pointer-events-none"
                                    >
                                        <div class="text-center px-6 py-4 rounded-lg bg-black/60 border border-white/[0.08] pointer-events-auto">
                                            <Icon name="lucide:ship" class="w-8 h-8 text-gray-600 mx-auto mb-2" />
                                            <div class="text-sm text-gray-300 mb-1">Pick a hull to start</div>
                                            <div class="text-fine text-gray-500">
                                                Choose a ship from the sidebar or paste an EFT / share link from the top bar.
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                </div>

                                <!-- Module table view — fills available height -->
                                <div
                                    v-show="fittingView === 'table'"
                                    class="flex-1 rounded-lg bg-white/[0.02] border border-white/[0.06] min-h-0"
                                >
                                    <FitTable />
                                </div>
                            </div>
                        </div>

                        <!-- Resource bar + bays span full width of left group -->
                        <ResourceBar v-model:expanded="expandedBay" />
                        <DroneBay v-if="expandedBay === 'drones'" />
                        <CargoBay v-if="expandedBay === 'cargo'" />
                    </div>

                    <!-- Right column: stats panel — h-full so it stretches with the grid row -->
                    <div class="min-h-0 h-full">
                        <ShipStatistics class="!h-full" />
                    </div>
                </div>
                <template #fallback>
                    <div style="color: #7a7a7a">Loading dogma engine…</div>
                </template>
            </ClientOnly>
        </div>
    </div>
</template>
