import './comment-thread'
import './battle-composition'
import './battle-replay'
import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {eventElement} from './shared/dom'
import {isRecord,isAbortError,errorMessage} from './shared/json'
import {readBattleSides} from './shared/battle-model'
import type {BattleSide} from '../utils/battleExplorer'

const tabs=['summary','kills','timeline','composition','intel','replay','comments'] as const
type BattleTab=typeof tabs[number]
function tabAt(url:URL):BattleTab|undefined{const value=url.pathname.split('/')[3]||'summary';return tabs.find(tab=>tab===value)}
export class BattleDetailController extends RequestComponent {
 private tabRequest:AbortController|undefined
 private killRequest:AbortController|undefined
 private killLimit=50
 private cursors:Array<string|null>=[null]
 private sides:BattleSide[]=[]
 private measure:'count'|'isk'='count'
 private interrupted=false
 private get content(){return this.q<HTMLElement>('[data-battle-content]')}
 private get id(){return this.element.dataset.battleId??''}
 connect(){try{const raw:unknown=JSON.parse(this.q('#battle-header-data').textContent??'null');this.sides=readBattleSides(isRecord(raw)?raw.team_entities:[])}catch{/* Rows remain usable without optional team accents. */}this.initializeTab()}
 private tintRows(){for(const row of this.content.querySelectorAll<HTMLElement>('.kill-row')){const corporation=Number(row.dataset.victimCorporationId),alliance=Number(row.dataset.victimAllianceId),side=this.sides.findIndex(side=>side.corps.includes(corporation)||side.alliances.includes(alliance));if(side<0)continue;const color=side===0?'var(--color-error)':'var(--color-brand-primary)';row.style.borderLeft=`2px solid color-mix(in srgb,${color} 40%,transparent)`;row.style.backgroundColor=`color-mix(in srgb,${color} 3%,transparent)`}}
 private initializeTab(){
  this.tintRows();const metric=this.content.querySelector<HTMLSelectElement>('[data-battle-measure]');if(metric){metric.value=this.measure;this.measureBars()}
  this.cursors=[null];const list=this.content.querySelector<HTMLElement>('#kill-list'),page=Number(list?.dataset.page)||1,limit=Number(list?.dataset.limit)||50;this.killLimit=[10,25,50,100].includes(limit)?limit:50;if(page>1)this.cursors[page-1]=new URLSearchParams(location.search).get('after');this.cursors[page]=list?.dataset.nextCursor||null
 }
 private measureBars(){for(const bar of this.content.querySelectorAll<HTMLElement>('[data-count-height]')){const value=Number(this.measure==='isk'?bar.dataset.iskHeight:bar.dataset.countHeight);bar.style.height=`${Number.isFinite(value)?Math.min(100,Math.max(0,value)):0}%`}}
 private cancelKill(){this.killRequest?.abort();this.killRequest=undefined;const list=this.content.querySelector<HTMLElement>('[data-battle-killfeed]');list?.classList.remove('loading');list?.removeAttribute('aria-busy')}
 private notice(scope:HTMLElement,error:unknown,kind:'tab'|'kill',page=1,limit=this.killLimit){
  scope.querySelector('[data-battle-request-error]')?.remove();const box=document.createElement('div'),message=document.createElement('p'),retry=document.createElement('button');box.className='glass-panel p-5 text-red-300';box.dataset.battleRequestError=kind;message.setAttribute('role','alert');message.textContent=errorMessage(error,kind==='tab'?'Unable to load battle section.':'Unable to load battle losses.');retry.type='button';retry.className='underline';retry.textContent='Retry';retry.dataset.battleRetry=kind;retry.dataset.page=String(page);retry.dataset.limit=String(limit);box.append(message,retry);scope.append(box)
 }
 private async navigate(url:URL,push=true){
  const tab=tabAt(url);if(!tab||url.origin!==location.origin||url.pathname.split('/')[2]!==this.id)return
  this.tabRequest?.abort();this.cancelKill();const request=new AbortController();this.tabRequest=request;const content=this.content,sameTab=tab===this.element.dataset.battleTab,focused=document.activeElement,focusName=sameTab&&(focused instanceof HTMLSelectElement||focused instanceof HTMLInputElement)&&focused.closest('[data-battle-filters]')?focused.name:undefined
  if(!sameTab)this.measure='count';content.querySelector('[data-battle-request-error="tab"]')?.remove();content.classList.add('loading');content.setAttribute('aria-busy','true');if(push)history.pushState(null,'',url)
  try{
   const text=await this.text(`/_fragments/battle/${encodeURIComponent(this.id)}/${tab}${url.search}`,'Unable to load battle section.',{signal:request.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('[data-battle-fragment]');if(!next)throw new Error('Battle section is unavailable.');if(this.tabRequest!==request)return
   content.replaceChildren(document.importNode(next,true));this.element.dataset.battleTab=tab;const label=this.element.querySelector<HTMLElement>(`[data-battle-tab-link="${tab}"]`)?.textContent?.trim()??tab;document.title=`${this.element.dataset.battleTitle}${tab==='summary'?'':` (${label})`} - EVE-KILL.com`
   for(const link of this.all<HTMLAnchorElement>('[data-battle-tab-link]')){const active=link.dataset.battleTabLink===tab;for(const name of ['text-white','border-white'])link.classList.toggle(name,active);for(const name of ['text-gray-500','border-transparent'])link.classList.toggle(name,!active);link.setAttribute('aria-current',active?'page':'false');const target=new URL(link.href);target.search=url.search;link.href=target.href}
   this.initializeTab();if(focusName)this.content.querySelector<HTMLElement>(`[data-battle-filters] [name="${CSS.escape(focusName)}"]`)?.focus({preventScroll:true})
  }catch(error){if(this.tabRequest===request&&!isAbortError(error))this.notice(content,error,'tab')}
  finally{if(this.tabRequest===request){this.tabRequest=undefined;content.classList.remove('loading');content.removeAttribute('aria-busy')}}
 }
 private async killfeed(page:number,limit=this.killLimit){
  const list=this.content.querySelector<HTMLElement>('[data-battle-killfeed]');if(!list)return;this.cancelKill();const request=new AbortController();this.killRequest=request;list.querySelector('[data-battle-request-error]')?.remove();list.classList.add('loading');list.setAttribute('aria-busy','true')
  const url=new URL(location.href);url.searchParams.set('page',String(page));url.searchParams.set('limit',String(limit));const cursor=this.cursors[page-1];if(page>1&&cursor)url.searchParams.set('after',cursor);else url.searchParams.delete('after')
  try{
   const text=await this.text(`/_fragments/battle/${encodeURIComponent(this.id)}/killfeed${url.search}`,'Unable to load battle losses.',{signal:request.signal}),next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('#kill-list');if(!next)throw new Error('Killmail rows are unavailable.');const {replaceFragmentChildren}=await import('./fragment');if(this.killRequest!==request||!list.isConnected)return;replaceFragmentChildren(list,next);this.killLimit=limit;this.cursors[page]=next.dataset.nextCursor||null;this.tintRows()
  }catch(error){if(this.killRequest===request&&!isAbortError(error))this.notice(list,error,'kill',page,limit)}
  finally{if(this.killRequest===request){this.killRequest=undefined;list.classList.remove('loading');list.removeAttribute('aria-busy')}}
 }
 click(event:MouseEvent){
  const target=eventElement(event.target),link=target?.closest<HTMLAnchorElement>('[data-battle-tab-link],[data-battle-navigation]');if(link&&!event.defaultPrevented&&event.button===0&&!event.metaKey&&!event.ctrlKey&&!event.shiftKey&&!event.altKey){const url=new URL(link.href);if(url.origin!==location.origin||!tabAt(url)||url.pathname.split('/')[2]!==this.id)return;event.preventDefault();if(url.href!==location.href){const sameTab=tabAt(url)===this.element.dataset.battleTab;if(link.hasAttribute('data-battle-navigation')&&sameTab)history.replaceState(null,'',url);void this.navigate(url,!sameTab)}return}
  const retry=target?.closest<HTMLElement>('[data-battle-retry]');if(retry){if(retry.dataset.battleRetry==='kill')void this.killfeed(Number(retry.dataset.page),Number(retry.dataset.limit));else void this.navigate(new URL(location.href),false);return}
  const limit=target?.closest<HTMLElement>('[data-kill-limit]');if(limit){const value=Number(limit.dataset.killLimit);if(![10,25,50,100].includes(value))return;this.cursors=[null];void this.killfeed(1,value);return}
  const pager=target?.closest<HTMLButtonElement>('[data-kill-page]');if(pager&&!pager.disabled){const page=Number(pager.dataset.killPage);if(Number.isSafeInteger(page)&&page>=1&&(page===1||this.cursors[page-1]))void this.killfeed(page)}
 }
 change(event:Event){
  const target=eventElement(event.target),filter=target?.closest<HTMLFormElement>('[data-battle-filters]');if(filter){const url=new URL(location.href);for(const [key,value]of new FormData(filter)){if(typeof value!=='string')continue;if(value===''||key==='minIsk'&&value==='0')url.searchParams.delete(key);else url.searchParams.set(key,value)}url.searchParams.delete('page');url.searchParams.delete('after');history.replaceState(null,'',url);void this.navigate(url,false);return}
  if(target instanceof HTMLSelectElement&&target.matches('[data-battle-measure]')){this.measure=target.value==='isk'?'isk':'count';this.measureBars()}
 }
 submit(event:SubmitEvent){if(eventElement(event.target)?.matches('[data-battle-filters]'))event.preventDefault()}
 popstate(){if(location.pathname.split('/')[2]===this.id)void this.navigate(new URL(location.href),false)}
 show(event:PageTransitionEvent){if(event.persisted&&this.interrupted){this.interrupted=false;void this.navigate(new URL(location.href),false)}}
 hide(){this.interrupted||=Boolean(this.tabRequest)||Boolean(this.killRequest);this.tabRequest?.abort();this.tabRequest=undefined;this.cancelKill();this.content.classList.remove('loading');this.content.removeAttribute('aria-busy');super.hide()}
}
registerController('battle-detail',BattleDetailController)
