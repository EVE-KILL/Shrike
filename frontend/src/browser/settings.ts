import {renderEsiVolume,initializeEsiLog} from './esi'
import './settings-domains'
import {registerController} from './controllers/application'
import {settingsState} from './controllers/settings/base'
import {FragmentTabsController} from './controllers/fragment-tabs'
import {eventElement} from './shared/dom'
import './controllers/settings/preferences'
import './controllers/settings/appearance'
import './controllers/settings/sessions'
import './controllers/settings/bio'
import './controllers/settings/wallet'
import './controllers/settings/comments'

export class SettingsController extends FragmentTabsController {
  protected readonly kind='settings'
  connect(){settingsState(this.element);super.connect()}
  protected preparePanel(panel:HTMLElement){
    for(const chart of panel.querySelectorAll<HTMLElement>('[data-esi-volume]'))renderEsiVolume(chart)
    for(const log of panel.querySelectorAll<HTMLElement>('[data-esi-log]'))initializeEsiLog(log)
  }
  async click(event:MouseEvent){
    await super.click(event)
    const logout=eventElement(event.target)?.closest<HTMLButtonElement>('[data-settings-logout]');if(!logout||logout.disabled)return
    logout.disabled=true
    try{await this.request('/auth/logout',{method:'POST'});location.assign('/')}catch(error){this.error('[data-settings-logout-error]',error);logout.disabled=false}
  }
}
registerController('settings',SettingsController)
