<script setup lang="ts">
/**
 * Top-of-page navbar for the fitting view. Modeled after the
 * `/kill/:id` KillNavbar — thin border-b strip, left-aligned
 * action buttons, right-side status indicators.
 *
 * Three modes via the `mode` prop:
 *
 *   - 'create' (default, used on /fits/create) — full action set:
 *     Paste, Copy EFT, Share Link, Save/Update, Rename, New
 *   - 'edit'   (used on /fit/:id when viewer owns the fit) — same as
 *     'create' but Paste is hidden (Paste is a create-page-only
 *     affordance; once a fit has an identity, paste means "discard
 *     this and load something else", which is what New is for)
 *   - 'view'   (used on /fit/:id when viewer ISN'T the owner) — only
 *     Copy EFT, Share Link, and Fork. Fork clones the current fit
 *     into a fresh /fits/create draft via the same `?fit=v3:...`
 *     query-param hand-off the share-link button uses, so the viewer
 *     gets an editable copy without ever touching the original row.
 *     Save and Rename are absent because non-owners can't update or
 *     rename someone else's saved fit.
 *
 * Right-side status:
 *   - Fit name display
 *   - "All Skills L5" skill profile indicator (clickable later
 *     when we add ESI skill import)
 *   - Loading spinner while calculateFit runs
 */

import { encodeFitV3 } from "~/composables/fit/encode";
import { exportFitToEft } from "~/composables/fit/eft";
import SaveFitModal from "./SaveFitModal.vue";

const props = withDefaults(
    defineProps<{
        mode?: "create" | "edit" | "view";
    }>(),
    {
        mode: "create",
    },
);

const { currentFit, isSaved } = useCurrentFit();
const { isLoading } = useFitStatistics();
const { sde } = useEveData();
const fitManager = useFitManager();
const loader = useFitLoader();
const { isAuthenticated, login } = useAuth();
const {
    total: fitIskTotal,
    isLoading: isPriceLoading,
    hasPrice: hasFitPrice,
} = useFitCost();

const showPaste = computed(() => props.mode === "create");
const showOwnerActions = computed(() => props.mode !== "view");
const showFork = computed(() => props.mode === "view");

// Destructure so the refs are top-level in setup — that way the
// template can reference them unwrapped without `.value`.
const { readText: readClipboard } = useClipboardFeedback();
const { copied: eftCopied, copy: copyEft } = useClipboardFeedback();
const { copied: linkCopied, copy: copyLink } = useClipboardFeedback();

const pasteStatus = ref<"idle" | "success" | "error">("idle");
let pasteTimer: ReturnType<typeof setTimeout> | null = null;

function flashPaste(state: "success" | "error") {
    pasteStatus.value = state;
    if (pasteTimer) clearTimeout(pasteTimer);
    pasteTimer = setTimeout(() => {
        pasteStatus.value = "idle";
    }, 1800);
}

const saveModalOpen = ref(false);
const saveFlash = ref(false);
let saveFlashTimer: ReturnType<typeof setTimeout> | null = null;

/**
 * Opens the save modal — or bounces unauthenticated viewers through
 * EVE SSO first. Before the redirect we encode the current fit into
 * the URL's ?fit= query param so useAuth().login() reads the full
 * path (including the encoded fit) as the redirect target. After SSO
 * the editor's onMounted hook sees the param and rehydrates the
 * draft, so the round-trip is transparent to the user.
 */
async function onSaveClick() {
    if (!currentFit.value) return;
    if (!isAuthenticated.value) {
        const router = useRouter();
        const route = useRoute();
        const encoded = await encodeFitV3(currentFit.value);
        await router.replace({
            path: route.path,
            query: { ...route.query, fit: encoded },
        });
        login();
        return;
    }
    saveModalOpen.value = true;
}

function onSaved() {
    saveFlash.value = true;
    if (saveFlashTimer) clearTimeout(saveFlashTimer);
    saveFlashTimer = setTimeout(() => {
        saveFlash.value = false;
    }, 2000);
}

/**
 * Fork = "give me an editable copy of this fit". Encodes the current
 * fit as a v3 share string and navigates to /fits/create with the
 * payload in the URL — the create page's onMounted hook picks up the
 * `?fit=` query param via useFitLoader.loadFromEncoded, which calls
 * loadFreshFit and clears any saved-id binding. The result is the
 * viewer is now editing their own draft, untethered from the original
 * row, free to save (which will create a new row owned by them).
 */
async function onForkClick() {
    if (!currentFit.value) return;
    const encoded = await encodeFitV3(currentFit.value);
    await navigateTo(`/fits/create?fit=${encoded}`);
}

async function onPasteClick() {
    const text = await readClipboard();
    if (!text) return flashPaste("error");
    const result = await loader.loadFromUnknown(text);
    flashPaste(result ? "success" : "error");
}

async function onCopyEftClick() {
    if (!currentFit.value || !sde.value) return;
    const eft = exportFitToEft(currentFit.value, sde.value);
    if (!eft) return;
    await copyEft(eft);
}

async function onCopyLinkClick() {
    if (!currentFit.value) return;
    const encoded = await encodeFitV3(currentFit.value);
    const origin =
        typeof window !== "undefined" ? window.location.origin : "https://eve-kill.com";
    const link = `${origin}${window.location.pathname}?fit=${encoded}`;
    await copyLink(link);
}

function onRenameClick() {
    if (!currentFit.value) return;
    const next = window.prompt("Rename fit", currentFit.value.name);
    if (next && next.trim()) fitManager.setName(next.trim());
}

function onNewFitClick() {
    // Confirm before nuking work in progress. Use the browser
    // confirm() for now — a prettier modal can land with the
    // rest of the ship-picker UI.
    if (
        currentFit.value &&
        currentFit.value.modules.length > 0 &&
        !window.confirm("Discard the current fit?")
    ) {
        return;
    }
    // Reset the hull too — clicking New should always give a blank
    // editor and a clean URL. Keeping the old shipTypeId meant the
    // sidebar stayed on Hardware and the URL kept the previous
    // encoded fit, so "New" felt like a no-op when the user was
    // already on an empty draft.
    fitManager.createNewFit(0, "New Fit");
}

// SSR + pre-mount: act as if no fit is loaded yet. This keeps every
// :disabled="!hasFit" attribute and v-if="currentFit" branch identical
// across the server and the client's first hydration pass — the real
// fit only gets loaded inside the page's onMounted hook (it depends on
// ?fit= decoding and/or useState that hasn't settled yet). Once we're
// mounted, hasFit tracks the real ref.
const isMounted = ref(false);
onMounted(() => { isMounted.value = true; });
const hasFit = computed(() => isMounted.value && currentFit.value !== null);
</script>

<template>
    <div class="flex items-center justify-between gap-2 mb-4 py-2 border-b border-white/[0.08]">
        <!-- Left: action buttons -->
        <div class="flex flex-wrap items-center gap-0.5">
            <button
                v-if="showPaste"
                type="button"
                class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors"
                @click="onPasteClick"
            >
                <Icon name="lucide:clipboard-paste" class="w-3.5 h-3.5" />
                <span>Paste</span>
                <span v-if="pasteStatus === 'success'" class="text-green-400 text-fine ml-1">✓</span>
                <span v-else-if="pasteStatus === 'error'" class="text-red-400 text-fine ml-1">×</span>
            </button>

            <button
                type="button"
                :disabled="!hasFit"
                class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors disabled:opacity-40 disabled:hover:text-gray-500 disabled:hover:bg-transparent"
                @click="onCopyEftClick"
            >
                <Icon name="lucide:copy" class="w-3.5 h-3.5" />
                <span>{{ eftCopied ? "Copied EFT" : "Copy EFT" }}</span>
            </button>

            <button
                type="button"
                :disabled="!hasFit"
                class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors disabled:opacity-40 disabled:hover:text-gray-500 disabled:hover:bg-transparent"
                @click="onCopyLinkClick"
            >
                <Icon name="lucide:link" class="w-3.5 h-3.5" />
                <span>{{ linkCopied ? "Link Copied" : "Share Link" }}</span>
            </button>

            <!-- Save / Update — owners + create page only -->
            <button
                v-if="showOwnerActions"
                type="button"
                :disabled="!hasFit"
                class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-blue-400 bg-blue-500/[0.08] border border-blue-500/20 hover:bg-blue-500/20 hover:border-blue-500/40 transition-colors disabled:opacity-40 disabled:hover:bg-blue-500/[0.08] disabled:hover:border-blue-500/20"
                @click="onSaveClick"
            >
                <Icon
                    :name="saveFlash ? 'lucide:check' : 'lucide:save'"
                    class="w-3.5 h-3.5"
                    :class="{ 'text-green-400': saveFlash }"
                />
                <span :class="{ 'text-green-400': saveFlash }">
                    {{ saveFlash ? "Saved" : isSaved ? "Update" : "Save" }}
                </span>
            </button>

            <!-- Fork — non-owners on /fit/:id -->
            <button
                v-if="showFork"
                type="button"
                :disabled="!hasFit"
                class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-amber-300 bg-amber-500/[0.08] border border-amber-500/20 hover:bg-amber-500/20 hover:border-amber-500/40 transition-colors disabled:opacity-40 disabled:hover:bg-amber-500/[0.08] disabled:hover:border-amber-500/20"
                @click="onForkClick"
                v-tooltip="'Open an editable copy of this fit on the create page'"
            >
                <Icon name="lucide:git-fork" class="w-3.5 h-3.5" />
                <span>Fork</span>
            </button>

            <button
                v-if="showOwnerActions"
                type="button"
                :disabled="!hasFit"
                class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors disabled:opacity-40 disabled:hover:text-gray-500 disabled:hover:bg-transparent"
                @click="onRenameClick"
            >
                <Icon name="lucide:pencil" class="w-3.5 h-3.5" />
                <span>Rename</span>
            </button>

            <button
                v-if="showOwnerActions"
                type="button"
                :disabled="!hasFit"
                class="flex items-center gap-1 px-2 py-1 rounded-md text-sm text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors disabled:opacity-40 disabled:hover:text-gray-500 disabled:hover:bg-transparent"
                @click="onNewFitClick"
            >
                <Icon name="lucide:plus" class="w-3.5 h-3.5" />
                <span>New</span>
            </button>
        </div>

        <!-- Right: fit name + ISK total + skill profile + loading indicator -->
        <div class="flex items-center gap-3 flex-shrink-0">
            <span v-if="isMounted && currentFit" class="text-xs text-gray-400 max-w-[200px] truncate">
                {{ currentFit.name }}
            </span>
            <ClientOnly>
                <span
                    v-if="currentFit"
                    class="flex items-center gap-1 px-2 py-1 rounded-md text-fine font-bold uppercase tracking-[0.12em] bg-amber-500/[0.08] text-amber-300/90 tabular-nums"
                    v-tooltip="hasFitPrice ? `${fitIskTotal.toLocaleString('en-US')} ISK` : 'No price data yet'"
                >
                    <Icon
                        :name="isPriceLoading ? 'lucide:loader' : 'lucide:coins'"
                        class="w-3 h-3"
                        :class="{ 'animate-spin': isPriceLoading }"
                    />
                    {{ hasFitPrice ? `${formatIsk(fitIskTotal)} ISK` : '— ISK' }}
                </span>
            </ClientOnly>
            <span class="flex items-center gap-1 px-2 py-1 rounded-md text-fine font-bold uppercase tracking-[0.12em] bg-blue-500/[0.08] text-blue-400/80">
                <Icon name="lucide:zap" class="w-3 h-3" />
                All Skills L5
            </span>
            <span
                v-if="isLoading"
                class="flex items-center gap-1 text-fine text-gray-500"
            >
                <Icon name="lucide:loader" class="w-3 h-3 animate-spin" />
                calculating
            </span>
        </div>

        <SaveFitModal v-model="saveModalOpen" @saved="onSaved" />
    </div>
</template>
