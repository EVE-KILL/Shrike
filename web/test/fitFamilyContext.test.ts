import { describe, expect, test } from "bun:test";
import { fitFamilyAdvancedSearchQuery, fitFamilyContextParts } from "../app/utils/fitFamilyContext";

describe("fitting family context", () => {
    test("keeps rendering while an older cached response has no context", () => {
        expect(fitFamilyContextParts(undefined)).toEqual([]);
    });

    test("summarizes a dominant security band and meaningful region", () => {
        expect(fitFamilyContextParts({
            security_distribution: [
                { name: "Nullsec", count: 72, pct: 72 },
                { name: "Lowsec", count: 28, pct: 28 },
            ],
            top_region: { region_id: 10000060, name: "Delve", count: 41, pct: 41 },
            median_attackers: 27,
            median_loss_value: 120_000_000,
        })).toEqual(["72% nullsec", "Delve 41%", "median 27 attackers", "median loss 120m"]);
    });

    test("shows the two leading security bands when neither dominates", () => {
        expect(fitFamilyContextParts({
            security_distribution: [
                { name: "Lowsec", count: 45, pct: 45 },
                { name: "Nullsec", count: 40, pct: 40 },
            ],
            top_region: null,
            median_attackers: 1,
            median_loss_value: 0,
        })).toEqual(["45% lowsec / 40% nullsec", "median solo loss"]);
    });

    test("does not repeat a special security area as its own region", () => {
        expect(fitFamilyContextParts({
            security_distribution: [{ name: "Pochven", count: 20, pct: 100 }],
            top_region: { region_id: 10000070, name: "Pochven", count: 20, pct: 100 },
            median_attackers: 8,
            median_loss_value: 700_000_000,
        })).toEqual(["100% pochven", "median 8 attackers", "median loss 700m"]);
    });

    test("builds the matching advanced-search time window", () => {
        expect(JSON.parse(fitFamilyAdvancedSearchQuery(90))).toEqual({
            timeRange: { preset: "90d" },
        });
    });
});
