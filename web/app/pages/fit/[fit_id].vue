<script setup lang="ts">
/**
 * /fit/[fit_id] — view / edit a saved user fitting.
 *
 * Same three-column editor layout as /fits/create. On mount we load
 * the fit via `useFitLoader().loadFromFitId()` — the composable sets
 * the shared currentFit + currentFitId state via setSavedFit, so the
 * FitNavbar's Save button immediately flips into "Update" mode.
 *
 * Ownership:
 *   - If the viewer owns the fit, Save = PATCH to the same row.
 *   - If the viewer does NOT own the fit (but can see it — public,
 *     corp, alliance), Save still fires PATCH which the server 403s.
 *     The error surfaces inline in the save modal. Non-owners can
 *     use the "New" button in the navbar to clone the current ring
 *     into a fresh draft before saving, which works because
 *     createNewFit clears currentFitId.
 *
 * Access errors:
 *   - 404 — the fit id doesn't exist
 *   - 403 — the viewer isn't allowed to see it
 *   Both surface as a full-page error card instead of the editor, so
 *   the user isn't staring at a blank ring wondering what happened.
 */

import ShipFit from "~/components/fit/ring/ShipFit.vue";
import ShipStatistics from "~/components/fit/stats/ShipStatistics.vue";
import HullListing from "~/components/fit/sidebar/HullListing.vue";
import HardwareListing from "~/components/fit/sidebar/HardwareListing.vue";
import FitNavbar from "~/components/fit/FitNavbar.vue";
import FitRating from "~/components/fit/FitRating.vue";
import ResourceBar from "~/components/fit/widgets/ResourceBar.vue";
import DroneBay from "~/components/fit/widgets/DroneBay.vue";
import CargoBay from "~/components/fit/widgets/CargoBay.vue";
import type { BayKey } from "~/components/fit/widgets/bayKey";

// Comments target_type 8 = fit. We use the slug path (target_id = 0,
// target_slug = fit_id) because fit_ids are 14-char text, not numeric.
const COMMENT_TARGET_FIT = 8;

const route = useRoute();
const loader = useFitLoader();
const { currentFit, currentFitOwnerId } = useCurrentFit();
const { user } = useAuth();

const fitId = computed(() => String(route.params.fit_id ?? ""));

// Owner check — drives the navbar mode and the rating widget's
// "interactive" flag (everyone with view access can rate; only owners
// edit/save). When the auth user hasn't loaded yet (initial paint
// before /auth/me resolves) we default to viewer mode so we don't
// briefly show owner controls to a non-owner.
const isOwner = computed(() => {
    const ownerId = currentFitOwnerId.value;
    const viewerId = user.value?.characterId ?? null;
    return ownerId !== null && viewerId !== null && ownerId === viewerId;
});

const navbarMode = computed<"edit" | "view">(() => (isOwner.value ? "edit" : "view"));

const sidebarTab = ref<"hulls" | "hardware">("hardware");
const expandedBay = ref<BayKey>(null);
const loading = ref(true);
const errorCode = ref<number | null>(null);

// Comments panel starts collapsed — users who don't care about the
// discussion get the clean editor view; those who do click "Comments"
// and the thread mounts below the editor (lazy: the CommentsCommentList
// component isn't instantiated until the flag flips).
const showComments = ref(false);

// Rating state — populated from the loader's return value. Kept local
// to the page (rather than in useCurrentFit) because it's specific to
// the detail view; the editor at /fits/create has no rating concept.
const ratingAvg = ref<number | null>(null);
const ratingCount = ref(0);
const viewerRating = ref<number | null>(null);

function onRatingUpdated(payload: {
    viewerRating: number | null;
    ratingAvg: number | null;
    ratingCount: number;
}) {
    viewerRating.value = payload.viewerRating;
    ratingAvg.value = payload.ratingAvg;
    ratingCount.value = payload.ratingCount;
}

useHead(() => ({
    title: currentFit.value ? `${currentFit.value.name} — Fit` : "Loading fit…",
}));

async function loadFit() {
    loading.value = true;
    errorCode.value = null;

    if (!fitId.value) {
        errorCode.value = 404;
        loading.value = false;
        return;
    }

    try {
        const meta = await loader.loadFromFitId(fitId.value);
        ratingAvg.value = meta.ratingAvg;
        ratingCount.value = meta.ratingCount;
        viewerRating.value = meta.viewerRating;
    } catch (err: unknown) {
        // apiFetch throws FetchError with a numeric .statusCode. Clamp
        // to the three states we have visuals for — 404/403/other.
        const status =
            err && typeof err === "object" && "statusCode" in err
                ? (err as { statusCode: number }).statusCode
                : 500;
        errorCode.value = status === 403 ? 403 : status === 404 ? 404 : 500;
    } finally {
        loading.value = false;
    }
}

onMounted(loadFit);

// If the user navigates between /fit/:a → /fit/:b (e.g. via a
// community-fits link on the landing page), reload the fit in place
// instead of leaving the previous one rendered.
watch(fitId, (next, prev) => {
    if (next && next !== prev) {
        loadFit();
        // Fresh fit = fresh collapsed state so comments for fit B don't
        // stay expanded when the user navigates away from fit A.
        showComments.value = false;
    }
});
</script>

<template>
    <div class="py-4">
        <FitNavbar :mode="navbarMode" />

        <!-- Error state -->
        <div
            v-if="errorCode !== null"
            class="flex justify-center"
        >
            <div
                class="glass-panel max-w-md w-full p-8 text-center"
            >
                <Icon
                    :name="errorCode === 404 ? 'lucide:search-x' : errorCode === 403 ? 'lucide:lock' : 'lucide:alert-triangle'"
                    class="w-10 h-10 text-gray-600 mx-auto mb-3"
                />
                <div class="text-sm text-gray-200 mb-1">
                    {{ errorCode === 404 ? "Fit not found" : errorCode === 403 ? "This fit is private" : "Something went wrong" }}
                </div>
                <div class="text-fine text-gray-500 mb-4">
                    <template v-if="errorCode === 404">
                        We couldn't find a fit with ID
                        <code class="font-mono text-gray-400">{{ fitId }}</code>.
                        It may have been deleted.
                    </template>
                    <template v-else-if="errorCode === 403">
                        The owner has restricted this fit to their corp or alliance and
                        you're not a member.
                    </template>
                    <template v-else>
                        Failed to load the fit. Try again in a moment.
                    </template>
                </div>
                <NuxtLink
                    to="/fits"
                    class="inline-flex items-center gap-2 px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md bg-blue-500/20 text-blue-400 border border-blue-500/30 hover:bg-blue-500/30 transition-colors"
                >
                    <Icon name="lucide:arrow-left" class="w-3.5 h-3.5" />
                    Browse Fits
                </NuxtLink>
            </div>
        </div>

        <!-- Info bar: rating widget + description, only when a fit is
             loaded and not in an error state. Placed outside ClientOnly
             because it doesn't touch the dogma engine. -->
        <div v-if="errorCode === null && currentFit" class="flex justify-center mb-4">
            <div class="flex items-center justify-between gap-4" style="width: 1252px;">
                <div class="flex items-center gap-4 min-w-0 flex-1">
                    <FitRating
                        :fit-id="fitId"
                        :rating-avg="ratingAvg"
                        :rating-count="ratingCount"
                        :viewer-rating="viewerRating"
                        :interactive="true"
                        @updated="onRatingUpdated"
                    />
                    <div
                        v-if="currentFit.description"
                        class="text-fine text-gray-500 italic truncate"
                    >
                        {{ currentFit.description }}
                    </div>
                </div>
            </div>
        </div>

        <!-- Editor (same layout as /fits/create) -->
        <div v-if="errorCode === null" class="flex justify-center">
            <ClientOnly>
                <div class="flex gap-4 items-start">
                    <!-- Editor stack -->
                    <div class="flex flex-col gap-3" style="width: 936px;">
                        <div class="flex gap-4">
                            <!-- Sidebar. Taller than the 600px ring so the
                                 ResourceBar (row 2) sits ~20px lower, flush
                                 with the bottom of the stats panel on the
                                 right. -->
                            <div
                                class="flex flex-col"
                                style="width: 320px; height: 650px;"
                            >
                                <div
                                    class="glass-panel flex gap-1 p-1 mb-2"
                                >
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
                                <div style="flex: 1; min-height: 0;">
                                    <HullListing v-if="sidebarTab === 'hulls'" />
                                    <HardwareListing v-else />
                                </div>
                            </div>

                            <!-- Ring -->
                            <div
                                style="width: 600px; height: 600px; position: relative;"
                            >
                                <ShipFit with-stats />
                                <div
                                    v-if="loading"
                                    class="absolute inset-0 flex items-center justify-center bg-black/30 pointer-events-none"
                                >
                                    <Icon
                                        name="lucide:loader-2"
                                        class="w-6 h-6 text-gray-400 animate-spin"
                                    />
                                </div>
                            </div>
                        </div>

                        <!-- Full-width resource readout -->
                        <ResourceBar v-model:expanded="expandedBay" />

                        <!-- Expanded bay drawer -->
                        <DroneBay v-if="expandedBay === 'drones'" />
                        <CargoBay v-if="expandedBay === 'cargo'" />
                    </div>

                    <!-- Stats panel -->
                    <div>
                        <ShipStatistics />
                    </div>
                </div>
                <template #fallback>
                    <div style="color: #7a7a7a">Loading dogma engine…</div>
                </template>
            </ClientOnly>
        </div>

        <!-- Comments section — collapsed by default. Sits below the editor so
             the ring is the first thing viewers see. The CommentsCommentList
             component is lazy-mounted via v-if so the thread fetch only runs
             after the user clicks in. Outside ClientOnly because comments
             don't touch the dogma engine. -->
        <div v-if="errorCode === null && currentFit" class="flex justify-center mt-6">
            <div style="width: 1252px;">
                <button
                    type="button"
                    class="w-full flex items-center justify-between gap-2 px-4 py-2.5 rounded-md bg-white/[0.04] hover:bg-white/[0.06] border border-white/[0.08] text-sm text-gray-300 transition-colors cursor-pointer"
                    @click="showComments = !showComments"
                >
                    <span class="flex items-center gap-2">
                        <Icon name="lucide:message-square" class="w-4 h-4 text-blue-400" />
                        <span class="font-semibold">Comments</span>
                        <span class="text-fine text-gray-500">Discussion for this fit</span>
                    </span>
                    <Icon
                        :name="showComments ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                        class="w-4 h-4 text-gray-500"
                    />
                </button>
                <div
                    v-if="showComments"
                    class="mt-3 p-4 rounded-md bg-white/[0.02] border border-white/[0.06]"
                >
                    <CommentsCommentList
                        :target-type="COMMENT_TARGET_FIT"
                        :target-id="0"
                        :target-slug="fitId"
                        :show-header="false"
                    />
                </div>
            </div>
        </div>
    </div>
</template>
