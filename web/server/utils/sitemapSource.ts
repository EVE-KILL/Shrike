import { createServerApiClient } from '../../shared/utils/serverApi'

export const sitemapKinds = [
    'kills', 'characters', 'coalitions', 'corporations', 'alliances',
    'systems', 'regions', 'ships', 'items', 'wars', 'battles',
] as const

export function isSitemapKind(kind: string): boolean {
    return (sitemapKinds as readonly string[]).includes(kind)
}

export async function loadSitemapSource(kind: string, config: { apiSocket: string, apiOrigin: string }) {
    if (!isSitemapKind(kind)) throw new Error('Unknown sitemap category')
    const api = createServerApiClient({
        socket: config.apiSocket,
        origin: config.apiOrigin,
        headers: { host: 'eve-kill.com', 'x-forwarded-host': 'eve-kill.com' },
    })
    return api<Array<{ loc: string, lastmod?: string, changefreq?: string, priority?: number }>>(
        `/api/__sitemap__/${kind}`,
    )
}
