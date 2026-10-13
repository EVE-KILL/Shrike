import {CSS_VAR_MAP,type ThemeConfig} from '../utils/themeData'

// Compiled separately as a tiny synchronous script to apply appearance before paint.
try {
  const cookies=Object.fromEntries(document.cookie.split(';').map(part=>{
    const index=part.indexOf('=')
    return [part.slice(0,index).trim(),decodeURIComponent(part.slice(index+1))]
  }))
  const match=/^\/backgrounds\/(bg[1-8])\.webp$/.exec(cookies.siteBackground||'')
  if(match)document.documentElement.style.setProperty('--site-bg',`image-set(url("/assets/backgrounds/${match[1]}.avif") type("image/avif"), url("/assets/backgrounds/${match[1]}.webp") type("image/webp"))`)
  const theme:Partial<ThemeConfig>=JSON.parse(cookies.ek_theme||'{}')
  for(const [key,variable] of Object.entries(CSS_VAR_MAP)){
    const value=theme[key as keyof ThemeConfig]
    if(typeof value==='string'&&CSS.supports('color',value))document.documentElement.style.setProperty(variable,value)
  }
}catch{/* Malformed preferences must not prevent rendering. */}
