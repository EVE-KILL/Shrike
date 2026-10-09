// @vitest-environment happy-dom
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import type {Announcement} from '../shared/api/types.gen'
import type {RelayCoreOptions} from '../src/utils/relaySocketCore'
import {AnnouncementsController,readAnnouncementEvent} from '../src/browser/announcements'
import {createRelayConnection} from '../src/browser/account-data'
import {registerController} from '../src/browser/controllers/application'
vi.mock('../src/browser/account-data',()=>({createRelayConnection:vi.fn()}))

const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0))
const errors=vi.fn(),connect=vi.fn(),disconnect=vi.fn(),dispose=vi.fn()
let options:RelayCoreOptions|undefined
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
const row=(id:number,tier:1|2|3=2):Announcement=>({id,tier,title:'Fixture '+id,body_html:'<strong>Safe server HTML</strong>',body_md:'Plain text',color:'info',icon:null,link_label:'Read',link_url:'/blog/test',created_at:new Date().toISOString(),starts_at:new Date(Date.now()-10000).toISOString(),expires_at:new Date(Date.now()+3600000).toISOString()})
function mount(){
  registerController('announcements',AnnouncementsController).handleError=errors
  document.body.setAttribute('data-controller','announcements')
  document.body.setAttribute('data-action','click->announcements#dismissClick pagehide@window->announcements#pageHide pageshow@window->announcements#pageShow')
  document.body.innerHTML='<div data-announcement-banners hidden></div><dialog data-announcement-modal data-action="cancel->announcements#cancel"><div data-announcement-modal-header></div><div data-announcement-modal-body></div><span data-announcement-modal-count></span><button data-announcement-ack data-action="click->announcements#acknowledge">Acknowledge</button></dialog><div data-announcement-ticker hidden><button data-announcement-expand data-action="click->announcements#toggle">Expand</button><div data-announcement-ticker-expanded><button data-action="click->announcements#toggle">Collapse</button><div data-announcement-scroll></div></div></div>'
}
beforeEach(()=>{
  localStorage.clear();errors.mockClear();connect.mockClear();disconnect.mockClear();dispose.mockClear()
  vi.mocked(createRelayConnection).mockImplementation(input=>{options=input;return {connected:false,connect,disconnect,dispose}})
  fetchMock=vi.fn<typeof fetch>(async path=>Response.json(String(path)==='/auth/me'?{user:null}:{announcements:[row(1),row(-1000000001,1),row(3,3)]}))
  vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{
  document.body.removeAttribute('data-controller');document.body.removeAttribute('data-action');document.body.replaceChildren();await settle()
  expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals()
})
test('announcement surfaces retain acknowledgment, dismissal, duplicate ticker accessibility and collapse behavior',async()=>{
  mount();await settle();await settle()
  const dialog=document.querySelector<HTMLDialogElement>('dialog')!
  expect(dialog.open).toBe(true);expect(dialog.textContent).toContain('Fixture 3')
  const cancel=new Event('cancel',{cancelable:true});dialog.dispatchEvent(cancel);expect(cancel.defaultPrevented).toBe(true)
  document.querySelector<HTMLButtonElement>('[data-announcement-ack]')!.click();await settle();expect(dialog.open).toBe(false)
  expect(document.querySelector('[data-announcement-pass="2"]')!.getAttribute('aria-hidden')).toBe('true')
  expect(document.querySelector('[data-announcement-pass="2"] a')!.getAttribute('tabindex')).toBe('-1')
  document.querySelector<HTMLButtonElement>('[data-announcement-ticker-expanded] button')!.click();await settle()
  expect(document.querySelector<HTMLElement>('[data-announcement-ticker-expanded]')!.hidden).toBe(true)
  document.querySelector<HTMLButtonElement>('[data-announcement-banners] button')!.click();await settle()
  expect(document.querySelector<HTMLElement>('[data-announcement-banners]')!.hidden).toBe(true)
  expect(JSON.parse(localStorage.getItem('ek_dismissed_announcements')!)).toEqual([3,1])
  expect(localStorage.getItem('ek_ticker_collapsed')).toBe('1')
})
test('live updates during a slow snapshot keep the newest data and detach aborts stale work',async()=>{
  let finish:((value:Response)=>void)|undefined,signal:AbortSignal|undefined
  fetchMock.mockImplementation((path,init)=>String(path)==='/auth/me'?Promise.resolve(Response.json({user:null})):new Promise(resolve=>{finish=resolve;signal=init?.signal??undefined}))
  mount();await settle()
  options!.onMessage('announcements',{event_type:'new',announcement:row(2)},[])
  options!.onMessage('announcements',{event_type:'archived',announcement:{id:1}},[])
  finish!(Response.json({announcements:[row(1)]}));await settle();await settle()
  expect(document.querySelector('[data-announcement-banners]')!.textContent).toContain('Fixture 2')
  expect(document.querySelector('[data-announcement-banners]')!.textContent).not.toContain('Fixture 1')
  window.dispatchEvent(Object.assign(new Event('pageshow'),{persisted:true}));await settle()
  document.body.removeAttribute('data-controller');await settle();expect(signal!.aborted).toBe(true);expect(dispose).toHaveBeenCalledOnce()
  finish!(Response.json({announcements:[row(99)]}));await settle()
  expect(document.body.textContent).not.toContain('Fixture 99')
})
test('page departure and bfcache restoration close and reopen the shared announcement relay',async()=>{
  mount();await settle();expect(connect).toHaveBeenCalledOnce()
  window.dispatchEvent(new Event('pagehide'));expect(disconnect).toHaveBeenCalledOnce()
  window.dispatchEvent(Object.assign(new Event('pageshow'),{persisted:true}));await settle();expect(connect).toHaveBeenCalledTimes(2)
})
test('unknown relay data only enters typed rendering after validation',()=>{
  expect(readAnnouncementEvent({event_type:'new',announcement:{id:1,title:42}})).toBeNull()
  expect(readAnnouncementEvent({event_type:'unknown',announcement:row(1)})).toBeNull()
  const announcement=row(1)
  expect(readAnnouncementEvent({event_type:'updated',announcement})).toEqual({event_type:'updated',announcement})
  expect(readAnnouncementEvent({event_type:'expired',announcement:{id:1}})).toEqual({event_type:'expired',announcement:{id:1}})
})
