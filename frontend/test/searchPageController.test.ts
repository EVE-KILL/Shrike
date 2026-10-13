// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {SearchPageController} from '../src/browser/search'
import {registerController} from '../src/browser/controllers/application'
import {readSearchResults,searchID,searchEntityImage,searchEntityPath,searchTypes} from '../src/browser/shared/search-model'
import {searchCard} from '../src/browser/shared/search-cards'
import wire from './fixtures/search/results.json'

const directory = resolve(import.meta.dirname,'fixtures/search')
const markup = Object.fromEntries(['initial','query'].map(name=>[name,readFileSync(resolve(directory,name+'.html'),'utf8').replace(/<script[^>]*type="module"[^>]*>[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'')]))
const errors = vi.fn(),settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing search fixture '+selector);return node}
async function mount(name='initial') {document.body.innerHTML = markup[name];await settle();await settle();return q<HTMLElement>(document.body,'.search-page')}
function type(root:HTMLElement,value:string) {const input = q<HTMLInputElement>(root,'input');input.value = value;input.dispatchEvent(new InputEvent('input',{bubbles:true}));return input}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{registerController('search-page',SearchPageController).handleError = errors;errors.mockClear();history.replaceState({fixture:'retained'},'','/search');fetchMock = vi.fn<typeof fetch>(async()=>Response.json(wire));vi.stubGlobal('fetch',fetchMock)})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('empty/short searches retain source initial suggestions without network work',async()=>{
  const root = await mount();vi.useFakeTimers();expect(fetchMock).not.toHaveBeenCalled();expect(q(root,'#search-page-initial').hidden).toBe(false);type(root,'a');await vi.advanceTimersByTimeAsync(250);expect(fetchMock).not.toHaveBeenCalled();expect(q(root,'#search-page-grid').hidden).toBe(true);expect(q(root,'#search-page-clear').hidden).toBe(false);expect(document.title).toBe('Search - EVE-KILL.com')
})
test('search debounces for 200ms, keeps query/history state and renders the real Go payload',async()=>{
  const root = await mount();history.replaceState(history.state,'','/search?keep=1');vi.useFakeTimers();type(root,'Fixture');expect(q(root,'#search-page-spinner').hidden).toBe(false);await vi.advanceTimersByTimeAsync(199);expect(fetchMock).not.toHaveBeenCalled();await vi.advanceTimersByTimeAsync(1);expect(fetchMock.mock.calls[0][0]).toBe('/api/search?q=Fixture');expect(root.querySelectorAll('.search-page-card')).toHaveLength(11);expect(q(root,'#search-page-stats').textContent).toBe(`11 results · ${wire.processingTimeMs}ms`);expect(history.state).toEqual({fixture:'retained'});expect(location.search).toContain('keep=1');expect(new URL(location.href).searchParams.get('q')).toBe('Fixture');expect(q(root,'#search-page-spinner').hidden).toBe(true)
})
test('all eleven result types retain routes, image variants, badges and affiliation logos',async()=>{
  const root = await mount();vi.useFakeTimers();type(root,'Fixture');await vi.advanceTimersByTimeAsync(201)
  const expected = new Map([['character','/character/4242'],['corporation','/corporation/10'],['alliance','/alliance/70121'],['faction','/faction/500003'],['system','/system/30002813'],['region','/region/10000042'],['constellation','/constellation/20000396'],['ship','/item/587'],['item','/item/34'],['shipgroup','/group/25'],['group','/group/52']])
  for(const [type,path] of expected) {const card = q(root,`.search-page-card[data-type="${type}"]`);expect(q(card,'.search-page-card-main').getAttribute('href')).toBe(path);expect(q(card,'.search-page-badge').textContent).toBe(type);expect(card.classList.contains('search-type-'+type)).toBe(true)}
  expect(q<HTMLImageElement>(root,'.search-type-character .search-page-media img').src).toBe('https://images.evetech.net/characters/4242/portrait?size=512');expect(q<HTMLImageElement>(root,'.search-type-ship img').getAttribute('src')).toBe('/images/types/587/overlayrender?size=512');expect(q<HTMLImageElement>(root,'.search-type-item img').getAttribute('src')).toBe('/images/types/34/icon?size=64');expect(q(root,'.search-type-group .search-page-media').classList.contains('small')).toBe(true);expect(root.querySelectorAll('.search-type-character .search-page-logo')).toHaveLength(2);expect(q(root,'.search-type-character .search-page-info').textContent).toContain('Fixture Corp [FC]');expect(q(root,'.search-type-character .search-page-info').textContent).toContain('Fixture Alliance [FA]')
})
test('filtering reuses existing cards/images and preserves selected button state without requests',async()=>{
  const root = await mount();vi.useFakeTimers();type(root,'Fixture');await vi.advanceTimersByTimeAsync(201);const original = q(root,'.search-type-character'),image = q(original,'img'),calls = fetchMock.mock.calls.length;q<HTMLButtonElement>(root,'.search-page-tab[data-type="character"]').click();expect(root.querySelectorAll('#search-page-grid .search-page-card')).toHaveLength(1);expect(q(root,'.search-type-character')).toBe(original);expect(q(original,'img')).toBe(image);expect(q(root,'.search-page-tab[data-type="character"]').getAttribute('aria-pressed')).toBe('true');q<HTMLButtonElement>(root,'.search-page-tab[data-type="all"]').click();expect(root.querySelectorAll('#search-page-grid .search-page-card')).toHaveLength(11);expect(q(root,'.search-type-character')).toBe(original);expect(fetchMock).toHaveBeenCalledTimes(calls)
})
test('suggestions use the same debounce and keep input focus; Enter does not submit a document',async()=>{
  const root = await mount();vi.useFakeTimers();q<HTMLButtonElement>(root,'[data-suggestion="Jita"]').click();const input = q<HTMLInputElement>(root,'input');expect(input.value).toBe('Jita');expect(document.activeElement).toBe(input);const event = new KeyboardEvent('keydown',{key:'Enter',bubbles:true,cancelable:true});input.dispatchEvent(event);expect(event.defaultPrevented).toBe(true);await vi.advanceTimersByTimeAsync(201);expect(fetchMock.mock.calls[0][0]).toBe('/api/search?q=Jita')
})
test('Clear cancels pending work, preserves other URL state and restores source suggestions/focus',async()=>{
  const root = await mount(),pending = pendingRequests();history.replaceState(history.state,'','/search?keep=1&q=Old');vi.useFakeTimers();type(root,'Fixture');await vi.advanceTimersByTimeAsync(201);q<HTMLButtonElement>(root,'#search-page-clear').click();expect(pending[0].signal.aborted).toBe(true);expect(q<HTMLInputElement>(root,'input').value).toBe('');expect(document.activeElement).toBe(q(root,'input'));expect(location.search).toBe('?keep=1');pending[0].complete(Response.json(wire));await vi.advanceTimersByTimeAsync(0);expect(root.querySelectorAll('.search-page-card')).toHaveLength(0);expect(q(root,'#search-page-initial').hidden).toBe(false)
})
test('a newer query aborts old work through response text and ignores a late payload',async()=>{
  const root = await mount(),pending = pendingRequests();vi.useFakeTimers();type(root,'Older');await vi.advanceTimersByTimeAsync(201);type(root,'Newest');await vi.advanceTimersByTimeAsync(201);expect(pending[0].signal.aborted).toBe(true);pending[1].complete(Response.json({hits:[{id:'ship_587',name:'Newest Rifter',type:'ship'}],entityCounts:{ship:1},total:1,processingTimeMs:2}));await vi.advanceTimersByTimeAsync(0);pending[0].complete(Response.json(wire));await vi.advanceTimersByTimeAsync(0);expect(root.querySelectorAll('.search-page-card')).toHaveLength(1);expect(q(root,'.search-page-info strong').textContent).toBe('Newest Rifter');expect(new URL(location.href).searchParams.get('q')).toBe('Newest')
})
test('errors retain usable prior cards and retry is distinct from no results',async()=>{
  const root = await mount();vi.useFakeTimers();type(root,'Fixture');await vi.advanceTimersByTimeAsync(201);const original = q(root,'.search-page-card'),pending = pendingRequests();type(root,'Failure');await vi.advanceTimersByTimeAsync(201);pending[0].complete(new Response('Failure',{status:503}));await vi.advanceTimersByTimeAsync(0);expect(q(root,'.search-page-card')).toBe(original);expect(q(root,'[data-search-error]').hidden).toBe(false);expect(q(root,'#search-page-empty').hidden).toBe(true);const retry = q<HTMLButtonElement>(root,'[data-search-retry]');retry.click();retry.click();await vi.advanceTimersByTimeAsync(201);expect(pending).toHaveLength(2);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(Response.json({hits:[],entityCounts:{},total:0,processingTimeMs:0}));await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-search-error]').hidden).toBe(true);expect(q(root,'#search-page-empty-message').textContent).toBe('No results found for "Failure"');expect(q(root,'#search-page-empty').hidden).toBe(false)
})
test('malformed JSON payloads expose retry without committing a search URL',async()=>{
  const root = await mount();vi.useFakeTimers();fetchMock.mockResolvedValue(Response.json({hits:'invalid'}));type(root,'Fixture');await vi.advanceTimersByTimeAsync(201);expect(q(root,'[data-search-error]').hidden).toBe(false);expect(location.search).toBe('');expect(q(root,'#search-page-spinner').hidden).toBe(true)
})
test('Back starts only its restored query and ignores history changes for another route',async()=>{
  const root = await mount();vi.useFakeTimers();history.replaceState(history.state,'','/search?q=Fixture&keep=1');window.dispatchEvent(new PopStateEvent('popstate'));await vi.advanceTimersByTimeAsync(201);expect(q<HTMLInputElement>(root,'input').value).toBe('Fixture');expect(root.querySelectorAll('.search-page-card')).toHaveLength(11);history.replaceState(history.state,'','/search?keep=1');window.dispatchEvent(new PopStateEvent('popstate'));expect(q(root,'#search-page-initial').hidden).toBe(false);expect(location.search).toBe('?keep=1');history.replaceState(history.state,'','/blog?q=Other');window.dispatchEvent(new PopStateEvent('popstate'));await vi.advanceTimersByTimeAsync(201);expect(fetchMock).toHaveBeenCalledTimes(1)
})
test('deep-link SSR input runs exactly one search after debounce',async()=>{
  history.replaceState(history.state,'','/search?q=Fixture');await mount('query');await new Promise(resolve=>setTimeout(resolve,220));expect(fetchMock).toHaveBeenCalledTimes(1);expect(document.querySelectorAll('.search-page-card')).toHaveLength(11)
})
test('page return resumes interrupted debounce/read without rerunning settled searches',async()=>{
  const root = await mount(),pending = pendingRequests();vi.useFakeTimers();type(root,'Fixture');hide();show();await vi.advanceTimersByTimeAsync(201);expect(pending).toHaveLength(1);hide();expect(pending[0].signal.aborted).toBe(true);show();await vi.advanceTimersByTimeAsync(201);expect(pending).toHaveLength(2);pending[1].complete(Response.json(wire));await vi.advanceTimersByTimeAsync(0);hide();show();await vi.advanceTimersByTimeAsync(201);expect(pending).toHaveLength(2);expect(root.querySelectorAll('.search-page-card')).toHaveLength(11)
})
test('JSON completing after detach cannot render cards or change history',async()=>{
  const root = await mount();let release:(value:unknown)=>void = ()=>{};const response = Response.json({});vi.spyOn(response,'json').mockImplementation(()=>new Promise(resolve=>release = resolve));fetchMock.mockResolvedValue(response);vi.useFakeTimers();type(root,'Fixture');await vi.advanceTimersByTimeAsync(201);root.remove();await vi.advanceTimersByTimeAsync(0);release(wire);await vi.advanceTimersByTimeAsync(0);expect(root.querySelectorAll('.search-page-card')).toHaveLength(0);expect(root.hasAttribute('aria-busy')).toBe(false);expect(location.search).toBe('')
})
test('names/tickers render as text and unsafe result IDs/types cannot become cards or routes',()=>{
  const data = readSearchResults({hits:[{id:'character_4242',type:'character',name:'<img onerror="bad">',ticker:'<b>X</b>'},{id:'ship_../evil',type:'ship'},{id:'character_42',type:'constructor'},{id:-1,type:'item'}],entityCounts:{character:1,constructor:2},total:1,processingTimeMs:1});expect(data.hits).toHaveLength(1);const card = searchCard(data.hits[0]);expect(card.querySelector('strong')!.textContent).toBe('<img onerror="bad">');expect(card.querySelector('b')).toBeNull();expect(card.querySelector('[onerror]')).toBeNull();expect(data.counts).toEqual({character:1})
})
test('wire boundaries preserve valid EVE IDs and all original image/route choices',()=>{
  expect([42,'character_4242','SHIP_587','item_2147483647'].map(searchID)).toEqual([42,4242,587,2147483647]);expect([true,{},'0','item_-1','ship_1e3','2147483648',NaN].map(searchID)).toEqual([0,0,0,0,0,0,0]);const data = readSearchResults(wire);expect(Object.keys(searchTypes).sort()).toEqual(data.hits.map(hit=>hit.type).sort());for(const hit of data.hits) {expect(searchEntityPath(hit)).toMatch(/^\/(item|group|character|corporation|alliance|faction|system|region|constellation)\/\d+$/);if(hit.type === 'shipgroup' || hit.type === 'group') expect(searchEntityImage(hit)).toBe('');else expect(searchEntityImage(hit)).toContain('?size=')}
})
