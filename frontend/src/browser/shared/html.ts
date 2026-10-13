const entities:Record<string,string>={'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}
export function escapeHTML(value:unknown):string {
  return String(value??'').replace(/[&<>"']/g,char=>entities[char]!)
}
