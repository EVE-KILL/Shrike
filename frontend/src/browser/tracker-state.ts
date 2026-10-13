// Navbar and tracker pages share one account state and relay connection.
import {requestJSON,type BrowserUser} from './browser-request'
import {createRelayConnection,trackerTopics} from './account-data'
import {isRecord} from './shared/json'
import type {EntityTracker,TrackerNotification} from '../models/trackers'
import type {AccountTrackerCreateData,AccountTrackerUpdateData,AccountTrackerCreateResponses,AccountTrackerUpdateResponses,AccountTrackersResponses,AccountTrackerNotificationsResponses} from '../../shared/api/types.gen'
export {trackerTopics}

export interface TrackerState {
  user:BrowserUser|null;trackers:EntityTracker[];limit:number;notifications:TrackerNotification[];unread:number
  loadingTrackers:boolean;loadingNotifications:boolean;error:Error|null
  browserAlerts:boolean;browserPermission:NotificationPermission|'unsupported'
}
type TrackedKill={killmail_id:number;ship_type_id?:number;ship_name?:string;solar_system_name?:string}
const listeners=new Set<(state:TrackerState)=>void>(),seenKills=new Set<number>()
const pending=new Set<AbortController>()
const failures:{trackers:Error|null;notifications:Error|null}={trackers:null,notifications:null}
const state:TrackerState={user:null,trackers:[],limit:50,notifications:[],unread:0,loadingTrackers:false,loadingNotifications:false,error:null,browserAlerts:false,browserPermission:'unsupported'}
let epoch=0,relay:ReturnType<typeof createRelayConnection>|null=null,topicKey='',active=true
const notify=()=>{for(const listener of listeners)listener(state)}
function setFailure(resource:keyof typeof failures,error:Error|null){
  failures[resource]=error;state.error=failures.trackers||failures.notifications
}
const json=(method:string,body:unknown):RequestInit=>({method,headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})
async function readState<T>(path:string):Promise<T>{
  const request=new AbortController();pending.add(request)
  try{return await requestJSON<T>(path,{signal:request.signal})}
  finally{pending.delete(request)}
}
export const trackerState=state
export function subscribeTrackers(listener:(state:TrackerState)=>void):()=>void {
  listeners.add(listener);listener(state)
  return ()=>{listeners.delete(listener)}
}
function stopRelay(){relay?.dispose();relay=null;topicKey=''}
function trackedKill(payload:unknown):TrackedKill|null {
  if(!isRecord(payload)||!isRecord(payload.killmail))return null
  const kill=payload.killmail,id=kill.killmail_id
  if(typeof id!=='number')return null
  return {killmail_id:id,
    ship_type_id:typeof kill.ship_type_id==='number'?kill.ship_type_id:undefined,
    ship_name:typeof kill.ship_name==='string'?kill.ship_name:undefined,
    solar_system_name:typeof kill.solar_system_name==='string'?kill.solar_system_name:undefined}
}
export function configureTrackerRelay(){
  const topics=trackerTopics(state.trackers),key=topics.join(',')
  if(key===topicKey)return
  stopRelay()
  if(!key||!state.user||!active)return
  topicKey=key
  relay=createRelayConnection({wsUrl:new URL('/ws',location.href).href,path:'/killlist',reconnect:true,visibilityPause:false,
    shouldReconnect:()=>Boolean(state.user)&&topicKey===key&&active,
    onOpen:ws=>ws.send(JSON.stringify({action:'subscribe',topics})),
    onMessage:async(channel,data)=>{
      const kill=channel==='killlist'?trackedKill(data):null
      if(!kill)return
      const generation=epoch
      try{await refreshTrackerNotifications()}catch{/* Shared state records errors. */}
      if(generation===epoch)showBrowserAlert(kill)
    }})
  relay.connect()
}
function showBrowserAlert(kill:TrackedKill){
  if(!state.browserAlerts||state.browserPermission!=='granted'||seenKills.has(kill.killmail_id))return
  seenKills.add(kill.killmail_id)
  if(seenKills.size>200){const oldest=seenKills.values().next().value;if(oldest!==undefined)seenKills.delete(oldest)}
  const persisted=state.notifications.find(n=>n.killmail_id===kill.killmail_id),target=persisted?.target_name?` · ${persisted.target_name}`:''
  const ship=kill.ship_name||persisted?.ship_name||'ship',system=kill.solar_system_name||persisted?.solar_system_name||'an unknown system'
  const alert=new Notification('Tracked kill'+target,{body:`${ship} destroyed in ${system}`,icon:kill.ship_type_id?`/images/types/${kill.ship_type_id}/icon?size=64`:undefined,tag:`tracker-kill-${kill.killmail_id}`})
  alert.onclick=()=>{window.focus();location.assign(`/kill/${kill.killmail_id}`);alert.close()}
}
export async function setTrackerUser(user:BrowserUser|null){
  if(state.user?.characterId===user?.characterId)return
  ++epoch;for(const request of pending)request.abort();pending.clear()
  stopRelay();state.user=user;state.trackers=[];state.notifications=[];state.unread=0
  state.loadingTrackers=false;state.loadingNotifications=false;state.error=null;seenKills.clear()
  failures.trackers=null;failures.notifications=null
  if('Notification'in window){
    state.browserPermission=Notification.permission
    try{state.browserAlerts=localStorage.getItem('ek_tracker_browser_alerts_while_open')==='1'&&Notification.permission==='granted'}catch{state.browserAlerts=false}
  }
  notify()
  if(user)await Promise.allSettled([refreshTrackers(),refreshTrackerNotifications()])
}
export async function refreshTrackers(){
  if(!state.user||state.loadingTrackers)return
  const generation=epoch;state.loadingTrackers=true;notify()
  try{
    const result=await readState<AccountTrackersResponses[200]>('/api/me/trackers')
    if(generation!==epoch)return
    state.trackers=result.trackers||[];state.limit=Number(result.limit??50);setFailure('trackers',null);configureTrackerRelay()
  }catch(error){if(generation===epoch)setFailure('trackers',error instanceof Error?error:new Error('Could not load trackers'));throw error}
  finally{if(generation===epoch){state.loadingTrackers=false;notify()}}
}
export async function refreshTrackerNotifications(){
  if(!state.user||state.loadingNotifications)return
  const generation=epoch;state.loadingNotifications=true;notify()
  try{
    const result=await readState<AccountTrackerNotificationsResponses[200]>('/api/me/tracker-notifications?limit=50')
    if(generation!==epoch)return
    state.notifications=result.notifications||[];state.unread=Number(result.unreadCount??0);setFailure('notifications',null)
  }catch(error){if(generation===epoch)setFailure('notifications',error instanceof Error?error:new Error('Could not load notifications'));throw error}
  finally{if(generation===epoch){state.loadingNotifications=false;notify()}}
}
export async function markTrackersRead(){
  if(!state.unread)return
  const generation=epoch
  state.notifications=state.notifications.map(item=>({...item,is_read:true}))
  state.trackers=state.trackers.map(item=>({...item,unread_notifications:0}));state.unread=0;notify()
  try{await requestJSON('/api/me/tracker-notifications/read',json('POST',{all:true}))}catch{if(generation===epoch)await refreshTrackerNotifications()}
}
export async function setBrowserAlerts(enabled:boolean):Promise<boolean>{
  if(!('Notification'in window)){state.browserPermission='unsupported';state.browserAlerts=false;notify();return false}
  let permission=Notification.permission
  if(enabled&&permission==='default')permission=await Notification.requestPermission()
  state.browserPermission=permission;state.browserAlerts=enabled&&permission==='granted'
  try{localStorage.setItem('ek_tracker_browser_alerts_while_open',state.browserAlerts?'1':'0')}catch{/* Storage can be disabled. */}
  notify();return state.browserAlerts
}
export async function createTracker(body:AccountTrackerCreateData['body']){
  const result=await requestJSON<AccountTrackerCreateResponses[201]>('/api/me/trackers',json('POST',body))
  await refreshTrackers();return result
}
export async function updateTracker(id:number,body:AccountTrackerUpdateData['body']){
  const result=await requestJSON<AccountTrackerUpdateResponses[200]>('/api/me/trackers/'+Number(id),json('PATCH',body))
  await refreshTrackers();return result
}
export async function deleteTracker(id:number){
  await requestJSON('/api/me/trackers/'+Number(id),{method:'DELETE'})
  await Promise.allSettled([refreshTrackers(),refreshTrackerNotifications()])
}
window.addEventListener('pagehide',()=>{active=false;stopRelay()})
window.addEventListener('pageshow',()=>{active=true;configureTrackerRelay()})
