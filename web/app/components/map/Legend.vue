<script setup lang="ts">
import type { MapLayer } from '~/utils/map/layers'
import { activityLayerLabel } from '~/utils/map/layers'

const props = defineProps<{
    layer: MapLayer
    geographyLabel: string
    maximum?: number
    showExternal?: boolean
}>()

const isActivity = computed(() => !['geography', 'security'].includes(props.layer))
</script>

<template>
    <div class="pointer-events-none flex max-w-[calc(100%-1rem)] flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border border-white/[0.08] bg-black/65 px-3 py-2 text-[10px] text-gray-400 shadow-lg backdrop-blur-md">
        <template v-if="layer === 'geography'">
            <span class="flex items-center gap-1.5">
                <span class="h-2.5 w-4 rounded-sm bg-gradient-to-r from-blue-500/50 via-purple-500/50 to-amber-500/50" />
                {{ geographyLabel }} grouping
            </span>
            <span class="flex items-center gap-1.5">
                <span class="h-2.5 w-2.5 rounded-full border border-emerald-300 bg-emerald-400/25" />
                System ring = security
            </span>
        </template>
        <template v-else-if="layer === 'security'">
            <span class="font-semibold text-gray-300">System security</span>
            <span class="h-2 w-28 rounded-full" style="background: linear-gradient(90deg, #22d3ee, #a3e635, #fbbf24, #b91c1c)" />
            <span>High</span><span>Low</span><span>Null</span>
        </template>
        <template v-else-if="isActivity">
            <span class="font-semibold text-gray-300">{{ activityLayerLabel(layer) }} · last 24h</span>
            <span class="h-2 w-28 rounded-full" style="background: linear-gradient(90deg, #475569, #3b82f6, #a855f7, #f97316, #ef4444)" />
            <span>Quiet</span><span>Active</span><span>Peak {{ formatNumber(maximum ?? 0) }}</span>
            <span class="flex items-center gap-1.5">
                <span class="h-2.5 w-2.5 rounded-full border border-emerald-300 bg-transparent" />
                Inner ring = security
            </span>
        </template>
        <span v-if="showExternal" class="flex items-center gap-1.5">
            <span class="w-4 border-t border-dashed border-blue-300/70" />
            Out-of-region gate
        </span>
    </div>
</template>
