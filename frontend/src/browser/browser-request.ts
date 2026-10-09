import type {AuthMeLegacyResponses} from '../../shared/api/types.gen'
import {isRecord,readCookieJSON} from './shared/json'
export class BrowserRequestError extends Error {
  constructor(message:string,readonly status:number){super(message);this.name='BrowserRequestError'}
}

export function requestErrorMessage(body:unknown,fallback:string):string {
 const message=isRecord(body)?[body.message,body.detail,body.error].find(value=>typeof value==='string'&&value):undefined
 return typeof message==='string'?message:fallback
}

// Share transport and API error semantics across page controllers.
export async function requestJSON<T=unknown>(path:string,options:RequestInit={},fallback?:string):Promise<T>{
  const response=await fetch(path,{credentials:'same-origin',...options})
  const body:unknown=await response.json().catch(()=>null)
  if(!response.ok){
    throw new BrowserRequestError(requestErrorMessage(body,fallback??`Request failed (${response.status})`),response.status)
  }
  return body as T
}
export type JSONRequestOptions=Omit<RequestInit,'body'>&{body?:unknown}
export async function requestBodyJSON<T=unknown>(path:string,options:JSONRequestOptions={},fallback='Request failed'):Promise<T>{
  const {body,...init}=options,headers=new Headers(init.headers)
  if(body!==undefined&&!headers.has('Content-Type'))headers.set('Content-Type','application/json')
  return requestJSON<T>(path,{...init,headers,body:body===undefined?undefined:JSON.stringify(body)},fallback)
}
export type BrowserUser=NonNullable<AuthMeLegacyResponses[200]['user']>
let sessionRequest:Promise<BrowserUser|null>|undefined
// One authoritative session lookup per document; failed requests remain retryable.
export function readBrowserSession():Promise<BrowserUser|null>{
  sessionRequest??=requestJSON<{user?:BrowserUser|null}>('/auth/me').then(data=>{
    const user=data?.user??null
    if(user&&typeof document!=='undefined'){
      const raw=readCookieJSON('ek_prefs'),prefs=isRecord(raw)?raw:{}
      const defaultTabs=user.settings?.defaultTabs
      document.cookie=`ek_prefs=${encodeURIComponent(JSON.stringify({...prefs,defaultTabs:defaultTabs&&typeof defaultTabs==='object'?defaultTabs:{}}))}; Max-Age=31536000; Path=/; SameSite=Lax${location.protocol==='https:'?'; Secure':''}`
    }
    return user
  }).catch(error=>{sessionRequest=undefined;throw error})
  return sessionRequest
}
