<script setup lang="ts">
/**
 * CPU / Power Grid / Rig capacity arc meters drawn along the ring.
 *
 * Each instance renders one filled-arc SVG on top of a set of evenly
 * spaced tick marks. The filled portion's angular extent is
 * proportional to used/total for the relevant resource. Positions are
 * computed with trig (not SVG transforms) so the arc and tick marks
 * line up with the ring's 464×464 viewBox.
 *
 * Ported from @eveshipfit/react's Usage.tsx — same radius (232),
 * same center offset (232 + 24 = 256 to match `viewBox="24 24 …"`),
 * same start angle (-90).
 */
import styles from "./ShipFit.module.css";
import type { FitStats } from "@evekill/dogma";

const props = defineProps<{
    type: "rig" | "cpu" | "pg";
    angle: number;
    intervals: number;
    markers: number;
    color: string;
}>();

const { sde } = useEveData();
const { stats } = useFitStatistics();

/** Resolve a dogma attribute name to its integer ID via the SDE. */
function attrId(name: string): number | undefined {
    return sde.value?.attributeNameToId.get(name);
}

/** Pull a numeric attribute off a ComputedItem's attributes Map. */
function attrValue(item: FitStats["hull"] | undefined, attributeName: string): number {
    if (!item) return 0;
    const id = attrId(attributeName);
    if (id === undefined) return 0;
    return item.attributes.get(id)?.value ?? 0;
}

const usage = computed(() => {
    const s = stats.value;
    if (!s) return { used: 0, total: 0 };

    let total = 0;
    let used = 0;

    switch (props.type) {
        case "rig":
            total = attrValue(s.hull, "upgradeCapacity");
            used = s.items.reduce((acc, item) => acc + attrValue(item, "upgradeCost"), 0);
            break;
        case "cpu":
            total = attrValue(s.hull, "cpuOutput");
            used = total - attrValue(s.hull, "cpuFree");
            break;
        case "pg":
            total = attrValue(s.hull, "powerOutput");
            used = total - attrValue(s.hull, "powerFree");
            break;
    }

    // Don't try to draw past full — overload renders just look broken.
    if (used > total) used = total;
    return { used, total };
});

function positionOnCircle(angleDeg: number, radius: number, center: number) {
    const rad = (angleDeg * Math.PI) / 180;
    return {
        x: radius * Math.cos(rad) + center,
        y: radius * Math.sin(rad) + center,
    };
}

const geometry = computed(() => {
    const radius = 232;
    const degrees = props.angle;
    const { used, total } = usage.value;
    const usageDegrees = total > 0 ? (degrees * used) / total : 0;

    const pointsInterval = degrees / props.intervals;
    const points = Array.from({ length: props.intervals }, (_, i) => {
        const a = i * pointsInterval - 90;
        return {
            inner: positionOnCircle(a, radius - 15, 232 + 24),
            outer: positionOnCircle(a, radius - 9, 232 + 24),
        };
    });

    const markersInterval = degrees / (props.markers - 1);
    const markers = Array.from({ length: props.markers }, (_, i) => {
        const a = i * markersInterval - 90;
        return {
            inner: positionOnCircle(a, radius - 15, 232 + 24),
            outer: positionOnCircle(a, radius - 2, 232 + 24),
        };
    });

    const start = positionOnCircle(-90, radius - 10, 232 + 24);
    const end = positionOnCircle(-90 + usageDegrees, radius - 10, 232 + 24);
    const sweepFlag = degrees >= 0 ? "1" : "0";
    const arcRadius = radius - 10;
    const arcPath = `M ${start.x} ${start.y} A ${arcRadius} ${arcRadius} 0 0 ${sweepFlag} ${end.x} ${end.y}`;

    return { points, markers, arcPath };
});
</script>

<template>
    <svg viewBox="24 24 464 464" xmlns="http://www.w3.org/2000/svg" :class="styles.ringOuter">
        <g>
            <path
                :style="{ fill: 'none', stroke: props.color, strokeWidth: 10 }"
                :d="geometry.arcPath"
            />

            <path
                v-for="(marker, i) in geometry.markers"
                :key="`m-${i}`"
                :style="{ fill: 'none', stroke: '#7a7a7a', strokeWidth: 1 }"
                :d="`M ${marker.inner.x} ${marker.inner.y} L ${marker.outer.x} ${marker.outer.y}`"
            />

            <path
                v-for="(point, i) in geometry.points"
                :key="`p-${i}`"
                :style="{ fill: 'none', stroke: '#7a7a7a', strokeWidth: 1 }"
                :d="`M ${point.inner.x} ${point.inner.y} L ${point.outer.x} ${point.outer.y}`"
            />
        </g>
    </svg>
</template>
