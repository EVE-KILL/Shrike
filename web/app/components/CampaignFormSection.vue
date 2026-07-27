<script setup lang="ts">
/**
 * Numbered step panel for the campaign creator. The step badge flips to a
 * checkmark once the section is satisfied, so the form reads as progress
 * rather than as one long undifferentiated stack of inputs.
 */
defineProps<{
    step: number
    title: string
    hint?: string
    optional?: boolean
    /** Satisfied — swaps the number for a check. Leave undefined for sections with nothing to validate. */
    done?: boolean
}>()
</script>

<template>
    <section class="glass-panel p-4">
        <header class="flex items-start gap-3 mb-3">
            <span class="w-6 h-6 rounded-md flex items-center justify-center flex-shrink-0 text-fine font-bold tabular-nums transition-colors"
                :class="done ? 'bg-green-500/15 text-isk' : 'bg-blue-500/15 text-blue-400'">
                <Icon v-if="done" name="lucide:check" class="text-fine" />
                <template v-else>{{ step }}</template>
            </span>
            <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                    <h2 class="text-xs font-semibold text-gray-200">{{ title }}</h2>
                    <span v-if="optional" class="text-fine text-gray-600">optional</span>
                </div>
                <p v-if="hint" class="text-fine text-gray-600 mt-0.5">{{ hint }}</p>
            </div>
            <div v-if="$slots.aside" class="ml-auto flex-shrink-0"><slot name="aside" /></div>
        </header>
        <slot />
    </section>
</template>
