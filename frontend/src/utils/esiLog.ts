/**
 * Shared vocabulary for ESI request-log rows.
 *
 * The admin ESI monitor and the user-facing settings ESI tab render the same
 * rows. Before this, each carried its own inline ternaries — the status dot in
 * six places and the endpoint-type chip in four — which is how the two ended up
 * disagreeing about whether a 304 counts as success in some spots.
 */

import type {AdminEsiLogsResponses} from '../../shared/api/types.gen'

// The same generated response row owns account and admin rendering fields.
export type EsiLogRow=AdminEsiLogsResponses[200]['rows'][number]

/**
 * A 304 means "you already have it" — the request did what it was asked to, so
 * it reads as success everywhere rather than as a failure.
 */
export const esiRowOk = (row: Pick<EsiLogRow, 'success' | 'status_code'>): boolean =>
    row.success || row.status_code === 304

/** Green for OK, red for a server fault, yellow for anything else (4xx, aborted). */
export const esiStatusDotClass = (row: Pick<EsiLogRow, 'success' | 'status_code'>): string => {
    if (esiRowOk(row)) return 'bg-green-400'
    return (row.status_code ?? 0) >= 500 ? 'bg-red-400' : 'bg-yellow-400'
}

/** Same three states, as a filled badge for the detail modal's status code. */
export const esiStatusBadgeClass = (row: Pick<EsiLogRow, 'success' | 'status_code'> | null | undefined): string => {
    if (!row) return 'bg-white/[0.06] text-gray-400'
    if (esiRowOk(row)) return 'bg-green-500/15 text-green-400'
    return (row.status_code ?? 0) >= 500 ? 'bg-red-500/20 text-red-400' : 'bg-yellow-500/20 text-yellow-400'
}

export const esiTypeChipClass = (type: string | null | undefined): string => {
    if (type === 'corporation') return 'bg-blue-500/10 text-blue-400'
    if (type === 'character') return 'bg-emerald-500/10 text-emerald-400'
    return 'bg-gray-500/10 text-gray-400'
}

/** Tint the whole row when a request actually failed. */
export const esiRowTintClass = (row: Pick<EsiLogRow, 'success' | 'status_code'>): string =>
    esiRowOk(row) ? '' : 'bg-red-500/[0.03]'

// useFetch returns a shallow ref by default. Replace the response object when
// applying a live batch, and count only IDs that were not already visible.
export function mergeEsiLogRows<T extends { id: number }>(data: { rows: T[]; total: number }, fresh: readonly T[], limit = 50) {
    const seen = new Set(data.rows.map(row => row.id))
    const added = fresh.filter(row => {
        if (seen.has(row.id)) return false
        seen.add(row.id)
        return true
    })
    return { rows: [...added, ...data.rows].slice(0, limit), total: data.total + added.length }
}
