import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'
import {isRecord,isAbortError} from './shared/json'
import {quickFilters,quickFilterCount,quickFilterPayload,matchesQuickFilters,pickerKinds,isPickerKind,isSecurityKind,type PickerKind} from './shared/kill-filters'
import {searchID,readSearchResults} from './shared/search-model'
import {replaceFragment} from './fragment'
import {createRelayConnection} from '../utils/relaySocketCore'
import {liveRow,parseLiveKill,friendlyDay,element as node} from './kill-row'

const tabs = {kills:'',top:'Top Lists',valuable:'Valuable'} as const
const validTab = (value:unknown):value is keyof typeof tabs=>typeof value === 'string' && Object.hasOwn(tabs,value)
const validPage = (value:number)=>Number.isSafeInteger(value) && value >= 1
const validLimit = (value:number)=>[10,25,50,100].includes(value)
type ListIntent = {page:number;limit:number;fresh:boolean}

// Persisted state belongs to the page. Its server-rendered widgets can be
// replaced without losing picker drafts, cursor history or live connection.
export class HomeController extends RequestComponent {
  private alive = false
  private filters = quickFilters()
  private filterOpen = false
  private cursors:Array<string|undefined> = []
  private listRead:AbortController|undefined
  private valueRead:AbortController|undefined
  private listIntent:ListIntent|undefined
  private valueIntent:string|undefined
  private searches = new Map<PickerKind,AbortController>()
  private searchTimers = new Map<PickerKind,ReturnType<typeof setTimeout>>()
  private blurTimers = new Map<PickerKind,ReturnType<typeof setTimeout>>()
  private relay:ReturnType<typeof createRelayConnection>|undefined
  private timesTimer:ReturnType<typeof setTimeout>|undefined
  private newCount = 0
  private get list() {return this.element.querySelector<HTMLElement>('#kill-list')}
  private get valuable() {return this.element.querySelector<HTMLElement>('#most-valuable')}
  private get page() {return Number(this.list?.dataset.page ?? 1)}
  private get limit() {return Number(this.list?.dataset.limit ?? 50)}
  connect() {this.alive = true;this.syncTab();this.renderFilters();this.startLive()}
  syncTab() {
    if(!this.element.hasAttribute('data-kills-page')) return
    const selected = location.hash.slice(1),tab = validTab(selected) ? selected : 'kills';this.element.dataset.tab = tab
    for(const button of this.all<HTMLElement>('[data-kills-tab]')) button.setAttribute('aria-selected',String(button.dataset.killsTab === tab))
    document.title = `${this.element.dataset.title}${tabs[tab] ? ` (${tabs[tab]})` : ''} - EVE-KILL.com`
    for(const box of this.all('[data-top-pending]')) box.dispatchEvent(new Event('home:top'))
  }
  imageError(event:Event) {
    const image = eventElement(event.target);if(!(image instanceof HTMLImageElement)) return
    const path = new URL(image.src,location.href).pathname;if(!path.startsWith('/images/')) return
    const fallback = node('span','image-fallback',/^\/images\/(systems|regions|constellations)\//.test(path) ? '◎' : '◇')
    if(image.alt) {fallback.setAttribute('role','img');fallback.setAttribute('aria-label',image.alt)}else fallback.setAttribute('aria-hidden','true')
    image.replaceWith(fallback)
  }
  private notice(root:HTMLElement,message:string,retry:'list'|'valuable') {root.querySelector('.widget-error')?.remove();const notice = node('p','widget-error',message+' '),button = node('button','','Retry');notice.setAttribute('role','status');button.type = 'button';button.dataset.homeRetry = retry;notice.append(button);root.prepend(notice)}
  private async loadValuable(kind:string) {
    const root = this.valuable;if(!this.alive || !root || !['kills','ships','structures'].includes(kind)) return
    this.valueRead?.abort();this.valueRead = undefined;root.removeAttribute('aria-busy');this.valueIntent = undefined
    if(root.dataset.kind === kind) {root.querySelector('.widget-error')?.remove();return}
    const request = new AbortController();this.valueRead = request;this.valueIntent = kind;root.setAttribute('aria-busy','true')
    try {
      const html = await this.text('/_fragments/most-valuable?kind='+encodeURIComponent(kind),'Could not load this list.',{signal:request.signal,headers:{Accept:'text/html'}})
      if(!this.alive || this.valueRead !== request || !root.isConnected) return
      const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('#most-valuable');if(!next || next.dataset.kind !== kind) throw new Error('Invalid valuable kills fragment')
      replaceFragment(root,next);this.valueIntent = undefined
    }catch(error) {if(this.alive && this.valueRead === request && !isAbortError(error) && root.isConnected) this.notice(root,'Could not load this list. Try again.','valuable')}
    finally {if(this.valueRead === request) {this.valueRead = undefined;root.removeAttribute('aria-busy')}}
  }
  private renderFilters(root = this.list) {
    const panel = root?.querySelector<HTMLElement>('.kill-filter-panel');if(!panel) return
    panel.hidden = !this.filterOpen;const trigger = root?.querySelector<HTMLElement>('.kill-filter-trigger');trigger?.setAttribute('aria-expanded',String(this.filterOpen));trigger?.setAttribute('aria-label',this.filterOpen ? 'Hide kill list filters' : 'Show kill list filters')
    const count = quickFilterCount(this.filters);let badge = trigger?.querySelector<HTMLElement>('.kill-filter-count')
    if(count && !badge && trigger) {badge = node('span','kill-filter-count');trigger.append(badge)}if(badge) {badge.textContent = String(count);badge.hidden = !count}
    for(const kind of pickerKinds) {const input = panel.querySelector<HTMLInputElement>(`[data-filter-search="${kind}"]`),clear = panel.querySelector<HTMLElement>(`[data-filter-clear="${kind}"]`),item = this.filters[kind];if(!input || !clear) continue;if(item) input.value = item.name;else if(input.readOnly) input.value = '';input.readOnly = Boolean(item);clear.hidden = !item}
    for(const button of panel.querySelectorAll<HTMLElement>('[data-filter-option]')) {const value = button.dataset.filterOption,active = isSecurityKind(value) ? this.filters.security.has(value) : value === 'solo' || value === 'npc' ? this.filters[value] : value === this.filters.isk || value === this.filters.tech;button.setAttribute('aria-pressed',String(active))}
    requiredElement<HTMLElement>(panel,'[data-filter-clear-all]').hidden = !count
  }
  private connected(connected:boolean) {const dot = this.list?.querySelector<HTMLElement>('.kill-live-dot');if(dot) {dot.classList.toggle('connected',connected);dot.title = connected ? 'Live — connected' : 'Disconnected';dot.setAttribute('aria-label',dot.title)}}
  private renderNewCount() {const button = this.list?.querySelector<HTMLElement>('.new-kills');if(button) {button.hidden = !this.newCount;requiredElement(button,'span').textContent = String(this.newCount)}}
  private updateTimes() {for(const time of this.all<HTMLTimeElement>('#kill-list time[datetime]')) {const minutes = Math.floor((Date.now()-Date.parse(time.dateTime))/60000);time.textContent = minutes < 1 ? 'just now' : minutes < 30 ? `${minutes}m ago` : time.dateTime.slice(11,16)}}
  private async loadKills(page:number,limit = this.listIntent?.limit ?? this.limit,fresh = false) {
    const root = this.list;if(!this.alive || !root || !validPage(page) || !validLimit(limit)) return
    this.listRead?.abort();const request = new AbortController();this.listRead = request;this.listIntent = {page,limit,fresh};const countAtStart = this.newCount
    const params = new URLSearchParams({type:root.dataset.kind ?? 'latest',page:String(page),limit:String(limit)});if(root.dataset.quickFilter === 'true') params.set('quick','1')
    const payload = quickFilterPayload(this.filters);if(payload) {params.set('filters',payload);const cursor = this.cursors[page-1];if(page > 1 && cursor) params.set('after',cursor)}else if(fresh || page === 1 && this.page !== 1) params.set('fresh','1')
    root.setAttribute('aria-busy','true')
    try {
      const html = await this.text('/_fragments/killlist?'+params,'Could not load kills.',{signal:request.signal,headers:{Accept:'text/html'}})
      if(!this.alive || this.listRead !== request || !root.isConnected) return
      const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('#kill-list');if(!next || next.dataset.kind !== root.dataset.kind || Number(next.dataset.page) !== page || Number(next.dataset.limit) !== limit) throw new Error('Invalid kill list fragment')
      replaceFragment(root,next,()=>{const panel = root.querySelector('.kill-filter-panel'),placeholder = next.querySelector('.kill-filter-panel');if(panel && placeholder) placeholder.replaceWith(panel);this.renderFilters(next)})
      this.listIntent = undefined;if(page === 1) this.newCount = Math.max(0,this.newCount-countAtStart);this.renderNewCount();this.updateTimes();this.connected(this.relay?.connected ?? false)
    }catch(error) {if(this.alive && this.listRead === request && !isAbortError(error) && root.isConnected) this.notice(root,'Could not load kills. Try again.','list')}
    finally {if(this.listRead === request) {this.listRead = undefined;root.removeAttribute('aria-busy')}}
  }
  private clearDrafts(clear = true) {
    for(const kind of pickerKinds) {this.stopTimer(this.searchTimers.get(kind));this.stopTimer(this.blurTimers.get(kind));this.searches.get(kind)?.abort();const input = this.list?.querySelector<HTMLInputElement>(`[data-filter-search="${kind}"]`);if(clear && input && !this.filters[kind]) input.value = '';const suggestions = this.list?.querySelector<HTMLElement>(`[data-filter-suggestions="${kind}"]`);if(suggestions) {suggestions.hidden = true;if(clear) suggestions.replaceChildren()}}
    this.searchTimers.clear();this.blurTimers.clear();this.searches.clear()
  }
  private changed() {this.cursors = [];this.renderFilters();void this.loadKills(1)}
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    const mobile = target.closest<HTMLElement>('[data-kills-tab]');if(mobile && validTab(mobile.dataset.killsTab)) {history.pushState(history.state,'',location.pathname+location.search+(mobile.dataset.killsTab === 'kills' ? '' : '#'+mobile.dataset.killsTab));this.syncTab();return}
    const valuable = target.closest<HTMLElement>('[data-valuable-tab]');if(valuable) {void this.loadValuable(valuable.dataset.valuableTab ?? '');return}
    const retry = target.closest<HTMLElement>('[data-home-retry]');if(retry) {if(retry.dataset.homeRetry === 'list' && this.listIntent) void this.loadKills(this.listIntent.page,this.listIntent.limit,this.listIntent.fresh);else if(this.valueIntent) void this.loadValuable(this.valueIntent);return}
    const list = target.closest<HTMLElement>('#kill-list');if(!list) return
    const limit = target.closest<HTMLElement>('[data-kill-limit]');if(limit) {this.cursors = [];void this.loadKills(1,Number(limit.dataset.killLimit));return}
    const page = target.closest<HTMLButtonElement>('[data-kill-page]');if(page && !page.disabled) {const next = Number(page.dataset.killPage);if(quickFilterCount(this.filters) && next === this.page+1) this.cursors[next-1] = list.dataset.nextCursor;void this.loadKills(next);return}
    if(target.closest('.new-kills')) {this.cursors = [];void this.loadKills(1,this.limit,true);return}
    if(target.closest('.kill-filter-trigger')) {this.filterOpen = !this.filterOpen;if(!this.filterOpen) this.clearDrafts();this.renderFilters();return}
    const clear = target.closest<HTMLElement>('[data-filter-clear]');if(clear && isPickerKind(clear.dataset.filterClear)) {this.filters[clear.dataset.filterClear] = null;this.changed();return}
    if(target.closest('[data-filter-clear-all]')) {this.filters = quickFilters();this.clearDrafts();this.cursors = [];this.renderFilters();void this.loadKills(1,this.limit,true);return}
    const hit = target.closest<HTMLElement>('[data-filter-hit]');if(hit && isPickerKind(hit.dataset.filterHit)) {const kind = hit.dataset.filterHit,id = searchID(hit.dataset.id);if(!id) return;this.filters[kind] = {id,name:hit.textContent?.trim() ?? ''};const suggestions = hit.closest<HTMLElement>('[data-filter-suggestions]');suggestions?.replaceChildren();if(suggestions) suggestions.hidden = true;if(kind === 'region') this.filters.system = null;if(kind === 'system') this.filters.region = null;this.changed();return}
    const option = target.closest<HTMLElement>('[data-filter-option]');if(!option) return
    const value = option.dataset.filterOption;if(isSecurityKind(value)) {if(this.filters.security.has(value)) this.filters.security.delete(value);else this.filters.security.add(value)}else if(value === 'solo' || value === 'npc') this.filters[value] = !this.filters[value];else if(value === '5b' || value === '10b') this.filters.isk = this.filters.isk === value ? null : value;else if(value === 't1' || value === 't2' || value === 't3') this.filters.tech = this.filters.tech === value ? null : value;else return
    this.changed()
  }
  input(event:Event) {
    const input = eventElement(event.target);if(!this.alive || !(input instanceof HTMLInputElement) || input.readOnly || !isPickerKind(input.dataset.filterSearch)) return
    const kind = input.dataset.filterSearch;this.stopTimer(this.searchTimers.get(kind));this.stopTimer(this.blurTimers.get(kind));this.searches.get(kind)?.abort();this.searches.delete(kind)
    const suggestions = this.list?.querySelector<HTMLElement>(`[data-filter-suggestions="${kind}"]`);if(!suggestions) return;suggestions.hidden = true;suggestions.replaceChildren();const query = input.value.trim();if(query.length < 2) return
    this.searchTimers.set(kind,this.after(250,()=>{this.searchTimers.delete(kind);void this.search(input,kind,query,suggestions)}))
  }
  private async search(input:HTMLInputElement,kind:PickerKind,query:string,suggestions:HTMLElement) {
    const request = new AbortController();this.searches.set(kind,request)
    try {
      const data = readSearchResults(await this.request<unknown>('/api/search?'+new URLSearchParams({q:query,type:kind,limit:'8'}),{signal:request.signal},'Search failed'))
      if(!this.alive || this.searches.get(kind) !== request || !input.isConnected || input.value.trim() !== query || !this.filterOpen) return
      const buttons = data.hits.filter(hit=>hit.type === kind).map(hit=>{const button = node('button','',hit.name ?? '');button.type = 'button';button.dataset.filterHit = kind;button.dataset.id = String(hit.id);return button});suggestions.replaceChildren(...buttons);suggestions.hidden = !buttons.length
    }catch(error) {if(this.alive && this.searches.get(kind) === request && !isAbortError(error) && suggestions.isConnected) suggestions.hidden = true}
    finally {if(this.searches.get(kind) === request) this.searches.delete(kind)}
  }
  focus(event:FocusEvent) {
    const input = eventElement(event.target);if(!(input instanceof HTMLInputElement) || !isPickerKind(input.dataset.filterSearch)) return
    const kind = input.dataset.filterSearch,suggestions = this.list?.querySelector<HTMLElement>(`[data-filter-suggestions="${kind}"]`);if(!suggestions) return
    this.stopTimer(this.blurTimers.get(kind));this.blurTimers.delete(kind)
    if(event.type === 'focusin') {if(!input.readOnly && suggestions.querySelectorAll('button').length && input.value.trim().length >= 2) suggestions.hidden = false}
    else this.blurTimers.set(kind,this.after(150,()=>{this.blurTimers.delete(kind);if(suggestions.isConnected && document.activeElement !== input) suggestions.hidden = true}))
  }
  private reconcile(root:HTMLElement,container:HTMLElement) {
    for(const divider of container.querySelectorAll('.kill-day-divider')) divider.remove()
    let previous = '';const rows = [...container.querySelectorAll<HTMLElement>('.kill-row')]
    rows.forEach((row,index)=>{row.classList.toggle('alternate',index%2 === 0);const date = row.querySelector<HTMLTimeElement>('time')?.dateTime ?? '',day = date.slice(0,10);if(previous && day && day !== previous) {const divider = node('div','kill-day-divider');divider.append(node('span'),node('strong','',friendlyDay(date)),node('span'));container.insertBefore(divider,row)}previous = day})
    const first = rows[0]?.querySelector<HTMLTimeElement>('time')?.dateTime;if(first) requiredElement(root,'.kill-day-heading').textContent = friendlyDay(first);this.updateTimes()
  }
  private live(value:unknown) {
    const kill = parseLiveKill(value),root = this.list;if(!this.alive || document.hidden || !root || !kill || !matchesQuickFilters(kill,this.filters)) return
    if(root.querySelector(`[data-kill-id="${kill.killmail_id}"]`)) return
    if(this.page !== 1 || this.listRead) {this.newCount++;this.renderNewCount();return}
    const row = liveRow(kill);if(!row) return
    let container = root.querySelector<HTMLElement>('.kill-rows')
    if(!container) {container = node('div','kill-rows');const header = node('div','kill-head');for(const label of ['Ship','Victim','Final Blow','Location','Details']) header.append(node('span','',label));container.append(header);root.querySelector('.kill-empty')?.replaceWith(container);if(!container.isConnected) root.append(container)}
    const rows = [...container.querySelectorAll<HTMLElement>('.kill-row')];if(rows.length && Number(row.dataset.killId) <= Number(rows[0].dataset.killId)) return
    container.insertBefore(row,rows[0] ?? null);for(const old of [...container.querySelectorAll('.kill-row')].slice(this.limit)) old.remove();this.reconcile(root,container)
  }
  private startLive() {
    if(!this.alive || document.hidden || !this.list) return
    this.updateTimes();this.renderNewCount();if(!this.timesTimer) this.timesTimer = this.after(30000,()=>{this.timesTimer = undefined;this.startLive()})
    if(this.relay) return
    const kind = this.list.dataset.kind ?? 'latest'
    this.relay = createRelayConnection({wsUrl:location.origin,path:'/ws/killlist',reconnect:true,shouldReconnect:()=>this.alive && !document.hidden,onState:connected=>this.connected(connected),onOpen:socket=>socket.send(JSON.stringify({action:'subscribe',topics:[kind === 'latest' ? 'all' : kind]})),onMessage:(channel,data)=>{if(channel === 'killlist' && isRecord(data)) this.live(data.killmail)}})
    this.relay.connect()
  }
  visibility() {if(document.hidden) this.stopLive();else this.startLive()}
  private stopLive() {this.stopTimer(this.timesTimer);this.timesTimer = undefined;this.relay?.dispose();this.relay = undefined;this.connected(false)}
  hide() {this.alive = false;this.listRead?.abort();this.valueRead?.abort();this.listRead = this.valueRead = undefined;this.list?.removeAttribute('aria-busy');this.valuable?.removeAttribute('aria-busy');this.clearDrafts(false);this.stopLive();super.hide()}
  show(event:PageTransitionEvent) {if(!event.persisted) return;this.alive = true;this.syncTab();this.renderFilters();this.startLive();if(this.listIntent) void this.loadKills(this.listIntent.page,this.listIntent.limit,this.listIntent.fresh);if(this.valueIntent) void this.loadValuable(this.valueIntent)}
  disconnect() {this.hide();super.disconnect()}
}

export class HomeTopController extends RequestComponent {
  private alive = false
  private loading = false
  private failed = false
  private revision = 0
  connect() {this.alive = true;void this.refresh()}
  async refresh() {
    const page = this.element.closest<HTMLElement>('[data-kills-page]');if(!this.alive || this.loading || this.failed || !page || matchMedia('(max-width: 767px)').matches && page.dataset.tab !== 'top') return
    const kind = page.querySelector<HTMLElement>('#kill-list')?.dataset.kind,type = this.element.dataset.topPending;if(!kind || !type) return
    this.loading = true;this.element.dataset.loading = 'true';const revision = ++this.revision
    try {const html = await this.text('/_fragments/kills/top?'+new URLSearchParams({type:kind,dataType:type}),'Could not load this list.',{headers:{Accept:'text/html'}});if(!this.alive || revision !== this.revision) return;const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('.top-box');if(next) {if(next.dataset.topType !== type) throw new Error('Invalid top list');replaceFragment(this.element,next)}else if(!html.trim()) this.element.remove();else throw new Error('Invalid top list')}
    catch(error) {if(this.alive && revision === this.revision && !isAbortError(error)) {this.failed = true;this.element.dataset.loading = 'failed';this.element.classList.remove('top-box-loading');const retry = node('button','top-box-retry','Could not load this list. Retry');retry.type = 'button';retry.dataset.topRetry = '';requiredElement(this.element,'.top-box-entries').replaceChildren(retry)}}
    finally {if(revision === this.revision) this.loading = false}
  }
  click(event:MouseEvent) {if(eventElement(event.target)?.closest('[data-top-retry]')) {this.failed = false;void this.refresh()}}
  hide() {this.alive = false;this.revision++;this.loading = false;super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;void this.refresh()}}
  disconnect() {this.hide();super.disconnect()}
}
registerController('home',HomeController)
registerController('home-top',HomeTopController)
