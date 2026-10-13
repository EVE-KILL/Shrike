// Server-rendered cursor pages and shared live rows for private killboards.
import {Controller} from '@hotwired/stimulus'
import {createRelayConnection,matchesStreamKillType} from './account-data'
import {liveRow,parseLiveKill,friendlyDay,element} from './kill-row'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'
import {isRecord} from './shared/json'
import {BrowserRequestError} from './browser-request'
import type {TrackerTargetType} from '../models/trackers'

export interface PrivateKillFeedOptions {
  endpoint:string;topics?:string[];lossType?:TrackerTargetType|'';lossId?:number;params?:Record<string,string>
}
type Configuration={options:PrivateKillFeedOptions;active:boolean}
const configurations=new WeakMap<HTMLElement,Configuration>()
const controllers=new WeakMap<HTMLElement,PrivateKillFeedController>()
const actions='click->private-killfeed#click pagehide@window->private-killfeed#hide pageshow@window->private-killfeed#show'

export function initializePrivateKillFeed(host:HTMLElement,options:PrivateKillFeedOptions){
  const config={options,active:true};configurations.set(host,config)
  const names=new Set((host.dataset.controller??'').split(/\s+/).filter(Boolean));names.add('private-killfeed');host.dataset.controller=[...names].join(' ')
  if(!host.dataset.action?.includes('click->private-killfeed#click'))host.dataset.action=[host.dataset.action,actions].filter(Boolean).join(' ')
  controllers.get(host)?.configure(config)
  return {
    refresh(){if(config.active&&configurations.get(host)===config)controllers.get(host)?.refresh()},
    destroy(){config.active=false;if(configurations.get(host)===config)controllers.get(host)?.dispose()},
  }
}
export class PrivateKillFeedController extends Controller<HTMLElement> {
  private feed:ReturnType<typeof createPrivateKillFeed>|undefined
  connect(){controllers.set(this.element,this);const config=configurations.get(this.element);if(config)this.configure(config)}
  disconnect(){this.dispose();controllers.delete(this.element)}
  configure(config:Configuration){this.dispose();if(config.active)this.feed=createPrivateKillFeed(this.element,config.options)}
  dispose(){this.feed?.destroy();this.feed=undefined}
  refresh(){void this.feed?.refresh()}
  click(event:MouseEvent){this.feed?.click(event)}
  hide(){this.feed?.hide()}
  show(){this.feed?.show()}
}
registerController('private-killfeed',PrivateKillFeedController)

function createPrivateKillFeed(host:HTMLElement,{endpoint,topics=[],lossType='',lossId=0,params={}}:PrivateKillFeedOptions){
  let page=1,limit=50,cursors:Array<string|null>=[null],request:AbortController|null=null,disposed=false,newCount=0,resumeRequest=false
  const list=()=>host.querySelector<HTMLElement>('.kill-list')
  const connected=(value:boolean)=>{const dot=list()?.querySelector<HTMLElement>('.kill-live-dot');if(dot){dot.classList.toggle('connected',value);dot.title=value?'Live — connected':'Disconnected';dot.setAttribute('aria-label',dot.title)}}
  const count=()=>{const button=list()?.querySelector<HTMLButtonElement>('.new-kills');if(button){button.hidden=!newCount;requiredElement<HTMLElement>(button,'span').textContent=String(newCount)}}
  function times(){for(const time of host.querySelectorAll<HTMLTimeElement>('time[datetime]')){const minutes=Math.floor((Date.now()-Date.parse(time.dateTime))/60000);time.textContent=minutes<1?'just now':minutes<30?`${minutes}m ago`:time.dateTime.slice(11,16)}}
  const lossKeys:Partial<Record<TrackerTargetType,string>>={character:'victimCharacterId',corporation:'victimCorporationId',alliance:'victimAllianceId'}
  function loss(row:HTMLElement){const key=lossType?lossKeys[lossType]:undefined;if(key&&Number(row.dataset[key])===lossId)row.classList.add('loss-row','loss-row-hover')}
  function reconcile(){
    const root=list(),container=root?.querySelector<HTMLElement>('.kill-rows');if(!root||!container)return
    container.querySelectorAll('.kill-day-divider').forEach(divider=>divider.remove());let previous=''
    const rows=[...container.querySelectorAll<HTMLElement>('.kill-row')]
    for(const [i,row]of rows.entries()){
      row.classList.toggle('alternate',i%2===0);loss(row)
      const date=row.querySelector<HTMLTimeElement>('time')?.dateTime||'',day=date.slice(0,10)
      if(previous&&day&&day!==previous){const divider=element('div','kill-day-divider');divider.append(element('span'),element('strong','',friendlyDay(date)),element('span'));container.insertBefore(divider,row)}previous=day
    }
    const first=rows[0]?.querySelector<HTMLTimeElement>('time')?.dateTime
    if(first)requiredElement<HTMLElement>(root,'.kill-day-heading').textContent=friendlyDay(first);times()
  }
  function failure(error:unknown){
    const message=element('div','glass-panel p-5 text-sm text-red-300',error instanceof BrowserRequestError&&error.status===401?'Your session expired. Log in again to continue.':'Could not load recorded activity. ')
    message.setAttribute('role','alert');const retry=element('button','ml-2 text-xs text-blue-400 hover:text-blue-300','Retry');retry.type='button';retry.dataset.privateKillRetry='';message.append(retry)
    host.querySelector('[data-private-kill-error]')?.remove();message.dataset.privateKillError='';if(list())host.prepend(message);else host.replaceChildren(message)
  }
  async function load(nextPage=page,nextLimit=limit){
    if(disposed)return
    request?.abort();const controller=new AbortController();request=controller;host.setAttribute('aria-busy','true');host.querySelector('[data-private-kill-error]')?.remove()
    if(!list())host.replaceChildren(element('div','glass-panel p-8 text-center text-sm text-gray-500','Loading kills…'))
    const query=new URLSearchParams({...params,page:String(nextPage),limit:String(nextLimit),live:topics.length?'1':'0',lossType,lossId:String(lossId)})
    const cursor=cursors[nextPage-1];if(nextPage>1&&cursor)query.set('after',cursor)
    try{
      const response=await fetch(endpoint+'?'+query,{credentials:'same-origin',signal:controller.signal})
      if(!response.ok)throw new BrowserRequestError('Could not load recorded activity',response.status)
      const parsed=new DOMParser().parseFromString(await response.text(),'text/html'),next=parsed.querySelector<HTMLElement>('.kill-list')
      if(!next)throw new Error('Missing killlist')
      if(disposed||request!==controller)return
      host.replaceChildren(next);page=nextPage;limit=nextLimit;cursors[page]=next.dataset.nextCursor||null;newCount=0;count();connected(relay?.connected??false);reconcile()
    }catch(error){if(request===controller&&!disposed)failure(error)}
    finally{if(request===controller){request=null;host.removeAttribute('aria-busy')}}
  }
  const relay=topics.length?createRelayConnection({wsUrl:new URL('/ws',location.href).href,path:'/killlist',reconnect:true,visibilityPause:true,shouldReconnect:()=>!disposed,onState:connected,
    onOpen:socket=>socket.send(JSON.stringify({action:'subscribe',topics})),
    onMessage:(channel,data)=>{
      const kill=channel==='killlist'&&isRecord(data)?parseLiveKill(data.killmail):null
      if(!kill||disposed||!matchesStreamKillType(kill,params.type))return
      const root=list();if(!root||page!==1||host.getAttribute('aria-busy')==='true'){newCount++;count();return}
      const row=liveRow(kill);if(!row||root.querySelector(`[data-kill-id="${row.dataset.killId}"]`))return
      let rows=root.querySelector<HTMLElement>('.kill-rows')
      if(!rows){root.querySelector('.kill-empty')?.remove();rows=element('div','kill-rows');const head=element('div','kill-head');for(const title of ['Ship','Victim','Final Blow','Location','Details'])head.append(element('span','',title));rows.append(head);root.append(rows)}
      const current=[...rows.querySelectorAll<HTMLElement>('.kill-row')];if(current.length&&Number(row.dataset.killId)<=Number(current[0].dataset.killId))return
      rows.insertBefore(row,current[0]||null);for(const old of [...rows.querySelectorAll('.kill-row')].slice(limit))old.remove();reconcile()
      if(!root.querySelector('.kill-bottom-pager')){const bottom=element('div','kill-bottom-pager');bottom.append(requiredElement(root,'.kill-pager').cloneNode(true));root.append(bottom)}
    }}):null
  function click(event:MouseEvent){
    const target=eventElement(event.target)?.closest<HTMLButtonElement>('button');if(!target||target.disabled)return
    if('privateKillRetry'in target.dataset){void load();return}
    if(target.matches('[data-kill-limit]')){cursors=[null];void load(1,Number(target.dataset.killLimit));return}
    if(target.matches('[data-kill-page]')){const next=Number(target.dataset.killPage);if(next>=1&&(next===1||cursors[next-1]))void load(next);return}
    if(target.matches('.new-kills')){cursors=[null];void load(1)}
  }
  const timer=setInterval(times,30000);relay?.connect();void load()
  return {
    refresh:()=>load(),click,
    hide(){resumeRequest=Boolean(request);request?.abort();request=null;host.removeAttribute('aria-busy');relay?.disconnect()},
    show(){if(!disposed){relay?.connect();times();if(!list()||resumeRequest)void load();resumeRequest=false}},
    destroy(){disposed=true;request?.abort();request=null;relay?.dispose();clearInterval(timer);host.removeAttribute('aria-busy')},
  }
}
