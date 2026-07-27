/**
 * UI `Fit` → POST /api/fit body converter.
 *
 * The server accepts a flat `items` array keyed by (slot_group, ordinal).
 * See frontend/server/utils/fitValidation.ts for the full schema and
 * packages/db/schema/UserFitting.ts for the column definitions.
 *
 * Mapping:
 *   Modules — slot.type → slot_group int (1..5), slot.index (1-based) →
 *             ordinal (0-based), state name → smallint, charge.typeId →
 *             charge_type_id, quantity always 1 (one slot, one module).
 *   Drones  — slot_group = 6. ONE row per (typeId, state) pair with
 *             quantity = count. So 5 Berserker IIs split 3 active / 2
 *             passive becomes two rows: one for the active group with
 *             quantity = 3, one for the passive group with quantity = 2.
 *             Originally we wrote N rows for N drones; the new schema
 *             carries a quantity column so the dense form is the right
 *             one going forward.
 *   Cargo   — slot_group = 7 (NEW). One row per typeId with quantity =
 *             count. State is always Passive (cargo doesn't run).
 *
 * Pure function — no Vue composable, no reactive state. Caller passes a
 * fit snapshot + visibility and gets back a request body ready to POST.
 */

import type { Fit } from "./types";

export interface SaveFitItem {
    slot_group: number;
    ordinal: number;
    type_id: number;
    state: number;
    charge_type_id: number | null;
    quantity: number;
}

export interface SaveFitBody {
    ship_type_id: number;
    name: string;
    description: string | null;
    visibility: number;
    items: SaveFitItem[];
}

/** UI slot type → server slot_group smallint. Matches UserFittingSlotGroup. */
const SLOT_TYPE_TO_GROUP = {
    High: 1,
    Medium: 2,
    Low: 3,
    Rig: 4,
    SubSystem: 5,
} as const;

/** UI state name → server state smallint. Matches UserFittingItemState.
 *  Preview collapses to Active so a user who hit Save mid-hover still
 *  persists a sensible baseline. */
const STATE_NAME_TO_INT = {
    Passive: 0,
    Online: 1,
    Active: 2,
    Overload: 3,
    Preview: 2,
} as const;

export function fitToSaveBody(fit: Fit, visibility: number): SaveFitBody {
    const items: SaveFitItem[] = [];

    // Modules — 1:1 row per module, quantity always 1. Slot index is
    // already unique within a slot type on the UI side (the fit manager
    // enforces that), so just convert to 0-based ordinal and pass through.
    for (const m of fit.modules) {
        items.push({
            slot_group: SLOT_TYPE_TO_GROUP[m.slot.type],
            ordinal: m.slot.index - 1,
            type_id: m.typeId,
            state: STATE_NAME_TO_INT[m.state],
            charge_type_id: m.charge?.typeId ?? null,
            quantity: 1,
        });
    }

    // Drones — one row per (typeId, state). The two states for a single
    // drone type get distinct ordinals so the (slot_group, ordinal) PK
    // stays unique. Active drones get even ordinals, passives odd, just
    // for stable ordering — readers don't care about the exact ordinal,
    // only the (typeId, state, quantity) tuple.
    let droneOrdinal = 0;
    for (const d of fit.drones) {
        if (d.states.Active > 0) {
            items.push({
                slot_group: 6,
                ordinal: droneOrdinal++,
                type_id: d.typeId,
                state: 2, // Active
                charge_type_id: null,
                quantity: d.states.Active,
            });
        }
        if (d.states.Passive > 0) {
            items.push({
                slot_group: 6,
                ordinal: droneOrdinal++,
                type_id: d.typeId,
                state: 0, // Passive
                charge_type_id: null,
                quantity: d.states.Passive,
            });
        }
    }

    // Cargo — slot_group 7, one row per typeId with quantity = count.
    // Cargo never runs, so state is always Passive (0).
    let cargoOrdinal = 0;
    for (const c of fit.cargo) {
        if (c.quantity <= 0) continue;
        items.push({
            slot_group: 7,
            ordinal: cargoOrdinal++,
            type_id: c.typeId,
            state: 0,
            charge_type_id: null,
            quantity: c.quantity,
        });
    }

    return {
        ship_type_id: fit.shipTypeId,
        name: fit.name,
        description: fit.description.trim() === "" ? null : fit.description,
        visibility,
        items,
    };
}
