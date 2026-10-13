import { eveInputToIso, isoToEveInput } from './eveTime'

export interface AdminAnnouncement {
    id: number
    tier: number
    title: string
    body_md: string
    body_html: string
    color: string
    icon: string | null
    link_url: string | null
    link_label: string | null
    starts_at: string
    expires_at: string
    created_by: number
    created_at: string
    updated_at: string
    archived_at: string | null
}

export const tickerDurations = [
    { value: 5, label: '5 min' }, { value: 10, label: '10 min' },
    { value: 15, label: '15 min' }, { value: 30, label: '30 min' },
    { value: 60, label: '1 hour' },
] as const
export const tierLabels: Record<number, string> = { 1: 'Ticker', 2: 'Banner', 3: 'Modal' }
export const tierColors: Record<number, string> = { 1: 'text-blue-400', 2: 'text-yellow-400', 3: 'text-red-400' }

export function announcementForm(a?: AdminAnnouncement, now = new Date()) {
    const remaining = a ? Math.max(5, Math.round((new Date(a.expires_at).getTime() - now.getTime()) / 60000)) : 15
    return {
        tier: a?.tier ?? 2, title: a?.title ?? '', body_md: a?.body_md ?? '',
        color: a?.color ?? 'info', icon: a?.icon ?? '', link_url: a?.link_url ?? '', link_label: a?.link_label ?? '',
        starts_at: isoToEveInput(a?.starts_at ?? now),
        expires_at: isoToEveInput(a?.expires_at ?? new Date(now.getTime() + 7 * 86400000)),
        duration: a?.tier === 1 ? tickerDurations.find(d => d.value >= remaining)?.value ?? 60 : 15,
    }
}

export function announcementPayload(form: ReturnType<typeof announcementForm>, now = new Date()) {
    return {
        tier: form.tier, title: form.title, body_md: form.body_md, color: form.color,
        starts_at: form.tier === 1 ? now.toISOString() : eveInputToIso(form.starts_at),
        expires_at: form.tier === 1 ? new Date(now.getTime() + form.duration * 60000).toISOString() : eveInputToIso(form.expires_at),
        icon: form.icon || null, link_url: form.link_url || null, link_label: form.link_label || null,
    }
}

export function annStatusLabel(a: Pick<AdminAnnouncement, 'archived_at' | 'starts_at' | 'expires_at'>, now = Date.now()) {
    if (a.archived_at) return { label: 'Archived', cls: 'bg-gray-500/20 text-gray-400 border-gray-500/30' }
    if (now < new Date(a.starts_at).getTime()) return { label: 'Scheduled', cls: 'bg-blue-500/20 text-blue-300 border-blue-500/30' }
    if (now > new Date(a.expires_at).getTime()) return { label: 'Expired', cls: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' }
    return { label: 'Active', cls: 'bg-green-500/20 text-green-300 border-green-500/30' }
}
