import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import './controllers/item-market'
import './controllers/market-search'
import {eventElement} from './shared/dom'
import {marketID} from './shared/market'
import {isAbortError} from './shared/json'

export class MarketBrowserController extends RequestComponent {
  private alive = false
  private reads = new Map<HTMLDetailsElement,AbortController>()
  connect() {this.alive = true;this.resume()}
  sidebar() {const sidebar = this.q<HTMLElement>('[data-market-sidebar]'),button = this.q<HTMLButtonElement>('[data-market-sidebar-toggle]'),open = sidebar.classList.toggle('open');button.textContent = `${open ? 'Hide' : 'Show'} ${this.element.classList.contains('market-standalone') ? 'Market Browser' : 'Categories'}`;button.setAttribute('aria-expanded',String(open))}
  click(event:MouseEvent) {if(this.alive && eventElement(event.target)?.closest('[data-market-sidebar-toggle]')) this.sidebar()}
  private visible(details:HTMLDetailsElement) {let parent = details.parentElement;while(parent && parent !== this.element) {if(parent instanceof HTMLDetailsElement && !parent.open) return false;parent = parent.parentElement}return details.open}
  private cancel(details:HTMLDetailsElement) {this.reads.get(details)?.abort();this.reads.delete(details);delete details.dataset.marketLoading;details.removeAttribute('aria-busy')}
  toggle(event:Event) {
    const details = eventElement(event.target);if(!(details instanceof HTMLDetailsElement) || !details.matches('[data-market-node]')) return
    if(!details.open) {for(const item of [...this.reads.keys()]) if(item === details || details.contains(item)) this.cancel(item)}else this.resume()
  }
  private resume() {if(this.alive) for(const details of this.all<HTMLDetailsElement>('details[data-market-node]')) if(this.visible(details)) void this.load(details)}
  private async load(details:HTMLDetailsElement) {
    if(!this.alive || !this.visible(details) || details.querySelector(':scope > .market-tree') || this.reads.has(details)) return
    const id = marketID(details.dataset.marketNode);if(!id) return
    const request = new AbortController();this.reads.set(details,request);details.dataset.marketLoading = 'true';details.setAttribute('aria-busy','true');details.querySelector(':scope > .widget-error')?.remove()
    try {
      const html = await this.text('/_fragments/market/tree/'+id,'Could not load categories.',{signal:request.signal})
      if(!this.alive || this.reads.get(details) !== request || !this.visible(details)) return
      const tree = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('.market-tree')
      if(!tree || marketID(tree.dataset.marketTreeParent) !== id) throw new Error('Invalid category tree')
      details.append(tree)
    } catch(error) {if(this.alive && this.reads.get(details) === request && !isAbortError(error)) {const message = document.createElement('p');message.className = 'widget-error';message.textContent = 'Could not load categories. Close and reopen to try again.';details.append(message)}}
    finally {if(this.reads.get(details) === request) {this.reads.delete(details);delete details.dataset.marketLoading;details.removeAttribute('aria-busy')}}
  }
  hide() {this.alive = false;for(const details of [...this.reads.keys()]) this.cancel(details);super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.resume()}}
}
registerController('market-browser',MarketBrowserController)
