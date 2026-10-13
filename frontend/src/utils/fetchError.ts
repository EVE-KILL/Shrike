/**
 * Pull the most human-readable message out of a apiFetch error. Depending on
 * how the server responded, ofetch surfaces the clean text as .statusMessage
 * or inside .data — the raw .message is deliberately skipped because it is
 * the ugly `[POST] "/api/...": 400` form.
 */
export function extractFetchError(err: unknown, fallback: string): string {
    if (err && typeof err === 'object') {
        const e = err as { statusMessage?: unknown; data?: { statusMessage?: unknown; message?: unknown; detail?: unknown; error?: unknown } }
        // Huma returns RFC 9457 problem details; legacy routes use message or
        // error. Prefer the useful body over generic HTTP text like "Bad Request".
        for (const message of [e.data?.message,e.data?.detail,e.data?.error,e.data?.statusMessage,e.statusMessage]) {
            if (typeof message === 'string' && message.trim()) return message
        }
    }
    return fallback
}
