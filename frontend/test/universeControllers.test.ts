// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {UniverseController} from '../src/browser/universe'
import {ItemFitsController} from '../src/browser/controllers/item-fits'
import {registerController} from '../src/browser/controllers/application'
import {parseFitNumber,sortForFitMetric} from '../src/browser/shared/fit-filters'

const session = vi.hoisted(()=>({read:vi.fn<()=>Promise<null>>()}))
vi.mock('../src/browser/browser-request',async importOriginal=>({...await importOriginal<typeof import('../src/browser/browser-request')>(),readBrowserSession:session.read}))
const directory = resolve(import.meta.dirname,'fixtures/universe')
const names = ['item','group','fittings','dashboard','kills','group-kills','group-dashboard','kill-page','kill-limit','fits-panel','fits-filtered','fit-meta','fit-meta-filtered','fit-distributions','fit-distributions-filtered'] as const
const markup = Object.fromEntries(names.map(name=>[name,readFileSync(resolve(directory,name+'.html'),'utf8').replace(/<script[^>]*type="module"[^>]*>[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'')]))
const errors = vi.fn(),settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing universe fixture '+selector);return node}
async function mount(name:typeof names[number]='item',prepare?:(root:HTMLElement)=>void) {document.body.innerHTML = markup[name];const root = q<HTMLElement>(document.body,':scope > *');// happy-dom misapplies parsed option[selected]; restore the browser's SSR selection.
  for(const control of root.querySelectorAll<HTMLSelectElement>('select')) {const selected = control.querySelector<HTMLOptionElement>('option[selected]');if(selected) control.value = selected.value}
  prepare?.(root);await settle();await settle();return root}
function main() {return q<HTMLElement>(document.body,'.universe-main')}
function fits() {return q<HTMLElement>(document.body,'[data-item-fits]')}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function fitResponse(path:string) {
  const url = new URL(path,location.href),query = new URLSearchParams(url.search),sort = query.get('sort') ?? 'observed';query.delete('sort');query.sort()
  return markup[query.has('min_ehp') ? 'fits-filtered' : 'fits-panel'].replace(/data-fit-query="[^"]*"/,'data-fit-query="'+query.toString()+'"').replace(/data-fit-sort="[^"]*"/,'data-fit-sort="'+sort+'"')
}
function supportResponse(path:string) {const url = new URL(path,location.href),query = new URLSearchParams(url.search);query.delete('sort');query.sort();const segment = url.pathname.includes('fit-meta') ? 'fit-meta' : 'fit-distributions';return markup[segment].replace(/data-fit-query="[^"]*"/,'data-fit-query="'+query.toString()+'"')}
function change(root:HTMLElement,selector:string,value:string) {const field = q<HTMLSelectElement>(root,selector);field.value = value;field.dispatchEvent(new Event('change',{bubbles:true}));return field}
function submit(root:HTMLElement,min:string,max='',metric='ehp') {const form = q<HTMLFormElement>(root,'[data-fit-filter-form]');q<HTMLSelectElement>(form,'[name="metric"]').value = metric;q<HTMLInputElement>(form,'[name="min"]').value = min;q<HTMLInputElement>(form,'[name="max"]').value = max;form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));return form}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{
  for(const [name,controller] of [['universe',UniverseController],['item-fits',ItemFitsController]] as const) registerController(name,controller).handleError = errors
  errors.mockClear();session.read.mockReset();session.read.mockResolvedValue(null);document.cookie = 'ek_prefs=; Max-Age=0; Path=/'
  history.replaceState({fixture:'retained'},'','/item/11186');document.head.innerHTML = '<meta name="description" content="Before"><link rel="canonical" href="https://eve-kill.com/item/11186">'
  fetchMock = vi.fn<typeof fetch>(async path=>{
    const url = String(path)
    if(url.includes('/fit-meta/') || url.includes('/fit-distributions/')) return new Response(supportResponse(url))
    if(url.includes('/item/fittings/')) return new Response(fitResponse(url))
    if(url.includes('/group/tab/')) return new Response(markup[url.includes('/kills') ? 'group-kills' : 'group-dashboard'])
    if(url.includes('/item/tab/')) return new Response(markup[url.includes('/kills') ? 'kills' : 'dashboard'])
    return new Response(markup[url.includes('page=2') ? 'kill-page' : 'kill-limit'])
  });vi.stubGlobal('fetch',fetchMock)
})
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks();document.cookie = 'ek_prefs=; Max-Age=0; Path=/'})

test('item/group SSR retain hero images and dashboard without fragment startup reads',async()=>{
  let root = await mount();expect(fetchMock).not.toHaveBeenCalled();expect(q<HTMLImageElement>(root,'.item-portrait img').width).toBe(160);history.replaceState(history.state,'','/group/25');root = await mount('group');expect(fetchMock).not.toHaveBeenCalled();expect(q(root,'.group-type-grid').textContent).toContain('Test Ship')
})
test('tab navigation retains header, changes only main and synchronizes metadata/history',async()=>{
  const root = await mount(),hero = q(root,'.item-hero'),old = main();q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();await settle();await settle();expect(q(root,'.item-hero')).toBe(hero);expect(main()).not.toBe(old);expect(location.pathname).toBe('/item/11186/kills');expect(history.state.fixture).toBe('retained');expect(q(root,'.universe-tabs a.active').getAttribute('aria-current')).toBe('page');expect(document.title).toContain('Test Item');expect(q<HTMLLinkElement>(document.head,'link[rel="canonical"]').href).toBe('https://eve-kill.com/item/11186/kills');expect(q<HTMLMetaElement>(document.head,'meta[name="description"]').content).not.toBe('Before')
})
test('group tabs use the group endpoint and Back retains the persistent hull strip',async()=>{
  history.replaceState(history.state,'','/group/25');const root = await mount('group'),strip = q(root,'.group-hull-strip');q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();await settle();await settle();expect(fetchMock.mock.calls[0][0]).toBe('/_fragments/group/tab/25/kills');history.replaceState(history.state,'','/group/25');window.dispatchEvent(new PopStateEvent('popstate'));await settle();await settle();expect(main().dataset.universeTab).toBe('dashboard');expect(q(root,'.group-hull-strip')).toBe(strip)
})
test('modified/external/new-window links keep ordinary browser navigation',async()=>{
  const root = await mount(),link = q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]');root.addEventListener('click',event=>event.preventDefault());link.dispatchEvent(new MouseEvent('click',{bubbles:true,cancelable:true,ctrlKey:true}));link.target = '_blank';link.dispatchEvent(new MouseEvent('click',{bubbles:true,cancelable:true}));expect(fetchMock).not.toHaveBeenCalled()
})
test('new tab/Back intent cancels an old read and rejects its stale response',async()=>{
  const root = await mount(),pending = pendingRequests();q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();history.replaceState(history.state,'','/item/11186');window.dispatchEvent(new PopStateEvent('popstate'));expect(pending[0].signal.aborted).toBe(true);pending[1].complete(new Response(markup.dashboard));await settle();await settle();pending[0].complete(new Response(markup.kills));await settle();expect(main().dataset.universeTab).toBe('dashboard');expect(location.pathname).toBe('/item/11186')
})
test('wrong identity/errors retain current tab and retry only the desired navigation',async()=>{
  const root = await mount(),old = main(),pending = pendingRequests();q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();pending[0].complete(new Response(markup.kills.replace('data-universe-id="11186"','data-universe-id="42"')));await settle();expect(main()).toBe(old);expect(location.pathname).toBe('/item/11186');const retry = q<HTMLButtonElement>(old,'[data-universe-retry]');retry.click();retry.click();expect(pending).toHaveLength(2);pending[1].complete(new Response(markup.kills));await settle();await settle();expect(main().dataset.universeTab).toBe('kills')
})
test('preferred tab replaces history after the shared session read',async()=>{
  document.cookie = 'ek_prefs='+encodeURIComponent(JSON.stringify({defaultTabs:{item:'kills'}}))+'; Path=/';await mount();await settle();await settle();expect(location.pathname).toBe('/item/11186/kills');expect(history.state.fixture).toBe('retained');expect(fetchMock.mock.calls[0][0]).toBe('/_fragments/item/tab/11186/kills')
})
test('late preferences cannot override a manually selected tab or detached root',async()=>{
  let complete:(value:null)=>void = ()=>{};session.read.mockImplementation(()=>new Promise(resolve=>complete = resolve));document.cookie = 'ek_prefs='+encodeURIComponent(JSON.stringify({defaultTabs:{item:'fittings'}}))+'; Path=/';const root = await mount();q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();await settle();await settle();complete(null);await settle();expect(location.pathname).toBe('/item/11186/kills');expect(fetchMock).toHaveBeenCalledTimes(1);root.remove();await settle();expect(document.querySelector('dialog')).toBeNull()
})
test('image dialog uses the larger source and cleans up on close/pagehide/detach',async()=>{
  const root = await mount();q<HTMLButtonElement>(root,'[data-item-image]').click();let dialog = q<HTMLDialogElement>(document.body,'dialog');expect(q<HTMLImageElement>(dialog,'img').width).toBe(512);expect(q<HTMLImageElement>(dialog,'img').src).toContain('size=512');q<HTMLButtonElement>(dialog,'button').click();await settle();expect(dialog.isConnected).toBe(false);q<HTMLButtonElement>(root,'[data-item-image]').click();hide();expect(document.querySelector('dialog')).toBeNull();show();q<HTMLButtonElement>(root,'[data-item-image]').click();root.remove();await settle();expect(document.querySelector('dialog')).toBeNull()
})
test('tab pagehide/return resumes its desired read and clears busy state',async()=>{
  const root = await mount(),pending = pendingRequests();q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();hide();expect(pending[0].signal.aborted).toBe(true);expect(main().hasAttribute('aria-busy')).toBe(false);show();expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(markup.kills));await settle();await settle();expect(main().dataset.universeTab).toBe('kills')
})
test('a detached tab scope rejects text that completes after headers and cannot push history',async()=>{
  const root = await mount(),old = main();let release:(text:string)=>void = ()=>{};const response = new Response('');vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>release = resolve));fetchMock.mockResolvedValue(response);q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();await settle();root.remove();await settle();release(markup.kills);await settle();expect(old.dataset.universeTab).toBe('dashboard');expect(old.hasAttribute('aria-busy')).toBe(false);expect(location.pathname).toBe('/item/11186')
})
test('a preference read completing after scope removal starts no navigation',async()=>{
  let release:(value:null)=>void = ()=>{};session.read.mockImplementation(()=>new Promise(resolve=>release = resolve));document.cookie = 'ek_prefs='+encodeURIComponent(JSON.stringify({defaultTabs:{item:'kills'}}))+'; Path=/';const root = await mount();root.remove();await settle();release(null);await settle();expect(fetchMock).not.toHaveBeenCalled();expect(location.pathname).toBe('/item/11186')
})
test('kill paging stores cursors and changing limit resets pagination without replacing hero',async()=>{
  const root = await mount();q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();await settle();await settle();const hero = q(root,'.item-hero'),list = q<HTMLElement>(root,'#kill-list');list.dataset.nextCursor = '123';q<HTMLButtonElement>(list,'[data-kill-page="2"]').click();await settle();await settle();expect(fetchMock.mock.calls[1][0]).toBe('/_fragments/item/kills/11186?page=2&limit=50&after=123');expect(q<HTMLElement>(root,'#kill-list').dataset.page).toBe('2');q<HTMLButtonElement>(root,'[data-kill-limit="10"]').click();await settle();await settle();expect(q<HTMLElement>(root,'#kill-list').dataset.limit).toBe('10');expect(q(root,'.item-hero')).toBe(hero)
})
test('failed kill paging retains data and retries while a detached tab cannot commit',async()=>{
  const root = await mount();q<HTMLAnchorElement>(root,'.universe-tabs a[href$="/kills"]').click();await settle();await settle();const old = q<HTMLElement>(root,'#kill-list'),pending = pendingRequests();q<HTMLButtonElement>(old,'[data-kill-limit="10"]').click();pending[0].complete(new Response('Failure',{status:503}));await settle();expect(q(root,'#kill-list')).toBe(old);q<HTMLButtonElement>(old,'[data-universe-retry]').click();root.remove();await settle();expect(pending[1].signal.aborted).toBe(true);pending[1].complete(new Response(markup['kill-limit']));await settle();expect(old.dataset.limit).toBe('50');expect(old.hasAttribute('aria-busy')).toBe(false)
})
test('fitting supporting fragments load independently and sort-only refresh keeps exact nodes',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');await mount('fittings');expect(fetchMock.mock.calls.map(call=>String(call[0]))).toEqual(['/_fragments/item/fit-meta/11186?','/_fragments/item/fit-distributions/11186?']);const old = fits(),meta = q(old,'[data-fit-meta]'),distribution = q(old,'[data-fit-distributions]');change(old,'[data-fit-sort]','ehp');await settle();await settle();expect(fits()).not.toBe(old);expect(q(fits(),'[data-fit-meta]')).toBe(meta);expect(q(fits(),'[data-fit-distributions]')).toBe(distribution);expect(fetchMock).toHaveBeenCalledTimes(3);expect(location.search).toBe('?sort=ehp')
})
test('fit filters combine pending groups/ranges; newer drafts and expanded families survive replacement',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');await mount('fittings');const old = fits(),pending = pendingRequests();q<HTMLButtonElement>(old,'[data-fit-group="52"]').click();submit(old,'3k');const form = q<HTMLFormElement>(old,'form');form.hidden = false;q<HTMLInputElement>(form,'[name="min"]').value = 'new draft';q<HTMLDetailsElement>(old,'[data-fit-family] details').open = true;expect(pending[0].signal.aborted).toBe(true);expect(new URL(pending[1].path,location.href).searchParams.get('groups')).toBe('52');pending[1].complete(new Response(markup['fits-filtered']));await settle();await settle();expect(q<HTMLInputElement>(fits(),'[name="min"]').value).toBe('new draft');expect(q<HTMLFormElement>(fits(),'form').hidden).toBe(false);expect(q<HTMLDetailsElement>(fits(),'[data-fit-family] details').open).toBe(true);expect(q(fits(),'[data-fit-group-chip]').textContent).toContain('Requires Warp Scrambler');pending[0].complete(new Response(fitResponse(pending[0].path)));await settle();expect(fits().dataset.fitQuery).toBe('groups=52&min_ehp=3000');hide()
})
test('fit form shorthand/validation and repair/NPC sorts retain the original query behavior',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');await mount('fittings');const pending = pendingRequests(),form = submit(fits(),'-1');expect(q<HTMLInputElement>(form,'[name="min"]').validationMessage).toContain('non-negative');expect(pending).toHaveLength(0);q<HTMLInputElement>(form,'[name="min"]').dispatchEvent(new InputEvent('input',{bubbles:true}));expect(q<HTMLInputElement>(form,'[name="min"]').validationMessage).toBe('');submit(fits(),'50k','1m','shield_repair');let query = new URL(pending[0].path,location.href).searchParams;expect(query.get('min_shield_repair')).toBe('50000');expect(query.get('max_shield_repair')).toBe('1000000');expect(query.get('sort')).toBe('repair');change(fits(),'[data-fit-metric]','npc_ehp');expect(q<HTMLElement>(fits(),'.item-fit-npc').hidden).toBe(false);query = new URL(pending[1].path,location.href).searchParams;expect(query.get('npc_profile')).toBe('guristas');expect(query.get('sort')).toBe('npc_ehp');hide()
})
test('histograms, chips, group reset and clear all keep sort while combining pending selections',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');await mount('fittings');const pending = pendingRequests();q<HTMLButtonElement>(fits(),'[data-fit-bucket-metric]').click();q<HTMLButtonElement>(fits(),'[data-fit-group="52"]').click();expect(new URL(pending[1].path,location.href).searchParams.get('min_ehp')).toBe('3000');pending[1].complete(new Response(fitResponse(pending[1].path)));await settle();await settle();q<HTMLButtonElement>(fits(),'[data-fit-remove-metric="ehp"]').click();let query = new URL(pending.at(-1)!.path,location.href).searchParams;expect(query.get('min_ehp')).toBeNull();expect(query.get('groups')).toBe('52');const clear = document.createElement('button');clear.dataset.fitClear = '';fits().append(clear);clear.click();query = new URL(pending.at(-1)!.path,location.href).searchParams;expect([...query]).toEqual([]);hide()
})
test('fit errors/wrong IDs retain results and retry preserves desired parameters',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');await mount('fittings');const old = fits(),pending = pendingRequests();submit(old,'3k');pending[0].complete(new Response(fitResponse(pending[0].path).replace('data-fit-type-id="11186"','data-fit-type-id="42"')));await settle();expect(fits()).toBe(old);q<HTMLButtonElement>(old,'[data-item-fits-retry="fits"]').click();expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(fitResponse(pending[1].path)));await settle();await settle();expect(fits()).not.toBe(old);hide()
})
test('supporting errors have independent retry and validate query/identity markers',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');const pending = pendingRequests();await mount('fittings');const loading = q<HTMLElement>(fits(),'[data-fit-meta] .item-fit-loading');expect(loading.hidden).toBe(false);pending[0].complete(new Response(markup['fit-meta'].replace('data-fit-type-id="11186"','data-fit-type-id="42"')));pending[1].complete(new Response(markup['fit-distributions']));await settle();expect(loading.hidden).toBe(true);const retry = q<HTMLButtonElement>(fits(),'[data-item-fits-retry="fit-meta"]');retry.click();expect(loading.hidden).toBe(false);retry.click();expect(pending).toHaveLength(3);pending[2].complete(new Response(markup['fit-meta']));await settle();expect(q(fits(),'[data-fit-meta]').textContent).toContain('Warp Scrambler');expect(fits().querySelector('[data-fit-meta] .item-fit-loading')).toBeNull();expect(q(fits(),'[data-fit-distributions]').textContent).toContain('Effective Hitpoints')
})
test('leaving fittings aborts child reads and late responses cannot change tab/history',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');const root = await mount('fittings'),pending = pendingRequests(),old = fits();submit(old,'3k');q<HTMLAnchorElement>(root,'.universe-tabs a[href="/item/11186"]').click();pending[1].complete(new Response(markup.dashboard));await settle();await settle();expect(pending[0].signal.aborted).toBe(true);pending[0].complete(new Response(fitResponse(pending[0].path)));await settle();expect(main().dataset.universeTab).toBe('dashboard');expect(location.search).toBe('');expect(old.hasAttribute('aria-busy')).toBe(false)
})
test('Back changing URL before a delayed fit body completes cannot write stale filters',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');await mount('fittings');const pending = pendingRequests();submit(fits(),'3k');history.replaceState(history.state,'','/item/11186');window.dispatchEvent(new PopStateEvent('popstate'));pending[0].complete(new Response(fitResponse(pending[0].path)));await settle();expect(location.search).toBe('');expect(main().dataset.universeTab).toBe('fittings');pending[1].complete(new Response(markup.dashboard));await settle();await settle();expect(main().dataset.universeTab).toBe('dashboard')
})
test('fit page return resumes canceled selection and supporting reads without stale commits',async()=>{
  history.replaceState(history.state,'','/item/11186/fittings');await mount('fittings');const pending = pendingRequests();submit(fits(),'3k');hide();show();expect(pending[0].signal.aborted).toBe(true);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(fitResponse(pending[1].path)));await settle();await settle();expect(fits().dataset.fitQuery).toBe('min_ehp=3000');hide()
})
test('shared shorthand numbers reject negatives/nonfinite values and preserve zero/repair sorts',()=>{
  expect(['0','12.5','50k','1,000m','2b'].map(parseFitNumber)).toEqual([0,12.5,50000,1e9,2e9]);expect(['','-1','1e9','NaN','Infinity'].map(parseFitNumber)).toEqual([null,null,null,null,null]);expect(sortForFitMetric('passive_shield')).toBe('repair');expect(sortForFitMetric('npc_ehp')).toBe('npc_ehp')
})
