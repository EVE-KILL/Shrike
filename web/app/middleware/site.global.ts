import type { SiteConfigurationResponse } from '#shared/api'

/**
 * Resolve the current Host before page/layout setup.
 *
 * SSR calls Go over the private transport and serializes the resulting state
 * into the Nuxt payload. Hydration and later client-side navigation reuse that
 * state, so the browser does not issue a duplicate bootstrap request.
 */
export default defineNuxtRouteMiddleware(async () => {
    const site = useState<SiteConfigurationResponse | null>(
        'site-configuration',
        () => null,
    )
    if (site.value === null) {
        try {
            site.value = await apiFetch<SiteConfigurationResponse>('/api/site')
        } catch (cause) {
            throw createError({
                statusCode: 503,
                statusMessage: 'Unable to load site configuration',
                cause,
            })
        }
    }
    if (site.value.isDomainHost && !site.value.domain) {
        throw createError({ statusCode: 404, statusMessage: 'Killboard not found' })
    }
})
