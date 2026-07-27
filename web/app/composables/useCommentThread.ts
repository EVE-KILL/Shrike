/**
 * useCommentThread — fetch and manage a comment thread for a single target.
 *
 * Loads roots cursor-paginated, descendants in one query, and merges live
 * `new` / `edited` / `deleted` events from useCommentStream into local state.
 *
 * Returns a flat list keyed by id plus a derived tree structure (roots + a
 * map from parent_id → children) for recursive rendering.
 */

import { useCommentStream, type CommentEvent } from './useCommentStream'

export interface CommentRow {
    id: number
    target_type: number
    target_id: number
    target_slug: string | null
    parent_id: number | null
    root_id: number | null
    depth: number
    body_html: string
    body_md: string
    character_id: number
    character_name: string
    corporation_id: number
    corporation_name: string
    alliance_id: number | null
    alliance_name: string | null
    created_at: string
    edited_at: string | null
    deleted_at: string | null
    moderation_status: number
    reports_count: number
    flagged: boolean
}

interface ThreadResponse {
    roots: CommentRow[]
    replies: CommentRow[]
    nextCursor: number | null
}

export function useCommentThread(
    targetType: number,
    targetId: number | (() => number),
    targetSlug: string | null | (() => string | null) = null,
    initialData: ThreadResponse | null = null,
) {
    const resolvedId = computed(() => typeof targetId === 'function' ? targetId() : targetId)
    const resolvedSlug = computed(() => typeof targetSlug === 'function' ? targetSlug() : targetSlug)

    const items = ref<Map<number, CommentRow>>(new Map())
    const nextCursor = ref<number | null>(null)
    const loading = ref(false)
    const error = ref<string | null>(null)

    // Seed items from SSR-embedded data synchronously at setup, if provided.
    // The caller (CommentList.vue) owns the useAsyncData call so the reactive
    // machinery lives in the *component's* effect scope — not here in the
    // composable. A previous version of this code did `useAsyncData` +
    // `watchEffect` inline, which leaked Vue reactivity objects under SSR
    // bot traffic: ~35k EffectScope instances retained per pod within 6
    // minutes of warmup, ballooning RSS to 6+ GB. The plain-parameter form
    // adds nothing to the effect scope and is a one-shot synchronous copy.
    if (initialData) {
        for (const r of initialData.roots) items.value.set(r.id, r)
        for (const r of initialData.replies) items.value.set(r.id, r)
        nextCursor.value = initialData.nextCursor
    }

    /** Roots in DESC order (newest first). */
    const roots = computed(() => {
        return Array.from(items.value.values())
            .filter((c) => c.parent_id === null && !c.deleted_at)
            .sort((a, b) => b.id - a.id)
    })

    /** Map of parent_id → children, sorted ASC by created_at. */
    const childrenByParent = computed(() => {
        const m = new Map<number, CommentRow[]>()
        for (const c of items.value.values()) {
            if (c.parent_id == null) continue
            if (!m.has(c.parent_id)) m.set(c.parent_id, [])
            m.get(c.parent_id)!.push(c)
        }
        for (const arr of m.values()) {
            arr.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
        }
        return m
    })

    const totalCount = computed(() => items.value.size)

    async function loadMore() {
        if (loading.value) return
        loading.value = true
        error.value = null
        try {
            const params: Record<string, any> = {
                target_type: targetType,
                target_id: resolvedId.value,
                limit: 50,
            }
            if (resolvedSlug.value) params.target_slug = resolvedSlug.value
            if (nextCursor.value) params.cursor = nextCursor.value

            const data = await apiFetch<ThreadResponse>('/api/comments/thread', { params })
            for (const r of data.roots) items.value.set(r.id, r)
            for (const r of data.replies) items.value.set(r.id, r)
            nextCursor.value = data.nextCursor
        } catch (err: any) {
            error.value = extractFetchError(err, 'Failed to load comments')
        } finally {
            loading.value = false
        }
    }

    async function reload() {
        items.value = new Map()
        nextCursor.value = null
        await loadMore()
    }

    async function postComment(body_md: string, parent_id: number | null = null): Promise<CommentRow> {
        const data = await apiFetch<{ comment: CommentRow }>('/api/comments', {
            method: 'POST',
            body: {
                target_type: targetType,
                target_id: resolvedId.value,
                target_slug: resolvedSlug.value ?? undefined,
                parent_id,
                body_md,
            },
        })
        items.value.set(data.comment.id, data.comment)
        return data.comment
    }

    async function editComment(id: number, body_md: string): Promise<CommentRow> {
        const data = await apiFetch<{ comment: CommentRow }>(`/api/comments/${id}`, {
            method: 'PATCH',
            body: { body_md },
        })
        items.value.set(data.comment.id, data.comment)
        return data.comment
    }

    async function deleteComment(id: number): Promise<void> {
        await apiFetch(`/api/comments/${id}`, { method: 'DELETE' })
        const c = items.value.get(id)
        if (c) c.deleted_at = new Date().toISOString()
    }

    async function reportComment(id: number, reason: string, message: string | null): Promise<void> {
        await apiFetch(`/api/comments/${id}/report`, {
            method: 'POST',
            body: { reason, message },
        })
    }

    // Live updates via WS. Slug targets (e.g. fits) need the slug passed in so
    // the stream filter can disambiguate between fits that all share target_id=0.
    // useCommentStream returns a stub on SSR (no refs, no closures), so this
    // call is free on the server. The stream watcher below is client-only for
    // the same reason: avoid adding reactive effects to the per-SSR effect
    // scope, which was a dominant source of the Vue reactivity leak hunted
    // in April 2026.
    const stream = useCommentStream({
        target_type: targetType,
        target_id: typeof resolvedId.value === 'number' ? resolvedId.value : 0,
        target_slug: resolvedSlug.value,
    })

    if (import.meta.client) {
        // Drain events on tick — Vue's reactivity system triggers this whenever
        // useCommentStream pushes a new event into its internal ref.
        watch(stream.events, () => {
            const evs = stream.consume() as CommentEvent[]
            for (const ev of evs) applyEvent(ev)
        }, { deep: true })
    }

    function applyEvent(ev: CommentEvent) {
        if (ev.event_type === 'deleted') {
            const c = items.value.get(ev.comment_id)
            if (c) c.deleted_at = new Date().toISOString()
            return
        }
        if (ev.comment) {
            items.value.set(ev.comment_id, ev.comment as unknown as CommentRow)
        }
    }

    return {
        items: readonly(items),
        roots,
        childrenByParent,
        totalCount,
        nextCursor: readonly(nextCursor),
        loading: readonly(loading),
        error: readonly(error),
        loadMore,
        reload,
        postComment,
        editComment,
        deleteComment,
        reportComment,
        connected: stream.connected,
    }
}
