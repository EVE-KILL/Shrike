<script setup lang="ts">
// Settings → Comments section (My Comments tab). Extracted verbatim from
// pages/settings/[[tab]].vue.

// ── Comments (My Comments tab) ──────────────────────────────────────────────
interface MyComment {
    id: number
    target_type: number
    target_id: number
    target_slug: string | null
    domain_id: number | null
    parent_id: number | null
    root_id: number | null
    depth: number
    body_md: string
    body_html: string
    character_id: number
    character_name: string
    corporation_id: number
    corporation_name: string
    alliance_id: number | null
    alliance_name: string | null
    created_at: string
    updated_at: string
    edited_at: string | null
    moderation_status: number
    reply_count: number
}

const myComments = ref<MyComment[]>([])
const myCommentsCursor = ref<number | null>(null)
const myCommentsHasMore = ref(false)
const myCommentsLoading = ref(false)
const myCommentsError = ref<string | null>(null)
const myCommentsLoaded = ref(false)
const deletingCommentId = ref<number | null>(null)

async function fetchMyComments(reset: boolean) {
    if (myCommentsLoading.value) return
    myCommentsLoading.value = true
    myCommentsError.value = null
    try {
        const params = new URLSearchParams({ limit: '25' })
        if (!reset && myCommentsCursor.value) params.set('cursor', String(myCommentsCursor.value))
        const data = await apiFetch<{ comments: MyComment[]; nextCursor: number | null }>(
            `/api/user/comments?${params.toString()}`,
        )
        if (reset) {
            myComments.value = data.comments
        } else {
            const seen = new Set(myComments.value.map((c) => c.id))
            for (const c of data.comments) if (!seen.has(c.id)) myComments.value.push(c)
        }
        myCommentsCursor.value = data.nextCursor
        myCommentsHasMore.value = data.nextCursor !== null
        myCommentsLoaded.value = true
    } catch (err: any) {
        myCommentsError.value = err?.statusMessage || err?.message || 'Failed to load comments'
    } finally {
        myCommentsLoading.value = false
    }
}

// Deleting is permanent, so the button arms on the first click and fires on the
// second. Replaces a window.confirm(), which cannot be styled and blocks the page.
const { pendingId: confirmDeleteId, confirm: confirmDelete } = useConfirmTwice()

async function deleteMyComment(id: number) {
    if (!confirmDelete(id)) return
    deletingCommentId.value = id
    try {
        await apiFetch(`/api/user/comments/${id}`, { method: 'DELETE' })
        myComments.value = myComments.value.filter((c) => c.id !== id)
    } catch (err: any) {
        myCommentsError.value = err?.statusMessage || err?.message || 'Failed to delete comment'
    } finally {
        deletingCommentId.value = null
    }
}


interface TargetMeta {
    label: string
    href: string
    icon: string
    chipBg: string
    chipText: string
}
function targetMeta(c: Pick<MyComment, 'target_type' | 'target_id' | 'target_slug'>): TargetMeta {
    switch (c.target_type) {
        case 1: return { label: `Killmail #${c.target_id}`, href: `/kill/${c.target_id}#comment-${c.target_id}`, icon: 'lucide:swords',   chipBg: 'bg-red-500/[0.08]',     chipText: 'text-red-300' }
        case 2: return { label: 'Character',                href: `/character/${c.target_id}`,                  icon: 'lucide:user',     chipBg: 'bg-blue-500/[0.08]',    chipText: 'text-blue-300' }
        case 3: return { label: 'Corporation',              href: `/corporation/${c.target_id}`,                icon: 'lucide:building', chipBg: 'bg-yellow-500/[0.08]',  chipText: 'text-yellow-400' }
        case 4: return { label: 'Alliance',                 href: `/alliance/${c.target_id}`,                   icon: 'lucide:flag',     chipBg: 'bg-blue-500/[0.14]',    chipText: 'text-blue-200' }
        case 5: return { label: 'Solar system',             href: `/system/${c.target_id}`,                     icon: 'lucide:globe',    chipBg: 'bg-emerald-500/[0.08]', chipText: 'text-emerald-300' }
        case 6: return { label: c.target_slug ? `Page: ${c.target_slug}` : 'Page', href: c.target_slug ? `/${c.target_slug}` : '#', icon: 'lucide:file-text', chipBg: 'bg-gray-500/[0.08]', chipText: 'text-gray-300' }
        case 7: return { label: `Battle #${c.target_id}`,   href: `/battle/${c.target_id}/comments`,            icon: 'lucide:shield',   chipBg: 'bg-yellow-500/[0.14]',  chipText: 'text-yellow-400' }
        case 8: return { label: c.target_slug ? `Fit ${c.target_slug}` : 'Fit', href: c.target_slug ? `/fit/${c.target_slug}` : '#', icon: 'lucide:wrench', chipBg: 'bg-emerald-500/[0.14]', chipText: 'text-emerald-200' }
        default: return { label: 'Unknown',                 href: '#',                                          icon: 'lucide:help-circle', chipBg: 'bg-gray-500/[0.08]', chipText: 'text-gray-300' }
    }
}

// Initial fetch on first mount — this component only mounts when the tab is
// opened (the parent keeps it alive via <KeepAlive>), so this replaces the
// old activeSection watch + deep-link onMounted.
// Fire after setup; OK to run client-side only since the page is noindex.
onMounted(() => { if (!myCommentsLoaded.value) fetchMyComments(true) })
</script>

<template>
    <div class="space-y-4">
        <div class="glass-panel p-5">
            <div class="flex items-center justify-between gap-3 mb-4">
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">My Comments</h2>
                <button
                    class="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs text-gray-400 hover:text-white hover:bg-white/[0.06] border border-white/[0.06] transition-colors cursor-pointer"
                    :disabled="myCommentsLoading"
                    @click="fetchMyComments(true)"
                >
                    <Icon name="lucide:refresh-cw" class="text-xs" :class="{ 'animate-spin': myCommentsLoading }" />
                    Refresh
                </button>
            </div>

            <p class="text-xs text-gray-600 mb-4">Every comment you've posted across EVE-KILL, newest first. Deleting is permanent — the thread will show a <em>[deleted]</em> placeholder in place of your message.</p>

            <!-- Error banner -->
            <div v-if="myCommentsError" class="mb-3 rounded-md border border-red-500/30 bg-red-500/[0.06] p-2.5 text-xs text-red-300 flex items-center gap-2">
                <Icon name="lucide:alert-circle" class="text-sm flex-shrink-0" />
                {{ myCommentsError }}
            </div>

            <!-- Loading skeleton (initial) -->
            <div v-if="myCommentsLoading && !myCommentsLoaded" class="space-y-3">
                <div v-for="i in 3" :key="i" class="rounded-md border border-white/[0.06] bg-white/[0.02] p-3 animate-pulse">
                    <div class="flex items-center gap-2 mb-2">
                        <div class="h-4 w-20 rounded bg-white/[0.06]" />
                        <div class="h-3 w-16 rounded bg-white/[0.04]" />
                    </div>
                    <div class="space-y-1.5">
                        <div class="h-3 w-full rounded bg-white/[0.06]" />
                        <div class="h-3 w-4/5 rounded bg-white/[0.04]" />
                    </div>
                </div>
            </div>

            <!-- Empty state -->
            <div v-else-if="myCommentsLoaded && !myComments.length" class="rounded-md border border-white/[0.06] bg-white/[0.02] py-10 px-4 text-center">
                <div class="inline-flex items-center justify-center w-12 h-12 rounded-full bg-white/[0.03] border border-white/[0.06] mb-3">
                    <Icon name="lucide:message-circle-off" class="text-xl text-gray-600" />
                </div>
                <p class="text-sm text-gray-500">You haven't posted any comments yet.</p>
            </div>

            <!-- Comment list -->
            <ul v-else class="space-y-2.5">
                <li
                    v-for="c in myComments"
                    :key="c.id"
                    class="rounded-md border border-white/[0.06] bg-white/[0.02] p-3 hover:border-white/[0.1] hover:bg-white/[0.035] transition-colors"
                >
                    <!-- Header: target chip + meta -->
                    <div class="flex items-center gap-2 mb-2 flex-wrap">
                        <NuxtLink
                            :to="targetMeta(c).href"
                            class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-semibold uppercase tracking-wider hover:brightness-125 transition-all"
                            :class="[targetMeta(c).chipBg, targetMeta(c).chipText]"
                        >
                            <Icon :name="targetMeta(c).icon" class="text-xs" />
                            {{ targetMeta(c).label }}
                            <Icon name="lucide:arrow-up-right" class="text-xs opacity-60" />
                        </NuxtLink>
                        <span v-if="c.parent_id" class="inline-flex items-center gap-1 text-fine text-gray-500">
                            <Icon name="lucide:corner-down-right" class="text-xs" />
                            reply
                        </span>
                        <span v-if="c.domain_id" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-fine text-gray-400 bg-white/[0.04] border border-white/[0.06]" v-tooltip="'Posted on a custom domain'">
                            <Icon name="lucide:globe" class="text-xs" />
                            custom domain
                        </span>
                        <span
                            v-if="c.moderation_status !== 0"
                            class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-fine text-yellow-400 bg-yellow-500/[0.08]"
                            v-tooltip="`Moderation status: ${c.moderation_status}`"
                        >
                            <Icon name="lucide:shield-alert" class="text-xs" />
                            {{ c.moderation_status === 1 ? 'pending' : c.moderation_status === 2 ? 'flagged' : 'hidden' }}
                        </span>
                        <div class="flex-1" />
                        <span
                            class="text-fine text-gray-600 whitespace-nowrap"
                            v-tooltip="fmtDate(c.created_at)"
                        >
                            {{ timeSince(c.created_at) }}
                            <template v-if="c.edited_at"> · edited</template>
                        </span>
                    </div>

                    <!-- Body (rendered HTML, clamped) -->
                    <div
                        class="comment-body text-sm text-gray-200 prose prose-invert max-w-none prose-sm line-clamp-4"
                        v-html="c.body_html"
                    />

                    <!-- Footer: reply count + delete -->
                    <div class="mt-2.5 pt-2.5 border-t border-white/[0.05] flex items-center gap-2">
                        <span
                            v-if="c.reply_count > 0"
                            class="inline-flex items-center gap-1 text-xs text-gray-500"
                            v-tooltip="`${c.reply_count} ${c.reply_count === 1 ? 'reply' : 'replies'}`"
                        >
                            <Icon name="lucide:message-square" class="text-xs" />
                            {{ c.reply_count }} {{ c.reply_count === 1 ? 'reply' : 'replies' }}
                        </span>
                        <div class="flex-1" />
                        <button
                            class="flex items-center gap-1 px-2 py-1 rounded text-xs text-gray-500 hover:text-red-400 hover:bg-red-500/[0.08] cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            :disabled="deletingCommentId === c.id"
                            v-tooltip="confirmDeleteId === c.id ? 'Click again to delete permanently' : 'Delete comment'"
                            :class="confirmDeleteId === c.id ? 'text-red-300 bg-red-500/[0.12] ring-1 ring-red-500/40' : ''"
                            @click="deleteMyComment(c.id)"
                        >
                            <Icon
                                :name="deletingCommentId === c.id ? 'lucide:loader-2' : 'lucide:trash-2'"
                                class="text-xs"
                                :class="{ 'animate-spin': deletingCommentId === c.id }"
                            />
                            {{ confirmDeleteId === c.id ? 'Confirm' : 'Delete' }}
                        </button>
                    </div>
                </li>
            </ul>

            <!-- Load more -->
            <div v-if="myComments.length && myCommentsHasMore" class="mt-4 flex justify-center">
                <button
                    class="flex items-center gap-2 px-4 py-2 rounded-md text-xs font-medium text-blue-300 bg-blue-500/[0.1] border border-blue-500/20 hover:bg-blue-500/[0.2] disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer transition-colors"
                    :disabled="myCommentsLoading"
                    @click="fetchMyComments(false)"
                >
                    <Icon
                        :name="myCommentsLoading ? 'lucide:loader-2' : 'lucide:chevron-down'"
                        class="text-sm"
                        :class="{ 'animate-spin': myCommentsLoading }"
                    />
                    {{ myCommentsLoading ? 'Loading…' : 'Load more' }}
                </button>
            </div>
            <div v-else-if="myComments.length && !myCommentsHasMore" class="mt-4 text-center text-xs text-gray-600">
                — End of list —
            </div>
        </div>
    </div>
</template>
