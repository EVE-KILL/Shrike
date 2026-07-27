/**
 * `ek_boards` — the visitor's killboard shortlist, shared across every
 * *.eve-kill.com subdomain.
 *
 * Scoped to `.eve-kill.com` like evelogin/ek_auth so a board pinned on one
 * subdomain is there on the next. localStorage cannot do this job: it is
 * partitioned per origin, so boring.eve-kill.com and eve-kill.com would each
 * keep their own copy and neither would ever see the other's.
 *
 * That reach has a price — the cookie rides along on every request to every
 * matching host and the same-origin `/images` route, which a killlist page
 * hits dozens of times. So the format is deliberately terse and hard-capped:
 *
 *     pinned|dismissed          e.g.  "boring,track|tremaljack"
 *
 * An entry with no dot is a *.eve-kill.com subdomain slug; one with a dot is a
 * board on its own hostname, stored whole because no slug can describe it. A
 * board on its own hostname is a separate registrable domain, so this cookie
 * never reaches it — such boards can be listed and linked, but a signed-out
 * visitor standing on one has no shortlist there. Signing in fixes that, since
 * the account copy in user_config is not origin-bound.
 */

export const BOARDS_COOKIE = 'ek_boards'

/** Per list, so a full shortlist stays well inside 4KB of request headers. */
export const MAX_BOARD_ENTRIES = 12

const SUBDOMAIN_SUFFIX = '.eve-kill.com'

export interface BoardListState {
    pinned: string[]
    dismissed: string[]
}

/** Slug (`boring`) or bare hostname (`kb.example.com`) — nothing else. */
function isValidKey(key: string): boolean {
    if (!key || key.length > 100) return false
    return key.includes('.')
        ? /^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(key)
        : /^[a-z0-9-]+$/.test(key)
}

function cleanList(entries: string[]): string[] {
    return [...new Set(
        entries.map(entry => String(entry).trim().toLowerCase()).filter(isValidKey),
    )].slice(0, MAX_BOARD_ENTRIES)
}

export function parseBoardCookie(raw: string | undefined | null): BoardListState {
    if (!raw) return { pinned: [], dismissed: [] }
    const [pinned = '', dismissed = ''] = String(raw).split('|')
    return {
        pinned: cleanList(pinned.split(',')),
        dismissed: cleanList(dismissed.split(',')),
    }
}

export function serializeBoardCookie(state: BoardListState): string {
    return `${cleanList(state.pinned).join(',')}|${cleanList(state.dismissed).join(',')}`
}

/** Normalize whatever `user_config.boards` holds into a usable pair of lists. */
export function parseStoredBoards(raw: unknown): BoardListState {
    const value = (raw ?? {}) as Partial<Record<keyof BoardListState, unknown>>
    const list = (input: unknown): string[] =>
        Array.isArray(input) ? cleanList(input.map(entry => String(entry))) : []
    return { pinned: list(value.pinned), dismissed: list(value.dismissed) }
}

/** `boring` → `boring.eve-kill.com`; a hostname key is already complete. */
export function boardKeyToHost(key: string): string {
    return key.includes('.') ? key : `${key}${SUBDOMAIN_SUFFIX}`
}

/** Inverse of boardKeyToHost — the apex itself is not a board. */
export function hostToBoardKey(host: string): string | null {
    const hostname = String(host || '').toLowerCase().split(':')[0] ?? ''
    if (!hostname || hostname === 'eve-kill.com' || hostname === 'www.eve-kill.com') {
        return null
    }
    if (hostname.endsWith(SUBDOMAIN_SUFFIX)) {
        const slug = hostname.slice(0, -SUBDOMAIN_SUFFIX.length)
        return isValidKey(slug) ? slug : null
    }
    return isValidKey(hostname) ? hostname : null
}

export function mergeBoardState(left: BoardListState, right: BoardListState): BoardListState {
    return {
        pinned: cleanList([...left.pinned, ...right.pinned]),
        // A dismissal always wins over a pin: the two only collide when a board
        // was pinned and later dismissed, and dismissing is the newer intent.
        dismissed: cleanList([...left.dismissed, ...right.dismissed]),
    }
}
