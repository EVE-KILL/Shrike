import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import './controllers/graph-search'
import {eventElement,requiredElement} from './shared/dom'
import {isRecord,isAbortError} from './shared/json'
import {graphMessage,renderGraphData} from './graph-render'
import {iconHref} from './icon-hrefs'
import {isGraphMode,isGraphEntityType,isGraphSearchKey,graphID,readGraphData,type GraphMode,type GraphEntityType,type GraphSearchKey,type GraphResult} from './shared/graph-model'

type GraphPanel = {panel:HTMLElement;mode:GraphMode;type:GraphEntityType;ids:Partial<Record<GraphSearchKey,number>>;read?:AbortController;cache:Map<string,GraphResult>;dirty:boolean;displayedKey?:string}
export class GraphController extends RequestComponent {
  private alive = false
  private panels = new Map<GraphMode,GraphPanel>()
  private active:GraphMode = 'path_finder'
  connect() {
    this.alive = true
    if(!this.panels.size) for(const panel of this.all<HTMLElement>('[data-graph-panel]')) {
      const mode = panel.dataset.graphPanel;if(!isGraphMode(mode)) throw new Error('Invalid graph panel')
      panel.dataset.graphEntityType = 'alliance';this.panels.set(mode,{panel,mode,type:'alliance',ids:{},cache:new Map(),dirty:false})
    }
    this.select(this.element.dataset.tab,false)
  }
  private result(state:GraphPanel) {return requiredElement<HTMLElement>(state.panel,'[data-graph-result]')}
  private query(state:GraphPanel):URLSearchParams|undefined {
    const q = new URLSearchParams({mode:state.mode})
    if(state.mode === 'path_finder') {if(!state.ids.from || !state.ids.to) return;q.set('fromId',String(state.ids.from));q.set('toId',String(state.ids.to))}
    else if(state.mode === 'rivalries') {q.set('entityType',state.type);q.set('limit','50')}
    else if(state.mode !== 'coalitions' && state.mode !== 'hot_zones') {if(!state.ids.entity) return;q.set('entityId',String(state.ids.entity));if(state.mode !== 'census' && state.mode !== 'migration') q.set('entityType',state.type)}
    return q
  }
  private signalSearches(state:GraphPanel,reset = false) {for(const search of state.panel.querySelectorAll('[data-graph-search]')) search.dispatchEvent(new Event(reset ? 'graph-search-reset' : 'graph-search-hide'))}
  private cancel(state:GraphPanel) {state.read?.abort();state.read = undefined;this.result(state).removeAttribute('aria-busy');this.result(state).querySelector('[data-graph-loading]')?.remove()}
  private clear(state:GraphPanel) {this.cancel(state);state.dirty = false;state.displayedKey = undefined;this.result(state).innerHTML = graphMessage(state.panel.dataset.empty)}
  private errorPanel(state:GraphPanel) {this.result(state).querySelector('[data-graph-error]')?.remove();const error = document.createElement('div');error.className = 'glass-panel p-8 text-center text-sm text-gray-400';error.dataset.graphError = '';error.setAttribute('role','alert');error.innerHTML = 'Unable to load graph intelligence. <button type="button" data-graph-retry class="ml-2 text-blue-400 hover:underline">Retry</button>';this.result(state).append(error)}
  private async load(state:GraphPanel,force = false) {
    if(!this.alive || state.mode !== this.active) return
    this.cancel(state);const query = this.query(state);if(!query) {this.clear(state);return}
    const key = query.toString(),cached = state.cache.get(key),container = this.result(state)
    if(!force && cached) {if(state.displayedKey !== key) {container.innerHTML = renderGraphData(cached,state.type);state.displayedKey = key}state.dirty = false;return}
    const request = new AbortController();state.read = request;state.dirty = true;container.setAttribute('aria-busy','true');container.querySelector('[data-graph-error]')?.remove()
    if(state.displayedKey === key) {const pending = document.createElement('div');pending.dataset.graphLoading = '';pending.setAttribute('role','status');pending.innerHTML = graphMessage(state.panel.dataset.loading,false);container.append(pending)}
    else {state.displayedKey = undefined;container.innerHTML = graphMessage(state.panel.dataset.loading,false)}
    try {
      const raw = await this.request<unknown>('/api/graph?'+key,{signal:request.signal},'Graph unavailable')
      if(!this.alive || state.read !== request || state.mode !== this.active) return
      const data = readGraphData(state.mode,raw)
      if(!state.cache.has(key) && state.cache.size >= 8) {const oldest = state.cache.keys().next().value;if(oldest !== undefined) state.cache.delete(oldest)}
      state.cache.set(key,data);container.innerHTML = renderGraphData(data,state.type);state.displayedKey = key;state.dirty = false
    } catch(error) {
      if(this.alive && state.read === request && !isAbortError(error)) {state.dirty = false;if(!state.displayedKey) container.replaceChildren();this.errorPanel(state)}
    } finally {if(state.read === request) {state.read = undefined;container.removeAttribute('aria-busy');container.querySelector('[data-graph-loading]')?.remove()}}
  }
  private select(raw:unknown,updateURL = true) {
    const mode = isGraphMode(raw) && this.panels.has(raw) ? raw : 'path_finder',state = this.panels.get(mode);if(!state) return
    this.active = mode;this.element.dataset.tab = mode
    const buttons = [...this.all<HTMLButtonElement>('[data-graph-tab]')]
    for(const button of buttons) {const selected = button.dataset.graphTab === mode;button.setAttribute('aria-selected',String(selected));button.tabIndex = selected ? 0 : -1;button.className = 'flex flex-col px-4 py-2 rounded-md text-sm transition-colors '+(selected ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white hover:bg-white/[0.08]')}
    for(const panel of this.panels.values()) {panel.panel.hidden = panel.mode !== mode;if(panel.mode !== mode) {this.cancel(panel);this.signalSearches(panel)}}
    const nav = this.q<HTMLElement>('[role="tablist"]'),chosen = buttons.find(button=>button.dataset.graphTab === mode)
    if(chosen) {const bounds = nav.getBoundingClientRect(),box = chosen.getBoundingClientRect();if(box.left < bounds.left) nav.scrollLeft += box.left-bounds.left;else if(box.right > bounds.right) nav.scrollLeft += box.right-bounds.right}
    if(updateURL) {const url = new URL(location.href);url.searchParams.set('tab',mode);history.replaceState(history.state,'',url)}
    const query = this.query(state)
    if(query && !state.read && (state.dirty || !state.cache.has(query.toString()))) void this.load(state,state.dirty)
  }
  history() {this.select(new URL(location.href).searchParams.get('tab'),false)}
  private searchEvent(event:CustomEvent<unknown>) {
    const detail = event.detail,panel = eventElement(event.target)?.closest<HTMLElement>('[data-graph-panel]')
    if(!isRecord(detail) || !isGraphMode(detail.mode) || !isGraphSearchKey(detail.key) || panel?.dataset.graphPanel !== detail.mode || detail.mode !== this.active) return
    if(detail.mode === 'path_finder' ? detail.key !== 'from' && detail.key !== 'to' : detail.key !== 'entity') return
    const state = this.panels.get(detail.mode);return state ? {state,key:detail.key,id:graphID(detail.id)} : undefined
  }
  searchChange(event:CustomEvent<unknown>) {const change = this.searchEvent(event);if(change) {delete change.state.ids[change.key];this.clear(change.state)}}
  pick(event:CustomEvent<unknown>) {const change = this.searchEvent(event);if(change?.id) {change.state.ids[change.key] = change.id;void this.load(change.state,true)}}
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!target || !this.alive) return
    const tab = target.closest<HTMLElement>('[data-graph-tab]');if(tab) {this.select(tab.dataset.graphTab);return}
    const panel = target.closest<HTMLElement>('[data-graph-panel]'),mode = panel?.dataset.graphPanel;if(!isGraphMode(mode) || mode !== this.active) return
    const state = this.panels.get(mode);if(!state) return
    const type = target.closest<HTMLButtonElement>('[data-graph-type]'),next = type?.dataset.graphType
    if(type && isGraphEntityType(next)) {
      if(mode === 'rivalries' ? next === 'corporation' : next === 'character' || !['entity_intel','hunting_grounds','spy_check'].includes(mode)) return
      state.type = next;state.panel.dataset.graphEntityType = next;state.ids = {};this.clear(state);this.signalSearches(state,true)
      const rivalry = mode === 'rivalries',color = rivalry ? 'red' : 'purple'
      for(const button of state.panel.querySelectorAll<HTMLButtonElement>('[data-graph-type]')) {const selected = button === type;button.setAttribute('aria-pressed',String(selected));button.className = 'px-3 py-1.5 rounded-md text-sm transition-colors '+(selected ? `bg-${color}-500/20 text-${color}-400 border border-${color}-500/30` : 'bg-white/[0.04] text-gray-400 border border-white/[0.06] hover:text-white')}
      for(const input of state.panel.querySelectorAll<HTMLInputElement>('[data-graph-input]')) {input.placeholder = mode === 'entity_intel' ? `Search for ${next === 'alliance' ? 'an alliance' : 'a corporation'}...` : `Search ${next}...`;input.setAttribute('aria-label',input.placeholder)}
      if(rivalry) void this.load(state);return
    }
    if(target.closest('[data-graph-retry]')) {if(!state.read) void this.load(state,true);return}
    const coalition = target.closest<HTMLButtonElement>('[data-graph-coalition]')
    if(coalition) {const members = coalition.nextElementSibling;if(members instanceof HTMLElement) {const expand = members.hidden;members.hidden = !expand;coalition.setAttribute('aria-expanded',String(expand));coalition.querySelector('use')?.setAttribute('href',iconHref(expand ? 'chevron-down' : 'chevron-right'))}}
  }
  key(event:KeyboardEvent) {
    if(!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return
    const button = eventElement(event.target)?.closest<HTMLButtonElement>('[data-graph-tab]'),buttons = [...this.all<HTMLButtonElement>('[data-graph-tab]')],index = button ? buttons.indexOf(button) : -1
    if(index < 0) return
    event.preventDefault();const next = event.key === 'Home' ? 0 : event.key === 'End' ? buttons.length-1 : (index+(event.key === 'ArrowRight' ? 1 : -1)+buttons.length)%buttons.length
    this.select(buttons[next].dataset.graphTab);buttons[next].focus()
  }
  hide() {this.alive = false;for(const state of this.panels.values()) {this.cancel(state);this.signalSearches(state)}super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.history()}}
}
registerController('graph',GraphController)
