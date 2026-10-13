// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
import {initializeSearchPickers,updateSearchPicker,closeSearchPickers,SearchPickerController,type SearchSelection} from '../src/browser/search-picker'
import {registerController} from '../src/browser/controllers/application'
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,15))
const errors=vi.fn()
function mount(){
  registerController('search-picker',SearchPickerController).handleError=errors
  document.body.innerHTML='<div data-search-picker data-picker-types="character" data-picker-debounce="1"><input data-picker-input placeholder="Add pilot"><span data-picker-spinner hidden></span><div data-picker-results hidden></div><template data-picker-hit-template><button><img data-picker-image><div data-picker-symbol><svg><use href="/assets/icons.svg#map-pin"></use></svg></div><span data-picker-hit-name></span><span data-picker-hit-type></span><span data-picker-hit-ticker></span></button></template></div>'
  initializeSearchPickers();initializeSearchPickers()
  return {root:document.querySelector<HTMLElement>('[data-search-picker]')!,input:document.querySelector<HTMLInputElement>('input')!,results:document.querySelector<HTMLElement>('[data-picker-results]')!}
}
function type(input:HTMLInputElement,value:string){input.value=value;input.dispatchEvent(new Event('input',{bubbles:true}))}
afterEach(async()=>{document.body.replaceChildren();await settle();expect(errors.mock.calls).toEqual([]);errors.mockClear();vi.unstubAllGlobals()})
test('parent options set before connection preserve caps, deduplication and typed keyboard selection',async()=>{
  const fetchMock=vi.fn<typeof fetch>(async()=>Response.json({hits:[{id:42,name:'Already tracked',type:'character',ticker:'TAG',url:'/character/42'},{id:43,name:'New pilot',type:'character',ticker:null,url:'/character/43'},{id:44,name:'Wrong category',type:'corporation',ticker:null,url:'/corporation/44'}]}))
  vi.stubGlobal('fetch',fetchMock)
  const {root,input,results}=mount();updateSearchPicker(root,{disabled:true,isPicked:(_type,id)=>id===42});await settle()
  type(input,'Pilot');await settle();expect(fetchMock).not.toHaveBeenCalled()
  updateSearchPicker(root,{disabled:false});type(input,'Pilot');await settle();await settle()
  expect(results.querySelectorAll('button')).toHaveLength(2);expect(results.querySelector<HTMLButtonElement>('button')!.disabled).toBe(true)
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true,cancelable:true}))
  expect(document.activeElement!.textContent).toContain('New pilot')
  const selected:SearchSelection[]=[];root.addEventListener('search-pick',event=>{if(event instanceof CustomEvent)selected.push(event.detail)})
  results.querySelector<HTMLButtonElement>('button:not([disabled])')!.click();expect(selected.map(hit=>hit.id)).toEqual([43]);expect(input.value).toBe('');expect(results.hidden).toBe(true)
})
test('closing or detaching a picker aborts and ignores results even if its transport resolves late',async()=>{
  const pending:Array<{signal:AbortSignal;complete:(response:Response)=>void}>=[]
  vi.stubGlobal('fetch',vi.fn<typeof fetch>((_path,options)=>new Promise(resolve=>pending.push({signal:options!.signal!,complete:resolve}))))
  const {root,input,results}=mount();await settle();type(input,'Pilot');await settle();expect(pending).toHaveLength(1)
  closeSearchPickers(document);expect(pending[0].signal.aborted).toBe(true)
  pending[0].complete(Response.json({hits:[{id:42,name:'Late result',type:'character',ticker:null,url:'/character/42'}]}));await settle();expect(results.hidden).toBe(true);expect(results.textContent).toBe('')
  type(input,'Next');await settle();root.remove();await settle();expect(pending[1].signal.aborted).toBe(true)
  pending[1].complete(Response.json({hits:[]}));await settle()
})
test('reinserted picker binds once, keeps its original placeholder, and closes on outside input',async()=>{
  const fetchMock=vi.fn<typeof fetch>(async()=>Response.json({hits:[{id:43,name:'Pilot',type:'character',ticker:null,url:'/character/43'}]}));vi.stubGlobal('fetch',fetchMock)
  const {root,input,results}=mount();await settle();updateSearchPicker(root,{placeholder:'Maximum reached'})
  root.remove();await settle();document.body.append(root);await settle();updateSearchPicker(root,{});expect(input.placeholder).toBe('Add pilot')
  type(input,'Pilot');await settle();await settle();expect(fetchMock).toHaveBeenCalledOnce();expect(results.hidden).toBe(false)
  document.body.dispatchEvent(new PointerEvent('pointerdown',{bubbles:true}));expect(results.hidden).toBe(true)
})
test('location results choose their canonical icon independently of the template SVG path',async()=>{
 vi.stubGlobal('fetch',vi.fn<typeof fetch>(async()=>Response.json({hits:[{id:10000002,name:'The Forge',type:'region',ticker:null,url:'/region/10000002'}]})))
 const {root,input,results}=mount();root.dataset.pickerTypes='region';root.querySelector<HTMLTemplateElement>('template')!.content.querySelector('use')!.setAttribute('href','/assets/lucide/map-pin.svg#icon');await settle();type(input,'Forge');await settle();await settle();expect(results.querySelector('use')!.getAttribute('href')).toBe('/assets/icons.svg#map')
})
