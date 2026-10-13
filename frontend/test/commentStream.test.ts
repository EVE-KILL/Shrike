// @vitest-environment happy-dom
import {afterEach,beforeEach,expect,test,vi} from 'vitest'
import {subscribeCommentStream,readCommentEvent,mergeCommentEvent} from '../src/browser/comment-stream'

class Socket {
  static instances:Socket[]=[]
  readyState=0
  onopen:(()=>void)|null=null
  onmessage:((event:{data:string})=>void)|null=null
  onclose:(()=>void)|null=null
  onerror:(()=>void)|null=null
  closed=false
  constructor(readonly url:string){Socket.instances.push(this)}
  close(){this.closed=true;this.readyState=3}
}
let releases:Array<()=>void>=[]
const event={event_type:'new',comment_id:7,target_type:10,target_id:0,target_slug:'FixtureCampaign',domain_id:null,comment:{id:7,character_id:4242,body_html:'<p>Fixture</p>',created_at:'2026-10-07T12:00:00Z'}}
beforeEach(()=>{Socket.instances=[];releases=[];vi.stubGlobal('WebSocket',Socket)})
afterEach(()=>{for(const release of releases)release();vi.useRealTimers();vi.unstubAllGlobals();vi.restoreAllMocks()})
test('threads share one comment socket and the last subscriber owns cleanup',()=>{
  const first=vi.fn(),second=vi.fn(),state=vi.fn();const a=subscribeCommentStream(first,state),b=subscribeCommentStream(second);releases.push(a,b);expect(Socket.instances).toHaveLength(1);const socket=Socket.instances[0];expect(socket.url).toMatch(/\/ws\/comments$/);socket.readyState=1;socket.onopen!();expect(state).toHaveBeenLastCalledWith(true)
  socket.onmessage!({data:JSON.stringify({channel:'comments',data:event})});expect(first).toHaveBeenCalledOnce();expect(second).toHaveBeenCalledOnce();a();a();expect(socket.closed).toBe(false);b();expect(socket.closed).toBe(true);expect(socket.onclose).toBeNull()
})
test('transport skips malformed envelopes and comment data without interrupting valid frames',()=>{
  const receive=vi.fn();releases.push(subscribeCommentStream(receive));const socket=Socket.instances[0]
  for(const data of ['bad','[]',JSON.stringify({channel:'killlist',data:event}),JSON.stringify({channel:'comments',data:{event_type:'new',comment_id:'bad'}}),JSON.stringify({channel:'comments',data:{...event,comment:{id:8}}})])socket.onmessage!({data})
  expect(receive).not.toHaveBeenCalled();socket.onmessage!({data:JSON.stringify({channel:'comments',data:event})});expect(receive).toHaveBeenCalledOnce()
})
test('visibility suspends sockets and reconnect backoff ends when the document has no listeners',async()=>{
  vi.useFakeTimers();let hidden=false;vi.spyOn(document,'hidden','get').mockImplementation(()=>hidden);releases.push(subscribeCommentStream(vi.fn()));const first=Socket.instances[0]
  first.onclose!();await vi.advanceTimersByTimeAsync(999);expect(Socket.instances).toHaveLength(1);await vi.advanceTimersByTimeAsync(1);expect(Socket.instances).toHaveLength(2)
  hidden=true;document.dispatchEvent(new Event('visibilitychange'));expect(Socket.instances[1].closed).toBe(true);await vi.advanceTimersByTimeAsync(30001);expect(Socket.instances).toHaveLength(2);hidden=false;document.dispatchEvent(new Event('visibilitychange'));expect(Socket.instances).toHaveLength(3)
  releases[0]();await vi.advanceTimersByTimeAsync(30001);expect(Socket.instances).toHaveLength(3)
})
test('minimal production frames preserve cached Markdown and narrow optional fields',()=>{
  const read=readCommentEvent(event)!;expect(read.target_id).toBe(0);expect(read.comment).not.toHaveProperty('body_md');const created=mergeCommentEvent(undefined,read)!;expect(created).toMatchObject({id:7,body_md:'',body_html:'<p>Fixture</p>',deleted_at:null,moderation_status:0});expect(mergeCommentEvent({...created,body_md:'Current Markdown'},read)?.body_md).toBe('Current Markdown')
  expect(readCommentEvent({...event,comment:{id:7,body_html:{bad:true},flagged:true,depth:'4'}})?.comment).toEqual({id:7,flagged:true,depth:4});expect(readCommentEvent({...event,target_id:-1})).toBeNull()
})
