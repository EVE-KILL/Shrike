<script setup lang="ts">
// Comments moderation state
interface AdminComment {
    id: number
    target_type: number
    target_id: number
    target_slug: string | null
    parent_id: number | null
    body_md: string
    body_html: string
    character_id: number
    character_name: string
    corporation_name: string
    created_at: string
    deleted_at: string | null
    reports_count: number
    flagged: boolean
    moderation_status: number
}
const commentFilter = ref<'flagged' | 'reported' | 'all'>('flagged')
const { data: commentsData, refresh: refreshComments } = useApiFetch<{ comments: AdminComment[] }>('/api/admin/comments/queue', {
    query: { filter: commentFilter, limit: 100 },
    immediate: true,
    lazy: true,
    watch: [commentFilter],
})

const toast = useToast()

const actingId = ref<number | null>(null)

/**
 * Delete asks twice: the first click arms the button, the second goes through.
 * Matches the pattern already used in Campaigns and Sessions, and replaces a
 * window.confirm() — which cannot be styled and blocks the whole page.
 */
const { pendingId: confirmDeleteId, confirm: confirmDelete } = useConfirmTwice()

const run = async (id: number, label: string, fn: () => Promise<unknown>) => {
    actingId.value = id
    try {
        await fn()
        await refreshComments()
    } catch (err: any) {
        toast.error(extractFetchError(err, `Failed to ${label}`))
    } finally {
        actingId.value = null
    }
}

const adminHideComment = (id: number) =>
    run(id, 'hide', () => apiFetch(`/api/admin/comments/${id}/hide`, { method: 'POST' }))

const adminRestoreComment = (id: number) =>
    run(id, 'restore', () => apiFetch(`/api/admin/comments/${id}/restore`, { method: 'POST' }))

const adminDeleteComment = async (id: number) => {
    if (!confirmDelete(id)) return
    await run(id, 'delete', () => apiFetch(`/api/comments/${id}`, { method: 'DELETE' }))
}


function commentTargetLabel(c: AdminComment): { label: string; href: string } {
    switch (c.target_type) {
        case 1: return { label: `Kill #${c.target_id}`, href: `/kill/${c.target_id}#comment-${c.id}` }
        case 2: return { label: `Char ${c.target_id}`, href: `/character/${c.target_id}` }
        case 3: return { label: `Corp ${c.target_id}`, href: `/corporation/${c.target_id}` }
        case 4: return { label: `Alli ${c.target_id}`, href: `/alliance/${c.target_id}` }
        case 5: return { label: `Sys ${c.target_id}`, href: `/system/${c.target_id}` }
        case 7: return { label: `Battle ${c.target_id}`, href: `/battle/${c.target_id}/comments#comment-${c.id}` }
        case 8: return { label: `Fit ${c.target_slug ?? '?'}`, href: c.target_slug ? `/fit/${c.target_slug}#comment-${c.id}` : '#' }
        default: return { label: 'Page', href: '#' }
    }
}

function moderationLabel(s: number): { label: string; cls: string } {
    switch (s) {
        case 0: return { label: 'OK', cls: 'text-gray-400' }
        case 1: return { label: 'Pending', cls: 'text-yellow-400' }
        case 2: return { label: 'AI flagged', cls: 'text-yellow-400' }
        case 3: return { label: 'Hidden', cls: 'text-red-400' }
        default: return { label: '?', cls: 'text-gray-500' }
    }
}

// The parent keeps this component alive (<KeepAlive>) across tab switches.
// Matching the old tab-switch watcher: on re-activation, refetch only if no
// data ever loaded (e.g. the first fetch errored).
let activatedOnce = false
onActivated(() => {
    if (!activatedOnce) { activatedOnce = true; return }
    if (!commentsData.value) refreshComments()
})
</script>

<template>
    <!-- ═══════════════ COMMENTS MODERATION ═══════════════ -->
    <div class="space-y-4">
        <!-- Filter pills -->
        <div class="flex items-center gap-2">
            <button
                v-for="f in ['flagged', 'reported', 'all'] as const"
                :key="f"
                class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors cursor-pointer"
                :class="commentFilter === f ? 'bg-blue-500/[0.15] text-blue-300 border border-blue-500/30' : 'bg-white/[0.04] text-gray-400 hover:bg-white/[0.08] border border-transparent'"
                @click="commentFilter = f"
            >
                {{ f === 'flagged' ? 'Flagged' : f === 'reported' ? 'Reported' : 'All active' }}
            </button>
            <button
                class="ml-auto px-3 py-1.5 rounded-md text-xs text-gray-400 hover:text-white hover:bg-white/[0.06] transition-colors cursor-pointer flex items-center gap-1"
                @click="refreshComments()"
            >
                <Icon name="lucide:refresh-cw" class="text-xs" />
                Refresh
            </button>
        </div>

        <!-- Comments list -->
        <div class="space-y-2">
            <article
                v-for="c in (commentsData?.comments || [])"
                :key="c.id"
                class="glass-panel p-4"
                :class="{ 'opacity-60': c.deleted_at || c.moderation_status >= 2 }"
            >
                <header class="flex items-start gap-3 mb-2">
                    <NuxtLink :to="`/character/${c.character_id}`" class="flex-shrink-0">
                        <img
                            :src="`/images/characters/${c.character_id}/portrait?size=64`"
                            :alt="c.character_name"
                            class="w-9 h-9 rounded-md"
                            loading="lazy"
                        >
                    </NuxtLink>
                    <div class="flex-1 min-w-0">
                        <div class="flex items-baseline gap-2 text-sm flex-wrap">
                            <NuxtLink :to="`/character/${c.character_id}`" class="font-semibold text-white hover:text-blue-400 truncate">
                                {{ c.character_name }}
                            </NuxtLink>
                            <span class="text-xs text-gray-600 truncate">{{ c.corporation_name }}</span>
                            <span class="text-xs text-gray-600">·</span>
                            <NuxtLink :to="commentTargetLabel(c).href" class="text-xs text-blue-400 hover:underline truncate">
                                {{ commentTargetLabel(c).label }}
                            </NuxtLink>
                            <span class="text-xs text-gray-600 ml-auto whitespace-nowrap">
                                {{ fmtDate(c.created_at) }}
                            </span>
                        </div>
                        <div class="flex items-center gap-3 mt-1 text-xs">
                            <span :class="moderationLabel(c.moderation_status).cls">
                                {{ moderationLabel(c.moderation_status).label }}
                            </span>
                            <span v-if="c.flagged" class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-red-500/20 text-red-400 font-medium">
                                <Icon name="lucide:flag" class="text-xs" />
                                Flagged
                            </span>
                            <span v-if="c.reports_count > 0" class="text-yellow-400">
                                {{ c.reports_count }} report{{ c.reports_count === 1 ? '' : 's' }}
                            </span>
                            <span v-if="c.deleted_at" class="text-red-400">deleted</span>
                        </div>
                    </div>
                </header>

                <!-- Body -->
                <div
                    class="text-sm text-gray-200 prose prose-invert max-w-none prose-sm pl-12"
                    v-html="c.body_html"
                />

                <!-- Actions -->
                <div class="flex items-center gap-2 mt-3 pl-12">
                    <button
                        v-if="c.moderation_status < 3 && !c.deleted_at"
                        class="px-3 py-1 rounded-md text-xs bg-yellow-500/[0.15] text-yellow-400 hover:bg-yellow-500/[0.25] cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        :disabled="actingId === c.id"
                        @click="adminHideComment(c.id)"
                    >
                        Hide
                    </button>
                    <button
                        v-if="c.moderation_status >= 2 || c.deleted_at"
                        class="px-3 py-1 rounded-md text-xs bg-green-500/[0.15] text-green-300 hover:bg-green-500/[0.25] cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        :disabled="actingId === c.id"
                        @click="adminRestoreComment(c.id)"
                    >
                        Restore
                    </button>
                    <button
                        v-if="!c.deleted_at"
                        class="px-3 py-1 rounded-md text-xs bg-red-500/[0.15] hover:bg-red-500/[0.25] cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        :class="confirmDeleteId === c.id ? 'text-red-200 ring-1 ring-red-500/40' : 'text-red-300'"
                        :disabled="actingId === c.id"
                        @click="adminDeleteComment(c.id)"
                    >
                        {{ confirmDeleteId === c.id ? 'Confirm delete' : 'Delete' }}
                    </button>
                    <NuxtLink
                        :to="commentTargetLabel(c).href"
                        class="ml-auto px-3 py-1 rounded-md text-xs text-blue-400 hover:bg-blue-500/[0.08] cursor-pointer transition-colors flex items-center gap-1"
                    >
                        View in context
                        <Icon name="lucide:external-link" class="text-xs" />
                    </NuxtLink>
                </div>
            </article>

            <div v-if="!commentsData?.comments?.length" class="text-sm text-gray-500 italic py-8 text-center">
                No comments to moderate.
            </div>
        </div>
    </div>
</template>
