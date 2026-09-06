/**
 * createRelaySocket — shared WebSocket core for the relay clients
 * (useKillStream, useCommentStream, useKillWatcher, useAnnouncements).
 *
 * Owns the mechanics every client used to hand-roll:
 *   - `wsUrl.replace(/^http/, 'ws')` + endpoint path
 *   - JSON.parse of `{ channel, data }` frames (malformed frames are dropped;
 *     channel filtering is the caller's job, done in `onMessage`)
 *   - optional exponential-backoff reconnect: 1s * 2^attempts, capped at 30s,
 *     attempts reset on successful open
 *   - optional visibilitychange pause: close while the tab is hidden, reopen
 *     when it becomes visible (backoff state survives the pause)
 *
 * What it deliberately does NOT own: listener sets / ref-counting, pooling,
 * per-consumer filtering, and any polling — those stay in the composables,
 * which differ per client.
 *
 * Client-only: on the server a shared inert stub is returned (same pattern as
 * the SSR stubs at the top of useKillStream/useCommentStream), so calling it
 * during SSR is safe but pointless.
 *
 * Intentional closes (disconnect/dispose, visibility pause) detach the socket's
 * handlers before closing, so a stale `onclose` can never clobber a newer
 * socket or schedule a reconnect after teardown.
 */

export interface RelaySocketOptions {
    /** http(s) relay base URL — converted to ws(s) internally. */
    wsUrl: string
    /** Endpoint path on the relay, e.g. '/killlist' or '/comments'. */
    path: string
    /**
     * Called for every well-formed `{ channel, data }` frame. Runs inside the
     * same try/catch as JSON.parse, so a throwing handler drops that frame
     * silently — identical to the original per-client handlers.
     */
    onMessage: (channel: string, data: any, routingKeys: string[]) => void
    /** Called when the socket opens, after backoff state is reset. */
    onOpen?: (ws: WebSocket) => void
    /** Reconnect with exponential backoff on unexpected close. Default false. */
    reconnect?: boolean
    /**
     * Guard consulted before an unexpected-close reconnect is scheduled, before
     * a scheduled reconnect fires, and before reopening on tab-visible.
     * Defaults to always-true (announcements). killStream/commentStream pass
     * "any listeners left?".
     */
    shouldReconnect?: () => boolean
    /**
     * Close the socket while `document.hidden` and reopen (if shouldReconnect)
     * when the tab becomes visible again. The listener is attached at creation
     * and removed by dispose(). Default false.
     */
    visibilityPause?: boolean
}

export interface RelaySocket {
    /** True while the socket is OPEN. One ref per instance. */
    connected: Ref<boolean>
    /** Open the socket unless it is already CONNECTING/OPEN. Idempotent. */
    connect: () => void
    /**
     * Intentional close: clears any pending reconnect timer, resets backoff,
     * closes without triggering auto-reconnect. The instance stays usable —
     * a later connect() reopens.
     */
    disconnect: () => void
    /**
     * disconnect() + remove the visibilitychange listener and reset the hidden
     * flag. Call when the instance is being discarded for good.
     */
    dispose: () => void
}

// Module-level inert stub for SSR — shared across every server-side call so we
// don't create per-request reactive objects.
const SSR_CONNECTED = ref(false)
const SSR_NOOP = () => { /* no-op */ }
const SSR_RELAY_STUB: RelaySocket = {
    connected: SSR_CONNECTED,
    connect: SSR_NOOP,
    disconnect: SSR_NOOP,
    dispose: SSR_NOOP,
}

export function createRelaySocket(options: RelaySocketOptions): RelaySocket {
    if (import.meta.server) {
        return SSR_RELAY_STUB
    }

    const {
        wsUrl,
        path,
        onMessage,
        onOpen,
        reconnect = false,
        shouldReconnect = () => true,
        visibilityPause = false,
    } = options

    const base = new URL(wsUrl, window.location.origin)
    base.protocol = base.protocol === 'https:' ? 'wss:' : 'ws:'
    base.pathname = `${base.pathname.replace(/\/$/, '')}${path}`
    const url = base.toString()

    const connected = ref(false)
    let ws: WebSocket | null = null
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null
    let reconnectAttempts = 0
    let hiddenByVisibility = false
    let visibilityHandler: (() => void) | null = null

    function clearReconnectTimer() {
        if (reconnectTimer) {
            clearTimeout(reconnectTimer)
            reconnectTimer = null
        }
    }

    function scheduleReconnect() {
        if (reconnectTimer) return
        const delay = Math.min(1000 * 2 ** reconnectAttempts, 30000)
        reconnectAttempts++
        reconnectTimer = setTimeout(() => {
            reconnectTimer = null
            if (shouldReconnect()) connect()
        }, delay)
    }

    function connect() {
        if (!import.meta.client) return
        if (visibilityPause && document.hidden) return
        if (ws && ws.readyState <= 1) return // CONNECTING or OPEN

        let sock: WebSocket
        try {
            sock = new WebSocket(url)
        } catch {
            if (reconnect) scheduleReconnect()
            return
        }
        ws = sock

        sock.onopen = () => {
            if (ws !== sock) return // superseded socket
            connected.value = true
            reconnectAttempts = 0
            onOpen?.(sock)
        }

        sock.onmessage = (event) => {
            if (ws !== sock) return
            try {
                const { channel, data, routing_keys } = JSON.parse(event.data)
                onMessage(channel, data, Array.isArray(routing_keys) ? routing_keys : [])
            } catch { /* skip malformed */ }
        }

        sock.onclose = () => {
            if (ws !== sock) return
            connected.value = false
            ws = null
            if (reconnect && !hiddenByVisibility && shouldReconnect()) {
                scheduleReconnect()
            }
        }

        sock.onerror = () => { /* onclose handles cleanup */ }
    }

    /** Close without auto-reconnect; optionally keep backoff state. */
    function closeSocket(preserveAttempts: boolean) {
        clearReconnectTimer()
        if (!preserveAttempts) reconnectAttempts = 0
        if (ws) {
            const sock = ws
            ws = null
            // Detach so the async close event can't schedule a reconnect or
            // clobber a socket opened after this intentional close.
            sock.onopen = null
            sock.onmessage = null
            sock.onclose = null
            sock.onerror = null
            try { sock.close() } catch { /* already closing */ }
        }
        connected.value = false
    }

    function disconnect() {
        closeSocket(false)
    }

    function dispose() {
        closeSocket(false)
        hiddenByVisibility = false
        if (visibilityHandler) {
            document.removeEventListener('visibilitychange', visibilityHandler)
            visibilityHandler = null
        }
    }

    if (visibilityPause) {
        visibilityHandler = () => {
            if (document.hidden) {
                hiddenByVisibility = true
                // Keep backoff state — a failing connection shouldn't restart
                // its backoff just because the tab was hidden for a moment.
                closeSocket(true)
            } else {
                hiddenByVisibility = false
                if (shouldReconnect()) connect()
            }
        }
        document.addEventListener('visibilitychange', visibilityHandler)
    }

    return { connected, connect, disconnect, dispose }
}
