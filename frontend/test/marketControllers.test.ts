// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {MarketBrowserController} from '../src/browser/market'
import {ItemMarketController} from '../src/browser/controllers/item-market'
import {MarketSearchController} from '../src/browser/controllers/market-search'
import {registerController} from '../src/browser/controllers/application'
import {readMarketFilters,marketParams,toggleMarketSecurity,sortMarketBook,readMarketHits,mergeMarketHits} from '../src/browser/shared/market'

const directory = resolve(import.meta.dirname,'fixtures/market')
const names = ['browser','standalone','item','standalone-panel','standalone-filtered','item-filtered','tree'] as const
const markup = Object.fromEntries(names.map(name=>[name,readFileSync(resolve(directory,name+'.html'),'utf8').replace(/<script[^>]*type="module"[^>]*>[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'')]))
const errors = vi.fn(),settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing market fixture '+selector);return node}
async function mount(name:typeof names[number] = 'standalone',prepare?:(root:HTMLElement)=>void) {document.body.innerHTML = markup[name];const root = q<HTMLElement>(document.body,':scope > *');prepare?.(root);await settle();await settle();return root}
function current() {return q<HTMLElement>(document.body,'[data-item-market]')}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
function type(root:HTMLElement,value:string) {const input = q<HTMLInputElement>(root,'[data-market-search] input');input.value = value;input.dispatchEvent(new InputEvent('input',{bubbles:true}));return input}
function region(root:HTMLElement,value:string) {const select = q<HTMLSelectElement>(root,'[data-market-region]');select.value = value;select.dispatchEvent(new Event('change',{bubbles:true}));return select}
function panelResponse(path:string) {const url = new URL(path,location.href),item = url.pathname.includes('/item/market/'),raw = markup[item ? 'item' : 'standalone-panel'];return raw.replace(/data-region="[^"]+"/g,'data-region="'+url.searchParams.get('region')+'"').replace(/data-days="[^"]+"/g,'data-days="'+url.searchParams.get('days')+'"').replace(/data-security="[^"]+"/g,'data-security="'+url.searchParams.get('security')+'"')}
function rows(root:ParentNode,kind = 'sellers') {return [...q(root,`[data-market-book="${kind}"] tbody`).querySelectorAll<HTMLTableRowElement>('tr[data-order-id]')].map(row=>row.dataset.orderId)}
function addOrders(root:HTMLElement,kind = 'sellers') {const book = q<HTMLElement>(root,`[data-market-book="${kind}"]`),tbody = q(book,'tbody'),original = q<HTMLTableRowElement>(tbody,'tr[data-order-id]');tbody.replaceChildren();for(const [id,price,quantity,min,expires] of [['3','100','2','5','300'],['1','100','9','1','100'],['2','50','1','3','200']]) {const row = original.cloneNode(true) as HTMLTableRowElement;Object.assign(row.dataset,{orderId:id,price,quantity,minVolume:min,expires});tbody.append(row)}return book}
beforeEach(()=>{
  for(const [name,controller] of [['market-browser',MarketBrowserController],['item-market',ItemMarketController],['market-search',MarketSearchController]] as const) registerController(name,controller).handleError = errors
  errors.mockClear();history.replaceState({fixture:'retained'},'','/market/item/11186')
  fetchMock = vi.fn<typeof fetch>(async path=>{const url = String(path);if(url.includes('/market/tree/')) return new Response(markup.tree);if(url.startsWith('/api/search')) {const type = new URL(url,location.href).searchParams.get('type');return Response.json({hits:[{id:type+'_587',name:'Fixture <b>Rifter</b>',type}]})}return new Response(panelResponse(url))});vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('standalone/item scopes retain initial SSR charts/images/order books without requests',async()=>{
  const root = await mount();expect(fetchMock).not.toHaveBeenCalled();expect(q(root,'[data-item-market] img')).toBeTruthy();expect(q(root,'[data-item-market] polyline')).toBeTruthy();expect(q(root,'[data-market-book="buyers"]')).toBeTruthy();await mount('item');expect(fetchMock).not.toHaveBeenCalled();expect(current().dataset.marketView).toBe('item')
})
test('all order-book sorts preserve rows, directions and order-ID tie breaking',async()=>{
  const root = await mount(),book = addOrders(root),original = [...book.querySelectorAll('tr[data-order-id]')]
  q<HTMLButtonElement>(book,'[data-market-sort="price"]').click();expect(rows(root)).toEqual(['1','3','2']);expect(q(book,'[data-market-sort="price"]').closest('th')!.getAttribute('aria-sort')).toBe('descending')
  q<HTMLButtonElement>(book,'[data-market-sort="price"]').click();expect(rows(root)).toEqual(['2','1','3']);q<HTMLButtonElement>(book,'[data-market-sort="quantity"]').click();expect(rows(root)).toEqual(['1','3','2']);q<HTMLButtonElement>(book,'[data-market-sort="min_volume"]').click();expect(rows(root)).toEqual(['1','2','3']);q<HTMLButtonElement>(book,'[data-market-sort="expires"]').click();expect(rows(root)).toEqual(['1','2','3']);expect([...book.querySelectorAll('tr[data-order-id]')].every(row=>original.includes(row))).toBe(true)
})
test('sort ties compare large numeric order IDs without losing integer precision',async()=>{
  const root = await mount(),book = addOrders(root),entries = [...book.querySelectorAll<HTMLTableRowElement>('tr[data-order-id]')];entries[0].dataset.orderId = '9007199254740993';entries[1].dataset.orderId = '9007199254740992';entries[2].dataset.price = '100';sortMarketBook(book,'price',1);expect(rows(root)).toEqual(['2','9007199254740992','9007199254740993'])
})
test('pending region/day/security choices combine and reject stale bodies while retaining SSR',async()=>{
  const root = await mount(),old = current(),image = q(old,'img'),pending = pendingRequests();region(root,'10000002');q<HTMLButtonElement>(old,'[data-market-days="90"]').click();q<HTMLButtonElement>(old,'[data-market-security="low"]').click();expect(pending).toHaveLength(3);expect(pending.slice(0,2).every(read=>read.signal.aborted)).toBe(true);expect(pending[2].path).toBe('/_fragments/market/item/11186?region=10000002&days=90&security=high%2Cnull');expect(q(old,'img')).toBe(image)
  pending[2].complete(new Response(markup['standalone-filtered']));await settle();await settle();expect(current()).not.toBe(old);expect(current().dataset.security).toBe('high,null');pending[0].complete(new Response(markup['standalone-panel']));pending[1].complete(new Response(panelResponse(pending[1].path)));await settle();expect(current().dataset.days).toBe('90');expect(history.state).toEqual({fixture:'retained'});expect(location.search).toContain('region=10000002')
})
test('sorting chosen during a filter read is applied to the newly returned books',async()=>{
  const root = await mount(),old = current();addOrders(root);const pending = pendingRequests();q<HTMLButtonElement>(old,'[data-market-days="90"]').click();q<HTMLButtonElement>(old,'[data-market-sort="quantity"]').click()
  const next = document.createElement('div');next.innerHTML = panelResponse(pending[0].path);addOrders(next);pending[0].complete(new Response(next.innerHTML));await settle();await settle();expect(rows(current())).toEqual(['1','3','2']);expect(q(current(),'[data-market-book="sellers"] [data-market-sort="quantity"]').closest('th')!.getAttribute('aria-sort')).toBe('descending')
})
test('market errors/wrong identities retain old data and offer one retry with desired filters',async()=>{
  const root = await mount(),old = current(),pending = pendingRequests();q<HTMLButtonElement>(old,'[data-market-days="90"]').click();pending[0].complete(new Response('Failed',{status:503}));await settle();expect(current()).toBe(old)
  const retry = q<HTMLButtonElement>(old,'[data-market-retry]');retry.click();retry.click();expect(pending).toHaveLength(2);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(panelResponse(pending[1].path).replace('data-market-type-id="11186"','data-market-type-id="42"')));await settle();expect(old.querySelectorAll('[data-market-error]')).toHaveLength(1);q<HTMLButtonElement>(old,'[data-market-retry]').click();pending[2].complete(new Response(panelResponse(pending[2].path)));await settle();await settle();expect(current()).not.toBe(old)
})
test('item market fragments use their own endpoint and leave parent tab history ownership intact',async()=>{
  history.replaceState({},'','/item/34/market');const root = await mount('item'),pending = pendingRequests();region(root,'10000002');q<HTMLButtonElement>(root,'[data-market-days="90"]').click();q<HTMLButtonElement>(root,'[data-market-security="low"]').click();expect(pending[2].path).toBe('/_fragments/item/market/34?region=10000002&days=90&security=high%2Cnull');pending[2].complete(new Response(markup['item-filtered']));await settle();await settle();const count = pending.length;window.dispatchEvent(new PopStateEvent('popstate'));expect(pending).toHaveLength(count);expect(current().dataset.marketView).toBe('item')
})
test('standalone Back reads URL filters and page return resumes an interrupted selection',async()=>{
  const root = await mount(),pending = pendingRequests();history.replaceState(history.state,'','/market/item/11186?days=90');window.dispatchEvent(new PopStateEvent('popstate'));expect(pending[0].path).toContain('days=90');hide();show();expect(pending[0].signal.aborted).toBe(true);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(panelResponse(pending[1].path)));await settle();await settle();expect(current().dataset.days).toBe('90');expect(root.contains(current())).toBe(true)
})
test('late filter text completed after detach cannot replace or leave its cached root busy',async()=>{
  const root = await mount(),old = current();let complete:(text:string)=>void = ()=>{};const response = new Response('');vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>complete = resolve));fetchMock.mockResolvedValue(response);q<HTMLButtonElement>(old,'[data-market-days="90"]').click();await settle();root.remove();await settle();complete(markup['standalone-filtered']);await settle();expect(old.dataset.days).toBe('30');expect(old.hasAttribute('aria-busy')).toBe(false)
})
test('item Back changes the route before a market response and prevents stale URL filters',async()=>{
  history.replaceState({},'','/item/34/market');const root = await mount('item'),pending = pendingRequests();q<HTMLButtonElement>(root,'[data-market-days="90"]').click();history.replaceState({},'','/item/34');pending[0].complete(new Response(panelResponse(pending[0].path)));await settle();expect(current()).toBe(root);expect(current().dataset.days).toBe('30');expect(location.search).toBe('');expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('browser sidebar labels/expanded state preserve category and standalone wording',async()=>{
  let root = await mount('browser'),button = q<HTMLButtonElement>(root,'[data-market-sidebar-toggle]');button.click();expect(button.textContent).toBe('Hide Categories');expect(button.getAttribute('aria-expanded')).toBe('true');button.click();expect(button.textContent).toBe('Show Categories');root = await mount();button = q(root,'[data-market-sidebar-toggle]');button.click();expect(button.textContent).toBe('Hide Market Browser')
})
test('lazy trees are single flight, cache exact nodes and cancel closed-parent requests',async()=>{
  const root = await mount('browser'),details = q<HTMLDetailsElement>(root,'details[data-market-node="4"]'),pending = pendingRequests();details.open = true;details.dispatchEvent(new Event('toggle'));details.dispatchEvent(new Event('toggle'));expect(pending).toHaveLength(1);details.open = false;details.dispatchEvent(new Event('toggle'));expect(pending[0].signal.aborted).toBe(true);details.open = true;details.dispatchEvent(new Event('toggle'));pending[1].complete(new Response(markup.tree));await settle();const tree = q(details,':scope > .market-tree');pending[0].complete(new Response('<div class="market-tree" data-market-tree-parent="4">Stale tree</div>'));await settle();expect(details.textContent).not.toContain('Stale tree');details.open = false;details.dispatchEvent(new Event('toggle'));details.open = true;details.dispatchEvent(new Event('toggle'));expect(q(details,':scope > .market-tree')).toBe(tree);expect(pending).toHaveLength(2)
})
test('failed/wrong-parent tree bodies retry on reopen and page return reconciles open nodes',async()=>{
  const root = await mount('browser'),details = q<HTMLDetailsElement>(root,'details[data-market-node="4"]'),pending = pendingRequests();details.open = true;details.dispatchEvent(new Event('toggle'));pending[0].complete(new Response(markup.tree.replace('data-market-tree-parent="4"','data-market-tree-parent="5"')));await settle();expect(details.textContent).toContain('Could not load categories');details.open = false;details.dispatchEvent(new Event('toggle'));details.open = true;details.dispatchEvent(new Event('toggle'));hide();show();expect(pending[1].signal.aborted).toBe(true);pending[2].complete(new Response(markup.tree));await settle();expect(q(details,':scope > .market-tree').textContent).toContain('Frigates')
})
test('search requests ship/item in parallel, deduplicates IDs and renders names as text',async()=>{
  const root = await mount('browser');vi.useFakeTimers();const input = type(root,'Fixture');await vi.advanceTimersByTimeAsync(219);expect(fetchMock).not.toHaveBeenCalled();await vi.advanceTimersByTimeAsync(1);expect(fetchMock).toHaveBeenCalledTimes(2);const results = q(root,'[data-market-results]');expect(results.querySelectorAll('a')).toHaveLength(1);expect(results.textContent).toContain('Fixture <b>Rifter</b>');expect(results.querySelector('b')).toBeNull();expect(q(results,'a').getAttribute('href')).toBe('/market/item/587');expect(q<HTMLImageElement>(results,'img').width).toBe(28);input.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true,cancelable:true}));expect(document.activeElement).toBe(q(results,'a'));q(results,'a').dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true,cancelable:true}));expect(document.activeElement).toBe(input);expect(results.hidden).toBe(true)
})
test('search errors are distinguished from empty results and retry cancels old parallel work',async()=>{
  const root = await mount('browser'),pending = pendingRequests();vi.useFakeTimers();type(root,'Fixture');await vi.advanceTimersByTimeAsync(221);pending[0].complete(new Response('Failed',{status:503}));await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-market-results]').textContent).toContain('Search unavailable');expect(pending[1].signal.aborted).toBe(true);q<HTMLButtonElement>(root,'[data-market-search-retry]').click();pending[2].complete(Response.json({hits:[]}));pending[3].complete(Response.json({hits:[]}));await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-market-results]').textContent).toBe('No market items found.');hide()
})
test('new search/escape/pagehide dispose reads and late payloads cannot reopen the dropdown',async()=>{
  const root = await mount('browser'),pending = pendingRequests();vi.useFakeTimers();type(root,'Older');await vi.advanceTimersByTimeAsync(221);const input = type(root,'Latest');await vi.advanceTimersByTimeAsync(221);expect(pending.slice(0,2).every(read=>read.signal.aborted)).toBe(true);input.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true,cancelable:true}));expect(pending.slice(2,4).every(read=>read.signal.aborted)).toBe(true);for(const read of pending) read.complete(Response.json({hits:[{id:'ship_587',name:'Stale',type:'ship'}]}));await vi.advanceTimersByTimeAsync(0);expect(q<HTMLElement>(root,'[data-market-results]').hidden).toBe(true);type(root,'Pending');hide();await vi.advanceTimersByTimeAsync(221);expect(pending).toHaveLength(4);expect(q<HTMLElement>(root,'.market-search-loading').hidden).toBe(true)
})
test('filter/hit boundaries preserve canonical securities and prevent empty/unsafe routes',()=>{
  const filters = readMarketFilters({region:'0',days:'30',security:'null,high'});expect(marketParams(filters).get('security')).toBe('high,null');expect(toggleMarketSecurity({...filters,security:['high']},'high')).toBeUndefined();expect(()=>readMarketFilters({region:'-1',days:'30',security:'high'})).toThrow('Invalid');expect(()=>readMarketFilters({region:'0',days:'45',security:'high'})).toThrow('Invalid');const hits = readMarketHits({hits:[{id:'item_34',name:'Tritanium',type:'item'},{id:{},name:'Bad'},{id:'item_-1',name:'Bad'}]},'item');expect(hits.map(hit=>hit.id)).toEqual([34]);expect(mergeMarketHits([hits,hits])).toHaveLength(1)
})
