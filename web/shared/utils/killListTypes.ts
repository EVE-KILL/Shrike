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
    'timezone-au', 'timezone-ru', 'timezone-eu', 'timezone-us-east', 'timezone-us-west',
    'solo', 'attackers-1', 'attackers-2-4', 'attackers-5-9', 'attackers-10-24',
    'attackers-25-49', 'attackers-50-99', 'attackers-100-999', 'attackers-1000-plus',
    'pvp', 'ganked', 'npc', 'big', '5b', '10b',
    'under-1b', '1b-5b', '5b-10b', '10b-100b', '100b-1t', '1t-plus',
    'category-deployable', 'category-drone',
    'category-fighter', 'category-orbital', 'category-starbase', 'category-ship',
    'category-sovereignty', 'category-structure', 'category-infantry',
    'frigates', 'destroyers', 'cruisers', 'battlecruisers', 'battleships',
    'capitals', 'freighters', 'supercarriers', 'titans',
    'citadels', 't1', 't2', 't3', 'faction',
] as const

export type KillListType = (typeof KILL_LIST_TYPES)[number]
