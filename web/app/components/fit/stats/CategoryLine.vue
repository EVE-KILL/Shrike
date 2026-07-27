<script setup lang="ts">
/**
 * One row inside a Category body. Two children sit side-by-side
 * at a consistent minimum row height. `lineClass` lets a caller
 * bump the vertical spacing for the Defense rows (which contain
 * stacked icon+value pairs that need more breathing room).
 */

defineProps<{
    lineClass?: string;
}>();
</script>

<template>
    <div
        :class="[
            'flex items-center justify-between gap-2 my-2 text-xs text-gray-300',
            lineClass,
        ]"
    >
        <slot />
    </div>
</template>
