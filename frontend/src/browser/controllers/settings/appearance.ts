import {DEFAULT_THEME,THEME_PRESETS,CSS_VAR_MAP,type ThemeConfig} from '../../../utils/themeData'
import {minimalTheme,normaliseTheme,applySettingsTheme} from '../../settings-theme'
import {SettingsComponent,settingsState,writeSettingsCookie} from './base'
import {registerController} from '../application'
import {eventElement,requiredElement} from '../../shared/dom'

const themeKey=(key:unknown):key is keyof ThemeConfig=>typeof key==='string'&&Object.hasOwn(CSS_VAR_MAP,key)
export class SettingsAppearanceController extends SettingsComponent{
  private snapshot:string|undefined
  private saving=false
  private get state(){return settingsState(this.element)}
  connect(){this.snapshot??=normaliseTheme(this.state.theme);this.render()}
  private preview(){applySettingsTheme(this.state.theme);const theme=minimalTheme(this.state.theme);writeSettingsCookie('ek_theme',Object.keys(theme).length?theme:null);this.render()}
  private render(){
    const state=this.state,resolved={...DEFAULT_THEME,...state.theme},dirty=normaliseTheme(state.theme)!==this.snapshot
    const status=this.q<HTMLElement>('[data-settings-theme-status]');status.textContent=dirty?'Unsaved changes — previewing locally':'Previewing your saved theme';status.className='text-fine truncate '+(dirty?'text-yellow-400':'text-gray-600')
    const button=this.q<HTMLButtonElement>('[data-settings-save-theme]');button.disabled=this.saving||!dirty;button.textContent=this.saving?'Saving...':'Save Theme';button.className='px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer flex-shrink-0 disabled:cursor-not-allowed '+(this.saving||!dirty?'bg-white/[0.04] text-gray-500':'bg-blue-500/15 text-blue-400 hover:bg-blue-500/25')
    for(const preset of this.all<HTMLElement>('[data-settings-preset]')){const active=preset.dataset.settingsPreset===state.preset;preset.className='relative flex flex-col items-center gap-2 p-3 rounded-lg border transition-all cursor-pointer '+(active?'border-blue-500/50 bg-blue-500/10':'border-white/[0.08] bg-white/[0.02] hover:bg-blue-500/[0.04] hover:border-white/[0.15]');requiredElement<HTMLElement>(preset,'[data-settings-preset-selected]').hidden=!active}
    this.q<HTMLElement>('[data-settings-theme-reset]').hidden=!Object.keys(state.theme).length
    for(const row of this.all<HTMLElement>('[data-settings-color-row]')){
      const key=row.dataset.settingsColorRow;if(!themeKey(key))continue
      const value=resolved[key];requiredElement<HTMLInputElement>(row,'[data-settings-color]').value=value.startsWith('#')?value.slice(0,7):value
      requiredElement<HTMLElement>(row,'[data-settings-color-value]').textContent=value;requiredElement<HTMLElement>(row,'[data-settings-color-reset]').hidden=!state.theme[key]||state.theme[key]===DEFAULT_THEME[key]
    }
    for(const item of this.all<HTMLElement>('[data-settings-preview-bg]')){const key=item.dataset.settingsPreviewBg;if(themeKey(key))item.style.backgroundColor=resolved[key]}
    for(const item of this.all<HTMLElement>('[data-settings-preview-color]')){const key=item.dataset.settingsPreviewColor;if(themeKey(key))item.style.color=resolved[key]}
    for(const item of this.all<HTMLElement>('[data-settings-preview-status]')){const key=item.dataset.settingsPreviewStatus;if(themeKey(key)){item.style.color=resolved[key];item.style.backgroundColor=resolved[key]+'20'}}
    const loss=this.q<HTMLElement>('[data-settings-preview-loss]');loss.style.backgroundColor=resolved.lossBg;loss.style.borderColor=resolved.lossBorder
  }
  async click(event:MouseEvent){
    const target=eventElement(event.target),preset=target?.closest<HTMLElement>('[data-settings-preset]')?.dataset.settingsPreset
    if(preset){const definition=THEME_PRESETS.find(item=>item.id===preset);if(definition){this.state.preset=preset;this.state.theme={...definition.overrides};this.preview()}return}
    if(target?.closest('[data-settings-theme-reset]')){this.state.preset='default';this.state.theme={};this.preview();return}
    const reset=target?.closest<HTMLElement>('[data-settings-color-reset]')?.dataset.settingsColorReset
    if(themeKey(reset)){this.state.preset='custom';this.state.theme={...this.state.theme,[reset]:DEFAULT_THEME[reset]};this.preview();return}
    if(!target?.closest('[data-settings-save-theme]')||this.saving)return
    this.saving=true;this.error('[data-settings-theme-error]');const theme=minimalTheme(this.state.theme);this.render()
    try{await this.request('/api/user/theme',{method:'PUT',body:{theme}},'Could not save your theme');this.snapshot=normaliseTheme(theme);this.saved('[data-settings-theme-saved]')}
    catch(error){this.error('[data-settings-theme-error]',error)}finally{this.saving=false;if(this.element.isConnected)this.render()}
  }
  input(event:Event){const input=event.target;if(!(input instanceof HTMLInputElement)||!themeKey(input.dataset.settingsColor))return;this.state.preset='custom';this.state.theme={...this.state.theme,[input.dataset.settingsColor]:input.value};this.preview()}
}
registerController('settings-appearance',SettingsAppearanceController)
