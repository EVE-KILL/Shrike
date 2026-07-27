/**
 * Smoke test for @evekill/dogma.
 *
 * Goal: prove the whole stack wires up end-to-end. Load the WASM, parse the
 * SDE protobufs, install callbacks, run the engine against a minimal Rifter
 * hull (no modules), and assert that calculate() returns a non-empty object.
 *
 * This test does NOT assert specific DPS/EHP numbers yet — that comes in a
 * follow-up once we know the output shape. For now we just care that every
 * moving piece works without crashing.
 */

import { describe, expect, test } from "bun:test";
import { calculateFit } from "../src/index";
import type { EsfFit } from "../src/types";

const RIFTER_TYPE_ID = 587;

describe("dogma engine smoke test", () => {
  test("empty Rifter hull computes without error", async () => {
    const fit: EsfFit = {
      ship_type_id: RIFTER_TYPE_ID,
      modules: [],
      drones: [],
    };

    const stats = await calculateFit(fit);

    expect(stats).toBeDefined();
    expect(typeof stats).toBe("object");

    // Log a preview of the result so we can see the shape and tighten the
    // FitStats type in a follow-up.
    const preview = JSON.stringify(stats, null, 2);
    console.log(
      "=== calculateFit output (first 3KB) ===\n" +
        preview.slice(0, 3000) +
        (preview.length > 3000 ? "\n... [truncated]" : ""),
    );
    console.log(`\n=== total output size: ${preview.length} chars ===`);
  }, 60_000);
});
