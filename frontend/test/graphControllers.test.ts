// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {GraphController} from '../src/browser/graph'
import {GraphSearchController} from '../src/browser/controllers/graph-search'
import {registerController} from '../src/browser/controllers/application'
import {readGraphData,readGraphHits,graphModes,isGraphMode,type GraphMode,type GraphSearchKey} from '../src/browser/shared/graph-model'
import {renderGraphData} from '../src/browser/graph-render'

const markup = readFileSync(resolve(import.meta.dirname,'fixtures/graph/page.html'),'utf8')
const red = {id:12,name:'Red Alliance'},blue = {id:20,name:'Blue Alliance'}
const samples:Record<GraphMode,unknown> = {
  path_finder:{path:{hops:1,nodes:[{id:1,name:'First Pilot',corporation:red,alliance:blue},{id:2,name:'Second Pilot'}],edges:[{weight:1234}]}},
  entity_intel:{allies:[{...red,shared_enemy_kills:1234,mutual_kills:20}],enemies:[{...blue,kills_given:200,kills_taken:100}]},
  hunting_grounds:{systems:[{id:30000001,name:'Fixture System',active_characters:42}]},
  hot_zones:{systems:[{id:30000002,name:'Fixture Conflict',alliances:2,characters:10}]},
  census:{corps:[{...red,total:12,fcs:1,logis:2,caps:3}],totals:{characters:12,fcs:1,logis:2,caps:3}},
  spy_check:{suspects:[{id:1,name:'Fixture Suspect',corp_name:'Fixture Corp',total_flights:7,connections:[{name:'Outside Group',flights:6}]}]},
  migration:{corp_name:'Fixture Corp',departed:[{id:1,name:'Departed Pilot',current_corp:blue}],joined:[{id:2,name:'Joined Pilot',previous_corp:red}]},
  coalitions:{coalitions:[{id:1,alliances:[{...red,cooperation:1234},{...blue,cooperation:12}]}]},
  rivalries:{items:[{entity_a:red,entity_b:blue,mutual_kills:1234,total_isk:1.234567e9}]},
}
const errors = vi.fn(),settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing graph fixture '+selector);return node}
async function mount(mode:GraphMode = 'path_finder') {history.replaceState(null,'','/graph?tab='+mode);document.body.innerHTML = markup;const root = q<HTMLElement>(document.body,'[data-graph-root]');root.dataset.tab = mode;await settle();await settle();return root}
function panel(root:HTMLElement,mode:GraphMode) {return q<HTMLElement>(root,`[data-graph-panel="${mode}"]`)}
function tab(root:HTMLElement,mode:GraphMode) {q<HTMLButtonElement>(root,`[data-graph-tab="${mode}"]`).click()}
function search(root:HTMLElement,mode:GraphMode,key:GraphSearchKey = 'entity') {return q<HTMLElement>(panel(root,mode),`[data-graph-search="${key}"]`)}
function pick(root:HTMLElement,mode:GraphMode,id = 12,key:GraphSearchKey = 'entity') {search(root,mode,key).dispatchEvent(new CustomEvent('graph-search:pick',{bubbles:true,detail:{mode,key,id}}))}
function input(root:HTMLElement,mode:GraphMode,value:string,key:GraphSearchKey = 'entity') {const control = q<HTMLInputElement>(search(root,mode,key),'input');control.value = value;control.dispatchEvent(new InputEvent('input',{bubbles:true}));return control}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{
  for(const [name,controller] of [['graph',GraphController],['graph-search',GraphSearchController]] as const) registerController(name,controller).handleError = errors
  errors.mockClear();fetchMock = vi.fn<typeof fetch>(async path=>{
    const url = new URL(String(path),location.href)
    if(url.pathname === '/api/search') {const type = url.searchParams.get('type');return Response.json({hits:[{id:type+'_42',name:'Fixture Result',type,ticker:'FIX',corporation_name:'Fixture Corp',alliance_name:'Fixture Alliance',alliance_ticker:'FA'}]})}
    const mode = url.searchParams.get('mode');return Response.json(isGraphMode(mode) ? samples[mode] : {})
  });vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('all nine deep-linked tools retain their Go tab/panel contracts and load only eligible data',async()=>{
  for(const mode of graphModes) {
    fetchMock.mockClear();const root = await mount(mode);expect(root.dataset.tab).toBe(mode);expect(q<HTMLButtonElement>(root,`[data-graph-tab="${mode}"]`).getAttribute('aria-selected')).toBe('true')
    expect([...root.querySelectorAll<HTMLElement>('[data-graph-panel]')].filter(panel=>!panel.hidden)).toHaveLength(1)
    const automatic = ['coalitions','hot_zones','rivalries'].includes(mode);expect(fetchMock).toHaveBeenCalledTimes(automatic ? 1 : 0)
    if(automatic) expect(panel(root,mode).textContent).not.toContain('Loading rivalries...')
  }
})
test('tab keyboard focus and history preserve the query and reveal the chosen button',async()=>{
  const root = await mount();history.replaceState(null,'','/graph?review=1&tab=path_finder')
  const nav = q<HTMLElement>(root,'[role="tablist"]'),button = q<HTMLButtonElement>(root,'[data-graph-tab="path_finder"]'),next = q<HTMLButtonElement>(root,'[data-graph-tab="entity_intel"]')
  vi.spyOn(nav,'getBoundingClientRect').mockReturnValue(new DOMRect(0,0,100,40));vi.spyOn(next,'getBoundingClientRect').mockReturnValue(new DOMRect(120,0,50,40))
  button.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowRight',bubbles:true,cancelable:true}));expect(document.activeElement).toBe(next);expect(nav.scrollLeft).toBe(70);expect(location.search).toContain('review=1');expect(location.search).toContain('tab=entity_intel')
  history.replaceState(null,'','/graph?tab=migration');window.dispatchEvent(new PopStateEvent('popstate'));expect(root.dataset.tab).toBe('migration')
  history.replaceState(null,'','/graph?tab=invalid');window.dispatchEvent(new PopStateEvent('popstate'));expect(root.dataset.tab).toBe('path_finder')
})
test('search debounces, renders escaped metadata and keyboard/pick drives the full path request',async()=>{
  const root = await mount();vi.useFakeTimers();const from = input(root,'path_finder','Fixture','from');from.focus();await vi.advanceTimersByTimeAsync(299);expect(fetchMock).not.toHaveBeenCalled();await vi.advanceTimersByTimeAsync(1)
  const fromSearch = search(root,'path_finder','from');expect(fromSearch.textContent).toContain('Fixture Corp');expect(fromSearch.textContent).toContain('[FA]');from.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true,cancelable:true}));expect(document.activeElement).toBe(q(fromSearch,'[data-graph-hit]'));q<HTMLButtonElement>(fromSearch,'[data-graph-hit]').click();expect(document.activeElement).toBe(from);expect(from.value).toBe('Fixture Result')
  const to = input(root,'path_finder','Second','to');to.focus();await vi.advanceTimersByTimeAsync(301);q<HTMLButtonElement>(search(root,'path_finder','to'),'[data-graph-hit]').click();await vi.advanceTimersByTimeAsync(0)
  expect(fetchMock.mock.calls.at(-1)![0]).toBe('/api/graph?mode=path_finder&fromId=42&toId=42');expect(panel(root,'path_finder').textContent).toContain('Connected');expect(panel(root,'path_finder').textContent).toContain('1.2k')
})
test('new autocomplete text and tab exit cancel debounce/body work without stale hits',async()=>{
  const root = await mount('entity_intel'),pending = pendingRequests();vi.useFakeTimers();input(root,'entity_intel','Older');await vi.advanceTimersByTimeAsync(301);input(root,'entity_intel','Latest');await vi.advanceTimersByTimeAsync(301);expect(pending[0].signal.aborted).toBe(true)
  pending[1].complete(Response.json({hits:[{id:'alliance_12',name:'Latest <b>name</b>',type:'alliance'}]}));await vi.advanceTimersByTimeAsync(0);pending[0].complete(Response.json({hits:[{id:'alliance_20',name:'Stale',type:'alliance'}]}));await vi.advanceTimersByTimeAsync(0)
  const field = search(root,'entity_intel');expect(field.textContent).toContain('Latest <b>name</b>');expect(field.querySelector('b')).toBeNull();expect(field.textContent).not.toContain('Stale')
  input(root,'entity_intel','Pending');tab(root,'migration');await vi.advanceTimersByTimeAsync(301);expect(pending).toHaveLength(2);expect(q<HTMLElement>(field,'[data-graph-hits]').hidden).toBe(true)
})
test('entity type changes reset scoped inputs, reject old hits and retain route kinds',async()=>{
  const root = await mount('entity_intel'),field = search(root,'entity_intel'),pending = pendingRequests();vi.useFakeTimers();input(root,'entity_intel','Alliance');await vi.advanceTimersByTimeAsync(301)
  q<HTMLButtonElement>(panel(root,'entity_intel'),'[data-graph-type="corporation"]').click();expect(pending[0].signal.aborted).toBe(true);expect(q<HTMLInputElement>(field,'input').value).toBe('');expect(q<HTMLInputElement>(field,'input').placeholder).toBe('Search for a corporation...')
  input(root,'entity_intel','Corporation');await vi.advanceTimersByTimeAsync(301);expect(pending[1].path).toContain('type=corporation');pending[1].complete(Response.json({hits:[{id:'corporation_12',name:'Corp',type:'corporation'}]}));await vi.advanceTimersByTimeAsync(0);q<HTMLButtonElement>(field,'[data-graph-hit]').click();expect(pending[2].path).toContain('entityType=corporation');pending[2].complete(Response.json(samples.entity_intel));await vi.advanceTimersByTimeAsync(0);expect(q(panel(root,'entity_intel'),'[data-graph-result]').innerHTML).toContain('/corporation/12')
})
test('each entity tool sends the existing mode/id/type query and preserves its rendered fields',async()=>{
  const root = await mount()
  for(const mode of ['entity_intel','hunting_grounds','census','spy_check','migration'] as const) {
    tab(root,mode);pick(root,mode);await settle();expect(fetchMock.mock.calls.at(-1)![0]).toBe('/api/graph?mode='+mode+'&entityId=12'+(['census','migration'].includes(mode) ? '' : '&entityType=alliance'))
    expect(q(panel(root,mode),'[data-graph-result]').textContent).not.toContain('Unable to load')
  }
  expect(panel(root,'census').textContent).toContain('12');expect(panel(root,'migration').textContent).toContain('now in Blue Alliance');expect(panel(root,'spy_check').textContent).toContain('Outside Group')
})
test('hidden graph reads are canceled and selected tools resume only their latest query',async()=>{
  const root = await mount('entity_intel'),pending = pendingRequests();pick(root,'entity_intel',12);tab(root,'migration');expect(pending[0].signal.aborted).toBe(true);tab(root,'entity_intel');expect(pending[1].path).toBe(pending[0].path)
  pending[1].complete(Response.json(samples.entity_intel));await settle();pending[0].complete(Response.json({allies:[{id:99,name:'Stale'}],enemies:[]}));await settle();expect(panel(root,'entity_intel').textContent).not.toContain('Stale');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('cached coalition DOM and its expanded member rows survive tabs without another read',async()=>{
  const root = await mount('coalitions'),section = panel(root,'coalitions'),button = q<HTMLButtonElement>(section,'[data-graph-coalition]');button.click();const members = q<HTMLElement>(section,'[data-graph-coalition-members]');expect(members.hidden).toBe(false);const calls = fetchMock.mock.calls.length
  tab(root,'path_finder');tab(root,'coalitions');await settle();expect(fetchMock).toHaveBeenCalledTimes(calls);expect(q(section,'[data-graph-coalition-members]')).toBe(members);expect(members.hidden).toBe(false);expect(button.getAttribute('aria-expanded')).toBe('true')
})
test('rivalry switching accepts alliance/character results and caches each type',async()=>{
  const root = await mount('rivalries'),section = panel(root,'rivalries');q<HTMLButtonElement>(section,'[data-graph-type="character"]').click();await settle();expect(fetchMock.mock.calls.at(-1)![0]).toBe('/api/graph?mode=rivalries&entityType=character&limit=50');expect(q(section,'[data-graph-result]').innerHTML).toContain('/character/12')
  const count = fetchMock.mock.calls.length;q<HTMLButtonElement>(section,'[data-graph-type="alliance"]').click();await settle();expect(fetchMock).toHaveBeenCalledTimes(count);expect(q(section,'[data-graph-result]').innerHTML).toContain('/alliance/12')
})
test('failed and wrong-mode bodies are visible and a single retry retains the requested IDs',async()=>{
  const root = await mount('entity_intel'),pending = pendingRequests();pick(root,'entity_intel',12);pending[0].complete(new Response('Failure',{status:503}));await settle();expect(panel(root,'entity_intel').textContent).toContain('Unable to load graph intelligence')
  const retry = q<HTMLButtonElement>(panel(root,'entity_intel'),'[data-graph-retry]');retry.click();retry.click();expect(pending).toHaveLength(2);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(Response.json({mode:'census',corps:[],totals:{}}));await settle();expect(panel(root,'entity_intel').querySelectorAll('[data-graph-error]')).toHaveLength(1)
  q<HTMLButtonElement>(panel(root,'entity_intel'),'[data-graph-retry]').click();pending[2].complete(Response.json(samples.entity_intel));await settle();expect(panel(root,'entity_intel').textContent).toContain('1.2k shared');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('a forced refresh keeps prior result nodes and failure leaves them usable for retry',async()=>{
  const root = await mount('entity_intel');pick(root,'entity_intel',12);await settle();const result = q(panel(root,'entity_intel'),'[data-graph-result]'),link = q(result,'a'),pending = pendingRequests();pick(root,'entity_intel',12);expect(q(result,'a')).toBe(link);pending[0].complete(new Response('Failure',{status:503}));await settle();expect(q(result,'a')).toBe(link);expect(result.textContent).toContain('Retry')
})
test('late JSON completed after detach cannot paint and busy state is cleared',async()=>{
  const root = await mount('entity_intel');let complete:(value:unknown)=>void = ()=>{};const response = Response.json({});vi.spyOn(response,'json').mockImplementation(()=>new Promise(resolve=>complete = resolve));fetchMock.mockResolvedValue(response);pick(root,'entity_intel',12);await settle();root.remove();await settle();complete(samples.entity_intel);await settle();expect(panel(root,'entity_intel').textContent).not.toContain('Red Alliance');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('page return resumes the selected interrupted read and never loads hidden tools',async()=>{
  const root = await mount('entity_intel'),pending = pendingRequests();pick(root,'entity_intel',12);hide();show();expect(pending[0].signal.aborted).toBe(true);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(Response.json(samples.entity_intel));await settle();expect(pending).toHaveLength(2);expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('picker Escape/outside input and pagehide clear requests and keyboard-expanded state',async()=>{
  const root = await mount('entity_intel'),field = search(root,'entity_intel');vi.useFakeTimers();const control = input(root,'entity_intel','Fixture');await vi.advanceTimersByTimeAsync(301);control.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true,cancelable:true}));q<HTMLButtonElement>(field,'[data-graph-hit]').dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true,cancelable:true}));expect(document.activeElement).toBe(control);expect(control.getAttribute('aria-expanded')).toBe('false')
  input(root,'entity_intel','Again');await vi.advanceTimersByTimeAsync(301);document.body.dispatchEvent(new PointerEvent('pointerdown',{bubbles:true}));expect(q<HTMLElement>(field,'[data-graph-hits]').hidden).toBe(true);input(root,'entity_intel','Pending');hide();const count = fetchMock.mock.calls.length;await vi.advanceTimersByTimeAsync(301);expect(fetchMock).toHaveBeenCalledTimes(count)
})
test('a rivalry cache hit cancels a competing type read and rejects its late body',async()=>{
  const root = await mount('rivalries'),section = panel(root,'rivalries'),pending = pendingRequests();q<HTMLButtonElement>(section,'[data-graph-type="character"]').click();q<HTMLButtonElement>(section,'[data-graph-type="alliance"]').click()
  expect(pending).toHaveLength(1);expect(pending[0].signal.aborted).toBe(true);expect(q(section,'[data-graph-result]').innerHTML).toContain('/alliance/12');pending[0].complete(Response.json({items:[{entity_a:{id:1,name:'Stale character'},entity_b:blue}]}));await settle();expect(section.textContent).not.toContain('Stale character');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('all graph models preserve Go field names and discard malformed rows/unsafe counters',()=>{
  for(const mode of graphModes) expect(renderGraphData(readGraphData(mode,samples[mode]))).toBeTruthy()
  const path = readGraphData('path_finder',samples.path_finder);if(path.mode !== 'path_finder' || !path.path) throw new Error('Missing fixture path');expect(path.path.nodes[0]).toMatchObject({corporation:{id:12,name:'Red Alliance'},alliance:{id:20,name:'Blue Alliance'}})
  const coalitions = readGraphData('coalitions',samples.coalitions);if(coalitions.mode !== 'coalitions') throw new Error('Missing fixture coalition');expect(coalitions.coalitions[0].alliances[0].cooperation).toBe(1234);expect(readGraphData('census',samples.census)).toMatchObject({totals:{characters:12}})
  expect(readGraphData('entity_intel',{allies:[null,{id:-1},{id:12,name:'A',shared_enemy_kills:Infinity}],enemies:[]})).toMatchObject({allies:[{id:12,shared_enemy_kills:0}]})
  expect(()=>readGraphData('hot_zones',{systems:{}})).toThrow('unavailable');expect(()=>readGraphData('path_finder',{path:{nodes:[{id:-1}],edges:[]}})).toThrow('unavailable')
  expect(readGraphHits({hits:[{id:'alliance_12',name:'A',type:'alliance'},{id:'character_1',name:'Wrong',type:'character'},{id:{},name:'Bad'}]},'alliance').map(hit=>hit.id)).toEqual([12])
})
