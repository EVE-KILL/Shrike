import {Controller} from '@hotwired/stimulus'
import {readAccountFragment,readBrowserSession,resetBrowserSession,requestJSON,type BrowserUser} from './browser-request'
import {formatWalletBalance,LOGIN_PREFS_DEFAULTS,ALLOWED_DELAYS} from './account-data'
import {setTrackerUser} from './tracker-state'
import './notifications'
import {registerController} from './controllers/application'
import {eventElement,formInput,requiredElement} from './shared/dom'
import {replaceFragmentChildren} from './fragment'
import {isRecord,readCookieJSON,errorMessage} from './shared/json'

export class AccountController extends Controller<HTMLElement> {
  static targets=['loginForm']
  declare readonly loginFormTargets:HTMLFormElement[]
  private prefs={...LOGIN_PREFS_DEFAULTS}
  private user:BrowserUser|null=null
  private wallet:number|null=null
  private walletLoading=false
  private sequence=0
  private generation=0
  private request:AbortController|undefined
  private placeholders=new Map<HTMLElement,Node[]>()

  connect(){
    for(const slot of this.all('[data-account-slot]'))if(!this.placeholders.has(slot))this.placeholders.set(slot,Array.from(slot.childNodes,node=>node.cloneNode(true)))
    const raw=readCookieJSON('ek_login_prefs'),saved=isRecord(raw)?raw:{}
    this.prefs={...LOGIN_PREFS_DEFAULTS,charKm:typeof saved.charKm==='boolean'?saved.charKm:true,
      corpKm:typeof saved.corpKm==='boolean'?saved.corpKm:true,delay:typeof saved.delay==='number'?saved.delay:0}
    void this.loadSession(++this.generation)
  }
  disconnect(){++this.generation;++this.sequence;this.request?.abort();this.user=null;void setTrackerUser(null)}
  private all<T extends HTMLElement=HTMLElement>(selector:string){return this.element.querySelectorAll<T>(selector)}
  private renderLogin(){
    for(const form of this.loginFormTargets){
      formInput(form,'charKm').checked=this.prefs.charKm;formInput(form,'corpKm').checked=this.prefs.corpKm
      formInput(form,'delay').value=String(this.prefs.delay)
      requiredElement<HTMLInputElement>(form,'[data-login-delay]').value=String(Math.max(0,ALLOWED_DELAYS.indexOf(this.prefs.delay)))
      requiredElement<HTMLElement>(form,'[data-login-delay-label]').textContent=this.prefs.delay===0?'No delay':this.prefs.delay===1?'1 hour':this.prefs.delay+' hours'
    }
  }
  preferences(event:Event){
    if(!(event.target instanceof HTMLInputElement))return
    const input=event.target
    if(input.name==='charKm'||input.name==='corpKm')this.prefs[input.name]=input.checked
    else if(input.matches('[data-login-delay]'))this.prefs.delay=ALLOWED_DELAYS[Number(input.value)]??0
    else return
    document.cookie=`ek_login_prefs=${encodeURIComponent(JSON.stringify(this.prefs))}; Max-Age=31536000; Path=/; SameSite=Lax`
    this.renderLogin()
  }
  private renderBalance(){for(const badge of this.all('[data-account-balance]'))badge.textContent=formatWalletBalance(this.wallet,this.walletLoading)}
  async refreshBalance(event?:Event){
    if(event&&(!(event.currentTarget instanceof HTMLDetailsElement)||!event.currentTarget.open))return
    if(!this.user)return
    const user=this.user,sequence=++this.sequence
    this.request?.abort();const request=new AbortController();this.request=request
    this.walletLoading=true;this.renderBalance()
    try{
      const data=await requestJSON<{availableBalance?:number;balance?:number}>('/api/user/wallet/balance',{signal:request.signal})
      if(this.user!==user||sequence!==this.sequence)return
      const value=Number(data.availableBalance??data.balance);this.wallet=Number.isFinite(value)?value:null
    }catch{if(sequence===this.sequence)this.wallet=null}
    finally{if(sequence===this.sequence){this.walletLoading=false;this.renderBalance()}}
  }
  private async loadSession(generation:number){
    const focused=document.activeElement
    const focusedSlot=focused instanceof HTMLElement?focused.closest<HTMLElement>('[data-account-slot]'):null
    for(const slot of this.all('[data-account-slot]')){
      slot.setAttribute('aria-busy','true')
      const button=slot.querySelector<HTMLButtonElement>('[data-account-retry]')
      if(button){button.disabled=true;button.title='Loading account';button.setAttribute('aria-label','Loading account')}
    }
    let user:BrowserUser|null
    try{
      const [fragment,session]=await Promise.all([readAccountFragment(),readBrowserSession()])
      if(generation!==this.generation||!this.element.isConnected)return
      for(const slot of this.all('[data-account-slot]')){
        const content=slot.dataset.accountSlot==='desktop'?fragment.desktop:fragment.mobile
        replaceFragmentChildren(slot,document.importNode(content,true))
        slot.removeAttribute('aria-busy')
      }
      if(focusedSlot&&(document.activeElement===focused||document.activeElement===document.body)){
        const next=focusedSlot.querySelector<HTMLElement>('[data-account-trigger]')??focusedSlot.querySelector<HTMLElement>('button[type=submit],a')
        next?.focus({preventScroll:true})
      }
      user=session
    }catch{
      if(generation!==this.generation||!this.element.isConnected)return
      for(const slot of this.all('[data-account-slot]')){
        slot.removeAttribute('aria-busy')
        const button=slot.querySelector<HTMLButtonElement>('[data-account-retry]')
        if(button){
          button.disabled=false;button.title='Account unavailable. Click to retry.';button.setAttribute('aria-label','Retry loading account')
          const label=button.querySelector('.account-loading-label');if(label)label.textContent='Account unavailable. Retry'
        }
      }
      return
    }
    this.user=user
    this.renderLogin()
    if(!user){void setTrackerUser(null);return}
    if(!readCookieJSON('ek_theme')&&Object.keys(user.settings?.theme||{}).length){
      const {applySettingsTheme,minimalTheme}=await import('./settings-theme')
      if(generation!==this.generation||!this.element.isConnected)return
      const theme=minimalTheme(user.settings?.theme);applySettingsTheme(theme)
      document.cookie=`ek_theme=${encodeURIComponent(JSON.stringify(theme))}; Max-Age=31536000; Path=/; SameSite=Lax`
    }
    void setTrackerUser(user);void this.refreshBalance()
  }
  retry(){void this.loadSession(++this.generation)}
  resume(event:PageTransitionEvent){
    if(!event.persisted)return
    ++this.sequence;this.request?.abort();this.user=null;this.wallet=null;this.walletLoading=false
    void setTrackerUser(null)
    resetBrowserSession()
    for(const [slot,nodes] of this.placeholders)replaceFragmentChildren(slot,...nodes.map(node=>node.cloneNode(true)))
    void this.loadSession(++this.generation)
  }
  async logout(event:MouseEvent){
    if(!(event.currentTarget instanceof HTMLButtonElement))return
    const button=event.currentTarget,error=button.parentElement?.querySelector<HTMLElement>('[data-account-error]')
    button.disabled=true;if(error)error.hidden=true
    try{await requestJSON('/auth/logout',{method:'POST'});resetBrowserSession();void setTrackerUser(null);location.assign('/')}
    catch(failure){if(button.isConnected){if(error){error.textContent=errorMessage(failure,'Logout failed');error.hidden=false}button.disabled=false}}
  }
  closeMenus(event:MouseEvent){if(eventElement(event.target)?.closest('a'))for(const menu of this.all<HTMLDetailsElement>('details[open]'))menu.open=false}
}
registerController('account',AccountController)
