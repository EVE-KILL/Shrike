import {Controller} from '@hotwired/stimulus'
import type {SearchHit,SearchOutput} from '../../shared/api/types.gen'
import {registerController} from './controllers/application'
import {iconHref} from './icon-hrefs'
import {eventElement,requiredElement} from './shared/dom'
import {requestJSON} from './browser-request'
import {searchID} from './shared/search-model'

export type SearchSelection=Omit<SearchHit,'id'|'ticker'>&{id:number;ticker:string|null}
export interface SearchPickerOptions {
  disabled?:boolean;placeholder?:string;isPicked?:(type:string,id:number)=>boolean
}
const pickers=new WeakMap<HTMLElement,SearchPickerController>()
const optionsByRoot=new WeakMap<HTMLElement,SearchPickerOptions>()
const actions='input->search-picker#search keydown->search-picker#keyboard click->search-picker#pick pointerdown@document->search-picker#outside'

// Keep the parent API while Stimulus owns connections, events and cancellation.
// Parents can configure newly inserted markup before its connection microtask.
export function initializeSearchPickers(parent:ParentNode=document){
  for(const root of parent.querySelectorAll<HTMLElement>('[data-search-picker]')){
    const controllers=new Set((root.dataset.controller??'').split(/\s+/).filter(Boolean));controllers.add('search-picker')
    root.dataset.controller=[...controllers].join(' ')
    if(!root.dataset.action?.includes('input->search-picker#search'))root.dataset.action=[root.dataset.action,actions].filter(Boolean).join(' ')
  }
}
export function updateSearchPicker(root:HTMLElement,options:SearchPickerOptions={}){
  optionsByRoot.set(root,{...options,isPicked:options.isPicked??optionsByRoot.get(root)?.isPicked});pickers.get(root)?.update(options)
}
export function closeSearchPickers(parent:ParentNode){for(const root of parent.querySelectorAll<HTMLElement>('[data-search-picker]'))pickers.get(root)?.hide()}

export class SearchPickerController extends Controller<HTMLElement> {
  private timer:ReturnType<typeof setTimeout>|undefined
  private request:AbortController|undefined
  private hits:SearchSelection[]=[]
  private input!:HTMLInputElement
  private results!:HTMLElement
  private spinner!:HTMLElement
  private template!:HTMLTemplateElement
  private types:string[]=[]
  private placeholder:string|undefined
  private isPicked:(type:string,id:number)=>boolean=()=>false
  connect(){
    this.input=requiredElement(this.element,'[data-picker-input]');this.results=requiredElement(this.element,'[data-picker-results]')
    this.spinner=requiredElement(this.element,'[data-picker-spinner]');this.template=requiredElement(this.element,'[data-picker-hit-template]')
    this.types=(this.element.dataset.pickerTypes??'').split(',');this.placeholder??=this.input.placeholder
    pickers.set(this.element,this);const options=optionsByRoot.get(this.element);if(options)this.update(options)
  }
  disconnect(){this.hide();pickers.delete(this.element)}
  private close(){this.results.hidden=true}
  hide(){clearTimeout(this.timer);this.request?.abort();this.request=undefined;this.spinner.hidden=true;this.close()}
  update(options:SearchPickerOptions){
    if(options.isPicked)this.isPicked=options.isPicked
    this.input.disabled=Boolean(options.disabled);this.input.placeholder=options.placeholder??this.placeholder??''
    if(this.input.disabled)this.hide();else if(!this.results.hidden)this.render()
  }
  private render(){
    this.results.replaceChildren(...this.hits.map((hit,index)=>{
      const row=this.template.content.firstElementChild?.cloneNode(true)
      if(!(row instanceof HTMLButtonElement))throw new Error('Missing search picker button template')
      row.dataset.pickerHit=String(index)
      const image=requiredElement<HTMLImageElement>(row,'[data-picker-image]'),symbol=requiredElement<HTMLElement>(row,'[data-picker-symbol]')
      const url=hit.type==='character'?`https://images.evetech.net/characters/${hit.id}/portrait?size=64`
        :hit.type==='corporation'?`https://images.evetech.net/corporations/${hit.id}/logo?size=64`
        :hit.type==='alliance'?`https://images.evetech.net/alliances/${hit.id}/logo?size=64`
        :this.element.dataset.pickerTypeImages==='true'&&['ship','item'].includes(hit.type)?`/images/types/${hit.id}/icon?size=32`
        :this.element.dataset.pickerLocationImages==='true'&&['system','region','constellation'].includes(hit.type)?`/images/${hit.type}s/${hit.id}?size=64`:''
      image.hidden=!url;symbol.hidden=Boolean(url);if(url){image.src=url;image.alt=hit.name}
      requiredElement<SVGUseElement>(symbol,'use').setAttribute('href',iconHref(hit.type==='region'?'map':hit.type==='constellation'?'compass':'map-pin'))
      requiredElement<HTMLElement>(row,'[data-picker-hit-name]').textContent=hit.name
      requiredElement<HTMLElement>(row,'[data-picker-hit-type]').textContent=this.element.dataset.pickerLocationImages==='true'&&hit.type==='system'?'Solar System':hit.type
      const ticker=requiredElement<HTMLElement>(row,'[data-picker-hit-ticker]');ticker.hidden=!hit.ticker;if(hit.ticker)ticker.textContent='['+hit.ticker+']'
      row.disabled=this.isPicked(hit.type,hit.id);return row
    }))
    this.results.hidden=!this.hits.length
  }
  search(event:Event){
    if(event.target!==this.input)return
    this.hide();const q=this.input.value.trim()
    if(q.length<2||this.input.disabled){this.hits=[];return}
    const controller=new AbortController();this.request=controller
    this.timer=setTimeout(async()=>{
      this.spinner.hidden=false
      try{
        const data=await requestJSON<SearchOutput>('/api/search?'+new URLSearchParams({q,type:this.types.join(','),limit:this.element.dataset.pickerLimit||'10'}),{signal:controller.signal})
        if(this.request!==controller||!this.element.isConnected)return
        this.hits=(data.hits??[]).filter(hit=>this.types.includes(hit.type))
          .map(hit=>({...hit,id:searchID(hit.id),ticker:hit.ticker??null}))
          .filter(hit=>hit.id>0)
        this.render()
      }catch{if(this.request===controller){this.hits=[];this.close()}}
      finally{if(this.request===controller)this.spinner.hidden=true}
    },Number(this.element.dataset.pickerDebounce)||300)
  }
  pick(event:MouseEvent){
    const button=eventElement(event.target)?.closest<HTMLButtonElement>('[data-picker-hit]')
    if(!button||button.disabled)return
    const hit=this.hits[Number(button.dataset.pickerHit)];if(!hit)return
    this.input.value='';this.hits=[];this.hide();this.element.dispatchEvent(new CustomEvent<SearchSelection>('search-pick',{bubbles:true,detail:hit}))
  }
  keyboard(event:KeyboardEvent){
    if(event.key==='Escape'){this.hide();this.input.focus()}
    if(event.target!==this.input)return
    if(event.key==='Enter')event.preventDefault()
    if(event.key==='ArrowDown'){event.preventDefault();this.results.querySelector<HTMLButtonElement>('button:not([disabled])')?.focus()}
  }
  outside(event:PointerEvent){if(!this.element.contains(eventElement(event.target)))this.hide()}
}
registerController('search-picker',SearchPickerController)
