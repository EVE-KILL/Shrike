<script setup lang="ts">
/**
 * Thin wrapper around an `<img>` that resolves a friendly icon name
 * to a URL on our mirrored CCP UI texture endpoint. Port of
 * @eveshipfit/react's Icon component, but pointed at
 * `/images/ui/*` instead of
 * `data.eveship.fit/v{ver}/ui/texture/…`.
 *
 * Friendly names match the React lib's `IconName` union exactly — see
 * `imageserver/updateUiIcons.ts` for the full source manifest. If a
 * new name lands here that isn't on our imageserver yet, the `<img>`
 * will 404 silently (no fallback — the alt text is the name).
 */

export type ShipFitIconName =
    | "align-time"
    | "armor-hp"
    | "armor-repair-rate"
    | "arrow-left"
    | "arrow-right"
    | "cargo-hold"
    | "damage-alpha"
    | "damage-dps"
    | "drone-bay"
    | "em-resistance"
    | "explosive-resistance"
    | "fitting-alliance"
    | "fitting-character"
    | "fitting-corporation"
    | "fitting-drones"
    | "fitting-hislot"
    | "fitting-hull"
    | "fitting-hull-restriction"
    | "fitting-local"
    | "fitting-lowslot"
    | "fitting-medslot"
    | "fitting-rig-subsystem"
    | "hardpoint-launcher"
    | "hardpoint-turret"
    | "hull-hp"
    | "hull-repair-rate"
    | "inertia-modifier"
    | "kinetic-resistance"
    | "mass"
    | "maximum-locked-targets"
    | "menu-collapse"
    | "menu-expand"
    | "passive-shield-recharge"
    | "scan-resolution"
    | "sensor-strength"
    | "shield-boost-rate"
    | "shield-hp"
    | "signature-radius"
    | "simulate"
    | "thermal-resistance"
    | "warp-speed";

const props = defineProps<{
    name: ShipFitIconName;
    size?: number;
    title?: string;
}>();

const UI_BASE = "/images/ui";
const src = computed(() => `${UI_BASE}/${props.name}`);
</script>

<template>
    <img :src="src" :width="props.size" :height="props.size" v-tooltip="props.title" :alt="props.name" />
</template>
