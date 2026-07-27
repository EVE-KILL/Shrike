<script setup lang="ts">
/**
 * Collapsible category panel for the stats panel. Restyled to
 * match the EK design: blue uppercase header, lucide chevron,
 * max-height transition on the body.
 *
 * Header label sits on the left at 11px bold uppercase (same
 * treatment EntityShipClasses uses); summary content from the
 * `#header` slot sits on the right in regular gray text.
 */

const props = withDefaults(
    defineProps<{
        headerLabel: string;
        defaultCollapsed?: boolean;
    }>(),
    { defaultCollapsed: false },
);

const expanded = ref(!props.defaultCollapsed);

function toggle() {
    expanded.value = !expanded.value;
}
</script>

<template>
    <div>
        <div
            class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-white/[0.02] transition-colors select-none"
            @click="toggle"
        >
            <Icon
                :name="expanded ? 'lucide:chevron-down' : 'lucide:chevron-right'"
                class="w-3 h-3 flex-shrink-0 text-gray-600"
            />
            <div class="flex-1 text-[11px] font-bold uppercase tracking-[0.15em] text-blue-400/80">
                {{ headerLabel }}
            </div>
            <div class="text-xs text-gray-300 tabular-nums">
                <slot name="header" />
            </div>
        </div>
        <div
            class="overflow-hidden transition-[max-height] duration-300 ease-in-out"
            :style="{ maxHeight: expanded ? '500px' : '0px' }"
        >
            <div class="px-3 pb-3">
                <slot />
            </div>
        </div>
    </div>
</template>
