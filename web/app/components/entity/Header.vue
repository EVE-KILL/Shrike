<script setup lang="ts">
const props = defineProps<{
    /** Show loading skeleton instead of content */
    loading?: boolean
    /** Entity identity accent (opaque CSS color) — adds a top strip + background wash */
    accent?: string | null
}>()

const accentStyle = computed(() => props.accent
    ? {
        backgroundImage: `linear-gradient(to bottom, color-mix(in srgb, ${props.accent} 8%, transparent), transparent 60%)`,
        boxShadow: `inset 0 2px 0 0 ${props.accent}`,
    }
    : undefined)
</script>

<template>
    <div v-if="loading" class="h-64 rounded-lg bg-white/[0.04] animate-pulse mb-6" />

    <div v-else class="glass-panel overflow-hidden mb-6" :style="accentStyle">
        <div class="p-6">
            <div class="flex flex-col md:flex-row gap-6">
                <!-- Image slot -->
                <div v-if="$slots.image" class="flex-shrink-0 flex justify-center md:justify-start">
                    <slot name="image" />
                </div>

                <!-- Middle: default slot -->
                <div class="flex-1 min-w-0">
                    <slot />
                </div>

                <!-- Right slot -->
                <div v-if="$slots.right" class="flex-shrink-0">
                    <slot name="right" />
                </div>
            </div>
        </div>

        <!-- Stats section -->
        <div v-if="$slots.stats" class="px-6 pb-6 pt-2 border-t border-white/[0.04]">
            <slot name="stats" />
        </div>
    </div>
</template>
