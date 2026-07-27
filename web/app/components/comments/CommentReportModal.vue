<script setup lang="ts">
const props = defineProps<{
    modelValue: boolean
    onSubmit: (reason: string, message: string | null) => Promise<void>
}>()

const emit = defineEmits<{
    (e: 'update:modelValue', v: boolean): void
}>()

const reason = ref('spam')
const message = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const REASONS = [
    { value: 'spam', label: 'Spam' },
    { value: 'harassment', label: 'Harassment' },
    { value: 'nsfw', label: 'NSFW / inappropriate' },
    { value: 'offtopic', label: 'Off-topic' },
    { value: 'other', label: 'Other' },
]

async function submit() {
    submitting.value = true
    error.value = null
    try {
        await props.onSubmit(reason.value, message.value.trim() || null)
        emit('update:modelValue', false)
        reason.value = 'spam'
        message.value = ''
    } catch (err: any) {
        error.value = extractFetchError(err, 'Failed to report')
    } finally {
        submitting.value = false
    }
}
</script>

<template>
    <Modal
        :model-value="modelValue"
        title="Report comment"
        @update:model-value="emit('update:modelValue', $event)"
    >
        <div class="space-y-3">
            <div>
                <label class="block text-xs font-semibold text-gray-400 mb-1">Reason</label>
                <select
                    v-model="reason"
                    class="w-full px-3 py-2 bg-black/40 border border-white/[0.08] rounded-md text-sm text-white focus:outline-none focus:border-blue-500/50"
                >
                    <option v-for="r in REASONS" :key="r.value" :value="r.value">{{ r.label }}</option>
                </select>
            </div>
            <div>
                <label class="block text-xs font-semibold text-gray-400 mb-1">Additional info <span class="text-gray-600">(optional)</span></label>
                <textarea
                    v-model="message"
                    rows="3"
                    maxlength="1000"
                    class="w-full px-3 py-2 bg-black/40 border border-white/[0.08] rounded-md text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 resize-y"
                    placeholder="Anything we should know?"
                />
            </div>
            <p v-if="error" class="text-xs text-red-400">{{ error }}</p>
        </div>
        <template #footer>
            <div class="flex items-center justify-end gap-2">
                <button
                    class="px-3 py-1 rounded-md text-sm text-gray-400 hover:text-white hover:bg-white/[0.06] cursor-pointer"
                    @click="emit('update:modelValue', false)"
                >
                    Cancel
                </button>
                <button
                    class="px-3 py-1 rounded-md text-sm bg-red-500/[0.15] text-red-300 hover:bg-red-500/[0.25] disabled:opacity-40 cursor-pointer"
                    :disabled="submitting"
                    @click="submit"
                >
                    {{ submitting ? 'Reporting…' : 'Report' }}
                </button>
            </div>
        </template>
    </Modal>
</template>
