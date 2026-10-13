import {CSS_VAR_MAP,THEME_PRESETS,type ThemeConfig} from '../utils/themeData'
import {isRecord} from './shared/json'

// Compiled separately as a tiny synchronous script to apply appearance before paint.
try {
 const root=document.documentElement
 const cookies:Record<string,string>={}
 for(const part of document.cookie.split(';')){
  const index=part.indexOf('=');if(index<0)continue
  try{cookies[part.slice(0,index).trim()]=decodeURIComponent(part.slice(index+1))}catch{/* Ignore this malformed cookie. */}
 }
 const background=cookies.siteBackground||root.dataset.domainBackground||''
 const match=/^\/backgrounds\/(bg[1-8])\.webp$/.exec(background)
 if(match)root.style.setProperty('--site-bg',`image-set(url("/assets/backgrounds/${match[1]}.avif") type("image/avif"), url("/assets/backgrounds/${match[1]}.webp") type("image/webp"))`)
 else if(/^\/images\/domains\/background\/\d+$/.test(background))root.style.setProperty('--site-bg',`url(${JSON.stringify(background)})`)
 let theme:unknown={}
 if(cookies.ek_theme){try{theme=JSON.parse(cookies.ek_theme)}catch{/* Keep the default if a saved preference is malformed. */}}
 else if(root.dataset.domainTheme){
  const domain:unknown=JSON.parse(root.dataset.domainTheme)
  if(isRecord(domain))theme={...THEME_PRESETS.find(preset=>preset.id===domain.defaultThemePreset)?.overrides,...(isRecord(domain.defaultThemeOverrides)?domain.defaultThemeOverrides:{})}
 }
 if(isRecord(theme))for(const [key,variable] of Object.entries(CSS_VAR_MAP)){
  const value=theme[key as keyof ThemeConfig]
  if(typeof value==='string'&&CSS.supports('color',value))root.style.setProperty(variable,value)
 }
}catch{/* Malformed preferences must not prevent rendering. */}
