import {FragmentSections} from './controllers/fragment-sections'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'
import {marketID} from './shared/market'
import {KillPages,matchingKillPage} from './shared/kill-paging'
import {preferredDefaultTab} from './default-tab'
import {readBrowserSession} from './browser-request'

type Tab = 'dashboard'|'kills'|'battles'
type Category = 'kills'|'ships'|'structures'
type Part = 'tab'|'kills'|'valuable'|'battles'
type Navigation = {tab:Tab;mode:boolean|'replace'}
const isTab = (value:unknown):value is Tab=>value === 'dashboard' || value === 'kills' || value === 'battles'
const isCategory = (value:unknown):value is Category=>value === 'kills' || value === 'ships' || value === 'structures'

export class LocationController extends FragmentSections<Part> {
  protected retryAttribute = 'data-location-retry'
  private alive = false
  private generation = 0
  private preferenceCanceled = false
  private cursors = new KillPages()
  private navigation:Navigation|undefined
  private paging:{page:number;limit:number}|undefined
  private category:Category|undefined
  private battlePage:number|undefined
  private get identity() {const kind = this.element.dataset.locationKind,id = marketID(this.element.dataset.locationId);if(!id || kind !== 'system' && kind !== 'region' && kind !== 'constellation') throw new Error('Invalid location');return {kind,id,base:`/${kind}/${id}`,fragments:kind === 'system' ? '/_fragments/system' : '/_fragments/location/'+kind}}
  private get main() {return this.q<HTMLElement>('[data-system-main]')}
  connect() {
    this.alive = true;const generation = ++this.generation
    void readBrowserSession().catch(()=>null).then(()=>{
      if(!this.alive || generation !== this.generation || this.preferenceCanceled || this.navigation) return
      const {kind,base} = this.identity,preferred = preferredDefaultTab(kind,base,location.href,[...this.all<HTMLAnchorElement>('a[data-system-tab]')].map(link=>new URL(link.href).pathname),document.cookie),tab = preferred?.pathname.split('/').at(-1)
      if(isTab(tab)) void this.showTab({tab,mode:'replace'})
    })
  }
  private async showTab(intent:Navigation) {
    if(!this.alive) return
    // A return to the currently displayed tab must cancel a different pending
    // intent too. Otherwise its late response would undo Back/the second click.
    this.cancelSection('tab');this.navigation = undefined;this.main.querySelector('[data-section-error="tab"]')?.remove()
    if(this.main.dataset.systemTab === intent.tab) return
    this.navigation = intent;const {kind,id,base,fragments} = this.identity,path = intent.tab === 'dashboard' ? base : base+'/'+intent.tab
    await this.section('tab',{container:this.main,path:`${fragments}/tab/${id}/${intent.tab}`,selector:'[data-system-main]',message:'Could not load this tab.',errorClass:'widget-error',replaceRoot:true,
      beforeCommit:next=>{if(next.dataset.locationKind !== kind || marketID(next.dataset.locationId) !== id || next.dataset.systemTab !== intent.tab) throw new Error('Invalid location tab')},
      loaded:next=>{
        this.cancelSection('kills');this.cancelSection('battles');this.paging = undefined;this.battlePage = undefined;this.cursors.clear()
        for(const link of this.all<HTMLAnchorElement>('a[data-system-tab]')) {const active = link.dataset.systemTab === intent.tab;link.classList.toggle('active',active);if(active) link.setAttribute('aria-current','page');else link.removeAttribute('aria-current')}
        if(intent.mode) history[intent.mode === 'replace' ? 'replaceState' : 'pushState'](history.state,'',path)
        document.title = next.dataset.pageTitle || document.title
        const description = document.querySelector<HTMLMetaElement>('meta[name="description"]');if(description && next.dataset.pageDescription) description.content = next.dataset.pageDescription
        const canonical = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');if(canonical) canonical.href = 'https://eve-kill.com'+path
        this.navigation = undefined
      }})
  }
  private async valuable(category:Category) {
    if(!this.alive) return
    const current = this.element.querySelector<HTMLElement>('#most-valuable');if(!current) return
    this.cancelSection('valuable');this.category = undefined;current.querySelector('[data-section-error="valuable"]')?.remove();if(current.dataset.kind === category) return
    this.category = category;const {id,fragments} = this.identity
    await this.section('valuable',{container:current,path:`${fragments}/valuable/${id}?category=${category}`,selector:'#most-valuable',message:'Could not load valuable kills.',errorClass:'widget-error',replaceRoot:true,beforeCommit:next=>{if(next.dataset.kind !== category) throw new Error('Invalid valuable category')},loaded:()=>{this.category = undefined}})
  }
  private async kills(page:number,limit:number) {
    const current = this.element.querySelector<HTMLElement>('#kill-list');if(!this.alive || !current) return
    const params = this.cursors.params(current,page,limit);if(!params) return
    this.paging = {page,limit};const {kind,id,fragments} = this.identity
    await this.section('kills',{container:current,path:`${fragments}/kills/${id}?${params}`,selector:'#kill-list',message:'Could not load kills.',errorClass:'widget-error',replaceRoot:true,beforeCommit:next=>{if(!matchingKillPage(next,kind,page,limit)) throw new Error('Invalid kill page')},loaded:()=>{if(page === 1) this.cursors.clear();this.paging = undefined}})
  }
  private async battles(page:number) {
    const current = this.element.querySelector<HTMLElement>('[data-entity-conflicts="battles"]');if(!this.alive || !current || !Number.isSafeInteger(page) || page < 1 || page > 1_000_000) return
    this.battlePage = page;const {kind,id,fragments} = this.identity
    await this.section('battles',{container:current,path:`${fragments}/battles/${id}?page=${page}`,selector:'[data-entity-conflicts="battles"]',message:'Could not load battles.',errorClass:'widget-error',replaceRoot:true,beforeCommit:next=>{if(next.dataset.entityKind !== kind || marketID(next.dataset.entityId) !== id || Number(next.dataset.page) !== page) throw new Error('Invalid battle page')},loaded:()=>{this.battlePage = undefined}})
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    const link = target.closest<HTMLAnchorElement>('a[data-system-tab]'),tab = link?.dataset.systemTab
    if(link && isTab(tab) && !event.defaultPrevented && event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey && !link.hasAttribute('download') && (!link.target || link.target === '_self')) {this.preferenceCanceled = true;event.preventDefault();void this.showTab({tab,mode:true});return}
    const retry = target.closest<HTMLElement>('[data-location-retry]')?.dataset.locationRetry
    if(retry === 'tab' && this.navigation && !this.sectionPending('tab')) {void this.showTab(this.navigation);return}
    if(retry === 'valuable' && this.category && !this.sectionPending('valuable')) {void this.valuable(this.category);return}
    if(retry === 'kills' && this.paging && !this.sectionPending('kills')) {void this.kills(this.paging.page,this.paging.limit);return}
    if(retry === 'battles' && this.battlePage && !this.sectionPending('battles')) {void this.battles(this.battlePage);return}
    const category = target.closest<HTMLElement>('[data-valuable-tab]')?.dataset.valuableTab;if(isCategory(category)) {void this.valuable(category);return}
    const limit = target.closest<HTMLElement>('[data-kill-limit]'),page = target.closest<HTMLButtonElement>('[data-kill-page]'),list = this.element.querySelector<HTMLElement>('#kill-list')
    if(list && limit) {this.cursors.clear();void this.kills(1,Number(limit.dataset.killLimit));return}
    if(list && page && !page.disabled) {void this.kills(Number(page.dataset.killPage),Number(list.dataset.limit));return}
    const battle = target.closest<HTMLButtonElement>('[data-conflict-page]');if(battle && !battle.disabled) void this.battles(Number(battle.dataset.conflictPage))
  }
  history() {
    if(!this.alive) return
    const {base} = this.identity,path = location.pathname,tab = path === base ? 'dashboard' : path.startsWith(base+'/') ? path.slice(base.length+1) : ''
    if(isTab(tab)) {this.preferenceCanceled = true;void this.showTab({tab,mode:false})}
  }
  hover(event:MouseEvent) {
    if(!this.alive) return
    const hour = eventElement(event.target)?.closest<HTMLElement>('[data-chart-hour]'),chart = hour?.closest<HTMLElement>('[data-system-chart]');if(!chart || !hour) return
    chart.classList.add('is-hovering');for(const slot of chart.querySelectorAll<HTMLElement>('[data-chart-hour]')) slot.classList.toggle('is-hovered',slot.dataset.chartHour === hour.dataset.chartHour)
    const readout = chart.querySelector<HTMLElement>('[data-chart-readout]');if(readout) readout.textContent = hour.dataset.recorded === 'false' ? `${hour.dataset.hourLabel} · not recorded` : `${hour.dataset.hourLabel} · Jumps ${Number(hour.dataset.jumps).toLocaleString()} · Ship kills ${Number(hour.dataset.shipKills).toLocaleString()} · NPC kills ${Number(hour.dataset.npcKills).toLocaleString()} · Pod kills ${Number(hour.dataset.podKills).toLocaleString()}`
  }
  leave(event:MouseEvent) {
    const chart = eventElement(event.target)?.closest('[data-chart-hour]')?.closest('[data-system-chart]');if(!chart || event.relatedTarget instanceof Node && chart.contains(event.relatedTarget)) return
    chart.classList.remove('is-hovering');for(const slot of chart.querySelectorAll('.is-hovered')) slot.classList.remove('is-hovered');const readout = chart.querySelector<HTMLElement>('[data-chart-readout]');if(readout) readout.textContent = 'Hover an hour for exact figures'
  }
  hide() {this.alive = false;this.generation++;super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.navigation) void this.showTab(this.navigation);else if(this.paging) void this.kills(this.paging.page,this.paging.limit);else if(this.battlePage) void this.battles(this.battlePage);if(this.category) void this.valuable(this.category)}}
}
registerController('location',LocationController)
