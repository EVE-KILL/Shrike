import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'
import {isAbortError} from './shared/json'
import {replaceFragmentChildren} from './fragment'

type Snapshot = {nodes:Node[];at:number}
type Card = {node:HTMLElement;key:string;period:boolean;section:string;content:HTMLElement;cache:Map<string,Snapshot>;displayed:string;read?:AbortController;queued:boolean}
type Job = {card:Card;desired:string;period:string}
export class StatisticsController extends RequestComponent {
  private alive = false
  private cards:Card[] = []
  private filter = 'all'
  private period = '7'
  private queue:Job[] = []
  private running = new Set<AbortController>()
  private get filters() {return [...this.all<HTMLButtonElement>('[data-stats-filter-button]')]}
  private get periods() {return [...this.all<HTMLButtonElement>('[data-stats-period-button]')]}
  connect() {
    this.alive = true;this.filter = this.validFilter(this.element.dataset.statsFilter);this.period = this.validPeriod(this.element.dataset.statsPeriod)
    if(!this.cards.length) for(const node of this.all<HTMLElement>('[data-stats-card]')) {
      const key = node.dataset.statsCard,section = node.closest<HTMLElement>('[data-stats-section]')?.dataset.statsSection
      if(!key || !section) throw new Error('Invalid statistics card')
      const card:Card = {node,key,section,period:node.dataset.statsKind === 'period',content:requiredElement(node,'[data-stats-card-content]'),cache:new Map(),displayed:'',queued:false}
      card.displayed = this.slot(card)
      if(node.dataset.statsLoaded === 'true') card.cache.set(card.displayed,{nodes:[...card.content.childNodes],at:Date.now()})
      this.cards.push(card)
    }
    this.selection();this.loadVisible()
  }
  private validFilter(value:unknown) {return typeof value === 'string' && this.filters.some(button=>button.dataset.statsFilterButton === value) ? value : 'all'}
  private validPeriod(value:unknown) {return this.periods.find(button=>Number(button.dataset.statsPeriodButton) === Number(value))?.dataset.statsPeriodButton ?? '7'}
  private slot(card:Card) {return card.period ? this.period : 'population'}
  private ttl(card:Card) {return card.period && Number(this.period) < 1 ? 30000 : 60000}
  private visible(card:Card) {return this.filter === 'all' || card.section === this.filter}
  private clone(selector:string) {return this.q<HTMLTemplateElement>(selector).content.cloneNode(true)}
  private styles(node:HTMLElement,active:boolean,chosen:string[],other:string[]) {for(const name of chosen) node.classList.toggle(name,active);for(const name of other) node.classList.toggle(name,!active);node.setAttribute('aria-pressed',String(active))}
  private selection() {
    this.element.dataset.statsFilter = this.filter;this.element.dataset.statsPeriod = this.period
    for(const button of this.filters) this.styles(button,button.dataset.statsFilterButton === this.filter,['bg-blue-500/20','text-blue-400','border-blue-500/30'],['text-gray-400','border-white/[0.08]','hover:text-blue-400','hover:bg-blue-500/[0.04]'])
    for(const button of this.periods) this.styles(button,button.dataset.statsPeriodButton === this.period,['bg-blue-500/20','text-blue-400'],['bg-white/[0.02]','text-gray-500','hover:bg-white/[0.05]','hover:text-gray-300'])
    const sections = [...this.all<HTMLElement>('[data-stats-section]')]
    this.q<HTMLElement>('[data-stats-period-controls]').hidden = this.filter !== 'all' && !sections.some(section=>section.hasAttribute('data-stats-period-section') && section.dataset.statsSection === this.filter)
    this.q<HTMLElement>('[data-stats-jumps]').hidden = this.filter !== 'all';const separator = this.element.querySelector<HTMLElement>('[data-stats-separator]');if(separator) separator.hidden = this.filter !== 'all'
    for(const section of sections) section.hidden = this.filter !== 'all' && section.dataset.statsSection !== this.filter
    const label = this.periods.find(button=>button.dataset.statsPeriodButton === this.period)?.textContent ?? ''
    for(const node of this.all('[data-stats-period-label]')) node.textContent = label
  }
  private display(card:Card) {
    const desired = this.slot(card),cached = card.cache.get(desired)
    if(card.displayed !== desired || cached && card.node.dataset.statsLoaded !== 'true' || !cached && card.content.querySelector('[data-stats-retry]')) {
      card.displayed = desired;replaceFragmentChildren(card.content,...(cached ? cached.nodes : [this.clone(card.period ? '[data-stats-period-placeholder]' : '[data-stats-population-placeholder]')]))
      card.node.dataset.statsLoaded = String(Boolean(cached))
    }
    return cached
  }
  private cancel(card:Card) {if(card.read) {card.read.abort();this.running.delete(card.read);card.read = undefined}card.queued = false;card.content.setAttribute('aria-busy','false')}
  private enqueue(card:Card,force = false) {
    if(!this.alive || !this.visible(card)) return
    const cached = this.display(card)
    if(!force && cached && Date.now()-cached.at < this.ttl(card)) {if(!card.read && !card.queued) card.content.setAttribute('aria-busy','false');return}
    if(card.read || card.queued) return
    card.content.querySelector('[data-stats-request-error]')?.remove();card.content.setAttribute('aria-busy','true');card.queued = true;this.queue.push({card,desired:this.slot(card),period:this.period})
  }
  private loadVisible() {for(const card of this.cards) this.enqueue(card);this.pump()}
  private pump() {
    while(this.alive && this.running.size < 4 && this.queue.length) {
      const job = this.queue.shift();if(!job) return;job.card.queued = false
      if(job.desired !== this.slot(job.card) || !this.visible(job.card)) {job.card.content.setAttribute('aria-busy','false');continue}
      void this.load(job)
    }
  }
  private async load({card,desired,period}:Job) {
    const request = new AbortController();card.read = request;this.running.add(request)
    try {
      const html = await this.text('/_fragments/stats/'+encodeURIComponent(card.key)+(card.period ? '?'+new URLSearchParams({period}) : ''),'Statistics unavailable',{signal:request.signal,headers:{Accept:'text/html'}})
      if(!this.alive || card.read !== request || desired !== this.slot(card) || !this.visible(card)) return
      const doc = new DOMParser().parseFromString(html,'text/html'),loaded = doc.querySelector<HTMLElement>('[data-stats-card-loaded]')
      if(!loaded || loaded.dataset.statsCardLoaded !== card.key || loaded.dataset.statsCardPeriod !== desired) throw new Error('Invalid statistics table')
      const nodes = [...doc.body.childNodes].map(node=>document.importNode(node,true));card.cache.set(desired,{nodes,at:Date.now()});replaceFragmentChildren(card.content,...nodes);card.displayed = desired;card.node.dataset.statsLoaded = 'true'
    } catch(error) {
      if(!isAbortError(error) && this.alive && card.read === request && desired === this.slot(card)) {
        const error = this.clone('[data-stats-error-template]');if(card.cache.has(desired)) {const panel = document.createElement('div');panel.dataset.statsRequestError = '';panel.append(error);card.content.append(panel)}else {replaceFragmentChildren(card.content,error);card.node.dataset.statsLoaded = 'false'}
      }
    } finally {this.running.delete(request);if(card.read === request) {card.read = undefined;card.content.setAttribute('aria-busy','false')}this.pump()}
  }
  private changePeriod(value:string) {
    if(value === this.period) return
    this.period = value;this.queue = this.queue.filter(job=>!job.card.period)
    for(const card of this.cards) if(card.period) {this.cancel(card);if(this.visible(card)) this.display(card)}
  }
  private changeFilter(value:string) {
    this.filter = value;this.queue = this.queue.filter(job=>this.visible(job.card))
    for(const card of this.cards) if(!this.visible(card)) this.cancel(card)
  }
  private historyWrite() {const url = new URL(location.href);if(this.filter === 'all') url.searchParams.delete('section');else url.searchParams.set('section',this.filter);if(this.period === '7') url.searchParams.delete('period');else url.searchParams.set('period',this.period);if(url.href !== location.href) history.pushState(history.state,'',url)}
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!target || !this.alive) return
    const choice = target.closest<HTMLElement>('[data-stats-filter-button]'),period = target.closest<HTMLElement>('[data-stats-period-button]'),jump = target.closest<HTMLElement>('[data-stats-jump]'),retry = target.closest('[data-stats-retry]')
    if(choice) this.changeFilter(this.validFilter(choice.dataset.statsFilterButton))
    else if(period) this.changePeriod(this.validPeriod(period.dataset.statsPeriodButton))
    else if(jump) {event.preventDefault();const section = [...this.all<HTMLElement>('[data-stats-section]')].find(section=>section.dataset.statsSection === jump.dataset.statsJump);section?.scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth',block:'start'});return}
    else if(retry) {const card = this.cards.find(card=>card.node === retry.closest('[data-stats-card]'));if(card) {this.enqueue(card,true);this.pump()}return}
    else return
    this.selection();this.historyWrite();this.loadVisible()
  }
  history() {const query = new URL(location.href).searchParams;this.changeFilter(this.validFilter(query.get('section')));this.changePeriod(this.validPeriod(query.get('period')));this.selection();this.loadVisible()}
  hide() {this.alive = false;this.queue = [];for(const card of this.cards) this.cancel(card);this.running.clear();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.selection();this.loadVisible()}}
}
registerController('statistics',StatisticsController)
