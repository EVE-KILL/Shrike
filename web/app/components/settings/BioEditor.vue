<script setup lang="ts">
/**
 * One custom-bio editor: format toggle, textarea, character count, save state.
 *
 * Description.vue had three copies of this — character, corporation, alliance —
 * identical apart from the heading, the placeholder and a two-row difference in
 * textarea height. Which is how the pending-review banner ended up worded
 * identically in three places and the save button repeated its Transition
 * wrapper three times.
 */
export interface BioEditorState {
    text: string
    format: 'markdown' | 'eve_html'
    saving: boolean
    /** Shown for ~3s after a successful submit. */
    submitted: boolean
    error: string
}

const props = withDefaults(defineProps<{
    /** Panel heading, e.g. "Character Bio". */
    title: string
    /** Whose bio this is — shown under the heading. */
    subtitle: string
    state: BioEditorState
    placeholder: string
    /** True while a previous submission is still awaiting admin review. */
    pending?: boolean
    rows?: number
}>(), {
    pending: false,
    rows: 10,
})

const emit = defineEmits<{ save: [] }>()

const MAX_LENGTH = 4000

const formats = [
    { value: 'markdown' as const, label: 'Markdown' },
    { value: 'eve_html' as const, label: 'EVE HTML' },
]

const overLimit = computed(() => props.state.text.length > MAX_LENGTH)
</script>

<template>
    <div class="glass-panel p-5">
        <div class="flex items-start justify-between mb-4">
            <div>
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">{{ title }}</h2>
                <div class="text-xs text-gray-400 mt-1">{{ subtitle }}</div>
            </div>
            <div class="flex items-center gap-2">
                <Transition
                    enter-active-class="transition duration-200"
                    enter-from-class="opacity-0"
                    enter-to-class="opacity-100"
                    leave-active-class="transition duration-200"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0"
                >
                    <span v-if="state.submitted" class="text-xs text-blue-400">Submitted</span>
                </Transition>
                <button
                    class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer"
                    :class="state.saving
                        ? 'bg-white/[0.04] text-gray-500'
                        : 'bg-blue-500/15 text-blue-400 hover:bg-blue-500/25'"
                    :disabled="state.saving"
                    @click="emit('save')"
                >{{ state.saving ? 'Saving...' : 'Save' }}</button>
            </div>
        </div>

        <div
            v-if="pending"
            class="mb-3 rounded-md bg-yellow-500/10 border border-yellow-500/20 px-3 py-2 text-xs text-yellow-400 flex items-start gap-2"
        >
            <Icon name="lucide:clock" class="w-3.5 h-3.5 mt-0.5 flex-shrink-0" />
            <span>
                You have a submission pending admin review. The text below is your last submission —
                save again to update it, or leave it blank and save to withdraw.
            </span>
        </div>

        <div class="flex gap-1 mb-3">
            <button
                v-for="f in formats" :key="f.value"
                class="px-3 py-1 rounded-md text-xs font-medium transition-colors cursor-pointer"
                :class="state.format === f.value
                    ? 'bg-blue-500/15 text-blue-400'
                    : 'bg-white/[0.04] text-gray-500 hover:bg-white/[0.08]'"
                @click="state.format = f.value"
            >{{ f.label }}</button>
        </div>

        <textarea
            v-model="state.text"
            :rows="rows"
            :maxlength="MAX_LENGTH"
            :placeholder="placeholder"
            class="w-full px-3 py-2 rounded-md bg-black/30 border border-white/[0.08] text-sm text-gray-200 placeholder-gray-600 focus:outline-none focus:border-blue-500/40 font-mono"
        />
        <div class="flex items-center justify-between mt-1">
            <span class="text-fine" :class="overLimit ? 'text-red-400' : 'text-gray-500'">
                {{ state.text.length }} / {{ MAX_LENGTH }}
            </span>
        </div>
        <p v-if="state.error" class="text-xs text-red-400 mt-2">{{ state.error }}</p>
    </div>
</template>
