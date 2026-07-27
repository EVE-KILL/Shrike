/**
 * Converts a UI `Fit` into the `EsfFit` shape that @evekill/dogma's
 * calculateFit() expects.
 *
 * Three transformations happen here:
 *   1. camelCase → snake_case key renames (shipTypeId → ship_type_id, etc.)
 *   2. Module state "Preview" collapses to "Active" — the engine doesn't
 *      know about preview and the preview tint is painted in CSS.
 *   3. Drone groups flatten out. A FitDrone with states={Passive:1, Active:3}
 *      becomes four separate EsfDrone entries (1 passive + 3 active).
 *
 * Cargo is intentionally not passed through — it has no effect on the
 * dogma calculation. Capacity usage is derived separately from the SDE
 * types lookup when we wire up the cargo bay widget.
 */

import type { EsfDrone, EsfFit, EsfModule, EsfStateName } from "@evekill/dogma";
import type { Fit } from "./types";

export function fitToEngine(fit: Fit): EsfFit {
    const modules: EsfModule[] = fit.modules.map((m) => ({
        type_id: m.typeId,
        slot: { type: m.slot.type, index: m.slot.index },
        state: (m.state === "Preview" ? "Active" : m.state) as EsfStateName,
        charge: m.charge ? { type_id: m.charge.typeId } : null,
    }));

    const drones: EsfDrone[] = [];
    for (const d of fit.drones) {
        for (let i = 0; i < d.states.Active; i++) {
            drones.push({ type_id: d.typeId, state: "Active" });
        }
        for (let i = 0; i < d.states.Passive; i++) {
            drones.push({ type_id: d.typeId, state: "Passive" });
        }
    }

    return {
        ship_type_id: fit.shipTypeId,
        modules,
        drones,
    };
}
