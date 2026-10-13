// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {RankingsController} from '../src/browser/rankings'
import {registerController} from '../src/browser/controllers/application'

const directory = resolve(import.meta.dirname,'fixtures/rankings')
const markup = readFileSync(resolve(directory,'page.html'),'utf8'),character = readFileSync(resolve(directory,'character.html'),'utf8'),corporation = readFileSync(resolve(directory,'corporation.html'),'utf8'),hourly = readFileSync(resolve(directory,'hourly.html'),'utf8')
const errors = vi.fn(),settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>,hidden = false
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing rankings fixture '+selector);return node}
async function mount(query = '') {history.replaceState({fixture:'retained'},'','/rankings'+query);document.body.innerHTML = markup;await settle();await settle();return q<HTMLElement>(document.body,'[data-rankings-page]')}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{
  registerController('rankings',RankingsController).handleError = errors;errors.mockClear();hidden = false;vi.spyOn(document,'hidden','get').mockImplementation(()=>hidden)
  fetchMock = vi.fn<typeof fetch>(async path=>{const query = new URL(String(path),location.href).searchParams,entity = query.get('entity') ?? 'character',period = query.get('window') ?? 'all_time';const fragment = entity === 'corporation' ? corporation : character;return new Response(fragment.replace(/data-ranking-result-entity="[^"]+"/g,'data-ranking-result-entity="'+entity+'"').replace(/data-ranking-result-window="[^"]+"/g,'data-ranking-result-window="'+period+'"'))});vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('rankings keeps the SSR podium/images and initializes deep-link controls without a read',async()=>{
  const root = await mount('?entity=ship&window=weekly');expect(fetchMock).not.toHaveBeenCalled();expect(q(root,'[data-ranking-results] img')).toBeTruthy();expect(q(root,'[data-ranking-entity="ship"]').getAttribute('aria-pressed')).toBe('true');expect(q(root,'[data-ranking-window="weekly"]').classList.contains('active')).toBe(true)
})
test('entity/window requests supersede bodies and retain images while loading',async()=>{
  const root = await mount(),image = q(root,'[data-ranking-results] img'),pending = pendingRequests();q<HTMLButtonElement>(root,'[data-ranking-entity="corporation"]').click();q<HTMLButtonElement>(root,'[data-ranking-window="weekly"]').click()
  expect(pending[0].signal.aborted).toBe(true);expect(q(root,'[data-ranking-results] img')).toBe(image);expect(pending[1].path).toBe('/_fragments/rankings?entity=corporation&window=weekly')
  pending[1].complete(new Response(corporation.replace('data-ranking-result-window="all_time"','data-ranking-result-window="weekly"')));await settle();pending[0].complete(new Response(corporation));await settle()
  expect(q(root,'.ranking-table').getAttribute('data-ranking-result-window')).toBe('weekly');expect(location.search).toBe('?entity=corporation&window=weekly');expect(history.state).toEqual({fixture:'retained'});expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('failed/wrong-selection fragments preserve rows, show one retry and retain requested filters',async()=>{
  const root = await mount(),table = q(root,'.ranking-table'),pending = pendingRequests();q<HTMLButtonElement>(root,'[data-ranking-entity="corporation"]').click();pending[0].complete(new Response('Failed',{status:503}));await settle();expect(q(root,'.ranking-table')).toBe(table)
  const retry = q<HTMLButtonElement>(root,'[data-ranking-retry]');retry.click();retry.click();expect(pending).toHaveLength(2);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(character));await settle();expect(root.querySelectorAll('[data-ranking-request-error]')).toHaveLength(1)
  q<HTMLButtonElement>(root,'[data-ranking-retry]').click();pending[2].complete(new Response(corporation));await settle();expect(root.querySelector('[data-ranking-request-error]')).toBeNull();expect(q(root,'.ranking-table').getAttribute('data-ranking-result-entity')).toBe('corporation')
})
test('Back uses validated URL filters and invalid values fall back without losing the query',async()=>{
  const root = await mount('?review=1'),pending = pendingRequests();history.replaceState(history.state,'','/rankings?review=1&entity=corporation&window=hourly');window.dispatchEvent(new PopStateEvent('popstate'));expect(pending[0].path).toContain('entity=corporation&window=hourly');expect(q(root,'[data-ranking-window="hourly"]').classList.contains('active')).toBe(true)
  history.replaceState(history.state,'','/rankings?review=1&entity=invalid&window=invalid');window.dispatchEvent(new PopStateEvent('popstate'));expect(pending[0].signal.aborted).toBe(true);expect(pending[1].path).toContain('entity=character&window=all_time');pending[1].complete(new Response(character));await settle();expect(location.search).toContain('review=1');expect(location.search).toContain('entity=invalid')
})
test('hourly refresh is silent, non-overlapping and stops when another period is selected',async()=>{
  const root = await mount();vi.useFakeTimers();q<HTMLButtonElement>(root,'[data-ranking-window="hourly"]').click();await vi.advanceTimersByTimeAsync(0);const pending = pendingRequests();await vi.advanceTimersByTimeAsync(60000)
  expect(pending).toHaveLength(1);expect(root.hasAttribute('data-loading-silent')).toBe(true);expect(q(root,'[data-ranking-results]').hasAttribute('data-loading-silent')).toBe(true);await vi.advanceTimersByTimeAsync(120000);expect(pending).toHaveLength(1)
  pending[0].complete(new Response(hourly));await vi.advanceTimersByTimeAsync(0);expect(root.hasAttribute('data-loading-silent')).toBe(false);q<HTMLButtonElement>(root,'[data-ranking-window="all_time"]').click();pending[1].complete(new Response(character));await vi.advanceTimersByTimeAsync(0);await vi.advanceTimersByTimeAsync(120000);expect(pending).toHaveLength(2)
})
test('visibility pauses polling and returning before its deadline does not repeat a fresh read',async()=>{
  const root = await mount();vi.useFakeTimers();q<HTMLButtonElement>(root,'[data-ranking-window="hourly"]').click();await vi.advanceTimersByTimeAsync(0);const count = fetchMock.mock.calls.length
  hidden = true;document.dispatchEvent(new Event('visibilitychange'));await vi.advanceTimersByTimeAsync(1000);hidden = false;document.dispatchEvent(new Event('visibilitychange'));await vi.advanceTimersByTimeAsync(0);expect(fetchMock).toHaveBeenCalledTimes(count)
  hidden = true;document.dispatchEvent(new Event('visibilitychange'));await vi.advanceTimersByTimeAsync(60001);expect(fetchMock).toHaveBeenCalledTimes(count);hidden = false;document.dispatchEvent(new Event('visibilitychange'));await vi.advanceTimersByTimeAsync(0);expect(fetchMock).toHaveBeenCalledTimes(count+1)
})
test('pagehide cancels pending reads and page return reconciles the desired selection with one read',async()=>{
  const root = await mount(),pending = pendingRequests();q<HTMLButtonElement>(root,'[data-ranking-entity="corporation"]').click();hide();show();expect(pending[0].signal.aborted).toBe(true);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(corporation));pending[0].complete(new Response(character));await settle();expect(q(root,'.ranking-table').getAttribute('data-ranking-result-entity')).toBe('corporation');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('late bodies after detach cannot paint or leave loading state on cached DOM',async()=>{
  const root = await mount();let complete:(text:string)=>void = ()=>{};const response = new Response('');vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>complete = resolve));fetchMock.mockResolvedValue(response);q<HTMLButtonElement>(root,'[data-ranking-entity="corporation"]').click();await settle();root.remove();await settle();complete(corporation);await settle();expect(q(root,'.ranking-table').getAttribute('data-ranking-result-entity')).toBe('character');expect(q(root,'[data-ranking-results]').classList.contains('rankings-loading')).toBe(false);expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('detaching an hourly page disposes its timer instead of leaving a hidden poller',async()=>{
  const root = await mount();vi.useFakeTimers();q<HTMLButtonElement>(root,'[data-ranking-window="hourly"]').click();await vi.advanceTimersByTimeAsync(0);const count = fetchMock.mock.calls.length;root.remove();await vi.advanceTimersByTimeAsync(0);await vi.advanceTimersByTimeAsync(120001);expect(fetchMock).toHaveBeenCalledTimes(count)
})
