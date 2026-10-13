import {SettingsComponent} from './base'
import {registerController} from '../application'
import {eventElement,requiredElement} from '../../shared/dom'

export class SettingsWalletController extends SettingsComponent{
  private loading=false
  connect(){
    this.loading=false;const formatter=new Intl.DateTimeFormat('en-GB',{dateStyle:'medium',timeStyle:'short'})
    for(const time of this.all<HTMLTimeElement>('[data-settings-wallet-date]')){const date=new Date(time.dateTime);if(time.dateTime&&!Number.isNaN(date.getTime()))time.textContent=formatter.format(date)}
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target),copy=target?.closest<HTMLButtonElement>('[data-settings-wallet-copy]')
    if(copy){
      try{
        await navigator.clipboard.writeText(copy.dataset.settingsWalletCopy??'');if(!this.element.isConnected)return
        requiredElement<HTMLElement>(copy,'[data-settings-wallet-copy-label]').textContent='Copied';requiredElement<SVGUseElement>(copy,'use').setAttribute('href','/assets/icons.svg#check')
        this.after(2000,()=>{requiredElement<HTMLElement>(copy,'[data-settings-wallet-copy-label]').textContent='Copy name';requiredElement<SVGUseElement>(copy,'use').setAttribute('href','/assets/icons.svg#copy')})
      }catch{this.error('[data-settings-wallet-error]','Could not copy the corporation name.')}return
    }
    const button=target?.closest<HTMLButtonElement>('[data-settings-wallet-page]');if(!button||button.disabled||this.loading)return
    const controls=[...this.all<HTMLButtonElement>('[data-settings-wallet-page]')].map(control=>({control,disabled:control.disabled}))
    this.loading=true;for(const {control}of controls)control.disabled=true;this.error('[data-settings-wallet-error]')
    try{
      const text=await this.text('/_fragments/settings/wallet?page='+button.dataset.settingsWalletPage,'Could not load your EK Wallet.')
      const doc=new DOMParser().parseFromString(text,'text/html'),next=doc.querySelector<HTMLElement>('[data-settings-wallet]')
      if(!next||!next.querySelector<HTMLElement>('[data-settings-wallet-error]')?.hidden)throw new Error('Could not load your EK Wallet.')
      this.element.replaceWith(document.importNode(next,true))
    }catch(error){this.error('[data-settings-wallet-error]',error)}finally{this.loading=false;for(const {control,disabled}of controls)control.disabled=disabled}
  }
}
registerController('settings-wallet',SettingsWalletController)
