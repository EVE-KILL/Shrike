import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement,formInput} from './shared/dom'
import {isAbortError} from './shared/json'
import {replaceFragmentChildren} from './fragment'
import {isLegacySort,legacyPresetShips,legacyAutocompleteFields,legacyYear,readLegacySuggestions,type LegacySort} from './shared/legacy'

export class LegacyController extends RequestComponent {
  private alive = false
  private sort:LegacySort = 'id_desc'
  private limit = 50
  private page = 1
  private year = 0
  private filters = new URLSearchParams()
  private cursors = new Map<number,string>()
  private version = 0
  private renderedVersion = 0
  private killRead:AbortController|undefined
  private topRead:AbortController|undefined
  private killDirty = false
  private topDirty = false
  private suggestionRead:AbortController|undefined
  private suggestionInput:HTMLInputElement|undefined
  private suggestionTimer:ReturnType<typeof setTimeout>|undefined
  private suggestionGeneration = 0
  private get form() {return this.q<HTMLFormElement>('[data-legacy-form]')}
  private get kills() {return this.q<HTMLElement>('[data-legacy-kills]')}
  private get sidebar() {return this.q<HTMLElement>('[data-legacy-sidebar]')}
  connect() {this.alive = true;this.refreshSort();this.mobile()}
  private current(request:AbortController,owner:AbortController|undefined) {return this.alive && this.element.isConnected && owner === request && !request.signal.aborted}
  private refreshSort() {for(const button of this.all<HTMLButtonElement>('[data-legacy-sort]')) {const active = button.dataset.legacySort === this.sort;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}}
  private resetPage() {this.page = 1;this.cursors.clear();this.version++}
  submit(event:SubmitEvent) {event.preventDefault();this.applyFilters()}
  private applyFilters() {
    const filters = new URLSearchParams()
    for(const field of this.all<HTMLInputElement>('[data-legacy-field]')) if(field.value.trim()) filters.set(field.name,field.value.trim())
    this.filters = filters;this.resetPage();this.q<HTMLElement>('[data-legacy-clear]').hidden = filters.size === 0
    this.clearSuggestions();void this.reloadKills()
  }
  private pending(container:HTMLElement,value:boolean) {container.classList.toggle('legacy-pending',value);if(value) container.setAttribute('aria-busy','true');else container.removeAttribute('aria-busy')}
  private loadError(container:HTMLElement,part:'kills'|'top',message:string) {
    container.querySelector('[data-legacy-request-error]')?.remove()
    const box = document.createElement('p'),retry = document.createElement('button')
    box.className = 'legacy-error';box.dataset.legacyRequestError = part;box.setAttribute('role','alert');box.textContent = message+' '
    retry.type = 'button';retry.dataset.legacyRetry = part;retry.textContent = 'Retry';box.append(retry);container.prepend(box)
  }
  private async reloadKills() {
    if(!this.alive) return
    this.killRead?.abort();const request = new AbortController();this.killRead = request;this.killDirty = true
    const query = new URLSearchParams(this.filters),version = this.version
    query.set('limit',String(this.limit));query.set('page',String(this.page))
    if(this.sort !== 'id_desc') query.set('sort',this.sort)
    const cursor = this.cursors.get(this.page);if(this.page > 1 && cursor) query.set('after',cursor)
    this.kills.querySelector('[data-legacy-request-error]')?.remove();this.pending(this.kills,true)
    try {
      const html = await this.text('/_fragments/legacy/kills?'+query,'Could not load historical killmails.',{signal:request.signal,headers:{Accept:'text/html'}})
      if(!this.current(request,this.killRead)) return
      const doc = new DOMParser().parseFromString(html,'text/html')
      if(!doc.querySelector('#kill-list')) throw new Error('Could not read historical killmails.')
      replaceFragmentChildren(this.kills,...doc.body.childNodes);this.renderedVersion = version;this.killDirty = false
    } catch(error) {if(this.current(request,this.killRead) && !isAbortError(error)) {this.killDirty = false;this.loadError(this.kills,'kills','Could not load historical killmails.')}}
    finally {if(this.killRead === request) {this.killRead = undefined;this.pending(this.kills,false)}}
  }
  private async reloadTop() {
    if(!this.alive) return
    this.topRead?.abort();const request = new AbortController();this.topRead = request;this.topDirty = true
    this.sidebar.querySelector('[data-legacy-request-error]')?.remove();this.pending(this.sidebar,true)
    try {
      const html = await this.text('/_fragments/legacy/top'+(this.year ? '?year='+this.year : ''),'Could not load historical top lists.',{signal:request.signal,headers:{Accept:'text/html'}})
      if(!this.current(request,this.topRead)) return
      const doc = new DOMParser().parseFromString(html,'text/html')
      if(!doc.querySelector('[data-legacy-year]')) throw new Error('Could not read historical top lists.')
      replaceFragmentChildren(this.sidebar,...doc.body.childNodes);this.topDirty = false
    } catch(error) {if(this.current(request,this.topRead) && !isAbortError(error)) {this.topDirty = false;this.loadError(this.sidebar,'top','Could not load historical top lists.')}}
    finally {if(this.topRead === request) {this.topRead = undefined;this.pending(this.sidebar,false)}}
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target)
    if(!target || !this.alive) return
    const suggestion = target.closest<HTMLButtonElement>('[data-legacy-suggestion]')
    if(suggestion) {this.choose(suggestion);return}
    const clear = target.closest('[data-legacy-clear]')
    if(clear) {this.form.reset();this.applyFilters();return}
    const retry = target.closest<HTMLElement>('[data-legacy-retry]')
    if(retry) {if(retry.dataset.legacyRetry === 'kills') {if(!this.killRead) void this.reloadKills()} else if(retry.dataset.legacyRetry === 'top' && !this.topRead) void this.reloadTop();return}
    const preset = target.closest<HTMLElement>('[data-legacy-preset]')
    if(preset) {
      const value = preset.dataset.legacyPreset ?? '',ships = legacyPresetShips[value]
      if(ships) formInput(this.form,'ship').value = ships
      else if(/^200[3-7]$/.test(value)) {formInput(this.form,'from').value = value+'-01-01';formInput(this.form,'to').value = value+'-12-31'}
      else formInput(this.form,'system').value = value
      this.applyFilters();return
    }
    const sort = target.closest<HTMLElement>('[data-legacy-sort]')
    if(sort && isLegacySort(sort.dataset.legacySort)) {this.sort = sort.dataset.legacySort;this.resetPage();this.refreshSort();void this.reloadKills();return}
    const limit = target.closest<HTMLElement>('[data-kill-limit]')
    if(limit && this.kills.contains(limit)) {
      const value = Number(limit.dataset.killLimit)
      if(!Number.isSafeInteger(value) || value < 1 || value > 500) return
      this.limit = value;this.resetPage();void this.reloadKills();return
    }
    const pager = target.closest<HTMLButtonElement>('[data-kill-page]')
    if(pager && this.kills.contains(pager) && !pager.disabled) {
      const next = Number(pager.dataset.killPage)
      if(!Number.isSafeInteger(next) || next < 1 || this.renderedVersion !== this.version) return
      if(next > this.page) {const cursor = this.kills.querySelector<HTMLElement>('#kill-list')?.dataset.nextCursor;if(!cursor) return;this.cursors.set(next,cursor)}
      this.page = next;void this.reloadKills();return
    }
    const year = target.closest<HTMLElement>('[data-legacy-year]'),value = legacyYear(year?.dataset.legacyYear)
    if(year && value !== undefined) {this.year = value;void this.reloadTop();return}
    const mobile = target.closest<HTMLElement>('[data-legacy-mobile]')
    if(mobile) {history.pushState({},'',location.pathname+location.search+(mobile.dataset.legacyMobile === 'top' ? '#top' : ''));this.mobile()}
  }
  mobile() {const name = location.hash === '#top' ? 'top' : 'kills';this.element.classList.toggle('legacy-mobile-top',name === 'top');for(const button of this.all<HTMLElement>('[data-legacy-mobile]')) {const active = button.dataset.legacyMobile === name;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}}
  private clearSuggestions() {
    this.suggestionGeneration++;this.stopTimer(this.suggestionTimer);this.suggestionTimer = undefined;this.suggestionRead?.abort();this.suggestionRead = undefined;this.suggestionInput = undefined
    for(const list of this.all<HTMLElement>('[data-legacy-options]')) {list.hidden = true;list.replaceChildren()}
  }
  input(event:InputEvent) {
    const input = eventElement(event.target)
    if(!(input instanceof HTMLInputElement) || !input.matches('[data-legacy-field]') || !legacyAutocompleteFields.has(input.name)) return
    this.clearSuggestions();const query = input.value.trim(),generation = this.suggestionGeneration
    if(query.length < 2) return
    this.suggestionInput = input
    this.suggestionTimer = this.after(200,()=>{this.suggestionTimer = undefined;void this.suggestions(input,query,generation)})
  }
  private async suggestions(input:HTMLInputElement,query:string,generation:number) {
    if(!this.alive || generation !== this.suggestionGeneration) return
    const request = new AbortController();this.suggestionRead = request
    const list = this.q<HTMLElement>(`[data-legacy-options="${input.name}"]`)
    try {
      const data = await this.request<unknown>('/api/legacy/autocomplete?'+new URLSearchParams({q:query,field:input.name}),{signal:request.signal})
      if(!this.current(request,this.suggestionRead) || generation !== this.suggestionGeneration || document.activeElement !== input || input.value.trim() !== query) return
      for(const name of readLegacySuggestions(data)) {const button = document.createElement('button');button.type = 'button';button.dataset.legacySuggestion = name;button.dataset.legacySuggestionField = input.name;button.textContent = name;list.append(button)}
      list.hidden = !list.childElementCount
    } catch(error) {if(this.current(request,this.suggestionRead) && generation === this.suggestionGeneration && document.activeElement === input && !isAbortError(error)) {list.textContent = 'Suggestions are unavailable. You can still search.';list.hidden = false}}
    finally {if(this.suggestionRead === request) this.suggestionRead = undefined}
  }
  private choose(button:HTMLButtonElement) {
    const field = button.dataset.legacySuggestionField
    if(!field || !legacyAutocompleteFields.has(field)) return
    formInput(this.form,field).value = button.dataset.legacySuggestion ?? '';this.clearSuggestions()
  }
  suggestionDown(event:MouseEvent) {const button = eventElement(event.target)?.closest<HTMLButtonElement>('[data-legacy-suggestion]');if(button) {event.preventDefault();this.choose(button)}}
  focusout(event:FocusEvent) {const input = eventElement(event.target);if(input instanceof HTMLInputElement && input.matches('[data-legacy-field]')) {const generation = this.suggestionGeneration;this.after(150,()=>{if(generation === this.suggestionGeneration && this.suggestionInput === input && document.activeElement !== input) this.clearSuggestions()})}}
  hide() {this.alive = false;this.killRead?.abort();this.topRead?.abort();this.killRead = this.topRead = undefined;this.pending(this.kills,false);this.pending(this.sidebar,false);this.clearSuggestions();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.mobile();if(this.killDirty) void this.reloadKills();if(this.topDirty) void this.reloadTop()}}
}
registerController('legacy',LegacyController)
