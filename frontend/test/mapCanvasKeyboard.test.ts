// @vitest-environment happy-dom
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {createGoMap,type GoMapController} from '../src/utils/map/goFrontendCanvas'

let frames:FrameRequestCallback[],map:GoMapController|undefined
const translate=vi.fn(),scale=vi.fn()
beforeEach(()=>{
 frames=[];translate.mockClear();scale.mockClear()
 const context=new Proxy({translate,scale},{get:(target,key)=>key in target?target[key as keyof typeof target]:vi.fn()})
 vi.spyOn(HTMLCanvasElement.prototype,'getContext').mockReturnValue(context as unknown as CanvasRenderingContext2D)
 vi.stubGlobal('requestAnimationFrame',(callback:FrameRequestCallback)=>{frames.push(callback);return frames.length})
 vi.stubGlobal('cancelAnimationFrame',vi.fn())
 vi.stubGlobal('ResizeObserver',class {observe(){} disconnect(){}})
})
afterEach(()=>{map?.destroy();map=undefined;document.body.replaceChildren();vi.useRealTimers();vi.unstubAllGlobals();vi.restoreAllMocks()})
function setup(baseLayer:'geography'|'live'='geography') {
 const host=document.createElement('div');document.body.append(host)
 Object.defineProperties(host,{clientWidth:{value:390},clientHeight:{value:600}})
 const inspect=vi.fn();map=createGoMap(host,{baseLayer,onBattleRegion:inspect})
 map.setData({systems:[{solar_system_id:1,system_name:'Fixture',region_id:10,security:.5,x:0,z:0}],regions:[{region_id:10,name:'Region'}]})
 const flush=()=>{const pending=frames.splice(0);pending.forEach(callback=>callback(0))}
 const canvas=host.querySelector('canvas')!
 const press=(key:string,options:KeyboardEventInit={})=>{const event=new KeyboardEvent('keydown',{key,cancelable:true,...options});canvas.dispatchEvent(event);flush();return event}
 flush();return {host,canvas,press,flush,inspect}
}
test('canvas exposes keyboard instructions and pans/zooms through the same camera as the buttons',()=>{
 const {host,canvas,press,flush}=setup()
 expect(canvas.tabIndex).toBe(0);expect(canvas.getAttribute('role')).toBe('application')
 expect(document.getElementById(canvas.getAttribute('aria-describedby')!)?.textContent).toContain('Tab leaves the map')
 const initial=translate.mock.calls.at(-1)!,initialScale=scale.mock.calls.at(-1)![0]
 expect(press('ArrowRight').defaultPrevented).toBe(true);expect(translate.mock.calls.at(-1)![0]).toBe(initial[0]-40)
 press('ArrowDown',{shiftKey:true});expect(translate.mock.calls.at(-1)![1]).toBe(initial[1]-120)
 press('+');expect(scale.mock.calls.at(-1)![0]).toBeGreaterThan(initialScale)
 host.querySelector<HTMLButtonElement>('[aria-label="Zoom out"]')!.click();flush();expect(scale.mock.calls.at(-1)![0]).toBeCloseTo(initialScale)
 press('0');expect(translate.mock.calls.at(-1)).toEqual(initial)
 expect(press('Tab').defaultPrevented).toBe(false);expect(press('+',{metaKey:true}).defaultPrevented).toBe(false)
})
test('Enter inspects the center and a detached map no longer reacts to keyboard input',()=>{
 const {canvas,press,inspect}=setup();press('Enter');expect(inspect).toHaveBeenCalledWith(10)
 map!.destroy();map=undefined;translate.mockClear();canvas.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowRight'}))
 expect(translate).not.toHaveBeenCalled()
})
test('reduced-motion live signals stay static, expire and release their timer on teardown',()=>{
 vi.useFakeTimers({toFake:['setTimeout','clearTimeout','performance']})
 const media=window.matchMedia('(prefers-reduced-motion: reduce)');Object.defineProperty(media,'matches',{value:true});vi.spyOn(window,'matchMedia').mockReturnValue(media)
 const {flush}=setup('live');map!.pulseKill(1,1e9);flush();expect(frames).toHaveLength(0);expect(vi.getTimerCount()).toBe(1)
 vi.advanceTimersByTime(45001);flush();expect(frames).toHaveLength(0);expect(vi.getTimerCount()).toBe(0)
 map!.pulseKill(1,1e9);flush();expect(vi.getTimerCount()).toBe(1);map!.destroy();map=undefined;expect(vi.getTimerCount()).toBe(0)
})
