/**
 * Platform-agnostic core for @evekill/dogma.
 *
 * Everything in this file is pure logic — no filesystem access, no Vite
 * asset URLs, no `fetch`. The `createDogma()` factory takes an IO adapter
 * that knows how to obtain the raw bytes of the WASM engine and the four
 * SDE protobuf blobs. The node entry (src/index.ts) provides an IO adapter
 * using fs/promises; the browser entry (src/index.browser.ts) provides one
 * using Vite ?url imports + fetch(). Both call into the same factory and
 * expose the same public API.
 *
 * The factory owns the caches so each entry-point import gets its own
 * (sde, engine) pair, loaded once per process lifetime.
 */

// Static-module classes generated from dist/upstream/data/dist/esf.proto via
// `bun run gen:proto` in this package. The generated module imports from
// `protobufjs/minimal`, which has no runtime codegen — every encode/decode/
// toObject body is written out as plain JS. That's the only thing that lets
// the engine load under our CSP without granting `unsafe-eval`; the full
// `protobufjs` build calls `new Function()` for performance and crashes hard
// when CSP blocks it. See packages/dogma/src/generated/esf.d.ts for the
// rationale behind the hand-written type shim.
import { esf } from "./generated/esf.js";
import * as engineBg from "../dist/upstream/dogma-engine/esf_dogma_engine_bg.js";
import { installCallbacks } from "./callbacks";
import { allFive } from "./skills";
import type { ComputedItem, EsfFit, FitStats, SkillMap } from "./types";

// ========== Shared types ==================================================

/**
 * Decoded SDE data, indexed into Maps for O(1) lookup by the callbacks.
 * `any` because protobufjs-decoded values are dynamic; we narrow inside the
 * callback wiring.
 */
export interface SdeData {
    /** type_id → { name, groupID, categoryID, capacity?, mass?, radius?, volume? } */
    types: Map<number, any>;
    /** type_id → { dogmaAttributes: [...], dogmaEffects: [...] } */
    typeDogma: Map<number, any>;
    /** attribute_id → { name, defaultValue, highIsGood, stackable } */
    dogmaAttributes: Map<number, any>;
    /** effect_id → DogmaEffect */
    dogmaEffects: Map<number, any>;
    /** category_id → { name, published } — e.g. 6=Ship, 7=Module, 8=Charge */
    categories: Map<number, any>;
    /** group_id → { name, categoryID, published } — e.g. Frigate, Cruiser, Drone */
    groups: Map<number, any>;
    /** market_group_id → { name, parentGroupID?, iconID? } — the tree the
     *  in-game market browser walks, used by HardwareListing's sidebar. */
    marketGroups: Map<number, any>;
    /** name → attribute_id (includes negative IDs for custom EVEShipFit stats) */
    attributeNameToId: Map<string, number>;
    /** name → type_id */
    typeNameToId: Map<string, number>;
    /** name → group_id */
    groupNameToId: Map<string, number>;
    /** name → market_group_id */
    marketGroupNameToId: Map<string, number>;
}

export interface LoadedEngine {
    init: () => void;
    calculate: (fit: unknown, skills: unknown) => unknown;
}

/**
 * Raw bytes for SDE decoding. Platform IO adapters produce one of these —
 * the seven binary fields are the protobuf-serialized SDE dumps from
 * @eveshipfit/data. The .proto schema itself is no longer needed at runtime
 * because the message classes were pre-generated into ./generated/esf.js.
 *
 * The last three (categories, groups, marketGroups) back the HardwareListing
 * sidebar — they aren't used by the dogma engine itself, but they're cheap
 * to ship alongside the four the engine needs (~120 KB total) and keeping
 * them in the same cache means consumers don't need to orchestrate a
 * separate SDE loader for the sidebar.
 */
export interface RawSdeBytes {
    types: Uint8Array | ArrayBuffer;
    typeDogma: Uint8Array | ArrayBuffer;
    dogmaAttributes: Uint8Array | ArrayBuffer;
    dogmaEffects: Uint8Array | ArrayBuffer;
    categories: Uint8Array | ArrayBuffer;
    groups: Uint8Array | ArrayBuffer;
    marketGroups: Uint8Array | ArrayBuffer;
}

// ========== Pure decoders =================================================

function asU8(buf: Uint8Array | ArrayBuffer): Uint8Array {
    return buf instanceof Uint8Array ? buf : new Uint8Array(buf);
}

/**
 * Decode the four binary SDE blobs into indexed Maps ready for callback
 * lookups. No IO; this is pure. The message classes are statically generated
 * (see ./generated/esf.js) so there is no runtime parsing of esf.proto and
 * no codegen — both of which would require `unsafe-eval` under CSP.
 */
export function decodeSdeFromBytes(bytes: RawSdeBytes): SdeData {
    // See the original loader for the rationale:
    //   enums:  Number — Domain/Func enums use serde_repr (i32) on Rust side
    //   longs:  Number — avoid BigInt at every attribute lookup
    //   arrays: true   — absent repeated fields become [] (serde needs the
    //                    modifierInfo Vec present even when empty)
    //   objects: true  — same story for absent map fields
    //   defaults: false — don't fill Option<T> fields with 0 masquerading
    //                    as Some(0)
    const toOpts = { enums: Number, longs: Number, arrays: true, objects: true };

    const typesDecoded = esf.Types.toObject(esf.Types.decode(asU8(bytes.types)), toOpts) as {
        entries: Record<string, any>;
    };
    const typeDogmaDecoded = esf.TypeDogma.toObject(
        esf.TypeDogma.decode(asU8(bytes.typeDogma)),
        toOpts,
    ) as { entries: Record<string, any> };
    const dogmaAttrsDecoded = esf.DogmaAttributes.toObject(
        esf.DogmaAttributes.decode(asU8(bytes.dogmaAttributes)),
        toOpts,
    ) as { entries: Record<string, any> };
    const dogmaEffectsDecoded = esf.DogmaEffects.toObject(
        esf.DogmaEffects.decode(asU8(bytes.dogmaEffects)),
        toOpts,
    ) as { entries: Record<string, any> };
    const categoriesDecoded = esf.Categories.toObject(
        esf.Categories.decode(asU8(bytes.categories)),
        toOpts,
    ) as { entries: Record<string, any> };
    const groupsDecoded = esf.Groups.toObject(
        esf.Groups.decode(asU8(bytes.groups)),
        toOpts,
    ) as { entries: Record<string, any> };
    const marketGroupsDecoded = esf.MarketGroups.toObject(
        esf.MarketGroups.decode(asU8(bytes.marketGroups)),
        toOpts,
    ) as { entries: Record<string, any> };

    const types = new Map<number, any>();
    const typeNameToId = new Map<string, number>();
    for (const [k, v] of Object.entries(typesDecoded.entries)) {
        const id = Number(k);
        types.set(id, v);
        if (v.name) typeNameToId.set(v.name, id);
    }

    const dogmaAttributes = new Map<number, any>();
    const attributeNameToId = new Map<string, number>();
    for (const [k, v] of Object.entries(dogmaAttrsDecoded.entries)) {
        const id = Number(k);
        dogmaAttributes.set(id, v);
        if (v.name) attributeNameToId.set(v.name, id);
    }

    const dogmaEffects = new Map<number, any>();
    for (const [k, v] of Object.entries(dogmaEffectsDecoded.entries)) {
        dogmaEffects.set(Number(k), v);
    }

    const typeDogma = new Map<number, any>();
    for (const [k, v] of Object.entries(typeDogmaDecoded.entries)) {
        typeDogma.set(Number(k), v);
    }

    const categories = new Map<number, any>();
    for (const [k, v] of Object.entries(categoriesDecoded.entries)) {
        categories.set(Number(k), v);
    }

    const groups = new Map<number, any>();
    const groupNameToId = new Map<string, number>();
    for (const [k, v] of Object.entries(groupsDecoded.entries)) {
        const id = Number(k);
        groups.set(id, v);
        if (v.name) groupNameToId.set(v.name, id);
    }

    const marketGroups = new Map<number, any>();
    const marketGroupNameToId = new Map<string, number>();
    for (const [k, v] of Object.entries(marketGroupsDecoded.entries)) {
        const id = Number(k);
        marketGroups.set(id, v);
        if (v.name) marketGroupNameToId.set(v.name, id);
    }

    // ---- Register EK custom attributes in the name→id map ------------------
    for (const { id, name, highIsGood } of EK_CUSTOM_ATTRS) {
        attributeNameToId.set(name, id);
        dogmaAttributes.set(id, {
            name,
            published: true,
            defaultValue: 0,
            highIsGood,
            stackable: true,
        });
    }

    // ---- Local SDE patches --------------------------------------------------
    //
    // EVEShipFit's rechargeShield.yaml and rechargeArmor.yaml patches only
    // target modules with the `shieldBoosting` / `armorRepair` effects. But
    // Ancillary Shield Boosters use `fueledShieldBoosting` and Ancillary
    // Armor Repairers use `fueledArmorRepair` instead — so they never get the
    // `shieldBoostRate` / `armorRepairRate` synthetic effects and produce 0
    // HP/s in the stats panel.
    //
    // Fix: for every type that has the fueled effect but is missing the
    // corresponding rate effect, inject it. Uses name→id lookups so this
    // survives upstream ID renumbering.
    patchFueledModules(typeDogma, dogmaEffects);

    return {
        types,
        typeDogma,
        dogmaAttributes,
        dogmaEffects,
        categories,
        groups,
        marketGroups,
        attributeNameToId,
        typeNameToId,
        groupNameToId,
        marketGroupNameToId,
    };
}

// ========== Custom EK attribute IDs =========================================
//
// These are our own synthetic attributes, separate from EVEShipFit's negative
// IDs (which start at -1 and go down). We start at -1000 to avoid collisions.
// The attributes are registered in `decodeSdeFromBytes` so `getHullStats` and
// `useShipAttribute` can resolve them by name.

/** @internal */ export const EK_ATTR_REMOTE_SHIELD_BOOST_RATE = -1001;
/** @internal */ export const EK_ATTR_REMOTE_ARMOR_REPAIR_RATE = -1002;
/** @internal */ export const EK_ATTR_REMOTE_HULL_REPAIR_RATE = -1003;
/** @internal */ export const EK_ATTR_ENERGY_NEUTRALIZER_RATE = -1004;
/** @internal */ export const EK_ATTR_ENERGY_NOSFERATU_RATE = -1005;
/** @internal */ export const EK_ATTR_REMOTE_CAP_TRANSFER_RATE = -1006;

const EK_CUSTOM_ATTRS: Array<{ id: number; name: string; highIsGood: boolean }> = [
    { id: EK_ATTR_REMOTE_SHIELD_BOOST_RATE, name: "remoteShieldBoostRate", highIsGood: true },
    { id: EK_ATTR_REMOTE_ARMOR_REPAIR_RATE, name: "remoteArmorRepairRate", highIsGood: true },
    { id: EK_ATTR_REMOTE_HULL_REPAIR_RATE, name: "remoteHullRepairRate", highIsGood: true },
    { id: EK_ATTR_ENERGY_NEUTRALIZER_RATE, name: "energyNeutralizerRate", highIsGood: true },
    { id: EK_ATTR_ENERGY_NOSFERATU_RATE, name: "energyNosferatuRate", highIsGood: true },
    { id: EK_ATTR_REMOTE_CAP_TRANSFER_RATE, name: "remoteCapTransferRate", highIsGood: true },
];

/**
 * Post-engine computation of remote assistance and warfare stats.
 *
 * The dogma engine computes per-module attributes (shieldBonus, duration,
 * etc.) correctly, but the upstream SDE patches only create ship-level
 * aggregate rates for self-repair modules. Remote modules (remote shield
 * boosters, remote armor repairers, neuts, NOS, cap transmitters) have
 * no ship-level aggregation.
 *
 * We compute these by iterating fitted modules, checking for the relevant
 * effects, and aggregating `amount / duration * 1000` (HP/s or GJ/s)
 * onto the hull's attribute map.
 */
function computeRemoteStats(sde: SdeData, stats: FitStats): void {
    // Build effect name → id lookup (cached per SDE load via closure,
    // but the SDE is already cached at a higher level so this is fine).
    const effectNameToId = new Map<string, number>();
    for (const [id, v] of sde.dogmaEffects) {
        if (v?.name) effectNameToId.set(v.name, id);
    }

    // Map: effect name(s) → { outputAttrId, amountAttrName }
    // Each entry says: "if a module has any of these effects AND is Active,
    // read `amountAttrName` and `duration` from the module, compute rate,
    // and add to the hull's outputAttrId."
    const remoteSpecs: Array<{
        effects: string[];
        amountAttr: string;
        outputId: number;
    }> = [
        {
            effects: ["shipModuleRemoteShieldBooster", "shipModuleAncillaryRemoteShieldBooster"],
            amountAttr: "shieldBonus",
            outputId: EK_ATTR_REMOTE_SHIELD_BOOST_RATE,
        },
        {
            effects: ["shipModuleRemoteArmorRepairer", "shipModuleAncillaryRemoteArmorRepairer", "ShipModuleRemoteArmorMutadaptiveRepairer"],
            amountAttr: "armorDamageAmount",
            outputId: EK_ATTR_REMOTE_ARMOR_REPAIR_RATE,
        },
        {
            effects: ["shipModuleRemoteHullRepairer"],
            amountAttr: "structureDamageAmount",
            outputId: EK_ATTR_REMOTE_HULL_REPAIR_RATE,
        },
        {
            effects: ["energyNeutralizerFalloff"],
            amountAttr: "energyNeutralizerAmount",
            outputId: EK_ATTR_ENERGY_NEUTRALIZER_RATE,
        },
        {
            effects: ["energyNosferatuFalloff"],
            amountAttr: "powerTransferAmount",
            outputId: EK_ATTR_ENERGY_NOSFERATU_RATE,
        },
        {
            effects: ["shipModuleRemoteCapacitorTransmitter"],
            amountAttr: "powerTransferAmount",
            outputId: EK_ATTR_REMOTE_CAP_TRANSFER_RATE,
        },
    ];

    // Resolve effect names to IDs and amount attr names to IDs.
    const durationId = sde.attributeNameToId.get("duration");
    if (durationId === undefined) return;

    const resolved = remoteSpecs.map((spec) => ({
        effectIds: spec.effects.map((n) => effectNameToId.get(n)).filter((id): id is number => id !== undefined),
        amountAttrId: sde.attributeNameToId.get(spec.amountAttr),
        outputId: spec.outputId,
    }));

    // Accumulate rates from fitted modules.
    const totals = new Map<number, number>();

    for (const item of stats.items) {
        // Only Active/Overload modules contribute.
        if (item.state !== "Active" && item.state !== "Overload") continue;

        // Check which type-level effects this module has.
        const td = sde.typeDogma.get(item.type_id);
        const typeEffects = (td?.dogmaEffects ?? []) as Array<{ effectID: number }>;

        for (const spec of resolved) {
            if (spec.amountAttrId === undefined) continue;
            const hasEffect = typeEffects.some((e) => spec.effectIds.includes(e.effectID));
            if (!hasEffect) continue;

            const amount = item.attributes.get(spec.amountAttrId)?.value ?? 0;
            const duration = item.attributes.get(durationId)?.value ?? 0;
            if (duration <= 0) continue;

            const rate = (amount / duration) * 1000;
            totals.set(spec.outputId, (totals.get(spec.outputId) ?? 0) + rate);
        }
    }

    // Write aggregated rates onto the hull's attribute map.
    for (const [attrId, value] of totals) {
        stats.hull.attributes.set(attrId, { base_value: 0, value, effects: [] });
    }
}

/**
 * Apply charge-to-parent range/tracking bonuses.
 *
 * The upstream dogma engine does not propagate turret ammo modifiers (e.g.
 * Antimatter's 0.5× optimal, Iron's 1.6× optimal) onto the parent turret's
 * maxRange / falloff / trackingSpeed. We overlay them here so the UI sees
 * ammo-adjusted range/tracking, and they update reactively when ammo is
 * swapped (calculateFit re-runs on fit identity change).
 *
 * Bonus attributes are multipliers: `1.0` = no change.
 */
function applyChargeRangeBonuses(sde: SdeData, stats: FitStats): void {
    const pairs: Array<{ target: string; bonus: string }> = [
        { target: "maxRange", bonus: "maxRangeBonus" },
        { target: "falloff", bonus: "falloffBonus" },
        { target: "trackingSpeed", bonus: "trackingSpeedBonus" },
    ];

    const resolved = pairs
        .map(p => ({
            targetId: sde.attributeNameToId.get(p.target),
            bonusId: sde.attributeNameToId.get(p.bonus),
        }))
        .filter((p): p is { targetId: number; bonusId: number } =>
            p.targetId !== undefined && p.bonusId !== undefined);

    if (resolved.length === 0) return;

    for (const item of stats.items) {
        const charge = item.charge;
        if (!charge) continue;
        for (const { targetId, bonusId } of resolved) {
            const bonus = charge.attributes.get(bonusId);
            const multiplier = bonus?.value ?? bonus?.base_value;
            if (multiplier === undefined || multiplier === null || multiplier === 1) continue;
            const target = item.attributes.get(targetId);
            if (!target) continue;
            const base = target.value ?? target.base_value;
            if (base === undefined || base === null) continue;
            item.attributes.set(targetId, {
                base_value: target.base_value,
                value: base * multiplier,
                effects: target.effects,
            });
        }
    }
}

/**
 * Inject `shieldBoostRate` / `armorRepairRate` effects into types that
 * have `fueledShieldBoosting` / `fueledArmorRepair` but are missing the
 * corresponding rate calculation effect.
 */
function patchFueledModules(
    typeDogma: Map<number, any>,
    dogmaEffects: Map<number, any>,
): void {
    // Build name → effectId map for the effects we care about.
    const effectNameToId = new Map<string, number>();
    for (const [id, v] of dogmaEffects) {
        if (v?.name) effectNameToId.set(v.name, id);
    }

    const patches: Array<{ fueled: string; rate: string }> = [
        { fueled: "fueledShieldBoosting", rate: "shieldBoostRate" },
        { fueled: "fueledArmorRepair", rate: "armorRepairRate" },
    ];

    for (const { fueled, rate } of patches) {
        const fueledId = effectNameToId.get(fueled);
        const rateId = effectNameToId.get(rate);
        if (fueledId === undefined || rateId === undefined) continue;

        for (const [, td] of typeDogma) {
            const effects = td?.dogmaEffects as Array<{ effectID: number; isDefault: boolean }> | undefined;
            if (!effects) continue;
            const hasFueled = effects.some((e) => e.effectID === fueledId);
            if (!hasFueled) continue;
            const hasRate = effects.some((e) => e.effectID === rateId);
            if (hasRate) continue;
            effects.push({ effectID: rateId, isDefault: false });
        }
    }
}

/**
 * Compile and instantiate the WASM engine from its raw bytes, wire the
 * wasm-bindgen JS glue, and install the Rust panic hook. No IO.
 */
export async function instantiateEngineFromBytes(
    wasmBytes: Uint8Array | ArrayBuffer,
): Promise<LoadedEngine> {
    // wasm-bindgen glue references `window.*` for extern callbacks. Browsers
    // already have window; Bun/Node don't — alias it to globalThis so the
    // callbacks resolve at call time. Must happen before `init()` below.
    const g = globalThis as unknown as { window?: unknown };
    if (typeof g.window === "undefined") {
        g.window = globalThis;
    }

    const ownedBytes = new Uint8Array(wasmBytes.byteLength);
    ownedBytes.set(wasmBytes instanceof ArrayBuffer ? new Uint8Array(wasmBytes) : wasmBytes);
    const wasmModule = await WebAssembly.compile(ownedBytes);

    const instance = await WebAssembly.instantiate(wasmModule, {
        "./esf_dogma_engine_bg.js": engineBg as unknown as WebAssembly.ModuleImports,
    });

    (engineBg as unknown as { __wbg_set_wasm: (w: unknown) => void }).__wbg_set_wasm(
        instance.exports,
    );
    (instance.exports as { __wbindgen_start?: () => void }).__wbindgen_start?.();

    // Install Rust's console_error_panic_hook so panics surface as readable
    // messages in stderr/devtools instead of "RuntimeError: Unreachable code".
    const init = (engineBg as unknown as { init: () => void }).init;
    init();

    return {
        init,
        calculate: (engineBg as unknown as {
            calculate: (fit: unknown, skills: unknown) => unknown;
        }).calculate,
    };
}

// ========== Factory ========================================================

/**
 * Platform IO adapter. The node entry implements this with fs/promises; the
 * browser entry implements it with fetch() against Vite asset URLs. Both are
 * async — cached at the factory level, so the actual IO runs exactly once.
 */
export interface DogmaIO {
    loadSdeBytes: () => Promise<RawSdeBytes>;
    loadEngineBytes: () => Promise<Uint8Array | ArrayBuffer>;
}

/** Public API returned by createDogma(). */
export interface DogmaApi {
    calculateFit: (fit: EsfFit, skills?: SkillMap) => Promise<FitStats>;
    loadSde: () => Promise<SdeData>;
    loadEngine: () => Promise<LoadedEngine>;
    getEngine: () => Promise<{ sde: SdeData; engine: LoadedEngine }>;
    getItemStat: (item: ComputedItem, name: string) => Promise<number | null>;
    getHullStat: (stats: FitStats, name: string) => Promise<number | null>;
    getHullStats: (
        stats: FitStats,
        names: readonly string[],
    ) => Promise<Record<string, number | null>>;
}

/**
 * Wire an IO adapter into a fully-cached dogma API. Each call to createDogma
 * gets its own module-scoped caches for SDE and engine, so node and browser
 * entries don't share state (which they shouldn't — different bytes, same
 * logic).
 */
export function createDogma(io: DogmaIO): DogmaApi {
    let sdeCache: Promise<SdeData> | null = null;
    let engineCache: Promise<LoadedEngine> | null = null;

    const loadSde = (): Promise<SdeData> => {
        sdeCache ??= io.loadSdeBytes().then(decodeSdeFromBytes);
        return sdeCache;
    };

    const loadEngine = (): Promise<LoadedEngine> => {
        engineCache ??= io.loadEngineBytes().then(instantiateEngineFromBytes);
        return engineCache;
    };

    const getEngine = async () => {
        const [sde, engine] = await Promise.all([loadSde(), loadEngine()]);
        return { sde, engine };
    };

    const calculateFit = async (fit: EsfFit, skills?: SkillMap): Promise<FitStats> => {
        const { sde, engine } = await getEngine();
        installCallbacks(sde);
        const resolvedSkills = skills ?? allFive(sde);
        const stats = engine.calculate(fit, resolvedSkills) as FitStats;
        applyChargeRangeBonuses(sde, stats);
        computeRemoteStats(sde, stats);
        return stats;
    };

    const getItemStat = async (item: ComputedItem, name: string): Promise<number | null> => {
        const sde = await loadSde();
        const id = sde.attributeNameToId.get(name);
        if (id === undefined) return null;
        const attr = item.attributes.get(id);
        if (!attr) return null;
        return attr.value ?? attr.base_value;
    };

    const getHullStat = (stats: FitStats, name: string) => getItemStat(stats.hull, name);

    const getHullStats = async (
        stats: FitStats,
        names: readonly string[],
    ): Promise<Record<string, number | null>> => {
        const sde = await loadSde();
        const out: Record<string, number | null> = {};
        for (const name of names) {
            const id = sde.attributeNameToId.get(name);
            if (id === undefined) {
                out[name] = null;
                continue;
            }
            const attr = stats.hull.attributes.get(id);
            out[name] = attr ? (attr.value ?? attr.base_value) : null;
        }
        return out;
    };

    return {
        calculateFit,
        loadSde,
        loadEngine,
        getEngine,
        getItemStat,
        getHullStat,
        getHullStats,
    };
}
