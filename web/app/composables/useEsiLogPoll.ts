import type { EsiLogRow } from '~/utils/esiLog'

const POLL_MS = 5000
const MAX_ROWS = 50

/**
 * Live tail for an ESI request log: every few seconds, ask for rows newer than
 * the one currently at the top and splice them in.
 *
 * Only ever runs on page 1 — prepending to a paged view would shift rows out
 * from under whoever is reading them.
 *
 * Existed as two copies (admin monitor, settings tab) that had already drifted:
 * the admin one forwarded its filters to the poll and cleaned up on
 * deactivate/unmount, the settings one forwarded only some filters and left the
 * interval running for the life of the page.
 */
export function useEsiLogPoll(opts: {
    /** Where to fetch from — `/api/admin/esi-logs` or `/api/user/esi-logs`. */
    endpoint: string
    /** The list to prepend into. Mutated in place so the table updates live. */
    data: Ref<{ rows: EsiLogRow[]; total: number } | null>
    /** Current page — polling pauses on anything but the first. */
    page: Ref<number>
    /** User's live on/off toggle. */
    enabled: Ref<boolean>
    /** Active filters, so the tail matches what is on screen. */
    params: () => Record<string, string>
}) {
    let timer: ReturnType<typeof setInterval> | null = null

    const tick = async () => {
        if (!opts.enabled.value || opts.page.value !== 1) return
        const topId = opts.data.value?.rows?.[0]?.id
        if (!topId) return

        const query = new URLSearchParams({ after_id: String(topId), limit: String(MAX_ROWS) })
        for (const [k, v] of Object.entries(opts.params())) {
            if (v) query.set(k, v)
        }

        try {
            const fresh = await apiFetch<{ rows: EsiLogRow[] }>(`${opts.endpoint}?${query}`)
            if (!fresh.rows?.length || !opts.data.value) return
            opts.data.value.rows = [...fresh.rows, ...opts.data.value.rows].slice(0, MAX_ROWS)
            opts.data.value.total += fresh.rows.length
        } catch {
            // A dropped poll is not worth surfacing — the next tick retries, and
            // the operator still has the rows they already had.
        }
    }

    const start = () => {
        stop()
        if (!import.meta.client) return
        timer = setInterval(tick, POLL_MS)
    }

    const stop = () => {
        if (timer) { clearInterval(timer); timer = null }
    }

    // Both callers live inside a <KeepAlive>, so deactivation is the signal that
    // the tab went away — unmount alone would leave the interval running.
    // onActivated already fires on first mount inside KeepAlive; onMounted is
    // here so this still works if a caller is ever used outside one. start() is
    // idempotent, so the overlap is harmless.
    onMounted(start)
    onActivated(start)
    onDeactivated(stop)
    onBeforeUnmount(stop)

    return { start, stop }
}
