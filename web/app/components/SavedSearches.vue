<script setup lang="ts">
import { onClickOutside, useElementBounding, useWindowSize } from "@vueuse/core"
import { readLocalSearches, savedSearchDocument, type LocalSavedSearch, type SavedSearchDocument } from '~/utils/savedSearches'

interface AccountSearch { id: number; name: string; document: SavedSearchDocument; revision: number; updated_at: string }
const props = defineProps<{ current: () => SavedSearchDocument }>()
const emit = defineEmits<{ load: [document: SavedSearchDocument] }>()
const { user } = useAuth()
const account = ref<AccountSearch[]>([])
const local = ref<LocalSavedSearch[]>([])
const open = ref(false)
const trigger = ref<HTMLElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const { bottom, right } = useElementBounding(trigger)
const { width: viewportWidth, height: viewportHeight } = useWindowSize()
const panelStyle = computed(() => {
    const width = Math.min(384, viewportWidth.value - 32)
    const top = Math.max(16, Math.min(bottom.value + 8, viewportHeight.value - 220))
    return { width: `${width}px`, left: `${Math.max(16, Math.min(right.value - width, viewportWidth.value - width - 16))}px`, top: `${top}px`, maxHeight: `${viewportHeight.value - top - 16}px` }
})
onClickOutside(panel, () => { open.value = false }, { ignore: [trigger] })

const busy = ref(false)
const loading = ref(false)
const ready = ref(false)
const error = ref('')
const message = ref('')
const editor = ref<{ kind: 'create' | 'rename' | 'replace' | 'import'; name: string; document: SavedSearchDocument; row?: AccountSearch; localIndex?: number } | null>(null)
const deleting = ref<AccountSearch | number | null>(null)
const storageKey = 'evekill-saved-searches'
let generation = 0

function showError(err: any) { error.value = err?.data?.error || err?.data?.detail || err?.data?.message || err?.message || 'Unable to save search' }
async function refresh() {
    const epoch = ++generation
    account.value = []
    ready.value = false
    if (!user.value) { loading.value = false; return }
    loading.value = true
    try {
        const result = await apiFetch<{ searches: AccountSearch[] }>('/api/me/saved-searches')
        if (epoch === generation) { account.value = result.searches ?? []; ready.value = true }
    } catch (err) { if (epoch === generation) showError(err) }
    finally { if (epoch === generation) loading.value = false }
}
onMounted(() => {
    try { local.value = readLocalSearches(localStorage.getItem(storageKey)) } catch (err) { showError(err) }
    refresh()
})
watch(() => user.value?.characterId, () => {
    editor.value = null; deleting.value = null; error.value = ''; message.value = ''
    refresh()
})
onBeforeUnmount(() => { generation++ })

function begin(kind: 'create' | 'rename' | 'replace' | 'import', row?: AccountSearch, localIndex?: number) {
    error.value = ''; message.value = ''; deleting.value = null
    try {
        const browser = localIndex == null ? undefined : local.value[localIndex]
        const document = kind === 'rename' ? row!.document : kind === 'import' ? browser!.document ?? savedSearchDocument(browser!.filters) : props.current()
        editor.value = { kind, name: row?.name ?? browser?.name ?? '', document, row, localIndex }
        open.value = true
    } catch (err) { showError(err); open.value = true }
}
function persistLocal(next: LocalSavedSearch[]) {
    localStorage.setItem(storageKey, JSON.stringify(next))
    local.value = next
}
async function save() {
    if (!editor.value || busy.value) return
    const draft = editor.value
    const name = draft.name.trim()
    if (!name || [...name].length > 100) { error.value = 'Use a name of 1–100 characters'; return }
    busy.value = true; error.value = ''
    const owner = user.value?.characterId
    try {
        if (owner) {
            if (!ready.value) throw new Error('Refresh account searches before saving')
            await apiFetch(`/api/me/saved-searches${draft.row ? `/${draft.row.id}` : ''}`, {
                method: draft.row ? 'PUT' : 'POST',
                body: { name, document: draft.document, revision: draft.row?.revision ?? 0 },
            })
            if (owner !== user.value?.characterId) return
            await refresh()
            message.value = draft.kind === 'import' ? 'Imported. The browser copy has been kept.' : 'Saved to your account.'
        } else {
            if (local.value.length >= 50) throw new Error('Browser saved-search limit reached (50)')
            if (local.value.some(row => row.name.trim().toLowerCase() === name.toLowerCase())) throw new Error('A browser search with this name already exists. Choose another name.')
            persistLocal([...local.value, { name, filters: JSON.stringify(draft.document.filters), document: draft.document, date: new Date().toISOString() }])
            message.value = 'Saved in this browser.'
        }
        editor.value = null
    } catch (err) { if (owner === user.value?.characterId) showError(err) }
    finally { busy.value = false }
}
async function remove() {
    if (deleting.value == null || busy.value) return
    busy.value = true; error.value = ''
    const owner = user.value?.characterId
    try {
        if (typeof deleting.value === 'number') persistLocal(local.value.filter((_, i) => i !== deleting.value))
        else {
            const row = deleting.value
            await apiFetch(`/api/me/saved-searches/${row.id}`, { method: 'DELETE', params: { revision: row.revision } })
            if (owner !== user.value?.characterId) return
            await refresh()
        }
        deleting.value = null
    } catch (err) { if (owner === user.value?.characterId) showError(err) }
    finally { busy.value = false }
}
function load(row: AccountSearch | LocalSavedSearch) {
    try {
        emit('load', row.document ?? savedSearchDocument((row as LocalSavedSearch).filters))
        open.value = false
    } catch (err) { showError(err) }
}
</script>

<template>
    <div ref="trigger" class="relative flex gap-2">
        <button class="saved-button" :disabled="busy || (!!user && !ready)" @click="begin('create')"><Icon name="lucide:bookmark" /> Save</button>
        <button class="saved-button" :aria-expanded="open" @click="open = !open"><Icon name="lucide:folder-open" /> Saved {{ user ? `(${account.length})` : `(${local.length})` }}</button>
        <Teleport to="body">
        <div v-if="open" ref="panel" role="dialog" aria-label="Saved searches" :style="panelStyle" class="fixed z-[100] overflow-y-auto rounded-lg bg-[#10131a] border border-white/10 shadow-2xl p-3" @keydown.esc="open = false">
            <div class="flex items-center justify-between mb-3">
                <span class="text-sm font-medium text-gray-200">{{ user ? 'Account searches' : 'Browser searches' }}</span>
                <button aria-label="Close saved searches" class="text-gray-400" @click="open = false"><Icon name="lucide:x" /></button>
            </div>
            <p v-if="error" role="alert" class="text-xs text-red-400 mb-3">{{ error }}</p>
            <p v-if="message" role="status" class="text-xs text-green-400 mb-3">{{ message }}</p>
            <p v-if="loading" class="text-xs text-gray-400">Loading searches…</p>
            <button v-if="user" class="text-xs text-blue-400 mb-3" :disabled="busy || loading" @click="refresh">Refresh account searches</button>
            <form v-if="editor" class="mb-3 p-3 rounded bg-white/[0.04]" @submit.prevent="save">
                <p class="text-xs text-gray-300 mb-2">{{ editor.kind === 'import' ? 'Import browser search' : editor.kind === 'replace' ? 'Replace with current filters' : editor.kind === 'rename' ? 'Rename search' : 'Save current search' }}</p>
                <label class="block text-xs text-gray-400 mb-1" for="saved-search-name">Search name</label>
                <input id="saved-search-name" v-model="editor.name" maxlength="100" required class="w-full px-2 py-2 text-sm bg-black/30 border border-white/10 rounded text-gray-200" />
                <p class="text-xs text-gray-500 mt-2">{{ user ? 'Private to this account, available across devices.' : 'Stored in this browser. Sign in to save across devices.' }} Rolling dates update each time you open the search.</p>
                <div class="flex gap-2 mt-3"><button class="saved-button" :disabled="busy || !editor.name.trim()">{{ busy ? 'Saving…' : 'Save search' }}</button><button type="button" class="saved-button" :disabled="busy" @click="editor = null">Cancel</button></div>
            </form>
            <div v-if="deleting !== null" class="mb-3 p-3 rounded bg-red-500/10">
                <p class="text-xs text-gray-300 mb-2">Delete this saved search?</p>
                <div class="flex gap-2"><button class="saved-button" :disabled="busy" @click="remove">Delete search</button><button class="saved-button" :disabled="busy" @click="deleting = null">Cancel</button></div>
            </div>
            <template v-if="user">
                <p v-if="ready && !account.length" class="text-xs text-gray-500 py-2">No account searches yet.</p>
                <div v-for="row in account" :key="row.id" class="py-3 border-b border-white/[0.06]">
                    <button class="text-sm text-gray-200 text-left break-words" @click="load(row)">{{ row.name }}</button>
                    <div class="flex flex-wrap gap-3 mt-2 text-xs text-gray-400">
                        <button :disabled="busy" @click="begin('rename', row)">Rename</button>
                        <button :disabled="busy" @click="begin('replace', row)">Use current filters</button>
                        <button :disabled="busy" @click="deleting = row; editor = null">Delete</button>
                    </div>
                </div>
            </template>
            <p v-if="user && local.length" class="text-xs text-gray-400 mt-4 mb-1">Browser searches — import individually; existing account searches are never overwritten.</p>
            <p v-if="!user && !local.length" class="text-xs text-gray-500 py-2">No browser searches yet.</p>
            <div v-for="(row, index) in local" :key="index" class="py-3 border-b border-white/[0.06]">
                <button class="text-sm text-gray-200 text-left break-words" @click="load(row)">{{ row.name }}</button>
                <div class="flex gap-3 mt-2 text-xs text-gray-400">
                    <button v-if="user" :disabled="busy || !ready" @click="begin('import', undefined, index)">Import to account</button>
                    <button :disabled="busy" @click="deleting = index; editor = null">Delete browser copy</button>
                </div>
            </div>
        </div>
        </Teleport>
    </div>
</template>

<style scoped>
.saved-button { display: inline-flex; align-items: center; gap: .35rem; padding: .375rem .625rem; font-size: .75rem; border: 1px solid rgb(255 255 255 / .08); border-radius: .25rem; background: rgb(255 255 255 / .04); color: #9ca3af; }
.saved-button:hover { background: rgb(59 130 246 / .08); }
button:disabled { opacity: .4; cursor: not-allowed; }
</style>
