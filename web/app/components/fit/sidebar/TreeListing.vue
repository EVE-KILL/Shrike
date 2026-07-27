<script setup lang="ts">
/**
 * Generic collapsible tree container — restyled to match the
 * EK site design language. Ported from @eveshipfit/react's
 * TreeListing but rendered with Tailwind utility classes instead
 * of CSS modules so it picks up the site's color tokens (blue
 * accents, translucent white overlays, subtle borders) rather
 * than the flat in-game `#1d1d1d` rectangles.
 *
 * Level 1 nodes get an uppercase "section header" look with a
 * blue accent; deeper levels fade to gray to keep the hierarchy
 * scannable without competing with the content.
 *
 * Children render via a slot gated behind `v-if="expanded"` so
 * collapsed branches don't pay the render cost — matching the
 * React lib's `getChildren` lazy-render pattern.
 */

import { treeSizeInjectKey } from "./treeSizeKey";

const props = withDefaults(
    defineProps<{
        level: number;
        hasHeader?: boolean;
        height?: number;
    }>(),
    { height: 20, hasHeader: false },
);

const emit = defineEmits<{
    /**
     * Double-click on the header. Consumers (HullListing) use this
     * for "apply a hull" without forcing the user to find the tiny
     * `+` action button. The single-click toggle still fires both
     * times alongside the dblclick — that's fine, it just produces
     * a brief expand/collapse flicker which is imperceptible.
     */
    dblclick: [e: MouseEvent];
}>();

// Default-expanded for header-less roots (pass-through). With a
// header we default collapsed — matches the React reference and
// makes the big market tree usable (expanding everything would
// render thousands of leaves up front).
const expanded = ref(!props.hasHeader);

function toggle() {
    expanded.value = !expanded.value;
}

provide(treeSizeInjectKey, () => props.height);

// Level-1 headers stand out with a blue uppercase label. Levels
// 2+ are quieter — just gray text at normal weight. The visual
// hierarchy prevents a 5-deep market tree from looking like a
// wall of identical rows.
const headerClass = computed(() => {
    if (props.level === 1) {
        return "text-blue-400/80 text-[11px] font-bold uppercase tracking-[0.15em]";
    }
    if (props.level === 2) {
        return "text-gray-300 text-xs font-medium";
    }
    return "text-gray-400 text-xs";
});

// Each level of nesting indents by ~12 px via left padding on the
// children container. The root (level 0) has no indent since it
// already sits inside the sidebar card's padding.
const childIndent = computed(() => {
    if (props.level === 0) return "";
    return "pl-3";
});
</script>

<template>
    <div>
        <div
            v-if="props.hasHeader"
            :class="[
                'flex items-center gap-1.5 px-2 cursor-pointer hover:bg-white/[0.04] transition-colors select-none',
                headerClass,
            ]"
            :style="{ height: `${props.height}px`, lineHeight: `${props.height}px` }"
            @click="toggle"
            @dblclick="emit('dblclick', $event)"
        >
            <Icon
                :name="expanded ? 'lucide:chevron-down' : 'lucide:chevron-right'"
                class="w-3 h-3 flex-shrink-0 text-gray-600"
            />
            <slot name="header" />
        </div>
        <div :class="[{ [childIndent]: props.hasHeader }]">
            <slot v-if="expanded" />
        </div>
    </div>
</template>
