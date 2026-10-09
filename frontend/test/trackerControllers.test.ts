// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {TrackersController} from '../src/browser/trackers'
import {TrackerDashboardController} from '../src/browser/tracker-dashboard'
import {registerController} from '../src/browser/controllers/application'
import {readTrackerSeed} from '../src/browser/shared/tracker-seeds'
import {readDashboardSeed} from '../src/browser/shared/dashboard-seeds'
import {BrowserRequestError} from '../src/browser/browser-request'
import {createTracker,updateTracker} from '../src/browser/tracker-state'
import type {TrackerState} from '../src/browser/tracker-state'
import type {BrowserUser} from '../src/browser/browser-request'
import type {EntityTracker} from '../src/models/trackers'

const mock=vi.hoisted(()=>{
  const user:BrowserUser={characterId:4242,characterName:'Fixture Owner',isAdmin:false,corporationId:null,corporationName:null,allianceId:null,allianceName:null,lastSeenNotificationId:0}
  const state:TrackerState={user,trackers:[],limit:50,notifications:[],unread:0,loadingTrackers:false,loadingNotifications:false,error:null,browserAlerts:false,browserPermission:'unsupported'}
  return {user,state,listeners:new Set<(state:TrackerState)=>void>(),unsubscribe:vi.fn()}
})
vi.mock('../src/browser/browser-request',async original=>({...await original<typeof import('../src/browser/browser-request')>(),readBrowserSession:vi.fn(async()=>mock.user)}))
vi.mock('../src/browser/tracker-state',()=>({
  trackerState:mock.state,setTrackerUser:vi.fn(async()=>{}),refreshTrackers:vi.fn(async()=>{}),setBrowserAlerts:vi.fn(async()=>false),
  subscribeTrackers:(listener:(state:TrackerState)=>void)=>{mock.listeners.add(listener);listener(mock.state);return ()=>{mock.listeners.delete(listener);mock.unsubscribe()}},
  createTracker:vi.fn(),updateTracker:vi.fn(),deleteTracker:vi.fn(async()=>{}),
}))
vi.mock('../src/browser/account-data',async original=>({...await original<typeof import('../src/browser/account-data')>(),createRelayConnection:()=>({connected:false,connect:vi.fn(),disconnect:vi.fn(),dispose:vi.fn()})}))
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn()
const markup={trackers:readFileSync(resolve(import.meta.dirname,'fixtures/trackers.html'),'utf8'),tracker:readFileSync(resolve(import.meta.dirname,'fixtures/tracker.html'),'utf8')}
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function notify(){for(const listener of mock.listeners)listener(mock.state)}
function mount(kind:'trackers'|'tracker'){
  document.body.innerHTML=markup[kind];const root=document.body.firstElementChild!
  if(!(root instanceof HTMLElement))throw new Error('Missing test root')
  const body=root.querySelector<HTMLElement>('[data-dashboard-body]')
  mock.state.trackers=readTrackerSeed(kind==='trackers'?root.dataset.trackersSeed:body?.dataset.dashboardTrackers)
  root.dataset.controller=kind==='trackers'?'trackers':'tracker-dashboard'
  root.dataset.action=kind==='trackers'?'click->trackers#click search-pick->trackers#pick':'click->tracker-dashboard#click change->tracker-dashboard#change input->tracker-dashboard#input search-pick->tracker-dashboard#pick pagehide@window->tracker-dashboard#hide pageshow@window->tracker-dashboard#show'
  return root
}
beforeEach(()=>{
  mock.listeners.clear();mock.unsubscribe.mockClear();mock.state.trackers=[];mock.state.error=null;mock.state.loadingTrackers=false;errors.mockClear()
  registerController('trackers',TrackersController).handleError=errors;registerController('tracker-dashboard',TrackerDashboardController)
  vi.mocked(createTracker).mockReset();vi.mocked(updateTracker).mockReset()
  fetchMock=vi.fn<typeof fetch>(async path=>String(path).includes('/killmails')?new Response('<div class="kill-list"><span class="kill-day-heading"></span><div class="kill-rows"></div></div>'):new Response('<p>Loaded widget</p>'))
  vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);expect(mock.listeners.size).toBe(0);vi.unstubAllGlobals()})
test('tracker controls use generated toggle fields and release their shared subscription on detach',async()=>{
  const root=mount('trackers');await settle();await settle()
  expect(root.querySelector('[data-trackers-count]')!.textContent).toBe('2/50');expect(mock.listeners.size).toBe(1)
  vi.mocked(updateTracker).mockImplementation(async(id,body)=>{const item=mock.state.trackers.find(row=>row.id===id)!;Object.assign(item,body);notify();return item})
  const row=root.querySelector<HTMLButtonElement>('[data-tracker-select]')!;row.focus()
  root.querySelector<HTMLButtonElement>('[data-tracker-toggle="enabled"]')!.click();await settle();await settle()
  expect(updateTracker).toHaveBeenCalledWith(Number(row.dataset.trackerSelect),{enabled:false})
  expect(root.querySelector('[data-tracker-toggle="enabled"]')!.textContent).toContain('Paused')
  root.remove();await settle();expect(mock.unsubscribe).toHaveBeenCalledOnce();const count=fetchMock.mock.calls.length;notify();await settle();expect(fetchMock.mock.calls.length).toBe(count)
})
test('tracker selection rejects invalid/duplicate targets and surfaces session expiry on failed creation',async()=>{
  const root=mount('trackers');await settle();await settle();const first=mock.state.trackers[0]
  root.dispatchEvent(new CustomEvent('search-pick',{bubbles:true,detail:{id:first.target_id,type:first.target_type}}))
  root.dispatchEvent(new CustomEvent('search-pick',{bubbles:true,detail:{id:42,type:'ship'}}));expect(createTracker).not.toHaveBeenCalled()
  vi.mocked(createTracker).mockRejectedValue(new BrowserRequestError('Expired',401))
  root.dispatchEvent(new CustomEvent('search-pick',{bubbles:true,detail:{id:43,type:'character'}}));await settle();await settle()
  expect(createTracker).toHaveBeenCalledWith({target_id:43,target_type:'character',notifications_enabled:false})
  expect(root.querySelector('[data-trackers-error-message]')!.textContent).toContain('session expired')
})
test('dashboard editor saves widget moves, enabled state, kill type and text content through its original API',async()=>{
  const root=mount('tracker'),body=root.querySelector<HTMLElement>('[data-dashboard-body]')!
  body.dataset.dashboardConfig=JSON.stringify({configured:true,widgets:{top:[],left:[{type:'textBlock',enabled:true,content:'Original'}],right:[{type:'killList',enabled:true,killlistType:'latest'}],columnRatio:'250px_1fr'}})
  fetchMock.mockImplementation(async(path)=>String(path).startsWith('/_fragments/tracker?')||String(path)==='/_fragments/tracker'?new Response(body.outerHTML):String(path)==='/api/me/tracked/config'?Response.json({configured:true}):new Response('<p>Loaded widget</p>'))
  await settle();await settle();root.querySelector<HTMLButtonElement>('[data-dashboard-customize]')!.click();await settle()
  const text=root.querySelector<HTMLTextAreaElement>('[data-widget-content]')!;text.value='Edited text';text.dispatchEvent(new Event('input',{bubbles:true}))
  const destination=root.querySelector<HTMLSelectElement>('[data-widget-destination]')!;destination.value='top';destination.dispatchEvent(new Event('change',{bubbles:true}));await settle()
  const killRow=root.querySelector<HTMLElement>('[data-dashboard-widget-row][data-widget-section="right"]')!
  const enabled=killRow.querySelector<HTMLInputElement>('[data-widget-enabled]')!;enabled.checked=false;enabled.dispatchEvent(new Event('change',{bubbles:true}))
  const kind=killRow.querySelector<HTMLSelectElement>('[data-widget-killtype]')!;kind.value='solo';kind.dispatchEvent(new Event('change',{bubbles:true}))
  root.querySelector<HTMLButtonElement>('[data-dashboard-save]')!.click();await settle();await settle()
  const call=fetchMock.mock.calls.find(([path])=>path==='/api/me/tracked/config')!
  expect(JSON.parse(String(call[1]!.body))).toMatchObject({widgets:{top:[{type:'textBlock',content:'Edited text'}],left:[],right:[{type:'killList',enabled:false,killlistType:'solo'}]}})
  expect(root.querySelector('[data-dashboard-body]')).not.toBe(body)
})
test('disconnect aborts dashboard fragments and stale responses cannot update the detached page',async()=>{
  const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[]
  fetchMock.mockImplementation((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve})))
  const root=mount('tracker'),body=root.querySelector<HTMLElement>('[data-dashboard-body]')!
  const seed=readDashboardSeed(body.dataset.dashboardConfig)!;seed.configured=true;body.dataset.dashboardConfig=JSON.stringify(seed)
  root.querySelector<HTMLElement>('[data-dashboard-board]')!.hidden=false
  await settle();await settle();expect(pending.length).toBeGreaterThan(0)
  root.remove();await settle();expect(pending.every(request=>request.signal.aborted)).toBe(true)
  for(const request of pending)request.complete(new Response('<p>Late fragment</p>'));await settle();expect(root.textContent).not.toContain('Late fragment')
})
test('embedded tracker/dashboard seeds discard malformed data instead of granting it a trusted type',()=>{
  expect(readTrackerSeed('[{"id":1}]')).toEqual([]);expect(readDashboardSeed('broken')).toBeNull()
  expect(readDashboardSeed(JSON.stringify({configured:false,widgets:{left:[{type:'invalid',enabled:true}],right:[],top:[]}}))!.widgets.left).toEqual([])
})
