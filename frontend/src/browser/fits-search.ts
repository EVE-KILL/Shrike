import {RequestComponent} from './controllers/request-component'
import {positionViewportPanel} from './shared/floating-panel'
import {registerController} from './controllers/application'
import {initializeSearchPickers,closeSearchPickers} from './search-picker'
import {searchParams,stateFromURL,parseStatisticValue,sortForMetric,setStatistic,statisticLabel,isFitSort,isFitComparison,type FitSearchState,type FitRequirement} from './fit-search-state'
import {isFitMetric,type FitMetric} from './shared/fit-filters'
import {readFitSearchSeed,readFitRoles,readFitAvailability,readFitResultState,type FitRole,type FitAvailability,type FitItemHit} from './shared/fit-search'
import {searchID,readSearchResults} from './shared/search-model'
import {eventElement,requiredElement,cloneComponent,formInput,placeElement} from './shared/dom'
import {escapeHTML} from './shared/html'
import {isRecord,isAbortError} from './shared/json'
import {element as node} from './kill-row'

type Part = 'results'|'distributions'|'availability'|'roles'|'popular'
type Identity = Pick<FitRequirement,'kind'> & {role_id?:string;type_id?:number}
const roleNames = {weapon:'Weapons',tank:'Tank',tackle:'Tackle',prop:'Propulsion',ewar:'EWAR',utility:'Utility'} as const
export class FitsSearchController extends RequestComponent {
 private alive = false
 private state!:FitSearchState
 private roles:FitRole[] = []
 private draftMetric:FitMetric = 'ehp'
 private draftNPC = 'guristas'
 private availability:FitAvailability|null = null
 private availabilityPending = false
 private itemPending = false
 private items:FitItemHit[] = []
 private itemRequest:AbortController|undefined
 private itemTimer:ReturnType<typeof setTimeout>|undefined
 private reads = new Map<Part,AbortController>()
 private intents = new Map<Part,boolean>()
 private offset = 0
 private more = false
 private queryGeneration = 0
 private requirementRows = new Map<FitRequirement,HTMLElement>()
 private statisticRows = new Map<FitMetric,HTMLButtonElement>()
 private sprite = ''
 connect() {
  const state:unknown = JSON.parse(this.q('#fit-search-state').textContent ?? 'null'),roles:unknown = JSON.parse(this.q('#fit-search-roles').textContent ?? '[]')
  this.state = readFitSearchSeed(state);this.roles = readFitRoles(roles ?? []);this.drafts();this.sprite = this.q<SVGUseElement>('.icon use').getAttribute('href')?.split('#')[0] ?? '/assets/icons.svg'
  this.alive = true;initializeSearchPickers(this.element);this.renderHull();this.renderFilters();if(this.state.ship) this.refresh()
 }
 private drafts() {this.draftMetric = this.state.statistics.at(-1)?.metric ?? 'ehp';this.draftNPC = this.state.statistics.at(-1)?.npcProfile ?? 'guristas'}
 private icon(name:string) {return `<svg class="icon" aria-hidden="true" viewBox="0 0 24 24"><use href="${this.sprite}#${escapeHTML(name)}"></use></svg>`}
 private roleCount(id:string) {return !this.state.ship ? 1 : this.availability ? this.availability.role_counts[id] ?? 0 : this.availabilityPending ? 0 : 1}
 private typeCount(id:number) {return !this.state.ship ? 1 : this.availability ? this.availability.type_counts[String(id)] ?? 0 : this.availabilityPending ? 0 : 1}
 private picked(filter:Identity) {return this.state.filters.some(item=>item.kind === filter.kind && (item.kind === 'role' ? item.role_id === filter.role_id : item.type_id === filter.type_id))}
 private syncURL() {const url = new URL(location.href);url.search = searchParams(this.state,{name:true}).toString();history.replaceState(history.state,'',url)}
 private renderHull() {
  const ship = this.state.ship,selected = Boolean(ship)
  for(const el of this.all<HTMLElement>('[data-fit-search-selected-only]')) el.hidden = !selected
  this.q<HTMLElement>('[data-fit-search-selected-card]').hidden = !selected;this.q<HTMLElement>('[data-fit-search-unselected]').hidden = selected;this.q<HTMLElement>('[data-fit-search-empty]').hidden = selected
  this.q<HTMLElement>('[data-fit-search-profiles]').hidden = !selected;this.q<HTMLElement>('[data-fit-search-results]').hidden = !selected;this.q('[data-fit-search-criteria]').classList.toggle('fit-search-criteria',selected)
  if(ship) {const url = `/images/types/${ship.id}/render?size=256`,image = this.q<HTMLImageElement>('[data-fit-search-hull-image]');if(image.getAttribute('src') !== url) image.src = url;image.alt = ship.name;this.q<HTMLImageElement>('[data-fit-search-hull-background]').src = url;this.q('[data-fit-search-hull-name]').textContent = ship.name;this.q<HTMLAnchorElement>('[data-fit-search-hull-link]').href = `/item/${ship.id}/fittings`}
 }
 private renderChoices() {
  for(const name of ['sort','metric','npc'] as const) {const value = name === 'sort' ? this.state.sort : name === 'metric' ? this.draftMetric : this.draftNPC,control = this.q(`[data-fit-search-choice-control="${name}"]`);for(const option of control.querySelectorAll<HTMLElement>('[data-fit-search-choice]')) {const on = option.dataset.fitSearchValue === value;option.classList.toggle('active',on);option.setAttribute('aria-pressed',String(on));if(on) requiredElement(control,'[data-fit-search-choice-label]').textContent = option.querySelector('span')?.textContent ?? ''}}
  this.q<HTMLElement>('[data-fit-search-npc-field]').hidden = this.draftMetric !== 'npc_ehp';this.q<HTMLElement>('[data-fit-search-npc-spacer]').hidden = this.draftMetric === 'npc_ehp';this.q<HTMLInputElement>('[data-fit-search-cap-stable]').checked = this.state.capStable;this.q<HTMLElement>('[data-fit-search-clear-stats]').hidden = !this.state.statistics.length && !this.state.capStable
 }
 private renderFilters() {
  const focused = document.activeElement instanceof HTMLElement ? document.activeElement : null,oldGroup = focused?.closest<HTMLElement>('[data-fit-search-requirement]'),focusIndex = oldGroup?.dataset.fitSearchRequirement,focusKey = focused?.hasAttribute('data-fit-search-op') ? 'fitSearchOp' : focused?.hasAttribute('data-fit-search-count-step') ? 'fitSearchCountStep' : null,focusValue = focusKey ? focused?.dataset[focusKey] : undefined
  const container = this.q('[data-fit-search-requirements]');let before:ChildNode|null = container.firstChild
  for(const [index,filter] of this.state.filters.entries()) {
   const row = this.requirementRows.get(filter) ?? cloneComponent(this.element,'[data-fit-search-requirement-template]');this.requirementRows.set(filter,row);row.dataset.fitSearchRequirement = String(index)
   const role = filter.kind === 'role' ? this.roles.find(role=>role.id === filter.role_id) : undefined
   requiredElement(row,'[data-fit-search-requirement-label]').textContent = filter.kind === 'role' ? role?.label ?? filter.role_id : filter.type_name ?? `Type ${filter.type_id}`
   const image = requiredElement<HTMLImageElement>(row,'[data-fit-search-requirement-image]'),symbol = requiredElement<HTMLElement>(row,'[data-fit-search-requirement-symbol]');image.hidden = filter.kind !== 'type';symbol.hidden = filter.kind === 'type'
   if(filter.kind === 'type') {const url = `/images/types/${filter.type_id}/icon?size=32`;if(image.getAttribute('src') !== url) image.src = url}else requiredElement(symbol,'use').setAttribute('href',`${this.sprite}#${(role?.icon ?? 'lucide:circle').replace('lucide:','')}`)
   for(const op of row.querySelectorAll<HTMLElement>('[data-fit-search-op]')) {const on = op.dataset.fitSearchOp === filter.op;op.classList.toggle('active',on);op.setAttribute('aria-pressed',String(on))}
   requiredElement(row,'[data-fit-search-count]').textContent = String(filter.count);if(row !== before) placeElement(container,row,before);before = row.nextSibling
  }
  for(const [filter,row] of this.requirementRows) if(!this.state.filters.includes(filter)) {row.remove();this.requirementRows.delete(filter)}
  // Initial SSR rows are replaced once; later updates retain typed row/image nodes.
  const retained = new Set(this.requirementRows.values());for(const child of [...container.children]) if(child instanceof HTMLElement && !retained.has(child)) child.remove()
  if(focusIndex !== undefined && focusKey && focused && !focused.isConnected) this.element.querySelector<HTMLElement>(`[data-fit-search-requirement="${focusIndex}"]`)?.querySelectorAll<HTMLButtonElement>('button').forEach(button=>{if(button.dataset[focusKey] === focusValue) button.focus({preventScroll:true})})
  this.q<HTMLElement>('[data-fit-search-no-requirements]').hidden = this.state.filters.length > 0;this.q<HTMLButtonElement>('[data-fit-search-add-requirement]').disabled = this.state.filters.length >= 8;this.q('[data-fit-search-requirement-count]').textContent = this.state.filters.length ? `(${this.state.filters.length}/8)` : ''
  const chips = this.q('[data-fit-search-stat-chips]');before = chips.firstChild
  for(const [index,filter] of this.state.statistics.entries()) {let button = this.statisticRows.get(filter.metric);if(!button) {const clone = cloneComponent(this.element,'[data-fit-search-stat-template]');if(!(clone instanceof HTMLButtonElement)) throw new Error('Invalid fitting statistic template');button = clone;this.statisticRows.set(filter.metric,button)}const label = statisticLabel(filter);button.dataset.fitSearchRemoveStat = String(index);button.title = `Remove ${label}`;requiredElement(button,'[data-fit-search-stat-label]').textContent = label;if(button !== before) placeElement(chips,button,before);before = button.nextSibling}
  for(const [metric,row] of this.statisticRows) if(!this.state.statistics.some(filter=>filter.metric === metric)) {row.remove();this.statisticRows.delete(metric)}
  const statNodes = new Set(this.statisticRows.values());for(const child of [...chips.children]) if(child instanceof HTMLButtonElement && !statNodes.has(child)) child.remove()
  this.renderChoices();this.renderRoles()
 }
 private roleButton(role:FitRole,searching = false) {const count = this.roleCount(role.id),selected = this.picked({kind:'role',role_id:role.id});return `<button type="button" class="fit-search-role-option" data-fit-search-role="${escapeHTML(role.id)}"${selected ? ' disabled' : ''}>${this.icon(role.icon.replace('lucide:',''))}<span>${escapeHTML(role.label)}</span><small>${count.toLocaleString('en-US')}${searching ? ' fits' : ''}</small></button>`}
 private renderRoles() {
  const focused = document.activeElement instanceof HTMLElement ? document.activeElement : null,focusAttribute = focused?.hasAttribute('data-fit-search-role') ? 'data-fit-search-role' : focused?.hasAttribute('data-fit-search-type') ? 'data-fit-search-type' : undefined,focusValue = focusAttribute ? focused?.getAttribute(focusAttribute) : undefined
  const query = this.q<HTMLInputElement>('[data-fit-search-role-query]').value.trim().toLowerCase(),container = this.q('[data-fit-search-role-results]'),visible = this.roles.filter(role=>this.roleCount(role.id) > 0 && (!query || role.label.toLowerCase().includes(query) || role.description?.toLowerCase().includes(query)))
  container.innerHTML = query ? visible.length ? `<div class="fit-search-role-group"><div class="fit-search-role-category">Roles</div>${visible.map(role=>this.roleButton(role,true)).join('')}</div>` : '' : Object.entries(roleNames).map(([category,label])=>{const group = visible.filter(role=>role.category === category && role.typeCount > 0);return group.length ? `<div class="fit-search-role-group"><div class="fit-search-role-category">${label}</div><div class="fit-search-role-buttons">${group.map(role=>this.roleButton(role)).join('')}</div></div>` : ''}).join('')
  const items = this.items.filter(hit=>this.typeCount(hit.typeID) > 0)
  this.q('[data-fit-search-item-results]').innerHTML = query && items.length ? `<div class="fit-search-role-category" style="color:#fbbf24b3">Specific items</div>${items.map(hit=>`<button type="button" class="fit-search-role-option" data-fit-search-type="${hit.typeID}"${this.picked({kind:'type',type_id:hit.typeID}) ? ' disabled' : ''}><img src="/images/types/${hit.typeID}/icon?size=32" width="20" height="20" alt="${escapeHTML(hit.name)}"><span>${escapeHTML(hit.name)}</span><small>${this.typeCount(hit.typeID).toLocaleString('en-US')} fits</small></button>`).join('')}` : ''
  this.q<HTMLElement>('[data-fit-search-role-tip]').hidden = Boolean(query);this.q<HTMLElement>('[data-fit-search-no-matches]').hidden = !query || visible.length > 0 || items.length > 0 || this.itemPending || this.availabilityPending;this.q<HTMLElement>('[data-fit-search-item-pending]').hidden = !this.itemPending;this.q<HTMLElement>('[data-fit-search-availability-pending]').hidden = !this.availabilityPending
  if(this.alive && focusAttribute && focusValue && this.q('#fit-search-requirement-picker').matches(':popover-open')) {const next = this.element.querySelector<HTMLButtonElement>(`[${focusAttribute}="${CSS.escape(focusValue)}"]`);if(next && !next.disabled) next.focus({preventScroll:true});else this.q<HTMLInputElement>('[data-fit-search-role-query]').focus({preventScroll:true})}
 }
 private positionMenu(menu:HTMLElement) {const trigger = this.q<HTMLElement>(`[popovertarget="${CSS.escape(menu.id)}"]`);positionViewportPanel(menu,trigger.getBoundingClientRect())}
 toggle(event:Event) {const menu = eventElement(event.target);if(!(menu instanceof HTMLElement) || !menu.hasAttribute('popover')) return;if('newState' in event && event.newState === 'open') {this.positionMenu(menu);if(menu.id === 'fit-search-requirement-picker') this.q<HTMLInputElement>('[data-fit-search-role-query]').focus({preventScroll:true})}else if(menu.id === 'fit-search-requirement-picker') {this.cancelItems();this.renderRoles()}}
 positionMenus() {if(this.alive) for(const menu of this.all<HTMLElement>('[popover]')) if(menu.matches(':popover-open')) this.positionMenu(menu)}
 private closeMenus() {for(const menu of this.all<HTMLElement>('[popover]')) if(menu.matches(':popover-open')) menu.hidePopover()}
 private notice(message:string,action:string) {const notice = node('p','fit-search-error',message),button = node('button','','Retry');notice.setAttribute('role','alert');button.type = 'button';button.dataset.fitSearchRetry = action;notice.append(button);return notice}
 private cancel(part:Part) {this.reads.get(part)?.abort();this.reads.delete(part)}
 private begin(part:Part,intent = false) {this.cancel(part);const request = new AbortController();this.reads.set(part,request);this.intents.set(part,intent);return request}
 private current(part:Part,request:AbortController,generation = this.queryGeneration) {return this.alive && this.reads.get(part) === request && generation === this.queryGeneration}
 private async fragment(part:Exclude<Part,'availability'|'roles'>,query:URLSearchParams,signal:AbortSignal) {const html = await this.text(`/_fragments/fits/search/${part}?${query}`,`Could not load ${part}`,{signal,headers:{Accept:'text/html'}}),template = document.createElement('template');template.innerHTML = html;return template.content}
 private async loadResults(append = false) {
  if(!this.alive || !this.state.ship) return
  const request = this.begin('results',append),generation = this.queryGeneration,offset = append ? this.offset+24 : 0,container = this.q('[data-fit-search-results]'),query = searchParams(this.state,{sort:true});query.set('offset',String(offset))
  container.querySelector('.fit-search-error')?.remove();container.setAttribute('aria-busy','true');if(!container.querySelector('.fit-search-updating')) {const updating = node('p','fit-search-updating');updating.setAttribute('role','status');updating.innerHTML = this.icon('loader-circle')+'Updating fits…';container.prepend(updating)}
  try {
   const content = await this.fragment('results',query,request.signal);if(!this.current('results',request,generation)) return
   const meta = requiredElement(content,'#fit-search-result-state'),value:unknown = JSON.parse(meta.textContent ?? 'null'),data = readFitResultState(value,offset);meta.remove()
   const rows = requiredElement(content,'[data-fit-search-rows]'),more = requiredElement(content,'[data-fit-search-more-wrap]'),total = requiredElement(content,'[data-fit-search-total]')
   if(append) {const existing = this.q('[data-fit-search-results] [data-fit-search-rows]');existing.append(...rows.children);this.q('[data-fit-search-results] [data-fit-search-more-wrap]').replaceWith(more);this.q('[data-fit-search-results] [data-fit-search-total]').replaceWith(total)}else container.replaceChildren(content)
   this.offset = offset;this.more = data.more;this.intents.delete('results');container.querySelector('[data-fit-search-rows]')?.removeAttribute('data-failed');if(data.shipName && this.state.ship && this.state.ship.name !== data.shipName) {this.state.ship.name = data.shipName;this.renderHull();this.syncURL()}
  }catch(error) {if(!isAbortError(error) && this.current('results',request,generation)) {const rows = container.querySelector('[data-fit-search-rows]');if(rows?.children.length) {rows.setAttribute('data-failed','');container.prepend(this.notice('Could not update fits. Previous results are shown.',append ? 'more' : 'results'))}else container.replaceChildren(this.notice('Unable to load fitting results.','results'))}}
  finally {if(this.reads.get('results') === request) {this.reads.delete('results');container.setAttribute('aria-busy','false');container.querySelector('.fit-search-updating')?.remove();const button = container.querySelector<HTMLButtonElement>('[data-fit-search-more]');if(button) button.disabled = false}}
 }
 private async loadProfiles() {
  if(!this.alive || !this.state.ship) return
  const request = this.begin('distributions'),generation = this.queryGeneration,container = this.q('[data-fit-search-profiles]');container.hidden = false;container.setAttribute('aria-busy','true')
  try {const content = await this.fragment('distributions',searchParams(this.state),request.signal);if(!this.current('distributions',request,generation)) return;container.replaceChildren(content);container.hidden = !container.children.length;this.intents.delete('distributions')}
  catch(error) {if(!isAbortError(error) && this.current('distributions',request,generation)) container.replaceChildren(this.notice('Unable to load observed fit profiles.','distributions'))}
  finally {if(this.reads.get('distributions') === request) {this.reads.delete('distributions');container.setAttribute('aria-busy','false')}}
 }
 private async loadAvailability() {
  const ship = this.state.ship;if(!this.alive || !ship) return
  const request = this.begin('availability'),generation = this.queryGeneration;this.availabilityPending = true;this.renderRoles()
  try {const data = readFitAvailability(await this.request<unknown>('/api/fits/search/availability?'+searchParams(this.state),{signal:request.signal}),ship.id);if(this.current('availability',request,generation)) {this.availability = data;this.intents.delete('availability')}}
  catch(error) {if(!isAbortError(error) && this.current('availability',request,generation)) this.availability = null}
  finally {if(this.reads.get('availability') === request) {this.reads.delete('availability');this.availabilityPending = false;if(this.alive) this.renderRoles()}}
 }
 private refresh({all = true}:{all?:boolean} = {}) {
  this.renderHull();this.renderFilters();this.syncURL();this.offset = 0;this.more = false
  if(all) {this.queryGeneration++;for(const part of ['results','distributions','availability'] as const) {this.cancel(part);this.intents.delete(part)}}
  if(!this.state.ship) {this.availability = null;this.availabilityPending = false;this.renderRoles();return}
  void this.loadResults();if(all) {void this.loadProfiles();void this.loadAvailability()}
 }
 private selectShip(ship:FitSearchState['ship']) {this.closeMenus();closeSearchPickers(this.element);this.state.ship = ship;this.availability = null;this.refresh()}
 pick(event:CustomEvent<unknown>) {const target = eventElement(event.target);if(!this.alive || !(target instanceof HTMLElement) || target.dataset.pickerName !== 'fit-ship' || !isRecord(event.detail) || !searchID(event.detail.id)) return;this.selectShip({id:searchID(event.detail.id),name:typeof event.detail.name === 'string' ? event.detail.name : `Type ${searchID(event.detail.id)}`})}
 private addRequirement(filter:FitRequirement) {if(this.state.filters.length >= 8 || this.picked(filter)) return;this.state.filters.push(filter);this.q<HTMLElement>('#fit-search-requirement-picker').hidePopover();this.q<HTMLInputElement>('[data-fit-search-role-query]').value = '';this.items = [];this.refresh()}
 async click(event:MouseEvent) {
  const button = eventElement(event.target)?.closest<HTMLButtonElement>('button');if(!this.alive || !button || button.disabled) return;const data = button.dataset
  if(data.fitSearchPickShip && searchID(data.fitSearchPickShip)) this.selectShip({id:searchID(data.fitSearchPickShip),name:data.fitSearchShipName ?? `Type ${data.fitSearchPickShip}`})
  else if(button.hasAttribute('data-fit-search-clear-ship')) {this.closeMenus();this.state.ship = null;this.state.filters = [];this.availability = null;this.refresh()}
  else if(data.fitSearchChoice) {button.closest<HTMLElement>('[popover]')?.hidePopover();const value = data.fitSearchValue;if(data.fitSearchChoice === 'sort' && isFitSort(value)) {this.state.sort = value;this.refresh({all:false})}else if(data.fitSearchChoice === 'metric' && isFitMetric(value)) {this.draftMetric = value;this.state.sort = sortForMetric(value);if(value === 'npc_ehp') setStatistic(this.state,value,null,null,this.draftNPC);this.refresh()}else if(data.fitSearchChoice === 'npc' && typeof value === 'string') {this.draftNPC = value;this.state.sort = 'npc_ehp';const filter = this.state.statistics.find(filter=>filter.metric === 'npc_ehp');setStatistic(this.state,'npc_ehp',filter?.min ?? null,filter?.max ?? null,value);this.refresh()}}
  else if(button.hasAttribute('data-fit-search-clear-stats')) {this.state.statistics = [];this.state.capStable = false;this.refresh()}
  else if(data.fitSearchRemoveStat !== undefined) {const index = Number(data.fitSearchRemoveStat);if(Number.isSafeInteger(index) && index >= 0) {this.state.statistics.splice(index,1);this.refresh()}}
  else if(data.fitSearchRole) {if(this.roles.some(role=>role.id === data.fitSearchRole)) this.addRequirement({kind:'role',role_id:data.fitSearchRole,op:'>=',count:1})}
  else if(data.fitSearchType) {const hit = this.items.find(hit=>hit.typeID === searchID(data.fitSearchType));if(hit) this.addRequirement({kind:'type',type_id:hit.typeID,type_name:hit.name,op:'>=',count:1})}
  else if(data.fitSearchOp || data.fitSearchCountStep || button.hasAttribute('data-fit-search-remove-requirement')) {const index = Number(button.closest<HTMLElement>('[data-fit-search-requirement]')?.dataset.fitSearchRequirement),filter = this.state.filters[index];if(!filter) return;if(isFitComparison(data.fitSearchOp)) filter.op = data.fitSearchOp;else if(data.fitSearchCountStep && [-1,1].includes(Number(data.fitSearchCountStep))) filter.count = Math.max(0,Math.min(16,filter.count+Number(data.fitSearchCountStep)));else if(button.hasAttribute('data-fit-search-remove-requirement')) this.state.filters.splice(index,1);this.refresh()}
  else if(isFitMetric(data.fitSearchBucket)) {const min = Number(data.fitSearchBucketMin),max = Number(data.fitSearchBucketMax);if(Number.isFinite(min) && Number.isFinite(max) && min >= 0 && max >= 0) {setStatistic(this.state,data.fitSearchBucket,min,max);this.state.sort = sortForMetric(data.fitSearchBucket);this.refresh()}}
  else if(button.hasAttribute('data-fit-search-more')) {if(!this.more || this.reads.has('results')) return;button.disabled = true;void this.loadResults(true)}
  else if(data.fitSearchRetry) {if(data.fitSearchRetry === 'results' || data.fitSearchRetry === 'more') void this.loadResults(data.fitSearchRetry === 'more');else void this.loadProfiles()}
  else if(button.hasAttribute('data-fit-search-retry-roles')) {const request = this.begin('roles');try {const response = await this.request<unknown>('/api/fits/roles',{signal:request.signal});if(!this.current('roles',request) || !isRecord(response)) return;this.roles = readFitRoles(response.roles);this.intents.delete('roles');this.q<HTMLElement>('[data-fit-search-role-error]').hidden = true;this.renderFilters()}catch{/* The source role retry remains available. */}finally {if(this.reads.get('roles') === request) this.reads.delete('roles')}}
  else if(button.hasAttribute('data-fit-search-retry-popular')) {const request = this.begin('popular');button.disabled = true;try {const content = await this.fragment('popular',new URLSearchParams(),request.signal);if(this.current('popular',request)) {this.q('[data-fit-search-popular]').replaceChildren(content);this.intents.delete('popular')}}catch{if(this.current('popular',request)) button.disabled = false}finally {if(this.reads.get('popular') === request) this.reads.delete('popular')}}
 }
 change(event:Event) {const input = eventElement(event.target);if(this.alive && input instanceof HTMLInputElement && input.hasAttribute('data-fit-search-cap-stable')) {this.state.capStable = input.checked;this.refresh()}}
 submit(event:SubmitEvent) {if(!this.alive || !(event.target instanceof HTMLFormElement) || !event.target.hasAttribute('data-fit-search-stat-form')) return;event.preventDefault();const form = event.target,min = parseStatisticValue(formInput(form,'minimum').value),max = parseStatisticValue(formInput(form,'maximum').value);this.state.sort = sortForMetric(this.draftMetric);if(min === null && max === null && this.draftMetric !== 'npc_ehp') {this.refresh();return}setStatistic(this.state,this.draftMetric,min,max,this.draftMetric === 'npc_ehp' ? this.draftNPC : undefined);formInput(form,'minimum').value = '';formInput(form,'maximum').value = '';this.refresh()}
 private cancelItems() {this.stopTimer(this.itemTimer);this.itemTimer = undefined;this.itemRequest?.abort();this.itemRequest = undefined;this.itemPending = false}
 input(event:Event) {
  const input = eventElement(event.target);if(!this.alive || !(input instanceof HTMLInputElement) || !input.hasAttribute('data-fit-search-role-query')) return
  this.cancelItems();this.items = [];const query = input.value.trim();this.renderRoles();if(query.length < 2) return
  const request = new AbortController();this.itemRequest = request;this.itemTimer = this.after(200,()=>{this.itemTimer = undefined;this.itemPending = true;this.renderRoles();void this.searchItems(query,input,request)})
 }
 private async searchItems(query:string,input:HTMLInputElement,request:AbortController) {
  try {const data = readSearchResults(await this.request<unknown>('/api/search?'+new URLSearchParams({q:query,type:'item'}),{signal:request.signal}));if(!this.alive || this.itemRequest !== request || input.value.trim() !== query) return;this.items = data.hits.filter(hit=>hit.type === 'item').slice(0,12).map(hit=>({typeID:hit.id,name:hit.name ?? `Type ${hit.id}`}))}
  catch(error) {if(!isAbortError(error) && this.alive && this.itemRequest === request) this.items = []}
  finally {if(this.itemRequest === request) {this.itemRequest = undefined;this.itemPending = false;if(this.alive) this.renderRoles()}}
 }
 keyboard(event:KeyboardEvent) {const menu = eventElement(event.target)?.closest('[popover]');if(!this.alive || !menu || !['ArrowDown','ArrowUp'].includes(event.key)) return;const choices = [...menu.querySelectorAll<HTMLButtonElement>('button:not([disabled])')].filter(button=>button.getBoundingClientRect().height > 0);if(!choices.length) return;event.preventDefault();const current = document.activeElement instanceof HTMLButtonElement ? choices.indexOf(document.activeElement) : -1;choices[(current+(event.key === 'ArrowDown' ? 1 : -1)+choices.length)%choices.length].focus()}
 back() {if(!this.alive) return;this.closeMenus();closeSearchPickers(this.element);this.state = stateFromURL(new URL(location.href).searchParams);this.drafts();this.availability = null;this.refresh()}
 hide() {this.alive = false;this.queryGeneration++;this.cancelItems();this.closeMenus();closeSearchPickers(this.element);for(const part of this.reads.keys()) this.cancel(part);this.availabilityPending = false;for(const el of this.all<HTMLElement>('[data-fit-search-results],[data-fit-search-profiles]')) {el.setAttribute('aria-busy','false');el.querySelector('.fit-search-updating')?.remove()}for(const button of this.all<HTMLButtonElement>('[data-fit-search-more],[data-fit-search-retry-popular]')) button.disabled = false;super.hide()}
 show(event:PageTransitionEvent) {if(!event.persisted) return;this.alive = true;this.renderHull();this.renderFilters();for(const [part,append] of this.intents) {if(part === 'results') void this.loadResults(append);else if(part === 'distributions') void this.loadProfiles();else if(part === 'availability') void this.loadAvailability();else {const button = this.element.querySelector<HTMLButtonElement>(part === 'roles' ? '[data-fit-search-retry-roles]' : '[data-fit-search-retry-popular]');button?.click()}}}
 disconnect() {this.hide();super.disconnect();this.requirementRows.clear();this.statisticRows.clear()}
}
registerController('fits-search',FitsSearchController)
