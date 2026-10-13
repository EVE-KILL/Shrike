import {AdminSectionComponent} from './controllers/admin/base'
import {registerController} from './controllers/application'
import {createSelectMenu,type SelectMenu} from './select-menu'
import {adminWalletIsk,adminWalletDate,adminWalletPaymentPrompt,adminWalletPaidMessage,adminWalletAuthorizationBanner,type PrizeSettlementRow} from './admin-data'
import {readQuerySeed} from './shared/query-seed'
import {readWalletSettlements} from './shared/admin-runtime-seeds'
import {eventElement} from './shared/dom'
import {errorMessage,isAbortError} from './shared/json'
import type {WalletAdminAuthorizeResponses,CampaignPrizePaidData} from '../../shared/api/types.gen'

export class AdminWalletController extends AdminSectionComponent {
  private query:Record<string,string>|undefined
  private pending:AbortController|undefined
  private menu:SelectMenu|undefined
  private settlements:PrizeSettlementRow[]=[]
  private authorizing:object|undefined
  private syncing:object|undefined
  private markingPaid:{key:string;token:object}|undefined
  private syncTimer:ReturnType<typeof setTimeout>|undefined
  private interrupted=false
  private get host(){return this.q<HTMLElement>('[data-wallet-results]')}
  connect(){super.connect();this.query??=readQuerySeed(this.element.dataset.query);this.initializeResult();this.banner()}
  private banner(){const params=new URLSearchParams(location.search),values:Record<string,unknown>={};for(const key of params.keys()){const items=params.getAll(key);values[key]=items.length===1?items[0]:items}const data=adminWalletAuthorizationBanner(values),node=this.q<HTMLElement>('[data-wallet-banner]');node.hidden=!data;node.textContent=data?.text??'';node.className='rounded-lg border px-4 py-3 text-sm '+(data?.kind==='success'?'border-green-500/20 bg-green-500/10 text-green-300':'border-red-500/20 bg-red-500/10 text-red-300')}
  private notice(selector:string,error:unknown=''){this.error(selector,error)}
  private failure(error:unknown=''){
    if(isAbortError(error)||!this.element.isConnected)return
    const text=typeof error==='string'?error:errorMessage(error),box=this.q<HTMLElement>('[data-wallet-error]');box.hidden=!text
    if(text){box.replaceChildren(document.createTextNode(text+' '));const retry=document.createElement('button');retry.type='button';retry.dataset.walletRefresh='';retry.className='text-blue-400';retry.textContent='Retry';box.append(retry)}
  }
  private render(){
    for(const node of this.host.querySelectorAll<HTMLElement>('[data-wallet-isk]'))node.textContent=adminWalletIsk(node.dataset.walletIsk)
    for(const node of this.host.querySelectorAll<HTMLElement>('[data-wallet-date]'))node.textContent=adminWalletDate(node.dataset.walletDate)
    const authorize=this.host.querySelector<HTMLButtonElement>('[data-wallet-authorize]');if(authorize)authorize.disabled=Boolean(this.authorizing)
    const sync=this.host.querySelector<HTMLButtonElement>('[data-wallet-sync]');if(sync){sync.disabled=Boolean(this.syncing)||sync.dataset.disabled==='true';sync.querySelector('svg')?.classList.toggle('animate-spin',Boolean(this.syncing))}
    for(const refresh of this.host.querySelectorAll<HTMLButtonElement>('[data-wallet-refresh]')){refresh.disabled=Boolean(this.pending);refresh.querySelector('svg')?.classList.toggle('animate-spin',Boolean(this.pending))}
    for(const paid of this.host.querySelectorAll<HTMLButtonElement>('[data-wallet-paid]'))paid.disabled=Boolean(this.markingPaid)
  }
  private initializeResult(){
    this.settlements=readWalletSettlements(this.host.querySelector<HTMLElement>('[data-wallet-result]')?.dataset.initial);this.removeMenu()
    const slot=this.host.querySelector<HTMLElement>('[data-wallet-division]')
    if(slot){this.menu=createSelectMenu('Wallet division',[{value:'',label:'All divisions'},...Array.from({length:7},(_,i)=>({value:String(i+1),label:`Division ${i+1}`}))],this.query?.division??'',value=>{if(!this.query||(this.query.division??'')===value)return;if(value)this.query.division=value;else delete this.query.division;this.query.page='1';void this.load()});slot.append(this.menu.root)}
    this.render()
  }
  private removeMenu(){this.menu?.dispose();this.menu?.root.remove();this.menu=undefined}
  private async load(){
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false;this.failure();this.render()
    const focused=document.activeElement?.closest<HTMLElement>('[data-wallet-refresh],[data-wallet-page],[data-wallet-paid]'),focusKey=focused?.hasAttribute('data-wallet-refresh')?'refresh':focused?.dataset.walletPage?`page:${focused.dataset.walletPage}`:focused?.closest<HTMLElement>('[data-admin-wallet-prize]')?.dataset.adminWalletPrize
    let replaced=false
    try{const text=await this.html('/_fragments/admin/wallet?'+new URLSearchParams(this.query),'Could not load the corporation wallet.',{signal:controller.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('[data-wallet-result]');if(!next||next.dataset.failed==='true')throw new Error('Could not load the corporation wallet.');if(this.pending!==controller)return;this.removeMenu();this.host.replaceChildren(document.importNode(next,true));this.initializeResult();replaced=true}
    catch(error){if(this.pending===controller)this.failure(error)}finally{if(this.pending===controller){this.pending=undefined;this.render();if(replaced&&focusKey){const controls=Array.from(this.host.querySelectorAll<HTMLButtonElement>('[data-wallet-refresh],[data-wallet-page],[data-wallet-paid]'));controls.find(node=>focusKey==='refresh'?node.hasAttribute('data-wallet-refresh'):focusKey.startsWith('page:')?node.dataset.walletPage===focusKey.slice(5):node.closest<HTMLElement>('[data-admin-wallet-prize]')?.dataset.adminWalletPrize===focusKey)?.focus({preventScroll:true})}}}
  }
  private async authorize(){
    if(this.authorizing)return;const token={};this.authorizing=token;this.notice('[data-wallet-action-error]');this.render()
    try{const result=await this.request<WalletAdminAuthorizeResponses[200]>('/api/admin/wallet/authorize');if(this.authorizing===token)location.assign(result.url)}catch(error){if(this.authorizing===token){this.notice('[data-wallet-action-error]',error);this.authorizing=undefined;this.render()}}
  }
  private async sync(){
    if(this.syncing)return;const token={};this.syncing=token;this.notice('[data-wallet-action-error]');this.notice('[data-wallet-message]');this.render()
    try{await this.request('/api/admin/wallet/sync',{method:'POST'});if(this.syncing!==token)return;this.notice('[data-wallet-message]','Wallet sync queued. The journal will update shortly.');this.stopTimer(this.syncTimer);this.syncTimer=this.after(2500,()=>{this.syncTimer=undefined;void this.load()})}
    catch(error){if(this.syncing===token)this.notice('[data-wallet-action-error]',error)}finally{if(this.syncing===token){this.syncing=undefined;this.render()}}
  }
  private async paid(key:string){
    if(this.markingPaid)return
    const row=this.settlements.find(row=>`${row.campaign_id}:${row.rank}`===key);if(!row||row.rank===null||row.paid_at||!row.claimed_at)return
    const note=window.prompt(adminWalletPaymentPrompt(row),'');if(note===null)return
    const token={};this.markingPaid={key,token};this.notice('[data-wallet-action-error]');this.render()
    try{const body:CampaignPrizePaidData['body']={note};await this.request(`/api/admin/campaign-prizes/${encodeURIComponent(row.campaign_id)}/${row.rank}/paid`,{method:'POST',body});if(this.markingPaid?.token!==token)return;this.notice('[data-wallet-message]',adminWalletPaidMessage(row));await this.load()}
    catch(error){if(this.markingPaid?.token===token)this.notice('[data-wallet-action-error]',error)}finally{if(this.markingPaid?.token===token){this.markingPaid=undefined;this.render()}}
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target)
    if(target?.closest('[data-wallet-refresh]')){await this.load();return}if(target?.closest('[data-wallet-authorize]')){await this.authorize();return}if(target?.closest('[data-wallet-sync]')){await this.sync();return}
    const page=target?.closest<HTMLButtonElement>('[data-wallet-page]');if(page&&!page.disabled&&page.dataset.walletPage&&this.query){this.query.page=page.dataset.walletPage;await this.load();return}
    const key=target?.closest('[data-wallet-paid]')?.closest<HTMLElement>('[data-admin-wallet-prize]')?.dataset.adminWalletPrize;if(key)await this.paid(key)
  }
  protected resumed(){this.banner();if(this.interrupted||!this.q<HTMLElement>('[data-wallet-error]').hidden)void this.load()}
  hide(){this.interrupted||=Boolean(this.pending)||Boolean(this.authorizing)||Boolean(this.syncing)||Boolean(this.syncTimer)||Boolean(this.markingPaid);this.pending?.abort();this.pending=undefined;this.stopTimer(this.syncTimer);this.syncTimer=undefined;this.authorizing=undefined;this.syncing=undefined;this.markingPaid=undefined;super.hide();this.render()}
  disconnect(){this.removeMenu();super.disconnect()}
}
registerController('admin-wallet',AdminWalletController)
