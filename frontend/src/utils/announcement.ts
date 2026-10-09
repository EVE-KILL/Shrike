import type { Announcement } from '#shared/api'

export const ANNOUNCEMENT_DISMISSED_KEY = 'ek_dismissed_announcements'
export const ANNOUNCEMENT_COLLAPSED_KEY = 'ek_ticker_collapsed'
export const announcementNewestFirst = (a: Announcement, b: Announcement) =>
    new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
export const announcementSort = (a: Announcement, b: Announcement) =>
    b.tier - a.tier || announcementNewestFirst(a, b)

export function announcementActive(a: Announcement, now = Date.now()) {
    return new Date(a.starts_at).getTime() <= now && new Date(a.expires_at).getTime() > now
}
export function announcementCategory(id: number): 'war' | 'battle' | 'kill' {
    return id <= -3_000_000_000 ? 'war' : id <= -2_000_000_000 ? 'battle' : 'kill'
}
export function announcementLists(items: readonly Announcement[], dismissed: ReadonlySet<number>, now = Date.now()) {
    const active = items.filter(a => !dismissed.has(a.id) && announcementActive(a, now))
    const buckets: Record<string, Announcement[]> = { war: [], battle: [], kill: [] }
    for (const a of active) if (a.tier === 1) buckets[announcementCategory(a.id)]!.push(a)
    const limits: Record<string, number> = { war: 4, battle: 5, kill: 8 }
    return {
        ticker: Object.entries(buckets).flatMap(([key, rows]) => rows.sort(announcementNewestFirst).slice(0, limits[key])).sort(announcementNewestFirst),
        banners: active.filter(a => a.tier === 2),
        modals: active.filter(a => a.tier === 3),
    }
}
export type AnnouncementEvent =
    | {event_type:'expired'|'archived';announcement:Pick<Announcement,'id'>}
    | {event_type:'new'|'updated';announcement:Announcement}
    | null
export function applyAnnouncementEvent(items: readonly Announcement[], channel: string, data: AnnouncementEvent) {
    if (channel !== 'announcements' || !data?.announcement?.id) return items
    if (data.event_type === 'expired' || data.event_type === 'archived') return items.filter(row => row.id !== data.announcement.id)
    if (data.event_type !== 'new' && data.event_type !== 'updated') return items
    const a = data.announcement
    const index = items.findIndex(row => row.id === a.id)
    if (index < 0) return [...items, a].sort(announcementSort)
    const result = [...items]; result[index] = a
    return result
}
export function reconcileAnnouncementSnapshot(snapshot: readonly Announcement[], events: readonly [string, AnnouncementEvent][]) {
    return events.reduce((items, [channel, data]) => applyAnnouncementEvent(items, channel, data), snapshot)
}
export function tickerFreshness(item: { expires_at?: string | Date | null }, now = Date.now()) {
    if (!item.expires_at) return 1
    const remaining = new Date(item.expires_at).getTime() - now
    if (!Number.isFinite(remaining) || remaining >= 600_000) return 1
    return remaining <= 0 ? 0.35 : 0.35 + 0.65 * remaining / 600_000
}
export const announcementBannerColors: Record<string, { bg: string; border: string; text: string; icon: string }> = {
    info: { bg: 'bg-blue-500/10', border: 'border-blue-400/30', text: 'text-blue-300', icon: 'lucide:info' },
    warning: { bg: 'bg-amber-500/10', border: 'border-amber-400/30', text: 'text-amber-300', icon: 'lucide:alert-triangle' },
    danger: { bg: 'bg-red-500/10', border: 'border-red-400/30', text: 'text-red-300', icon: 'lucide:alert-circle' },
    success: { bg: 'bg-green-500/10', border: 'border-green-400/30', text: 'text-green-300', icon: 'lucide:check-circle' },
}
export const announcementModalColors: Record<string, { accent: string; button: string }> = {
    info: { accent: 'text-blue-400', button: 'bg-blue-600 hover:bg-blue-500' },
    warning: { accent: 'text-amber-400', button: 'bg-amber-600 hover:bg-amber-500' },
    danger: { accent: 'text-red-400', button: 'bg-red-600 hover:bg-red-500' },
    success: { accent: 'text-green-400', button: 'bg-green-600 hover:bg-green-500' },
}
export const announcementTickerColors: Record<string, { text: string; bg: string }> = {
    info: { text: 'text-blue-300', bg: 'bg-blue-500/15 border-blue-500/25' },
    warning: { text: 'text-amber-300', bg: 'bg-amber-500/15 border-amber-500/25' },
    danger: { text: 'text-red-300', bg: 'bg-red-500/15 border-red-500/25' },
    success: { text: 'text-green-300', bg: 'bg-green-500/15 border-green-500/25' },
}
