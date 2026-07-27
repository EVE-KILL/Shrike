<script setup lang="ts">
// ── Announcements section state ─────────────────────────────────────────────
interface AdminAnnouncement {
    id: number
    tier: number
    title: string
    body_md: string
    body_html: string
    color: string
    icon: string | null
    link_url: string | null
    link_label: string | null
    starts_at: string
    expires_at: string
    created_by: number
    created_at: string
    updated_at: string
    archived_at: string | null
}

// One of: all | active | scheduled | expired | archived — goes straight to ?status=
const annStatusFilter = ref('active')
const { data: announcementsData, refresh: refreshAnnouncements } = useApiFetch<{ announcements: AdminAnnouncement[] }>('/api/admin/announcements', {
    query: { status: annStatusFilter, limit: 100 },
    immediate: true,
    lazy: true,
    watch: [annStatusFilter],
})

const annEditOpen = ref(false)
const annEditId = ref<number | null>(null)
const annForm = ref({
    tier: 2 as number,
    title: '',
    body_md: '',
    color: 'info',
    icon: '',
    link_url: '',
    link_label: '',
    starts_at: '',
    expires_at: '',
    duration: 15 as number, // ticker only: minutes until auto-expire
})

const tickerDurations = [
    { value: 5, label: '5 min' },
    { value: 10, label: '10 min' },
    { value: 15, label: '15 min' },
    { value: 30, label: '30 min' },
    { value: 60, label: '1 hour' },
] as const

function annNewForm() {
    annEditId.value = null
    annFormError.value = ''
    const now = new Date()
    const inOneWeek = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)
    annForm.value = {
        tier: 2,
        title: '',
        body_md: '',
        color: 'info',
        icon: '',
        link_url: '',
        link_label: '',
        starts_at: isoToEveInput(now),
        expires_at: isoToEveInput(inOneWeek),
        duration: 15,
    }
    annEditOpen.value = true
}

function annEditForm(a: AdminAnnouncement) {
    annEditId.value = a.id
    annFormError.value = ''
    // For tickers, compute remaining duration from expires_at
    const remainingMs = new Date(a.expires_at).getTime() - Date.now()
    const remainingMin = Math.max(5, Math.round(remainingMs / 60000))
    // Snap to nearest preset or keep raw
    const snapped = tickerDurations.find(d => d.value >= remainingMin)?.value ?? 60
    annForm.value = {
        tier: a.tier,
        title: a.title,
        body_md: a.body_md,
        color: a.color,
        icon: a.icon || '',
        link_url: a.link_url || '',
        link_label: a.link_label || '',
        starts_at: isoToEveInput(a.starts_at),
        expires_at: isoToEveInput(a.expires_at),
        duration: a.tier === 1 ? snapped : 15,
    }
    annEditOpen.value = true
}

const toast = useToast()
const annSaving = ref(false)
const annFormError = ref('')

async function annSave() {
    const title = annForm.value.title.trim()
    if (!title) { annFormError.value = 'Title is required'; return }
    annFormError.value = ''
    const isTicker = annForm.value.tier === 1
    const now = new Date()

    // Ticker: starts now, expires after selected duration
    // Banner/Modal: use the explicit datetime fields
    const startsAt = isTicker ? now.toISOString() : eveInputToIso(annForm.value.starts_at)
    const expiresAt = isTicker
        ? new Date(now.getTime() + annForm.value.duration * 60 * 1000).toISOString()
        : eveInputToIso(annForm.value.expires_at)

    const payload = {
        tier: annForm.value.tier,
        title: annForm.value.title,
        body_md: annForm.value.body_md,
        color: annForm.value.color,
        starts_at: startsAt,
        expires_at: expiresAt,
        icon: annForm.value.icon || null,
        link_url: annForm.value.link_url || null,
        link_label: annForm.value.link_label || null,
    }

    annSaving.value = true
    try {
        if (annEditId.value) {
            await apiFetch(`/api/admin/announcements/${annEditId.value}`, { method: 'PATCH', body: payload })
        } else {
            await apiFetch('/api/admin/announcements', { method: 'POST', body: payload })
        }
        annEditOpen.value = false
        await refreshAnnouncements()
    } catch (err: any) {
        // Inline rather than a toast: the modal stays open on failure.
        annFormError.value = extractFetchError(err, 'Failed to save')
    } finally {
        annSaving.value = false
    }
}

// Archiving pulls an announcement off every user's screen, so it arms on the
// first click and fires on the second.
const { pendingId: confirmArchiveId, confirm: confirmArchive } = useConfirmTwice()

async function annArchive(id: number) {
    if (!confirmArchive(id)) return
    try {
        await apiFetch(`/api/admin/announcements/${id}/archive`, { method: 'POST' })
        await refreshAnnouncements()
    } catch (err: any) {
        toast.error(extractFetchError(err, 'Failed to archive'))
    }
}



function annStatusLabel(a: AdminAnnouncement): { label: string; cls: string } {
    if (a.archived_at) return { label: 'Archived', cls: 'bg-gray-500/20 text-gray-400 border-gray-500/30' }
    const now = Date.now()
    const starts = new Date(a.starts_at).getTime()
    const expires = new Date(a.expires_at).getTime()
    if (now < starts) return { label: 'Scheduled', cls: 'bg-blue-500/20 text-blue-300 border-blue-500/30' }
    if (now > expires) return { label: 'Expired', cls: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' }
    return { label: 'Active', cls: 'bg-green-500/20 text-green-300 border-green-500/30' }
}

const tierLabels: Record<number, string> = { 1: 'Ticker', 2: 'Banner', 3: 'Modal' }
const tierColors: Record<number, string> = { 1: 'text-blue-400', 2: 'text-yellow-400', 3: 'text-red-400' }

// The parent keeps this component alive (<KeepAlive>) across tab switches.
// Matching the old tab-switch watcher: on re-activation, refetch only if no
// data ever loaded (e.g. the first fetch errored).
let activatedOnce = false
onActivated(() => {
    if (!activatedOnce) { activatedOnce = true; return }
    if (!announcementsData.value) refreshAnnouncements()
})

// Helpers
</script>

<template>
    <!-- ═══════════════ ANNOUNCEMENTS ═══════════════ -->
    <div class="space-y-4">
        <!-- Header + Create button -->
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
                <SelectMenu
                    v-model="annStatusFilter"
                    :options="[
                        { value: 'all', label: 'All announcements' },
                        { value: 'active', label: 'Active' },
                        { value: 'scheduled', label: 'Scheduled' },
                        { value: 'expired', label: 'Expired' },
                        { value: 'archived', label: 'Archived' },
                    ]"
                />
            </div>
            <button
                class="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium transition-colors cursor-pointer"
                @click="annNewForm"
            >
                <Icon name="lucide:plus" class="text-sm" />
                New Announcement
            </button>
        </div>

        <!-- Announcements list -->
        <div v-if="!announcementsData?.announcements?.length" class="text-center py-12 text-gray-500 text-sm">
            No announcements found.
        </div>
        <div v-else class="space-y-2">
            <div
                v-for="a in announcementsData.announcements"
                :key="a.id"
                class="glass-panel p-4"
            >
                <div class="flex items-start justify-between gap-4">
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2 mb-1">
                            <span
                                class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border"
                                :class="annStatusLabel(a).cls"
                            >
                                {{ annStatusLabel(a).label }}
                            </span>
                            <span class="text-xs font-medium" :class="tierColors[a.tier] || 'text-gray-400'">
                                {{ tierLabels[a.tier] || `Tier ${a.tier}` }}
                            </span>
                            <span class="text-[10px] text-gray-600">#{{ a.id }}</span>
                        </div>
                        <div class="text-sm font-medium text-white">{{ a.title }}</div>
                        <div v-if="a.body_md" class="text-xs text-gray-500 mt-1 line-clamp-2">{{ a.body_md }}</div>
                        <div class="flex items-center gap-3 mt-2 text-[10px] text-gray-600">
                            <span>Start: {{ fmtDate(a.starts_at) }}</span>
                            <span>Expires: {{ fmtDate(a.expires_at) }}</span>
                            <span v-if="a.link_url" class="text-blue-400/60">Has link</span>
                        </div>
                    </div>
                    <div class="flex items-center gap-1 flex-shrink-0">
                        <button
                            class="flex items-center justify-center w-7 h-7 rounded-md text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors cursor-pointer"
                            v-tooltip="'Edit'"
                            @click="annEditForm(a)"
                        >
                            <Icon name="lucide:pencil" class="text-sm" />
                        </button>
                        <button
                            v-if="!a.archived_at"
                            class="flex items-center justify-center h-7 rounded-md transition-colors cursor-pointer"
                            :class="confirmArchiveId === a.id
                                ? 'px-2 gap-1 text-red-300 bg-red-500/15 ring-1 ring-red-500/40'
                                : 'w-7 text-gray-500 hover:text-red-400 hover:bg-red-500/[0.08]'"
                            v-tooltip="confirmArchiveId === a.id ? 'Click again to archive — it stops showing to users' : 'Archive'"
                            @click="annArchive(a.id)"
                        >
                            <Icon name="lucide:archive" class="text-sm" />
                            <span v-if="confirmArchiveId === a.id" class="text-fine font-medium">Confirm</span>
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Create / Edit modal -->
        <Modal v-model="annEditOpen" :title="annEditId ? 'Edit Announcement' : 'New Announcement'" max-width="max-w-lg">
            <div class="space-y-4">
                <!-- Tier -->
                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Tier</label>
                    <div class="flex gap-2">
                        <button
                            v-for="t in [1, 2, 3]" :key="t"
                            class="flex-1 px-3 py-2 rounded-lg text-xs font-medium border transition-colors cursor-pointer"
                            :class="annForm.tier === t
                                ? 'bg-blue-500/20 border-blue-500/40 text-blue-300'
                                : 'bg-white/[0.02] border-white/[0.08] text-gray-500 hover:text-gray-300'"
                            @click="annForm.tier = t"
                        >
                            {{ tierLabels[t] }}
                        </button>
                    </div>
                </div>

                <!-- Title -->
                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Title *</label>
                    <input
                        v-model="annForm.title"
                        type="text"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50"
                        placeholder="Announcement title"
                    >
                </div>

                <!-- Body (markdown) -->
                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Body (markdown, optional)</label>
                    <textarea
                        v-model="annForm.body_md"
                        rows="4"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50 resize-y"
                        placeholder="Optional markdown body..."
                    />
                </div>

                <!-- Color -->
                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Color</label>
                    <div class="flex gap-2">
                        <button
                            v-for="c in ['info', 'warning', 'danger', 'success']" :key="c"
                            class="flex-1 px-3 py-2 rounded-lg text-xs font-medium border transition-colors capitalize cursor-pointer"
                            :class="annForm.color === c
                                ? (c === 'info' ? 'bg-blue-500/20 border-blue-500/40 text-blue-300'
                                  : c === 'warning' ? 'bg-yellow-500/20 border-yellow-500/40 text-yellow-400'
                                  : c === 'danger' ? 'bg-red-500/20 border-red-500/40 text-red-300'
                                  : 'bg-green-500/20 border-green-500/40 text-green-300')
                                : 'bg-white/[0.02] border-white/[0.08] text-gray-500 hover:text-gray-300'"
                            @click="annForm.color = c"
                        >
                            {{ c }}
                        </button>
                    </div>
                </div>

                <!-- Icon -->
                <div>
                    <label class="block text-xs font-medium text-gray-400 mb-1">Icon (lucide name, optional)</label>
                    <input
                        v-model="annForm.icon"
                        type="text"
                        class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50"
                        placeholder="e.g. lucide:megaphone"
                    >
                </div>

                <!-- Link -->
                <div class="grid grid-cols-2 gap-3">
                    <div>
                        <label class="block text-xs font-medium text-gray-400 mb-1">Link URL (optional)</label>
                        <input
                            v-model="annForm.link_url"
                            type="text"
                            class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50"
                            placeholder="/path or https://..."
                        >
                    </div>
                    <div>
                        <label class="block text-xs font-medium text-gray-400 mb-1">Link Label</label>
                        <input
                            v-model="annForm.link_label"
                            type="text"
                            class="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg text-sm text-white px-3 py-2 focus:outline-none focus:border-blue-500/50"
                            placeholder="Learn more"
                        >
                    </div>
                </div>

                <!-- Ticker: duration selector -->
                <div v-if="annForm.tier === 1">
                    <label class="block text-xs font-medium text-gray-400 mb-1">Auto-expire after</label>
                    <div class="flex gap-2">
                        <button
                            v-for="d in tickerDurations" :key="d.value"
                            class="flex-1 px-3 py-2 rounded-lg text-xs font-medium border transition-colors cursor-pointer"
                            :class="annForm.duration === d.value
                                ? 'bg-blue-500/20 border-blue-500/40 text-blue-300'
                                : 'bg-white/[0.02] border-white/[0.08] text-gray-500 hover:text-gray-300'"
                            @click="annForm.duration = d.value"
                        >
                            {{ d.label }}
                        </button>
                    </div>
                    <p class="text-[10px] text-gray-600 mt-1">Starts immediately, disappears after the selected duration.</p>
                </div>

                <!-- Banner / Modal: start + expire datetimes -->
                <div v-else class="grid grid-cols-2 gap-3">
                    <div>
                        <label class="block text-xs font-medium text-gray-400 mb-1">Starts at <span class="text-gray-600 font-normal">(EVE time)</span></label>
                        <DateTimePicker v-model="annForm.starts_at" :clearable="false" placeholder="Start" />
                    </div>
                    <div>
                        <label class="block text-xs font-medium text-gray-400 mb-1">Expires at * <span class="text-gray-600 font-normal">(EVE time)</span></label>
                        <DateTimePicker v-model="annForm.expires_at" :clearable="false" placeholder="Expiry" />
                    </div>
                </div>
            </div>

            <template #footer>
                <div class="flex items-center justify-end gap-3">
                    <p v-if="annFormError" class="flex-1 text-xs text-red-400 flex items-center gap-1.5">
                        <Icon name="lucide:alert-circle" class="text-sm flex-shrink-0" />
                        {{ annFormError }}
                    </p>
                    <button
                        class="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white transition-colors cursor-pointer"
                        @click="annEditOpen = false"
                    >
                        Cancel
                    </button>
                    <button
                        class="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                        :disabled="annSaving"
                        @click="annSave"
                    >
                        <Icon v-if="annSaving" name="lucide:loader-2" class="text-sm animate-spin mr-1 inline" />
                        {{ annEditId ? 'Update' : 'Create' }}
                    </button>
                </div>
            </template>
        </Modal>
    </div>
</template>
