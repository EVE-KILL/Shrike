import {SettingsComponent} from './base'
import {registerController} from '../application'
import {escapeHTML as e} from '../../shared/html'
import {eventElement,requiredElement} from '../../shared/dom'
import {isAbortError} from '../../shared/json'
import {accountError} from '../../shared/account-actions'
import {formatDateTime} from '../../settings-data'
import type {UserSessionsLegacyResponses} from '../../../../shared/api/types.gen'
const symbol=(name:string)=>`<svg class="icon" aria-hidden="true"><use href="/assets/icons.svg#${name}"></use></svg>`

export class SettingsSessionsController extends SettingsComponent{
  private entries:UserSessionsLegacyResponses[200]['sessions']=[]
  private revoking:number|null=null
  private others=false
  private confirm=false
  private timer:ReturnType<typeof setTimeout>|undefined
  private pending=false
  private loadError=''
  private epoch=0
  private regionNames=typeof Intl.DisplayNames!=='undefined'?new Intl.DisplayNames(['en'],{type:'region'}):null
  connect(){this.pending=false;this.revoking=null;this.others=false;this.confirm=false;void this.load()}
  hide(){++this.epoch;this.pending=false;super.hide()}
  show(event:PageTransitionEvent){if(event.persisted)void this.load()}
  private timeAgo(iso:string){const seconds=Math.round((new Date(iso).getTime()-Date.now())/1000),f=new Intl.RelativeTimeFormat('en',{numeric:'auto'});if(Math.abs(seconds)<60)return f.format(seconds,'second');const minutes=Math.round(seconds/60);if(Math.abs(minutes)<60)return f.format(minutes,'minute');const hours=Math.round(minutes/60);if(Math.abs(hours)<24)return f.format(hours,'hour');const days=Math.round(hours/24);return Math.abs(days)<30?f.format(days,'day'):formatDateTime(iso)}
    private render() {
      const count=this.entries.filter(s=>!s.current).length,button=this.q<HTMLButtonElement>('[data-settings-revoke-others]');button.hidden=!count;button.disabled=this.others||this.revoking!==null||this.pending;requiredElement<HTMLElement>(button,'[data-settings-revoke-icon]').innerHTML=symbol(this.others?'loader-2':'log-out');requiredElement<HTMLElement>(button,'[data-settings-revoke-icon]').className='mr-1.5 '+(this.others?'animate-spin':'');this.q<HTMLElement>('[data-settings-revoke-label]').textContent=this.others?'Revoking…':this.confirm?'Confirm logout of '+count:'Log out everywhere else';button.className='px-3 py-2 rounded-lg border text-xs font-medium transition-colors cursor-pointer disabled:opacity-50 '+(this.confirm?'border-red-500/40 bg-red-500/10 text-red-300':'border-white/[0.08] bg-white/[0.03] text-gray-400 hover:text-red-300 hover:border-red-500/30');
      const list=this.q<HTMLElement>('[data-settings-sessions-list]');
      if(this.pending&&!this.entries.length){list.innerHTML='<div class="glass-panel p-8 text-center text-sm text-gray-500"><span class="animate-spin mr-2">'+symbol('loader-2')+'</span>Loading sessions…</div>';return;}
      if(this.loadError){list.innerHTML='<div class="glass-panel p-5 text-sm text-red-300" role="alert">'+e(this.loadError)+' <button type="button" data-settings-sessions-retry class="ml-2 cursor-pointer text-xs text-blue-400 hover:text-blue-300">Retry</button></div>';return;}
      list.innerHTML='<div class="space-y-3">'+this.entries.map(s=>`<div class="glass-panel p-4 flex flex-col sm:flex-row sm:items-center gap-4 ${s.current?'border-blue-500/25':''}"><div data-settings-session-device class="w-10 h-10 rounded-lg bg-white/[0.04] border border-white/[0.07] flex items-center justify-center flex-shrink-0"><span class="text-lg text-gray-400">${symbol(s.device==='Mobile'?'smartphone':s.device==='Tablet'?'tablet':'monitor')}</span></div><div class="flex-1 min-w-0"><div class="flex items-center gap-2 flex-wrap"><span class="text-sm font-semibold text-white">${e(s.browser)} on ${e(s.operatingSystem)}</span>${s.current?'<span class="px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-300 text-fine font-semibold uppercase tracking-wider">This device</span>':''}</div><div class="text-xs text-gray-500 mt-1 truncate">${e([s.countryCode?(this.regionNames?.of(s.countryCode)||s.countryCode):null,s.ipAddress].filter(Boolean).join(' · ')||'Location unavailable')}</div><div class="text-fine text-gray-600 mt-1">Last active ${e(this.timeAgo(s.lastSeenAt))} · Signed in ${e(formatDateTime(s.createdAt))}</div></div><button type="button" data-settings-revoke="${Number(s.id)}" ${this.revoking!==null||this.others||this.pending?'disabled':''} class="self-start sm:self-center px-3 py-2 rounded-lg text-xs font-medium border border-white/[0.08] text-gray-400 hover:text-red-300 hover:border-red-500/30 hover:bg-red-500/[0.06] transition-colors cursor-pointer disabled:opacity-50">${this.revoking===s.id?'<span class="animate-spin mr-1">'+symbol('loader-2')+'</span>':''}${s.current?'Log out':'Revoke'}</button></div>`).join('')+'</div>';
    }
  private async load(){
    if(this.pending)return
    this.pending=true;this.loadError='';const epoch=this.epoch;this.render()
    try{const data=await this.request<UserSessionsLegacyResponses[200]>('/api/user/sessions');if(epoch===this.epoch)this.entries=data.sessions}
    catch(error){if(epoch===this.epoch&&!isAbortError(error))this.loadError=accountError(error,'Could not load your sessions.')}
    finally{if(epoch===this.epoch){this.pending=false;if(this.element.isConnected)this.render()}}
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target)
    if(target?.closest('[data-settings-sessions-retry]')){await this.load();return}
    if(this.others||this.revoking!==null||this.pending)return
    const revoke=target?.closest<HTMLButtonElement>('[data-settings-revoke]'),all=target?.closest<HTMLButtonElement>('[data-settings-revoke-others]')
    if(!revoke&&!all)return
    if(all&&!this.confirm){this.confirm=true;this.stopTimer(this.timer);this.timer=this.after(5000,()=>{this.confirm=false;this.render()});this.render();return}
    this.stopTimer(this.timer);this.error('[data-settings-sessions-error]')
    if(revoke){const entry=this.entries.find(session=>session.id===Number(revoke.dataset.settingsRevoke));if(!entry)return;if(entry.current){await this.logout(revoke);return}this.revoking=entry.id}else this.others=true
    this.render();const epoch=this.epoch
    try{await this.request(all?'/api/user/sessions/revoke-others':`/api/user/sessions/${this.revoking}/revoke`,{method:'POST'},all?'Could not revoke the other sessions':'Could not revoke that session');if(epoch===this.epoch){this.confirm=false;await this.load()}}
    catch(error){this.error('[data-settings-sessions-error]',error)}finally{if(epoch===this.epoch){this.revoking=null;this.others=false;if(this.element.isConnected)this.render()}}
  }
  private async logout(button:HTMLButtonElement){
    button.disabled=true
    try{await this.request('/auth/logout',{method:'POST'});location.assign('/')}
    catch(error){this.error('[data-settings-sessions-error]',error);button.disabled=false}
  }
}
registerController('settings-sessions',SettingsSessionsController)
