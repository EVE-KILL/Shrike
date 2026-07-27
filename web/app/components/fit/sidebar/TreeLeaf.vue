<script setup lang="ts">
/**
 * One row at the bottom of a tree branch — a module/charge in
 * the hardware listing, or a fit entry under a hull. Emits
 * click/dblclick/dragstart/enter/leave so the parent can wire
 * them to fit manager mutators.
 *
 * Draggable leaves get a grab cursor; non-interactive leaves
 * (placeholder "No Item" rows) are dimmed and don't respond to
 * hover.
 *
 * `icon` is a Lucide icon name — use this for contextual glyphs
 * like a "saved fit" marker next to a fit row. We intentionally
 * don't support the CCP UI textures here because they look
 * oversized next to 12 px text.
 */

const props = withDefaults(
    defineProps<{
        content: string;
        /** When true, the row becomes a drag source. */
        draggable?: boolean;
        /** When false, the row gets muted styling and no hover. */
        interactive?: boolean;
        /** Optional lucide icon name (e.g. "lucide:bookmark"). */
        icon?: string;
        iconTitle?: string;
        /** Optional image URL (e.g. CCP type icon). Shown before text. */
        imageUrl?: string;
    }>(),
    { draggable: false, interactive: true },
);

const emit = defineEmits<{
    click: [e: MouseEvent];
    dblclick: [e: MouseEvent];
    dragstart: [e: DragEvent];
    enter: [];
    leave: [];
}>();
</script>

<template>
    <div
        :class="[
            'flex items-center gap-1.5 px-2 py-0.5 text-xs text-gray-300 select-none',
            props.interactive
                ? 'hover:bg-white/[0.04] cursor-grab active:cursor-grabbing'
                : 'text-gray-600',
        ]"
        :draggable="props.draggable"
        @click="emit('click', $event)"
        @dblclick="emit('dblclick', $event)"
        @dragstart="emit('dragstart', $event)"
        @mouseenter="emit('enter')"
        @mouseleave="emit('leave')"
    >
        <Icon
            v-if="icon"
            :name="icon"
            v-tooltip="iconTitle"
            class="w-3 h-3 flex-shrink-0 text-gray-500"
        />
        <img
            v-if="imageUrl"
            :src="imageUrl"
            class="w-4 h-4 rounded-sm flex-shrink-0"
            loading="lazy"
        />
        <span class="flex-1 whitespace-nowrap overflow-hidden text-ellipsis">{{ content }}</span>
    </div>
</template>
