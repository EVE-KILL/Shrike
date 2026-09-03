<script setup lang="ts">
import type { MapLayer } from '~/utils/map/layers'
import { MAP_LAYERS } from '~/utils/map/layers'

defineProps<{ modelValue: MapLayer }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: MapLayer): void }>()
</script>

<template>
    <div class="flex min-w-0 items-center gap-2 overflow-x-auto">
        <span class="shrink-0 px-1 text-fine font-bold uppercase tracking-[0.14em] text-gray-600">Layer</span>
        <button
            v-for="layer in MAP_LAYERS"
            :key="layer.id"
            type="button"
            class="shrink-0 rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors"
            :class="modelValue === layer.id
                ? 'border-blue-400/25 bg-blue-500/12 text-blue-300'
                : 'border-transparent text-gray-500 hover:border-white/[0.08] hover:bg-white/[0.04] hover:text-gray-300'"
            :title="layer.label"
            @click="emit('update:modelValue', layer.id)"
        >
            {{ layer.shortLabel }}
        </button>
    </div>
</template>
