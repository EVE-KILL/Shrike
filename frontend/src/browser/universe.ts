import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import './controllers/item-fits'
import {eventElement} from './shared/dom'
import {isAbortError} from './shared/json'
import {marketID} from './shared/market'
import {replaceFragment} from './fragment'
import {preferredDefaultTab} from './default-tab'
import {readBrowserSession} from './browser-request'
import {KillPages,matchingKillPage} from './shared/kill-paging'

type Navigation = {url:URL;mode:boolean|'replace'}
export class UniverseController extends RequestComponent {
  private alive = false
  private generation = 0
  private preferenceCanceled = false
  private tabRead:AbortController|undefined
  private listRead:AbortController|undefined
  private navigation:Navigation|undefined
  private paging:{page:number;limit:number}|undefined
  private cursors = new KillPages()
  private dialog:HTMLDialogElement|undefined
  private get identity() {const kind = this.element.dataset.universeKind,id = marketID(this.element.dataset.universeId);if(!id || kind !== 'item' && kind !== 'group') throw new Error('Invalid universe component');return {kind,id,base:`/${kind}/${id}`}}
  private get nav() {return this.q<HTMLElement>('.universe-tabs')}
  connect() {
    this.alive = true;const generation = ++this.generation
    // This shared session read refreshes browser preferences; it does not belong
    // to an individual tab. Detached or manually navigated scopes ignore it.
    void readBrowserSession().catch(()=>null).then(()=>{
      if(!this.alive || generation !== this.generation || this.preferenceCanceled || this.navigation) return
      const {kind,base} = this.identity,preferred = preferredDefaultTab(kind,base,location.href,[...this.nav.querySelectorAll<HTMLAnchorElement>('a')].map(link=>new URL(link.href).pathname),document.cookie)
      if(preferred) void this.navigate({url:preferred,mode:'replace'})
    })
  }
  private tab(url:URL):string|null {
    const {base} = this.identity
    if(url.origin !== location.origin) return null
    if(url.pathname === base) return 'dashboard'
    if(!url.pathname.startsWith(base+'/') || ![...this.nav.querySelectorAll<HTMLAnchorElement>('a')].some(link=>new URL(link.href).pathname === url.pathname)) return null
    return url.pathname.slice(base.length+1)
  }
  private showError(root:HTMLElement,kind:'tab'|'kills',message:string) {
    root.querySelector(`[data-universe-error="${kind}"]`)?.remove()
    const box = document.createElement('p'),retry = document.createElement('button');box.className = 'widget-error';box.dataset.universeError = kind;box.setAttribute('role','alert');box.textContent = message+' ';retry.type = 'button';retry.dataset.universeRetry = kind;retry.textContent = 'Retry';box.append(retry);root.prepend(box)
  }
  private async navigate(intent:Navigation) {
    const tab = this.tab(intent.url);if(!this.alive || !tab) return
    const {kind,id} = this.identity,current = this.q<HTMLElement>('.universe-main'),request = new AbortController();this.tabRead?.abort();this.tabRead = request;this.navigation = intent;current.querySelector('[data-universe-error="tab"]')?.remove();current.setAttribute('aria-busy','true')
    try {
      const html = await this.text(`/_fragments/${kind}/tab/${id}/${tab}${intent.url.search}`,'Could not load tab.',{signal:request.signal})
      if(!this.alive || this.tabRead !== request || !current.isConnected) return
      const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('.universe-main')
      if(!next || next.dataset.universeKind !== kind || marketID(next.dataset.universeId) !== id || next.dataset.universeTab !== tab) throw new Error('Invalid universe tab')
      this.listRead?.abort();this.listRead = undefined;this.paging = undefined;this.cursors.clear()
      replaceFragment(current,next)
      if(intent.mode) history[intent.mode === 'replace' ? 'replaceState' : 'pushState']({...history.state,universeTab:tab},'',intent.url)
      document.title = next.dataset.pageTitle || document.title
      const description = document.querySelector<HTMLMetaElement>('meta[name="description"]');if(description && next.dataset.pageDescription) description.content = next.dataset.pageDescription
      const canonical = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');if(canonical) canonical.href = 'https://eve-kill.com'+intent.url.pathname
      for(const link of this.nav.querySelectorAll<HTMLAnchorElement>('a')) {const active = new URL(link.href).pathname === intent.url.pathname;link.classList.toggle('active',active);if(active) link.setAttribute('aria-current','page');else link.removeAttribute('aria-current')}
      this.navigation = undefined
    } catch(error) {if(this.alive && this.tabRead === request && !isAbortError(error) && current.isConnected) this.showError(current,'tab','Could not load tab.')}
    finally {if(this.tabRead === request) {this.tabRead = undefined;current.removeAttribute('aria-busy')}}
  }
  private async loadKills(page:number,limit:number) {
    const current = this.element.querySelector<HTMLElement>('#kill-list');if(!this.alive || !current) return
    const params = this.cursors.params(current,page,limit);if(!params) return
    const {kind,id} = this.identity,request = new AbortController();this.listRead?.abort();this.listRead = request;this.paging = {page,limit};current.querySelector('[data-universe-error="kills"]')?.remove();current.setAttribute('aria-busy','true')
    try {
      const html = await this.text(`/_fragments/${kind}/kills/${id}?${params}`,'Could not load kills.',{signal:request.signal})
      if(!this.alive || this.listRead !== request || !current.isConnected) return
      const next = new DOMParser().parseFromString(html,'text/html').querySelector<HTMLElement>('#kill-list')
      if(!next || !matchingKillPage(next,kind,page,limit)) throw new Error('Invalid kill list')
      replaceFragment(current,next);this.paging = undefined
    } catch(error) {if(this.alive && this.listRead === request && !isAbortError(error) && current.isConnected) this.showError(current,'kills','Could not load kills.')}
    finally {if(this.listRead === request) {this.listRead = undefined;current.removeAttribute('aria-busy')}}
  }
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    const retry = target.closest<HTMLElement>('[data-universe-retry]')
    if(retry?.dataset.universeRetry === 'tab' && this.navigation && !this.tabRead) {void this.navigate(this.navigation);return}
    if(retry?.dataset.universeRetry === 'kills' && this.paging && !this.listRead) {void this.loadKills(this.paging.page,this.paging.limit);return}
    const image = target.closest('[data-item-image]')?.querySelector<HTMLImageElement>('img');if(image) {this.openImage(image);return}
    const list = target.closest<HTMLElement>('#kill-list'),limit = target.closest<HTMLElement>('[data-kill-limit]'),page = target.closest<HTMLButtonElement>('[data-kill-page]')
    if(list && limit) {this.cursors.clear();void this.loadKills(1,Number(limit.dataset.killLimit));return}
    if(list && page && !page.disabled) {void this.loadKills(Number(page.dataset.killPage),Number(list.dataset.limit));return}
    const link = target.closest<HTMLAnchorElement>('a[href]');if(!link || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || link.hasAttribute('download') || link.target && link.target !== '_self') return
    const url = new URL(link.href);if(!this.tab(url)) return
    this.preferenceCanceled = true;event.preventDefault();if(url.href !== location.href) void this.navigate({url,mode:true})
  }
  private openImage(source:HTMLImageElement) {
    this.closeImage();const dialog = document.createElement('dialog'),image = document.createElement('img'),close = document.createElement('button');dialog.className = 'item-image-dialog';dialog.setAttribute('aria-label',source.alt+' full size');image.src = source.src.replace(/size=\d+/,'size=512');image.alt = source.alt;image.width = image.height = 512;close.type = 'button';close.textContent = 'Close';close.addEventListener('click',()=>dialog.close());dialog.append(image,close);dialog.addEventListener('close',()=>{dialog.remove();if(this.dialog === dialog) this.dialog = undefined},{once:true});document.body.append(dialog);this.dialog = dialog;dialog.showModal()
  }
  private closeImage() {this.dialog?.close();this.dialog?.remove();this.dialog = undefined}
  history() {if(this.alive && this.tab(new URL(location.href))) {this.preferenceCanceled = true;void this.navigate({url:new URL(location.href),mode:false})}}
  hide() {this.alive = false;this.generation++;this.tabRead?.abort();this.tabRead = undefined;this.listRead?.abort();this.listRead = undefined;this.element.querySelectorAll('.universe-main,#kill-list').forEach(node=>node.removeAttribute('aria-busy'));this.closeImage();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.navigation) void this.navigate(this.navigation);else if(this.paging) void this.loadKills(this.paging.page,this.paging.limit)}}
}
registerController('universe',UniverseController)
