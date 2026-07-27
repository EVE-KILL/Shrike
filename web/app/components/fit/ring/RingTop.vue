<script setup lang="ts">
/**
 * Full-size container that holds every `RingTopItem` (slot wrappers,
 * radial menu markers, hardpoint markers, usage arcs). Just a
 * positioning anchor — all actual placement happens inside each
 * RingTopItem via its `--rotation` CSS var.
 */
import styles from "./ShipFit.module.css";
</script>

<template>
    <div :class="styles.ringTop">
        <slot />
    </div>
</template>
