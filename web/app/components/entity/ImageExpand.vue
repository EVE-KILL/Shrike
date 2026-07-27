<script setup lang="ts">
const props = defineProps<{
    /** Full-resolution image URL for the modal */
    fullSrc: string
    /** Alt text for the full-res image */
    alt?: string
    /** Optional second image URL (e.g. legacy character portrait) */
    altSrc?: string
    /** Label for the primary image when altSrc is present */
    primaryLabel?: string
    /** Label for the alt image */
    altLabel?: string
}>()

const open = ref(false)
const showAlt = ref(false)

const onKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') open.value = false
}

watch(open, (v) => {
    if (v) showAlt.value = false
    if (import.meta.server) return
    if (v) document.addEventListener('keydown', onKeydown)
    else document.removeEventListener('keydown', onKeydown)
})

const activeSrc = computed(() => showAlt.value && props.altSrc ? props.altSrc : props.fullSrc)
</script>

<template>
    <div class="relative group/expand">
        <slot />
        <!-- Expand button — visible on hover -->
        <button
            class="absolute top-1.5 left-1.5 w-7 h-7 rounded-lg flex items-center justify-center bg-black/50 text-gray-400 hover:text-blue-400 hover:bg-black/70 opacity-0 group-hover/expand:opacity-100 transition-all cursor-pointer z-30"
            v-tooltip="'View full size'"
            @click.stop.prevent="open = true"
        >
            <Icon name="lucide:expand" class="w-3.5 h-3.5" />
        </button>

        <!-- Modal overlay -->
        <Teleport to="body">
            <Transition name="fade">
                <div v-if="open" class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80 backdrop-blur-sm" @click.self="open = false">
                    <!-- Close button -->
                    <button class="absolute top-4 right-4 w-9 h-9 rounded-lg flex items-center justify-center bg-black/50 text-gray-400 hover:text-blue-400 hover:bg-black/70 transition-all cursor-pointer z-10" @click="open = false">
                        <Icon name="lucide:x" class="w-6 h-6" />
                    </button>

                    <!-- Portrait toggle (character old/new) -->
                    <div v-if="altSrc" class="absolute top-4 left-4 flex gap-1 z-10">
                        <button
                            class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer"
                            :class="!showAlt ? 'bg-blue-500/30 text-blue-400' : 'bg-black/50 text-gray-400 hover:text-blue-400 hover:bg-black/70'"
                            @click="showAlt = false"
                        >
                            {{ primaryLabel || 'Current' }}
                        </button>
                        <button
                            class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer"
                            :class="showAlt ? 'bg-blue-500/30 text-blue-400' : 'bg-black/50 text-gray-400 hover:text-blue-400 hover:bg-black/70'"
                            @click="showAlt = true"
                        >
                            {{ altLabel || 'Legacy' }}
                        </button>
                    </div>

                    <EveImage
                        :key="activeSrc"
                        :src="activeSrc"
                        :alt="props.alt ?? ''"
                        class="max-w-[90vw] max-h-[90vh] rounded-lg shadow-2xl object-contain"
                        loading="eager"
                        fetchpriority="high"
                    />
                </div>
            </Transition>
        </Teleport>
    </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
    transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
    opacity: 0;
}
</style>
