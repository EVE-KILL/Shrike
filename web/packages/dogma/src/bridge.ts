import { createInterface } from "node:readline";
import { calculateFit, getHullStats } from "./index";
import type { EsfFit, SkillMap } from "./types";

const HULL_STAT_NAMES = [
    "alignTime",
    "ehp", "shieldEhp", "armorEhp", "hullEhp",
    "capacitorCapacity", "capacitorDepletesIn", "capacitorPeakDelta",
    "damagePerSecondWithoutReload", "damagePerSecondWithReload", "damageAlpha",
    "maxVelocity", "signatureRadius", "maxTargetRange", "scanResolution",
    "maxLockedTargets", "powerOutput", "cpuOutput", "upgradeCapacity", "mass",
] as const;

interface Request {
    id: number;
    fit: EsfFit;
    skills?: SkillMap;
}

const lines = createInterface({ input: process.stdin, crlfDelay: Infinity });

for await (const line of lines) {
    let id = 0;
    try {
        const request = JSON.parse(line) as Request;
        id = request.id;
        const stats = await calculateFit(request.fit, request.skills);
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
        } })}\n`);
    } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        process.stdout.write(`${JSON.stringify({ id, error: message })}\n`);
    }
}
