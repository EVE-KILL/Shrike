import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import './controllers/coalition-activity'
import {readBrowserSession,BrowserRequestError} from './browser-request'
import {eventElement,requiredElement,formInput} from './shared/dom'
import {isRecord,isAbortError,errorMessage} from './shared/json'
import {readSearchResults,searchID,type SearchEntity} from './shared/search-model'
import {replaceFragment} from './fragment'
import type {CoalitionCreateData,CoalitionUpdateData,CoalitionCreateResponses,CoalitionUpdateResponses} from '../../shared/api/types.gen'

type Draft = CoalitionCreateData['body']
type Saved = Pick<CoalitionCreateResponses[201]['coalition'] & CoalitionUpdateResponses[200]['coalition'],'slug'|'revision'>
type Baseline = {name:string;description:string;source:string;alliances:Node[]}
function readSaved(value:unknown):Saved {
  const coalition = isRecord(value) ? value.coalition : null
  if(!isRecord(coalition) || typeof coalition.slug !== 'string' || !coalition.slug || typeof coalition.revision !== 'number' || !Number.isSafeInteger(coalition.revision) || coalition.revision < 1) throw new Error('Invalid coalition response')
  return {slug:coalition.slug,revision:coalition.revision}
}

export class CoalitionsController extends RequestComponent {
  private alive = false
  private authenticated = false
  private authEpoch = 0
  private baselines = new WeakMap<HTMLFormElement,Baseline>()
  private pickerTimer:ReturnType<typeof setTimeout>|undefined
  private pickerRead:AbortController|undefined
  private detailRead:AbortController|undefined
  private write:AbortController|undefined
  private loginRead:AbortController|undefined
  private reconcile = false
  connect() {this.alive = true;this.initializeForms();this.filter();void this.auth()}
  private forms() {return this.all<HTMLFormElement>('[data-coalition-form]')}
  private selected(form:HTMLFormElement) {return requiredElement<HTMLElement>(form,'[data-coalition-selected]')}
  private ids(form:HTMLFormElement) {return [...this.selected(form).querySelectorAll<HTMLElement>('[data-alliance-id]')].map(row=>searchID(row.dataset.allianceId)).filter(Boolean)}
  private description(form:HTMLFormElement) {return requiredElement<HTMLTextAreaElement>(form,'textarea[name="description"]')}
  private draft(form:HTMLFormElement):Draft {return {name:formInput(form,'name').value.trim(),description:this.description(form).value,source_url:formInput(form,'source_url').value.trim() || null,alliance_ids:this.ids(form)}}
  private snapshot(form:HTMLFormElement) {return JSON.stringify(this.draft(form))}
  private baseline(form:HTMLFormElement):Baseline {return {name:formInput(form,'name').value,description:this.description(form).value,source:formInput(form,'source_url').value,alliances:[...this.selected(form).childNodes].map(node=>node.cloneNode(true))}}
  private initializeForms() {for(const form of this.forms()) {if(!this.baselines.has(form)) this.baselines.set(form,this.baseline(form));this.syncForm(form)}}
  private syncForm(form:HTMLFormElement) {const count = this.ids(form).length,label = form.querySelector('[data-selected-count]');if(label) label.textContent = `${count} selected`;requiredElement<HTMLButtonElement>(form,'[type="submit"]').disabled = Boolean(this.write) || this.reconcile || formInput(form,'name').value.trim().length < 2 || !count}
  private syncAuth() {for(const node of this.all<HTMLElement>('[data-coalition-login]')) node.hidden = this.authenticated;for(const node of this.all<HTMLElement>('[data-coalition-create-toggle],[data-coalition-edit-toggle]')) node.hidden = !this.authenticated;if(!this.authenticated) for(const form of this.forms()) {form.hidden = true;this.syncToggle(form)}}
  private async auth() {const epoch = ++this.authEpoch;try{const user = await readBrowserSession();if(this.alive && epoch === this.authEpoch) {this.authenticated = Boolean(user);this.syncAuth()}}catch{if(this.alive && epoch === this.authEpoch) {this.authenticated = false;this.syncAuth()}}}
  private syncToggle(form:HTMLFormElement) {const create = form.dataset.coalitionForm === 'create',toggle = this.element.querySelector<HTMLElement>(create ? '[data-coalition-create-toggle]' : '[data-coalition-edit-toggle]');if(!toggle) return;requiredElement<HTMLElement>(toggle,'[data-icon-open]').hidden = !form.hidden;requiredElement<HTMLElement>(toggle,'[data-icon-close]').hidden = form.hidden;requiredElement(toggle,'[data-coalition-toggle-label]').textContent = form.hidden ? (create ? 'Add coalition' : 'Edit coalition') : (create ? 'Close creator' : 'Close editor')}
  private toggle(form:HTMLFormElement) {
    this.cancelPicker();this.detailRead?.abort();this.detailRead = undefined
    if(form.hidden && form.dataset.coalitionForm === 'edit') {const baseline = this.baselines.get(form);if(baseline) {formInput(form,'name').value = baseline.name;this.description(form).value = baseline.description;formInput(form,'source_url').value = baseline.source;this.selected(form).replaceChildren(...baseline.alliances.map(node=>node.cloneNode(true)));this.syncForm(form)}}
    form.hidden = !form.hidden;this.syncToggle(form)
  }
  private feedback(form:HTMLFormElement,message:string,conflict=false,href?:string) {
    const region = requiredElement<HTMLElement>(form,'[data-coalition-error]');region.replaceChildren(document.createTextNode(message));region.hidden = false
    if(conflict) {const button = document.createElement('button');button.type = 'button';button.textContent = 'Reload now';button.dataset.coalitionReload = '';region.append(' ',button)}
    if(href) {const link = document.createElement('a');link.href = href;link.textContent = 'Open coalition';region.append(' ',link)}
  }
  filter() {
    const input = this.element.querySelector<HTMLInputElement>('[data-coalition-filter]');if(!input) return
    const needle = input.value.trim().toLowerCase();let visible = 0
    for(const card of this.all<HTMLElement>('.coalition-card')) {card.hidden = !(card.dataset.searchable ?? '').includes(needle);if(!card.hidden) visible++}
    const empty = this.element.querySelector<HTMLElement>('[data-coalition-empty]');if(empty) {empty.hidden = visible !== 0;empty.textContent = needle ? `No coalitions match “${input.value.trim()}”.` : 'No coalitions have been added yet.'}
  }
  private cancelPicker() {this.stopTimer(this.pickerTimer);this.pickerTimer = undefined;this.pickerRead?.abort();this.pickerRead = undefined;for(const list of this.all<HTMLElement>('[data-coalition-picker-results]')) list.hidden = true}
  input(event:InputEvent) {
    const input = eventElement(event.target);if(!(input instanceof HTMLInputElement)) return
    if(input.matches('[data-coalition-filter]')) {this.filter();return}
    const form = input.closest<HTMLFormElement>('[data-coalition-form]');if(!form) return
    if(input.name === 'name') this.syncForm(form)
    if(!input.matches('[data-coalition-picker]')) return
    this.cancelPicker();const list = requiredElement<HTMLElement>(form,'[data-coalition-picker-results]');list.replaceChildren();const query = input.value.trim();if(query.length < 2) return
    this.pickerTimer = this.after(300,()=>{this.pickerTimer = undefined;void this.search(form,input,query)})
  }
  private async search(form:HTMLFormElement,input:HTMLInputElement,query:string) {
    if(!this.alive || form.hidden) return
    const request = new AbortController();this.pickerRead = request
    try {
      const url = new URL('/api/search',location.href);url.search = new URLSearchParams({q:query,type:'alliance',limit:'10'}).toString()
      const results = readSearchResults(await this.request<unknown>(url.pathname+url.search,{signal:request.signal},'Could not search alliances'))
      if(!this.alive || this.pickerRead !== request || input.value.trim() !== query || form.hidden) return
      const selected = new Set(this.ids(form)),list = requiredElement<HTMLElement>(form,'[data-coalition-picker-results]');list.replaceChildren()
      for(const hit of results.hits) if(hit.type === 'alliance') list.append(this.option(hit,selected.has(hit.id)))
      list.hidden = !list.childElementCount
    }catch(error) {if(!isAbortError(error) && this.alive && this.pickerRead === request) this.feedback(form,errorMessage(error,'Could not search alliances'))}
    finally {if(this.pickerRead === request) this.pickerRead = undefined}
  }
  private option(hit:SearchEntity,disabled:boolean) {
    const option = document.createElement('button');option.type = 'button';option.dataset.pickAlliance = String(hit.id);option.dataset.name = hit.name ?? '';option.dataset.ticker = hit.ticker ?? '';option.disabled = disabled
    const image = document.createElement('img');image.src = `https://images.evetech.net/alliances/${hit.id}/logo?size=64`;image.alt = hit.name ?? ''
    const text = document.createElement('span'),name = document.createElement('strong'),sub = document.createElement('small');name.textContent = hit.name ?? '';sub.textContent = `alliance${hit.ticker ? ` [${hit.ticker}]` : ''}`;text.append(name,sub);option.append(image,text);return option
  }
  private pick(form:HTMLFormElement,option:HTMLButtonElement) {
    const id = searchID(option.dataset.pickAlliance);if(!id || option.disabled || this.ids(form).includes(id)) return
    const name = option.dataset.name ?? '',chip = document.createElement('div');chip.className = 'coalition-selected-alliance';chip.dataset.allianceId = String(id)
    const image = document.createElement('img');image.src = `https://images.evetech.net/alliances/${id}/logo?size=64`;image.alt = name;image.width = image.height = 32
    const text = document.createElement('span'),label = document.createElement('strong'),ticker = document.createElement('small'),remove = document.createElement('button');label.textContent = name;ticker.textContent = `[${option.dataset.ticker ?? ''}] · ${id}`;text.append(label,ticker);remove.type = 'button';remove.dataset.removeAlliance = '';remove.setAttribute('aria-label',`Remove ${name}`);remove.textContent = '×';chip.append(image,text,remove);this.selected(form).append(chip)
    const input = requiredElement<HTMLInputElement>(form,'[data-coalition-picker]');input.value = '';input.focus();this.cancelPicker();requiredElement(form,'[data-coalition-picker-results]').replaceChildren();this.syncForm(form)
  }
  openURL(url:string) {location.assign(url)}
  private async login(button:HTMLButtonElement) {
    if(this.loginRead) return
    const request = new AbortController();this.loginRead = request;button.disabled = true
    try {const query = new URLSearchParams({redirect:location.pathname+location.search,charKm:'1',corpKm:'1',delay:'0'}),response = await this.request<unknown>(`/auth/login?${query}`,{signal:request.signal},'Login is unavailable');if(!isRecord(response) || typeof response.url !== 'string') throw new Error('Invalid login destination');const url = new URL(response.url);if(url.protocol !== 'https:') throw new Error('Invalid login destination');if(this.alive && this.loginRead === request) this.openURL(url.href)}
    catch(error) {if(this.alive && !isAbortError(error)) button.textContent = errorMessage(error,'Login is unavailable')}
    finally {button.disabled = false;if(this.loginRead === request) this.loginRead = undefined}
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    const login = target.closest<HTMLButtonElement>('[data-coalition-login]');if(login) {void this.login(login);return}
    const toggle = target.closest('[data-coalition-create-toggle],[data-coalition-edit-toggle]');if(toggle && this.authenticated) {this.toggle(this.q<HTMLFormElement>(toggle.matches('[data-coalition-create-toggle]') ? '[data-coalition-form="create"]' : '[data-coalition-form="edit"]'));return}
    const form = target.closest<HTMLFormElement>('[data-coalition-form]'),remove = target.closest('[data-remove-alliance]');if(remove && form) {remove.closest('[data-alliance-id]')?.remove();this.syncForm(form);return}
    const pick = target.closest<HTMLButtonElement>('[data-pick-alliance]');if(pick && form) {this.pick(form,pick);return}
    if(target.closest('[data-coalition-reload]') && form) {void this.refresh(form).catch(error=>{if(this.alive && !isAbortError(error)) this.feedback(form,errorMessage(error),true)});return}
    if(!target.closest('.coalition-picker')) this.cancelPicker()
  }
  private async refresh(form:HTMLFormElement,preserve?:()=>boolean) {
    const current = this.element.querySelector<HTMLElement>('[data-coalition-detail-content]');if(!current) return
    const slug = current.dataset.coalitionSlug;if(!slug) throw new Error('Missing coalition identity')
    this.detailRead?.abort();const request = new AbortController();this.detailRead = request
    try {
      const html = await this.text(`/_fragments/coalitions/${encodeURIComponent(slug)}`,'Could not reload the coalition',{signal:request.signal,headers:{Accept:'text/html'}})
      if(!this.alive || this.detailRead !== request || !current.isConnected) return
      const replacement = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('[data-coalition-detail-content]'),revision = Number(replacement?.dataset.coalitionRevision)
      if(!replacement || replacement.dataset.coalitionSlug !== slug || !Number.isSafeInteger(revision) || revision < Number(current.dataset.coalitionRevision)) throw new Error('Could not read the updated coalition')
      const retaining = Boolean(preserve?.()),freshForm = requiredElement<HTMLFormElement>(replacement,'[data-coalition-form="edit"]')
      this.baselines.set(retaining ? form : freshForm,this.baseline(freshForm))
      replaceFragment(current,replacement,()=>{if(retaining) freshForm.replaceWith(form)})
      this.reconcile = false;this.initializeForms();this.syncAuth();for(const next of this.forms()) this.syncToggle(next)
      if(retaining) this.feedback(form,'Coalition saved or reloaded. Your newer editor changes are retained.')
    }finally {if(this.detailRead === request) this.detailRead = undefined}
  }
  submit(event:SubmitEvent) {
    const form = eventElement(event.target);if(!(form instanceof HTMLFormElement) || !form.matches('[data-coalition-form]')) return
    event.preventDefault();if(!this.alive || !this.authenticated || this.write || this.reconcile) return
    this.syncForm(form);if(requiredElement<HTMLButtonElement>(form,'[type="submit"]').disabled) return
    void this.save(form)
  }
  private async save(form:HTMLFormElement) {
    const request = new AbortController(),create = form.dataset.coalitionForm === 'create',snapshot = this.snapshot(form),body = this.draft(form),detail = this.element.querySelector<HTMLElement>('[data-coalition-detail-content]')
    const update:CoalitionUpdateData['body'] = {...body,revision:Number(detail?.dataset.coalitionRevision)}
    this.write = request;this.reconcile = !create;this.syncForm(form);requiredElement<HTMLElement>(form,'[data-coalition-error]').hidden = true
    let acknowledged = false
    try {
      const path = create ? '/api/coalitions' : '/api/coalitions/'+encodeURIComponent(detail?.dataset.coalitionSlug ?? ''),result = readSaved(await this.request<unknown>(path,{method:create ? 'POST' : 'PATCH',body:create ? body : update,signal:request.signal},`Unable to ${create ? 'create' : 'save'} the coalition.`))
      if(!this.alive || this.write !== request) return
      if(!create && (!detail || result.slug !== detail.dataset.coalitionSlug || result.revision < Number(detail.dataset.coalitionRevision))) throw new Error('Invalid coalition identity')
      acknowledged = true
      if(create) {const href = '/coalitions/'+encodeURIComponent(result.slug);if(this.snapshot(form) === snapshot) this.openURL(href);else this.feedback(form,'Coalition created. Your newer draft is retained.',false,href)}
      else if(detail) {detail.dataset.coalitionRevision = String(result.revision);await this.refresh(form,()=>this.snapshot(form) !== snapshot)}
    }catch(error) {if(!acknowledged && error instanceof BrowserRequestError && error.status < 500) this.reconcile = false;if(this.alive && !isAbortError(error)) this.feedback(form,errorMessage(error),!create && (this.reconcile || /reload|revision|conflict/i.test(errorMessage(error))))}
    finally {if(this.write === request) {this.write = undefined;if(form.isConnected) this.syncForm(form);for(const next of this.forms()) this.syncForm(next)}}
  }
  hide() {this.alive = false;this.authEpoch++;this.cancelPicker();this.detailRead?.abort();this.detailRead = undefined;this.loginRead?.abort();this.loginRead = undefined;this.write?.abort();this.write = undefined;super.hide();for(const form of this.forms()) this.syncForm(form)}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;void this.auth();const form = this.element.querySelector<HTMLFormElement>('[data-coalition-form="edit"]');if(this.reconcile && form) void this.refresh(form,()=>true).catch(error=>{if(this.alive && !isAbortError(error)) this.feedback(form,errorMessage(error),true)})}this.filter()}
  visibility() {if(document.hidden) this.cancelPicker()}
}
registerController('coalitions',CoalitionsController)
