import {CSS_VAR_MAP,DEFAULT_THEME} from './themeData'

// Used before first paint and by the live appearance editor. Derived surface
// values preserve the original dark defaults until a background is customized.
export function applyThemeColors(theme:Record<string,unknown>,root:HTMLElement):void {
 for(const [key,variable] of Object.entries(CSS_VAR_MAP)) {
  const value=theme[key]
  if(typeof value==='string'&&CSS.supports('color',value))root.style.setProperty(variable,value)
  else root.style.removeProperty(variable)
 }
 const background=theme.bgPrimary
 if(typeof background==='string'&&background!==DEFAULT_THEME.bgPrimary&&CSS.supports('color',background))root.style.setProperty('--theme-surface-primary',background)
 else root.style.removeProperty('--theme-surface-primary')
 // Native selects, date controls and scrollbars should follow the palette too.
 // The appearance editor and presets provide hex colors; other CSS colors
 // retain the site's native dark fallback.
 const hex=typeof background==='string'?/^#([\da-f]{6}|[\da-f]{3})$/i.exec(background)?.[1]:undefined
 if(hex) {
  const full=hex.length===3?[...hex].map(char=>char+char).join(''):hex
  const [r,g,b]=[0,2,4].map(index=>parseInt(full.slice(index,index+2),16))
  root.style.colorScheme=(r!*.299+g!*.587+b!*.114)>150?'light':'dark'
 }else root.style.removeProperty('color-scheme')
}
