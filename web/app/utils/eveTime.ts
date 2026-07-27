/**
 * EVE time (UTC) helpers for wall-clock datetime strings.
 *
 * The site speaks EVE time everywhere. The one exception is relative "x ago"
 * stamps, which are differences between two instants and so carry no timezone
 * at all.
 *
 * A wall-clock string ("2026-07-21T13:00") has no zone attached, and both
 * `new Date(value)` and `Date.prototype.getHours()` silently apply the
 * RUNTIME's zone to it. That turns a field labelled "EVE time" into local
 * time — a viewer at UTC+2 asking for 13:00 EVE actually queries 11:00 EVE,
 * and gets a plausible-looking empty result instead of an error.
 *
 * DateTimePicker emits exactly this shape, so route every conversion into or
 * out of a picker value through here.
 */

/** True when a datetime string already carries an explicit zone. */
const hasZone = (value: string): boolean => /([Zz]|[+-]\d{2}:?\d{2})$/.test(value)

/** Parse a wall-clock value as EVE time. Missing/invalid input → Invalid Date. */
export function eveInputToDate(value: string | null | undefined): Date {
    if (!value) return new Date(Number.NaN)
    return new Date(hasZone(value) ? value : `${value}Z`)
}

/** Parse a wall-clock value as EVE time and return an ISO instant. */
export function eveInputToIso(value: string | null | undefined): string {
    return eveInputToDate(value).toISOString()
}

/** Render an instant as an EVE wall-clock value. */
export function isoToEveInput(value: string | Date | null | undefined): string {
    if (!value) return ''
    const date = value instanceof Date ? value : new Date(value)
    if (Number.isNaN(date.getTime())) return ''
    return date.toISOString().slice(0, 16)
}

const pad = (n: number) => String(n).padStart(2, '0')

/**
 * `HH:mm` in EVE time, 24-hour.
 *
 * Built from UTC parts rather than toLocaleTimeString: 'en-US' renders
 * "02:30 PM", and the browser's own locale would render differently on the
 * server than on the client, which is a hydration mismatch waiting to happen.
 * EVE runs on a 24-hour clock, so this is what people expect anyway.
 */
export function formatEveTime(value: string | Date | null | undefined): string {
    if (!value) return ''
    const date = value instanceof Date ? value : new Date(value)
    if (Number.isNaN(date.getTime())) return ''
    return `${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}`
}

/** `HH:mm:ss` in EVE time, 24-hour. */
export function formatEveTimeSeconds(value: string | Date | null | undefined): string {
    if (!value) return ''
    const date = value instanceof Date ? value : new Date(value)
    if (Number.isNaN(date.getTime())) return ''
    return `${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}:${pad(date.getUTCSeconds())}`
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

/**
 * `28 May 2026` in EVE time.
 *
 * The month name is looked up from a table rather than via toLocaleDateString,
 * for the same reason as formatEveTime: the browser's locale would render
 * "28.5.2026" for some visitors and "5/28/2026" for others, and neither would
 * match what the server rendered.
 */
export function formatEveDate(value: string | Date | null | undefined): string {
    if (!value) return ''
    const date = value instanceof Date ? value : new Date(value)
    if (Number.isNaN(date.getTime())) return ''
    return `${date.getUTCDate()} ${MONTHS[date.getUTCMonth()]} ${date.getUTCFullYear()}`
}

/** `28 May 2026, 14:30` in EVE time. Pass `withSuffix` to append " EVE". */
export function formatEveDateTime(value: string | Date | null | undefined, withSuffix = false): string {
    if (!value) return ''
    const date = value instanceof Date ? value : new Date(value)
    if (Number.isNaN(date.getTime())) return ''
    return `${formatEveDate(date)}, ${formatEveTime(date)}${withSuffix ? ' EVE' : ''}`
}
