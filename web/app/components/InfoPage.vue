<script setup lang="ts">
/**
 * Shell for the pages in the Information menu.
 *
 * Those eight pages were each built on their own, and it showed: on a 1600px
 * viewport they rendered at five different content widths — 606, 672, 834, 852
 * and 1024 — with anywhere from zero to eighteen glass panels between them.
 * Several also centred their heading while the rest of the site (campaigns,
 * battles, stats) leads with a left-aligned icon badge.
 *
 * Two widths only, chosen by what the page holds rather than by whoever wrote
 * it: prose and short forms at 1024px, reference and tabular content at 1152px
 * — the width /docs and /mcp already used for their two-column layouts.
 */
withDefaults(defineProps<{
    title: string
    subtitle?: string
    icon?: string
    /** Reference or tabular content — docs, status, wallet — rather than prose. */
    wide?: boolean
}>(), {
    wide: false,
})
</script>

<template>
    <div class="mx-auto py-4" :class="wide ? 'max-w-6xl' : 'max-w-5xl'">
        <div class="glass-panel p-5 mb-4">
            <div class="flex items-start gap-3">
                <span v-if="icon"
                    class="w-9 h-9 rounded-lg bg-blue-500/15 flex items-center justify-center flex-shrink-0">
                    <Icon :name="icon" class="text-base text-blue-400" />
                </span>
                <div class="min-w-0">
                    <h1 class="text-xl md:text-2xl font-bold text-white leading-tight">{{ title }}</h1>
                    <p v-if="subtitle" class="text-sm text-gray-500 mt-1.5">{{ subtitle }}</p>
                </div>
                <!-- Right-hand side of the header: actions, status pills, counts. -->
                <div v-if="$slots.actions" class="ml-auto flex-shrink-0">
                    <slot name="actions" />
                </div>
            </div>
            <slot name="header" />
        </div>

        <slot />
    </div>
</template>
