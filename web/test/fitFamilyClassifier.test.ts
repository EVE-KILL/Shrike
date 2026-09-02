import { describe, expect, test } from "bun:test";
import { classifyFitFamily } from "../app/utils/fitFamilyClassifier";

const modules = (...names: string[]) => names.map((name) => ({ name }));
const drones = (...names: string[]) =>
  names.map((name) => ({ name, quantity: 5 }));

describe("fitting family classifier", () => {
  test("identifies remote repair doctrines before their local tank", () => {
    expect(
      classifyFitFamily(
        modules("Remote Armor Repairer II", "1600mm Steel Plates II"),
      ),
    ).toBe("Remote Armor Repair Armor Buffer");
  });

  test("combines covert, weapon, tank and engagement style", () => {
    expect(
      classifyFitFamily(
        modules(
          "Covert Ops Cloaking Device II",
          "Heavy Assault Missile Launcher II",
          "Large Shield Extender II",
          "Warp Scrambler II",
        ),
      ),
    ).toBe("Covert Heavy Assault Missile Shield Brawler");
  });

  test("keeps railguns as the primary identity when a fit also has neutralizers", () => {
    expect(
      classifyFitFamily(
        modules(
          "425mm Railgun II",
          "Heavy Energy Neutralizer II",
          "1600mm Steel Plates II",
        ),
      ),
    ).toBe("Railgun Neut Armor Buffer");
  });

  test("keeps blasters as the primary identity when a fit also has neutralizers", () => {
    expect(
      classifyFitFamily(
        modules(
          "Neutron Blaster Cannon II",
          "Heavy Energy Neutralizer II",
          "Large Armor Repairer II",
          "Warp Scrambler II",
        ),
      ),
    ).toBe("Blaster Neut Armor Brawler");
  });

  test("identifies a weaponless neutralizer fit as a neut fit", () => {
    expect(
      classifyFitFamily(
        modules("Heavy Energy Neutralizer II", "Large Armor Repairer II"),
      ),
    ).toBe("Neut Armor Active");
  });

  test("identifies capital weapon systems and siege without losing neutralizers", () => {
    expect(
      classifyFitFamily(
        modules(
          "Dual Giga Pulse Laser II",
          "Capital Energy Neutralizer II",
          "Siege Module II",
          "25000mm Steel Plates I",
        ),
      ),
    ).toBe("Siege Pulse Laser Neut Armor Buffer");
  });

  test("distinguishes interdiction fits from ordinary weapon fits", () => {
    expect(
      classifyFitFamily(
        modules(
          "Interdiction Sphere Launcher I",
          "125mm Gatling AutoCannon II",
          "Medium Shield Extender II",
          "Warp Scrambler II",
        ),
      ),
    ).toBe("Interdiction Autocannon Shield Brawler");
  });

  test("identifies covert cyno and industrial cyno fits", () => {
    expect(
      classifyFitFamily(
        modules(
          "Covert Ops Cloaking Device II",
          "Covert Cynosural Field Generator I",
          "Large Shield Extender II",
        ),
      ),
    ).toBe("Covert Cyno Shield Buffer");
    expect(
      classifyFitFamily(
        modules(
          "Industrial Cynosural Field Generator",
          "Expanded Cargohold II",
        ),
      ),
    ).toBe("Industrial Cyno");
  });

  test("recognizes gas harvesting and electronic warfare roles", () => {
    expect(
      classifyFitFamily(
        modules("Abyssal Gas Cloud Harvester", "Warp Core Stabilizer II"),
      ),
    ).toBe("Gas Harvesting");
    expect(
      classifyFitFamily(
        modules("Multispectral ECM II", "Signal Distortion Amplifier II"),
      ),
    ).toBe("ECM");
    expect(
      classifyFitFamily(
        modules(
          "Stasis Webifier II",
          "Target Painter II",
          "Medium Shield Extender II",
        ),
      ),
    ).toBe("Web/Painter Shield Buffer");
    expect(
      classifyFitFamily(
        modules("Light Missile Launcher II", "Multispectral ECM II"),
      ),
    ).toBe("Light Missile ECM");
  });

  test("retains neutralizer and web roles together on control fits", () => {
    expect(
      classifyFitFamily(
        modules(
          "Heavy Energy Neutralizer II",
          "Federation Navy Stasis Webifier",
          "25000mm Crystalline Carbonide Restrained Plates",
        ),
      ),
    ).toBe("Neut Web Support Armor Buffer");
  });

  test("describes bastion weapon fits instead of their utility repairer", () => {
    expect(
      classifyFitFamily(
        modules(
          "Bastion Module I",
          "Tachyon Beam Laser II",
          "Heavy Energy Neutralizer II",
          "Large Remote Armor Repairer II",
          "1600mm Steel Plates II",
        ),
      ),
    ).toBe("Bastion Beam Laser Neut Armor Buffer");
  });

  test("uses drones and modules together to describe hybrid weapon platforms", () => {
    expect(
      classifyFitFamily(
        modules(
          "Drone Damage Amplifier II",
          "Rapid Light Missile Launcher II",
          "Medium Shield Booster II",
        ),
        drones("Caldari Navy Vespa"),
      ),
    ).toBe("Drone/Rapid Light Missile Shield Active");
    expect(
      classifyFitFamily(
        modules(
          "Drone Damage Amplifier II",
          "Omnidirectional Tracking Link II",
          "1600mm Steel Plates II",
        ),
        drones("Garde II"),
      ),
    ).toBe("Sentry Drone Armor Buffer");
  });

  test("recognizes triage, jump-field and rapid-heavy fits", () => {
    expect(
      classifyFitFamily(
        modules(
          "Triage Module II",
          "CONCORD Capital Remote Armor Repairer",
          "Capital Armor Repairer I",
        ),
      ),
    ).toBe("Triage Remote Armor Repair Armor Active");
    expect(
      classifyFitFamily(
        modules(
          "Micro Jump Field Generator",
          "Rocket Launcher II",
          "Medium Shield Extender II",
        ),
      ),
    ).toBe("Jump Field Rocket Shield Buffer");
    expect(
      classifyFitFamily(
        modules("Rapid Heavy Missile Launcher II", "Large Shield Extender II"),
      ),
    ).toBe("Rapid Heavy Missile Shield Buffer");
  });

  test("distinguishes scanning, hauling and travel fits", () => {
    expect(
      classifyFitFamily(
        modules(
          "Expanded Probe Launcher II",
          "Scan Acquisition Array I",
          "Scan Rangefinding Array I",
          "Interdiction Nullifier I",
        ),
      ),
    ).toBe("Scanning");
    expect(
      classifyFitFamily(
        modules("Expanded Cargohold II", "Medium Cargohold Optimization II"),
      ),
    ).toBe("Hauling");
    expect(
      classifyFitFamily(
        modules("Interdiction Nullifier I", "Nanofiber Internal Structure II"),
      ),
    ).toBe("Travel");
    expect(
      classifyFitFamily(
        modules(
          "Inertial Stabilizers II",
          "Medium Hyperspatial Velocity Optimizer I",
        ),
        [],
        { hullGroupName: "Industrial" },
      ),
    ).toBe("Hauling");
  });

  test("recognizes wormhole rolling fits", () => {
    expect(
      classifyFitFamily(
        modules("100MN Y-S8 Compact Afterburner", "Medium Higgs Anchor I"),
      ),
    ).toBe("Rolling");
  });

  test("falls back cleanly for an unrecognized fitting", () => {
    expect(classifyFitFamily(modules("Damage Control II"))).toBe(
      "General Purpose",
    );
  });
});
