import { describe, expect, test } from "bun:test";
import { fuzzyFitMatch, fuzzyFitScore } from "../app/utils/fuzzyFitSearch";

describe("fitting sidebar fuzzy search", () => {
    test("matches abbreviated module words and Arabic tech levels", () => {
        expect(fuzzyFitMatch("mega p 2", "Mega Pulse Laser II")).toBe(true);
    });

    test("matches tokens independently of filler words", () => {
        expect(fuzzyFitMatch("heavy assault 2", "Heavy Assault Missile Launcher II")).toBe(true);
    });

    test("does not match unrelated modules", () => {
        expect(fuzzyFitMatch("mega p 2", "Medium Energy Neutralizer II")).toBe(false);
    });

    test("ranks exact words ahead of looser prefixes", () => {
        expect(fuzzyFitScore("pulse", "Mega Pulse Laser II"))
            .toBeLessThan(fuzzyFitScore("pulse", "Medium Pulsed Energy Weapon"));
    });
});
