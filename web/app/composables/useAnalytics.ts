import { isbot } from 'isbot'

// Allowed event-name domains. Keeps the umami events table low-cardinality
// and makes track() typo-loud during dev.
const ALLOWED_DOMAINS = ['search', 'filter', 'tab', 'auth', 'comment', 'theme'] as const
type Domain = typeof ALLOWED_DOMAINS[number]
type EventName = `${Domain}.${string}`

type DataValue = string | number | boolean
type EventData = Record<string, DataValue>

const NAME_RE = /^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$/

export function useAnalytics() {
    const track = (name: EventName, data?: EventData) => {
        if (import.meta.server) return
        if (!NAME_RE.test(name)) {
            if (import.meta.dev) console.warn('[analytics] bad event name:', name)
            return
        }
        const domain = name.split('.', 1)[0] as Domain
        if (!ALLOWED_DOMAINS.includes(domain)) {
            if (import.meta.dev) console.warn('[analytics] unknown domain:', domain)
            return
        }
        if (typeof navigator !== 'undefined' && isbot(navigator.userAgent)) return

        // umTrackEvent is auto-imported by nuxt-umami; fire-and-forget.
        umTrackEvent(name, data).catch(() => {})
    }

    return { track }
}
