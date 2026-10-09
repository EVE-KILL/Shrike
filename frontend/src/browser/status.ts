import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'
import {requiredElement,eventElement} from './shared/dom'
import {createRelayConnection} from '../utils/relaySocketCore'
import {readStatusSnapshot,statusNumber} from './shared/status-model'
import {renderStatusPanels} from './shared/status-render'
import {escapeHTML} from './shared/html'

export class StatusController extends Controller<HTMLElement> {
  private alive = false
  private transport:ReturnType<typeof createRelayConnection>|undefined
  private hoveredHour:number|null = null
  private get panels() {return requiredElement<HTMLElement>(this.element,'[data-status-panels]')}
  private q<T extends Element = HTMLElement>(selector:string) {return requiredElement<T>(this.element,selector)}
  connect() {this.alive = true;this.start()}
  private state(connected:boolean) {
    this.element.classList.toggle('status-connected',connected)
    this.q('[data-status-connection]').textContent = connected ? 'Live' : 'Connecting...'
    this.q<HTMLElement>('.status-dot').title = connected ? 'Live — connected' : 'Disconnected'
    this.q<HTMLElement>('[data-status-error]').hidden = connected || !this.panels.hidden
  }
  private start() {
    if(this.transport || !this.alive) return
    this.state(false)
    const transport = createRelayConnection({
      wsUrl:location.origin,path:'/ws/status',reconnect:true,visibilityPause:true,
      shouldReconnect:()=>this.alive && this.element.isConnected,
      onState:connected=>{if(this.transport === transport && this.alive) this.state(connected)},
      onMessage:(channel,value)=>{
        if(channel !== 'status' || this.transport !== transport || !this.alive || !this.element.isConnected) return
        const data = readStatusSnapshot(value);if(!data) return
        this.panels.innerHTML = renderStatusPanels(data);this.panels.hidden = false
        this.q<HTMLElement>('[data-status-loading]').hidden = true;this.q<HTMLElement>('[data-status-error]').hidden = true
        if(this.hoveredHour !== null) this.showHour(this.panels.querySelector<HTMLElement>(`[data-status-hour="${this.hoveredHour}"]`))
      },
    })
    this.transport = transport;transport.connect()
  }
  private showHour(bar:HTMLElement|null) {
    if(!bar) return
    const chart = bar.closest<HTMLElement>('.status-ingest-chart'),tip = chart?.querySelector<HTMLElement>('.status-ingest-tooltip')
    if(!chart || !tip) return
    for(const old of chart.querySelectorAll('.status-ingest-hour.active')) old.classList.remove('active')
    const hours = statusNumber(chart.dataset.windowHours),index = statusNumber(bar.dataset.statusHour),ago = hours-1-index
    const start = new Date(Math.floor(Date.now()/3600000)*3600000-ago*3600000),end = new Date(start.getTime()+3600000)
    const time = (value:Date)=>value.toLocaleTimeString('en-US',{hour:'numeric',minute:'2-digit'})
    const label = ago === 0 ? 'Current hour' : start.toLocaleDateString('en-US',{month:'short',day:'numeric'})
    const number = (value:unknown)=>new Intl.NumberFormat('en-US').format(statusNumber(value))
    tip.innerHTML = `<strong>${escapeHTML(label)} · ${escapeHTML(time(start))}–${escapeHTML(time(end))}</strong><div><span>New kills</span><b>${number(bar.dataset.new)}</b></div><div><span>Reposts</span><b>${number(bar.dataset.repost)}</b></div><div><span>Total events</span><b>${number(statusNumber(bar.dataset.new)+statusNumber(bar.dataset.repost))}</b></div>`
    tip.hidden = false
    const width = chart.clientWidth,half = Math.min(width/2,tip.offsetWidth/2)
    const position = (index+.5)/Math.max(1,hours)*width
    tip.style.left = `${Math.min(width-half,Math.max(half,position))}px`
    chart.classList.add('status-ingest-hover');bar.classList.add('active')
  }
  hour(event:MouseEvent|FocusEvent) {
    const bar = eventElement(event.target)?.closest<HTMLElement>('[data-status-hour]')
    if(bar) {this.hoveredHour = statusNumber(bar.dataset.statusHour);this.showHour(bar)}
  }
  leave(event:MouseEvent|FocusEvent) {
    const bar = eventElement(event.target)?.closest<HTMLElement>('[data-status-hour]')
    if(!bar || event.relatedTarget instanceof Node && bar.contains(event.relatedTarget)) return
    this.clearHour()
  }
  private clearHour() {
    this.hoveredHour = null
    for(const bar of this.panels.querySelectorAll('.status-ingest-hour.active')) bar.classList.remove('active')
    for(const chart of this.panels.querySelectorAll('.status-ingest-chart')) chart.classList.remove('status-ingest-hover')
    for(const tip of this.panels.querySelectorAll<HTMLElement>('.status-ingest-tooltip')) tip.hidden = true
  }
  hide() {this.alive = false;this.transport?.dispose();this.transport = undefined;this.clearHour();this.state(false)}
  disconnect() {this.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;this.start()}}
}
registerController('status',StatusController)
