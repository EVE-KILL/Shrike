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
export interface AccountFragment {
  user:BrowserUser|null
  desktop:DocumentFragment
  mobile:DocumentFragment
}
let accountRequest:Promise<AccountFragment>|undefined
let accountAbort:AbortController|undefined
let sessionRequest:Promise<BrowserUser|null>|undefined

// All session consumers share this private request on pages with the navbar.
// Template contents stay detached; controllers import fresh nodes when needed.
export function readAccountFragment():Promise<AccountFragment>{
  if(accountRequest)return accountRequest
  const abort=new AbortController();accountAbort=abort
  const request=fetch('/_fragments/account',{credentials:'same-origin',cache:'no-store',signal:AbortSignal.any([abort.signal,AbortSignal.timeout(10000)]),headers:{Accept:'text/html'}}).then(async response=>{
    if(!response.ok)throw new BrowserRequestError('Unable to load account. Please try again.',response.status)
    const body=await response.text();abort.signal.throwIfAborted()
    const page=new DOMParser().parseFromString(body,'text/html')
    const root=page.querySelector<HTMLElement>('[data-account-fragment]')
    const desktop=root?.querySelector<HTMLTemplateElement>('template[data-account-content="desktop"]')
    const mobile=root?.querySelector<HTMLTemplateElement>('template[data-account-content="mobile"]')
    if(!root||!desktop||!mobile||!root.dataset.accountSession)throw new Error('Invalid account response')
    const user:unknown=JSON.parse(root.dataset.accountSession)
    if(user!==null&&(!isRecord(user)||!Number.isSafeInteger(user.characterId)||Number(user.characterId)<=0||typeof user.characterName!=='string'||typeof user.isAdmin!=='boolean'))throw new Error('Invalid account session')
    return {user:user as BrowserUser|null,desktop:desktop.content,mobile:mobile.content}
  }).catch(error=>{if(accountRequest===request)accountRequest=undefined;throw error})
  accountRequest=request
  return request
}

// BFCache restores the document's memory, including old identity. Refresh on
// restore so logging out in another tab cannot leave a stale account menu.
export function resetBrowserSession():void {accountAbort?.abort();accountAbort=undefined;sessionRequest=undefined;accountRequest=undefined}

// One authoritative session lookup per document; failed requests remain retryable.
export function readBrowserSession():Promise<BrowserUser|null>{
  if(sessionRequest)return sessionRequest
  const request=(typeof document!=='undefined'&&document.querySelector('[data-account-slot]')
    ?readAccountFragment().then(fragment=>fragment.user)
    :requestJSON<{user?:BrowserUser|null}>('/auth/me').then(data=>data?.user??null)
  ).then(user=>{
    if(sessionRequest===request&&user&&typeof document!=='undefined'){
      const raw=readCookieJSON('ek_prefs'),prefs=isRecord(raw)?raw:{}
      const defaultTabs=user.settings?.defaultTabs
      document.cookie=`ek_prefs=${encodeURIComponent(JSON.stringify({...prefs,defaultTabs:defaultTabs&&typeof defaultTabs==='object'?defaultTabs:{}}))}; Max-Age=31536000; Path=/; SameSite=Lax${location.protocol==='https:'?'; Secure':''}`
    }
    return user
  }).catch(error=>{if(sessionRequest===request)sessionRequest=undefined;throw error})
  sessionRequest=request
  return request
}
