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
        label: 'Kills', href: '/kills/latest',
        children: [
            {
                label: 'Activity',
                items: [
                    { label: 'Latest', href: '/kills/latest' },
                    { label: 'Big Kills', href: '/kills/big' },
                    { label: 'Solo', href: '/kills/solo' },
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
                ],
            },
            {
                label: 'Ship Classes',
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
                ],
            },
            {
                label: 'Structures',
                items: [
                    { label: 'Citadels', href: '/kills/citadels' },
                    { label: 'T1', href: '/kills/t1' },
                    { label: 'T2', href: '/kills/t2' },
                    { label: 'T3', href: '/kills/t3' },
                ],
            },
            {
                label: 'Meta',
                items: [
                    { label: 'Faction', href: '/kills/faction' },
                    { label: '5B+', href: '/kills/5b' },
                    { label: '10B+', href: '/kills/10b' },
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
    {
        label: 'Tools', href: '/tools/localscan',
        children: [
            {
                items: [
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
