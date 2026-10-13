import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'
import {escapeHTML as escape} from './shared/html'

export interface SelectOption {value:string;label:string}
export interface SelectMenu {
  root:HTMLDivElement
  setValue(value:string):void
  setOptions(options:SelectOption[]):void
  dispose():void
}

// Keep the original markup/colors; Stimulus owns bindings and their cleanup.
export class SelectMenuController extends Controller<HTMLDivElement> {
  static targets=['trigger','list','label','icon']
  static values={options:Array,selected:String}
  declare readonly triggerTarget:HTMLButtonElement
  declare readonly listTarget:HTMLDivElement
  declare readonly labelTarget:HTMLSpanElement
  declare readonly iconTarget:SVGElement
  declare optionsValue:SelectOption[]
  declare selectedValue:string
  private open=false

  connect(){this.render()}
  disconnect(){this.open=false}
  optionsValueChanged(){this.render()}
  selectedValueChanged(){this.render()}
  toggle(){this.open=!this.open;this.render()}
  choose(event:MouseEvent){
    const option=event.target instanceof Element?event.target.closest<HTMLButtonElement>('[data-select-value]'):null
    if(!option||!this.listTarget.contains(option))return
    const value=option.dataset.selectValue
    if(value===undefined||!this.optionsValue.some(item=>item.value===value))return
    this.open=false;this.selectedValue=value;this.render();this.triggerTarget.focus()
    this.dispatch('change',{detail:{value}})
  }
  keydown(event:KeyboardEvent){
    if(event.key==='Escape'){this.open=false;this.render();this.triggerTarget.focus()}
    else if(event.key==='ArrowDown'&&event.target===this.triggerTarget){
      event.preventDefault();this.open=true;this.render();this.listTarget.querySelector('button')?.focus()
    }
  }
  outside(event:PointerEvent){
    if(this.open&&event.target instanceof Node&&!this.element.contains(event.target)){this.open=false;this.render()}
  }
  private render(){renderMenu(this.element,this.optionsValue,this.selectedValue,this.open)}
}

function renderMenu(root:HTMLDivElement,options:SelectOption[],value:string,open=false){
  const trigger=root.querySelector<HTMLButtonElement>('[data-select-menu-target=trigger]')!
  const list=root.querySelector<HTMLDivElement>('[data-select-menu-target=list]')!
  trigger.className='flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs transition-colors cursor-pointer border '+(open?'bg-blue-500/10 text-blue-400 border-white/[0.12]':value?'bg-white/[0.06] text-gray-300 border-white/[0.08] hover:border-white/[0.12]':'bg-white/[0.04] text-gray-500 border-white/[0.06] hover:border-white/[0.10] hover:text-blue-400')
  trigger.setAttribute('aria-expanded',String(open))
  root.querySelector<SVGElement>('[data-select-menu-target=icon]')!.classList.toggle('rotate-180',open)
  list.hidden=!open
  root.querySelector<HTMLSpanElement>('[data-select-label]')!.textContent=options.find(o=>o.value===value)?.label||value
  list.innerHTML=options.map(o=>`<button type="button" data-select-value="${escape(o.value)}" class="flex items-center w-full px-2.5 py-1.5 rounded-md text-xs transition-colors text-left cursor-pointer ${o.value===value?'text-white bg-blue-500/[0.12]':'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'}">${escape(o.label)}</button>`).join('')
}

export function createSelectMenu(label:string,options:SelectOption[],value:string,onChange:(value:string)=>void):SelectMenu {
  registerController('select-menu',SelectMenuController)
  const root=document.createElement('div');root.className='relative inline-flex'
  root.setAttribute('data-controller','select-menu')
  root.setAttribute('data-select-menu-options-value',JSON.stringify(options))
  root.setAttribute('data-select-menu-selected-value',value)
  root.setAttribute('data-action','keydown->select-menu#keydown pointerdown@document->select-menu#outside')
  root.innerHTML=`<button type="button" data-select-menu-target="trigger" data-action="click->select-menu#toggle" aria-label="${escape(label)}" aria-expanded="false"><span data-select-label data-select-menu-target="label" class="truncate max-w-[120px]"></span><svg data-select-menu-target="icon" class="icon text-[10px] opacity-60 transition-transform" style="width:1em;height:1em" aria-hidden="true"><use href="/assets/icons.svg#chevron-down"></use></svg></button><div hidden data-select-options data-select-menu-target="list" data-action="click->select-menu#choose" class="absolute top-full left-0 mt-1.5 z-50 rounded-xl border border-white/[0.10] bg-[#141414]/95 backdrop-blur-2xl shadow-2xl shadow-black/60 overflow-y-auto max-h-[50vh] p-1.5 min-w-[10rem]"></div>`
  const change=(event:Event)=>{if(event instanceof CustomEvent){value=event.detail.value;onChange(value)}}
  root.addEventListener('select-menu:change',change)
  // Consumers can update/read the detached control before Stimulus connects.
  renderMenu(root,options,value)
  return {root,
    setValue(next){value=next;root.setAttribute('data-select-menu-selected-value',next);renderMenu(root,options,value,root.querySelector('button')?.getAttribute('aria-expanded')==='true')},
    setOptions(next){options=next;root.setAttribute('data-select-menu-options-value',JSON.stringify(next));renderMenu(root,options,value,root.querySelector('button')?.getAttribute('aria-expanded')==='true')},
    dispose(){root.removeEventListener('select-menu:change',change);root.removeAttribute('data-controller')},
  }
}
