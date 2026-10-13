import {FragmentSections} from './fragment-sections'
import {registerController} from './application'
import {eventElement} from '../shared/dom'
import {entityIdentity,entityNode as node} from '../shared/entity'
import {preferredDefaultTab} from '../default-tab'
import {readBrowserSession} from '../browser-request'
import {searchID} from '../shared/search-model'

type Navigation = {url:URL;mode:boolean|'replace'}
export class EntityNavigationController extends FragmentSections<'tab'> {
  protected retryAttribute = 'data-entity-navigation-retry'
  private alive = false
  private generation = 0
  private preferenceCanceled = false
  private displayedURL = ''
  private navigation:Navigation|undefined
  private dialog:HTMLDialogElement|undefined
  private get identity() {return entityIdentity(this.element)}
  private get nav() {return this.q<HTMLElement>('.entity-tabs')}
  connect() {
    this.alive = true;this.displayedURL = location.href;this.element.dataset.entityReady = 'false';const generation = ++this.generation;this.reveal()
    void readBrowserSession().catch(()=>null).then(()=>{
      if(!this.alive || generation !== this.generation || this.navigation) return
      if(this.preferenceCanceled) {this.activate();return}
      const {kind,path} = this.identity,preferred = preferredDefaultTab(kind,path,location.href,[...this.nav.querySelectorAll<HTMLAnchorElement>('a')].map(link=>new URL(link.href).pathname),document.cookie)
      if(preferred) void this.navigate({url:preferred,mode:'replace'});else this.activate()
    })
  }
  private activate() {this.element.dataset.entityReady = 'true';this.element.querySelector('.entity-main')?.dispatchEvent(new Event('entity-activate'))}
  private tab(url:URL):string|null {
    const {path} = this.identity;if(url.origin !== location.origin) return null
    if(url.pathname === path) return 'dashboard'
    if(!url.pathname.startsWith(path+'/') || ![...this.nav.querySelectorAll<HTMLAnchorElement>('a')].some(link=>new URL(link.href).pathname === url.pathname)) return null
    const tab = url.pathname.slice(path.length+1);return /^[a-z]+$/.test(tab) ? tab : null
  }
  private async navigate(intent:Navigation) {
    const tab = this.tab(intent.url);if(!this.alive || !tab) return
    this.cancelSection('tab');this.navigation = undefined
    const current = this.q<HTMLElement>('.entity-main');current.querySelector('[data-section-error="tab"]')?.remove()
    if(intent.url.href === this.displayedURL && current.dataset.entityTab === tab) {this.activate();return}
    this.navigation = intent;const {kind,id} = this.identity,params = new URLSearchParams(intent.url.searchParams),previous = current.querySelector<HTMLElement>('#entity-kill-list')
    if(previous && ['combined','kills','losses'].includes(tab)) {params.set('limit',previous.dataset.limit ?? '50');if((params.get('ship_group') ?? '0') === previous.dataset.shipGroup && Number(previous.dataset.totalPages) > 0) params.set('page',previous.dataset.page ?? '1')}
    const query = params.toString()
    const success = await this.section('tab',{container:current,path:`/_fragments/entity/tab/${kind}/${id}/${tab}${query ? '?'+query : ''}`,selector:'.entity-main',replaceRoot:true,errorClass:'widget-error',message:'Could not load tab. Try again.',
      beforeCommit:next=>{if(next.dataset.entityKind !== kind || searchID(next.dataset.entityId) !== id || next.dataset.entityTab !== tab) throw new Error('Invalid entity tab')},
      loaded:next=>{
        this.element.dataset.entityReady = 'true'
        if(intent.mode) history[intent.mode === 'replace' ? 'replaceState' : 'pushState']({...history.state,entityTab:tab},'',intent.url)
        document.title = next.dataset.pageTitle || document.title
        const description = document.querySelector<HTMLMetaElement>('meta[name="description"]');if(description && next.dataset.pageDescription) description.content = next.dataset.pageDescription
        const canonical = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');if(canonical) canonical.href = 'https://eve-kill.com'+intent.url.pathname+intent.url.search
        for(const link of this.nav.querySelectorAll<HTMLAnchorElement>('a')) {const active = new URL(link.href).pathname === intent.url.pathname;link.classList.toggle('active',active);if(active) link.setAttribute('aria-current','page');else link.removeAttribute('aria-current')}
        this.displayedURL = intent.url.href;this.navigation = undefined;this.reveal()
      }})
    if(!success && this.alive) this.activate()
  }
  reveal() {const active = this.nav.querySelector('[aria-current="page"]');if(!active) return;const strip = this.nav.getBoundingClientRect(),tab = active.getBoundingClientRect();if(tab.left < strip.left) this.nav.scrollLeft += tab.left-strip.left;else if(tab.right > strip.right) this.nav.scrollLeft += tab.right-strip.right}
  click(event:MouseEvent) {
    const target = eventElement(event.target);if(!this.alive || !target) return
    if(target.closest('[data-entity-navigation-retry]') && this.navigation && !this.sectionPending('tab')) {void this.navigate(this.navigation);return}
    const portrait = target.closest('[data-entity-portrait]')?.querySelector<HTMLImageElement>('img');if(portrait) {this.openImage(portrait);return}
    const link = target.closest<HTMLAnchorElement>('.entity-tabs a,.entity-ship-classes a[href]');if(!link || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || link.hasAttribute('download') || link.target && link.target !== '_self') return
    const url = new URL(link.href);if(!this.tab(url)) return
    this.preferenceCanceled = true;event.preventDefault();void this.navigate({url,mode:true})
  }
  private openImage(source:HTMLImageElement) {
    this.closeImage();const dialog = node('dialog','entity-image-dialog'),image = node('img'),close = node('button','entity-image-close');dialog.setAttribute('aria-label',source.alt+' full size');image.src = source.src.replace(/size=\d+/,'size=512');image.alt = source.alt;close.type = 'button';close.setAttribute('aria-label','Close full-size image');close.innerHTML = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 6 6 18M6 6l12 12"/></svg>';close.addEventListener('click',()=>dialog.close());dialog.addEventListener('click',event=>{if(event.target === dialog) dialog.close()})
    if(this.element.dataset.oldPortrait === 'true') {const actions = node('div','entity-image-choices'),current = node('button','active','Current'),legacy = node('button','','Legacy'),url = image.src;current.type = legacy.type = 'button';legacy.addEventListener('click',()=>{image.src = '/images/oldcharacters/'+this.identity.id;legacy.classList.add('active');current.classList.remove('active')});current.addEventListener('click',()=>{image.src = url;current.classList.add('active');legacy.classList.remove('active')});actions.append(current,legacy);dialog.append(actions)}
    dialog.append(close,image);dialog.addEventListener('close',()=>{dialog.remove();if(this.dialog === dialog) this.dialog = undefined},{once:true});document.body.append(dialog);this.dialog = dialog;dialog.showModal()
  }
  private closeImage() {this.dialog?.close();this.dialog?.remove();this.dialog = undefined}
  history() {if(this.alive && this.tab(new URL(location.href))) {this.preferenceCanceled = true;void this.navigate({url:new URL(location.href),mode:false})}}
  imageError(event:Event) {const image = eventElement(event.target);if(!(image instanceof HTMLImageElement) || !/^\/images\/(systems|regions|constellations)\//.test(new URL(image.src).pathname)) return;const fallback = node('span','image-fallback','◎');fallback.setAttribute('aria-hidden','true');image.replaceWith(fallback)}
  hide() {this.alive = false;this.generation++;this.closeImage();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;if(this.navigation) void this.navigate(this.navigation);else this.activate();this.reveal()}}
}
registerController('entity-navigation',EntityNavigationController)
