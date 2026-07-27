/**
 * Table formatters for the admin + settings dashboards. Distinct from
 * formatters.ts on purpose: these render '—' for missing values (table
 * cells) and use compact machine-style datetimes rather than the
 * reader-style dates the public site uses.
 *
 * Like the rest of the site these read in EVE time (UTC), not the viewer's
 * zone — an admin comparing a queue timestamp against a killmail should not
 * have to apply an offset in their head. `timeSince` is the exception, and
 * only because a difference between two instants has no timezone.
 */

export const fmt = (n: number | null | undefined): string => {
    if (n == null) return '—'
    return n.toLocaleString('en-US')
}

/** `2026-07-21 18:45` EVE time, '—' when missing. */
export const fmtDate = (d: string | null | undefined): string => {
    if (!d) return '—'
    const dt = new Date(d)
    if (Number.isNaN(dt.getTime())) return '—'
    return `${dt.toISOString().slice(0, 10)} ${dt.toISOString().slice(11, 16)}`
}

export const fmtHour = (d: string | null | undefined): string => {
    if (!d) return '—'
    const dt = new Date(d)
    if (Number.isNaN(dt.getTime())) return '—'
    return `${String(dt.getUTCHours()).padStart(2, '0')}:${String(dt.getUTCMinutes()).padStart(2, '0')}`
}

export const fmtDateShort = (d: string | null | undefined): string => {
    if (!d) return '—'
    return new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' })
}

export const timeSince = (d: string | null | undefined): string => {
    if (!d) return '—'
    const diff = Date.now() - new Date(d).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 60) return `${mins}m ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours}h ago`
    return `${Math.floor(hours / 24)}d ago`
}
