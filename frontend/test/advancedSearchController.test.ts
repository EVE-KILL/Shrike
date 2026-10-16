// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {AdvancedSearchController} from '../src/browser/advanced-search'
import {registerController} from '../src/browser/controllers/application'
import {readBrowserSession} from '../src/browser/browser-request'
import {savedSearchDocument} from '../src/browser/saved-search-data'

vi.mock('../src/browser/browser-request',async original=>({...await original<typeof import('../src/browser/browser-request')>(),readBrowserSession:vi.fn()}))
const markup=readFileSync(resolve(import.meta.dirname,'fixtures/advanced-search.html'),'utf8'),settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn()
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(root:ParentNode,selector:string):T{const node=root.querySelector<T>(selector);if(!node)throw Error('Missing advanced-search fixture '+selector);return node}
async function mount(){document.body.innerHTML=markup;await settle();return q(document,'[data-advanced-search]')}
const result='<div data-advanced-result-content><div id="kill-list" data-page="1" data-limit="50" data-next-cursor="123">Fixture results</div></div>'
function pendingReads(){const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}>=[];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
beforeEach(()=>{registerController('advanced-search',AdvancedSearchController).handleError=errors;errors.mockClear();history.replaceState({},'','/advancedsearch');localStorage.clear();vi.mocked(readBrowserSession).mockResolvedValue(null);fetchMock=vi.fn<typeof fetch>(async path=>String(path).startsWith('/api/search')?Response.json({hits:[]}):new Response(result));vi.stubGlobal('fetch',fetchMock)})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('initial SSR mounts without issuing an unfiltered result query and keeps declarative saved searches',async()=>{const root=await mount();expect(fetchMock).not.toHaveBeenCalled();expect(q(root,'[data-advanced-results]').textContent).toContain('Start with a pilot');expect(q<HTMLDetailsElement>(root,'[data-advanced-more]').open).toBe(false);q<HTMLButtonElement>(root,'[data-saved-create]').click();expect(q<HTMLFormElement>(root,'[data-saved-editor]').hidden).toBe(false);expect(q(root,'[data-saved-error]').hidden).toBe(true)})
test('additional filter disclosure preserves the query, applied choices and result controls',async()=>{
 const root=await mount(),more=q<HTMLDetailsElement>(root,'[data-advanced-more]');vi.useFakeTimers()
 more.open=true;q<HTMLButtonElement>(root,'[data-advanced-option="shipCategory"][data-value="frigates"]').click();await vi.advanceTimersByTimeAsync(1)
 const selectedURL=location.href,reads=fetchMock.mock.calls.length;more.open=false
 q<HTMLButtonElement>(root,'[data-advanced-collapse]').click();expect(q(root,'[data-advanced-panels]').hidden).toBe(true);expect(q(root,'[data-advanced-sort]').hidden).toBe(false)
 expect(location.href).toBe(selectedURL);expect(fetchMock.mock.calls).toHaveLength(reads);expect(q(root,'[data-advanced-applied]').textContent).toContain('Frigates');expect(q(root,'[data-advanced-more-count]').textContent).toBe('1 applied')
 q<HTMLButtonElement>(root,'[data-advanced-collapse]').click();q<HTMLButtonElement>(root,'[data-advanced-option="security"][data-value="highsec"]').click();await vi.advanceTimersByTimeAsync(1);expect(more.open).toBe(false)
 q<HTMLButtonElement>(root,'[data-advanced-clear]').click();await vi.advanceTimersByTimeAsync(1);expect(more.open).toBe(false);expect(q(root,'[data-advanced-more-count]').hidden).toBe(true)
})
test('Back and saved searches reveal restored specialist filters without adding disclosure state to links',async()=>{
 const root=await mount(),more=q<HTMLDetailsElement>(root,'[data-advanced-more]');vi.useFakeTimers()
 q<HTMLButtonElement>(root,'[data-advanced-collapse]').click();history.replaceState({},'','/advancedsearch?q='+encodeURIComponent('{"iskMin":"1b"}'));window.dispatchEvent(new PopStateEvent('popstate'));await vi.advanceTimersByTimeAsync(1)
 expect(more.open).toBe(true);expect(q(root,'[data-advanced-panels]').hidden).toBe(false);expect(q<HTMLInputElement>(root,'[data-advanced-isk="iskMin"]').value).toBe('1b')
 more.open=false;const detail={document:savedSearchDocument('{"techLevel":"t2","timeRange":{"to":"2026-10-16T12:00"}}',{view:'fits',dedup:'exact'})};root.dispatchEvent(new CustomEvent('saved-searches:load',{bubbles:true,detail}));await vi.advanceTimersByTimeAsync(1)
 expect(more.open).toBe(true);expect(q(root,'[data-advanced-more-count]').textContent).toBe('2 applied');expect(q(root,'[data-advanced-option="techLevel"][data-value="t2"]').getAttribute('aria-pressed')).toBe('true');expect(q<HTMLInputElement>(root,'[name="advanced_to"]').value).toBe('2026-10-16T12:00')
 const params=new URL(location.href).searchParams;expect(params.get('view')).toBe('fits');expect(params.get('dedup')).toBe('exact');expect([...params.keys()].sort()).toEqual(['dedup','q','view'])
})
test('filters update links, debounce ISK input and cancel superseded result reads',async()=>{
  const root=await mount(),pending=pendingReads();vi.useFakeTimers();const input=q<HTMLInputElement>(root,'[data-advanced-isk="iskMin"]');input.value='500m';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(499);expect(pending).toHaveLength(0);await vi.advanceTimersByTimeAsync(2);expect(pending).toHaveLength(1);expect(JSON.parse(new URL(pending[0].path,location.origin).searchParams.get('q')!).iskMin).toBe('500m')
  q<HTMLButtonElement>(root,'[data-advanced-option="security"][data-value="highsec"]').click();await vi.advanceTimersByTimeAsync(1);expect(pending[0].signal.aborted).toBe(true);pending[1].complete(new Response(result));await vi.advanceTimersByTimeAsync(0);pending[0].complete(new Response('<div data-advanced-result-content>Stale results</div>'));await vi.advanceTimersByTimeAsync(0);expect(root.textContent).not.toContain('Stale results');expect(q(root,'[data-advanced-results]').hasAttribute('aria-busy')).toBe(false)
})
test('lookup actions fit source entity roles, items and locations while excluding malformed hits',async()=>{
  const root=await mount();vi.useFakeTimers();fetchMock.mockImplementation(async path=>String(path).startsWith('/api/search')?Response.json({hits:String(path).includes('type=group')?[]:[{id:'character_42',type:'character',name:'Pilot'},{id:'item_425',type:'item',name:'Autocannon'},{id:'system_30000142',type:'system',name:'Jita'},{id:'bad',type:'character',name:'Bad'}]}):new Response(result))
  const input=q<HTMLInputElement>(root,'[data-advanced-query]');input.value='Fixture';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301);expect(root.querySelectorAll('[data-advanced-suggestions] [data-hit-name]')).toHaveLength(3)
  q<HTMLButtonElement>(root,'[data-advanced-suggestions] button').click();await vi.advanceTimersByTimeAsync(1);expect(JSON.parse(new URL(location.href).searchParams.get('q')!).entities.victim[0]).toMatchObject({id:42,name:'Pilot',role:'victim'})
  input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301);const system=Array.from(root.querySelectorAll<HTMLElement>('[data-hit-name]')).find(node=>node.textContent==='Jita')!.parentElement!.parentElement!.parentElement!;q<HTMLButtonElement>(system,'button').click();await vi.advanceTimersByTimeAsync(1);expect(JSON.parse(new URL(location.href).searchParams.get('q')!).location.systemId).toBe(30000142)
})
test('overlapping normal and group lookup responses produce one suggestion per entity',async()=>{
  const root=await mount();vi.useFakeTimers();fetchMock.mockResolvedValue(Response.json({hits:[{id:'character_42',type:'character',name:'Pilot'}]}));const input=q<HTMLInputElement>(root,'[data-advanced-query]');input.value='Pilot';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301);expect(root.querySelectorAll('[data-advanced-suggestions] [data-hit-name]')).toHaveLength(1)
})
test('custom date changes apply when either endpoint changes; Back restores filter state',async()=>{
  const root=await mount();vi.useFakeTimers();const from=q<HTMLInputElement>(root,'[name="advanced_from"]'),to=q<HTMLInputElement>(root,'[name="advanced_to"]');to.value='2026-10-07T12:00';from.value='2026-10-06T12:00';from.dispatchEvent(new Event('change',{bubbles:true}));await vi.advanceTimersByTimeAsync(1);expect(JSON.parse(new URL(String(fetchMock.mock.calls[0][0]),location.origin).searchParams.get('q')!).timeRange.from).toBe('2026-10-06T12:00')
  history.replaceState({},'','/advancedsearch?q='+encodeURIComponent('{"iskMin":"1b","location":{"securityTypes":["nullsec"]}}'));window.dispatchEvent(new PopStateEvent('popstate'));await vi.advanceTimersByTimeAsync(0);expect(q<HTMLInputElement>(root,'[data-advanced-isk="iskMin"]').value).toBe('1b');expect(q(root,'[data-advanced-option="security"][data-value="nullsec"]').getAttribute('aria-pressed')).toBe('true')
})
test('fitting mode and hash drilldown retain query controls while result failures release loading',async()=>{
  const root=await mount();vi.useFakeTimers();q<HTMLButtonElement>(root,'[data-advanced-option="security"][data-value="highsec"]').click();await vi.advanceTimersByTimeAsync(1);q<HTMLButtonElement>(root,'[data-advanced-view="fits"]').click();await vi.advanceTimersByTimeAsync(0);expect(new URL(location.href).searchParams.get('view')).toBe('fits')
  fetchMock.mockResolvedValueOnce(new Response('<div data-advanced-result-content><div data-fit-preview><button data-fit-show-kills="'+ 'a'.repeat(64)+'" data-fit-dedup="family" data-fit-ship-name="Rifter">Show kills</button></div></div>'));q<HTMLButtonElement>(root,'[data-advanced-option="techLevel"][data-value="t2"]').click();await vi.advanceTimersByTimeAsync(1);q<HTMLButtonElement>(root,'[data-fit-show-kills]').click();await vi.advanceTimersByTimeAsync(0);expect(new URL(location.href).searchParams.get('dh')).toBe('a'.repeat(64));expect(new URL(location.href).searchParams.get('dm')).toBe('family')
  fetchMock.mockResolvedValueOnce(new Response('',{status:503}));q<HTMLButtonElement>(root,'[data-advanced-option="security"][data-value="lowsec"]').click();await vi.advanceTimersByTimeAsync(1);expect(q(root,'[data-advanced-results]').textContent).toContain('Could not load');expect(q(root,'[data-advanced-results]').hasAttribute('aria-busy')).toBe(false)
})
test('disconnect cancels both lookup requests and result timers; restoring interrupted work uses a read',async()=>{
  const root=await mount(),pending=pendingReads();vi.useFakeTimers();const input=q<HTMLInputElement>(root,'[data-advanced-query]');input.value='Fixture';input.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301);expect(pending).toHaveLength(2)
  const isk=q<HTMLInputElement>(root,'[data-advanced-isk="iskMin"]');isk.value='1b';isk.dispatchEvent(new Event('input',{bubbles:true}));root.remove();await vi.advanceTimersByTimeAsync(0);expect(pending.every(read=>read.signal.aborted)).toBe(true);await vi.advanceTimersByTimeAsync(600);expect(pending).toHaveLength(2);document.body.append(root);await vi.advanceTimersByTimeAsync(0);expect(pending).toHaveLength(3);pending[2].complete(new Response(result));await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-advanced-results]').textContent).toContain('Fixture results')
})
