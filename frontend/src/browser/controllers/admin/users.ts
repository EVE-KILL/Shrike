import {AdminComponent} from './base'
import {registerController} from '../application'
import {readQuerySeed} from '../../shared/query-seed'
import {eventElement} from '../../shared/dom'
import {errorMessage,isAbortError} from '../../shared/json'
import {escapeHTML} from '../../shared/html'
import {fmt,timeSince} from '../../esi-data'

export class AdminUsersController extends AdminComponent {
  private query:Record<string,string>|undefined
  private pending:AbortController|undefined
  private timer:ReturnType<typeof setTimeout>|undefined
  private interrupted=false
  private section:HTMLElement|null=null
  private visible=(event:Event)=>{if(event instanceof CustomEvent){if(!event.detail)this.hide();else if(this.interrupted)void this.load()}}
  connect(){this.query??=readQuerySeed(this.element.dataset.query);this.section=this.element.closest('[data-admin-section]');this.section?.addEventListener('admin-visibility',this.visible);this.times()}
  private get host(){return this.q<HTMLElement>('[data-admin-users-results]')}
  private times(){for(const time of this.host.querySelectorAll<HTMLElement>('[data-admin-relative-date]'))time.textContent=timeSince(time.dataset.adminRelativeDate||null)}
  private async load(){
    if(!this.query)return
    this.pending?.abort();const controller=new AbortController();this.pending=controller;this.interrupted=false
    this.host.setAttribute('aria-busy','true');this.error('[data-admin-users-error]')
    try{
      const text=await this.html('/_fragments/admin/users?'+new URLSearchParams(this.query),'Could not load users.',{signal:controller.signal})
      const doc=new DOMParser().parseFromString(text,'text/html'),next=doc.querySelector<HTMLElement>('[data-admin-users-result]')
      if(!next||doc.querySelector('[data-admin-users-error]:not([hidden])'))throw new Error('Could not load users.')
      if(this.pending!==controller)return
      this.host.replaceChildren(document.importNode(next,true));this.q<HTMLElement>('[data-admin-users-count]').textContent=fmt(Number(next.dataset.total))+' users';this.times()
    }catch(error){if(this.pending===controller&&!isAbortError(error)&&this.element.isConnected){const box=this.q<HTMLElement>('[data-admin-users-error]');box.innerHTML=`${escapeHTML(errorMessage(error))} <button type="button" data-admin-users-retry class="text-blue-400">Retry</button>`;box.hidden=false}}
    finally{if(this.pending===controller){this.pending=undefined;this.host.removeAttribute('aria-busy')}}
  }
  search(event:Event){
    const input=event.target;if(!(input instanceof HTMLInputElement)||!input.matches('[data-admin-user-search]')||!this.query)return
    this.stopTimer(this.timer);this.pending?.abort();this.query.search=input.value;this.query.page='1';this.interrupted=true;this.timer=this.after(300,()=>void this.load())
  }
  async click(event:MouseEvent){
    if(!this.query)return
    const target=eventElement(event.target),sort=target?.closest<HTMLElement>('[data-admin-user-sort]')?.dataset.adminUserSort
    if(sort){if(this.query.sort===sort)this.query.dir=this.query.dir==='desc'?'asc':'desc';else{this.query.sort=sort;this.query.dir='desc'}await this.load();return}
    const page=target?.closest<HTMLButtonElement>('[data-admin-user-page]')
    if(page&&!page.disabled&&page.dataset.adminUserPage){this.query.page=page.dataset.adminUserPage;await this.load();return}
    if(target?.closest('[data-admin-users-retry]')){await this.load();return}
    const row=target?.closest<HTMLElement>('[data-admin-user-row]');if(row?.dataset.adminUserRow)location.assign(row.dataset.adminUserRow)
  }
  keyboard(event:KeyboardEvent){const row=eventElement(event.target)?.closest<HTMLElement>('[data-admin-user-row]');if(row?.dataset.adminUserRow&&event.key==='Enter'){event.preventDefault();location.assign(row.dataset.adminUserRow)}}
  hide(){this.interrupted||=Boolean(this.pending);this.stopTimer(this.timer);this.timer=undefined;this.pending?.abort();this.pending=undefined;this.host.removeAttribute('aria-busy');super.hide()}
  show(event:PageTransitionEvent){if(event.persisted&&this.interrupted)void this.load()}
  disconnect(){this.section?.removeEventListener('admin-visibility',this.visible);this.section=null;super.disconnect()}
}
registerController('admin-users',AdminUsersController)
