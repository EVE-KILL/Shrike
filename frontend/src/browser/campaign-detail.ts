import './comment-thread'
import './campaign-activity'
import './campaign-prize'
import './campaign-owner'
import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {createVictimSideLookup,colorKilllistSides,type KilllistSide,type VictimSideLookup} from './killlist-sides'
import {formatDate,formatDateTime} from './campaign-data'
import {eventElement} from './shared/dom'
import {errorMessage,isAbortError,isRecord} from './shared/json'
import type {CampaignDetailResponses} from '../../shared/api/types.gen'

type CampaignTab='overview'|'intel'|'comments'
function campaignTab(value:unknown):CampaignTab|undefined{return value==='overview'||value==='intel'||value==='comments'?value:undefined}
function readSides(raw:unknown):{entities:KilllistSide[];colors:string[]}{
 const source=isRecord(raw)?raw:{},ids=(value:unknown)=>Array.isArray(value)?value.filter((id):id is number=>typeof id==='number'&&Number.isSafeInteger(id)&&id>0):[]
 return {entities:Array.isArray(source.sideEntities)?source.sideEntities.map(side=>{const item=isRecord(side)?side:{};return {characterIds:ids(item.characterIds),corporationIds:ids(item.corporationIds),allianceIds:ids(item.allianceIds)}}):[],colors:Array.isArray(source.sideColors)?source.sideColors.filter((color):color is string=>typeof color==='string'):[]}
}
export class CampaignDetailController extends RequestComponent {
 private tabRequest:AbortController|undefined
 private killRequest:AbortController|undefined
 private pollRequest:AbortController|undefined
 private pollTimer:ReturnType<typeof setTimeout>|undefined
 private attempts=0
 private killLimit=50
 private cursors:Array<string|null>=[null]
 private victimSide:VictimSideLookup=()=>undefined
 private colors:string[]=[]
 private interrupted=false
 private retryTab:CampaignTab|undefined
 private get content(){return this.q<HTMLElement>('[data-campaign-detail-content]')}
 private get id(){return encodeURIComponent(this.element.dataset.campaignId??'')}
 connect(){
  for(const time of this.all<HTMLTimeElement>('[data-campaign-local-time]'))time.textContent=formatDateTime(time.dateTime)
  for(const time of this.all<HTMLTimeElement>('[data-campaign-local-date]'))time.textContent=formatDate(time.dateTime)
  try{const source=readSides(JSON.parse(this.q('#campaign-kill-sides').textContent??'null'));this.victimSide=createVictimSideLookup(source.entities);this.colors=source.colors}catch{/* Untinted rows still remain usable. */}
  this.initializeTab();this.schedulePoll()
 }
 private section(){return this.content.querySelector<HTMLElement>('[data-campaign-killfeed]')}
 private initializeTab(){const section=this.section();if(section){this.killLimit=50;this.cursors=[null];void this.loadKillfeed(section)}}
 private notice(scope:HTMLElement,message:string,kind:'tab'|'kill',page=1,limit=this.killLimit){
  scope.querySelector('[data-campaign-request-error]')?.remove();if(kind==='kill'&&!scope.querySelector('#kill-list'))scope.replaceChildren()
  const box=document.createElement('div'),text=document.createElement('p'),retry=document.createElement('button');box.dataset.campaignRequestError=kind;text.setAttribute('role','alert');text.textContent=message;retry.type='button';retry.className='campaign-detail-retry';retry.textContent='Retry';retry.dataset.campaignRetry=kind;retry.dataset.page=String(page);retry.dataset.limit=String(limit);box.append(text,retry);scope.append(box)
 }
 private async loadKillfeed(section:HTMLElement,page=1,limit=this.killLimit){
  this.killRequest?.abort();const request=new AbortController();this.killRequest=request;const list=section.querySelector<HTMLElement>('[data-campaign-killlist]');if(!list)return
  list.querySelector('[data-campaign-request-error]')?.remove();list.setAttribute('aria-busy','true');list.classList.add('loading');if(!list.querySelector('#kill-list'))list.textContent='Loading killmails…'
  const params=new URLSearchParams({page:String(page),limit:String(limit)}),side=section.querySelector<HTMLElement>('[data-campaign-side].active')?.dataset.campaignSide??'';if(side)params.set('side',side);const cursor=this.cursors[page-1];if(page>1&&cursor)params.set('after',cursor)
  try{
   const text=await this.text(`/_fragments/campaign/${this.id}/killfeed?${params}`,'Unable to load campaign killmails',{signal:request.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('#kill-list');if(!next)throw new Error('Killmail rows are unavailable')
   const {replaceFragmentChildren}=await import('./fragment');if(this.killRequest!==request||!list.isConnected)return
   replaceFragmentChildren(list,next);colorKilllistSides(list,this.victimSide,this.colors);this.killLimit=limit;this.cursors[page]=next.dataset.nextCursor||null
  }catch(error){if(this.killRequest===request&&!isAbortError(error))this.notice(list,errorMessage(error,'Unable to load killmails'),'kill',page,limit)}
  finally{if(this.killRequest===request){this.killRequest=undefined;list.classList.remove('loading');list.removeAttribute('aria-busy')}}
 }
 private cancelKill(){this.killRequest?.abort();this.killRequest=undefined;const list=this.section()?.querySelector<HTMLElement>('[data-campaign-killlist]');list?.classList.remove('loading');list?.removeAttribute('aria-busy')}
 private async loadTab(tab:CampaignTab){
  this.tabRequest?.abort();this.cancelKill();const request=new AbortController();this.tabRequest=request;const content=this.content;content.querySelector('[data-campaign-request-error="tab"]')?.remove();content.classList.add('loading');content.setAttribute('aria-busy','true')
  try{
   const text=await this.text(`/_fragments/campaign/${this.id}/${tab}`,'Unable to load campaign section',{signal:request.signal}),doc=new DOMParser().parseFromString(text,'text/html');if(!doc.querySelector('[data-campaign-killfeed], [data-campaign-intel], [data-comment-thread], [data-campaign-overview]'))throw new Error('Campaign section is unavailable')
   if(this.tabRequest!==request)return;content.replaceChildren(...Array.from(doc.body.childNodes,node=>document.importNode(node,true)));this.element.dataset.campaignTab=tab;this.retryTab=undefined
   for(const link of this.all<HTMLElement>('[data-campaign-detail-tab]'))link.classList.toggle('active',link.dataset.campaignDetailTab===tab);this.initializeTab()
  }catch(error){if(this.tabRequest===request&&!isAbortError(error)){this.retryTab=tab;this.notice(content,errorMessage(error),'tab')}}
  finally{if(this.tabRequest===request){this.tabRequest=undefined;content.classList.remove('loading');content.removeAttribute('aria-busy')}}
 }
 click(event:MouseEvent){
  const target=eventElement(event.target),tab=target?.closest<HTMLAnchorElement>('[data-campaign-detail-tab]');if(tab){if(event.button!==0||event.metaKey||event.ctrlKey||event.shiftKey||event.altKey)return;const selected=campaignTab(tab.dataset.campaignDetailTab);if(!selected)return;event.preventDefault();if(selected!==this.element.dataset.campaignTab||this.tabRequest){history.pushState(null,'',tab.href);void this.loadTab(selected)}return}
  if(!target)return;const section=this.section()
  const retry=target.closest<HTMLElement>('[data-campaign-retry]');if(retry){if(retry.dataset.campaignRetry==='tab'&&this.retryTab)void this.loadTab(this.retryTab);else if(retry.dataset.campaignRetry==='kill'&&section)void this.loadKillfeed(section,Number(retry.dataset.page),Number(retry.dataset.limit));return}
  if(!section)return
  const side=target.closest<HTMLElement>('[data-campaign-side]');if(side&&section.contains(side)){if(side.classList.contains('active'))return;for(const button of section.querySelectorAll<HTMLElement>('[data-campaign-side]'))button.classList.toggle('active',button===side);this.killLimit=50;this.cursors=[null];void this.loadKillfeed(section,1,50);return}
  const limit=target.closest<HTMLElement>('[data-kill-limit]');if(limit&&section.contains(limit)){const value=Number(limit.dataset.killLimit);if(![10,25,50,100].includes(value))return;this.cursors=[null];void this.loadKillfeed(section,1,value);return}
  const pager=target.closest<HTMLButtonElement>('[data-kill-page]');if(pager&&!pager.disabled&&section.contains(pager)){const page=Number(pager.dataset.killPage);if(Number.isSafeInteger(page)&&page>=1&&(page===1||this.cursors[page-1]))void this.loadKillfeed(section,page)}
 }
 popstate(){const tab=campaignTab(location.pathname.split('/').at(-1))??'overview';if(tab!==this.element.dataset.campaignTab||this.tabRequest)void this.loadTab(tab)}
 private schedulePoll(){if(this.element.dataset.campaignComputing!=='true'||this.attempts>=40||this.pollTimer)return;this.pollTimer=this.after(8000,()=>{this.pollTimer=undefined;void this.poll()})}
 private async poll(){
  if(this.element.querySelector('[data-campaign-edit-form], [data-campaign-prize-dialog][open]')){this.schedulePoll();return}
  ++this.attempts;const request=new AbortController();this.pollRequest=request
  try{const detail=await this.request<CampaignDetailResponses[200]>(`/api/campaign/${this.id}`,{signal:request.signal});if(this.pollRequest===request&&!detail.processing?.paused&&detail.status!==0&&detail.stats&&!this.element.querySelector('[data-campaign-edit-form], [data-campaign-prize-dialog][open]')){this.element.dataset.campaignComputing='false';location.reload()}}
  catch{/* Retry on the next owned timer. */}finally{if(this.pollRequest===request){this.pollRequest=undefined;this.schedulePoll()}}
 }
 show(event:PageTransitionEvent){if(event.persisted){if(this.interrupted){this.interrupted=false;const tab=campaignTab(location.pathname.split('/').at(-1))??'overview';void this.loadTab(tab)}this.schedulePoll()}}
 hide(){this.interrupted||=Boolean(this.tabRequest)||Boolean(this.killRequest);this.tabRequest?.abort();this.killRequest?.abort();this.pollRequest?.abort();this.tabRequest=this.killRequest=this.pollRequest=undefined;this.stopTimer(this.pollTimer);this.pollTimer=undefined;this.content.classList.remove('loading');this.content.removeAttribute('aria-busy');const list=this.section()?.querySelector<HTMLElement>('[data-campaign-killlist]');list?.classList.remove('loading');list?.removeAttribute('aria-busy');super.hide()}
}
registerController('campaign-detail',CampaignDetailController)
