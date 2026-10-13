import {SettingsComponent} from './base'
import {registerController} from '../application'
import {eventElement} from '../../shared/dom'
import type {UserDescriptionUpdateCompatResponses,UserDescriptionUpdateCompatData} from '../../../../shared/api/types.gen'

export class SettingsBioController extends SettingsComponent{
  connect(){this.render()}
  input(){this.render()}
  private render(){
    this.q<HTMLElement>('[data-settings-bio-count]').textContent=`${this.q<HTMLTextAreaElement>('textarea').value.length} / 4000`
    for(const format of this.all<HTMLElement>('[data-settings-bio-format]')){
      const active=format.dataset.settingsBioFormat===this.element.dataset.format
      format.className='px-3 py-1 rounded-md text-xs font-medium transition-colors cursor-pointer '+(active?'bg-blue-500/15 text-blue-400':'bg-white/[0.04] text-gray-500 hover:bg-white/[0.08]');format.setAttribute('aria-pressed',String(active))
    }
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target),format=target?.closest<HTMLElement>('[data-settings-bio-format]')?.dataset.settingsBioFormat
    if(format==='markdown'||format==='eve_html'){this.element.dataset.format=format;this.render();return}
    const button=target?.closest<HTMLButtonElement>('[data-settings-bio-save]');if(!button||button.disabled)return
    const entity=this.element.dataset.settingsBio,formatValue=this.element.dataset.format
    if((entity!=='character'&&entity!=='corporation'&&entity!=='alliance')||(formatValue!=='markdown'&&formatValue!=='eve_html'))return
    const body:UserDescriptionUpdateCompatData['body']={entity,format:formatValue,description:this.q<HTMLTextAreaElement>('textarea').value}
    button.disabled=true;button.textContent='Saving...';button.className='px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer bg-white/[0.04] text-gray-500'
    this.error('[data-settings-bio-error]');this.q<HTMLElement>('[data-settings-bio-submitted]').hidden=true
    try{
      const result=await this.request<UserDescriptionUpdateCompatResponses[200]>('/api/user/descriptions',{method:'PUT',body},'Failed to save')
      this.q<HTMLElement>('[data-settings-bio-pending]').hidden=result.status==='cleared';this.saved('[data-settings-bio-submitted]',3000)
    }catch(error){this.error('[data-settings-bio-error]',error)}
    finally{button.disabled=false;button.textContent='Save';button.className='px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer bg-blue-500/15 text-blue-400 hover:bg-blue-500/25'}
  }
}
registerController('settings-bio',SettingsBioController)
