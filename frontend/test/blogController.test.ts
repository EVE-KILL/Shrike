// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {BlogController,BlogDatesController} from '../src/browser/blog'
import {registerController} from '../src/browser/controllers/application'

const directory = resolve(import.meta.dirname,'fixtures/blog'),names = ['listing','tagged','detail','cards'] as const
const markup = Object.fromEntries(names.map(name=>[name,readFileSync(resolve(directory,name+'.html'),'utf8').replace(/<script[^>]*type="module"[^>]*>[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'')]))
const errors = vi.fn(),settle = ()=>new Promise<void>(resolve=>setTimeout(resolve,0))
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing blog fixture '+selector);return node}
async function mount(name:typeof names[number]='listing',prepare?:(root:HTMLElement)=>void) {document.body.innerHTML = markup[name];const root = q<HTMLElement>(document.body,':scope > *');prepare?.(root);await settle();await settle();return root}
function pendingRequests() {const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}> = [];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
function more(root:HTMLElement) {q<HTMLButtonElement>(root,'[data-blog-more]').click()}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{for(const [name,controller] of [['blog',BlogController],['blog-dates',BlogDatesController]] as const) registerController(name,controller).handleError = errors;errors.mockClear();history.replaceState({},'','/blog');fetchMock = vi.fn<typeof fetch>(async()=>new Response(markup.cards));vi.stubGlobal('fetch',fetchMock)})
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('SSR cards/detail localize dates once without an initial request',async()=>{
  let root = await mount(),date = q<HTMLTimeElement>(root,'time'),expected = new Date(date.dateTime).toLocaleDateString(undefined,{year:'numeric',month:'short',day:'numeric'});expect(date.textContent).toBe(expected);expect(date.dataset.localized).toBe('true');expect(fetchMock).not.toHaveBeenCalled();root = await mount('detail');date = q(root,'time');expected = new Date(date.dateTime).toLocaleDateString(undefined,{year:'numeric',month:'long',day:'numeric'});expect(date.textContent).toBe(expected);expect(fetchMock).not.toHaveBeenCalled()
})
test('tagged cursor pagination retains cards/covers and localizes appended dates',async()=>{
  const root = await mount('tagged'),original = q(root,'.blog-card'),cover = q(original,'img'),pending = pendingRequests(),button = q<HTMLButtonElement>(root,'[data-blog-more]');more(root);more(root);expect(pending).toHaveLength(1);expect(button.disabled).toBe(true);expect(button.textContent).toBe('Loading...');expect(q(root,'[data-blog-list]').getAttribute('aria-busy')).toBe('true');expect(q(original,'img')).toBe(cover);const query = new URL(pending[0].path,location.href).searchParams;expect(query.get('tag')).toBe('ships');expect(query.get('cursor')).toBe('2026-09-24T12:00:00.000Z');pending[0].complete(new Response(markup.cards));await settle();await settle();expect(q(root,'.blog-card')).toBe(original);expect(root.querySelectorAll('.blog-card')).toHaveLength(2);expect(q<HTMLTimeElement>(root,'.blog-card[href$="second-post"] time').dataset.localized).toBe('true');expect(button.disabled).toBe(false);expect(q(root,'.blog-more').hidden).toBe(true)
})
test('next cursor advances and duplicate cards do not replace or repeat earlier DOM',async()=>{
  const root = await mount(),original = q(root,'.blog-card');fetchMock.mockResolvedValueOnce(new Response(markup.cards,{headers:{'X-Next-Cursor':'next-page'}}));more(root);await settle();await settle();const second = q(root,'.blog-card[href$="second-post"]');expect(q(root,'.blog-more').hidden).toBe(false);more(root);await settle();await settle();expect(new URL(String(fetchMock.mock.calls[1][0]),location.href).searchParams.get('cursor')).toBe('next-page');expect(root.querySelectorAll('.blog-card')).toHaveLength(2);expect(q(root,'.blog-card')).toBe(original);expect(q(root,'.blog-card[href$="second-post"]')).toBe(second)
})
test('pagination errors retain cards/cursor and the same button retries',async()=>{
  const root = await mount(),original = q(root,'.blog-card'),pending = pendingRequests();more(root);pending[0].complete(new Response('Failed',{status:503}));await settle();expect(q(root,'.blog-card')).toBe(original);expect(q(root,'[data-blog-error]').textContent).toBe('Could not load more posts.');expect(q(root,'[data-blog-error]').hidden).toBe(false);more(root);expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(markup.cards));await settle();expect(q(root,'[data-blog-error]').hidden).toBe(true);expect(root.querySelectorAll('.blog-card')).toHaveLength(2)
})
test('unexpected HTML cannot append a full page and leaves pagination retryable',async()=>{
  const root = await mount(),pending = pendingRequests();more(root);pending[0].complete(new Response('<h1>Unexpected renderer</h1>'));await settle();expect(root.querySelectorAll('.blog-card')).toHaveLength(1);expect(root.querySelector('h1')?.textContent).not.toBe('Unexpected renderer');expect(q(root,'[data-blog-error]').hidden).toBe(false);expect(q<HTMLButtonElement>(root,'[data-blog-more]').disabled).toBe(false)
})
test('an empty terminal fragment hides pagination while preserving the previous cards',async()=>{
  const root = await mount();fetchMock.mockResolvedValue(new Response(''));more(root);await settle();expect(root.querySelectorAll('.blog-card')).toHaveLength(1);expect(q(root,'.blog-more').hidden).toBe(true);expect(q(root,'[data-blog-error]').hidden).toBe(true)
})
test('pagehide/return resumes an interrupted cursor and ignores its older response',async()=>{
  const root = await mount(),pending = pendingRequests();more(root);hide();expect(pending[0].signal.aborted).toBe(true);expect(q<HTMLButtonElement>(root,'[data-blog-more]').disabled).toBe(false);expect(q(root,'[data-blog-list]').hasAttribute('aria-busy')).toBe(false);show();expect(pending[1].path).toBe(pending[0].path);pending[1].complete(new Response(markup.cards));await settle();pending[0].complete(new Response(markup.cards,{headers:{'X-Next-Cursor':'stale'}}));await settle();expect(root.querySelectorAll('.blog-card')).toHaveLength(2);expect(q<HTMLElement>(root,'[data-blog-more]').dataset.cursor).toBe('')
})
test('text completing after detach cannot append cards and cached controls are usable',async()=>{
  const root = await mount();let release:(text:string)=>void = ()=>{};const response = new Response('');vi.spyOn(response,'text').mockImplementation(()=>new Promise(resolve=>release = resolve));fetchMock.mockResolvedValue(response);more(root);await settle();root.remove();await settle();release(markup.cards);await settle();expect(root.querySelectorAll('.blog-card')).toHaveLength(1);expect(q<HTMLButtonElement>(root,'[data-blog-more]').disabled).toBe(false);expect(q(root,'[data-blog-list]').hasAttribute('aria-busy')).toBe(false)
})
test('an exhausted SSR list has no pagination work and invalid dates retain their text',async()=>{
  const root = await mount('listing',root=>{q(root,'.blog-more').remove();const date = q<HTMLTimeElement>(root,'time');date.dateTime = 'invalid';date.textContent = 'Server fallback'});expect(q(root,'time').textContent).toBe('Server fallback');hide();show();expect(fetchMock).not.toHaveBeenCalled()
})
