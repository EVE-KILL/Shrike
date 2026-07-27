<script setup lang="ts">
// Unified moderation queue state (bios + comments)
interface AiVerdict {
    ai_action: string | null
    ai_category: string | null
    ai_max_score: number | null
    ai_source: string | null
}
interface ModerationQueueItem extends AiVerdict {
    id: number
    target_kind: number
    target_id: number
    body: string
    body_format: string | null
    rendered_html: string | null
    character_id: number
    character_name: string
    corporation_id: number | null
    corporation_name: string | null
    alliance_id: number | null
    alliance_name: string | null
    status: number
    submitted_at: string
    reviewed_at: string | null
    reviewed_by: number | null
    review_notes: string | null
    comment_context: { target_type: number; target_id: number; target_slug: string | null } | null
}
interface ModerationQueueResponse {
    items: ModerationQueueItem[]
    nextCursor: number | null
    counts: { pending: number; pending_comments: number; pending_bios: number; total: number }
}

const modKindFilter = ref<'all' | 'comments' | 'bios'>('all')
const modStatusFilter = ref<'pending' | 'approved' | 'rejected' | 'all'>('pending')
const { data: moderationData, refresh: refreshModeration, pending: moderationPending } = useApiFetch<ModerationQueueResponse>('/api/admin/moderation/queue', {
    query: { kind: modKindFilter, status: modStatusFilter, limit: 100 },
    immediate: true,
    lazy: true,
    watch: [modKindFilter, modStatusFilter],
})

const toast = useToast()

// Which row is mid-request, so its buttons can disable rather than accept a
// second click while the first is still in flight.
const actingId = ref<number | null>(null)

const reviewModerationItem = async (id: number, action: 'approve' | 'reject') => {
    actingId.value = id
    try {
        await apiFetch(`/api/admin/moderation/${id}/${action}`, { method: 'POST' })
        await refreshModeration()
    } catch (err: any) {
        toast.error(extractFetchError(err, `Failed to ${action}`))
    } finally {
        actingId.value = null
    }
}

// 0=comment, 1=bio_char, 2=bio_corp, 3=bio_alliance
const moderationKindLabel = (kind: number): string => {
    if (kind === 0) return 'Comment'
    if (kind === 1) return 'Character bio'
    if (kind === 2) return 'Corporation bio'
    if (kind === 3) return 'Alliance bio'
    return 'Unknown'
}

// Resolve a link to whatever the moderation item is about:
//   - Comment: link to the comment's own target (killmail, character, ...)
//   - Bio: link to the entity the bio belongs to
const moderationTargetHref = (item: ModerationQueueItem): string => {
    if (item.target_kind === 0 && item.comment_context) {
        const t = item.comment_context.target_type
        const tid = item.comment_context.target_id
        const slug = item.comment_context.target_slug
        if (t === 1) return `/kill/${tid}`
        if (t === 2) return `/character/${tid}`
        if (t === 3) return `/corporation/${tid}`
        if (t === 4) return `/alliance/${tid}`
        if (t === 5) return `/system/${tid}`
        if (t === 7) return `/battle/${tid}`
        if (t === 8 && slug) return `/fit/${slug}`
        return `/comments/${item.target_id}`
    }
    if (item.target_kind === 1) return `/character/${item.target_id}`
    if (item.target_kind === 2) return `/corporation/${item.target_id}`
    if (item.target_kind === 3) return `/alliance/${item.target_id}`
    return '#'
}

const moderationStatusLabel = (status: number): { label: string; cls: string } => {
    if (status === 0) return { label: 'Pending', cls: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' }
    if (status === 1) return { label: 'Approved', cls: 'bg-green-500/20 text-green-300 border-green-500/30' }
    if (status === 2) return { label: 'Rejected', cls: 'bg-red-500/20 text-red-300 border-red-500/30' }
    if (status === 3) return { label: 'Auto-approved', cls: 'bg-green-500/10 text-green-400/70 border-green-500/20' }
    if (status === 4) return { label: 'Auto-rejected', cls: 'bg-red-500/10 text-red-400/70 border-red-500/20' }
    return { label: String(status), cls: 'bg-white/[0.04] text-gray-400' }
}

// The parent keeps this component alive (<KeepAlive>) across tab switches.
// Matching the old tab-switch watcher: on re-activation, refetch only if no
// data ever loaded (e.g. the first fetch errored).
let activatedOnce = false
onActivated(() => {
    if (!activatedOnce) { activatedOnce = true; return }
    if (!moderationData.value) refreshModeration()
})
</script>

<template>
    <!-- ═══════════════ UNIFIED MODERATION QUEUE ═══════════════ -->
    <div class="space-y-4">
        <!-- Kind + status filter pills -->
        <div class="flex items-center gap-2 flex-wrap">
            <div class="flex items-center gap-1 rounded-md bg-white/[0.03] border border-white/[0.06] p-1">
                <button
                    v-for="k in ['all', 'comments', 'bios'] as const"
                    :key="k"
                    class="px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer"
                    :class="modKindFilter === k ? 'bg-blue-500/[0.15] text-blue-300' : 'text-gray-400 hover:text-white'"
                    @click="modKindFilter = k"
                >
                    {{ k === 'all' ? 'All' : k === 'comments' ? 'Comments' : 'Bios' }}
                </button>
            </div>
            <div class="flex items-center gap-1 rounded-md bg-white/[0.03] border border-white/[0.06] p-1">
                <button
                    v-for="s in ['pending', 'approved', 'rejected', 'all'] as const"
                    :key="s"
                    class="px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer"
                    :class="modStatusFilter === s ? 'bg-blue-500/[0.15] text-blue-300' : 'text-gray-400 hover:text-white'"
                    @click="modStatusFilter = s"
                >
                    {{ s.charAt(0).toUpperCase() + s.slice(1) }}
                </button>
            </div>

            <div v-if="moderationData?.counts" class="flex items-center gap-3 text-xs text-gray-400 ml-2">
                <span class="inline-flex items-center gap-1.5">
                    <span class="w-1.5 h-1.5 rounded-full bg-yellow-400"></span>
                    {{ moderationData.counts.pending }} pending
                </span>
                <span class="text-gray-600">
                    {{ moderationData.counts.pending_comments }} comments ·
                    {{ moderationData.counts.pending_bios }} bios
                </span>
            </div>

            <button
                class="ml-auto px-3 py-1.5 rounded-md text-xs text-gray-400 hover:text-white hover:bg-white/[0.06] transition-colors cursor-pointer flex items-center gap-1"
                @click="refreshModeration()"
            >
                <Icon name="lucide:refresh-cw" class="text-xs" />
                Refresh
            </button>
        </div>

        <!-- Queue list -->
        <div v-if="moderationPending" class="text-sm text-gray-500 italic py-8 text-center">
            Loading…
        </div>
        <div v-else class="space-y-2">
            <article
                v-for="item in (moderationData?.items || [])"
                :key="item.id"
                class="glass-panel p-4"
            >
                <header class="flex items-start gap-3 mb-3">
                    <NuxtLink :to="`/character/${item.character_id}`" class="flex-shrink-0">
                        <img
                            :src="`/images/characters/${item.character_id}/portrait?size=64`"
                            :alt="item.character_name"
                            class="w-9 h-9 rounded-md"
                            loading="lazy"
                        >
                    </NuxtLink>
                    <div class="flex-1 min-w-0">
                        <div class="flex items-baseline gap-2 text-sm flex-wrap">
                            <NuxtLink :to="`/character/${item.character_id}`" class="font-semibold text-white hover:text-blue-400 truncate">
                                {{ item.character_name }}
                            </NuxtLink>
                            <span v-if="item.corporation_name" class="text-xs text-gray-600 truncate">{{ item.corporation_name }}</span>
                            <span class="text-xs text-gray-600">·</span>
                            <NuxtLink :to="moderationTargetHref(item)" class="text-xs text-blue-400 hover:underline">
                                {{ moderationKindLabel(item.target_kind) }}
                                <span v-if="item.target_kind === 0 && item.comment_context">
                                    #{{ item.comment_context.target_id }}
                                </span>
                                <span v-else>#{{ item.target_id }}</span>
                            </NuxtLink>
                            <span class="text-xs text-gray-600 ml-auto whitespace-nowrap">
                                {{ fmtDate(item.submitted_at) }}
                            </span>
                        </div>
                        <div class="flex items-center gap-2 mt-1 text-xs flex-wrap">
                            <span
                                class="inline-flex items-center px-2 py-0.5 rounded-full border font-medium"
                                :class="moderationStatusLabel(item.status).cls"
                            >
                                {{ moderationStatusLabel(item.status).label }}
                            </span>
                            <span v-if="item.ai_action" class="text-gray-500">
                                AI: {{ item.ai_action }}<span v-if="item.ai_category"> ({{ item.ai_category }})</span><span v-if="item.ai_max_score != null"> · {{ (item.ai_max_score * 100).toFixed(0) }}%</span>
                            </span>
                            <span v-if="item.body_format" class="text-gray-600">· {{ item.body_format }}</span>
                        </div>
                    </div>
                </header>

                <!-- Rendered preview -->
                <div
                    v-if="item.rendered_html"
                    class="text-sm text-gray-200 prose prose-invert max-w-none prose-sm pl-12 entity-bio"
                    v-html="item.rendered_html"
                />
                <pre
                    v-else
                    class="text-xs text-gray-400 pl-12 whitespace-pre-wrap font-mono"
                >{{ item.body }}</pre>

                <div v-if="item.review_notes" class="mt-2 pl-12 text-xs text-gray-500 italic">
                    Note: {{ item.review_notes }}
                </div>

                <!-- Actions (only for pending rows) -->
                <div v-if="item.status === 0" class="flex items-center gap-2 mt-3 pl-12">
                    <button
                        class="px-3 py-1 rounded-md text-xs bg-green-500/[0.15] text-green-300 hover:bg-green-500/[0.25] cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        :disabled="actingId === item.id"
                        @click="reviewModerationItem(item.id, 'approve')"
                    >
                        <Icon name="lucide:check" class="text-xs inline -mt-0.5" /> Approve
                    </button>
                    <button
                        class="px-3 py-1 rounded-md text-xs bg-red-500/[0.15] text-red-300 hover:bg-red-500/[0.25] cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        :disabled="actingId === item.id"
                        @click="reviewModerationItem(item.id, 'reject')"
                    >
                        <Icon name="lucide:x" class="text-xs inline -mt-0.5" /> Reject
                    </button>
                    <NuxtLink
                        :to="moderationTargetHref(item)"
                        class="ml-auto px-3 py-1 rounded-md text-xs text-blue-400 hover:bg-blue-500/[0.08] cursor-pointer transition-colors flex items-center gap-1"
                    >
                        View in context
                        <Icon name="lucide:external-link" class="text-xs" />
                    </NuxtLink>
                </div>
                <!-- Reviewed info for non-pending rows -->
                <div v-else class="flex items-center gap-2 mt-3 pl-12 text-xs text-gray-500">
                    <span v-if="item.reviewed_at">
                        Reviewed {{ fmtDate(item.reviewed_at) }}
                        <span v-if="item.reviewed_by">by #{{ item.reviewed_by }}</span>
                    </span>
                    <NuxtLink
                        :to="moderationTargetHref(item)"
                        class="ml-auto px-3 py-1 rounded-md text-xs text-blue-400 hover:bg-blue-500/[0.08] cursor-pointer transition-colors flex items-center gap-1"
                    >
                        View in context
                        <Icon name="lucide:external-link" class="text-xs" />
                    </NuxtLink>
                </div>
            </article>

            <div v-if="!moderationData?.items?.length" class="text-sm text-gray-500 italic py-8 text-center">
                Nothing in the queue.
            </div>
        </div>
    </div>
</template>
