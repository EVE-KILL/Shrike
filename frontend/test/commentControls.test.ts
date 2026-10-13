// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {createCommentInput,CommentInputController} from '../src/browser/comment-input'
import {openKlipyPicker,readKlipyTiles} from '../src/browser/klipy-picker'
import {mountCommentDialog} from '../src/browser/comment-controls'
import {registerController} from '../src/browser/controllers/application'
import {KLIPY_FAVORITES_KEY} from '../shared/utils/commentInput'

const markup=readFileSync(resolve(import.meta.dirname,'fixtures/comment-controls.html'),'utf8'),settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0)),errors=vi.fn()
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(scope:ParentNode,selector:string):T{const element=scope.querySelector<T>(selector);if(!element)throw Error('Missing comment control '+selector);return element}
async function composer(initial=''){const submit=vi.fn<(body:string,signal:AbortSignal)=>Promise<void>>(async()=>{}),handle=createCommentInput({initial,onSubmit:submit});document.body.append(handle.element);await settle();return Object.assign(handle,{submit})}
function edit(textarea:HTMLTextAreaElement|HTMLInputElement,value:string){textarea.value=value;textarea.dispatchEvent(new Event('input',{bubbles:true}))}
function pendingReads(){const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}>=[];fetchMock.mockImplementation((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve})));return pending}
const tile=(id:number)=>({id,title:'Fixture GIF '+id,file:{sm:{webp:{url:`https://fixture.example/${id}.webp`}},md:{mp4:{url:`https://fixture.example/${id}.mp4`,width:128,height:128}}}})
beforeEach(()=>{errors.mockClear();registerController('comment-input',CommentInputController).handleError=errors;document.body.innerHTML=markup;document.documentElement.style.overflow='';localStorage.clear();fetchMock=vi.fn<typeof fetch>(async path=>String(path).includes('/klipy/')?Response.json({items:[tile(1)],has_next:true,page:1,per_page:24}):Response.json({html:'<p>Fixture preview</p>'}));vi.stubGlobal('fetch',fetchMock)})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);expect(document.documentElement.style.overflow).toBe('');vi.unstubAllGlobals();vi.restoreAllMocks()})

test('the composer formats selections, counts Unicode and respects minimum/maximum body length',async()=>{
  const {element,textarea}=await composer('🚀 hello');textarea.focus();textarea.setSelectionRange(3,8);textarea.dispatchEvent(new Event('select',{bubbles:true}));q<HTMLButtonElement>(element,'[data-comment-action="bold"]').click();expect(textarea.value).toBe('🚀 **hello**');expect(textarea.selectionStart).toBe(5);expect(textarea.selectionEnd).toBe(10)
  edit(textarea,'x');expect(q<HTMLButtonElement>(element,'[data-comment-submit]').disabled).toBe(true);edit(textarea,'x'.repeat(2001));expect(q<HTMLButtonElement>(element,'[data-comment-submit]').disabled).toBe(true);expect(q(element,'[data-comment-count]').textContent).toBe('2001 / 2000')
})
test('posting prevents duplicates, preserves newer drafts and exposes errors without erasing input',async()=>{
  const handle=await composer('Submitted body');let complete:()=>void=()=>{};handle.submit.mockImplementation(()=>new Promise(resolve=>complete=resolve));const form=q<HTMLFormElement>(handle.element,'form');form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));expect(handle.submit).toHaveBeenCalledOnce();edit(handle.textarea,'Newer body');complete();await settle();expect(handle.value).toBe('Newer body')
  handle.submit.mockRejectedValueOnce(Error('Fixture save unavailable'));form.dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));await settle();expect(q(handle.element,'[data-comment-error-text]').textContent).toBe('Fixture save unavailable');expect(handle.value).toBe('Newer body');expect(q<HTMLButtonElement>(handle.element,'[data-comment-submit]').disabled).toBe(false)
})
test('preview requests abort on tab change and late results cannot replace current content',async()=>{
  const handle=await composer('Preview body'),pending=pendingReads();q<HTMLButtonElement>(handle.element,'[data-comment-tab="preview"]').click();expect(pending[0].path).toBe('/api/comments/preview');q<HTMLButtonElement>(handle.element,'[data-comment-tab="write"]').click();expect(pending[0].signal.aborted).toBe(true);pending[0].complete(Response.json({html:'Stale preview'}));await settle();expect(q(handle.element,'[data-comment-preview]').textContent).not.toBe('Stale preview');expect(handle.textarea.hidden).toBe(false)
})
test('keyboard tabs, post shortcut and disconnect own their requests and dialogs',async()=>{
  const handle=await composer('Keyboard body');q<HTMLButtonElement>(handle.element,'[data-comment-tab="write"]').dispatchEvent(new KeyboardEvent('keydown',{key:'End',bubbles:true,cancelable:true}));await settle();expect(q(handle.element,'[data-comment-tab="preview"]').getAttribute('aria-selected')).toBe('true');q<HTMLButtonElement>(handle.element,'[data-comment-tab="preview"]').dispatchEvent(new KeyboardEvent('keydown',{key:'Home',bubbles:true,cancelable:true}));handle.textarea.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',metaKey:true,bubbles:true,cancelable:true}));await settle();expect(handle.submit).toHaveBeenCalledOnce()
  const pending=pendingReads();q<HTMLButtonElement>(handle.element,'[data-comment-action="gif"]').click();await settle();expect(pending).toHaveLength(1);handle.element.remove();await settle();expect(pending[0].signal.aborted).toBe(true);expect(document.querySelector('dialog')).toBeNull()
})
test('embed validation, failed thumbnails and insertion retain editor focus and selection',async()=>{
  const handle=await composer('before after');handle.textarea.focus();handle.textarea.setSelectionRange(7,7);handle.textarea.dispatchEvent(new Event('select',{bubbles:true}));q<HTMLButtonElement>(handle.element,'[data-comment-action="image"]').click();await settle();const dialog=q<HTMLDialogElement>(document,'dialog'),url=q<HTMLInputElement>(dialog,'[data-embed-url]');edit(url,'javascript:alert(1).png');expect(q<HTMLButtonElement>(dialog,'[data-embed-insert]').disabled).toBe(true)
  url.value='https://fixture.example/ship.png';url.dispatchEvent(new Event('input',{bubbles:true}));q<HTMLImageElement>(dialog,'img').dispatchEvent(new Event('error'));expect(q<HTMLImageElement>(dialog,'img').style.display).toBe('none');q<HTMLFormElement>(dialog,'form').dispatchEvent(new SubmitEvent('submit',{bubbles:true,cancelable:true}));await settle();expect(handle.value).toBe('before [image](https://fixture.example/ship.png)after');expect(document.activeElement).toBe(handle.textarea);expect(document.querySelector('dialog')).toBeNull()
})
test('multiple dialogs share scroll ownership and closing one leaves the other usable',async()=>{
  const a=mountCommentDialog('comment-embed-template'),b=mountCommentDialog('comment-report-template');await settle();expect(document.documentElement.style.overflow).toBe('hidden');a.dispose();await settle();expect(document.documentElement.style.overflow).toBe('hidden');b.dialog.dispatchEvent(new Event('cancel',{cancelable:true}));await settle();expect(document.documentElement.style.overflow).toBe('')
})
test('GIF picker narrows media URLs, pages results, stores favorites and inserts via keyboard',async()=>{
  const insert=vi.fn(),modal=openKlipyPicker(insert);await settle();await settle();expect(modal.dialog.querySelectorAll('[data-gif-item]')).toHaveLength(1);q<HTMLButtonElement>(modal.dialog,'[data-gif-favorite]').click();expect(JSON.parse(localStorage.getItem(KLIPY_FAVORITES_KEY)!)[0]).toMatchObject({id:1,insert_url:'https://fixture.example/1.mp4'})
  fetchMock.mockResolvedValueOnce(Response.json({items:[tile(2)],has_next:false}));q<HTMLButtonElement>(modal.dialog,'[data-gif-more]').click();await settle();expect(modal.dialog.querySelectorAll('[data-gif-item]')).toHaveLength(2);expect(q(modal.dialog,'[data-gif-more-wrap]').hidden).toBe(true)
  q<HTMLButtonElement>(modal.dialog,'[data-gif-tab="favorites"]').click();q(modal.dialog,'[data-gif-item]').dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true,cancelable:true}));await settle();expect(insert).toHaveBeenCalledWith('[GIF](https://fixture.example/1.mp4)');expect(document.querySelector('dialog')).toBeNull();expect(readKlipyTiles([null,{id:1,file:{sm:{webp:{url:'javascript:bad'}}}},tile(3)]).map(item=>item.id)).toEqual([3])
})
test('GIF search debounce aborts superseded reads and closure cannot render a late response',async()=>{
  const pending=pendingReads(),modal=openKlipyPicker(vi.fn());await settle();const search=q<HTMLInputElement>(modal.dialog,'[data-gif-search]');vi.useFakeTimers();search.value='Fixture';search.dispatchEvent(new Event('input',{bubbles:true}));expect(pending[0].signal.aborted).toBe(true);await vi.advanceTimersByTimeAsync(301);expect(pending[1].path).toContain('q=Fixture');modal.dispose();await vi.advanceTimersByTimeAsync(0);expect(pending[1].signal.aborted).toBe(true);pending[1].complete(Response.json({items:[tile(9)],has_next:false}));await vi.advanceTimersByTimeAsync(0);expect(document.querySelector('dialog')).toBeNull()
})
