<script setup lang="ts">
import type { RoamReport } from '~/utils/roamReport'

const props = defineProps<{ report: RoamReport }>()
const emit = defineEmits<{ saved: [report: RoamReport], cancel: [] }>()
const form = reactive({
    namesText: [...props.report.roster.map(pilot => pilot.name), ...props.report.unresolved].join('\n'),
    startTime: isoToEveInput(new Date(props.report.start_time)),
    endTime: isoToEveInput(new Date(props.report.end_time)),
})
const saving = ref(false)
const error = ref('')
const names = computed(() => {
    const seen = new Set<string>()
    return form.namesText.split(/\r?\n/).map(name => name.trim()).filter(name => {
        if (!name || seen.has(name.toLowerCase())) return false
        seen.add(name.toLowerCase())
        return true
    })
})
const timeError = computed(() => {
    const start = eveInputToDate(form.startTime)
    const end = eveInputToDate(form.endTime)
    if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return 'Choose a start and end time.'
    if (end <= start) return 'End must be after start.'
    if (end.getTime() - start.getTime() > 72 * 60 * 60 * 1000) return 'Maximum window is 72 hours.'
    if (end.getTime() > Date.now() + 5 * 60 * 1000) return 'End cannot be in the future.'
    return ''
})
const canSave = computed(() => names.value.length > 0 && names.value.length <= 256 && !timeError.value && !saving.value)

async function save() {
    if (!canSave.value) return
    saving.value = true
    error.value = ''
    try {
        const updated = await apiFetch<RoamReport>(`/api/tools/roam-report/${props.report.id}`, {
            method: 'PUT',
            body: {
                names_text: form.namesText,
                start_time: eveInputToIso(form.startTime),
                end_time: eveInputToIso(form.endTime),
            },
        })
        emit('saved', updated)
    } catch (cause: any) {
        error.value = cause?.data?.error || cause?.data?.message || 'Could not update the report. Please try again.'
    } finally {
        saving.value = false
    }
}
</script>

<template>
    <form class="glass-panel p-5 sm:p-7" @submit.prevent="save">
        <div class="mb-5 flex flex-wrap items-start justify-between gap-3">
            <div>
                <h2 class="text-lg font-semibold text-white">Edit Roam Report</h2>
                <p class="mt-1 text-sm text-gray-400">Change the pilots or time window, then rebuild the report at this same URL.</p>
            </div>
            <button type="button" class="rounded-lg border border-white/10 px-3 py-2 text-sm text-gray-300 hover:bg-white/5" @click="emit('cancel')">Cancel</button>
        </div>
        <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.7fr)]">
            <div>
                <label for="roam-edit-pilots" class="mb-2 block text-sm font-medium text-gray-200">Fleet pilots</label>
                <textarea id="roam-edit-pilots" v-model="form.namesText" maxlength="16384" spellcheck="false"
                    class="h-56 w-full resize-y rounded-lg border border-white/10 bg-black/25 p-4 font-mono text-sm leading-6 text-gray-100 outline-none focus:border-blue-500/50" />
                <p class="mt-2 text-xs text-gray-500">{{ names.length }} unique {{ names.length === 1 ? 'pilot' : 'pilots' }} · up to 256</p>
            </div>
            <div>
                <p class="mb-2 text-sm font-medium text-gray-200">Roam window · EVE time (UTC)</p>
                <label class="mb-4 block text-sm text-gray-300">
                    <span class="mb-2 block">Start</span>
                    <DateTimePicker v-model="form.startTime" placeholder="Choose a start" />
                </label>
                <label class="block text-sm text-gray-300">
                    <span class="mb-2 block">End</span>
                    <DateTimePicker v-model="form.endTime" placeholder="Choose an end" />
                </label>
                <p v-if="timeError" class="mt-3 text-sm text-amber-300">{{ timeError }}</p>
            </div>
        </div>
        <p class="mt-5 text-xs leading-5 text-gray-500">Anyone with the share link can edit this report. Saving updates the existing report for everyone viewing it.</p>
        <p v-if="names.length > 256" class="mt-3 text-sm text-amber-300">A report can include at most 256 pilots.</p>
        <p v-if="error" role="alert" class="mt-4 rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-300">{{ error }}</p>
        <button type="submit" :disabled="!canSave"
            class="mt-5 inline-flex items-center gap-2 rounded-lg bg-blue-500 px-5 py-2.5 font-semibold text-white hover:bg-blue-400 disabled:cursor-not-allowed disabled:opacity-40">
            <Icon :name="saving ? 'lucide:loader-2' : 'lucide:save'" :class="saving ? 'animate-spin' : ''" />
            {{ saving ? 'Rebuilding report…' : 'Save changes' }}
        </button>
    </form>
</template>
