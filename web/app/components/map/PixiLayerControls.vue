<script setup lang="ts">
import type { MapActivityHours, MapActivityLayer, MapRenderBaseLayer } from '~/utils/map/layers'
import { MAP_ACTIVITY_LAYERS, MAP_ACTIVITY_WINDOWS, MAP_BASE_LAYERS, SOVEREIGNTY_BASE_LAYERS } from '~/utils/map/layers'

defineProps<{
    baseLayer: MapRenderBaseLayer
    activityLayer: MapActivityLayer
    hours: MapActivityHours
    showConnections: boolean
    showSystems: boolean
    showLabels: boolean
    allowSovereignty?: boolean
    compact?: boolean
}>()

const emit = defineEmits<{
    (event: 'update:baseLayer', value: MapRenderBaseLayer): void
    (event: 'update:activityLayer', value: MapActivityLayer): void
    (event: 'update:hours', value: MapActivityHours): void
    (event: 'update:showConnections', value: boolean): void
    (event: 'update:showSystems', value: boolean): void
    (event: 'update:showLabels', value: boolean): void
}>()
</script>

<template>
    <div class="px-3 py-2.5">
        <div class="flex flex-nowrap items-center gap-x-4 overflow-x-auto">
            <div class="flex shrink-0 items-center gap-1.5">
                <span class="mr-1 shrink-0 text-[9px] font-bold uppercase tracking-[0.16em] text-gray-600">Base</span>
                <button
                    v-for="option in (allowSovereignty ? SOVEREIGNTY_BASE_LAYERS : MAP_BASE_LAYERS)"
                    :key="option.id"
                    type="button"
                    class="rounded border px-2.5 py-1 text-[11px] transition-colors"
                    :class="baseLayer === option.id ? 'border-blue-400/25 bg-blue-500/12 text-blue-300' : 'border-white/[0.06] text-gray-500 hover:bg-white/[0.04] hover:text-gray-300'"
                    @click="emit('update:baseLayer', option.id)"
                >
                    {{ option.label }}
                </button>
            </div>

            <div class="flex shrink-0 items-center gap-1.5">
                <span class="mr-1 shrink-0 text-[9px] font-bold uppercase tracking-[0.16em] text-gray-600">Activity</span>
                <button
                    v-for="option in MAP_ACTIVITY_LAYERS"
                    :key="option.id"
                    type="button"
                    class="shrink-0 rounded border px-2.5 py-1 text-[11px] transition-colors"
                    :class="activityLayer === option.id ? 'border-orange-400/25 bg-orange-500/10 text-orange-300' : 'border-white/[0.06] text-gray-500 hover:bg-white/[0.04] hover:text-gray-300'"
                    @click="emit('update:activityLayer', option.id)"
                >
                    {{ option.label }}
                </button>
            </div>

            <div v-if="!compact" class="flex items-center gap-1">
                <span class="mr-1 text-[9px] font-bold uppercase tracking-[0.16em] text-gray-600">Window</span>
                <button
                    v-for="window in MAP_ACTIVITY_WINDOWS"
                    :key="window.value"
                    type="button"
                    class="rounded px-2 py-1 text-[10px] transition-colors"
                    :class="hours === window.value ? 'bg-white/[0.09] text-white' : 'text-gray-600 hover:bg-white/[0.04] hover:text-gray-300'"
                    :disabled="activityLayer === 'none'"
                    @click="emit('update:hours', window.value)"
                >
                    {{ window.label }}
                </button>
            </div>

            <div v-if="!compact" class="ml-auto flex shrink-0 items-center gap-1">
                <span class="mr-1 text-[9px] font-bold uppercase tracking-[0.16em] text-gray-600">Show</span>
                <button type="button" class="rounded px-2 py-1 text-[10px] transition-colors" :class="showConnections ? 'bg-white/[0.08] text-gray-200' : 'text-gray-600'" @click="emit('update:showConnections', !showConnections)">Routes</button>
                <button type="button" class="rounded px-2 py-1 text-[10px] transition-colors" :class="showSystems ? 'bg-white/[0.08] text-gray-200' : 'text-gray-600'" @click="emit('update:showSystems', !showSystems)">Systems</button>
                <button type="button" class="rounded px-2 py-1 text-[10px] transition-colors" :class="showLabels ? 'bg-white/[0.08] text-gray-200' : 'text-gray-600'" @click="emit('update:showLabels', !showLabels)">Labels</button>
            </div>
        </div>
    </div>
</template>
