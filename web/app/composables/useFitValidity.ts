export interface FitIssue {
    key: string;
    label: string;
}

export function useFitValidity() {
    const { currentFit } = useCurrentFit();
    const capacity = useHullCapacity();

    const issues = computed<FitIssue[]>(() => {
        const fit = currentFit.value;
        if (!fit?.shipTypeId) return [];
        const result: FitIssue[] = [];
        const cpuOver = capacity.cpu.value.used - capacity.cpu.value.total;
        const powerOver = capacity.power.value.used - capacity.power.value.total;
        if (cpuOver > 0.01) result.push({ key: "cpu", label: `CPU exceeded by ${cpuOver.toFixed(1)} tf` });
        if (powerOver > 0.01) result.push({ key: "power", label: `Powergrid exceeded by ${powerOver.toFixed(1)} MW` });

        for (const slot of ["High", "Medium", "Low", "Rig", "SubSystem"] as const) {
            const over = capacity.slotsUsed.value[slot] - capacity.slotCounts.value[slot];
            if (over > 0) result.push({ key: `slot-${slot}`, label: `${over} too many ${slot.toLowerCase()} modules` });
        }

        if (capacity.slotCounts.value.SubSystem > 0 && capacity.slotsUsed.value.SubSystem < capacity.slotCounts.value.SubSystem) {
            result.push({ key: "subsystems", label: `${capacity.slotCounts.value.SubSystem - capacity.slotsUsed.value.SubSystem} subsystem slots empty` });
        }
        if (capacity.droneBayUsed.value > capacity.droneBayCapacity.value) result.push({ key: "drone-bay", label: "Drone bay capacity exceeded" });
        if (capacity.droneBandwidthUsed.value > capacity.droneBandwidth.value) result.push({ key: "drone-bandwidth", label: "Drone bandwidth exceeded" });

        const launchers = fit.modules.filter(module => capacity.needsLauncherHardpoint(module.typeId));
        const turrets = fit.modules.filter(module => capacity.needsTurretHardpoint(module.typeId));
        const launcherLimit = launchers[0] ? capacity.hardpointLimitFor(launchers[0].typeId) : null;
        const turretLimit = turrets[0] ? capacity.hardpointLimitFor(turrets[0].typeId) : null;
        if (launcherLimit !== null && launchers.length > launcherLimit) result.push({ key: "launchers", label: "Launcher hardpoints exceeded" });
        if (turretLimit !== null && turrets.length > turretLimit) result.push({ key: "turrets", label: "Turret hardpoints exceeded" });

        const calibration = capacity.hullAttr("upgradeLoad");
        const calibrationLimit = capacity.hullAttr("upgradeCapacity");
        if (calibrationLimit > 0 && calibration > calibrationLimit) result.push({ key: "calibration", label: "Calibration exceeded" });
        return result;
    });

    return {
        issues,
        isValid: computed(() => Boolean(currentFit.value?.shipTypeId) && issues.value.length === 0),
    };
}
