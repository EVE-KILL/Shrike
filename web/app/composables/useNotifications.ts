/**
 * useNotifications — global notification state for the navbar bell.
 *
 * Read state is server-authoritative via `users.last_seen_notification_id`
 * so the unread cursor stays consistent across devices. localStorage is a
 * write-through cache that gives instant UI before /api/auth/me lands and
 * smooths the optimistic mark-read flow. The server uses GREATEST() in the
 * UPDATE so a stale tab can't accidentally regress the cursor.
 *
 * On first load we backfill recent replies via /api/notifications/replies
 * so users who were offline still see what they missed.
 *
 * TODO: add a dedicated /notifications WS endpoint for live push.
 */

export interface NotificationItem {
    id: number
    target_type: number
    target_id: number
    parent_id: number | null
    parent_comment_id?: number
    parent_snippet?: string
    body_html: string
    created_at: string
    character_id: number
    character_name: string
    corporation_id: number
    corporation_name: string
    alliance_id: number | null
    alliance_name: string | null
}

interface RepliesResponse {
    replies: NotificationItem[]
    highestId: number
}

const LAST_SEEN_KEY = 'ek_notifications_last_seen_id'
/** Cap retained items to keep memory bounded for long-running tabs. */
const MAX_ITEMS = 100

let initialized = false
let backfilling = false

export function useNotifications() {
    const { user, isAuthenticated } = useAuth()

    // useState gives per-request isolation on the server and a shared singleton
    // on the client, which is exactly what we want for global notification state.
    const items = useState<NotificationItem[]>('notifications-items', () => [])
    const lastSeenId = useState<number>('notifications-last-seen', () => 0)

    const unreadCount = computed(() => {
        if (!isAuthenticated.value) return 0
        return items.value.filter((i) => i.id > lastSeenId.value).length
    })

    const isUnread = (item: NotificationItem) => item.id > lastSeenId.value

    /**
     * Bump the read cursor to the newest item. Optimistically updates the local
     * ref and the localStorage write-through cache, then POSTs to the server so
     * the new value is shared across devices. The server uses GREATEST() to
     * resolve races with other tabs and returns the authoritative cursor.
     */
    async function markAllRead() {
        const top = items.value[0]
        if (!top) return
        if (top.id <= lastSeenId.value) return

        const target = top.id
        // Optimistic local update so the badge clears instantly.
        lastSeenId.value = target
        if (import.meta.client) {
            try { localStorage.setItem(LAST_SEEN_KEY, String(target)) } catch { /* quota */ }
        }

        try {
            const res = await apiFetch<{ lastSeenNotificationId: number }>('/api/notifications/mark-read', {
                method: 'POST',
                body: { id: target },
            })
            // Trust the server's response — another device may have already
            // pushed the cursor higher than what we just sent.
            const authoritative = Number(res.lastSeenNotificationId ?? target)
            if (authoritative > lastSeenId.value) {
                lastSeenId.value = authoritative
                if (import.meta.client) {
                    try { localStorage.setItem(LAST_SEEN_KEY, String(authoritative)) } catch { /* */ }
                }
            }
        } catch {
            // Silent — the optimistic local update still works for this tab,
            // and the cursor will sync next time the user opens the page (the
            // server value is loaded into AuthUser by getSessionUser).
        }
    }

    /** Reset state on logout. Called by the auth flow. */
    function reset() {
        items.value = []
        lastSeenId.value = 0
        if (import.meta.client) {
            try { localStorage.removeItem(LAST_SEEN_KEY) } catch { /* */ }
        }
        initialized = false
    }

    async function backfill() {
        if (backfilling || !isAuthenticated.value) return
        backfilling = true
        try {
            const since = lastSeenId.value
            // Use apiFetch directly so SSR doesn't try to share state.
            const res = await apiFetch<RepliesResponse>('/api/notifications/replies', {
                params: { since: since > 0 ? since : undefined, limit: 50 },
            })
            mergeItems(res.replies)
        } catch {
            // Silent — the bell can stay empty until the next WS event lands.
        } finally {
            backfilling = false
        }
    }

    function mergeItems(incoming: NotificationItem[]) {
        if (!incoming.length) return
        const seen = new Set(items.value.map((i) => i.id))
        for (const item of incoming) {
            if (!seen.has(item.id)) {
                items.value.unshift(item)
                seen.add(item.id)
            }
        }
        // Sort newest-first and cap.
        items.value.sort((a, b) => b.id - a.id)
        if (items.value.length > MAX_ITEMS) items.value.splice(MAX_ITEMS)
    }

    /** Idempotent — multiple components can call this; only the first wins. */
    function init() {
        if (initialized) return
        if (!import.meta.client) return
        if (!isAuthenticated.value) return
        initialized = true

        // Read cursor: server is source of truth, localStorage is a fast cache
        // for the brief moment between hydration and the /api/auth/me response.
        // Take whichever is higher so we never regress an already-acknowledged
        // cursor; the user-watch below resyncs when the full auth lands.
        let bootstrap = 0
        try {
            const stored = localStorage.getItem(LAST_SEEN_KEY)
            if (stored) bootstrap = Number(stored) || 0
        } catch { /* */ }
        const fromUser = user.value?.lastSeenNotificationId ?? 0
        lastSeenId.value = Math.max(bootstrap, fromUser)

        backfill()
    }

    // React to auth state changes — log in starts the stream, log out tears down.
    if (import.meta.client) {
        watch(isAuthenticated, (v) => {
            if (v) init()
            else reset()
        }, { immediate: true })

        // When the full auth payload replaces the cookie-hint stub, pick up
        // the server-side cursor. Use the higher of (current local, server) so
        // an optimistic mark-read in this tab isn't undone by an older value.
        watch(() => user.value?.lastSeenNotificationId, (serverValue) => {
            if (serverValue == null) return
            if (serverValue > lastSeenId.value) {
                lastSeenId.value = serverValue
                try { localStorage.setItem(LAST_SEEN_KEY, String(serverValue)) } catch { /* */ }
            }
        })
    }

    return {
        items,
        unreadCount,
        isUnread,
        markAllRead,
        reset,
        refresh: backfill,
    }
}
