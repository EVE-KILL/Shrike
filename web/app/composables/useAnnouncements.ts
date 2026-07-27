/**
 * useAnnouncements — global announcement state for the three-tier system.
 *
 * Fetches active announcements from the server on init and provides
 * tier-filtered computed lists for ticker (1), banner (2), and modal (3).
 *
 * Real-time updates via WebSocket:
 *   - Connects to /announcements WS endpoint
 *   - Handles new/updated/expired/archived events reactively
 *   - Display components need zero changes (already reactive to computed lists)
 *
 * Dismissal is hybrid:
 *   - localStorage (ek_dismissed_announcements) for all visitors
 *   - Server sync (POST /api/announcements/:id/dismiss) for authenticated users
 *
 * On page load for logged-in users, the server returns their dismissed IDs,
 * which are merged into localStorage so new devices are seeded.
 */

import type {
    Announcement,
    AnnouncementsResponse,
    DismissedAnnouncementIdsResponse,
} from '#shared/api'
import { createRelaySocket, type RelaySocket } from './useRelaySocket'

const LS_KEY = 'ek_dismissed_announcements'
const LS_TICKER_COLLAPSED = 'ek_ticker_collapsed'

// Module-level WS state (client-only singleton). The relay (see
// useRelaySocket) owns reconnect backoff; visibility handling stays here in
// init() because it also couples refresh() + the expiry timer.
let relay: RelaySocket | null = null
let expiryTimer: ReturnType<typeof setInterval> | null = null
let fetched = false

export function useAnnouncements() {
    const { isAuthenticated } = useAuth()
    const config = useRuntimeConfig()

    const items = useState<Announcement[]>('announcements-items', () => [])
    const dismissedIds = useState<Set<number>>('announcements-dismissed', () => new Set())
    const tickerCollapsed = useState<boolean>('announcements-ticker-collapsed', () => false)

    // Reactive clock tick — incremented every 10s to force computed re-evaluation
    // so expired announcements are dropped client-side without waiting for WS.
    const tick = useState<number>('announcements-tick', () => 0)

    function isActive(a: Announcement): boolean {
        const now = Date.now()
        const expires = new Date(a.expires_at as string).getTime()
        const starts = new Date(a.starts_at as string).getTime()
        return starts <= now && expires > now
    }

    // ── Tier-filtered lists (excluding dismissed + expired) ─────────────────
    // Ephemeral ticker ids are banded by category — see the ID map in
    // backend/src/services/TickerAnnouncements.ts. Each category is capped
    // separately so a burst of one kind can't crowd out the others: a busy
    // hour of battles used to be able to fill the whole strip.
    const BATTLE_ID_MAX = -2_000_000_000
    const WAR_ID_MAX = -3_000_000_000
    const MAX_PER_CATEGORY: Record<string, number> = { battle: 5, war: 4, kill: 8 }

    const categoryOf = (id: number): 'war' | 'battle' | 'kill' => {
        if (id <= WAR_ID_MAX) return 'war'
        if (id <= BATTLE_ID_MAX) return 'battle'
        return 'kill'
    }

    const newestFirst = (a: Announcement, b: Announcement) =>
        new Date(b.created_at as string).getTime() - new Date(a.created_at as string).getTime()

    const tickerItems = computed(() => {
        void tick.value // depend on tick for reactivity
        const active = items.value.filter(a => a.tier === 1 && !dismissedIds.value.has(a.id) && isActive(a))

        const buckets: Record<string, Announcement[]> = { war: [], battle: [], kill: [] }
        for (const a of active) buckets[categoryOf(a.id)]!.push(a)

        return Object.entries(buckets)
            .flatMap(([key, list]) => list.sort(newestFirst).slice(0, MAX_PER_CATEGORY[key] ?? 8))
            .sort(newestFirst)
    })
    const bannerItems = computed(() => {
        void tick.value
        return items.value.filter(a => a.tier === 2 && !dismissedIds.value.has(a.id) && isActive(a))
    })
    const modalItems = computed(() => {
        void tick.value
        return items.value.filter(a => a.tier === 3 && !dismissedIds.value.has(a.id) && isActive(a))
    })

    // ── localStorage helpers ────────────────────────────────────────────────
    function loadDismissedFromStorage(): Set<number> {
        if (!import.meta.client) return new Set()
        try {
            const raw = localStorage.getItem(LS_KEY)
            if (!raw) return new Set()
            const arr = JSON.parse(raw)
            if (Array.isArray(arr)) return new Set(arr.map(Number).filter(Boolean))
        } catch { /* corrupt data */ }
        return new Set()
    }

    function saveDismissedToStorage() {
        if (!import.meta.client) return
        try {
            localStorage.setItem(LS_KEY, JSON.stringify([...dismissedIds.value]))
        } catch { /* quota */ }
    }

    // ── Dismiss action ──────────────────────────────────────────────────────
    async function dismiss(id: number) {
        dismissedIds.value = new Set([...dismissedIds.value, id])
        saveDismissedToStorage()

        // Server sync for authenticated users (fire and forget)
        if (isAuthenticated.value) {
            apiFetch(`/api/me/announcements/${id}/dismissal`, { method: 'PUT' }).catch(() => {})
        }
    }

    // ── Ticker collapse toggle ──────────────────────────────────────────────
    function toggleTicker() {
        tickerCollapsed.value = !tickerCollapsed.value
        if (import.meta.client) {
            try {
                localStorage.setItem(LS_TICKER_COLLAPSED, tickerCollapsed.value ? '1' : '0')
            } catch { /* */ }
        }
    }

    // ── Sort helper ─────────────────────────────────────────────────────────
    function sortItems() {
        items.value.sort((a, b) =>
            b.tier - a.tier
            || new Date(b.created_at as string).getTime() - new Date(a.created_at as string).getTime(),
        )
    }

    // ── WebSocket ───────────────────────────────────────────────────────────
    function handleWsEvent(channel: string, data: any) {
        // Runs inside the relay's malformed-frame try/catch, so a bad payload
        // throwing here is dropped silently — same as the old handler.
        if (channel !== 'announcements') return

        const { event_type, announcement } = data
        if (!announcement?.id) return

        if (event_type === 'new' || event_type === 'updated') {
            // Upsert by id. Backend re-emits battle tickers as kill counts grow
            // during backfill; replacing instead of ignoring keeps the displayed
            // count fresh and avoids duplicate rows if a re-detection somehow
            // produces a fresh id for the same real-world battle.
            const idx = items.value.findIndex(a => a.id === announcement.id)
            if (idx >= 0) {
                items.value[idx] = announcement
            } else {
                items.value.push(announcement)
                sortItems()
            }
        } else if (event_type === 'expired' || event_type === 'archived') {
            items.value = items.value.filter(a => a.id !== announcement.id)
        }
    }

    function connectWs() {
        if (!import.meta.client) return
        const wsUrl = config.public.wsUrl as string | undefined
        if (!wsUrl) return

        if (!relay) {
            // handleWsEvent closes over this call's `items`, but that's a
            // shared useState — any component instance's closure is equivalent.
            relay = createRelaySocket({
                wsUrl,
                path: '/announcements',
                onMessage: handleWsEvent,
                reconnect: true,
            })
        }
        relay.connect()
    }

    function disconnectWs() {
        // Clears any pending reconnect timer, resets backoff, and closes
        // without triggering the auto-reconnect — same as the old inline code.
        relay?.disconnect()
    }

    // ── Fetch and init ──────────────────────────────────────────────────────
    async function refresh() {
        try {
            // The shared list is cached for the anonymous majority; per-user
            // dismissals come from their own authenticated endpoint.
            const [res, dismissed] = await Promise.all([
                apiFetch<AnnouncementsResponse>('/api/announcements'),
                isAuthenticated.value
                    ? apiFetch<DismissedAnnouncementIdsResponse>(
                        '/api/me/announcements/dismissed',
                    ).catch(() => null)
                    : Promise.resolve(null),
            ])

            items.value = res.announcements

            // Merge server-side dismissals into local set
            if (dismissed?.dismissedIds?.length) {
                const merged = new Set([...dismissedIds.value, ...dismissed.dismissedIds])
                dismissedIds.value = merged
                saveDismissedToStorage()
            }
        } catch {
            // Silent — announcements are non-critical
        }
    }

    function init() {
        if (fetched) return
        if (!import.meta.client) return
        fetched = true

        // Bootstrap from localStorage
        dismissedIds.value = loadDismissedFromStorage()

        // Restore ticker collapsed state
        try {
            tickerCollapsed.value = localStorage.getItem(LS_TICKER_COLLAPSED) === '1'
        } catch { /* */ }

        refresh()
        connectWs()
        startExpiryTimer()

        // Hidden tabs don't need a live announcement feed — drop the socket
        // and expiry timer entirely, then catch up with a refresh() when the
        // tab becomes visible again (same pattern as useKillStream).
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                disconnectWs()
                stopExpiryTimer()
            } else {
                refresh()
                connectWs()
                startExpiryTimer()
            }
        })
    }

    // Tick every 10s to re-evaluate expiry in computed lists.
    // Also prunes fully-expired items from the raw array to keep it clean.
    function startExpiryTimer() {
        if (expiryTimer) return
        expiryTimer = setInterval(() => {
            tick.value++
            // Prune items expired more than 60s ago (buffer for WS lag)
            const cutoff = Date.now() - 60_000
            items.value = items.value.filter(a => {
                const expires = new Date(a.expires_at as string).getTime()
                return expires > cutoff
            })
        }, 10_000)
    }

    function stopExpiryTimer() {
        if (expiryTimer) {
            clearInterval(expiryTimer)
            expiryTimer = null
        }
    }

    if (import.meta.client) {
        init()
    }

    return {
        items,
        tickerItems,
        bannerItems,
        modalItems,
        dismissedIds: readonly(dismissedIds),
        tickerCollapsed: readonly(tickerCollapsed),
        dismiss,
        toggleTicker,
        refresh,
    }
}
