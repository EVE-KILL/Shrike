// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {AdminController} from '../src/browser/admin'
import {registerController} from '../src/browser/controllers/application'
import {readQuerySeed} from '../src/browser/shared/query-seed'
import type {AdminSummary} from '../src/browser/shared/admin-summary'

const kinds=['overview','users','user','moderation','comments','esi'] as const
type Kind=typeof kinds[number]
const markup=Object.fromEntries(kinds.map(kind=>[kind,readFileSync(resolve(import.meta.dirname,'fixtures/admin',kind+'.html'),'utf8')]))
const summary:AdminSummary={rateLimit:{request_count:1234},responseTime:{avg_ms:12,p95_ms:4001},volumeByHour:[{hour:'2026-10-07T12:00:00Z',total:10,errors:1,new_items:2}]}
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn()
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(root:ParentNode,selector:string):T{const node=root.querySelector<T>(selector);if(!node)throw Error('Missing admin fixture '+selector);return node}
function mount(kind:Kind){
  history.replaceState({},'',kind==='overview'?'/admin':kind==='user'?'/admin/users/4343':'/admin/'+kind)
  if(kind==='user'){document.body.innerHTML=markup.user;return q(document,'[data-admin-user]')}
  document.body.innerHTML=markup.overview;const root=q(document,'[data-admin-root]')
  if(kind!=='overview')q(root,'[data-admin-panels]').innerHTML=markup[kind]
  root.dataset.active=kind;return root
}
function result(kind:Kind){const template=document.createElement('template');template.innerHTML=markup[kind];const next=template.content.querySelector('[data-admin-users-result],[data-admin-review-result]');if(!next)throw Error('Missing fragment');return next.outerHTML}
function pendingReads(){
  const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}>=[]
  fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})))
  return pending
}
beforeEach(()=>{
  errors.mockClear();registerController('admin',AdminController).handleError=errors
  fetchMock=vi.fn<typeof fetch>(async path=>String(path)==='/api/admin/esi'?Response.json(summary):Response.json({}))
  vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('admin attention badges and ESI summary are shared across cached panels without duplicated controls',async()=>{
  fetchMock.mockImplementation(async path=>String(path)==='/api/admin/esi'?Response.json(summary):new Response(markup.esi))
  const root=mount('overview');await settle();await settle()
  expect(q(root,'[data-esi-volume]').querySelectorAll('[data-esi-volume-slot]')).toHaveLength(24)
  q<HTMLAnchorElement>(root,'.admin-nav [data-admin-tab="esi"]').click();await settle();await settle()
  expect(q(root,'[data-admin-esi-count]').textContent).toBe('1,234');expect(q(root,'[data-admin-esi-p95-tone]').classList.contains('text-yellow-400')).toBe(true)
  expect(fetchMock.mock.calls.filter(([path])=>path==='/api/admin/esi')).toHaveLength(1)
  const count=root.querySelectorAll('[data-select-menu-target="trigger"]').length
  q<HTMLAnchorElement>(root,'.admin-nav [data-admin-tab="overview"]').click();q<HTMLAnchorElement>(root,'.admin-nav [data-admin-tab="esi"]').click();await settle()
  expect(root.querySelectorAll('[data-select-menu-target="trigger"]')).toHaveLength(count)
})
test('ESI outlets wait for a parent registered after the child has connected to SSR markup',async()=>{
  const app=registerController('admin',AdminController);app.unload('admin')
  try{
    const root=mount('overview');await settle();await settle();expect(fetchMock).not.toHaveBeenCalled();expect(q(root,'[data-admin-esi-error]').hidden).toBe(true)
    app.register('admin',AdminController);await settle();await settle()
    expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/admin/esi',expect.any(Object));expect(q(root,'[data-esi-volume]').querySelectorAll('[data-esi-volume-slot]')).toHaveLength(24)
  }finally{app.register('admin',AdminController)}
})
test('admin navigation preserves cached user-search input, restores Back and rejects mismatched fragments',async()=>{
  fetchMock.mockImplementation(async path=>String(path)==='/api/admin/esi'?Response.json(summary):new Response(markup.comments))
  const root=mount('users');await settle();const input=q<HTMLInputElement>(root,'[data-admin-user-search]');input.value='Retained search'
  q<HTMLAnchorElement>(root,'.admin-nav [data-admin-tab="moderation"]').click();await settle()
  expect(q(root,'[data-admin-section="moderation"] [role="alert"]').textContent).toContain('Could not load')
  q<HTMLAnchorElement>(root,'.admin-nav [data-admin-tab="users"]').click();expect(q<HTMLInputElement>(root,'[data-admin-user-search]')).toBe(input);expect(input.value).toBe('Retained search')
  history.replaceState({},'','/admin/moderation');window.dispatchEvent(new PopStateEvent('popstate'));await settle()
  expect(root.dataset.active).toBe('moderation')
})
test('superseded admin fragments abort without adopting a stale response',async()=>{
  const root=mount('users');await settle();const pending=pendingReads()
  q<HTMLAnchorElement>(root,'.admin-nav [data-admin-tab="moderation"]').click();q<HTMLAnchorElement>(root,'.admin-nav [data-admin-tab="comments"]').click()
  expect(pending[0].signal.aborted).toBe(true);pending[1].complete(new Response(markup.comments));await settle();pending[0].complete(new Response(markup.moderation));await settle()
  expect(root.dataset.active).toBe('comments');expect(root.querySelector('[data-admin-section="moderation"]')).toBeNull();expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('failed ESI summary reads remain retryable and the successful value replaces the error',async()=>{
  fetchMock.mockResolvedValueOnce(Response.json({message:'Fixture unavailable'},{status:503})).mockResolvedValueOnce(Response.json(summary))
  const root=mount('overview');await settle();await settle();expect(q(root,'[data-admin-esi-error]').textContent).toContain('Fixture unavailable')
  q<HTMLButtonElement>(root,'[data-admin-esi-retry]').click();await settle();expect(q(root,'[data-admin-esi-error]').hidden).toBe(true);expect(fetchMock).toHaveBeenCalledTimes(2)
})
test('users debounce search, keep sort/page query parameters and release permission errors for retry',async()=>{
  const root=mount('users');await settle();vi.useFakeTimers()
  fetchMock.mockResolvedValueOnce(new Response('',{status:403})).mockResolvedValueOnce(new Response(result('users')))
  const input=q<HTMLInputElement>(root,'[data-admin-user-search]');input.value='Fixture';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301)
  const query=new URL(String(fetchMock.mock.calls[0][0]),location.origin).searchParams;expect(query.get('search')).toBe('Fixture');expect(query.get('page')).toBe('1')
  expect(q(root,'[data-admin-users-error]').textContent).toContain('Admin access required.');expect(q(root,'[data-admin-users-results]').hasAttribute('aria-busy')).toBe(false)
  q<HTMLButtonElement>(root,'[data-admin-users-retry]').click();await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-admin-users-error]').hidden).toBe(true)
})
test('user-search supersession aborts the previous body read and retains only the newest result',async()=>{
  const root=mount('users');await settle();const pending=pendingReads();vi.useFakeTimers();const input=q<HTMLInputElement>(root,'[data-admin-user-search]')
  input.value='Older';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301)
  input.value='Newer';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301);expect(pending[0].signal.aborted).toBe(true)
  pending[1].complete(new Response(result('users').replace('data-total="63"','data-total="12"')));await vi.advanceTimersByTimeAsync(0)
  const content=q(root,'[data-admin-users-results]').innerHTML;pending[0].complete(new Response('<div data-admin-users-result>Late directory</div>'));await vi.advanceTimersByTimeAsync(0)
  expect(q(root,'[data-admin-users-results]').innerHTML).toBe(content);expect(root.textContent).not.toContain('Late directory')
})
test('user detail submits trimmed Discord IDs once, replaces its component and ignores late detached updates',async()=>{
  const root=mount('user');await settle();const id=q<HTMLInputElement>(root,'[data-admin-discord-id]');id.value=' 1234567890 '
  fetchMock.mockResolvedValueOnce(Response.json({character_id:4343,discord_user_id:'1234567890'})).mockResolvedValueOnce(new Response(markup.user))
  q<HTMLFormElement>(root,'[data-admin-discord]').dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));await settle();await settle()
  expect(JSON.parse(String(fetchMock.mock.calls[0][1]!.body))).toEqual({discord_user_id:'1234567890'});expect(q(document,'[data-admin-user]')).not.toBe(root)
  const current=q(document,'[data-admin-user]'),pending=pendingReads();q<HTMLButtonElement>(current,'[data-admin-user-toggle]').click();q<HTMLButtonElement>(current,'[data-admin-user-toggle]').click()
  expect(pending).toHaveLength(1);current.remove();await settle();expect(pending[0].signal.aborted).toBe(true);pending[0].complete(Response.json({character_id:4343,is_admin:true}));await settle()
  expect(pending).toHaveLength(1);expect(document.querySelector('[data-admin-user]')).toBeNull()
})
test('a stale user-action finally cannot re-enable controls for a new operation after reconnect',async()=>{
  const root=mount('user');await settle();const pending=pendingReads(),button=q<HTMLButtonElement>(root,'[data-admin-user-toggle]')
  button.click();root.remove();await settle();document.body.append(root);await settle();button.click();expect(pending).toHaveLength(2)
  pending[0].complete(Response.json({message:'Stale failure'},{status:503}));await settle();expect(button.disabled).toBe(true);expect(q(root,'[data-admin-user-error]').hidden).toBe(true)
  q<HTMLInputElement>(root,'[data-admin-discord-id]').value='Newer unrelated draft'
  pending[1].complete(Response.json({character_id:4343,is_admin:false}));await settle();pending[2].complete(new Response(markup.user));await settle()
  expect(q<HTMLInputElement>(document,'[data-admin-discord-id]').value).toBe('Newer unrelated draft')
})
test('user forms display API validation errors and restore spinners and controls without refreshing',async()=>{
  const root=mount('user');await settle();fetchMock.mockResolvedValue(Response.json({message:'Discord ID must be numeric'},{status:400}))
  q<HTMLInputElement>(root,'[data-admin-discord-id]').value='bad-id';q<HTMLFormElement>(root,'[data-admin-discord]').dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));await settle()
  expect(q(root,'[data-admin-discord-error]').textContent).toBe('Discord ID must be numeric');expect(q<HTMLInputElement>(root,'[data-admin-discord-id]').disabled).toBe(false);expect(q(root,'[data-admin-discord-spinner]').hidden).toBe(true);expect(fetchMock).toHaveBeenCalledOnce()
})
test('restoring an interrupted user action reconciles its state with a read without repeating the mutation',async()=>{
  const root=mount('user');await settle();const pending=pendingReads();q<HTMLButtonElement>(root,'[data-admin-user-toggle]').click()
  window.dispatchEvent(new PageTransitionEvent('pagehide'));expect(pending[0].signal.aborted).toBe(true)
  const show=new PageTransitionEvent('pageshow');Object.defineProperty(show,'persisted',{value:true});window.dispatchEvent(show)
  expect(pending).toHaveLength(2);expect(pending[1].path).toBe('/_fragments/admin/users/4343');expect(fetchMock.mock.calls.filter(([,options])=>options?.method==='POST')).toHaveLength(1)
  pending[1].complete(new Response(markup.user));await settle();pending[0].complete(Response.json({message:'Late failure'},{status:503}));await settle()
  expect(q(document,'[data-admin-user-error]').hidden).toBe(true);expect(q<HTMLButtonElement>(document,'[data-admin-user-toggle]').disabled).toBe(false)
})
test('review refresh retains matching media elements while filters preserve queue query state',async()=>{
  const root=mount('moderation');await settle();const host=q(root,'[data-admin-review-results]'),image=q<HTMLImageElement>(host,'img')
  fetchMock.mockImplementation(async()=>new Response(result('moderation')))
  q<HTMLButtonElement>(root,'[data-admin-review-refresh]').click();await settle();expect(q<HTMLImageElement>(host,'img')).toBe(image)
  const filter=q<HTMLButtonElement>(root,'[data-admin-review-filter="status"][data-value="approved"]');filter.click();await settle()
  expect(new URL(String(fetchMock.mock.calls.at(-1)![0]),location.origin).searchParams.get('status')).toBe('approved');expect(filter.getAttribute('aria-pressed')).toBe('true')
  expect(q(root,'[data-admin-review-error]').hidden).toBe(true)
})
test('review actions block duplicate submissions and keep API failures visible without disabling future work',async()=>{
  const root=mount('moderation');await settle();const pending=pendingReads(),button=q<HTMLButtonElement>(root,'[data-admin-review-action="approve"]'),row=button.closest<HTMLElement>('[data-admin-review-id]')!
  button.click();button.click();expect(pending).toHaveLength(1);expect(pending[0].path).toBe('/api/admin/moderation/'+row.dataset.adminReviewId+'/approve')
  pending[0].complete(Response.json({message:'Fixture moderation unavailable'},{status:503}));await settle();expect(button.disabled).toBe(false);expect(document.body.textContent).toContain('Fixture moderation unavailable')
})
test('comment deletion requires a second click and times out without sending a delete',async()=>{
  const root=mount('comments');await settle();vi.useFakeTimers();const button=q<HTMLButtonElement>(root,'[data-admin-review-action="delete"]')
  button.click();expect(fetchMock).not.toHaveBeenCalled();expect(button.textContent).toBe('Confirm delete');await vi.advanceTimersByTimeAsync(5001);expect(button.textContent).toBe('Delete')
  const pending=pendingReads();button.click();button.click();expect(pending).toHaveLength(1);expect(fetchMock.mock.calls[0][1]!.method).toBe('DELETE')
  pending[0].complete(Response.json({message:'Fixture delete unavailable'},{status:503}));await vi.advanceTimersByTimeAsync(0);expect(button.disabled).toBe(false)
})
test('hidden review queues cancel loading and restore interrupted filters on visibility',async()=>{
  const root=mount('comments');await settle();const pending=pendingReads(),section=q(root,'[data-admin-section]')
  q<HTMLButtonElement>(root,'[data-admin-review-refresh]').click();section.hidden=true;section.dispatchEvent(new CustomEvent('admin-visibility',{detail:false}));expect(pending[0].signal.aborted).toBe(true)
  section.hidden=false;section.dispatchEvent(new CustomEvent('admin-visibility',{detail:true}));expect(pending).toHaveLength(2)
  pending[1].complete(new Response(result('comments')));await settle();const before=q(root,'[data-admin-review-results]').innerHTML
  pending[0].complete(new Response('<div data-admin-review-result>Late moderation</div>'));await settle();expect(q(root,'[data-admin-review-results]').innerHTML).toBe(before)
})
test('query seeds keep scalar strings and discard nested or malformed input',()=>{
  expect(readQuerySeed('{"page":["2"],"sort":"lastLogin","bad":[42],"nested":{}}')).toEqual({page:'2',sort:'lastLogin'});expect(readQuerySeed('broken')).toEqual({})
})
