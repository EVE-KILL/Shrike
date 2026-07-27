<script setup lang="ts">
import type { CommentRow } from '~/composables/useCommentThread'

const props = defineProps<{
    node: CommentRow
    childrenByParent: Map<number, CommentRow[]>
    collapsedReplies: Set<number>
    replyingTo: number | null
    editingId: number | null
    canModerate: boolean
}>()

const emit = defineEmits<{
    (e: 'reply', id: number): void
    (e: 'edit', id: number): void
    (e: 'delete', id: number): void
    (e: 'report', id: number): void
    (e: 'cancel-reply'): void
    (e: 'cancel-edit'): void
    (e: 'submit-reply', parentId: number, body: string): Promise<void> | void
    (e: 'submit-edit', id: number, body: string): Promise<void> | void
    (e: 'toggle-collapse', id: number): void
}>()

const children = computed(() => props.childrenByParent.get(props.node.id) ?? [])
const hasChildren = computed(() => children.value.length > 0)
const isCollapsed = computed(() => props.collapsedReplies.has(props.node.id))

const isReplying = computed(() => props.replyingTo === props.node.id)
const isEditing = computed(() => props.editingId === props.node.id)

const descendantCount = computed(() => {
    let count = 0
    const stack = [...children.value]
    while (stack.length > 0) {
        const c = stack.pop()!
        count++
        const sub = props.childrenByParent.get(c.id)
        if (sub) stack.push(...sub)
    }
    return count
})

async function submitReply(body: string) {
    await emit('submit-reply', props.node.id, body)
}
async function submitEdit(body: string) {
    await emit('submit-edit', props.node.id, body)
}
</script>

<template>
    <div class="comment-node" :class="{ 'has-line': hasChildren && !isCollapsed }">
        <!-- Continuous thread line — runs from the top of THIS comment all the
             way down past the last descendant. Click to collapse the subtree. -->
        <button
            v-if="hasChildren && !isCollapsed"
            class="thread-line"
            v-tooltip="`Collapse ${descendantCount} ${descendantCount === 1 ? 'reply' : 'replies'}`"
            :aria-label="`Collapse ${descendantCount} replies`"
            @click="emit('toggle-collapse', node.id)"
        />

        <CommentsComment
            v-if="!isEditing"
            :comment="node"
            :can-moderate="canModerate"
            :nested="false"
            @reply="emit('reply', $event)"
            @edit="emit('edit', $event)"
            @delete="emit('delete', $event)"
            @report="emit('report', $event)"
        />

        <!-- Inline edit form -->
        <div v-else class="my-2">
            <CommentsCommentInput
                :initial="node.body_md"
                :on-submit="submitEdit"
                :on-cancel="() => emit('cancel-edit')"
                submit-label="Save"
            />
        </div>

        <!-- Inline reply form -->
        <div v-if="isReplying" class="mt-2">
            <CommentsCommentInput
                :on-submit="submitReply"
                :on-cancel="() => emit('cancel-reply')"
                submit-label="Reply"
                placeholder="Write a reply…"
            />
        </div>

        <!-- Collapsed indicator -->
        <button
            v-if="hasChildren && isCollapsed"
            class="thread-expand-btn"
            @click="emit('toggle-collapse', node.id)"
        >
            <Icon name="lucide:plus-square" class="text-xs" />
            Show {{ descendantCount }} {{ descendantCount === 1 ? 'reply' : 'replies' }}
        </button>

        <!-- Children container -->
        <div v-if="hasChildren && !isCollapsed" class="children-content">
            <CommentsCommentNode
                v-for="child in children"
                :key="child.id"
                :node="child"
                :children-by-parent="childrenByParent"
                :collapsed-replies="collapsedReplies"
                :replying-to="replyingTo"
                :editing-id="editingId"
                :can-moderate="canModerate"
                @reply="emit('reply', $event)"
                @edit="emit('edit', $event)"
                @delete="emit('delete', $event)"
                @report="emit('report', $event)"
                @cancel-reply="emit('cancel-reply')"
                @cancel-edit="emit('cancel-edit')"
                @submit-reply="(pid, body) => emit('submit-reply', pid, body)"
                @submit-edit="(id, body) => emit('submit-edit', id, body)"
                @toggle-collapse="emit('toggle-collapse', $event)"
            />
        </div>
    </div>
</template>

<style scoped>
/*
 * Reddit-style threaded layout — simple version.
 *
 *   - Each comment-node sits inside its parent's children-content with a fixed
 *     left margin, so every reply level is uniformly indented.
 *   - When a comment has visible children, its .comment-node gets a left
 *     padding gutter (.has-line) and a thread-line is drawn in that gutter.
 *     The line spans from the top of the parent comment all the way down to
 *     the bottom of the last descendant, since the parent's .comment-node
 *     wraps everything below it.
 *   - Click anywhere on the line to collapse the subtree.
 */
.comment-node {
    position: relative;
}

/* Visual gutter for the thread line. Pushes the parent comment + children to
   the right so the line lives in the empty space on the left. */
.comment-node.has-line {
    padding-left: 18px;
}

/* The clickable thread line — wide hit target, thin visible bar centered. */
.thread-line {
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 18px;
    display: flex;
    justify-content: center;
    align-items: stretch;
    background: transparent;
    border: none;
    padding: 0;
    cursor: pointer;
    z-index: 1;
}
.thread-line::before {
    content: '';
    width: 2px;
    background-color: rgba(255, 255, 255, 0.12);
    border-radius: 1px;
    transition: background-color 0.15s, width 0.15s;
}
.thread-line:hover::before {
    background-color: rgba(96, 165, 250, 0.7);
    width: 3px;
}
.thread-line:active::before {
    background-color: rgba(96, 165, 250, 0.95);
}

/* Children container. Margin-left guarantees every child is indented from the
   parent comment, regardless of whether the parent has its own thread line. */
.children-content {
    margin-top: 0.6rem;
    margin-left: 14px;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
}

/* "Show X replies" expander when collapsed. */
.thread-expand-btn {
    margin-top: 0.5rem;
    margin-left: 14px;
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.25rem 0.6rem;
    border-radius: 0.25rem;
    font-size: 0.75rem;
    color: rgb(96, 165, 250);
    background-color: rgba(96, 165, 250, 0.08);
    border: 1px solid rgba(96, 165, 250, 0.15);
    cursor: pointer;
    transition: background-color 0.15s, color 0.15s;
}
.thread-expand-btn:hover {
    background-color: rgba(96, 165, 250, 0.18);
    color: rgb(147, 197, 253);
}
</style>
