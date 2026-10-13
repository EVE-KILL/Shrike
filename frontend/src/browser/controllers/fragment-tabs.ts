import {RequestComponent} from './request-component'
import {BrowserRequestError} from '../browser-request'
import {accountLoginDestination} from '../shared/account-actions'
import {eventElement} from '../shared/dom'
import {escapeHTML} from '../shared/html'
import {errorMessage,isAbortError} from '../shared/json'

// Both private page families retain their panels and drafts in the same document.
export abstract class FragmentTabsController extends RequestComponent {
  protected abstract readonly kind:'settings'|'admin'
  protected active='overview'
  private initialized=new WeakSet<HTMLElement>()
  private navigation:AbortController|undefined
  private sectionRequests=new Map<HTMLElement,AbortController>()
  protected get panels(){return this.element.querySelector<HTMLElement>(`[data-${this.kind}-panels]`)}
  protected abstract preparePanel(panel:HTMLElement):void
  protected beforeDiscard(_panel:HTMLElement){}
  protected fragmentQuery(_id:string){return ''}
  connect(){
    this.active=this.element.dataset.active??'overview'
    for(const panel of this.panels?.children??[])if(panel instanceof HTMLElement)this.prepareOnce(panel)
  }
  private prepareOnce(panel:HTMLElement){if(!this.initialized.has(panel)){this.initialized.add(panel);this.preparePanel(panel)}}
  private setActive(id:string){
    this.active=id;this.element.dataset.active=id
    for(const panel of this.panels?.children??[]){
      if(!(panel instanceof HTMLElement))continue
      const visible=panel.dataset[`${this.kind}Section`]===id,changed=panel.hidden===visible
      panel.hidden=!visible
      if(changed)panel.dispatchEvent(new CustomEvent(`${this.kind}-visibility`,{detail:visible}))
    }
    let label='Overview'
    for(const link of this.all<HTMLElement>(`[data-${this.kind}-tab]`)){
      const selected=link.dataset[`${this.kind}Tab`]===id
      link.setAttribute('aria-current',selected?'page':'false')
      // Attention shortcuts keep their own card style; the shared nav uses this class.
      if(this.kind==='settings'||link.closest('.admin-nav'))link.className='flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors text-left cursor-pointer whitespace-nowrap flex-shrink-0 '+(selected?'bg-blue-500/10 text-blue-400':'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]')
      if(selected&&(this.kind==='settings'||link.closest('.admin-nav')))label=link.dataset.label??link.innerText.trim()
    }
    document.title=`${label} — ${this.kind==='settings'?'Settings':'Admin'} — EVE-KILL`
  }
  protected async navigate(requested:string|undefined,push=true,force=false){
    const panels=this.panels;if(!panels)return
    const tabSelector=this.kind==='admin'?'.admin-nav [data-admin-tab]':'[data-settings-tab]'
    const allowed=new Set([...this.all<HTMLElement>(tabSelector)].map(link=>link.dataset[`${this.kind}Tab`]))
    const id=requested&&allowed.has(requested)?requested:'overview'
    this.navigation?.abort();this.navigation=undefined
    if(push)history.pushState(history.state,'',id==='overview'?`/${this.kind}`:`/${this.kind}/${id}`)
    let panel=panels.querySelector<HTMLElement>(`[data-${this.kind}-section="${id}"]`)
    if(panel&&!force&&panel.getAttribute('aria-busy')!=='true'){this.setActive(id);this.prepareOnce(panel);return}
    const controller=new AbortController();this.navigation=controller
    if(!panel){panel=document.createElement('section');panel.dataset[`${this.kind}Section`]=id;panels.append(panel)}
    const placeholder=panel;this.beforeDiscard(placeholder)
    this.sectionRequests.set(placeholder,controller);this.setActive(id);placeholder.setAttribute('aria-busy','true')
    placeholder.innerHTML='<div class="glass-panel p-8 text-sm text-gray-500">Loading…</div>'
    const fallback=`Could not load this ${this.kind} section.`
    try{
      const text=await this.text(`/_fragments/${this.kind}/${id}`+this.fragmentQuery(id),fallback,{signal:controller.signal})
      const next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>(`[data-${this.kind}-section]`)
      if(!next||next.dataset[`${this.kind}Section`]!==id)throw new Error(fallback)
      if(this.navigation!==controller)return
      const replacement=document.importNode(next,true);placeholder.replaceWith(replacement);this.prepareOnce(replacement)
    }catch(error){
      const message=this.kind==='admin'&&error instanceof BrowserRequestError&&error.status===403?'Admin access required.':errorMessage(error,fallback)
      if(!isAbortError(error)&&this.navigation===controller&&placeholder.isConnected)placeholder.innerHTML=`<div class="glass-panel p-5 text-sm text-red-300" role="alert">${escapeHTML(message)} <button type="button" data-${this.kind}-section-retry class="text-blue-400">Retry</button></div>`
      else if(isAbortError(error)&&this.sectionRequests.get(placeholder)===controller)placeholder.remove()
    }finally{
      if(this.navigation===controller)this.navigation=undefined
      if(this.sectionRequests.get(placeholder)===controller){this.sectionRequests.delete(placeholder);placeholder.removeAttribute('aria-busy')}
    }
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target),link=target?.closest<HTMLAnchorElement>(`[data-${this.kind}-tab]`)
    if(link&&!event.metaKey&&!event.ctrlKey&&!event.shiftKey&&!event.altKey&&event.button===0){event.preventDefault();await this.navigate(link.dataset[`${this.kind}Tab`]);return}
    if(target?.closest(`[data-${this.kind}-section-retry]`)){await this.navigate(this.active,false,true);return}
    if(target?.closest('[data-utility-login]')){
      try{const destination=await accountLoginDestination(location.pathname+location.search);if(this.element.isConnected)location.assign(destination)}catch(error){this.error('[data-utility-login-error]',error)}
    }
  }
  restore(){void this.navigate(location.pathname.split('/')[2]||'overview',false)}
  hide(){this.navigation?.abort();this.navigation=undefined;super.hide()}
  show(event:PageTransitionEvent){
    if(event.persisted&&this.panels){const panel=this.panels.querySelector<HTMLElement>(`[data-${this.kind}-section="${this.active}"]`);if(!panel||panel.getAttribute('aria-busy')==='true')void this.navigate(this.active,false,true)}
  }
}
