// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
import {createEntitySearchSelect,EntitySearchSelectController} from '../src/browser/entity-search-select'
import {WalletController} from '../src/browser/wallet'
import {EsiLogController,initializeEsiLog,renderEsiVolume} from '../src/browser/esi'
import {registerController} from '../src/browser/controllers/application'
import type {EsiLogRow} from '../src/utils/esiLog'
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn()
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);errors.mockClear();vi.unstubAllGlobals();vi.restoreAllMocks()})

test('entity picker keeps its detached API, emits validated selections and clears without duplicate events',async()=>{
  registerController('entity-search-select',EntitySearchSelectController).handleError=errors
  vi.stubGlobal('fetch',vi.fn<typeof fetch>(async()=>Response.json({results:[{id:42,name:'<Pilot>',type:'character'},{id:99,name:'Wrong category',type:'alliance'}]})))
  const change=vi.fn(),picker=createEntitySearchSelect('Pilot / Corp',change)
  picker.setValue({id:10,name:'Initial',type:'corporation'});expect(picker.root.textContent).toContain('Initial')
  document.body.append(picker.root);await settle();picker.root.querySelector<HTMLButtonElement>('[data-entity-open]')!.click()
  const input=picker.root.querySelector<HTMLInputElement>('input')!;vi.useFakeTimers();input.value='Pilot';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(260)
  expect(picker.root.querySelectorAll('[data-entity-hit]')).toHaveLength(1);expect(picker.root.querySelector('pilot')).toBeNull()
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true,cancelable:true}));expect(document.activeElement!.textContent).toContain('<Pilot>')
  picker.root.querySelector<HTMLButtonElement>('[data-entity-hit]')!.click();expect(change).toHaveBeenCalledExactlyOnceWith({id:42,name:'<Pilot>',type:'character'})
  picker.root.querySelector<HTMLButtonElement>('[data-entity-clear]')!.click();expect(change).toHaveBeenLastCalledWith(null)
})
test('entity picker aborts Escape/detach reads and never renders a late result after closure',async()=>{
  const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[]
  vi.stubGlobal('fetch',vi.fn<typeof fetch>((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve}))))
  const picker=createEntitySearchSelect('Pilot / Corp',()=>{});document.body.append(picker.root);await settle();vi.useFakeTimers()
  const input=picker.root.querySelector<HTMLInputElement>('input')!
  picker.root.querySelector<HTMLButtonElement>('[data-entity-open]')!.click();input.value='Pilot';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(260)
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true}));expect(pending[0].signal.aborted).toBe(true)
  pending[0].complete(Response.json({results:[{id:42,name:'Late pilot',type:'character'}]}));await vi.advanceTimersByTimeAsync(0)
  expect(picker.root.querySelector<HTMLElement>('[data-entity-results]')!.hidden).toBe(true);expect(picker.root.textContent).not.toContain('Late pilot')
  picker.root.querySelector<HTMLButtonElement>('[data-entity-open]')!.click();input.value='Next';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(260)
  picker.dispose();expect(pending[1].signal.aborted).toBe(true);pending[1].complete(Response.json({results:[]}));await vi.advanceTimersByTimeAsync(0)
})
function wallet(){
  registerController('wallet',WalletController).handleError=errors
  document.body.innerHTML='<div data-controller="wallet" data-action="change->wallet#change click->wallet#click popstate@window->wallet#restore"><div data-wallet-content><select data-wallet-division><option value="">All</option><option value="2">Second</option></select><button data-wallet-page-number="2">Next</button><time datetime="2026-10-07T12:00:00Z"></time></div></div>'
  return document.body.firstElementChild as HTMLElement
}
test('wallet navigation keeps division/page query semantics and releases loading after failure',async()=>{
  const fetchMock=vi.fn<typeof fetch>(async()=>new Response('',{status:503}));vi.stubGlobal('fetch',fetchMock)
  const root=wallet();await settle();const select=root.querySelector<HTMLSelectElement>('select')!;select.value='2';select.dispatchEvent(new Event('change',{bubbles:true}));await settle()
  expect(new URL(String(fetchMock.mock.calls[0][0])).searchParams.get('division')).toBe('2');expect(root.querySelector('[role=alert]')).not.toBeNull()
  expect(root.querySelector('[data-wallet-content]')!.hasAttribute('aria-busy')).toBe(false)
  root.querySelector<HTMLButtonElement>('button')!.click();await settle();expect(root.querySelectorAll('[role=alert]')).toHaveLength(1)
  expect(new URL(String(fetchMock.mock.calls[1][0])).searchParams.get('page')).toBe('2')
})
test('wallet disconnect aborts and ignores a stale response without changing history',async()=>{
  let complete:((response:Response)=>void)|undefined,signal:AbortSignal|undefined
  vi.stubGlobal('fetch',vi.fn<typeof fetch>((_path,options)=>new Promise(resolve=>{complete=resolve;signal=options?.signal??undefined})))
  const historySpy=vi.spyOn(history,'pushState'),root=wallet();await settle();root.querySelector<HTMLButtonElement>('button')!.click();root.remove();await settle()
  expect(signal!.aborted).toBe(true);complete!(new Response('<p>Late journal</p>'));await settle();expect(root.textContent).not.toContain('Late journal');expect(historySpy).not.toHaveBeenCalled()
})

const row=(id=1):EsiLogRow=>({id,created_at:'2026-10-07T12:00:00Z',endpoint:'/characters/42/killmails/recent/',endpoint_type:'character',endpoint_action:'killmails',method:'GET',source:'fixture',status_code:200,success:true,error_message:null,request_duration_ms:12,items_returned:2,new_items:2,endpoint_entity_id:42,endpoint_entity_name:'Pilot',character_id:42,character_name:'Pilot',new_item_ids:[100,101],stored_item_ids:[100]})
function esi(){
  registerController('esi-log',EsiLogController).handleError=errors
  const section=document.createElement('section');section.dataset.adminSection='esi'
  section.innerHTML='<div data-esi-log data-admin="true" data-show-entity="true" data-endpoint="/api/admin/esi-logs"><div data-esi-log-filters><input data-esi-log-search></div><div data-esi-log-actions></div><span data-esi-log-count></span><button data-esi-log-live><svg><use></use></svg></button><p data-esi-log-error hidden></p><div data-esi-log-table></div><nav data-esi-log-pagination><span data-esi-log-page-label></span><button data-esi-log-page="prev">Previous</button><button data-esi-log-page="next">Next</button></nav><dialog data-esi-log-dialog><div data-esi-log-detail-header></div><div data-esi-log-detail-body></div><button data-esi-log-detail-close>Close</button></dialog></div>'
  const root=section.firstElementChild as HTMLElement;root.dataset.initial=JSON.stringify({rows:[row()],total:1,pages:2,sources:['fixture']});document.body.append(section)
  return {section,root,controller:initializeEsiLog(root)}
}
test('ESI details keep entity metadata and distinguish stored discovered killmails from pending IDs',async()=>{
  vi.stubGlobal('fetch',vi.fn<typeof fetch>(async()=>Response.json({rows:[],total:0,pages:0,sources:[]})))
  const {root}=esi();await settle();await settle();const entry=root.querySelector<HTMLElement>('[data-esi-log-row]')!
  entry.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true,cancelable:true}))
  const dialog=root.querySelector<HTMLDialogElement>('dialog')!;expect(dialog.open).toBe(true);expect(dialog.textContent).toContain('Pilot')
  expect(dialog.querySelector('a[href="/kill/100"]')).not.toBeNull();expect(dialog.querySelector('a[href="/kill/101"]')).toBeNull();expect(dialog.textContent).toContain('101 pending')
  root.querySelector<HTMLButtonElement>('[data-esi-log-detail-close]')!.click();expect(dialog.open).toBe(false)
})
test('ESI filters remain retryable and disconnect removes requests, timers and generated menus',async()=>{
  const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[]
  vi.stubGlobal('fetch',vi.fn<typeof fetch>((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve}))))
  const {section,root,controller}=esi();await settle();await settle();void controller.reload();expect(root.getAttribute('aria-busy')).toBe('true')
  pending[0].complete(Response.json({message:'Unavailable'},{status:503}));await settle();expect(root.querySelector('[data-esi-log-error]')!.textContent).toContain('Unavailable')
  root.querySelector<HTMLButtonElement>('[data-esi-log-retry]')!.click();expect(pending).toHaveLength(2)
  section.hidden=true;section.dispatchEvent(new Event('admin-visibility'));expect(pending[1].signal.aborted).toBe(true)
  section.hidden=false;section.dispatchEvent(new Event('admin-visibility'));expect(pending).toHaveLength(3)
  section.remove();await settle();expect(pending[2].signal.aborted).toBe(true);expect(root.querySelectorAll('[data-select-menu-target="trigger"]')).toHaveLength(0)
  for(const item of pending.slice(1))item.complete(Response.json({rows:[row(99)],sources:[],total:99,pages:1}));await settle();expect(root.querySelector('[data-esi-log-count]')!.textContent).toBe('1 rows')
})
test('ESI live polling merges once, pauses away from page one, and stops on detach',async()=>{
  const fetchMock=vi.fn<typeof fetch>(async()=>Response.json({rows:[row(2),row(1)]}));vi.stubGlobal('fetch',fetchMock)
  const {root,section}=esi();await settle();await settle();vi.useFakeTimers()
  // Existing real interval is restarted by the explicit live toggle.
  root.querySelector<HTMLButtonElement>('[data-esi-log-live]')!.click();root.querySelector<HTMLButtonElement>('[data-esi-log-live]')!.click()
  await vi.advanceTimersByTimeAsync(5000);expect(fetchMock).toHaveBeenCalledOnce();expect(root.querySelector('[data-esi-log-count]')!.textContent).toBe('2 rows')
  root.querySelector<HTMLButtonElement>('[data-esi-log-page="next"]')!.click();await vi.advanceTimersByTimeAsync(0);const count=fetchMock.mock.calls.length;await vi.advanceTimersByTimeAsync(5000);expect(fetchMock.mock.calls.length).toBe(count)
  section.remove();await vi.advanceTimersByTimeAsync(0);await vi.advanceTimersByTimeAsync(10000);expect(fetchMock.mock.calls.length).toBe(count)
})
test('ESI volume keeps payload-anchored 24-hour chart geometry',()=>{
  const root=document.createElement('div');root.dataset.hours=JSON.stringify([{hour:'2026-10-07 12:00:00+00',total:10,errors:2,new_items:3}]);root.dataset.newItems='true';renderEsiVolume(root)
  expect(root.querySelectorAll('[data-esi-volume-slot]')).toHaveLength(24);expect(root.querySelector('[title*="3 new items"]')).not.toBeNull()
})
