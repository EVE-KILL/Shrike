<script setup lang="ts">
defineProps<{
    /** Lucide icon name for the box header */
    icon: string
    /** Icon color class (e.g. 'text-red-500') */
    iconColor?: string
    /** Box title (e.g. 'Combat', 'ISK') */
    title: string
}>()
</script>

<template>
    <div class="rounded-lg border border-white/[0.08] p-3.5">
        <div class="flex items-center gap-2 mb-3 pb-2 border-b border-white/[0.04]">
            <Icon :name="icon" class="w-4 h-4" :class="iconColor" />
            <span class="text-fine font-semibold text-gray-300">{{ title }}</span>
        </div>
        <div class="stat-box__rows space-y-2">
            <slot />
        </div>
    </div>
</template>

<style scoped>
/* Large entity totals need their own line in narrow cards. Keep the value
   together and right-aligned instead of overflowing or splitting digits. */
.stat-box__rows :deep(.flex.justify-between) {
    flex-wrap: wrap;
    column-gap: 0.5rem;
}

.stat-box__rows :deep(.flex.justify-between > :last-child) {
    margin-inline-start: auto;
    white-space: nowrap;
}
</style>
