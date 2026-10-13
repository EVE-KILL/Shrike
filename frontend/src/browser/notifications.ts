import {Controller} from '@hotwired/stimulus'
import {readBrowserSession,requestJSON} from './browser-request'
import {targetFor,timeAgo,snippet,trackerSummary,trackerRole,formatIsk} from './account-data'
import {subscribeTrackers,trackerState,refreshTrackerNotifications,markTrackersRead} from './tracker-state'
import {registerController} from './controllers/application'
import {eventElement,placeElement} from './shared/dom'
import {escapeHTML as e} from './shared/html'
import type {AccountNotificationRepliesResponses} from '../../shared/api/types.gen'
import type {TrackerNotification} from '../models/trackers'

type Reply=AccountNotificationRepliesResponses[200]['replies'][number]
type NotificationRow={kind:'reply';created_at:string;item:Reply}|{kind:'tracker';created_at:string;item:TrackerNotification}
const icon=(name:string,classes='')=>`<svg class="icon ${classes}" aria-hidden="true"><use href="/assets/icons.svg#${name.replace('lucide:','')}"></use></svg>`

// Reuse links so updates retain focus, scroll and already-loaded images.
export function reconcileNotificationRows(list:HTMLElement,markup:string){
  const focus=list.contains(list.ownerDocument.activeElement)?list.ownerDocument.activeElement:null
  const desired=document.createElement('template');desired.innerHTML=markup
  const existing=new Map([...list.querySelectorAll<HTMLAnchorElement>('[data-notification-row]')].map(row=>[row.dataset.notificationRow,row]))
  const retained=new Set<HTMLAnchorElement>();let index=0
  for(const next of desired.content.children){
    if(!(next instanceof HTMLAnchorElement))continue
    let row=existing.get(next.dataset.notificationRow)
    if(row){row.className=next.className;row.setAttribute('href',next.getAttribute('href')||'#');if(row.innerHTML!==next.innerHTML)row.innerHTML=next.innerHTML}
    else row=next.cloneNode(true) as HTMLAnchorElement
    retained.add(row);if(list.children[index]!==row)placeElement(list,row,list.children[index]||null);index++
  }
  for(const row of [...list.children])if(!(row instanceof HTMLAnchorElement)||!retained.has(row))row.remove()
  if(focus instanceof HTMLElement&&focus.isConnected&&list.contains(focus)&&list.ownerDocument.activeElement!==focus)focus.focus({preventScroll:true})
}

function renderRow(row:NotificationRow,cursor:number):string {
  const {kind,item,created_at}=row
  let content:string,href:string,isUnread:boolean
  if(kind==='reply'){
    const target=targetFor(item.target_type)
    href=target.pathFn(item.target_id,item.id,item.target_slug);isUnread=item.id>cursor
    content=`<img src="https://images.evetech.net/characters/${Number(item.character_id)}/portrait?size=64" alt="${e(item.character_name)}" width="32" height="32" class="h-8 w-8 flex-shrink-0 rounded-md ring-1 ring-white/[0.08]"><div class="min-w-0 flex-1"><div class="flex items-center gap-1.5 text-fine"><span class="truncate font-semibold text-white">${e(item.character_name)}</span><span class="text-gray-600">replied</span><span class="inline-flex items-center gap-1 rounded px-1.5 py-px text-fine font-medium ${target.color}">${icon(target.icon,'text-fine')}${e(target.label)}</span><span class="ml-auto tabular-nums text-gray-600">${e(timeAgo(created_at))}</span></div><p class="mt-1 line-clamp-2 text-xs text-gray-300">${e(snippet(item.body_html))}</p>${item.parent_snippet?`<p class="mt-1 line-clamp-1 text-fine italic text-gray-500" title="${e(item.parent_snippet)}">in reply to: ${e(item.parent_snippet)}</p>`:''}</div>`
  }else{
    href=`/kill/${item.killmail_id}`;isUnread=!item.is_read
    content=`${item.ship_type_id?`<img src="/images/types/${Number(item.ship_type_id)}/icon?size=64" alt="${e(item.ship_name||'Destroyed ship')}" width="32" height="32" class="h-8 w-8 flex-shrink-0 rounded-md ring-1 ring-white/[0.08]">`:`<span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md bg-red-500/10 text-red-400">${icon('crosshair')}</span>`}<div class="min-w-0 flex-1"><div class="flex items-center gap-1.5 text-fine"><span class="truncate font-semibold text-white">${e(item.target_name)}</span><span class="rounded bg-amber-500/10 px-1.5 py-px capitalize text-amber-300">${e(trackerRole(item))}</span><span class="ml-auto tabular-nums text-gray-600">${e(timeAgo(created_at))}</span></div><p class="mt-1 line-clamp-2 text-xs text-gray-300">${e(trackerSummary(item))}</p><p class="mt-1 text-fine tabular-nums text-gray-500">${e(formatIsk(item.total_value))} ISK</p></div>`
  }
  return `<a href="${e(href)}" data-notification-row="${kind}:${item.id}" class="relative block w-full cursor-pointer border-b border-white/[0.04] px-3 py-2.5 text-left transition-colors last:border-b-0 hover:bg-blue-500/[0.06] ${isUnread?'bg-blue-500/[0.03]':''}">${isUnread?'<span class="absolute left-1 top-1/2 h-1.5 w-1.5 -translate-y-1/2 rounded-full bg-blue-400"></span>':''}<div class="flex items-start gap-2.5 pl-3">${content}</div></a>`
}

export class NotificationsController extends Controller<HTMLDetailsElement> {
  static targets=['list','badge','count','read','footer']
  declare readonly listTarget:HTMLElement
  declare readonly badgeTarget:HTMLElement
  declare readonly countTarget:HTMLElement
  declare readonly readTarget:HTMLButtonElement
  declare readonly footerTarget:HTMLElement
  private replies:Reply[]=[]
  private cursor=0
  private key=''
  private empty:string|undefined
  private loading=false
  private lifetime:AbortController|undefined
  private unsubscribe:(()=>void)|undefined

  connect(){
    this.empty??=this.listTarget.innerHTML
    this.lifetime=new AbortController()
    void this.loadSession(this.lifetime)
  }
  disconnect(){this.lifetime?.abort();this.unsubscribe?.();this.unsubscribe=undefined;this.loading=false}
  private async loadSession(lifetime:AbortController){
    let user
    try{user=await readBrowserSession()}catch{return}
    if(!user||lifetime.signal.aborted||!this.element.isConnected)return
    this.key='ek_notifications_last_seen_id:'+user.characterId
    this.cursor=Math.max(this.cursor,Number(user.lastSeenNotificationId)||0)
    try{this.cursor=Math.max(this.cursor,Number(localStorage.getItem(this.key))||0)}catch{/* Storage can be disabled. */}
    this.unsubscribe=subscribeTrackers(()=>this.render())
    void this.backfill()
  }
  private render(){
    if(this.lifetime?.signal.aborted||!this.element.isConnected)return
    const unread=this.replies.filter(item=>item.id>this.cursor).length+trackerState.unread
    this.badgeTarget.hidden=!unread;this.badgeTarget.textContent=unread>99?'99+':String(unread)
    this.countTarget.hidden=!unread;this.countTarget.textContent=unread+' new';this.readTarget.hidden=!unread
    const rows:NotificationRow[]=[...this.replies.map(item=>({kind:'reply' as const,created_at:item.created_at,item})),
      ...trackerState.notifications.map(item=>({kind:'tracker' as const,created_at:item.created_at,item}))]
    rows.sort((a,b)=>Date.parse(b.created_at)-Date.parse(a.created_at));this.footerTarget.hidden=!rows.length
    if(rows.length)reconcileNotificationRows(this.listTarget,rows.map(row=>renderRow(row,this.cursor)).join(''))
    else if(this.listTarget.innerHTML!==this.empty)this.listTarget.innerHTML=this.empty||''
  }
  async backfill(){
    const lifetime=this.lifetime
    if(this.loading||!this.key||!lifetime||lifetime.signal.aborted)return
    this.loading=true
    try{
      const result=await requestJSON<AccountNotificationRepliesResponses[200]>('/api/notifications/replies?'+new URLSearchParams({limit:'50',...(this.cursor>0?{since:String(this.cursor)}:{})}),{signal:lifetime.signal})
      if(lifetime.signal.aborted||this.lifetime!==lifetime)return
      const known=new Set(this.replies.map(item=>item.id))
      for(const item of result.replies||[])if(!known.has(item.id)){this.replies.push(item);known.add(item.id)}
      this.replies.sort((a,b)=>b.id-a.id);this.replies=this.replies.slice(0,100);this.render()
    }catch{/* Preserve existing notifications when backfill is unavailable. */}
    finally{if(this.lifetime===lifetime)this.loading=false}
  }
  private async markRepliesRead(){
    const target=this.replies[0]?.id,lifetime=this.lifetime
    if(!target||target<=this.cursor||!lifetime||lifetime.signal.aborted)return
    this.cursor=target
    try{localStorage.setItem(this.key,String(this.cursor))}catch{/* Storage can be disabled. */}
    this.render()
    try{
      const result=await requestJSON<{lastSeenNotificationId:number}>('/api/notifications/mark-read',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({id:target}),signal:lifetime.signal})
      if(lifetime.signal.aborted||this.lifetime!==lifetime)return
      this.cursor=Math.max(this.cursor,Number(result.lastSeenNotificationId??target))
      try{localStorage.setItem(this.key,String(this.cursor))}catch{/* Storage can be disabled. */}
      this.render()
    }catch{/* Keep the local read cursor, as before. */}
  }
  open(){if(this.element.open)void refreshTrackerNotifications().catch(()=>{})}
  markRead(){void Promise.allSettled([this.markRepliesRead(),markTrackersRead()])}
  follow(event:MouseEvent){if(eventElement(event.target)?.closest('a'))this.element.open=false}
}
registerController('notifications',NotificationsController)
