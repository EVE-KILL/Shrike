import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'
import {eventElement,requiredElement} from './shared/dom'
import {isRecord} from './shared/json'
import {placeFloatingPanel} from './shared/floating-panel'

type Zone='eve'|'local'
type Parts={y:number;mo:number;d:number;h:number;mi:number}
const months=['January','February','March','April','May','June','July','August','September','October','November','December']
const pad=(number:number)=>String(number).padStart(2,'0')
let zone:Zone=document.cookie.split('; ').find(value=>value.startsWith('ek_tz='))?.slice(6)==='local'?'local':'eve'
const connected=new Set<DateTimePickerController>()
const partsIn=(date:Date,current:Zone):Parts=>current==='eve'?{y:date.getUTCFullYear(),mo:date.getUTCMonth(),d:date.getUTCDate(),h:date.getUTCHours(),mi:date.getUTCMinutes()}:{y:date.getFullYear(),mo:date.getMonth(),d:date.getDate(),h:date.getHours(),mi:date.getMinutes()}
const instantFrom=(parts:Parts,current:Zone)=>current==='eve'?new Date(Date.UTC(parts.y,parts.mo,parts.d,parts.h,parts.mi)):new Date(parts.y,parts.mo,parts.d,parts.h,parts.mi)
const label=(parts:Parts)=>`${parts.y}-${pad(parts.mo+1)}-${pad(parts.d)} ${pad(parts.h)}:${pad(parts.mi)}`
export function setDateTimeZone(value:unknown){
  if(value!=='eve'&&value!=='local')return
  zone=value;document.cookie=`ek_tz=${zone}; Max-Age=31536000; Path=/; SameSite=Lax${location.protocol==='https:'?'; Secure':''}`
  for(const picker of connected)picker.zoneChanged()
}
// The preference is document-wide; individual picker bindings remain scoped.
document.addEventListener('eve-time-zone-change',event=>{if(event instanceof CustomEvent)setDateTimeZone(event.detail)})

export class DateTimePickerController extends Controller<HTMLElement> {
  private year=0
  private month=0
  private q<T extends Element=HTMLElement>(selector:string){return requiredElement<T>(this.element,selector)}
  private get input(){return this.q<HTMLInputElement>('[data-date-time-value]')}
  private get trigger(){return this.q<HTMLButtonElement>('[data-date-time-trigger]')}
  private get panel(){return this.q<HTMLElement>('[data-date-time-panel]')}
  connect(){connected.add(this);this.syncView();this.render()}
  disconnect(){connected.delete(this);this.close()}
  private instant(){const date=new Date(`${this.input.value}Z`);return this.input.value&&!Number.isNaN(date.getTime())?date:null}
  private shown(){const date=this.instant();return date?partsIn(date,zone):null}
  private syncView(){const parts=this.shown()??partsIn(new Date(),zone);this.year=parts.y;this.month=parts.mo}
  close(){this.panel.hidden=true;this.trigger.setAttribute('aria-expanded','false');this.trigger.classList.remove('border-blue-500/40');this.trigger.classList.add('border-white/[0.08]');requiredElement<SVGElement>(this.trigger,'.icon:last-child').classList.remove('rotate-180')}
  zoneChanged(){this.syncView();this.render()}
  private render(){
    const parts=this.shown(),now=partsIn(new Date(),zone),text=this.q<HTMLElement>('[data-date-time-label]')
    text.textContent=parts?label(parts):this.element.dataset.placeholder??'';text.classList.toggle('text-gray-600',!parts);text.classList.toggle('text-gray-200',Boolean(parts))
    const suffix=this.q<HTMLElement>('[data-date-time-zone-label]');suffix.textContent=zone==='eve'?'EVE':'Local';suffix.classList.toggle('text-blue-400/70',zone==='eve');suffix.classList.toggle('text-amber-400/70',zone==='local')
    if(this.panel.hidden)return
    this.q<HTMLElement>('[data-date-time-month-label]').textContent=`${months[this.month]} ${this.year}`
    for(const button of this.panel.querySelectorAll<HTMLButtonElement>('[data-date-time-zone]')){button.className=`px-2 py-1 text-fine font-medium transition-colors cursor-pointer ${button.dataset.dateTimeZone===zone?'bg-blue-500/20 text-blue-400':'bg-white/[0.02] text-gray-500 hover:bg-white/[0.05] hover:text-gray-300'}`;if(button.dataset.dateTimeZone==='local')button.title=Intl.DateTimeFormat().resolvedOptions().timeZone||'Local'}
    const lead=(new Date(Date.UTC(this.year,this.month,1)).getUTCDay()+6)%7
    const cells=Array.from({length:42},(_,index)=>{
      const day=new Date(Date.UTC(this.year,this.month,index-lead+1)),cell={y:day.getUTCFullYear(),mo:day.getUTCMonth(),d:day.getUTCDate()}
      const selected=parts&&cell.y===parts.y&&cell.mo===parts.mo&&cell.d===parts.d,today=cell.y===now.y&&cell.mo===now.mo&&cell.d===now.d
      const button=document.createElement('button');button.type='button';button.textContent=String(cell.d);button.dataset.dateTimeDay=JSON.stringify(cell)
      button.setAttribute('aria-label',`${cell.y}-${pad(cell.mo+1)}-${pad(cell.d)}`);button.setAttribute('aria-pressed',String(Boolean(selected)))
      button.className=`h-8 rounded-md text-xs tabular-nums transition-colors cursor-pointer ${selected?'bg-blue-500 text-white font-semibold':cell.mo===this.month?'text-gray-300 hover:bg-blue-500/[0.12]':'text-gray-700 hover:bg-white/[0.05]'}${!selected&&today?' ring-1 ring-inset ring-blue-500/40':''}`;return button
    })
    this.q<HTMLElement>('[data-date-time-calendar]').replaceChildren(...cells)
    this.q<HTMLSelectElement>('[data-date-time-hour]').value=String(parts?.h??0);this.q<HTMLSelectElement>('[data-date-time-minute]').value=String(parts?.mi??0)
    const counterpart=this.q<HTMLElement>('[data-date-time-counterpart]'),instant=this.instant();counterpart.hidden=!parts
    if(instant)counterpart.textContent=`= ${label(partsIn(instant,zone==='eve'?'local':'eve'))} ${zone==='eve'?'local':'EVE'}`
  }
  private commit(date:Date|null){this.input.value=date?date.toISOString().slice(0,16):'';this.render();this.input.dispatchEvent(new Event('change',{bubbles:true}))}
  click(event:MouseEvent){
    const button=eventElement(event.target)?.closest<HTMLButtonElement>('button');if(!button)return
    if(button===this.trigger){
      if(!this.panel.hidden){this.close();return}
      this.syncView();this.panel.hidden=false;this.trigger.setAttribute('aria-expanded','true');this.trigger.classList.remove('border-white/[0.08]');this.trigger.classList.add('border-blue-500/40');requiredElement<SVGElement>(this.trigger,'.icon:last-child').classList.add('rotate-180')
      const right=this.element.getBoundingClientRect().left+304>innerWidth-8;this.panel.classList.toggle('right-0',right);this.panel.classList.toggle('left-0',!right);this.render();placeFloatingPanel(this.element,this.panel);return
    }
    if(!this.panel.contains(button))return
    if(button.dataset.dateTimeDay){
      let cell:unknown;try{cell=JSON.parse(button.dataset.dateTimeDay)}catch{return}
      if(!isRecord(cell)||typeof cell.y!=='number'||typeof cell.mo!=='number'||typeof cell.d!=='number')return
      const parts=this.shown();this.year=cell.y;this.month=cell.mo;this.commit(instantFrom({y:cell.y,mo:cell.mo,d:cell.d,h:parts?.h??0,mi:parts?.mi??0},zone))
    }else if(button.dataset.dateTimeMonth){const month=this.month+Number(button.dataset.dateTimeMonth);this.year+=Math.floor(month/12);this.month=(month+12)%12;this.render()}
    else if(button.dataset.dateTimeZone)setDateTimeZone(button.dataset.dateTimeZone)
    else if(button.hasAttribute('data-date-time-now')){const now=new Date();now.setSeconds(0,0);this.commit(now);this.syncView();this.render()}
    else if(button.hasAttribute('data-date-time-clear')){this.commit(null);this.close()}
    else if(button.hasAttribute('data-date-time-done'))this.close()
  }
  change(event:Event){
    const target=event.target;if(!(target instanceof HTMLInputElement||target instanceof HTMLSelectElement))return
    if(target.matches('[data-date-time-value]')){this.syncView();this.render();return}
    const parts=this.shown()??partsIn(new Date(),zone)
    if(target.matches('[data-date-time-hour]'))this.commit(instantFrom({...parts,h:Number(target.value)},zone))
    else if(target.matches('[data-date-time-minute]'))this.commit(instantFrom({...parts,mi:Number(target.value)},zone))
  }
  outside(event:PointerEvent){if(!this.panel.hidden&&!this.element.contains(eventElement(event.target)))this.close()}
  keyboard(event:KeyboardEvent){if(event.key==='Escape')this.close()}
}
registerController('date-time-picker',DateTimePickerController)
export function closeDateTimePickers(parent:Node){for(const picker of connected)if(parent.contains(picker.element))picker.close()}
export function initializeDateTimePickers(parent:ParentNode=document){
  for(const root of parent.querySelectorAll<HTMLElement>('[data-date-time-picker]')){
    const controllers=root.dataset.controller?.split(/\s+/)??[];if(!controllers.includes('date-time-picker'))root.dataset.controller=[...controllers,'date-time-picker'].join(' ')
    const action='click->date-time-picker#click change->date-time-picker#change pointerdown@document->date-time-picker#outside keydown@document->date-time-picker#keyboard pagehide@window->date-time-picker#close'
    if(!root.dataset.action?.includes('click->date-time-picker#click'))root.dataset.action=[root.dataset.action,action].filter(Boolean).join(' ')
  }
}
