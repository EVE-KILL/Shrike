import {Controller} from '@hotwired/stimulus'
import {escapeHTML as e} from './shared/html'
import {requestJSON} from './browser-request'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'
import {isRecord} from './shared/json'
import type {AdminEsiEntitiesResponses} from '../../shared/api/types.gen'

export type EntitySelection=Omit<AdminEsiEntitiesResponses[200]['results'][number],'type'>&{type:'character'|'corporation'}
export function readEntitySelection(value:unknown):EntitySelection|null{
  if(!isRecord(value)||(typeof value.id!=='number'&&typeof value.id!=='string')||(value.type!=='character'&&value.type!=='corporation')||typeof value.name!=='string')return null
  const id=Number(value.id);return Number.isSafeInteger(id)&&id>0?{id,name:value.name,type:value.type}:null
}
const icon=(name:string)=>`<svg class="icon" aria-hidden="true"><use href="/assets/icons.svg#${name}"></use></svg>`
const image=(entity:EntitySelection,size:number)=>`https://images.evetech.net/${entity.type==='corporation'?'corporations':'characters'}/${entity.id}/${entity.type==='corporation'?'logo':'portrait'}?size=${size}`
type Configuration={placeholder:string;onChange:(value:EntitySelection|null)=>void;selected:EntitySelection|null;searchUrl:string;active:boolean}
const configurations=new WeakMap<HTMLElement,Configuration>(),controllers=new WeakMap<HTMLElement,EntitySearchSelectController>()

// Caller configures the detached element; Stimulus owns its DOM lifetime.
export function createEntitySearchSelect(placeholder:string,onChange:Configuration['onChange'],initial:EntitySelection|null=null,searchUrl='/api/admin/esi-entities'){
  const root=document.createElement('div');root.className='relative inline-flex'
  root.dataset.controller='entity-search-select';root.dataset.action='click->entity-search-select#click input->entity-search-select#search keydown->entity-search-select#keyboard pointerdown@document->entity-search-select#outside'
  root.innerHTML='<div data-entity-trigger></div><div hidden data-entity-search class="flex items-center gap-1.5 bg-white/[0.08] border border-white/[0.12] rounded-lg px-2.5 py-1.5 min-w-[200px]"><span class="text-gray-500 text-xs flex-shrink-0">'+icon('search')+'</span><input type="text" class="bg-transparent text-xs text-white placeholder-gray-600 outline-none w-full min-w-0"><span hidden data-entity-loading class="text-xs text-gray-500 animate-spin flex-shrink-0">'+icon('loader-2')+'</span></div><div hidden data-entity-results class="absolute top-full left-0 mt-1.5 z-50 rounded-xl border border-white/[0.10] bg-[#141414]/95 backdrop-blur-2xl shadow-2xl shadow-black/60 overflow-y-auto max-h-[50vh] p-1.5 min-w-[220px]"></div>'
  const input=requiredElement<HTMLInputElement>(root,'input');input.placeholder=placeholder;input.setAttribute('aria-label',placeholder)
  const config:Configuration={placeholder,onChange,selected:initial,searchUrl,active:true};configurations.set(root,config)
  renderTrigger(root,config,false)
  return {root,
    setValue(value:EntitySelection|null,emit=false){config.selected=value;const controller=controllers.get(root);if(controller)controller.setValue(value,emit);else{renderTrigger(root,config,false);if(emit&&config.active)onChange(value)}},
    dispose(){config.active=false;controllers.get(root)?.dispose();root.removeAttribute('data-controller')},
  }
}
function renderTrigger(root:HTMLElement,config:Configuration,open:boolean){
  const trigger=requiredElement<HTMLElement>(root,'[data-entity-trigger]'),selected=config.selected
  trigger.className='flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs transition-colors cursor-pointer border max-w-[200px] '+(selected?'bg-white/[0.06] text-gray-300 border-white/[0.08] hover:border-white/[0.12]':'bg-white/[0.04] text-gray-500 border-white/[0.06] hover:border-white/[0.10] hover:text-blue-400')
  trigger.innerHTML=`<button type="button" data-entity-open aria-label="${e(config.placeholder)}" aria-expanded="${open}" class="flex items-center gap-1.5 min-w-0 cursor-pointer">${selected?`<img src="${image(selected,32)}" alt="" width="16" height="16" class="w-4 h-4 rounded-full flex-shrink-0"><span class="truncate">${e(selected.name)}</span>`:`<span class="text-[10px] opacity-60">${icon('search')}</span><span>${e(config.placeholder)}</span>`}</button>${selected?`<button type="button" data-entity-clear aria-label="Clear selected entity" class="ml-0.5 text-gray-600 hover:text-blue-400 transition-colors flex-shrink-0 cursor-pointer"><span class="text-[10px]">${icon('x')}</span></button>`:''}`
}
export class EntitySearchSelectController extends Controller<HTMLElement>{
  private config:Configuration|undefined
  private open=false
  private timer:ReturnType<typeof setTimeout>|undefined
  private pending:AbortController|undefined
  private hits:EntitySelection[]=[]
  private get input(){return requiredElement<HTMLInputElement>(this.element,'input')}
  private get trigger(){return requiredElement<HTMLElement>(this.element,'[data-entity-trigger]')}
  private get searchBox(){return requiredElement<HTMLElement>(this.element,'[data-entity-search]')}
  private get loading(){return requiredElement<HTMLElement>(this.element,'[data-entity-loading]')}
  private get results(){return requiredElement<HTMLElement>(this.element,'[data-entity-results]')}
  connect(){this.config=configurations.get(this.element);controllers.set(this.element,this);if(this.config)renderTrigger(this.element,this.config,false)}
  disconnect(){this.dispose();controllers.delete(this.element)}
  dispose(){this.cancel();this.close()}
  private cancel(){this.pending?.abort();this.pending=undefined;clearTimeout(this.timer);this.loading.hidden=true}
  private close(){this.open=false;this.searchBox.hidden=true;this.trigger.hidden=false;this.results.hidden=true;if(this.config)renderTrigger(this.element,this.config,false)}
  setValue(value:EntitySelection|null,emit=false){
    if(!this.config?.active)return
    this.config.selected=value;this.input.value='';this.results.replaceChildren();this.cancel();this.close()
    if(emit)this.config.onChange(value)
  }
  click(event:MouseEvent){
    if(!this.config?.active)return
    const target=eventElement(event.target)
    if(target?.closest('[data-entity-open]')){this.open=true;this.trigger.hidden=true;this.searchBox.hidden=false;this.input.focus();return}
    if(target?.closest('[data-entity-clear]')){this.setValue(null,true);return}
    const button=target?.closest<HTMLButtonElement>('[data-entity-hit]'),hit=button?this.hits[Number(button.dataset.entityHit)]:undefined
    if(hit)this.setValue(hit,true)
  }
  search(event:Event){
    const config=this.config;if(!config?.active||event.target!==this.input)return
    this.cancel();this.results.hidden=true;const q=this.input.value
    if(q.length<2){this.results.replaceChildren();return}
    const controller=new AbortController();this.pending=controller
    this.timer=setTimeout(async()=>{
      this.loading.hidden=false
      try{
        const data=await requestJSON<AdminEsiEntitiesResponses[200]>(config.searchUrl+'?'+new URLSearchParams({q}),{signal:controller.signal})
        if(!config.active||this.pending!==controller||!this.element.isConnected)return
        this.hits=(data.results??[]).map(readEntitySelection).filter((hit):hit is EntitySelection=>hit!==null)
        this.results.replaceChildren(...this.hits.map((hit,index)=>{
          const button=document.createElement('button');button.type='button';button.dataset.entityHit=String(index);button.className='flex items-center gap-2.5 w-full px-2.5 py-2 rounded-md text-xs text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors text-left cursor-pointer'
          button.innerHTML=`<img src="${image(hit,32)}" alt="" width="20" height="20" class="w-5 h-5 rounded-full flex-shrink-0"><div class="flex-1 min-w-0"><div class="truncate">${e(hit.name)}</div></div><span class="text-[10px] text-gray-600 uppercase flex-shrink-0">${e(hit.type)}</span>`;return button
        }))
        if(!this.hits.length)this.results.innerHTML='<div class="px-2.5 py-3 text-xs text-gray-600 text-center">No matching entities with ESI logs</div>'
        this.results.hidden=!this.open
      }catch{if(this.pending===controller&&config.active){this.results.innerHTML='<div class="px-2.5 py-3 text-xs text-gray-600 text-center">Could not search entities. Try again.</div>';this.results.hidden=!this.open}}
      finally{if(this.pending===controller)this.loading.hidden=true}
    },250)
  }
  keyboard(event:KeyboardEvent){
    if(event.key==='Escape'){this.cancel();this.close();this.trigger.querySelector<HTMLButtonElement>('button')?.focus()}
    else if(event.key==='ArrowDown'&&event.target===this.input){event.preventDefault();this.results.querySelector<HTMLButtonElement>('button')?.focus()}
  }
  outside(event:PointerEvent){if(this.open&&!this.element.contains(eventElement(event.target))){this.cancel();this.close()}}
}
registerController('entity-search-select',EntitySearchSelectController)
