import {DEFAULT_THEME,CSS_VAR_MAP,THEME_PRESETS,type ThemeConfig} from '../utils/themeData'
import {applyThemeColors} from '../utils/applyTheme'
export type ThemeOverrides=Partial<ThemeConfig>
export function minimalTheme(overrides:ThemeOverrides|Record<string,unknown>={}):ThemeOverrides {
  return Object.fromEntries(Object.entries(overrides).filter(([key,value])=>Object.hasOwn(CSS_VAR_MAP,key)&&typeof value==='string'&&value&&value!==DEFAULT_THEME[key as keyof ThemeConfig]))
}
export function normaliseTheme(overrides:ThemeOverrides):string {
  return JSON.stringify(Object.entries(minimalTheme(overrides)).sort(([a],[b])=>a.localeCompare(b)))
}
export function detectThemePreset(overrides:ThemeOverrides):string {
  for(const preset of THEME_PRESETS){
    if(preset.id==='default'&&Object.keys(overrides).length===0)return 'default'
    if(preset.id!=='default'){
      const keys=Object.keys(preset.overrides) as (keyof ThemeConfig)[]
      if(keys.length&&keys.every(key=>overrides[key]===preset.overrides[key]))return preset.id
    }
  }
  return Object.keys(overrides).length===0?'default':'custom'
}
export function applySettingsTheme(overrides:ThemeOverrides,root:HTMLElement=document.documentElement):void {
  applyThemeColors(overrides,root)
}
