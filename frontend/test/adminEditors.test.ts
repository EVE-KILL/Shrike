// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {registerController} from '../src/browser/controllers/application'
import {AdminAnnouncementsController} from '../src/browser/admin-announcements'
import '../src/browser/admin-blog'
import '../src/browser/admin-campaigns'
import '../src/browser/admin-domains'
import '../src/browser/admin-river'
import '../src/browser/admin-wallet'
import {initializeDateTimePickers,setDateTimeZone} from '../src/browser/date-time-picker'
import {readAnnouncementSeed,readBlogSeed} from '../src/browser/shared/admin-editor-seeds'
import {readRiverJobs,readWalletSettlements} from '../src/browser/shared/admin-runtime-seeds'

const kinds=['announcements','blog','campaigns','domains','domain','river','wallet'] as const
type Kind=typeof kinds[number]
const markup=Object.fromEntries(kinds.map(kind=>[kind,readFileSync(resolve(import.meta.dirname,'fixtures/admin',kind+'.html'),'utf8')]))
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn()
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(root:ParentNode,selector:string):T{const node=root.querySelector<T>(selector);if(!node)throw Error('Missing fixture '+selector);return node}
function mount(kind:Kind){history.replaceState({},'','/admin/'+kind);document.body.innerHTML=markup[kind];return q(document,kind==='domain'?'[data-admin-domain-detail]':'[data-admin-section]')}
function result(kind:Kind,selector:string){const template=document.createElement('template');template.innerHTML=markup[kind];return q(template.content,selector).outerHTML}
function input(root:ParentNode,selector:string,value:string){const node=q<HTMLInputElement>(root,selector);node.value=value;node.dispatchEvent(new Event('input',{bubbles:true}));return node}
function visibility(root:HTMLElement,value:boolean){root.hidden=!value;root.dispatchEvent(new CustomEvent('admin-visibility',{detail:value}))}
function pendingReads(){const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}>=[];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
beforeEach(()=>{errors.mockClear();registerController('admin-announcements',AdminAnnouncementsController).handleError=errors;fetchMock=vi.fn<typeof fetch>(async()=>Response.json({}));vi.stubGlobal('fetch',fetchMock);setDateTimeZone('eve')})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals();vi.unstubAllEnvs();vi.restoreAllMocks()})

for(const [kind,prefix,rows,responseKey] of [['announcements','announcement','[data-admin-announcement-list]','announcement'],['blog','blog','[data-admin-blog-list]','post']] as const){
  test(kind+' validates empty titles, preserves a newer draft during create and updates its saved ID',async()=>{
    const root=mount(kind);await settle();q<HTMLButtonElement>(root,`[data-admin-${prefix}-new]`).click();const dialog=q<HTMLDialogElement>(root,'dialog');expect(dialog.open).toBe(true)
    q<HTMLButtonElement>(root,`[data-admin-${prefix}-save]`).click();expect(fetchMock).not.toHaveBeenCalled();expect(q(root,`[data-admin-${prefix}-form-message]`).textContent).toContain('Title is required')
    input(root,'[name="title"]','Original draft');const pending=pendingReads();q<HTMLButtonElement>(root,`[data-admin-${prefix}-save]`).click();q<HTMLButtonElement>(root,`[data-admin-${prefix}-save]`).click();expect(pending).toHaveLength(1)
    input(root,'[name="title"]','Newer draft');pending[0].complete(Response.json({[responseKey]:{id:123}}));await settle();pending[1].complete(new Response(result(kind,rows)));await settle()
    expect(dialog.open).toBe(true);expect(q<HTMLInputElement>(root,'[name="title"]').value).toBe('Newer draft');expect(q(root,`[data-admin-${prefix}-save-label]`).textContent).toBe('Update')
    q<HTMLButtonElement>(root,`[data-admin-${prefix}-save]`).click();expect(pending[2].path).toBe(`/api/admin/${kind}/123`);expect(fetchMock.mock.calls[2][1]!.method).toBe('PATCH');expect(JSON.parse(String(fetchMock.mock.calls[2][1]!.body)).title).toBe('Newer draft')
  })
  test(kind+' reports save failures, keeps the form, and cancels late responses on leaving its tab',async()=>{
    const root=mount(kind);await settle();q<HTMLButtonElement>(root,`[data-admin-${prefix}-new]`).click();input(root,'[name="title"]','Retained editor')
    fetchMock.mockResolvedValueOnce(Response.json({message:'Fixture save unavailable'},{status:503}));q<HTMLButtonElement>(root,`[data-admin-${prefix}-save]`).click();await settle()
    expect(q<HTMLDialogElement>(root,'dialog').open).toBe(true);expect(q(root,`[data-admin-${prefix}-form-message]`).textContent).toBe('Fixture save unavailable');expect(q<HTMLButtonElement>(root,`[data-admin-${prefix}-save]`).disabled).toBe(false)
    const pending=pendingReads();q<HTMLButtonElement>(root,`[data-admin-${prefix}-save]`).click();visibility(root,false);expect(pending[0].signal.aborted).toBe(true);expect(q<HTMLDialogElement>(root,'dialog').open).toBe(false)
    visibility(root,true);expect(pending).toHaveLength(2);pending[1].complete(new Response(result(kind,rows)));await settle();pending[0].complete(Response.json({message:'Late failure'},{status:503}));await settle()
    expect(root.textContent).not.toContain('Late failure');expect(fetchMock.mock.calls.filter(([,options])=>options?.method==='POST')).toHaveLength(2)
  })
  test(kind+' restores SSR rows and removes duplicate generated select controls on reconnect',async()=>{
    const root=mount(kind);await settle();const count=root.querySelectorAll('[data-select-menu-target="trigger"]').length;expect(count).toBe(1)
    root.remove();await settle();document.body.append(root);await settle();expect(root.querySelectorAll('[data-select-menu-target="trigger"]')).toHaveLength(count)
    q<HTMLButtonElement>(root,`[data-admin-${prefix}-edit]`).click();expect(q<HTMLInputElement>(root,'[name="title"]').value).not.toBe('');expect(fetchMock).not.toHaveBeenCalled()
  })
  test(kind+' disarms two-click removal on timeout and tab exit without sending a mutation',async()=>{
    const root=mount(kind);await settle();vi.useFakeTimers();const action=kind==='blog'?'delete':'archive',button=q<HTMLButtonElement>(root,`[data-admin-${prefix}-${action}]`)
    button.click();expect(button.getAttribute('aria-pressed')).toBe('true');expect(fetchMock).not.toHaveBeenCalled();await vi.advanceTimersByTimeAsync(5001);expect(button.getAttribute('aria-pressed')).toBe('false')
    button.click();visibility(root,false);visibility(root,true);expect(button.getAttribute('aria-pressed')).toBe('false');expect(fetchMock).not.toHaveBeenCalled()
  })
}
test('date pickers keep UTC values across zones, handle year rollover and bubble selection changes',async()=>{
  vi.stubEnv('TZ','Europe/Copenhagen');const root=mount('blog');await settle();q<HTMLButtonElement>(root,'[data-admin-blog-new]').click();const picker=q(root,'[data-date-time-picker]'),value=q<HTMLInputElement>(picker,'[data-date-time-value]');value.value='2026-12-31T23:45';value.dispatchEvent(new Event('change',{bubbles:true}))
  q<HTMLButtonElement>(picker,'[data-date-time-trigger]').click();expect(picker.querySelectorAll('[data-date-time-day]')).toHaveLength(42);expect(q(picker,'[data-date-time-month-label]').textContent).toBe('December 2026')
  q<HTMLButtonElement>(picker,'[data-date-time-month="1"]').click();expect(q(picker,'[data-date-time-month-label]').textContent).toBe('January 2027');q<HTMLButtonElement>(picker,'[data-date-time-month="-1"]').click();expect(q(picker,'[data-date-time-month-label]').textContent).toBe('December 2026')
  setDateTimeZone('local');expect(value.value).toBe('2026-12-31T23:45');expect(q(picker,'[data-date-time-zone-label]').textContent).toBe('Local');expect(q(picker,'[data-date-time-label]').textContent).toBe('2027-01-01 00:45');setDateTimeZone('eve')
  const change=vi.fn();picker.addEventListener('change',change);q<HTMLButtonElement>(picker,'[data-date-time-day][aria-label="2026-12-25"]').click();expect(value.value).toBe('2026-12-25T23:45');expect(change).toHaveBeenCalledOnce()
  const hour=q<HTMLSelectElement>(picker,'[data-date-time-hour]');hour.value='12';hour.dispatchEvent(new Event('change',{bubbles:true}));expect(value.value).toBe('2026-12-25T12:45')
  q<HTMLButtonElement>(picker,'[data-date-time-clear]').click();expect(value.value).toBe('');expect(q<HTMLElement>(picker,'[data-date-time-panel]').hidden).toBe(true)
})
test('dialog Escape captures non-bubbling cancel and closes its date popup before reopening',async()=>{
  const root=mount('blog');await settle();q<HTMLButtonElement>(root,'[data-admin-blog-new]').click();const picker=q(root,'[data-date-time-picker]');q<HTMLButtonElement>(picker,'[data-date-time-trigger]').click();expect(q<HTMLElement>(picker,'[data-date-time-panel]').hidden).toBe(false)
  q<HTMLDialogElement>(root,'dialog').dispatchEvent(new Event('cancel',{bubbles:false,cancelable:true}));expect(q<HTMLElement>(picker,'[data-date-time-panel]').hidden).toBe(true);expect(q<HTMLDialogElement>(root,'dialog').open).toBe(false)
  q<HTMLButtonElement>(root,'[data-admin-blog-new]').click();expect(q<HTMLElement>(picker,'[data-date-time-panel]').hidden).toBe(true)
})
test('a date calendar near a scrolling modal footer opens above its trigger and caps its own height',async()=>{
  const root=mount('blog');await settle();q<HTMLButtonElement>(root,'[data-admin-blog-new]').click();const picker=q(root,'[data-date-time-picker]'),panel=q(picker,'[data-date-time-panel]'),scroller=picker.parentElement!
  scroller.style.overflowY='auto';vi.spyOn(scroller,'getBoundingClientRect').mockReturnValue(new DOMRect(16,100,358,580));vi.spyOn(picker,'getBoundingClientRect').mockReturnValue(new DOMRect(37,400,316,40));vi.spyOn(panel,'getBoundingClientRect').mockReturnValue(new DOMRect(37,440,304,428))
  q<HTMLButtonElement>(picker,'[data-date-time-trigger]').click();expect(panel.classList.contains('bottom-full')).toBe(true);expect(panel.style.maxHeight).toBe('292px');expect(panel.style.overflowY).toBe('auto')
})
test('compatibility initialization is idempotent and outside/Escape handlers stop on disconnect',async()=>{
  const template=document.createElement('template');template.innerHTML=markup.blog;const picker=q(template.content,'[data-date-time-picker]');picker.removeAttribute('data-controller');picker.removeAttribute('data-action');document.body.append(picker);initializeDateTimePickers(document);initializeDateTimePickers(document);await settle()
  expect(picker.dataset.controller).toBe('date-time-picker');q<HTMLButtonElement>(picker,'[data-date-time-trigger]').click();document.body.dispatchEvent(new PointerEvent('pointerdown',{bubbles:true}));expect(q<HTMLElement>(picker,'[data-date-time-panel]').hidden).toBe(true)
  q<HTMLButtonElement>(picker,'[data-date-time-trigger]').click();document.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape'}));expect(q<HTMLElement>(picker,'[data-date-time-panel]').hidden).toBe(true);picker.remove();await settle();setDateTimeZone('local');expect(q(picker,'[data-date-time-zone-label]').textContent).toBe('EVE')
})
test('campaign search debounces, aborts superseded reads and retains only the current response',async()=>{
  const root=mount('campaigns');await settle();const pending=pendingReads();vi.useFakeTimers();input(root,'[data-admin-campaign-search]','Older');await vi.advanceTimersByTimeAsync(251);input(root,'[data-admin-campaign-search]','Newer');expect(pending[0].signal.aborted).toBe(true);await vi.advanceTimersByTimeAsync(251)
  expect(new URL(pending[1].path,location.origin).searchParams.get('q')).toBe('Newer');pending[1].complete(new Response(result('campaigns','[data-admin-campaign-result]')));await vi.advanceTimersByTimeAsync(0);pending[0].complete(new Response('<div data-admin-campaign-result>Stale campaign</div>'));await vi.advanceTimersByTimeAsync(0);expect(root.textContent).not.toContain('Stale campaign');expect(q(root,'[data-admin-campaign-results]').hasAttribute('aria-busy')).toBe(false)
})
test('campaign actions reject invalid values, prevent duplicate requests and reconcile interrupted work',async()=>{
  const root=mount('campaigns');await settle();const pending=pendingReads(),button=q<HTMLButtonElement>(root,'[data-admin-campaign-action="pause"]');button.click();button.click();expect(pending).toHaveLength(1);expect(JSON.parse(String(fetchMock.mock.calls[0][1]!.body))).toEqual({action:'pause'})
  visibility(root,false);expect(pending[0].signal.aborted).toBe(true);visibility(root,true);expect(pending).toHaveLength(2);pending[1].complete(new Response(result('campaigns','[data-admin-campaign-result]')));await settle();pending[0].complete(Response.json({message:'Late mutation failure'},{status:503}));await settle();expect(q(root,'[data-admin-campaign-action-error]').hidden).toBe(true)
  const invalid=q<HTMLButtonElement>(root,'[data-admin-campaign-action]');invalid.dataset.adminCampaignAction='unknown';invalid.click();expect(pending).toHaveLength(2)
})
test('domain directory filters and searches retain their state through disconnect without listeners accumulating',async()=>{
  const root=mount('domains');await settle();q<HTMLButtonElement>(root,'[data-admin-domain-filter="inactive"]').click();const visible=()=>Array.from(root.querySelectorAll<HTMLElement>('[data-admin-domain-row]')).filter(row=>!row.hidden);expect(visible().length).toBeGreaterThan(0)
  input(root,'[data-admin-domain-search]','No domain matches');expect(visible()).toHaveLength(0);expect(q(root,'[data-admin-domain-empty]').hidden).toBe(false);root.remove();await settle();document.body.append(root);await settle();expect(q(root,'[data-admin-domain-filter="inactive"]').getAttribute('aria-pressed')).toBe('true');expect(visible()).toHaveLength(0);expect(fetchMock).not.toHaveBeenCalled()
})
test('domain review keeps media and newer rejection notes through a refresh',async()=>{
  const root=mount('domain');await settle();q<HTMLButtonElement>(root,'[data-admin-domain-tab="files"]').click();const image=q<HTMLImageElement>(root,'[data-admin-domain-panel="files"] img');expect(image.hasAttribute('data-src')).toBe(false)
  q<HTMLButtonElement>(root,'[data-admin-domain-tab="approvals"]').click();input(root,'[data-admin-domain-reason]','Submitted reason');const pending=pendingReads(),button=q<HTMLButtonElement>(root,'[data-admin-domain-review="reject"]');button.click();button.click();expect(pending).toHaveLength(1);input(root,'[data-admin-domain-reason]','Newer reason');expect(JSON.parse(String(fetchMock.mock.calls[0][1]!.body))).toEqual({action:'reject',reason:'Submitted reason'})
  pending[0].complete(Response.json({success:true}));await settle();pending[1].complete(new Response(markup.domain));await settle();expect(q<HTMLInputElement>(root,'[data-admin-domain-reason]').value).toBe('Newer reason');expect(q<HTMLImageElement>(root,'[data-admin-domain-panel="files"] img')).toBe(image)
})
test('domain refresh rejects another domain and offers retry after permission failure',async()=>{
  const root=mount('domain');await settle();fetchMock.mockResolvedValueOnce(Response.json({},{status:403}));q<HTMLButtonElement>(root,'[data-admin-domain-toggle]').click();await settle();expect(q<HTMLButtonElement>(root,'[data-admin-domain-toggle]').disabled).toBe(false)
  fetchMock.mockResolvedValueOnce(Response.json({success:true})).mockResolvedValueOnce(new Response(markup.domain.replace(/data-domain-id="\d+"/,'data-domain-id="999"')));q<HTMLButtonElement>(root,'[data-admin-domain-toggle]').click();await settle();await settle();expect(q(root,'[data-admin-domain-refresh-error]').textContent).toContain('Could not refresh this domain')
  fetchMock.mockResolvedValueOnce(new Response(markup.domain));q<HTMLButtonElement>(root,'[data-admin-domain-refresh]').click();await settle();expect(root.querySelector('[data-admin-domain-refresh-error]')).toBeNull()
})
test('River inspect renders structured arguments safely, confirmations time out and hidden panes cancel both reads',async()=>{
  const root=mount('river');await settle();q<HTMLButtonElement>(root,'[data-river-inspect]').click();expect(q<HTMLDialogElement>(root,'dialog').open).toBe(true);expect(q(root,'[data-river-field="args"]').textContent).toContain('{');expect(q(root,'[data-river-field="title"]').textContent).toContain('Job ')
  vi.useFakeTimers();const clear=q<HTMLButtonElement>(root,'[data-river-queue-action="clear"]');clear.click();expect(clear.textContent).toBe('Confirm clear');await vi.advanceTimersByTimeAsync(5001);expect(clear.textContent).toBe('Clear history');expect(fetchMock).not.toHaveBeenCalled()
  const pending=pendingReads();q<HTMLButtonElement>(root,'[data-river-refresh]').click();expect(pending).toHaveLength(2);visibility(root,false);expect(pending.every(read=>read.signal.aborted)).toBe(true);expect(q<HTMLDialogElement>(root,'dialog').open).toBe(false);visibility(root,true);expect(pending).toHaveLength(4)
  pending[2].complete(new Response(result('river','[data-river-result="queues"]')));pending[3].complete(new Response(result('river','[data-river-result="jobs"]')));await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-river-spinner] svg').classList.contains('animate-spin')).toBe(false)
})
test('River filter clears cursor and queue actions use their generated contract once',async()=>{
  const root=mount('river');await settle();const pending=pendingReads(),select=q<HTMLSelectElement>(root,'[data-river-filter="state"]');select.value='running';select.dispatchEvent(new Event('change',{bubbles:true}));expect(new URL(pending[0].path,location.origin).searchParams.get('state')).toBe('running')
  const button=q<HTMLButtonElement>(root,'[data-river-queue-action="pause"]');button.click();button.click();expect(pending).toHaveLength(2);expect(JSON.parse(String(fetchMock.mock.calls[1][1]!.body))).toEqual({action:'pause'});pending[1].complete(Response.json({message:'Fixture queue unavailable'},{status:503}));await settle();expect(button.disabled).toBe(false)
})
test('wallet paging keeps focus and replaces the division control once; authorization errors restore controls',async()=>{
  const root=mount('wallet');await settle();fetchMock.mockResolvedValueOnce(new Response(result('wallet','[data-wallet-result]')));const refresh=q<HTMLButtonElement>(root,'[data-wallet-refresh]');refresh.focus();refresh.click();await settle();expect(document.activeElement).toBe(q(root,'[data-wallet-refresh]'));expect(root.querySelectorAll('[data-select-menu-target="trigger"]')).toHaveLength(1)
  fetchMock.mockResolvedValueOnce(Response.json({message:'Fixture authorization unavailable'},{status:503}));q<HTMLButtonElement>(root,'[data-wallet-authorize]').click();await settle();expect(q(root,'[data-wallet-action-error]').textContent).toBe('Fixture authorization unavailable');expect(q<HTMLButtonElement>(root,'[data-wallet-authorize]').disabled).toBe(false)
  root.remove();await settle();document.body.append(root);await settle();expect(root.querySelectorAll('[data-select-menu-target="trigger"]')).toHaveLength(1)
})
test('wallet sync delay is canceled on leaving its pane and resumes with a read only',async()=>{
  const root=mount('wallet');await settle();vi.useFakeTimers();fetchMock.mockResolvedValueOnce(Response.json({queued:true}));q<HTMLButtonElement>(root,'[data-wallet-sync]').click();await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-wallet-message]').textContent).toContain('sync queued')
  visibility(root,false);await vi.advanceTimersByTimeAsync(3000);expect(fetchMock).toHaveBeenCalledOnce();fetchMock.mockResolvedValueOnce(new Response(result('wallet','[data-wallet-result]')));visibility(root,true);await vi.advanceTimersByTimeAsync(0);expect(fetchMock.mock.calls[1][1]!.method).toBeUndefined();expect(fetchMock.mock.calls.filter(([,options])=>options?.method==='POST')).toHaveLength(1)
})
test('wallet settlement cancellation sends nothing and a confirmed mock payment cannot be duplicated',async()=>{
  const root=mount('wallet'),resultNode=q(root,'[data-wallet-result]'),settlements=readWalletSettlements(resultNode.dataset.initial);settlements[0].claimed_at='2026-10-07T12:00:00Z';settlements[0].paid_at=null;resultNode.dataset.initial=JSON.stringify({prizeSettlements:settlements})
  // The retained fixture has no claimed winner. Model that read-only state here;
  // all settlement actions in this test are intercepted by the fetch mock.
  const button=document.createElement('button');button.type='button';button.dataset.walletPaid='';q(root,'[data-admin-wallet-prize]').append(button)
  await settle();const prompt=vi.fn().mockReturnValueOnce(null).mockReturnValueOnce('Fixture note');vi.stubGlobal('prompt',prompt);button.click();expect(fetchMock).not.toHaveBeenCalled();const pending=pendingReads();button.click();button.click();expect(prompt).toHaveBeenCalledTimes(2);expect(pending).toHaveLength(1);expect(JSON.parse(String(fetchMock.mock.calls[0][1]!.body))).toEqual({note:'Fixture note'})
  visibility(root,false);expect(pending[0].signal.aborted).toBe(true);visibility(root,true);pending[1].complete(new Response(result('wallet','[data-wallet-result]')));await settle();pending[0].complete(Response.json({message:'Late financial failure'},{status:503}));await settle();expect(q(root,'[data-wallet-action-error]').hidden).toBe(true)
})
test('admin seed boundaries discard malformed collections and keep structured River values as data',()=>{
  expect(readAnnouncementSeed('broken')).toEqual([]);expect(readBlogSeed('{"posts":[null,{"id":"bad"}]}')).toEqual([]);expect(readRiverJobs('{"jobs":[{"id":-1},{"id":1,"args":{"text":"<script>plain text</script>"}}]}')[0].args).toEqual({text:'<script>plain text</script>'});expect(readWalletSettlements('{"prizeSettlements":[{},null]}')).toEqual([])
})
