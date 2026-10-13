import {FragmentSections} from './controllers/fragment-sections';
import {registerController} from './controllers/application';
import {requiredElement,eventElement} from './shared/dom';
import {escapeHTML} from './shared/html';
import {isAbortError,errorMessage} from './shared/json';
import {formatIsk,formatNumber} from '../utils/formatters';
import {isWarsTab,isWarsSort,isWarFilter,isEligibleSort,warFilterNames,readWars,readEligible,sortEligible,type WarsTab,type WarsSort,type EligibleEntry,type EligibleSort,type WarSide,type WarRow,type WarFilter} from './shared/war-model';
type WarsState={tab:WarsTab;sort:WarsSort;page:number;eligible:Map<'corporations'|'alliances',EligibleEntry[]>;eligibleSort:EligibleSort;eligibleDirection:'asc'|'desc';generation:number;controller:AbortController|null}&Record<WarFilter,boolean>;
function createWars(root:HTMLElement,io:WarsController) {
  const q=<T extends Element=HTMLElement>(selector:string)=>requiredElement<T>(root,selector);
  let interrupted=false;
  const results = q<HTMLElement>('[data-wars-results]');
  const filters = q<HTMLElement>('[data-wars-filters]');
  const filterNames=warFilterNames;
  // Load the expensive global aggregates independently of war navigation.
  const faction = root.querySelector<HTMLElement>('[data-wars-faction-content]');
  function loadFaction(){if(faction)void io.faction(faction)}
  const state:WarsState = {
    tab: isWarsTab(root.dataset.tab)?root.dataset.tab:'wars', sort: isWarsSort(root.dataset.sort)?root.dataset.sort:'recent', page: Number(root.dataset.page) || 1,
    mutual: root.dataset.mutual === 'true', hasActivity: root.dataset.hasActivity === 'true',
    hasKills: root.dataset.hasKills === 'true', hasAllies: root.dataset.hasAllies === 'true',
    eligible: new Map(), eligibleSort: 'members', eligibleDirection: 'desc',
    generation: 0, controller: null,
  };
  const escape=escapeHTML,number=formatNumber,isk=formatIsk;
  const timeAgo = (raw:string) => {
    if (!raw) return '';
    const diff = Date.now() - Date.parse(raw);
    if (!Number.isFinite(diff)) return '';
    const future = diff < 0, minutes = Math.floor(Math.abs(diff) / 60000);
    const value = minutes < 60 ? minutes : minutes < 1440 ? Math.floor(minutes / 60) : minutes < 43200 ? Math.floor(minutes / 1440) : minutes < 525600 ? Math.floor(minutes / 43200) : Math.floor(minutes / 525600);
    const unit = minutes < 60 ? 'm' : minutes < 1440 ? 'h' : minutes < 43200 ? 'd' : minutes < 525600 ? 'mo' : 'y';
    return future ? `in ${value}${unit}` : `${value}${unit} ago`;
  };
  const entityImage = (side:WarSide) => side?.id > 0 && ['alliance', 'corporation'].includes(side.type) ? `https://images.evetech.net/${side.type}s/${side.id}/logo?size=64` : '';
  const sideHTML = (side:WarSide) => `<span class="wars-row-side"><img src="${entityImage(side)}" alt="" width="32" height="32" loading="lazy"><span><strong>${escape(side?.name || 'Unknown')}</strong><small><em>${isk(side?.isk_destroyed)}</em><span> · </span>${number(side?.ships_killed)} ships</small></span></span>`;
  const statusHTML = (war:WarRow) => {
    const upcoming = war.started && Date.parse(war.started) > Date.now();
    return `<span class="wars-row-status"><span class="${upcoming ? 'upcoming' : war.finished ? '' : 'active'}">${upcoming ? 'Upcoming' : war.finished ? 'Finished' : 'Active'}</span>${war.mutual ? '<span class="mutual">Mutual</span>' : ''}${war.open_for_allies ? '<span class="allies">Allies</span>' : ''}</span>`;
  };
  const listHTML = (wars:WarRow[]) => {
    const rows = wars.map((war) => `<a class="wars-row" href="/war/${Number(war.war_id)}">${sideHTML(war.aggressor)}<span class="wars-versus">VS</span>${sideHTML(war.defender)}<time>${timeAgo(war.started)}</time>${statusHTML(war)}</a>`).join('');
    const empty = state.tab === 'upcoming' ? 'No upcoming wars' : state.tab === 'finished' ? 'No finished wars found' : 'No active wars found';
    return `<div class="glass-panel wars-list"><div class="wars-list-header"><span>Aggressor</span><span></span><span>Defender</span><span>${state.tab === 'upcoming' ? 'Starts' : 'Started'}</span><span>Status</span></div>${rows || `<div class="wars-empty">${empty}</div>`}</div>${(wars.length >= 50 || state.page > 1) ? `<div class="wars-pagination"><button type="button" data-wars-page="previous" ${state.page <= 1 ? 'disabled' : ''}>Previous</button><span>Page ${state.page}</span><button type="button" data-wars-page="next" ${wars.length<50?'disabled':''}>Next</button></div>` : ''}`;
  };
  const eligibleType = ():'alliances'|'corporations' => state.tab === 'eligible-alliances' ? 'alliances' : 'corporations';
  const eligibleHTML = (entries:EligibleEntry[]) => {
    const sorted=sortEligible(entries,state.eligibleSort,state.eligibleDirection);
    const alliances = eligibleType() === 'alliances';
    const heading = (key:EligibleSort, label:string) => `<button type="button" data-wars-eligible-sort="${key}">${label}<span aria-hidden="true">${state.eligibleSort === key ? state.eligibleDirection === 'desc' ? '⌄' : '⌃' : '⌃⌄'}</span></button>`;
    if (!sorted.length) return `<div class="glass-panel wars-empty">No war-eligible ${alliances ? 'alliances' : 'corporations'} found</div>`;
    return `<div class="glass-panel wars-eligible"><table><thead><tr><th>${heading('name', alliances ? 'Alliance' : 'Corporation')}</th>${alliances ? '<th class="desktop-only">Corps</th>' : ''}<th>${heading('members', 'Members')}</th><th>${heading('kills', 'Kills')}</th><th>${heading('losses', 'Losses')}</th><th class="desktop-only">${heading('isk_destroyed', 'ISK Destroyed')}</th><th class="desktop-only">${heading('isk_lost', 'ISK Lost')}</th></tr></thead><tbody>${sorted.map((entry) => `<tr><td><a href="/${alliances ? 'alliance' : 'corporation'}/${Number(entry.id)}"><img src="https://images.evetech.net/${alliances ? 'alliances' : 'corporations'}/${Number(entry.id)}/logo?size=64" width="24" height="24" alt="" loading="lazy"><span><strong>${escape(entry.name)}${entry.ticker ? ` <span class="wars-ticker">[${escape(entry.ticker)}]</span>` : ''}</strong>${entry.alliance_name ? `<small>${escape(entry.alliance_name)}</small>` : ''}</span></a></td>${alliances ? `<td class="desktop-only">${number(entry.corporation_count)}</td>` : ''}<td>${number(entry.member_count)}</td><td class="${entry.kills ? 'positive' : 'muted'}">${entry.kills ? number(entry.kills) : '—'}</td><td class="${entry.losses ? 'negative' : 'muted'}">${entry.losses ? number(entry.losses) : '—'}</td><td class="desktop-only ${entry.isk_destroyed ? 'isk' : 'muted'}">${entry.isk_destroyed ? isk(entry.isk_destroyed) : '—'}</td><td class="desktop-only ${entry.isk_lost ? 'negative' : 'muted'}">${entry.isk_lost ? isk(entry.isk_lost) : '—'}</td></tr>`).join('')}</tbody></table></div>`;
  };
  function syncURL(replace = false) {
    const url = new URL(location.href);
    for (const name of ['tab', 'sort', 'page', ...filterNames]) url.searchParams.delete(name);
    if (state.tab !== 'wars') url.searchParams.set('tab', state.tab);
    if (state.sort !== 'recent') url.searchParams.set('sort', state.sort);
    if (state.page !== 1) url.searchParams.set('page', String(state.page));
    for (const name of filterNames) if (state[name]) url.searchParams.set(name, 'true');
    history[replace ? 'replaceState' : 'pushState'](history.state, '', url);
  }
  function syncUI() {
    root.dataset.tab = state.tab;
    root.querySelectorAll<HTMLElement>('[data-wars-tab]').forEach((button) => {
      const active = button.dataset.warsTab === state.tab;
      button.classList.toggle('active', active);
      button.setAttribute('aria-pressed', String(active));
    });
    filters.hidden = !['wars', 'finished'].includes(state.tab);
    root.querySelectorAll<HTMLElement>('[data-wars-filter]').forEach((button) => {
      const active = isWarFilter(button.dataset.warsFilter)&&state[button.dataset.warsFilter];
      button.classList.toggle('active', active);
      button.setAttribute('aria-pressed', String(active));
    });
    root.querySelectorAll<HTMLElement>('[data-wars-sort]').forEach((button) => button.classList.toggle('active', button.dataset.warsSort === state.sort));
    q<HTMLElement>('[data-wars-clear]').hidden = !filterNames.some((name) => state[name]) && state.sort === 'recent';
  }
  async function load() {
    state.controller?.abort();
    const controller = new AbortController(), generation = ++state.generation;
    state.controller = controller;
    results.querySelector('[data-wars-request-error]')?.remove();results.setAttribute('aria-busy','true');
    try {
      if (state.tab.startsWith('eligible-')) {
        const type = eligibleType();
        let entries = state.eligible.get(type);
        if (!entries) {
          entries=readEligible(await io.json(`/api/wars/eligible?type=${type}`,controller.signal));
          if(state.controller!==controller)return;
          state.eligible.set(type, entries);
        }
        if (generation === state.generation) results.innerHTML = eligibleHTML(entries);
        return;
      }
      const query = new URLSearchParams({ page: String(state.page), limit: '50' });
      if (state.tab === 'upcoming') query.set('upcoming', 'true');
      else {
        query.set(state.tab === 'finished' ? 'finished' : 'ongoing', 'true');
        query.set('sort', state.sort);
        for (const name of filterNames) if (state[name]) query.set(name, 'true');
      }
      const wars=readWars(await io.json(`/api/conflicts/wars?${query}`,controller.signal));
      if (generation === state.generation) results.innerHTML = listHTML(wars);
    } catch (error) {
      if (!isAbortError(error) && generation === state.generation){
        results.querySelector('.wars-loading')?.remove();const box=document.createElement('div'),retry=document.createElement('button');box.className='glass-panel wars-empty';box.dataset.warsRequestError='';box.setAttribute('role','alert');box.textContent=errorMessage(error,'Could not load wars')+' ';retry.type='button';retry.dataset.warsRetry='';retry.textContent='Retry';box.append(retry);results.append(box);
      }
    } finally {if(state.controller===controller){state.controller=null;results.removeAttribute('aria-busy')}}
  }
  function change() { syncUI(); syncURL(); load(); }
  function click(event:MouseEvent){const target=eventElement(event.target);if(!target)return;const button=target.closest<HTMLButtonElement>('button');if(button?.disabled)return;
    const tab = target.closest<HTMLElement>('[data-wars-tab]');
    if (tab && isWarsTab(tab.dataset.warsTab)) { if (state.tab !== tab.dataset.warsTab) { state.tab = tab.dataset.warsTab; state.page = 1; change(); } return; }
    const filter = target.closest<HTMLElement>('[data-wars-filter]');
    if (filter && isWarFilter(filter.dataset.warsFilter)) { state[filter.dataset.warsFilter] = !state[filter.dataset.warsFilter]; state.page = 1; change(); return; }
    const sort = target.closest<HTMLElement>('[data-wars-sort]');
    if (sort && isWarsSort(sort.dataset.warsSort)) { state.sort = sort.dataset.warsSort; state.page = 1; change(); return; }
    if (target.closest<HTMLElement>('[data-wars-clear]')) { for (const name of filterNames) state[name] = false; state.sort = 'recent'; state.page = 1; change(); return; }
    const page = target.closest<HTMLElement>('[data-wars-page]');
    if (page?.dataset.warsPage === 'previous' && state.page > 1) { state.page--; change(); return; }
    if (page?.dataset.warsPage === 'next' && state.page < 10000) { state.page++; change(); return; }
    const eligibleSort = target.closest<HTMLElement>('[data-wars-eligible-sort]');
    if (eligibleSort) { const key = eligibleSort.dataset.warsEligibleSort;if(!isEligibleSort(key))return; if (state.eligibleSort === key) state.eligibleDirection = state.eligibleDirection === 'desc' ? 'asc' : 'desc'; else { state.eligibleSort = key; state.eligibleDirection = 'desc'; } results.innerHTML = eligibleHTML(state.eligible.get(eligibleType()) || []); return; }
    if (target.closest<HTMLElement>('[data-wars-retry]'))void load();
    if (target.closest<HTMLElement>('[data-wars-faction-retry]'))loadFaction();
  }
  function popstate(){
    const query = new URLSearchParams(location.search);
    const tab=query.get('tab'),sort=query.get('sort');state.tab=isWarsTab(tab)?tab:'wars';
    state.sort = isWarsSort(sort)?sort:'recent';
    state.page = Math.min(10000, Math.max(1, Math.floor(Number(query.get('page'))) || 1));
    for (const name of filterNames) state[name] = query.get(name) === 'true';
    syncUI(); load();
  }
  syncUI();
  loadFaction();
  if (state.tab.startsWith('eligible-'))void load();
  function hide(){interrupted||=Boolean(state.controller);state.controller?.abort();state.controller=null;++state.generation;results.removeAttribute('aria-busy')}
  function resume(){syncUI();if(faction&&!faction.querySelector('[data-wars-faction-loaded]'))loadFaction();if(interrupted){interrupted=false;void load()}}
  return {click,popstate,hide,resume};
}
export class WarsController extends FragmentSections<'faction'>{
 protected retryAttribute='data-wars-faction-retry';
 private ui:ReturnType<typeof createWars>|undefined;
 connect(){if(!this.ui)this.ui=createWars(this.element,this);else this.ui.resume()}
 json(path:string,signal:AbortSignal){return this.request<unknown>(path,{signal},'Could not load wars')}
 faction(container:HTMLElement){if(this.sectionPending('faction'))return;return this.section('faction',{container,path:'/_fragments/wars/faction',selector:'[data-wars-faction-loaded]',message:'Unable to load faction warfare.',errorClass:'glass-panel wars-faction-error',loading:'Loading faction warfare…'})}
 click(event:MouseEvent){this.ui?.click(event)}
 popstate(){this.ui?.popstate()}
 hide(){this.ui?.hide();super.hide()}
 show(event:PageTransitionEvent){if(event.persisted)this.ui?.resume()}
}
registerController('wars',WarsController);

