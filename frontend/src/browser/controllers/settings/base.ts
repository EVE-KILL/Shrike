import {isRecord,readCookieJSON} from '../../shared/json'
import {minimalTheme,applySettingsTheme,detectThemePreset,type ThemeOverrides} from '../../settings-theme'

export function writeSettingsCookie(name:string,value:unknown){
  document.cookie=`${name}=${encodeURIComponent(JSON.stringify(value))}; Max-Age=${value===null?0:31536000}; Path=/; SameSite=Lax${location.protocol==='https:'?'; Secure':''}`
}
type SettingsState={theme:ThemeOverrides;preset:string}
const states=new WeakMap<HTMLElement,SettingsState>()
export function settingsState(element:HTMLElement):SettingsState{
  const root=element.closest<HTMLElement>('[data-settings-root]')
  if(!root)throw new Error('Settings component needs its page scope')
  let state=states.get(root)
  if(!state){
    let raw:unknown;try{raw=JSON.parse(root.dataset.userSettings??'{}')}catch{raw={}}
    const cookie=readCookieJSON('ek_theme');let theme=minimalTheme(isRecord(cookie)?cookie:{})
    if(!Object.keys(theme).length&&isRecord(raw)&&isRecord(raw.theme)){
      theme=minimalTheme(raw.theme);applySettingsTheme(theme);if(Object.keys(theme).length)writeSettingsCookie('ek_theme',theme)
    }
    state={theme,preset:detectThemePreset(theme)};states.set(root,state)
  }
  return state
}

export {RequestComponent as SettingsComponent} from '../request-component'
