import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {iconHref} from './icon-hrefs'
import {readBrowserSession,type BrowserUser} from './browser-request'
import {formatIsk as iskText,formatDateTime} from './campaign-data'
import {eventElement} from './shared/dom'
import {isBackdropClick} from './shared/dialog'
import {isAbortError,errorMessage} from './shared/json'
import type {WalletAccountBalanceLegacyResponses,CampaignPrizeContributeData,CampaignPrizeContributeResponses,AuthLoginLegacyResponses} from '../../shared/api/types.gen'

export function campaignContributionError(amount:string,available:number){if(!/^\d+(?:\.\d{1,2})?$/.test(amount)||Number(amount)<=0)return 'Enter a positive ISK amount with at most two decimals.';if(Number(amount)>available)return 'That exceeds your available EK Wallet balance.';return ''}
export function campaignContributionMessage(contributed:string|number){return `${iskText(Number(contributed))} added to the prize pool.`}
export class CampaignPrizeController extends RequestComponent {
  private enabled=false
  private user:BrowserUser|null=null
  private available=0
  private availableAmount=''
  private refreshRequest:AbortController|undefined
  private walletPending:AbortController|undefined
  private contributing:object|undefined
  private claiming:object|undefined
  private contribution:{amount:string;id:string}|undefined
  private generation=0
  private revision=0
  private interrupted=false
  private get dialog(){return this.q<HTMLDialogElement>('[data-campaign-prize-dialog]')}
  private get id(){return this.element.dataset.campaignId??''}
  connect(){this.enabled=Boolean(this.element.querySelector('[data-campaign-prize-dialog]'));if(!this.enabled)return;const generation=++this.generation;void readBrowserSession().then(user=>{if(generation!==this.generation||!this.element.isConnected)return;this.user=user;this.controls();if(this.dialog.open&&user)void this.loadWallet()}).catch(()=>{});this.times();this.controls()}
  private times(){for(const time of this.dialog.querySelectorAll<HTMLTimeElement>('[data-campaign-local-time]'))time.textContent=formatDateTime(time.dateTime)}
  private message(value:string,error=false){const target=this.dialog.querySelector<HTMLElement>(error?'[data-campaign-prize-error]':'[data-campaign-prize-message]'),other=this.dialog.querySelector<HTMLElement>(error?'[data-campaign-prize-message]':'[data-campaign-prize-error]');if(other)other.hidden=true;if(target){target.textContent=value;target.hidden=!value}}
  private controls(){
    if(!this.enabled)return
    const form=this.dialog.querySelector<HTMLElement>('[data-campaign-wallet-form]'),login=this.dialog.querySelector<HTMLElement>('[data-campaign-prize-login]');if(form)form.hidden=!this.user;if(login)login.hidden=Boolean(this.user)
    const max=this.dialog.querySelector<HTMLButtonElement>('[data-campaign-wallet-max]');if(max){max.disabled=Boolean(this.walletPending)||this.available<=0;max.textContent=this.walletPending?'Loading balance…':`${iskText(this.available)} available · use max`}
    const submit=this.dialog.querySelector<HTMLButtonElement>('[data-campaign-wallet-contribute]');if(submit){submit.disabled=Boolean(this.contributing)||Boolean(this.claiming)||Boolean(this.walletPending)||this.available<=0;submit.textContent=this.contributing?'Adding…':'Add ISK'}
    for(const button of this.dialog.querySelectorAll<HTMLButtonElement>('[data-campaign-prize-claim]'))button.disabled=Boolean(this.claiming)||Boolean(this.contributing)
  }
  private async loadWallet(){
    if(!this.user)return;this.walletPending?.abort();const controller=new AbortController();this.walletPending=controller;this.controls()
    try{const wallet=await this.request<WalletAccountBalanceLegacyResponses[200]>('/api/user/wallet/balance',{signal:controller.signal});if(this.walletPending!==controller)return;this.availableAmount=String(wallet.availableBalance??wallet.balance??'0');this.available=Number(this.availableAmount)||0}
    catch(error){if(this.walletPending===controller&&!isAbortError(error)){this.available=0;this.availableAmount=''}}finally{if(this.walletPending===controller){this.walletPending=undefined;this.controls()}}
  }
  private async refresh(preserveAmount?:()=>string|undefined){
    this.refreshRequest?.abort();const request=new AbortController();this.refreshRequest=request;
    const text=await this.text(`/_fragments/campaign/${encodeURIComponent(this.id)}/prize`,'Could not refresh prize pool',{signal:request.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLDialogElement>('[data-campaign-prize-dialog]');if(!next)throw new Error('Prize pool is unavailable')
    if(this.refreshRequest!==request)return;this.refreshRequest=undefined;
    const current=this.dialog.querySelector<HTMLInputElement>('[data-campaign-wallet-amount]'),focused=current===document.activeElement,value=preserveAmount?.(),selection=current?[current.selectionStart,current.selectionEnd]:undefined;
    const wasOpen=this.dialog.open;if(wasOpen)this.dialog.close();this.dialog.replaceWith(document.importNode(next,true));this.times();this.controls()
    const amount=this.dialog.querySelector<HTMLInputElement>('[data-campaign-wallet-amount]');if(amount&&value!==undefined)amount.value=value
    if(wasOpen){this.dialog.showModal();if(focused&&amount){amount.focus();if(selection)amount.setSelectionRange(selection[0],selection[1])}}const funded=this.dialog.querySelector<HTMLElement>('.campaign-prize-summary strong')?.textContent;if(funded)this.q<HTMLElement>('[data-campaign-prize-total]').textContent=funded
  }
  private async contribute(){
    if(this.contributing||this.claiming||this.walletPending||!this.user)return
    const amount=this.dialog.querySelector<HTMLInputElement>('[data-campaign-wallet-amount]')?.value.trim()??'',error=campaignContributionError(amount,this.available);this.message('');if(error){this.message(error,true);return}
    if(this.contribution?.amount!==amount)this.contribution={amount,id:crypto.randomUUID()};const request=this.contribution,token={},revision=this.revision;this.contributing=token;this.controls()
    try{const body:CampaignPrizeContributeData['body']={amount,requestId:request.id},result=await this.request<CampaignPrizeContributeResponses[200]>(`/api/campaign/${encodeURIComponent(this.id)}/prize/contribute`,{method:'POST',body});if(this.contributing!==token)return;await this.refresh(()=>this.revision!==revision?this.dialog.querySelector<HTMLInputElement>('[data-campaign-wallet-amount]')?.value:undefined);await this.loadWallet();if(this.contributing===token){this.contribution=undefined;this.message(campaignContributionMessage(result.contributed))}}
    catch(error){if(this.contributing===token&&!isAbortError(error))this.message(errorMessage(error,'Could not fund the prize pool'),true)}finally{if(this.contributing===token){this.contributing=undefined;this.controls()}}
  }
  private async claim(){
    if(this.claiming||this.contributing)return;const token={};this.claiming=token;this.controls();this.message('')
    try{await this.request(`/api/campaign/${encodeURIComponent(this.id)}/prize/claim`,{method:'POST'});if(this.claiming!==token)return;await this.refresh();if(this.claiming===token)this.message('Prize claimed. Join Discord to arrange the in-game payment.')}
    catch(error){if(this.claiming===token&&!isAbortError(error))this.message(errorMessage(error,'Could not claim prize'),true)}finally{if(this.claiming===token){this.claiming=undefined;this.controls()}}
  }
  async click(event:MouseEvent){
    if(!this.enabled)return;const target=eventElement(event.target)
    try{
      if(isBackdropClick(this.dialog,event)||target?.closest('[data-campaign-prize-close]'))this.dialog.close()
      else if(target?.closest('[data-campaign-prize-open]')){if(this.interrupted){await this.refresh();this.interrupted=false}if(!this.dialog.open)this.dialog.showModal();this.controls();if(this.user)await this.loadWallet()}
      else if(target?.closest('[data-campaign-prize-copy]')){const reference=this.dialog.querySelector<HTMLElement>('.campaign-prize-reference code')?.textContent??'';if(reference)await navigator.clipboard.writeText(reference);const button=this.dialog.querySelector<HTMLButtonElement>('[data-campaign-prize-copy]'),svg=button?.querySelector<SVGUseElement>('use'),href=svg?.getAttribute('href');if(svg&&href)svg.setAttribute('href',iconHref('check'));if(button?.lastChild)button.lastChild.textContent='Copied';this.after(2000,()=>{if(button?.isConnected&&button.lastChild){button.lastChild.textContent='Copy';if(svg&&href)svg.setAttribute('href',href)}})}
      else if(target?.closest('[data-campaign-prize-login]')){const result=await this.request<AuthLoginLegacyResponses[200]>('/auth/login?'+new URLSearchParams({redirect:location.pathname+location.search}));if(result.url)location.assign(result.url)}
      else if(target?.closest('[data-campaign-wallet-max]')){const input=this.dialog.querySelector<HTMLInputElement>('[data-campaign-wallet-amount]');if(input){input.value=this.availableAmount;++this.revision}}
      else if(target?.closest('[data-campaign-prize-claim]'))await this.claim()
    }catch(error){if(!isAbortError(error)&&this.element.isConnected)this.message(errorMessage(error,'Prize action failed'),true)}
  }
  submit(event:SubmitEvent){if(event.target instanceof HTMLFormElement&&event.target.matches('[data-campaign-wallet-form]')){event.preventDefault();void this.contribute()}}
  input(event:Event){if(event.target instanceof HTMLInputElement&&event.target.matches('[data-campaign-wallet-amount]')){++this.revision}}
  show(event:PageTransitionEvent){if(event.persisted&&this.enabled&&this.interrupted)void this.refresh().then(()=>this.interrupted=false).catch(error=>{if(!isAbortError(error))this.message(errorMessage(error),true)})}
  hide(){if(this.enabled){this.interrupted||=Boolean(this.contributing)||Boolean(this.claiming);this.walletPending?.abort();this.walletPending=undefined;this.refreshRequest?.abort();this.refreshRequest=undefined;this.contributing=this.claiming=undefined;this.dialog.close();this.controls()}super.hide()}
  disconnect(){++this.generation;super.disconnect()}
}
registerController('campaign-prize',CampaignPrizeController)
