// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {CommentsFeedController} from '../src/browser/comments-feed'
import {registerController} from '../src/browser/controllers/application'
import {readBrowserSession,type BrowserUser} from '../src/browser/browser-request'

vi.mock('../src/browser/browser-request',async original=>({...await original<typeof import('../src/browser/browser-request')>(),readBrowserSession:vi.fn()}))
const markup=readFileSync(resolve(import.meta.dirname,'fixtures/comments-feed.html'),'utf8'),settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn(),owner:BrowserUser={characterId:4242,characterName:'Fixture Owner',isAdmin:false,corporationId:null,corporationName:null,allianceId:null,allianceName:null,lastSeenNotificationId:0}
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(root:ParentNode,selector:string):T{const element=root.querySelector<T>(selector);if(!element)throw Error('Missing feed fixture '+selector);return element}
async function mount(){document.body.innerHTML=markup;await settle();return q(document,'[data-comments-feed]')}
const fragment=(id:number)=>`<div data-comments-fragment><article class="comment-feed-card" data-comment-id="${id}" data-author-id="4242"><p>Fixture card ${id}</p><button data-comment-delete="${id}">Delete</button></article></div>`
function pendingReads(){const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}>=[];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
beforeEach(()=>{registerController('comments-feed',CommentsFeedController).handleError=errors;errors.mockClear();vi.mocked(readBrowserSession).mockResolvedValue(owner);fetchMock=vi.fn<typeof fetch>(async()=>new Response(fragment(99),{headers:{'X-Next-Cursor':'next-fixture'}}));vi.stubGlobal('fetch',fetchMock)})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})
test('SSR feed retains cards during sort and only exposes owner deletion controls',async()=>{
  const root=await mount(),cards=Array.from(root.querySelectorAll<HTMLElement>('.comment-feed-card')),sort=q<HTMLSelectElement>(root,'[data-comments-sort]');expect(cards.length).toBeGreaterThan(0);sort.value='oldest';sort.dispatchEvent(new Event('change',{bubbles:true}));expect(root.querySelector('.comment-feed-card')).toBe(cards.at(-1));expect(fetchMock).not.toHaveBeenCalled();for(const button of root.querySelectorAll<HTMLButtonElement>('[data-comment-delete]'))expect(button.hidden).toBe(Number(button.closest<HTMLElement>('[data-author-id]')?.dataset.authorId)!==4242)
})
test('search debounces and rejects obsolete or mismatched fragments without replacing current rows',async()=>{
  const root=await mount(),pending=pendingReads();vi.useFakeTimers();const search=q<HTMLInputElement>(root,'[data-comments-search]');search.value='Older';search.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301);search.value='Newer';search.dispatchEvent(new Event('input',{bubbles:true}));await vi.advanceTimersByTimeAsync(301);expect(pending[0].signal.aborted).toBe(true);pending[1].complete(new Response(fragment(99)));await vi.advanceTimersByTimeAsync(0);pending[0].complete(new Response(fragment(88)));await vi.advanceTimersByTimeAsync(0);expect(root.querySelector('[data-comment-id="88"]')).toBeNull();expect(root.querySelector('[data-comment-id="99"]')).not.toBeNull()
  q<HTMLButtonElement>(root,'[data-comments-refresh]').click();pending[2].complete(new Response('<main>Wrong page</main>'));await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-comments-error]').textContent).toBe('Failed to load comments');expect(root.querySelector('[data-comment-id="99"]')).not.toBeNull()
})
test('paging deduplicates cards and carries cursor and target filters',async()=>{
  const root=await mount();q<HTMLButtonElement>(root,'[data-comments-refresh]').click();await settle();fetchMock.mockResolvedValueOnce(new Response(fragment(99)+fragment(100),{headers:{'X-Next-Cursor':'next-fixture-2'}}));q<HTMLButtonElement>(root,'[data-comments-more]').click();await settle();expect(root.querySelectorAll('[data-comment-id="99"]')).toHaveLength(1);expect(root.querySelectorAll('.comment-feed-card')).toHaveLength(2);expect(new URL(String(fetchMock.mock.calls.at(-1)![0]),location.origin).searchParams.get('cursor')).toBe('next-fixture')
})
test('deletion confirmation, failures and disconnects preserve the displayed feed',async()=>{
  const root=await mount();q<HTMLButtonElement>(root,'[data-comments-refresh]').click();await settle();const confirm=vi.fn().mockReturnValueOnce(false).mockReturnValueOnce(true);vi.stubGlobal('confirm',confirm);vi.stubGlobal('alert',vi.fn());const button=q<HTMLButtonElement>(root,'[data-comment-delete="99"]'),before=fetchMock.mock.calls.length;button.click();expect(fetchMock).toHaveBeenCalledTimes(before);const pending=pendingReads();button.click();button.click();expect(pending).toHaveLength(1);root.remove();await settle();expect(pending[0].signal.aborted).toBe(true);pending[0].complete(new Response('',{status:503}));await settle();expect(alert).not.toHaveBeenCalled();expect(root.querySelector('[data-comment-id="99"]')).not.toBeNull()
})
