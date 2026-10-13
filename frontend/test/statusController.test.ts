// @vitest-environment happy-dom
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {StatusController} from '../src/browser/status'
import {registerController} from '../src/browser/controllers/application'
import {readStatusSnapshot} from '../src/browser/shared/status-model'
import {renderStatusPanels} from '../src/browser/shared/status-render'

const directory = resolve(import.meta.dirname,'fixtures/status')
const markup = readFileSync(resolve(directory,'page.html'),'utf8')
const snapshot:unknown = JSON.parse(readFileSync(resolve(directory,'snapshot.json'),'utf8'))
const errors = vi.fn(),settle = () => new Promise<void>(resolve=>setTimeout(resolve,0))
class Socket {
  static instances:Socket[] = []
  static failures = 0
  readyState = 0
  onopen:(()=>void)|null = null
  onmessage:((event:{data:string})=>void)|null = null
  onclose:(()=>void)|null = null
  onerror:(()=>void)|null = null
  closed = false
  constructor(readonly url:string) {if(Socket.failures > 0) {Socket.failures--;throw new Error('Fixture socket unavailable')}Socket.instances.push(this)}
  close() {this.closed = true;this.readyState = 3}
  open() {this.readyState = 1;this.onopen?.()}
  send(data:unknown = snapshot,channel = 'status') {this.onmessage?.({data:JSON.stringify({channel,data})})}
}
let hidden = false
function q<T extends Element = HTMLElement>(root:ParentNode,selector:string):T {const node = root.querySelector<T>(selector);if(!node) throw new Error('Missing status fixture '+selector);return node}
async function mount() {document.body.innerHTML = markup;await settle();await settle();return q<HTMLElement>(document.body,'[data-status-page]')}
function hide() {window.dispatchEvent(new PageTransitionEvent('pagehide'))}
function show() {const event = new PageTransitionEvent('pageshow');Object.defineProperty(event,'persisted',{value:true});window.dispatchEvent(event)}
beforeEach(()=>{
  registerController('status',StatusController).handleError = errors;errors.mockClear();Socket.instances = [];Socket.failures = 0;hidden = false
  vi.spyOn(document,'hidden','get').mockImplementation(()=>hidden);vi.stubGlobal('WebSocket',Socket)
})
afterEach(async()=>{vi.useRealTimers();document.body.replaceChildren();await settle();expect(errors.mock.calls.map(call=>String(call[0]))).toEqual([]);vi.unstubAllGlobals();vi.restoreAllMocks()})

test('the status stream renders the real Go payload while retaining dashboard classes/formatting',async()=>{
  const root = await mount(),socket = Socket.instances[0]
  expect(Socket.instances).toHaveLength(1);expect(socket.url).toMatch(/\/ws\/status$/)
  expect(q<HTMLElement>(root,'[data-status-panels]').hidden).toBe(true);socket.open();socket.send()
  expect(q(root,'[data-status-connection]').textContent).toBe('Live');expect(root.classList.contains('status-connected')).toBe(true)
  expect(q<HTMLElement>(root,'[data-status-loading]').hidden).toBe(true);expect(q<HTMLElement>(root,'[data-status-error]').hidden).toBe(true)
  const panels = q(root,'[data-status-panels]');expect(panels.textContent).toContain('42,000 pilots');expect(panels.textContent).toContain('1b ISK');expect(panels.textContent).toContain('125/s');expect(panels.textContent).toContain('250ms')
  expect(root.querySelectorAll('.status-ingest-hour')).toHaveLength(3);expect(root.querySelectorAll('.status-db-graph polyline')).toHaveLength(5)
  expect(panels.textContent).toContain('killmails');expect(panels.textContent).toContain('90% hit')
})
test('invalid envelopes/channels and malformed nested values cannot replace a valid snapshot',async()=>{
  const root = await mount(),socket = Socket.instances[0];socket.send();const panels = q(root,'[data-status-panels]'),before = panels.innerHTML
  for(const raw of ['bad','[]','null',JSON.stringify({channel:'status',data:{queues:[]}})]) socket.onmessage!({data:raw})
  socket.send(snapshot,'killlist');expect(panels.innerHTML).toBe(before)
  socket.send({queues:{},system:{platform:'Fixture',load_average:{bad:true}},database:{history:'wrong',tables:[]},zkb_ingest:{new_hourly:['bad',-1,Infinity],repost_hourly:[]}})
  expect(panels.textContent).toContain('Fixture');expect(panels.innerHTML).not.toContain('NaN');expect(panels.innerHTML).not.toContain('Infinity')
})
test('missing optional Go sources render unavailable/collecting states without inventing panels',async()=>{
  const root = await mount();Socket.instances[0].send({queues:{},system:{platform:'linux'}})
  const panels = q(root,'[data-status-panels]');expect(panels.textContent).toContain('collecting');expect(panels.textContent).toContain('statistics unavailable');expect(panels.textContent).toContain('— ISK')
  expect(panels.textContent).not.toContain('Cloudflare');expect(panels.textContent).not.toContain('zKB Ingest');expect(panels.textContent).not.toContain('Image Server')
})
test('read-only external panel models preserve grouping, totals and escaped labels',()=>{
  const data = readStatusSnapshot({queues:{},cloudflare:{last_24h:{requests:100,status_codes:{200:90,304:5,500:2}},last_1h:{requests:10}},image_server:{folderStats:{types:{fileCount:999},regions:{fileCount:5,sizeKB:1024},'<script>bad</script>':{fileCount:2,sizeKB:1}},cacheValidation:{validationErrors:3}},websocket:{status:'ok',redis:true,uptime:3600,channels:['all'],connections:3,endpoints:{'/ws/killlist':{connections:3,subscriptions:{'character.1':1,'character.2':2,all:3}}}}})!
  const html = renderStatusPanels(data),holder = document.createElement('div');holder.innerHTML = html
  expect(holder.textContent).toContain('Cloudflare');expect(holder.textContent).toContain('95');expect(holder.textContent).toContain('7 files');expect(holder.textContent).not.toContain('999');expect(holder.textContent).toContain('character');expect(holder.querySelector('script')).toBeNull();expect(html).toContain('&lt;script&gt;')
})
test('ingest hover/focus/tap tooltip survives a new frame with current counts',async()=>{
  const root = await mount(),socket = Socket.instances[0];socket.send();let bar = q<HTMLElement>(root,'[data-status-hour="2"]')
  bar.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));expect(q<HTMLElement>(root,'.status-ingest-tooltip').hidden).toBe(false);expect(q(root,'.status-ingest-tooltip').textContent).toContain('Current hour')
  socket.send({...readStatusSnapshot(snapshot),zkb_ingest:{window_hours:3,new_hourly:[2,4,20],repost_hourly:[1,2,3]}})
  expect(q(root,'.status-ingest-tooltip').textContent).toContain('23');bar = q<HTMLElement>(root,'[data-status-hour="2"]');expect(bar.classList.contains('active')).toBe(true)
  bar.dispatchEvent(new MouseEvent('mouseout',{bubbles:true}));expect(q<HTMLElement>(root,'.status-ingest-tooltip').hidden).toBe(true)
  const first = q<HTMLElement>(root,'[data-status-hour="0"]');first.dispatchEvent(new FocusEvent('focusin',{bubbles:true}));expect(first.getAttribute('tabindex')).toBe('0');expect(q<HTMLElement>(root,'.status-ingest-tooltip').hidden).toBe(false)
  first.dispatchEvent(new FocusEvent('focusout',{bubbles:true}));expect(q<HTMLElement>(root,'.status-ingest-tooltip').hidden).toBe(true);bar.click();expect(bar.classList.contains('active')).toBe(true)
})
test('moving within a chart bar keeps its tooltip while selecting another clears the old highlight',async()=>{
  const root = await mount();Socket.instances[0].send();const first = q<HTMLElement>(root,'[data-status-hour="0"]'),last = q<HTMLElement>(root,'[data-status-hour="2"]')
  first.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));first.dispatchEvent(new MouseEvent('mouseout',{bubbles:true,relatedTarget:first.firstElementChild}));expect(q<HTMLElement>(root,'.status-ingest-tooltip').hidden).toBe(false)
  last.dispatchEvent(new MouseEvent('mouseover',{bubbles:true}));expect(first.classList.contains('active')).toBe(false);expect(last.classList.contains('active')).toBe(true)
})
test('hour tooltips stay within the chart at both edges and on very narrow panels',async()=>{
  const root = await mount();Socket.instances[0].send()
  const chart = q<HTMLElement>(root,'.status-ingest-chart'),tip = q<HTMLElement>(root,'.status-ingest-tooltip')
  vi.spyOn(chart,'clientWidth','get').mockReturnValue(330);vi.spyOn(tip,'offsetWidth','get').mockReturnValue(190)
  q(root,'[data-status-hour="0"]').dispatchEvent(new FocusEvent('focusin',{bubbles:true}));expect(tip.style.left).toBe('95px')
  q(root,'[data-status-hour="2"]').dispatchEvent(new FocusEvent('focusin',{bubbles:true}));expect(tip.style.left).toBe('235px')
  vi.spyOn(chart,'clientWidth','get').mockReturnValue(150)
  q(root,'[data-status-hour="2"]').dispatchEvent(new FocusEvent('focusin',{bubbles:true}));expect(tip.style.left).toBe('75px')
})
test('disconnect keeps the latest snapshot while using capped reconnect backoff',async()=>{
  const root = await mount();vi.useFakeTimers();Socket.instances[0].open();Socket.instances[0].send();const before = q(root,'[data-status-panels]').innerHTML
  Socket.instances[0].onclose!();expect(q(root,'[data-status-connection]').textContent).toBe('Connecting...');expect(q<HTMLElement>(root,'[data-status-error]').hidden).toBe(true);expect(q(root,'[data-status-panels]').innerHTML).toBe(before)
  await vi.advanceTimersByTimeAsync(999);expect(Socket.instances).toHaveLength(1);await vi.advanceTimersByTimeAsync(1);expect(Socket.instances).toHaveLength(2)
  const delays = [2000,4000,8000,16000,30000,30000]
  for(const delay of delays) {const count = Socket.instances.length;Socket.instances.at(-1)!.onclose!();await vi.advanceTimersByTimeAsync(delay-1);expect(Socket.instances).toHaveLength(count);await vi.advanceTimersByTimeAsync(1);expect(Socket.instances).toHaveLength(count+1)}
  Socket.instances.at(-1)!.open();Socket.instances.at(-1)!.onclose!();const count = Socket.instances.length;await vi.advanceTimersByTimeAsync(1000);expect(Socket.instances).toHaveLength(count+1)
})
test('socket constructor failures remain retryable without duplicate connections',async()=>{
  await mount();vi.useFakeTimers();Socket.failures = 1;Socket.instances[0].onclose!();await vi.advanceTimersByTimeAsync(1000);expect(Socket.instances).toHaveLength(1)
  await vi.advanceTimersByTimeAsync(2000);expect(Socket.instances).toHaveLength(2);Socket.instances[1].open();await vi.advanceTimersByTimeAsync(30000);expect(Socket.instances).toHaveLength(2)
})
test('visibility closes its socket, clears retry timers and resumes on the same mounted page',async()=>{
  const root = await mount();vi.useFakeTimers();Socket.instances[0].open();Socket.instances[0].send();Socket.instances[0].onclose!()
  hidden = true;document.dispatchEvent(new Event('visibilitychange'));await vi.advanceTimersByTimeAsync(30001);expect(Socket.instances).toHaveLength(1)
  hidden = false;document.dispatchEvent(new Event('visibilitychange'));expect(Socket.instances).toHaveLength(2);Socket.instances[1].open();expect(root.classList.contains('status-connected')).toBe(true)
})
test('pagehide disposes callbacks and page return starts one fresh stream while retaining panels',async()=>{
  const root = await mount(),socket = Socket.instances[0];socket.open();socket.send();const message = socket.onmessage!,before = q(root,'[data-status-panels]').innerHTML
  hide();expect(socket.closed).toBe(true);expect(socket.onclose).toBeNull();expect(socket.onmessage).toBeNull();show();show();expect(Socket.instances).toHaveLength(2)
  message({data:JSON.stringify({channel:'status',data:{queues:{},system:{platform:'Late host'}}})});expect(q(root,'[data-status-panels]').innerHTML).toBe(before)
  Socket.instances[1].open();Socket.instances[1].send({queues:{},system:{platform:'Returned host'}});expect(root.textContent).toContain('Returned host')
})
test('detaching a dashboard removes its reconnect and visibility listeners',async()=>{
  const root = await mount(),socket = Socket.instances[0];vi.useFakeTimers();socket.onclose!();root.remove();await vi.advanceTimersByTimeAsync(0)
  hidden = true;document.dispatchEvent(new Event('visibilitychange'));hidden = false;document.dispatchEvent(new Event('visibilitychange'));await vi.advanceTimersByTimeAsync(30001)
  expect(Socket.instances).toHaveLength(1);expect(socket.onmessage).toBeNull()
})
test('a hidden document opens no stream until visible and disconnect errors remain visible without data',async()=>{
  hidden = true;const root = await mount();expect(Socket.instances).toHaveLength(0);hidden = false;document.dispatchEvent(new Event('visibilitychange'));expect(Socket.instances).toHaveLength(1)
  Socket.instances[0].open();expect(q<HTMLElement>(root,'[data-status-error]').hidden).toBe(true);Socket.instances[0].onclose!();expect(q<HTMLElement>(root,'[data-status-error]').hidden).toBe(false)
})
test('model boundaries normalize metric strings/invalid arrays and keep decimal wallet precision',()=>{
  expect(readStatusSnapshot(null)).toBeUndefined();expect(readStatusSnapshot({database:[]})).toBeUndefined()
  const data = readStatusSnapshot({queues:{broken:null,killmails:{active:'2',waiting:-5,failed:'bad'}},system:{load_average:['1.25',Infinity,{},-5]},database:{tables:{bad:null},history:[null,{queries_per_second:'2.5',read_bytes_per_second:-1}]},wallet:{total_balance:'123.45'}})!
  expect(data.queues.killmails).toEqual({active:2,waiting:0});expect(data.system?.load_average).toEqual([1.25,0,0,0]);expect(data.database?.history).toEqual([{queries_per_second:2.5,read_bytes_per_second:0}]);expect(data.wallet?.total_balance).toBe('123.45')
  expect(renderStatusPanels(data)).not.toMatch(/NaN|Infinity/)
})
