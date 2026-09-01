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

const { currentFit, loadFreshFit, undo, redo, canUndo, canRedo } = useCurrentFit();
const loader = useFitLoader();
const { closeSearch: closeSpotlightSearch } = useSpotlightSearch();
const route = useRoute();
const router = useRouter();

function replaceEditorQuery(updates: Record<string, string | undefined>) {
    if (!import.meta.client) return;
    const url = new URL(window.location.href);
    for (const [name, value] of Object.entries(updates)) {
        if (value === undefined) url.searchParams.delete(name);
        else url.searchParams.set(name, value);
    }
    window.history.replaceState(window.history.state, "", url);
}

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
const workspaceExpanded = ref(false);
const showShortcutHelp = ref(false);
const commandSection = ref<"hulls" | "modules" | "charges">("hulls");
const PREFS_KEY = "ek-fit-workspace-v1";

function toggleWorkspace() {
    workspaceExpanded.value = !workspaceExpanded.value;
    if (workspaceExpanded.value) fittingView.value = "table";
}

// Sidebar component refs for Cmd+F / Ctrl+F focus
type SidebarSearch = { focusSearch: () => void; openCommandSearch: (mode?: "modules" | "charges") => void };
const hullListingRef = ref<SidebarSearch | null>(null);
const hardwareListingRef = ref<SidebarSearch | null>(null);

function onKeydown(e: KeyboardEvent) {
    const target = e.target as HTMLElement | null;
    const isTyping = target?.tagName === "INPUT" || target?.tagName === "TEXTAREA" || target?.isContentEditable;
    const isFitSearch = target instanceof HTMLInputElement
        && (target.placeholder === "Search hulls…" || target.placeholder === "Search modules & charges…");
    if (workspaceExpanded.value && isFitSearch && e.key === "Tab") {
        e.preventDefault();
        const sections = ["hulls", "modules", "charges"] as const;
        const direction = e.shiftKey ? -1 : 1;
        const index = sections.indexOf(commandSection.value);
        commandSection.value = sections[(index + direction + sections.length) % sections.length]!;
        if (commandSection.value === "hulls") {
            sidebarTab.value = "hulls";
            nextTick(() => hullListingRef.value?.openCommandSearch());
        } else {
            sidebarTab.value = "hardware";
            nextTick(() => hardwareListingRef.value?.openCommandSearch(commandSection.value as "modules" | "charges"));
        }
        return;
    }
    if ((!isTyping || isFitSearch) && (e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "z") {
        e.preventDefault();
        if (e.shiftKey) {
            if (canRedo.value) redo();
        } else if (canUndo.value) undo();
        return;
    }
    // Escape belongs to transient UI (menus, search, tooltips) while the
    // editor is fullscreen. Leaving fullscreen is deliberately button-only
    // so an attempt to dismiss a popup never destroys the workspace layout.
    if ((e.metaKey || e.ctrlKey) && e.key === "f") {
        e.preventDefault();
        if (sidebarTab.value === "hulls") {
            hullListingRef.value?.focusSearch();
        } else {
            hardwareListingRef.value?.focusSearch();
        }
    }
}

function onFullscreenCommandKeydown(e: KeyboardEvent) {
    if (!workspaceExpanded.value) return;
    const target = e.target as HTMLElement | null;
    const isTyping = target?.tagName === "INPUT" || target?.tagName === "TEXTAREA" || target?.isContentEditable;
    if (!isTyping && e.key === "?") {
        e.preventDefault();
        e.stopImmediatePropagation();
        showShortcutHelp.value = !showShortcutHelp.value;
        return;
    }
    if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== "k") return;
    e.preventDefault();
    e.stopImmediatePropagation();
    closeSpotlightSearch();
    if (currentFit.value?.shipTypeId) {
        commandSection.value = "modules";
        sidebarTab.value = "hardware";
        nextTick(() => hardwareListingRef.value?.openCommandSearch("modules"));
    } else {
        commandSection.value = "hulls";
        sidebarTab.value = "hulls";
        nextTick(() => hullListingRef.value?.openCommandSearch());
    }
}

onMounted(() => {
    document.addEventListener("keydown", onFullscreenCommandKeydown, true);
    document.addEventListener("keydown", onKeydown);
});
onUnmounted(() => {
    document.removeEventListener("keydown", onFullscreenCommandKeydown, true);
    document.removeEventListener("keydown", onKeydown);
});

let previousBodyOverflow = "";
watch(workspaceExpanded, (expanded) => {
    if (!import.meta.client) return;
    if (expanded) {
        previousBodyOverflow = document.body.style.overflow;
        document.body.style.overflow = "hidden";
    } else {
        document.body.style.overflow = previousBodyOverflow;
    }
});
onUnmounted(() => {
    if (import.meta.client) document.body.style.overflow = previousBodyOverflow;
});

onMounted(async () => {
    try {
        const saved = JSON.parse(localStorage.getItem(PREFS_KEY) ?? "null") as {
            sidebarTab?: "hulls" | "hardware";
            fittingView?: "ship" | "table";
            expandedBay?: BayKey;
            workspaceExpanded?: boolean;
        } | null;
        if (saved?.sidebarTab) sidebarTab.value = saved.sidebarTab;
        if (saved?.fittingView) fittingView.value = saved.fittingView;
        if (saved?.expandedBay !== undefined) expandedBay.value = saved.expandedBay;
        if (saved?.workspaceExpanded && window.innerWidth >= 1024) workspaceExpanded.value = true;
    } catch { /* Ignore stale local preferences. */ }
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
            replaceEditorQuery({ fit: undefined });
            return;
        }
        const encoded = await encodeFitV3(fit);
        if (new URL(window.location.href).searchParams.get("fit") === encoded) return;
        replaceEditorQuery({ fit: encoded });
    },
    { deep: true },
);

// Keep ?view= in sync with the fitting view toggle.
watch(fittingView, (view) => {
    replaceEditorQuery({ view: view === "ship" ? undefined : view });
});

watch([sidebarTab, fittingView, expandedBay, workspaceExpanded], () => {
    if (!import.meta.client) return;
    try {
        localStorage.setItem(PREFS_KEY, JSON.stringify({
            sidebarTab: sidebarTab.value,
            fittingView: fittingView.value,
            expandedBay: expandedBay.value,
            workspaceExpanded: workspaceExpanded.value,
        }));
    } catch { /* Storage may be unavailable in private browsing. */ }
});
</script>

<template>
    <Teleport to="body" :disabled="!workspaceExpanded">
    <div
        :class="workspaceExpanded
            ? 'fixed inset-0 z-[1000] flex h-dvh w-dvw flex-col overflow-hidden bg-[#07080b] p-3'
            : 'py-4'"
    >
        <FitNavbar
            expandable
            :expanded="workspaceExpanded"
            :class="workspaceExpanded ? '!mb-2 shrink-0' : ''"
            @toggle-expanded="toggleWorkspace"
        />

        <div class="flex min-h-0 justify-center" :class="workspaceExpanded ? 'flex-1' : ''">
            <ClientOnly>
                <!-- 2-column grid: [sidebar+center group] [stats].
                     Heights sync via items-stretch. min 650px. -->
                <div
                    class="editor-grid grid gap-4 items-stretch"
                    :class="workspaceExpanded ? 'workspace-grid--expanded h-full min-h-0 w-full' : ''"
                    :style="{
                        gridTemplateColumns: workspaceExpanded ? undefined : '1fr 300px',
                        minHeight: workspaceExpanded ? '0' : '650px',
                    }"
                >
                    <!-- Left group: sidebar + center + resource bar -->
                    <div class="workspace-editor-stack flex flex-col gap-2 min-h-0">
                        <!-- Top area: sidebar + ring/table side by side -->
                        <div
                            class="workspace-editor-top flex gap-4 min-h-0"
                            :class="workspaceExpanded || fittingView === 'table' ? 'flex-1' : 'flex-none'"
                            :style="workspaceExpanded
                                ? { minHeight: '0' }
                                : fittingView === 'ship'
                                    ? { height: '632px' }
                                    : { minHeight: '632px' }"
                        >
                            <!-- Sidebar -->
                            <div
                                class="workspace-editor-sidebar flex flex-col min-h-0"
                                :class="{ 'workspace-sidebar--expanded': workspaceExpanded }"
                                :style="{ width: workspaceExpanded ? '300px' : '260px' }"
                            >
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
                                    <FitHullIdentity v-if="fittingView === 'table'" compact />
                                </div>

                                <!-- Ship ring view — pin to top so expanding the sidebar
                                     doesn't drag the ring downward to stay vertically centered. -->
                                <div
                                    v-show="fittingView === 'ship'"
                                    class="flex-1 flex items-start justify-center min-h-0"
                                >
                                <div class="fit-ring-frame" :class="{ 'fit-ring-frame--expanded': workspaceExpanded }">
                                    <div class="fit-ring-canvas relative">
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
                                </div>

                                <!-- Module table view — fills available height -->
                                <div
                                    v-show="fittingView === 'table'"
                                    class="flex-1 rounded-lg bg-white/[0.02] border border-white/[0.06] min-h-0"
                                >
                                    <FitTable :constrained="workspaceExpanded" />
                                </div>

                                <div v-if="workspaceExpanded" class="mt-2 shrink-0 space-y-2">
                                    <ResourceBar v-model:expanded="expandedBay" />
                                    <div class="grid h-56 grid-cols-2 gap-2">
                                        <div class="min-h-0 overflow-y-auto rounded-lg">
                                            <DroneBay class="h-full" />
                                        </div>
                                        <div class="min-h-0 overflow-y-auto rounded-lg">
                                            <CargoBay class="h-full" />
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- Resource bar + bays span full width of left group -->
                        <template v-if="!workspaceExpanded">
                            <ResourceBar v-model:expanded="expandedBay" />
                            <DroneBay v-if="expandedBay === 'drones'" />
                            <CargoBay v-if="expandedBay === 'cargo'" />
                        </template>
                    </div>

                    <!-- Right column: stats panel — h-full so it stretches with the grid row -->
                    <div class="flex min-h-0 h-full flex-col" :class="workspaceExpanded ? 'overflow-y-auto' : ''">
                        <FitHullIdentity v-if="workspaceExpanded && fittingView === 'table'" />
                        <ShipStatistics
                            :key="workspaceExpanded ? 'expanded' : 'standard'"
                            class="min-h-0 flex-1"
                            :expand-details="workspaceExpanded"
                        />
                    </div>

                    <FitFullscreenFitVariants
                        v-if="workspaceExpanded"
                        class="workspace-variants min-h-0"
                    />
                </div>
                <template #fallback>
                    <div style="color: #7a7a7a">Loading dogma engine…</div>
                </template>
            </ClientOnly>
        </div>
    </div>
    </Teleport>

    <Teleport to="body">
        <div v-if="showShortcutHelp" class="fixed inset-0 z-[1200] flex items-center justify-center bg-black/70 p-4" @click.self="showShortcutHelp = false">
            <div class="w-full max-w-lg rounded-xl border border-white/10 bg-[#111318] p-5 shadow-2xl">
                <div class="mb-4 flex items-center justify-between">
                    <h2 class="text-sm font-bold uppercase tracking-[0.14em] text-blue-300">Fitting shortcuts</h2>
                    <button type="button" class="text-gray-500 hover:text-white" @click="showShortcutHelp = false"><Icon name="lucide:x" class="h-4 w-4" /></button>
                </div>
                <div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-xs">
                    <kbd>⌘/Ctrl K</kbd><span class="text-gray-400">Search hulls or hardware</span>
                    <kbd>↑ / ↓</kbd><span class="text-gray-400">Move through search or table rows</span>
                    <kbd>Enter</kbd><span class="text-gray-400">Fit the selected result</span>
                    <kbd>Shift Enter</kbd><span class="text-gray-400">Fill compatible high slots</span>
                    <kbd>Shift Click</kbd><span class="text-gray-400">Fill compatible high slots</span>
                    <kbd>Delete</kbd><span class="text-gray-400">Remove charge, then selected module</span>
                    <kbd>⌘/Ctrl Z</kbd><span class="text-gray-400">Undo</span>
                    <kbd>⌘/Ctrl Shift Z</kbd><span class="text-gray-400">Redo</span>
                    <kbd>?</kbd><span class="text-gray-400">Toggle this help</span>
                    <kbd>Escape</kbd><span class="text-gray-400">Close transient UI; fullscreen stays open</span>
                </div>
            </div>
        </div>
    </Teleport>
</template>

<style scoped>
.fit-ring-frame,
.fit-ring-canvas {
    width: 600px;
    height: 600px;
}

@media (min-height: 1000px) {
    .fit-ring-frame--expanded {
        width: 720px;
        height: 720px;
    }

    .fit-ring-frame--expanded .fit-ring-canvas {
        transform: scale(1.2);
        transform-origin: top left;
    }
}

.workspace-sidebar--expanded :deep(.text-xs) {
    font-size: 0.8125rem;
    line-height: 1.15rem;
}

.workspace-sidebar--expanded :deep(.text-fine) {
    font-size: 0.75rem;
}

.workspace-grid--expanded {
    grid-template-columns: minmax(0, 1fr) 300px minmax(240px, 15vw);
}

kbd {
    border: 1px solid rgb(255 255 255 / 0.1);
    border-radius: 0.3rem;
    background: rgb(255 255 255 / 0.04);
    padding: 0.15rem 0.4rem;
    color: rgb(209 213 219);
    font-size: 0.6875rem;
}

@media (max-width: 1199px) {
    .workspace-grid--expanded {
        grid-template-columns: minmax(0, 1fr) 300px;
    }

    .workspace-variants {
        display: none;
    }
}

@media (max-width: 900px) {
    .editor-grid,
    .workspace-grid--expanded {
        grid-template-columns: minmax(0, 1fr) !important;
    }

    .workspace-grid--expanded > :nth-child(2),
    .workspace-variants {
        display: none;
    }

    .workspace-grid--expanded .workspace-editor-top {
        flex-direction: column;
        overflow-y: auto;
    }

    .workspace-grid--expanded .workspace-editor-sidebar {
        width: 100% !important;
        min-height: 16rem;
        height: 38dvh;
        flex: none;
    }

    .editor-grid:not(.workspace-grid--expanded) .workspace-editor-top {
        overflow-x: auto;
        padding-bottom: 0.5rem;
    }
}
</style>
