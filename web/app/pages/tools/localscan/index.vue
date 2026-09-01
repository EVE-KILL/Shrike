<script setup lang="ts">
useHead({ title: 'Local Scan' })
useSeoMeta({
    description: 'Paste your EVE Online local chat to analyze pilot affiliations, threat levels, and fleet composition.',
    ogTitle: 'Local Scan — EVE-KILL',
})

interface ScanResult {
    alliances: Record<number, any>
    corporations: Record<number, any>
    unresolved: string[]
    totalCharacters: number
    totalDangerous: number
}

const { isAuthenticated, login } = useAuth()

const isLoading = ref(false)
const isSaving = ref(false)
const error = ref<string | null>(null)
const scanResult = ref<ScanResult | null>(null)
const rawNames = ref<string[]>([])

const processLocal = async (text: string) => {
    if (!text.trim()) {
        error.value = 'Clipboard is empty — copy your local chat from EVE first.'
        return
    }

    const names = text.split('\n').map(n => n.trim()).filter(n => n.length > 0 && n.length <= 64)
    if (names.length === 0) {
        error.value = 'No valid character names found.'
        return
    }

    try {
        isLoading.value = true
        error.value = null
        scanResult.value = null
        rawNames.value = names

        const data = await apiFetch<ScanResult>('/api/tools/localscan', {
            method: 'POST',
            body: names,
        })

        scanResult.value = data
    } catch (e: any) {
        error.value = e?.data?.message || 'Failed to process local scan.'
    } finally {
        isLoading.value = false
    }
}

const handlePasteButton = async () => {
    try {
        const text = await navigator.clipboard.readText()
        await processLocal(text)
    } catch {
        error.value = 'Could not read clipboard. Try pressing Ctrl+V instead.'
    }
}

const saveAndShare = async () => {
    if (!scanResult.value || !rawNames.value.length) return
    try {
        isSaving.value = true
        const { hash } = await apiFetch<{ hash: string }>('/api/tools/localscan/save', {
            method: 'POST',
            body: { names: rawNames.value, result: scanResult.value },
        })
        navigateTo(`/tools/localscan/${hash}`)
    } catch (e: any) {
        error.value = e?.data?.message || 'Failed to save scan.'
    } finally {
        isSaving.value = false
    }
}

const onDocumentPaste = async (event: ClipboardEvent) => {
    const text = event.clipboardData?.getData('text') || ''
    await processLocal(text)
}

onMounted(() => document.addEventListener('paste', onDocumentPaste))
onUnmounted(() => document.removeEventListener('paste', onDocumentPaste))
</script>

<template>
    <div>
        <PageHeader class="mb-4" title="Local Scan" eyebrow="Local intelligence" icon="lucide:scan-line"
            description="Paste the character list from local chat to identify pilots, corporations, alliances, and potential threats." />

        <!-- Input -->
        <div class="glass-panel p-6 mb-4">
            <p class="text-sm text-gray-500 mb-4">
                Copy the character names from your EVE local chat and paste them here.
            </p>

            <div class="rounded-lg p-8 text-center">
                <p class="text-gray-400 text-sm mb-2">Press <kbd class="px-1.5 py-0.5 text-xs bg-white/[0.08] rounded font-mono text-gray-300">Ctrl+V</kbd> anywhere on this page</p>
                <p class="text-xs text-gray-600 mb-4">or use the button below</p>
                <button
                    class="px-4 py-2 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 text-sm rounded border border-blue-500/20 transition-colors disabled:opacity-50"
                    :disabled="isLoading"
                    @click="handlePasteButton"
                >
                    <template v-if="isLoading">
                        <Icon name="lucide:loader-2" class="animate-spin mr-1.5 inline-block" />
                        Processing...
                    </template>
                    <template v-else>
                        <Icon name="lucide:clipboard-paste" class="mr-1.5 inline-block" />
                        Paste from Clipboard
                    </template>
                </button>
            </div>

            <div v-if="error" class="mt-4 p-3 rounded bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
                {{ error }}
            </div>
        </div>

        <!-- Save / auth hint -->
        <div v-if="scanResult" class="mb-4 flex items-center justify-end gap-3">
            <button
                v-if="isAuthenticated"
                class="px-3 py-1.5 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 text-xs rounded border border-blue-500/20 transition-colors disabled:opacity-50"
                :disabled="isSaving"
                @click="saveAndShare"
            >
                <template v-if="isSaving">
                    <Icon name="lucide:loader-2" class="animate-spin mr-1 inline-block" />
                    Saving...
                </template>
                <template v-else>
                    <Icon name="lucide:share-2" class="mr-1 inline-block" />
                    Save &amp; Share
                </template>
            </button>
            <span v-else class="text-xs text-gray-600">
                <button type="button" class="text-blue-400/60 hover:text-blue-400 hover:underline cursor-pointer" @click="login()">Log in</button> to save and share scans
            </span>
        </div>

        <!-- Results -->
        <ToolsLocalscanResult v-if="scanResult" :result="scanResult" />
    </div>
</template>
