/**
 * The kill-list browse routes served by /kills/[type].
 *
 * Shared because two places need the same list and they must not drift: the
 * page validates the route param against it (anything else 404s), and the
 * sitemap emits one URL per entry. When these were separate the sitemap simply
 * omitted all of them, so /kills/latest — a priority-0.8 route in routeRules —
 * was never advertised to crawlers at all.
 */
export const KILL_LIST_TYPES = [
    'latest', 'highsec', 'lowsec', 'nullsec', 'wspace', 'abyssal', 'pochven', 'jove',
    'big', 'solo', 'npc', '5b', '10b',
    'frigates', 'destroyers', 'cruisers', 'battlecruisers', 'battleships',
    'capitals', 'freighters', 'supercarriers', 'titans',
    'citadels', 't1', 't2', 't3', 'faction',
] as const

export type KillListType = (typeof KILL_LIST_TYPES)[number]
