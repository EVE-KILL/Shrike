import { escapeHTML } from './shared/html';
import { eventElement, requiredElement } from './shared/dom';
import { errorMessage, isAbortError } from './shared/json';
import { domainDraft, readDomainSeed, readDomainList, readCampaignList, domainEntity, newDomainDraft, isAssetSlot, isUploadSlot, type DomainDraft, type DomainCampaign, type UploadSlot } from './shared/domain-seeds';
import { isDashboardSection } from './shared/dashboard-seeds';
import { registerController } from './controllers/application';
import { SettingsComponent } from './controllers/settings/base';
import type { JSONRequestOptions } from './browser-request';
import type { DomainSnapshot } from '../utils/domainEditor';
import type { DomainsMineCompatResponses, DomainCampaignSearchCompatResponses, DomainSubdomainCheckCompatResponses, DomainUpdatePatchCompatResponses, DomainAssetUploadCompatResponses, SiteDomainNavbarLink, SiteDomainNavbarGroup, SiteDomainNavbarItem, SiteDomainEntity } from '../../shared/api/types.gen';
type Transport = {
    request: <T = unknown>(path: string, options?: JSONRequestOptions, fallback?: string) => Promise<T>;
    upload: (path: string, body: FormData, fallback: string) => Promise<DomainAssetUploadCompatResponses[200]>;
    after: (delay: number, run: () => void) => void;
};
import { THEME_PRESETS, DEFAULT_NAVBAR, WIDGET_LABELS, WIDGET_ICONS, ALL_WIDGET_TYPES, COLUMN_RATIO_OPTIONS, defaultDomainWidgets, snapshotEditableDomain, buildDomainPatch, rebaseDomainEditor, isDomainWidgetType, DOMAIN_NAVBAR_LIMIT } from './settings-data';
const escape = escapeHTML;
const icon = (name: string, classes = '') => `<svg class="icon ${classes}" aria-hidden="true"><use href="/assets/icons.svg#${name.replace('lucide:', '')}"></use></svg>`;
const clone = <T>(value: T): T => structuredClone(value);
const entityImage = (entity: SiteDomainEntity, size = 32) => `https://images.evetech.net/${entity.type === 'character' ? 'characters' : entity.type === 'corporation' ? 'corporations' : 'alliances'}/${Number(entity.id)}/${entity.type === 'character' ? 'portrait' : 'logo'}?size=${size}`;
const url = (value: unknown) => { const s = String(value || ''); return /^https?:\/\//i.test(s) || /^\/(?!\/)/.test(s) ? escape(s) : ''; };
const inputClass = 'px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white placeholder-gray-600';
function domainRuntime(panel: HTMLElement, { request, upload: uploadForm, after }: Transport) {
    const q = <T extends HTMLElement = HTMLElement>(selector: string) => requiredElement<T>(panel, selector), all = <T extends HTMLElement = HTMLElement>(selector: string) => panel.querySelectorAll<T>(selector);
    let domains = readDomainSeed(panel.dataset.domains), editing: DomainDraft | null = null, baseline: DomainSnapshot | null = null, tab = 'general', saving = false, saved = false, navIndex: number | null = null, widgetMenu: string | null = null;
    let availability: DomainSubdomainCheckCompatResponses[200] | null = null, checking = false, entityResults: SiteDomainEntity[] = [], campaignResults: DomainCampaign[] = [], ownCampaigns: DomainCampaign[] = [], uploading = new Set<UploadSlot>(), listRequest = 0;
    const searches = new Map<string, {
        controller: AbortController;
        timer: ReturnType<typeof setTimeout> | null;
    }>();
    function cancelSearches() { for (const item of searches.values()) {
        if (item.timer !== null)
            clearTimeout(item.timer);
        item.controller?.abort();
    } searches.clear(); }
    function suspend() { cancelSearches(); checking = false; saved = false; q('[data-domain-campaign-loading]').hidden = true; renderAvailability(); renderSave(); }
    function search(key: string, delay: number, run: (signal: AbortSignal, current: () => boolean) => Promise<void>) { const old = searches.get(key); if (old?.timer != null)
        clearTimeout(old.timer); old?.controller?.abort(); const controller = new AbortController(), item: {
        controller: AbortController;
        timer: ReturnType<typeof setTimeout> | null;
    } = { controller, timer: null }; searches.set(key, item); item.timer = setTimeout(async () => { try {
        await run(controller.signal, () => searches.get(key) === item);
    }
    catch (error) {
        if (!isAbortError(error) && searches.get(key) === item)
            showError(error);
    }
    finally {
        if (searches.get(key) === item)
            searches.delete(key);
    } }, delay); }
    function showError(error: unknown) { if(!panel.isConnected||isAbortError(error))return;const message=typeof error==='string'?error:errorMessage(error);const box = q('[data-domain-error]'); box.hidden = !message; box.textContent = message || ''; if (!editing && message) {
        const retry = document.createElement('button');
        retry.type = 'button';
        retry.dataset.domainRefresh = '';
        retry.className = 'text-blue-400 ml-2';
        retry.textContent = 'Retry';
        box.append(retry);
    } }
    function setURL(value: string | null) { const address = new URL(location.href); if (value)
        address.searchParams.set('edit', value);
    else
        address.searchParams.delete('edit'); history.replaceState(history.state, '', address); }
    function fillFields(d: DomainDraft) { for (const input of all<HTMLInputElement|HTMLTextAreaElement>('[data-domain-field]')) {
        const key = input.dataset.domainField;
        input.value = key === 'subdomain' || key === 'site_name' || key === 'site_description' ? d[key] ?? '' : '';
    } }
    function close() { cancelSearches(); editing = null; baseline = null; availability = null; checking = false; setURL(null); render(); }
    function open(domain?: DomainDraft | null) { cancelSearches(); const d = domain ? clone(domain) : newDomainDraft(); d.theme ||= {}; for (const key of ['showLogoInBanner', 'showNameInBanner', 'showDescriptionInBanner'])
        if (d.theme[key] === undefined)
            d.theme[key] = true; d.entities ||= []; d.navbar_links ||= []; d.widgets ||= defaultDomainWidgets(); d.campaigns ||= []; d.backgrounds ||= []; editing = d; baseline = d.id ? snapshotEditableDomain(d) : null; tab = 'general'; navIndex = null; widgetMenu = null; availability = null; checking = false; entityResults = []; campaignResults = []; ownCampaigns = []; q<HTMLInputElement>('[data-domain-entity-search]').value = ''; q<HTMLInputElement>('[data-domain-campaign-search]').value = ''; showError(''); fillFields(d); setURL(d.id ? String(d.id) : 'new'); render(); if (d.id)
        loadCampaigns(d); }
    async function refresh() { const sequence = ++listRequest, data = await request<DomainsMineCompatResponses[200]>('/api/user/domains', {}, 'Could not load your custom domains'); if (sequence !== listRequest)
        return; domains = readDomainList(data.domains); renderList(); const d = editing, fresh = d && domains.find(domain => domain.id === d.id); if (fresh && d) {
        for (const key of ['bannerUrl', 'logoUrl']) {
            if (fresh.theme?.[key])
                d.theme[key] = fresh.theme[key];
            else
                delete d.theme[key];
        }
        d.bannerAsset = fresh.bannerAsset || null;
        d.logoAsset = fresh.logoAsset || null;
        d.backgrounds = clone(fresh.backgrounds || []);
        renderAppearance();
    } }
    async function loadCampaigns(d: DomainDraft) { try {
        const data = await request<DomainCampaignSearchCompatResponses[200]>(`/api/user/domains/${d.id}/campaigns/search`);
        if (editing !== d)
            return;
        ownCampaigns = readCampaignList(data.campaigns);
        renderCampaigns();
    }
    catch (error) {
        if (editing === d)
            showError(error);
    } }
    function renderList() {
        q('[data-domain-cards]').innerHTML = domains.length ? domains.map(d => `<div class="glass-panel p-4" data-domain-card="${Number(d.id)}"><div class="flex items-start justify-between gap-4"><div class="flex-1 min-w-0"><div class="flex items-center gap-2 mb-1"><a href="https://${escape(d.subdomain)}.eve-kill.com" target="_blank" rel="noopener" class="text-sm font-bold text-blue-400 hover:text-blue-300 transition-colors">${escape(d.subdomain)}.eve-kill.com</a><span class="px-1.5 py-0.5 rounded text-fine font-medium ${d.active ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}">${d.active ? 'Active' : 'Inactive'}</span></div>${d.site_name ? `<div class="text-xs text-gray-400 mb-2">${escape(d.site_name)}</div>` : ''}<div class="flex flex-wrap gap-1.5">${(d.entities || []).map(entity => `<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-fine bg-white/[0.06] text-gray-300"><img src="${entityImage(entity)}" width="16" height="16" alt="" class="w-4 h-4 rounded-full">${escape(entity.name)}</span>`).join('')}</div></div><div class="flex items-center gap-1 flex-shrink-0"><button type="button" data-domain-edit="${Number(d.id)}" aria-label="Edit ${escape(d.subdomain)}" class="p-1.5 rounded text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors">${icon('pencil')}</button><button type="button" data-domain-delete="${Number(d.id)}" aria-label="Delete ${escape(d.subdomain)}" class="p-1.5 rounded text-gray-500 hover:text-red-400 hover:bg-red-500/10 transition-colors">${icon('trash-2')}</button></div></div></div>`).join('') : `<div class="glass-panel p-8 text-center">${icon('globe', 'text-3xl text-gray-600 mb-3')}<p class="text-sm text-gray-500">No custom domains yet.</p><p class="text-xs text-gray-600 mt-1">Create a subdomain to get your own personalized killboard.</p></div>`;
    }
    function render() {
        const error = q('[data-domain-error]');
        if (editing)
            q('[data-domain-editor]').insertBefore(error, q('[data-domain-pane="general"]'));
        else
            panel.prepend(error);
        q('[data-domain-list]').hidden = Boolean(editing);
        q('[data-domain-editor]').hidden = !editing;
        if (!editing)
            return;
        q('[data-domain-title]').textContent = editing.id ? 'Edit Domain' : 'Create Domain';
        q<HTMLInputElement>('[data-domain-field="subdomain"]').disabled = Boolean(editing.id);
        for (const pane of all('[data-domain-pane]'))
            pane.hidden = pane.dataset.domainPane !== tab;
        for (const button of all('[data-domain-tab]')) {
            const active = button.dataset.domainTab === tab;
            button.setAttribute('aria-pressed', String(active));
            button.className = 'px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer ' + (active ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]');
        }
        renderAvailability();
        renderEntities();
        renderCampaigns();
        renderAppearance();
        renderWidgets();
        renderNav();
        renderSave();
    }
    function renderSave() { if (!editing)
        return; const button = q<HTMLButtonElement>('[data-domain-save]'); button.disabled = saving || !editing.subdomain || !editing.entities.length || (!editing.id && !availability?.available); button.textContent = saving ? 'Saving...' : saved ? 'Saved' : editing.id ? 'Save Changes' : 'Create Domain'; button.className = 'px-4 py-2 rounded-md text-sm font-medium transition-colors disabled:opacity-50 cursor-pointer ' + (saved ? 'bg-green-500 text-white' : 'bg-blue-500 text-white hover:bg-blue-600'); }
    function renderAvailability() { if (!editing)
        return; const d = editing; const status = q('[data-domain-subdomain-status]'); status.hidden = Boolean(d.id) || (!checking && !availability); status.textContent = checking ? 'Checking availability...' : availability?.available ? 'Subdomain is available' : availability?.reason || ''; status.className = 'mt-1.5 text-xs ' + (checking ? 'text-gray-500' : availability?.available ? 'text-green-400' : 'text-red-400'); const input = q<HTMLInputElement>('[data-domain-field="subdomain"]'); input.className = 'flex-1 px-3 py-2 rounded-l-md bg-white/[0.04] border text-sm text-white placeholder-gray-600 focus:outline-none disabled:opacity-50 ' + (!d.id && availability ? (availability.available ? 'border-green-500/50 focus:border-green-500/50' : 'border-red-500/50 focus:border-red-500/50') : 'border-white/[0.08] focus:border-blue-500/50'); }
    function checkSubdomain(value: string) {
        availability = null;
        checking = false;
        const subdomain = value.toLowerCase().trim();
        search('subdomain', 400, async (signal, current) => { if (!subdomain || subdomain.length < 3 || subdomain.length > 32 || !/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/.test(subdomain))
            return; try {
            const result = await request<DomainSubdomainCheckCompatResponses[200]>('/api/user/domains/check-subdomain?' + new URLSearchParams({ subdomain }), { signal });
            if (current() && editing) {
                availability = result;
            }
        }
        finally {
            if (current() && editing) {
                checking = false;
                renderAvailability();
                renderSave();
            }
        } });
        if (subdomain.length && (subdomain.length < 3 || subdomain.length > 32))
            availability = { available: false, reason: 'Subdomain must be 3-32 characters' };
        else if (subdomain && !/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/.test(subdomain))
            availability = { available: false, reason: 'Lowercase alphanumeric and hyphens only' };
        else
            checking = Boolean(subdomain);
        renderAvailability();
        renderSave();
    }
    function renderEntities() {
        if (!editing)
            return;
        const d = editing;
        q('[data-domain-entities-label]').textContent = `Entities (${d.entities.length}/5)`;
        q('[data-domain-entities-full]').hidden = d.entities.length < 5;
        q('[data-domain-entity-picker]').hidden = d.entities.length >= 5;
        q('[data-domain-entities]').innerHTML = d.entities.map((entity, i) => `<span class="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-white/[0.06] text-xs text-gray-300"><img src="${entityImage(entity)}" width="16" height="16" alt="" class="w-4 h-4 rounded-full">${escape(entity.name)}<button type="button" data-domain-entity-remove="${i}" aria-label="Remove ${escape(entity.name)}" class="ml-0.5 text-gray-500 hover:text-red-400">${icon('x')}</button></span>`).join('');
        const box = q('[data-domain-entity-results]');
        box.hidden = !entityResults.length;
        box.innerHTML = entityResults.map((entity, i) => `<button type="button" data-domain-entity-add="${i}" class="w-full flex items-center gap-3 px-3 py-2.5 text-sm text-gray-300 hover:bg-blue-500/[0.08] transition-colors"><img src="${entityImage(entity, 64)}" width="32" height="32" alt="" class="w-8 h-8 rounded-full flex-shrink-0"><div class="flex-1 min-w-0 text-left"><div class="text-sm text-white truncate">${escape(entity.name)}</div><div class="text-fine text-gray-500 capitalize">${escape(entity.type)}</div></div></button>`).join('');
    }
    function renderCampaigns() {
        if (!editing)
            return;
        const d = editing;
        const selected = new Set(d.campaigns.map(c => c.campaign_id)), active = q<HTMLInputElement>('[data-domain-campaign-search]').value.trim().length >= 2, suggestions = ownCampaigns.filter(c => !selected.has(c.campaign_id)), matches = campaignResults.filter(c => !selected.has(c.campaign_id));
        for (const button of all('[data-domain-policy]'))
            button.className = 'rounded-lg border p-3 text-left transition-colors ' + (Number(button.dataset.domainPolicy) === d.campaign_policy ? 'border-blue-500/40 bg-blue-500/[0.08]' : 'border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.05]');
        q('[data-domain-campaign-count]').textContent = `Selected campaigns (${d.campaigns.length}/50)`;
        q('[data-domain-campaign-create-warning]').hidden = Boolean(d.id);
        q('[data-domain-campaign-picker]').hidden = !d.id;
        q('[data-domain-own-campaigns]').hidden = !d.id || active || !suggestions.length;
        q('[data-domain-campaign-results]').hidden = !active || !matches.length;
        function suggestion(c: DomainCampaign, own: boolean) { return `<button type="button" data-domain-campaign-add="${escape(c.campaign_id)}" class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-blue-500/[0.08] transition-colors">${icon(c.visibility === 1 ? 'lock' : c.visibility === 2 ? 'shield' : 'globe', own ? `text-sm flex-shrink-0 ${c.visibility === 1 ? 'text-yellow-400' : c.visibility === 2 ? 'text-blue-400' : 'text-green-400'}` : 'text-sm text-gray-500 flex-shrink-0')}<div class="flex-1 min-w-0"><div class="text-sm text-white truncate">${escape(c.name)}</div><div class="text-fine text-gray-600 font-mono">${escape(c.campaign_id)}</div></div>${own ? `<span class="text-fine text-gray-600">${c.visibility === 1 ? 'Private' : c.visibility === 2 ? 'Killboard' : 'Public'}</span>` : ''}<span class="text-fine text-blue-400">Add</span></button>`; }
        q('[data-domain-campaign-suggestions]').innerHTML = suggestions.map(c => suggestion(c, true)).join('');
        q('[data-domain-campaign-results]').innerHTML = matches.map(c => suggestion(c, false)).join('');
        q('[data-domain-campaign-empty]').hidden = Boolean(d.campaigns.length);
        q('[data-domain-selected-campaigns]').innerHTML = d.campaigns.map(c => `<div class="flex items-center gap-3 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5">${icon(c.visibility === 1 ? 'lock' : c.visibility === 2 ? 'shield' : 'globe', `text-sm ${c.visibility === 1 ? 'text-yellow-400' : c.visibility === 2 ? 'text-blue-400' : 'text-green-400'}`)}<div class="flex-1 min-w-0"><a href="/campaign/${encodeURIComponent(c.campaign_id)}" class="text-sm text-gray-200 hover:text-blue-400 truncate block">${escape(c.name)}</a><span class="text-fine text-gray-600 font-mono">${escape(c.campaign_id)}</span></div>${c.visibility === 1 ? `<button type="button" data-domain-campaign-audience="${escape(c.campaign_id)}" title="${c.public_on_domain ? 'Anyone visiting this killboard can read this private campaign — click for members only' : 'Only signed-in members of this killboard’s entities can read it — click to show everyone'}" class="inline-flex items-center gap-1.5 px-2 py-1 rounded border text-fine font-medium transition-colors cursor-pointer flex-shrink-0 ${c.public_on_domain ? 'border-amber-500/30 bg-amber-500/10 text-amber-300 hover:bg-amber-500/20' : 'border-white/[0.08] bg-white/[0.04] text-gray-400 hover:bg-white/[0.07] hover:text-gray-200'}">${icon(c.public_on_domain ? 'eye' : 'users', 'text-fine')}${c.public_on_domain ? 'Everyone' : 'Members'}</button>` : `<span class="text-fine text-gray-600">${c.visibility === 2 ? 'Killboard' : 'Public'}</span>`}<button type="button" data-domain-campaign-remove="${escape(c.campaign_id)}" title="Remove campaign" class="p-1.5 rounded text-gray-500 hover:text-red-400 hover:bg-red-500/10 transition-colors">${icon('x')}</button></div>`).join('');
    }
    function renderAppearance() {
        if (!editing)
            return;
        const d = editing;
        const theme = d.theme;
        for (const button of all('[data-domain-preset]')) {
            const active = button.dataset.domainPreset === (theme.defaultThemePreset || 'default');
            button.className = 'relative flex flex-col items-center gap-1.5 p-2.5 rounded-lg border transition-all cursor-pointer ' + (active ? 'border-blue-500/50 bg-blue-500/10' : 'border-white/[0.06] hover:border-white/[0.12] bg-white/[0.02]');
            requiredElement<HTMLElement>(button, '[data-domain-preset-active]').hidden = !active;
        }
        const opacity = Number.isFinite(Number(theme.contentOpacity)) ? Math.min(100, Math.max(20, Number(theme.contentOpacity))) : 100;
        q<HTMLInputElement>('[data-domain-opacity]').value = String(opacity);
        q('[data-domain-opacity-label]').textContent = opacity + '%';
        q('[data-domain-opacity-reset]').hidden = opacity === 100;
        for (const input of all<HTMLInputElement>('[data-domain-theme-toggle]')) {
            const key = input.dataset.domainThemeToggle;
            input.checked = key ? Boolean(theme[key]) : false;
            if (input.dataset.domainThemeToggle === 'transparentBanner')
                input.disabled = Boolean(theme.bannerUrl);
        }
        q('[data-domain-transparent-label]').className = 'flex items-center gap-2.5 text-sm cursor-pointer ' + (theme.bannerUrl ? 'text-gray-600 cursor-not-allowed' : 'text-gray-300');
        q('[data-domain-transparent-help]').textContent = theme.bannerUrl ? 'Not applicable while a banner image is set — remove the image to use it.' : 'Keeps the header without an image: the same logo, name and description sit in a banner-shaped strip, with your background bleeding through and fading into the page.';
        renderAssets();
    }
    function renderAssets() {
        if (!editing)
            return;
        const d = editing;
        for (const type of ['banner', 'logo'] as const) {
            const asset = editing[type === 'banner' ? 'bannerAsset' : 'logoAsset'], pending = asset?.status === 'pending', rejected = asset?.status === 'rejected', busy = uploading.has(type), image = d.theme[type === 'banner' ? 'bannerUrl' : 'logoUrl'];
            const disabled = busy || !d.id ? ' opacity-50 pointer-events-none' : '';
            const pendingImage = asset ? `/api/domains/preview/${Number(asset.id)}` : '';
            if (type === 'banner')
                q('[data-domain-assets="banner"]').innerHTML = (pending ? `<div class="relative rounded-lg overflow-hidden border border-yellow-500/30"><img src="${pendingImage}" alt="Pending banner" class="w-full h-32 object-cover opacity-70"><div class="absolute top-2 left-2 flex items-center gap-1.5 px-2 py-1 rounded bg-yellow-500/90 text-black text-fine font-bold uppercase tracking-wider">${icon('clock')}Pending approval</div></div>` : rejected ? `<div class="flex items-start gap-3 p-4 rounded-lg border border-red-500/30 bg-red-500/5">${icon('circle-x', 'text-lg text-red-400 flex-shrink-0')}<div class="min-w-0 flex-1"><p class="text-sm text-red-300 font-medium">Banner rejected</p>${asset.reject_reason ? `<p class="text-fine text-gray-400 mt-0.5">${escape(asset.reject_reason)}</p>` : ''}<p class="text-fine text-gray-500 mt-1">Upload a new banner below to submit it for review.</p></div></div>` : image ? `<div class="relative rounded-lg overflow-hidden border border-white/[0.08]"><img src="${url(image)}" alt="Banner preview" class="w-full h-32 object-cover"><button type="button" data-domain-asset-remove="banner" aria-label="Remove banner" ${busy ? 'disabled' : ''} class="absolute top-2 right-2 p-1.5 rounded-md bg-black/60 text-white hover:bg-red-500/80 transition-colors">${icon('trash-2')}</button></div>` : '') + (!pending ? `<button type="button" data-domain-upload="banner" ${busy || !d.id ? 'disabled' : ''} class="relative w-full flex flex-col items-center justify-center gap-2 p-6 rounded-lg border-2 border-dashed border-white/[0.08] hover:border-blue-500/30 transition-colors cursor-pointer${disabled}">${icon(busy ? 'loader-2' : 'image-plus', `text-2xl ${busy ? 'text-blue-400 animate-spin' : 'text-gray-600'}`)}<span class="text-xs text-gray-500">${busy ? 'Uploading...' : rejected ? 'Upload a replacement' : image ? 'Drop image to replace' : 'Drop image or click to upload'}</span><span class="text-fine text-gray-600">Max 4 MB · JPEG, PNG, WebP, GIF</span></button>` : '') + (!d.id ? '<p class="text-fine text-gray-600 mt-1">Save the domain first to enable image uploads</p>' : '');
            else
                q('[data-domain-assets="logo"]').innerHTML = `<div class="flex items-center gap-4">${pending ? `<div class="relative"><img src="${pendingImage}" alt="Pending logo" class="w-16 h-16 rounded-lg object-cover border border-yellow-500/30 opacity-70"><div class="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-yellow-500/90 text-black flex items-center justify-center" title="Pending approval">${icon('clock', 'text-xs')}</div></div>` : image ? `<div class="relative"><img src="${url(image)}" alt="Logo preview" class="w-16 h-16 rounded-lg object-cover border border-white/[0.08]"><button type="button" data-domain-asset-remove="logo" aria-label="Remove logo" ${busy ? 'disabled' : ''} class="absolute -top-1.5 -right-1.5 p-0.5 rounded-full bg-black/60 text-white hover:bg-red-500/80 transition-colors">${icon('x', 'text-xs')}</button></div>` : `<button type="button" data-domain-upload="logo" ${busy || !d.id ? 'disabled' : ''} class="flex flex-col items-center justify-center gap-1 w-16 h-16 rounded-lg border-2 border-dashed border-white/[0.08] hover:border-blue-500/30 transition-colors cursor-pointer${disabled}">${icon(busy ? 'loader-2' : 'plus', `text-lg ${busy ? 'text-blue-400 animate-spin' : 'text-gray-600'}`)}</button>`}<div class="flex-1 min-w-0 space-y-1">${pending ? '<p class="text-xs text-yellow-400 font-medium">Pending approval</p><p class="text-fine text-gray-500">Your logo is awaiting admin review.</p>' : rejected ? `<p class="text-xs text-red-300 font-medium">Logo rejected</p>${asset.reject_reason ? `<p class="text-fine text-gray-400">${escape(asset.reject_reason)}</p>` : ''}<p class="text-fine text-gray-500">Upload a replacement to submit it for review.</p>` : `<p class="text-xs text-gray-500">Square image works best</p><p class="text-fine text-gray-600">Max 2 MB · JPEG, PNG, WebP, GIF</p>${!d.id ? '<p class="text-fine text-gray-600 mt-1">Save first to enable uploads</p>' : ''}`}</div></div>`;
        }
        const busy = uploading.has('background');
        q('[data-domain-background-count]').textContent = d.backgrounds.length + ' / 8';
        q('[data-domain-assets="background"]').innerHTML = `<div class="grid grid-cols-2 sm:grid-cols-4 gap-2">${d.backgrounds.map(bg => `<div class="relative aspect-video rounded-lg overflow-hidden border border-white/[0.08] bg-black/20 group"><img src="/api/domains/preview/${Number(bg.id)}" alt="Background preview" class="w-full h-full object-cover ${bg.status === 'pending' ? 'opacity-50' : ''}">${bg.status === 'pending' ? '<div class="absolute top-1 left-1 px-1.5 py-0.5 rounded bg-yellow-500/80 text-fine font-medium text-black">Pending</div>' : ''}<button type="button" data-domain-background-remove="${Number(bg.id)}" aria-label="Remove background" class="absolute top-1 right-1 p-1 rounded bg-black/60 text-white opacity-0 group-hover:opacity-100 hover:bg-red-500/80 transition-all">${icon('trash-2', 'text-xs')}</button></div>`).join('')}${d.backgrounds.length < 8 ? `<button type="button" data-domain-upload="background" ${busy || !d.id ? 'disabled' : ''} class="relative flex flex-col items-center justify-center gap-1 aspect-video rounded-lg border-2 border-dashed border-white/[0.08] hover:border-blue-500/30 transition-colors cursor-pointer ${busy || !d.id ? 'opacity-50 pointer-events-none' : ''}">${icon(busy ? 'loader-2' : 'image-plus', `text-xl ${busy ? 'text-blue-400 animate-spin' : 'text-gray-600'}`)}<span class="text-fine text-gray-600">${busy ? 'Uploading...' : 'Add background'}</span></button>` : ''}</div><p class="text-fine text-gray-600">Max 6 MB each · JPEG, PNG, WebP, GIF · pending uploads need admin approval before visitors see them</p>${!d.id ? '<p class="text-fine text-gray-600">Save the domain first to enable background uploads</p>' : ''}`;
    }
    function renderWidgets() { if (!editing)
        return; const d = editing; q('[data-domain-column]').innerHTML = COLUMN_RATIO_OPTIONS.map(opt => `<option value="${opt.value}" ${d.widgets.columnRatio === opt.value ? 'selected' : ''}>${opt.label}</option>`).join(''); q('[data-domain-widgets]').innerHTML = (['top', 'left', 'right'] as const).map(section => { const rows = d.widgets[section] || [], available = ALL_WIDGET_TYPES.filter(type => type === 'textBlock' || !rows.some(w => w.type === type)); return `<div class="glass-panel p-4 space-y-3"><div class="flex items-center justify-between"><label class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">${section === 'top' ? 'Top Section (full width)' : section === 'left' ? 'Left Column (sidebar)' : 'Right Column (main)'}</label><div class="relative"><button type="button" data-domain-widget-menu="${section}" class="px-2 py-1 rounded text-fine text-blue-400 hover:bg-blue-500/10 transition-colors">+ Add Widget</button>${widgetMenu === section ? `<div class="absolute right-0 mt-1 z-10 w-48 rounded-lg bg-gray-900 border border-white/[0.1] shadow-xl max-h-60 overflow-y-auto">${available.map(type => `<button type="button" data-domain-widget-add="${section}:${type}" class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-300 hover:bg-blue-500/[0.08] transition-colors text-left">${icon(WIDGET_ICONS[type] || 'box', 'text-sm text-gray-500')}${escape(WIDGET_LABELS[type])}</button>`).join('') || '<p class="px-3 py-2 text-fine text-gray-600">All widgets added</p>'}</div>` : ''}</div></div>${rows.length ? '' : '<div class="py-4 text-center text-fine text-gray-600">No widgets in this section</div>'}${rows.map((w, i) => `<div class="flex items-center gap-2 p-2 rounded-lg ${w.enabled ? 'bg-white/[0.02]' : 'bg-white/[0.01] opacity-50'}">${icon(WIDGET_ICONS[w.type] || 'box', 'text-base text-gray-500 flex-shrink-0')}<div class="flex-1 min-w-0"><span class="text-sm text-white">${escape(WIDGET_LABELS[w.type] || w.type)}</span></div>${w.type === 'textBlock' && w.enabled ? `<div class="flex-1"><textarea data-domain-widget-field="${section}:${i}:content" placeholder="Enter text..." rows="2" class="w-full px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white placeholder-gray-600 resize-none">${escape(w.content)}</textarea></div>` : ''}${w.type === 'killList' ? `<select data-domain-widget-field="${section}:${i}:killlistType" aria-label="Kill list type" class="px-2 py-1 rounded bg-white/[0.04] border border-white/[0.08] text-xs text-white">${[['latest', 'Latest'], ['big', 'Big Kills'], ['solo', 'Solo Kills'], ['npc', 'NPC Kills']].map(([v, label]) => `<option value="${v}" ${w.killlistType === v ? 'selected' : ''}>${label}</option>`).join('')}</select>` : ''}<button type="button" data-domain-widget-move="${section}:${i}:-1" title="Move up" ${i === 0 ? 'disabled' : ''} class="p-1 text-gray-500 hover:text-white disabled:opacity-30 transition-colors">${icon('chevron-up')}</button><button type="button" data-domain-widget-move="${section}:${i}:1" title="Move down" ${i === rows.length - 1 ? 'disabled' : ''} class="p-1 text-gray-500 hover:text-white disabled:opacity-30 transition-colors">${icon('chevron-down')}</button><label class="flex items-center cursor-pointer" title="${w.enabled ? 'Disable' : 'Enable'}"><input type="checkbox" data-domain-widget-field="${section}:${i}:enabled" ${w.enabled ? 'checked' : ''} class="rounded border-white/[0.08]"></label><button type="button" data-domain-widget-remove="${section}:${i}" title="Remove" class="p-1 text-gray-500 hover:text-red-400 transition-colors">${icon('x')}</button></div>`).join('')}</div>`; }).join(''); }
    function renderNavPreview() { if (!editing)
        return; const d = editing; q('[data-domain-nav-preview]').innerHTML = '<span class="px-2 py-1 rounded text-xs text-white/80 bg-white/[0.06] flex-shrink-0">Home</span>' + d.navbar_links.map((item, i) => `<button type="button" data-domain-nav-expand="${i}" class="px-2 py-1 rounded text-xs flex-shrink-0 cursor-pointer transition-colors ${navIndex === i ? 'text-blue-400 bg-blue-500/10' : 'text-white/60 hover:text-white/80 hover:bg-white/[0.04]'}">${escape(item.label || '...')}${item.children?.length ? icon('chevron-down', 'text-[9px] ml-0.5 opacity-50 inline') : ''}</button>`).join(''); }
    function renderNav() { if (!editing)
        return; const d = editing; renderNavPreview(); q('[data-domain-nav-add]').hidden = d.navbar_links.length >= DOMAIN_NAVBAR_LIMIT; q('[data-domain-nav-items]').innerHTML = d.navbar_links.length ? d.navbar_links.map((item, i) => `<div class="rounded-lg border overflow-hidden ${navIndex === i ? 'border-blue-500/30' : 'border-white/[0.06]'}"><div class="flex items-center gap-2 p-2.5 bg-white/[0.02]"><button type="button" data-domain-nav-move="${i}:-1" ${i === 0 ? 'disabled' : ''} title="Move up" class="p-1 text-gray-500 hover:text-white disabled:opacity-30">${icon('chevron-up')}</button><button type="button" data-domain-nav-move="${i}:1" ${i === d.navbar_links.length - 1 ? 'disabled' : ''} title="Move down" class="p-1 text-gray-500 hover:text-white disabled:opacity-30">${icon('chevron-down')}</button><input data-domain-nav-field="${i}:label" value="${escape(item.label)}" placeholder="Label" class="w-24 ${inputClass}"><input data-domain-nav-field="${i}:href" value="${escape(item.href)}" placeholder="/path or https://..." class="flex-1 ${inputClass}"><label class="flex items-center gap-1 text-fine text-gray-500 cursor-pointer"><input type="checkbox" data-domain-nav-field="${i}:external" ${item.external ? 'checked' : ''} class="rounded border-white/[0.08]">Ext</label><button type="button" data-domain-nav-expand="${i}" title="${navIndex === i ? 'Collapse' : 'Edit submenu'}" class="p-1 transition-colors ${navIndex === i ? 'text-blue-400' : 'text-gray-500 hover:text-blue-400'}">${icon(navIndex === i ? 'chevron-up' : 'layers')}</button><button type="button" data-domain-nav-remove="${i}" title="Remove item" class="p-1 text-gray-500 hover:text-red-400">${icon('x')}</button></div>${navIndex === i ? `<div class="p-3 border-t border-white/[0.06] space-y-3 bg-white/[0.01]"><div class="flex items-center justify-between"><span class="text-fine text-gray-500">Dropdown groups (${item.children?.length || 0}) — multiple named groups create a mega menu</span><button type="button" data-domain-nav-group-add="${i}" class="px-2 py-1 rounded text-fine text-blue-400 hover:bg-blue-500/10 transition-colors">+ Add Group</button></div>${!item.children?.length ? '<p class="text-fine text-gray-600 py-2">No dropdown — this item is a plain link. Add a group to make it a dropdown menu.</p>' : ''}${(item.children || []).map((group, g) => `<div class="rounded-md border border-white/[0.06] p-2.5 space-y-2"><div class="flex items-center gap-2"><input data-domain-nav-field="${i}:${g}:label" value="${escape(group.label)}" placeholder="Column header (optional — leave blank for simple dropdown)" class="flex-1 ${inputClass}"><button type="button" data-domain-nav-group-remove="${i}:${g}" title="Remove group" class="p-1 text-gray-500 hover:text-red-400">${icon('x')}</button></div>${group.items.map((child, c) => `<div class="flex items-center gap-2 ml-3">${icon('corner-down-right', 'text-xs text-gray-600 flex-shrink-0')}<input data-domain-nav-field="${i}:${g}:${c}:label" value="${escape(child.label)}" placeholder="Label" class="w-24 ${inputClass}"><input data-domain-nav-field="${i}:${g}:${c}:href" value="${escape(child.href)}" placeholder="/path" class="flex-1 ${inputClass}"><button type="button" data-domain-nav-child-remove="${i}:${g}:${c}" title="Remove link" class="p-1 text-gray-500 hover:text-red-400">${icon('x', 'text-xs')}</button></div>`).join('')}<button type="button" data-domain-nav-child-add="${i}:${g}" class="ml-3 px-2 py-1 rounded text-fine text-blue-400 hover:bg-blue-500/10 transition-colors">+ Add Link</button></div>`).join('')}</div>` : ''}</div>`).join('') : '<div class="py-6 text-center text-fine text-gray-600">No custom navigation. The default menu will be shown.</div>'; }
    async function save() {
        if (saving || !editing)
            return;
        const d = editing, submitted = clone(d);
        saving = true;
        saved = false;
        showError('');
        renderSave();
        try {
            const body = d.id ? buildDomainPatch(d, baseline || {}) : { subdomain: d.subdomain, ...snapshotEditableDomain(submitted) };
            const response = d.id && !Object.keys(body).length ? null : await request<DomainUpdatePatchCompatResponses[200]>(d.id ? `/api/user/domains/${d.id}` : '/api/user/domains', { method: d.id ? 'PATCH' : 'POST', body }, 'Failed to save domain');
            if (editing !== d)
                return;
            const fresh = response ? domainDraft(response.domain) : null;
            if (fresh) {
                baseline = rebaseDomainEditor(d, fresh, submitted);
                fillFields(d);
                setURL(String(d.id));
                render();
                loadCampaigns(d);
            }
            saved = true;
            after(2000, () => { if (editing === d) {
                saved = false;
                renderSave();
            } });
            await refresh();
        }
        catch (error) {
            if (editing === d)
                showError(error);
        }
        finally {
            saving = false;
            renderSave();
        }
    }
    async function upload(type: UploadSlot, file: File) { if (!editing?.id || uploading.has(type))
        return; const d = editing; uploading.add(type); showError(''); renderAssets(); try {
        const fd = new FormData();
        fd.append('type', type);
        fd.append('file', file);
        const result = await uploadForm(`/api/user/domains/${d.id}/upload`, fd, 'Failed to upload ' + type);
        if (editing !== d)
            return;
        const asset = { id: result.assetId, status: result.status || 'pending', reject_reason: null };
        if (type === 'background')
            d.backgrounds.push(asset);
        else
            d[type === 'banner' ? 'bannerAsset' : 'logoAsset'] = asset;
        await refresh();
    }
    catch (error) {
        if (editing === d)
            showError(error);
    }
    finally {
        uploading.delete(type);
        if (editing === d)
            renderAssets();
    } }
    function input(event: Event) {
        const input = event.target;
        if (!(input instanceof HTMLInputElement || input instanceof HTMLTextAreaElement || input instanceof HTMLSelectElement) || !editing)
            return;
        const field = input.dataset.domainField;
        if (field === 'subdomain' || field === 'site_name' || field === 'site_description') {
            editing[field] = input.value;
            if (field === 'subdomain' && !editing.id)
                checkSubdomain(input.value);
            renderSave();
            return;
        }
        if (input.matches('[data-domain-entity-search]')) {
            const query = input.value;
            entityResults = [];
            renderEntities();
            search('entities', 300, async (signal, current) => { if (query.length < 2)
                return; const data = await request<{
                hits?: unknown[];
            }>('/api/search?' + new URLSearchParams({ q: query, limit: '10' }), { signal }); if (!current() || !editing)
                return; entityResults = (data.hits || []).map(domainEntity).filter((entity): entity is SiteDomainEntity => entity !== null); renderEntities(); });
            return;
        }
        if (input.matches('[data-domain-campaign-search]')) {
            const query = input.value, d = editing;
            campaignResults = [];
            q('[data-domain-campaign-loading]').hidden = true;
            renderCampaigns();
            search('campaigns', 300, async (signal, current) => { if (query.trim().length < 2 || !d.id)
                return; q('[data-domain-campaign-loading]').hidden = false; try {
                const data = await request<DomainCampaignSearchCompatResponses[200]>(`/api/user/domains/${d.id}/campaigns/search?` + new URLSearchParams({ q: query }), { signal });
                if (current() && editing === d) {
                    campaignResults = readCampaignList(data.campaigns);
                    renderCampaigns();
                }
            }
            finally {
                if (current())
                    q('[data-domain-campaign-loading]').hidden = true;
            } });
            return;
        }
        if (input.matches('[data-domain-opacity]')) {
            editing.theme.contentOpacity = Number(input.value);
            renderAppearance();
            return;
        }
        if (input.dataset.domainNavField) {
            const segments = input.dataset.domainNavField.split(':'), key = segments.pop(), link = editing.navbar_links[Number(segments[0])], group = link?.children?.[Number(segments[1])];
            const item: SiteDomainNavbarLink | SiteDomainNavbarGroup | SiteDomainNavbarItem | undefined = segments.length === 1 ? link : segments.length === 2 ? group : group?.items[Number(segments[2])];
            if (item) {
                if (key === 'label')
                    item.label = input.value;
                else if (key === 'href' && 'href' in item)
                    item.href = input.value;
                else if (key === 'external' && 'href' in item && input instanceof HTMLInputElement)
                    item.external = input.checked;
            }
            renderNavPreview();
            return;
        }
        if (input.dataset.domainWidgetField) {
            const [section, i, key] = input.dataset.domainWidgetField.split(':');
            if (!isDashboardSection(section))
                return;
            const widget = editing.widgets[section][Number(i)];
            if (!widget)
                return;
            if (key === 'enabled' && input instanceof HTMLInputElement) {
                widget.enabled = input.checked;
                renderWidgets();
            }
            else if (key === 'content' || key === 'killlistType')
                widget[key] = input.value;
        }
    }
    function change(event: Event) { const input = event.target; if (!(input instanceof HTMLInputElement || input instanceof HTMLSelectElement) || !editing)
        return; if (input instanceof HTMLInputElement && input.dataset.domainThemeToggle) {
        editing.theme[input.dataset.domainThemeToggle] = input.checked;
        renderAppearance();
    } if (input.matches('[data-domain-column]'))
        editing.widgets.columnRatio = input.value; const file = input instanceof HTMLInputElement ? input.files?.[0] : undefined; if (file && isUploadSlot(input.dataset.domainFile)) {
        void upload(input.dataset.domainFile, file);
        input.value = '';
    } }
    async function click(event: MouseEvent) {
        const button = eventElement(event.target)?.closest<HTMLButtonElement>('button');
        if (!button)
            return;
        const a = button.dataset;
        if ('domainCreate' in a) {
            open(null);
            return;
        }
        if ('domainClose' in a) {
            close();
            return;
        }
        if (a.domainEdit) {
            open(domains.find(d => d.id === Number(a.domainEdit)));
            return;
        }
        if ('domainRefresh' in a) {
            showError('');
            try {
                await refresh();
            }
            catch (error) {
                showError(error);
            }
            return;
        }
        if (a.domainDelete) {
            button.disabled = true;
            showError('');
            try {
                await request(`/api/user/domains/${a.domainDelete}`, { method: 'DELETE' }, 'Failed to delete domain');
                await refresh();
            }
            catch (error) {
                showError(error);
            }
            finally {
                button.disabled = false;
            }
            return;
        }
        if (!editing)
            return;
        if (a.domainTab) {
            tab = a.domainTab;
            render();
            return;
        }
        if ('domainSave' in a) {
            save();
            return;
        }
        if (a.domainUpload) {
            q(`[data-domain-file="${a.domainUpload}"]`).click();
            return;
        }
        if (a.domainEntityAdd) {
            const entity = entityResults[Number(a.domainEntityAdd)];
            if (entity && editing.entities.length < 5 && !editing.entities.some(e => e.type === entity.type && e.id === entity.id))
                editing.entities.push(entity);
            entityResults = [];
            q<HTMLInputElement>('[data-domain-entity-search]').value = '';
            renderEntities();
            renderSave();
            return;
        }
        if ('domainEntityRemove' in a) {
            editing.entities.splice(Number(a.domainEntityRemove), 1);
            renderEntities();
            renderSave();
            return;
        }
        if ('domainPolicy' in a) {
            editing.campaign_policy = Number(a.domainPolicy);
            renderCampaigns();
            return;
        }
        if (a.domainCampaignAdd) {
            const c = [...campaignResults, ...ownCampaigns].find(c => c.campaign_id === a.domainCampaignAdd);
            if (c && editing.campaigns.length < 50 && !editing.campaigns.some(x => x.campaign_id === c.campaign_id))
                editing.campaigns.push(clone(c));
            renderCampaigns();
            return;
        }
        if (a.domainCampaignRemove) {
            editing.campaigns = editing.campaigns.filter(c => c.campaign_id !== a.domainCampaignRemove);
            renderCampaigns();
            return;
        }
        if (a.domainCampaignAudience) {
            const c = editing.campaigns.find(c => c.campaign_id === a.domainCampaignAudience);
            if (c)
                c.public_on_domain = !c.public_on_domain;
            renderCampaigns();
            return;
        }
        if (a.domainPreset) {
            const p = THEME_PRESETS.find(p => p.id === a.domainPreset);
            if (!p)
                return;
            editing.theme.defaultThemePreset = p.id === 'default' ? undefined : p.id;
            editing.theme.defaultThemeOverrides = p.id === 'default' ? undefined : { ...p.overrides };
            renderAppearance();
            return;
        }
        if ('domainOpacityReset' in a) {
            editing.theme.contentOpacity = 100;
            renderAppearance();
            return;
        }
        if (a.domainAssetRemove || a.domainBackgroundRemove) {
            const d = editing, type = a.domainAssetRemove;
            if (type && !isAssetSlot(type))
                return;
            showError('');
            button.disabled = true;
            try {
                await request(type ? `/api/user/domains/${d.id}/upload` : `/api/user/domains/${d.id}/assets/${a.domainBackgroundRemove}`, { method: 'DELETE', ...(type ? { body: { type } } : {}) }, 'Failed to remove image');
                if (editing !== d)
                    return;
                if (type) {
                    if (isAssetSlot(type)) {
                        d.theme[type === 'banner' ? 'bannerUrl' : 'logoUrl'] = '';
                        d[type === 'banner' ? 'bannerAsset' : 'logoAsset'] = null;
                    }
                }
                else
                    d.backgrounds = d.backgrounds.filter(bg => bg.id !== Number(a.domainBackgroundRemove));
                renderAppearance();
                await refresh();
            }
            catch (error) {
                if (editing === d)
                    showError(error);
            }
            finally {
                button.disabled = false;
            }
            return;
        }
        if (a.domainWidgetMenu) {
            widgetMenu = widgetMenu === a.domainWidgetMenu ? null : a.domainWidgetMenu;
            renderWidgets();
            return;
        }
        if (a.domainWidgetAdd) {
            const [section, type] = a.domainWidgetAdd.split(':');
            if (!isDashboardSection(section) || !isDomainWidgetType(type))
                return;
            editing.widgets[section].push({ type, enabled: true, ...(type === 'textBlock' ? { content: '' } : {}), ...(type === 'killList' ? { killlistType: 'latest' } : {}) });
            widgetMenu = null;
            renderWidgets();
            return;
        }
        if (a.domainWidgetRemove) {
            const [section, i] = a.domainWidgetRemove.split(':');
            if (!isDashboardSection(section))
                return;
            editing.widgets[section].splice(Number(i), 1);
            renderWidgets();
            return;
        }
        if (a.domainWidgetMove) {
            const [section, index, dir] = a.domainWidgetMove.split(':');
            if (!isDashboardSection(section))
                return;
            const rows = editing.widgets[section], i = Number(index), to = i + Number(dir);
            if (to >= 0 && to < rows.length)
                [rows[i], rows[to]] = [rows[to], rows[i]];
            renderWidgets();
            return;
        }
        if ('domainNavReset' in a)
            editing.navbar_links = clone(DEFAULT_NAVBAR);
        else if ('domainNavClear' in a)
            editing.navbar_links = [];
        else if ('domainNavAdd' in a) {
            if (editing.navbar_links.length < DOMAIN_NAVBAR_LIMIT)
                editing.navbar_links.push({ label: '', href: '/' });
        }
        else if ('domainNavExpand' in a)
            navIndex = navIndex === Number(a.domainNavExpand) ? null : Number(a.domainNavExpand);
        else if ('domainNavRemove' in a)
            editing.navbar_links.splice(Number(a.domainNavRemove), 1);
        else if (a.domainNavMove) {
            const [index, dir] = a.domainNavMove.split(':'), rows = editing.navbar_links, i = Number(index), to = i + Number(dir);
            if (to >= 0 && to < rows.length)
                [rows[i], rows[to]] = [rows[to], rows[i]];
        }
        else if ('domainNavGroupAdd' in a) {
            const item = editing.navbar_links[Number(a.domainNavGroupAdd)];
            item.children ||= [];
            item.children.push({ label: '', items: [] });
        }
        else if (a.domainNavGroupRemove) {
            const [i, g] = a.domainNavGroupRemove.split(':');
            editing.navbar_links[Number(i)]?.children?.splice(Number(g), 1);
        }
        else if (a.domainNavChildAdd) {
            const [i, g] = a.domainNavChildAdd.split(':');
            editing.navbar_links[Number(i)]?.children?.[Number(g)]?.items.push({ label: '', href: '/' });
        }
        else if (a.domainNavChildRemove) {
            const [i, g, c] = a.domainNavChildRemove.split(':');
            editing.navbar_links[Number(i)]?.children?.[Number(g)]?.items.splice(Number(c), 1);
        }
        else
            return;
        renderNav();
    }
    function dragover(event: DragEvent) { if (eventElement(event.target)?.closest('[data-domain-upload]'))
        event.preventDefault(); }
    function drop(event: DragEvent) { const zone = eventElement(event.target)?.closest<HTMLElement>('[data-domain-upload]'); if (!zone)
        return; event.preventDefault(); const file = event.dataTransfer?.files[0]; if (file && file.type.startsWith('image/') && isUploadSlot(zone.dataset.domainUpload))
        void upload(zone.dataset.domainUpload, file); }
    function visibility(event: Event) { const shown = event instanceof CustomEvent && event.detail === true; if (!shown) {
        suspend();
    }
    else if (editing) {
        setURL(editing.id ? String(editing.id) : 'new');
        if (!editing.id && !availability && editing.subdomain)
            checkSubdomain(editing.subdomain);
    } }
    const edit = new URL(location.href).searchParams.get('edit');
    if (edit === 'new')
        open(null);
    else if (edit && domains.some(d => d.id === Number(edit)))
        open(domains.find(d => d.id === Number(edit)));
    else
        render();
    return { input, change, click, dragover, drop, visibility, suspend };
}
export class SettingsDomainsController extends SettingsComponent {
    private runtime: ReturnType<typeof domainRuntime> | undefined;
    private section: HTMLElement | null = null;
    private visibilityListener = (event: Event) => this.runtime?.visibility(event);
    connect() {
        this.runtime ??= domainRuntime(this.element, { request: <T>(path: string, options?: JSONRequestOptions, fallback?: string) => this.request<T>(path, options, fallback), upload: (path, body, fallback) => this.requestForm<DomainAssetUploadCompatResponses[200]>(path, body, fallback), after: (delay, run) => { this.after(delay, run); } });
        this.section = this.element.closest('[data-settings-section]');
        this.section?.addEventListener('settings-visibility', this.visibilityListener);
    }
    input(event: Event) { this.runtime?.input(event); }
    change(event: Event) { this.runtime?.change(event); }
    click(event: MouseEvent) { return this.runtime?.click(event); }
    dragover(event: DragEvent) { this.runtime?.dragover(event); }
    drop(event: DragEvent) { this.runtime?.drop(event); }
    hide() { this.runtime?.suspend(); super.hide(); }
    show(event: PageTransitionEvent) { if (event.persisted)
        this.runtime?.visibility(new CustomEvent('settings-visibility', { detail: !this.section?.hidden })); }
    disconnect() { this.section?.removeEventListener('settings-visibility', this.visibilityListener); this.section = null; super.disconnect(); }
}
registerController('settings-domains', SettingsDomainsController);
