import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {primeAIIDAudio,playAIIDAlarm} from './map-aiid-audio'
import {loadMapCanvas} from './shared/map-module'
import type {createMapLiveFeed} from './map-live'
import {readMapPageData,mapPageState,isMapWindow,isMapToggle,isSpecialMapBase,type MapPageState,type MapPageData} from './shared/map-page'
import {isMapScope} from './shared/map-scope'
import {isMapRenderBaseLayer,isMapActivityLayer} from '../utils/map/layers'
import type {GoMapController,SovereigntyAllianceSummary,MapRegionSummary} from '../utils/map/goFrontendCanvas'
import {requiredElement,eventElement} from './shared/dom'
import {readSearchResults} from './shared/search-model'
import {isAbortError,errorMessage} from './shared/json'
export class MapPageController extends RequestComponent {
 private readonly root = this.element;
 private alive = false;
 private generation = 0;
 private surface = this.q<HTMLElement>('[data-map-canvas]');
 private status = this.q<HTMLElement>('[data-map-status]');
 private search = this.q<HTMLInputElement>('[data-map-search]');
 private searchResults = this.q<HTMLElement>('[data-map-search-results]');
 private watchSearch = this.q<HTMLInputElement>('[data-map-aiid-search]');
 private watchResults = this.q<HTMLElement>('[data-map-aiid-results]');
 private watchChips = this.q<HTMLElement>('[data-map-aiid-chips]');
 private watchCoverage = this.q<HTMLElement>('[data-map-aiid-coverage]');
 private sovereigntySearch = this.q<HTMLInputElement>('[data-map-sovereignty-search]');
 private sovereigntyResults = this.q<HTMLElement>('[data-map-sovereignty-results]');
 private sovereigntyList = this.q<HTMLElement>('[data-map-sovereignty-list]');
 private sovereigntyHover = this.q<HTMLElement>('[data-map-sovereignty-hover]');
 private regionHover = this.q<HTMLElement>('[data-map-region-hover]');
 private legend = this.q<HTMLElement>('[data-map-legend]');
 private specialBases = new Set<string>(['sovereignty','live','aiid']);
 private state:MapPageState = mapPageState(new URLSearchParams(location.search));
 private renderer:GoMapController|null = null;
 private liveFeed:ReturnType<typeof createMapLiveFeed>|null = null;
 private scopeData:MapPageData|null = null;
 private controller:AbortController|null = null;
 private searchTimer:ReturnType<typeof setTimeout>|undefined;
 private watchSearchTimer:ReturnType<typeof setTimeout>|undefined;
 private watchSearchRequest:AbortController|undefined;
 private watchNames = new Map<number,string>();
 private activityWindowLabel = () => this.state.hours === 168 ? '7 days' : `${this.state.hours} hour${this.state.hours === 1 ? '' : 's'}`;
 private parseURL = ()=>mapPageState(new URLSearchParams(location.search),this.state);
 private url = () => {
    const q = new URLSearchParams({ scope: this.state.scope, base: this.state.base, activity: this.state.activity, hours: String(this.state.hours) });
    if (!this.state.routes)
        q.set('routes', '0');
    if (!this.state.systems)
        q.set('systems', '0');
    if (!this.state.labels)
        q.set('labels', '0');
    if (this.state.base === 'sovereignty' && this.state.changes)
        q.set('changes', '1');
    if (this.state.base === 'aiid' && this.state.watch.length)
        q.set('watch', this.state.watch.join(','));
    return `/map?${q}`;
};
 private regionURL = (id:number) => {
    const q = new URLSearchParams({ base: this.state.base, activity: this.state.activity, hours: String(this.state.hours) });
    if (!this.state.routes)
        q.set('routes', '0');
    if (!this.state.systems)
        q.set('systems', '0');
    if (!this.state.labels)
        q.set('labels', '0');
    return `/map/region/${id}?${q}`;
};
 private sync = () => {
    this.root.dataset.mapCurrentBase = this.state.base;
    this.q<HTMLElement>('[data-map-live]').hidden = this.state.base !== 'live';
    this.q<HTMLElement>('[data-map-live-status]').hidden = this.state.base !== 'live';
    this.q<HTMLElement>('[data-map-aiid]').hidden = this.state.base !== 'aiid';
    this.q<HTMLElement>('[data-map-aiid-status]').hidden = this.state.base !== 'aiid';
    this.q<HTMLElement>('[data-map-sovereignty]').hidden = this.state.base !== 'sovereignty';
    this.q<HTMLElement>('[data-map-sovereignty-status]').hidden = this.state.base !== 'sovereignty' || !this.scopeData;
    if (this.state.base !== 'sovereignty')
        this.sovereigntyHover.hidden = true;
    if (this.specialBases.has(this.state.base))
        this.regionHover.hidden = true;
    this.q<HTMLElement>('[data-map-general-status]').hidden = !this.scopeData || this.specialBases.has(this.state.base);
    for (const button of this.root.querySelectorAll<HTMLElement>('[data-map-base-option]')) {
        const active = button.dataset.mapBaseOption === this.state.base;
        button.classList.toggle('active', active);
        button.setAttribute('aria-pressed', String(active));
        button.hidden = this.state.scope !== 'new-eden' && isSpecialMapBase(button.dataset.mapBaseOption);
    }
    for (const button of this.root.querySelectorAll<HTMLButtonElement>('[data-map-activity-option]')) {
        const active = button.dataset.mapActivityOption === this.state.activity;
        button.classList.toggle('active', active);
        button.setAttribute('aria-pressed', String(active));
    }
    for (const button of this.root.querySelectorAll<HTMLButtonElement>('[data-map-scope-option]')) {
        const active = button.dataset.mapScopeOption === this.state.scope;
        button.classList.toggle('active', active);
        button.setAttribute('aria-pressed', String(active));
    }
    for (const button of this.root.querySelectorAll<HTMLButtonElement>('[data-map-hours-option]')) {
        const active = Number(button.dataset.mapHoursOption) === this.state.hours;
        button.classList.toggle('active', active);
        button.setAttribute('aria-pressed', String(active));
        button.disabled = this.state.activity === 'none';
    }
    for (const button of this.root.querySelectorAll<HTMLButtonElement>('[data-map-toggle]')) {
        const toggle = button.dataset.mapToggle;if(!isMapToggle(toggle))continue;
        const active = this.state[toggle];
        button.classList.toggle('active', active);
        button.setAttribute('aria-pressed', String(active));
    }
    const changes = this.q<HTMLElement>('[data-map-changes]');
    changes.hidden = this.state.base !== 'sovereignty';
    changes.classList.toggle('active', this.state.changes);
    changes.setAttribute('aria-pressed', String(this.state.changes));
    this.q<HTMLElement>('[data-map-alarms]').hidden = this.state.base !== 'aiid';
    for (const button of this.root.querySelectorAll<HTMLButtonElement>('[data-map-alarm]')) {
        const active = button.dataset.mapAlarm === 'near' ? this.state.nearAlarm : this.state.outerAlarm;
        button.classList.toggle('active', active);
        button.setAttribute('aria-pressed', String(active));
        button.title = active ? 'Disable alarm' : button.dataset.mapAlarm === 'near'
            ? 'Sound an urgent alarm for kills fewer than five jumps away'
            : 'Sound a scanner alert for kills five to ten jumps away';
    }
    requiredElement(this.legend,'[data-map-legend-base]').textContent = this.state.base === 'security' ? 'System security'
        : this.state.base === 'sovereignty' ? 'Alliance territory' : this.state.base === 'live' ? 'Live killmail pulse'
            : this.state.base === 'aiid' ? 'Watched system / danger pulse' : 'Region grouping';
    requiredElement(this.legend,'[data-map-legend-changes]').hidden = this.state.base !== 'sovereignty' || !this.state.changes;
    const activity = this.renderer?.activityLegend();
    requiredElement(this.legend,'[data-map-legend-activity]').hidden = this.state.activity === 'none';
    requiredElement(this.legend,'[data-map-activity-label]').textContent = activity?.label || this.state.activity;
    requiredElement(this.legend,'[data-map-activity-peak]').textContent = `Peak ${activity?.peak || '0'}`;
    requiredElement(this.legend,'[data-map-activity-window]').textContent = `last ${this.activityWindowLabel()} · log scale`;
    for (const hourly of this.root.querySelectorAll<HTMLElement>('[data-map-hourly]')) {
        hourly.hidden = this.state.activity === 'none';
        hourly.textContent = `Hourly data · ${this.activityWindowLabel()}`;
    }
    this.syncWatch();
};
 private syncWatch = () => {
    this.watchChips.replaceChildren();
    for (const id of this.state.watch) {
        const button = document.createElement('button');
        button.type = 'button';
        button.dataset.mapWatchRemove = String(id);
        button.title = `Stop watching ${this.watchNames.get(id) || `System ${id}`}`;
        const name = document.createElement('span');
        name.textContent = this.watchNames.get(id) || `System ${id}`;
        const close = document.createElement('b');
        close.textContent = '×';
        close.setAttribute('aria-hidden', 'true');
        button.append(name, close);
        this.watchChips.append(button);
    }
    this.watchChips.hidden = !this.state.watch.length;
    this.watchSearch.disabled = this.state.watch.length >= 8;
    const count = this.scopeData?.scope === 'aiid' ? this.scopeData.systems?.length || 0 : 0;
    const regions = this.scopeData?.scope === 'aiid' ? this.scopeData.regions?.length || 0 : 0;
    const systemsLabel = document.createElement('span');
    systemsLabel.textContent = `${count.toLocaleString('en-US')} systems covered`;
    const regionsLabel = document.createElement('span');
    regionsLabel.textContent = `${regions} regions`;
    this.watchCoverage.replaceChildren(systemsLabel, regionsLabel);
    this.watchCoverage.hidden = !this.state.watch.length;
};
 private renderSovereignty = (data:MapPageData) => {
    const renderer = this.renderer;if(!renderer)return;
    const alliances = renderer.allianceSummaries();
    const topSystems = alliances[0]?.systems || 1;
    this.q<HTMLElement>('[data-map-sovereignty-alliance-count]').textContent = String(alliances.length);
    this.q<HTMLElement>('[data-map-sovereignty-claimed]').textContent = `${(data.sovereignty || []).length.toLocaleString('en-US')} systems claimed`;
    const snapshot = data.snapshot_at ? ` · ${new Date(data.snapshot_at).toLocaleString('en-US', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}` : '';
    this.q<HTMLElement>('[data-map-sovereignty-summary]').textContent = `${(data.sovereignty || []).length.toLocaleString('en-US')} claimed · ${alliances.length} alliances${snapshot}`;
    const rows = document.createDocumentFragment();
    alliances.forEach((alliance, index) => {
        const color = renderer.allianceColor(alliance.id);
        const row = document.createElement('a');
        row.className = 'map-sovereignty-row';
        row.href = `/alliance/${alliance.id}`;
        const rank = document.createElement('span');
        rank.className = 'map-sovereignty-rank';
        rank.textContent = String(index + 1);
        const logo = document.createElement('span');
        logo.className = 'map-sovereignty-logo';
        const image = document.createElement('img');
        image.src = `https://images.evetech.net/alliances/${alliance.id}/logo?size=64`;
        image.alt = '';
        image.loading = 'lazy';
        image.width = 32;
        image.height = 32;
        const accent = document.createElement('i');
        accent.style.backgroundColor = color;
        logo.append(image, accent);
        const identity = document.createElement('span');
        identity.className = 'map-sovereignty-name';
        const name = document.createElement('strong');
        name.textContent = alliance.name;
        const ticker = document.createElement('small');
        ticker.textContent = `[${alliance.ticker}]`;
        identity.append(name, ticker);
        const total = document.createElement('span');
        total.className = 'map-sovereignty-total';
        const value = document.createElement('strong');
        value.textContent = String(alliance.systems);
        const unit = document.createElement('small');
        unit.textContent = 'systems';
        total.append(value, unit);
        const bar = document.createElement('span');
        bar.className = 'map-sovereignty-bar';
        bar.style.color = color;
        bar.style.backgroundColor = color;
        bar.style.width = `${Math.max(2, alliance.systems / topSystems * 100)}%`;
        row.append(rank, logo, identity, total, bar);
        row.dataset.mapAlliance = String(alliance.id);
        rows.append(row);
    });
    this.sovereigntyList.replaceChildren(rows);
};
 private positionHover = (card:HTMLElement, point:{x:number;y:number}, height:number) => {
    const area = this.q<HTMLElement>('[data-map-surface]');
    const width = 280;
    card.style.left = `${Math.max(8, point.x + width + 24 > area.clientWidth ? point.x - width - 14 : point.x + 14)}px`;
    card.style.top = `${Math.max(8, point.y + height + 24 > area.clientHeight ? point.y - height - 14 : point.y + 14)}px`;
};
 private showSovereigntyHover = (alliance:SovereigntyAllianceSummary|null, point:{x:number;y:number}) => {
    if (!alliance || this.state.base !== 'sovereignty') {
        this.sovereigntyHover.hidden = true;
        return;
    }
    this.positionHover(this.sovereigntyHover, point, 150);
    const logo = requiredElement<HTMLImageElement>(this.sovereigntyHover,'[data-map-hover-logo]');
    const src = `https://images.evetech.net/alliances/${alliance.id}/logo?size=64`;
    if (logo.getAttribute('src') !== src)
        logo.src = src;
    logo.alt = alliance.name;
    requiredElement(this.sovereigntyHover,'[data-map-hover-name]').textContent = alliance.name;
    requiredElement(this.sovereigntyHover,'[data-map-hover-ticker]').textContent = `[${alliance.ticker}] · Click to open alliance`;
    requiredElement(this.sovereigntyHover,'[data-map-hover-systems]').textContent = String(alliance.systems);
    requiredElement(this.sovereigntyHover,'[data-map-hover-regions]').textContent = String(alliance.regions);
    requiredElement(this.sovereigntyHover,'[data-map-hover-members]').textContent = alliance.memberCount > 0
        ? `${alliance.memberCount.toLocaleString('en-US')} members` : 'Current sovereignty';
    const changes = requiredElement(this.sovereigntyHover,'[data-map-hover-changes]');
    changes.textContent = alliance.recentChanges ? `${alliance.recentChanges} changed in 7d` : 'No recent changes';
    changes.classList.toggle('has-changes', !!alliance.recentChanges);
    this.sovereigntyHover.hidden = false;
};
 private showRegionHover = (region:MapRegionSummary|null, point:{x:number;y:number}) => {
    if (!region || this.specialBases.has(this.state.base)) {
        this.regionHover.hidden = true;
        return;
    }
    this.positionHover(this.regionHover, point, 190);
    requiredElement(this.regionHover,'[data-map-region-name]').textContent = region.name;
    requiredElement(this.regionHover,'[data-map-region-systems]').textContent = `${region.systems} systems`;
    requiredElement(this.regionHover,'[data-map-region-window]').textContent = `last ${this.activityWindowLabel()}`;
    for (const metric of this.regionHover.querySelectorAll<HTMLElement>('[data-map-region-metric]'))
        if(['ship_kills','pod_kills','npc_kills','ship_jumps'].some(key=>key === metric.dataset.mapRegionMetric)) {const key = metric.dataset.mapRegionMetric;if(key === 'ship_kills' || key === 'pod_kills' || key === 'npc_kills' || key === 'ship_jumps')metric.textContent = region[key].toLocaleString('en-US');}
    for (const type of ['high', 'low', 'null'] as const)
        requiredElement(this.regionHover,`[data-map-region-${type}]`).textContent = String(region[type]);
    const hotspot = requiredElement(this.regionHover,'[data-map-region-hotspot]');
    hotspot.replaceChildren();
    if (region.busiest) {
        const name = document.createElement('b');
        name.textContent = region.busiest.system_name;
        hotspot.append(document.createTextNode('Hotspot: '), name);
    }
    this.regionHover.hidden = false;
};
 private persist = () => history.replaceState(history.state, '', this.url());
 private drawLayers = () => {
    this.regionHover.hidden = true;
    this.renderer?.setLayers({ baseLayer: this.state.base, activityLayer: this.state.activity,
        showConnections: this.state.routes, showSystems: this.state.systems, showLabels: this.state.labels,
        showChanges: this.state.changes });
    this.sync();
    this.persist();
};
 private load = async () => {
    if(!this.alive)return;
    const generation = ++this.generation;
    this.controller?.abort();
    // Changing watch anchors retains the relay, streamed rows and freshness
    // history, just as PixiScopeView does. A mode change gets a new feed.
    if (this.liveFeed && this.liveFeed.mode !== this.state.base) {
        this.liveFeed.destroy();
        this.liveFeed = null;
    }
    this.scopeData = null;
    this.sovereigntyHover.hidden = true;
    this.regionHover.hidden = true;
    this.q<HTMLElement>('[data-map-general-status]').hidden = true;
    this.q<HTMLElement>('[data-map-sovereignty-status]').hidden = true;
    const request = new AbortController();
    this.controller = request;
    this.status.hidden = false;
    this.status.textContent = 'Loading map...';
    try {
        const modulePromise = this.importCanvas();
        const liveModulePromise = this.state.base === 'live' || this.state.base === 'aiid' ? import('./map-live') : null;
        const endpoint = this.state.base === 'sovereignty'
            ? `/api/map/sovereignty?hours=${this.state.hours}`
            : this.state.base === 'aiid'
                ? `/api/map/aiid?${new URLSearchParams({ systems: this.state.watch.join(','), hours: String(this.state.hours) })}`
                : `/api/map/scope?${new URLSearchParams({ type: this.state.scope, hours: String(this.state.hours) })}`;
        const [module,raw,liveModule] = await Promise.all([modulePromise,this.request<unknown>(endpoint,{signal:request.signal},'Unable to load map data'),liveModulePromise]);
        const data=readMapPageData(raw);
        if (!this.alive || this.generation !== generation || this.controller !== request)
            return;
        this.scopeData = data;
        this.q<HTMLElement>('[data-map-general-count]').textContent = `${(data.systems || []).length.toLocaleString('en-US')} systems · ${(data.regions || []).length} regions`;
        if (this.state.base === 'aiid')
            for (const anchor of data.anchors || [])
                this.watchNames.set(anchor.solar_system_id, anchor.system_name);
        this.renderer?.destroy();
        this.renderer = module.createGoMap(this.surface, {
            baseLayer: this.state.base, activityLayer: this.state.activity,
            showConnections: this.state.routes, showSystems: this.state.systems, showLabels: this.state.labels,
            showChanges: this.state.changes,
            panel: this.state.base === 'live' ? this.q<HTMLElement>('[data-map-live]')
                : this.state.base === 'aiid' ? this.q<HTMLElement>('[data-map-aiid]')
                    : this.state.base === 'sovereignty' ? this.q<HTMLElement>('[data-map-sovereignty]') : null,
            onBattleRegion: id => {if(this.current(request,generation))this.navigate(this.regionURL(id))},
            onAlliance: id => {if(this.current(request,generation))this.navigate(`/alliance/${id}`)},
            onHoverAlliance: (alliance,point)=>{if(this.current(request,generation))this.showSovereigntyHover(alliance,point)},
            onHoverRegion: (region,point)=>{if(this.current(request,generation))this.showRegionHover(region,point)},
        });
        this.renderer.setData(data);
        if (this.state.base === 'sovereignty')
            this.renderSovereignty(data);
        this.status.hidden = true;
        this.sync();
        if (liveModule) {
            if (!this.alive || this.generation !== generation || this.controller !== request)
                return;
            if (this.liveFeed)
                this.liveFeed.update(this.renderer, data);
            else if (this.state.base === 'live')
                this.liveFeed = liveModule.createMapLiveFeed(this.root, this.renderer, data);
            else if (this.controller === request && this.state.base === 'aiid')
                this.liveFeed = liveModule.createMapAIIDFeed(this.root, this.renderer, data, band => {
                    if (this.alive && (band === 'near' ? this.state.nearAlarm : this.state.outerAlarm))
                        playAIIDAlarm(band);
                });
        }
    }
    catch (error) {
        if (this.alive && this.generation === generation && this.controller === request && !isAbortError(error)) {
            this.status.textContent = errorMessage(error,'Unable to load map data');
            this.status.hidden = false;
        }
    }
};
 private searchMap = (value:string) => {
    this.searchResults.replaceChildren();
    if (!this.scopeData || !value) {
        this.searchResults.hidden = true;
        return;
    }
    const term = value.toLocaleLowerCase();
    const regionNames = new Map((this.scopeData.regions || []).map(region => [region.region_id, region.name]));
    const regions = (this.scopeData.regions || []).filter(region => region.name?.toLocaleLowerCase().includes(term)).slice(0, 5)
        .map(region => ({ name: region.name, detail: 'Region', href: this.regionURL(region.region_id) }));
    const systems = (this.scopeData.systems || []).filter(system => system.system_name?.toLocaleLowerCase().includes(term)).slice(0, Math.max(0, 8 - regions.length))
        .map(system => ({ name: system.system_name, detail: regionNames.get(system.region_id) || 'System', href: `/system/${system.solar_system_id}` }));
    for (const hit of [...regions, ...systems]) {
        const link = document.createElement('a');
        link.href = hit.href;
        const name = document.createElement('span');
        name.textContent = hit.name;
        const detail = document.createElement('small');
        detail.textContent = hit.detail;
        link.append(name, detail);
        this.searchResults.append(link);
    }
    if (!this.searchResults.childElementCount) {
        const empty = document.createElement('p');
        empty.textContent = 'No matching regions or systems';
        this.searchResults.append(empty);
    }
    this.searchResults.hidden = false;
};
 private searchWatch = async (value:string) => {
    this.watchSearchRequest?.abort();
    this.watchResults.replaceChildren();
    const query = value.trim();
    if (query.length < 2 || this.state.watch.length >= 8) {
        this.watchResults.hidden = true;
        return;
    }
    const request = new AbortController();
    this.watchSearchRequest = request;
    try {
        const data = readSearchResults(await this.request<unknown>(`/api/search?${new URLSearchParams({q:query,type:'system',limit:'10'})}`,{signal:request.signal}));
        if (!this.alive || this.watchSearchRequest !== request || this.watchSearch.value.trim() !== query || this.state.watch.length >= 8)
            return;
        for (const hit of data.hits || []) {
            if (hit.type !== 'system')
                continue;
            const id = Number(String(hit.id).replace(/^[a-z]+_/i, ''));
            if (!Number.isInteger(id) || id < 1)
                continue;
            const button = document.createElement('button');
            button.type = 'button';
            button.dataset.mapWatchAdd = String(id);
            button.dataset.mapWatchName = hit.name ?? `System ${id}`;
            button.disabled = this.state.watch.includes(id);
            const icon = document.createElement('span');
            icon.textContent = '⌖';
            icon.setAttribute('aria-hidden', 'true');
            const detail = document.createElement('span');
            detail.textContent = hit.name ?? `System ${id}`;
            const type = document.createElement('small');
            type.textContent = 'system';
            detail.append(type);
            button.append(icon, detail);
            this.watchResults.append(button);
        }
        this.watchResults.hidden = !this.watchResults.childElementCount;
    }
    catch (error) {
        if(this.alive && this.watchSearchRequest === request && !isAbortError(error))this.watchResults.hidden = true;
    }
};
 private searchSovereignty = (value:string) => {
    this.sovereigntyResults.replaceChildren();
    const term = value.trim().toLocaleLowerCase();
    if (!term || !this.scopeData || this.state.base !== 'sovereignty') {
        this.sovereigntyResults.hidden = true;
        return;
    }
    const data = this.scopeData;if(!data)return;
    const alliances = this.renderer?.allianceSummaries() || [];
    const matches = alliances.filter(alliance => alliance.name.toLocaleLowerCase().includes(term) || alliance.ticker.toLocaleLowerCase().includes(term)).slice(0, 5)
        .map(alliance => ({ name: alliance.name, detail: `[${alliance.ticker}] Alliance`, href: `/alliance/${alliance.id}` }));
    const systems = (this.scopeData.systems || []).filter(system => system.system_name.toLocaleLowerCase().includes(term)).slice(0, Math.max(0, 8 - matches.length))
        .map(system => ({ name: system.system_name, detail: (data.sovereignty || []).find(claim => claim.system_id === system.solar_system_id)?.alliance_name || 'System', href: `/system/${system.solar_system_id}` }));
    for (const result of [...matches, ...systems]) {
        const link = document.createElement('a');
        link.href = result.href;
        const name = document.createElement('span');
        name.textContent = result.name;
        const detail = document.createElement('small');
        detail.textContent = result.detail;
        link.append(name, detail);
        this.sovereigntyResults.append(link);
    }
    if (!this.sovereigntyResults.childElementCount) {
        const empty = document.createElement('p');
        empty.textContent = 'No matching alliances or systems';
        this.sovereigntyResults.append(empty);
    }
    this.sovereigntyResults.hidden = false;
};
 click = (event:MouseEvent) => {
    const button = eventElement(event.target)?.closest<HTMLButtonElement>('button');
 if(!this.alive)return;
    if (!button || !this.root.contains(button) || button.disabled)
        return;
    if (button.matches('[data-map-base-option]')) {
        const base = button.dataset.mapBaseOption;if(!isMapRenderBaseLayer(base))return;
        if(this.state.base === base || this.state.scope !== 'new-eden' && isSpecialMapBase(base))return;
        const specialChange = this.specialBases.has(base) || this.specialBases.has(this.state.base);
        this.state.base = base;
        this.sync();
        this.persist();
        if (specialChange)
            this.load();
        else
            this.drawLayers();
    }
    else if (button.matches('[data-map-activity-option]')) {
        const activity = button.dataset.mapActivityOption;if(!isMapActivityLayer(activity))return;this.state.activity = activity;
        this.drawLayers();
    }
    else if (button.matches('[data-map-scope-option]')) {
        const scope = button.dataset.mapScopeOption;if(!isMapScope(scope))return;
        if (this.state.scope === scope)
            return;
        this.state.scope = scope;
        if (scope !== 'new-eden' && this.specialBases.has(this.state.base))
            this.state.base = 'geography';
        this.sync();
        this.persist();
        this.load();
    }
    else if (button.matches('[data-map-hours-option]')) {
        const hours = Number(button.dataset.mapHoursOption);if(!isMapWindow(hours))return;this.state.hours = hours;
        this.sync();
        this.persist();
        this.load();
    }
    else if (button.matches('[data-map-toggle]')) {
        const key = button.dataset.mapToggle;if(!isMapToggle(key))return;this.state[key] = !this.state[key];
        this.drawLayers();
    }
    else if (button.matches('[data-map-changes]')) {
        this.state.changes = !this.state.changes;
        this.drawLayers();
    }
    else if (button.matches('[data-map-alarm]')) {
        const key = button.dataset.mapAlarm === 'near' ? 'nearAlarm' : 'outerAlarm';
        this.state[key] = !this.state[key];
        this.sync();
        if (this.state[key])
            void primeAIIDAudio();
    }
    else if (button.matches('[data-map-watch-add]')) {
        const id = Number(button.dataset.mapWatchAdd);
        if (!Number.isSafeInteger(id) || id < 1 || id > 2147483647 || this.state.watch.includes(id) || this.state.watch.length >= 8)
            return;
        this.stopTimer(this.watchSearchTimer);
        this.watchSearchRequest?.abort();
        this.state.watch = [...this.state.watch, id];
        this.watchNames.set(id, button.dataset.mapWatchName ?? `System ${id}`);
        this.watchSearch.value = '';
        this.watchResults.hidden = true;
        this.watchResults.replaceChildren();
        this.sync();
        this.persist();
        this.load();
    }
    else if (button.matches('[data-map-watch-remove]')) {
        this.state.watch = this.state.watch.filter(id => id !== Number(button.dataset.mapWatchRemove));
        this.sync();
        this.persist();
        this.load();
    }
    else if (button.matches('[data-map-reset]')) {
        this.renderer?.resetView();
    }
};
 private current(request:AbortController,generation:number){return this.alive && this.element.isConnected && this.controller === request && this.generation === generation}
 navigate(path:string){location.assign(path)}
 connect(){this.alive = true;this.sync();void this.load()}
 importCanvas(){return loadMapCanvas()}
 input(event:Event){const input = eventElement(event.target);if(!this.alive)return;if(input === this.search){this.stopTimer(this.searchTimer);this.searchTimer=this.after(100,()=>this.searchMap(this.search.value.trim()))}else if(input === this.watchSearch){this.stopTimer(this.watchSearchTimer);this.watchSearchRequest?.abort();this.watchSearchTimer=this.after(300,()=>{void this.searchWatch(this.watchSearch.value)})}else if(input === this.sovereigntySearch)this.searchSovereignty(this.sovereigntySearch.value)}
 focus(event:FocusEvent){if(!this.alive)return;const input=eventElement(event.target);this.focusRow(event);if(input===this.search&&this.searchResults.childElementCount)this.searchResults.hidden=false;else if(input===this.watchSearch&&this.watchResults.childElementCount)this.watchResults.hidden=false;else if(input===this.sovereigntySearch&&this.sovereigntySearch.value)this.searchSovereignty(this.sovereigntySearch.value)}
 blur(event:FocusEvent){
  this.clearRow(event);
  const target=eventElement(event.target),results=target===this.search?this.searchResults:target===this.watchSearch?this.watchResults:target===this.sovereigntySearch?this.sovereigntyResults:target?.closest<HTMLElement>('[data-map-search-results],[data-map-aiid-results],[data-map-sovereignty-results]');
  if(!results || event.relatedTarget instanceof Node && results.contains(event.relatedTarget))return;
  const input=results===this.searchResults?this.search:results===this.watchResults?this.watchSearch:this.sovereigntySearch;
  this.after(150,()=>{const focused=this.element.ownerDocument.activeElement;if(this.alive && focused!==input && !(focused && results.contains(focused)))results.hidden=true});
 }
 back(){if(!this.alive)return;this.state=this.parseURL();this.sync();void this.load()}
 hide(){this.alive=false;this.generation++;this.controller?.abort();this.controller=null;this.watchSearchRequest?.abort();this.watchSearchRequest=undefined;this.stopTimer(this.searchTimer);this.stopTimer(this.watchSearchTimer);this.stopTimers();this.liveFeed?.destroy();this.liveFeed=null;this.renderer?.destroy();this.renderer=null;for(const node of [this.searchResults,this.watchResults,this.sovereigntyResults,this.regionHover,this.sovereigntyHover])node.hidden=true;super.hide()}
 show(event:PageTransitionEvent){if(event.persisted){this.alive=true;void this.load()}}
 private focusRow(event:FocusEvent|PointerEvent){if(!this.alive)return;const row=eventElement(event.target)?.closest<HTMLElement>('[data-map-alliance]');if(!row || event.relatedTarget instanceof Node && row.contains(event.relatedTarget))return;const id=Number(row.dataset.mapAlliance);if(Number.isSafeInteger(id) && id>0)this.renderer?.focusAlliance(id)}
 private clearRow(event:FocusEvent|PointerEvent){if(!this.alive)return;const row=eventElement(event.target)?.closest<HTMLElement>('[data-map-alliance]');if(row && !(event.relatedTarget instanceof Node && row.contains(event.relatedTarget)))this.renderer?.focusAlliance(null)}
 hover(event:PointerEvent){this.focusRow(event)}
 leave(event:PointerEvent){this.clearRow(event)}
 imageError(event:Event){const image=eventElement(event.target);if(image instanceof HTMLImageElement && image.matches('.map-sovereignty-logo img,[data-map-hover-logo]'))image.style.visibility='hidden'}
 disconnect(){this.hide();super.disconnect()}
}
registerController('map-page',MapPageController)
