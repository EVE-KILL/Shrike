<script setup lang="ts">
/**
 * Connected progress stepper for multi-step tools. Completed steps stay
 * clickable so going back to fix something is one click, not a re-run.
 */
const props = defineProps<{
    steps: { n: number; label: string; hint?: string }[]
    current: number
}>()

defineEmits<{ select: [n: number] }>()

const stateOf = (n: number) => n < props.current ? 'done' : n === props.current ? 'current' : 'todo'
</script>

<template>
    <ol class="glass-panel flex items-center p-3 md:p-4 overflow-x-auto scrollbar-hide">
        <li v-for="(step, i) in steps" :key="step.n"
            class="flex items-center min-w-0" :class="i < steps.length - 1 ? 'flex-1' : ''">
            <button type="button" :disabled="step.n > current"
                class="flex items-center gap-2.5 min-w-0 transition-opacity"
                :class="step.n <= current ? 'cursor-pointer hover:opacity-80' : 'cursor-default'"
                @click="step.n <= current && $emit('select', step.n)">
                <span class="w-7 h-7 rounded-full flex items-center justify-center text-fine font-bold tabular-nums flex-shrink-0 border transition-colors"
                    :class="{
                        'bg-blue-500/20 text-blue-400 border-blue-500/40': stateOf(step.n) === 'current',
                        'bg-green-500/15 text-isk border-green-500/30': stateOf(step.n) === 'done',
                        'bg-white/[0.03] text-gray-600 border-white/[0.08]': stateOf(step.n) === 'todo',
                    }">
                    <Icon v-if="stateOf(step.n) === 'done'" name="lucide:check" class="text-fine" />
                    <template v-else>{{ step.n }}</template>
                </span>
                <span class="hidden sm:block text-left min-w-0">
                    <span class="block text-xs font-medium truncate"
                        :class="stateOf(step.n) === 'todo' ? 'text-gray-600' : 'text-gray-200'">{{ step.label }}</span>
                    <span v-if="step.hint" class="block text-fine text-gray-600 truncate">{{ step.hint }}</span>
                </span>
            </button>

            <div v-if="i < steps.length - 1" class="flex-1 h-px mx-3 min-w-[1.5rem] transition-colors"
                :class="steps[i + 1] && steps[i + 1]!.n <= current ? 'bg-blue-500/40' : 'bg-white/[0.08]'" />
        </li>
    </ol>
</template>
