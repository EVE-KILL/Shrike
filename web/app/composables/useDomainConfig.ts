import type {
    SiteConfigurationResponse,
    SiteDomainConfiguration,
    SiteDomainEntity,
    SiteDomainNavbarGroup,
    SiteDomainNavbarLink,
    SiteDomainWidget,
    SiteDomainWidgets,
} from '#shared/api'

// Keep the component-facing names stable while making the Huma-generated
// contract their sole definition.
export type DomainConfig = SiteDomainConfiguration
export type DomainEntity = SiteDomainEntity
export type NavbarGroup = SiteDomainNavbarGroup
export type NavbarLink = SiteDomainNavbarLink
export type WidgetConfig = SiteDomainWidget
export type HomepageWidgets = SiteDomainWidgets

export const DEFAULT_NAVBAR: NavbarLink[] = [
    {
        label: 'Menu', href: '/kills/latest',
        children: [
            {
                label: 'Explore',
                items: [
                    { label: 'Categories', href: '/categories', icon: 'lucide:tags' },
                    { label: 'Coalitions', href: '/coalitions', icon: 'lucide:network' },
                    { label: 'Market', href: '/market', icon: 'lucide:shopping-cart' },
                    { label: 'Rankings', href: '/rankings', icon: 'lucide:trophy' },
                    { label: 'Advanced Search', href: '/advancedsearch', icon: 'lucide:search' },
                    { label: 'Battle Generator', href: '/battlegenerator', icon: 'lucide:shield' },
                    { label: 'Campaign Creator', href: '/campaigncreator', icon: 'lucide:flag' },
                    { label: 'Comments', href: '/comments', icon: 'lucide:message-square' },
                    { label: 'Local Scan', href: '/tools/localscan', icon: 'lucide:scan-search' },
                    { label: 'D-Scan', href: '/tools/dscan', icon: 'lucide:radar' },
                    { label: 'Legacy Kills', href: '/legacy', icon: 'lucide:archive' },
                ],
            },
        ],
    },
    {
        label: 'Kills', href: '/kills/latest',
        children: [
            {
                label: 'Activity',
                items: [
                    { label: 'Latest', href: '/kills/latest' },
                    { label: 'Big Kills', href: '/kills/big' },
                    { label: 'Solo', href: '/kills/solo' },
                    { label: 'Awox', href: '/kills/awox' },
                    { label: 'PvP', href: '/kills/pvp' },
                    { label: 'Highsec Ganks', href: '/kills/ganked' },
                    { label: 'NPC', href: '/kills/npc' },
                ],
            },
            {
                label: 'Security',
                items: [
                    { label: 'High Sec', href: '/kills/highsec' },
                    { label: 'Low Sec', href: '/kills/lowsec' },
                    { label: 'Null Sec', href: '/kills/nullsec' },
                    { label: 'Wormhole', href: '/kills/wspace' },
                    { label: 'Abyssal', href: '/kills/abyssal' },
                    { label: 'Pochven', href: '/kills/pochven' },
                    { label: 'Jove Space', href: '/kills/jove' },
                ],
            },
            {
                label: 'Timezones',
                items: [
                    { label: 'AU Timezone', href: '/kills/timezone-au' },
                    { label: 'RU Timezone', href: '/kills/timezone-ru' },
                    { label: 'EU Timezone', href: '/kills/timezone-eu' },
                    { label: 'US East Timezone', href: '/kills/timezone-us-east' },
                    { label: 'US West Timezone', href: '/kills/timezone-us-west' },
                ],
            },
            {
                label: 'Attacker Counts',
                items: [
                    { label: '2–4 Attackers', href: '/kills/attackers-2-4' },
                    { label: '5–9 Attackers', href: '/kills/attackers-5-9' },
                    { label: '10–24 Attackers', href: '/kills/attackers-10-24' },
                    { label: '25–49 Attackers', href: '/kills/attackers-25-49' },
                    { label: '50–99 Attackers', href: '/kills/attackers-50-99' },
                    { label: '100–999 Attackers', href: '/kills/attackers-100-999' },
                    { label: '1,000+ Attackers', href: '/kills/attackers-1000-plus' },
                ],
            },
            {
                label: 'Involvement',
                items: [
                    { label: 'Capital Involved', href: '/kills/capital-involved' },
                    { label: 'Supercarrier Involved', href: '/kills/supercarrier-involved' },
                    { label: 'Titan Involved', href: '/kills/titan-involved' },
                    { label: 'AT Ship Involved', href: '/kills/at-ship-involved' },
                ],
            },
            {
                label: 'Faction Warfare',
                items: [
                    { label: 'Caldari Victories', href: '/kills/fw-caldari-winner' },
                    { label: 'Gallente Victories', href: '/kills/fw-gallente-winner' },
                    { label: 'Amarr Victories', href: '/kills/fw-amarr-winner' },
                    { label: 'Minmatar Victories', href: '/kills/fw-minmatar-winner' },
                    { label: 'Caldari–Gallente', href: '/kills/fw-caldari-gallente' },
                    { label: 'Amarr–Minmatar', href: '/kills/fw-amarr-minmatar' },
                ],
            },
            {
                label: 'Value',
                items: [
                    { label: '1B+', href: '/kills/big' },
                    { label: '5B+', href: '/kills/5b' },
                    { label: '10B+', href: '/kills/10b' },
                    { label: 'Under 1B', href: '/kills/under-1b' },
                    { label: '1B–5B', href: '/kills/1b-5b' },
                    { label: '5B–10B', href: '/kills/5b-10b' },
                    { label: '10B–100B', href: '/kills/10b-100b' },
                    { label: '100B–1T', href: '/kills/100b-1t' },
                    { label: '1T+', href: '/kills/1t-plus' },
                ],
            },
            {
                label: 'Victim Hulls',
                items: [
                    { label: 'Frigates', href: '/kills/frigates' },
                    { label: 'Destroyers', href: '/kills/destroyers' },
                    { label: 'Cruisers', href: '/kills/cruisers' },
                    { label: 'Battlecruisers', href: '/kills/battlecruisers' },
                    { label: 'Battleships', href: '/kills/battleships' },
                    { label: 'Capitals', href: '/kills/capitals' },
                    { label: 'Supercarriers', href: '/kills/supercarriers' },
                    { label: 'Titans', href: '/kills/titans' },
                    { label: 'Freighters', href: '/kills/freighters' },
                    { label: 'Structures', href: '/kills/citadels' },
                ],
            },
            {
                label: 'Victim Categories',
                items: [
                    { label: 'Deployable', href: '/kills/category-deployable' },
                    { label: 'Drone', href: '/kills/category-drone' },
                    { label: 'Fighter', href: '/kills/category-fighter' },
                    { label: 'Orbital', href: '/kills/category-orbital' },
                    { label: 'Starbase', href: '/kills/category-starbase' },
                    { label: 'Ship', href: '/kills/category-ship' },
                    { label: 'Sovereignty', href: '/kills/category-sovereignty' },
                    { label: 'Structure', href: '/kills/category-structure' },
                    { label: 'Infantry', href: '/kills/category-infantry' },
                ],
            },
            {
                label: 'Technology',
                items: [
                    { label: 'T1', href: '/kills/t1' },
                    { label: 'T2', href: '/kills/t2' },
                    { label: 'T3', href: '/kills/t3' },
                    { label: 'Faction', href: '/kills/faction' },
                ],
            },
        ],
    },
    { label: 'Wars', href: '/wars' },
    { label: 'Battles', href: '/battles' },
    { label: 'Campaigns', href: '/campaigns' },
    { label: 'Stats', href: '/stats' },
    {
        label: 'Fits', href: '/fits',
        children: [
            {
                items: [
                    { label: 'Browse Fits', href: '/fits', icon: 'lucide:wrench' },
                    { label: 'Search Fits', href: '/fits/search', icon: 'lucide:search' },
                    { label: 'Create New Fit', href: '/fits/create', icon: 'lucide:plus' },
                ],
            },
        ],
    },
]

export function useDomainConfig() {
    const site = useState<SiteConfigurationResponse | null>(
        'site-configuration',
        () => null,
    )
    const config = computed(() => site.value?.domain ?? null)
    const isDomainMode = computed(() => config.value !== null)
    /** True when on a subdomain that has no matching custom_domains record */
    const isUnknownDomain = computed(
        () => site.value?.isDomainHost === true && !config.value,
    )

    return {
        domainConfig: config,
        isDomainMode,
        isUnknownDomain,
    }
}
