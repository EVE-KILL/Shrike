export function isRecord(value:unknown):value is Record<string,unknown> {
  return typeof value==='object'&&value!==null&&!Array.isArray(value)
}
export function readCookieJSON(name:string):unknown {
  try{
    const part=document.cookie.split('; ').find(cookie=>cookie.startsWith(name+'='))
    return JSON.parse(decodeURIComponent(part?.slice(name.length+1)||'null')) as unknown
  }catch{return null}
}
export function errorMessage(error:unknown,fallback='Request failed'):string {
  return error instanceof Error?error.message:fallback
}
export function isAbortError(error:unknown):boolean {
  return typeof error==='object'&&error!==null&&'name'in error&&error.name==='AbortError'
}
