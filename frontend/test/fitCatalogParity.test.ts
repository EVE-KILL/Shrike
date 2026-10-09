import { expect, test } from "vitest";
import { fuzzyFitMatch as sourceMatch, fuzzyFitScore as sourceScore } from "../src/utils/fuzzyFitSearch";
import { fuzzyFitMatch as goMatch, fuzzyFitScore as goScore } from "../src/browser/fit-catalog-search";

test("Go fitting catalog search keeps the Nuxt word and tech-level ranking", () => {
    const candidates = ["Damage Control II", "Damage Control I", "Mega Pulse Laser II", "Medium Pulsed Energy Weapon", "Vexor Navy Issue", "Vexor"];
    for (const query of ["damage 2", "mega p 2", "pulse", "vex nav", "medium", "not present"]) {
        expect(candidates.filter(name => goMatch(query, name))).toEqual(candidates.filter(name => sourceMatch(query, name)));
        expect([...candidates].sort((a, b) => goScore(query, a) - goScore(query, b)))
            .toEqual([...candidates].sort((a, b) => sourceScore(query, a) - sourceScore(query, b)));
    }
});
