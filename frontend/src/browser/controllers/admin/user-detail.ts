import {AdminComponent} from './base'
import {registerController} from '../application'
import {eventElement} from '../../shared/dom'
import {requiredElement} from '../../shared/dom'
import type {AdminUsersSetDiscordData} from '../../../../shared/api/types.gen'

export class AdminUserDetailController extends AdminComponent {
  private operation:object|undefined
  private interrupted=false
  private release(){for(const control of this.all<HTMLInputElement|HTMLButtonElement>('[data-admin-discord] input,[data-admin-discord] button,[data-admin-user-toggle]'))control.disabled=false;for(const spinner of this.all<HTMLElement>('[data-admin-discord-spinner],[data-admin-user-toggle-spinner]'))spinner.hidden=true}
  connect(){
    this.operation=undefined;this.release()
    for(const cell of this.all<HTMLElement>('[data-admin-config-value]')){let value:unknown;try{value=JSON.parse(cell.dataset.adminConfigValue??'null')}catch{value=null}cell.textContent=JSON.stringify(value)??'null'}
  }
  private async update(kind:'set-discord'|'toggle-admin'|'refresh',body:AdminUsersSetDiscordData['body']|Record<string,never>){
    if(this.operation)return;const token={};this.operation=token;this.interrupted=false
    const error=kind==='set-discord'?'[data-admin-discord-error]':'[data-admin-user-error]'
    const controls=kind==='set-discord'?[...this.q<HTMLFormElement>('[data-admin-discord]').querySelectorAll<HTMLInputElement|HTMLButtonElement>('input,button')]:kind==='refresh'?[...this.all<HTMLInputElement|HTMLButtonElement>('[data-admin-discord] input,[data-admin-discord] button,[data-admin-user-toggle]')]:[this.q<HTMLButtonElement>('[data-admin-user-toggle]')]
    const spinner=this.q<HTMLElement>(kind==='set-discord'?'[data-admin-discord-spinner]':'[data-admin-user-toggle-spinner]')
    const draft=this.q<HTMLInputElement>('[data-admin-discord-id]').value
    this.error(error);for(const control of controls)control.disabled=true;spinner.hidden=false
    try{
      const id=this.element.dataset.characterId;if(!id||!/^\d+$/.test(id))throw new Error('Could not identify this user.')
      if(kind!=='refresh')await this.request(`/api/admin/users/${id}/${kind}`,{method:'POST',body})
      const text=await this.html(`/_fragments/admin/users/${id}`,kind==='refresh'?'Could not refresh this user. Reload this page.':'The change was saved, but the user could not be refreshed. Reload this page.')
      const next=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>('[data-admin-user]')
      if(!next||next.dataset.characterId!==id)throw new Error('Could not refresh this user.')
      const local=this.q<HTMLInputElement>('[data-admin-discord-id]').value;if(local!==draft)requiredElement<HTMLInputElement>(next,'[data-admin-discord-id]').value=local
      this.element.replaceWith(document.importNode(next,true))
    }catch(failure){this.error(error,failure)}finally{if(this.operation===token){this.operation=undefined;spinner.hidden=true;for(const control of controls)control.disabled=false}}
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target)
    if(target?.closest('[data-admin-user-toggle]')){await this.update('toggle-admin',{});return}
    if(target?.closest('[data-admin-discord-clear]')){this.q<HTMLInputElement>('[data-admin-discord-id]').value='';await this.update('set-discord',{discord_user_id:null})}
  }
  async submit(event:SubmitEvent){if(eventElement(event.target)?.matches('[data-admin-discord]')){event.preventDefault();await this.update('set-discord',{discord_user_id:this.q<HTMLInputElement>('[data-admin-discord-id]').value.trim()||null})}}
  hide(){this.interrupted||=Boolean(this.operation);this.operation=undefined;this.release();super.hide()}
  show(event:PageTransitionEvent){if(event.persisted&&this.interrupted)void this.update('refresh',{})}
}
registerController('admin-user-detail',AdminUserDetailController)
