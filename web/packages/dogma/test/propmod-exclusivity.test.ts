/**
 * Propulsion module exclusivity tests.
 *
 * EVE only allows one propulsion module (MWD or Afterburner) active at a
 * time. These tests verify:
 *
 *   1. The fittingToEsfFit converter downgrades the second propmod to
 *      Online when given an isPropulsionModule callback.
 *   2. A fit with two propmods both Active produces a higher (wrong)
 *      speed than a fit with only one Active — confirming the engine
 *      stacks them and our exclusivity guard is necessary.
 */

import { describe, expect, test } from "bun:test";
import { calculateFit, fittingToEsfFit, getHullStats, loadSde } from "../src/index";
import type { EsfFit } from "../src/types";
import type { FittingItemRow } from "../src/converter";

/** EVE SDE groupID for all fittable propulsion modules (AB + MWD). */
const PROPULSION_MODULE_GROUP_ID = 46;

/**
 * Minimal Caracal fit with both a 10MN AB and 50MN MWD in medium slots.
 * This never happens in real gameplay (nobody would fit both on the same
 * ship except in rare edge cases), but it's the test scenario for
 * exclusivity enforcement.
 */
async function buildDualPropFit(): Promise<{
  fit: EsfFit;
  isPropulsionModule: (typeId: number) => boolean;
}> {
  const sde = await loadSde();

  const caracal = sde.typeNameToId.get("Caracal");
  if (caracal === undefined) throw new Error("Caracal not found in SDE");

  const mwd50mn = sde.typeNameToId.get("50MN Quad LiF Restrained Microwarpdrive");
  const ab10mn = sde.typeNameToId.get("10MN Afterburner II");
  if (mwd50mn === undefined) throw new Error("50MN MWD not found in SDE");
  if (ab10mn === undefined) throw new Error("10MN AB not found in SDE");

  const isPropulsionModule = (typeId: number) => {
    const type = sde.types.get(typeId);
    return type?.groupID === PROPULSION_MODULE_GROUP_ID;
  };

  const items: FittingItemRow[] = [
    { slot_group: 2, ordinal: 0, type_id: mwd50mn },
    { slot_group: 2, ordinal: 1, type_id: ab10mn },
  ];

  return {
    fit: fittingToEsfFit(caracal, items, { isPropulsionModule }),
    isPropulsionModule,
  };
}

describe("propmod exclusivity", () => {
  test("converter only activates the first propmod when isPropulsionModule is set", async () => {
    const { fit } = await buildDualPropFit();

    expect(fit.modules).toHaveLength(2);
    // First propmod (MWD) should be Active
    expect(fit.modules[0].state).toBe("Active");
    // Second propmod (AB) should be downgraded to Online
    expect(fit.modules[1].state).toBe("Online");
  }, 60_000);

  test("converter without isPropulsionModule keeps both Active (old behavior)", async () => {
    const sde = await loadSde();
    const caracal = sde.typeNameToId.get("Caracal")!;
    const mwd50mn = sde.typeNameToId.get("50MN Quad LiF Restrained Microwarpdrive")!;
    const ab10mn = sde.typeNameToId.get("10MN Afterburner II")!;

    const items: FittingItemRow[] = [
      { slot_group: 2, ordinal: 0, type_id: mwd50mn },
      { slot_group: 2, ordinal: 1, type_id: ab10mn },
    ];

    const fit = fittingToEsfFit(caracal, items);
    expect(fit.modules[0].state).toBe("Active");
    expect(fit.modules[1].state).toBe("Active");
  }, 60_000);

  test("single propmod active gives lower speed than both active (proving stacking is real)", async () => {
    const sde = await loadSde();
    const caracal = sde.typeNameToId.get("Caracal")!;
    const mwd50mn = sde.typeNameToId.get("50MN Quad LiF Restrained Microwarpdrive")!;
    const ab10mn = sde.typeNameToId.get("10MN Afterburner II")!;

    // Fit with BOTH propmods Active (broken — what we're fixing)
    const bothActive: EsfFit = {
      ship_type_id: caracal,
      modules: [
        { type_id: mwd50mn, slot: { type: "Medium", index: 0 }, state: "Active" },
        { type_id: ab10mn, slot: { type: "Medium", index: 1 }, state: "Active" },
      ],
      drones: [],
    };

    // Fit with only MWD Active, AB Online (correct)
    const mwdOnly: EsfFit = {
      ship_type_id: caracal,
      modules: [
        { type_id: mwd50mn, slot: { type: "Medium", index: 0 }, state: "Active" },
        { type_id: ab10mn, slot: { type: "Medium", index: 1 }, state: "Online" },
      ],
      drones: [],
    };

    const [bothStats, mwdStats] = await Promise.all([
      calculateFit(bothActive),
      calculateFit(mwdOnly),
    ]);

    const bothSpeed = (await getHullStats(bothStats, ["maxVelocity"])).maxVelocity!;
    const mwdSpeed = (await getHullStats(mwdStats, ["maxVelocity"])).maxVelocity!;

    console.log(`\n=== Dual-prop Caracal speed comparison ===`);
    console.log(`  Both propmods Active:     ${bothSpeed.toFixed(1)} m/s (WRONG — should never happen)`);
    console.log(`  Only MWD Active:          ${mwdSpeed.toFixed(1)} m/s (correct)`);

    // The dual-active speed should be higher than single — this proves the
    // engine stacks them and our UI-level exclusivity guard is necessary.
    expect(bothSpeed).toBeGreaterThan(mwdSpeed);
    // MWD-only speed should still be substantial (multi-km/s for a cruiser)
    expect(mwdSpeed).toBeGreaterThan(1000);
  }, 60_000);
});
