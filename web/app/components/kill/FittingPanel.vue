<script setup lang="ts">
// Fitting wheel + "Load in Editor" button — shared by the desktop and mobile
// layouts of the killmail page. The wrapper stays position:relative so the
// button can sit in the top-right corner over the wheel.
import { killmailFitToEditorUrl } from '~/composables/fit/killmailToFit'

const props = defineProps<{
    killmailId: number
    shipTypeId: number
    shipName: string | null
    items: any[]
    /** Extra class for the fitting wheel itself (mobile passes mb-4). */
    wheelClass?: string
}>()

// "Load in Fitting Editor" button state + handler. Calls the
// /api/killmail/:id/fit endpoint to server-extract the fit (same
// module/charge/drone shape the fittings-tab "Load in Editor" uses)
// then hands off to killmailFitToEditorUrl which encodes it as ESF v3
// and navigates to /fits/create?fit=...
const loadingFitEditor = ref(false)
async function openFitInEditor() {
    if (!props.killmailId || loadingFitEditor.value) return
    loadingFitEditor.value = true
    try {
        const data = await apiFetch<{
            shipTypeId: number
            name: string
            modules: { slot_group: number; ordinal: number; type_id: number; name: string | null; charge_type_id: number | null }[]
            drones: { type_id: number; name: string | null; quantity: number }[]
        }>(`/api/killmail/${props.killmailId}/fit`)
        const url = await killmailFitToEditorUrl({
            shipTypeId: data.shipTypeId,
            modules: data.modules,
            drones: data.drones,
            name: data.name,
            description: `From killmail #${props.killmailId}`,
        })
        await navigateTo(url)
    } finally {
        loadingFitEditor.value = false
    }
}
</script>

<template>
    <div class="relative">
        <button
            type="button"
            class="absolute top-1 right-1 z-10 inline-flex items-center gap-1 px-2 py-1 rounded bg-blue-500/[0.12] hover:bg-blue-500/[0.20] border border-blue-500/30 text-fine font-bold uppercase tracking-[0.1em] text-blue-300 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-wait"
            :disabled="loadingFitEditor"
            v-tooltip="'Open this fit in the editor'"
            @click="openFitInEditor"
        >
            <Icon :name="loadingFitEditor ? 'lucide:loader-2' : 'lucide:wrench'" :class="['w-3 h-3', loadingFitEditor && 'animate-spin']" />
            <span>Load in Editor</span>
        </button>
        <KillFittingWheel :ship-type-id="shipTypeId" :ship-name="shipName" :items="items" :class="wheelClass" />
    </div>
</template>
