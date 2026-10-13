// Browser-only transport core shared by the Go widgets.
import {isRecord} from '../browser/shared/json'
export interface RelayCoreOptions {
    wsUrl:string;path:string;
    onMessage:(channel:string,data:unknown,routingKeys:string[])=>void;
    onOpen?:(ws:WebSocket)=>void;
    onState?:(connected:boolean)=>void;
    reconnect?:boolean;shouldReconnect?:()=>boolean;visibilityPause?:boolean;
}
export function createRelayConnection(options:RelayCoreOptions) {
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
    base.protocol = base.protocol === 'https:' || base.protocol === 'wss:' ? 'wss:' : 'ws:'
    base.pathname = `${base.pathname.replace(/\/$/, '')}${path}`
    const url = base.toString()

    let connected = false
    let disposed = false
    const setConnected = (value:boolean) => { connected = value; options.onState?.(value) }
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
        if (disposed) return
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
            setConnected(true)
            reconnectAttempts = 0
            onOpen?.(sock)
        }

        sock.onmessage = (event) => {
            if (ws !== sock) return
            try {
                if(typeof event.data!=='string')return
                const message:unknown=JSON.parse(event.data)
                if(!isRecord(message)||typeof message.channel!=='string')return
                const keys=Array.isArray(message.routing_keys)?message.routing_keys.filter((key):key is string=>typeof key==='string'):[]
                onMessage(message.channel,message.data,keys)
            } catch { /* skip malformed */ }
        }

        sock.onclose = () => {
            if (ws !== sock) return
            // Release listeners on naturally closed sockets too. Their stale
            // callbacks must not retain a mounted component during backoff.
            sock.onopen = null
            sock.onmessage = null
            sock.onclose = null
            sock.onerror = null
            setConnected(false)
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
        setConnected(false)
    }

    function disconnect() {
        closeSocket(false)
    }

    function dispose() {
        disposed = true
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

    return { get connected(){return connected}, connect, disconnect, dispose }
}
