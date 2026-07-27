<script setup lang="ts">
/**
 * "Open in" — every external destination for this killmail, laid out so the
 * whole set is readable at a glance once opened.
 *
 * This replaces the bar that used to run across the top of the page. That bar
 * did make all the tools visible without a click, but it spent the most
 * valuable strip of the page on outbound links. The panel keeps the
 * at-a-glance property — one column per tool, every sub-link listed, nothing
 * nested behind a second dropdown — without owning the top of the page.
 */
import type { KillToolContext } from '~/composables/useKillTools'

const props = defineProps<{ context: KillToolContext }>()

const tools = useKillTools(() => props.context)

const open = ref(false)
const root = ref<HTMLElement | null>(null)

const onPointerDown = (event: PointerEvent) => {
    if (!open.value) return
    if (root.value && !root.value.contains(event.target as Node)) open.value = false
}
const onKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && open.value) open.value = false
}

onMounted(() => {
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
    document.removeEventListener('pointerdown', onPointerDown)
    document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
    <div ref="root" class="relative">
        <button type="button"
            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-fine font-medium border transition-colors cursor-pointer"
            :class="open
                ? 'bg-blue-500/15 text-blue-400 border-blue-500/30'
                : 'text-gray-400 bg-white/[0.04] border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400'"
            @click="open = !open">
            <Icon name="lucide:external-link" class="text-fine" />
            Open in
            <span class="text-gray-600 tabular-nums">{{ tools.length }}</span>
            <Icon name="lucide:chevron-down" class="text-fine transition-transform" :class="open ? 'rotate-180' : ''" />
        </button>

        <div v-if="open"
            class="fixed sm:absolute inset-x-3 sm:inset-x-auto sm:right-0 z-50 mt-1 sm:w-[min(46rem,calc(100vw-1.5rem))] max-h-[70vh] overflow-y-auto rounded-lg border border-white/[0.10] bg-[#141414]/97 backdrop-blur-xl shadow-2xl shadow-black/60 p-3">
            <!-- Multi-column rather than a grid: tools have wildly different link
                 counts (zKillboard 9, EVEEye 1), and grid rows align to the
                 tallest cell, leaving holes. Columns let them pack. -->
            <div class="columns-2 sm:columns-3 lg:columns-4 gap-4">
                <div v-for="tool in tools" :key="tool.name" class="break-inside-avoid mb-3 min-w-0">
                    <div class="flex items-center gap-1.5 pb-1 mb-1 border-b border-white/[0.06]">
                        <img v-if="tool.icon" :src="tool.icon" class="w-3.5 h-3.5 rounded-sm flex-shrink-0" loading="lazy" alt="">
                        <Icon v-else name="lucide:code" class="text-fine text-gray-500 flex-shrink-0" />
                        <span class="text-fine font-bold uppercase tracking-[0.12em] text-gray-400 truncate">{{ tool.name }}</span>
                    </div>
                    <a v-for="item in tool.items" :key="item.label"
                        :href="item.url" target="_blank" rel="noopener noreferrer"
                        class="flex items-baseline gap-1.5 px-1 py-0.5 rounded text-fine text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">
                        <span class="flex-shrink-0">{{ item.label }}</span>
                        <span v-if="item.desc" class="text-gray-600 truncate">{{ item.desc }}</span>
                    </a>
                </div>
            </div>
        </div>
    </div>
</template>
