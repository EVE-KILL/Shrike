<script setup lang="ts">
/**
 * One module slot on the ring.
 *
 * Ported 1:1 from @eveshipfit/react's Slot.tsx. Each slot renders:
 *
 *   1. The slot wedge SVG (filled according to module state via a
 *      `data-state` attribute which the shared CSS reads)
 *   2. The fitted module's icon (or a placeholder slot-type glyph if
 *      the slot is empty), possibly with a sepia tint if the module
 *      is a hover-preview
 *   3. Three action glyphs (uncharge / unfit / offline) that only
 *      appear on hover when a module is fitted and readOnly is off
 *
 * The `main` slot prop is a one-off: exactly one Slot instance per
 * ring renders the shared SVG `<g id="slot">`, `<g id="slot-active">`,
 * `<g id="slot-passive">`, and the three action glyph `<g>` defs.
 * Every other slot picks them up via `<use href="#slot">`. We put the
 * defs on the first High-slot's element with `display: none`. This
 * avoids 24 copies of the same path.
 *
 * Drag protocol (custom MIME):
 *   application/esf-type-id    — numeric EVE type id
 *   application/esf-slot-type  — "High"|"Medium"|"Low"|"Rig"|
 *                                "SubSystem"|"Charge"|"DroneBay"
 *   application/esf-slot-index — index when dragging from an existing
 *                                slot (empty when from the listing)
 */

import type { FitSlotType, FitState } from "../../../composables/fit/types";
import FitContextMenu from "../table/FitContextMenu.vue";
import styles from "./ShipFit.module.css";
import ShipFitIcon, { type ShipFitIconName } from "../ShipFitIcon.vue";

const props = defineProps<{
    type: FitSlotType;
    index: number;
    fittable: boolean;
    main?: boolean;
    readOnly?: boolean;
}>();

const { sde } = useEveData();
const { fit } = useCurrentFit();
const { stats } = useFitStatistics();
const fitManager = useFitManager();

/** Cycle order per max_state — matches the React lib's stateRotation. */
const stateRotation: Record<string, FitState[]> = {
    Passive: ["Passive"],
    Online: ["Passive", "Online"],
    Active: ["Passive", "Online", "Active"],
    Overload: ["Passive", "Online", "Active", "Overload"],
};

/** The computed item from the engine output for this (type, index). */
const computedItem = computed(() => {
    return stats.value?.items.find(
        (item) => item.slot.type === props.type && item.slot.index === props.index,
    );
});

/** The raw fit module (may have state === "Preview" during hover overlay). */
const fitModule = computed(() => {
    return fit.value?.modules.find(
        (item) => item.slot.type === props.type && item.slot.index === props.index,
    );
});

const hasItem = computed(() => computedItem.value !== undefined);

/** True if the module has a state spectrum that includes Active/Overload. */
const isActiveType = computed(() => {
    const maxState = computedItem.value?.max_state;
    return maxState !== "Passive" && maxState !== "Online";
});

/** Name of the fitted type, for the slot's v-tooltip. */
const typeName = computed(() => {
    const id = computedItem.value?.type_id;
    if (id === undefined || !sde.value) return undefined;
    return sde.value.types.get(id)?.name;
});

/** Display state for data-state attribute. "Passive" on an active-capable
 *  module means "Offline" (turned off) and paints at 30% opacity. */
const displayState = computed(() => {
    const c = computedItem.value;
    if (!c) return undefined;
    if (c.state === "Passive" && c.max_state !== "Passive") return "Offline";
    return c.state;
});

const isPreview = computed(() => fitModule.value?.state === "Preview");

/** Placeholder icon for empty fittable slots. */
const placeholderIconName = computed<ShipFitIconName | undefined>(() => {
    switch (props.type) {
        case "Low":
            return "fitting-lowslot";
        case "Medium":
            return "fitting-medslot";
        case "High":
            return "fitting-hislot";
        case "Rig":
            return "fitting-rig-subsystem";
        default:
            return undefined;
    }
});

// ---------- interactions ----------

function cycleState(e: MouseEvent) {
    if (props.readOnly) return;
    const c = computedItem.value;
    if (!c) return;

    const states = stateRotation[c.max_state] ?? ["Passive"];
    const currentIdx = states.indexOf(c.state as FitState);
    const delta = e.shiftKey ? -1 : 1;
    const nextIdx = (currentIdx + delta + states.length) % states.length;
    const nextState = states[nextIdx];
    if (nextState) {
        fitManager.setModuleState(props.type, props.index, nextState);
    }
}

function onOfflineClick(e: Event) {
    e.stopPropagation();
    const c = computedItem.value;
    if (!c) return;
    if (c.state === "Passive") {
        fitManager.setModuleState(props.type, props.index, "Online");
    } else {
        fitManager.setModuleState(props.type, props.index, "Passive");
    }
}

function onUnfitClick(e: Event) {
    e.stopPropagation();
    if (!computedItem.value) return;
    fitManager.removeModule(props.type, props.index);
}

function onUnchargeClick(e: Event) {
    e.stopPropagation();
    if (!computedItem.value) return;
    fitManager.removeCharge(props.type, props.index);
}

function onDragStart(e: DragEvent) {
    const c = computedItem.value;
    if (!c || !e.dataTransfer) return;
    e.dataTransfer.setData("application/esf-type-id", String(c.type_id));
    e.dataTransfer.setData("application/esf-slot-type", c.slot.type);
    e.dataTransfer.setData("application/esf-slot-index", String(c.slot.index ?? ""));
}

function onDragOver(e: DragEvent) {
    e.preventDefault();
}

function parseNumber(raw: string): number | undefined {
    const n = Number.parseInt(raw, 10);
    return Number.isInteger(n) ? n : undefined;
}

function onDrop(e: DragEvent) {
    e.preventDefault();
    if (!e.dataTransfer) return;

    const draggedTypeId = parseNumber(e.dataTransfer.getData("application/esf-type-id"));
    const draggedSlotIndex = parseNumber(e.dataTransfer.getData("application/esf-slot-index"));
    const draggedSlotType = e.dataTransfer.getData("application/esf-slot-type");

    if (draggedTypeId === undefined || draggedSlotType === "DroneBay") return;

    // Charges land directly on this slot regardless of slot type.
    if (draggedSlotType === "Charge") {
        fitManager.setCharge(props.type, props.index, draggedTypeId);
        return;
    }

    // Modules only accept matching slot types.
    if (draggedSlotType !== props.type) return;

    if (draggedSlotIndex !== undefined) {
        // Swap between two fitted slots of the same type.
        fitManager.swapModule(props.type, draggedSlotIndex, props.type, props.index);
    } else {
        // Fresh from the listing — place at this slot.
        fitManager.addModule(draggedTypeId, props.type, { index: props.index });
    }
}

// Context menu
const ctxMenu = reactive({
    visible: false,
    x: 0,
    y: 0,
});

function onContextMenu(e: MouseEvent) {
    if (props.readOnly || !props.fittable) return;
    e.preventDefault();
    e.stopPropagation();
    ctxMenu.visible = true;
    ctxMenu.x = e.clientX;
    ctxMenu.y = e.clientY;
}

const itemIconSrc = computed(() => {
    const c = computedItem.value;
    if (!c) return undefined;
    const iconTypeId = c.charge?.type_id ?? c.type_id;
    return `https://images.evetech.net/types/${iconTypeId}/icon?size=64`;
});

interface TooltipData {
    name: string;
    chargeName: string;
    stats: { label: string; value: string }[];
    bonuses: { label: string; value: string }[];
}

/**
 * Curated list of module bonus attributes. The dogma engine stores each type's
 * bonus source values on its attribute map; all we need to do is look them up
 * and format them.
 *
 * Value conventions in the SDE:
 *   - `fromOne`: multiplier around 1.0 (0.898 = -10.2%, 1.099 = +9.9%)
 *   - `abs`:     percentage stored directly (e.g. -60 for -60%)
 *   - `flat`:    raw number, shown with a unit
 *
 * `invertSign` flips the displayed sign — useful for "less is better" stats
 * like cycle time (0.9× cycle = faster rate of fire → show as +10% ROF).
 */
interface BonusSpec {
    name: string;
    label: string;
    kind: "fromOne" | "abs" | "flat";
    unit?: string;
    invertSign?: boolean;
    /**
     * Skip this attribute when the value falls outside these bounds. Used for
     * multiplier attributes like `damageMultiplier` that are also carried by
     * non-bonus modules with compound values (e.g. a Drone Damage Amp reports
     * its stacked drone-damage figure through the same attribute, ~3.19×,
     * which would otherwise render as "+219% Damage").
     */
    validRange?: [number, number];
}

const MODULE_BONUSES: BonusSpec[] = [
    // Damage upgrades (Heat Sinks, Mag Stabs, Gyros, BCS, Entropic RS)
    // — gated to sane multiplier range so compound values on DDAs etc. don't
    //   masquerade as a +200%-ish "damage" bonus.
    { name: "damageMultiplier", label: "Damage", kind: "fromOne", validRange: [0.5, 1.5] },
    { name: "speedMultiplier", label: "Rate of fire", kind: "fromOne", invertSign: true, validRange: [0.5, 1.5] },
    { name: "droneDamageBonus", label: "Drone damage", kind: "abs" },

    // Missile tweaks (rigs, BCS)
    { name: "missileDamageMultiplier", label: "Missile dmg", kind: "fromOne" },
    { name: "missileVelocityMultiplier", label: "Missile vel", kind: "fromOne" },
    { name: "aoeVelocityBonus", label: "Explosion vel", kind: "abs" },
    { name: "aoeCloudSizeBonus", label: "Explosion rad", kind: "abs" },

    // Turret range/tracking tweaks
    { name: "maxRangeBonus", label: "Optimal", kind: "abs" },
    { name: "falloffBonus", label: "Falloff", kind: "abs" },
    { name: "trackingSpeedBonus", label: "Tracking", kind: "abs" },

    // HP / capacity
    { name: "armorHPBonusAdd", label: "Armor HP", kind: "flat", unit: "hp" },
    { name: "armorHpBonus", label: "Armor HP", kind: "abs" },
    { name: "hpMultiplier", label: "Hull HP", kind: "fromOne" },
    { name: "capacityBonus", label: "Shield HP", kind: "flat", unit: "hp" },
    { name: "shieldCapacityMultiplier", label: "Shield HP", kind: "fromOne" },
    { name: "shieldBoostMultiplier", label: "Shield boost", kind: "fromOne" },

    // Resistances (hardeners & rigs)
    { name: "armorEmDamageResistanceBonus", label: "EM armor res", kind: "fromOne", invertSign: true },
    { name: "armorThermalDamageResistanceBonus", label: "Therm armor res", kind: "fromOne", invertSign: true },
    { name: "armorKineticDamageResistanceBonus", label: "Kin armor res", kind: "fromOne", invertSign: true },
    { name: "armorExplosiveDamageResistanceBonus", label: "Exp armor res", kind: "fromOne", invertSign: true },
    { name: "shieldEmDamageResistanceBonus", label: "EM shield res", kind: "fromOne", invertSign: true },
    { name: "shieldThermalDamageResistanceBonus", label: "Therm shield res", kind: "fromOne", invertSign: true },
    { name: "shieldKineticDamageResistanceBonus", label: "Kin shield res", kind: "fromOne", invertSign: true },
    { name: "shieldExplosiveDamageResistanceBonus", label: "Exp shield res", kind: "fromOne", invertSign: true },

    // Navigation
    { name: "speedFactor", label: "Speed", kind: "abs" },
    { name: "signatureRadiusBonus", label: "Sig radius", kind: "abs" },
    { name: "maxVelocityBonus", label: "Velocity", kind: "abs" },
    { name: "agilityBonus", label: "Agility", kind: "abs" },

    // CPU / PG (engineering rigs, upgrades)
    { name: "cpuMultiplier", label: "CPU", kind: "fromOne" },
    { name: "powerMultiplier", label: "Powergrid", kind: "fromOne" },
    { name: "cpuOutputBonus2", label: "CPU", kind: "abs" },
    { name: "powerOutputBonus2", label: "Powergrid", kind: "abs" },

    // Capacitor
    { name: "capacitorCapacityMultiplier", label: "Cap capacity", kind: "fromOne" },
    { name: "rechargeRateMultiplier", label: "Cap recharge", kind: "fromOne", invertSign: true },

    // Targeting
    { name: "scanResolutionBonus", label: "Scan res", kind: "abs" },
    { name: "maxTargetRangeBonus", label: "Lock range", kind: "abs" },
    { name: "maxLockedTargetsBonus", label: "Max targets", kind: "flat" },

    // Active modules — raw amounts (repairers, boosters, neuts, transfers)
    { name: "armorDamageAmount", label: "Armor rep", kind: "flat", unit: "hp" },
    { name: "shieldBonus", label: "Shield boost", kind: "flat", unit: "hp" },
    { name: "structureDamageAmount", label: "Hull rep", kind: "flat", unit: "hp" },
    { name: "energyNeutralizerAmount", label: "Neut", kind: "flat", unit: "GJ" },
    { name: "powerTransferAmount", label: "Energy xfer", kind: "flat", unit: "GJ" },

    // EWar strength (redundant with EWar panel but handy in hover)
    { name: "warpScrambleStrength", label: "Warp disrupt str", kind: "flat" },

    // Cap boosters
    { name: "capacitorBonus", label: "Cap boost", kind: "flat", unit: "GJ" },

    // Probe launchers & scanning
    { name: "scanStrengthBonus", label: "Probe str", kind: "abs" },
    { name: "maxScanDeviationModifier", label: "Scan deviation", kind: "abs" },
];

const tooltipData = computed<TooltipData | null>(() => {
    const c = computedItem.value;
    if (!c || !sde.value) return null;
    const sdeData = sde.value;
    const name = sdeData.types.get(c.type_id)?.name ?? "";
    const chargeName = c.charge ? sdeData.types.get(c.charge.type_id)?.name ?? "" : "";

    const attrVal = (item: typeof c, attr: string): number | null => {
        const id = sdeData.attributeNameToId.get(attr);
        if (id === undefined) return null;
        const raw = item.attributes.get(id);
        return raw?.value ?? raw?.base_value ?? null;
    };

    const km = (m: number) => (m / 1000).toFixed(m >= 10_000 ? 0 : 1);
    const stats: { label: string; value: string }[] = [];

    const optimal = attrVal(c, "maxRange");
    const falloff = attrVal(c, "falloff");
    if (optimal != null && optimal > 0) {
        if (falloff != null && falloff > 0) {
            stats.push({ label: "Optimal", value: `${km(optimal)} km` });
            stats.push({ label: "Falloff", value: `${km(falloff)} km` });
        } else {
            stats.push({ label: "Range", value: `${km(optimal)} km` });
        }
    } else if (c.charge) {
        const vel = attrVal(c.charge, "maxVelocity");
        const flight = attrVal(c.charge, "explosionDelay");
        if (vel && flight && vel > 0 && flight > 0) {
            stats.push({ label: "Flight range", value: `${((vel * flight) / 1_000_000).toFixed(1)} km` });
        }
    }

    const tracking = attrVal(c, "trackingSpeed");
    if (tracking != null && tracking > 0) {
        stats.push({ label: "Tracking", value: `${tracking.toFixed(3)} rad/s` });
    }

    const duration = attrVal(c, "duration");
    if (duration != null && duration > 0) {
        stats.push({ label: "Cycle", value: `${(duration / 1000).toFixed(1)} s` });
    }

    // Flat damage per cycle — doomsdays, smartbombs, bombs carry em/therm/kin/exp
    // directly on the module. Turrets (charge-based damage) and missile launchers
    // are skipped because their damage lives on the charge and is already
    // reflected in the ship-level DPS elsewhere. Compact formatter for big
    // numbers (Titan DDs deal millions of HP per cycle).
    const compact = (n: number): string => {
        const abs = Math.abs(n);
        if (abs >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
        if (abs >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
        return n.toFixed(0);
    };
    if (!c.charge) {
        const em = attrVal(c, "emDamage") ?? 0;
        const therm = attrVal(c, "thermalDamage") ?? 0;
        const kin = attrVal(c, "kineticDamage") ?? 0;
        const exp = attrVal(c, "explosiveDamage") ?? 0;
        const total = em + therm + kin + exp;
        if (total > 0) {
            stats.push({ label: "Damage", value: `${compact(total)} hp` });
            const parts: string[] = [];
            if (em) parts.push(`EM ${compact(em)}`);
            if (therm) parts.push(`Th ${compact(therm)}`);
            if (kin) parts.push(`Kn ${compact(kin)}`);
            if (exp) parts.push(`Ex ${compact(exp)}`);
            if (parts.length >= 2) {
                stats.push({ label: "", value: parts.join(" · ") });
            }
            if (duration != null && duration > 0) {
                stats.push({ label: "DPS", value: `${(total / (duration / 1000)).toFixed(1)}` });
            }
        }
    }

    // AoE — smartbombs, bombs, doomsdays, burst jammers.
    const aoeCloud = attrVal(c, "aoeCloudSize");
    if (aoeCloud != null && aoeCloud > 0) {
        stats.push({ label: "AoE radius", value: `${aoeCloud.toFixed(0)} m` });
    }
    const aoeVelocity = attrVal(c, "aoeVelocity");
    if (aoeVelocity != null && aoeVelocity > 0) {
        stats.push({ label: "AoE velocity", value: `${aoeVelocity.toFixed(0)} m/s` });
    }

    // Bonuses — what the module does. Scan the curated list and format each
    // non-identity value. Identity: 1.0 for multipliers, 0 for abs/flat.
    const formatFlat = (v: number, unit: string | undefined): string => {
        const sign = v > 0 ? "+" : "";
        const abs = Math.abs(v);
        const precision = abs >= 100 ? 0 : abs >= 10 ? 1 : 2;
        return `${sign}${v.toFixed(precision)}${unit ? ` ${unit}` : ""}`;
    };
    const formatPct = (v: number): string => {
        const sign = v > 0 ? "+" : "";
        return `${sign}${v.toFixed(2)}%`;
    };

    const bonuses: { label: string; value: string }[] = [];

    // Weapons carry `damageMultiplier` and `speedMultiplier` as their own
    // intrinsic damage and cycle time — not as bonuses applied to other
    // modules. Detect weapons by their "fires" effects and skip those
    // attributes when collecting bonuses.
    const isWeapon = (() => {
        const td = sdeData.typeDogma.get(c.type_id);
        const typeEffects = (td?.dogmaEffects ?? []) as Array<{ effectID: number }>;
        if (typeEffects.length === 0) return false;
        const weaponEffectNames = new Set(["projectileFired", "targetAttack", "useMissiles"]);
        const weaponEffectIds = new Set<number>();
        for (const [id, eff] of sdeData.dogmaEffects) {
            if (eff?.name && weaponEffectNames.has(eff.name)) weaponEffectIds.add(id);
        }
        return typeEffects.some(e => weaponEffectIds.has(e.effectID));
    })();
    const WEAPON_INTRINSIC_ATTRS = new Set(["damageMultiplier", "speedMultiplier"]);

    // Resistance handling — Damage Controls, active hardeners, and rigs modify
    // the ship's `*DamageResonance` attributes directly. The source values live
    // on the module, so we iterate them here. Collapse identical values across
    // all 4 damage types into a single line so DCU isn't 12 lines tall.
    const LAYERS: Array<[string, string]> = [
        ["armor", "Armor"],
        ["shield", "Shield"],
        ["hull", "Hull"],
    ];
    const DMG = ["Em", "Thermal", "Kinetic", "Explosive"] as const;
    const DMG_SHORT = ["EM", "Therm", "Kin", "Exp"] as const;
    for (const [prefix, label] of LAYERS) {
        const attrName = (d: string) =>
            prefix === "hull"
                ? `${d.charAt(0).toLowerCase()}${d.slice(1)}DamageResonance`
                : `${prefix}${d}DamageResonance`;
        const vals = DMG.map(d => attrVal(c, attrName(d)));
        const present = vals
            .map((v, i) => ({ v, i }))
            .filter((x): x is { v: number; i: number } => x.v != null && x.v !== 1);
        if (present.length === 0) continue;
        const firstVal = present[0]!.v;
        const allEqual = present.every(x => x.v === firstVal);
        if (allEqual) {
            const pct = -(firstVal - 1) * 100;
            bonuses.push({ label: `${label} res`, value: formatPct(pct) });
        } else {
            for (const { v, i } of present) {
                const pct = -(v - 1) * 100;
                bonuses.push({ label: `${label} ${DMG_SHORT[i]}`, value: formatPct(pct) });
            }
        }
    }

    for (const spec of MODULE_BONUSES) {
        if (isWeapon && WEAPON_INTRINSIC_ATTRS.has(spec.name)) continue;
        const v = attrVal(c, spec.name);
        if (v == null) continue;
        if (spec.validRange && (v < spec.validRange[0] || v > spec.validRange[1])) continue;
        let pct: number | null = null;
        let formatted: string;
        switch (spec.kind) {
            case "fromOne":
                if (v === 1) continue;
                pct = (v - 1) * 100;
                if (spec.invertSign) pct = -pct;
                formatted = formatPct(pct);
                break;
            case "abs":
                if (v === 0) continue;
                pct = spec.invertSign ? -v : v;
                formatted = formatPct(pct);
                break;
            case "flat":
                if (v === 0) continue;
                formatted = formatFlat(v, spec.unit);
                break;
        }
        bonuses.push({ label: spec.label, value: formatted });
    }

    // Generic fallback — catch any *Bonus / *Multiplier / *Add / *Modifier
    // attribute we haven't curated. Common rigs (Core Defense Field Extender,
    // Drone Link Augmentors, Projectile Ambit Extenders, etc.) store their
    // bonus in attributes the curated list can't keep up with. This gives them
    // at least a humanised label + value so the module isn't silent.
    const coveredNames = new Set(MODULE_BONUSES.map(b => b.name));
    // Core attributes that aren't bonuses, or are handled elsewhere.
    const NON_BONUS_ATTRS = new Set<string>([
        "cpu", "power", "mass", "volume", "capacity", "radius", "metaLevel",
        "techLevel", "hp", "armorHP", "shieldCapacity", "capacitorCapacity",
        "rechargeRate", "shieldRechargeRate",
        "maxRange", "falloff", "trackingSpeed", "duration",
        "signatureRadius", "scanResolution", "maxTargetRange", "maxLockedTargets",
        "agility", "maxVelocity", "warpSpeedMultiplier",
        "damageMultiplier", "speedMultiplier",
        "hiSlots", "medSlots", "lowSlots", "rigSlots", "maxSubSystems",
        "droneBandwidth", "droneCapacity", "droneBandwidthUsed", "droneCapacityNeeded",
        "upgradeCapacity", "upgradeCost",
        "aimedLaunch", "aoeVelocity", "aoeCloudSize",
        "heatDamage", "heatAbsorbtionRateModifier", "heatAttenuationHi",
        "heatAttenuationLow", "heatAttenuationMed", "heatDissipationRateHi",
        "heatDissipationRateLow", "heatDissipationRateMed",
        "heatGenerationMultiplier",
        "thermalDamage", "kineticDamage", "emDamage", "explosiveDamage",
        "explosionRange", "trackingAudio",
        "activationTime", "reloadTime",
        // Prerequisites / fit checks
        "requiredSkill1", "requiredSkill2", "requiredSkill3", "requiredSkill4",
        "requiredSkill1Level", "requiredSkill2Level", "requiredSkill3Level", "requiredSkill4Level",
        "fitsToShipType",
        "isCapitalSize", "disallowAssistance", "disallowOffensiveModifiers",
        // Resistance values themselves (resonance is handled above)
        "armorUniformity", "shieldUniformity", "structureUniformity",
    ]);
    // Build attrId → name once
    const attrIdToName = new Map<number, string>();
    for (const [n, id] of sdeData.attributeNameToId) attrIdToName.set(id, n);

    // Humanize an attribute name: camelCase → "Title Case", trim common suffix
    const humanize = (attrName: string): string => {
        let s = attrName.replace(/([A-Z])/g, " $1").trim();
        s = s.replace(/\s+(Bonus|Multiplier|Add|Modifier)$/i, "");
        // Capitalize first letter, lowercase the rest for tight labels
        return s.charAt(0).toUpperCase() + s.slice(1);
    };

    // EVE SDE unit IDs — we only care about the ones that change formatting.
    // Full list lives in the SDE but these are the ones that matter for bonuses.
    const UNIT_PERCENT = 105;            // Percentage
    const UNIT_METER = 1;                 // Length, m
    const UNIT_ABS_PERCENT = 127;         // Absolute percentage
    const UNIT_INVERSE_ABS_PERCENT = 108; // inverseAbsolutePercent (resistance-style)
    const UNIT_MULTIPLIER = 104;          // Multiplier
    const UNIT_INVERSE_MULTIPLIER = 111;  // inversedModifier
    const UNIT_HP = 113;                  // hitpoints
    const UNIT_GJ = 112;                  // capacitorUnits (GJ)
    const UNIT_SECONDS = 3;               // Time, s
    const UNIT_MS = 101;                  // milliseconds

    const formatByUnit = (v: number, unitID: number | undefined): string | null => {
        switch (unitID) {
            case UNIT_METER: {
                if (Math.abs(v) >= 1000) {
                    const km = v / 1000;
                    return `${v > 0 ? "+" : ""}${km.toFixed(km >= 10 ? 0 : 1)} km`;
                }
                return formatFlat(v, "m");
            }
            case UNIT_HP: return formatFlat(v, "hp");
            case UNIT_GJ: return formatFlat(v, "GJ");
            case UNIT_SECONDS: return formatFlat(v, "s");
            case UNIT_MS: return formatFlat(v / 1000, "s");
            case UNIT_PERCENT:
            case UNIT_ABS_PERCENT:
                return formatPct(v);
            case UNIT_INVERSE_ABS_PERCENT:
                return formatPct(-v);
            case UNIT_MULTIPLIER:
                if (v === 1) return null;
                return formatPct((v - 1) * 100);
            case UNIT_INVERSE_MULTIPLIER:
                if (v === 1) return null;
                return formatPct(-(v - 1) * 100);
            default:
                return null;
        }
    };

    for (const [attrId, attr] of c.attributes) {
        const attrName = attrIdToName.get(attrId);
        if (!attrName) continue;
        if (coveredNames.has(attrName)) continue;
        if (NON_BONUS_ATTRS.has(attrName)) continue;
        if (/DamageResonance$/.test(attrName)) continue;          // handled above
        if (/^(canFit|fitsToShip|requiredSkill)/.test(attrName)) continue;
        if (/canFitShipGroup/.test(attrName)) continue;

        // Only surface attributes whose name looks like a bonus source.
        const isMultiplierName = /Multiplier$/.test(attrName);
        const isBonusName = /Bonus$/.test(attrName) || /Modifier$/.test(attrName);
        const isAddName = /Add$/.test(attrName);
        if (!isMultiplierName && !isBonusName && !isAddName) continue;

        const v = attr.value ?? attr.base_value;
        if (v == null) continue;
        if (isMultiplierName && v === 1) continue;
        if (v === 0) continue;

        // Prefer the SDE's declared unit — it's right for 99% of attributes.
        // Fall back to name-based inference when unitID is missing (shouldn't
        // happen in published attrs but be defensive).
        const unitID = sdeData.dogmaAttributes.get(attrId)?.unitID as number | undefined;
        let formatted: string | null = formatByUnit(v, unitID);
        if (formatted == null) {
            if (isMultiplierName) {
                formatted = formatPct((v - 1) * 100);
            } else if (isBonusName || isAddName) {
                // Heuristic: large round values are usually distance in meters.
                if (Math.abs(v) >= 500 && /Range|Distance/.test(attrName)) {
                    const km = v / 1000;
                    formatted = `${v > 0 ? "+" : ""}${km.toFixed(km >= 10 ? 0 : 1)} km`;
                } else if (isAddName) {
                    formatted = formatFlat(v, undefined);
                } else {
                    formatted = formatPct(v);
                }
            }
        }
        if (formatted == null) continue;
        bonuses.push({ label: humanize(attrName), value: formatted });
    }

    return { name, chargeName, stats, bonuses };
});

// Hover tooltip — opens fast (~60ms) and anchors to the hovered element rather
// than the cursor, so it lands directly under the module icon. pointer-events:
// none on the teleported node keeps right-click and drag events flowing to the
// underlying <img>.
const tooltip = reactive({ visible: false, x: 0, y: 0, placement: "below" as "below" | "above" });
let tooltipTimer: ReturnType<typeof setTimeout> | null = null;

function anchorToElement(el: HTMLElement) {
    const rect = el.getBoundingClientRect();
    const centerX = rect.left + rect.width / 2;
    // Flip above the icon if it would clip the bottom of the viewport.
    const vh = typeof window !== "undefined" ? window.innerHeight : 800;
    if (rect.bottom + 160 > vh) {
        tooltip.placement = "above";
        tooltip.y = rect.top - 8;
    } else {
        tooltip.placement = "below";
        tooltip.y = rect.bottom + 8;
    }
    tooltip.x = centerX;
}

function showTooltipSoon(e: MouseEvent) {
    const target = e.currentTarget as HTMLElement | null;
    if (!target) return;
    anchorToElement(target);
    if (tooltipTimer) clearTimeout(tooltipTimer);
    tooltipTimer = setTimeout(() => { tooltip.visible = true; }, 60);
}

function hideTooltip() {
    if (tooltipTimer) { clearTimeout(tooltipTimer); tooltipTimer = null; }
    tooltip.visible = false;
}

onUnmounted(() => {
    if (tooltipTimer) clearTimeout(tooltipTimer);
});
</script>

<template>
    <!-- Not fittable and nothing fitted: render a muted wedge. Still
         attach dragover/drop handlers so the event is EATEN here rather
         than sneaking through to HullDraggable (which would then happily
         stuff a 9th high-slot module into slot 9 on an 8-slot hull via
         firstFreeIndex). .prevent on dragover is required to mark the
         element as a drop zone at all; .stop.prevent on drop ensures
         both the browser default AND any bubble-up to an ancestor
         handler are blocked. The final useFitManager.addModule check
         is still the ground truth — this is defense in depth. -->
    <div
        v-if="!hasItem && !props.fittable"
        :class="styles.slotOuter"
        data-hasitem="false"
        @dragover.prevent
        @drop.stop.prevent
    >
        <div :class="styles.slot" data-state="Unavailable">
            <!-- Non-main slots still need the <use> reference to render something -->
            <svg
                viewBox="235 40 52 50"
                xmlns="http://www.w3.org/2000/svg"
                :class="styles.ringInner"
                preserveAspectRatio="xMidYMin slice"
            >
                <use href="#slot" />
            </svg>
        </div>
    </div>

    <div v-else :class="styles.slotOuter" :data-hasitem="hasItem">
        <div
            :class="styles.slot"
            :data-state="displayState"
            @click="cycleState"
            @contextmenu="onContextMenu"
            @dragover="onDragOver"
            @drop="onDrop"
        >
            <!-- Shared SVG defs, rendered exactly once on the 'main' slot. -->
            <template v-if="props.main">
                <svg
                    viewBox="235 40 52 50"
                    xmlns="http://www.w3.org/2000/svg"
                    preserveAspectRatio="xMidYMin slice"
                    style="display: none"
                >
                    <g id="slot">
                        <path
                            :style="{ fillOpacity: 0.1, strokeWidth: 1, strokeOpacity: 0.5 }"
                            d="M 243 46 A 210 210 0 0 1 279 46.7 L 276 84.7 A 172 172 0 0 0 246 84 L 243 46"
                        />
                    </g>
                    <g id="slot-active">
                        <path
                            :style="{ fillOpacity: 0.6, strokeWidth: 1 }"
                            d="M 250 84 L 254 79 L 268 79 L 272 84"
                        />
                    </g>
                    <g id="slot-passive">
                        <path
                            :style="{ strokeWidth: 1 }"
                            d="M 245 48 A 208 208 0 0 1 250 47.5 L 248 50 L 245 50"
                        />
                        <path
                            :style="{ strokeWidth: 1 }"
                            d="M 277.5 48.5 A 208 208 0 0 0 273 48 L 275 50.5 L 277.5 50.5"
                        />
                        <path
                            :style="{ strokeWidth: 1 }"
                            d="M 247 82 A 170 170 0 0 1 252 82 L 250 80 L 246.8 80"
                        />
                        <path
                            :style="{ strokeWidth: 1 }"
                            d="M 275 82.5 A 170 170 0 0 0 270 82 L 272 80 L 275.2 80"
                        />
                    </g>
                </svg>
                <svg viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg" style="display: none">
                    <g id="uncharge">
                        <path :style="{ fill: 'none', strokeWidth: 1 }" d="M 4 6 A 8 8 0 1 1 4 14" />
                        <path :style="{ fill: 'none', strokeWidth: 1 }" d="M 11 6 L 6 10 L 11 14" />
                        <path :style="{ fill: 'none', strokeWidth: 1 }" d="M 6 10 L 16 10" />
                    </g>
                    <g id="unfit">
                        <path :style="{ fill: 'none', strokeWidth: 1 }" d="M 4 6 A 8 8 0 1 1 4 14" />
                        <path :style="{ fill: 'none', strokeWidth: 1 }" d="M 11 6 L 6 10 L 11 14" />
                        <path :style="{ fill: 'none', strokeWidth: 1 }" d="M 6 10 L 16 10" />
                    </g>
                    <g id="offline">
                        <path :style="{ fill: 'none', strokeWidth: 1 }" d="M 12 4 A 8 8 0 1 1 6 4" />
                        <path :style="{ fill: 'none', strokeWidth: 1 }" d="M 9 2 L 9 12" />
                    </g>
                </svg>
            </template>

            <svg
                viewBox="235 40 52 50"
                xmlns="http://www.w3.org/2000/svg"
                :class="styles.ringInner"
                preserveAspectRatio="xMidYMin slice"
            >
                <use href="#slot" />
                <use
                    v-if="props.fittable && !isPreview && hasItem && isActiveType"
                    href="#slot-active"
                />
                <use
                    v-if="props.fittable && !isPreview && hasItem && !isActiveType"
                    href="#slot-passive"
                />
            </svg>

            <!-- Fitted module icon, or placeholder slot glyph if empty. -->
            <div v-if="hasItem" :class="styles.slotImage">
                <img
                    :src="itemIconSrc"
                    :class="{ [styles.preview]: isPreview }"
                    draggable="true"
                    @dragstart="(e: DragEvent) => { onDragStart(e); hideTooltip(); }"
                    @mouseenter="showTooltipSoon"
                    @mouseleave="hideTooltip"
                />
            </div>
            <div v-else-if="placeholderIconName !== undefined" :class="styles.slotImagePlaceholder">
                <ShipFitIcon :name="placeholderIconName" />
            </div>
        </div>

        <FitContextMenu
            :visible="ctxMenu.visible"
            :x="ctxMenu.x"
            :y="ctxMenu.y"
            :type-id="computedItem?.type_id ?? 0"
            :slot-type="props.type"
            :slot-index="props.index"
            :current-state="computedItem?.state ?? 'Passive'"
            :has-charge="computedItem?.charge != null"
            :empty="!hasItem"
            @close="ctxMenu.visible = false"
        />

        <Teleport to="body">
            <div
                v-if="tooltip.visible && tooltipData"
                class="fixed pointer-events-none z-[9999] rounded-md bg-black/90 backdrop-blur-sm border border-white/10 px-3 py-2 text-xs text-gray-200 shadow-xl min-w-[180px] max-w-[280px]"
                :style="{
                    left: `${tooltip.x}px`,
                    top: `${tooltip.y}px`,
                    transform: tooltip.placement === 'above' ? 'translate(-50%, -100%)' : 'translate(-50%, 0)',
                }"
            >
                <div class="font-semibold text-white">{{ tooltipData.name }}</div>
                <div v-if="tooltipData.chargeName" class="text-[11px] text-blue-400/80 mt-0.5">
                    {{ tooltipData.chargeName }}
                </div>
                <div v-if="tooltipData.stats.length" class="mt-1.5 flex flex-col gap-0.5 border-t border-white/[0.08] pt-1.5">
                    <div v-for="s in tooltipData.stats" :key="s.label" class="flex justify-between gap-4">
                        <span class="text-gray-500">{{ s.label }}</span>
                        <span class="tabular-nums text-gray-200">{{ s.value }}</span>
                    </div>
                </div>
                <div v-if="tooltipData.bonuses.length" class="mt-1.5 flex flex-col gap-0.5 border-t border-white/[0.08] pt-1.5">
                    <div v-for="b in tooltipData.bonuses" :key="b.label" class="flex justify-between gap-4">
                        <span class="text-gray-500">{{ b.label }}</span>
                        <span
                            class="tabular-nums"
                            :class="b.value.startsWith('-') ? 'text-red-400/90' : 'text-green-400/90'"
                        >{{ b.value }}</span>
                    </div>
                </div>
            </div>
        </Teleport>
    </div>
</template>
