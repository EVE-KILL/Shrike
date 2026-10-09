import {Controller} from '@hotwired/stimulus'
import {registerController} from './controllers/application'
import {requiredElement} from './shared/dom'

type Metrics=Record<string,number>
type RequestRow={path:string;metrics:Metrics}
export function parseServerTiming(entry:{serverTiming:ReadonlyArray<Pick<PerformanceServerTiming,'name'|'description'|'duration'>>}|undefined):Metrics|null{
  const metrics:Metrics=Object.fromEntries((entry?.serverTiming??[]).map(metric=>[
    metric.name,metric.name.startsWith('db')&&metric.name!=='db'&&metric.name!=='dbwait'?Number(metric.description):metric.duration,
  ]))
  return Object.hasOwn(metrics,'app')?metrics:null
}
export function formatTiming(value:number|undefined){return typeof value==='number'&&Number.isFinite(value)?value.toFixed(value<10?2:1):'—'}
const counts=new Set(['dbcount','dberrors','dbbatches','dbcopies'])

export class DiagnosticsController extends Controller<HTMLDetailsElement> {
  private rows:RequestRow[]=[]
  private observer:PerformanceObserver|undefined
  private timer:ReturnType<typeof setTimeout>|undefined
  connect(){
    this.updateNavigation()
    if(document.readyState==='complete')this.loaded()
    if(!('PerformanceObserver'in window))return
    this.observer=new PerformanceObserver(list=>{
      for(const entry of list.getEntries()){
        if(entry.entryType!=='resource')continue
        const metrics=parseServerTiming(entry as PerformanceResourceTiming)
        if(!metrics)continue
        const url=new URL(entry.name)
        if(url.origin!==location.origin)continue
        this.rows.unshift({path:url.pathname,metrics});this.rows.length=Math.min(this.rows.length,12)
      }
      this.drawRows()
    })
    this.observer.observe({type:'resource',buffered:true})
  }
  disconnect(){this.observer?.disconnect();clearTimeout(this.timer);this.rows=[]}
  loaded(){clearTimeout(this.timer);this.timer=setTimeout(()=>this.updateNavigation(),0)}
  close(){this.element.open=false}
  keyboard(event:KeyboardEvent){if(event.key==='Escape'){this.close();requiredElement<HTMLElement>(this.element,'summary').focus()}}
  clear(){this.rows=[];this.drawRows()}
  private updateNavigation(){
    const navigation=performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming|undefined
    const metrics=parseServerTiming(navigation),status=requiredElement<HTMLElement>(this.element,'#diagnostics-status')
    if(!navigation||!metrics){status.textContent='Timing headers are unavailable. Reload the page after the development server rebuilds.';return}
    metrics.other=Math.max(0,metrics.app-(metrics.load||0)-(metrics.render||0))
    for(const element of this.element.querySelectorAll<HTMLElement>('[data-metric]')){
      const key=element.dataset.metric??''
      element.textContent=counts.has(key)?String(metrics[key]??0):formatTiming(metrics[key])
    }
    for(const bar of this.element.querySelectorAll<HTMLElement>('[data-part]')){
      const portion=metrics[bar.dataset.part??'']||0
      bar.style.width=`${metrics.app>0?Math.min(100,Math.max(0,portion/metrics.app*100)):0}%`
    }
    const browser=(key:string,value:number|undefined)=>{requiredElement<HTMLElement>(this.element,`[data-browser="${key}"]`).textContent=formatTiming(value)}
    browser('navigation',navigation.responseStart-navigation.startTime)
    browser('setup',navigation.requestStart-navigation.startTime)
    browser('ttfb',navigation.responseStart-navigation.requestStart)
    browser('transfer',navigation.responseEnd-navigation.responseStart)
    requiredElement<HTMLElement>(this.element,'[data-browser="dom"]').textContent=navigation.domContentLoadedEventEnd>0?formatTiming(navigation.domContentLoadedEventEnd-navigation.startTime):'…'
    status.textContent=metrics.dbcount===0?'This document used no SQL queries. Browser API requests are measured separately below.':''
  }
  private drawRows(){
    const tbody=requiredElement<HTMLTableSectionElement>(this.element,'#diagnostics-requests')
    if(!this.rows.length){
      const row=document.createElement('tr'),cell=document.createElement('td')
      cell.colSpan=4;cell.textContent='Open search or use a page control to see its requests.';row.append(cell);tbody.replaceChildren(row);return
    }
    tbody.replaceChildren(...this.rows.map(({path,metrics})=>{
      const row=document.createElement('tr');row.dataset.failed=String(metrics.dberrors>0);row.dataset.slow=String(metrics.app>250)
      for(const value of [path,`${formatTiming(metrics.app)} ms`,String(metrics.dbcount??0),`${formatTiming(metrics.db)} ms`]){
        const cell=document.createElement('td');cell.textContent=value;row.append(cell)
      }
      return row
    }))
  }
}
registerController('diagnostics',DiagnosticsController)
