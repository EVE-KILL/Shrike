import {RequestComponent} from './request-component'
import {registerController} from './application'
import {eventElement} from '../shared/dom'
import {entityIdentity,entityNode as node,entityISK,readEntityValuable,readEntityShipClasses,isEntityCategory,type EntityCategory} from '../shared/entity'
import {formatNumber} from '../../utils/formatters'
import {isAbortError} from '../shared/json'
import {moveTabFocus} from '../shared/tabs'

export class EntityValuableController extends RequestComponent {
  private alive = false
  private read:AbortController|undefined
  private intent:EntityCategory|undefined
  connect() {this.alive = true;if(this.intent) void this.load(this.intent);else if(!this.element.dataset.loaded) void this.load('kills')}
  private async load(category:EntityCategory) {
    if(!this.alive) return
    this.read?.abort();const request = new AbortController();this.read = request;this.intent = category;const content = this.q<HTMLElement>('[data-entity-valuable-content]')
    content.querySelector('[data-entity-valuable-retry]')?.parentElement?.remove();if(!this.element.dataset.loaded) content.replaceChildren(node('p','','Loading valuable kills…'))
    content.setAttribute('aria-busy','true')
    try {
      const {api} = entityIdentity(this.element),params = new URLSearchParams({dataType:'most_valuable_'+category,limit:'8',days:'7'}),entries = readEntityValuable(await this.request<unknown>(api+'/most-valuable?'+params,{signal:request.signal}))
      if(!this.alive || this.read !== request) return
      if(!entries.length) content.replaceChildren(node('p','','Nothing in the last 7d'))
      else {const grid = node('div','valuable-grid');for(const [index,kill] of entries.entries()) {const card = node('a','valuable-card'+(index >= 6 ? ' valuable-card-secondary' : '')),image = node('img','valuable-ship'),labels = node('span','valuable-labels');card.href = '/kill/'+kill.killmail_id;image.src = `/images/types/${kill.ship_type_id}/overlayrender?size=256`;image.alt = kill.ship_name || 'Ship';image.loading = index < 6 ? 'eager' : 'lazy';image.width = image.height = 256;labels.append(node('strong','',kill.ship_name || 'Unknown'));if(kill.victim_character_name) labels.append(node('span','',kill.victim_character_name));if(kill.victim_corporation_name) labels.append(node('span','',kill.victim_corporation_name));card.append(image,node('span','valuable-shade'),node('span','valuable-isk',entityISK(kill.total_value)),labels);grid.append(card)}content.replaceChildren(grid)}
      content.setAttribute('aria-labelledby','entity-valuable-tab-'+category)
      for(const button of this.all<HTMLElement>('[data-entity-valuable]')) {const active = button.dataset.entityValuable === category;button.classList.toggle('active',active);button.setAttribute('aria-selected',String(active));button.tabIndex = active ? 0 : -1}
      this.element.dataset.loaded = category;this.intent = undefined
    }catch(error) {if(this.alive && this.read === request && !isAbortError(error)) {const notice = node('p','','Could not load valuable kills. '),retry = node('button','','Retry');retry.type = 'button';retry.dataset.entityValuableRetry = '';notice.append(retry);content.append(notice)}}
    finally {if(this.read === request) {this.read = undefined;content.removeAttribute('aria-busy')}}
  }
  keydown(event:KeyboardEvent) {moveTabFocus(event)}
  click(event:MouseEvent) {const target = eventElement(event.target);if(!this.alive || !target) return;const category = target.closest<HTMLElement>('[data-entity-valuable]')?.dataset.entityValuable;if(isEntityCategory(category)) void this.load(category);else if(target.closest('[data-entity-valuable-retry]') && this.intent && !this.read) void this.load(this.intent)}
  hide() {this.alive = false;this.read?.abort();this.read = undefined;this.element.querySelector('[data-entity-valuable-content]')?.removeAttribute('aria-busy');super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.intent) void this.load(this.intent)}}
}

export class EntityShipClassesController extends RequestComponent {
  private alive = false
  private read:AbortController|undefined
  private loaded = false
  connect() {this.alive = true;if(!this.loaded) void this.load()}
  async load() {
    if(!this.alive || this.read) return
    const request = new AbortController();this.read = request;this.element.querySelector('[data-entity-classes-error]')?.remove()
    try {
      const {api,kind,id} = entityIdentity(this.element),groups = readEntityShipClasses(await this.request<unknown>(api+'/ship-classes',{signal:request.signal}));if(!this.alive || this.read !== request) return
      if(!groups.length) {this.element.remove();return}
      const top = groups[0];this.q('[data-entity-ship-summary]').textContent = `${groups.length} hull groups · top: ${top.group_name} ${formatNumber(top.losses)} lost`
      const list = node('div','entity-ship-grid');for(const group of groups.sort((a,b)=>a.group_name.localeCompare(b.group_name))) {const row = node('div','entity-ship-row'),name = node('a','',group.group_name),losses = node('a','',formatNumber(group.losses));name.href = '/group/'+group.group_id;losses.href = `/${kind}/${id}/losses?ship_group=${group.group_id}`;row.append(name,losses);list.append(row)}
      const legend = node('div','entity-ship-legend'),lost = node('span','','lost → /losses'),marker = node('i');marker.setAttribute('aria-hidden','true');lost.prepend(marker);legend.append(lost,node('span','','class → group · losses → filtered losses'));this.q('[data-entity-ship-content]').replaceChildren(list,legend);this.loaded = true
    }catch(error) {if(this.alive && this.read === request && !isAbortError(error)) {this.q('[data-entity-ship-summary]').textContent = 'Could not load ship classes';const retry = node('button','','Retry');retry.type = 'button';retry.dataset.entityClassesError = '';retry.dataset.action = 'click->entity-classes#load';this.q('[data-entity-ship-content]').append(retry)}}
    finally {if(this.read === request) this.read = undefined}
  }
  hide() {this.alive = false;this.read?.abort();this.read = undefined;super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(!this.loaded) void this.load()}}
}
registerController('entity-valuable',EntityValuableController)
registerController('entity-classes',EntityShipClassesController)
