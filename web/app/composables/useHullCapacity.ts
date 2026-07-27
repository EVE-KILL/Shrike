/**
 * Single source of truth for hull capacity numbers — slot counts,
 * drone bay, cargo hold, CPU, power grid, drone bandwidth.
 *
 * Reads the dogma-engine computed values off `stats.value.hull` for
 * final (post-module-bonus) numbers, and the currentFit for "used"
 * counts that the engine doesn't expose directly (drone bay volume,
 * cargo hold volume, and slot occupancy from the module list).
 *
 * Used by:
 *   - ResourceBar — compact numeric readouts below the ring
 *   - DroneBay / CargoBay widgets — fill bars
 *   - useFitManager.addModule / addDrone — hard capacity checks
 *     so the user can't fit a 9th high-slot on an 8-high hull or
 *     stuff a 1500m³ drone bay into a Vexor's 100m³ bay
 *
 * All refs are computed, so they reactively follow fit mutations
 * without any watchers. Null safety: every field returns 0 when the
 * hull isn't loaded yet (e.g. SSR, pre-stats paint), so consumers
 * can render "CPU 0 / 0 tf" placeholders instead of blanking.
 */

import type { SdeData } from "@evekill/dogma";

interface AttrBearing {
    attributes: Map<number, { value?: number; base_value?: number }>;
}

export function useHullCapacity() {
    const { sde } = useEveData();
    const { stats } = useFitStatistics();
    const { currentFit } = useCurrentFit();

    /** Resolve a named hull attribute to a number, with 0 fallback. */
    function hullAttr(name: string): number {
        const s = stats.value;
        const sdeData = sde.value;
        if (!s || !sdeData) return 0;
        const id = sdeData.attributeNameToId.get(name);
        if (id === undefined) return 0;
        const a = (s.hull as AttrBearing).attributes.get(id);
        return a?.value ?? a?.base_value ?? 0;
    }

    /** Pull a typeDogma attribute (raw base value, no engine calc). */
    function typeAttr(sdeData: SdeData, typeId: number, name: string): number {
        const attrId = sdeData.attributeNameToId.get(name);
        if (attrId === undefined) return 0;
        const td = sdeData.typeDogma.get(typeId);
        if (!td?.dogmaAttributes) return 0;
        const a = (td.dogmaAttributes as Array<{ attributeID: number; value: number }>).find(
            (x) => x.attributeID === attrId,
        );
        return a?.value ?? 0;
    }

    // ========== Slot counts ==================================================

    const slotCounts = computed(() => ({
        High: Math.floor(hullAttr("hiSlots") + hullAttr("hiSlotModifier")),
        Medium: Math.floor(hullAttr("medSlots") + hullAttr("medSlotModifier")),
        Low: Math.floor(hullAttr("lowSlots") + hullAttr("lowSlotModifier")),
        // Rig count lives on `rigSlots` — ships with no rig slots return 0
        // (structures, capitals sometimes) and the widget hides itself.
        Rig: Math.floor(hullAttr("rigSlots")),
        // Subsystems cap at 4 by mechanic (one per T3 slot: Defense / Core /
        // Offensive / Propulsion). The attribute may report the hull max but
        // we clamp anyway.
        SubSystem: Math.min(Math.floor(hullAttr("maxSubSystems")), 4),
    }));

    const slotsUsed = computed(() => {
        const fit = currentFit.value;
        if (!fit) return { High: 0, Medium: 0, Low: 0, Rig: 0, SubSystem: 0 };
        const used = { High: 0, Medium: 0, Low: 0, Rig: 0, SubSystem: 0 } as Record<
            "High" | "Medium" | "Low" | "Rig" | "SubSystem",
            number
        >;
        for (const m of fit.modules) used[m.slot.type] += 1;
        return used;
    });

    // ========== CPU / PG =====================================================

    const cpu = computed(() => {
        const total = hullAttr("cpuOutput");
        const free = hullAttr("cpuFree");
        return { total, used: total - free };
    });

    const power = computed(() => {
        const total = hullAttr("powerOutput");
        const free = hullAttr("powerFree");
        return { total, used: total - free };
    });

    // ========== Drone bay ====================================================

    const droneBayCapacity = computed(() => hullAttr("droneCapacity"));

    const droneBayUsed = computed(() => {
        const fit = currentFit.value;
        const sdeData = sde.value;
        if (!fit || !sdeData) return 0;
        let total = 0;
        for (const d of fit.drones) {
            const type = sdeData.types.get(d.typeId);
            const vol = (type?.volume as number | undefined) ?? 0;
            total += vol * (d.states.Active + d.states.Passive);
        }
        return total;
    });

    /** Hull's drone bandwidth cap (Mbit/s). */
    const droneBandwidth = computed(() => hullAttr("droneBandwidth"));

    /** Active drones' bandwidth load — the engine precomputes this. */
    const droneBandwidthUsed = computed(() => hullAttr("droneBandwidthLoad"));

    // ========== Cargo ========================================================

    /**
     * Ships expose cargo hold size two ways: as the dogma attribute
     * "capacity" (which the engine can modify via cargo rigs + expanded
     * cargohold mods) and as a raw `capacity` property on the inv_types
     * row (the base value, no modifiers). Prefer the dogma-computed
     * value when present — that's what picks up rig bonuses — and fall
     * back to the raw type capacity for hulls that don't surface the
     * attribute through the engine.
     */
    const cargoCapacity = computed(() => {
        const fromAttr = hullAttr("capacity");
        if (fromAttr > 0) return fromAttr;
        const sdeData = sde.value;
        const shipTypeId = currentFit.value?.shipTypeId;
        if (!sdeData || shipTypeId === undefined) return 0;
        const type = sdeData.types.get(shipTypeId);
        return (type?.capacity as number | undefined) ?? 0;
    });

    const cargoUsed = computed(() => {
        const fit = currentFit.value;
        const sdeData = sde.value;
        if (!fit || !sdeData) return 0;
        let total = 0;
        for (const c of fit.cargo) {
            const type = sdeData.types.get(c.typeId);
            const vol = (type?.volume as number | undefined) ?? 0;
            total += vol * c.quantity;
        }
        return total;
    });

    // ========== Helpers for the fit manager =================================

    /**
     * "Would adding `count` drones of this type still fit in the bay?"
     * Returns true if the combined volume (current + new) stays at or
     * under the hull's drone capacity. Used by useFitManager.addDrone
     * to refuse over-capacity adds before mutating state.
     */
    function canAddDrone(typeId: number, count: number): boolean {
        const sdeData = sde.value;
        if (!sdeData) return false;
        const type = sdeData.types.get(typeId);
        const vol = (type?.volume as number | undefined) ?? 0;
        const cap = droneBayCapacity.value;
        if (cap <= 0) return false;
        return droneBayUsed.value + vol * count <= cap;
    }

    /**
     * "Would adding `quantity` of this type fit in the cargo hold?"
     * Volume × quantity must fit in the remaining cargo capacity.
     */
    function canAddCargo(typeId: number, quantity: number): boolean {
        const sdeData = sde.value;
        if (!sdeData) return false;
        const type = sdeData.types.get(typeId);
        const vol = (type?.volume as number | undefined) ?? 0;
        const cap = cargoCapacity.value;
        if (cap <= 0) return false;
        return cargoUsed.value + vol * quantity <= cap;
    }

    /**
     * Per-drone bandwidth cost — `droneBandwidthUsed` attribute off the
     * drone type. Returns 0 if the type isn't a drone or the attribute
     * isn't present (e.g. fighter types on carriers use different
     * mechanics; we don't handle them yet).
     */
    function droneBandwidthUsedBy(typeId: number): number {
        const sdeData = sde.value;
        if (!sdeData) return 0;
        return typeAttr(sdeData, typeId, "droneBandwidthUsed");
    }

    /**
     * Total bandwidth currently committed to active drones, computed
     * directly from the fit's drone list (not from the engine's
     * async-updated `droneBandwidthLoad`). Using the fit-derived value
     * matters for rapid-succession adds: a user dropping 5 Berserker
     * IIs on a Vexor needs each drop to see the previous drop's
     * bandwidth impact synchronously, not after a calculateFit round
     * trip. The engine value (exposed as `droneBandwidthUsed` above)
     * is still what the ResourceBar reads for display because it
     * already agrees with this once stats are up to date.
     */
    const droneBandwidthCommitted = computed(() => {
        const fit = currentFit.value;
        const sdeData = sde.value;
        if (!fit || !sdeData) return 0;
        let total = 0;
        for (const d of fit.drones) {
            if (d.states.Active === 0) continue;
            total += droneBandwidthUsedBy(d.typeId) * d.states.Active;
        }
        return total;
    });

    /**
     * EVE's hard cap on simultaneously-controlled drones. Even with
     * bandwidth to spare, you can never have more than 5 drones in
     * space at once — the Drones skill tops out at level V which
     * grants +1 per level from a base of 0, capping at 5. A handful
     * of drone-bonused hulls (some carriers, the Gila, etc.) get
     * hull-level +N bonuses, but we don't model those yet; when we
     * do this constant moves to a per-hull computed attribute read
     * from `maxActiveDrones` or similar and the Drones-skill delta.
     */
    const MAX_ACTIVE_DRONES = 5;

    /**
     * Sum of active drones across every drone type in the fit. This
     * is the LHS of the "no more than 5 in space" cap check. Separate
     * from droneBandwidthCommitted because the two caps are orthogonal:
     * you can hit the drone-count cap long before bandwidth (e.g. 5
     * Warrior IIs at 5 Mbit each = 25/75 used on a Vexor but the 6th
     * Warrior still refuses on the count cap).
     */
    const totalActiveDrones = computed(() => {
        const fit = currentFit.value;
        if (!fit) return 0;
        let total = 0;
        for (const d of fit.drones) total += d.states.Active;
        return total;
    });

    /**
     * How many drones of this type can still be activated. Hit by
     * both caps — whichever is stricter wins:
     *
     *   1. Bandwidth:  floor((droneBandwidth − committed) / per-drone cost)
     *   2. Count:      MAX_ACTIVE_DRONES − totalActiveDrones
     *
     * Example on a Vexor (75 Mbit bandwidth, 5-drone count cap):
     *   - 0 Warriors active → byBw=floor(75/5)=15, byCount=5, min=5
     *     (user can launch all 5 Warriors; the 6th refuses on count)
     *   - 3 Berserkers active → byBw=floor((75-75)/25)=0, byCount=2,
     *     min=0 (bandwidth is the binding constraint)
     *   - 2 Warriors + 3 Hammerheads active → byCount=0 for any type
     *     (count cap hit first)
     */
    function maxActivatableDrones(typeId: number): number {
        const cap = droneBandwidth.value;
        if (cap <= 0) return 0;

        // Bandwidth branch
        const free = Math.max(0, cap - droneBandwidthCommitted.value);
        const cost = droneBandwidthUsedBy(typeId);
        const byBandwidth =
            cost <= 0 ? Number.MAX_SAFE_INTEGER : Math.floor(free / cost);

        // Count-cap branch
        const byCount = Math.max(0, MAX_ACTIVE_DRONES - totalActiveDrones.value);

        return Math.min(byBandwidth, byCount);
    }

    /**
     * "Can `count` more drones of this type be activated?" Delegates
     * to maxActivatableDrones so both the bandwidth and count caps
     * are enforced together — callers don't need to know which one
     * is the binding constraint.
     */
    function canActivateDrone(typeId: number, count: number): boolean {
        return count <= maxActivatableDrones(typeId);
    }

    /**
     * "Is there a free slot of this type?" True when the current
     * module count in the slot is strictly less than the hull's cap.
     */
    function hasFreeSlot(slotType: "High" | "Medium" | "Low" | "Rig" | "SubSystem"): boolean {
        return slotsUsed.value[slotType] < slotCounts.value[slotType];
    }

    // ========== Turret / launcher hardpoints =================================
    //
    // High slots split into turret hardpoints (guns) and launcher hardpoints
    // (missiles). A ship can have 6 high slots but only 3 launcher hardpoints
    // — fitting a 4th missile launcher would be mechanically impossible in
    // game even though there's an empty high slot. The dogma engine publishes
    // `turretSlotsLeft` / `launcherSlotsLeft` as the REMAINING count after
    // subtracting fitted hardpoint modules; we read that directly rather than
    // re-deriving it from the hull + fit.

    /** One-shot lookup of the hardpoint effects' IDs. Same pattern as
     *  ShipFit.vue's findEffectId — walks dogmaEffects once per SDE load
     *  and caches via computed identity. */
    const hardpointEffectIds = computed(() => {
        const sdeData = sde.value;
        if (!sdeData) return null;
        let launcher: number | undefined;
        let turret: number | undefined;
        for (const [id, eff] of sdeData.dogmaEffects) {
            if (!eff?.name) continue;
            if (eff.name === "launcherFitted") launcher = id;
            else if (eff.name === "turretFitted") turret = id;
            if (launcher !== undefined && turret !== undefined) break;
        }
        return { launcher, turret };
    });

    /** True if this module type needs a launcher hardpoint to fit. */
    function needsLauncherHardpoint(typeId: number): boolean {
        const sdeData = sde.value;
        const ids = hardpointEffectIds.value;
        if (!sdeData || !ids?.launcher) return false;
        const td = sdeData.typeDogma.get(typeId);
        if (!td?.dogmaEffects) return false;
        for (const eff of td.dogmaEffects as Array<{ effectID: number }>) {
            if (eff.effectID === ids.launcher) return true;
        }
        return false;
    }

    /** True if this module type needs a turret hardpoint to fit. */
    function needsTurretHardpoint(typeId: number): boolean {
        const sdeData = sde.value;
        const ids = hardpointEffectIds.value;
        if (!sdeData || !ids?.turret) return false;
        const td = sdeData.typeDogma.get(typeId);
        if (!td?.dogmaEffects) return false;
        for (const eff of td.dogmaEffects as Array<{ effectID: number }>) {
            if (eff.effectID === ids.turret) return true;
        }
        return false;
    }

    /**
     * "Does this module have a hardpoint available to fit into?"
     *
     * Non-hardpoint modules (reps, hardeners, most highs, most meds)
     * bypass the check and return true. Turrets/launchers check the
     * respective engine-published `*SlotsLeft` counter; zero or negative
     * means no room. The check only fires in the addModule commit path
     * so hover preview still renders, and a rapid sequence of double-
     * clicks may briefly slip one extra through while the engine's
     * async recalc catches up — rare and easy to undo.
     */
    function hasHardpointFor(typeId: number): boolean {
        if (needsLauncherHardpoint(typeId)) {
            return hullAttr("launcherSlotsLeft") > 0;
        }
        if (needsTurretHardpoint(typeId)) {
            return hullAttr("turretSlotsLeft") > 0;
        }
        return true;
    }

    // ========== Propulsion module detection ====================================
    //
    // EVE only allows one propulsion module (MWD or Afterburner) active at a
    // time. Group 46 covers all fittable propulsion modules — both AB and MWD
    // variants. The fit manager uses this to enforce exclusivity when toggling
    // module states: activating one propmod deactivates the others.

    /** EVE SDE groupID for all fittable propulsion modules (AB + MWD). */
    const PROPULSION_MODULE_GROUP_ID = 46;

    function isPropulsionModule(typeId: number): boolean {
        const sdeData = sde.value;
        if (!sdeData) return false;
        const type = sdeData.types.get(typeId);
        return type?.groupID === PROPULSION_MODULE_GROUP_ID;
    }

    return {
        slotCounts,
        slotsUsed,
        cpu,
        power,
        droneBayCapacity,
        droneBayUsed,
        droneBandwidth,
        droneBandwidthUsed,
        totalActiveDrones,
        maxActiveDrones: MAX_ACTIVE_DRONES,
        cargoCapacity,
        cargoUsed,
        canAddDrone,
        canAddCargo,
        canActivateDrone,
        maxActivatableDrones,
        droneBandwidthUsedBy,
        hasFreeSlot,
        needsLauncherHardpoint,
        needsTurretHardpoint,
        hasHardpointFor,
        isPropulsionModule,
        // Expose the raw helpers too in case other widgets want the
        // same "attr from hull" shortcut without duplicating the
        // sde.attributeNameToId dance.
        hullAttr,
        typeAttr,
    };
}
