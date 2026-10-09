export interface AdminDomainSummary {
    id: number
    subdomain: string
    site_name?: string | null
    owner_name?: string | null
    custom_hostname?: string | null
    active: boolean
    pending_assets?: number | null
}
export function adminDomainCounts(domains: readonly AdminDomainSummary[]) {
    return { all: domains.length, pending: domains.filter(d => (d.pending_assets ?? 0) > 0).length, inactive: domains.filter(d => !d.active).length }
}
export function filterAdminDomains<T extends AdminDomainSummary>(domains: readonly T[], search: string, filter: 'all' | 'pending' | 'inactive') {
    const q = search.trim().toLowerCase()
    return domains.filter(d => {
        if (filter === 'pending' && !((d.pending_assets ?? 0) > 0)) return false
        if (filter === 'inactive' && d.active) return false
        return !q || [d.subdomain, d.site_name, d.owner_name, d.custom_hostname].some(v => v?.toLowerCase().includes(q))
    })
}
export function domainAssetStatusColor(status: string) {
    return status === 'approved' ? 'text-green-400 bg-green-500/20' : status === 'rejected' ? 'text-red-400 bg-red-500/20' : 'text-yellow-400 bg-yellow-500/20'
}
export function formatDomainAssetSize(bytes: number) {
    return bytes < 1024 ? `${bytes} B` : bytes < 1048576 ? `${(bytes / 1024).toFixed(1)} KB` : `${(bytes / 1048576).toFixed(1)} MB`
}
