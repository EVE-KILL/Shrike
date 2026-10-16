// @vitest-environment happy-dom
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {SpotlightSearch} from '../src/browser/shared/spotlight'

const response={hits:[
  {id:'ship_587',type:'ship',name:'Rifter'},
  {id:'character_42',type:'character',name:'Pilot <script>'},
  {id:'ship_588',type:'ship',name:'Rifter Fleet'},
  {id:'corporation_77',type:'corporation',name:'Corporation',ticker:'CORP'},
],entityCounts:{ship:2,character:1,corporation:1},total:4,processingTimeMs:1}
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function mount(){
  const dialog=document.createElement('dialog')
  dialog.innerHTML='<input data-shell-target="searchInput"><button data-spotlight-clear hidden>Clear</button><div data-shell-target="searchResults"></div><template data-spotlight-default><div class="spotlight-default"><div class="spotlight-actions"><a href="/kills/latest">Latest Kills</a><a href="/map">Map</a></div><section data-spotlight-recents hidden><div data-spotlight-recent-list></div></section><section>EVE Tools</section></div></template>'
  document.body.append(dialog);dialog.showModal()
  const search=new SpotlightSearch(dialog),input=dialog.querySelector('input')!,navigated:string[]=[]
  dialog.addEventListener('click',event=>{
    const link=(event.target as Element).closest<HTMLAnchorElement>('a')
    if(link){navigated.push(link.getAttribute('href')!);event.preventDefault()}
  })
  dialog.addEventListener('click',event=>search.click(event))
  dialog.addEventListener('keydown',event=>search.key(event))
  dialog.addEventListener('close',()=>search.close())
  input.addEventListener('input',()=>search.search());search.open()
  return {dialog,search,input,navigated,type:(value:string)=>{input.value=value;input.dispatchEvent(new Event('input',{bubbles:true}))}}
}
beforeEach(()=>{
  vi.useFakeTimers();localStorage.clear()
  fetchMock=vi.fn<typeof fetch>(async()=>Response.json(response));vi.stubGlobal('fetch',fetchMock)
})
afterEach(()=>{
  document.body.replaceChildren();document.documentElement.classList.remove('spotlight-open')
  vi.useRealTimers();vi.unstubAllGlobals();vi.restoreAllMocks()
})
test('default actions, stored recents and the original three-character debounce',async()=>{
  localStorage.setItem('evekill-recent-searches',JSON.stringify(['Jita',null,42,'Rifter']))
  const {dialog,input,type}=mount()
  expect(document.activeElement).toBe(input);expect(dialog.textContent).toContain('Latest Kills')
  expect(dialog.querySelectorAll('[data-spotlight-recent]')).toHaveLength(2)
  type('Ri');await vi.advanceTimersByTimeAsync(500);expect(fetchMock).not.toHaveBeenCalled()
  expect(dialog.textContent).toContain('Type at least 3 characters')
  type('Rifter');await vi.advanceTimersByTimeAsync(299);expect(fetchMock).not.toHaveBeenCalled()
  await vi.advanceTimersByTimeAsync(1);expect(fetchMock).toHaveBeenCalledOnce()
  expect(dialog.querySelectorAll('.spotlight-result')).toHaveLength(4)
  expect(JSON.parse(localStorage.getItem('evekill-recent-searches')!)).toEqual(['Jita','Rifter'])
})
test('groups, small images, tickers, category toggles and keyboard follow the displayed order',async()=>{
  const {dialog,input,type,navigated}=mount();type('Rifter');await vi.advanceTimersByTimeAsync(300)
  expect([...dialog.querySelectorAll('.spotlight-result')].map(a=>a.getAttribute('href'))).toEqual(['/item/587','/item/588','/character/42','/corporation/77'])
  expect(dialog.querySelector('.spotlight-result img')!.getAttribute('src')).toBe('/images/types/587/icon?size=64')
  expect(dialog.textContent).toContain('[CORP]');expect(dialog.querySelector('script')).toBeNull()
  const ships=dialog.querySelector<HTMLButtonElement>('[data-spotlight-filter=ship]')!;ships.click()
  expect(dialog.querySelectorAll('.spotlight-result')).toHaveLength(2);expect(fetchMock).toHaveBeenCalledOnce()
  input.focus();input.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowUp',bubbles:true}))
  expect(dialog.querySelector('[aria-selected=true]')!.getAttribute('href')).toBe('/item/588')
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true}))
  expect(navigated).toEqual(['/item/588']);expect(dialog.open).toBe(false)
})
test('empty search supports arrow navigation through default actions without intercepting native activation',()=>{
  const {dialog,input}=mount(),actions=dialog.querySelectorAll<HTMLAnchorElement>('.spotlight-actions a')
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true}));expect(document.activeElement).toBe(actions[0])
  actions[0]!.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true}));expect(document.activeElement).toBe(actions[1])
  const enter=new KeyboardEvent('keydown',{key:'Enter',bubbles:true,cancelable:true});actions[1]!.dispatchEvent(enter);expect(enter.defaultPrevented).toBe(false)
})
test('IME composition and modified Enter never open a result',async()=>{
  const {input,type,navigated}=mount();type('Rifter');await vi.advanceTimersByTimeAsync(300)
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true}))
  for(const init of [{isComposing:true},{metaKey:true},{ctrlKey:true}])input.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',...init,bubbles:true}))
  expect(navigated).toEqual([])
})
test('multiple results require keyboard selection, while one visible result opens on Enter',async()=>{
  const {dialog,input,type,navigated}=mount();type('Rifter');await vi.advanceTimersByTimeAsync(300)
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true}));expect(navigated).toEqual([])
  dialog.querySelector<HTMLButtonElement>('[data-spotlight-filter=character]')!.click()
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true}));expect(navigated).toEqual(['/character/42'])
})
test('tab-focused results retain native Enter and arrow navigation starts from their position',async()=>{
  const {dialog,type}=mount();type('Rifter');await vi.advanceTimersByTimeAsync(300)
  const links=dialog.querySelectorAll<HTMLAnchorElement>('.spotlight-result')
  const enter=new KeyboardEvent('keydown',{key:'Enter',bubbles:true,cancelable:true})
  links[2]!.focus();links[2]!.dispatchEvent(enter);expect(enter.defaultPrevented).toBe(false)
  links[2]!.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true}))
  expect(document.activeElement).toBe(links[3])
  expect(links[3]!.getAttribute('aria-selected')).toBe('true')
})
test('new queries reset filtering, retain a bounded history and expose full search',async()=>{
  const {dialog,type}=mount();type('Rifter');await vi.advanceTimersByTimeAsync(300)
  dialog.querySelector<HTMLButtonElement>('[data-spotlight-filter=ship]')!.click()
  type('Jita');await vi.advanceTimersByTimeAsync(300)
  expect(dialog.querySelector('[data-spotlight-filter=all]')!.getAttribute('aria-pressed')).toBe('true')
  expect(dialog.querySelector('.spotlight-view-all')!.getAttribute('href')).toBe('/search?q=Jita')
  for(const query of ['Amarr','Hek','Dodixie','Rens']){type(query);await vi.advanceTimersByTimeAsync(300)}
  expect(JSON.parse(localStorage.getItem('evekill-recent-searches')!)).toEqual(['Rens','Dodixie','Hek','Amarr','Jita'])
})
test('clear, select and remove recent queries use the shared storage key',async()=>{
  localStorage.setItem('evekill-recent-searches',JSON.stringify(['Rifter','Jita']))
  const {dialog,input}=mount()
  dialog.querySelector<HTMLButtonElement>('[data-spotlight-recent="0"]')!.click();await vi.advanceTimersByTimeAsync(300)
  expect(input.value).toBe('Rifter')
  dialog.querySelector<HTMLButtonElement>('[data-spotlight-clear]')!.click()
  expect(input.value).toBe('');expect(dialog.querySelector('.spotlight-default')).not.toBeNull()
  dialog.querySelector<HTMLButtonElement>('[data-spotlight-remove="0"]')!.click()
  expect(JSON.parse(localStorage.getItem('evekill-recent-searches')!)).toEqual(['Jita'])
})
test('late or cancelled responses cannot repopulate a closed or newer search',async()=>{
  const pending:Array<{resolve:(r:Response)=>void;signal:AbortSignal}>=[]
  fetchMock.mockImplementation((_input,init)=>new Promise(resolve=>pending.push({resolve,signal:init!.signal!})))
  const {dialog,search,type}=mount();type('Rifter');await vi.advanceTimersByTimeAsync(300)
  type('Jita');await vi.advanceTimersByTimeAsync(300);expect(pending[0]!.signal.aborted).toBe(true)
  pending[0]!.resolve(Response.json(response));await vi.advanceTimersByTimeAsync(0)
  expect(dialog.querySelector('.spotlight-result')).toBeNull()
  dialog.close();expect(pending[1]!.signal.aborted).toBe(true)
  pending[1]!.resolve(Response.json(response));await vi.advanceTimersByTimeAsync(0)
  expect(dialog.querySelector('.spotlight-result')).toBeNull()
  dialog.showModal();search.open();expect(dialog.querySelector('.spotlight-default')).not.toBeNull()
})
test('empty results, malformed hits and failed requests have distinct safe states',async()=>{
  const {dialog,type}=mount();fetchMock.mockResolvedValueOnce(Response.json({hits:[{id:'ship_javascript:1',type:'ship',name:'Bad'}]}))
  type('Nothing');await vi.advanceTimersByTimeAsync(300)
  expect(dialog.textContent).toContain('No results for "Nothing"');expect(dialog.textContent).toContain('Try a different name or fewer words.');expect(dialog.querySelector('.spotlight-result')).toBeNull()
  fetchMock.mockResolvedValueOnce(new Response('',{status:503}));type('Retry');await vi.advanceTimersByTimeAsync(300)
  expect(dialog.textContent).toContain('Search is unavailable')
  type('Retry');await vi.advanceTimersByTimeAsync(300);expect(dialog.querySelectorAll('.spotlight-result')).toHaveLength(4)
})
test('disabled browser storage does not prevent search',async()=>{
  vi.spyOn(Storage.prototype,'getItem').mockImplementation(()=>{throw new Error('disabled')})
  vi.spyOn(Storage.prototype,'setItem').mockImplementation(()=>{throw new Error('disabled')})
  const {dialog,type}=mount();type('Rifter');await vi.advanceTimersByTimeAsync(300)
  expect(dialog.querySelectorAll('.spotlight-result')).toHaveLength(4)
})
