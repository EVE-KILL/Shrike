import { describe, expect, test } from "bun:test";
import { createSovereigntyTerritories, sovereigntyOwnerAt } from "../app/utils/map/territories";

const bounds = { x: 0, y: 0, width: 200, height: 120 };

describe("sovereignty territory contours", () => {
  test("joins nearby systems owned by the same alliance into a smooth territory", () => {
    const nodes = [
      { id: 1, x: 40, y: 60 },
      { id: 2, x: 75, y: 60 },
    ];
    const result = createSovereigntyTerritories(nodes, [{ from: 1, to: 2 }], bounds, new Map([[1, 10], [2, 10]]), 160);

    expect(result.territories).toHaveLength(1);
    expect(result.territories[0]?.allianceId).toBe(10);
    expect(result.territories[0]?.polygons).toHaveLength(1);
    expect(sovereigntyOwnerAt(result, 58, 60)).toBe(10);
  });

  test("resolves overlapping holders and preserves unclaimed system voids", () => {
    const nodes = [
      { id: 1, x: 50, y: 60 },
      { id: 2, x: 85, y: 60 },
      { id: 3, x: 68, y: 60 },
    ];
    const result = createSovereigntyTerritories(nodes, [], bounds, new Map([[1, 10], [2, 20]]), 160);

    expect(new Set(result.territories.map(territory => territory.allianceId))).toEqual(new Set([10, 20]));
    expect(sovereigntyOwnerAt(result, 50, 60)).toBe(10);
    expect(sovereigntyOwnerAt(result, 85, 60)).toBe(20);
    expect(sovereigntyOwnerAt(result, 68, 60)).toBeNull();
  });
});
