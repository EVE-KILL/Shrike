import {RequestComponent} from './request-component'
import {registerController} from './application'
import {eventElement,formInput,requiredElement} from '../shared/dom'
import {isAbortError} from '../shared/json'
import {marketID} from '../shared/market'
import {replaceFragment} from '../fragment'
import {isFitMetric,parseFitNumber,sortForFitMetric,fitFilterKey,itemFitsKey} from '../shared/fit-filters'

type Supporting = 'fit-meta'|'fit-distributions'
const selectors = {'fit-meta':'[data-fit-meta]','fit-distributions':'[data-fit-distributions]'} as const
function select(form:HTMLFormElement,name:string):HTMLSelectElement {const field = form.elements.namedItem(name);if(!(field instanceof HTMLSelectElement)) throw new Error('Missing fitting select '+name);return field}
function panelParams(panel:HTMLElement):URLSearchParams {const params = new URLSearchParams(panel.dataset.fitQuery);if(panel.dataset.fitSort && panel.dataset.fitSort !== 'observed') params.set('sort',panel.dataset.fitSort);return params}

export class ItemFitsController extends RequestComponent {
  private alive = false
  private desired:URLSearchParams|undefined
  private read:AbortController|undefined
  private dirty = false
  private supporting = new Map<Supporting,{request:AbortController;target:HTMLElement}>()
  private get id() {const id = marketID(this.element.dataset.fitTypeId);if(!id) throw new Error('Invalid item fittings');return id}
  private get form() {return this.q<HTMLFormElement>('[data-fit-filter-form]')}
  connect() {this.alive = true;this.desired ??= panelParams(this.element);this.activate()}
  private params() {return new URLSearchParams(this.desired ?? panelParams(this.element))}
  private syncLabels(panel:HTMLElement) {
    const names = new Map([...panel.querySelectorAll<HTMLElement>('[data-fit-meta] [data-fit-group]')].map(group=>[group.dataset.fitGroup,group.querySelector('span')?.textContent]))
    for(const chip of panel.querySelectorAll<HTMLElement>('[data-fit-group-chip]')) {const label = names.get(chip.dataset.fitGroup) || 'Module group '+chip.dataset.fitGroup;requiredElement(chip,'span').textContent = 'Requires '+label;chip.title = 'Remove required '+label+' module group'}
  }
  private showError(target:HTMLElement,kind:'fits'|Supporting,message:string) {
    target.querySelector(`[data-item-fits-error="${kind}"]`)?.remove()
    const box = document.createElement('p'),retry = document.createElement('button');box.className = 'widget-error';box.dataset.itemFitsError = kind;box.setAttribute('role','alert');box.textContent = message+' ';retry.type = 'button';retry.dataset.itemFitsRetry = kind;retry.textContent = 'Retry';box.append(retry);target.prepend(box)
  }
  private activate() {for(const segment of ['fit-meta','fit-distributions'] as const) void this.support(segment)}
  private async support(segment:Supporting,force = false) {
    const target = this.element.querySelector<HTMLElement>(selectors[segment]),query = this.element.dataset.fitQuery ?? '',id = this.id
    if(!this.alive || !target || this.supporting.has(segment) || !force && target.dataset.fitQuery === query) return
    const request = new AbortController();this.supporting.set(segment,{request,target});target.setAttribute('aria-busy','true');target.querySelector('[data-item-fits-error]')?.remove()
    const loading = target.querySelector<HTMLElement>('.item-fit-loading');if(loading) loading.hidden = false
    try {
      const html = await this.text(`/_fragments/item/${segment}/${id}?${query}`,'Could not load fit data.',{signal:request.signal})
      if(!this.alive || this.supporting.get(segment)?.request !== request || !target.isConnected) return
      const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>(selectors[segment])
      if(!next || marketID(next.dataset.fitTypeId) !== id || next.dataset.fitQuery === undefined || fitFilterKey(new URLSearchParams(next.dataset.fitQuery)) !== fitFilterKey(new URLSearchParams(query))) throw new Error('Invalid supporting fit data')
      replaceFragment(target,next);this.syncLabels(this.element)
    } catch(error) {if(this.alive && this.supporting.get(segment)?.request === request && target.isConnected && !isAbortError(error)) {if(loading) loading.hidden = true;this.showError(target,segment,'Could not load fit data.')}}
    finally {if(this.supporting.get(segment)?.request === request) {this.supporting.delete(segment);target.removeAttribute('aria-busy')}}
  }
  private async reload(params:URLSearchParams) {
    if(!this.alive) return
    this.desired = new URLSearchParams(params);this.dirty = true;this.read?.abort();const request = new AbortController(),id = this.id,path = location.pathname;this.read = request;this.element.querySelector('[data-item-fits-error="fits"]')?.remove()
    try {
      const html = await this.text(`/_fragments/item/fittings/${id}?${params}`,'Could not load fits.',{signal:request.signal})
      if(!this.alive || this.read !== request || location.pathname !== path || location.pathname !== `/item/${id}/fittings`) return
      const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('[data-item-fits]')
      if(!next || marketID(next.dataset.fitTypeId) !== id || itemFitsKey(panelParams(next)) !== itemFitsKey(params)) throw new Error('Invalid fitting response')
      const oldForm = this.form,nextForm = requiredElement<HTMLFormElement>(next,'[data-fit-filter-form]');nextForm.hidden = oldForm.hidden
      for(const field of oldForm.querySelectorAll<HTMLInputElement|HTMLSelectElement>('input[name],select[name]')) {const replacement = nextForm.elements.namedItem(field.name);if(replacement instanceof HTMLInputElement || replacement instanceof HTMLSelectElement) replacement.value = field.value}
      requiredElement<HTMLElement>(nextForm,'.item-fit-npc').hidden = select(nextForm,'metric').value !== 'npc_ehp'
      const expanded = new Set([...this.all<HTMLElement>('[data-fit-family]')].filter(family=>family.querySelector('details')?.open).map(family=>family.dataset.fitFamily))
      for(const family of next.querySelectorAll<HTMLElement>('[data-fit-family]')) {const details = family.querySelector('details');if(details) details.open = expanded.has(family.dataset.fitFamily)}
      const selected = new Set((params.get('groups') ?? '').split(','))
      // Retain supporting DOM until its matching refresh finishes. The new scope
      // owns replacement reads; disconnect cancels this scope's old requests.
      replaceFragment(this.element,next,()=>{
        for(const selector of Object.values(selectors)) {const previous = this.element.querySelector<HTMLElement>(selector),replacement = next.querySelector(selector);if(previous?.hasAttribute('data-fit-query') && replacement) {previous.removeAttribute('aria-busy');previous.querySelector('[data-item-fits-error]')?.remove();replacement.replaceWith(previous)}}
        for(const group of next.querySelectorAll<HTMLElement>('[data-fit-group]')) {const active = selected.has(group.dataset.fitGroup ?? '');group.classList.toggle('active',active);group.setAttribute('aria-pressed',String(active))}
        this.syncLabels(next)
      })
      const url = new URL(location.href);for(const key of [...url.searchParams.keys()]) if(key.startsWith('min_') || key.startsWith('max_') || ['groups','npc_profile','sort'].includes(key)) url.searchParams.delete(key)
      for(const [key,value] of params) url.searchParams.set(key,value);history.replaceState(history.state,'',url);this.dirty = false
    } catch(error) {if(this.alive && this.read === request && !isAbortError(error)) {this.dirty = false;this.showError(this.element,'fits','Could not load fits.')}}
    finally {if(this.read === request) this.read = undefined}
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    const retry = target.closest<HTMLElement>('[data-item-fits-retry]')?.dataset.itemFitsRetry
    if(retry === 'fits') {if(!this.read) void this.reload(this.params());return}
    if(retry === 'fit-meta' || retry === 'fit-distributions') {void this.support(retry,true);return}
    if(target.closest('[data-fit-filter-toggle]')) {this.form.hidden = !this.form.hidden;return}
    const params = this.params(),group = target.closest<HTMLElement>('[data-fit-group]'),bucket = target.closest<HTMLElement>('[data-fit-bucket-metric]'),remove = target.closest<HTMLElement>('[data-fit-remove-metric]')
    if(group && marketID(group.dataset.fitGroup)) {const value = group.dataset.fitGroup!,selected = (params.get('groups') ?? '').split(',').filter(Boolean),next = selected.includes(value) ? selected.filter(id=>id !== value) : [...selected,value];if(next.length) params.set('groups',next.join(','));else params.delete('groups')}
    else if(bucket && isFitMetric(bucket.dataset.fitBucketMetric) && bucket.dataset.fitBucketMin !== undefined && bucket.dataset.fitBucketMax !== undefined) {params.set('min_'+bucket.dataset.fitBucketMetric,bucket.dataset.fitBucketMin);params.set('max_'+bucket.dataset.fitBucketMetric,bucket.dataset.fitBucketMax)}
    else if(remove && isFitMetric(remove.dataset.fitRemoveMetric)) {params.delete('min_'+remove.dataset.fitRemoveMetric);params.delete('max_'+remove.dataset.fitRemoveMetric);if(remove.dataset.fitRemoveMetric === 'npc_ehp') params.delete('npc_profile')}
    else if(target.closest('[data-fit-clear-groups]')) params.delete('groups')
    else if(target.closest('[data-fit-clear]')) {for(const key of [...params.keys()]) if(key !== 'sort') params.delete(key)}
    else return
    void this.reload(params)
  }
  change(event:Event) {
    const target = eventElement(event.target);if(!this.alive || !(target instanceof HTMLSelectElement)) return
    const params = this.params()
    if(target.matches('[data-fit-sort]')) {if(target.value === 'observed') params.delete('sort');else params.set('sort',target.value)}
    else if(target.matches('[data-fit-metric]') && isFitMetric(target.value)) {this.q<HTMLElement>('.item-fit-npc').hidden = target.value !== 'npc_ehp';params.set('sort',sortForFitMetric(target.value));if(target.value === 'npc_ehp') params.set('npc_profile',select(this.form,'npc_profile').value)}
    else if(target.name === 'npc_profile') {params.set('npc_profile',target.value);params.set('sort','npc_ehp')}
    else return
    void this.reload(params)
  }
  input(event:Event) {if(eventElement(event.target)?.closest('[data-fit-filter-form]')) formInput(this.form,'min').setCustomValidity('')}
  submit(event:SubmitEvent) {
    if(!this.alive || event.target !== this.form) return
    event.preventDefault();const form = this.form,metric = select(form,'metric').value;if(!isFitMetric(metric)) return
    const min = formInput(form,'min'),max = formInput(form,'max'),minimum = parseFitNumber(min.value),maximum = parseFitNumber(max.value)
    if(min.value.trim() && minimum === null || max.value.trim() && maximum === null || minimum === null && maximum === null && metric !== 'npc_ehp') {min.setCustomValidity('Enter a non-negative number, optionally followed by k, m, or b.');min.reportValidity();return}
    min.setCustomValidity('');const params = this.params();params.delete('min_'+metric);params.delete('max_'+metric);if(minimum !== null) params.set('min_'+metric,String(minimum));if(maximum !== null) params.set('max_'+metric,String(maximum));if(metric === 'npc_ehp') params.set('npc_profile',select(form,'npc_profile').value);params.set('sort',sortForFitMetric(metric));min.value = '';max.value = '';void this.reload(params)
  }
  hide() {this.alive = false;this.read?.abort();this.read = undefined;for(const {request,target} of this.supporting.values()) {request.abort();target.removeAttribute('aria-busy')}this.supporting.clear();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.dirty) void this.reload(this.params());else this.activate()}}
}
registerController('item-fits',ItemFitsController)
