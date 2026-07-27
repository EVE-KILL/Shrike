/**
 * Integration test: real popular Muninn fit from Peter's fitting_items data,
 * piped through the fittingToEsfFit converter and into the dogma engine,
 * then asserted against sensible numeric ranges.
 *
 * This fit (shield HML Muninn, 98 uses in 90 days, The Initiative.) has no
 * charges or drones in the fitting_items shape by design, so DPS comes out
 * as 0 here — that's expected and documented in the converter comment. The
 * asserts below cover tank, speed, cap, fitting, and align time.
 */

import { describe, expect, test } from "bun:test";
import { calculateFit, fittingToEsfFit, getHullStats } from "../src/index";
import {
  MUNINN_TOP_FIT_HASH,
  MUNINN_TOP_FIT_ITEMS,
  MUNINN_TYPE_ID,
} from "./fixtures/muninn.fit";

describe("muninn top fit integration", () => {
  test("HML shield Muninn produces sensible all-V statistics", async () => {
    const fit = fittingToEsfFit(MUNINN_TYPE_ID, MUNINN_TOP_FIT_ITEMS);
    expect(fit.modules).toHaveLength(17);
    expect(fit.ship_type_id).toBe(MUNINN_TYPE_ID);

    const stats = await calculateFit(fit);

    // ----- Shape sanity -----
    expect(stats.hull.type_id).toBe(MUNINN_TYPE_ID);
    expect(stats.hull.attributes.size).toBeGreaterThan(400); // ~477 in practice
    expect(stats.items).toHaveLength(17);
    expect(stats.skills.length).toBeGreaterThan(500); // all-V → ~510 published skills

    // Every fitted module should have computed attributes
    for (const item of stats.items) {
      expect(item.attributes.size).toBeGreaterThan(0);
    }

    // ----- Named stats via the name→id helper -----
    const s = await getHullStats(stats, [
      // Tank
      "ehp",
      "shieldEhp",
      "armorEhp",
      "hullEhp",
      // Capacitor
      "capacitorCapacity",
      "capacitorDepletesIn",
      "capacitorPeakDeltaPercentage",
      // Navigation
      "maxVelocity",
      "alignTime",
      "signatureRadius",
      // Fitting
      "cpuFree",
      "powerFree",
      // Damage (will be 0 — no ammo in fitting_items)
      "damagePerSecondWithReload",
    ]);

    console.log(`\n=== Muninn all-V stats (fit_hash ${MUNINN_TOP_FIT_HASH.slice(0, 12)}…) ===`);
    for (const [name, value] of Object.entries(s)) {
      const formatted =
        value === null ? "null" : typeof value === "number" ? value.toFixed(2) : value;
      console.log(`  ${name.padEnd(32)} = ${formatted}`);
    }

    // ----- Numeric assertions -----
    // Tank: shield HAC should clear 40k EHP with all-V + 2x LSE II + multispec hardener
    expect(s.ehp).toBeGreaterThan(40_000);
    expect(s.ehp).toBeLessThan(80_000);
    expect(s.shieldEhp).toBeGreaterThan(s.armorEhp!);
    expect(s.shieldEhp).toBeGreaterThan(s.hullEhp!);

    // Capacitor: this fit is cap-stable (engine sentinel: depletesIn === -1)
    expect(s.capacitorDepletesIn).toBe(-1);
    expect(s.capacitorPeakDeltaPercentage).toBeGreaterThan(0);

    // Navigation: MWD active → multi-km/s; align should be in the 7-10s range
    expect(s.maxVelocity).toBeGreaterThan(1_500);
    expect(s.alignTime).toBeGreaterThan(5);
    expect(s.alignTime).toBeLessThan(15);

    // Fitting: fit is valid → no overflow (non-negative leftover PG/CPU)
    expect(s.cpuFree).toBeGreaterThanOrEqual(0);
    expect(s.powerFree).toBeGreaterThanOrEqual(0);

    // DPS expected 0 with no ammo/drones — this is documented and a reminder
    // to build the charge overlay before surfacing DPS on the site.
    expect(s.damagePerSecondWithReload).toBe(0);
  }, 60_000);
});
