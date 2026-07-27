<script setup lang="ts">
/**
 * Static outer black ring + 5 corner bracket rects.
 *
 * Ported 1:1 from @eveshipfit/react's RingOuter.tsx. The viewBox
 * (24 24 464 464) is shared with RingInner so both SVGs compose
 * exactly. The 5th rect uses `transform="rotate(56)"` — that's the
 * upper-right corner that sits at a different angle than the other
 * four to match the in-game ring art.
 */
import styles from "./ShipFit.module.css";
</script>

<template>
    <svg viewBox="24 24 464 464" xmlns="http://www.w3.org/2000/svg" :class="styles.ringOuter">
        <g>
            <circle :style="{ fill: 'none', stroke: '#000000', strokeWidth: 16 }" cx="256" cy="256" r="224" />

            <rect :style="{ fill: '#000000' }" width="17" height="17" x="98" y="89" />
            <rect :style="{ fill: '#000000' }" width="17" height="17" x="401" y="93" />
            <rect :style="{ fill: '#000000' }" width="17" height="17" x="402" y="401" />
            <rect :style="{ fill: '#000000' }" width="17" height="17" x="94" y="402" />
            <rect :style="{ fill: '#000000' }" width="12" height="12" x="196" y="82" transform="rotate(56)" />
        </g>
    </svg>
</template>
