// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
import type {BrowserUser} from '../src/browser/browser-request'
import type {EntityTracker} from '../src/models/trackers'
const owner:BrowserUser={characterId:4242,characterName:'Owner',isAdmin:false,corporationId:null,corporationName:null,allianceId:null,allianceName:null,lastSeenNotificationId:0}
const second:BrowserUser={...owner,characterId:4343,characterName:'Second'}
const tracked:EntityTracker={id:1,target_id:4242,target_type:'character',target_name:'Owner',target_ticker:null,enabled:true,notifications_enabled:true,created_at:'2026-10-07T00:00:00Z',updated_at:'2026-10-07T00:00:00Z',event_count:0,last_event_at:null,unread_notifications:0}
let store:typeof import('../src/browser/tracker-state')|undefined
const settle=()=>new Promise<void>(resolve=>setTimeout(resolve,0))
afterEach(async()=>{await store?.setTrackerUser(null);store=undefined;vi.unstubAllGlobals();vi.resetModules()})

test('changing account cancels old reads and ignores a response which arrives despite cancellation',async()=>{
  const pending:Array<{path:string;signal:AbortSignal;complete:(response:Response)=>void}>=[]
  vi.stubGlobal('fetch',vi.fn<typeof fetch>((path,options)=>new Promise(resolve=>pending.push({path:String(path),signal:options!.signal!,complete:resolve}))))
  store=await import('../src/browser/tracker-state')
  const old=store.setTrackerUser(owner),next=store.setTrackerUser(second)
  expect(pending).toHaveLength(4);expect(pending.slice(0,2).every(request=>request.signal.aborted)).toBe(true)
  for(const request of pending.slice(2))request.complete(Response.json(request.path.includes('tracker-notifications')?{notifications:[],unreadCount:2}:{trackers:[],limit:7}))
  await next
  for(const request of pending.slice(0,2))request.complete(Response.json(request.path.includes('tracker-notifications')?{notifications:[],unreadCount:99}:{trackers:[],limit:99}))
  await old
  expect(store.trackerState).toMatchObject({user:{characterId:4343},limit:7,unread:2,loadingTrackers:false,loadingNotifications:false,error:null})
})

test('failed tracker reads keep their HTTP error and remain retryable',async()=>{
  let fail=true
  vi.stubGlobal('fetch',vi.fn<typeof fetch>(async path=>String(path).includes('tracker-notifications')?Response.json({notifications:[],unreadCount:0}):fail?Response.json({message:'Unavailable'},{status:503}):Response.json({trackers:[],limit:50})))
  store=await import('../src/browser/tracker-state');await store.setTrackerUser(owner)
  expect(store.trackerState.error).toMatchObject({status:503})
  await store.refreshTrackerNotifications()
  expect(store.trackerState.error).toMatchObject({status:503})
  await expect(store.refreshTrackers()).rejects.toMatchObject({status:503})
  expect(store.trackerState.error).toMatchObject({status:503,message:'Unavailable'})
  fail=false;await store.refreshTrackers();expect(store.trackerState.error).toBeNull()
})

class Socket {
  static instances:Socket[]=[]
  readyState=0
  onopen:(()=>void)|null=null
  onmessage:((event:{data:string})=>void)|null=null
  onclose:(()=>void)|null=null
  onerror:(()=>void)|null=null
  messages:string[]=[]
  closed=false
  constructor(readonly url:string){Socket.instances.push(this)}
  send(value:string){this.messages.push(value)}
  close(){this.closed=true;this.readyState=3}
}
test('tracker relay subscriptions survive page restoration and old sockets cannot reconnect after logout',async()=>{
  Socket.instances=[];vi.stubGlobal('WebSocket',Socket)
  vi.stubGlobal('fetch',vi.fn<typeof fetch>(async path=>Response.json(String(path).includes('tracker-notifications')?{notifications:[],unreadCount:0}:{trackers:[tracked],limit:50})))
  store=await import('../src/browser/tracker-state');await store.setTrackerUser(owner)
  const first=Socket.instances[0]!;first.readyState=1;first.onopen!()
  expect(JSON.parse(first.messages[0]!)).toEqual({action:'subscribe',topics:['attacker.4242','victim.4242']})
  window.dispatchEvent(new Event('pagehide'));expect(first.closed).toBe(true);expect(first.onclose).toBeNull()
  window.dispatchEvent(new Event('pageshow'));expect(Socket.instances).toHaveLength(2)
  await store.setTrackerUser(null);expect(Socket.instances[1]!.closed).toBe(true)
  window.dispatchEvent(new Event('pageshow'));await settle();expect(Socket.instances).toHaveLength(2)
})

test('relay transport keeps wss and rejects malformed envelopes while accepting typed routing keys',async()=>{
  Socket.instances=[];vi.stubGlobal('WebSocket',Socket)
  const {createRelayConnection}=await import('../src/utils/relaySocketCore'),messages:unknown[]=[]
  const relay=createRelayConnection({wsUrl:'wss://example.test/ws',path:'/killlist',onMessage:(channel,data,keys)=>messages.push({channel,data,keys})})
  relay.connect();const socket=Socket.instances[0]!
  expect(socket.url).toBe('wss://example.test/ws/killlist')
  socket.onmessage!({data:'bad JSON'});socket.onmessage!({data:'{"channel":7}'});socket.onmessage!({data:'[]'})
  expect(messages).toHaveLength(0)
  socket.onmessage!({data:'{"channel":"killlist","data":{"killmail":{"killmail_id":1}},"routing_keys":["victim.1",7]}'})
  expect(messages).toEqual([{channel:'killlist',data:{killmail:{killmail_id:1}},keys:['victim.1']}])
  relay.dispose()
})
