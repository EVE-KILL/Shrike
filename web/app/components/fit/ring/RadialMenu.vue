<script setup lang="ts">
/**
 * The tiny in-game "radial menu" group marker that sits at the top
 * of each of the High/Medium/Low slot rows. A 12×12 SVG with a
 * circular mask punched through; the `highlightSettings` rect picks
 * which of the three positions (top/right/bottom) gets the solid
 * indicator strip.
 *
 * The three variants match the in-game radial menu's High (top bar)
 * / Medium (right bar) / Low (bottom bar) orientations, so when the
 * three RadialMenu instances are rotated to -45°, 43°, 133° around
 * the ring, their highlight bars all point at the same place (the
 * ring's center).
 */
import styles from "./ShipFit.module.css";

const props = defineProps<{
    type: "Low" | "Medium" | "High";
}>();

const highlightSettings = {
    Low: { width: 12, height: 3, x: 0, y: 9 },
    Medium: { width: 3, height: 12, x: 9, y: 0 },
    High: { width: 12, height: 3, x: 0, y: 0 },
} as const;

const highlight = computed(() => highlightSettings[props.type]);
const maskId = computed(() => `radial-menu-${props.type}`);
</script>

<template>
    <svg viewBox="0 0 12 12" xmlns="http://www.w3.org/2000/svg" :class="styles.radialMenu">
        <defs>
            <mask :id="maskId">
                <rect :style="{ fill: '#ffffff', fillOpacity: 0.5 }" :width="12" :height="12" :x="0" :y="0" />

                <rect
                    :style="{ fill: '#ffffff' }"
                    :width="highlight.width"
                    :height="highlight.height"
                    :x="highlight.x"
                    :y="highlight.y"
                />

                <circle :style="{ fill: '#000000' }" :cx="6" :cy="6" :r="5" />
                <rect :style="{ fill: '#000000' }" :width="3" :height="3" :x="0" :y="0" />
                <rect :style="{ fill: '#000000' }" :width="4" :height="3" :x="9" :y="0" />
                <rect :style="{ fill: '#000000' }" :width="3" :height="4" :x="0" :y="9" />
                <rect :style="{ fill: '#000000' }" :width="4" :height="4" :x="9" :y="9" />
            </mask>
        </defs>

        <g>
            <rect :style="{ fill: '#ffffff' }" :width="12" :height="12" :x="0" :y="0" :mask="`url(#${maskId})`" />
        </g>
    </svg>
</template>
