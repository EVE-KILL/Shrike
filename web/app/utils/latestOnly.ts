/**
 * Guard for debounced search-as-you-type: keeps only the newest request's
 * result and discards responses that arrive out of order.
 *
 * Debouncing alone does NOT fix this. It limits how often a request *starts*;
 * it does nothing about two in-flight requests resolving in the wrong order.
 * Type "avat", pause past the debounce, type "ar" — two requests go out, and
 * if the first resolves last you render results for "avat" under "avatar".
 *
 *     const search = latestOnly()
 *     const seq = search.begin()
 *     const data = await apiFetch(...)
 *     if (!search.isCurrent(seq)) return   // a newer search superseded us
 *     results.value = data.hits
 *
 * Chosen over an AbortController because aborting rejects the in-flight
 * request, and these call sites clear their results on error — so cancelling
 * would blank the list on every keystroke instead of leaving the previous
 * results up until the new ones land.
 */
export function latestOnly() {
    let seq = 0
    return {
        /** Claim the next sequence number. Call immediately before the request. */
        begin: () => ++seq,
        /** True only if no later request has been started since `seq`. */
        isCurrent: (n: number) => n === seq,
    }
}
