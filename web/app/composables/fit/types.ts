/**
 * UI-layer fit types — what the Vue components pass around.
 *
 * These mirror the shape used by @eveshipfit/react (camelCase, grouped
 * drones) rather than the snake_case shape that @evekill/dogma's engine
 * boundary expects. Conversion happens in `toEngine.ts` at the moment
 * we call `calculateFit`.
 *
 * The UI talks in terms of "how humans think about a fit" — named, with
 * a cargo hold, with drones grouped by type and per-state counts — and
 * that shape round-trips cleanly through save/load/import/export. The
 * engine talks in terms of a flat module+drone list. Keeping the two
 * separated means the engine can change its wire format without touching
 * the UI and vice-versa.
 */

/** Module state. UI never shows "Preview" — that's a transient slot flag. */
export type FitState = "Passive" | "Online" | "Active" | "Overload";

/** Slot types a module can occupy in the UI (no engine-internal pseudo-slots). */
export type FitSlotType = "High" | "Medium" | "Low" | "Rig" | "SubSystem";

export interface FitSlot {
    type: FitSlotType;
    index: number;
}

export interface FitCharge {
    typeId: number;
}

export interface FitModule {
    typeId: number;
    slot: FitSlot;
    /**
     * "Preview" is a transient UI state used when hovering a module in the
     * hardware listing. The statistics layer treats it as "Active" for the
     * calculation but the slot paints it with a sepia tint.
     */
    state: FitState | "Preview";
    charge?: FitCharge;
}

/**
 * Drones are grouped by type with per-state counts — matches how the drone
 * bay UI wants to render them (one row per type, split into active vs
 * passive). The engine receives this flattened: N active entries + M
 * passive entries.
 */
export interface FitDrone {
    typeId: number;
    states: {
        Passive: number;
        Active: number;
    };
}

export interface FitCargo {
    typeId: number;
    quantity: number;
}

export interface Fit {
    name: string;
    description: string;
    shipTypeId: number;
    modules: FitModule[];
    drones: FitDrone[];
    cargo: FitCargo[];
}

/**
 * Creates an empty fit for a given hull. Used by "new fit" flows in the
 * hull listing and the landing page "Create Fit" button.
 */
export function emptyFit(shipTypeId: number, name = "New Fit"): Fit {
    return {
        name,
        description: "",
        shipTypeId,
        modules: [],
        drones: [],
        cargo: [],
    };
}
