/**
 * Global watcher for in-flight posted killmails.
 *
 * Lifecycle:
 *  1. `watch(ids)` adds killmail ids to a pending set and ensures the WebSocket
 *     subscription to topic "all" is open.
 *  2. As killmails arrive (either via WS broadcast or a periodic safety poll),
 *     they are removed from `pending` and pushed onto `notifications` with a
 *     staggered dismissal time: first arrival 60s, then +15s for each subsequent.
 *  3. When `pending` becomes empty, the WS is closed. Notifications stay on
 *     screen until they hit their dismissAt or the user clicks them.
 *
 * Notifications survive route changes because the underlying state lives in
 * useState — the matching <KillWatcherToasts> component is mounted in the
 * default layout once.
 */

import { createRelaySocket, type RelaySocket } from './useRelaySocket'

interface ArrivedKill {
    killmail_id: number
    ship_type_id: number | null
    ship_name: string | null
    victim_character_id: number | null
    victim_character_name: string | null
    victim_corporation_id: number | null
    victim_corporation_name: string | null
    total_value: number | null
}

export interface KillNotification extends ArrivedKill {
    uid: number
    dismissAt: number  // epoch ms
}

let _uid = 0
// Lazy singleton relay (see useRelaySocket): opened after a killmail paste,
// closed when the pending set drains. Deliberately no reconnect — the 5s
// safety poll self-heals if the WS drops or a broadcast is missed.
let _relay: RelaySocket | null = null
let _safetyTimer: ReturnType<typeof setInterval> | null = null
let _arrivedCount = 0  // counter for stagger durations

export function useKillWatcher() {
    const pending = useState<number[]>('km-watcher-pending', () => [])
    const notifications = useState<KillNotification[]>('km-watcher-notifs', () => [])

    const config = useRuntimeConfig()

    function ensureSocket() {
        if (!import.meta.client) return
        const wsUrl = config.public.wsUrl
        if (!wsUrl) return
        if (!_relay) {
            // The handlers close over this call's `pending`/`onArrived`, but
            // both resolve to shared useState + module-level counters, so it
            // doesn't matter which component instance created the relay.
            _relay = createRelaySocket({
                wsUrl: wsUrl as string,
                path: '/killlist',
                onOpen: (ws) => {
                    ws.send(JSON.stringify({ action: 'subscribe', topics: ['all'] }))
                },
                onMessage: (channel, data) => {
                    if (channel !== 'killlist' || !data?.killmail) return
                    const km = data.killmail
                    if (pending.value.includes(km.killmail_id)) {
                        onArrived({
                            killmail_id: km.killmail_id,
                            ship_type_id: km.ship_type_id ?? null,
                            ship_name: km.ship_name ?? null,
                            victim_character_id: km.victim_character_id ?? null,
                            victim_character_name: km.victim_character_name ?? null,
                            victim_corporation_id: km.victim_corporation_id ?? null,
                            victim_corporation_name: km.victim_corporation_name ?? null,
                            total_value: km.total_value ?? null,
                        })
                    }
                },
                reconnect: false,  // 5s safety poll self-heals missed broadcasts
            })
        }
        _relay.connect()
    }

    function closeSocket() {
        _relay?.disconnect()
    }

    function ensureSafetyPoll() {
        if (_safetyTimer) return
        _safetyTimer = setInterval(async () => {
            if (pending.value.length === 0) {
                if (_safetyTimer) { clearInterval(_safetyTimer); _safetyTimer = null }
                return
            }
            // Check each pending killmail; if it now exists in DB but we missed
            // the WS broadcast, fetch its data and synthesize an arrival.
            for (const id of [...pending.value]) {
                try {
                    const res = await apiFetch<{ exists: boolean }>(`/api/killmail/${id}/exists`)
                    if (res.exists) {
                        // Pull minimal display info from the full killmail endpoint
                        const km = await apiFetch<any>(`/api/killmail/${id}`).catch(() => null)
                        onArrived({
                            killmail_id: id,
                            ship_type_id: km?.victim?.ship_type_id ?? null,
                            ship_name: km?.victim?.ship_name ?? null,
                            victim_character_id: km?.victim?.character_id ?? null,
                            victim_character_name: km?.victim?.character_name ?? null,
                            victim_corporation_id: km?.victim?.corporation_id ?? null,
                            victim_corporation_name: km?.victim?.corporation_name ?? null,
                            total_value: km?.total_value ?? null,
                        })
                    }
                } catch { /* keep waiting */ }
            }
        }, 5000)
    }

    function onArrived(km: ArrivedKill) {
        // Remove from pending
        const idx = pending.value.indexOf(km.killmail_id)
        if (idx === -1) return
        pending.value = pending.value.filter(id => id !== km.killmail_id)

        // Stagger: 60s, 75s, 90s, 105s, 120s, ...
        _arrivedCount++
        const seconds = 60 + (_arrivedCount - 1) * 15
        notifications.value = [
            ...notifications.value,
            {
                ...km,
                uid: ++_uid,
                dismissAt: Date.now() + seconds * 1000,
            },
        ]

        if (pending.value.length === 0) closeSocket()
    }

    function watch(ids: number[]) {
        if (!import.meta.client || ids.length === 0) return
        const merged = [...new Set([...pending.value, ...ids])]
        pending.value = merged
        _arrivedCount = 0  // reset stagger counter for this batch
        ensureSocket()
        ensureSafetyPoll()
    }

    function dismiss(uid: number) {
        notifications.value = notifications.value.filter(n => n.uid !== uid)
    }

    return {
        pending: readonly(pending),
        notifications: readonly(notifications),
        watch,
        dismiss,
    }
}
