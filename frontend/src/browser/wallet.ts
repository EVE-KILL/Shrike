import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'

export class WalletController extends Controller<HTMLElement>{
  private request:AbortController|undefined
  private get content(){return requiredElement<HTMLElement>(this.element,'[data-wallet-content]')}
  connect(){this.refreshDates()}
  disconnect(){this.cancel()}
  private cancel(){this.request?.abort();this.request=undefined;this.content.classList.remove('wallet-loading');this.content.removeAttribute('aria-busy')}
  private refreshDates(){
    const formatter=new Intl.DateTimeFormat('en-GB',{dateStyle:'medium',timeStyle:'short'})
    for(const element of this.content.querySelectorAll<HTMLTimeElement>('time[datetime]')){
      const date=new Date(element.dateTime);if(!Number.isNaN(date.getTime()))element.textContent=formatter.format(date)
    }
  }
  private async load(target:string|URL,historyMode:'push'|'none'='push'){
    this.request?.abort();const pending=new AbortController();this.request=pending
    const url=new URL(target,location.href),fragment=new URL('/_fragments/wallet',location.href);fragment.search=url.search
    const content=this.content;content.classList.add('wallet-loading');content.setAttribute('aria-busy','true');content.querySelector('[data-wallet-error]')?.remove()
    try{
      const response=await fetch(fragment,{signal:pending.signal,headers:{Accept:'text/html'}})
      if(!response.ok)throw new Error('Unable to load wallet journal')
      const html=await response.text();if(this.request!==pending||!this.element.isConnected)return
      content.innerHTML=html;this.refreshDates();if(historyMode==='push')history.pushState({},'',url)
    }catch{
      if(this.request===pending&&this.element.isConnected)content.insertAdjacentHTML('afterbegin','<p data-wallet-error class="wallet-empty" role="alert">Could not load the corporation wallet. Please try again.</p>')
    }finally{if(this.request===pending){this.request=undefined;content.classList.remove('wallet-loading');content.removeAttribute('aria-busy')}}
  }
  change(event:Event){
    const select=event.target;if(!(select instanceof HTMLSelectElement)||!select.matches('[data-wallet-division]'))return
    const url=new URL('/wallet',location.href);if(select.value)url.searchParams.set('division',select.value);void this.load(url)
  }
  click(event:MouseEvent){
    const button=eventElement(event.target)?.closest<HTMLButtonElement>('[data-wallet-page-number]');if(!button||button.disabled)return
    const url=new URL(location.href),page=Number(button.dataset.walletPageNumber)
    if(page>1)url.searchParams.set('page',String(page));else url.searchParams.delete('page');void this.load(url)
  }
  restore(){void this.load(location.href,'none')}
}
registerController('wallet',WalletController)
