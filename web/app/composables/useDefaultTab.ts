/**
 * Redirects to the user's preferred default tab for an entity page,
 * but only when landing on the base URL (no tab param in route).
 *
 * Priority:
 * 1. Explicit tab in URL (no redirect)
 * 2. User preference from ek_prefs cookie
 * 3. Fallback default
 */

export function useDefaultTab(
    entityType: string,
    basePath: string,
    fallback: string,
    validTabs: Set<string>,
) {
    const route = useRoute()
    const param = route.params.tab as string

    // If URL has the default tab explicitly (e.g. /character/123/dashboard),
    // redirect to the clean base URL to avoid duplicate content
    if (param && param === fallback) {
        navigateTo(basePath, { replace: true })
        return
    }

    // Explicit tab in URL — respect it, provided it names a real tab. Unknown
    // segments used to render the fallback tab at a 200 whose canonical points
    // back at basePath, so /system/30000142/<anything> was an unbounded set of
    // URLs collapsing onto a single canonical.
    if (param) {
        if (!validTabs.has(param)) {
            throw createError({ statusCode: 404, statusMessage: 'Tab not found' })
        }
        return
    }

    // User preference redirect is CLIENT-ONLY. During SSR the fallback tab
    // renders, which keeps the HTML identical for all visitors of a given
    // URL — required for Cloudflare edge caching. If the redirect ran
    // during SSR, CF would cache User A's preferred-tab 302 and serve it
    // to every subsequent visitor.
    if (import.meta.client) {
        const prefs = useCookie<Record<string, Record<string, string>> | null>('ek_prefs')
        const preferred = prefs.value?.defaultTabs?.[entityType]

        if (preferred && preferred !== fallback && validTabs.has(preferred)) {
            navigateTo(`${basePath}/${preferred}`, { replace: true })
        }
    }
}
