import type { DomainsMineCompatResponses, SiteDomainEntity, SiteDomainNavbarLink, SiteDomainNavbarGroup, SiteDomainNavbarItem, SiteDomainWidget, SiteDomainWidgets } from '../../../shared/api/types.gen';
import { defaultDomainWidgets, isDomainWidgetType } from '../../utils/domainEditor';
import { isRecord } from './json';
type OwnedDomain = DomainsMineCompatResponses[200]['domains'][number];
export type DomainTheme = NonNullable<OwnedDomain['theme']>;
export type DomainCampaign = NonNullable<OwnedDomain['campaigns']>[number] & {
    campaign_id: string;
    name: string;
    visibility: number;
};
export type DomainAsset = NonNullable<OwnedDomain['backgrounds']>[number] & {
    id: number;
    status: string;
};
export type DomainDraft = Pick<OwnedDomain, 'id' | 'active' | 'site_name' | 'site_description' | 'campaign_policy'> & {
    subdomain: string;
    entities: SiteDomainEntity[];
    theme: DomainTheme;
    widgets: SiteDomainWidgets;
    navbar_links: SiteDomainNavbarLink[];
    campaigns: DomainCampaign[];
    backgrounds: DomainAsset[];
    bannerAsset?: DomainAsset | null;
    logoAsset?: DomainAsset | null;
};
export const isAssetSlot = (value: unknown): value is 'banner' | 'logo' => value === 'banner' || value === 'logo';
export const isUploadSlot = (value: unknown): value is 'banner' | 'logo' | 'background' => isAssetSlot(value) || value === 'background';
export type UploadSlot = 'banner' | 'logo' | 'background';
const positiveID = (value: unknown): value is number => typeof value === 'number' && Number.isSafeInteger(value) && value > 0;
export function domainEntity(value: unknown): SiteDomainEntity | null {
    if (!isRecord(value) || (value.type !== 'character' && value.type !== 'corporation' && value.type !== 'alliance') || typeof value.name !== 'string')
        return null;
    // Current APIs use numeric IDs; keep the original prefixed hit form compatible.
    const id = typeof value.id === 'number' ? value.id : typeof value.id === 'string' ? Number(value.id.replace(/^(character|corporation|alliance)_/, '')) : NaN;
    return positiveID(id) ? { id, type: value.type, name: value.name } : null;
}
function rows<T>(value: unknown, read: (value: unknown) => T | null): T[] { return Array.isArray(value) ? value.map(read).filter((item): item is T => item !== null) : []; }
function navItem(value: unknown): SiteDomainNavbarItem | null {
    if (!isRecord(value) || typeof value.label !== 'string' || typeof value.href !== 'string')
        return null;
    return { label: value.label, href: value.href, ...(typeof value.icon === 'string' ? { icon: value.icon } : {}), ...(typeof value.external === 'boolean' ? { external: value.external } : {}) };
}
function navGroup(value: unknown): SiteDomainNavbarGroup | null {
    if (!isRecord(value) || !Array.isArray(value.items))
        return null;
    return { items: rows(value.items, navItem), ...(typeof value.label === 'string' ? { label: value.label } : {}) };
}
function navLink(value: unknown): SiteDomainNavbarLink | null {
    const item = navItem(value);
    if (!item || !isRecord(value))
        return null;
    return { ...item, ...(Array.isArray(value.children) ? { children: rows(value.children, navGroup) } : {}) };
}
function widgets(value: unknown): SiteDomainWidgets {
    if (!isRecord(value))
        return defaultDomainWidgets();
    const read = (value: unknown) => {
        if (!isRecord(value) || !isDomainWidgetType(value.type) || typeof value.enabled !== 'boolean')
            return null;
        const result: SiteDomainWidget & Record<string,unknown> = {...value,type:value.type,enabled:value.enabled};
        for(const key of ['content','killlistType'] as const){if(typeof value[key]==='string')result[key]=value[key];else delete result[key]}
        return result;
    };
    return { top: rows(value.top, read), left: rows(value.left, read), right: rows(value.right, read), ...(typeof value.columnRatio === 'string' ? { columnRatio: value.columnRatio } : {}) };
}
function theme(value: unknown): DomainTheme {
    if (!isRecord(value))
        return {};
    const result: DomainTheme = {};
    // Preserve older custom theme keys while validating the current known fields.
    for (const [key, item] of Object.entries(value))
        if (typeof item === 'string' || typeof item === 'boolean' || typeof item === 'number')
            result[key] = item;
    for (const key of ['bannerUrl', 'logoUrl', 'defaultThemePreset', 'accentColor', 'bgColor', 'primaryColor', 'textColor'] as const) {
        if (typeof value[key] === 'string')
            result[key] = value[key];
        else
            delete result[key];
    }
    for (const key of ['transparentBanner', 'showLogoInBanner', 'showNameInBanner', 'showDescriptionInBanner'] as const) {
        if (typeof value[key] === 'boolean')
            result[key] = value[key];
        else
            delete result[key];
    }
    if (typeof value.contentOpacity === 'number' && Number.isFinite(value.contentOpacity))
        result.contentOpacity = value.contentOpacity;
    else
        delete result.contentOpacity;
    if (isRecord(value.defaultThemeOverrides))
        result.defaultThemeOverrides = Object.fromEntries(Object.entries(value.defaultThemeOverrides).filter((entry): entry is [
            string,
            string
        ] => typeof entry[1] === 'string'));
    return result;
}
export function domainCampaign(value: unknown): DomainCampaign | null {
    if (!isRecord(value) || typeof value.campaign_id !== 'string' || typeof value.name !== 'string')
        return null;
    return { ...value, campaign_id: value.campaign_id, name: value.name, visibility: typeof value.visibility === 'number' ? value.visibility : 0, public_on_domain: value.public_on_domain === true };
}
function asset(value: unknown): DomainAsset | null {
    if (!isRecord(value) || !positiveID(value.id) || typeof value.status !== 'string')
        return null;
    return { ...value, id: value.id, status: value.status, reject_reason: typeof value.reject_reason === 'string' ? value.reject_reason : null };
}
export function domainDraft(value: unknown): DomainDraft | null {
    if (!isRecord(value) || typeof value.subdomain !== 'string' || !positiveID(value.id))
        return null;
    return { id: value.id, subdomain: value.subdomain, active: value.active === true,
        site_name: typeof value.site_name === 'string' ? value.site_name : null, site_description: typeof value.site_description === 'string' ? value.site_description : null,
        campaign_policy: value.campaign_policy === 1 ? 1 : 0, entities: rows(value.entities, domainEntity), theme: theme(value.theme), widgets: widgets(value.widgets), navbar_links: rows(value.navbar_links, navLink),
        campaigns: rows(value.campaigns, domainCampaign), backgrounds: rows(value.backgrounds, asset), bannerAsset: asset(value.bannerAsset), logoAsset: asset(value.logoAsset) };
}
export function readDomainList(value: unknown): DomainDraft[] { return rows(value, domainDraft); }
export function readDomainSeed(text: string | undefined): DomainDraft[] { try {
    return readDomainList(JSON.parse(text ?? 'null'));
}
catch {
    return [];
} }
export function readCampaignList(value: unknown): DomainCampaign[] { return rows(value, domainCampaign); }
export function newDomainDraft(): DomainDraft { return { subdomain: '', entities: [], theme: {}, navbar_links: [], widgets: defaultDomainWidgets(), site_name: '', site_description: '', campaign_policy: 0, campaigns: [], backgrounds: [] }; }
