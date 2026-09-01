/**
 * All mutations to the current fit go through here. Components should
 * never touch `currentFit.value` directly — they call these methods,
 * which handle the fit-cloning, slot-index assignment, and drone-state
 * transitions consistently.
 *
 * Design notes:
 *
 *   - Every mutator clones the fit before modifying it, so the reactive
 *     watcher in `useFitStatistics` fires on identity change. Mutating
 *     in place would also work with `deep: true` but makes it harder to
 *     reason about when the computation re-runs.
 *
 *   - Slot-index assignment for `addModule` picks the lowest unused
 *     index in that slot type. The React lib consults the stats for a
 *     max-slot cap; we deliberately don't, to avoid a circular dep
 *     between the manager and the statistics composable. If the caller
 *     asks to fit past the ship's capacity the extra module just won't
 *     render in the ring, which is what the HardwareListing's drag-drop
 *     validation prevents anyway.
 *
 *   - Preview mutations (`addModule(…, {preview: true})`) operate on a
 *     clone of `currentFit`, not of `fit`. This matches the React lib:
 *     hover always previews against the committed state, so hovering a
 *     second item doesn't stack the first preview's effects.
 *
 *   - Drone mutations operate on the grouped `states: {Passive, Active}`
 *     shape. Adding a new drone type with `state="Active"` creates a
 *     new group; adding to an existing type increments the count.
 */

import { emptyFit, type Fit, type FitModule, type FitSlotType, type FitState, type FitDrone } from "./fit/types";

export function useFitManager() {
    const { currentFit, setFit, loadFreshFit, setPreview, clearPreview } = useCurrentFit();
    // Capacity checks use the dogma-engine-computed hull attributes, so we
    // don't hand-roll slot-count math from the SDE here. Calling this
    // composable also no-ops if the stats watcher is already installed,
    // so there's no extra work.
    const capacity = useHullCapacity();
    const { sde } = useEveData();

    function cloneFit(f: Fit): Fit {
        return {
            name: f.name,
            description: f.description,
            shipTypeId: f.shipTypeId,
            modules: f.modules.map((m) => ({
                typeId: m.typeId,
                slot: { type: m.slot.type, index: m.slot.index },
                state: m.state,
                charge: m.charge ? { typeId: m.charge.typeId } : undefined,
            })),
            drones: f.drones.map((d) => ({
                typeId: d.typeId,
                states: { Passive: d.states.Passive, Active: d.states.Active },
            })),
            cargo: f.cargo.map((c) => ({ typeId: c.typeId, quantity: c.quantity })),
        };
    }

    /**
     * Lowest unused 1-based slot index for this slot type.
     *
     * 1-based matches the @eveshipfit/react reference (slots are
     * `index={1}..{8}`), and in turn the killmail `fittings` schema
     * already uses 0-based `ordinal` so we convert at the import
     * boundary rather than sprinkling off-by-ones through the UI.
     */
    function firstFreeIndex(f: Fit, slotType: FitSlotType): number {
        const used = new Set(f.modules.filter((m) => m.slot.type === slotType).map((m) => m.slot.index));
        for (let i = 1; i <= 64; i++) {
            if (!used.has(i)) return i;
        }
        return 1;
    }

    function defaultStateFor(slotType: FitSlotType): FitState {
        // Match the React lib: highs/meds default to Active, lows to Online,
        // rigs/subsystems to Passive. Users cycle state with click.
        switch (slotType) {
            case "High":
            case "Medium":
                return "Active";
            case "Low":
                return "Online";
            case "Rig":
            case "SubSystem":
                return "Passive";
        }
    }

    function createNewFit(shipTypeId: number, name = "New Fit") {
        // Fresh draft — clears currentFitId so the next save writes a
        // new row instead of updating the fit that used to be loaded.
        loadFreshFit(emptyFit(shipTypeId, name));
    }

    function setName(name: string) {
        if (!currentFit.value) return;
        setFit({ ...currentFit.value, name });
    }

    function setDescription(description: string) {
        if (!currentFit.value) return;
        setFit({ ...currentFit.value, description });
    }

    function addModule(
        typeId: number,
        slotType: FitSlotType,
        options: { preview?: boolean; index?: number } = {},
    ) {
        const base = currentFit.value;
        if (!base) return;

        const next = cloneFit(base);
        const index = options.index ?? firstFreeIndex(next, slotType);

        // Hard slot-count cap — checked against the FINAL chosen index,
        // not the count, so it catches every path:
        //   - HullDraggable hands off with no index: firstFreeIndex
        //     picks the lowest unused slot which may be past the cap
        //     when all fittable slots are taken
        //   - Slot-level drops carry an explicit index; we still want
        //     to refuse them if someone manages to target a non-
        //     fittable wedge
        //   - Replaces (drop onto an existing fitted slot of the same
        //     type) stay within [1..cap] by definition, so they pass
        //
        // Previews always go through so hover-preview never blinks
        // (hover past-cap still shows the effect; the commit is what
        // gets refused).
        //
        // Guard against `max === 0`: stats may not be loaded yet on
        // the very first frame, and we'd rather let the add succeed
        // than silently refuse everything before the engine catches
        // up. The ring's own `fittable` gating already handles the
        // visual side; this check is the backstop for data integrity.
        if (!options.preview) {
            const max = capacity.slotCounts.value[slotType];
            if (max > 0 && index > max) return;
            // Hardpoint cap — a 6-high Vexor with only 3 launcher
            // hardpoints must still refuse the 4th launcher. Applies
            // only to modules with the turret/launcher effects; all
            // other high-slot modules (reps, hardeners, NOS, etc.) fall
            // through. Preview hover still goes through so the user can
            // see the would-be effect before committing.
            if (!capacity.hasHardpointFor(typeId)) return;
        }

        // If this index is already occupied, replace the existing module.
        const existingIdx = next.modules.findIndex(
            (m) => m.slot.type === slotType && m.slot.index === index,
        );
        // Propmod exclusivity: if adding a propulsion module and another
        // propmod is already Active, default the new one to Online so we
        // don't end up with two propmods active simultaneously.
        let moduleState: FitState | "Preview" = options.preview ? "Preview" : defaultStateFor(slotType);
        if (!options.preview && capacity.isPropulsionModule(typeId)) {
            const hasActivePropmod = next.modules.some(
                (m) =>
                    !(m.slot.type === slotType && m.slot.index === index) &&
                    capacity.isPropulsionModule(m.typeId) &&
                    (m.state === "Active" || m.state === "Overload"),
            );
            if (hasActivePropmod) moduleState = "Online";
        }

        const module: FitModule = {
            typeId,
            slot: { type: slotType, index },
            state: moduleState,
        };
        if (existingIdx >= 0) {
            next.modules[existingIdx] = module;
        } else {
            next.modules.push(module);
        }

        if (options.preview) {
            setPreview(next);
        } else {
            setFit(next);
        }
    }

    function removeModule(slotType: FitSlotType, index: number) {
        if (!currentFit.value) return;
        const next = cloneFit(currentFit.value);
        next.modules = next.modules.filter(
            (m) => !(m.slot.type === slotType && m.slot.index === index),
        );
        setFit(next);
    }

    /** Fill every currently empty high slot with one turret/launcher type. */
    function fillHighRack(typeId: number, chargeTypeId?: number): number {
        const fit = currentFit.value;
        if (!fit) return 0;
        const hardpoints = capacity.hardpointsAvailableFor(typeId);
        if (hardpoints === null || hardpoints <= 0) return 0;

        const next = cloneFit(fit);
        let fitted = 0;
        for (let index = 1; index <= capacity.slotCounts.value.High && fitted < hardpoints; index++) {
            if (next.modules.some(module => module.slot.type === "High" && module.slot.index === index)) continue;
            next.modules.push({
                typeId,
                slot: { type: "High", index },
                state: defaultStateFor("High"),
                charge: chargeTypeId ? { typeId: chargeTypeId } : undefined,
            });
            fitted++;
        }
        if (fitted > 0) setFit(next);
        return fitted;
    }

    function setModuleState(slotType: FitSlotType, index: number, state: FitState) {
        if (!currentFit.value) return;
        const next = cloneFit(currentFit.value);
        const m = next.modules.find((x) => x.slot.type === slotType && x.slot.index === index);
        if (!m) return;
        m.state = state;

        // Propmod exclusivity: EVE only allows one propulsion module
        // (MWD or Afterburner) active at a time. When activating a
        // propmod, deactivate all other propmods in the fit.
        if ((state === "Active" || state === "Overload") && capacity.isPropulsionModule(m.typeId)) {
            for (const other of next.modules) {
                if (other === m) continue;
                if (!capacity.isPropulsionModule(other.typeId)) continue;
                if (other.state === "Active" || other.state === "Overload") {
                    other.state = "Online";
                }
            }
        }

        setFit(next);
    }

    function toggleRackOverload(slotType: FitSlotType) {
        if (!currentFit.value) return;
        const next = cloneFit(currentFit.value);
        const rackModules = next.modules.filter((m) => m.slot.type === slotType);
        if (rackModules.length === 0) return;

        // If any module in the rack is Overloaded, disable overload on
        // all of them (back to Active). Otherwise, overload all that are
        // currently Active.
        const anyOverloaded = rackModules.some((m) => m.state === "Overload");
        for (const m of rackModules) {
            if (anyOverloaded && m.state === "Overload") {
                m.state = "Active";
            } else if (!anyOverloaded && m.state === "Active") {
                m.state = "Overload";
            }
        }
        setFit(next);
    }

    function swapModule(
        fromSlotType: FitSlotType,
        fromIndex: number,
        toSlotType: FitSlotType,
        toIndex: number,
    ) {
        if (!currentFit.value) return;
        // Only allow swap within the same slot type — cross-type swaps are
        // a drag-out-and-back operation handled by setModule.
        if (fromSlotType !== toSlotType) return;

        const next = cloneFit(currentFit.value);
        const a = next.modules.find((x) => x.slot.type === fromSlotType && x.slot.index === fromIndex);
        const b = next.modules.find((x) => x.slot.type === toSlotType && x.slot.index === toIndex);
        if (!a) return;

        if (b) {
            b.slot.index = fromIndex;
        }
        a.slot.index = toIndex;
        setFit(next);
    }

    function setCharge(slotType: FitSlotType, index: number, chargeTypeId: number) {
        if (!currentFit.value) return;
        const next = cloneFit(currentFit.value);
        const m = next.modules.find((x) => x.slot.type === slotType && x.slot.index === index);
        if (!m) return;
        m.charge = { typeId: chargeTypeId };
        setFit(next);
    }

    /**
     * Double-click-from-sidebar charge apply. Finds every fitted module
     * that can accept this charge (group + size match) and loads the
     * charge into all of them at once.
     *
     * Compatibility rules, straight from EVE:
     *
     *   - The charge's `groupID` must match one of the module's
     *     `chargeGroup1..5` attributes (604, 605, 606, 609, 610).
     *   - The module's `chargeSize` (attr 128) must equal the charge's
     *     `chargeSize`, OR the module doesn't declare a chargeSize
     *     (which means "any size", e.g. scripts and some special
     *     modules).
     *
     * We apply to ALL matching modules — double-clicking Scourge Heavy
     * Missiles with three HMLs fitted loads all three at once, which is
     * the "barrier removal" behavior the user explicitly asked for. No-op
     * if no compatible module is found (wrong ammo type for the fit).
     */
    function setChargeAuto(chargeTypeId: number): boolean {
        const fit = currentFit.value;
        const sdeData = sde.value;
        if (!fit || !sdeData) return false;

        const attrId = (name: string) => sdeData.attributeNameToId.get(name);
        const chargeSizeAttr = attrId("chargeSize");
        const chargeGroupAttrs = [
            attrId("chargeGroup1"),
            attrId("chargeGroup2"),
            attrId("chargeGroup3"),
            attrId("chargeGroup4"),
            attrId("chargeGroup5"),
        ];
        if (chargeSizeAttr === undefined) return false;

        const chargeType = sdeData.types.get(chargeTypeId);
        if (!chargeType) return false;
        const chargeGroupId = chargeType.groupID as number | undefined;
        if (chargeGroupId === undefined) return false;

        // Read the charge's chargeSize (e.g. Small, Medium, Large). A
        // missing attribute means the charge is size-agnostic (e.g.
        // scripts), so the match still succeeds.
        const chargeTypeDogma = sdeData.typeDogma.get(chargeTypeId);
        let chargeSize: number | null = null;
        if (chargeTypeDogma?.dogmaAttributes) {
            const found = (chargeTypeDogma.dogmaAttributes as Array<{ attributeID: number; value: number }>)
                .find((a) => a.attributeID === chargeSizeAttr);
            if (found) chargeSize = found.value;
        }

        const next = cloneFit(fit);
        let appliedAny = false;

        for (const mod of next.modules) {
            const td = sdeData.typeDogma.get(mod.typeId);
            if (!td?.dogmaAttributes) continue;
            const attrs = td.dogmaAttributes as Array<{ attributeID: number; value: number }>;

            // Module must accept this charge's group via one of its
            // chargeGroup1..5 attributes.
            let groupOk = false;
            for (const ga of chargeGroupAttrs) {
                if (ga === undefined) continue;
                const found = attrs.find((a) => a.attributeID === ga);
                if (found && Number(found.value) === chargeGroupId) {
                    groupOk = true;
                    break;
                }
            }
            if (!groupOk) continue;

            // Size match: module's chargeSize must equal the charge's,
            // or the module must not declare chargeSize at all (open-
            // ended modules). Charges without chargeSize (scripts, a
            // handful of edge cases) match every size.
            const modChargeSizeEntry = attrs.find((a) => a.attributeID === chargeSizeAttr);
            const modChargeSize = modChargeSizeEntry?.value ?? null;
            if (modChargeSize !== null && chargeSize !== null && modChargeSize !== chargeSize) {
                continue;
            }

            // Skip modules that already hold this exact charge — no
            // point rewriting identical state.
            if (mod.charge?.typeId === chargeTypeId) continue;
            mod.charge = { typeId: chargeTypeId };
            appliedAny = true;
        }

        if (!appliedAny) return false;
        setFit(next);
        return true;
    }

    function removeCharge(slotType: FitSlotType, index: number) {
        if (!currentFit.value) return;
        const next = cloneFit(currentFit.value);
        const m = next.modules.find((x) => x.slot.type === slotType && x.slot.index === index);
        if (!m) return;
        m.charge = undefined;
        setFit(next);
    }

    function addDrone(typeId: number, count = 1, state: "Passive" | "Active" = "Active") {
        if (!currentFit.value) return;
        // Hard drone-bay volume limit. Without this a user could drag
        // 500 Hammerheads onto a Vexor and the engine would just
        // happily calculate whatever. canAddDrone() reads the current
        // drones + the new type's volume × count against the hull's
        // droneCapacity attribute.
        if (!capacity.canAddDrone(typeId, count)) return;

        const next = cloneFit(currentFit.value);
        let group = next.drones.find((d) => d.typeId === typeId);
        if (!group) {
            group = { typeId, states: { Passive: 0, Active: 0 } };
            next.drones.push(group);
        }

        if (state === "Active") {
            // Bandwidth split — adding a Berserker II (25 Mbit) to a
            // Vexor (75 Mbit) can only put 3 active; the 4th and 5th
            // overflow to the bay as Passive. The drones still fit in
            // the bay (canAddDrone already checked volume), they just
            // can't all be in space at the same time. Active-cap uses
            // the live `droneBandwidthLoad` so rapid multi-drops see
            // each other's side effects.
            const activatable = capacity.maxActivatableDrones(typeId);
            const goActive = Math.min(count, Math.max(0, activatable));
            const goPassive = count - goActive;
            group.states.Active += goActive;
            group.states.Passive += goPassive;
        } else {
            group.states.Passive += count;
        }
        setFit(next);
    }

    function removeDrone(typeId: number, count = 1) {
        if (!currentFit.value) return;
        const next = cloneFit(currentFit.value);
        const group = next.drones.find((d) => d.typeId === typeId);
        if (!group) return;
        // Remove passive first, then active, matching the React lib's
        // "drop the least-engaged drones first" behavior.
        let remaining = count;
        const takePassive = Math.min(remaining, group.states.Passive);
        group.states.Passive -= takePassive;
        remaining -= takePassive;
        const takeActive = Math.min(remaining, group.states.Active);
        group.states.Active -= takeActive;
        if (group.states.Passive === 0 && group.states.Active === 0) {
            next.drones = next.drones.filter((d) => d !== group);
        }
        setFit(next);
    }

    function activateDrones(typeId: number, count: number) {
        if (!currentFit.value) return;
        const next = cloneFit(currentFit.value);
        const group = next.drones.find((d) => d.typeId === typeId);
        if (!group) return;
        // Positive count: move passive → active (up to count AND up to
        // what bandwidth allows). Negative: active → passive, no cap.
        if (count > 0) {
            const activatable = capacity.maxActivatableDrones(typeId);
            const move = Math.min(count, group.states.Passive, Math.max(0, activatable));
            if (move <= 0) return;
            group.states.Passive -= move;
            group.states.Active += move;
        } else if (count < 0) {
            const move = Math.min(-count, group.states.Active);
            group.states.Active -= move;
            group.states.Passive += move;
        }
        setFit(next);
    }

    return {
        // fit lifecycle
        setFit,
        createNewFit,
        setName,
        setDescription,
        // preview overlay
        setPreview,
        removePreview: clearPreview,
        // modules
        addModule,
        fillHighRack,
        removeModule,
        setModuleState,
        toggleRackOverload,
        swapModule,
        setCharge,
        setChargeAuto,
        removeCharge,
        // drones
        addDrone,
        removeDrone,
        activateDrones,
    };
}

/** Re-export for components that bind to fit types in templates. */
export type { Fit, FitModule, FitSlotType, FitState, FitDrone } from "./fit/types";
