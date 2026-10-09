import {RequestComponent} from './controllers/request-component';
import {registerController} from './controllers/application';
import {requiredElement,eventElement} from './shared/dom';
import {errorMessage,isAbortError} from './shared/json';
import type {SearchOutput} from '../../shared/api/types.gen';
import type {GoMapController,MapScopeData} from '../utils/map/goFrontendCanvas';
import type {BattleMapSystem} from '../utils/map/battles';
import {isMapScope as isScope,type MapScope as Scope} from './shared/map-scope';
import {loadMapCanvas} from './shared/map-module';

function createBattles(root:HTMLElement,io:BattlesController) {
 const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(root,selector),get=<T extends Element=HTMLElement>(parent:ParentNode,selector:string)=>requiredElement<T>(parent,selector);
 let interrupted=false,blurTimer:ReturnType<typeof setTimeout>|undefined,updateRegion:(()=>void)|undefined;
  const search = q<HTMLInputElement>('[data-battles-search]');
  const searchResults = q<HTMLElement>('[data-battles-search-results]');
  const searchLoading = q<HTMLElement>('[data-battles-search-loading]');
  const errorBox = q<HTMLElement>('[data-battles-error]');
  const count = new Intl.NumberFormat('en-US');
  let controller:AbortController|null = null;
  let searchController:AbortController|null = null;
  let searchTimer:ReturnType<typeof setTimeout>|undefined;
  let mapController:AbortController|null = null;
  let mapRenderer:GoMapController|null = null;
  let mapRegion:number|null = null;
  let mapScope:Scope = 'new-eden';
  const geographyCache = new Map<Scope,MapScopeData>();
  const allowedEntities = new Set(['alliance', 'corporation', 'system', 'region']);

  const query = () => new URLSearchParams(location.search);
  const apply = (patch:Record<string,string|number|null|undefined>, resetPage = true) => {
    const params = query();
    for (const [key, value] of Object.entries(patch)) {
      if (value == null || value === '') params.delete(key);
      else params.set(key, String(value));
    }
    if (resetPage) { params.delete('page'); mapRegion = null; }
    history.pushState(null, '', `${location.pathname}${params.size ? '?' + params : ''}`);
    sync();
    load();
  };
  const sync = () => {
    const params = query();
    const hours = params.get('hours') || '';
    const year = params.get('year') || '';
    const minKills = params.get('minKills') || '';
    const minISK = params.get('minIsk') || '';
    const custom = params.get('custom') === '1';
    const view = params.get('view') === 'map' ? 'map' : 'list';
    for (const button of root.querySelectorAll<HTMLElement>('[data-battles-hours]')) {
      const active = button.dataset.battlesHours === hours && (hours !== '' || year === '');
      button.classList.toggle('active', active);
      button.setAttribute('aria-pressed', String(active));
    }
    for (const button of root.querySelectorAll<HTMLElement>('[data-battles-year]')) {
      const active = button.dataset.battlesYear === year;
      button.classList.toggle('active', active);
      button.setAttribute('aria-pressed', String(active));
    }
    q('[data-battles-clear-year]').hidden = !year;
    const customButton = q('[data-battles-custom]');
    customButton.classList.toggle('active', custom);
    customButton.setAttribute('aria-pressed', String(custom));
    for (const button of root.querySelectorAll<HTMLElement>('[data-battles-kills]')) {
      const active = button.dataset.battlesKills === minKills;
      button.classList.toggle('active', active);
      button.setAttribute('aria-pressed', String(active));
    }
    for (const button of root.querySelectorAll<HTMLElement>('[data-battles-isk]')) {
      const active = button.dataset.battlesIsk === minISK;
      button.classList.toggle('active', active);
      button.setAttribute('aria-pressed', String(active));
    }
    for (const button of root.querySelectorAll<HTMLElement>('[data-battles-view]')) {
      const active = button.dataset.battlesView === view;
      button.classList.toggle('active', active);
      button.setAttribute('aria-selected', String(active));
    }
  };
  const load = async () => {
    controller?.abort();
    const request = new AbortController();
    controller = request;
    errorBox.hidden = true;
    root.querySelector('[data-battles-results]')?.classList.add('loading');
    try {
      const text=await io.fragment(`/_fragments/battles?${query()}`,request.signal),doc=new DOMParser().parseFromString(text,'text/html');
      const next = doc.querySelector('[data-battles-results]');
      const chips = doc.querySelector('[data-battles-chips]');
      if (!next || !chips) throw new Error('Battle results are unavailable');
      if (controller !== request) return;
      mapController?.abort();mapRenderer?.destroy();mapRenderer=null;updateRegion=undefined;
      q('[data-battles-chips]').replaceWith(document.importNode(chips, true));
      q('[data-battles-results]').replaceWith(document.importNode(next, true));
      if (query().get('view') === 'map') loadMap();
    } catch (error) {
      if (!isAbortError(error) && controller === request) {
        errorBox.hidden = false;
        if(errorBox.firstChild)errorBox.firstChild.textContent = `${errorMessage(error,'Unable to load battles')}. `;
      }
    } finally {
      if (controller === request) {
        controller = null;
        root.querySelector('[data-battles-results]')?.classList.remove('loading');
      }
    }
  };
  const clear = (what:string|undefined) => {
    switch (what) {
      case 'entity': apply({ eType: null, eId: null }); break;
      case 'year': apply({ year: null }); break;
      case 'kills': apply({ minKills: null }); break;
      case 'isk': apply({ minIsk: null }); break;
      default: apply({ hours: null, year: null, minKills: null, minIsk: null, custom: null, eType: null, eId: null });
    }
  };

  const selectEntity = (type:string|undefined, id:number, fromMap = false) => {
    if (!type||!allowedEntities.has(type) || !Number.isSafeInteger(id) || id <= 0) return;
    io.cancelTimer(searchTimer);searchController?.abort();searchController=null;searchLoading.hidden=true;search.value = '';
    searchResults.hidden = true;
    searchResults.replaceChildren();
    if (fromMap) mapRegion = null;
    apply({ eType: type, eId: id, ...(fromMap ? { view: null } : {}) });
  };
  const selectView = (view:'map'|'list') => {
    if ((query().get('view') === 'map' ? 'map' : 'list') === view) return;
    mapRegion = null; mapScope = 'new-eden';
    apply({view: view === 'map' ? 'map' : null}, false);
  };
  const searchEntities = async (value:string) => {
    searchController?.abort();
    if (value.length < 3) { searchLoading.hidden = true; searchResults.hidden = true; searchResults.replaceChildren(); return; }
    const request = new AbortController();
    searchController = request; searchLoading.hidden = false;
    try {
      const data=await io.search(`/api/search?${new URLSearchParams({q:value,limit:'10'})}`,request.signal);
      if (searchController !== request || search.value.trim() !== value) return;
      searchResults.replaceChildren();
      for (const hit of (data.hits || []).filter(item => allowedEntities.has(item.type))) {
        const id = Number(String(hit.id).split('_').at(-1));
        if (!Number.isSafeInteger(id) || id <= 0) continue;
        const button = document.createElement('button');
        button.type = 'button';
        button.dataset.entityType = hit.type;
        button.dataset.entityId = String(id);
        button.setAttribute('aria-label', `${hit.name}${hit.ticker ? ` [${hit.ticker}]` : ''} ${hit.type[0].toUpperCase() + hit.type.slice(1)}`);
        const image = hit.type === 'alliance' || hit.type === 'corporation' ? `https://images.evetech.net/${hit.type}s/${id}/logo?size=64` : '';
        if (image) { const img = document.createElement('img'); img.src = image; img.alt = ''; img.width = img.height = 28; button.append(img); }
        if (!image) {
          const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
          svg.classList.add('icon', 'search-fallback'); svg.setAttribute('aria-hidden', 'true'); svg.setAttribute('viewBox', '0 0 24 24');
          const use = document.createElementNS('http://www.w3.org/2000/svg', 'use'); use.setAttribute('href', `/assets/icons.svg#${hit.type === 'system' ? 'globe' : 'map'}`); svg.append(use); button.append(svg);
        }
        const label = document.createElement('span');
        label.textContent = hit.name;
        if (hit.ticker) { const ticker = document.createElement('span'); ticker.className = 'search-ticker'; ticker.textContent = ` [${hit.ticker}]`; label.append(ticker); }
        const kind = document.createElement('small'); kind.textContent = hit.type;
        label.append(' ', kind); button.append(label); searchResults.append(button);
      }
      searchResults.hidden = !searchResults.childElementCount;
    } catch (error) {
      if (!isAbortError(error) && searchController === request) searchResults.hidden = true;
    } finally { if (searchController === request) { searchLoading.hidden = true; searchController = null; } }
  };

  const loadMap = async () => {
    const panel = root.querySelector<HTMLElement>('[data-battles-map-panel]');
    if (!panel) return;
    mapController?.abort(); mapRenderer?.destroy(); mapRenderer = null;updateRegion=undefined;
    const request = new AbortController(); mapController = request;
    const current = () => mapController === request && panel.isConnected;
    const status = get<HTMLElement>(panel,'[data-battles-map-status]');
    const surface = get<HTMLElement>(panel,'[data-battles-map-surface]');
    const markersError = get<HTMLElement>(panel,'[data-battles-markers-error]');
    const geographyError = get<HTMLElement>(panel,'[data-battles-geography-error]');
    const empty = get<HTMLElement>(panel,'[data-battles-map-empty]');
    get<HTMLSelectElement>(panel,'[data-battles-map-scope]').value = mapScope;
    status.textContent = 'Loading matching battles…';
    markersError.hidden = geographyError.hidden = empty.hidden = true;
    surface.setAttribute('aria-busy', 'true');
    const loading = document.createElement('div');
    loading.className = 'battles-map-loading'; loading.setAttribute('role', 'status'); loading.textContent = 'Loading map data…';
    surface.replaceChildren(loading);
    try {
      const params = query();
      const apiQuery = new URLSearchParams({ map: 'true' });
      for (const key of ['hours', 'year', 'minKills']) if (params.has(key)) apiQuery.set(key, params.get(key)??'');
      const iskValues:Record<string,number>={ '1b': 1e9, '10b': 1e10, '50b': 5e10, '100b': 1e11, '1t': 1e12 },isk=iskValues[params.get('minIsk')??''];
      if (isk) apiQuery.set('minIsk', String(isk));
      if (params.get('custom') === '1') apiQuery.set('custom', 'true');
      if (params.has('eType') && params.has('eId')) apiQuery.set(`${params.get('eType')}Id`, params.get('eId')??'');
      // Markers can fail or load slowly without delaying usable geography.
      let matches:BattleMapSystem[] = [];
      void io.matches(`/api/conflicts/battles?${apiQuery}`,request.signal)
        .then(data => {
          if (!current()) return;
          matches = data.systems || [];
          const matched = matches.reduce((sum, system) => sum + Number(system.battle_count || 0), 0);
          status.textContent = `${count.format(matched)} matching battles across all space · ${count.format(matches.length)} systems`;
          empty.hidden = matched !== 0;
          mapRenderer?.setBattleSystems(matches);
        }).catch(error => {
          if (!isAbortError(error) && current()) { status.textContent = ''; markersError.hidden = false; }
        });
      const scope = mapScope;
      const cached=geographyCache.get(scope),geography=cached?Promise.resolve(cached):io.geography(`/api/map/scope?${new URLSearchParams({type:scope,hours:'24'})}`,request.signal).then(data=>{if(!Array.isArray(data?.systems))throw new Error('Map data is unavailable');geographyCache.set(scope,data);return data});
      const [module, data] = await Promise.all([loadMapCanvas(), geography]);
      if (!current()) return;
      updateRegion = () => {
        if(!current()||!mapRenderer)return;
        get<HTMLElement>(panel,'[data-battles-map-back]').hidden = !mapRegion;
        get<HTMLElement>(panel,'[data-battles-map-title]').textContent = mapRegion
          ? data.regions?.find(region => Number(region.region_id) === mapRegion)?.name || `Region ${mapRegion}` : '';
        mapRenderer.setRegion(mapRegion);
      };
      mapRenderer = module.createGoMap(surface, {
        battleSystems: matches, showScopeDetails: true, regionId: mapRegion,
        onBattleRegion: id => { if(current()){mapRegion = id; updateRegion?.();} },
        onBattleSystem: system => {if(current())selectEntity('system', Number(system.solar_system_id), true)},
      });
      mapRenderer.setData(data);
      surface.setAttribute('aria-busy', 'false');
      updateRegion();
    } catch (error) {
      if (!isAbortError(error) && current()) {
        surface.replaceChildren(); surface.setAttribute('aria-busy', 'false');
        geographyError.hidden = false;
      }
    }
  };

  function click(event:MouseEvent) {
    const button = eventElement(event.target)?.closest<HTMLButtonElement>('button');
    if (!button || !root.contains(button) || button.disabled) return;
    if (button.matches('[data-battles-hours]')) { apply({ hours: button.dataset.battlesHours || null, year: null }); return; }
    if (button.matches('[data-battles-year]')) { const selected = query().get('year') === button.dataset.battlesYear; apply({ year: selected ? null : button.dataset.battlesYear, hours: null }); return; }
    if (button.matches('[data-battles-clear-year]')) { clear('year'); return; }
    if (button.matches('[data-battles-custom]')) { apply({ custom: query().get('custom') === '1' ? null : '1' }); return; }
    if (button.matches('[data-battles-kills]')) { apply({ minKills: query().get('minKills') === button.dataset.battlesKills ? null : button.dataset.battlesKills }); return; }
    if (button.matches('[data-battles-isk]')) { apply({ minIsk: query().get('minIsk') === button.dataset.battlesIsk ? null : button.dataset.battlesIsk }); return; }
    if (button.matches('[data-battles-view]')) { selectView(button.dataset.battlesView==='map'?'map':'list'); return; }
    if (button.matches('[data-battles-page]')) { apply({ page: Number(button.dataset.battlesPage) > 1 ? button.dataset.battlesPage : null }, false); root.scrollIntoView({ behavior: 'smooth', block: 'start' }); return; }
    if (button.matches('[data-battles-clear]')) { clear(button.dataset.battlesClear); return; }
    if(button.matches('[data-battles-map-back]')){mapRegion=null;updateRegion?.();return;}
    if (button.matches('[data-battles-map-retry]')) { void loadMap(); return; }
    if (button.matches('[data-battles-retry]')) { load(); return; }
    if (button.matches('[data-entity-type]')) { selectEntity(button.dataset.entityType, Number(button.dataset.entityId)); }
  }
  function keyboard(event:KeyboardEvent) {
    const tab = eventElement(event.target)?.closest<HTMLElement>('[data-battles-view]');
    if (!tab || !['ArrowLeft', 'ArrowRight'].includes(event.key)) return;
    event.preventDefault();
    const view = tab.dataset.battlesView === 'map' ? 'list' : 'map';
    selectView(view);
    q(`[data-battles-view="${view}"]`).focus();
  }
  function change(event:Event) {const target=event.target;if(target instanceof HTMLSelectElement&&target.matches('[data-battles-map-scope]')&&isScope(target.value)){mapScope=target.value;mapRegion=null;void loadMap();}}
  function input(event:Event){if(event.target!==search)return;
    io.cancelTimer(searchTimer); searchController?.abort(); searchController = null;
    searchResults.hidden = true; searchResults.replaceChildren(); searchLoading.hidden = true;
    searchTimer = io.schedule(() => {searchTimer=undefined;void searchEntities(search.value.trim())}, 300);
  }
  function blur(event:FocusEvent){if(event.target!==search)return;io.cancelTimer(blurTimer);blurTimer=io.schedule(()=>{searchResults.hidden=true;blurTimer=undefined},200);}
  function focus(event:FocusEvent){if(event.target!==search)return;io.cancelTimer(blurTimer);blurTimer=undefined;if(searchResults.childElementCount)searchResults.hidden=false;else if(search.value.trim().length>=3)void searchEntities(search.value.trim());}
  function popstate(){sync();void load();}
  function hide(){interrupted||=Boolean(controller)||Boolean(mapController);controller?.abort();controller=null;searchController?.abort();searchController=null;mapController?.abort();mapController=null;mapRenderer?.destroy();mapRenderer=null;updateRegion=undefined;io.cancelTimer(searchTimer);io.cancelTimer(blurTimer);searchTimer=blurTimer=undefined;searchResults.hidden=true;searchLoading.hidden=true;root.querySelector('[data-battles-results]')?.classList.remove('loading');}
  function resume(){sync();if(interrupted){interrupted=false;void load();}}
  sync();if(query().get('view')==='map')void loadMap();
  return {click,keyboard,change,input,blur,focus,popstate,hide,resume};
}
export class BattlesController extends RequestComponent {
 private ui:ReturnType<typeof createBattles>|undefined
 connect(){if(!this.ui)this.ui=createBattles(this.element,this);else this.ui.resume()}
 fragment(path:string,signal:AbortSignal){return this.text(path,'Unable to load battles',{signal})}
 search(path:string,signal:AbortSignal){return this.request<SearchOutput>(path,{signal})}
 matches(path:string,signal:AbortSignal){return this.request<{systems:BattleMapSystem[]}>(path,{signal})}
 geography(path:string,signal:AbortSignal){return this.request<MapScopeData>(path,{signal})}
 schedule(run:()=>void,delay:number){return this.after(delay,run)}
 cancelTimer(timer:ReturnType<typeof setTimeout>|undefined){this.stopTimer(timer)}
 click(event:MouseEvent){this.ui?.click(event)}
 keyboard(event:KeyboardEvent){this.ui?.keyboard(event)}
 change(event:Event){this.ui?.change(event)}
 input(event:Event){this.ui?.input(event)}
 blur(event:FocusEvent){this.ui?.blur(event)}
 focus(event:FocusEvent){this.ui?.focus(event)}
 popstate(){this.ui?.popstate()}
 show(event:PageTransitionEvent){if(event.persisted)this.ui?.resume()}
 hide(){this.ui?.hide();super.hide()}
}
registerController('battles',BattlesController)
