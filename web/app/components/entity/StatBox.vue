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
        <div class="space-y-2">
            <slot />
        </div>
    </div>
</template>
