import { createInterface } from "node:readline";
import { calculateFit, getHullStats, loadSde } from "./index";
import type { EsfFit, EsfModule, SkillMap } from "./types";

const HULL_STAT_NAMES = [
    "alignTime",
    "ehp", "shieldEhp", "armorEhp", "hullEhp",
    "capacitorCapacity", "capacitorDepletesIn", "capacitorPeakDelta",
    "damagePerSecondWithoutReload", "damagePerSecondWithReload", "damageAlpha",
    "maxVelocity", "signatureRadius", "maxTargetRange", "scanResolution",
    "maxLockedTargets", "powerOutput", "cpuOutput", "upgradeCapacity", "mass",
    "shieldBoostRate", "shieldEffectiveBoostRate",
    "armorRepairRate", "armorEffectiveRepairRate",
    "hullRepairRate", "hullEffectiveRepairRate",
    "passiveShieldRechargeRate", "passiveShieldEffectiveRechargeRate",
    "remoteShieldBoostRate", "remoteArmorRepairRate", "remoteHullRepairRate",
    "remoteCapTransferRate", "energyNeutralizerRate", "energyNosferatuRate",
    "shieldCapacity", "armorHP", "hp",
    "shieldEmDamageResonance", "shieldThermalDamageResonance", "shieldKineticDamageResonance", "shieldExplosiveDamageResonance",
    "armorEmDamageResonance", "armorThermalDamageResonance", "armorKineticDamageResonance", "armorExplosiveDamageResonance",
    "hullEmDamageResonance", "hullThermalDamageResonance", "hullKineticDamageResonance", "hullExplosiveDamageResonance",
] as const;

interface Request {
    id: number;
    fit: EsfFit;
    skills?: SkillMap;
}

const lines = createInterface({ input: process.stdin, crlfDelay: Infinity });

const NON_RUNNING_EFFECT = /(cloak|cyno)/i;
const PROPULSION_EFFECT = /(afterburner|microwarpdrive)/i;

async function canonicalStates(fit: EsfFit): Promise<EsfFit> {
    const sde = await loadSde();
    let activePropulsion = false;
    const modules = fit.modules.map((module: EsfModule) => {
        if (module.slot.type === "Rig") return { ...module, state: "Passive" as const };
        if (module.slot.type === "SubSystem") return { ...module, state: "Online" as const };
        const dogma = sde.typeDogma.get(module.type_id);
        const names = (dogma?.dogmaEffects ?? []).map((item: { effectID: number }) => sde.dogmaEffects.get(item.effectID)?.name ?? "");
        const hasActive = (dogma?.dogmaEffects ?? []).some((item: { effectID: number }) => sde.dogmaEffects.get(item.effectID)?.effectCategory === 1);
        const propulsion = names.some((name: string) => PROPULSION_EFFECT.test(name));
        const forbidden = names.some((name: string) => NON_RUNNING_EFFECT.test(name));
        let state: "Online" | "Active" = hasActive && !forbidden ? "Active" : "Online";
        if (propulsion && state === "Active") {
            if (activePropulsion) state = "Online";
            else activePropulsion = true;
        }
        return { ...module, state };
    });
    return { ...fit, modules };
}

for await (const line of lines) {
    let id = 0;
    try {
        const request = JSON.parse(line) as Request;
        id = request.id;
        const canonicalFit = await canonicalStates(request.fit);
        const stats = await calculateFit(canonicalFit, request.skills);
        const raw = await getHullStats(stats, HULL_STAT_NAMES);
        process.stdout.write(`${JSON.stringify({ id, hull: {
            align_time: raw.alignTime ?? null,
            max_velocity: raw.maxVelocity ?? null,
            signature_radius: raw.signatureRadius ?? null,
            mass: raw.mass ?? null,
            ehp: raw.ehp ?? null,
            shield_ehp: raw.shieldEhp ?? null,
            armor_ehp: raw.armorEhp ?? null,
            hull_ehp: raw.hullEhp ?? null,
            cap_capacity: raw.capacitorCapacity ?? null,
            cap_depletes_in: raw.capacitorDepletesIn ?? null,
            cap_peak_delta: raw.capacitorPeakDelta ?? null,
            dps_with_reload: raw.damagePerSecondWithReload ?? null,
            dps_without_reload: raw.damagePerSecondWithoutReload ?? null,
            alpha: raw.damageAlpha ?? null,
            max_target_range: raw.maxTargetRange ?? null,
            scan_resolution: raw.scanResolution ?? null,
            max_locked_targets: raw.maxLockedTargets ?? null,
            pg_output: raw.powerOutput ?? null,
            cpu_output: raw.cpuOutput ?? null,
            calibration: raw.upgradeCapacity ?? null,
            shield_boost: raw.shieldBoostRate ?? null,
            shield_effective_boost: raw.shieldEffectiveBoostRate ?? null,
            armor_repair: raw.armorRepairRate ?? null,
            armor_effective_repair: raw.armorEffectiveRepairRate ?? null,
            hull_repair: raw.hullRepairRate ?? null,
            hull_effective_repair: raw.hullEffectiveRepairRate ?? null,
            passive_shield: raw.passiveShieldRechargeRate ?? null,
            passive_shield_effective: raw.passiveShieldEffectiveRechargeRate ?? null,
            remote_shield: raw.remoteShieldBoostRate ?? null,
            remote_armor: raw.remoteArmorRepairRate ?? null,
            remote_hull: raw.remoteHullRepairRate ?? null,
            remote_cap: raw.remoteCapTransferRate ?? null,
            neut: raw.energyNeutralizerRate ?? null,
            nos: raw.energyNosferatuRate ?? null,
            shield_hp: raw.shieldCapacity ?? null,
            armor_hp: raw.armorHP ?? null,
            hull_hp: raw.hp ?? null,
            shield_em_resist: raw.shieldEmDamageResonance == null ? null : 1 - raw.shieldEmDamageResonance,
            shield_thermal_resist: raw.shieldThermalDamageResonance == null ? null : 1 - raw.shieldThermalDamageResonance,
            shield_kinetic_resist: raw.shieldKineticDamageResonance == null ? null : 1 - raw.shieldKineticDamageResonance,
            shield_explosive_resist: raw.shieldExplosiveDamageResonance == null ? null : 1 - raw.shieldExplosiveDamageResonance,
            armor_em_resist: raw.armorEmDamageResonance == null ? null : 1 - raw.armorEmDamageResonance,
            armor_thermal_resist: raw.armorThermalDamageResonance == null ? null : 1 - raw.armorThermalDamageResonance,
            armor_kinetic_resist: raw.armorKineticDamageResonance == null ? null : 1 - raw.armorKineticDamageResonance,
            armor_explosive_resist: raw.armorExplosiveDamageResonance == null ? null : 1 - raw.armorExplosiveDamageResonance,
            hull_em_resist: raw.hullEmDamageResonance == null ? null : 1 - raw.hullEmDamageResonance,
            hull_thermal_resist: raw.hullThermalDamageResonance == null ? null : 1 - raw.hullThermalDamageResonance,
            hull_kinetic_resist: raw.hullKineticDamageResonance == null ? null : 1 - raw.hullKineticDamageResonance,
            hull_explosive_resist: raw.hullExplosiveDamageResonance == null ? null : 1 - raw.hullExplosiveDamageResonance,
        } })}\n`);
    } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        process.stdout.write(`${JSON.stringify({ id, error: message })}\n`);
    }
}
