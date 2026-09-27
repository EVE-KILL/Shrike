<script setup lang="ts">
useHead({ title: 'Roam Report' })
useSeoMeta({
    description: 'Turn an EVE Online fleet list into a shareable report of recorded fights, losses, targets, and systems.',
    ogTitle: 'Roam Report — EVE-KILL',
})

const form = useState('roam-report-form', () => {
    const end = new Date()
    end.setUTCSeconds(0, 0)
    return {
        namesText: '',
        startTime: isoToEveInput(new Date(end.getTime() - 3 * 60 * 60 * 1000)),
        endTime: isoToEveInput(end),
    }
})

const isCreating = ref(false)
const error = ref('')
const names = computed(() => {
    const seen = new Set<string>()
    return form.value.namesText.split(/\r?\n/).map(name => name.trim()).filter(name => {
        if (!name || seen.has(name.toLowerCase())) return false
        seen.add(name.toLowerCase())
        return true
    })
})
const timeError = computed(() => {
    const start = eveInputToDate(form.value.startTime)
    const end = eveInputToDate(form.value.endTime)
    if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return 'Choose a start and end time.'
    if (end <= start) return 'End must be after start.'
    if (end.getTime() - start.getTime() > 72 * 60 * 60 * 1000) return 'Maximum window is 72 hours.'
    if (end.getTime() > Date.now() + 5 * 60 * 1000) return 'End cannot be in the future.'
    return ''
})
const canCreate = computed(() => names.value.length > 0 && names.value.length <= 256 && !timeError.value && !isCreating.value)

async function pasteFleet() {
    try {
        form.value.namesText = await navigator.clipboard.readText()
        error.value = ''
    } catch {
        error.value = 'Clipboard access failed. Paste the fleet list into the text box instead.'
    }
}

async function createReport() {
    if (!canCreate.value) return
    isCreating.value = true
    error.value = ''
    try {
        const result = await apiFetch<{ id: string }>('/api/tools/roam-report', {
            method: 'POST',
            body: {
                names_text: form.value.namesText,
                start_time: eveInputToIso(form.value.startTime),
                end_time: eveInputToIso(form.value.endTime),
            },
        })
        await navigateTo(`/tools/roam-report/${result.id}`)
    } catch (cause: any) {
        error.value = cause?.data?.error || cause?.data?.message || 'Could not create the report. Please try again.'
    } finally {
        isCreating.value = false
    }
}
</script>

<template>
    <div class="mx-auto max-w-5xl pb-16">
        <PageHeader
            class="mb-8"
            title="Roam Report"
            eyebrow="Fleet tools"
            icon="lucide:route"
            description="Paste the pilots from your fleet window, choose the roam window, and turn their recorded combat into a report you can share."
        />

        <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_18rem]">
            <form class="glass-panel p-5 sm:p-7" @submit.prevent="createReport">
                <div class="mb-4 flex items-start justify-between gap-4">
                    <div>
                        <h2 class="text-lg font-semibold text-white">Fleet pilots</h2>
                        <p class="mt-1 text-sm text-gray-400">Copy the character list from the in-game fleet window. Use one name per line.</p>
                    </div>
                    <button type="button" class="shrink-0 rounded-lg border border-white/10 px-3 py-2 text-xs text-blue-300 hover:bg-white/5" @click="pasteFleet">
                        <Icon name="lucide:clipboard-paste" class="mr-1" /> Paste
                    </button>
                </div>
                <textarea
                    v-model="form.namesText"
                    class="h-52 w-full resize-y rounded-lg border border-white/10 bg-black/25 p-4 font-mono text-sm leading-6 text-gray-100 outline-none placeholder:text-gray-600 focus:border-blue-500/50"
                    placeholder="Pilot One&#10;Pilot Two&#10;Pilot Three"
                    maxlength="16384"
                    aria-label="Fleet character names, one per line"
                    spellcheck="false"
                />
                <p class="mt-2 text-xs text-gray-500">{{ names.length }} unique {{ names.length === 1 ? 'pilot' : 'pilots' }} · up to 256</p>

                <div class="mt-8 border-t border-white/[0.08] pt-6">
                    <h2 class="text-lg font-semibold text-white">Roam window</h2>
                    <p class="mt-1 text-sm text-gray-400">Times are EVE time (UTC). The last three hours are filled in for you.</p>
                    <div class="mt-4 grid gap-4 sm:grid-cols-2">
                        <label class="block text-sm text-gray-300">
                            <span class="mb-2 block font-medium">Start</span>
                            <DateTimePicker v-model="form.startTime" placeholder="Choose a start" />
                        </label>
                        <label class="block text-sm text-gray-300">
                            <span class="mb-2 block font-medium">End</span>
                            <DateTimePicker v-model="form.endTime" placeholder="Choose an end" />
                        </label>
                    </div>
                    <p v-if="timeError" class="mt-3 text-sm text-amber-300">{{ timeError }}</p>
                </div>

                <p v-if="names.length > 256" class="mt-5 text-sm text-amber-300">A report can include at most 256 pilots.</p>
                <p v-if="error" role="alert" class="mt-5 rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-300">{{ error }}</p>
                <button
                    type="submit"
                    :disabled="!canCreate"
                    class="mt-7 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-blue-500 px-5 py-3 font-semibold text-white transition-colors hover:bg-blue-400 disabled:cursor-not-allowed disabled:opacity-40 sm:w-auto"
                >
                    <Icon :name="isCreating ? 'lucide:loader-2' : 'lucide:chart-no-axes-combined'" :class="isCreating ? 'animate-spin' : ''" />
                    {{ isCreating ? 'Building report…' : 'Create Roam Report' }}
                </button>
            </form>

            <aside class="space-y-4">
                <div class="glass-panel p-5">
                    <Icon name="lucide:map-pinned" class="text-2xl text-blue-400" />
                    <h2 class="mt-3 font-semibold text-white">A combat trail</h2>
                    <p class="mt-2 text-sm leading-6 text-gray-400">See the systems and regions where these pilots appear on killmails, ordered by time.</p>
                </div>
                <div class="glass-panel p-5">
                    <Icon name="lucide:swords" class="text-2xl text-amber-400" />
                    <h2 class="mt-3 font-semibold text-white">Fights and outcomes</h2>
                    <p class="mt-2 text-sm leading-6 text-gray-400">Review engagements, kills, losses, ISK, opponents, and each pilot's recorded contribution.</p>
                </div>
                <p class="px-1 text-xs leading-5 text-gray-500">Reports use recorded killmails. They cannot show jumps without combat, ships that never appeared on a killmail, or prove that every listed pilot flew together.</p>
            </aside>
        </div>
    </div>
</template>
