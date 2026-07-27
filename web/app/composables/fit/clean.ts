/**
 * Normalises an imported `Fit` — collapses duplicated drone groups
 * and cargo entries into one per type. Ported from
 * @eveshipfit/react's CleanImportFit hook.
 *
 * Used on every import path (v3, EFT, killmail, paste) so the UI
 * state stays sane regardless of what the source file looked like.
 */

import type { Fit, FitDrone, FitCargo } from "./types";

export function cleanFit(fit: Fit): Fit {
    // Accumulate duplicated drones by type id. Each drone group
    // tracks Active + Passive counts, so we just sum both.
    const dronesByType = new Map<number, FitDrone>();
    for (const drone of fit.drones) {
        const existing = dronesByType.get(drone.typeId);
        if (existing) {
            existing.states.Active += drone.states.Active;
            existing.states.Passive += drone.states.Passive;
        } else {
            dronesByType.set(drone.typeId, {
                typeId: drone.typeId,
                states: {
                    Active: drone.states.Active,
                    Passive: drone.states.Passive,
                },
            });
        }
    }

    const cargoByType = new Map<number, FitCargo>();
    for (const cargo of fit.cargo) {
        const existing = cargoByType.get(cargo.typeId);
        if (existing) {
            existing.quantity += cargo.quantity;
        } else {
            cargoByType.set(cargo.typeId, {
                typeId: cargo.typeId,
                quantity: cargo.quantity,
            });
        }
    }

    return {
        name: fit.name,
        description: fit.description,
        shipTypeId: fit.shipTypeId,
        modules: fit.modules,
        drones: Array.from(dronesByType.values()),
        cargo: Array.from(cargoByType.values()),
    };
}
