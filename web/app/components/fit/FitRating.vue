<script setup lang="ts">
/**
 * Star rating widget for a user fit.
 *
 * Two modes via the `interactive` prop:
 *   - interactive=false: read-only. Shows filled stars up to the rounded
 *     average, the numeric average, and the rating count. Used for fit
 *     list rows and the landing page.
 *   - interactive=true: clickable. Hovering a star previews that rating,
 *     clicking commits it via PUT /api/fit/:id/rating. Logged-out click
 *     bounces to EVE SSO (keyed back to the current page). Used on the
 *     fit detail page (/fit/:fit_id).
 *
 * The parent is responsible for showing viewer_rating (the authed user's
 * own rating, if any) vs the aggregate. On commit, the server returns
 * the fresh aggregate + the rater's new rating — the widget emits both
 * so the parent can update its local state without refetching.
 */

const props = withDefaults(
    defineProps<{
        fitId: string;
        ratingAvg: number | null;
        ratingCount: number;
        viewerRating: number | null;
        interactive?: boolean;
        size?: "sm" | "md";
    }>(),
    {
        interactive: false,
        size: "md",
    },
);

const emit = defineEmits<{
    (
        e: "updated",
        payload: {
            viewerRating: number | null;
            ratingAvg: number | null;
            ratingCount: number;
        },
    ): void;
}>();

const { isAuthenticated, login } = useAuth();

const hovered = ref<number | null>(null);
const saving = ref(false);

// What the stars should light up to. Priority:
//   1. Hover preview (interactive only)
//   2. The viewer's own rating if they've rated it
//   3. The rounded aggregate
const displayRating = computed(() => {
    if (hovered.value !== null) return hovered.value;
    if (props.viewerRating !== null) return props.viewerRating;
    if (props.ratingAvg !== null) return Math.round(props.ratingAvg);
    return 0;
});

// Width class per size preset. Inline styles on the svg for easy
// tweaking without piling up Tailwind size classes.
const starBox = computed(() => (props.size === "sm" ? "w-3.5 h-3.5" : "w-5 h-5"));

function starIsOn(index: number): boolean {
    return index <= displayRating.value;
}

function onMouseEnter(index: number) {
    if (!props.interactive) return;
    hovered.value = index;
}

function onMouseLeave() {
    if (!props.interactive) return;
    hovered.value = null;
}

async function onClick(index: number) {
    if (!props.interactive || saving.value) return;

    if (!isAuthenticated.value) {
        // Bounce through SSO; the landing page will be the redirect
        // target after login so the user can click the star again.
        login();
        return;
    }

    saving.value = true;
    try {
        const res = await apiFetch<{
            rating: number;
            aggregate: { rating_avg: number | null; rating_count: number };
        }>(`/api/fit/${props.fitId}/rating`, {
            method: "PUT",
            body: { rating: index },
        });
        emit("updated", {
            viewerRating: res.rating,
            ratingAvg: res.aggregate.rating_avg,
            ratingCount: res.aggregate.rating_count,
        });
    } catch {
        // Swallow — the star visual snaps back on next render if the
        // server refused. No toast yet; we can add a shared notifier
        // later if users ask for it.
    } finally {
        saving.value = false;
    }
}

async function onClearClick() {
    if (!props.interactive || saving.value) return;
    if (!isAuthenticated.value) return;
    if (props.viewerRating === null) return;

    saving.value = true;
    try {
        const res = await apiFetch<{
            deleted: boolean;
            aggregate: { rating_avg: number | null; rating_count: number };
        }>(`/api/fit/${props.fitId}/rating`, {
            method: "DELETE",
        });
        emit("updated", {
            viewerRating: null,
            ratingAvg: res.aggregate.rating_avg,
            ratingCount: res.aggregate.rating_count,
        });
    } catch {
        // ignore — same rationale as above
    } finally {
        saving.value = false;
    }
}
</script>

<template>
    <div class="flex items-center gap-2" @mouseleave="onMouseLeave">
        <div class="flex items-center gap-0.5">
            <button
                v-for="i in 5"
                :key="i"
                type="button"
                :disabled="!interactive || saving"
                :aria-label="interactive ? `Rate ${i} star${i === 1 ? '' : 's'}` : `Rating star ${i} of 5`"
                v-tooltip="interactive ? `Rate ${i} star${i === 1 ? '' : 's'}` : ''"
                class="flex items-center justify-center rounded transition-colors"
                :class="[
                    interactive
                        ? 'cursor-pointer hover:bg-amber-500/10'
                        : 'cursor-default',
                ]"
                @mouseenter="onMouseEnter(i)"
                @click="onClick(i)"
            >
                <Icon
                    :name="starIsOn(i) ? 'lucide:star' : 'lucide:star'"
                    :class="[
                        starBox,
                        starIsOn(i)
                            ? hovered !== null
                                ? 'text-amber-300 fill-amber-300'
                                : viewerRating !== null
                                    ? 'text-blue-400 fill-blue-400'
                                    : 'text-amber-400 fill-amber-400'
                            : 'text-gray-700',
                    ]"
                />
            </button>
        </div>

        <!-- Aggregate number + vote count -->
        <div v-if="ratingCount > 0" class="text-fine tabular-nums text-gray-500">
            <span class="text-gray-300 font-medium">{{ ratingAvg?.toFixed(1) ?? "–" }}</span>
            <span class="text-gray-600 ml-1">({{ ratingCount }})</span>
        </div>
        <div v-else class="text-fine text-gray-600">unrated</div>

        <!-- Viewer's own rating affordance / clear button -->
        <button
            v-if="interactive && viewerRating !== null"
            type="button"
            aria-label="Clear your rating"
            v-tooltip="'Clear your rating'"
            class="flex items-center justify-center w-5 h-5 rounded-md text-gray-600 hover:text-red-400 hover:bg-red-500/10 transition-colors"
            @click="onClearClick"
        >
            <Icon name="lucide:x" class="w-3 h-3" />
        </button>
        <span v-else-if="interactive && !isAuthenticated" class="text-fine text-gray-600">
            (log in to rate)
        </span>
    </div>
</template>
