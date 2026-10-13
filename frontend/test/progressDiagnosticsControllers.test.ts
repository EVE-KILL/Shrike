// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
import {NavigationProgressController} from '../src/browser/navigation-progress'
import {DiagnosticsController,formatTiming,parseServerTiming} from '../src/browser/diagnostics'
import {registerController} from '../src/browser/controllers/application'

const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0))
const errors=vi.fn()
afterEach(async()=>{
  document.body.removeAttribute('data-controller');document.body.removeAttribute('data-action');document.body.replaceChildren()
  await settle();expect(errors.mock.calls).toEqual([]);errors.mockClear();vi.unstubAllGlobals();vi.restoreAllMocks()
})
function mountProgress(){
  registerController('navigation-progress',NavigationProgressController).handleError=errors
  document.body.setAttribute('data-controller','navigation-progress')
  document.body.setAttribute('data-action','click@document->navigation-progress#click:capture input@document->navigation-progress#remember:capture keydown@document->navigation-progress#keyboard:capture pagehide@window->navigation-progress#pageHide pageshow@window->navigation-progress#pageShow')
  document.body.innerHTML='<div id="navigation-progress" hidden></div><div id="navigation-loading-status"></div><main id="main"><button>Filter</button><section id="fragment"></section><a href="#main">Anchor</a><a href="/kill/42">Kill</a><div data-admin-user-row="/admin/42" tabindex="0">User row</div></main>'
}
function navigate(source:Element|null=null){return Object.assign(new Event('navigate',{cancelable:true}),{
  destination:{sameDocument:false,url:'https://eve-kill.com/kill/42'},hashChange:false,downloadRequest:null,signal:new AbortController().signal,sourceElement:source,
})}
test('native JS navigation uses the recent clicked row and releases feedback on failure and disconnect',async()=>{
  const navigation=new EventTarget();vi.stubGlobal('navigation',navigation);mountProgress();await settle()
  const row=document.querySelector<HTMLElement>('[data-admin-user-row]')!;row.click();navigation.dispatchEvent(navigate())
  expect(document.querySelector<HTMLElement>('#navigation-progress')!.hidden).toBe(false)
  expect(row.classList.contains('navigation-pending')).toBe(true)
  navigation.dispatchEvent(new Event('navigateerror'));expect(row.classList.contains('navigation-pending')).toBe(false)
  navigation.dispatchEvent(navigate(row));document.body.removeAttribute('data-controller');await settle()
  expect(document.querySelector<HTMLElement>('#navigation-progress')!.hidden).toBe(true)
  navigation.dispatchEvent(navigate(row));expect(row.classList.contains('navigation-pending')).toBe(false)
})
test('fallback ignores anchors/modifier clicks, honors cancelled clicks and supports keyboard JS rows',async()=>{
  vi.stubGlobal('navigation',undefined);mountProgress();await settle()
  const bar=document.querySelector<HTMLElement>('#navigation-progress')!
  document.querySelector<HTMLAnchorElement>('[href="#main"]')!.click();expect(bar.hidden).toBe(true)
  const link=document.querySelector<HTMLAnchorElement>('[href="/kill/42"]')!
  link.dispatchEvent(new MouseEvent('click',{bubbles:true,ctrlKey:true}));expect(bar.hidden).toBe(true)
  link.addEventListener('click',event=>event.preventDefault(),{once:true});link.click();await settle();expect(bar.hidden).toBe(true)
  const row=document.querySelector<HTMLElement>('[data-admin-user-row]')!
  row.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true}));expect(bar.hidden).toBe(false)
  window.dispatchEvent(new Event('pageshow'));expect(bar.hidden).toBe(true)
})
test('fragment busy states highlight the source only while visible and release on removal',async()=>{
  vi.stubGlobal('navigation',undefined);mountProgress();await settle()
  const button=document.querySelector<HTMLButtonElement>('button')!,fragment=document.querySelector<HTMLElement>('#fragment')!,bar=document.querySelector<HTMLElement>('#navigation-progress')!
  button.click();fragment.setAttribute('aria-busy','true');await settle()
  await new Promise(resolve=>setTimeout(resolve,175))
  expect(bar.hidden).toBe(false);expect(button.classList.contains('navigation-pending')).toBe(true)
  fragment.hidden=true;await settle();expect(bar.hidden).toBe(true)
  fragment.hidden=false;await settle();await new Promise(resolve=>setTimeout(resolve,175));expect(bar.hidden).toBe(false)
  fragment.remove();await settle();expect(bar.hidden).toBe(true);expect(button.classList.contains('navigation-pending')).toBe(false)
})

const serverTiming=[{name:'app',duration:12.5,description:''},{name:'render',duration:4,description:''},{name:'load',duration:3,description:''},{name:'db',duration:2,description:''},{name:'dbcount',duration:0,description:'3'},{name:'dberrors',duration:0,description:'1'}]
const navigationTiming={serverTiming,startTime:0,requestStart:5,responseStart:30,responseEnd:40,domContentLoadedEventEnd:50}
function timingEntry(){return {duration:0,entryType:'navigation',name:location.href,startTime:0,toJSON:()=>({})} satisfies PerformanceEntry}
test('diagnostics separates counts from durations and formats sub-millisecond values',()=>{
  expect(parseServerTiming({serverTiming})).toMatchObject({app:12.5,db:2,dbcount:3,dberrors:1})
  expect(parseServerTiming({serverTiming:[]})).toBeNull()
  expect(formatTiming(.46)).toBe('0.46');expect(formatTiming(25)).toBe('25.0');expect(formatTiming(undefined)).toBe('—')
})
test('diagnostics renders only same-origin measured requests and disconnects its observer',async()=>{
  vi.spyOn(performance,'getEntriesByType').mockReturnValue([{...timingEntry(),...navigationTiming}])
  let callback:PerformanceObserverCallback|undefined
  let observer:PerformanceObserver|undefined
  const disconnect=vi.fn()
  class Observer implements PerformanceObserver {constructor(cb:PerformanceObserverCallback){callback=cb;observer=this}observe=vi.fn();disconnect=disconnect;takeRecords=()=>[]}
  vi.stubGlobal('PerformanceObserver',Observer)
  registerController('diagnostics',DiagnosticsController).handleError=errors
  document.body.innerHTML=`<details open data-controller="diagnostics" data-action="keydown->diagnostics#keyboard"><summary>Diagnostics</summary>
    <span data-metric="dbcount"></span><span data-metric="other"></span><span data-part="render"></span>
    ${['navigation','setup','ttfb','transfer','dom'].map(key=>`<span data-browser="${key}"></span>`).join('')}
    <p id="diagnostics-status"></p><table><tbody id="diagnostics-requests"></tbody></table><button data-action="click->diagnostics#clear">Clear</button></details>`
  await settle()
  expect(document.querySelector('[data-metric="dbcount"]')!.textContent).toBe('3')
  expect(document.querySelector('[data-metric="other"]')!.textContent).toBe('5.50')
  expect(document.querySelector('[data-browser="ttfb"]')!.textContent).toBe('25.0')
  const entries=[{entryType:'resource',name:location.origin+'/api/search?q=private',serverTiming},{entryType:'resource',name:'https://other.test/api/search',serverTiming}].map(entry=>({...timingEntry(),...entry}))
  callback!({getEntries:()=>entries,getEntriesByName:name=>entries.filter(entry=>entry.name===name),getEntriesByType:type=>entries.filter(entry=>entry.entryType===type)},observer!)
  const rows=document.querySelectorAll('#diagnostics-requests tr');expect(rows).toHaveLength(1)
  expect(rows[0].textContent).toContain('/api/search');expect(rows[0].textContent).not.toContain('private')
  expect((rows[0]as HTMLElement).dataset.failed).toBe('true')
  document.querySelector<HTMLButtonElement>('button')!.click();expect(document.querySelector('#diagnostics-requests')!.textContent).toContain('Open search')
  const panel=document.querySelector<HTMLDetailsElement>('details')!;panel.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true}))
  expect(panel.open).toBe(false);expect(document.activeElement).toBe(panel.querySelector('summary'))
  panel.remove();await settle();expect(disconnect).toHaveBeenCalledOnce()
})
