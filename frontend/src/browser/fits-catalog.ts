import {FragmentSections} from './controllers/fragment-sections'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'

const parts = ['flavors','doctrines','top-rated'] as const
type Part = typeof parts[number]
const isPart = (value:unknown):value is Part=>parts.some(part=>part === value)
export function catalogueSelection(query = new URLSearchParams(location.search)) {return {flavor:query.get('flavor') === 'final_blows' || query.get('flavor') === 'losses' ? query.get('flavor') : 'kills',doctrines:query.get('doctrines') === 'corporation' ? 'corporation' : 'alliance'}}
export class FitsCatalogueController extends FragmentSections<Part> {
  protected retryAttribute = 'data-fits-retry'
  private alive = false
  private intents = new Set<Part>()
  connect() {this.alive = true;this.sync()}
  private sync() {const state = catalogueSelection();for(const [key,attribute] of [['flavor','data-fits-flavor'],['doctrines','data-fits-doctrine']] as const) for(const button of this.all<HTMLElement>(`[${attribute}]`)) {const active = button.getAttribute(attribute) === state[key];button.classList.toggle('active',active);button.setAttribute('aria-pressed',String(active))}}
  private async load(part:Part) {
    const container = this.element.querySelector<HTMLElement>(`[data-fits-part="${part}"]`);if(!this.alive || !container) return
    this.intents.add(part)
    const success = await this.section(part,{container,path:`/_fragments/fits/catalog/${part}${location.search}`,selector:`[data-fits-part="${part}"]`,message:'Unable to load fittings.',errorClass:'glass-panel p-6 text-sm text-red-300',replaceRoot:true,loaded:next=>{if(part === 'top-rated') {const panel = this.element.querySelector<HTMLElement>('[data-fits-top-rated]');if(panel) panel.hidden = !next.querySelector('[data-community-fit]')}}})
    if(success) this.intents.delete(part)
  }
  click(event:MouseEvent) {
    const button = eventElement(event.target)?.closest<HTMLButtonElement>('button');if(!this.alive || !button) return
    if(isPart(button.dataset.fitsRetry)) {void this.load(button.dataset.fitsRetry);return}
    const key = button.hasAttribute('data-fits-flavor') ? 'flavor' : button.hasAttribute('data-fits-doctrine') ? 'doctrines' : null;if(!key) return
    const value = key === 'flavor' ? button.dataset.fitsFlavor : button.dataset.fitsDoctrine;if(key === 'flavor' ? !['kills','final_blows','losses'].includes(value ?? '') : !['alliance','corporation'].includes(value ?? '')) return
    if(catalogueSelection()[key] === value) return
    const query = new URLSearchParams(location.search);if(value === (key === 'flavor' ? 'kills' : 'alliance')) query.delete(key);else query.set(key,value!)
    history.pushState(history.state,'',location.pathname+(query.size ? '?'+query : '')+location.hash);this.sync();void this.load(key === 'flavor' ? 'flavors' : 'doctrines')
  }
  back() {if(!this.alive) return;this.sync();void this.load('flavors');void this.load('doctrines')}
  hide() {this.alive = false;super.hide()}
  show(event:PageTransitionEvent) {if(!event.persisted) return;this.alive = true;this.sync();const resume = new Set([...this.intents,'flavors','doctrines'] as Part[]);for(const part of resume) void this.load(part)}
  disconnect() {this.hide();super.disconnect()}
}
registerController('fits-catalogue',FitsCatalogueController)
