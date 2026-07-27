import { createFetch, type $Fetch } from 'ofetch'

interface ServerApiOptions {
    socket?: string
    origin?: string
    headers?: HeadersInit
}

/**
 * Build the SSR-only transport to Shrike.
 *
 * In the combined process image Bun talks directly to Go over a Unix socket.
 * The ordinary HTTP origin remains as a development fallback for running
 * Nuxt separately. Browser code never calls this helper: it keeps using
 * relative same-origin `/api` requests through Caddy.
 */
export function createServerApiClient(options: ServerApiOptions): $Fetch {
    const defaults = {
        baseURL: options.socket
            ? 'http://shrike.internal'
            : options.origin || 'http://127.0.0.1:4000',
        headers: options.headers,
    }

    if (!options.socket) {
        return createFetch({ defaults })
    }

    const socket = options.socket
    const unixFetch = (
        input: string | URL | Request,
        init?: RequestInit,
    ): Promise<Response> => globalThis.fetch(
        input,
        { ...init, unix: socket } as RequestInit,
    )

    return createFetch({
        defaults,
        fetch: unixFetch as typeof globalThis.fetch,
    })
}
