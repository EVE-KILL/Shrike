// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {SavedSearchesController} from '../src/browser/saved-searches'
import {readBrowserSession,type BrowserUser} from '../src/browser/browser-request'
import {registerController} from '../src/browser/controllers/application'
import {savedSearchDocument,type SavedSearchDocument} from '../src/utils/savedSearches'
import type {AccountSavedSearchesGetResponses} from '../shared/api/types.gen'

vi.mock('../src/browser/browser-request',async original=>({...await original<typeof import('../src/browser/browser-request')>(),readBrowserSession:vi.fn()}))
const markup=readFileSync(resolve(import.meta.dirname,'fixtures/advanced-search.html'),'utf8'),settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn()
const owner:BrowserUser={characterId:4242,characterName:'Fixture Owner',isAdmin:false,corporationId:null,corporationName:null,allianceId:null,allianceName:null,lastSeenNotificationId:0}
const documentModel=savedSearchDocument('{"iskMin":"500m"}')
type Search=AccountSavedSearchesGetResponses[200]['searches'][number]
const saved:Search={id:7,name:'Fixture account search',revision:1,document:documentModel,created_at:'2026-10-07T12:00:00Z',updated_at:'2026-10-07T12:00:00Z'}
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(root:ParentNode,selector:string):T{const value=root.querySelector<T>(selector);if(!value)throw Error('Missing saved-search fixture '+selector);return value}
async function mount(){document.body.innerHTML=markup;const root=q(document,'[data-advanced-search]'),load=vi.fn<(document:SavedSearchDocument)=>void>();root.addEventListener('saved-searches:current',event=>{if(event instanceof CustomEvent)event.detail.document=documentModel});root.addEventListener('saved-searches:load',event=>{if(event instanceof CustomEvent)load(event.detail.document)});await settle();await settle();return {root,load}}
function start(root:ParentNode,name='New fixture search'){q<HTMLButtonElement>(root,'[data-saved-create]').click();const input=q<HTMLInputElement>(root,'[data-saved-editor] [name="name"]');input.value=name;input.dispatchEvent(new Event('input',{bubbles:true}));return q<HTMLFormElement>(root,'[data-saved-editor]')}
function submit(form:HTMLFormElement){form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}))}
function pendingReads(){const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}>=[];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
beforeEach(()=>{errors.mockClear();registerController('saved-searches',SavedSearchesController).handleError=errors;localStorage.clear();history.replaceState({},'','/advancedsearch');vi.mocked(readBrowserSession).mockResolvedValue(null);fetchMock=vi.fn<typeof fetch>(async path=>String(path)==='/auth/me'?Response.json({user:owner}):Response.json({searches:[saved],limit:50}));vi.stubGlobal('fetch',fetchMock)})
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('anonymous searches save canonical filters locally, load them, and reject duplicate names',async()=>{
  const {root,load}=await mount();submit(start(root));await settle();expect(fetchMock).not.toHaveBeenCalled();expect(q(root,'[data-saved-message]').textContent).toBe('Saved in this browser.')
  const rows=JSON.parse(localStorage.getItem('evekill-saved-searches')!);expect(rows[0].document.filters.iskMin).toBe(500e6);q<HTMLButtonElement>(root,'[data-saved-row-action="load-local"]').click();expect(load).toHaveBeenCalledWith(documentModel);expect(q(root,'[data-saved-panel]').hidden).toBe(true)
  submit(start(root,' new FIXTURE search '));await settle();expect(q(root,'[data-saved-error]').textContent).toContain('already exists');expect(JSON.parse(localStorage.getItem('evekill-saved-searches')!)).toHaveLength(1)
})
test('invalid names and storage errors keep the editor available',async()=>{
  const {root}=await mount();submit(start(root,'x'.repeat(101)));await settle();expect(q(root,'[data-saved-error]').textContent).toContain('1–100');expect(q<HTMLFormElement>(root,'[data-saved-editor]').hidden).toBe(false)
  vi.stubGlobal('localStorage',{setItem:()=>{throw Error('Fixture storage quota exceeded')},getItem:()=>null});submit(start(root));await settle();expect(q(root,'[data-saved-error]').textContent).toContain('quota exceeded');expect(q<HTMLButtonElement>(root,'[data-saved-submit]').disabled).toBe(false)
})
test('account saves carry revisions, preserve a newer name and use the returned revision on the next save',async()=>{
  vi.mocked(readBrowserSession).mockResolvedValue(owner);const {root}=await mount();const pending=pendingReads(),form=start(root);submit(form);submit(form);expect(pending).toHaveLength(1);expect(JSON.parse(String(fetchMock.mock.calls.at(-1)![1]!.body))).toMatchObject({name:'New fixture search',revision:0,document:documentModel})
  const input=q<HTMLInputElement>(root,'[data-saved-editor] [name="name"]');input.value='Newer name';input.dispatchEvent(new Event('input',{bubbles:true}));pending[0].complete(Response.json({...saved,id:8,revision:2}));await settle();pending[1].complete(Response.json({searches:[saved],limit:50}));await settle();expect(form.hidden).toBe(false);expect(input.value).toBe('Newer name')
  submit(form);expect(pending[2].path).toBe('/api/me/saved-searches/8');expect(fetchMock.mock.calls.at(-1)![1]!.method).toBe('PUT');expect(JSON.parse(String(fetchMock.mock.calls.at(-1)![1]!.body)).revision).toBe(2)
})
test('account permission failures show an error and refresh successfully clears it',async()=>{
  vi.mocked(readBrowserSession).mockResolvedValue(owner);fetchMock.mockResolvedValueOnce(Response.json({message:'Fixture permission denied'},{status:403}));const {root}=await mount();expect(q(root,'[data-saved-error]').textContent).toBe('Fixture permission denied');expect(q<HTMLButtonElement>(root,'[data-saved-create]').disabled).toBe(true)
  q<HTMLButtonElement>(root,'[data-saved-toggle]').click();await settle();await settle();expect(q(root,'[data-saved-error]').hidden).toBe(true);expect(q<HTMLButtonElement>(root,'[data-saved-create]').disabled).toBe(false)
})
test('refresh keeps the last account rows through a slow or failed read without enabling stale mutations',async()=>{
  vi.mocked(readBrowserSession).mockResolvedValue(owner);const {root}=await mount(),pending=pendingReads()
  q<HTMLButtonElement>(root,'[data-saved-refresh]').click();expect(q(root,'[data-saved-account-rows]').textContent).toContain(saved.name);expect(q<HTMLButtonElement>(root,'[data-saved-row-action="rename"]').disabled).toBe(true)
  pending[0].complete(Response.json({message:'Temporary outage'},{status:503}));await settle();expect(q(root,'[data-saved-account-rows]').textContent).toContain(saved.name);expect(q(root,'[data-saved-error]').textContent).toBe('Temporary outage');expect(q<HTMLButtonElement>(root,'[data-saved-row-action="load-account"]').disabled).toBe(false)
  q<HTMLButtonElement>(root,'[data-saved-refresh]').click();pending[1].complete(Response.json({searches:[saved],limit:50}));await settle();expect(q(root,'[data-saved-error]').hidden).toBe(true);expect(q<HTMLButtonElement>(root,'[data-saved-row-action="rename"]').disabled).toBe(false)
})
test('importing preserves the browser copy and sends its canonical document to the account',async()=>{
  localStorage.setItem('evekill-saved-searches',JSON.stringify([{name:'Legacy local',filters:'{"iskMin":"1.5b"}',date:'2026-01-01'}]));vi.mocked(readBrowserSession).mockResolvedValue(owner);const {root}=await mount();q<HTMLButtonElement>(root,'[data-saved-toggle]').click();await settle();q<HTMLButtonElement>(root,'[data-saved-row-action="import"]').click();submit(q<HTMLFormElement>(root,'[data-saved-editor]'));await settle();await settle()
  expect(JSON.parse(String(fetchMock.mock.calls.find(([,options])=>options?.method==='POST')![1]!.body)).document.filters.iskMin).toBe(1.5e9);expect(JSON.parse(localStorage.getItem('evekill-saved-searches')!)).toHaveLength(1);expect(q(root,'[data-saved-message]').textContent).toContain('browser copy has been kept')
})
test('deletion is explicitly confirmed and late aborted writes cannot affect a restored editor',async()=>{
  vi.mocked(readBrowserSession).mockResolvedValue(owner);const {root}=await mount();q<HTMLButtonElement>(root,'[data-saved-toggle]').click();await settle();q<HTMLButtonElement>(root,'[data-saved-row-action="delete-account"]').click();const before=fetchMock.mock.calls.length;expect(q(root,'[data-saved-deleting]').hidden).toBe(false);expect(fetchMock).toHaveBeenCalledTimes(before)
  const pending=pendingReads();q<HTMLButtonElement>(root,'[data-saved-delete-confirm]').click();q<HTMLButtonElement>(root,'[data-saved-delete-confirm]').click();expect(pending).toHaveLength(1);expect(pending[0].path).toBe('/api/me/saved-searches/7?revision=1')
  window.dispatchEvent(new PageTransitionEvent('pagehide'));expect(pending[0].signal.aborted).toBe(true);expect(q(root,'[data-saved-deleting]').hidden).toBe(true);const show=new PageTransitionEvent('pageshow');Object.defineProperty(show,'persisted',{value:true});window.dispatchEvent(show);pending[1].complete(Response.json({user:owner}));await settle();pending[2].complete(Response.json({searches:[saved],limit:50}));await settle();pending[0].complete(Response.json({message:'Late failure'},{status:503}));await settle();expect(q(root,'[data-saved-error]').hidden).toBe(true);expect(fetchMock.mock.calls.filter(([,options])=>options?.method==='DELETE')).toHaveLength(1)
})
test('identity changes cancel account reads and replace them with the new account scope',async()=>{
  vi.mocked(readBrowserSession).mockResolvedValue(owner);const {root}=await mount(),pending=pendingReads();q<HTMLButtonElement>(root,'[data-saved-refresh]').click();window.dispatchEvent(new Event('focus'));expect(pending).toHaveLength(2);pending[1].complete(Response.json({user:null}));await settle();expect(pending[0].signal.aborted).toBe(true);pending[0].complete(Response.json({searches:[saved],limit:50}));await settle();expect(q(root,'[data-saved-title]').textContent).toBe('Browser searches');expect(q(root,'[data-saved-count]').textContent).toBe('Saved (0)')
})
test('disconnect owns aborts, generated controls and document/window listeners across reconnection',async()=>{
  vi.mocked(readBrowserSession).mockResolvedValue(owner);const {root}=await mount(),pending=pendingReads();q<HTMLButtonElement>(root,'[data-saved-refresh]').click();root.remove();await settle();expect(pending[0].signal.aborted).toBe(true);window.dispatchEvent(new Event('focus'));expect(pending).toHaveLength(1)
  document.body.append(root);await settle();pending[1].complete(Response.json({searches:[saved],limit:50}));await settle();expect(root.querySelectorAll('[data-saved-row-action="rename"]')).toHaveLength(1);pending[0].complete(Response.json({message:'Late failure'},{status:503}));await settle();expect(q(root,'[data-saved-error]').hidden).toBe(true)
})
