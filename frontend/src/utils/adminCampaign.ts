export const adminCampaignStates = [
    { value: 'all', label: 'All campaigns' },
    { value: 'paused', label: 'Paused' },
    { value: 'failed', label: 'Failed' },
    { value: 'pending', label: 'Pending' },
    { value: 'active', label: 'Active' },
    { value: 'archived', label: 'Archived' },
]

export function adminCampaignStatus(campaign: { processing_paused?: boolean; status: number | string }) {
    if (campaign.processing_paused) return { label: 'Paused', className: 'text-red-400 bg-red-500/10' }
    if (Number(campaign.status) === 0) return { label: 'Pending', className: 'text-yellow-400 bg-yellow-500/10' }
    if (Number(campaign.status) === 2) return { label: 'Archived', className: 'text-gray-500 bg-white/[0.04]' }
    return { label: 'Active', className: 'text-green-400 bg-green-500/10' }
}

export function formatCampaignRuntime(milliseconds: number | null | undefined) {
    if (!milliseconds) return '—'
    if (milliseconds < 1000) return `${milliseconds}ms`
    if (milliseconds < 60_000) return `${(milliseconds / 1000).toFixed(1)}s`
    return `${(milliseconds / 60_000).toFixed(1)}m`
}

export function adminCampaignProcessed(campaign: { last_processing_killmails?: number | null; totals?: { killCount?: number } | null }) {
    return campaign.last_processing_killmails ?? campaign.totals?.killCount ?? 0
}
