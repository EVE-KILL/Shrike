<script setup lang="ts">
/**
 * Same as ShipAttribute but reads from the calculated character's
 * attributes (skill bonuses, drone control range, etc.) instead of
 * the hull's.
 */

import { AttributeChange, type ShipAttributeProps } from "../../../composables/useShipAttribute";

const props = defineProps<ShipAttributeProps>();

const result = useShipAttribute("Char", () => props);

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
