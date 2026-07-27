/**
 * Helper for taking a killmail-derived fit (the shape returned by
 * /api/item/:id/fittings, /api/fits/flavors-of-the-week, and
 * /api/fits/top-alliance-doctrines) and converting it into a UI `Fit`
 * ready to encode + load into the editor.
 *
 * Used by:
 *   - ItemFittings.vue's "Load in Editor" button
 *   - The flavors of the week section on /fits
 *   - The top alliance doctrines section on /fits
 *
 * Slot indexes are renumbered 1..N per group instead of preserving the
 * original ordinal, because killmail records may have gaps (destroyed
 * modules). Drones come in as their own array since the killmail
 * extractor splits them out at slot_group 6 with per-type quantities.
 * Charges (if any) come along on each module row as charge_type_id.
 */

import { encodeFitV3 } from "./encode";
import type { Fit, FitSlotType } from "./types";

export interface KillmailModule {
    slot_group: number;
    ordinal: number;
    type_id: number;
    name?: string | null;
    /** Loaded ammo / script for this module slot, if any. */
    charge_type_id?: number | null;
}

export interface KillmailDrone {
    type_id: number;
    name?: string | null;
    quantity: number;
}

/** Killmail slot_group int → UI FitSlotType. Slot 6 (drones) is handled separately. */
const SLOT_TYPE_MAP: Record<number, FitSlotType> = {
    1: "High",
    2: "Medium",
    3: "Low",
    4: "Rig",
    5: "SubSystem",
};
const SLOT_ORDER = [1, 2, 3, 4, 5] as const;

/** Build a UI Fit from killmail-derived (ship, modules, drones) data. */
export function killmailFitToUiFit(args: {
    shipTypeId: number;
    modules: readonly KillmailModule[];
    drones?: readonly KillmailDrone[];
    name?: string;
    description?: string;
}): Fit {
    const grouped: Record<number, KillmailModule[]> = {};
    for (const m of args.modules) {
        if (!grouped[m.slot_group]) grouped[m.slot_group] = [];
        grouped[m.slot_group]!.push(m);
    }
    const modules = SLOT_ORDER.flatMap((slotGroup) => {
        const items = grouped[slotGroup] ?? [];
        return items.map((mod, i) => ({
            typeId: mod.type_id,
            slot: {
                type: SLOT_TYPE_MAP[slotGroup]!,
                index: i + 1,
            },
            state: "Active" as const,
            charge:
                mod.charge_type_id !== null && mod.charge_type_id !== undefined
                    ? { typeId: mod.charge_type_id }
                    : undefined,
        }));
    });

    // Drones — quantity stays as a count split across Active vs Passive.
    // Killmail data doesn't tell us which were deployed, so we greedily
    // fill Active up to the universal 5-drone count cap (carriers and
    // supercarriers get more, but they field fighters, not regular drones,
    // so the cap is fine here) and spill the remainder to Passive. The
    // editor's bandwidth check still runs on top and can demote more
    // drones to Passive if bandwidth doesn't cover the 5.
    const MAX_ACTIVE_DRONES = 5;
    let activeBudget = MAX_ACTIVE_DRONES;
    const drones = (args.drones ?? []).map((d) => {
        const active = Math.min(d.quantity, activeBudget);
        activeBudget -= active;
        return {
            typeId: d.type_id,
            states: { Active: active, Passive: d.quantity - active },
        };
    });

    return {
        name: args.name ?? "Community Fit",
        description: args.description ?? "",
        shipTypeId: args.shipTypeId,
        modules,
        drones,
        cargo: [],
    };
}

/**
 * Convenience: build a Fit, encode it as v3, and return the
 * `/fits/create?fit=...` URL ready to pass to navigateTo. Async because
 * the gzip+base64 inside encodeFitV3 is async.
 */
export async function killmailFitToEditorUrl(args: {
    shipTypeId: number;
    modules: readonly KillmailModule[];
    drones?: readonly KillmailDrone[];
    name?: string;
    description?: string;
}): Promise<string> {
    const fit = killmailFitToUiFit(args);
    const encoded = await encodeFitV3(fit);
    return `/fits/create?fit=${encoded}`;
}
