import { myCommentTarget } from './myCommentTarget'

export interface AdminCommentTarget { id: number; target_type: number; target_id: number; target_slug: string | null }
export interface AdminModerationTarget {
    target_kind: number; target_id: number
    comment_context: { target_type: number; target_id: number; target_slug: string | null } | null
}

export function adminCommentTarget(c: AdminCommentTarget) {
    const canonical = myCommentTarget(c)
    let label = 'Page'
    switch (c.target_type) {
        case 1: label = `Kill #${c.target_id}`; break
        case 2: label = `Char ${c.target_id}`; break
        case 3: label = `Corp ${c.target_id}`; break
        case 4: label = `Alli ${c.target_id}`; break
        case 5: label = `Sys ${c.target_id}`; break
        case 7: label = `Battle ${c.target_slug || c.target_id}`; break
        case 8: label = `Fit ${c.target_slug ?? '?'}`; break
        case 9: label = 'Blog post'; break
        case 10: label = 'Campaign'; break
    }
    const href = canonical.href !== '#' && [6, 8].includes(c.target_type)
        ? `${canonical.href}#comment-${c.id}` : canonical.href
    return { label, href }
}

export function moderationKindLabel(kind: number) {
    return ['Comment', 'Character bio', 'Corporation bio', 'Alliance bio'][kind] ?? 'Unknown'
}
export function moderationTargetHref(item: AdminModerationTarget) {
    if (item.target_kind === 0) {
        if (!item.comment_context) return '/comments'
        const target = adminCommentTarget({ id: item.target_id, ...item.comment_context })
        return target.href === '#' ? '/comments' : target.href
    }
    const kind = ['', 'character', 'corporation', 'alliance'][item.target_kind]
    return kind ? `/${kind}/${item.target_id}` : '#'
}
export function moderationTargetIdentity(item: AdminModerationTarget) {
    return item.target_kind === 0 && item.comment_context
        ? item.comment_context.target_slug || String(item.comment_context.target_id) : String(item.target_id)
}
export function moderationStatusLabel(status: number) {
    const states = [
        { label: 'Pending', cls: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' },
        { label: 'Approved', cls: 'bg-green-500/20 text-green-300 border-green-500/30' },
        { label: 'Rejected', cls: 'bg-red-500/20 text-red-300 border-red-500/30' },
        { label: 'Auto-approved', cls: 'bg-green-500/10 text-green-400/70 border-green-500/20' },
        { label: 'Auto-rejected', cls: 'bg-red-500/10 text-red-400/70 border-red-500/20' },
    ]
    return states[status] ?? { label: String(status), cls: 'bg-white/[0.04] text-gray-400' }
}
export function adminCommentStatus(status: number) {
    return [
        { label: 'OK', cls: 'text-gray-400' }, { label: 'Pending', cls: 'text-yellow-400' },
        { label: 'AI flagged', cls: 'text-yellow-400' }, { label: 'Hidden', cls: 'text-red-400' },
    ][status] ?? { label: '?', cls: 'text-gray-500' }
}
