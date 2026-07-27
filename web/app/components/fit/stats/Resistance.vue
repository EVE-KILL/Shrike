<script setup lang="ts">
/**
 * Resistance bar — fixed-width box with a colored fill proportional
 * to the resistance value, and the percent text overlaid on top.
 * Ported from @eveshipfit/react's Resistance with inline-style colors
 * (so we don't need a CSS module just for 4 background colors).
 */

import ShipAttribute from "./ShipAttribute.vue";

const props = defineProps<{
    name: string;
}>();

// Infer damage type from the attribute name — substring match on
// the lowercased name so future resistance attributes pick the
// right color automatically. Uses the in-game damage colors
// players already know: blue/red/grey/orange.
const damageType = computed(() => {
    const lname = props.name.toLowerCase();
    if (lname.includes("emdamage")) return "em";
    if (lname.includes("thermaldamage")) return "thermal";
    if (lname.includes("kineticdamage")) return "kinetic";
    if (lname.includes("explosivedamage")) return "explosive";
    return "";
});

const barColor = computed(() => {
    switch (damageType.value) {
        case "em":
            return "#195e8c";
        case "thermal":
            return "#8c1919";
        case "kinetic":
            return "#727272";
        case "explosive":
            return "#8c5e19";
        default:
            return "#444";
    }
});

// Raw resistance percent (0-100) for the bar width. Goes through
// useShipAttribute with isResistance: true so the flip + rounding
// rules match the text display exactly.
const barValue = useShipAttribute("Ship", () => ({
    name: props.name,
    fixed: 1,
    isResistance: true,
}));
</script>

<template>
    <span class="relative inline-block w-[50px] h-5 bg-white/[0.05] rounded-sm overflow-hidden align-middle">
        <span
            class="absolute left-0 top-0 bottom-0 z-0"
            :style="{ width: `${barValue.value}%`, backgroundColor: barColor }"
        ></span>
        <span class="absolute inset-0 z-10 flex items-center justify-center text-[11px] leading-none">
            <ShipAttribute :name="props.name" :fixed="1" unit="%" :is-resistance="true" />
        </span>
    </span>
</template>
