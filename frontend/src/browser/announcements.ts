import {escapeHTML} from './shared/html';
import {requestJSON,readBrowserSession} from './browser-request';
import {createRelayConnection} from './account-data.js';
import {iconHref} from './icon-hrefs.js';
import {ANNOUNCEMENT_DISMISSED_KEY,ANNOUNCEMENT_COLLAPSED_KEY,announcementLists,applyAnnouncementEvent,reconcileAnnouncementSnapshot,tickerFreshness,announcementBannerColors,announcementModalColors,announcementTickerColors} from './announcement-data.js';

import {Controller} from '@hotwired/stimulus';
import {registerController} from './controllers/application';
import {eventElement,requiredElement} from './shared/dom';
import {isRecord} from './shared/json';
import type {Announcement,AnnouncementsResponse,DismissedAnnouncementIdsResponse} from '../../shared/api/types.gen';
import type {AnnouncementEvent} from './announcement-data';
const e=escapeHTML;
const icon=(name:string|null,classes?:string)=>`<svg class="icon ${e(classes||'')}" aria-hidden="true"><use href="${e(iconHref(name||'info'))}"></use></svg>`;

function createAnnouncementSurfaces(root:HTMLElement){
  const banners=requiredElement<HTMLElement>(root,'[data-announcement-banners]'),dialog=requiredElement<HTMLDialogElement>(root,'[data-announcement-modal]'),ticker=requiredElement<HTMLElement>(root,'[data-announcement-ticker]');
  let items:readonly Announcement[]=[],dismissed=new Set<number>(),collapsed=false,active=true,refreshOnOpen=false;
  let expiryTimer:ReturnType<typeof setInterval>|null=null,freshnessTimer:ReturnType<typeof setInterval>|null=null,pending:AbortController|null=null,currentId:number|null=null,setWidth=0;
  let bannerHTML='',tickerHTML='',refreshEvents:[string,AnnouncementEvent][]=[];
  try{const stored:unknown=JSON.parse(localStorage.getItem(ANNOUNCEMENT_DISMISSED_KEY)||'[]');if(Array.isArray(stored))dismissed=new Set(stored.map(Number).filter(Boolean));collapsed=localStorage.getItem(ANNOUNCEMENT_COLLAPSED_KEY)==='1';}catch{}
  const scroll=requiredElement<HTMLElement>(ticker,'[data-announcement-scroll]'),observer=new ResizeObserver(resync);
  observer.observe(scroll);
  const footerObserver=new ResizeObserver(footerOffset);footerObserver.observe(ticker);
  function footerOffset(){document.documentElement.style.setProperty('--announcement-footer-height',`${ticker.hidden?0:ticker.getBoundingClientRect().height}px`);}
  function resync(){const width=scroll.querySelector<HTMLElement>('.ticker-set')?.scrollWidth||0;if(width===setWidth)return;let offset=0;if(width>0){const matrix=getComputedStyle(scroll).transform.match(/matrix\(([^)]+)\)/);if(matrix)offset=((-Number(matrix[1].split(',')[4])%width)+width)%width;}setWidth=width;if(!width)return;scroll.style.setProperty('--set-width',`${width}px`);scroll.style.setProperty('--marquee-duration',`${(width/80).toFixed(3)}s`);scroll.style.animationName='none';void scroll.offsetWidth;scroll.style.animationDelay=`-${(offset/80).toFixed(3)}s`;scroll.style.animationName='';}
  const style=<T,>(map:Record<string,T>,a:Announcement):T=>map[a.color]||map.info;
  function saveDismissals(){try{localStorage.setItem(ANNOUNCEMENT_DISMISSED_KEY,JSON.stringify([...dismissed]));}catch{}}
  async function dismiss(id:number){dismissed.add(id);saveDismissals();render();void readBrowserSession().then(identity=>{if(identity)return requestJSON(`/api/me/announcements/${id}/dismissal`,{method:'PUT'});}).catch(()=>{});}
  function toggle(){collapsed=!collapsed;try{localStorage.setItem(ANNOUNCEMENT_COLLAPSED_KEY,collapsed?'1':'0');}catch{}render();}
  function render(){
    const lists=announcementLists(items,dismissed),visible=lists.banners.slice(0,2);
    const nextBanner=visible.map(a=>{const s=style(announcementBannerColors,a);return `<div data-announcement-banner="${a.id}" class="relative rounded-lg border px-4 py-3 text-sm backdrop-blur-sm ${s.bg} ${s.border}"><div class="flex items-start gap-3">${icon(a.icon||s.icon,`text-base flex-shrink-0 mt-0.5 ${s.text}`)}<div class="flex-1 min-w-0"><div class="font-medium ${s.text}">${e(a.title)}</div>${a.body_html?`<div class="mt-1 text-xs text-gray-400 prose prose-invert prose-xs max-w-none">${a.body_html}</div>`:''}${a.link_url?`<a href="${e(a.link_url)}" class="inline-flex items-center gap-1 mt-2 text-xs font-medium hover:underline ${s.text}">${e(a.link_label||'Learn more')}${icon('arrow-right','text-[10px]')}</a>`:''}</div><button type="button" data-announcement-dismiss="${a.id}" aria-label="Dismiss ${e(a.title)}" title="Dismiss" class="flex items-center justify-center w-6 h-6 rounded-md text-gray-500 hover:text-white hover:bg-white/[0.06] transition-colors flex-shrink-0 cursor-pointer">${icon('x','text-sm')}</button></div></div>`;}).join('');
    banners.hidden=!visible.length;banners.classList.add('space-y-2','mb-4');if(nextBanner!==bannerHTML){banners.innerHTML=nextBanner;bannerHTML=nextBanner;}
    const current=lists.modals[0];
    if(!current){dialog.close();currentId=null;}
    else {
      const s=style(announcementModalColors,current);currentId=current.id;
      const header=`${icon('circle-alert',`text-lg flex-shrink-0 ${s.accent}`)}<h2 class="text-sm font-semibold text-white">${e(current.title)}</h2>`,body=`${current.body_html?`<div class="text-sm text-gray-300 prose prose-invert prose-sm max-w-none">${current.body_html}</div>`:`<p class="text-sm text-gray-400">${e(current.title)}</p>`}${current.link_url?`<a href="${e(current.link_url)}" class="inline-flex items-center gap-1 mt-3 text-sm font-medium hover:underline ${s.accent}">${e(current.link_label||'Learn more')}${icon('external-link','text-xs')}</a>`:''}`;
      for(const [name,html]of [['header',header],['body',body]] as const){const node=requiredElement<HTMLElement>(dialog,`[data-announcement-modal-${name}]`);if(node.innerHTML!==html)node.innerHTML=html;}
      dialog.setAttribute('aria-label',current.title);dialog.dataset.announcementId=String(current.id);requiredElement<HTMLElement>(dialog,'[data-announcement-modal-count]').textContent=lists.modals.length>1?`${lists.modals.length} alerts remaining`:'';requiredElement<HTMLElement>(dialog,'[data-announcement-ack]').className='px-4 py-2 rounded-lg text-sm font-medium text-white transition-colors cursor-pointer '+s.button;
      if(!dialog.open)dialog.showModal();
    }
    ticker.hidden=!lists.ticker.length;requiredElement<HTMLElement>(ticker,'[data-announcement-expand]').hidden=!collapsed;requiredElement<HTMLElement>(ticker,'[data-announcement-ticker-expanded]').hidden=collapsed;
    const markup=[1,2].map(pass=>`<span class="ticker-set" data-announcement-pass="${pass}"${pass===2?' aria-hidden="true"':''}>${lists.ticker.map(a=>{const s=style(announcementTickerColors,a);return `<span data-announcement-ticker-item="${a.id}" class="inline-flex items-center gap-1.5 text-xs mx-3 px-2.5 py-1 rounded-md border transition-opacity duration-1000 ${s.bg}">${icon(a.icon||'info',`text-[10px] flex-shrink-0 ${s.text}`)}${a.link_url?`<a href="${e(a.link_url)}"${pass===2?' tabindex="-1"':''} class="hover:underline ${s.text}">`:'<span>'}<strong class="${s.text}">${e(a.title)}</strong>${a.body_md?`<span class="text-gray-400 ml-1">— ${e(a.body_md)}</span>`:''}${a.link_url?'</a>':'</span>'}<button type="button" data-announcement-dismiss="${a.id}" aria-label="Dismiss ${e(a.title)}" title="Dismiss"${pass===2?' tabindex="-1"':''} class="text-gray-600 hover:text-gray-400 transition-colors cursor-pointer ml-1">${icon('x','text-[10px]')}</button></span>`;}).join('')}</span>`).join('');
    if(markup!==tickerHTML){const focused=eventElement(document.activeElement)?.closest<HTMLElement>('[data-announcement-dismiss]')?.dataset.announcementDismiss;scroll.innerHTML=markup;tickerHTML=markup;resync();if(focused)scroll.querySelector<HTMLElement>(`[data-announcement-dismiss="${Number(focused)}"]`)?.focus();}freshness();footerOffset();
  }
  function freshness(){const byID=new Map(items.map(a=>[String(a.id),a]));for(const node of scroll.querySelectorAll<HTMLElement>('[data-announcement-ticker-item]')){const item=byID.get(node.dataset.announcementTickerItem??'');if(item)node.style.opacity=String(tickerFreshness(item));}}
  async function refresh(){
    pending?.abort();const controller=new AbortController();pending=controller;refreshEvents=[];
    try {
      const [result,dismissals]=await Promise.all([requestJSON<AnnouncementsResponse>('/api/announcements',{cache:'no-cache',signal:controller.signal}),readBrowserSession().then(identity=>{return identity?requestJSON<DismissedAnnouncementIdsResponse>('/api/me/announcements/dismissed',{signal:controller.signal}).catch(()=>null):null;}).catch(()=>null)]);
      if(pending!==controller||!active)return;items=reconcileAnnouncementSnapshot(result.announcements||[],refreshEvents);for(const id of dismissals?.dismissedIds||[])dismissed.add(id);if(dismissals?.dismissedIds?.length)saveDismissals();render();
    }catch{}finally{if(pending===controller)pending=null;}
  }
  const relay=createRelayConnection({wsUrl:new URL('/ws',location.href).href,path:'/announcements',reconnect:true,shouldReconnect:()=>active&&!document.hidden,onOpen:()=>{if(refreshOnOpen)void refresh();refreshOnOpen=true;},onMessage:(channel,raw)=>{const data=readAnnouncementEvent(raw);if(!data)return;if(pending)refreshEvents.push([channel,data]);const next=applyAnnouncementEvent(items,channel,data);if(next===items)return;items=next;render();}});
  function stop(){relay.disconnect();if(expiryTimer!==null)clearInterval(expiryTimer);if(freshnessTimer!==null)clearInterval(freshnessTimer);expiryTimer=null;freshnessTimer=null;pending?.abort();pending=null;}
  function start(){if(!active||document.hidden)return;refreshOnOpen=false;void refresh();relay.connect();expiryTimer??=setInterval(()=>{const cutoff=Date.now()-60000;items=items.filter(a=>new Date(a.expires_at).getTime()>cutoff);render();},10000);freshnessTimer??=setInterval(freshness,15000);}
  return {
    start,
    visibility(){document.hidden?stop():start();},
    pageHide(){active=false;stop();observer.disconnect();footerObserver.disconnect();},
    pageShow(event:PageTransitionEvent){if(event.persisted){active=true;observer.observe(scroll);footerObserver.observe(ticker);start();}},
    cancel(event:Event){event.preventDefault();},
    acknowledge(){if(currentId!=null)void dismiss(currentId);},
    dismissClick(event:MouseEvent){const button=eventElement(event.target)?.closest<HTMLElement>('[data-announcement-dismiss]');if(button){event.preventDefault();void dismiss(Number(button.dataset.announcementDismiss));}},
    toggle,
    dispose(){active=false;stop();relay.dispose();observer.disconnect();footerObserver.disconnect();dialog.close();document.documentElement.style.removeProperty('--announcement-footer-height');},
  };
}

export class AnnouncementsController extends Controller<HTMLElement>{
  private surfaces:ReturnType<typeof createAnnouncementSurfaces>|undefined;
  connect(){this.surfaces=createAnnouncementSurfaces(this.element);this.surfaces.start();}
  disconnect(){this.surfaces?.dispose();this.surfaces=undefined;}
  visibility(){this.surfaces?.visibility();}
  pageHide(){this.surfaces?.pageHide();}
  pageShow(event:PageTransitionEvent){this.surfaces?.pageShow(event);}
  cancel(event:Event){this.surfaces?.cancel(event);}
  acknowledge(){this.surfaces?.acknowledge();}
  dismissClick(event:MouseEvent){this.surfaces?.dismissClick(event);}
  toggle(){this.surfaces?.toggle();}
}
registerController('announcements',AnnouncementsController);

export function readAnnouncementEvent(raw:unknown):AnnouncementEvent {
  if(!isRecord(raw)||typeof raw.event_type!=='string'||!isRecord(raw.announcement))return null;
  const a=raw.announcement;
  if(typeof a.id!=='number'||!Number.isSafeInteger(a.id)||!a.id)return null;
  if(raw.event_type==='expired'||raw.event_type==='archived')return {event_type:raw.event_type,announcement:{id:a.id}};
  if(raw.event_type!=='new'&&raw.event_type!=='updated')return null;
  if(typeof a.title!=='string'||typeof a.body_html!=='string'||typeof a.body_md!=='string'||
    typeof a.created_at!=='string'||typeof a.starts_at!=='string'||typeof a.expires_at!=='string'||
    (a.tier!==1&&a.tier!==2&&a.tier!==3)||(a.color!=='info'&&a.color!=='warning'&&a.color!=='danger'&&a.color!=='success')||
    !(a.icon===null||typeof a.icon==='string')||!(a.link_url===null||typeof a.link_url==='string')||!(a.link_label===null||typeof a.link_label==='string'))return null;
  return {event_type:raw.event_type,announcement:{id:a.id,title:a.title,body_html:a.body_html,body_md:a.body_md,
    created_at:a.created_at,starts_at:a.starts_at,expires_at:a.expires_at,tier:a.tier,color:a.color,icon:a.icon,link_url:a.link_url,link_label:a.link_label}};
}
