// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {StatisticsController} from '../src/browser/statistics'
import {registerController} from '../src/browser/controllers/application'

const directory = resolve(import.meta.dirname,'fixtures/statistics')
const cold = readFileSync(resolve(directory,'cold.html'),'utf8'),warm = readFileSync(resolve(directory,'warm.html'),'utf8'),period = readFileSync(resolve(directory,'period.html'),'utf8'),population = readFileSync(resolve(directory,'population.html'),'utf8')
const errors = vi.fn(),settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing statistics fixture '+selector);return node}
function fragment(path:string,label = 'Fixture loaded statistic') {
  const url = new URL(path,location.href),key = decodeURIComponent(url.pathname.split('/').at(-1)!),slot = url.searchParams.get('period') ?? 'population',raw = slot === 'population' ? population : period
  return raw.replace(/data-stats-card-loaded="[^"]+"/g,'data-stats-card-loaded="'+key+'"').replace(/data-stats-card-period="[^"]+"/g,'data-stats-card-period="'+slot+'"').replace(/Pilot[^<]*&lt;script&gt;[\s\S]*?&lt;\/script&gt;/g,label)
}
async function mount(kind:'cold'|'warm' = 'cold',section = 'all') {history.replaceState({fixture:'retained'},'','/stats'+(section === 'all' ? '' : '?section='+section));document.body.innerHTML = kind === 'warm' ? warm : cold;const root = q<HTMLElement>(document.body,'[data-statistics]');root.dataset.statsFilter = section;await settle();await settle();return root}
function filter(root:HTMLElement,section:string) {q<HTMLButtonElement>(root,`[data-stats-filter-button="${section}"]`).click()}
function choosePeriod(root:HTMLElement,value:string) {q<HTMLButtonElement>(root,`[data-stats-period-button="${value}"]`).click()}
function card(root:HTMLElement,key = 'characters') {return q<HTMLElement>(root,`[data-stats-card="${key}"]`)}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{registerController('statistics',StatisticsController).handleError = errors;errors.mockClear();fetchMock = vi.fn<typeof fetch>(async path=>new Response(fragment(String(path))));vi.stubGlobal('fetch',fetchMock)})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('warm SSR cards retain image/link nodes and need no duplicate startup queries',async()=>{
  let image:HTMLImageElement|undefined;document.body.innerHTML = warm;const root = q<HTMLElement>(document.body,'[data-statistics]');image = q<HTMLImageElement>(root,'[data-stats-card="characters"] img');await settle();await settle();expect(fetchMock).not.toHaveBeenCalled();expect(q(card(root),'img')).toBe(image);expect(root.querySelectorAll('[data-stats-card]').length).toBeGreaterThan(20)
})
test('cold startup limits active reads to four and pumps one slot when a card completes',async()=>{
  const pending = pendingRequests(),root = await mount();expect(pending).toHaveLength(4);pending[0].complete(new Response(fragment(pending[0].path)));await settle();expect(pending).toHaveLength(5);expect(card(root).dataset.statsLoaded).toBe('true');expect(q(card(root),'[data-stats-card-loaded]').getAttribute('data-stats-card-period')).toBe('7');hide()
})
test('filter changes cancel hidden reads/queued work and immediately release slots for visible cards',async()=>{
  const pending = pendingRequests(),root = await mount();filter(root,'largest');expect(pending.slice(0,4).every(request=>request.signal.aborted)).toBe(true);expect(pending).toHaveLength(6);expect(pending.slice(4).every(request=>request.path.includes('/largest-'))).toBe(true);expect(q<HTMLElement>(root,'[data-stats-period-controls]').hidden).toBe(true);expect(q<HTMLElement>(root,'[data-stats-jumps]').hidden).toBe(true)
  for(const request of pending.slice(4)) request.complete(new Response(fragment(request.path)));await settle();expect(card(root,'largest-alliance').dataset.statsLoaded).toBe('true');for(const request of pending.slice(0,4)) request.complete(new Response(fragment(request.path,'Stale hidden data')));await settle();expect(card(root).dataset.statsLoaded).toBe('false')
})
test('period changes supersede active bodies and accept only the current key/period',async()=>{
  const pending = pendingRequests(),root = await mount('cold','activity');expect(pending).toHaveLength(3);choosePeriod(root,'30');expect(pending.slice(0,3).every(request=>request.signal.aborted)).toBe(true);expect(pending).toHaveLength(6);expect(pending.slice(3).every(request=>request.path.endsWith('?period=30'))).toBe(true)
  for(const request of pending.slice(3)) request.complete(new Response(fragment(request.path)));await settle();for(const request of pending.slice(0,3)) request.complete(new Response(fragment(request.path)));await settle();expect(q(card(root),'[data-stats-card-loaded]').getAttribute('data-stats-card-period')).toBe('30');expect(q(root,'[data-stats-period-label]').textContent).toBe('30d')
})
test('cached period changes retain exact node identity, including images, when returning',async()=>{
  const root = await mount('warm','activity'),original = q(card(root),'[data-stats-card-loaded]'),image = q(card(root),'img');choosePeriod(root,'14');await settle();await settle();expect(q(card(root),'[data-stats-card-loaded]')).not.toBe(original);const count = fetchMock.mock.calls.length;choosePeriod(root,'7');await settle();expect(fetchMock).toHaveBeenCalledTimes(count);expect(q(card(root),'[data-stats-card-loaded]')).toBe(original);expect(q(card(root),'img')).toBe(image)
})
test('period changes do not invalidate population cards or issue their requests again',async()=>{
  const root = await mount('warm'),populationNode = q(card(root,'largest-alliance'),'[data-stats-card-loaded]');choosePeriod(root,'14');await settle();await settle();expect(fetchMock.mock.calls.every(([path])=>String(path).includes('?period=14'))).toBe(true);expect(q(card(root,'largest-alliance'),'[data-stats-card-loaded]')).toBe(populationNode)
})
test('expired cached data stays visible through refresh failure and one retry recovers it',async()=>{
  const root = await mount('warm','activity'),original = q(card(root),'[data-stats-card-loaded]');vi.useFakeTimers();await vi.advanceTimersByTimeAsync(60001);const pending = pendingRequests();filter(root,'activity');expect(pending).toHaveLength(3);expect(q(card(root),'[data-stats-card-loaded]')).toBe(original);pending[0].complete(new Response('Failed',{status:503}));await vi.advanceTimersByTimeAsync(0);expect(q(card(root),'[data-stats-card-loaded]')).toBe(original)
  const retry = q<HTMLButtonElement>(card(root),'[data-stats-retry]');retry.click();retry.click();expect(pending).toHaveLength(4);pending[3].complete(new Response(fragment(pending[3].path)));await vi.advanceTimersByTimeAsync(0);expect(card(root).querySelector('[data-stats-retry]')).toBeNull()
})
test('short periods use thirty-second freshness while population cards use sixty seconds',async()=>{
  const root = await mount('warm','activity');vi.useFakeTimers();choosePeriod(root,'0.041666666666666664');await vi.advanceTimersByTimeAsync(0);const count = fetchMock.mock.calls.length;await vi.advanceTimersByTimeAsync(29999);filter(root,'activity');await vi.advanceTimersByTimeAsync(0);expect(fetchMock).toHaveBeenCalledTimes(count);await vi.advanceTimersByTimeAsync(2);filter(root,'activity');await vi.advanceTimersByTimeAsync(0);expect(fetchMock).toHaveBeenCalledTimes(count+3)
  filter(root,'largest');await vi.advanceTimersByTimeAsync(0);expect(fetchMock).toHaveBeenCalledTimes(count+3)
})
test('wrong loaded keys/periods fail visibly instead of replacing a different card',async()=>{
  const pending = pendingRequests(),root = await mount('cold','activity');pending[0].complete(new Response(fragment(pending[0].path).replace('data-stats-card-period="7"','data-stats-card-period="30"')));await settle();expect(card(root).textContent).toContain('Could not load this table');expect(card(root).dataset.statsLoaded).toBe('false');q<HTMLButtonElement>(card(root),'[data-stats-retry]').click();pending[3].complete(new Response(fragment(pending[3].path)));await settle();expect(card(root).dataset.statsLoaded).toBe('true')
})
test('Back restores validated section/period controls, defaulting invalid links to all/seven days',async()=>{
  const root = await mount('warm');history.replaceState(history.state,'','/stats?section=largest&period=30&review=1');window.dispatchEvent(new PopStateEvent('popstate'));expect(root.dataset.statsFilter).toBe('largest');expect(root.dataset.statsPeriod).toBe('30');expect(q<HTMLElement>(root,'[data-stats-period-controls]').hidden).toBe(true)
  history.replaceState(history.state,'','/stats?section=invalid&period=invalid&review=1');window.dispatchEvent(new PopStateEvent('popstate'));expect(root.dataset.statsFilter).toBe('all');expect(root.dataset.statsPeriod).toBe('7');expect(fetchMock).not.toHaveBeenCalled()
})
test('filter/period links preserve unrelated query/state and respect reduced-motion jumps',async()=>{
  const root = await mount('warm');history.replaceState({fixture:'retained'},'','/stats?review=1');filter(root,'activity');expect(location.search).toContain('section=activity');expect(location.search).toContain('review=1');expect(history.state).toEqual({fixture:'retained'});filter(root,'all');expect(location.search).not.toContain('section=')
  const section = q<HTMLElement>(root,'#stats-activity'),scroll = vi.spyOn(section,'scrollIntoView');vi.spyOn(window,'matchMedia').mockImplementation(()=>({matches:true,media:'',onchange:null,addListener(){},removeListener(){},addEventListener(){},removeEventListener(){},dispatchEvent(){return true}}));q<HTMLAnchorElement>(root,'[data-stats-jump="activity"]').click();expect(scroll).toHaveBeenCalledWith({behavior:'instant',block:'start'})
})
test('page return resumes interrupted visible cards without old reads consuming queue slots',async()=>{
  const pending = pendingRequests(),root = await mount('cold','activity');hide();show();expect(pending.slice(0,3).every(request=>request.signal.aborted)).toBe(true);expect(pending).toHaveLength(6);for(const request of pending.slice(3)) request.complete(new Response(fragment(request.path)));await settle();for(const request of pending.slice(0,3)) request.complete(new Response(fragment(request.path)));await settle();expect(card(root).dataset.statsLoaded).toBe('true');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('late text after detach cannot paint or retain busy flags on a cached page',async()=>{
  let complete:(text:string)=>void = ()=>{};const response = new Response('');vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>complete = resolve));fetchMock.mockResolvedValue(response);const root = await mount('cold','activity');root.remove();await settle();complete(period);await settle();expect(card(root).dataset.statsLoaded).toBe('false');expect(q(card(root),'[data-stats-card-content]').getAttribute('aria-busy')).toBe('false');expect(root.hasAttribute('aria-busy')).toBe(false)
})
