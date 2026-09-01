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
        <PageHeader class="mb-4" :title="title" :description="subtitle" :icon="icon || 'lucide:info'">
            <template v-if="$slots.actions" #actions>
                <div>
                    <slot name="actions" />
                </div>
            </template>
            <template v-if="$slots.header" #meta>
                <slot name="header" />
            </template>
        </PageHeader>

        <slot />
    </div>
</template>
