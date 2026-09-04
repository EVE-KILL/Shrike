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
 * The outer shell follows the same full-width content boundary as every other
 * route. Individual prose blocks can still constrain their own line length,
 * but page headers, panels, and controls all start and end on the same rails.
 */
withDefaults(defineProps<{
    title: string
    subtitle?: string
    icon?: string
    /** Retained for callers that identify reference-heavy pages. */
    wide?: boolean
}>(), {
    wide: false,
})
</script>

<template>
    <div class="w-full">
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
