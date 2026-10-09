// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {LegacyController} from '../src/browser/legacy'
import {registerController} from '../src/browser/controllers/application'
import {readLegacySuggestions,legacyYear} from '../src/browser/shared/legacy'

const directory = resolve(import.meta.dirname,'fixtures/legacy')
const markup = readFileSync(resolve(directory,'page.html'),'utf8')
const kills = readFileSync(resolve(directory,'kills.html'),'utf8')
const top = readFileSync(resolve(directory,'top.html'),'utf8')
const errors = vi.fn(),settle = () => new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing legacy fixture '+selector);return node}
async function mount() {document.body.innerHTML = markup;await settle();await settle();return q<HTMLElement>(document.body,'[data-legacy-page]')}
function input(root:HTMLElement,name:string,value:string) {const field = q<HTMLInputElement>(root,`[name="${name}"]`);field.value = value;field.dispatchEvent(new InputEvent('input',{bubbles:true}));return field}
function submit(root:HTMLElement) {q<HTMLFormElement>(root,'form').dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}))}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{
  registerController('legacy',LegacyController).handleError = errors;errors.mockClear();history.replaceState(null,'','/legacy')
  fetchMock = vi.fn<typeof fetch>(async path=>String(path).includes('/autocomplete') ? Response.json([{name:'Fixture Pilot'}]) : new Response(String(path).includes('/top') ? top : kills))
  vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('legacy filters retain SSR rows during owned reads, trim values and clear the form',async()=>{
  const root = await mount(),row = q(root,'.kill-row'),pending = pendingRequests()
  input(root,'victim',' Fixture Pilot ');input(root,'ship','Capsule');submit(root)
  expect(pending[0].path).toContain('victim=Fixture+Pilot&ship=Capsule&limit=50&page=1')
  expect(q(root,'.kill-row')).toBe(row);expect(q(root,'[data-legacy-kills]').getAttribute('aria-busy')).toBe('true')
  pending[0].complete(new Response(kills));await settle();expect(q(root,'.kill-row')).not.toBe(row)
  expect(q<HTMLElement>(root,'[data-legacy-clear]').hidden).toBe(false)
  q<HTMLButtonElement>(root,'[data-legacy-clear]').click();expect(pending[1].path).toBe('/_fragments/legacy/kills?limit=50&page=1')
  expect(q<HTMLInputElement>(root,'[name="victim"]').value).toBe('');hide()
})
test('presets preserve ship lists, system names and historical date windows',async()=>{
  const root = await mount(),pending = pendingRequests()
  q<HTMLButtonElement>(root,'[data-legacy-preset="capitals"]').click();expect(q<HTMLInputElement>(root,'[name="ship"]').value).toContain('Avatar,Erebus')
  q<HTMLButtonElement>(root,'[data-legacy-preset="pods"]').click();expect(pending[0].signal.aborted).toBe(true);expect(q<HTMLInputElement>(root,'[name="ship"]').value).toBe('Capsule')
  q<HTMLButtonElement>(root,'[data-legacy-preset="Jita"]').click();expect(q<HTMLInputElement>(root,'[name="system"]').value).toBe('Jita')
  q<HTMLButtonElement>(root,'[data-legacy-preset="2005"]').click();expect(pending[3].path).toContain('from=2005-01-01&to=2005-12-31');hide()
})
test('paging uses current cursors and discards old pagination after sort/limit changes',async()=>{
  const root = await mount(),pending = pendingRequests()
  q<HTMLButtonElement>(root,'[data-kill-page="2"]').click();expect(pending[0].path).toContain('page=2&after=123')
  pending[0].complete(new Response(kills));await settle()
  q<HTMLButtonElement>(root,'[data-legacy-sort="value_desc"]').click();expect(pending[1].path).toContain('page=1&sort=value_desc');expect(pending[1].path).not.toContain('after=')
  q<HTMLButtonElement>(root,'[data-kill-page="2"]').click();expect(pending).toHaveLength(2)
  q<HTMLButtonElement>(root,'[data-kill-limit="100"]').click();expect(pending[1].signal.aborted).toBe(true);expect(pending[2].path).toContain('limit=100&page=1&sort=value_desc')
  expect(q(root,'[data-legacy-sort="value_desc"]').getAttribute('aria-pressed')).toBe('true');hide()
})
test('new filters supersede bodies without allowing late responses to overwrite current rows',async()=>{
  const root = await mount(),pending = pendingRequests();input(root,'victim','First');submit(root);input(root,'victim','Latest');submit(root)
  expect(pending[0].signal.aborted).toBe(true);pending[1].complete(new Response(kills.replaceAll('Apocalypse','Current ship')));await settle()
  pending[0].complete(new Response(kills.replaceAll('Apocalypse','Stale ship')));await settle()
  expect(q(root,'[data-legacy-kills]').textContent).toContain('Current ship');expect(root.textContent).not.toContain('Stale ship');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('kill and top errors retain content, expose one retry and preserve requested filters/year',async()=>{
  const root = await mount(),pending = pendingRequests();input(root,'system','Jita');submit(root)
  pending[0].complete(new Response('Failure',{status:503}));await settle();expect(q(root,'.kill-row')).toBeTruthy()
  q<HTMLButtonElement>(root,'[data-legacy-retry="kills"]').click();expect(pending[1].path).toContain('system=Jita');pending[1].complete(new Response('<div>Wrong fragment</div>'));await settle()
  expect(root.querySelectorAll('[data-legacy-request-error="kills"]')).toHaveLength(1)
  q<HTMLButtonElement>(root,'[data-legacy-year="2005"]').click();pending[2].complete(new Response('Failure',{status:503}));await settle()
  expect(q(root,'[data-legacy-sidebar] .legacy-year-picker')).toBeTruthy();q<HTMLButtonElement>(root,'[data-legacy-retry="top"]').click()
  expect(pending[3].path).toBe('/_fragments/legacy/top?year=2005');hide()
})
test('year top reads are independent of kills and accept only the most recent body',async()=>{
  const root = await mount(),pending = pendingRequests();submit(root);q<HTMLButtonElement>(root,'[data-legacy-year="2003"]').click();q<HTMLButtonElement>(root,'[data-legacy-year="2007"]').click()
  expect(pending[0].signal.aborted).toBe(false);expect(pending[1].signal.aborted).toBe(true)
  pending[2].complete(new Response(top+'<p>Current top</p>'));await settle();pending[1].complete(new Response(top+'<p>Stale top</p>'));await settle()
  expect(q(root,'[data-legacy-sidebar]').textContent).toContain('Current top');expect(root.textContent).not.toContain('Stale top');expect(q(root,'[data-legacy-kills]').getAttribute('aria-busy')).toBe('true');hide()
})
test('late fragment text cannot paint after a component is detached',async()=>{
  const root = await mount();let complete:(text:string)=>void = ()=>{};const response = new Response('')
  vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>complete = resolve));fetchMock.mockResolvedValue(response)
  submit(root);await settle();root.remove();await settle();complete(kills.replaceAll('Apocalypse','Late ship'));await settle()
  expect(root.textContent).not.toContain('Late ship');expect(root.hasAttribute('aria-busy')).toBe(false);expect(q(root,'[data-legacy-kills]').classList.contains('legacy-pending')).toBe(false)
})
test('page return resumes interrupted reads without resetting selected filters',async()=>{
  const root = await mount(),pending = pendingRequests();input(root,'ship','Capsule');submit(root);q<HTMLButtonElement>(root,'[data-legacy-year="2003"]').click();hide();show()
  expect(pending[0].signal.aborted).toBe(true);expect(pending[1].signal.aborted).toBe(true)
  expect(pending[2].path).toBe(pending[0].path);expect(pending[3].path).toBe(pending[1].path)
  pending[2].complete(new Response(kills));pending[3].complete(new Response(top));await settle();expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('mobile top/kill tabs preserve the query and follow browser history',async()=>{
  history.replaceState(null,'','/legacy?review=1#top');const root = await mount()
  expect(root.classList.contains('legacy-mobile-top')).toBe(true);q<HTMLButtonElement>(root,'[data-legacy-mobile="kills"]').click()
  expect(location.pathname+location.search+location.hash).toBe('/legacy?review=1');expect(root.classList.contains('legacy-mobile-top')).toBe(false)
  history.replaceState(null,'','/legacy?review=1#top');window.dispatchEvent(new PopStateEvent('popstate'))
  expect(q(root,'[data-legacy-mobile="top"]').getAttribute('aria-pressed')).toBe('true')
})
test('autocomplete is debounced, scoped and supports pointer selection without losing focus',async()=>{
  const root = await mount();vi.useFakeTimers();const field = input(root,'victim','Fi');field.focus();input(root,'victim','Fix');await vi.advanceTimersByTimeAsync(199);expect(fetchMock).not.toHaveBeenCalled()
  await vi.advanceTimersByTimeAsync(2);expect(fetchMock).toHaveBeenCalledWith('/api/legacy/autocomplete?q=Fix&field=victim',expect.anything())
  const button = q<HTMLButtonElement>(root,'[data-legacy-suggestion]'),event = new MouseEvent('mousedown',{bubbles:true,cancelable:true});button.dispatchEvent(event)
  expect(event.defaultPrevented).toBe(true);expect(field.value).toBe('Fixture Pilot');expect(q<HTMLElement>(root,'[data-legacy-options="victim"]').hidden).toBe(true);expect(document.activeElement).toBe(field)
})
test('autocomplete rejects stale text and old body completion after selecting another field',async()=>{
  const root = await mount(),pending = pendingRequests();vi.useFakeTimers();const first = input(root,'victim','First');first.focus();await vi.advanceTimersByTimeAsync(201)
  const second = input(root,'attacker','Latest');second.focus();await vi.advanceTimersByTimeAsync(201);expect(pending[0].signal.aborted).toBe(true)
  pending[1].complete(Response.json([{name:'Latest Pilot'}]));await vi.advanceTimersByTimeAsync(0);pending[0].complete(Response.json([{name:'Stale Pilot'}]));await vi.advanceTimersByTimeAsync(0)
  expect(q(root,'[data-legacy-options="attacker"]').textContent).toBe('Latest Pilot');expect(root.textContent).not.toContain('Stale Pilot')
})
test('autocomplete failure remains actionable and pagehide cancels debounce/late work',async()=>{
  const root = await mount();vi.useFakeTimers();fetchMock.mockResolvedValue(new Response('Failure',{status:503}));const field = input(root,'victim','Fixture');field.focus();await vi.advanceTimersByTimeAsync(201)
  expect(q(root,'[data-legacy-options="victim"]').textContent).toContain('You can still search')
  fetchMock.mockClear();input(root,'victim','Pending');hide();await vi.advanceTimersByTimeAsync(201);expect(fetchMock).not.toHaveBeenCalled()
  expect(q<HTMLElement>(root,'[data-legacy-options="victim"]').hidden).toBe(true)
})
test('legacy boundaries ignore malformed suggestions and reject invalid year values',()=>{
  expect(readLegacySuggestions([null,{name:'Pilot'},{name:42},{name:' '}])).toEqual(['Pilot'])
  expect(()=>readLegacySuggestions({hits:[]})).toThrow('Suggestions are unavailable')
  expect(legacyYear('2003')).toBe(2003);expect(legacyYear('0')).toBe(0);expect(legacyYear('2008')).toBeUndefined();expect(legacyYear('../2003')).toBeUndefined()
})
