<script setup lang="ts">
useHead({ title: 'Post Killmail' })
useSeoMeta({
    description: 'Manually post EVE Online killmails to EVE-KILL by pasting ESI killmail URLs.',
})

const toast = useToast()
const watcher = useKillWatcher()
const text = ref('')
const submitting = ref(false)
const lastResult = ref<{ accepted: number; rejected: number; existing: number; total: number; killmails: number[]; existingIds: number[] } | null>(null)

async function postFromClipboard() {
    try {
        const clip = await navigator.clipboard.readText()
        if (!clip?.trim()) {
            toast.info('Clipboard is empty')
            return
        }
        await submit(clip)
    } catch {
        toast.error('Could not read clipboard — paste the link into the text box instead')
    }
}

async function submitFromTextarea() {
    if (!text.value.trim()) {
        toast.info('Paste one or more killmail links first')
        return
    }
    await submit(text.value)
}

async function submit(payload: string) {
    submitting.value = true
    lastResult.value = null
    try {
        const res = await apiFetch<{ accepted: number; rejected: number; existing: number; total: number; killmails: number[]; existingIds: number[]; message?: string }>(
            '/api/killmail/post',
            { method: 'POST', body: { text: payload } },
        )
        lastResult.value = res
        if (res.total === 0) {
            toast.info(res.message || 'No valid ESI killmail links found')
            return
        }
        // Existing killmails: if there's exactly one and it already existed, jump to it.
        if (res.total === 1 && res.existing === 1) {
            const id = res.killmails[0]!
            toast.info(`Killmail #${id} already exists`)
            text.value = ''
            await navigateTo(`/kill/${id}`)
            return
        }
        // Watch any newly queued killmails — pop a notification when each arrives
        const newIds = res.killmails.filter(id => !res.existingIds.includes(id))
        if (newIds.length > 0) watcher.watch(newIds)

        const parts: string[] = []
        if (res.accepted) parts.push(`${res.accepted} queued — watching for arrival`)
        if (res.existing) parts.push(`${res.existing} already exist`)
        if (res.rejected) parts.push(`${res.rejected} failed`)
        const msg = parts.join(', ')
        if (res.rejected) toast.error(msg)
        else toast.success(msg)
        if (res.accepted + res.existing === res.total) text.value = ''
    } catch (err: any) {
        toast.error(extractFetchError(err, 'Failed to queue'))
    } finally {
        submitting.value = false
    }
}
</script>

<template>
    <InfoPage
        title="Post Killmail"
        subtitle="Paste an ESI killmail URL — or paste one anywhere on the site and it is queued automatically."
        icon="lucide:clipboard-paste"
    >
        <!-- The instruction itself is the page subtitle now; only the example
             belongs in the body. -->
        <p class="text-fine text-gray-600 mb-6">
            Example: <code class="text-blue-400/70">https://esi.evetech.net/killmails/134199691/b6b969654468cefe5891b40b054d20fad8787f64/</code>
        </p>

        <!-- Big blue button: post from clipboard -->
        <button
            type="button"
            class="w-full flex items-center justify-center gap-3 px-6 py-5 mb-6 rounded-xl bg-blue-500 hover:bg-blue-600 active:bg-blue-700 text-white font-semibold text-lg shadow-lg shadow-blue-500/20 transition-colors disabled:opacity-50"
            :disabled="submitting"
            @click="postFromClipboard"
        >
            <Icon name="lucide:clipboard-paste" class="text-2xl" />
            Post from clipboard
        </button>

        <!-- Bulk text area -->
        <div class="rounded-xl border border-white/[0.08] bg-white/[0.02] p-4">
            <label class="block text-sm font-medium text-gray-300 mb-2">
                Or paste many links (one per line)
            </label>
            <textarea
                v-model="text"
                rows="8"
                placeholder="https://esi.evetech.net/killmails/134199691/b6b969654468cefe5891b40b054d20fad8787f64/&#10;https://esi.evetech.net/killmails/.../"
                class="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-sm text-gray-200 placeholder-gray-600 font-mono focus:outline-none focus:border-blue-400/40"
                :disabled="submitting"
            ></textarea>
            <button
                type="button"
                class="mt-3 px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 active:bg-blue-700 text-white font-medium text-sm transition-colors disabled:opacity-50"
                :disabled="submitting || !text.trim()"
                @click="submitFromTextarea"
            >
                <Icon v-if="submitting" name="lucide:loader" class="text-base animate-spin mr-1" />
                Queue {{ submitting ? '...' : 'killmails' }}
            </button>
        </div>

        <!-- Result -->
        <div v-if="lastResult && lastResult.total > 0" class="mt-4 p-3 rounded-lg bg-white/[0.02] border border-white/[0.08] text-xs text-gray-400">
            <div class="font-medium text-gray-300 mb-1">Last submission</div>
            <div>Total parsed: <span class="text-white tabular-nums">{{ lastResult.total }}</span></div>
            <div>Accepted: <span class="text-green-400 tabular-nums">{{ lastResult.accepted }}</span></div>
            <div v-if="lastResult.rejected">Rejected: <span class="text-red-400 tabular-nums">{{ lastResult.rejected }}</span></div>
            <div class="mt-2 flex flex-wrap gap-1.5">
                <NuxtLink v-for="id in lastResult.killmails.slice(0, 20)" :key="id" :to="`/kill/${id}`" class="px-2 py-0.5 rounded bg-blue-500/10 text-blue-400 hover:bg-blue-500/20">
                    #{{ id }}
                </NuxtLink>
                <span v-if="lastResult.killmails.length > 20" class="px-2 py-0.5 text-gray-600">+{{ lastResult.killmails.length - 20 }} more</span>
            </div>
        </div>
    </InfoPage>
</template>
