// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {LocationController} from '../src/browser/system'
import {registerController} from '../src/browser/controllers/application'
import {KillPages} from '../src/browser/shared/kill-paging'

const session = vi.hoisted(()=>({read:vi.fn<()=>Promise<null>>()}))
vi.mock('../src/browser/browser-request',async importOriginal=>({...await importOriginal<typeof import('../src/browser/browser-request')>(),readBrowserSession:session.read}))
const locations = [{kind:'system',id:30002813,prefix:'/_fragments/system'},{kind:'region',id:10000042,prefix:'/_fragments/location/region'},{kind:'constellation',id:20000396,prefix:'/_fragments/location/constellation'}] as const
type Kind = typeof locations[number]['kind']
const names = ['page','dashboard','kills','battles','kill-page','kill-limit','valuable-kills','valuable-ships','valuable-structures','battle-page'] as const
type Fixture = typeof names[number]
const directory = resolve(import.meta.dirname,'fixtures/location')
const markup = new Map<string,string>(locations.flatMap(({kind})=>names.map(name=>[kind+'-'+name,readFileSync(resolve(directory,kind+'-'+name+'.html'),'utf8').replace(/<script[^>]*type="module"[^>]*>[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'')] as [string,string])))
const html = (kind:Kind,name:Fixture)=>markup.get(kind+'-'+name)!
const errors = vi.fn(),settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing location fixture '+selector);return node}
async function mount(kind:Kind='system',prepare?:(root:HTMLElement)=>void) {const where = locations.find(location=>location.kind === kind)!;history.replaceState({fixture:'retained'},'',`/${kind}/${where.id}`);document.body.innerHTML = html(kind,'page');const root = q<HTMLElement>(document.body,'.system-page');prepare?.(root);await settle();await settle();return root}
function main() {return q<HTMLElement>(document.body,'[data-system-main]')}
function valuable() {return q<HTMLElement>(document.body,'#most-valuable')}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function tab(root:HTMLElement,value:string) {q<HTMLAnchorElement>(root,`a[data-system-tab="${value}"]`).click()}
function category(root:HTMLElement,value:string) {q<HTMLButtonElement>(root,`[data-valuable-tab="${value}"]`).click()}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{
  registerController('location',LocationController).handleError = errors;errors.mockClear();session.read.mockReset();session.read.mockResolvedValue(null);document.cookie = 'ek_prefs=; Max-Age=0; Path=/'
  document.head.innerHTML = '<meta name="description" content="Before"><link rel="canonical" href="https://eve-kill.com/system/30002813">'
  fetchMock = vi.fn<typeof fetch>(async path=>{
    const url = new URL(String(path),location.href),where = locations.find(value=>url.pathname.startsWith(value.prefix+'/'))!;let name:Fixture = 'dashboard'
    if(url.pathname.includes('/tab/')) name = url.pathname.endsWith('/kills') ? 'kills' : url.pathname.endsWith('/battles') ? 'battles' : 'dashboard'
    else if(url.pathname.includes('/valuable/')) name = url.searchParams.get('category') === 'ships' ? 'valuable-ships' : url.searchParams.get('category') === 'structures' ? 'valuable-structures' : 'valuable-kills'
    else if(url.pathname.includes('/battles/')) name = 'battle-page'
    else if(url.pathname.includes('/kills/')) name = url.searchParams.get('page') === '2' ? 'kill-page' : 'kill-limit'
    return new Response(html(where.kind,name))
  });vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks();document.cookie = 'ek_prefs=; Max-Age=0; Path=/'})

test.each(locations)('$kind retains SSR hero/images/valuable without startup reads',async({kind})=>{
  const root = await mount(kind);expect(fetchMock).not.toHaveBeenCalled();expect(q<HTMLImageElement>(root,'.system-hero-background').width).toBe(1024);expect(q<HTMLImageElement>(root,'#most-valuable img').height).toBe(256);expect(main().dataset.locationKind).toBe(kind)
})
test.each(locations)('$kind tab/Back retain hero/valuable and synchronize metadata',async({kind,id,prefix})=>{
  const root = await mount(kind),hero = q(root,'.system-hero'),original = main(),value = valuable();tab(root,'kills');await settle();await settle();expect(main()).not.toBe(original);expect(q(root,'.system-hero')).toBe(hero);expect(valuable()).toBe(value);expect(fetchMock.mock.calls[0][0]).toBe(`${prefix}/tab/${id}/kills`);expect(location.pathname).toBe(`/${kind}/${id}/kills`);expect(history.state).toEqual({fixture:'retained'});expect(q(root,'a[data-system-tab="kills"]').getAttribute('aria-current')).toBe('page');expect(q<HTMLLinkElement>(document.head,'link[rel="canonical"]').href).toBe(`https://eve-kill.com/${kind}/${id}/kills`);expect(q<HTMLMetaElement>(document.head,'meta').content).not.toBe('Before');history.replaceState(history.state,'',`/${kind}/${id}`);window.dispatchEvent(new PopStateEvent('popstate'));await settle();await settle();expect(main().dataset.systemTab).toBe('dashboard');expect(q(root,'.system-hero')).toBe(hero)
})
test('returning to the displayed tab cancels another pending tab without a new read',async()=>{
  const root = await mount(),old = main(),pending = pendingRequests();tab(root,'kills');tab(root,'dashboard');expect(pending).toHaveLength(1);expect(pending[0].signal.aborted).toBe(true);expect(old.hasAttribute('aria-busy')).toBe(false);pending[0].complete(new Response(html('system','kills')));await settle();expect(main()).toBe(old);expect(location.pathname).toBe('/system/30002813')
})
test('rapid tab changes commit only the newest tab/body and selected link',async()=>{
  const root = await mount(),pending = pendingRequests();tab(root,'kills');tab(root,'battles');expect(pending[0].signal.aborted).toBe(true);pending[1].complete(new Response(html('system','battles')));await settle();pending[0].complete(new Response(html('system','kills')));await settle();expect(main().dataset.systemTab).toBe('battles');expect(q(root,'a[data-system-tab="battles"]').classList.contains('active')).toBe(true);expect(document.title).toContain('Battles')
})
test('invalid tab identity/failures retain content and offer one retry for the desired tab',async()=>{
  const root = await mount(),old = main(),pending = pendingRequests();tab(root,'kills');pending[0].complete(new Response(html('system','kills').replace('data-location-id="30002813"','data-location-id="42"')));await settle();expect(main()).toBe(old);const retry = q<HTMLButtonElement>(old,'[data-location-retry="tab"]');retry.click();retry.click();expect(pending).toHaveLength(2);pending[1].complete(new Response('Failed',{status:503}));await settle();expect(old.querySelectorAll('[data-section-error]')).toHaveLength(1);q<HTMLButtonElement>(old,'[data-location-retry="tab"]').click();pending[2].complete(new Response(html('system','kills')));await settle();expect(main().dataset.systemTab).toBe('kills')
})
test('modified/new-window/map links use normal browser navigation',async()=>{
  const root = await mount(),link = q<HTMLAnchorElement>(root,'a[data-system-tab="kills"]');root.addEventListener('click',event=>event.preventDefault());link.dispatchEvent(new MouseEvent('click',{bubbles:true,cancelable:true,metaKey:true}));link.target = '_blank';link.click();q<HTMLAnchorElement>(root,'a[aria-label="Map"]').click();expect(fetchMock).not.toHaveBeenCalled()
})
test('preferences replace history and explicit navigation wins a delayed session read',async()=>{
  document.cookie = 'ek_prefs='+encodeURIComponent(JSON.stringify({defaultTabs:{region:'battles'}}))+'; Path=/';await mount('region');await settle();expect(location.pathname).toBe('/region/10000042/battles');expect(history.state).toEqual({fixture:'retained'});let release:(value:null)=>void = ()=>{};session.read.mockImplementation(()=>new Promise(resolve=>release = resolve));document.cookie = 'ek_prefs='+encodeURIComponent(JSON.stringify({defaultTabs:{system:'battles'}}))+'; Path=/';fetchMock.mockClear();const root = await mount();tab(root,'kills');await settle();release(null);await settle();expect(location.pathname).toBe('/system/30002813/kills');expect(fetchMock).toHaveBeenCalledTimes(1)
})
test('pagehide/return resume canceled tab and category reads independently',async()=>{
  const root = await mount(),pending = pendingRequests();tab(root,'kills');category(root,'ships');hide();expect(pending.every(read=>read.signal.aborted)).toBe(true);show();expect(pending).toHaveLength(4);pending[2].complete(new Response(html('system','kills')));pending[3].complete(new Response(html('system','valuable-ships')));await settle();expect(main().dataset.systemTab).toBe('kills');expect(valuable().dataset.kind).toBe('ships');expect(main().hasAttribute('aria-busy')).toBe(false)
})
test('text completing after detach cannot replace a tab or write history',async()=>{
  const root = await mount(),old = main();let release:(text:string)=>void = ()=>{};const response = new Response('');vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>release = resolve));fetchMock.mockResolvedValue(response);tab(root,'kills');await settle();root.remove();await settle();release(html('system','kills'));await settle();expect(old.dataset.systemTab).toBe('dashboard');expect(old.hasAttribute('aria-busy')).toBe(false);expect(location.pathname).toBe('/system/30002813')
})
test.each(locations)('$kind valuable selections use their endpoint and preserve data during reads',async({kind,id,prefix})=>{
  const root = await mount(kind),old = valuable(),image = q(old,'img'),pending = pendingRequests();category(root,'ships');category(root,'structures');expect(pending[0].signal.aborted).toBe(true);expect(q(old,'img')).toBe(image);expect(pending[1].path).toBe(`${prefix}/valuable/${id}?category=structures`);pending[1].complete(new Response(html(kind,'valuable-structures')));await settle();pending[0].complete(new Response(html(kind,'valuable-ships')));await settle();expect(valuable().dataset.kind).toBe('structures')
})
test('returning to the active valuable category cancels an unfinished category',async()=>{
  const root = await mount(),old = valuable(),pending = pendingRequests();category(root,'ships');category(root,'kills');expect(pending[0].signal.aborted).toBe(true);expect(pending).toHaveLength(1);pending[0].complete(new Response(html('system','valuable-ships')));await settle();expect(valuable()).toBe(old);expect(old.dataset.kind).toBe('kills')
})
test('wrong valuable categories/failures retain cards and retry can recover',async()=>{
  const root = await mount(),old = valuable(),pending = pendingRequests();category(root,'ships');pending[0].complete(new Response(html('system','valuable-structures')));await settle();expect(valuable()).toBe(old);expect(q(old,'img')).toBeTruthy();q<HTMLButtonElement>(old,'[data-location-retry="valuable"]').click();pending[1].complete(new Response(html('system','valuable-ships')));await settle();expect(valuable().dataset.kind).toBe('ships')
})
test.each(locations)('$kind kill cursors/limit changes retain the hero and isolate paging',async({kind,id,prefix})=>{
  const root = await mount(kind);tab(root,'kills');await settle();const hero = q(root,'.system-hero'),list = q<HTMLElement>(root,'#kill-list');q<HTMLButtonElement>(list,'[data-kill-page="2"]').click();await settle();expect(fetchMock.mock.calls[1][0]).toBe(`${prefix}/kills/${id}?page=2&limit=50&after=123`);expect(q<HTMLElement>(root,'#kill-list').dataset.page).toBe('2');q<HTMLButtonElement>(root,'[data-kill-limit="10"]').click();await settle();expect(q<HTMLElement>(root,'#kill-list').dataset.page).toBe('1');expect(q<HTMLElement>(root,'#kill-list').dataset.limit).toBe('10');expect(q(root,'.system-hero')).toBe(hero)
})
test('wrong kill page and network errors retain rows with a retry for the same cursor',async()=>{
  const root = await mount();tab(root,'kills');await settle();const old = q<HTMLElement>(root,'#kill-list'),pending = pendingRequests();q<HTMLButtonElement>(old,'[data-kill-page="2"]').click();pending[0].complete(new Response(html('system','kill-limit')));await settle();expect(q(root,'#kill-list')).toBe(old);q<HTMLButtonElement>(old,'[data-location-retry="kills"]').click();expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(html('system','kill-page')));await settle();expect(q<HTMLElement>(root,'#kill-list').dataset.page).toBe('2')
})
test.each(locations)('$kind battle paging validates page/identity and preserves the hero',async({kind,id,prefix})=>{
  const root = await mount(kind);tab(root,'battles');await settle();const hero = q(root,'.system-hero');q<HTMLButtonElement>(root,'[data-conflict-page="2"]').click();await settle();expect(fetchMock.mock.calls[1][0]).toBe(`${prefix}/battles/${id}?page=2`);expect(q<HTMLElement>(root,'[data-entity-conflicts]').dataset.page).toBe('2');expect(q(root,'.system-hero')).toBe(hero)
})
test('failed battle paging retains cards and tab exit cancels the old paging request',async()=>{
  const root = await mount();tab(root,'battles');await settle();const old = q<HTMLElement>(root,'[data-entity-conflicts]'),pending = pendingRequests();q<HTMLButtonElement>(old,'[data-conflict-page="2"]').click();pending[0].complete(new Response('Failure',{status:503}));await settle();expect(q(root,'[data-entity-conflicts]')).toBe(old);q<HTMLButtonElement>(old,'[data-location-retry="battles"]').click();tab(root,'dashboard');pending[2].complete(new Response(html('system','dashboard')));await settle();expect(pending[1].signal.aborted).toBe(true);pending[1].complete(new Response(html('system','battle-page')));await settle();expect(main().dataset.systemTab).toBe('dashboard');expect(old.hasAttribute('aria-busy')).toBe(false)
})
test('chart hover highlights the same hour across series and distinguishes missing observations',async()=>{
  const root = await mount(),chart = q(root,'[data-system-chart]'),recorded = q<HTMLElement>(chart,'[data-recorded="true"][data-jumps="2000"]'),missing = q<HTMLElement>(chart,'[data-recorded="false"]');recorded.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));expect(chart.querySelectorAll('.is-hovered')).toHaveLength(4);expect(q(chart,'[data-chart-readout]').textContent).toContain('Jumps 2,000 · Ship kills 5 · NPC kills 120 · Pod kills 3');recorded.dispatchEvent(new MouseEvent('mouseout',{bubbles:true,relatedTarget:missing}));expect(chart.classList.contains('is-hovering')).toBe(true);missing.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));expect(q(chart,'[data-chart-readout]').textContent).toContain('not recorded');missing.dispatchEvent(new MouseEvent('mouseout',{bubbles:true,relatedTarget:root}));expect(chart.querySelectorAll('.is-hovered')).toHaveLength(0);expect(q(chart,'[data-chart-readout]').textContent).toBe('Hover an hour for exact figures')
})
test('shared cursor history rejects unknown pages and resets on limit changes',()=>{
  const pages = new KillPages(),list = document.createElement('section');Object.assign(list.dataset,{page:'1',nextCursor:'9007199254740993'});expect(pages.params(list,2,50)?.get('after')).toBe('9007199254740993');list.dataset.page = '2';expect(pages.params(list,2,50)?.get('after')).toBe('9007199254740993');expect(pages.params(list,4,50)).toBeNull();expect(pages.params(list,0,50)).toBeNull();expect(pages.params(list,1,999)).toBeNull();pages.clear();expect(pages.params(list,2,10)).toBeNull()
})
