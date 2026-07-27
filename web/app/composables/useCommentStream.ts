/**
 * useCommentStream — connect to the WebSocket relay's /comments endpoint and
 * receive `new` / `edited` / `deleted` lifecycle events for comments.
 *
 * The WS server broadcasts all comment events to every client.
 * A single shared socket is used for all consumers on the page; each consumer
 * filters client-side by `target_type + target_id` and `domain_id`.
 *
 * Pass `null` to disable.
 */

import { useDomainConfig } from './useDomainConfig'
import { createRelaySocket, type RelaySocket } from './useRelaySocket'

export interface CommentAncestor {
    id: number
    character_id: number
    character_name: string
}

export interface CommentEvent {
    event_type: 'new' | 'edited' | 'deleted'
    comment_id: number
    target_type: number
    target_id: number
    target_slug?: string | null
    domain_id: number | null
    parent_id: number | null
    /** Immediate parent author id — convenience for "is this reply for me" checks. */
    parent_character_id?: number | null
    parent_character_name?: string | null
    /** Full ancestor chain, root → immediate parent. Only populated on 'new' reply events. */
    ancestors?: CommentAncestor[]
    comment?: Record<string, any>
}

export interface CommentStreamFilter {
    target_type: number
    target_id: number
    /** For slug-based targets (fits, pages) — narrows live updates to the exact slug. */
    target_slug?: string | null
}

// Module-level empty refs for the SSR stub. Shared across every SSR call
// so we don't create per-request reactive objects.
const SSR_EMPTY_EVENTS = readonly(ref<CommentEvent[]>([]))
const SSR_DISCONNECTED = readonly(ref(false))
const SSR_STUB = {
    events: SSR_EMPTY_EVENTS,
    connected: SSR_DISCONNECTED,
    consume: (): CommentEvent[] => [],
}

// ── Shared WS connection (module-level, client-only) ───────────────────────
// Since the server broadcasts all comment events regardless of target,
// we only ever need one socket. Ref-counted across all consumers; the relay
// socket (see useRelaySocket) owns reconnect + visibility handling. When the
// last consumer unmounts the relay is disposed and nulled so a later mount
// starts completely fresh.

type CommentListener = (ev: CommentEvent) => void

let relay: RelaySocket | null = null
let sharedListeners: Set<CommentListener> | null = null

function ensureSharedSocket(wsUrl: string): Ref<boolean> {
    if (!sharedListeners) sharedListeners = new Set()

    if (!relay) {
        // Capture the set this relay fans out to — it lives exactly as long
        // as the relay itself (both are torn down together in removeListener).
        const listeners = sharedListeners
        relay = createRelaySocket({
            wsUrl,
            path: '/comments',
            onMessage: (channel, data) => {
                if (channel !== 'comments') return
                const ev = data as CommentEvent
                if (!ev || typeof ev.event_type !== 'string') return
                for (const listener of listeners) {
                    listener(ev)
                }
            },
            reconnect: true,
            shouldReconnect: () => (sharedListeners?.size ?? 0) > 0,
            visibilityPause: true,
        })
    }

    relay.connect()
    return relay.connected
}

function removeListener(listener: CommentListener) {
    sharedListeners?.delete(listener)
    if (sharedListeners && sharedListeners.size === 0) {
        // Last consumer gone — full teardown
        relay?.dispose()
        relay = null
        sharedListeners = null
    }
}

// ── Public composable ──────────────────────────────────────────────────────

export function useCommentStream(filter: CommentStreamFilter | null) {
    if (import.meta.server) {
        return SSR_STUB
    }

    const events = ref<CommentEvent[]>([])
    const { domainConfig } = useDomainConfig()
    const currentDomainId = domainConfig.value?.id ?? null

    const config = useRuntimeConfig()

    const listener: CommentListener = (ev) => {
        // Domain scoping — main site sees null-domain only; custom domain sees its own
        if ((ev.domain_id ?? null) !== currentDomainId) return

        // Target filter
        if (ev.target_type !== filter!.target_type) return
        if (Number(ev.target_id) !== Number(filter!.target_id)) return
        // Slug-scoped filter — fits and pages share target_id=0 across
        // every entry of their type, so the slug is the real key.
        if (filter!.target_slug != null && (ev.target_slug ?? null) !== filter!.target_slug) return

        events.value.push(ev)
        // Cap retained events so a long-lived page doesn't accumulate forever
        if (events.value.length > 500) events.value.splice(0, events.value.length - 500)
    }

    let connectedRef: Ref<boolean> | null = null

    onMounted(() => {
        if (!filter) return
        const wsUrl = config.public.wsUrl as string | undefined
        if (!wsUrl) return

        connectedRef = ensureSharedSocket(wsUrl)
        sharedListeners!.add(listener)
    })

    onUnmounted(() => {
        removeListener(listener)
    })

    /** Drain events that have been processed by the consumer. */
    const consume = (): CommentEvent[] => {
        const out = events.value.slice()
        events.value = []
        return out
    }

    return {
        events: readonly(events),
        connected: computed(() => connectedRef?.value ?? false),
        consume,
    }
}
