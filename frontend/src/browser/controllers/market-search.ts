import {RequestComponent} from './request-component'
import {registerController} from './application'
import {eventElement} from '../shared/dom'
import {readMarketHits,mergeMarketHits,type MarketHit} from '../shared/market'
import {isAbortError} from '../shared/json'

export class MarketSearchController extends RequestComponent {
  private alive = false
  private timer:ReturnType<typeof setTimeout>|undefined
  private read:AbortController|undefined
  private get inputControl() {return this.q<HTMLInputElement>('input')}
  private get results() {return this.q<HTMLElement>('[data-market-results]')}
  connect() {this.alive = true}
  private busy(value:boolean) {const spinner = this.element.querySelector<HTMLElement>('.market-search-loading');if(spinner) spinner.hidden = !value}
  private cancel() {this.stopTimer(this.timer);this.timer = undefined;this.read?.abort();this.read = undefined;this.busy(false)}
  input() {this.cancel();const query = this.inputControl.value.trim();this.results.hidden = true;this.results.replaceChildren();if(!this.alive || query.length < 2) return;this.timer = this.after(220,()=>{this.timer = undefined;void this.search(query)})}
  private async search(query:string) {
    const request = new AbortController();this.read = request;this.busy(true)
    try {
      const payloads = await Promise.all((['ship','item'] as const).map(async type=>readMarketHits(await this.request<unknown>('/api/search?'+new URLSearchParams({q:query,type,limit:'8'}),{signal:request.signal},'Search unavailable'),type)))
      if(this.alive && this.read === request && this.inputControl.value.trim() === query) this.render(mergeMarketHits(payloads))
    } catch(error) {
      if(this.alive && this.read === request && !isAbortError(error)) {const message = document.createElement('p'),retry = document.createElement('button');message.textContent = 'Search unavailable. ';retry.type = 'button';retry.dataset.marketSearchRetry = '';retry.textContent = 'Retry';message.append(retry);this.results.replaceChildren(message);this.results.hidden = false}
    } finally {request.abort();if(this.read === request) {this.read = undefined;this.busy(false)}}
  }
  private render(hits:MarketHit[]) {
    this.results.replaceChildren()
    if(!hits.length) {const empty = document.createElement('p');empty.textContent = 'No market items found.';this.results.append(empty)}
    for(const hit of hits) {const link = document.createElement('a');link.href = '/market/item/'+hit.id;const icon = document.createElement('img');icon.src = '/images/types/'+hit.id+'/icon?size=32';icon.alt = '';icon.width = icon.height = 28;const name = document.createElement('span'),type = document.createElement('small');name.textContent = hit.name;type.textContent = hit.type;link.append(icon,name,type);this.results.append(link)}
    this.results.hidden = false
  }
  click(event:MouseEvent) {if(eventElement(event.target)?.closest('[data-market-search-retry]') && !this.read) void this.search(this.inputControl.value.trim())}
  focus(event:FocusEvent) {if(event.target === this.inputControl && this.results.childElementCount) this.results.hidden = false}
  key(event:KeyboardEvent) {
    if(event.key === 'Escape') {this.cancel();this.inputControl.focus();this.results.hidden = true;return}
    if(event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
    const links = [...this.results.querySelectorAll<HTMLAnchorElement>('a')]
    if(this.results.hidden || !links.length) return
    const target = eventElement(event.target),index = target instanceof HTMLAnchorElement ? links.indexOf(target) : -1,next = index+(event.key === 'ArrowDown' ? 1 : -1)
    event.preventDefault();if(next < 0) this.inputControl.focus();else links[Math.min(next,links.length-1)]?.focus()
  }
  outside(event:PointerEvent) {if(event.target instanceof Node && !this.element.contains(event.target)) this.results.hidden = true}
  hide() {this.alive = false;this.cancel();this.results.hidden = true;super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) this.alive = true}
}
registerController('market-search',MarketSearchController)
