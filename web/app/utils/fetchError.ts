/**
 * Pull the most human-readable message out of a apiFetch error. Depending on
 * how the server responded, ofetch surfaces the clean text as .statusMessage
 * or inside .data — the raw .message is deliberately skipped because it is
 * the ugly `[POST] "/api/...": 400` form.
 */
export function extractFetchError(err: unknown, fallback: string): string {
    if (err && typeof err === 'object') {
        const e = err as { statusMessage?: string; data?: { statusMessage?: string; message?: string } }
        return e.statusMessage ?? e.data?.statusMessage ?? e.data?.message ?? fallback
    }
    return fallback
}
