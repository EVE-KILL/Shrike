export type TimeDisplayZone = 'eve' | 'local'

const COOKIE = 'ek_tz'

/**
 * Which clock date/time pickers render in. EVE time (UTC) is the default and
 * the site's native language — "local" is a convenience for people who would
 * rather think in their own wall clock.
 *
 * The choice only ever changes what is DISPLAYED. Every picker still stores
 * and emits an EVE wall-clock value, so switching zones never moves the
 * instant a user picked.
 *
 * Backed by useState so every picker on the page flips together, mirrored to
 * a cookie so the choice survives reloads (same pattern as ek_theme).
 */
export function useTimeZonePref() {
    const cookie = useCookie<TimeDisplayZone>(COOKIE, {
        default: () => 'eve',
        maxAge: 60 * 60 * 24 * 365,
        sameSite: 'lax',
        path: '/',
    })

    const zone = useState<TimeDisplayZone>(COOKIE, () => (cookie.value === 'local' ? 'local' : 'eve'))

    watch(zone, (value) => { cookie.value = value })

    return zone
}

/** IANA name of the viewer's zone, for labelling. Empty during SSR. */
export function localZoneLabel(): string {
    try {
        return Intl.DateTimeFormat().resolvedOptions().timeZone || 'Local'
    } catch {
        return 'Local'
    }
}
