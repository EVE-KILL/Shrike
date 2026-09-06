import { isIP } from 'node:net'

export function seoHostname(host: string): string {
    try { return new URL(`http://${host}`).hostname.toLowerCase().replace(/\.$/, '') }
    catch { return '' }
}

export function isCanonicalSeoHost(host: string): boolean {
    return seoHostname(host) === 'eve-kill.com'
}

export function canServeSitemap(host: string): boolean {
    const name = seoHostname(host)
    // Preserve local rendering and integration checks without advertising
    // duplicate sitemap indexes on tenant or unknown public hosts.
    return name === 'eve-kill.com' || name === 'localhost'
        || isIP(name.replace(/^\[|\]$/g, '')) !== 0
}

export function isSitemapPath(path: string): boolean {
    return /^\/sitemap(?:_index)?\.xml\/?$/.test(path)
        || path.startsWith('/__sitemap__/') || path.startsWith('/_sitemap-data/')
}
