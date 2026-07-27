/**
 * Holds the two fit states the UI juggles:
 *
 *   - `currentFit` is the committed fit — what the user actually built.
 *     This is what gets saved, shared, and exported.
 *   - `previewFit` is a transient overlay set while the user hovers a
 *     module in the hardware listing. When non-null, the ring and stats
 *     panel paint the preview instead of the committed fit, so the user
 *     can see the impact of "what if I fit this" before committing.
 *
 * `fit` is the derived "whichever is active" value. All display-only
 * components should bind to `fit`. Only the fit manager's mutators touch
 * `currentFit` directly; only hover-handlers touch `previewFit`.
 *
 * Backed by Nuxt's `useState` so multiple components on the same page
 * (ship picker, ring, stats panel, save button) all reactively share
 * the same underlying state without needing a provider component.
 */

import type { Fit } from "./fit/types";

export function useCurrentFit() {
    const currentFit = useState<Fit | null>("fit:current", () => null);
    const previewFit = useState<Fit | null>("fit:preview", () => null);
    /**
     * The server-side fit_id of the currently loaded fit, if any.
     * Non-null means the fit was either loaded via /api/fit/:id or
     * successfully saved via POST — the Save button then becomes an
     * Update (PATCH). Cleared on new-fit, paste, and encoded-link
     * loads so those all start as fresh unsaved drafts.
     */
    const currentFitId = useState<string | null>("fit:currentId", () => null);
    /**
     * Owner character_id of the loaded fit, when applicable. Used by
     * FitNavbar to switch into "viewer" mode (Fork instead of Update,
     * no Rename/Paste/New) when the authed character isn't the owner.
     * Cleared alongside currentFitId for fresh drafts.
     */
    const currentFitOwnerId = useState<number | null>(
        "fit:currentOwnerId",
        () => null,
    );

    const fit = computed<Fit | null>(() => previewFit.value ?? currentFit.value);
    const isPreview = computed(() => previewFit.value !== null);
    const isSaved = computed(() => currentFitId.value !== null);

    // markRaw keeps Vue from wrapping the Fit object in a deep reactive
    // Proxy. Every mutator clones the fit before modifying, so the
    // reactivity we need is "identity changed" (the ref is still
    // reactive) — not "some nested field changed" (Vue traversing a
    // fresh Proxy on every mutation is pure overhead and the
    // computeds that read `fit.modules.find(...)` only care about the
    // current snapshot anyway).
    //
    // Crucially, setFit does NOT touch `currentFitId`. Every mutator in
    // useFitManager calls setFit with a cloned-and-edited snapshot, and
    // we want those edits to be savable as updates to the same row. The
    // saved-id lifecycle is owned explicitly by loadFreshFit (clears it,
    // for external imports + new-fit flows), setSavedFit (sets it, for
    // /api/fit/:id loads), and markSaved (sets it, after POST succeeds).
    function setFit(next: Fit | null) {
        currentFit.value = next ? markRaw(next) : null;
        previewFit.value = null;
    }

    /**
     * Drop the editor into "fresh draft" state: replace (or clear) the
     * current fit AND clear `currentFitId` + `currentFitOwnerId` so
     * the next save creates a new row instead of updating whichever
     * fit used to be loaded.
     *
     * Used by:
     *   - useFitLoader.loadFromEncoded / loadFromEft / loadFit
     *   - useFitManager.createNewFit
     *   - /fits/create onMounted, when the viewer arrived from /fit/:id
     */
    function loadFreshFit(next: Fit | null) {
        currentFit.value = next ? markRaw(next) : null;
        previewFit.value = null;
        currentFitId.value = null;
        currentFitOwnerId.value = null;
    }

    /**
     * Load a fit that already exists on the server. Binds the fit to
     * its fit_id AND owner_character_id so the next save becomes an
     * in-place update (for owners) or a Fork (for non-owners), and so
     * FitNavbar can lock down editor controls in viewer mode.
     */
    function setSavedFit(next: Fit, fitId: string, ownerCharacterId: number) {
        currentFit.value = markRaw(next);
        previewFit.value = null;
        currentFitId.value = fitId;
        currentFitOwnerId.value = ownerCharacterId;
    }

    /**
     * Flip the current draft from "unsaved" to "saved" without
     * touching the fit contents. Called after a successful POST so
     * subsequent edits update instead of creating duplicates. The
     * caller passes their own character id as the owner — POST always
     * creates a row owned by the authed user.
     */
    function markSaved(fitId: string, ownerCharacterId: number) {
        currentFitId.value = fitId;
        currentFitOwnerId.value = ownerCharacterId;
    }

    function setPreview(next: Fit | null) {
        previewFit.value = next ? markRaw(next) : null;
    }

    function clearPreview() {
        previewFit.value = null;
    }

    return {
        currentFit,
        previewFit,
        currentFitId,
        currentFitOwnerId,
        fit,
        isPreview,
        isSaved,
        setFit,
        loadFreshFit,
        setSavedFit,
        markSaved,
        setPreview,
        clearPreview,
    };
}
