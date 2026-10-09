import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement,requiredElement,placeElement} from './shared/dom'
import {isAbortError} from './shared/json'
import {escapeHTML} from './shared/html'
import {safeID,element as node} from './kill-row'
import {readKillNavigation} from '../utils/killNavigation'
import {readEditorFit,killFitDocument,readMatchup} from './shared/kill-detail'
import {encodeFitText} from '../fit/codec'

type Sort = {col:'name'|'dropped'|'destroyed'|'value';dir:'asc'|'desc'}
const isSort = (value:unknown):value is Sort['col']=>value === 'name' || value === 'dropped' || value === 'destroyed' || value === 'value'

export class KillDetailController extends RequestComponent {
  private alive = false
  private visibleAttackers = 10
  private sort:Sort|undefined
  private showTipTimer:ReturnType<typeof setTimeout>|undefined
  private hideTipTimer:ReturnType<typeof setTimeout>|undefined
  private fitting:AbortController|undefined
  private originalFitButton:string|undefined
  private fitErrorTimer:ReturnType<typeof setTimeout>|undefined
  private matchup:AbortController|undefined
  private matchupLoaded = false
  private get id() {const id = safeID(this.element.dataset.killId);if(!id) throw new Error('Invalid killmail');return id}
  connect() {this.alive = true;this.renderContext();this.imageFallbacks();this.syncTab();if(!this.matchupLoaded) void this.loadMatchup()}
  private renderContext() {
    const nav = this.element.querySelector<HTMLElement>('[data-kill-context]'),token = new URLSearchParams(location.search).get('nav');if(!nav || !token) return
    nav.hidden = false;nav.replaceChildren();const context = readKillNavigation(token),id = this.id
    if(!context || !context.ids.includes(id)) {const notice = node('p','','This browsing context has expired or is unavailable. '),link = node('a','','View direct killmail link');link.href = '/kill/'+id;notice.append(link);nav.append(notice);return}
    const index = context.ids.indexOf(id),previous = context.ids[index-1],next = context.ids[index+1],group = node('div'),back = node('a','','← '+context.label),pager = node('div');back.href = context.returnTo
    for(const [position,[target,label]] of ([[previous,'‹ Previous'],[next,'Next ›']] as const).entries()) {if(target) {const link = node('a','',label);link.href = `/kill/${target}?nav=${encodeURIComponent(token)}`;pager.append(link)}else {const disabled = node('span','',position === 0 ? 'Previous' : 'Next');disabled.setAttribute('aria-disabled','true');pager.append(disabled)}if(position === 0) pager.append(node('span','',`${index+1} / ${context.ids.length} on this page`))}
    group.append(back,pager);nav.append(group);if(!previous || !next) nav.append(node('p','',`${!previous ? 'Start' : 'End'} of this results page. Return to the list to browse another page.`))
  }
  imageError(event:Event) {const image = eventElement(event.target);if(image instanceof HTMLImageElement) this.fallback(image)}
  private fallback(image:HTMLImageElement) {const icon = image.dataset.killImageFallback;if(icon) {image.removeAttribute('data-kill-image-fallback');if(image.getAttribute('src') !== icon) image.src = icon}}
  private imageFallbacks() {for(const image of this.all<HTMLImageElement>('img[data-kill-image-fallback]')) if(image.complete && !image.naturalWidth) this.fallback(image)}
  closeMenus(event:Event) {
    if(event instanceof KeyboardEvent && event.key !== 'Escape') return
    for(const menu of this.all<HTMLDetailsElement>('.kill-dropdown[open]')) {if(event instanceof PointerEvent && event.target instanceof Node && menu.contains(event.target)) continue;const focused = event instanceof KeyboardEvent && menu.contains(document.activeElement);menu.open = false;if(focused) menu.querySelector<HTMLElement>('summary')?.focus({preventScroll:true})}
  }
  private more(button:HTMLButtonElement) {this.visibleAttackers += 25;const rows = [...this.all<HTMLElement>('[data-kill-attacker-index]')];for(const row of rows) row.classList.toggle('extra',Number(row.dataset.killAttackerIndex) >= this.visibleAttackers);const remaining = Math.max(0,rows.length-this.visibleAttackers);if(remaining) button.textContent = `Show more (${remaining} remaining)`;else button.hidden = true}
  private sortItems(col:Sort['col'],button:HTMLElement) {
    const sort:Sort = this.sort?.col === col ? {col,dir:this.sort.dir === 'asc' ? 'desc' : 'asc'} : {col,dir:col === 'name' ? 'asc' : 'desc'};this.sort = sort
    for(const other of this.all<HTMLElement>('[data-kill-sort]')) {requiredElement(other,'[data-sort-arrow]').textContent = other === button ? (sort.dir === 'asc' ? '↑' : '↓') : '';other.setAttribute('aria-sort',other === button ? (sort.dir === 'asc' ? 'ascending' : 'descending') : 'none')}
    for(const group of this.all<HTMLElement>('.kill-group-rows')) {const blocks:HTMLElement[][] = [];for(const child of group.children) if(child instanceof HTMLElement) {if(!child.classList.contains('child')) blocks.push([child]);else blocks.at(-1)?.push(child)}blocks.sort((a,b)=>{const av = a[0].dataset[col] ?? '',bv = b[0].dataset[col] ?? '',result = col === 'name' ? av.localeCompare(bv) : Number(av)-Number(bv);return sort.dir === 'asc' ? result : -result});let before:ChildNode|null = group.firstChild;for(const block of blocks) for(const row of block) {if(row !== before) placeElement(group,row,before);before = row.nextSibling}}
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    const more = target.closest<HTMLButtonElement>('[data-kill-more]');if(more) {this.more(more);return}
    const collapse = target.closest<HTMLElement>('[data-kill-collapse]');if(collapse) {const rows = collapse.closest('[data-kill-slot]')?.querySelector<HTMLElement>('.kill-group-rows');if(rows) {rows.hidden = !rows.hidden;collapse.setAttribute('aria-expanded',String(!rows.hidden));collapse.querySelector<HTMLElement>('.icon')?.style.setProperty('transform',rows.hidden ? 'rotate(-90deg)' : 'none')}return}
    const sort = target.closest<HTMLElement>('[data-kill-sort]');if(sort && isSort(sort.dataset.killSort)) {this.sortItems(sort.dataset.killSort,sort);return}
    const tab = target.closest<HTMLElement>('[data-kill-tab]');if(tab) {const hash = tab.dataset.killTab === 'kill' ? '' : '#tools';history.pushState(history.state,'',location.pathname+location.search+hash);this.syncTab();return}
    if(target.closest('[data-kill-load-fit]')) void this.loadFit()
  }
  syncTab() {const tab = location.hash === '#tools' ? 'tools' : 'kill';this.element.dataset.mobileTab = tab;for(const button of this.all<HTMLElement>('[data-kill-tab]')) button.classList.toggle('active',button.dataset.killTab === tab);const tools = this.element.querySelector<HTMLElement>('[data-kill-tools]');if(tools) tools.hidden = tab !== 'tools'}
  private hideTip() {this.stopTimer(this.showTipTimer);this.showTipTimer = undefined;this.stopTimer(this.hideTipTimer);this.hideTipTimer = undefined;const tip = this.element.querySelector<HTMLElement>('[data-kill-wheel-tooltip]');if(tip) tip.hidden = true;this.element.querySelector('.kill-wheel-background')?.classList.remove('dimmed')}
  hover(event:MouseEvent) {
    const module = eventElement(event.target)?.closest<HTMLElement>('[data-kill-wheel-module]');if(!this.alive || !module) return
    const tip = this.element.querySelector<HTMLElement>('[data-kill-wheel-tooltip]');if(!tip) return;this.stopTimer(this.showTipTimer);this.stopTimer(this.hideTipTimer)
    this.showTipTimer = this.after(tip.hidden ? 250 : 0,()=>{this.showTipTimer = undefined;if(!this.alive || !module.isConnected) return;requiredElement(tip,'[data-tip-name]').textContent = module.dataset.name || 'Module';const ammo = requiredElement<HTMLElement>(tip,'[data-tip-ammo]');ammo.textContent = module.dataset.ammo ?? '';ammo.hidden = !ammo.textContent;const dropped = Number(module.dataset.dropped) || 0,destroyed = Number(module.dataset.destroyed) || 0,quantities = requiredElement(tip,'[data-tip-quantities]');quantities.replaceChildren();if(dropped) quantities.append(node('span','green',`+${dropped} dropped`));if(dropped && destroyed) quantities.append(' · ');if(destroyed) quantities.append(node('span','red',`-${destroyed} destroyed`));tip.hidden = false;this.element.querySelector('.kill-wheel-background')?.classList.add('dimmed')})
  }
  leave(event:MouseEvent) {const target = eventElement(event.target),wheel = target?.closest('[data-kill-wheel]'),module = target?.closest('[data-kill-wheel-module]');if(!wheel || (event.relatedTarget instanceof Node && (module ? module.contains(event.relatedTarget) : wheel.contains(event.relatedTarget)))) return;this.stopTimer(this.showTipTimer);this.stopTimer(this.hideTipTimer);this.hideTipTimer = this.after(150,()=>{this.hideTipTimer = undefined;this.hideTip()})}
  openFit(url:string) {location.assign(url)}
  private resetFitButton() {const button = this.element.querySelector<HTMLButtonElement>('[data-kill-load-fit]');if(button && this.originalFitButton !== undefined) {button.innerHTML = this.originalFitButton;button.disabled = false};this.originalFitButton = undefined}
  private async loadFit() {
    const button = this.element.querySelector<HTMLButtonElement>('[data-kill-load-fit]');if(!this.alive || !button || button.disabled || this.fitting) return
    const request = new AbortController();this.fitting = request;this.originalFitButton = button.innerHTML;button.disabled = true;button.textContent = 'Loading…'
    try {const fit = readEditorFit(await this.request<unknown>(`/api/killmail/${this.id}/fit`,{signal:request.signal},'Could not load fitting')),encoded = await encodeFitText(killFitDocument(fit,this.id));if(this.alive && this.fitting === request && !request.signal.aborted) this.openFit('/fits/create?fit='+encodeURIComponent(encoded))}
    catch(error) {if(this.alive && this.fitting === request && !isAbortError(error)) {button.textContent = error instanceof Error ? error.message : 'Could not load fitting';this.fitErrorTimer = this.after(2200,()=>{this.fitErrorTimer = undefined;this.resetFitButton()})}}
    finally {if(this.fitting === request) this.fitting = undefined}
  }
  private async loadMatchup() {
    const root = this.element.querySelector<HTMLElement>('[data-kill-matchup]');if(!this.alive || !root || this.matchup) return
    const attacker = safeID(root.dataset.attacker),victim = safeID(root.dataset.victim);if(!attacker || !victim) return
    const request = new AbortController();this.matchup = request
    try {
      const data = readMatchup(await this.request<unknown>(`/api/matchup?attacker=${attacker}&victim=${victim}`,{signal:request.signal}));if(!this.alive || this.matchup !== request) return;this.matchupLoaded = true;if(!data.enough) return
      const esc = escapeHTML,attackerName = esc(root.dataset.attackerName || 'Attacker'),victimName = esc(root.dataset.victimName || 'Victim'),sample = data.sample,rate = data.attacker_win_rate
      const stats = data.mirror ? `<p class="kill-matchup-mirror">Mirror matchup — 50% by definition over ${sample} solo fight${sample === 1 ? '' : 's'}.</p>` : `<div class="kill-matchup-rate"><span>${attackerName} wins ${rate}%</span><small>${data.attacker_wins}–${data.victim_wins}</small></div><div class="kill-matchup-bar"><i style="width:${rate}%"></i></div><p>${sample} solo fight${sample === 1 ? '' : 's'} (${attackerName} vs ${victimName})</p>`
      const fits = data.top_fits.map(fit=>{const modules = [...new Map(fit.modules.map(module=>[module.type_id,module])).values()],icons = modules.slice(0,7).map(module=>`<img src="/images/types/${module.type_id}/icon?size=32" alt="${esc(module.name)}" title="${esc(module.name)}" loading="lazy">`).join('');return `<div class="kill-matchup-fit"><span>${icons}${modules.length > 7 ? `<small>+${modules.length-7}</small>` : ''}</span><small>${fit.uses} · ${fit.pct}%</small></div>`}).join('')
      root.innerHTML = `<div class="kill-matchup glass-panel"><header>⚔ Solo matchup <small>last ${data.window_days}d</small></header><div class="kill-matchup-hulls"><a href="/item/${attacker}"><img src="/images/types/${attacker}/icon?size=64" alt="">${attackerName}</a><span>vs</span><a href="/item/${victim}">${victimName}<img src="/images/types/${victim}/icon?size=64" alt=""></a></div>${stats}${fits ? `<div class="kill-matchup-fits"><h4>${victimName} fits that die to ${attackerName}</h4>${fits}</div>` : ''}<p class="kill-matchup-foot">From solo killmails only — disengages and bait aren't counted, and only the loser's fit is known.</p></div>`
    }catch{/* Optional matchup stays absent on failed or canceled reads. */}
    finally {if(this.matchup === request) this.matchup = undefined}
  }
  hide() {this.alive = false;this.hideTip();this.stopTimer(this.fitErrorTimer);this.fitErrorTimer = undefined;this.fitting?.abort();this.fitting = undefined;this.matchup?.abort();this.matchup = undefined;this.resetFitButton();this.closeMenus(new KeyboardEvent('keydown',{key:'Escape'}));super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.renderContext();this.syncTab();if(!this.matchupLoaded) void this.loadMatchup()}}
}
registerController('kill-detail',KillDetailController)
