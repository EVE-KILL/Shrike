// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {FitEditorController} from '../src/browser/fit-editor'
import {registerController} from '../src/browser/controllers/application'
import {encodeEditorFit} from '../src/fit/editor-codec'
import {emptyFit,type Fit} from '../src/fit/types'
import * as dogma from '../src/browser/dogma'

vi.mock('../src/browser/dogma',async()=>{
 const {createDogma} = await import('../dogma/src/core'),{readFile} = await import('node:fs/promises'),{resolve} = await import('node:path')
 const engine = createDogma({loadSdeBytes:()=>readFile(resolve(import.meta.dirname,'../dogma/dist/upstream/data/dist/sde.dat')),loadEngineBytes:()=>readFile(resolve(import.meta.dirname,'../dogma/dist/upstream/dogma-engine/esf_dogma_engine_bg.wasm'))})
 return {loadSde:vi.fn(engine.loadSde),calculateFit:vi.fn(engine.calculateFit),getHullStats:vi.fn(engine.getHullStats)}
})
const markup = readFileSync(resolve(import.meta.dirname,'fixtures/fitting/editor.html'),'utf8').replace(/<script(?![^>]*type="application\/json")[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,''),errors = vi.fn()
const settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node)throw new Error('Missing editor fixture '+selector);return node}
function click(root:HTMLElement,action:string) {q<HTMLButtonElement>(root,`[data-fit-action="${action}"]`).click()}
function input(node:HTMLInputElement|HTMLTextAreaElement,value:string) {node.value = value;node.dispatchEvent(new Event('input',{bubbles:true}))}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
async function mount(fit?:Fit) {
 history.replaceState({retained:true},'','/fits/create'+(fit ? '?fit='+encodeURIComponent(await encodeEditorFit(fit)) : ''))
 const doc = new DOMParser().parseFromString(markup,'text/html'),root = q<HTMLElement>(doc,'[data-fit-editor]'),wrapper = document.createElement('main');wrapper.append(root);document.body.append(wrapper)
 await settle();await settle();await vi.waitFor(()=>expect(q(root,'[data-fit-status]').textContent).toBe(''));await settle();return root
}
function coldMount() {const doc = new DOMParser().parseFromString(markup,'text/html'),root = q<HTMLElement>(doc,'[data-fit-editor]'),wrapper = document.createElement('main');wrapper.append(root);document.body.append(wrapper);return root}
function drop(node:HTMLElement,typeId:number,slotType:string,index?:number) {const data = new DataTransfer();data.setData('application/esf-type-id',String(typeId));data.setData('application/esf-slot-type',slotType);if(index)data.setData('application/esf-slot-index',String(index));const event = new DragEvent('drop',{bubbles:true,cancelable:true});Object.defineProperty(event,'dataTransfer',{value:data});node.dispatchEvent(event)}
async function completed(root:HTMLElement) {await vi.waitFor(()=>expect(q(root,'[data-fit-status]').textContent).toBe(''));await settle()}
function pendingReads() {const reads:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>reads.push({path:String(path),signal:options!.signal!,complete:resolve})));return reads}
beforeEach(()=>{
 registerController('fit-editor',FitEditorController).handleError = errors;errors.mockClear();localStorage.clear();document.body.style.overflow = ''
 fetchMock = vi.fn<typeof fetch>(async path=>String(path) === '/auth/me' ? Response.json({user:{characterId:1}}) : String(path).startsWith('/api/prices') ? Response.json({prices:{587:100,484:12}}) : String(path).startsWith('/_fragments/fits/variants') ? new Response('<script id="fit-variants-data" type="application/json">{"window_days":90,"families":[]}</script>') : Response.json({fit_id:'FixtureFit0001'}));vi.stubGlobal('fetch',fetchMock)
 vi.stubGlobal('ResizeObserver',class {observe(){}disconnect(){}})
})
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors).not.toHaveBeenCalled();vi.restoreAllMocks();vi.unstubAllGlobals()})
test('actual Go editor starts at hulls, loads real SDE and supports keyboard hull selection',async()=>{
 const root = await mount();expect(fetchMock).not.toHaveBeenCalled();const search = q<HTMLInputElement>(root,'[data-fit-search]');input(search,'Rifter');expect(root.querySelectorAll('[data-fit-hull]')).toHaveLength(1);search.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true,cancelable:true}));await completed(root);expect(q<HTMLImageElement>(root,'[data-fit-hull-image]').getAttribute('src')).toContain('/587/');expect(root.querySelectorAll('[data-fit-slot]')).toHaveLength(31);expect(history.state).toEqual({retained:true})
})
test('catalogue tabs have one tab stop and arrow activation labels their retained panel',async()=>{
 const root=await mount(),hulls=q<HTMLButtonElement>(root,'[data-fit-tab="hulls"]'),hardware=q<HTMLButtonElement>(root,'[data-fit-tab="hardware"]'),panel=q(root,'[data-fit-catalog]')
 expect(hulls.tabIndex).toBe(0);expect(hardware.tabIndex).toBe(-1);expect(hulls.getAttribute('aria-controls')).toBe(panel.id)
 hulls.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowRight',bubbles:true,cancelable:true}));expect(document.activeElement).toBe(hardware);expect(hardware.getAttribute('aria-selected')).toBe('true');expect(panel.getAttribute('aria-labelledby')).toBe(hardware.id)
 hardware.dispatchEvent(new KeyboardEvent('keydown',{key:'Home',bubbles:true,cancelable:true}));expect(hulls.tabIndex).toBe(0);expect(hardware.tabIndex).toBe(-1);expect(hulls.getAttribute('aria-selected')).toBe('true')
})

test('a hull query typed during a cold SDE load filters the catalogue when ready',async()=>{
 const sde = await dogma.loadSde();let release!:(value:typeof sde)=>void
 vi.mocked(dogma.loadSde).mockImplementationOnce(()=>new Promise(resolve=>release = resolve))
 history.replaceState({},'','/fits/create');const root = coldMount();await settle()
 const search = q<HTMLInputElement>(root,'[data-fit-search]');input(search,'Rifter');release(sde)
 await vi.waitFor(()=>expect(root.querySelectorAll('[data-fit-hull]')).toHaveLength(1));await completed(root)
 expect(search.value).toBe('Rifter');expect(q(root,'[data-fit-hull]').textContent).toContain('Rifter')
})

test('mobile action disclosure dismisses without changing the fit, and desktop resize exposes the same actions',async()=>{
 let mobile=true;const media=window.matchMedia.bind(window)
 vi.spyOn(window,'matchMedia').mockImplementation(query=>{const result=media(query);Object.defineProperty(result,'matches',{get:()=>mobile});return result})
 const root=await mount(emptyFit(587)),more=q<HTMLDetailsElement>(root,'[data-fit-more-actions]'),summary=q<HTMLElement>(more,'summary'),wheel=q(root,'[data-fit-slot="High:1"]')
 expect(more.open).toBe(false);more.open=true;summary.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true,cancelable:true}));expect(more.open).toBe(false);expect(document.activeElement).toBe(summary)
 more.open=true;click(root,'rename');expect(more.open).toBe(false);expect(q<HTMLDialogElement>(root,'[data-fit-rename-dialog]').open).toBe(true);click(root,'cancel-rename')
 more.open=true;q<HTMLInputElement>(root,'[data-fit-search]').click();expect(more.open).toBe(false)
 mobile=false;window.dispatchEvent(new Event('resize'));expect(more.open).toBe(true)
 mobile=true;window.dispatchEvent(new Event('resize'));expect(more.open).toBe(false);expect(q(root,'[data-fit-slot="High:1"]')).toBe(wheel)
})

test('a failed catalogue load can retry once without losing the URL or typed query',async()=>{
 vi.spyOn(console,'error').mockImplementation(()=>{});vi.mocked(dogma.loadSde).mockRejectedValueOnce(new Error('Temporary download failure'))
 history.replaceState({retained:true},'','/fits/create?keep=1');const root=coldMount();await settle();await settle()
 const error=q<HTMLElement>(root,'[data-fit-load-error]'),search=q<HTMLInputElement>(root,'[data-fit-search]');expect(error.hidden).toBe(false);input(search,'Rifter')
 const before=vi.mocked(dogma.loadSde).mock.calls.length;q<HTMLButtonElement>(root,'[data-fit-retry]').click();q<HTMLButtonElement>(root,'[data-fit-retry]').click()
 await completed(root);expect(vi.mocked(dogma.loadSde).mock.calls.length-before).toBe(1);expect(error.hidden).toBe(true);expect(search.value).toBe('Rifter');expect(root.querySelectorAll('[data-fit-hull]')).toHaveLength(1);expect(location.search).toBe('?keep=1');expect(history.state).toEqual({retained:true});expect(document.activeElement).toBe(search)
})

test('actual Go saved-fit page starts its editor and retains modules, charges, drones and cargo',async()=>{
 const saved=readFileSync(resolve(import.meta.dirname,'fixtures/fitting/saved-editor.html'),'utf8').replace(/<script(?![^>]*type="application\/json")[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'')
 history.replaceState({},'','/fit/PublicFit00001');document.body.innerHTML=saved;await settle();const root=q<HTMLElement>(document,'[data-fit-editor]')
 await vi.waitFor(()=>expect(root.querySelector('[data-fit-slot="High:1"]')).not.toBeNull());await completed(root)
 expect(q(root,'[data-fit-name]').textContent).toBe('Fixture Vexor');expect(root.querySelector('[data-fit-shortcuts]')).toBeNull();expect(q(root,'[data-fit-slot="High:1"]').getAttribute('aria-label')).toContain('125mm Gatling AutoCannon I');expect(q<HTMLImageElement>(root,'[data-fit-slot="High:1"] img[src*="/178/"]').getAttribute('src')).toContain('/178/');expect(q(root,'[data-fit-stats]').textContent).toContain('Defense')
 q<HTMLButtonElement>(root,'[data-fit-bay="drones"]').click();expect(q(root,'[data-fit-drone-state="deactivate:2454"]').textContent).toContain('3');expect(q(root,'[data-fit-drone-state="activate:2454"]').textContent).toContain('2');q<HTMLButtonElement>(root,'[data-fit-bay="cargo"]').click();expect(q(root,'[data-fit-bay-panel="cargo"]').textContent).toContain('500')
})
test('wheel slot selection, module fitting, table view and Cmd+Z/redo preserve the document',async()=>{
 const root = await mount(emptyFit(587));q(root,'[data-fit-slot="High:1"]').dispatchEvent(new MouseEvent('click',{bubbles:true}));const search = q<HTMLInputElement>(root,'[data-fit-search]');input(search,'125mm Gatling AutoCannon I');search.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true,cancelable:true}));await completed(root);expect(q(root,'[data-fit-slot="High:1"] img').getAttribute('src')).toContain('/484/');q<HTMLButtonElement>(root,'[data-fit-view="table"]').click();expect(root.querySelector('.fit-editor-table')).not.toBeNull();root.dispatchEvent(new KeyboardEvent('keydown',{key:'z',metaKey:true,bubbles:true,cancelable:true}));await completed(root);expect(q(root,'[data-fit-slot="High:1"]').textContent).not.toContain('125mm Gatling AutoCannon I');click(root,'redo');await completed(root);expect(q(root,'[data-fit-slot="High:1"]').textContent).toContain('125mm Gatling AutoCannon I')
})
test('fullscreen retains the same editor/history and restores body scrolling on detach',async()=>{
 document.body.style.overflow = 'auto';const root = await mount(emptyFit(587)),search = q<HTMLInputElement>(root,'[data-fit-search]');click(root,'expand');await settle();expect(root.classList.contains('expanded')).toBe(true);expect(document.body.style.overflow).toBe('hidden');expect(q(root,'[data-fit-search]')).toBe(search);click(root,'expand');await settle();expect(root.classList.contains('expanded')).toBe(false);expect(document.body.style.overflow).toBe('auto');click(root,'expand');await settle();root.remove();await settle();expect(document.body.style.overflow).toBe('auto')
})
test('detached editor aborts prices/families and ignores late results',async()=>{
 const root = await mount(emptyFit(587)),reads = pendingReads();click(root,'expand');await settle();expect(reads.some(read=>read.path.includes('variants'))).toBe(true);root.remove();await settle();expect(reads.every(read=>read.signal.aborted)).toBe(true);for(const read of reads)read.complete(new Response('<script id="fit-variants-data">{"window_days":90,"families":[]}</script>'));await settle();expect(root.querySelector('.fit-editor-context-menu')).toBeNull();expect(root.hasAttribute('aria-busy')).toBe(false)
})
test('saving is single flight and does not replace newer unsaved fitting changes',async()=>{
 const root = await mount(emptyFit(587));click(root,'save');await vi.waitFor(()=>expect(q<HTMLDialogElement>(root,'[data-fit-save-dialog]').open).toBe(true));const reads = pendingReads(),form = q<HTMLFormElement>(root,'[data-fit-save-form]');input(q<HTMLInputElement>(form,'[name="name"]'),'Submitted name');form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));expect(reads).toHaveLength(1);click(root,'rename');const rename = q<HTMLFormElement>(root,'[data-fit-rename-form]');input(q<HTMLInputElement>(rename,'[name="name"]'),'Newer draft');rename.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));reads[0].complete(Response.json({fit_id:'FixtureFit0001'}));await settle();expect(q(root,'[data-fit-name]').textContent).toBe('Newer draft');expect(q(root,'[data-fit-toast]').textContent).toContain('newer changes remain unsaved')
})
test('pagehide cancels pending reads and cached return resumes without rebuilding the workspace',async()=>{
 const root = await mount(emptyFit(587)),reads = pendingReads();click(root,'expand');await settle();hide();expect(reads.every(read=>read.signal.aborted)).toBe(true);for(const read of reads)read.complete(new Response('late'));await settle();show();await settle();expect(root.classList.contains('expanded')).toBe(true);expect(reads.some(read=>!read.signal.aborted)).toBe(true);for(const read of reads.filter(read=>!read.signal.aborted))read.complete(read.path.includes('variants') ? new Response('<script id="fit-variants-data">{"window_days":90,"families":[]}</script>') : Response.json({prices:{587:100}}));await completed(root);expect(root.querySelectorAll('.fit-editor-context-menu')).toHaveLength(1)
})
test('module drops fit a slot, move between slots and auto-fit onto the ship image',async()=>{
 const root = await mount(emptyFit(587));drop(q(root,'[data-fit-slot="High:1"]'),484,'High');await completed(root);expect(q(root,'[data-fit-slot="High:1"] img').getAttribute('src')).toContain('/484/');drop(q(root,'[data-fit-slot="High:2"]'),484,'High',1);await completed(root);expect(q(root,'[data-fit-slot="High:1"]').getAttribute('data-hasitem')).toBe('false');expect(q(root,'[data-fit-slot="High:2"] img').getAttribute('src')).toContain('/484/');drop(q(root,'[data-fit-hull-drop]'),484,'High');await completed(root);expect(root.querySelectorAll('.slotOuter[data-hasitem="true"]')).toHaveLength(2);drop(q(root,'[data-fit-hull-drop]'),178,'Charge');await settle();expect(root.querySelectorAll('.slotOuter[data-hasitem="true"]')).toHaveLength(2)
})
test('module context menu loads a charge, changes state, fills the rack and un-fits',async()=>{
 const fit = emptyFit(587);fit.modules = [{typeId:484,slot:{type:'High',index:1},state:'Active'}];const root = await mount(fit)
 const menu = ()=>{q(root,'[data-fit-slot="High:1"]').dispatchEvent(new MouseEvent('contextmenu',{bubbles:true,cancelable:true,clientX:120,clientY:120}));return q(root,'.fit-editor-context-menu')}
 q<HTMLButtonElement>(menu(),'[data-fit-context-action="charge:178"]').click();await completed(root);expect(q(root,'[data-fit-slot="High:1"] img').getAttribute('src')).toContain('/178/');q<HTMLButtonElement>(menu(),'[data-fit-context-action="state:Overload"]').click();await completed(root);expect(q<HTMLButtonElement>(menu(),'[data-fit-context-action="state:Overload"]').classList.contains('current')).toBe(true);q<HTMLButtonElement>(menu(),'[data-fit-context-action="fill-rack"]').click();await completed(root);expect(root.querySelectorAll('.slotOuter[data-hasitem="true"]')).toHaveLength(3);q<HTMLButtonElement>(menu(),'[data-fit-context-action="unfit"]').click();await completed(root);expect(q(root,'[data-fit-slot="High:1"]').getAttribute('data-hasitem')).toBe('false')
})
test('cold Paste reads the clipboard during the click and imports once SDE is ready',async()=>{
 const sde = await dogma.loadSde();let release!:(value:typeof sde)=>void;vi.mocked(dogma.loadSde).mockImplementationOnce(()=>new Promise(resolve=>release = resolve));const clipboard = vi.spyOn(navigator.clipboard,'readText').mockResolvedValue('[Rifter, Clipboard fitting]\n125mm Gatling AutoCannon I\n');history.replaceState({},'','/fits/create');const root = coldMount();await settle();click(root,'paste');expect(clipboard).toHaveBeenCalledTimes(1);release(sde);await vi.waitFor(()=>expect(q(root,'[data-fit-name]').textContent).toBe('Clipboard fitting'));await completed(root);expect(q(root,'[data-fit-slot="High:1"] img').getAttribute('src')).toContain('/484/')
})
test('leaving during a cold clipboard/SDE load cannot later commit or navigate',async()=>{
 const sde = await dogma.loadSde();let release!:(value:typeof sde)=>void;vi.mocked(dogma.loadSde).mockImplementationOnce(()=>new Promise(resolve=>release = resolve));vi.spyOn(navigator.clipboard,'readText').mockResolvedValue('[Rifter, Late import]\n');history.replaceState({},'','/fits/create');const root = coldMount();await settle();click(root,'paste');hide();release(sde);await settle();expect(q(root,'[data-fit-name]').textContent).toBe('');expect(location.search).toBe('');show();await completed(root);expect(q(root,'[data-fit-name]').textContent).toBe('')
})
test('failed new saves retain the form, expose an error and do not auto-replay on return',async()=>{
 const root = await mount(emptyFit(587));click(root,'save');await vi.waitFor(()=>expect(q<HTMLDialogElement>(root,'[data-fit-save-dialog]').open).toBe(true));fetchMock.mockRejectedValue(new Error('Connection dropped'));const form = q<HTMLFormElement>(root,'[data-fit-save-form]');input(q<HTMLInputElement>(form,'[name="name"]'),'Retained save');form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));await settle();expect(q(root,'[data-fit-save-error]').textContent).toContain('could not be confirmed');expect(q<HTMLInputElement>(form,'[name="name"]').value).toBe('Retained save');expect(q<HTMLButtonElement>(form,'[type="submit"]').disabled).toBe(true);const count = fetchMock.mock.calls.filter(call=>call[1]?.method === 'POST').length;hide();show();await settle();expect(fetchMock.mock.calls.filter(call=>call[1]?.method === 'POST')).toHaveLength(count)
})
test('new text entered during a successful save remains in the open form',async()=>{
 const root = await mount(emptyFit(587));click(root,'save');await vi.waitFor(()=>expect(q<HTMLDialogElement>(root,'[data-fit-save-dialog]').open).toBe(true));const form = q<HTMLFormElement>(root,'[data-fit-save-form]'),reads = pendingReads();input(q<HTMLInputElement>(form,'[name="name"]'),'Submitted');form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));input(q<HTMLInputElement>(form,'[name="name"]'),'New form draft');reads[0].complete(Response.json({fit_id:'FixtureFit0001'}));await settle();expect(q<HTMLDialogElement>(root,'[data-fit-save-dialog]').open).toBe(true);expect(q<HTMLInputElement>(form,'[name="name"]').value).toBe('New form draft');expect(q(root,'[data-fit-save-label]').textContent).toBe('Update')
})

test('populated Go observed fits preview, reset modifications and preserve the custom workspace',async()=>{
 const original=fetchMock.getMockImplementation()!,variants=readFileSync(resolve(import.meta.dirname,'fixtures/fitting/variants.html'),'utf8')
 fetchMock.mockImplementation((path,options)=>String(path).startsWith('/_fragments/fits/variants')?Promise.resolve(new Response(variants)):original(path,options))
 const fit=emptyFit(587);fit.name='Custom workspace';fit.modules=[{typeId:484,slot:{type:'High',index:1},state:'Active'}]
 const root=await mount(fit);click(root,'expand');await vi.waitFor(()=>expect(root.querySelector('[data-fit-variant="fixture-rifter-autocannon"]')).not.toBeNull())
 q<HTMLAnchorElement>(root,'[data-fit-variant="fixture-rifter-autocannon"]').click();await vi.waitFor(()=>expect(q(root,'[data-fit-name]').textContent).toBe('Rifter Observed Fit #1'));await completed(root)
 expect(q(root,'.fit-editor-comparison').textContent).toContain('+2 / −0 modules')
 q<HTMLButtonElement>(root,'[data-fit-view="table"]').click();const low=q<HTMLTableRowElement>(root,'tr[data-fit-slot="Low:1"]');low.click();low.dispatchEvent(new KeyboardEvent('keydown',{key:'Delete',bubbles:true,cancelable:true}));await completed(root)
 expect(q(root,'.fit-editor-comparison').textContent).toContain('This observed fit has been modified.')
 q<HTMLButtonElement>(root,'[data-fit-variant-action="reset-preview"]').click();await completed(root);expect(q(root,'.fit-editor-comparison').textContent).not.toContain('This observed fit has been modified.')
 q<HTMLButtonElement>(root,'[data-fit-variant-custom]').click();await completed(root);expect(q(root,'[data-fit-name]').textContent).toBe('Custom workspace');expect(root.querySelector('.fit-editor-comparison')).toBeNull();expect(q(root,'tr[data-fit-slot="High:1"]').textContent).toContain('125mm Gatling AutoCannon I')
})
