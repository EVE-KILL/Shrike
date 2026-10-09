import {escapeHTML} from './shared/html';
import {esiRowTintClass,esiStatusDotClass,esiStatusBadgeClass,esiTypeChipClass,esiVolumeSlots,mergeEsiLogRows,fmt,fmtDate} from './esi-data.js';
import {requestJSON} from './browser-request.js';
import {createSelectMenu,type SelectOption,type SelectMenu} from './select-menu';
import {createEntitySearchSelect,readEntitySelection} from './entity-search-select';

import {Controller} from '@hotwired/stimulus';
import {registerController} from './controllers/application';
import {eventElement,requiredElement} from './shared/dom';
import {errorMessage} from './shared/json';
import {readEsiLogSeed,readEsiVolumeSeed,parseEsiLogData} from './shared/esi-seeds';
import type {EsiLogRow} from '../utils/esiLog';
import type {AdminEsiLogsResponses} from '../../shared/api/types.gen';
const e=escapeHTML;
const icon=(name:string)=>`<svg class="icon" aria-hidden="true"><use href="/assets/icons.svg#${name}"></use></svg>`;
const entityImage=(kind:string,id:number|null|undefined,size:number)=>`https://images.evetech.net/${kind==='corporation'?'corporations':'characters'}/${Number(id)}/${kind==='corporation'?'logo':'portrait'}?size=${size}`;
const hour=(at:number)=>`${String(new Date(at).getUTCHours()).padStart(2,'0')}:00`;

export function renderEsiVolume(root:HTMLElement) {
  const slots=esiVolumeSlots(readEsiVolumeSeed(root.dataset.hours),root.dataset.newItems==='true');
  if(!slots.length){root.replaceChildren();return;}
  root.innerHTML=`<div class="glass-panel p-5"><h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">${e(root.dataset.title||'Request Volume (24h)')}</h2><div class="flex items-end gap-px" style="height:96px">${slots.map(s=>`<div title="${e(s.tooltip)}" data-esi-volume-slot="${s.at}" class="flex-1 flex flex-col justify-end gap-px min-w-0 h-full">${s.empty?'<div class="bg-white/[0.06] rounded-t-sm" style="height:1px"></div>':`${s.errors?`<div class="bg-red-500/60 rounded-t-sm" style="height:${s.errorHeight}px"></div>`:''}<div class="bg-blue-500/40 rounded-t-sm" style="height:${s.okHeight}px"></div>`}</div>`).join('')}</div><div class="flex justify-between text-fine text-gray-600 mt-1"><span>${hour(slots[0].at)}</span><span class="flex items-center gap-3"><span class="flex items-center gap-1"><span class="w-2 h-2 rounded-sm bg-blue-500/40"></span> OK</span><span class="flex items-center gap-1"><span class="w-2 h-2 rounded-sm bg-red-500/60"></span> Errors</span></span><span>${hour(slots.at(-1)?.at??slots[0].at)}</span></div></div>`;
}

function createEsiLog(root:HTMLElement) {
  const endpoint=root.dataset.endpoint??'/api/user/esi-logs',showEntity=root.dataset.showEntity==='true',admin=root.dataset.admin==='true';
  let data=readEsiLogSeed(root.dataset.initial),active=true;
  let page=1,live=true,filterRequest:AbortController|null=null,pollRequest:AbortController|null=null,timer:ReturnType<typeof setInterval>|null=null,version=0,resumeLoad=false;
  const filters:Record<string,string>={endpoint_type:'',source:'',status:'',...(admin?{search:'',character_id:'',corporation_id:'',has_new:''}:{})},menus:Array<[string,SelectMenu]>=[];
  const dialog=requiredElement<HTMLDialogElement>(root,'[data-esi-log-dialog]');
  const status=(message:string)=>{const box=requiredElement<HTMLElement>(root,'[data-esi-log-error]');box.hidden=!message;box.innerHTML=message?`${e(message)} <button type="button" data-esi-log-retry class="text-blue-400">Retry</button>`:'';};
  const filterRoot=requiredElement<HTMLElement>(root,'[data-esi-log-filters]');
  let entityPicker:ReturnType<typeof createEntitySearchSelect>|null=null,searchTimer:ReturnType<typeof setTimeout>|null=null;
  if(admin){
    let selected=null;try{selected=readEntitySelection(JSON.parse(root.dataset.entity||'null'));}catch{}if(selected)filters[selected.type==='corporation'?'corporation_id':'character_id']=String(selected.id);
    entityPicker=createEntitySearchSelect('Character / Corp...',value=>{filters.character_id=value?.type==='character'?String(value.id):'';filters.corporation_id=value?.type==='corporation'?String(value.id):'';page=1;load();},selected);filterRoot.append(entityPicker.root);

  }
  const choices:Array<[string,string,SelectOption[]]>=[['endpoint_type','Endpoint type',[{value:'',label:admin?'All endpoints':'All types'},{value:'character',label:'Character'},{value:'corporation',label:'Corporation'}]],['source','Source',[{value:'',label:'All sources'},...(data.sources||[]).map(value=>({value,label:value}))]],['status','Status',[{value:'',label:'All status'},{value:'success',label:'Success'},{value:'error',label:admin?'Errors':'Error'}]]];
  for(const [key,label,options] of choices){const menu=createSelectMenu(label,options,'',value=>{filters[key]=value;page=1;load();});filterRoot.append(menu.root);menus.push([key,menu]);}
  let hasNew:HTMLButtonElement|null=null;
  if(admin){hasNew=document.createElement('button');hasNew.type='button';hasNew.dataset.esiLogHasNew='';hasNew.innerHTML=`<span class="text-xs">${icon('sparkles')}</span> Has new`;filterRoot.append(hasNew);}
  const clear=document.createElement('button');clear.type='button';clear.innerHTML=admin?`<span class="text-xs">${icon('x')}</span> Clear`:'Clear';clear.hidden=true;clear.className=admin?'flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors cursor-pointer':'text-xs text-gray-500 hover:text-blue-400 transition-colors';clear.dataset.esiLogClear='';function clearFilters(){if(searchTimer!==null)clearTimeout(searchTimer);searchTimer=null;for(const key of Object.keys(filters))filters[key]='';for(const [,menu]of menus)menu.setValue('');entityPicker?.setValue(null);if(admin)requiredElement<HTMLInputElement>(root,'[data-esi-log-search]').value='';page=1;void load();}if(admin)requiredElement<HTMLElement>(root,'[data-esi-log-actions]').prepend(clear);else filterRoot.append(clear);
  function query(extra:Record<string,string>={}){return new URLSearchParams({...filters,page:String(page),...(admin?{limit:'50'}:{}),...extra});}
  function entity(row:EsiLogRow,mobile=false){return row.endpoint_entity_name&&row.endpoint_type&&row.endpoint_entity_id!=null?`<div class="flex items-center gap-1.5 ${mobile?'min-w-0':'text-xs text-gray-300'}"><img src="${entityImage(row.endpoint_type,row.endpoint_entity_id,32)}" alt="" width="16" height="16" class="w-4 h-4 rounded-full flex-shrink-0"><span class="${mobile?'text-fine text-gray-400 truncate':'truncate max-w-[180px]'}">${e(row.endpoint_entity_name)}</span></div>`:`<span class="${mobile?'text-fine':'text-xs'} text-gray-700">${e(row.endpoint_entity_id||'—')}</span>`;}
  function rowAttrs(row:EsiLogRow){return `data-esi-log-row="${Number(row.id)}" role="button" tabindex="0" aria-label="View ESI request ${Number(row.id)}"`;}
  function render() {
    clear.hidden=!Object.values(filters).some(Boolean);
    requiredElement<HTMLElement>(root,'[data-esi-log-count]').textContent=fmt(data.total)+' rows';
    const toggle=requiredElement<HTMLElement>(root,'[data-esi-log-live]'),hint=page!==1?'Auto-reload paused (not on page 1)':live?'Auto-reload ON':'Auto-reload OFF';
    toggle.title=hint;toggle.setAttribute('aria-label',hint);toggle.setAttribute('aria-pressed',String(live));toggle.className=admin?'flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer '+(live&&page===1?'bg-green-500/15 text-green-400 border border-green-500/20':'bg-white/[0.04] text-gray-500 border border-white/[0.06]'):'w-6 h-6 flex items-center justify-center rounded transition-colors '+(live&&page===1?'text-green-400 bg-green-500/10':'text-gray-600 hover:text-gray-400');requiredElement<SVGUseElement>(toggle,'use').setAttribute('href','/assets/icons.svg#'+(live?'play':'pause'));
    if(hasNew){hasNew.className='flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer border '+(filters.has_new?'bg-blue-500/15 text-blue-400 border-blue-500/20':'bg-white/[0.04] text-gray-400 border-white/[0.06] hover:text-blue-400');hasNew.title=filters.has_new?'Showing only requests with new data':'Show only requests with new data';hasNew.setAttribute('aria-pressed',String(!!filters.has_new));}
    const rows=data.rows||[],empty='<div class="px-5 py-8 text-center text-gray-600 text-sm">No ESI requests found.</div>';
    const headings=['Time','Type','Action',...(showEntity?['Entity']:[]),'New','Status'];
    const desktop=rows.length?`<table class="w-full text-sm"><thead><tr class="border-b border-white/[0.06]">${headings.map(label=>`<th class="${label==='New'?'text-right':label==='Status'?'text-center w-[50px]':'text-left'} px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">${label}</th>`).join('')}</tr></thead><tbody>${rows.map(row=>`<tr ${rowAttrs(row)} class="border-b border-white/[0.04] hover:bg-blue-500/[0.04] cursor-pointer transition-colors ${esiRowTintClass(row)}"><td class="px-4 py-2 text-gray-500 text-xs tabular-nums whitespace-nowrap">${fmtDate(row.created_at)}</td><td class="px-4 py-2"><span class="text-xs px-1.5 py-0.5 rounded ${esiTypeChipClass(row.endpoint_type)}">${e(row.endpoint_type)}</span></td><td class="px-4 py-2 text-gray-300 text-xs">${e(row.endpoint_action||'—')}</td>${showEntity?`<td class="px-4 py-2">${entity(row)}</td>`:''}<td class="px-4 py-2 text-right tabular-nums text-xs ${(row.new_items??0)>0?'text-blue-400':'text-gray-700'}">${e(row.new_items??0)}</td><td class="px-4 py-2 text-center"><span class="inline-block w-2 h-2 rounded-full ${esiStatusDotClass(row)}"></span></td></tr>`).join('')}</tbody></table>`:empty;
    const mobile=rows.length?rows.map(row=>`<div ${rowAttrs(row)} class="px-3 py-2.5 border-b border-white/[0.04] active:bg-white/[0.03] transition-colors ${esiRowTintClass(row)}"><div class="flex items-center justify-between gap-2 mb-1"><div class="flex items-center gap-2 min-w-0"><span class="inline-block w-2 h-2 rounded-full flex-shrink-0 ${esiStatusDotClass(row)}"></span><span class="text-xs text-gray-300 truncate">${e(row.endpoint_action||'—')}</span><span class="text-fine px-1.5 py-0.5 rounded flex-shrink-0 ${esiTypeChipClass(row.endpoint_type)}">${e(row.endpoint_type)}</span></div>${(row.new_items??0)>0?`<span class="text-fine text-blue-400 tabular-nums flex-shrink-0">+${e(row.new_items)}</span>`:''}</div><div class="flex items-center gap-2 ${showEntity?'justify-between':'justify-end'}">${showEntity?entity(row,true):''}<span class="text-fine text-gray-600 tabular-nums flex-shrink-0">${fmtDate(row.created_at)}</span></div></div>`).join(''):empty;
    requiredElement<HTMLElement>(root,'[data-esi-log-table]').innerHTML=`<div><div class="hidden md:block overflow-x-auto">${desktop}</div><div class="md:hidden">${mobile}</div></div>`;
    requiredElement<HTMLElement>(root,'[data-esi-log-pagination]').hidden=!(data.pages>1);requiredElement<HTMLElement>(root,'[data-esi-log-page-label]').textContent=`Page ${page} of ${fmt(data.pages)}`;
    requiredElement<HTMLButtonElement>(root,'[data-esi-log-page="prev"]').disabled=page<=1;requiredElement<HTMLButtonElement>(root,'[data-esi-log-page="next"]').disabled=page>=data.pages;
  }
  async function load() {
    if(!active)return;
    filterRequest?.abort();pollRequest?.abort();version++;
    const controller=new AbortController();filterRequest=controller;root.setAttribute('aria-busy','true');
    status('');render();
    try{const result=await requestJSON<AdminEsiLogsResponses[200]>(endpoint+'?'+query(),{signal:controller.signal});if(filterRequest!==controller||!active||!root.isConnected)return;data=parseEsiLogData(result);menus.find(([key])=>key==='source')?.[1].setOptions([{value:'',label:'All sources'},...(data.sources||[]).map(value=>({value,label:value}))]);render();}
    catch(error){if(filterRequest===controller&&active)status(errorMessage(error));}
    finally{if(filterRequest===controller){filterRequest=null;root.removeAttribute('aria-busy');}}
  }
  function field(label:string,value:string|number|null|undefined,tone='text-gray-300'){return `<div><div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">${label}</div><div class="text-xs tabular-nums ${tone}">${e(value)}</div></div>`;}
  function detail(row:EsiLogRow) {
    requiredElement<HTMLElement>(root,'[data-esi-log-detail-header]').innerHTML=`<span class="px-2 py-0.5 rounded text-xs font-mono tabular-nums ${esiStatusBadgeClass(row)}">${e(row.status_code??'—')}</span><span class="text-sm font-medium text-white">${e(row.method)} Request</span>`;
    const newIds=Array.isArray(row.new_item_ids)?row.new_item_ids:[],stored=new Set((row.stored_item_ids||[]).map(Number));
    requiredElement<HTMLElement>(root,'[data-esi-log-detail-body]').innerHTML=`<div class="space-y-4"><div><div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Endpoint</div><div class="text-xs font-mono text-gray-300 break-all bg-white/[0.04] rounded-lg px-3 py-2">${e(row.endpoint)}</div></div>${showEntity&&row.endpoint_entity_name&&row.endpoint_type&&row.endpoint_entity_id!=null?`<div class="flex items-center gap-3"><img src="${entityImage(row.endpoint_type,row.endpoint_entity_id,64)}" alt="" width="40" height="40" class="w-10 h-10 rounded-full"><div><div class="text-sm text-white font-medium">${e(row.endpoint_entity_name)}</div><div class="text-xs text-gray-500">${e(row.endpoint_type)} #${Number(row.endpoint_entity_id)}</div></div></div>`:''}${row.error_message?`<div><div class="text-fine font-bold uppercase tracking-[0.15em] text-red-400/80 mb-1">Error</div><div class="text-xs text-red-400/70 bg-red-500/[0.06] rounded-lg px-3 py-2 break-all">${e(row.error_message)}</div></div>`:''}<div class="grid grid-cols-2 gap-x-6 gap-y-3">${field('Time',fmtDate(row.created_at))}${field('Duration',(row.request_duration_ms??'—')+' ms',(row.request_duration_ms??0)>2000?'text-yellow-400':'text-gray-300')}${field('Items Returned',row.items_returned??0)}${field('New Items',row.new_items??0,(row.new_items??0)>0?'text-blue-400':'text-gray-300')}${field('Source',row.source)}${showEntity?`<div><div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">Requested By</div>${row.character_name?`<div class="flex items-center gap-1.5 text-xs text-gray-300"><img src="${entityImage('character',row.character_id,32)}" alt="" width="16" height="16" class="w-4 h-4 rounded-full" loading="lazy">${e(row.character_name)}</div>`:'<div class="text-xs text-gray-600">—</div>'}</div>`:''}${field('Method',row.method)}</div>${showEntity&&(row.new_items??0)>0?`<div><div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1.5 flex items-center gap-1.5">${icon('sparkles')} Discovered Killmails <span class="text-gray-600 font-normal normal-case tracking-normal">(${e(row.new_items)})</span></div>${newIds.length?`<div class="flex flex-wrap gap-1.5 bg-blue-500/[0.04] rounded-lg px-3 py-2">${newIds.map(id=>stored.has(Number(id))?`<a href="/kill/${Number(id)}" target="_blank" rel="noopener" class="text-xs font-mono tabular-nums px-1.5 py-0.5 rounded bg-blue-500/15 text-blue-400 hover:bg-blue-500/25 transition-colors">${Number(id)}</a>`:`<span title="Discovered by ESI, but not stored yet" class="text-xs font-mono tabular-nums px-1.5 py-0.5 rounded bg-white/[0.05] text-gray-500">${Number(id)} <span class="font-sans">pending</span></span>`).join('')}</div>`:'<div class="text-xs text-gray-500 italic bg-white/[0.02] rounded-lg px-3 py-2">Item IDs not recorded for this request (logged before tracking was added).</div>'}</div>`:''}</div>`;
    dialog.showModal();
  }
  function click(event:MouseEvent){
    const target=eventElement(event.target);if(!target)return;
    if(target.closest('[data-esi-log-clear]')){clearFilters();return;}
    if(target.closest('[data-esi-log-has-new]')){filters.has_new=filters.has_new?'':'1';page=1;void load();return;}
    const row=target.closest<HTMLElement>('[data-esi-log-row]');if(row){const entry=data.rows.find(entry=>Number(entry.id)===Number(row.dataset.esiLogRow));if(entry)detail(entry);return;}
    if(target.closest('[data-esi-log-retry]'))load();
    if(target.closest('[data-esi-log-live]')){live=!live;render();lifecycle();}
    const pagination=target.closest<HTMLButtonElement>('[data-esi-log-page]');if(pagination&&!pagination.disabled){page+=pagination.dataset.esiLogPage==='prev'?-1:1;load();}
    if(target.closest('[data-esi-log-detail-close]'))dialog.close();
    if(event.target===dialog){const rect=dialog.getBoundingClientRect();if(event.clientX<rect.left||event.clientX>rect.right||event.clientY<rect.top||event.clientY>rect.bottom)dialog.close();}
  }
  function keyboard(event:KeyboardEvent){const row=eventElement(event.target)?.closest<HTMLElement>('[data-esi-log-row]');if(row&&(event.key==='Enter'||event.key===' ')){event.preventDefault();row.click();}}
  async function tick(){
    if(!active||!root.isConnected){stop();return;}
    if(!live||page!==1||filterRequest||pollRequest||searchTimer||document.hidden)return;
    const topId=data.rows?.[0]?.id;if(!topId)return;
    const currentVersion=version,controller=new AbortController();pollRequest=controller;
    try{const fresh=await requestJSON<AdminEsiLogsResponses[200]>(endpoint+'?'+query({after_id:String(topId),limit:'50'}),{signal:controller.signal});if(currentVersion!==version||pollRequest!==controller||!live||page!==1||section?.hidden||!fresh.rows?.length)return;data={...data,...mergeEsiLogRows(data,parseEsiLogData(fresh).rows)};render();}catch{}finally{if(pollRequest===controller)pollRequest=null;}
  }
  function stop(){if(timer!==null)clearInterval(timer);timer=null;pollRequest?.abort();pollRequest=null;}
  const section=root.closest<HTMLElement>('[data-settings-section],[data-admin-section]');
  function lifecycle(){stop();if(active&&live&&!section?.hidden&&!document.hidden)timer=setInterval(tick,5000);}
  const hide=()=>{resumeLoad=Boolean(filterRequest||searchTimer);stop();filterRequest?.abort();filterRequest=null;root.removeAttribute('aria-busy');if(searchTimer!==null)clearTimeout(searchTimer);searchTimer=null;},show=(event:PageTransitionEvent)=>{if(event.persisted){lifecycle();if(resumeLoad){resumeLoad=false;void load();}}};
  render();lifecycle();if(root.dataset.failed==='true')status(admin?'Could not load ESI requests.':'Could not load your ESI requests.');
  return {reload:load,click,keyboard,hide,show,
    search(event:Event){const input=event.target;if(!admin||!(input instanceof HTMLInputElement)||!input.matches('[data-esi-log-search]'))return;if(searchTimer!==null)clearTimeout(searchTimer);filterRequest?.abort();filterRequest=null;pollRequest?.abort();pollRequest=null;version++;filters.search=input.value;page=1;searchTimer=setTimeout(()=>{searchTimer=null;void load();},300);},
    visibility(){if(section?.hidden){hide();dialog.close();}else{lifecycle();if(resumeLoad&&!document.hidden){resumeLoad=false;void load();}}},
    dispose(){active=false;hide();for(const [,menu]of menus){menu.dispose();menu.root.remove();}entityPicker?.dispose();entityPicker?.root.remove();hasNew?.remove();clear.remove();dialog.close();},
  };
}

const instances=new WeakMap<HTMLElement,EsiLogController>();
const actions='click->esi-log#click keydown->esi-log#keyboard input->esi-log#search pagehide@window->esi-log#hide pageshow@window->esi-log#show visibilitychange@document->esi-log#visibility';
export function initializeEsiLog(root:HTMLElement){
  const names=new Set((root.dataset.controller??'').split(/\s+/).filter(Boolean));names.add('esi-log');root.dataset.controller=[...names].join(' ');
  if(!root.dataset.action?.includes('click->esi-log#click'))root.dataset.action=[root.dataset.action,actions].filter(Boolean).join(' ');
  instances.get(root)?.ensure();
  return {reload:()=>instances.get(root)?.reload(),dispose(){instances.get(root)?.dispose();root.dataset.controller=(root.dataset.controller??'').split(/\s+/).filter(name=>name!=='esi-log').join(' ');}};
}
export class EsiLogController extends Controller<HTMLElement>{
  private log:ReturnType<typeof createEsiLog>|undefined;
  private section:HTMLElement|null=null;
  private visibilityEvent='';
  private sectionVisibility=()=>this.log?.visibility();
  connect(){instances.set(this.element,this);this.ensure();this.section=this.element.closest('[data-settings-section],[data-admin-section]');this.visibilityEvent=this.element.dataset.admin==='true'?'admin-visibility':'settings-visibility';this.section?.addEventListener(this.visibilityEvent,this.sectionVisibility);}
  disconnect(){this.dispose();this.section?.removeEventListener(this.visibilityEvent,this.sectionVisibility);instances.delete(this.element);}
  ensure(){this.log??=createEsiLog(this.element);}
  dispose(){this.log?.dispose();this.log=undefined;}
  reload(){return this.log?.reload();}
  click(event:MouseEvent){this.log?.click(event);}
  keyboard(event:KeyboardEvent){this.log?.keyboard(event);}
  search(event:Event){this.log?.search(event);}
  hide(){this.log?.hide();}
  show(event:PageTransitionEvent){this.log?.show(event);}
  visibility(){this.log?.visibility();}
}
registerController('esi-log',EsiLogController);
