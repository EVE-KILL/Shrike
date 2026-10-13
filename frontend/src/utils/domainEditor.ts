import type { DomainsMineCompatResponses, SiteDomainWidget, SiteDomainNavbarLink, SiteDomainWidgets } from '#shared/api';
// Shared data transforms; DOM lifecycle lives in the browser controller.
type OwnedDomain = DomainsMineCompatResponses[200]['domains'][number];
export type EditableDomain = {
    entities?: NonNullable<OwnedDomain['entities']>;
    theme?: NonNullable<OwnedDomain['theme']>;
    navbar_links: SiteDomainNavbarLink[];
    widgets?: Partial<SiteDomainWidgets>;
    site_name?: string | null;
    site_description?: string | null;
    campaign_policy?: number;
    campaigns?: Array<{
        campaign_id: string;
        public_on_domain?: boolean;
    }>;
};
export type DomainSnapshot = {
    entities: NonNullable<EditableDomain['entities']>;
    theme: NonNullable<EditableDomain['theme']>;
    navbar_links: SiteDomainNavbarLink[];
    widgets: Partial<SiteDomainWidgets>;
    site_name: string | null;
    site_description: string | null;
    campaign_policy: number;
    campaign_ids: string[];
    campaign_public_ids: string[];
};
export type DomainPatch = Partial<DomainSnapshot>;
export type DomainWidgetType = SiteDomainWidget['type'];
// Matches the existing API maximum for top-level navigation links.
export const DOMAIN_NAVBAR_LIMIT = 10;
export const WIDGET_LABELS: Record<DomainWidgetType, string> = {
    mostValuable: 'Most Valuable Kills',
    killList: 'Kill List',
    topCharacters: 'Top Characters',
    topCorporations: 'Top Corporations',
    topAlliances: 'Top Alliances',
    topShips: 'Top Ships',
    topSystems: 'Top Systems',
    topRegions: 'Top Regions',
    entityInfo: 'Entity Info',
    textBlock: 'Text Block',
    campaigns: 'Campaigns',
};
export const WIDGET_ICONS: Record<DomainWidgetType, string> = {
    mostValuable: 'lucide:gem',
    killList: 'lucide:swords',
    topCharacters: 'lucide:user',
    topCorporations: 'lucide:building-2',
    topAlliances: 'lucide:shield',
    topShips: 'lucide:rocket',
    topSystems: 'lucide:map-pin',
    topRegions: 'lucide:globe',
    entityInfo: 'lucide:info',
    textBlock: 'lucide:file-text',
    campaigns: 'lucide:target',
};
export const ALL_WIDGET_TYPES = Object.keys(WIDGET_LABELS) as DomainWidgetType[];
export function isDomainWidgetType(value:unknown):value is DomainWidgetType {
    return typeof value==='string'&&Object.hasOwn(WIDGET_LABELS,value);
}
export const COLUMN_RATIO_OPTIONS = [
    { value: '250px_1fr', label: '250px / auto' },
    { value: '300px_1fr', label: '300px / auto' },
    { value: '1fr_1fr', label: '50% / 50%' },
    { value: '1fr_2fr', label: '33% / 66%' },
    { value: '1fr_3fr', label: '25% / 75%' },
];
const SERVER_MANAGED_THEME_KEYS = new Set(['bannerUrl', 'logoUrl']);
const cloneEditableValue = <T>(value: T): T => JSON.parse(JSON.stringify(value));
export const snapshotEditableDomain = (domain: EditableDomain): DomainSnapshot => ({
    entities: cloneEditableValue(domain.entities || []),
    theme: cloneEditableValue(domain.theme || {}),
    navbar_links: cloneEditableValue(domain.navbar_links.filter((link) => link.label && link.href)),
    widgets: cloneEditableValue(domain.widgets || {}),
    site_name: domain.site_name || null,
    site_description: domain.site_description || null,
    campaign_policy: Number(domain.campaign_policy ?? 0),
    campaign_ids: (domain.campaigns || []).map((campaign) => campaign.campaign_id),
    campaign_public_ids: (domain.campaigns || [])
        .filter((campaign) => campaign.public_on_domain)
        .map((campaign) => campaign.campaign_id),
});
const valuesEqual = (left: unknown, right: unknown) => JSON.stringify(left) === JSON.stringify(right);
const buildThemePatch = (current: Record<string, unknown>, previous: Record<string, unknown>) => {
    const patch: Record<string, unknown> = {};
    const keys = new Set([...Object.keys(previous || {}), ...Object.keys(current || {})]);
    for (const key of keys) {
        if (SERVER_MANAGED_THEME_KEYS.has(key))
            continue;
        const currentHasKey = Object.prototype.hasOwnProperty.call(current, key);
        const currentValue = currentHasKey && current[key] !== undefined
            ? current[key]
            : null;
        if (!valuesEqual(currentValue, previous?.[key] ?? null)) {
            patch[key] = currentValue;
        }
    }
    return patch;
};
export const buildDomainPatch = (current: EditableDomain, previous: Partial<DomainSnapshot>) => {
    const snapshot = snapshotEditableDomain(current);
    const patch: DomainPatch = {};
    for (const key of [
        'entities',
        'navbar_links',
        'widgets',
        'site_name',
        'site_description',
        'campaign_policy',
        'campaign_ids',
        'campaign_public_ids',
    ] as const) {
        if (!valuesEqual(snapshot[key], previous[key])) {
            Object.assign(patch, { [key]: snapshot[key] });
        }
    }
    // The server reads campaign_public_ids only alongside campaign_ids (it is a
    // per-selection flag, rewritten with the selection), so send them together
    // whenever either moved.
    if (patch.campaign_ids !== undefined || patch.campaign_public_ids !== undefined) {
        patch.campaign_ids = snapshot.campaign_ids;
        patch.campaign_public_ids = snapshot.campaign_public_ids;
    }
    const theme = buildThemePatch(snapshot.theme || {}, previous.theme || {});
    if (Object.keys(theme).length > 0)
        patch.theme = theme;
    return patch;
};
export const defaultDomainWidgets = (): SiteDomainWidgets => ({
    top: [{ type: 'mostValuable', enabled: true }],
    left: [
        { type: 'topCharacters', enabled: true },
        { type: 'topCorporations', enabled: true },
        { type: 'topAlliances', enabled: true },
        { type: 'topShips', enabled: true },
        { type: 'topSystems', enabled: true },
        { type: 'topRegions', enabled: true },
    ],
    right: [{ type: 'killList', enabled: true, killlistType: 'latest' }],
    columnRatio: '250px_1fr',
});
// A save acknowledges the submitted draft. Edits made while it is in flight
// stay local and remain different from the server baseline for the next save.
export const rebaseDomainEditor = <T extends EditableDomain>(editor: T, fresh: Partial<EditableDomain>, submitted: T) => {
    const changes = buildDomainPatch(editor, snapshotEditableDomain(submitted));
    const campaigns = cloneEditableValue(editor.campaigns || []);
    const merged = { ...cloneEditableValue(submitted), ...cloneEditableValue(fresh) };
    merged.theme ||= {};
    for (const key of ['showLogoInBanner', 'showNameInBanner', 'showDescriptionInBanner']) {
        if (merged.theme[key] === undefined)
            merged.theme[key] = true;
    }
    const baseline = snapshotEditableDomain(merged);
    Object.assign(editor, merged);
    for (const [key, value] of Object.entries(changes)) {
        if (key === 'theme' && changes.theme)
            Object.assign(editor.theme ??= {}, changes.theme);
        else if (key !== 'campaign_ids' && key !== 'campaign_public_ids')
            Object.assign(editor, { [key]: value });
    }
    if (changes.campaign_ids)
        editor.campaigns = campaigns;
    return baseline;
};
