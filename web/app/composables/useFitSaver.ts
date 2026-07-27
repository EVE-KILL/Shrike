/**
 * Fit save/update composable.
 *
 * Wraps the POST/PATCH side of /api/fit. The calling component (typically
 * the save modal in FitNavbar) hands us a fit snapshot + visibility and
 * we decide based on `currentFitId` whether to create a new row or
 * update the existing one in place.
 *
 * On success we hydrate `useCurrentFit` with the server response so the
 * navbar immediately flips into "saved" state (Save → Update) and the
 * fit carries its server `fit_id` for subsequent edits and the /fit/{id}
 * share URL.
 *
 * All state is component-local — no useState, no sharing across pages.
 * That's intentional: each save modal instance owns its own
 * saving/error flags, and there's never more than one saver active at
 * once on a page.
 */

import type { Fit } from "./fit/types";
import { fitToSaveBody } from "./fit/toServer";

interface ApiFitRow {
    fit_id: string;
    owner_character_id: number;
    ship_type_id: number;
    name: string;
    description: string | null;
    visibility: number;
    created_at: string;
    updated_at: string;
}

interface SaveSuccess {
    ok: true;
    fitId: string;
}

interface SaveFailure {
    ok: false;
    error: string;
}

export type SaveResult = SaveSuccess | SaveFailure;

export function useFitSaver() {
    const { currentFitId, markSaved } = useCurrentFit();

    const saving = ref(false);
    const error = ref<string | null>(null);

    /**
     * Persist the supplied fit. Creates a new row when currentFitId is
     * null, otherwise updates the existing row. Returns a discriminated
     * result so callers can branch without a try/catch.
     */
    async function save(fit: Fit, visibility: number): Promise<SaveResult> {
        saving.value = true;
        error.value = null;

        const body = fitToSaveBody(fit, visibility);

        try {
            if (currentFitId.value) {
                // Update existing row. PATCH accepts the same item shape
                // as POST but wraps it in a partial; sending every field
                // simplifies the client.
                await apiFetch<{ fit: ApiFitRow }>(
                    `/api/fit/${currentFitId.value}`,
                    {
                        method: "PATCH",
                        body: {
                            name: body.name,
                            description: body.description,
                            visibility: body.visibility,
                            items: body.items,
                        },
                    },
                );
                return { ok: true, fitId: currentFitId.value };
            }

            // Create new fit. The POST response carries the new fit_id
            // and the owner_character_id (the authed character) so we
            // can flip the navbar straight into "owner of a saved fit"
            // mode without a follow-up GET.
            const res = await apiFetch<ApiFitRow>("/api/fit", {
                method: "POST",
                body,
            });
            markSaved(res.fit_id, res.owner_character_id);
            return { ok: true, fitId: res.fit_id };
        } catch (err: unknown) {
            const message = extractFetchError(err, "Failed to save fit");
            error.value = message;
            return { ok: false, error: message };
        } finally {
            saving.value = false;
        }
    }

    return {
        saving: readonly(saving),
        error: readonly(error),
        save,
    };
}
