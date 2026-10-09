// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {createMapAIIDFeed,createMapLiveFeed,type MapFeedScope} from '../src/browser/map-live'
import {parseLiveKill} from '../src/browser/kill-row'
class Socket {
 static instances:Socket[] = []
 static OPEN = 1
 readyState = 1
 onopen:(()=>void)|null = null
 onmessage:((event:{data:string})=>void)|null = null
 onclose:(()=>void)|null = null
 onerror:(()=>void)|null = null
 closed = false
 messages:string[] = []
 constructor(readonly url:string) {Socket.instances.push(this)}
 send(message:string) {this.messages.push(message)}
 close() {this.closed = true;this.readyState = 3;this.onclose?.()}
 kill(kill:unknown) {this.onmessage?.({data:JSON.stringify({channel:'killlist',data:{killmail:kill}})})}
}
const markup = readFileSync(resolve(import.meta.dirname,'fixtures/map/page.html'),'utf8').replace(/<script(?![^>]*type="application\/json")[\s\S]*?<\/script>/g,'').replace(/<link[^>]*rel="stylesheet"[^>]*>/g,'')
const renderer = ()=>({pulseKill:vi.fn(),focusLiveSystem:vi.fn()})
const system = (id:number,distance = 0)=>({solar_system_id:id,distance,system_name:'System '+id,region_id:10000001,security:0.5,x:id,z:id})
const kill = (id:number,system = 1,minutesAgo = 0)=>parseLiveKill({killmail_id:id,solar_system_id:system,ship_type_id:587,ship_name:'Rifter',victim_character_name:'<script>pilot</script>',killmail_time:new Date(Date.now()-minutesAgo*60000).toISOString(),total_value:1e9})!
const scope = (systems:ReturnType<typeof system>[],kills:ReturnType<typeof kill>[] = []):MapFeedScope=>({systems,kills,anchors:systems.map(system=>({solar_system_id:system.solar_system_id,system_name:system.system_name}))})
const feeds:Array<{destroy():void}> = []
let fetchMock:ReturnType<typeof vi.fn<typeof fetch>>
function q<T extends Element=HTMLElement>(root:ParentNode,selector:string):T {const element = root.querySelector<T>(selector);if(!element)throw new Error('Missing map fixture '+selector);return element}
function mount() {const doc = new DOMParser().parseFromString(markup,'text/html'),root = q<HTMLElement>(doc,'[data-map-root]');document.body.append(root);return root}
function hidden(value:boolean) {Object.defineProperty(document,'hidden',{configurable:true,value});document.dispatchEvent(new Event('visibilitychange'))}
beforeEach(()=>{vi.useFakeTimers();vi.setSystemTime(new Date('2026-10-08T12:00:00Z'));Socket.instances = [];vi.stubGlobal('WebSocket',Socket);hidden(false);fetchMock = vi.fn<typeof fetch>(async()=>Response.json({kills:[]}));vi.stubGlobal('fetch',fetchMock)})
afterEach(()=>{for(const feed of feeds.splice(0))feed.destroy();document.body.replaceChildren();vi.restoreAllMocks();vi.unstubAllGlobals();vi.useRealTimers()})

test('AIID watch changes retain rows, relay and freshness history without replaying alarms',()=>{
 hidden(true);const root = mount(),first = renderer(),next = renderer(),alarms = vi.fn(),feed = createMapAIIDFeed(root,first,scope([system(1)],[kill(1)]),alarms);feeds.push(feed)
 expect(Socket.instances).toHaveLength(1);expect(first.pulseKill).toHaveBeenCalledWith(1,1e9,0);expect(alarms).not.toHaveBeenCalled();const socket = Socket.instances[0];socket.onopen?.();expect(JSON.parse(socket.messages[0])).toEqual({action:'subscribe',topics:['all']});socket.kill(kill(2));const list = q(root,'[data-map-aiid-list]'),retained = list.firstElementChild;expect(retained?.getAttribute('data-kill-id')).toBe('2');expect(alarms.mock.calls).toEqual([['near']])
 feed.update(next,scope([system(1,5),system(2,10)],[kill(3,2)]));expect(Socket.instances).toHaveLength(1);expect(list.firstElementChild).toBe(retained);expect([...list.children].map(row=>row.getAttribute('data-kill-id'))).toEqual(['2','3']);expect(next.pulseKill).toHaveBeenCalledWith(2,1e9,0)
 socket.kill(kill(2));socket.kill(kill(4,2,16));socket.kill(kill(5,3));expect(alarms.mock.calls).toEqual([['near']]);socket.kill(kill(6,2));expect(alarms.mock.calls).toEqual([['near'],['outer']]);expect(list.querySelectorAll('[data-kill-id="2"]')).toHaveLength(1);expect(list.querySelector('[data-kill-id="5"]')).toBeNull()
 feed.update(next,scope([system(3)]));expect([...list.children].map(row=>row.getAttribute('data-kill-id'))).toEqual(['5']);hidden(true);expect(socket.closed).toBe(false);feed.update(next,scope([]));expect(list.textContent).toContain('Choose your position');feed.destroy();expect(socket.closed).toBe(true);expect(vi.getTimerCount()).toBe(0)
})
test('live bootstrap preserves earlier streamed rows, bounds visible rows and rejects malformed payloads',async()=>{
 let resolveRead!:(response:Response)=>void;fetchMock.mockImplementation(()=>new Promise(resolve=>resolveRead = resolve));const root = mount(),map = renderer(),feed = createMapLiveFeed(root,map,scope([system(1)]));feeds.push(feed);const socket = Socket.instances[0];socket.kill(kill(1));const row = q(root,'[data-kill-id="1"]');resolveRead(Response.json({kills:[kill(1),kill(2),{killmail_id:'bad'}]}));await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-kill-id="1"]')).toBe(row);expect(root.querySelectorAll('[data-kill-id]')).toHaveLength(2)
 for(let id=3;id<250;id++)socket.kill(kill(id));expect(root.querySelectorAll('[data-kill-id]')).toHaveLength(60);expect(q(root,'[data-map-live-count]').textContent).toBe('60 recent kills');socket.kill({killmail_id:'bad'});expect(root.querySelectorAll('[data-kill-id]')).toHaveLength(60)
})
test('AIID alarms use both distance bands and ignore stale, future and duplicate pulses',()=>{
 const root = mount(),map = renderer(),alarm = vi.fn(),feed = createMapAIIDFeed(root,map,scope([system(1,4),system(2,5),system(3,10)]),alarm);feeds.push(feed);const socket = Socket.instances[0];socket.kill(kill(1));socket.kill(kill(2,2));socket.kill(kill(3,3));socket.kill(kill(3,3));socket.kill(kill(4,1,16));socket.kill(kill(5,1,-2));socket.kill(kill(6,99));expect(alarm.mock.calls).toEqual([['near'],['outer'],['outer']]);expect(map.pulseKill).toHaveBeenCalledTimes(3)
})
test('live maps pause hidden relay traffic, reconnect with backoff and dispose listeners/timers',async()=>{
 const root = mount(),feed = createMapLiveFeed(root,renderer(),scope([system(1)]));feeds.push(feed);const first = Socket.instances[0];first.onopen?.();expect(q(root,'[data-map-live-connection]').textContent).toBe('Connected');hidden(true);expect(first.closed).toBe(true);expect(first.onmessage).toBeNull();hidden(false);expect(Socket.instances).toHaveLength(2);Socket.instances[1].close();await vi.advanceTimersByTimeAsync(999);expect(Socket.instances).toHaveLength(2);await vi.advanceTimersByTimeAsync(1);expect(Socket.instances).toHaveLength(3);feed.destroy();hidden(false);await vi.advanceTimersByTimeAsync(60_000);expect(Socket.instances).toHaveLength(3);expect(vi.getTimerCount()).toBe(0)
})
test('disposed bootstrap cannot render late results and disposed scope updates do nothing',async()=>{
 let complete!:(response:Response)=>void;fetchMock.mockImplementation(()=>new Promise(resolve=>complete = resolve));const root = mount(),first = renderer(),feed = createMapLiveFeed(root,first,scope([system(1)]));feeds.push(feed);const list = q(root,'[data-map-live-list]'),before = list.innerHTML;feed.destroy();expect(fetchMock.mock.calls[0][1]?.signal?.aborted).toBe(true);complete(Response.json({kills:[kill(1)]}));await vi.advanceTimersByTimeAsync(0);const next = renderer();feed.update(next,scope([system(2)]));expect(list.innerHTML).toBe(before);expect(next.focusLiveSystem).not.toHaveBeenCalled()
})
test('row image fallback, escaped text, hover focus and relative clocks retain source behavior',async()=>{
 const root = mount(),map = renderer(),feed = createMapAIIDFeed(root,map,scope([system(1)],[kill(1)]));feeds.push(feed);const row = q<HTMLAnchorElement>(root,'[data-kill-id="1"]');expect(row.href).toContain('/kill/1');expect(row.textContent).toContain('<script>pilot</script>');expect(row.querySelector('script')).toBeNull();expect(q(row,'.map-live-value strong').textContent).toBe('1.00b');q<HTMLImageElement>(row,'img').dispatchEvent(new Event('error'));expect(q(row,'.map-live-image-fallback').textContent).toBe('?');row.dispatchEvent(new MouseEvent('mouseenter'));expect(map.focusLiveSystem).toHaveBeenCalledWith(1,{x:0,y:0});row.dispatchEvent(new MouseEvent('mouseleave'));expect(map.focusLiveSystem).toHaveBeenLastCalledWith(null);await vi.advanceTimersByTimeAsync(60_000);expect(q(row,'[data-kill-time]').textContent).toBe('1m ago')
})
test('empty Go slices are a legitimate live bootstrap',async()=>{
 fetchMock.mockResolvedValue(Response.json({kills:null}));const root = mount(),warning = vi.spyOn(console,'warn'),feed = createMapLiveFeed(root,renderer(),scope([system(1)]));feeds.push(feed);await vi.advanceTimersByTimeAsync(0);expect(q(root,'[data-map-live-list]').textContent).toBe('Waiting for killmails…');expect(warning).not.toHaveBeenCalled()
})
test('recreating a feed cannot leave duplicated rows or active old hover callbacks',()=>{
 const root = mount(),first = renderer(),feed = createMapAIIDFeed(root,first,scope([system(1)],[kill(1)]));feeds.push(feed);const row = q(root,'[data-kill-id="1"]');feed.destroy();first.focusLiveSystem.mockClear();row.dispatchEvent(new MouseEvent('mouseenter'));expect(first.focusLiveSystem).not.toHaveBeenCalled();const replacement = createMapAIIDFeed(root,renderer(),scope([system(1)],[kill(1)]));feeds.push(replacement);expect(root.querySelectorAll('[data-kill-id="1"]')).toHaveLength(1)
})
