// Presentation is shared by the Nuxt bell and the Go browser UI.
import type { TrackerNotification } from '../models/trackers'

const slugPath=(prefix:string,slug:string|null|undefined,commentId:number)=>slug?`${prefix}${slug.replace(/^\/+/, '').split('/').map(encodeURIComponent).join('/')}#comment-${commentId}`:'#'
const TARGET_META: Record<number, { label: string; icon: string; color: string; pathFn: (id: number, commentId: number, slug?:string|null) => string }> = {
    1: { label: 'Killmail', icon: 'lucide:swords', color: 'text-red-400', pathFn: (id, c) => `/kill/${id}#comment-${c}` },
    2: { label: 'Character', icon: 'lucide:user', color: 'text-blue-400', pathFn: (id, c) => `/character/${id}#comment-${c}` },
    3: { label: 'Corp', icon: 'lucide:building', color: 'text-amber-400', pathFn: (id, c) => `/corporation/${id}#comment-${c}` },
    4: { label: 'Alliance', icon: 'lucide:flag', color: 'text-purple-400', pathFn: (id, c) => `/alliance/${id}#comment-${c}` },
    5: { label: 'System', icon: 'lucide:globe', color: 'text-emerald-400', pathFn: (id, c) => `/system/${id}#comment-${c}` },
    6: { label:'Page',icon:'lucide:link',color:'text-gray-400',pathFn:(_id,c,slug)=>slugPath('/',slug,c) },
    7: { label: 'Battle', icon: 'lucide:shield', color: 'text-orange-400', pathFn: (id, c, slug) => `/battle/${slug?encodeURIComponent(slug):id}/comments#comment-${c}` },
    8: { label:'Fit',icon:'lucide:wrench',color:'text-blue-400',pathFn:(_id,c,slug)=>slugPath('/fit/',slug,c) },
    9: { label:'Blog',icon:'lucide:file-text',color:'text-blue-400',pathFn:(_id,c,slug)=>slugPath('/blog/',slug,c) },
    10: { label:'Campaign',icon:'lucide:flag',color:'text-emerald-400',pathFn:(_id,c,slug)=>slug?`/campaign/${encodeURIComponent(slug)}/comments#comment-${c}`:'#' },
}

export function targetFor(targetType: number) {
    return TARGET_META[targetType] ?? { label: 'Page', icon: 'lucide:link', color: 'text-gray-400', pathFn: () => '#' }
}

export function timeAgo(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime()
    if (diff < 60_000) return 'just now'
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m`
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h`
    if (diff < 30 * 86_400_000) return `${Math.floor(diff / 86_400_000)}d`
    return new Date(iso).toLocaleDateString()
}

export function snippet(html: string, max = 120): string {
    const txt = html.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()
    return txt.length > max ? `${txt.slice(0, max - 1)}…` : txt
}

export function trackerSummary(item: TrackerNotification): string {
    const ship = item.ship_name || 'Ship'
    const victim = item.victim_character_name ? ` piloted by ${item.victim_character_name}` : ''
    const system = item.solar_system_name ? ` in ${item.solar_system_name}` : ''
    return `${ship}${victim}${system}`
}

export function trackerRole(item: TrackerNotification): string {
    if (item.match_role === 'victim') return 'loss'
    if (item.match_role === 'attacker') return 'kill'
    if (item.match_role === 'both') return 'both sides'
    return 'location'
}

export function trackerTopics(trackers: Array<{enabled:boolean;notifications_enabled:boolean;target_type:string;target_id:number}>): string[] {
    return [...new Set(trackers.filter(t=>t.enabled&&t.notifications_enabled).flatMap(t=>['character','corporation','alliance'].includes(t.target_type)?[`victim.${t.target_id}`,`attacker.${t.target_id}`]:[`${t.target_type}.${t.target_id}`]))].sort()
}
