import {RequestComponent} from './request-component'
import {registerController} from './application'
import {eventElement} from '../shared/dom'
import {isAbortError} from '../shared/json'
import {replaceFragment} from '../fragment'
import {marketID,isMarketRegion,isMarketSort,isMarketSecurity,readMarketFilters,marketParams,toggleMarketSecurity,sortMarketBook,type MarketFilters} from '../shared/market'

export class ItemMarketController extends RequestComponent {
  private alive = false
  private filters:MarketFilters|undefined
  private read:AbortController|undefined
  private dirty = false
  private updateHistory = false
  private get identity() {const id = marketID(this.element.dataset.marketTypeId),view = this.element.dataset.marketView;if(!id || view !== 'item' && view !== 'standalone') throw new Error('Invalid market component');return {id,view}}
  connect() {this.alive = true;this.filters ??= readMarketFilters(this.element.dataset)}
  private async reload(updateHistory = true) {
    if(!this.alive || !this.filters) return
    this.read?.abort();const request = new AbortController(),filters = this.filters,{id,view} = this.identity,path = location.pathname;this.read = request;this.dirty = true;this.updateHistory = updateHistory
    this.element.querySelector('[data-market-error]')?.remove()
    try {
      const html = await this.text((view === 'standalone' ? '/_fragments/market/item/' : '/_fragments/item/market/')+id+'?'+marketParams(filters),'Could not load market.',{signal:request.signal})
      if(!this.alive || this.read !== request || location.pathname !== path || location.pathname !== (view === 'item' ? `/item/${id}/market` : `/market/item/${id}`)) return
      const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('[data-item-market]')
      if(!next || marketID(next.dataset.marketTypeId) !== id || next.dataset.marketView !== view || marketParams(readMarketFilters(next.dataset)).toString() !== marketParams(filters).toString()) throw new Error('Invalid market response')
      for(const book of this.all<HTMLElement>('[data-market-book]')) {
        const header = book.querySelector<HTMLElement>('th[aria-sort="ascending"],th[aria-sort="descending"]'),key = header?.querySelector<HTMLElement>('[data-market-sort]')?.dataset.marketSort
        const replacement = [...next.querySelectorAll<HTMLElement>('[data-market-book]')].find(candidate=>candidate.dataset.marketBook === book.dataset.marketBook)
        if(replacement && isMarketSort(key)) sortMarketBook(replacement,key,header?.getAttribute('aria-sort') === 'ascending' ? 1 : -1)
      }
      if(updateHistory) {const url = new URL(location.href);for(const key of ['region','days','security']) url.searchParams.delete(key);if(filters.region !== '0') url.searchParams.set('region',filters.region);if(filters.days !== '30') url.searchParams.set('days',filters.days);if(filters.security.join(',') !== 'high,low,null') url.searchParams.set('security',filters.security.join(','));history.replaceState(history.state,'',url)}
      this.dirty = false;replaceFragment(this.element,next)
    } catch(error) {
      if(this.alive && this.read === request && !isAbortError(error)) {this.dirty = false;const box = document.createElement('p'),retry = document.createElement('button');box.className = 'widget-error';box.dataset.marketError = '';box.setAttribute('role','alert');box.textContent = 'Could not load market. ';retry.type = 'button';retry.dataset.marketRetry = '';retry.textContent = 'Retry';box.append(retry);this.element.prepend(box)}
    } finally {if(this.read === request) this.read = undefined}
  }
  change(event:Event) {const control = eventElement(event.target);if(this.alive && this.filters && control instanceof HTMLSelectElement && control.matches('[data-market-region]') && isMarketRegion(control.value)) {this.filters = {...this.filters,region:control.value};void this.reload()}}
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target || !this.filters) return
    if(target.closest('[data-market-retry]')) {if(!this.read) void this.reload();return}
    const days = target.closest<HTMLElement>('[data-market-days]'),security = target.closest<HTMLElement>('[data-market-security]')
    if(days?.dataset.marketDays === '30' || days?.dataset.marketDays === '90') {this.filters = {...this.filters,days:days.dataset.marketDays};void this.reload();return}
    if(security && isMarketSecurity(security.dataset.marketSecurity)) {const next = toggleMarketSecurity(this.filters,security.dataset.marketSecurity);if(next) {this.filters = next;void this.reload()}return}
    const sort = target.closest<HTMLElement>('[data-market-sort]'),key = sort?.dataset.marketSort,book = sort?.closest<HTMLElement>('[data-market-book]'),header = sort?.closest('th')
    if(book && header && isMarketSort(key)) {const current = header.getAttribute('aria-sort'),direction = current === 'ascending' ? -1 : current === 'descending' ? 1 : key === 'quantity' || key === 'price' && book.dataset.marketBook === 'buyers' ? -1 : 1;sortMarketBook(book,key,direction)}
  }
  history() {
    if(!this.alive) return
    const url = new URL(location.href),{id,view} = this.identity
    // Entity-tab navigation owns its entire fragment, including Back. The
    // standalone market has no parent navigator and reconciles filters here.
    if(view === 'item') return
    if(url.pathname !== '/market/item/'+id) return
    try {this.filters = readMarketFilters({region:url.searchParams.get('region') ?? '0',days:url.searchParams.get('days') ?? '30',security:url.searchParams.get('security') ?? 'high,low,null'});void this.reload(false)}catch{/* Invalid URL filters are left to the server's full-page validation. */}
  }
  hide() {this.alive = false;this.read?.abort();this.read = undefined;super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.dirty) void this.reload(this.updateHistory)}}
}
registerController('item-market',ItemMarketController)
