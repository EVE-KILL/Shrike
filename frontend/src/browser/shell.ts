import {Controller} from '@hotwired/stimulus'
import {SpotlightSearch} from './shared/spotlight'
import {registerController} from './controllers/application'
import {eventElement,formInput,requiredElement} from './shared/dom'
import {errorMessage} from './shared/json'

export class ShellController extends Controller<HTMLBodyElement> {
  static targets=['navbar','content','dialog','mobile','view','searchInput','searchResults']
  declare readonly navbarTarget:HTMLElement
  declare readonly contentTarget:HTMLElement
  declare readonly dialogTarget:HTMLDialogElement
  declare readonly mobileTarget:HTMLDetailsElement
  declare readonly viewTarget:HTMLButtonElement
  declare readonly searchInputTarget:HTMLInputElement
  declare readonly searchResultsTarget:HTMLElement
  private searchUI:SpotlightSearch|undefined
  private get spotlight(){return this.searchUI??=new SpotlightSearch(this.dialogTarget)}
  private readonly killTokens=new WeakMap<HTMLElement,string>()

  connect(){
    let initial='bg2'
    try{
      const saved=/(?:^|;\s*)siteBackground=([^;]+)/.exec(document.cookie)
      const value=decodeURIComponent(saved?.[1]||this.root.dataset.domainBackground||'');initial=/^\/backgrounds\/(bg[1-8])\.webp$/.exec(value)?.[1]||(/^\/images\/domains\/background\/\d+$/.test(value)?value:'bg2')
    }catch{/* A malformed cookie falls back to the default background. */}
    this.element.querySelectorAll<HTMLElement>('[data-background]').forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.background===initial)))
    this.scroll()
  }
  disconnect(){this.cancelSearch();this.searchUI=undefined}
  private get root(){return this.element.ownerDocument.documentElement}
  private closeMenus(){this.element.querySelectorAll<HTMLDetailsElement>('.nav-dropdown[open]').forEach(menu=>{menu.open=false})}
  private restoreView(){
    this.root.classList.remove('background-view')
    this.contentTarget.inert=false;this.navbarTarget.inert=false
    this.viewTarget.setAttribute('aria-pressed','false')
    requiredElement<SVGUseElement>(this.viewTarget,'use').setAttribute('href','/assets/icons.svg#eye')
  }
  openSearch(){
    this.restoreView();this.closeMenus();this.mobileTarget.open=false
    if(!this.dialogTarget.open)this.dialogTarget.showModal()
    this.spotlight.open()
  }
  closeMobile(){
    const wasOpen=this.mobileTarget.open
    this.mobileTarget.open=false
    if(wasOpen)requiredElement<HTMLElement>(this.mobileTarget,'summary').focus()
  }
  toggleMobile(){
    const open=this.mobileTarget.open
    this.root.classList.toggle('mobile-menu-open',open)
    this.contentTarget.inert=open||this.root.classList.contains('background-view')
    requiredElement<HTMLElement>(this.navbarTarget,'.home-link').inert=open
    requiredElement<HTMLElement>(this.navbarTarget,'.search-trigger').inert=open
    if(open)requiredElement<HTMLButtonElement>(this.mobileTarget,'[data-close-mobile]').focus()
  }
  outside(event:MouseEvent){
    const target=event.target
    if(!(target instanceof Node))return
    this.element.querySelectorAll<HTMLDetailsElement>('.nav-dropdown[open]').forEach(menu=>{if(!menu.contains(target))menu.open=false})
  }
  exclusiveMenu(event:Event){
    const menu=event.currentTarget
    if(!(menu instanceof HTMLDetailsElement)||!menu.open)return
    this.element.querySelectorAll<HTMLDetailsElement>('.nav-dropdown[open]').forEach(other=>{if(other!==menu)other.open=false})
  }
  keyboard(event:KeyboardEvent){
    if((event.metaKey||event.ctrlKey)&&event.key==='k'){
      event.preventDefault();event.stopPropagation()
      if(this.dialogTarget.open)this.dialogTarget.close()
      else if(!(event.target instanceof Element&&event.target.closest('input,textarea,[contenteditable=true]')))this.openSearch()
      return
    }
    if(event.key==='Escape'){
      if(this.dialogTarget.open){event.preventDefault();this.dialogTarget.close()}
      const menu=this.element.querySelector<HTMLDetailsElement>('.nav-dropdown[open]')
      const mobileWasOpen=this.mobileTarget.open
      this.closeMenus();this.closeMobile();this.restoreView()
      if(!mobileWasOpen)menu?.querySelector<HTMLElement>('summary')?.focus()
    }
  }
  backdrop(event:MouseEvent){
    if(event.target!==this.dialogTarget)return
    const rect=this.dialogTarget.getBoundingClientRect()
    if(event.clientX<rect.left||event.clientX>rect.right||event.clientY<rect.top||event.clientY>rect.bottom)this.dialogTarget.close()
  }
  scroll(){this.navbarTarget.classList.toggle('scrolled',window.scrollY>1)}
  selectBackground(event:MouseEvent){
    const name=event.currentTarget instanceof HTMLElement?event.currentTarget.dataset.background:undefined
    if(name&&(/^bg[1-8]$/.test(name)||/^\/images\/domains\/background\/\d+$/.test(name)))this.pickBackground(name)
  }
  private pickBackground(name:string){
    const domain=/^\/images\/domains\/background\/\d+$/.test(name)
    if(domain)this.root.style.setProperty('--site-bg',`url(${JSON.stringify(name)})`)
    else this.root.style.setProperty('--site-bg',`image-set(url("/assets/backgrounds/${name}.avif") type("image/avif"), url("/assets/backgrounds/${name}.webp") type("image/webp"))`)
    document.cookie=`siteBackground=${encodeURIComponent(domain?name:`/backgrounds/${name}.webp`)}; Path=/; Max-Age=31536000; SameSite=Lax`
    this.element.querySelectorAll<HTMLElement>('[data-background]').forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.background===name)))
  }
  shuffle(){
    const current=this.element.querySelector<HTMLElement>('[data-background][aria-pressed="true"]')?.dataset.background
    const choices=Array.from({length:8},(_,i)=>`bg${i+1}`).filter(name=>name!==current)
    this.pickBackground(choices[Math.floor(Math.random()*choices.length)]!)
  }
  toggleBackgroundView(){
    const hidden=this.root.classList.toggle('background-view')
    this.closeMenus();this.mobileTarget.open=false
    this.contentTarget.inert=hidden;this.navbarTarget.inert=hidden
    this.viewTarget.setAttribute('aria-pressed',String(hidden))
    requiredElement<SVGUseElement>(this.viewTarget,'use').setAttribute('href',`/assets/icons.svg#${hidden?'eye-off':'eye'}`)
  }
  cancelSearch(){this.searchUI?.close()}
  search(){this.spotlight.search()}
  searchKeys(event:KeyboardEvent){this.spotlight.key(event)}
  resultKeys(event:KeyboardEvent){this.spotlight.key(event)}
  spotlightClick(event:MouseEvent){this.spotlight.click(event)}
  async login(event:SubmitEvent){
    if(!(event.target instanceof HTMLFormElement))return
    event.preventDefault()
    const form=event.target,button=requiredElement<HTMLButtonElement>(form,'button'),error=requiredElement<HTMLElement>(form,'.login-error')
    button.disabled=true;error.textContent='';error.hidden=true
    const params=new URLSearchParams({redirect:location.pathname+location.search+location.hash,charKm:formInput(form,'charKm').checked?'1':'0',corpKm:formInput(form,'corpKm').checked?'1':'0',delay:formInput(form,'delay').value})
    try{
      const response=await fetch(`/auth/login?${params}`)
      if(!response.ok)throw new Error('Login is unavailable. Please try again.')
      const data:{url:string}=await response.json(),destination=new URL(data.url)
      if(destination.protocol!=='https:')throw new Error('Invalid login response.')
      if(this.element.isConnected)location.assign(destination.href)
    }catch(failure){if(form.isConnected){error.textContent=errorMessage(failure,'Login is unavailable. Please try again.');error.hidden=false;button.disabled=false}}
  }
  // Capture before navigation feedback, so it sees the final killmail URL.
  rememberKillNavigation(event:MouseEvent){
    const anchor=eventElement(event.target)?.closest<HTMLAnchorElement>('a[href^="/kill/"]')
    const list=anchor?.closest<HTMLElement>('.kill-list')
    if(!list||!anchor)return
    const ids=[...new Set([...list.querySelectorAll<HTMLElement>('.kill-row[data-kill-id]')].map(row=>Number(row.dataset.killId)).filter(id=>Number.isSafeInteger(id)&&id>0))].slice(0,100)
    if(!ids.length)return
    let token=this.killTokens.get(list)
    if(!token){token=crypto.randomUUID();this.killTokens.set(list,token)}
    const path=location.pathname,label=path==='/advancedsearch'?'Advanced Search':path.startsWith('/battle/')?'Battle results':path==='/'?'Latest kills':'Back to results'
    try{
      sessionStorage.setItem(`evekill-kill-navigation:${token}`,JSON.stringify({version:1,createdAt:Date.now(),returnTo:location.pathname+location.search+location.hash,label,ids}))
      const url=new URL(anchor.href,location.href);url.searchParams.set('nav',token);anchor.href=url.pathname+url.search+url.hash
    }catch{/* A blocked store leaves the ordinary kill link intact. */}
  }
}
registerController('shell',ShellController)
