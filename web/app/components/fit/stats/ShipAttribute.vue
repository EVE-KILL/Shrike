<script setup lang="ts">
/**
 * Render a single ship attribute — formatted value + unit prefix,
 * green/red tint when the preview differs from the committed fit.
 * Port of @eveshipfit/react's ShipAttribute with EK color tokens.
 *
 * Unit prefix quirk preserved from the reference: `s`, `x`, `%`
 * units render without a leading space; everything else gets
 * ` `-separated.
 */

import { AttributeChange, type ShipAttributeProps } from "../../../composables/useShipAttribute";

const props = defineProps<ShipAttributeProps>();

const result = useShipAttribute("Ship", () => props);

const prefix = computed(() => {
    if (props.unit === undefined) return "";
    if (props.unit === "s" || props.unit === "x" || props.unit === "%") return props.unit;
    return ` ${props.unit}`;
});
</script>

<template>
    <span
        :class="{
            'text-green-400': result.change === AttributeChange.Increase,
            'text-red-400': result.change === AttributeChange.Decrease,
        }"
    >{{ result.value }}{{ prefix }}</span>
</template>
