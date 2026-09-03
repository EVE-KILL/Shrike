import { describe, expect, test } from "bun:test";
import { aiidAlarmBand, isMapRenderBaseLayer, mapActivityRatio, mapActivityValue, sovereigntyAllianceColors } from "../app/utils/map/layers";

const maximums = {
  ship_kills: 100,
  pod_kills: 50,
  npc_kills: 10_000,
  ship_jumps: 20_000,
};

describe("map activity layers", () => {
  test("recognizes the interactive New Eden base layers", () => {
    expect(isMapRenderBaseLayer("sovereignty")).toBe(true);
    expect(isMapRenderBaseLayer("live")).toBe(true);
    expect(isMapRenderBaseLayer("aiid")).toBe(true);
    expect(isMapRenderBaseLayer("unknown")).toBe(false);
  });

  test("splits AIID alerts at five jumps", () => {
    expect(aiidAlarmBand(0)).toBe("near");
    expect(aiidAlarmBand(4)).toBe("near");
    expect(aiidAlarmBand(5)).toBe("outer");
    expect(aiidAlarmBand(10)).toBe("outer");
    expect(aiidAlarmBand(11)).toBeNull();
  });

  test("danger weights pod losses and floors low traffic", () => {
    expect(
      mapActivityValue(
        "danger",
        { ship_kills: 1, pod_kills: 2, npc_kills: 0, ship_jumps: 0 },
        maximums,
      ),
    ).toBe(30);
    expect(
      mapActivityValue(
        "danger",
        { ship_kills: 1, pod_kills: 2, npc_kills: 0, ship_jumps: 100 },
        maximums,
      ),
    ).toBe(6);
  });

  test("combined activity remains a normalized score", () => {
    const score = mapActivityValue(
      "activity",
      { ship_kills: 100, pod_kills: 50, npc_kills: 10_000, ship_jumps: 20_000 },
      maximums,
    );
    expect(score).toBe(100);
    expect(mapActivityRatio("activity", score, 100)).toBe(1);
  });

  test("inactive systems remain at the quiet end of the scale", () => {
    expect(
      mapActivityValue(
        "activity",
        { ship_kills: 0, pod_kills: 0, npc_kills: 0, ship_jumps: 0 },
        maximums,
      ),
    ).toBe(0);
    expect(mapActivityRatio("danger", 0, 25)).toBe(0);
  });
});

describe("sovereignty alliance colors", () => {
  test("assigns every current holder a unique deterministic color", () => {
    const allianceIds = Array.from({ length: 100 }, (_, index) => 99_000_000 + index);
    const first = sovereigntyAllianceColors(allianceIds);
    const second = sovereigntyAllianceColors(allianceIds);

    expect(new Set(first.values()).size).toBe(allianceIds.length);
    expect([...first]).toEqual([...second]);
  });
});
