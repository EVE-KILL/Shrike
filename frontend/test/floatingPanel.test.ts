// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
import {placeFloatingPanel,positionViewportPanel} from '../src/browser/shared/floating-panel'
function box(top:number,height:number){return new DOMRect(0,top,304,height)}
function fixture(dialog=false){document.body.innerHTML='<nav aria-label="Main navigation" style="position:fixed"></nav>'+(dialog?'<dialog open>':'<div>')+'<div data-anchor><div data-panel></div></div>'+(dialog?'</dialog>':'</div>');const nav=document.querySelector<HTMLElement>('nav')!,anchor=document.querySelector<HTMLElement>('[data-anchor]')!,panel=document.querySelector<HTMLElement>('[data-panel]')!;vi.spyOn(nav,'getBoundingClientRect').mockReturnValue(box(0,48));vi.spyOn(anchor,'getBoundingClientRect').mockReturnValue(box(400,40));vi.spyOn(panel,'getBoundingClientRect').mockReturnValue(box(0,389));vi.stubGlobal('innerHeight',700);return {anchor,panel}}
afterEach(()=>{document.body.replaceChildren();vi.unstubAllGlobals();vi.restoreAllMocks()})
test('page popups open above their trigger without covering fixed navbar controls',()=>{const {anchor,panel}=fixture();placeFloatingPanel(anchor,panel);expect(panel.classList.contains('bottom-full')).toBe(true);expect(panel.style.maxHeight).toBe('340px');expect(panel.style.overflowY).toBe('auto')})
test('native modal popups use modal clipping bounds instead of the navbar behind the dialog',()=>{const {anchor,panel}=fixture(true);placeFloatingPanel(anchor,panel);expect(panel.classList.contains('bottom-full')).toBe(true);expect(panel.style.maxHeight).toBe('388px');expect(panel.style.overflowY).toBe('auto')})
test('floating panels still respect smaller scrolling boundaries and reset prior height limits',()=>{const {anchor,panel}=fixture(),boundary=anchor.parentElement!;boundary.style.overflowY='auto';vi.spyOn(boundary,'getBoundingClientRect').mockReturnValue(box(150,350));panel.style.maxHeight='10px';placeFloatingPanel(anchor,panel);expect(panel.style.maxHeight).toBe('242px');expect(panel.classList.contains('bottom-full')).toBe(true)})
test('absolute date panels clear phone edges and reset a previous horizontal correction',()=>{
  const {anchor,panel}=fixture();vi.stubGlobal('innerWidth',320)
  vi.spyOn(panel,'getBoundingClientRect').mockReturnValue(new DOMRect(-1,0,296,389));placeFloatingPanel(anchor,panel);expect(panel.style.translate).toBe('9px 0px')
  vi.spyOn(panel,'getBoundingClientRect').mockReturnValue(new DOMRect(29,0,296,389));placeFloatingPanel(anchor,panel);expect(panel.style.translate).toBe('-13px 0px')
  vi.spyOn(panel,'getBoundingClientRect').mockReturnValue(new DOMRect(12,0,296,389));placeFloatingPanel(anchor,panel);expect(panel.style.translate).toBe('')
})
test('viewport popovers stay inside phone edges and choose the side with room',()=>{
  const panel=document.createElement('div');vi.spyOn(panel,'getBoundingClientRect').mockReturnValue(new DOMRect(0,0,320,240));vi.stubGlobal('innerWidth',390);vi.stubGlobal('innerHeight',844)
  positionViewportPanel(panel,new DOMRect(380,830,0,0));expect(panel.style.left).toBe('62px');expect(panel.style.top).toBe('586px')
  positionViewportPanel(panel,new DOMRect(0,4,30,20));expect(panel.style.left).toBe('8px');expect(panel.style.top).toBe('28px')
})
test('viewport popovers reserve scrollbar space and constrain wide menus to the content viewport',()=>{
  const panel=document.createElement('div');vi.spyOn(panel,'getBoundingClientRect').mockReturnValue(new DOMRect(0,0,320,240));vi.stubGlobal('innerWidth',390);vi.stubGlobal('innerHeight',844)
  vi.spyOn(document.documentElement,'clientWidth','get').mockReturnValue(379)
  positionViewportPanel(panel,new DOMRect(380,400,0,0));expect(panel.style.left).toBe('51px');expect(panel.style.maxWidth).toBe('363px')
  vi.spyOn(document.documentElement,'clientWidth','get').mockReturnValue(309)
  positionViewportPanel(panel,new DOMRect(300,400,0,0));expect(panel.style.maxWidth).toBe('293px')
})
