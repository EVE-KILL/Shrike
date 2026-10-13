import {FragmentSections} from './fragment-sections'
import {registerController} from './application'
import {eventElement} from '../shared/dom'
import {entityIdentity,entityPeriods,entityNode as node} from '../shared/entity'
import {searchID} from '../shared/search-model'
import {isRecord,isAbortError} from '../shared/json'
import {KillPages,matchingKillPage} from '../shared/kill-paging'
import {replaceFragment,replaceFragmentChildren} from '../fragment'
import {createRelayConnection} from '../../utils/relaySocketCore'

type Part = 'dashboard'|'top-lists'|'faction'|'militia'|'kills'|'members'|'corporations'|'conflicts'|'top:left'|'top:right'
type MemberSort = 'name'|'last_active'|'security_status'
type MemberChoice = {page:number;sort:MemberSort;activity:number;corporation_id:number}
const memberSort = (value:unknown):value is MemberSort=>value === 'name' || value === 'last_active' || value === 'security_status'
const validPage = (value:number)=>Number.isSafeInteger(value) && value > 0 && value <= 1_000_000

// The scope is the currently rendered tab. Replacing a tab disconnects all its
// reads, sockets and timers; replacing one table retains its siblings and pager.
export class EntityContentController extends FragmentSections<Part> {
  protected retryAttribute = 'data-entity-content-retry'
  private alive = false
  private activated = false
  private intents = new Map<Part,()=>void>()
  private memberChoice:MemberChoice|undefined
  private intelRead:AbortController|undefined
  private militiaRead:AbortController|undefined
  private intelDays = 90
  private intelLoaded = false
  private intelSections = new Map<string,HTMLElement>()
  private cursors = new KillPages()
  private relay:ReturnType<typeof createRelayConnection>|undefined
  private refreshTimer:ReturnType<typeof setTimeout>|undefined
  private timesTimer:ReturnType<typeof setTimeout>|undefined
  private newCount = 0
  private topLoaded = false
  private get identity() {return entityIdentity(this.element)}
  private get list() {return this.element.querySelector<HTMLElement>('#entity-kill-list')}
  connect() {this.alive = true;if(this.element.closest<HTMLElement>('.entity-page')?.dataset.entityReady !== 'false') this.activate()}
  activate() {
    if(!this.alive) return
    if(this.activated) {for(const run of [...this.intents.values()]) run();if(!this.intelLoaded && !this.intelRead && this.element.querySelector('[data-entity-intel]')) void this.intel(this.intelDays);this.startLive();return}
    this.activated = true
    if(this.element.querySelector('[data-entity-intel]')) void this.intel(90)
    if(this.element.querySelector('[data-faction-overview]')) void this.faction()
    void this.topLists();this.startLive()
  }
  private async read(part:Part,resolve:()=>HTMLElement|null,path:string,selector:string,message:string,validate?:(next:HTMLElement)=>void,loaded?:(next:HTMLElement)=>void,replaceRoot=true) {
    const root = resolve();if(!this.alive || !root) return
    this.intents.set(part,()=>{void this.read(part,resolve,path,selector,message,validate,loaded,replaceRoot)})
    return this.section(part,{container:root,path,selector,message,errorClass:'widget-error',replaceRoot,beforeCommit:validate,loaded:next=>{this.intents.delete(part);loaded?.(next)}})
  }
  private errorIntel() {
    const root = this.element.querySelector<HTMLElement>('[data-entity-intel]');if(!root) return
    root.querySelector('[data-entity-intel-error]')?.remove();const notice = node('p','widget-error','Could not load intel. Try again. '),retry = node('button','','Retry');notice.dataset.entityIntelError = '';retry.type = 'button';retry.dataset.entityIntelRetry = '';notice.append(retry);root.append(notice)
  }
  private placeIntel() {
    for(const [name,section] of this.intelSections) {const slot = this.element.querySelector<HTMLElement>(`[data-entity-intel-slot="${name}"]`);if(slot && !(slot.childElementCount === 1 && slot.firstElementChild === section)) slot.replaceChildren(section)}
  }
  private async intel(days:number) {
    const root = this.element.querySelector<HTMLElement>('[data-entity-intel]');if(!this.alive || !root || !entityPeriods.has(days)) return
    const {kind,id} = this.identity;this.intelRead?.abort();const request = new AbortController();this.intelRead = request;this.intelDays = days;this.intelLoaded = false;root.querySelector('[data-entity-intel-error]')?.remove();root.setAttribute('aria-busy','true')
    try {
      const html = await this.text(`/_fragments/entity/intel/${kind}/${id}?days=${days}`,'Could not load intel.',{signal:request.signal}),content = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('.entity-intel-content')
      if(!this.alive || this.intelRead !== request) return
      if(!content || content.dataset.entityKind !== kind) throw new Error('Invalid entity intel')
      if(kind === 'corporation' || kind === 'alliance') {this.intelSections.clear();for(const section of content.querySelectorAll<HTMLElement>('[data-entity-intel-section]')) if(['active','relations','recent'].includes(section.dataset.entityIntelSection ?? '')) this.intelSections.set(section.dataset.entityIntelSection!,section);this.placeIntel()}
      else {const current = this.element.querySelector<HTMLElement>('[data-entity-intel]');if(current) replaceFragmentChildren(current,content)}
      this.intelLoaded = true
    }catch(error) {if(this.alive && this.intelRead === request && !isAbortError(error)) {this.intelLoaded = false;this.errorIntel()}}
    finally {if(this.intelRead === request) {this.intelRead = undefined;this.element.querySelector('[data-entity-intel]')?.removeAttribute('aria-busy')}}
  }
  private async dashboard(days:number) {
    if(!entityPeriods.has(days)) return
    const {kind,id} = this.identity
    if(kind === 'character') void this.intel(days)
    for(const button of this.all<HTMLElement>('[data-entity-period]')) button.classList.toggle('active',Number(button.dataset.entityPeriod) === days)
    await this.read('dashboard',()=>this.element.querySelector('[data-entity-dashboard-stats]'),`/_fragments/entity/dashboard/${kind}/${id}?days=${days}`,'.entity-dashboard-content','Could not load dashboard. Try again.',next=>{if(Number(next.dataset.days) !== days) throw new Error('Invalid dashboard period')},()=>this.placeIntel(),false)
  }
  private async faction() {
    const {id} = this.identity;this.militiaRead?.abort();this.militiaRead = undefined;this.intents.delete('militia')
    await this.read('faction',()=>this.element.querySelector('[data-faction-overview]'),`/_fragments/entity/faction/${id}`,'.entity-faction-overview','Could not load warzone. Try again.',next=>{if(searchID(next.dataset.entityId) !== id) throw new Error('Invalid warzone')},()=>{void this.militia()},false)
  }
  private async militia() {
    const slot = this.element.querySelector<HTMLElement>('[data-faction-militia]');if(!this.alive || !slot) return
    const {id} = this.identity;this.militiaRead?.abort();const request = new AbortController();this.militiaRead = request;this.intents.set('militia',()=>{void this.militia()});slot.querySelector('[data-section-error]')?.remove()
    try {
      const html = await this.text(`/_fragments/entity/faction/${id}/militia`,'Could not load militia corporations.',{signal:request.signal})
      if(!this.alive || this.militiaRead !== request || !slot.isConnected) return
      // Empty success is the Go contract for a faction without militia rows.
      if(!html.trim()) slot.remove()
      else {const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('[data-faction-militia]');if(!next) throw new Error('Invalid militia corporations');replaceFragment(slot,next)}
      this.intents.delete('militia')
    }catch(error) {if(this.alive && this.militiaRead === request && !isAbortError(error) && slot.isConnected) {const notice = node('p','widget-error','Could not load militia corporations. '),retry = node('button','','Retry');notice.dataset.sectionError = 'militia';retry.type = 'button';retry.dataset.entityContentRetry = 'militia';notice.append(retry);slot.append(notice);slot.hidden = false}}
    finally {if(this.militiaRead === request) this.militiaRead = undefined}
  }
  private async topLists() {
    if(!this.alive || this.topLoaded || this.sectionPending('top-lists') || !matchMedia('(min-width: 768px)').matches) return
    const {kind,id} = this.identity,role = this.list?.dataset.role ?? 'combined'
    await this.read('top-lists',()=>this.element.querySelector('[data-entity-top-lists]'),`/_fragments/entity/top-lists/${kind}/${id}?tab=${role}`,'.entity-top-lists-body','Could not load top lists.',undefined,()=>{this.topLoaded = true},false)
  }
  resize() {if(this.activated) void this.topLists()}
  private async top(side:'left'|'right',days:string) {
    const root = this.element.querySelector<HTMLElement>(`[data-entity-top-slice="${side}"]`);if(!root || ![...root.querySelectorAll<HTMLElement>('[data-top-period]')].some(button=>button.dataset.topPeriod === days)) return
    const {kind,id} = this.identity,part:Part = side === 'left' ? 'top:left' : 'top:right',params = new URLSearchParams({slice:side,days})
    await this.read(part,()=>this.element.querySelector(`[data-entity-top-slice="${side}"]`),`/_fragments/entity/top/${kind}/${id}?${params}`,'[data-entity-top-slice]','Could not load top lists. Try again.',next=>{if(next.dataset.entityKind !== kind || next.dataset.entityTopSlice !== side || next.dataset.days !== days) throw new Error('Invalid top list')})
  }
  private async members(choice:MemberChoice) {
    if(!validPage(choice.page) || !memberSort(choice.sort) || ![0,7,30,90].includes(choice.activity) || choice.corporation_id !== 0 && !searchID(choice.corporation_id)) return
    this.memberChoice = choice;const {kind,id} = this.identity,params = new URLSearchParams({page:String(choice.page),sort:choice.sort,activity:String(choice.activity)});if(choice.corporation_id) params.set('corporation_id',String(choice.corporation_id))
    await this.read('members',()=>this.element.querySelector('[data-entity-members]'),`/_fragments/entity/members/${kind}/${id}?${params}`,'[data-entity-members]','Could not load members. Try again.',next=>{if(next.dataset.kind !== kind || searchID(next.dataset.id) !== id || Number(next.dataset.page) !== choice.page || next.dataset.sort !== choice.sort || Number(next.dataset.activity) !== choice.activity || Number(next.dataset.corp) !== choice.corporation_id) throw new Error('Invalid member list')},()=>{this.memberChoice = undefined})
  }
  private async corporations(sort:'name'|'members',dir:'asc'|'desc') {
    const {id} = this.identity,params = new URLSearchParams({sort,dir})
    await this.read('corporations',()=>this.element.querySelector('[data-entity-corporations]'),`/_fragments/entity/corporations/${id}?${params}`,'[data-entity-corporations]','Could not load corporations. Try again.',next=>{if(searchID(next.dataset.id) !== id || next.dataset.sort !== sort || next.dataset.dir !== dir) throw new Error('Invalid corporation list')})
  }
  private async conflicts(type:'battles'|'wars'|'campaigns',page:number,sort:string) {
    if(!validPage(page) || type === 'wars' && !['recent','kills','isk'].includes(sort)) return
    const {kind,id} = this.identity,params = new URLSearchParams({page:String(page)});if(type === 'wars') params.set('sort',sort)
    await this.read('conflicts',()=>this.element.querySelector('[data-entity-conflicts]'),`/_fragments/entity/${type}/${kind}/${id}?${params}`,'[data-entity-conflicts]',`Could not load ${type}. Try again.`,next=>{if(next.dataset.entityConflicts !== type || next.dataset.entityKind !== kind || searchID(next.dataset.entityId) !== id || Number(next.dataset.page) !== page || type === 'wars' && next.dataset.sort !== sort) throw new Error('Invalid conflict list')})
  }
  private async kills(page:number,limit:number) {
    const current = this.list;if(!this.alive || !current) return
    const params = this.cursors.params(current,page,limit),role = current.dataset.role,group = searchID(current.dataset.shipGroup);if(!params || !['combined','kills','losses'].includes(role ?? '')) return
    params.set('role',role!);if(group) params.set('ship_group',String(group));const {kind,id} = this.identity,countAtStart = this.newCount
    await this.read('kills',()=>this.list,`/_fragments/entity/killlist/${kind}/${id}?${params}`,'#entity-kill-list','Could not load kills. Try again.',next=>{if(!matchingKillPage(next,kind,page,limit) || searchID(next.dataset.entityId) !== id || next.dataset.role !== role || Number(next.dataset.shipGroup) !== group) throw new Error('Invalid kill list')},()=>{if(page === 1) this.newCount = Math.max(0,this.newCount-countAtStart);this.setConnected(this.relay?.connected ?? false);this.showNewCount();this.updateTimes()})
  }
  private setConnected(connected:boolean) {const dot = this.list?.querySelector<HTMLElement>('.kill-live-dot');if(dot) {dot.classList.toggle('connected',connected);dot.title = connected ? 'Live — connected' : 'Disconnected';dot.setAttribute('aria-label',dot.title)}}
  private showNewCount() {const button = this.list?.querySelector<HTMLElement>('.new-kills');if(button) {button.hidden = !this.newCount;const count = button.querySelector('span');if(count) count.textContent = String(this.newCount)}}
  private updateTimes() {for(const time of this.all<HTMLTimeElement>('#entity-kill-list time[datetime]')) {const minutes = Math.floor((Date.now()-Date.parse(time.dateTime))/60000);time.textContent = minutes < 1 ? 'just now' : minutes < 30 ? `${minutes}m ago` : time.dateTime.slice(11,16)}}
  private startLive() {
    if(!this.alive || document.hidden || !this.list) return
    this.updateTimes();this.showNewCount()
    if(!this.timesTimer) this.timesTimer = this.after(30000,()=>{this.timesTimer = undefined;this.startLive()})
    if(this.relay) return
    const {id} = this.identity
    this.relay = createRelayConnection({wsUrl:location.origin,path:'/ws/killlist',reconnect:true,shouldReconnect:()=>this.alive && !document.hidden && Boolean(this.list),onState:connected=>this.setConnected(connected),onOpen:socket=>socket.send(JSON.stringify({action:'subscribe',topics:[`victim.${id}`,`attacker.${id}`]})),onMessage:(channel,data,keys)=>{
      const list = this.list;if(!this.alive || document.hidden || channel !== 'killlist' || !list || !isRecord(data) || !isRecord(data.killmail)) return
      const role = list.dataset.role;if(role === 'kills' && !keys.includes(`attacker.${id}`) || role === 'losses' && !keys.includes(`victim.${id}`)) return
      const group = searchID(list.dataset.shipGroup);if(group && searchID(data.killmail.ship_group_id) !== group) return
      if(Number(list.dataset.page) !== 1 || list.getAttribute('aria-busy') === 'true') {this.newCount++;this.showNewCount();return}
      this.stopTimer(this.refreshTimer);this.refreshTimer = this.after(250,()=>{this.refreshTimer = undefined;void this.kills(1,Number(this.list?.dataset.limit ?? 50))})
    }});this.relay.connect()
  }
  private stopLive() {this.relay?.dispose();this.relay = undefined;this.stopTimer(this.refreshTimer);this.refreshTimer = undefined;this.stopTimer(this.timesTimer);this.timesTimer = undefined;this.setConnected(false)}
  private achievements(target:Element) {
    const root = target.closest<HTMLElement>('.entity-achievements');if(!root) return false
    const statusButton = target.closest<HTMLElement>('[data-achievement-status]'),categoryButton = target.closest<HTMLElement>('[data-achievement-category]')
    if(statusButton) for(const button of root.querySelectorAll('[data-achievement-status]')) button.classList.toggle('active',button === statusButton)
    else if(categoryButton) {const active = categoryButton.classList.contains('active') && Boolean(categoryButton.dataset.achievementCategory),selected = active ? root.querySelector('[data-achievement-category=""]') : categoryButton;for(const button of root.querySelectorAll('[data-achievement-category]')) button.classList.toggle('active',button === selected)}else return false
    const status = root.querySelector<HTMLElement>('[data-achievement-status].active')?.dataset.achievementStatus ?? 'all',category = root.querySelector<HTMLElement>('[data-achievement-category].active')?.dataset.achievementCategory ?? '';let visible = 0
    for(const card of root.querySelectorAll<HTMLElement>('.achievement-card')) {const completed = card.dataset.completed === 'true',progressed = Number(card.dataset.progress) > 0;card.hidden = Boolean(category && card.dataset.category !== category) || status === 'completed' && !completed || status === 'in_progress' && (completed || !progressed);if(!card.hidden) visible++}
    const empty = root.querySelector<HTMLElement>('.achievement-empty');if(empty) empty.hidden = visible !== 0;return true
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    const retry = target.closest<HTMLElement>('[data-entity-content-retry]')?.dataset.entityContentRetry;if(retry) {for(const [part,run] of this.intents) if(part === retry && !this.sectionPending(part)) {run();return}}
    if(target.closest('[data-entity-intel-retry]')) {void this.intel(this.intelDays);return}
    if(this.achievements(target)) return
    const period = target.closest<HTMLElement>('[data-entity-period]');if(period) {void this.dashboard(Number(period.dataset.entityPeriod));return}if(target.closest('[data-entity-show-all]')) {void this.dashboard(0);return}
    const top = target.closest<HTMLElement>('[data-top-period]'),side = top?.closest<HTMLElement>('[data-entity-top-slice]')?.dataset.entityTopSlice;if(top && (side === 'left' || side === 'right')) {void this.top(side,top.dataset.topPeriod ?? '');return}
    const member = target.closest<HTMLElement>('[data-member-page],[data-member-sort],[data-member-activity],[data-member-corp]'),members = this.element.querySelector<HTMLElement>('[data-entity-members]')
    if(member && members) {const sort = members.dataset.sort,choice:MemberChoice = {...this.memberChoice ?? {page:Number(members.dataset.page) || 1,sort:memberSort(sort) ? sort : 'name',activity:Number(members.dataset.activity) || 0,corporation_id:searchID(members.dataset.corp)}};if(member.dataset.memberPage) choice.page = Number(member.dataset.memberPage);if(memberSort(member.dataset.memberSort)) {choice.sort = member.dataset.memberSort;choice.page = 1}if(member.dataset.memberActivity !== undefined) {const activity = Number(member.dataset.memberActivity);choice.activity = activity === choice.activity ? 0 : activity;choice.page = 1}if(member.dataset.memberCorp !== undefined) {choice.corporation_id = Number(member.dataset.memberCorp);choice.page = 1}void this.members(choice);return}
    const corporation = target.closest<HTMLElement>('[data-corp-sort]'),corps = this.element.querySelector<HTMLElement>('[data-entity-corporations]'),sort = corporation?.dataset.corpSort;if(corps && (sort === 'name' || sort === 'members')) {const dir = sort === corps.dataset.sort ? (corps.dataset.dir === 'desc' ? 'asc' : 'desc') : (sort === 'members' ? 'desc' : 'asc');void this.corporations(sort,dir);return}
    const conflict = target.closest<HTMLElement>('[data-conflict-page],[data-war-sort]'),conflicts = this.element.querySelector<HTMLElement>('[data-entity-conflicts]'),type = conflicts?.dataset.entityConflicts;if(conflict && conflicts && (type === 'battles' || type === 'wars' || type === 'campaigns')) {void this.conflicts(type,conflict.dataset.warSort ? 1 : Number(conflict.dataset.conflictPage),conflict.dataset.warSort ?? conflicts.dataset.sort ?? 'recent');return}
    const list = target.closest('#entity-kill-list'),limit = target.closest<HTMLElement>('[data-entity-kill-limit]'),page = target.closest<HTMLButtonElement>('[data-kill-page]');if(list && this.list) {if(limit) {this.cursors.clear();void this.kills(1,Number(limit.dataset.entityKillLimit))}else if(page && !page.disabled) void this.kills(Number(page.dataset.killPage),Number(this.list.dataset.limit));else if(target.closest('.new-kills')) void this.kills(1,Number(this.list.dataset.limit))}
  }
  toggle(event:Event) {const filter = eventElement(event.target);if(!(filter instanceof HTMLDetailsElement) || !filter.matches('.entity-corp-filter')) return;if(filter.open) filter.querySelector<HTMLElement>('[data-corp-search]')?.focus({preventScroll:true});else {const search = filter.querySelector<HTMLInputElement>('[data-corp-search]');if(search) search.value = '';for(const button of filter.querySelectorAll<HTMLElement>('.entity-corp-filter-list button')) button.hidden = false}}
  outside(event:MouseEvent) {for(const filter of this.all<HTMLDetailsElement>('.entity-corp-filter[open]')) if(event.target instanceof Node && !filter.contains(event.target)) filter.open = false}
  keydown(event:KeyboardEvent) {const filter = eventElement(event.target)?.closest<HTMLDetailsElement>('.entity-corp-filter[open]');if(event.key === 'Escape' && filter) {event.preventDefault();filter.open = false;filter.querySelector<HTMLElement>('summary')?.focus({preventScroll:true})}}
  input(event:InputEvent) {const input = eventElement(event.target);if(!(input instanceof HTMLInputElement) || !input.matches('[data-corp-search]')) return;const query = input.value.trim().toLowerCase(),all = this.element.querySelector<HTMLElement>('.entity-corp-filter-list [data-member-corp="0"]');if(all) all.hidden = Boolean(query);for(const button of this.all<HTMLElement>('.entity-corp-filter-list [data-corp-name]')) button.hidden = !`${button.dataset.corpName ?? ''} ${button.dataset.corpTicker ?? ''}`.toLowerCase().includes(query)}
  visibility() {if(document.hidden) this.stopLive();else if(this.activated) this.startLive()}
  hide() {this.alive = false;this.stopLive();this.intelRead?.abort();this.intelRead = undefined;this.militiaRead?.abort();this.militiaRead = undefined;this.element.querySelector('[data-entity-intel]')?.removeAttribute('aria-busy');super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.activate()}}
}
registerController('entity-content',EntityContentController)
