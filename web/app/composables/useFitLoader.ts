/**
 * High-level fit loader — takes one of several input formats
 * (ESF v3 encoded string, EFT text, raw Fit object, fit ID from
 * our API, ?fit= URL query) and writes the result into the shared
 * fit state.
 *
 * Callers split into two camps:
 *
 *   - "Fresh draft" sources (encoded string, EFT text, raw Fit,
 *     createNewFit): these go through `loadFreshFit` which clears
 *     `currentFitId`, so the next save writes a brand-new row.
 *   - "Persisted fit" sources (loadFromFitId hitting /api/fit/:id):
 *     these go through `setSavedFit` which binds the fit to its
 *     server-side id, so the next save updates the row in place.
 *
 * Used by:
 *
 *   - The FitNavbar's "paste from clipboard" button
 *   - The /fit/:id page that loads a user_fitting from our API
 *   - The URL query watcher on /fits/create that auto-loads a
 *     shared fit on page mount
 *   - The "Load in Editor" button on /item/:id/fittings
 *   - Future integrations (killmail → fit, ESI import, etc.)
 */

import type { Fit } from "./fit/types";
import { decodeEsfEncoded } from "./fit/decode";
import { importFitFromEft } from "./fit/eft";

/** EVE SDE groupID for all fittable propulsion modules (AB + MWD). */
const PROPULSION_MODULE_GROUP_ID = 46;

export function useFitLoader() {
    const { loadFreshFit, setSavedFit } = useCurrentFit();
    const { sde } = useEveData();

    /**
     * EVE only allows one propulsion module (MWD / Afterburner) active
     * at a time. Imported fits may have multiple propmods set to Active
     * (e.g. v3 URL shares, EFT pastes, API loads). This normalises the
     * fit so only the first propmod stays Active and the rest downgrade
     * to Online. Mutates the fit in place for convenience — callers
     * always pass a freshly constructed object.
     */
    function enforcePropmodExclusivity(fit: Fit): void {
        const sdeData = sde.value;
        if (!sdeData) return;

        let hasActivePropmod = false;
        for (const m of fit.modules) {
            if (m.state !== "Active" && m.state !== "Overload") continue;
            const type = sdeData.types.get(m.typeId);
            if (type?.groupID !== PROPULSION_MODULE_GROUP_ID) continue;
            if (hasActivePropmod) {
                m.state = "Online";
            } else {
                hasActivePropmod = true;
            }
        }
    }

    /** Load a raw Fit object directly — use this from API callers. */
    function loadFit(fit: Fit) {
        enforcePropmodExclusivity(fit);
        loadFreshFit(fit);
    }

    /**
     * Load from an ESF v3 encoded string (`v3:xxx` or a full URL
     * containing `?fit=v3:xxx`). Returns true on success. The result
     * is treated as a fresh draft — sharers and paste flows shouldn't
     * silently inherit a previous fit's server id.
     */
    async function loadFromEncoded(encoded: string): Promise<boolean> {
        const fit = await decodeEsfEncoded(encoded);
        if (!fit) return false;
        enforcePropmodExclusivity(fit);
        loadFreshFit(fit);
        return true;
    }

    /**
     * Load from an EFT clipboard string. Needs the SDE to be
     * loaded — returns false if it isn't yet.
     */
    function loadFromEft(eft: string): boolean {
        const sdeData = sde.value;
        if (!sdeData) return false;
        const fit = importFitFromEft(eft, sdeData);
        if (!fit) return false;
        enforcePropmodExclusivity(fit);
        loadFreshFit(fit);
        return true;
    }

    /**
     * Fetch a user_fitting from /api/fit/:id and load it. The
     * server shape uses snake_case (`fit_id`, `ship_type_id`,
     * `owner_character_id`, …) — convert to the UI Fit shape.
     *
     * Errors propagate — the server distinguishes 404 (not found),
     * 403 (private / corp-only), and 5xx (transient). The page-level
     * caller decides how to surface each; we don't swallow any of it.
     *
     * Returns the rating metadata alongside the load so the caller
     * can paint the star widget without a second round trip. Everything
     * the rating widget needs (aggregate + viewer's own rating) is
     * already in the /api/fit/:id response.
     */
    async function loadFromFitId(fitId: string): Promise<{
        ratingAvg: number | null;
        ratingCount: number;
        viewerRating: number | null;
        ownerCharacterId: number;
    }> {
        interface ApiFitRow {
            fit_id: string;
            owner_character_id: number;
            ship_type_id: number;
            name: string;
            description: string | null;
            rating_avg: number | null;
            rating_count: number;
        }
        interface ApiFitItem {
            slot_group: number;
            ordinal: number;
            type_id: number;
            state: number;
            charge_type_id: number | null;
            quantity?: number;
        }
        const res = await apiFetch<{
            fit: ApiFitRow;
            items: ApiFitItem[];
            viewer_rating: number | null;
        }>(`/api/fit/${fitId}`);
        // setSavedFit binds the loaded fit to its server fit_id +
        // owner_character_id so the navbar's Save button can switch
        // between Update (owner) and Fork (non-owner), and so any
        // edits a non-owner makes go to a fresh row instead of
        // 403'ing on PATCH.
        const loadedFit = apiFitToUi(res.fit, res.items);
        enforcePropmodExclusivity(loadedFit);
        setSavedFit(
            loadedFit,
            res.fit.fit_id,
            res.fit.owner_character_id,
        );
        return {
            ratingAvg: res.fit.rating_avg,
            ratingCount: res.fit.rating_count,
            viewerRating: res.viewer_rating,
            ownerCharacterId: res.fit.owner_character_id,
        };
    }

    /**
     * Try to decode a pasted string as ESF v3 first, then fall
     * back to EFT. Returns the source format on success so
     * callers can show "imported as EFT" / "imported as ESF" UI.
     */
    async function loadFromUnknown(text: string): Promise<"esf" | "eft" | null> {
        const trimmed = text.trim();
        if (trimmed.startsWith("v3:") || trimmed.includes("?fit=v3:")) {
            if (await loadFromEncoded(trimmed)) return "esf";
        }
        if (trimmed.startsWith("[")) {
            if (loadFromEft(trimmed)) return "eft";
        }
        return null;
    }

    return {
        loadFit,
        loadFromEncoded,
        loadFromEft,
        loadFromFitId,
        loadFromUnknown,
    };
}

/**
 * Convert the flat (fit_row, items[]) shape the /api/fit/:id
 * endpoint returns into the UI-layer `Fit`. Three slot_group ranges
 * fan out into three different fields on the Fit object:
 *
 *   1..5  → modules (one row per slot, quantity always 1)
 *   6     → drones (one row per (typeId, state), quantity > 1 stacks)
 *   7     → cargo  (one row per typeId, quantity > 1 stacks)
 */
function apiFitToUi(
    row: {
        fit_id: string;
        ship_type_id: number;
        name: string;
        description: string | null;
    },
    items: Array<{
        slot_group: number;
        ordinal: number;
        type_id: number;
        state: number;
        charge_type_id: number | null;
        quantity?: number;
    }>,
): Fit {
    const SLOT_GROUP_TO_TYPE: Record<number, "High" | "Medium" | "Low" | "Rig" | "SubSystem"> = {
        1: "High",
        2: "Medium",
        3: "Low",
        4: "Rig",
        5: "SubSystem",
    };
    const STATE_TO_NAME: Record<number, "Passive" | "Online" | "Active" | "Overload"> = {
        0: "Passive",
        1: "Online",
        2: "Active",
        3: "Overload",
    };

    // Modules: slot_groups 1..5
    const modules = items
        .filter((i) => i.slot_group >= 1 && i.slot_group <= 5)
        .map((i) => ({
            typeId: i.type_id,
            // Backend is 0-based ordinal; UI is 1-based.
            slot: {
                type: SLOT_GROUP_TO_TYPE[i.slot_group] ?? "High",
                index: i.ordinal + 1,
            },
            state: STATE_TO_NAME[i.state] ?? "Active",
            charge: i.charge_type_id ? { typeId: i.charge_type_id } : undefined,
        }));

    // Drones: slot_group 6. Each row carries a count and a state. Merge
    // by typeId so the UI's grouped representation (per-type, with both
    // states) round-trips cleanly across save → load.
    const dronesByType = new Map<
        number,
        { typeId: number; states: { Active: number; Passive: number } }
    >();
    for (const i of items) {
        if (i.slot_group !== 6) continue;
        const qty = i.quantity ?? 1;
        let entry = dronesByType.get(i.type_id);
        if (!entry) {
            entry = { typeId: i.type_id, states: { Active: 0, Passive: 0 } };
            dronesByType.set(i.type_id, entry);
        }
        // Active state code is 2; everything else folds back into Passive.
        // The save path only ever writes 0 or 2 for drone rows so this
        // covers it.
        if (i.state === 2) {
            entry.states.Active += qty;
        } else {
            entry.states.Passive += qty;
        }
    }
    const drones = Array.from(dronesByType.values());

    // Cargo: slot_group 7. One row per typeId, quantity = count.
    const cargoByType = new Map<number, { typeId: number; quantity: number }>();
    for (const i of items) {
        if (i.slot_group !== 7) continue;
        const qty = i.quantity ?? 1;
        const existing = cargoByType.get(i.type_id);
        if (existing) {
            existing.quantity += qty;
        } else {
            cargoByType.set(i.type_id, { typeId: i.type_id, quantity: qty });
        }
    }
    const cargo = Array.from(cargoByType.values());

    return {
        name: row.name,
        description: row.description ?? "",
        shipTypeId: row.ship_type_id,
        modules,
        drones,
        cargo,
    };
}
