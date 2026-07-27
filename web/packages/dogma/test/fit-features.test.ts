/**
 * Feature coverage for the full EsfFit input shape.
 *
 * The muninn.fit.test.ts exercises the "modules without charges/drones" path
 * because that's what the killmail-derived fitting_items table ships. These
 * tests exercise the rest of EsfFit:
 *
 *   1. charges       — EsfModule.charge (Option<EsfCharge>) pass-through
 *   2. drones        — EsfFit.drones array
 *
 * Together with the Muninn test, this gives us confidence that every field
 * of EsfFit is wired through to the Rust dogma engine and produces real
 * numbers. This is the gate before designing the user_fittings schema and
 * building the fit-builder UI that will actually exercise all of these.
 *
 * Type IDs are resolved at runtime via the SDE name lookup rather than
 * hardcoded — that way, an SDE refresh that re-numbers a type won't silently
 * break the test, it will fail loudly at the lookup step.
 */

import { describe, expect, test } from "bun:test";
import { calculateFit, fittingToEsfFit, getHullStats, loadSde } from "../src/index";
import type { EsfFit } from "../src/types";
import { MUNINN_TOP_FIT_ITEMS, MUNINN_TYPE_ID } from "./fixtures/muninn.fit";

async function resolveTypeId(name: string): Promise<number> {
  const sde = await loadSde();
  const id = sde.typeNameToId.get(name);
  if (id === undefined) {
    throw new Error(`SDE type_name_to_id: '${name}' not found`);
  }
  return id;
}

describe("EsfFit feature coverage", () => {
  test("charges produce non-zero DPS on Muninn with Scourge Fury HM", async () => {
    const scourgeFuryHm = await resolveTypeId("Scourge Fury Heavy Missile");

    // Start from the same Muninn fit as muninn.fit.test.ts and inject a
    // Scourge Fury HM charge on every high-slot module (all 5 HMLs).
    const base = fittingToEsfFit(MUNINN_TYPE_ID, MUNINN_TOP_FIT_ITEMS);
    const fit: EsfFit = {
      ...base,
      modules: base.modules.map((m) =>
        m.slot.type === "High"
          ? { ...m, charge: { type_id: scourgeFuryHm } }
          : m,
      ),
    };

    const stats = await calculateFit(fit);
    const s = await getHullStats(stats, [
      "damagePerSecondWithReload",
      "damagePerSecondWithoutReload",
      "damageAlpha",
    ]);

    console.log(`\n=== Muninn + Scourge Fury HM (all-V) ===`);
    console.log(`  damageAlpha                   = ${s.damageAlpha?.toFixed(2)}`);
    console.log(`  damagePerSecondWithoutReload  = ${s.damagePerSecondWithoutReload?.toFixed(2)}`);
    console.log(`  damagePerSecondWithReload     = ${s.damagePerSecondWithReload?.toFixed(2)}`);

    // DPS should be substantial — 5x HML II with Fury missiles, 2x BCS II,
    // 2x MGE II at all-V. Expect somewhere in the 400-1200 DPS band; leave
    // the range wide so a future rebalance doesn't break the test without
    // a real regression.
    expect(s.damagePerSecondWithReload).toBeGreaterThan(300);
    expect(s.damagePerSecondWithReload).toBeLessThan(1500);
    expect(s.damagePerSecondWithoutReload).toBeGreaterThan(s.damagePerSecondWithReload!);
    expect(s.damageAlpha).toBeGreaterThan(0);
  }, 60_000);

  test("drones produce non-zero DPS on a bare Vexor with 2x Ogre II", async () => {
    const vexor = await resolveTypeId("Vexor");
    const ogreII = await resolveTypeId("Ogre II");

    // Minimal fit: Vexor hull, no modules, 2 active heavy drones. This
    // exercises the EsfFit.drones field in isolation — DPS must be non-zero
    // and come entirely from drone contribution (there are no turrets).
    const fit: EsfFit = {
      ship_type_id: vexor,
      modules: [],
      drones: [
        { type_id: ogreII, state: "Active" },
        { type_id: ogreII, state: "Active" },
      ],
    };

    const stats = await calculateFit(fit);
    const s = await getHullStats(stats, [
      "damagePerSecondWithReload",
      "damagePerSecondWithoutReload",
    ]);

    console.log(`\n=== Bare Vexor + 2x Ogre II (all-V) ===`);
    console.log(`  damagePerSecondWithoutReload  = ${s.damagePerSecondWithoutReload?.toFixed(2)}`);
    console.log(`  damagePerSecondWithReload     = ${s.damagePerSecondWithReload?.toFixed(2)}`);

    // 2x Ogre II with all-V Gallente Cruiser drone damage bonuses should
    // easily clear 100 DPS; Ogre II base is ~130 DPS each.
    expect(s.damagePerSecondWithReload).toBeGreaterThan(100);
    expect(stats.items).toHaveLength(2); // the 2 drones show up as items
  }, 60_000);
});
