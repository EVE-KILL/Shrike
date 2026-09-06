import { canServeSitemap, isCanonicalSeoHost, isSitemapPath } from '../utils/seoHost'

export default defineEventHandler((event) => {
    const host = getRequestHost(event)
    const path = getRequestURL(event).pathname
    if (!canServeSitemap(host) && isSitemapPath(path)) {
        throw createError({ statusCode: 404, statusMessage: 'Sitemap not available on this host' })
    }
    if (!isCanonicalSeoHost(host) && path === '/robots.txt') {
        setResponseHeader(event, 'content-type', 'text/plain; charset=utf-8')
        // Crawlers must be able to fetch pages to observe noindex or 404.
        return 'User-agent: *\nAllow: /\nDisallow: /api/\n'
    }
})
