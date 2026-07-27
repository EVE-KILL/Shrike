<script setup lang="ts">
/**
 * Header content for a TreeListing node — optional image icon
 * (e.g. ship render from images.evetech.net), text label, and
 * an optional action slot rendered to the right.
 *
 * The icon is sized from the nearest TreeListing's `height` prop
 * via the treeSize inject key.
 */

import { treeSizeInjectKey } from "./treeSizeKey";

defineProps<{
    icon?: string;
    text: string;
}>();

const getTreeSize = inject(treeSizeInjectKey, () => 24);
const size = computed(() => getTreeSize());
</script>

<template>
    <img
        v-if="icon"
        :src="icon"
        :height="size"
        :width="size"
        alt=""
        class="flex-shrink-0"
    />
    <span class="flex-1 whitespace-nowrap overflow-hidden text-ellipsis">{{ text }}</span>
    <span v-if="$slots.action" class="flex-shrink-0"><slot name="action" /></span>
</template>
