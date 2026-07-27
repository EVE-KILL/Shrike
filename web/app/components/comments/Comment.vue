<script setup lang="ts">
import type { CommentRow } from '~/composables/useCommentThread'

const props = defineProps<{
    comment: CommentRow
    canModerate?: boolean
    /** Render as a nested reply (slimmer header, no avatar size bump). */
    nested?: boolean
}>()

const emit = defineEmits<{
    (e: 'reply', id: number): void
    (e: 'edit', id: number): void
    (e: 'delete', id: number): void
    (e: 'report', id: number): void
}>()

const { user, isAuthenticated } = useAuth()

const isAuthor = computed(() => user.value?.characterId === props.comment.character_id)
const isHidden = computed(() => props.comment.moderation_status >= 2 || props.comment.deleted_at != null)
const canEdit = computed(() => {
    if (!isAuthor.value) return false
    const created = new Date(props.comment.created_at).getTime()
    return Date.now() - created < 24 * 60 * 60 * 1000
})
const canDelete = computed(() => isAuthor.value || props.canModerate)
const canReport = computed(() => isAuthenticated.value && !isAuthor.value && !isHidden.value)

// EVE time, like every other timestamp on the site. This was
// Intl.DateTimeFormat(undefined, …), which takes the runtime's locale and
// timezone: it rendered local time, and rendered it differently on the server
// than in the browser — a hydration mismatch as well as the wrong clock.
const formattedDate = computed(() => formatEveDateTime(props.comment.created_at))
</script>

<template>
    <section
        :id="`comment-${comment.id}`"
        class="comment-section"
        :class="{ 'comment-hidden': isHidden, 'comment-nested': nested }"
    >
        <!-- Header -->
        <header class="flex items-center pb-2 mb-3 border-b border-white/[0.06]">
            <NuxtLink :to="`/character/${comment.character_id}`" class="flex-shrink-0">
                <img
                    :src="`/images/characters/${comment.character_id}/portrait?size=64`"
                    :alt="comment.character_name"
                    class="rounded-md ring-1 ring-white/[0.08]"
                    :class="nested ? 'w-8 h-8 mr-2.5' : 'w-10 h-10 mr-3'"
                    loading="lazy"
                >
            </NuxtLink>

            <div class="flex-1 min-w-0">
                <NuxtLink
                    :to="`/character/${comment.character_id}`"
                    class="block text-sm font-semibold text-white hover:text-blue-400 truncate transition-colors"
                >
                    {{ comment.character_name }}
                </NuxtLink>
                <div class="text-xs text-gray-500 truncate">
                    <NuxtLink
                        :to="`/corporation/${comment.corporation_id}`"
                        class="hover:text-blue-400 hover:underline"
                    >
                        {{ comment.corporation_name }}
                    </NuxtLink>
                    <template v-if="comment.alliance_name">
                        <span class="text-gray-600"> / </span>
                        <NuxtLink
                            :to="`/alliance/${comment.alliance_id}`"
                            class="hover:text-blue-400 hover:underline"
                        >
                            {{ comment.alliance_name }}
                        </NuxtLink>
                    </template>
                </div>
            </div>

            <div class="ml-2 flex items-center gap-1 flex-shrink-0">
                <Tooltip :text="formatEveDateTime(comment.created_at, true)" class="mr-1 hidden sm:inline">
                    <span class="text-xs text-gray-600">{{ formattedDate }}</span>
                </Tooltip>
                <span v-if="comment.edited_at" class="text-xs text-gray-700 italic mr-1" v-tooltip="'Edited'">
                    (edited)
                </span>

                <button
                    v-if="canReport"
                    class="header-action-btn"
                    v-tooltip="'Report comment'"
                    @click="emit('report', comment.id)"
                >
                    <Icon name="lucide:flag" class="text-xs" />
                </button>
                <button
                    v-if="canEdit"
                    class="header-action-btn"
                    v-tooltip="'Edit'"
                    @click="emit('edit', comment.id)"
                >
                    <Icon name="lucide:pencil" class="text-xs" />
                </button>
                <button
                    v-if="canDelete && !isHidden"
                    class="header-action-btn hover:text-red-400 hover:bg-red-500/[0.08]"
                    v-tooltip="'Delete'"
                    @click="emit('delete', comment.id)"
                >
                    <Icon name="lucide:x" class="text-xs" />
                </button>
            </div>
        </header>

        <!-- Body -->
        <div
            v-if="!isHidden"
            class="comment-body text-sm text-gray-200 prose prose-invert max-w-none prose-sm"
            v-html="comment.body_html"
        />
        <div v-else class="text-sm text-gray-500 italic">
            [comment removed]
        </div>

        <!-- Reply action footer -->
        <footer
            v-if="!isHidden && isAuthenticated"
            class="mt-3 pt-2 border-t border-white/[0.04] flex items-center"
        >
            <button
                class="reply-btn"
                @click="emit('reply', comment.id)"
            >
                <Icon name="lucide:reply" class="text-xs" />
                Reply
            </button>
        </footer>
    </section>
</template>

<style scoped>
.comment-section {
    padding: 0.85rem;
    border-radius: 0.5rem;
    border: 1px solid rgba(255, 255, 255, 0.06);
    background: rgba(26, 26, 26, 0.3);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
    transition: border-color 0.2s, background-color 0.2s;
}
.comment-section:hover {
    border-color: rgba(255, 255, 255, 0.1);
}
.comment-section.comment-nested {
    padding: 0.65rem 0.75rem;
    background: rgba(26, 26, 26, 0.2);
}
.comment-hidden {
    opacity: 0.5;
}

.header-action-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 0.25rem;
    color: rgb(107, 114, 128);
    cursor: pointer;
    transition: color 0.15s, background-color 0.15s;
}
.header-action-btn:hover {
    color: var(--color-brand-primary);
    background-color: rgba(255, 255, 255, 0.06);
}

.reply-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.35rem 0.75rem;
    border-radius: 0.375rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--color-brand-primary);
    background-color: color-mix(in srgb, var(--color-brand-primary) 8%, transparent);
    border: 1px solid color-mix(in srgb, var(--color-brand-primary) 18%, transparent);
    cursor: pointer;
    transition: background-color 0.15s, border-color 0.15s, color 0.15s;
}
.reply-btn:hover {
    background-color: color-mix(in srgb, var(--color-brand-primary) 18%, transparent);
    border-color: color-mix(in srgb, var(--color-brand-primary) 35%, transparent);
    color: white;
}

.comment-body :deep(p) {
    margin: 0.3rem 0;
}
.comment-body :deep(p:first-child) {
    margin-top: 0;
}
.comment-body :deep(p:last-child) {
    margin-bottom: 0;
}
.comment-body :deep(a) {
    color: var(--color-brand-primary);
    text-decoration: underline;
}
.comment-body :deep(.embed) {
    margin: 0.5rem 0;
    border-radius: 0.5rem;
    overflow: hidden;
}
.comment-body :deep(.embed-youtube) {
    position: relative;
    padding-bottom: 56.25%;
    height: 0;
}
.comment-body :deep(.embed-youtube iframe) {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: 0;
}
.comment-body :deep(.embed-image img),
.comment-body :deep(.embed-gif img) {
    max-width: 100%;
    max-height: 400px;
    border-radius: 0.5rem;
}
.comment-body :deep(.embed-video video) {
    max-width: 100%;
    max-height: 400px;
    border-radius: 0.5rem;
}
.comment-body :deep(.embed-media) {
    margin: 0.5rem 0;
}
.comment-body :deep(.embed-media video),
.comment-body :deep(.embed-media img) {
    max-width: 100%;
    max-height: 600px;
    border-radius: 0.5rem;
    display: block;
    background: #000;
}
.comment-body :deep(pre) {
    background: rgba(0, 0, 0, 0.4);
    padding: 0.5rem;
    border-radius: 0.25rem;
    overflow-x: auto;
    font-size: 0.75rem;
}
.comment-body :deep(code) {
    background: rgba(255, 255, 255, 0.08);
    padding: 0.1rem 0.3rem;
    border-radius: 0.2rem;
    font-size: 0.85em;
}
.comment-body :deep(blockquote) {
    border-left: 2px solid rgba(255, 255, 255, 0.15);
    padding-left: 0.75rem;
    color: rgb(156, 163, 175);
    margin: 0.5rem 0;
}

/* Highlight pulse when scrolling to a comment via fragment */
.comment-section:target {
    animation: highlight-pulse 2s ease-in-out;
}
@keyframes highlight-pulse {
    0% { background-color: rgba(26, 26, 26, 0.3); }
    30% { background-color: color-mix(in srgb, var(--color-brand-primary) 15%, transparent); }
    100% { background-color: rgba(26, 26, 26, 0.3); }
}
</style>
