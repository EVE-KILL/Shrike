import { createRelaySocket, type RelaySocket } from './useRelaySocket'
import type { KilllistRow } from '#shared/utils/killlistRow'

export type TrackerTargetType = 'character' | 'corporation' | 'alliance' | 'system' | 'constellation' | 'region'

export interface EntityTracker {
    id: number
    target_type: TrackerTargetType
    target_id: number
    target_name: string
    target_ticker: string | null
    enabled: boolean
    notifications_enabled: boolean
    created_at: string
    updated_at: string
    event_count: number
    last_event_at: string | null
    unread_notifications: number
}

export interface TrackerNotification {
    id: number
    created_at: string
    is_read: boolean
    tracker_id: number
    target_type: TrackerTargetType
    target_id: number
    target_name: string
    target_ticker: string | null
    match_role: 'victim' | 'attacker' | 'both' | 'location'
    killmail_id: number
    occurred_at: string
    total_value: number
    ship_type_id: number | null
    ship_name: string | null
    victim_character_id: number | null
    victim_character_name: string | null
    solar_system_id: number
    solar_system_name: string | null
    region_id: number | null
    region_name: string | null
}

interface TrackerListResponse {
    trackers: EntityTracker[]
    limit: number
}

interface TrackerNotificationsResponse {
    notifications: TrackerNotification[]
    unreadCount: number
}

const BROWSER_ALERTS_KEY = 'ek_tracker_browser_alerts_while_open'
const MAX_SEEN_KILLS = 200

let initialized = false
let authWatchStarted = false
let relay: RelaySocket | null = null
let relayTopicKey = ''
const seenKillIds = new Set<number>()

function trackerTopic(tracker: EntityTracker): string[] {
    const id = tracker.target_id
    if (tracker.target_type === 'character' || tracker.target_type === 'corporation' || tracker.target_type === 'alliance') {
        return [`victim.${id}`, `attacker.${id}`]
    }
    return [`${tracker.target_type}.${id}`]
}

function disposeRelay() {
    relay?.dispose()
    relay = null
    relayTopicKey = ''
}

export function useTrackerNotifications() {
    const { isAuthenticated } = useAuth()
    const config = useRuntimeConfig()
    const router = useRouter()

    const trackers = useState<EntityTracker[]>('entity-trackers', () => [])
    const trackerLimit = useState<number>('entity-trackers-limit', () => 50)
    const notifications = useState<TrackerNotification[]>('entity-tracker-notifications', () => [])
    const unreadCount = useState<number>('entity-tracker-unread', () => 0)
    const trackersLoading = useState<boolean>('entity-trackers-loading', () => false)
    const notificationsLoading = useState<boolean>('entity-tracker-notifications-loading', () => false)
    const browserAlertsEnabled = useState<boolean>('entity-tracker-browser-alerts', () => false)
    const browserPermission = useState<NotificationPermission | 'unsupported'>('entity-tracker-browser-permission', () => 'default')

    async function refreshNotifications() {
        if (!isAuthenticated.value || notificationsLoading.value) return
        notificationsLoading.value = true
        try {
            const result = await apiFetch<TrackerNotificationsResponse>('/api/me/tracker-notifications', {
                params: { limit: 50 },
            })
            notifications.value = result.notifications ?? []
            unreadCount.value = Number(result.unreadCount ?? 0)
        } finally {
            notificationsLoading.value = false
        }
    }

    function showBrowserAlert(kill: KilllistRow) {
        if (!browserAlertsEnabled.value || browserPermission.value !== 'granted') return
        if (seenKillIds.has(kill.killmail_id)) return
        seenKillIds.add(kill.killmail_id)
        if (seenKillIds.size > MAX_SEEN_KILLS) {
            const oldest = seenKillIds.values().next().value
            if (oldest !== undefined) seenKillIds.delete(oldest)
        }

        const persisted = notifications.value.find(item => item.killmail_id === kill.killmail_id)
        const target = persisted?.target_name ? ` · ${persisted.target_name}` : ''
        const ship = kill.ship_name || persisted?.ship_name || 'ship'
        const system = kill.solar_system_name || persisted?.solar_system_name || 'an unknown system'
        const alert = new Notification(`Tracked kill${target}`, {
            body: `${ship} destroyed in ${system}`,
            icon: kill.ship_type_id ? `/images/types/${kill.ship_type_id}/icon?size=64` : undefined,
            tag: `tracker-kill-${kill.killmail_id}`,
        })
        alert.onclick = () => {
            window.focus()
            void router.push(`/kill/${kill.killmail_id}`)
            alert.close()
        }
    }

    function configureRelay() {
        if (!import.meta.client || !isAuthenticated.value) {
            disposeRelay()
            return
        }
        const topics = [...new Set(trackers.value
            .filter(tracker => tracker.enabled && tracker.notifications_enabled)
            .flatMap(trackerTopic))].sort()
        const nextKey = topics.join(',')
        if (nextKey === relayTopicKey) return

        disposeRelay()
        if (!topics.length) return
        const wsUrl = config.public.wsUrl as string | undefined
        if (!wsUrl) return

        relayTopicKey = nextKey
        relay = createRelaySocket({
            wsUrl,
            path: '/killlist',
            onOpen: ws => ws.send(JSON.stringify({ action: 'subscribe', topics })),
            onMessage: async (channel, data) => {
                if (channel !== 'killlist' || !data?.killmail) return
                const kill = data.killmail as KilllistRow
                try { await refreshNotifications() } catch { /* refresh on next bell open */ }
                showBrowserAlert(kill)
            },
            reconnect: true,
            shouldReconnect: () => isAuthenticated.value && relayTopicKey === nextKey,
            // Intentionally remains connected in a background tab, but only
            // while this site is open. There is no service worker or Push API.
            visibilityPause: false,
        })
        relay.connect()
    }

    async function refreshTrackers() {
        if (!isAuthenticated.value || trackersLoading.value) return
        trackersLoading.value = true
        try {
            const result = await apiFetch<TrackerListResponse>('/api/me/trackers')
            trackers.value = result.trackers ?? []
            trackerLimit.value = Number(result.limit ?? 50)
            configureRelay()
        } finally {
            trackersLoading.value = false
        }
    }

    async function markAllRead() {
        if (!unreadCount.value) return
        notifications.value = notifications.value.map(item => ({ ...item, is_read: true }))
        trackers.value = trackers.value.map(tracker => ({ ...tracker, unread_notifications: 0 }))
        unreadCount.value = 0
        try {
            await apiFetch('/api/me/tracker-notifications/read', {
                method: 'POST',
                body: { all: true },
            })
        } catch {
            await refreshNotifications()
        }
    }

    async function setBrowserAlerts(enabled: boolean): Promise<boolean> {
        if (!import.meta.client) return false
        if (!('Notification' in window)) {
            browserPermission.value = 'unsupported'
            browserAlertsEnabled.value = false
            return false
        }
        let permission = Notification.permission
        if (enabled && permission === 'default') {
            permission = await Notification.requestPermission()
        }
        browserPermission.value = permission
        browserAlertsEnabled.value = enabled && permission === 'granted'
        try {
            localStorage.setItem(BROWSER_ALERTS_KEY, browserAlertsEnabled.value ? '1' : '0')
        } catch { /* storage may be unavailable */ }
        return browserAlertsEnabled.value
    }

    function reset() {
        trackers.value = []
        notifications.value = []
        unreadCount.value = 0
        seenKillIds.clear()
        disposeRelay()
        initialized = false
    }

    function init() {
        if (!import.meta.client || initialized || !isAuthenticated.value) return
        initialized = true
        if ('Notification' in window) {
            browserPermission.value = Notification.permission
            try {
                browserAlertsEnabled.value = localStorage.getItem(BROWSER_ALERTS_KEY) === '1'
                    && Notification.permission === 'granted'
            } catch { /* */ }
        } else {
            browserPermission.value = 'unsupported'
        }
        void Promise.allSettled([refreshTrackers(), refreshNotifications()])
    }

    if (import.meta.client && !authWatchStarted) {
        authWatchStarted = true
        watch(isAuthenticated, authenticated => {
            if (authenticated) init()
            else reset()
        }, { immediate: true })
    }

    return {
        trackers,
        trackerLimit,
        notifications,
        unreadCount,
        trackersLoading,
        notificationsLoading,
        browserAlertsEnabled,
        browserPermission,
        refreshTrackers,
        refreshNotifications,
        markAllRead,
        setBrowserAlerts,
        configureRelay,
        init,
        reset,
    }
}
