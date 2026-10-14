import {SettingsComponent,writeSettingsCookie} from './base'
import {registerController} from '../application'
import {eventElement,requiredElement} from '../../shared/dom'
import {isRecord,readCookieJSON} from '../../shared/json'
import type {UserPreferencesUpdateCompatResponses} from '../../../../shared/api/types.gen'

export class SettingsPreferencesController extends SettingsComponent{
  connect(){this.zone()}
  private get menus(){return [...this.all<HTMLElement>('[data-settings-select]')]}
  private setOpen(menu:HTMLElement,open:boolean){
    const trigger=requiredElement<HTMLButtonElement>(menu,'button');requiredElement<HTMLElement>(menu,'[data-settings-select-options]').hidden=!open
    trigger.setAttribute('aria-expanded',String(open));trigger.className='flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs transition-colors cursor-pointer border w-full justify-between '+(open?'bg-blue-500/10 text-blue-400 border-white/[0.12]':'bg-white/[0.06] text-gray-300 border-white/[0.08] hover:border-white/[0.12]')
    requiredElement<HTMLElement>(trigger,'span:last-child').classList.toggle('rotate-180',open)
  }
  private closeMenus(){for(const menu of this.menus)this.setOpen(menu,false)}
  async click(event:MouseEvent){
    const target=eventElement(event.target),option=target?.closest<HTMLButtonElement>('[data-settings-select-option]'),menu=target?.closest<HTMLElement>('[data-settings-select]')
    if(option&&menu){
      menu.dataset.value=option.dataset.settingsSelectOption;requiredElement<HTMLElement>(menu,'[data-settings-select-label]').textContent=option.textContent
      for(const item of requiredElement<HTMLElement>(menu,'[data-settings-select-options]').children)item.className='flex items-center w-full px-2.5 py-1.5 rounded-md text-xs transition-colors text-left cursor-pointer '+(item===option?'text-white bg-blue-500/[0.12]':'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]')
      this.closeMenus();requiredElement<HTMLButtonElement>(menu,'button').focus();return
    }
    if(menu){const open=Boolean(requiredElement<HTMLElement>(menu,'[data-settings-select-options]').hidden);this.closeMenus();this.setOpen(menu,open);return}
    const zone=target?.closest<HTMLElement>('[data-settings-zone]')?.dataset.settingsZone
    if(zone==='local'||zone==='eve'){
      document.cookie=`ek_tz=${zone}; Max-Age=31536000; Path=/; SameSite=Lax${location.protocol==='https:'?'; Secure':''}`
      document.dispatchEvent(new CustomEvent('eve-time-zone-change',{detail:zone}));this.zone();return
    }
    const button=target?.closest<HTMLButtonElement>('[data-settings-save-prefs]');if(!button||button.disabled)return
    button.disabled=true;button.textContent='Saving...';this.error('[data-settings-prefs-error]')
    const defaultTabs=Object.fromEntries(this.menus.map(menu=>[menu.dataset.settingsSelect??'',menu.dataset.value??'']))
    try{
      const data=await this.request<UserPreferencesUpdateCompatResponses[200]>('/api/user/preferences',{method:'PUT',body:{defaultTabs}},'Could not save preferences')
      const cookie=readCookieJSON('ek_prefs');writeSettingsCookie('ek_prefs',{...(isRecord(cookie)?cookie:{}),defaultTabs:data.defaultTabs});this.saved('[data-settings-prefs-saved]')
    }catch(error){this.error('[data-settings-prefs-error]',error)}finally{button.disabled=false;button.textContent='Save'}
  }
  keyboard(event:KeyboardEvent){
    const menu=eventElement(event.target)?.closest<HTMLElement>('[data-settings-select]');if(!menu)return
    const trigger=requiredElement<HTMLButtonElement>(menu,'button')
    if(event.key==='Escape'){this.closeMenus();trigger.focus()}
    if(event.key==='ArrowDown'&&event.target===trigger){event.preventDefault();this.setOpen(menu,true);requiredElement<HTMLElement>(menu,'[data-settings-select-options]').querySelector<HTMLButtonElement>('button')?.focus()}
  }
  outside(event:PointerEvent){if(!this.menus.some(menu=>menu.contains(eventElement(event.target))))this.closeMenus()}
  private zone(){
    const selected=document.cookie.split('; ').find(cookie=>cookie.startsWith('ek_tz='))?.slice(6)==='local'?'local':'eve'
    for(const button of this.all<HTMLElement>('[data-settings-zone]')){
      const on=button.dataset.settingsZone===selected
      button.className='flex-1 min-w-0 text-left px-4 py-3 rounded-lg border transition-colors cursor-pointer '+(on?'bg-blue-500/10 border-blue-500/40 text-blue-300':'bg-white/[0.02] border-white/[0.08] text-gray-400 hover:border-white/[0.15] hover:bg-blue-500/[0.04]')
      requiredElement<SVGUseElement>(button,'use').setAttribute('href','/assets/icons.svg#'+(on?'circle-check':'circle'));button.setAttribute('aria-pressed',String(on))
      if(button.dataset.settingsZone==='local')requiredElement<HTMLElement>(button,'[data-settings-zone-hint]').textContent=Intl.DateTimeFormat().resolvedOptions().timeZone||'Local'
    }
  }
}
registerController('settings-preferences',SettingsPreferencesController)
