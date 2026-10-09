import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'
import {isAbortError} from './shared/json'
import {replaceFragmentChildren} from './fragment'

const entities = ['character','corporation','alliance','ship','system','region'] as const
const windows = ['hourly','weekly','fourteen_days','thirty_days','ninety_days','one_eighty_days','one_year','all_time'] as const
type Entity = typeof entities[number]
type Window = typeof windows[number]
const isEntity = (value:unknown):value is Entity=>entities.some(entity=>entity === value)
const isWindow = (value:unknown):value is Window=>windows.some(window=>window === value)
export class RankingsController extends RequestComponent {
  private alive = false
  private entity:Entity = 'character'
  private period:Window = 'all_time'
  private read:AbortController|undefined
  private poll:ReturnType<typeof setTimeout>|undefined
  private dirty = false
  private pendingHistory = false
  private nextPollAt = 0
  private get results() {return this.q<HTMLElement>('[data-ranking-results]')}
  connect() {this.alive = true;this.selection();this.buttons();if(!this.nextPollAt) this.nextPollAt = Date.now()+60000;this.schedule()}
  private selection() {const params = new URLSearchParams(location.search),entity = params.get('entity'),period = params.get('window');this.entity = isEntity(entity) ? entity : 'character';this.period = isWindow(period) ? period : 'all_time'}
  private buttons() {for(const button of this.all<HTMLButtonElement>('[data-ranking-entity],[data-ranking-window]')) {const selected = button.dataset.rankingEntity === this.entity || button.dataset.rankingWindow === this.period;button.classList.toggle('active',selected);button.setAttribute('aria-pressed',String(selected))}}
  private schedule() {
    this.stopTimer(this.poll);this.poll = undefined
    if(!this.alive || this.period !== 'hourly' || document.hidden || this.read) return
    this.poll = this.after(Math.max(1,this.nextPollAt-Date.now()),()=>{this.poll = undefined;if(this.alive && !document.hidden) void this.load(false,true)})
  }
  private async load(updateHistory = true,silent = false) {
    if(!this.alive) return
    this.read?.abort();this.stopTimer(this.poll);this.poll = undefined
    const request = new AbortController(),entity = this.entity,period = this.period;this.read = request;this.dirty = true;this.pendingHistory = updateHistory
    this.results.querySelector('[data-ranking-request-error]')?.remove();this.results.classList.add('rankings-loading');this.results.setAttribute('aria-busy','true');this.element.toggleAttribute('data-loading-silent',silent);this.results.toggleAttribute('data-loading-silent',silent)
    try {
      const html = await this.text('/_fragments/rankings?'+new URLSearchParams({entity,window:period}),'Could not load rankings.',{signal:request.signal,headers:{Accept:'text/html'}})
      if(!this.alive || this.read !== request) return
      const doc = new DOMParser().parseFromString(html,'text/html'),table = doc.querySelector<HTMLElement>('.ranking-table')
      if(!table || table.dataset.rankingResultEntity !== entity || table.dataset.rankingResultWindow !== period) throw new Error('Could not read rankings.')
      replaceFragmentChildren(this.results,...doc.body.childNodes);this.dirty = false
      if(updateHistory) {const page = new URL(location.href);page.searchParams.set('entity',entity);page.searchParams.set('window',period);history.replaceState(history.state,'',page)}
    } catch(error) {
      if(this.alive && this.read === request && !isAbortError(error)) {this.dirty = false;const panel = document.createElement('p'),retry = document.createElement('button');panel.className = 'ranking-empty';panel.dataset.rankingRequestError = '';panel.setAttribute('role','alert');panel.textContent = 'Could not load rankings. ';retry.type = 'button';retry.dataset.rankingRetry = '';retry.textContent = 'Retry';panel.append(retry);this.results.prepend(panel)}
    } finally {if(this.read === request) {this.read = undefined;this.clearBusy();this.nextPollAt = Date.now()+60000;this.schedule()}}
  }
  private clearBusy() {this.results.classList.remove('rankings-loading');this.results.removeAttribute('aria-busy');this.results.removeAttribute('data-loading-silent');this.element.removeAttribute('data-loading-silent')}
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!target || !this.alive) return
    if(target.closest('[data-ranking-retry]')) {if(!this.read) void this.load();return}
    const button = target.closest<HTMLElement>('[data-ranking-entity],[data-ranking-window]');if(!button) return
    if(isEntity(button.dataset.rankingEntity)) this.entity = button.dataset.rankingEntity
    else if(isWindow(button.dataset.rankingWindow)) this.period = button.dataset.rankingWindow
    else return
    this.buttons();void this.load()
  }
  history() {this.selection();this.buttons();void this.load(false)}
  visibility() {if(document.hidden) {this.stopTimer(this.poll);this.poll = undefined}else if(this.alive && this.period === 'hourly' && !this.read) {if(Date.now() >= this.nextPollAt) void this.load(false,true);else this.schedule()}}
  hide() {this.alive = false;this.read?.abort();this.read = undefined;this.stopTimer(this.poll);this.poll = undefined;this.clearBusy();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.buttons();if(this.dirty) void this.load(this.pendingHistory);else this.schedule()}}
}
registerController('rankings',RankingsController)
