<script setup lang="ts">
import { useCommentThread, type CommentRow } from '~/composables/useCommentThread'

const props = defineProps<{
    targetType: number
    targetId: number
    /**
     * Slug identifier for targets whose key is text rather than numeric
     * (fits, pages). When set, thread queries and WS events are scoped by
     * slug as well as type/id. Leave unset for entity targets.
     */
    targetSlug?: string | null
    /** Show the global header + count? */
    showHeader?: boolean
}>()

// No SSR-embedded initial fetch. A previous attempt embedded the thread
// data during SSR (via useAsyncData + watchEffect inside useCommentThread)
// to save the client-side round trip on direct page loads, but it created
// per-request Vue reactivity state that wasn't getting disposed under SSR
// bot traffic — retained EffectScopes / ComputedRefImpls / Deps piled into
// the millions within minutes, pushing pod RSS past 6 GB. Reverted to the
// client-only load pattern: onMounted calls thread.reload() which hits
// /api/comments/thread, and that endpoint is SWR-cached at 30s via
// cachedApiHandler, so the trip is ~1ms for cache hits. Net cost for
// losing the SSR-embed optimization is essentially zero.
const thread = useCommentThread(
    props.targetType,
    () => props.targetId,
    () => props.targetSlug ?? null,
)
const { user, isAuthenticated, isAdmin, login } = useAuth()

// Client-only initial load + scroll-to-fragment. Runs once on mount for the
// current target; the watch below handles SPA navigation to different
// targets that reuse this component instance (e.g. tabbed entity pages).
onMounted(() => {
    thread.reload().then(() => {
        if (window.location.hash.startsWith('#comment-')) {
            nextTick(() => {
                const el = document.getElementById(window.location.hash.slice(1))
                if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
            })
        }
    })
})

watch(() => [props.targetId, props.targetSlug], () => { thread.reload() })

// Reply / edit state
const replyingTo = ref<number | null>(null)
const editingId = ref<number | null>(null)

// Report modal
const reportingId = ref<number | null>(null)
const reportOpen = computed({
    get: () => reportingId.value !== null,
    set: (v) => { if (!v) reportingId.value = null },
})

// Collapsed thread state. Holds IDs of comments whose children are hidden.
// Defaults to collapsing chains at depth >= COLLAPSE_DEPTH so very deep nests
// don't blow out the layout (Reddit-style "continue this thread").
const COLLAPSE_DEPTH = 4
const collapsedReplies = ref<Set<number>>(new Set())
function toggleCollapse(id: number) {
    const next = new Set(collapsedReplies.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    collapsedReplies.value = next
}

// Auto-collapse anything past COLLAPSE_DEPTH on first load / when items change.
// We only ADD ids — never remove — so user expand/collapse choices persist.
watch(() => thread.totalCount.value, () => {
    const next = new Set(collapsedReplies.value)
    for (const c of thread.items.value.values()) {
        if (c.depth >= COLLAPSE_DEPTH && !next.has(c.id)) {
            next.add(c.id)
        }
    }
    if (next.size !== collapsedReplies.value.size) collapsedReplies.value = next
})

async function onPostRoot(body: string) {
    await thread.postComment(body, null)
}

async function onPostReply(parentId: number, body: string) {
    await thread.postComment(body, parentId)
    replyingTo.value = null
}

async function onEdit(id: number, body: string) {
    await thread.editComment(id, body)
    editingId.value = null
}

async function onDelete(id: number) {
    if (!confirm('Delete this comment?')) return
    await thread.deleteComment(id)
}

async function onReport(reason: string, message: string | null) {
    if (reportingId.value == null) return
    await thread.reportComment(reportingId.value, reason, message)
}
</script>

<template>
    <section class="comment-list-wrap">
        <header v-if="showHeader !== false" class="flex items-center justify-between mb-3 pb-2.5 border-b border-white/[0.06]">
            <h2 class="text-base font-semibold text-white flex items-center gap-2">
                <Icon name="lucide:message-square" class="text-blue-400" />
                Comments
                <span
                    v-if="thread.totalCount.value > 0"
                    class="ml-1 inline-flex items-center justify-center min-w-[1.25rem] h-5 px-1.5 rounded-full text-fine font-semibold text-blue-300 bg-blue-500/[0.12] border border-blue-500/20"
                >
                    {{ thread.totalCount.value }}
                </span>
            </h2>
            <ClientOnly>
                <div
                    v-if="thread.connected.value"
                    class="flex items-center gap-1.5 text-xs text-gray-500"
                    v-tooltip="'Live updates connected'"
                >
                    <span class="relative flex w-1.5 h-1.5">
                        <span class="absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-60 animate-ping" />
                        <span class="relative inline-flex w-1.5 h-1.5 rounded-full bg-green-400" />
                    </span>
                    <span>Live</span>
                </div>
                <div
                    v-else
                    class="flex items-center gap-1.5 text-xs text-amber-400"
                >
                    <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse" />
                    <span>Reconnecting…</span>
                </div>
            </ClientOnly>
        </header>

        <!-- Loading / empty / error -->
        <div v-if="thread.loading.value && thread.totalCount.value === 0" class="text-center py-8 text-sm text-gray-500 italic">
            <Icon name="lucide:loader-2" class="animate-spin inline-block mr-2" />
            Loading comments…
        </div>
        <div v-else-if="thread.error.value" class="comment-section text-sm text-red-400 text-center py-6">
            {{ thread.error.value }}
        </div>
        <div v-else-if="thread.totalCount.value === 0" class="comment-section text-sm text-gray-500 italic text-center py-6">
            No comments yet. Be the first.
        </div>

        <!-- Recursive thread -->
        <div v-for="root in thread.roots.value" :key="root.id" class="comment-thread mb-3">
            <CommentsCommentNode
                :node="root"
                :children-by-parent="thread.childrenByParent.value"
                :collapsed-replies="collapsedReplies"
                :replying-to="replyingTo"
                :editing-id="editingId"
                :can-moderate="isAdmin"
                @reply="(id) => replyingTo = id"
                @edit="(id) => editingId = id"
                @delete="onDelete"
                @report="(id) => reportingId = id"
                @cancel-reply="replyingTo = null"
                @cancel-edit="editingId = null"
                @submit-reply="onPostReply"
                @submit-edit="onEdit"
                @toggle-collapse="toggleCollapse"
            />
        </div>

        <!-- Load more -->
        <button
            v-if="thread.nextCursor.value"
            class="w-full text-xs text-blue-400 hover:text-blue-300 py-2 cursor-pointer mb-3"
            :disabled="thread.loading.value"
            @click="thread.loadMore()"
        >
            {{ thread.loading.value ? 'Loading…' : 'Load more comments' }}
        </button>

        <!-- ═══════════════ COMPOSER SECTION ═══════════════ -->
        <ClientOnly>
            <div v-if="isAuthenticated && user" class="mt-4">
                <!-- Identity strip -->
                <div class="flex items-center gap-2 mb-2 px-1 text-xs text-gray-500">
                    <NuxtLink :to="`/character/${user.characterId}`" class="flex-shrink-0">
                        <img
                            :src="`/images/characters/${user.characterId}/portrait?size=64`"
                            :alt="user.characterName"
                            class="w-6 h-6 rounded ring-1 ring-white/[0.08]"
                            loading="lazy"
                        >
                    </NuxtLink>
                    <span>Posting as</span>
                    <NuxtLink
                        :to="`/character/${user.characterId}`"
                        class="font-semibold text-white hover:text-blue-400 truncate transition-colors"
                    >
                        {{ user.characterName }}
                    </NuxtLink>
                    <span v-if="user.corporationName" class="hidden sm:inline truncate">
                        <span class="mx-1 text-gray-700">·</span>
                        <NuxtLink
                            v-if="user.corporationId"
                            :to="`/corporation/${user.corporationId}`"
                            class="text-gray-500 hover:text-blue-400"
                        >
                            {{ user.corporationName }}
                        </NuxtLink>
                    </span>
                </div>

                <CommentsCommentInput
                    :on-submit="onPostRoot"
                    placeholder="Share a thought… Markdown supported."
                    submit-label="Post comment"
                />
            </div>

            <div v-else class="composer-signin mt-4">
                <Icon name="lucide:message-square-plus" class="text-3xl text-blue-400/70 mb-2" />
                <p class="text-sm text-gray-300 font-medium">Join the conversation</p>
                <p class="text-xs text-gray-500 mt-1 mb-4">Sign in with EVE Online to leave a comment.</p>
                <button
                    class="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-blue-500/[0.15] text-blue-300 border border-blue-500/30 hover:bg-blue-500/[0.25] cursor-pointer transition-colors text-sm font-medium"
                    @click="login()"
                >
                    <Icon name="lucide:log-in" class="text-sm" />
                    Sign in with EVE Online
                </button>
            </div>

            <template #fallback>
                <div class="comment-section text-center py-6 text-sm text-gray-500 mt-4">
                    <Icon name="lucide:loader-2" class="animate-spin inline-block mr-2" />
                    Loading…
                </div>
            </template>
        </ClientOnly>

        <CommentsCommentReportModal
            v-model="reportOpen"
            :on-submit="onReport"
        />
    </section>
</template>

<style scoped>
.comment-section {
    padding: 0.85rem;
    border-radius: 0.5rem;
    border: 1px solid rgba(255, 255, 255, 0.06);
    background: rgba(26, 26, 26, 0.3);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.composer-signin {
    text-align: center;
    padding: 2rem 1rem;
    border-radius: 0.75rem;
    border: 1px dashed rgba(255, 255, 255, 0.1);
    background: linear-gradient(180deg, rgba(96, 165, 250, 0.04), rgba(255, 255, 255, 0.01));
}
</style>
