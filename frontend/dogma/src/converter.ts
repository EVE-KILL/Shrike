/**
 * Converter from Peter's fitting_items row shape to the EsfFit shape the
 * dogma engine consumes.
 *
 * Input shape mirrors the `fitting_items` table exactly:
 *   { slot_group: 1..5, ordinal: number, type_id: number }
 *
 * The slot_group → EsfSlotType mapping matches the FittingExtractor's
 * numbering (see backend/src/services/FittingExtractor.ts):
 *   1 = high slot   → "High"
 *   2 = med slot    → "Medium"
 *   3 = low slot    → "Low"
 *   4 = rig slot    → "Rig"
 *   5 = subsystem   → "SubSystem"
 *
 * State defaults are a heuristic based on what slot a module sits in — we
 * don't have the per-module active/online distinction stored anywhere, so
 * we pick the state that gives the "intended" fit statistic:
 *
 *   High slots → Active   (weapons/neuts/remote reps that contribute DPS or
 *                          cap drain only when running)
 *   Med slots  → Active   (hardeners, prop mods, tackle, shield boosters —
 *                          all of these apply their bonus only when active)
 *   Low slots  → Online   (BCS, gyros, damage control, plates — passive mods
 *                          that apply just by being fit. Armor reppers are
 *                          the edge case and get under-counted, fine for now)
 *   Rig slots  → Passive  (rigs don't have an on/off state in EVE)
 *   Subsystem  → Online   (T3C subsystems are always-on once fit)
 *
 * Note: the input does NOT include charges, drones, or ammo (Peter's
 * extractor drops them) — so the resulting EsfFit will compute DPS with
 * whatever the engine's default-missile-for-launcher behavior is. This is
 * fine for smoke-testing but will need a charge/drone overlay before we
 * expose DPS numbers on the site.
 */

import type {
  EsfFit,
  EsfModule,
  EsfSlotTypeName,
  EsfStateName,
} from "./types";

/** Raw row shape from the `fitting_items` table. */
export interface FittingItemRow {
  slot_group: number;
  ordinal: number;
  type_id: number;
}

const SLOT_GROUP_TO_TYPE: Record<number, EsfSlotTypeName> = {
  1: "High",
  2: "Medium",
  3: "Low",
  4: "Rig",
  5: "SubSystem",
};

const DEFAULT_STATE_BY_SLOT: Record<EsfSlotTypeName, EsfStateName> = {
  High: "Active",
  Medium: "Active",
  Low: "Online",
  Rig: "Passive",
  SubSystem: "Online",
  Service: "Online",
};

/**
 * Optional options for fittingToEsfFit. When `isPropulsionModule` is
 * supplied the converter enforces EVE's propmod exclusivity: only the
 * first propulsion module (MWD / Afterburner) defaults to Active, the
 * rest default to Online. Without the callback the old behavior is
 * preserved (all medium-slot modules default to Active).
 */
export interface FittingToEsfOptions {
  isPropulsionModule?: (typeId: number) => boolean;
}

/**
 * Convert a list of fitting_items rows + ship type into an EsfFit ready to
 * hand to calculateFit(). Preserves per-slot `ordinal` as the EsfSlot.index.
 *
 * Items with an unrecognized slot_group are silently dropped — this should
 * never happen for data produced by FittingExtractor but we're defensive in
 * case the extractor is ever extended with new slot groups (e.g. fighter
 * tubes) that haven't been mapped here yet.
 */
export function fittingToEsfFit(
  shipTypeId: number,
  items: ReadonlyArray<FittingItemRow>,
  options?: FittingToEsfOptions,
): EsfFit {
  const modules: EsfModule[] = [];
  let hasActivePropmod = false;

  for (const row of items) {
    const slotType = SLOT_GROUP_TO_TYPE[row.slot_group];
    if (!slotType) continue;

    let state = DEFAULT_STATE_BY_SLOT[slotType];

    // Propmod exclusivity: only one propulsion module can be Active.
    // The first one keeps its Active default; subsequent ones downgrade
    // to Online so the engine doesn't stack both velocity bonuses.
    if (options?.isPropulsionModule?.(row.type_id)) {
      if (hasActivePropmod) {
        state = "Online";
      } else {
        hasActivePropmod = true;
      }
    }

    modules.push({
      type_id: row.type_id,
      slot: { type: slotType, index: row.ordinal },
      state,
    });
  }

  return {
    ship_type_id: shipTypeId,
    modules,
    drones: [],
  };
}
