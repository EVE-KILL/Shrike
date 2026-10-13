// Explicit routes take precedence over browser preferences; SSR stays shared.
export function preferredDefaultTab(kind:string,basePath:string,currentURL:string,
  availablePaths:readonly string[],cookies:string):URL|null{
  const current=new URL(currentURL)
  if(current.pathname!==basePath)return null
  let prefs:{defaultTabs?:Record<string,unknown>}|null
  try{
    const value=cookies.split('; ').find(cookie=>cookie.startsWith('ek_prefs='))?.slice(9)
    prefs=value?JSON.parse(decodeURIComponent(value)):null
  }catch{return null}
  const preferred=prefs?.defaultTabs?.[kind==='group'?'item':kind]
  if(typeof preferred!=='string'||preferred==='dashboard')return null
  const path=basePath+'/'+preferred
  if(!availablePaths.includes(path))return null
  return new URL(path,current.origin)
}
