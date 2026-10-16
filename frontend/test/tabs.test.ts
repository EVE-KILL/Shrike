// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
import {moveTabFocus} from '../src/browser/shared/tabs'

afterEach(()=>document.body.replaceChildren())
test('async tabs wrap focus without loading a panel; Home and End retain one tab stop',()=>{
 document.body.innerHTML='<div role="tablist"><button role="tab" aria-selected="true" tabindex="0">Kills</button><button role="tab" tabindex="-1" disabled>Hidden</button><button role="tab" tabindex="-1">Ships</button><button role="tab" tabindex="-1">Structures</button></div>'
 const list=document.querySelector<HTMLElement>('[role=tablist]')!,tabs=[...list.querySelectorAll('button')],click=vi.fn()
 list.addEventListener('keydown',event=>moveTabFocus(event));list.addEventListener('click',click)
 const press=(key:string)=>document.activeElement!.dispatchEvent(new KeyboardEvent('keydown',{key,bubbles:true,cancelable:true}))
 tabs[0]!.focus();press('ArrowRight');expect(document.activeElement).toBe(tabs[2]);expect(click).not.toHaveBeenCalled()
 expect(tabs.filter(tab=>tab.tabIndex===0)).toEqual([tabs[2]])
 press('End');expect(document.activeElement).toBe(tabs[3]);press('ArrowRight');expect(document.activeElement).toBe(tabs[0])
 press('ArrowLeft');expect(document.activeElement).toBe(tabs[3]);press('Home');expect(document.activeElement).toBe(tabs[0])
 expect(tabs[0]!.getAttribute('aria-selected')).toBe('true')
})
test('local tabs activate when requested and leave unrelated or modified keys alone',()=>{
 document.body.innerHTML='<div role="tablist"><button role="tab">Kills</button><button role="tab">Top Lists</button></div>'
 const tabs=[...document.querySelectorAll('button')],activate=vi.fn()
 document.body.addEventListener('keydown',event=>moveTabFocus(event,true),{once:true});tabs[1]!.addEventListener('click',activate)
 tabs[0]!.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowRight',bubbles:true}));expect(activate).toHaveBeenCalledOnce()
 for(const init of [{key:'Tab'},{key:'Enter'},{key:'ArrowLeft',metaKey:true}]) {
  const event=new KeyboardEvent('keydown',{...init,cancelable:true});Object.defineProperty(event,'target',{value:tabs[1]});moveTabFocus(event)
  expect(event.defaultPrevented).toBe(false)
 }
})
