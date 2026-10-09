import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'
import {isAbortError} from './shared/json'
import {readSearchResults,searchTypes,isSearchType,type SearchType} from './shared/search-model'
import {searchNode,searchIcon,searchCard} from './shared/search-cards'

export class SearchPageController extends RequestComponent {
  private alive = false
  private timer:ReturnType<typeof setTimeout>|undefined
  private read:AbortController|undefined
  private cards:Array<{type:SearchType;node:HTMLElement}> = []
  private counts:Partial<Record<SearchType,number>> = {}
  private total = 0
  private active:SearchType|'all' = 'all'
  private searched = false
  private interrupted = false
  private get inputControl() {return this.q<HTMLInputElement>('#search-page-input')}
  private get spinner() {return this.q<HTMLElement>('#search-page-spinner')}
  private get filters() {return this.q<HTMLElement>('#search-page-filters')}
  connect() {this.alive = true;if(this.inputControl.value.trim().length >= 2) this.search();else this.reset(false)}
  private cancel() {this.stopTimer(this.timer);this.timer = undefined;this.read?.abort();this.read = undefined;this.spinner.hidden = true}
  private url(query:string) {const url = new URL(location.href);if(query) url.searchParams.set('q',query);else url.searchParams.delete('q');history.replaceState(history.state,'',url)}
  private render() {
    const grid = this.q<HTMLElement>('#search-page-grid');grid.replaceChildren(...this.cards.filter(card=>this.active === 'all' || card.type === this.active).map(card=>card.node));grid.hidden = !grid.childElementCount;this.q<HTMLElement>('#search-page-initial').hidden = this.searched || this.inputControl.value.trim().length >= 2
    const empty = this.q<HTMLElement>('#search-page-empty');empty.hidden = !this.searched || grid.childElementCount > 0;if(!empty.hidden) this.q('#search-page-empty-message').textContent = `No results found for "${this.inputControl.value.trim()}"`
  }
  private renderTabs() {
    this.filters.replaceChildren()
    const choices = Object.entries(this.counts).flatMap(([type,count])=>isSearchType(type) && typeof count === 'number' && count > 0 ? [{type,label:searchTypes[type].label,count}] : [])
    if(!this.searched || !choices.length) {this.filters.hidden = true;return}
    for(const choice of [{type:'all',label:'All',count:this.total},...choices]) {const button = searchNode('button','search-page-tab'+(this.active === choice.type ? ' active' : ''));button.type = 'button';button.dataset.type = choice.type;button.setAttribute('aria-pressed',String(this.active === choice.type));button.append(searchIcon(isSearchType(choice.type) ? searchTypes[choice.type].icon : 'tags'),searchNode('span','',choice.label),searchNode('small','',choice.count));this.filters.append(button)}
    this.filters.hidden = false
  }
  private reset(updateURL = true) {this.cancel();this.interrupted = false;this.cards = [];this.counts = {};this.total = 0;this.active = 'all';this.searched = false;this.q<HTMLElement>('#search-page-clear').hidden = !this.inputControl.value;this.q<HTMLElement>('#search-page-stats').hidden = true;this.filters.hidden = true;this.error('[data-search-error]');if(updateURL) this.url('');document.title = 'Search - EVE-KILL.com';this.render()}
  search() {
    this.cancel();if(!this.alive) return
    const query = this.inputControl.value.trim();this.q<HTMLElement>('#search-page-clear').hidden = !query
    if(query.length < 2) {this.reset();return}
    this.interrupted = true;this.spinner.hidden = false;this.q<HTMLElement>('#search-page-stats').hidden = true;this.q<HTMLElement>('#search-page-empty').hidden = true;this.q<HTMLElement>('#search-page-initial').hidden = true;this.error('[data-search-error]');document.title = `Search: ${query} - EVE-KILL.com`
    this.timer = this.after(200,()=>{this.timer = undefined;void this.load(query)})
  }
  private async load(query:string) {
    const request = new AbortController(),path = location.pathname;this.read = request
    try {
      const data = readSearchResults(await this.request<unknown>('/api/search?q='+encodeURIComponent(query),{signal:request.signal},'Search is unavailable. Please try again.'))
      if(!this.alive || this.read !== request || this.inputControl.value.trim() !== query || location.pathname !== path) return
      this.cards = data.hits.map(hit=>({type:hit.type,node:searchCard(hit)}));this.counts = data.counts;this.total = data.total;this.active = 'all';this.searched = true
      const stats = this.q<HTMLElement>('#search-page-stats');stats.textContent = `${data.total.toLocaleString()} result${data.total === 1 ? '' : 's'} · ${data.processingTimeMs}ms`;stats.hidden = false;this.renderTabs();this.render();this.url(query);document.title = `Search: ${query} - EVE-KILL.com`;this.interrupted = false
    } catch(error) {if(this.alive && this.read === request && !isAbortError(error)) {this.interrupted = false;const box = this.q<HTMLElement>('[data-search-error]');box.replaceChildren('Search is unavailable. Please try again. ',searchNode('button','search-page-tab','Retry'));const retry = box.querySelector('button')!;retry.type = 'button';retry.dataset.searchRetry = '';box.hidden = false}}
    finally {if(this.read === request) {this.read = undefined;this.spinner.hidden = true}}
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    if(target.closest('#search-page-clear')) {this.inputControl.value = '';this.reset();this.inputControl.focus();return}
    const suggestion = target.closest<HTMLElement>('[data-suggestion]');if(suggestion?.dataset.suggestion) {this.inputControl.value = suggestion.dataset.suggestion;this.search();this.inputControl.focus();return}
    if(target.closest('[data-search-retry]')) {if(!this.read && !this.timer) this.search();return}
    const filter = target.closest<HTMLElement>('.search-page-tab[data-type]'),type = filter?.dataset.type
    if(type === 'all' || isSearchType(type)) {this.active = type;for(const button of this.filters.querySelectorAll<HTMLElement>('.search-page-tab')) {const active = button.dataset.type === type;button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}this.render()}
  }
  input(event:Event) {if(event.target === this.inputControl) this.search()}
  key(event:KeyboardEvent) {if(event.target === this.inputControl && event.key === 'Enter') event.preventDefault()}
  history() {if(this.alive && location.pathname === '/search') {this.inputControl.value = new URL(location.href).searchParams.get('q') ?? '';if(this.inputControl.value.trim().length >= 2) this.search();else this.reset(false)}}
  hide() {this.alive = false;this.cancel();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.interrupted) this.search()}}
}
registerController('search-page',SearchPageController)
