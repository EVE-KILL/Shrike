<script setup lang="ts">
/**
 * Right-side stats panel — 6 collapsible categories showing the
 * current fit's capacitor, offense, defense, targeting, navigation,
 * and drones stats.
 *
 * Restyled to match the EK site design: single card container
 * wrapping the whole panel with subtle dividers between categories,
 * blue uppercase category headers with lucide chevrons, EK color
 * tokens throughout. Functional port from @eveshipfit/react's
 * ShipStatistics.tsx preserved — same attribute names, same
 * formatting rules, same Navigation/Drones hiding for structures.
 */

import Category from "./Category.vue";
import CategoryLine from "./CategoryLine.vue";
import CharAttribute from "./CharAttribute.vue";
import Resistance from "./Resistance.vue";
import ShipAttribute from "./ShipAttribute.vue";
import ShipFitIcon from "../ShipFitIcon.vue";

withDefaults(defineProps<{
    expandDetails?: boolean;
}>(), {
    expandDetails: false,
});

const { sde } = useEveData();
const { stats } = useFitStatistics();
const { currentFit } = useCurrentFit();

const isStructure = computed(() => {
    const shipTypeId = currentFit.value?.shipTypeId;
    if (shipTypeId === undefined || !sde.value) return false;
    return sde.value.types.get(shipTypeId)?.categoryID === 65;
});

// ---------- Weapon range (optimal + falloff + tracking) ------------
//
// The dogma engine computes per-item attributes for every fitted
// module and publishes them on `stats.items[].attributes`. Turrets
// expose `maxRange` (optimal), `falloff`, and `trackingSpeed`;
// missile launchers expose `maxRange` too (the engine resolves it
// from the loaded missile's flight time × velocity).
//
// We pick the first item with a `turretFitted` or `launcherFitted`
// effect. For homogeneous weapon fits (8 Mega Pulse IIs with
// identical ammo) this is the canonical read. Mixed-weapon fits
// get "primary weapon" coverage; per-module detail still lives on
// hover in the ring.
const weaponRangeEffectIds = computed(() => {
    const sdeData = sde.value;
    if (!sdeData) return null;
    let turretFitted: number | undefined;
    let launcherFitted: number | undefined;
    for (const [id, eff] of sdeData.dogmaEffects) {
        if (!eff?.name) continue;
        if (eff.name === "turretFitted") turretFitted = id;
        else if (eff.name === "launcherFitted") launcherFitted = id;
        if (turretFitted !== undefined && launcherFitted !== undefined) break;
    }
    return { turretFitted, launcherFitted };
});

const primaryWeapon = computed(() => {
    const sdeData = sde.value;
    const s = stats.value;
    const ids = weaponRangeEffectIds.value;
    if (!sdeData || !s || !ids) return null;

    const attr = (name: string) => sdeData.attributeNameToId.get(name);
    const maxRangeAttr = attr("maxRange");
    const falloffAttr = attr("falloff");
    const trackingAttr = attr("trackingSpeed");
    if (maxRangeAttr === undefined) return null;

    const hasEffect = (typeId: number, effectId: number | undefined): boolean => {
        if (effectId === undefined) return false;
        const td = sdeData.typeDogma.get(typeId);
        const effects = (td?.dogmaEffects ?? []) as Array<{ effectID: number }>;
        return effects.some((e) => e.effectID === effectId);
    };

    // First pass: turrets. Second pass: missile launchers. Turrets
    // win because they're more common and their range triple
    // (optimal + falloff + tracking) is the more information-dense
    // readout — missiles only carry optimal.
    let chosen: typeof s.items[number] | null = null;
    let kind: "turret" | "launcher" | null = null;
    for (const item of s.items) {
        if (hasEffect(item.type_id, ids.turretFitted)) {
            chosen = item;
            kind = "turret";
            break;
        }
    }
    if (!chosen) {
        for (const item of s.items) {
            if (hasEffect(item.type_id, ids.launcherFitted)) {
                chosen = item;
                kind = "launcher";
                break;
            }
        }
    }
    if (!chosen || !kind) return null;

    let rawMax = chosen.attributes.get(maxRangeAttr)?.value ?? 0;
    const rawFalloff = falloffAttr !== undefined
        ? chosen.attributes.get(falloffAttr)?.value ?? 0
        : 0;
    const rawTracking = trackingAttr !== undefined
        ? chosen.attributes.get(trackingAttr)?.value ?? 0
        : 0;

    // Missile launchers don't have maxRange — the flight range comes
    // from the loaded charge: maxVelocity × explosionDelay / 1000.
    // explosionDelay is the missile flight time in ms.
    if (kind === "launcher" && rawMax === 0 && chosen.charge) {
        const velAttr = attr("maxVelocity");
        const expDelayAttr = attr("explosionDelay");
        if (velAttr !== undefined && expDelayAttr !== undefined) {
            const vel = chosen.charge.attributes.get(velAttr)?.value ?? 0;
            const flightTime = chosen.charge.attributes.get(expDelayAttr)?.value ?? 0;
            rawMax = (vel * flightTime) / 1000;
        }
    }

    // Format km with 2 decimals; tracking as rad/s with 4 decimals.
    const km = (m: number) => (m / 1000).toLocaleString("en", {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
    });

    return {
        kind,
        optimalKm: km(Number(rawMax)),
        falloffKm: km(Number(rawFalloff)),
        tracking: Number(rawTracking).toLocaleString("en", {
            minimumFractionDigits: 4,
            maximumFractionDigits: 4,
        }),
        hasFalloff: kind === "turret" && rawFalloff > 0,
        hasTracking: kind === "turret" && rawTracking > 0,
    };
});

const capacitorState = computed(() => {
    if (!sde.value || !stats.value) return "Stable";
    const id = sde.value.attributeNameToId.get("capacitorDepletesIn");
    if (id === undefined) return "Stable";
    const depletesIn = stats.value.hull.attributes.get(id)?.value;
    if (depletesIn === undefined || depletesIn === null || depletesIn < 0) return "Stable";
    const hours = Math.floor(depletesIn / 3600);
    const minutes = Math.floor((depletesIn % 3600) / 60);
    const seconds = Math.floor(depletesIn % 60);
    const pad = (n: number) => n.toString().padStart(2, "0");
    return `Depletes in ${pad(hours)}:${pad(minutes)}:${pad(seconds)}`;
});

const isCapacitorStable = computed(() => capacitorState.value === "Stable");

// ---------- Remote assistance / warfare stats -----------------
//
// These are EK-custom attributes computed post-engine in core.ts.
// The section is hidden entirely when no remote modules are fitted.
// Each line only renders when its value is non-zero.

const REMOTE_ATTRS = [
    "remoteShieldBoostRate",
    "remoteArmorRepairRate",
    "remoteHullRepairRate",
    "energyNeutralizerRate",
    "energyNosferatuRate",
    "remoteCapTransferRate",
] as const;

function readHullAttr(name: string): number {
    const s = stats.value;
    const sdeData = sde.value;
    if (!s || !sdeData) return 0;
    const id = sdeData.attributeNameToId.get(name);
    if (id === undefined) return 0;
    const a = s.hull.attributes.get(id);
    return a?.value ?? 0;
}

const hasRemoteStats = computed(() => REMOTE_ATTRS.some((n) => readHullAttr(n) > 0));
const hasDamage = computed(() => readHullAttr("damagePerSecondWithReload") > 0);

// ---------- EWar / disruption modules -----------------
//
// One row per fitted EWar module (grouped by type so N identical webs show as
// one line with ×N). Each row carries the primary effect strength plus range.
//
// Keyed by EVE group_id, not market group — market groups reshuffle, group ids
// don't. Unknown groups get skipped.
type EwarKind = "web" | "scram" | "ecm" | "damp" | "painter" | "td" | "burst" | "hic";
const EWAR_GROUP_KINDS: Record<number, EwarKind> = {
    65: "web",       // Stasis Webifier
    52: "scram",     // Warp Scrambler / Warp Disruptor
    201: "ecm",      // ECM
    208: "damp",     // Sensor Dampener
    211: "painter",  // Target Painter
    291: "td",       // Tracking / Weapon Disruptor
    643: "burst",    // Burst Jammer
    899: "hic",      // Warp Disruption Field Generator
};

interface EwarEntry {
    key: string;
    typeId: number;
    name: string;
    count: number;
    effect: string;
    rangeText: string;
}

const ewarModules = computed<EwarEntry[]>(() => {
    const sdeData = sde.value;
    const s = stats.value;
    if (!sdeData || !s) return [];

    const attr = (name: string) => sdeData.attributeNameToId.get(name);
    const readVal = (item: typeof s.items[number], name: string): number => {
        const id = attr(name);
        if (id === undefined) return 0;
        const a = item.attributes.get(id);
        return a?.value ?? a?.base_value ?? 0;
    };
    const km = (m: number) => (m / 1000).toFixed(m >= 10_000 ? 0 : 1);
    const pct = (v: number) => `${v > 0 ? "+" : ""}${v.toFixed(0)}%`;

    // Group fitted modules by type_id so duplicates collapse to "x N".
    type StatItem = (typeof s.items)[number];
    const groups = new Map<number, { items: StatItem[]; kind: EwarKind }>();
    for (const item of s.items) {
        const t = sdeData.types.get(item.type_id);
        if (!t) continue;
        const kind = EWAR_GROUP_KINDS[t.groupID];
        if (!kind) continue;
        let entry = groups.get(item.type_id);
        if (!entry) {
            entry = { items: [], kind };
            groups.set(item.type_id, entry);
        }
        entry.items.push(item);
    }
    if (groups.size === 0) return [];

    const out: EwarEntry[] = [];
    for (const [typeId, { items, kind }] of groups) {
        const first = items[0]!;
        const name = sdeData.types.get(typeId)?.name ?? "Unknown";
        const count = items.length;

        const optimal = readVal(first, "maxRange");
        const falloff = readVal(first, "falloff");
        const rangeText = optimal > 0
            ? (falloff > 0 ? `${km(optimal)}+${km(falloff)} km` : `${km(optimal)} km`)
            : "";

        let effect = "";
        switch (kind) {
            case "web": {
                const speedFactor = readVal(first, "speedFactor");
                effect = speedFactor !== 0 ? pct(speedFactor) : "";
                break;
            }
            case "scram": {
                const strength = readVal(first, "warpScrambleStrength");
                effect = strength > 0 ? `+${strength.toFixed(0)} str` : "";
                break;
            }
            case "ecm": {
                const vals = [
                    readVal(first, "scanGravimetricStrengthBonus"),
                    readVal(first, "scanLadarStrengthBonus"),
                    readVal(first, "scanMagnetometricStrengthBonus"),
                    readVal(first, "scanRadarStrengthBonus"),
                ];
                const peak = Math.max(...vals);
                effect = peak > 0 ? `${peak.toFixed(1)} jam str` : "";
                break;
            }
            case "damp": {
                const res = readVal(first, "scanResolutionBonus");
                const lockRange = readVal(first, "maxTargetRangeBonus");
                const parts: string[] = [];
                if (res !== 0) parts.push(`${pct(res)} res`);
                if (lockRange !== 0) parts.push(`${pct(lockRange)} lock`);
                effect = parts.join(" / ");
                break;
            }
            case "painter": {
                const sig = readVal(first, "signatureRadiusBonus");
                effect = sig !== 0 ? `${pct(sig)} sig` : "";
                break;
            }
            case "td": {
                const trk = readVal(first, "trackingSpeedBonus");
                const opt = readVal(first, "maxRangeBonus");
                const parts: string[] = [];
                if (trk !== 0) parts.push(`${pct(trk)} trk`);
                if (opt !== 0) parts.push(`${pct(opt)} opt`);
                effect = parts.join(" / ");
                break;
            }
            case "burst":
                effect = "AoE";
                break;
            case "hic": {
                const strength = readVal(first, "warpScrambleStrength");
                effect = strength > 0 ? `+${strength.toFixed(0)} str` : "bubble";
                break;
            }
        }

        out.push({
            key: `ewar-${typeId}`,
            typeId,
            name,
            count,
            effect,
            rangeText,
        });
    }

    // Stable sort: webs → scrams → eccm/ecm → others, by name within bucket.
    const kindOrder: Record<string, number> = {
        web: 0, scram: 1, hic: 2, ecm: 3, damp: 4, painter: 5, td: 6, burst: 7,
    };
    out.sort((a, b) => {
        const ka = EWAR_GROUP_KINDS[sdeData.types.get(a.typeId)?.groupID ?? 0] ?? "burst";
        const kb = EWAR_GROUP_KINDS[sdeData.types.get(b.typeId)?.groupID ?? 0] ?? "burst";
        return (kindOrder[ka] ?? 99) - (kindOrder[kb] ?? 99) || a.name.localeCompare(b.name);
    });

    return out;
});

const hasEwar = computed(() => ewarModules.value.length > 0);
</script>

<template>
    <div class="glass-panel w-[300px] overflow-y-auto divide-y divide-white/[0.04]">
        <!-- ========== Capacitor ========== -->
        <Category header-label="Capacitor">
            <template #header>
                <span :class="isCapacitorStable ? 'text-green-400' : 'text-red-400'">{{ capacitorState }}</span>
            </template>
            <CategoryLine>
                <span>
                    <ShipAttribute name="capacitorCapacity" :fixed="1" unit="GJ" />
                    /
                    <ShipAttribute name="rechargeRate" :fixed="2" :divide-by="1000" unit=" s" />
                </span>
            </CategoryLine>
            <CategoryLine>
                <span>
                    Δ <ShipAttribute name="capacitorPeakDelta" :fixed="1" unit="GJ/s" /> (<ShipAttribute
                        name="capacitorPeakDeltaPercentage"
                        :fixed="1"
                        unit="%"
                    />)
                </span>
            </CategoryLine>
        </Category>

        <!-- ========== Offense (hidden when 0 DPS and remote modules present) ========== -->
        <Category v-if="hasDamage || !hasRemoteStats" header-label="Offense">
            <template #header>
                <span>
                    <ShipAttribute name="damagePerSecondWithReload" :fixed="1" unit="dps" />
                </span>
            </template>
            <CategoryLine>
                <span v-tooltip="'Damage Per Second (with reload / without reload)'" class="flex items-center gap-1">
                    <ShipFitIcon name="damage-dps" :size="20" />
                    <span>
                        <ShipAttribute name="damagePerSecondWithReload" :fixed="1" unit="dps" />
                        <span class="text-gray-500">(<ShipAttribute name="damagePerSecondWithoutReload" :fixed="1" unit="dps" />)</span>
                    </span>
                </span>
                <span v-tooltip="'Volley Damage'" class="flex items-center gap-1">
                    <ShipFitIcon name="damage-alpha" :size="20" />
                    <ShipAttribute name="damageAlpha" :fixed="0" unit="HP" />
                </span>
            </CategoryLine>
            <CategoryLine v-if="primaryWeapon || readHullAttr('droneDamagePerSecond') > 0">
                <span
                    v-if="primaryWeapon"
                    v-tooltip="primaryWeapon.kind === 'turret' ? 'Optimal + falloff' : 'Missile flight range'"
                    class="flex items-center gap-1"
                >
                    <Icon name="lucide:target" class="w-5 h-5 text-gray-400" />
                    <span>
                        {{ primaryWeapon.optimalKm }} km<template v-if="primaryWeapon.hasFalloff">
                            + {{ primaryWeapon.falloffKm }} km
                        </template>
                    </span>
                </span>
                <span v-else></span>
                <span v-if="readHullAttr('droneDamagePerSecond') > 0" v-tooltip="'Drone DPS'" class="flex items-center gap-1">
                    <ShipFitIcon name="fitting-drones" :size="20" />
                    <ShipAttribute name="droneDamagePerSecond" :fixed="1" unit="dps" />
                </span>
            </CategoryLine>
            <CategoryLine v-if="primaryWeapon && primaryWeapon.hasTracking">
                <span v-tooltip="'Tracking speed (rad/s)'" class="flex items-center gap-1">
                    <Icon name="lucide:crosshair" class="w-5 h-5 text-gray-400" />
                    <span>{{ primaryWeapon.tracking }} rad/s</span>
                </span>
                <span></span>
            </CategoryLine>
        </Category>

        <!-- ========== EWar (only when EWar modules fitted) ========== -->
        <Category v-if="hasEwar" header-label="EWar">
            <template #header>
                <span class="text-gray-400 text-[10px]">{{ ewarModules.length }} module{{ ewarModules.length === 1 ? '' : 's' }}</span>
            </template>
            <CategoryLine v-for="m in ewarModules" :key="m.key">
                <span class="flex items-center gap-2 min-w-0">
                    <img
                        :src="`https://images.evetech.net/types/${m.typeId}/icon?size=64`"
                        v-tooltip="m.name"
                        class="w-5 h-5 rounded flex-shrink-0"
                    >
                    <span class="truncate">
                        {{ m.name }}<span v-if="m.count > 1" class="text-gray-500"> ×{{ m.count }}</span>
                    </span>
                </span>
                <span class="flex items-center gap-2 flex-shrink-0 text-gray-400">
                    <span v-if="m.effect">{{ m.effect }}</span>
                    <span v-if="m.rangeText" class="tabular-nums">{{ m.rangeText }}</span>
                </span>
            </CategoryLine>
        </Category>

        <!-- ========== Remote (only when remote modules fitted) ========== -->
        <Category v-if="hasRemoteStats" header-label="Remote">
            <template #header>
                <span class="text-gray-400 text-[10px]">Assistance / Warfare</span>
            </template>
            <!-- All remote stats on a single compact line, wrapping when needed -->
            <div class="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-xs text-gray-300">
                <span v-if="readHullAttr('remoteShieldBoostRate') > 0" v-tooltip="'Remote Shield Boost'" class="flex items-center gap-1">
                    <ShipFitIcon name="shield-boost-rate" :size="18" />
                    <ShipAttribute name="remoteShieldBoostRate" :fixed="1" unit="hp/s" />
                </span>
                <span v-if="readHullAttr('remoteArmorRepairRate') > 0" v-tooltip="'Remote Armor Repair'" class="flex items-center gap-1">
                    <ShipFitIcon name="armor-repair-rate" :size="18" />
                    <ShipAttribute name="remoteArmorRepairRate" :fixed="1" unit="hp/s" />
                </span>
                <span v-if="readHullAttr('remoteHullRepairRate') > 0" v-tooltip="'Remote Hull Repair'" class="flex items-center gap-1">
                    <ShipFitIcon name="hull-repair-rate" :size="18" />
                    <ShipAttribute name="remoteHullRepairRate" :fixed="1" unit="hp/s" />
                </span>
                <span v-if="readHullAttr('remoteCapTransferRate') > 0" v-tooltip="'Remote Capacitor Transfer'" class="flex items-center gap-1">
                    <Icon name="lucide:zap" class="w-[18px] h-[18px] text-yellow-400" />
                    <ShipAttribute name="remoteCapTransferRate" :fixed="1" unit="GJ/s" />
                </span>
                <span v-if="readHullAttr('energyNeutralizerRate') > 0" v-tooltip="'Energy Neutralizer Drain'" class="flex items-center gap-1">
                    <Icon name="lucide:zap-off" class="w-[18px] h-[18px] text-red-400" />
                    <ShipAttribute name="energyNeutralizerRate" :fixed="1" unit="GJ/s" />
                </span>
                <span v-if="readHullAttr('energyNosferatuRate') > 0" v-tooltip="'Energy Nosferatu Drain'" class="flex items-center gap-1">
                    <Icon name="lucide:droplets" class="w-[18px] h-[18px] text-purple-400" />
                    <ShipAttribute name="energyNosferatuRate" :fixed="1" unit="GJ/s" />
                </span>
            </div>
        </Category>

        <!-- ========== Defense ========== -->
        <Category header-label="Defense">
            <template #header>
                <ShipAttribute name="ehp" :fixed="0" round-down unit="ehp" />
            </template>
            <!-- Tank rates — all non-zero rates shown inline, EHP/s primary -->
            <div class="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-xs text-gray-300">
                <span v-if="readHullAttr('shieldBoostRate') > 0" v-tooltip="'Shield Boost Rate'" class="flex items-center gap-1">
                    <ShipFitIcon name="shield-boost-rate" :size="18" />
                    <ShipAttribute name="shieldEffectiveBoostRate" :fixed="1" unit="ehp/s" />
                    <span class="text-gray-500">(<ShipAttribute name="shieldBoostRate" :fixed="1" unit="hp/s" />)</span>
                </span>
                <span v-if="readHullAttr('armorRepairRate') > 0" v-tooltip="'Armor Repair Rate'" class="flex items-center gap-1">
                    <ShipFitIcon name="armor-repair-rate" :size="18" />
                    <ShipAttribute name="armorEffectiveRepairRate" :fixed="1" unit="ehp/s" />
                    <span class="text-gray-500">(<ShipAttribute name="armorRepairRate" :fixed="1" unit="hp/s" />)</span>
                </span>
                <span v-if="readHullAttr('hullRepairRate') > 0" v-tooltip="'Hull Repair Rate'" class="flex items-center gap-1">
                    <ShipFitIcon name="hull-repair-rate" :size="18" />
                    <ShipAttribute name="hullEffectiveRepairRate" :fixed="1" unit="ehp/s" />
                    <span class="text-gray-500">(<ShipAttribute name="hullRepairRate" :fixed="1" unit="hp/s" />)</span>
                </span>
                <span v-tooltip="'Passive Shield Recharge'" class="flex items-center gap-1">
                    <ShipFitIcon name="passive-shield-recharge" :size="18" />
                    <ShipAttribute name="passiveShieldEffectiveRechargeRate" :fixed="1" unit="ehp/s" />
                    <span class="text-gray-500">(<ShipAttribute name="passiveShieldRechargeRate" :fixed="1" unit="hp/s" />)</span>
                </span>
            </div>
            <!-- HP pools -->
            <CategoryLine>
                <span v-tooltip="'Shield HP'" class="flex items-center gap-1">
                    <ShipFitIcon name="shield-hp" :size="20" />
                    <span class="flex flex-col leading-tight">
                        <ShipAttribute name="shieldCapacity" :fixed="0" round-down unit="hp" />
                        <span class="text-[10px] text-gray-500"><ShipAttribute
                            name="shieldRechargeRate"
                            :fixed="0"
                            :divide-by="1000"
                            round-down
                            unit="s"
                        /> recharge</span>
                    </span>
                </span>
                <span v-tooltip="'Armor HP'" class="flex items-center gap-1">
                    <ShipFitIcon name="armor-hp" :size="20" />
                    <ShipAttribute name="armorHP" :fixed="0" round-down unit="hp" />
                </span>
                <span v-tooltip="'Hull HP'" class="flex items-center gap-1">
                    <ShipFitIcon name="hull-hp" :size="20" />
                    <ShipAttribute name="hp" :fixed="0" round-down unit="hp" />
                </span>
            </CategoryLine>
            <!-- Resistance header + rows -->
            <div class="px-3 pb-2 pt-1">
                <!-- Column headers: icon spacer + 4 damage type icons -->
                <div class="flex items-center mb-1">
                    <span class="w-[24px] flex-none"></span>
                    <span class="flex-1 flex justify-between">
                        <span class="w-[50px] flex justify-center"><ShipFitIcon name="em-resistance" :size="18" /></span>
                        <span class="w-[50px] flex justify-center"><ShipFitIcon name="thermal-resistance" :size="18" /></span>
                        <span class="w-[50px] flex justify-center"><ShipFitIcon name="kinetic-resistance" :size="18" /></span>
                        <span class="w-[50px] flex justify-center"><ShipFitIcon name="explosive-resistance" :size="18" /></span>
                    </span>
                </div>
                <!-- Shield resists -->
                <div class="flex items-center my-1">
                    <span class="w-[24px] flex-none flex justify-center">
                        <ShipFitIcon name="shield-hp" :size="18" />
                    </span>
                    <span class="flex-1 flex justify-between">
                        <Resistance name="shieldEmDamageResonance" />
                        <Resistance name="shieldThermalDamageResonance" />
                        <Resistance name="shieldKineticDamageResonance" />
                        <Resistance name="shieldExplosiveDamageResonance" />
                    </span>
                </div>
                <!-- Armor resists -->
                <div class="flex items-center my-1">
                    <span class="w-[24px] flex-none flex justify-center">
                        <ShipFitIcon name="armor-hp" :size="18" />
                    </span>
                    <span class="flex-1 flex justify-between">
                        <Resistance name="armorEmDamageResonance" />
                        <Resistance name="armorThermalDamageResonance" />
                        <Resistance name="armorKineticDamageResonance" />
                        <Resistance name="armorExplosiveDamageResonance" />
                    </span>
                </div>
                <!-- Hull resists -->
                <div class="flex items-center my-1">
                    <span class="w-[24px] flex-none flex justify-center">
                        <ShipFitIcon name="hull-hp" :size="18" />
                    </span>
                    <span class="flex-1 flex justify-between">
                        <Resistance name="emDamageResonance" />
                        <Resistance name="thermalDamageResonance" />
                        <Resistance name="kineticDamageResonance" />
                        <Resistance name="explosiveDamageResonance" />
                    </span>
                </div>
            </div>
        </Category>

        <!-- ========== Targeting ========== -->
        <Category header-label="Targeting" :default-collapsed="!expandDetails">
            <template #header>
                <ShipAttribute name="maxTargetRange" :fixed="2" :divide-by="1000" unit="km" />
            </template>
            <CategoryLine>
                <span v-tooltip="'Scan Strength'" class="flex items-center gap-1">
                    <ShipFitIcon name="sensor-strength" :size="20" />
                    <ShipAttribute name="scanStrength" :fixed="2" unit="points" />
                </span>
                <span v-tooltip="'Scan Resolution'" class="flex items-center gap-1">
                    <ShipFitIcon name="scan-resolution" :size="20" />
                    <ShipAttribute name="scanResolution" :fixed="0" unit="mm" />
                </span>
            </CategoryLine>
            <CategoryLine>
                <span v-tooltip="'Signature Radius'" class="flex items-center gap-1">
                    <ShipFitIcon name="signature-radius" :size="20" />
                    <ShipAttribute name="signatureRadius" :fixed="0" unit="m" />
                </span>
                <span v-tooltip="'Maximum Locked Targets'" class="flex items-center gap-1">
                    <ShipFitIcon name="maximum-locked-targets" :size="20" />
                    <ShipAttribute name="maxLockedTargets" :fixed="0" unit="x" />
                </span>
            </CategoryLine>
        </Category>

        <!-- ========== Navigation (hidden for structures) ========== -->
        <Category v-if="!isStructure" header-label="Navigation" :default-collapsed="!expandDetails">
            <template #header>
                <ShipAttribute name="maxVelocity" :fixed="1" unit="m/s" />
            </template>
            <CategoryLine>
                <span v-tooltip="'Mass'" class="flex items-center gap-1">
                    <ShipFitIcon name="mass" :size="20" />
                    <ShipAttribute name="mass" :fixed="2" :divide-by="1000" unit="t" />
                </span>
                <span v-tooltip="'Inertia Modifier'" class="flex items-center gap-1">
                    <ShipFitIcon name="inertia-modifier" :size="20" />
                    <ShipAttribute name="agility" :fixed="4" unit="x" />
                </span>
            </CategoryLine>
            <CategoryLine>
                <span v-tooltip="'Ship Warp Speed'" class="flex items-center gap-1">
                    <ShipFitIcon name="warp-speed" :size="20" />
                    <ShipAttribute name="warpSpeedMultiplier" :fixed="2" unit="AU/s" />
                </span>
                <span v-tooltip="'Align Time'" class="flex items-center gap-1">
                    <ShipFitIcon name="align-time" :size="20" />
                    <ShipAttribute name="alignTime" :fixed="2" unit="s" />
                </span>
            </CategoryLine>
        </Category>

        <!-- ========== Drones (hidden for structures) ========== -->
        <Category v-if="!isStructure" header-label="Drones" :default-collapsed="!expandDetails">
            <template #header>
                <ShipAttribute name="droneDamagePerSecond" :fixed="1" unit="dps" />
            </template>
            <CategoryLine>
                <span v-tooltip="'Drone Bandwidth'" class="flex items-center gap-1">
                    <ShipFitIcon name="mass" :size="20" />
                    <span>
                        <ShipAttribute name="droneBandwidthLoad" :fixed="0" />/<ShipAttribute
                            name="droneBandwidth"
                            :fixed="0"
                        />
                        Mbit/sec
                    </span>
                </span>
                <span v-tooltip="'Drone Control Range'" class="flex items-center gap-1">
                    <ShipFitIcon name="inertia-modifier" :size="20" />
                    <CharAttribute
                        name="droneControlDistance"
                        :fixed="2"
                        :divide-by="1000"
                        unit="km"
                    />
                </span>
            </CategoryLine>
            <CategoryLine>
                <span class="flex items-center gap-1">
                    <span class="w-5"></span>
                    <ShipAttribute name="droneActive" :fixed="0" /> Active
                </span>
            </CategoryLine>
        </Category>
    </div>
</template>
