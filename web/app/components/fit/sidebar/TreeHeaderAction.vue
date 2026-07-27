<script setup lang="ts">
/**
 * Clickable action icon on the right side of a TreeHeader, used
 * for e.g. the "simulate" button on hull rows. Emits a click
 * event so the parent can wire it up to a fit manager mutator.
 *
 * Name must be a Lucide icon name — the ship-fit-icon CCP
 * textures don't render well at 14-16 px next to EK-style text.
 */

const emit = defineEmits<{
    click: [e: MouseEvent];
}>();

defineProps<{
    icon: string;
    title?: string;
}>();

function onClick(e: MouseEvent) {
    // Stop the parent TreeListing header from re-toggling its
    // expand state when the user clicks the action glyph.
    e.stopPropagation();
    emit("click", e);
}
</script>

<template>
    <button
        type="button"
        v-tooltip="title"
        class="p-1 -mr-1 rounded text-gray-600 hover:text-blue-400 hover:bg-white/[0.04] transition-colors"
        @click="onClick"
    >
        <Icon :name="icon" class="w-3.5 h-3.5" />
    </button>
</template>
