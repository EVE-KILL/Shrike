/**
 * EFT format import/export — the in-game clipboard fit format
 * that every EVE tool speaks. Ported from @eveshipfit/react's
 * ExportEft + ImportEft hooks.
 *
 * Pure functions that take a snapshot `SdeData` rather than
 * composables, so they can be called from navbar buttons without
 * hauling around Vue reactivity.
 *
 * EFT file format:
 *
 *   [ShipName, Fit Name]
 *   Low Slot Module
 *   Low Slot Module /offline
 *
 *   Med Slot Module
 *   Med Slot Module, Charge Name
 *
 *   High Slot Module
 *
 *   Rig
 *
 *   Subsystem
 *
 *
 *   Drone Name x5
 *
 *   Cargo x100
 *
 * Two blank lines separates modules from drones, one blank line
 * separates each section. `/offline` marks a module put offline.
 */

import type { SdeData } from "@evekill/dogma";
import type { Fit, FitCargo, FitDrone, FitModule, FitSlotType } from "./types";

// Hardcoded CCP effect/attribute IDs for slot-type resolution on
// EFT import. Stable assignments — they haven't changed in a
// decade and the React reference bakes them in too. If CCP ever
// renumbers, fall back to the effect name lookup in HardwareListing.
const EFFECT_SLOT_MAP: Record<number, FitSlotType | "DroneBay"> = {
    11: "Low",
    12: "High",
    13: "Medium",
    2663: "Rig",
    3772: "SubSystem",
};
const ATTRIBUTE_SLOT_MAP: Record<number, FitSlotType | "DroneBay"> = {
    1272: "DroneBay",
};

const SLOT_TYPE_TO_EFT: Record<FitSlotType, string> = {
    Low: "Low Slot",
    Medium: "Med Slot",
    High: "High Slot",
    Rig: "Rig Slot",
    SubSystem: "Subsystem Slot",
};

const SLOT_ORDER: FitSlotType[] = ["Low", "Medium", "High", "Rig", "SubSystem"];

// ========== Export ==========

/**
 * Convert a `Fit` to the in-game EFT clipboard format. Needs the
 * SDE to resolve type names. Returns null if the ship type isn't
 * in the SDE (shouldn't happen for a valid fit but guard anyway).
 *
 * Doesn't emit `[Empty Low Slot]` placeholders because we don't
 * know the hull's slot counts without a full stats calculation —
 * the in-game EVE client happily parses either format.
 */
export function exportFitToEft(fit: Fit, sde: SdeData): string | null {
    const shipType = sde.types.get(fit.shipTypeId);
    if (!shipType) return null;

    let eft = `[${shipType.name}, ${fit.name || "Fit"}]\n`;

    // Group modules by slot type, preserving the fit's internal
    // slot index order.
    for (const slotType of SLOT_ORDER) {
        const modules = fit.modules
            .filter((m) => m.slot.type === slotType)
            .sort((a, b) => a.slot.index - b.slot.index);

        if (modules.length === 0) {
            // Still emit the section separator so the drones marker
            // (double blank line) works out.
            eft += "\n";
            continue;
        }

        for (const m of modules) {
            eft += moduleLine(m, sde);
        }
        eft += "\n";
    }

    // The React lib emits an extra blank line here as the
    // "end-of-modules" marker — the importer uses two consecutive
    // blanks to know drone/cargo sections follow.
    eft += "\n";

    for (const drone of fit.drones) {
        const type = sde.types.get(drone.typeId);
        if (!type) continue;
        const count = drone.states.Active + drone.states.Passive;
        eft += `${type.name}${count > 1 ? ` x${count}` : ""}\n`;
    }
    eft += "\n";

    for (const cargo of fit.cargo) {
        const type = sde.types.get(cargo.typeId);
        if (!type) continue;
        eft += `${type.name} x${cargo.quantity}\n`;
    }

    return eft.trim() + "\n";
}

function moduleLine(module: FitModule, sde: SdeData): string {
    const type = sde.types.get(module.typeId);
    if (!type) return `[Empty ${SLOT_TYPE_TO_EFT[module.slot.type]}]\n`;

    let line = type.name as string;
    if (module.charge) {
        const chargeType = sde.types.get(module.charge.typeId);
        if (chargeType) line += `, ${chargeType.name}`;
    }
    if (module.state === "Passive") line += " /offline";
    return line + "\n";
}

// ========== Import ==========

/**
 * Parse an EFT clipboard string into a `Fit`. Returns null if the
 * first line isn't a valid `[ShipName, FitName]` header or the
 * ship can't be resolved in the SDE.
 *
 * Unknown modules/charges are silently skipped — matching the
 * React reference's "do the best you can" behavior for typos or
 * stale EFT exports. Mutated module metadata (`[<index>]` lines)
 * is recognized but not preserved; we just grab the base module.
 */
export function importFitFromEft(eft: string, sde: SdeData): Fit | null {
    const lines = eft.trim().split("\n");
    if (lines.length === 0) return null;
    const header = lines[0]!.trim();
    if (!header.startsWith("[") || !header.endsWith("]")) return null;

    const headerParts = header.slice(1, -1).split(",");
    const shipName = headerParts[0]!.trim();
    const fitName = headerParts.slice(1).join(",").trim() || "EFT Import";

    const shipTypeId = sde.typeNameToId.get(shipName);
    if (shipTypeId === undefined) return null;

    const fit: Fit = {
        shipTypeId,
        name: fitName,
        description: "",
        modules: [],
        drones: [],
        cargo: [],
    };

    // Per-slot-type running index. EFT has no explicit slot
    // indices — modules are positional within their section.
    const slotIndex: Record<FitSlotType, number> = {
        Low: 1,
        Medium: 1,
        High: 1,
        Rig: 1,
        SubSystem: 1,
    };

    let lastSlotType: FitSlotType | "DroneBay" | "CargoBay" | undefined;
    let blankCount = 0;
    let inCargoSection = false;

    for (let i = 1; i < lines.length; i++) {
        const line = lines[i]!;
        const trimmed = line.trim();

        if (trimmed === "") {
            blankCount++;
            // Two blanks in a row marks the end of modules.
            if (blankCount >= 2) inCargoSection = true;
            continue;
        }
        blankCount = 0;

        // `[Empty Low Slot]` → advance the slot index without
        // adding anything.
        if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
            if (
                lastSlotType !== undefined &&
                lastSlotType !== "DroneBay" &&
                lastSlotType !== "CargoBay"
            ) {
                slotIndex[lastSlotType]++;
            }
            continue;
        }

        // Lines starting with 2 spaces are indented mutated-module
        // metadata. Skip them — we don't preserve mutations.
        if (line.startsWith("  ")) continue;

        // Strip `/offline` suffix.
        let itemLine = trimmed;
        let isOffline = false;
        if (itemLine.endsWith("/offline")) {
            isOffline = true;
            itemLine = itemLine.slice(0, -"/offline".length).trim();
        }

        // Split off charge (after first comma).
        const commaIdx = itemLine.indexOf(",");
        const itemNameRaw = (commaIdx >= 0 ? itemLine.slice(0, commaIdx) : itemLine).trim();
        const chargeNameRaw = commaIdx >= 0 ? itemLine.slice(commaIdx + 1).trim() : "";

        // Split off quantity suffix `x123`.
        const xMatch = itemNameRaw.match(/^(.*)\s+x(\d+)$/);
        const itemName = xMatch ? xMatch[1]!.trim() : itemNameRaw;
        const itemCount = xMatch ? Number.parseInt(xMatch[2]!, 10) : 1;

        const itemTypeId = sde.typeNameToId.get(itemName);
        if (itemTypeId === undefined) continue;

        // Determine slot type from the module's dogma effects /
        // attributes. We fall back to CargoBay if we're past the
        // end-of-modules marker.
        let slotType = slotTypeForImport(itemTypeId, sde);
        if (inCargoSection && slotType !== "DroneBay") {
            slotType = "CargoBay";
        }
        if (slotType === undefined) continue;
        lastSlotType = slotType;

        const chargeTypeId = chargeNameRaw ? sde.typeNameToId.get(chargeNameRaw) : undefined;
        const charge = chargeTypeId ? { typeId: chargeTypeId } : undefined;

        if (slotType === "DroneBay") {
            const loadCount = Math.min(itemCount, Math.max(0, 5 - activeDroneCount(fit.drones)));
            fit.drones.push({
                typeId: itemTypeId,
                states: { Active: loadCount, Passive: itemCount - loadCount },
            });
            continue;
        }

        if (slotType === "CargoBay") {
            fit.cargo.push({ typeId: itemTypeId, quantity: itemCount });
            continue;
        }

        const moduleSlot: FitSlotType = slotType;
        fit.modules.push({
            slot: { type: moduleSlot, index: slotIndex[moduleSlot] },
            typeId: itemTypeId,
            state: isOffline ? "Passive" : "Active",
            charge,
        });
        slotIndex[moduleSlot]++;
    }

    return fit;
}

/**
 * Resolve a fitted item's slot type by scanning its raw dogma
 * effects (for modules) and attributes (for drones). Uses the
 * hardcoded effect/attribute ID maps above rather than walking
 * the SDE name→id map — ~10× faster per import.
 */
function slotTypeForImport(
    typeId: number,
    sde: SdeData,
): FitSlotType | "DroneBay" | undefined {
    const typeDogma = sde.typeDogma.get(typeId);
    if (!typeDogma) return undefined;

    const effects = typeDogma.dogmaEffects as Array<{ effectID: number }> | undefined;
    if (effects) {
        for (const eff of effects) {
            const slot = EFFECT_SLOT_MAP[eff.effectID];
            if (slot) return slot;
        }
    }

    const attributes = typeDogma.dogmaAttributes as
        | Array<{ attributeID: number }>
        | undefined;
    if (attributes) {
        for (const attr of attributes) {
            const slot = ATTRIBUTE_SLOT_MAP[attr.attributeID];
            if (slot) return slot;
        }
    }

    return undefined;
}

function activeDroneCount(drones: FitDrone[]): number {
    let total = 0;
    for (const d of drones) total += d.states.Active;
    return total;
}

/** Re-export for navbar convenience. */
export type { FitCargo };
