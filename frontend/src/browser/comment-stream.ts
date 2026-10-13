// One transport per document, shared by every mounted comment thread.
import {createRelayConnection} from '../utils/relaySocketCore'
import {isRecord} from './shared/json'
import type {CommentsThreadResponses} from '../../shared/api/types.gen'

export type Comment=CommentsThreadResponses[200]['roots'][number]
export interface CommentEvent {event_type:string;comment_id:number;target_type:number;target_id:number;target_slug:string|null;domain_id:number|null;comment?:Partial<Comment>}
type Listener={onEvent:(event:CommentEvent)=>void;onState?:((connected:boolean)=>void)}
const listeners=new Set<Listener>()
let transport:ReturnType<typeof createRelayConnection>|undefined
let connected=false
const integer=(value:unknown,min:number)=>{if(typeof value!=='number'&&typeof value!=='string')return null;const number=Number(value);return Number.isSafeInteger(number)&&number>=min?number:null}
export function readCommentEvent(value:unknown):CommentEvent|null {
  if(!isRecord(value)||typeof value.event_type!=='string')return null
  const id=integer(value.comment_id,1),type=integer(value.target_type,1),target=integer(value.target_id,0)
  if(id===null||type===null||target===null)return null
  const event:CommentEvent={event_type:value.event_type,comment_id:id,target_type:type,target_id:target,target_slug:typeof value.target_slug==='string'?value.target_slug:null,domain_id:integer(value.domain_id,1)}
  const raw=value.comment
  if(isRecord(raw)){
    if(raw.id!==undefined&&integer(raw.id,1)!==id)return null
    const comment:Partial<Comment>={id}
    for(const key of ['body_md','body_html','character_name','corporation_name','created_at','updated_at'] as const)if(typeof raw[key]==='string')comment[key]=raw[key]
    for(const key of ['alliance_name','target_slug','edited_at','deleted_at'] as const){const field=raw[key];if(field===null||typeof field==='string')comment[key]=field}
    for(const key of ['character_id','corporation_id','target_type','target_id','depth','moderation_status','reports_count','reply_count','visibility'] as const){const field=integer(raw[key],0);if(field!==null)comment[key]=field}
    for(const key of ['parent_id','root_id','domain_id','alliance_id','deleted_by'] as const){const field=raw[key];if(field===null)comment[key]=null;else{const number=integer(field,0);if(number!==null)comment[key]=number}}
    if(typeof raw.flagged==='boolean')comment.flagged=raw.flagged
    event.comment=comment
  }
  return event
}
export function mergeCommentEvent(previous:Comment|undefined,event:CommentEvent):Comment|undefined {
  if(!event.comment)return previous
  const empty:Comment={id:event.comment_id,target_type:event.target_type,target_id:event.target_id,target_slug:event.target_slug,domain_id:event.domain_id,parent_id:null,root_id:null,depth:0,body_md:'',body_html:'',character_id:0,character_name:'',corporation_id:0,corporation_name:'',alliance_id:null,alliance_name:null,created_at:'',updated_at:'',edited_at:null,deleted_at:null,deleted_by:null,flagged:false,moderation_status:0,reports_count:0,visibility:0}
  return {...empty,...previous,...event.comment}
}
export function subscribeCommentStream(onEvent:Listener['onEvent'],onState?:Listener['onState']){
  const listener:Listener={onEvent,...(onState?{onState}:{})};listeners.add(listener);onState?.(connected)
  if(!transport){
    transport=createRelayConnection({wsUrl:new URL('/ws',location.href).href,path:'/comments',reconnect:true,visibilityPause:true,shouldReconnect:()=>listeners.size>0,
      onState:value=>{connected=value;for(const subscriber of listeners)subscriber.onState?.(value)},
      onMessage:(channel,data)=>{if(channel!=='comments')return;const event=readCommentEvent(data);if(event)for(const subscriber of listeners)subscriber.onEvent(event)},
    });transport.connect()
  }
  let subscribed=true
  return ()=>{if(!subscribed)return;subscribed=false;listeners.delete(listener);if(!listeners.size){const previous=transport;transport=undefined;previous?.dispose();connected=false}}
}
