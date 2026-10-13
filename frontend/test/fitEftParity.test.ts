import { expect, test } from "vitest";
import { loadSde } from "../dogma/src/index.ts";
import { exportFitToEft as sourceExport, importFitFromEft as sourceImport } from "../src/fit/eft";
import { exportFitToEft as goExport, importFitFromEft as goImport } from "../src/browser/fit-eft";

const sde = await loadSde();
const id = (name: string) => {
    const value = sde.typeNameToId.get(name);
    if (value === undefined) throw new Error(`Missing fitting fixture type: ${name}`);
    return value;
};

test("Go EFT export and import retain the Nuxt module, charge, drone and cargo contract", () => {
    const fit = {
        shipTypeId: id("Vexor"), name: "Vexor, parity", description: "",
        modules: [
            { slot: { type: "Low" as const, index: 1 }, typeId: id("Damage Control II"), state: "Online" as const },
            { slot: { type: "High" as const, index: 1 }, typeId: id("Light Neutron Blaster II"), state: "Active" as const, charge: { typeId: id("Void S") } },
        ],
        drones: [{ typeId: id("Hobgoblin II"), states: { Active: 2, Passive: 3 } }],
        cargo: [{ typeId: id("Tritanium"), quantity: 100 }],
    };
    const eft = sourceExport(fit, sde);
    expect(goExport(fit, sde)).toBe(eft);
    expect(goImport(eft!, sde)).toEqual(sourceImport(eft!, sde));
});

test("Go EFT parser matches Nuxt for empty slots, offline modules and extra drones", () => {
    const eft = `[Vexor, Sparse]\nDamage Control II /offline\n[Empty Low Slot]\nDrone Damage Amplifier II\n\n\nLight Neutron Blaster II, Void S\n\n\n\n\n\nHobgoblin II x7\n\nTritanium x100\n`;
    expect(goImport(eft, sde)).toEqual(sourceImport(eft, sde));
    expect(goImport(eft, sde)?.drones[0]?.states).toEqual({ Active: 5, Passive: 2 });
});
