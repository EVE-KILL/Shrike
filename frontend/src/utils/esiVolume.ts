import { fmtHour } from './adminFormat'

export interface EsiVolumeHour {
    hour: string
    total: number
    errors: number
    new_items?: number
}
export const ESI_CHART_HEIGHT = 96
const HOUR_MS = 3_600_000

// Shared by the Vue chart and the Go frontend's browser component. Keep the
// payload-anchored 24-hour geometry and PostgreSQL/Safari date normalization.
export function esiVolumeSlots(hours: readonly EsiVolumeHour[], showNewItems = false) {
    const byHour = new Map<number, EsiVolumeHour>()
    for (const h of hours) {
        const iso = h.hour.replace(' ', 'T').replace(/([+-]\d{2})$/, '$1:00')
        const parsed = new Date(iso).getTime()
        const t = Number.isNaN(parsed) ? new Date(h.hour).getTime() : parsed
        if (!Number.isNaN(t)) byHour.set(Math.floor(t / HOUR_MS) * HOUR_MS, h)
    }
    if (!byHour.size) return []
    const latest = Math.max(...byHour.keys())
    const peak = Math.max(1, ...hours.map(h => h.total ?? 0))
    return Array.from({ length: 24 }, (_, i) => {
        const at = latest - (23 - i) * HOUR_MS
        const rec = byHour.get(at)
        const errors = rec?.errors ?? 0
        const ok = Math.max(0, (rec?.total ?? 0) - errors)
        return {
            at, empty: !rec, errors,
            errorHeight: Math.max(2, (errors / peak) * ESI_CHART_HEIGHT),
            okHeight: Math.max(2, (ok / peak) * ESI_CHART_HEIGHT),
            tooltip: rec
                ? (showNewItems
                    ? `${fmtHour(new Date(at).toISOString())}: ${rec.total} requests, ${errors} errors, ${rec.new_items ?? 0} new items`
                    : `${fmtHour(new Date(at).toISOString())}: ${rec.total} requests, ${errors} errors`)
                : `${String(new Date(at).getUTCHours()).padStart(2, '0')}:00 — no requests logged`,
        }
    })
}
