/**
 * Format a number in compact form: 1.5k, 2.3m, 1.05b, 3.20t
 * Works for ISK values, volumes, counts, etc.
 */
export function formatCompact(v: number): string {
    if (v == null || Number.isNaN(v)) return '0'
    if (v >= 1e12) return `${(v / 1e12).toFixed(2)}t`
    if (v >= 1e9) return `${(v / 1e9).toFixed(2)}b`
    if (v >= 1e6) return `${(v / 1e6).toFixed(2)}m`
    if (v >= 1e3) return `${(v / 1e3).toFixed(1)}k`
    return v.toFixed(0)
}

/** Format ISK values in compact form. Alias for formatCompact. */
export const formatIsk = formatCompact

/** Full number with thousands separators: 1,234,567 */
export function formatNumber(v: number | null | undefined): string {
    return v?.toLocaleString('en-US') ?? '0'
}

/** Site-standard date: "Jan 5, 2026" */
export function formatDate(iso: string | null | undefined, fallback = ''): string {
    if (!iso) return fallback
    return new Date(iso).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}

/** Site-standard date+time: "Jan 5, 2026, 14:30" */
export function formatDateTime(iso: string | null | undefined, fallback = ''): string {
    if (!iso) return fallback
    return new Date(iso).toLocaleString('en-US', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

/** Whole years elapsed since the given date, or null if unknown. */
export function entityAgeYears(iso: string | null | undefined): number | null {
    if (!iso) return null
    const born = new Date(iso)
    if (Number.isNaN(born.getTime())) return null
    const now = new Date()
    let age = now.getFullYear() - born.getFullYear()
    const m = now.getMonth() - born.getMonth()
    if (m < 0 || (m === 0 && now.getDate() < born.getDate())) age--
    return age
}
