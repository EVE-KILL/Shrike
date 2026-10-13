import './roam-composition'
import './controllers/roam-form'
import {roamFormSnapshot} from './controllers/roam-form'
import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'
import {isAbortError,errorMessage} from './shared/json'
import {isRoamTab,isRoamID,isRoamSaved,sameRoamDraft,readRoamMarkers,type RoamTab} from './shared/roam-report'
import {isMapScope,type MapScope} from './shared/map-scope'
import {loadMapCanvas} from './shared/map-module'
import {iconHref} from './icon-hrefs'
import {replaceFragment,replaceFragmentChildren} from './fragment'
import type {GoMapController,MapScopeData} from '../utils/map/goFrontendCanvas'

export class RoamReportController extends RequestComponent{
 private tabRequest:AbortController|undefined
 private refreshRequest:AbortController|undefined
 private pendingTab:{tab:RoamTab;params:URLSearchParams}|undefined
 private mapRequest:AbortController|undefined
 private mapRenderer:GoMapController|undefined
 private mapRegion:number|null=null
 private selectedSystem:number|null=null
 private geography=new Map<MapScope,MapScopeData>()
 private copyTimer:ReturnType<typeof setTimeout>|undefined
 private copyGeneration=0
 private interruptedTab=false
 private interruptedRefresh=false
 private interruptedMap=false
 private get id(){return this.element.dataset.roamId??''}
 private get content(){return this.q<HTMLElement>('[data-roam-content]')}
 private tab():RoamTab{return isRoamTab(this.element.dataset.roamTab)?this.element.dataset.roamTab:'information'}
 connect(){this.initializeTab()}
 private tabAt(url:URL):RoamTab|undefined{const parts=url.pathname.split('/');if(url.origin!==location.origin||parts[1]!=='tools'||parts[2]!=='roam-report'||parts[3]!==this.id||parts.length>5)return;const tab=parts[4]||'information';return isRoamTab(tab)?tab:undefined}
 private icon(button:HTMLElement,name:string){const use=button.querySelector<SVGUseElement>('use');if(use)use.setAttribute('href',iconHref(name))}
 private notice(message:string,refresh=false){const box=this.q<HTMLElement>('[data-roam-status]');box.hidden=!message;box.replaceChildren(message);box.setAttribute('role',refresh?'alert':'status');if(refresh){const retry=document.createElement('button');retry.type='button';retry.className='underline';retry.dataset.roamRefresh='';retry.textContent='Refresh report';box.append(' ',retry)}}
 private stopMap(){this.mapRequest?.abort();this.mapRequest=undefined;this.mapRenderer?.destroy();this.mapRenderer=undefined;this.element.querySelector('[data-roam-map]')?.removeAttribute('aria-busy')}
 private selectSystem(id:number|null){this.selectedSystem=this.selectedSystem===id?null:id;const section=this.element.querySelector<HTMLElement>('[data-roam-systems]');if(!section)return;for(const button of section.querySelectorAll<HTMLElement>('[data-roam-system]')){const active=Number(button.dataset.roamSystem)===this.selectedSystem;button.setAttribute('aria-pressed',String(active));button.classList.toggle('border-blue-500/40',active);button.classList.toggle('bg-blue-500/[0.08]',active);button.classList.toggle('border-white/[0.06]',!active)}for(const article of section.querySelectorAll<HTMLElement>('[data-roam-engagement-system]'))article.hidden=this.selectedSystem!==null&&Number(article.dataset.roamEngagementSystem)!==this.selectedSystem;const clear=section.querySelector<HTMLElement>('[data-roam-system-clear]');if(clear){clear.hidden=this.selectedSystem===null;clear.textContent=`${section.querySelector<HTMLElement>(`[data-roam-system="${this.selectedSystem}"]`)?.dataset.systemName??''} · clear filter`}}
 private async initializeMap(){
  const section=this.element.querySelector<HTMLElement>('[data-roam-systems]'),host=section?.querySelector<HTMLElement>('[data-roam-map]');if(!section||!host)return;const scope=requiredElement<HTMLSelectElement>(section,'[data-roam-map-scope]').value;if(!isMapScope(scope))return
  this.stopMap();const request=new AbortController();this.mapRequest=request;const error=requiredElement<HTMLElement>(section,'[data-roam-map-error]');error.hidden=true;host.setAttribute('aria-busy','true');const current=()=>this.mapRequest===request&&host.isConnected
  try{const markers=readRoamMarkers(JSON.parse(requiredElement(section,'#roam-map-data').textContent??'null')),cached=this.geography.get(scope),geography=cached?Promise.resolve(cached):this.request<MapScopeData>(`/api/map/scope?type=${encodeURIComponent(scope)}&hours=24`,{signal:request.signal},'Unable to load combat map').then(data=>{if(!Array.isArray(data?.systems))throw new Error('Combat map data is unavailable');this.geography.set(scope,data);return data}),[module,data]=await Promise.all([loadMapCanvas(),geography]);if(!current())return
   this.mapRenderer=module.createGoMap(host,{baseLayer:'geography',activityLayer:'none',regionId:this.mapRegion,showConnections:true,showSystems:true,showLabels:true,battleSystems:markers,markerNoun:'engagements',markerValueLabel:'ISK involved',showScopeDetails:true,onBattleRegion:id=>{if(!current())return;this.mapRegion=id;this.mapRenderer?.setRegion(id);requiredElement<HTMLElement>(section,'[data-roam-map-back]').hidden=false},onBattleSystem:system=>{if(!current())return;this.selectedSystem=null;this.selectSystem(Number(system.solar_system_id));section.querySelector<HTMLElement>('#roam-system-order')?.scrollIntoView({behavior:'smooth',block:'start'})}});this.mapRenderer.setData(data)
  }catch(cause){if(!isAbortError(cause)&&current()){error.hidden=false;const retry=document.createElement('button');retry.type='button';retry.className='underline';retry.dataset.roamMapRetry='';retry.textContent='Retry';error.replaceChildren(errorMessage(cause,'Unable to load combat map'),' ',retry)}}
  finally{if(current())host.removeAttribute('aria-busy')}
 }
 private initializeTab(){this.selectedSystem=null;this.mapRegion=null;void this.initializeMap()}
 private syncTabs(tab:RoamTab){this.element.dataset.roamTab=tab;for(const link of this.all<HTMLAnchorElement>('nav [data-roam-tab-link]')){const active=link.dataset.roamTabLink===tab;link.setAttribute('aria-current',active?'page':'false');link.className=`flex shrink-0 items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors ${active?'border-white text-white':'border-transparent text-gray-500 hover:text-blue-400'}`}const label=this.element.querySelector<HTMLElement>(`nav [data-roam-tab-link="${tab}"]`)?.textContent?.trim()??tab;document.title=`Roam Report${tab==='information'?'':' — '+label} - EVE-KILL.com`;const canonical=document.querySelector<HTMLLinkElement>('link[rel="canonical"]');if(canonical)canonical.href=`https://eve-kill.com/tools/roam-report/${this.id}${tab==='information'?'':'/'+tab}`}
 private async loadTab(tab:RoamTab,params=new URLSearchParams()){
  this.tabRequest?.abort();this.refreshRequest?.abort();this.refreshRequest=undefined;const request=new AbortController();this.tabRequest=request;this.pendingTab={tab,params:new URLSearchParams(params)};const content=this.content;content.querySelector('[data-roam-tab-error]')?.remove();content.classList.add('roam-tab-loading');content.setAttribute('aria-busy','true')
  try{const text=await this.text(`/_fragments/roam/${this.id}/${tab}${params.size?'?'+params:''}`,'Unable to load report section',{signal:request.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('[data-roam-tab-content]');if(!next||next.dataset.roamTabContent!==tab)throw new Error('Report section is unavailable');if(this.tabRequest!==request)return;this.stopMap();replaceFragmentChildren(content,document.importNode(next,true));this.syncTabs(tab);this.initializeTab()}
  catch(cause){if(!isAbortError(cause)&&this.tabRequest===request){const error=document.createElement('div'),retry=document.createElement('button');error.className='glass-panel p-4 mt-4 text-sm text-red-300';error.dataset.roamTabError='';error.setAttribute('role','alert');retry.type='button';retry.className='underline';retry.dataset.roamTabRetry='';retry.textContent='Retry';error.append(errorMessage(cause,'Unable to load report section'),' ',retry);content.append(error)}}
  finally{if(this.tabRequest===request){this.tabRequest=undefined;content.classList.remove('roam-tab-loading');content.removeAttribute('aria-busy')}}
 }
 private async refresh(){
  this.tabRequest?.abort();this.tabRequest=undefined;this.refreshRequest?.abort();const request=new AbortController();this.refreshRequest=request;const tab=this.tabAt(new URL(location.href))??this.tab(),form=this.element.querySelector<HTMLFormElement>('[data-roam-form]'),draft=form?roamFormSnapshot(form):undefined;this.content.classList.remove('roam-tab-loading');this.content.removeAttribute('aria-busy')
  try{const text=await this.text(`/_fragments/roam/${this.id}/page?tab=${tab}`,'Report saved, but the refreshed report could not load.',{signal:request.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('[data-roam-report]');if(!next||next.dataset.roamId!==this.id)throw new Error('Report saved, but the refreshed report could not load.');if(this.refreshRequest!==request)return;if(form&&draft&&!sameRoamDraft(draft,roamFormSnapshot(form))){this.notice('Report saved. You have newer unsaved changes.');return}this.refreshRequest=undefined;this.stopMap();replaceFragment(this.element,document.importNode(next,true))}
  catch(cause){if(!isAbortError(cause)&&this.refreshRequest===request)this.notice(errorMessage(cause,'Report saved, but the refreshed report could not load.'),true)}
  finally{if(this.refreshRequest===request)this.refreshRequest=undefined}
 }
 reportSaved(event:CustomEvent<unknown>){if(!isRoamSaved(event.detail)||event.detail.id!==this.id)return;if(!event.detail.hasNewerDraft)event.detail.refresh=this.refresh()}
 private async copy(button:HTMLElement){const generation=++this.copyGeneration;this.stopTimer(this.copyTimer);try{await navigator.clipboard.writeText(location.href);if(!button.isConnected||generation!==this.copyGeneration)return;this.icon(button,'check');requiredElement(button,'span').textContent='Copied';this.copyTimer=this.after(2500,()=>{this.icon(button,'link');requiredElement(button,'span').textContent='Copy link'})}catch{/* Clipboard failure leaves the share URL available. */}}
 click(event:MouseEvent){
  const target=eventElement(event.target),link=target?.closest<HTMLAnchorElement>('[data-roam-tab-link]');if(link&&!event.defaultPrevented&&!event.ctrlKey&&!event.metaKey&&!event.shiftKey&&!event.altKey&&event.button===0){const tab=this.tabAt(new URL(link.href));if(!tab||!isRoamID(this.id))return;event.preventDefault();if(tab!==this.tab()){history.pushState(null,'',link.href);void this.loadTab(tab,new URL(link.href).searchParams)}return}
  const button=target?.closest<HTMLButtonElement>('button');if(!button||button.disabled)return
  if(button.hasAttribute('data-roam-copy')){void this.copy(button);return}
  if(button.hasAttribute('data-roam-edit-open')){const edit=this.q<HTMLElement>('[data-roam-edit]');edit.hidden=false;edit.scrollIntoView({behavior:'smooth',block:'start'});return}
  if(button.hasAttribute('data-roam-cancel')){const edit=this.q<HTMLElement>('[data-roam-edit]');requiredElement<HTMLFormElement>(edit,'form').reset();edit.hidden=true;return}
  if(button.hasAttribute('data-roam-refresh')){void this.refresh();return}
  if(button.hasAttribute('data-roam-tab-retry')){if(this.pendingTab)void this.loadTab(this.pendingTab.tab,this.pendingTab.params);return}
  if(button.hasAttribute('data-roam-map-retry')){void this.initializeMap();return}
  if(button.hasAttribute('data-roam-system')){const id=Number(button.dataset.roamSystem);if(Number.isSafeInteger(id)&&id>0)this.selectSystem(id);return}
  if(button.hasAttribute('data-roam-system-clear')){this.selectedSystem=null;this.selectSystem(null);return}
  if(button.hasAttribute('data-roam-map-back')){this.mapRegion=null;this.mapRenderer?.setRegion(null);button.hidden=true;return}
  if(button.hasAttribute('data-roam-kill-role')||button.hasAttribute('data-kill-limit')||button.hasAttribute('data-kill-page')){const section=this.element.querySelector<HTMLElement>('[data-roam-kills]');if(!section)return;const list=requiredElement<HTMLElement>(section,'#kill-list'),currentRole=section.querySelector<HTMLElement>('[data-roam-kill-role][aria-pressed="true"]')?.dataset.roamKillRole??'kills',pending=this.tabRequest&&this.pendingTab?.tab==='kills'?this.pendingTab.params:undefined,role=button.dataset.roamKillRole??pending?.get('role')??currentRole,limit=Number(button.dataset.killLimit??pending?.get('limit')??list.dataset.limit),page=Number(button.dataset.killPage??1);if(!['kills','losses','all'].includes(role)||![10,25,50,100].includes(limit)||!Number.isSafeInteger(page)||page<1||page>1000000)return;const params=new URLSearchParams({role,limit:String(limit),page:String(page)});if(page===Number(list.dataset.page)+1&&list.dataset.nextCursor&&role===currentRole&&limit===Number(list.dataset.limit))params.set('after',list.dataset.nextCursor);history.replaceState(null,'',`${location.pathname}?${params}`);void this.loadTab('kills',params)}
 }
 change(event:Event){const target=event.target;if(target instanceof HTMLSelectElement&&target.matches('[data-roam-map-scope]')&&isMapScope(target.value)){this.mapRegion=null;this.q<HTMLElement>('[data-roam-map-back]').hidden=true;void this.initializeMap()}}
 popstate(){const url=new URL(location.href),tab=this.tabAt(url);if(tab)void this.loadTab(tab,url.searchParams)}
 hide(){this.interruptedTab||=Boolean(this.tabRequest);this.interruptedRefresh||=Boolean(this.refreshRequest);this.interruptedMap||=Boolean(this.mapRequest);this.tabRequest?.abort();this.tabRequest=undefined;this.refreshRequest?.abort();this.refreshRequest=undefined;this.stopMap();this.copyGeneration++;this.stopTimer(this.copyTimer);this.copyTimer=undefined;const button=this.element.querySelector<HTMLElement>('[data-roam-copy]');if(button){this.icon(button,'link');requiredElement(button,'span').textContent='Copy link'}this.content.classList.remove('roam-tab-loading');this.content.removeAttribute('aria-busy');super.hide()}
 show(event:PageTransitionEvent){if(!event.persisted)return;if(this.interruptedRefresh){this.interruptedRefresh=this.interruptedTab=this.interruptedMap=false;void this.refresh()}else if(this.interruptedTab){this.interruptedTab=this.interruptedMap=false;const tab=this.tabAt(new URL(location.href))??this.pendingTab?.tab??this.tab();void this.loadTab(tab,new URL(location.href).searchParams)}else if(this.interruptedMap){this.interruptedMap=false;void this.initializeMap()}}
}
registerController('roam-report',RoamReportController)
