/**
 * Kill stream — connects via WebSocket to the relay.
 * Killlist events arrive pre-hydrated with all names resolved.
 *
 * Pass null to disable the stream (e.g. for war/battle pages).
 *
 * Multiple components subscribing to the same topics share a single WebSocket
 * connection (ref-counted). The socket closes when the last consumer unmounts.
 */

import { createRelaySocket, type RelaySocket } from './useRelaySocket'
import type { KilllistRow } from '#shared/utils/killlistRow'
import { matchesDomainKeys, streamPoolKey } from '~/utils/killStreamPolicy'

/**
 * Relay `killlist` events carry exactly the row shape the REST killlist
 * endpoints return — see KilllistRow. Kept as an alias rather than a second
 * hand-maintained interface, which is how the two drifted apart before.
 */
type HydratedKill = KilllistRow

// Module-level empty refs for the SSR stub. Shared across every SSR call so
// we don't create per-request reactive objects.
const SSR_EMPTY_KILLS = readonly(ref<HydratedKill[]>([]))
const SSR_DISCONNECTED = readonly(ref(false))
const SSR_FALSE = ref(false)
const SSR_ZERO = readonly(ref(0))
const SSR_NOOP = () => { /* no-op */ }
const SSR_STUB = {
    kills: SSR_EMPTY_KILLS,
    connected: SSR_DISCONNECTED,
    paused: SSR_FALSE,
    newCount: SSR_ZERO,
    pause: SSR_NOOP,
    resume: SSR_NOOP,
    flush: SSR_NOOP,
    disconnect: SSR_NOOP,
}

// ── Shared WS connection pool (module-level, client-only) ──────────────────
// Keyed by sorted topic string. Each entry pairs a relay socket (which owns
// reconnect + visibility handling, see useRelaySocket) with the ref-counted
// set of listener callbacks.

type KillListener = (km: HydratedKill, keys: string[]) => void

interface PoolEntry {
    socket: RelaySocket
    listeners: Set<KillListener>
}

const socketPool = new Map<string, PoolEntry>()

function getOrCreateSocket(topics: string[], wsUrl: string, background: boolean): PoolEntry {
    const key = streamPoolKey(topics, wsUrl, background)
    let entry = socketPool.get(key)
    if (entry) return entry

    const listeners = new Set<KillListener>()
    const socket = createRelaySocket({
        wsUrl,
        path: '/killlist',
        onOpen: (ws) => {
            ws.send(JSON.stringify({ action: 'subscribe', topics }))
        },
        onMessage: (channel, data, keys) => {
            if (channel === 'killlist' && data?.killmail) {
                const km = data.killmail as HydratedKill
                for (const listener of listeners) {
                    listener(km, keys)
                }
            }
        },
        reconnect: true,
        shouldReconnect: () => listeners.size > 0,
        visibilityPause: !background,
    })

    entry = { socket, listeners }
    socketPool.set(key, entry)
    socket.connect()

    return entry
}

function releaseSocket(key: string, listener: KillListener) {
    const entry = socketPool.get(key)
    if (!entry) return

    entry.listeners.delete(listener)
    if (entry.listeners.size > 0) return

    // Last consumer gone — tear down
    entry.socket.dispose()
    socketPool.delete(key)
}

// ── Public composable ──────────────────────────────────────────────────────

export function useKillStream(topics: string[] | null, options: { background?: boolean; onKill?: (kill: HydratedKill) => void } = {}) {
    if (import.meta.server) {
        return SSR_STUB
    }

    const kills = ref<HydratedKill[]>([])
    const paused = ref(false)
    const newCount = ref(0)

    // Domain mode: cache entity IDs for client-side filtering
    const { domainConfig } = useDomainConfig()
    const domainEntityIds = domainConfig.value?.entityIds ?? null

    const MAX_KILLS = 200

    const config = useRuntimeConfig()

    // Listener callback — receives every kill from the shared socket,
    // applies domain filtering and pause logic per-consumer.
    const onKill: KillListener = (km, keys) => {
        if (domainEntityIds) {
            if (!matchesDomainKeys(keys, domainEntityIds, km)) return
        }

        options.onKill?.(km)

        if (paused.value) {
            newCount.value++
        } else {
            kills.value.unshift(km)
            if (kills.value.length > MAX_KILLS) {
                kills.value.length = MAX_KILLS
            }
        }
    }

    let sharedSocket: PoolEntry | null = null
    let poolKey = ''
    const connected = ref(false)
    let stopConnectedWatch: (() => void) | null = null

    onMounted(() => {
        if (!topics || topics.length === 0) return
        const wsUrl = config.public.wsUrl as string | undefined
        if (!wsUrl) return

        poolKey = streamPoolKey(topics, wsUrl, options.background ?? false)
        sharedSocket = getOrCreateSocket(topics, wsUrl, options.background ?? false)
        sharedSocket.listeners.add(onKill)
        stopConnectedWatch = watch(sharedSocket.socket.connected, (value) => {
            connected.value = value
        }, { immediate: true })
    })

    onUnmounted(() => {
        stopConnectedWatch?.()
        stopConnectedWatch = null
        if (topics && sharedSocket) {
            releaseSocket(poolKey, onKill)
            sharedSocket = null
        }
    })

    const pause = () => { paused.value = true }

    const resume = () => {
        paused.value = false
        newCount.value = 0
    }

    const flush = () => {
        newCount.value = 0
    }

    const disconnect = () => {
        stopConnectedWatch?.()
        stopConnectedWatch = null
        connected.value = false
        if (topics && sharedSocket) {
            releaseSocket(poolKey, onKill)
            sharedSocket = null
        }
    }

    return {
        kills: readonly(kills),
        connected: readonly(connected),
        paused,
        newCount: readonly(newCount),
        pause,
        resume,
        flush,
        disconnect,
    }
}
