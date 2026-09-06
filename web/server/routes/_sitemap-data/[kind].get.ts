import { isSitemapKind, loadSitemapSource } from '../../utils/sitemapSource'

export default defineEventHandler(async (event) => {
    const kind = getRouterParam(event, 'kind') || ''
    if (!isSitemapKind(kind)) {
        throw createError({ statusCode: 404, statusMessage: 'Sitemap category not found' })
    }
    return loadSitemapSource(kind, useRuntimeConfig(event))
})
