/**
 * `getCachedData` implementation for useApiFetch/useAsyncData.
 *
 * Nuxt refetches on every client-side navigation by default, so returning to a
 * page you already loaded re-requests data that is sitting in the payload.
 * Handing this to `getCachedData` reuses the SSR payload (and any statically
 * rendered data) instead.
 *
 * Was copy-pasted inline at seventeen call sites before it lived here — one
 * shared implementation, so the behaviour can't drift between pages.
 *
 * Note this caches for the lifetime of the payload entry, with no staleness
 * window: a route the user returns to shows what it showed the first time
 * until a hard reload. That is the intent for entity dashboards and top lists,
 * which is where every current caller sits. Anything needing revalidation
 * should pass its own getCachedData rather than reaching for this.
 */
export function cachedPayload(key: string) {
    const nuxtApp = useNuxtApp()
    return nuxtApp.payload.data[key] || nuxtApp.static.data[key]
}
