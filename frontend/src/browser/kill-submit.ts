import {RequestComponent} from './controllers/request-component'
import {registerController} from './controllers/application'
import {createRelayConnection} from '../utils/relaySocketCore'
import {BrowserRequestError} from './browser-request'
import {eventElement} from './shared/dom'
import {isRecord,isAbortError,errorMessage} from './shared/json'
import {submissionStorageKey,submissionMaxAge,esiKillmailLink,submittedIDs,readSubmissionState,readSubmissionResult,readArrival,detailArrival,compactSubmissionValue,type ReadyNotice,type Arrival,type SubmissionResult} from './shared/kill-submissions'

type ToastType = 'info'|'error'|'success'
let owner:KillSubmissionController|undefined
export function killSubmissionToast(type:ToastType,message:string) {owner?.toast(type,message)}
export function submitKillmails(text:string,onResult?:(result:SubmissionResult)=>void,signal?:AbortSignal):Promise<SubmissionResult|null> {return owner?.submit(text,onResult,signal) ?? Promise.resolve(null)}

export class KillSubmissionController extends RequestComponent {
  private alive = false
  private pending = new Set<number>()
  private uncertain = new Set<number>()
  private attempts = new Map<object,number[]>()
  private notices:ReadyNotice[] = []
  private startedAt = 0
  private arrivedCount = 0
  private uid = 0
  private pollRead:AbortController|undefined
  private pollTimer:ReturnType<typeof setTimeout>|undefined
  private noticeTimer:ReturnType<typeof setTimeout>|undefined
  private relay:ReturnType<typeof createRelayConnection>|undefined
  private toasts = new Map<HTMLElement,{expires:number;timer:ReturnType<typeof setTimeout>|undefined}>()
  private get readyRoot() {return this.q<HTMLElement>('[data-kill-ready-host]')}
  private get toastRoot() {return this.q<HTMLElement>('[data-kill-submit-host]')}
  connect() {owner = this;this.alive = true;this.restore();this.render();this.resume()}
  private restore() {
    let value:unknown = null,unavailable = false;try{value = JSON.parse(sessionStorage.getItem(submissionStorageKey) ?? 'null')}catch{unavailable = true}
    if(unavailable && (this.pending.size || this.uncertain.size || this.notices.length)) {for(const id of this.uncertain) this.pending.add(id);this.uncertain.clear();this.attempts.clear();return}
    const state = readSubmissionState(value);this.startedAt = state.startedAt;this.pending = new Set([...state.pending,...state.uncertain]);this.uncertain.clear();this.attempts.clear();this.notices = state.notices;this.uid = this.notices.reduce((value,notice)=>Math.max(value,notice.uid),this.uid);this.persist()
  }
  private persist() {try{sessionStorage.setItem(submissionStorageKey,JSON.stringify({startedAt:this.startedAt,pending:[...this.pending],uncertain:[...this.uncertain],notices:this.notices}))}catch{/* Watcher continues in this document. */}}
  private visible() {return this.alive && !document.hidden}
  toast(type:ToastType,message:string) {
    if(!this.alive) return
    const item = document.createElement('button');item.type = 'button';item.className = 'kill-submit-toast '+type;item.dataset.killSubmitToast = '';item.textContent = message;this.toastRoot.append(item)
    const state:{expires:number;timer:ReturnType<typeof setTimeout>|undefined} = {expires:Date.now()+6000,timer:undefined};this.toasts.set(item,state);state.timer = this.after(6000,()=>{this.toasts.delete(item);item.remove()})
  }
  private render() {
    this.stopTimer(this.noticeTimer);this.noticeTimer = undefined;this.notices = this.notices.filter(notice=>notice.dismissAt > Date.now())
    const retained = new Set(this.notices.map(notice=>String(notice.uid)))
    for(const button of this.readyRoot.querySelectorAll<HTMLElement>('[data-kill-ready-uid]')) if(!retained.has(button.dataset.killReadyUid ?? '')) button.remove()
    const rendered = new Set([...this.readyRoot.querySelectorAll<HTMLElement>('[data-kill-ready-uid]')].map(button=>Number(button.dataset.killReadyUid)))
    for(const notice of this.notices) {
      if(rendered.has(notice.uid)) continue
      const button = document.createElement('button');button.type = 'button';button.className = 'kill-ready-toast';button.dataset.killReadyUid = String(notice.uid)
      if(notice.ship_type_id) {const image = document.createElement('img');image.src = `/images/types/${notice.ship_type_id}/icon?size=64`;image.alt = '';image.width = image.height = 48;button.append(image)}
      const span = document.createElement('span'),label = document.createElement('b'),name = document.createElement('strong'),summary = document.createElement('small'),arrow = document.createElement('i');label.textContent = 'Killmail Ready';name.textContent = notice.victim_character_name || notice.victim_corporation_name || '#'+notice.killmail_id;summary.textContent = (notice.ship_name ?? '')+(notice.ship_name && notice.total_value ? ' · ' : '')
      if(notice.total_value) {const value = document.createElement('em');value.textContent = compactSubmissionValue(notice.total_value);summary.append(value)}span.append(label,name,summary);arrow.textContent = '↗';button.append(span,arrow);this.readyRoot.append(button)
    }
    if(this.visible() && this.notices.length) this.noticeTimer = this.after(Math.max(1,Math.min(...this.notices.map(notice=>notice.dismissAt))-Date.now()),()=>{this.render();this.persist()})
  }
  private arrive(data:Arrival) {if(!this.visible() || !this.pending.delete(data.killmail_id)) return;this.arrivedCount++;this.notices.push({...data,uid:++this.uid,dismissAt:Date.now()+(60+(this.arrivedCount-1)*15)*1000});if(!this.pending.size) this.stopWatcher();this.persist();this.render()}
  private stopWatcher() {this.relay?.dispose();this.relay = undefined;this.stopTimer(this.pollTimer);this.pollTimer = undefined;this.pollRead?.abort();this.pollRead = undefined;if(!this.pending.size && !this.uncertain.size) this.startedAt = 0}
  private ensureWatcher() {
    if(!this.visible() || !this.pending.size) return
    if(!this.relay) {this.relay = createRelayConnection({wsUrl:location.origin,path:'/ws/killlist',reconnect:true,shouldReconnect:()=>this.visible() && this.pending.size > 0,onOpen:socket=>socket.send(JSON.stringify({action:'subscribe',topics:['all']})),onMessage:(channel,data)=>{if(channel === 'killlist' && isRecord(data)) {const arrival = readArrival(data.killmail);if(arrival) this.arrive(arrival)}}});this.relay.connect()}
    if(!this.pollTimer && !this.pollRead) this.pollTimer = this.after(5000,()=>{this.pollTimer = undefined;void this.poll()})
  }
  private async poll() {
    if(!this.visible() || !this.pending.size || this.pollRead) return
    if(Date.now()-this.startedAt >= submissionMaxAge) {this.pending.clear();this.uncertain.clear();this.stopWatcher();this.persist();return}
    const read = new AbortController();this.pollRead = read
    try {for(const id of [...this.pending]) {
      if(!this.visible() || this.pollRead !== read) break
      try {const exists:unknown = await this.request(`/api/killmail/${id}/exists`,{signal:read.signal,cache:'no-store'});if(!isRecord(exists) || exists.exists !== true || !this.pending.has(id)) continue
        let detail:unknown = null;try{detail = await this.request(`/api/killmail/${id}`,{signal:read.signal,cache:'no-store'})}catch(error){if(isAbortError(error)) throw error}
        if(this.pollRead === read) this.arrive(detailArrival(id,detail))
      }catch(error){if(isAbortError(error)) break}
    }}finally{if(this.pollRead === read) {this.pollRead = undefined;this.ensureWatcher()}}
  }
  private watch(ids:number[]) {if(!ids.length) return;if(!this.pending.size && !this.uncertain.size) {this.startedAt = Date.now();this.arrivedCount = 0}for(const id of ids) this.pending.add(id);this.persist();this.ensureWatcher()}
  private releaseAttempt(attempt:object) {this.attempts.delete(attempt);this.uncertain = new Set([...this.attempts.values()].flat())}
  async submit(text:string,onResult?:(result:SubmissionResult)=>void,signal?:AbortSignal):Promise<SubmissionResult|null> {
    if(!this.alive || signal?.aborted) return null
    const payload = text.trim();if(!payload) {this.toast('info','Paste one or more killmail links first');return null}
    const candidates = submittedIDs(payload),attempt = {};this.attempts.set(attempt,candidates);if(candidates.length && !this.startedAt) this.startedAt = Date.now();for(const id of candidates) this.uncertain.add(id);this.persist()
    try {
      const result = readSubmissionResult(await this.request<unknown>('/api/killmail/post',{method:'POST',body:{text:payload}},'Failed to queue'))
      if(!this.alive) return null
      this.releaseAttempt(attempt)
      const existing = new Set(result.existingIds);if(result.accepted) this.watch(result.killmails.filter(id=>!existing.has(id)).slice(0,result.accepted));this.persist()
      if(!signal?.aborted) onResult?.(result)
      if(!result.total) {this.toast('info',result.message || 'No valid ESI killmail links found');return result}
      if(result.total === 1 && result.existing === 1) {this.toast('info',`Killmail #${result.killmails[0]} already exists`);if(!signal?.aborted) this.openKill(result.killmails[0]);return result}
      const parts:string[] = [];if(result.accepted) parts.push(`${result.accepted} queued — watching for arrival`);if(result.existing) parts.push(`${result.existing} already exist`);if(result.rejected) parts.push(`${result.rejected} failed`);this.toast(result.rejected ? 'error' : 'success',parts.join(', '));return result
    }catch(error){if(error instanceof BrowserRequestError && error.status < 500) {this.releaseAttempt(attempt);this.persist()}if(this.alive && !isAbortError(error)) this.toast('error',errorMessage(error,'Failed to queue killmail'));return null}
  }
  openKill(id:number) {location.assign('/kill/'+id)}
  paste(event:ClipboardEvent) {if(!this.alive || event.defaultPrevented) return;const target = eventElement(event.target);if(target instanceof HTMLElement && (target.closest('input,textarea,select,[contenteditable="true"]') || target.isContentEditable)) return;const text = event.clipboardData?.getData('text/plain')?.trim();if(text && esiKillmailLink.test(text)) {event.preventDefault();void this.submit(text)}}
  click(event:MouseEvent) {
    const target = eventElement(event.target),toast = target?.closest<HTMLElement>('[data-kill-submit-toast]');if(toast) {this.stopTimer(this.toasts.get(toast)?.timer);this.toasts.delete(toast);toast.remove();return}
    const uid = Number(target?.closest<HTMLElement>('[data-kill-ready-uid]')?.dataset.killReadyUid),notice = this.notices.find(notice=>notice.uid === uid);if(notice) {this.notices = this.notices.filter(item=>item !== notice);this.persist();this.render();this.openKill(notice.killmail_id)}
  }
  private pause() {this.stopWatcher();this.stopTimer(this.noticeTimer);this.noticeTimer = undefined;for(const state of this.toasts.values()) {this.stopTimer(state.timer);state.timer = undefined}}
  private resume() {if(!this.visible()) return;for(const [item,state] of this.toasts) {if(state.expires <= Date.now()) {item.remove();this.toasts.delete(item)}else if(!state.timer) state.timer = this.after(state.expires-Date.now(),()=>{item.remove();this.toasts.delete(item)})}this.render();this.ensureWatcher();void this.poll()}
  visibility() {if(document.hidden) this.pause();else this.resume()}
  hide() {this.alive = false;this.pause();this.persist();super.hide()}
  show(event:PageTransitionEvent) {if(event.persisted) {this.alive = true;for(const id of this.uncertain) this.pending.add(id);this.uncertain.clear();this.attempts.clear();this.persist();this.resume()}}
  disconnect() {super.disconnect();if(owner === this) owner = undefined}
}
registerController('kill-submission',KillSubmissionController)
