<script setup lang="ts">
/**
 * The circular in-game-style ship fitting display.
 *
 * Ported from @eveshipfit/react's ShipFit.tsx, composing:
 *
 *   - RingOuter / RingInner — the static ring chrome
 *   - Hull — center ship render
 *   - FitLink — curved share-link text along the top
 *   - RingTop — container for all rotation-positioned items:
 *       * RadialMenu markers for High/Medium/Low rows
 *       * Turret/Launcher hardpoint indicators (when withStats)
 *       * CPU / Power Grid / Rig usage arcs (when withStats)
 *       * 8 high + 8 med + 8 low slots + 3 rig + 4 subsystem slots
 *   - HullDraggable — invisible drop target over the hull
 *
 * Slot rotations are hard-coded numbers copied verbatim from the
 * reference. Don't "simplify" them into a loop generator — the exact
 * values matter for visual parity.
 *
 * Caller must give this component a square box (the ring is circular
 * and sizes to its container). The eveship.fit app uses 730×730 px.
 */

import FitLink from "./FitLink.vue";
import Hull from "./Hull.vue";
import HullDraggable from "./HullDraggable.vue";
import RadialMenu from "./RadialMenu.vue";
import RingInner from "./RingInner.vue";
import RingOuter from "./RingOuter.vue";
import RingTop from "./RingTop.vue";
import RingTopItem from "./RingTopItem.vue";
import ShipFitIcon from "../ShipFitIcon.vue";
import Slot from "./Slot.vue";
import Usage from "./Usage.vue";
import styles from "./ShipFit.module.css";

const props = defineProps<{
    withStats?: boolean;
    isPreview?: boolean;
    readOnly?: boolean;
}>();

const { sde } = useEveData();
const { stats } = useFitStatistics();

/** Resolve a dogma attribute name to its ID. Returns 0 sentinel if the SDE
 *  isn't loaded yet — Map.get(0) returns undefined which we then coalesce. */
function attrId(name: string): number {
    return sde.value?.attributeNameToId.get(name) ?? 0;
}

/** Pull a final value from the hull's computed attributes. Falls back to
 *  `base_value` when no effects modified the attribute (engine returns null
 *  in that case). */
function hullAttr(name: string): number {
    const h = stats.value?.hull;
    if (!h) return 0;
    const id = attrId(name);
    const a = h.attributes.get(id);
    if (!a) return 0;
    return a.value ?? a.base_value ?? 0;
}

/** Derived slot counts — matches the React StatisticsProvider's
 *  SlotAttributeMapping logic. SubSystem is clamped to 4 (T3Cs have
 *  5 on paper but you can only fit 4 in the ring). */
const slots = computed(() => {
    return {
        High: Math.floor(hullAttr("hiSlots") + hullAttr("hiSlotModifier")),
        Medium: Math.floor(hullAttr("medSlots") + hullAttr("medSlotModifier")),
        Low: Math.floor(hullAttr("lowSlots") + hullAttr("lowSlotModifier")),
        Rig: Math.floor(hullAttr("rigSlots")),
        SubSystem: Math.min(Math.floor(hullAttr("maxSubSystems")), 4),
        Turret: Math.floor(hullAttr("turretSlotsLeft")),
        Launcher: Math.floor(hullAttr("launcherSlotsLeft")),
    };
});

/** One-shot lookup of effect id by name. dogmaEffects has ~3000 entries
 *  so this is O(n) but gated behind a computed so it only runs when
 *  stats change. `@evekill/dogma` doesn't yet expose an effectNameToId
 *  map — add one when we have another reason to touch the package. */
function findEffectId(name: string): number | undefined {
    if (!sde.value) return undefined;
    for (const [id, eff] of sde.value.dogmaEffects) {
        if (eff?.name === name) return id;
    }
    return undefined;
}

/** Count fitted items whose typeDogma effects include the given effect id.
 *  Used to decorate turret/launcher hardpoint circles with a "used" fill. */
function countItemsWithEffect(effectId: number | undefined): number {
    if (effectId === undefined || !sde.value || !stats.value) return 0;
    let count = 0;
    for (const item of stats.value.items) {
        const td = sde.value.typeDogma.get(item.type_id);
        if (!td) continue;
        const effects: Array<{ effectID: number }> = td.dogmaEffects ?? [];
        if (effects.some((e) => e.effectID === effectId)) count++;
    }
    return count;
}

const turretFittedId = computed(() => findEffectId("turretFitted"));
const launcherFittedId = computed(() => findEffectId("launcherFitted"));

const hardpoints = computed(() => ({
    turretTotal: slots.value.Turret,
    turretUsed: countItemsWithEffect(turretFittedId.value),
    launcherTotal: slots.value.Launcher,
    launcherUsed: countItemsWithEffect(launcherFittedId.value),
}));

// Pre-compute slot arrays so the template stays readable. Each entry
// carries its rotation + fittable flag. Indices are 1-based to match
// the React reference.
const highSlots = computed(() =>
    Array.from({ length: 8 }, (_, i) => ({
        index: i + 1,
        rotation: -36.5 + (71 / 7) * i,
        fittable: slots.value.High >= i + 1,
    })),
);
const mediumSlots = computed(() =>
    Array.from({ length: 8 }, (_, i) => ({
        index: i + 1,
        rotation: 53 + (72 / 7) * i,
        fittable: slots.value.Medium >= i + 1,
    })),
);
const lowSlots = computed(() =>
    Array.from({ length: 8 }, (_, i) => ({
        index: i + 1,
        rotation: 142 + (72 / 7) * i,
        fittable: slots.value.Low >= i + 1,
    })),
);
const rigSlots = computed(() =>
    Array.from({ length: 3 }, (_, i) => ({
        index: i + 1,
        rotation: -74 + (21 / 2) * i,
        fittable: slots.value.Rig >= i + 1,
    })),
);
const subsystemSlots = computed(() =>
    Array.from({ length: 4 }, (_, i) => ({
        index: i + 1,
        rotation: -128 + (38 / 3) * i,
        fittable: slots.value.SubSystem >= i + 1,
    })),
);
</script>

<template>
    <div :class="styles.fit">
        <RingOuter />
        <RingInner />

        <Hull />
        <FitLink :is-preview="props.isPreview" />

        <RingTop>
            <!-- ========== withStats decorations ========== -->
            <template v-if="props.withStats">
                <!-- Turret hardpoint icon + row of dots (filled = used) -->
                <RingTopItem :rotation="-45" background>
                    <div :class="styles.turretLauncherIcon">
                        <ShipFitIcon name="hardpoint-turret" :size="16" />
                    </div>
                </RingTopItem>
                <RingTopItem
                    v-for="(_, i) in Array.from({ length: hardpoints.turretTotal })"
                    :key="`t-${i}`"
                    :rotation="-40 + i * 3"
                    background
                >
                    <div
                        :class="[
                            styles.turretLauncherItem,
                            i < hardpoints.turretUsed && styles.turretLauncherItemUsed,
                        ]"
                    >&nbsp;</div>
                </RingTopItem>

                <!-- Launcher hardpoint icon + row of dots (fills right-to-left) -->
                <RingTopItem :rotation="43" background>
                    <div :class="styles.turretLauncherIcon">
                        <ShipFitIcon name="hardpoint-launcher" :size="16" />
                    </div>
                </RingTopItem>
                <RingTopItem
                    v-for="(_, i) in Array.from({ length: hardpoints.launcherTotal })"
                    :key="`l-${i}`"
                    :rotation="39 - i * 3"
                    background
                >
                    <div
                        :class="[
                            styles.turretLauncherItem,
                            i < hardpoints.launcherUsed && styles.turretLauncherItemUsed,
                        ]"
                    >&nbsp;</div>
                </RingTopItem>

                <!-- CPU / PG / Rig arc meters -->
                <RingTopItem :rotation="-47" background>
                    <div :class="styles.usage">
                        <Usage type="rig" :angle="-30" :intervals="30" :markers="2" color="#3d4547" />
                    </div>
                </RingTopItem>
                <RingTopItem :rotation="134.5" background>
                    <div :class="styles.usage">
                        <Usage type="cpu" :angle="-44" :intervals="40" :markers="5" color="#2a504f" />
                    </div>
                </RingTopItem>
                <RingTopItem :rotation="135" background>
                    <div :class="styles.usage">
                        <Usage type="pg" :angle="44" :intervals="40" :markers="5" color="#541208" />
                    </div>
                </RingTopItem>
            </template>

            <!-- ========== High slots ========== -->
            <RingTopItem :rotation="-45" background>
                <RadialMenu type="High" />
            </RingTopItem>
            <RingTopItem
                v-for="slot in highSlots"
                :key="`h-${slot.index}`"
                :rotation="slot.rotation"
            >
                <Slot
                    type="High"
                    :index="slot.index"
                    :fittable="slot.fittable"
                    :main="slot.index === 1"
                    :read-only="props.readOnly"
                />
            </RingTopItem>

            <!-- ========== Medium slots ========== -->
            <RingTopItem :rotation="43" background>
                <RadialMenu type="Medium" />
            </RingTopItem>
            <RingTopItem
                v-for="slot in mediumSlots"
                :key="`m-${slot.index}`"
                :rotation="slot.rotation"
            >
                <Slot
                    type="Medium"
                    :index="slot.index"
                    :fittable="slot.fittable"
                    :read-only="props.readOnly"
                />
            </RingTopItem>

            <!-- ========== Low slots ========== -->
            <RingTopItem :rotation="133" background>
                <RadialMenu type="Low" />
            </RingTopItem>
            <RingTopItem
                v-for="slot in lowSlots"
                :key="`l-${slot.index}`"
                :rotation="slot.rotation"
            >
                <Slot
                    type="Low"
                    :index="slot.index"
                    :fittable="slot.fittable"
                    :read-only="props.readOnly"
                />
            </RingTopItem>

            <!-- ========== Rig slots ========== -->
            <RingTopItem
                v-for="slot in rigSlots"
                :key="`r-${slot.index}`"
                :rotation="slot.rotation"
            >
                <Slot
                    type="Rig"
                    :index="slot.index"
                    :fittable="slot.fittable"
                    :read-only="props.readOnly"
                />
            </RingTopItem>

            <!-- ========== Subsystem slots ========== -->
            <RingTopItem
                v-for="slot in subsystemSlots"
                :key="`s-${slot.index}`"
                :rotation="slot.rotation"
            >
                <Slot
                    type="SubSystem"
                    :index="slot.index"
                    :fittable="slot.fittable"
                    :read-only="props.readOnly"
                />
            </RingTopItem>
        </RingTop>

        <HullDraggable />
    </div>
</template>
