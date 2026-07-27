import {
    MAX_BOARD_ENTRIES,
    parseBoardCookie,
    serializeBoardCookie,
} from '#shared/utils/boardList'

export interface ShortlistBoard {
    key: string
    host: string
    url: string
    name: string
    tracked: boolean
    pinned: boolean
}

interface BoardsResponse {
    boards: ShortlistBoard[]
    current: { key: string; name: string; listed: boolean } | null
    authenticated: boolean
    atCapacity: boolean
}

/**
 * The bottom-right killboard shortlist.
 *
 * Nothing is fetched until the switcher is opened for the first time — the
 * collapsed state is just the E, so a visitor who never opens it never pays
 * for a request.
 *
 * Writes go to the ek_boards cookie first (that is what makes a pin visible on
 * the next subdomain, and the only store a signed-out visitor has), then to the
 * account when there is a session, which is what carries the list to other
 * devices and to boards on their own hostname.
 */
export function useBoards() {
    const state = useState<BoardsResponse | null>('boards-shortlist', () => null)
    const loading = useState<boolean>('boards-shortlist-loading', () => false)
    const loaded = useState<boolean>('boards-shortlist-loaded', () => false)

    // `.eve-kill.com` is what makes this shared across subdomains; on localhost
    // a domain would make the cookie unsettable, so leave it host-scoped there.
    const cookie = useCookie<string>('ek_boards', {
        maxAge: 365 * 86400,
        path: '/',
        sameSite: 'lax',
        domain: import.meta.dev ? undefined : '.eve-kill.com',
        default: () => '',
    })

    const load = async (force = false) => {
        if (import.meta.server) return
        if (loading.value || (loaded.value && !force)) return
        loading.value = true
        try {
            state.value = await apiFetch<BoardsResponse>('/api/boards/mine')
            loaded.value = true
        } catch {
            state.value = null
        } finally {
            loading.value = false
        }
    }

    /** Cookie write, then best-effort account mirror — never block the UI on it. */
    const persist = async (pinned: string[], dismissed: string[]) => {
        cookie.value = serializeBoardCookie({ pinned, dismissed })
        if (!state.value?.authenticated) return
        try {
            await apiFetch('/api/user/boards', { method: 'PUT', body: { pinned, dismissed } })
        } catch {
            // The cookie already took; the account copy catches up next write.
        }
    }

    const mutate = async (fn: (lists: { pinned: string[]; dismissed: string[] }) => void) => {
        const lists = parseBoardCookie(cookie.value)
        // Fold in whatever the server knows, so an account-only pin is not
        // dropped by a cookie-only write from a fresh browser.
        for (const board of state.value?.boards ?? []) {
            if (board.pinned && !lists.pinned.includes(board.key)) lists.pinned.push(board.key)
        }
        fn(lists)
        await persist(lists.pinned, lists.dismissed)
        await load(true)
    }

    const pin = (key: string) => mutate((lists) => {
        lists.dismissed = lists.dismissed.filter(entry => entry !== key)
        if (!lists.pinned.includes(key)) lists.pinned.push(key)
    })

    /** Removing covers both sources: drop the pin and remember the dismissal. */
    const remove = (key: string) => mutate((lists) => {
        lists.pinned = lists.pinned.filter(entry => entry !== key)
        if (!lists.dismissed.includes(key)) lists.dismissed.push(key)
    })

    return {
        boards: computed(() => state.value?.boards ?? []),
        current: computed(() => state.value?.current ?? null),
        authenticated: computed(() => state.value?.authenticated ?? false),
        atCapacity: computed(() => state.value?.atCapacity ?? false),
        loading,
        loaded,
        load,
        pin,
        remove,
        maxEntries: MAX_BOARD_ENTRIES,
    }
}
