<script setup lang="ts">
/**
 * Rotation-positioned wrapper. Sits absolutely over the ring and
 * applies `transform: rotate(var(--rotation))` around the center; the
 * child then sits at `top: 3.5%; left: 50%` (top of the ring) and
 * counter-rotates via `--reverse-rotation` so the content stays
 * upright.
 *
 * `background=true` disables pointer events on the inner child — used
 * for decorative items (hardpoint markers, radial menu glyphs, usage
 * arcs) that shouldn't swallow drag/drop or click events from the
 * interactive slots sitting underneath them.
 */
import styles from "./ShipFit.module.css";

const props = defineProps<{
    rotation: number;
    background?: boolean;
}>();
</script>

<template>
    <div
        :class="[styles.ringTopItem, { [styles.background]: props.background }]"
        :style="{ '--rotation': `${props.rotation}deg` }"
    >
        <slot />
    </div>
</template>
