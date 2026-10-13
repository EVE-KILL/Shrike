import {AdminSectionComponent,AdminComponent} from './controllers/admin/base'
import {registerController} from './controllers/application'
import {filterAdminDomains,type AdminDomainSummary} from './admin-data'
import {eventElement} from './shared/dom'
import {isRecord,errorMessage,isAbortError} from './shared/json'
import {toast} from './toast'
import type {AdminDomainAssetReviewData} from '../../shared/api/types.gen'

type DomainFilter='all'|'pending'|'inactive'
export class AdminDomainsController extends AdminSectionComponent {
  private domains:AdminDomainSummary[]=[]
  private filter:DomainFilter='all'
  connect(){
    super.connect();let seed:unknown;try{seed=JSON.parse(this.element.dataset.initial??'null')}catch{seed=null}
    this.domains=isRecord(seed)&&Array.isArray(seed.domains)?seed.domains.flatMap(row=>isRecord(row)&&typeof row.id==='number'&&typeof row.subdomain==='string'?[{id:row.id,subdomain:row.subdomain,active:row.active===true,site_name:typeof row.site_name==='string'?row.site_name:null,owner_name:typeof row.owner_name==='string'?row.owner_name:null,custom_hostname:typeof row.custom_hostname==='string'?row.custom_hostname:null,pending_assets:typeof row.pending_assets==='number'?row.pending_assets:0}]:[]):[]
    this.render()
  }
  render(){
    const shown=new Set(filterAdminDomains(this.domains,this.q<HTMLInputElement>('[data-admin-domain-search]').value,this.filter).map(row=>String(row.id)))
    for(const row of this.all<HTMLElement>('[data-admin-domain-row]'))row.hidden=!shown.has(row.dataset.adminDomainRow??'')
    this.q<HTMLElement>('[data-admin-domain-empty]').hidden=shown.size>0
    for(const button of this.all<HTMLButtonElement>('[data-admin-domain-filter]')){const selected=button.dataset.adminDomainFilter===this.filter;button.setAttribute('aria-pressed',String(selected));button.className='px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer flex items-center gap-1.5 '+(selected?'bg-blue-500/[0.15] text-blue-300':'text-gray-400 hover:text-white');button.querySelector<HTMLElement>('span')?.setAttribute('class','tabular-nums '+(selected?'text-blue-400/70':'text-gray-600'))}
  }
  click(event:MouseEvent){const value=eventElement(event.target)?.closest<HTMLElement>('[data-admin-domain-filter]')?.dataset.adminDomainFilter;if(value==='all'||value==='pending'||value==='inactive'){this.filter=value;this.render()}}
}

export class AdminDomainDetailController extends AdminComponent {
  private active='approvals'
  private reason=''
  private reasonRevision=0
  private pending:AbortController|undefined
  private busy=new Map<number,object>()
  private toggling:object|undefined
  private interrupted=false
  private get domainID(){return this.element.dataset.domainId??''}
  private get header(){return this.q<HTMLElement>('[data-admin-domain-header]')}
  private get content(){return this.q<HTMLElement>('[data-admin-domain-content]')}
  connect(){this.render()}
  private render(){
    for(const panel of this.content.querySelectorAll<HTMLElement>('[data-admin-domain-panel]'))panel.hidden=panel.dataset.adminDomainPanel!==this.active
    for(const button of this.content.querySelectorAll<HTMLButtonElement>('[data-admin-domain-tab]')){const selected=button.dataset.adminDomainTab===this.active;button.className='px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer '+(selected?'bg-blue-500/10 text-blue-400':'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]');button.setAttribute('aria-pressed',String(selected))}
    for(const input of this.all<HTMLInputElement>('[data-admin-domain-reason]'))if(input.value!==this.reason)input.value=this.reason
    for(const row of this.content.querySelectorAll<HTMLElement>('[data-admin-domain-asset]'))for(const button of row.querySelectorAll<HTMLButtonElement>('[data-admin-domain-review]'))button.disabled=this.busy.has(Number(row.dataset.adminDomainAsset))
    const toggle=this.q<HTMLButtonElement>('[data-admin-domain-toggle]');toggle.disabled=Boolean(this.toggling);this.q<HTMLElement>('[data-admin-domain-toggle-spinner]').hidden=!this.toggling
    if(this.active==='files')for(const image of this.content.querySelectorAll<HTMLImageElement>('img[data-src]')){image.src=image.dataset.src??'';delete image.dataset.src}
  }
  private failure(error:unknown){
    if(isAbortError(error)||!this.element.isConnected)return
    let box=this.element.querySelector<HTMLElement>('[data-admin-domain-refresh-error]')
    if(!box){box=document.createElement('p');box.dataset.adminDomainRefreshError='';box.className='glass-panel p-4 mb-4 text-sm text-red-300';box.setAttribute('role','alert');this.element.insertBefore(box,this.content)}
    box.replaceChildren(document.createTextNode(errorMessage(error)+' '));const retry=document.createElement('button');retry.type='button';retry.dataset.adminDomainRefresh='';retry.className='text-blue-400';retry.textContent='Retry';box.append(retry)
  }
  private async refresh(){
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false
    try{
      const text=await this.html('/_fragments/admin/domains/'+encodeURIComponent(this.domainID),'Could not refresh this domain.',{signal:controller.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('[data-admin-domain-detail]')
      const header=next?.querySelector<HTMLElement>('[data-admin-domain-header]'),content=next?.querySelector<HTMLElement>('[data-admin-domain-content]')
      if(!next||next.dataset.domainId!==this.domainID||!header||!content)throw new Error('Could not refresh this domain.')
      if(this.pending!==controller)return
      const key=(image:HTMLImageElement)=>(image.closest<HTMLElement>('[data-admin-domain-panel]')?.dataset.adminDomainPanel??'')+':'+(image.getAttribute('src')??image.dataset.src)
      const images=new Map(Array.from(this.content.querySelectorAll<HTMLImageElement>('img'),image=>[key(image),image]))
      for(const image of content.querySelectorAll<HTMLImageElement>('img')){const existing=images.get(key(image));if(existing){images.delete(key(image));image.replaceWith(existing)}}
      this.header.replaceChildren(...header.childNodes);this.content.replaceChildren(...content.childNodes);this.element.querySelector('[data-admin-domain-refresh-error]')?.remove();this.render()
    }catch(error){if(this.pending===controller)this.failure(error)}finally{if(this.pending===controller)this.pending=undefined}
  }
  private async mutate(path:string,body?:AdminDomainAssetReviewData['body'],id?:number){
    if(id===undefined?this.toggling:this.busy.has(id))return
    const token={},revision=this.reasonRevision;if(id===undefined)this.toggling=token;else this.busy.set(id,token);this.render()
    try{await this.request('/api/admin/domains/'+encodeURIComponent(this.domainID)+'/'+path,{method:'POST',...(body?{body}:{})});if(id!==undefined&&this.reasonRevision===revision)this.reason='';await this.refresh()}
    catch(error){if(!isAbortError(error)&&this.element.isConnected)toast(errorMessage(error),'error')}
    finally{if(id===undefined){if(this.toggling===token)this.toggling=undefined}else if(this.busy.get(id)===token)this.busy.delete(id);this.render()}
  }
  input(event:Event){const input=event.target;if(input instanceof HTMLInputElement&&input.matches('[data-admin-domain-reason]')){this.reason=input.value;++this.reasonRevision;this.render()}}
  async click(event:MouseEvent){
    const target=eventElement(event.target),tab=target?.closest<HTMLElement>('[data-admin-domain-tab]')?.dataset.adminDomainTab
    if(tab==='approvals'||tab==='files'){this.active=tab;this.render();return}
    const review=target?.closest<HTMLElement>('[data-admin-domain-review]'),action=review?.dataset.adminDomainReview,id=Number(review?.closest<HTMLElement>('[data-admin-domain-asset]')?.dataset.adminDomainAsset)
    if((action==='approve'||action==='reject')&&Number.isSafeInteger(id)&&id>0){await this.mutate(`assets/${id}/review`,{action,...(action==='reject'?{reason:this.reason}:{})},id);return}
    if(target?.closest('[data-admin-domain-toggle]'))await this.mutate('toggle-active')
    if(target?.closest('[data-admin-domain-refresh]'))await this.refresh()
  }
  show(event:PageTransitionEvent){if(event.persisted&&this.interrupted)void this.refresh()}
  hide(){this.interrupted||=Boolean(this.pending)||Boolean(this.toggling)||this.busy.size>0;this.pending?.abort();this.pending=undefined;this.toggling=undefined;this.busy.clear();super.hide();this.render()}
}
registerController('admin-domains',AdminDomainsController)
registerController('admin-domain-detail',AdminDomainDetailController)
