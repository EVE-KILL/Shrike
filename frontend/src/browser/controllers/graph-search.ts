import {RequestComponent} from './request-component'
import {registerController} from './application'
import {eventElement} from '../shared/dom'
import {escapeHTML} from '../shared/html'
import {isGraphMode,isGraphEntityType,isGraphSearchKey,graphSearchType,readGraphHits,type GraphHit} from '../shared/graph-model'

export class GraphSearchController extends RequestComponent {
  private alive = false
  private timer:ReturnType<typeof setTimeout>|undefined
  private read:AbortController|undefined
  private hits:GraphHit[] = []
  private get inputControl() {return this.q<HTMLInputElement>('[data-graph-input]')}
  private get results() {return this.q<HTMLElement>('[data-graph-hits]')}
  private get panel() {const panel = this.element.closest<HTMLElement>('[data-graph-panel]');if(!panel) throw new Error('Missing graph search panel');return panel}
  connect() {this.alive = true}
  private metadata() {
    const mode = this.panel.dataset.graphPanel,key = this.element.dataset.graphSearch,type = this.panel.dataset.graphEntityType ?? 'alliance'
    if(!isGraphMode(mode) || !isGraphSearchKey(key) || !isGraphEntityType(type)) throw new Error('Invalid graph search component')
    return {mode,key,type:graphSearchType(mode,type)}
  }
  input() {
    if(!this.alive) return
    this.hideResults();this.hits = [];const {mode,key,type} = this.metadata(),query = this.inputControl.value.trim()
    this.dispatch('change',{detail:{mode,key}})
    if(query.length < 3) return
    const request = new AbortController();this.read = request
    this.timer = this.after(300,()=>{this.timer = undefined;void this.search(query,type,request)})
  }
  private async search(query:string,type:ReturnType<typeof graphSearchType>,request:AbortController) {
    try {
      const raw = await this.request<unknown>('/api/search?'+new URLSearchParams({q:query,type}),{signal:request.signal},'Search unavailable')
      if(!this.alive || this.read !== request || this.inputControl.value.trim() !== query) return
      this.hits = readGraphHits(raw,type);this.renderHits()
    } catch {if(this.alive && this.read === request) {this.hits = [];this.renderHits()}}
    finally {if(this.read === request) this.read = undefined}
  }
  private renderHits() {
    const {mode} = this.metadata()
    this.results.replaceChildren(...this.hits.map((hit,index)=>{
      const button = document.createElement('button');button.type = 'button';button.dataset.graphHit = String(index);button.className = 'w-full text-left px-3 py-2 text-sm hover:bg-white/[0.06] flex items-baseline gap-2'
      button.innerHTML = `<span class="text-white">${escapeHTML(hit.name)}</span>`+(mode === 'path_finder'
        ? `${hit.corporation_name ? `<span class="text-xs text-gray-500">${escapeHTML(hit.corporation_name)}</span>` : ''}${hit.alliance_name ? `<span class="text-xs text-gray-600">[${escapeHTML(hit.alliance_ticker || hit.alliance_name)}]</span>` : ''}`
        : hit.ticker ? `<span class="text-xs text-gray-500">[${escapeHTML(hit.ticker)}]</span>` : '')
      return button
    }))
    this.results.hidden = !this.hits.length;this.inputControl.setAttribute('aria-expanded',String(this.hits.length > 0))
  }
  choose(event:MouseEvent) {
    const button = eventElement(event.target)?.closest<HTMLElement>('[data-graph-hit]'),index = Number(button?.dataset.graphHit)
    if(!this.alive || !button || !Number.isSafeInteger(index)) return
    const hit = this.hits[index];if(!hit) return
    const {mode,key} = this.metadata();this.hideResults();this.inputControl.value = hit.name
    this.dispatch('pick',{detail:{mode,key,id:hit.id}});this.inputControl.focus()
  }
  key(event:KeyboardEvent) {
    const target = eventElement(event.target)
    if(target === this.inputControl) {
      if(event.key === 'ArrowDown' && !this.results.hidden) {event.preventDefault();this.results.querySelector<HTMLButtonElement>('button')?.focus()}
      if(event.key === 'Escape') this.hideResults()
      if(event.key === 'Enter') event.preventDefault()
      return
    }
    const button = target?.closest<HTMLButtonElement>('[data-graph-hit]');if(!button) return
    if(event.key === 'Escape') {this.hideResults();this.inputControl.focus();return}
    if(event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();const options = [...this.results.querySelectorAll<HTMLButtonElement>('button')],index = options.indexOf(button)+(event.key === 'ArrowDown' ? 1 : -1)
      if(index < 0) this.inputControl.focus();else options[Math.min(index,options.length-1)]?.focus()
    }
  }
  outside(event:PointerEvent) {if(event.target instanceof Node && !this.element.contains(event.target)) this.hideResults()}
  hideResults() {this.stopTimer(this.timer);this.timer = undefined;this.read?.abort();this.read = undefined;this.results.hidden = true;this.inputControl.setAttribute('aria-expanded','false')}
  reset() {this.hideResults();this.hits = [];this.inputControl.value = '';this.results.replaceChildren()}
  hide() {this.alive = false;this.hideResults();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) this.alive = true}
}
registerController('graph-search',GraphSearchController)
