import type { FetchOptions } from 'ofetch'
import { createServerApiClient } from '#shared/utils/serverApi'

const forwardedHeaderNames = [
    'cookie',
    'host',
    'user-agent',
    'cf-connecting-ip',
    'x-forwarded-for',
    'x-forwarded-host',
    'x-forwarded-proto',
] as const

/**
 * Return the one HTTP transport used by the frontend.
 *
 * Browser requests remain relative and therefore go straight to the
 * same-origin Go /api and /auth handlers. During SSR, a relative Nuxt fetch
 * would resolve against Nitro and accidentally re-enter the old server/api
 * tree. Instead, SSR uses the private loopback origin and forwards only the
 * request identity needed for sessions, tenant domains, redirects, logging,
 * and per-client rate limits.
 */
export function useApiClient(): typeof $fetch {
    if (import.meta.client) {
        return $fetch
    }

    const config = useRuntimeConfig()
    const incoming = useRequestHeaders([...forwardedHeaderNames])
    const headers = new Headers()
    for (const name of forwardedHeaderNames) {
        const value = incoming[name]
        if (value) headers.set(name, value)
    }

    const host = incoming.host
    if (host && !headers.has('x-forwarded-host')) {
        headers.set('x-forwarded-host', host)
    }
    if (!headers.has('user-agent')) {
        headers.set(
            'user-agent',
            'eve-kill-nuxt-ssr/1.0 (contact: admin@eve-kill.com)',
        )
    }

    return createServerApiClient({
        socket: config.apiSocket,
        origin: config.apiOrigin,
        headers,
    }) as unknown as typeof $fetch
}

/**
 * Imperative API requests, matching the $fetch call shape.
 */
export function apiFetch<T = any>(
    url: string,
    opts: FetchOptions = {},
): Promise<T> {
    // Nitro augments $fetch with route inference for its own server tree. This
    // transport deliberately targets Shrike instead, so the response type is
    // supplied by the Huma-generated contract at each call site.
    return useApiClient()(url, opts as any) as Promise<T>
}

/**
 * Reactive API requests, matching the useFetch call shape.
 *
 * The options bag remains `any` because Nuxt's useFetch overload chain does
 * not survive a thin generic wrapper: `default` otherwise collapses to
 * `Ref<undefined>`. Callers retain response typing through their explicit
 * generic argument.
 */
export function useApiFetch<T = any>(
    url: string | (() => string) | Ref<string> | ComputedRef<string>,
    opts: any = {},
) {
    return useFetch<T>(url, {
        ...opts,
        $fetch: useApiClient(),
    })
}
// Deliberately no `getCachedData` default here: several callers (TopBox,
// MostValuable, the battle panels) rely on refetching when you navigate back,
// and defaulting it would silently serve them the payload instead. Pass
// `getCachedData: cachedPayload` explicitly where payload reuse is wanted.
