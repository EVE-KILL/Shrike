// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
import {createSelectMenu} from '../src/browser/select-menu'
import '../src/browser/categories'
import '../src/browser/legacy-kill'
import '../src/browser/controllers/coalition-activity'

const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0))
afterEach(async()=>{document.body.replaceChildren();await settle();vi.unstubAllGlobals()})

test('select menu retains keyboard, selection, outside click and current option when options update',async()=>{
  const changes:string[]=[]
  const menu=createSelectMenu('Group',[{value:'',label:'All'},{value:'ship',label:'Ships'}],'',value=>changes.push(value))
  document.body.append(menu.root);await settle()
  const trigger=menu.root.querySelector('button')!,list=menu.root.querySelector<HTMLElement>('[data-select-options]')!
  trigger.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowDown',bubbles:true}));await settle()
  expect(trigger.getAttribute('aria-expanded')).toBe('true');expect(document.activeElement).toBe(list.querySelector('button'))
  list.querySelector<HTMLButtonElement>('[data-select-value=ship]')!.click();await settle()
  expect(changes).toEqual(['ship']);expect(menu.root.querySelector('[data-select-label]')!.textContent).toBe('Ships')
  expect(document.activeElement).toBe(trigger);expect(list.hidden).toBe(true)
  menu.setOptions([{value:'ship',label:'Ships only'}]);await settle()
  expect(menu.root.querySelector('[data-select-label]')!.textContent).toBe('Ships only')
  trigger.click();await settle();expect(list.hidden).toBe(false)
  document.body.dispatchEvent(new PointerEvent('pointerdown',{bubbles:true}));await settle();expect(list.hidden).toBe(true)
  trigger.click();await settle();menu.root.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true}));await settle()
  expect(list.hidden).toBe(true)
  menu.dispose()
})

test('fragment removal/reconnection cleans action bindings and registering another control preserves an open menu',async()=>{
  const changes:string[]=[]
  const menu=createSelectMenu('First',[{value:'one',label:'One'},{value:'two',label:'Two'}],'one',value=>changes.push(value))
  document.body.append(menu.root);await settle()
  const trigger=menu.root.querySelector('button')!
  trigger.click();await settle()
  const second=createSelectMenu('Second',[],'',()=>{})
  document.body.append(second.root);await settle()
  expect(trigger.getAttribute('aria-expanded')).toBe('true')
  menu.root.remove();await settle()
  trigger.click();await settle();expect(changes).toEqual([])
  document.body.append(menu.root);await settle()
  expect(trigger.getAttribute('aria-expanded')).toBe('false')
  trigger.click();await settle()
  menu.root.querySelector<HTMLButtonElement>('[data-select-value=two]')!.click();await settle()
  expect(changes).toEqual(['two'])
  menu.dispose();await settle();trigger.click();await settle()
  expect(trigger.getAttribute('aria-expanded')).toBe('false')
})

test('newly inserted category fragments filter immediately, including empty state and counts',async()=>{
  document.body.innerHTML=`<section data-controller="categories">
    <input data-categories-target="input" data-action="input->categories#filter" value="frigate">
    <span data-categories-target="count"></span>
    <div data-categories-target="group"><span class="category-group-count"></span>
      <article class="category-card" data-category-search="frigate"></article><article class="category-card" data-category-search="carrier"></article></div>
    <div data-categories-target="group"><span class="category-group-count"></span><article class="category-card" data-category-search="battleship"></article></div>
    <p data-categories-target="empty" hidden><span data-categories-target="query"></span></p></section>`
  await settle()
  const input=document.querySelector('input')!
  expect(document.querySelector('[data-categories-target=count]')!.textContent).toBe('1')
  expect([...document.querySelectorAll<HTMLElement>('.category-card')].map(card=>card.hidden)).toEqual([false,true,true])
  input.value='no results';input.dispatchEvent(new Event('input',{bubbles:true}));await settle()
  expect(document.querySelector<HTMLElement>('[data-categories-target=empty]')!.hidden).toBe(false)
  expect(document.querySelector('[data-categories-target=query]')!.textContent).toBe('no results')
})

test('legacy attackers expand and preserve nested-cell hover on a fragment-mounted controller',async()=>{
  document.body.innerHTML=`<section data-controller="legacy-kill" data-action="mouseover->legacy-kill#highlight mouseout->legacy-kill#unhighlight">
    ${Array.from({length:40},(_,i)=>`<div data-legacy-kill-target="row" data-attacker-index="${i}" ${i>=10?'hidden':''}><span data-cell="character"><a href="#">Pilot ${i}</a></span></div>`).join('')}
    <button data-legacy-kill-target="more" data-action="click->legacy-kill#more">More</button></section>`
  await settle()
  const more=document.querySelector('button')!,pilot=document.querySelector('a')!,row=pilot.closest<HTMLElement>('[data-attacker-index]')!
  more.click();await settle()
  expect(document.querySelectorAll('[data-attacker-index]:not([hidden])')).toHaveLength(35)
  expect(more.textContent).toBe('Show more (5 remaining)')
  more.click();await settle();expect(more.hidden).toBe(true)
  pilot.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));expect(row.dataset.hovered).toBe('character')
  pilot.dispatchEvent(new MouseEvent('mouseout',{bubbles:true,relatedTarget:pilot.parentElement}));expect(row.dataset.hovered).toBe('character')
  pilot.dispatchEvent(new MouseEvent('mouseout',{bubbles:true,relatedTarget:document.body}));expect(row.dataset.hovered).toBeUndefined()
})

function coalitionDirectory(){
  const root=document.createElement('section');root.setAttribute('data-controller','coalition-activity')
  root.innerHTML='<article><div data-coalition-activity="1">—</div></article><p data-coalition-activity-status aria-busy="false"><span></span><button data-coalition-activity-retry data-action="click->coalition-activity#load" hidden>Retry</button></p>'
  document.body.append(root)
  return root
}
test('coalition controller retries failed activity without replacing the card',async()=>{
  const request=vi.fn().mockResolvedValueOnce(new Response('',{status:503})).mockResolvedValueOnce(new Response('<div data-coalition-activity="1">42 kills</div>'))
  vi.stubGlobal('fetch',request)
  const root=coalitionDirectory(),card=root.querySelector('article')!
  await settle()
  const retry=root.querySelector<HTMLButtonElement>('button')!
  expect(retry.hidden).toBe(false);expect(root.querySelector('[data-coalition-activity]')!.textContent).toBe('—')
  retry.click();await settle()
  expect(request).toHaveBeenCalledTimes(2);expect(root.querySelector('article')).toBe(card)
  expect(root.querySelector('[data-coalition-activity]')!.textContent).toBe('42 kills')
  expect(root.querySelector<HTMLElement>('[data-coalition-activity-status]')!.hidden).toBe(true)
})
test('fragment reconnection cancels the old request and ignores its delayed response',async()=>{
  let completeOld:(response:Response)=>void=()=>{}
  const request=vi.fn().mockImplementationOnce(()=>new Promise<Response>(resolve=>{completeOld=resolve}))
    .mockResolvedValueOnce(new Response('<div data-coalition-activity="1">Fresh totals</div>'))
  vi.stubGlobal('fetch',request)
  const root=coalitionDirectory();await settle()
  const signal=request.mock.calls[0]![1].signal as AbortSignal
  root.remove();await settle();expect(signal.aborted).toBe(true)
  document.body.append(root);await settle()
  expect(request).toHaveBeenCalledTimes(2)
  completeOld(new Response('<div data-coalition-activity="1">Stale totals</div>'));await settle()
  expect(root.querySelector('[data-coalition-activity]')!.textContent).toBe('Fresh totals')
  root.remove();await settle();document.body.append(root);await settle()
  expect(request).toHaveBeenCalledTimes(2)
})
