<script setup lang="ts">
const props = defineProps<{
    /**
     * 'boxed' — 2-col mobile, 4-col desktop (character/corp/alliance stat boxes)
     * 'inline' — responsive flat grid (space/item pages)
     */
    variant?: 'boxed' | 'inline'
    /** For variant='inline': lg breakpoint column count (default 6) */
    columns?: 4 | 5 | 6 | 8
}>()

const gridClass = computed(() => {
    if (props.variant === 'boxed') return 'grid grid-cols-2 lg:grid-cols-4 gap-4'
    const base = 'grid grid-cols-2 gap-4'
    switch (props.columns ?? 6) {
        case 4: return `${base} sm:grid-cols-2 lg:grid-cols-4`
        case 5: return `${base} sm:grid-cols-3 lg:grid-cols-5`
        case 6: return `${base} sm:grid-cols-3 lg:grid-cols-6`
        case 8: return `${base} sm:grid-cols-4 lg:grid-cols-8`
        default: return `${base} sm:grid-cols-3 lg:grid-cols-6`
    }
})
</script>

<template>
    <div :class="gridClass">
        <slot />
    </div>
</template>
