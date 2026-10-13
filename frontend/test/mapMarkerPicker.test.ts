// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
import {createMapMarkerPicker,type MapMarkerChoice} from '../src/utils/map/markerPicker'

function fixture(){
  const host=document.createElement('div');document.body.append(host)
  const choices:MapMarkerChoice[]=['The Forge','The Citadel'].map((name,index)=>{
    const node=document.createElement('button');node.textContent=name;host.append(node)
    vi.spyOn(node,'getBoundingClientRect').mockReturnValue(new DOMRect(100+index*12,200,75,30))
    return {node,name,count:`${index+2} battles`,select:vi.fn()}
  })
  const picker=createMapMarkerPicker(host),panel=host.querySelector('dialog')!
  return {host,choices,picker,panel}
}
const pointer=()=>new MouseEvent('click',{detail:1,clientX:140,clientY:215})
afterEach(()=>{document.body.replaceChildren();vi.restoreAllMocks()})

test('a pointer hit covered by multiple labels offers each actual location before navigating',()=>{
  const {choices,picker,panel}=fixture()
  picker.activate(pointer(),choices[1]!,choices)
  expect(panel.open).toBe(true)
  expect(panel.textContent).toContain('The Forge2 battles')
  expect(panel.textContent).toContain('The Citadel3 battles')
  expect(choices[0]!.select).not.toHaveBeenCalled();expect(choices[1]!.select).not.toHaveBeenCalled()
  panel.querySelector<HTMLButtonElement>('header + button')!.click()
  expect(choices[0]!.select).toHaveBeenCalledOnce();expect(choices[1]!.select).not.toHaveBeenCalled()
  expect(panel.open).toBe(false)
})

test('keyboard activation and a single exposed pointer marker retain direct selection',()=>{
  const {choices,picker,panel}=fixture()
  picker.activate(new MouseEvent('click',{detail:0}),choices[0]!,choices)
  expect(choices[0]!.select).toHaveBeenCalledOnce();expect(panel.open).toBe(false)
  picker.activate(new MouseEvent('click',{detail:1,clientX:104,clientY:215}),choices[0]!,choices)
  expect(choices[0]!.select).toHaveBeenCalledTimes(2);expect(panel.open).toBe(false)
})

test('hidden, detached and zero-size labels cannot turn a direct hit into an ambiguous choice',()=>{
  const {choices,picker,panel}=fixture()
  choices[1]!.node.hidden=true;picker.activate(pointer(),choices[0]!,choices)
  expect(choices[0]!.select).toHaveBeenCalledOnce();expect(panel.open).toBe(false)
  choices[1]!.node.hidden=false;vi.spyOn(choices[1]!.node,'getBoundingClientRect').mockReturnValue(new DOMRect())
  picker.activate(pointer(),choices[0]!,choices)
  choices[1]!.node.remove();picker.activate(pointer(),choices[0]!,choices)
  expect(choices[0]!.select).toHaveBeenCalledTimes(3);expect(panel.open).toBe(false)
})

test('dismissal, replacement and detached maps cannot deliver old choices',()=>{
  const {host,choices,picker,panel}=fixture()
  picker.activate(pointer(),choices[0]!,choices)
  const old=panel.querySelector<HTMLButtonElement>('header + button')!
  picker.hide();old.click();expect(choices[0]!.select).not.toHaveBeenCalled()
  picker.activate(pointer(),choices[1]!,choices);old.click()
  expect(panel.open).toBe(true);expect(choices[0]!.select).not.toHaveBeenCalled()
  const current=panel.querySelector<HTMLButtonElement>('header + button')!
  host.remove();current.click();picker.activate(pointer(),choices[0]!,choices)
  expect(choices[0]!.select).not.toHaveBeenCalled()
})

test('native popover selection uses the same location contract and close control',()=>{
  const {host,choices}=fixture();host.querySelector('dialog')!.remove()
  // Exercise the native branch while happy-dom supplies no Popover API.
  const create=document.createElement.bind(document)
  let nativeOpen=false
  vi.spyOn(document,'createElement').mockImplementation((tag,options)=>{
    const node=create(tag,options)
    if(tag==='dialog'){
      const panel=node as HTMLDialogElement,original=panel.matches.bind(panel)
      panel.showPopover=()=>{nativeOpen=true};panel.hidePopover=()=>{nativeOpen=false}
      vi.spyOn(panel,'matches').mockImplementation(selector=>selector===':popover-open'?nativeOpen:original(selector))
    }
    return node
  })
  const picker=createMapMarkerPicker(host),panel=host.querySelector('dialog')!
  picker.activate(pointer(),choices[1]!,choices)
  expect(nativeOpen).toBe(true);expect(document.activeElement).toBe(panel.querySelector('header + button'))
  panel.querySelector<HTMLButtonElement>('header button')!.click()
  expect(nativeOpen).toBe(false);expect(choices[1]!.select).not.toHaveBeenCalled()
  picker.activate(pointer(),choices[1]!,choices)
  panel.querySelectorAll<HTMLButtonElement>(':scope > button')[1]!.click()
  expect(choices[1]!.select).toHaveBeenCalledOnce();expect(nativeOpen).toBe(false)
})

test('location labels remain plain text instead of becoming executable markup',()=>{
  const {choices,picker,panel}=fixture();choices[0]!.name='<img src=x onerror=alert(1)>'
  picker.activate(pointer(),choices[1]!,choices)
  expect(panel.querySelector('img')).toBeNull();expect(panel.textContent).toContain(choices[0]!.name)
})
