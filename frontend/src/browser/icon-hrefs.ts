import names from '../../data/icon-names.json'
const common=new Set<string>(names)
export function iconHref(value:unknown):string{
  const name=String(value||'').replace(/^lucide:/,'')
  if(!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(name))return ''
  return common.has(name)?'/assets/icons.svg#'+name:'/assets/lucide/'+name+'.svg#icon'
}
