import {AdminSectionComponent} from './controllers/admin/base'
import {registerController} from './controllers/application'
import {createSelectMenu,type SelectMenu} from './select-menu'
import {adminCampaignStates} from './admin-data'
import {readQuerySeed} from './shared/query-seed'
import {eventElement} from './shared/dom'
import {errorMessage,isAbortError} from './shared/json'
import type {CampaignAdminActionData} from '../../shared/api/types.gen'

type CampaignAction=CampaignAdminActionData['body']['action']
const isAction=(value:unknown):value is CampaignAction=>value==='pause'||value==='resume'||value==='reprocess'||value==='archive'||value==='delete'
export class AdminCampaignsController extends AdminSectionComponent {
  private query:Record<string,string>|undefined
  private pending:AbortController|undefined
  private timer:ReturnType<typeof setTimeout>|undefined
  private confirmation:string|null=null
  private confirmTimer:ReturnType<typeof setTimeout>|undefined
  private busy=new Map<string,object>()
  private interrupted=false
  private menu:SelectMenu|undefined
  private get host(){return this.q<HTMLElement>('[data-admin-campaign-results]')}
  connect(){super.connect();this.query??=readQuerySeed(this.element.dataset.query);this.menu=createSelectMenu('Campaign state',adminCampaignStates,this.query.state??'all',value=>{if(this.query){this.query.state=value;this.query.page='1';void this.load()}});this.q<HTMLElement>('[data-admin-campaign-filter]').append(this.menu.root);this.render()}
  private render(){for(const row of this.host.querySelectorAll<HTMLElement>('[data-admin-campaign-id]'))for(const button of row.querySelectorAll<HTMLButtonElement>('[data-admin-campaign-action]')){button.disabled=this.busy.has(row.dataset.adminCampaignId??'');if(button.dataset.adminCampaignAction==='delete'){const armed=this.confirmation===row.dataset.adminCampaignId;button.textContent=armed?'Confirm delete':'Delete';button.className='px-2.5 py-1.5 rounded-md text-fine bg-red-500/10 hover:bg-red-500/20 '+(armed?'text-red-300 ring-1 ring-red-500/40':'text-red-400')}}}
  private listError(error:unknown=''){
    if(!this.element.isConnected||isAbortError(error))return
    const message=typeof error==='string'?error:errorMessage(error),box=this.q<HTMLElement>('[data-admin-campaign-error]');box.hidden=!message
    if(message){box.replaceChildren(document.createTextNode(message+' '));const retry=document.createElement('button');retry.type='button';retry.dataset.adminCampaignRetry='';retry.className='text-blue-400';retry.textContent='Retry';box.append(retry)}
  }
  private async load(){
    this.stopTimer(this.timer);this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false;this.host.setAttribute('aria-busy','true');this.listError()
    try{
      const text=await this.html('/_fragments/admin/campaigns?'+new URLSearchParams(this.query),'Could not load campaigns.',{signal:controller.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('[data-admin-campaign-result]')
      if(!next||next.dataset.failed==='true')throw new Error('Could not load campaigns.')
      if(this.pending!==controller)return
      this.host.replaceChildren(document.importNode(next,true));this.render()
    }catch(error){if(this.pending===controller)this.listError(error)}finally{if(this.pending===controller){this.pending=undefined;this.host.removeAttribute('aria-busy')}}
  }
  private async act(id:string,action:CampaignAction){
    if(this.busy.has(id))return
    if(action==='delete'&&this.confirmation!==id){this.stopTimer(this.confirmTimer);this.confirmation=id;this.render();this.confirmTimer=this.after(5000,()=>{this.confirmation=null;this.render()});return}
    const token={};this.busy.set(id,token);this.error('[data-admin-campaign-action-error]');this.render()
    try{const body:CampaignAdminActionData['body']={action};await this.request(`/api/admin/campaigns/${encodeURIComponent(id)}/action`,{method:'POST',body});this.stopTimer(this.confirmTimer);this.confirmation=null;await this.load()}
    catch(error){if(this.busy.get(id)===token)this.error('[data-admin-campaign-action-error]',error)}finally{if(this.busy.get(id)===token)this.busy.delete(id);this.render()}
  }
  search(event:Event){const input=event.target;if(!(input instanceof HTMLInputElement)||!input.matches('[data-admin-campaign-search]')||!this.query)return;this.query.q=input.value;this.query.page='1';this.stopTimer(this.timer);this.pending?.abort();this.interrupted=true;this.timer=this.after(250,()=>void this.load())}
  async click(event:MouseEvent){
    const target=eventElement(event.target),button=target?.closest<HTMLElement>('[data-admin-campaign-action]'),id=button?.closest<HTMLElement>('[data-admin-campaign-id]')?.dataset.adminCampaignId,action=button?.dataset.adminCampaignAction
    if(id&&isAction(action)){await this.act(id,action);return}
    const page=target?.closest<HTMLButtonElement>('[data-admin-campaign-page]');if(page&&!page.disabled&&page.dataset.adminCampaignPage&&this.query){this.query.page=page.dataset.adminCampaignPage;await this.load();return}
    if(target?.closest('[data-admin-campaign-retry]'))await this.load()
  }
  protected resumed(){if(this.interrupted||!this.q<HTMLElement>('[data-admin-campaign-error]').hidden)void this.load()}
  hide(){this.interrupted||=Boolean(this.pending)||this.busy.size>0;this.stopTimer(this.timer);this.stopTimer(this.confirmTimer);this.confirmation=null;this.pending?.abort();this.pending=undefined;this.busy.clear();this.host.removeAttribute('aria-busy');super.hide();this.render()}
  disconnect(){this.menu?.dispose();this.menu?.root.remove();this.menu=undefined;super.disconnect()}
}
registerController('admin-campaigns',AdminCampaignsController)
