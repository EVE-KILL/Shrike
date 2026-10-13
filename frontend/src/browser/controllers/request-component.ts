import {Controller} from '@hotwired/stimulus'
import {BrowserRequestError,requestBodyJSON,requestJSON,requestErrorMessage,type JSONRequestOptions} from '../browser-request'
import {errorMessage,isAbortError} from '../shared/json'
import {requiredElement} from '../shared/dom'

// Several scoped controllers can own reads on the same component root.
const busyRequests=new WeakMap<HTMLElement,number>()
function changeBusy(root:HTMLElement,delta:number){const count=Math.max(0,(busyRequests.get(root)??0)+delta);if(count){busyRequests.set(root,count);root.setAttribute('aria-busy','true')}else{busyRequests.delete(root);root.removeAttribute('aria-busy')}}

export abstract class RequestComponent extends Controller<HTMLElement>{
  private ownedRequests=new Set<AbortController>()
  private timers=new Set<ReturnType<typeof setTimeout>>()
  protected q<T extends Element=HTMLElement>(selector:string){return requiredElement<T>(this.element,selector)}
  protected all<T extends Element=HTMLElement>(selector:string){return this.element.querySelectorAll<T>(selector)}
  protected error(selector:string,error:unknown=''){
    if(!this.element.isConnected||isAbortError(error))return
    const box=this.element.querySelector<HTMLElement>(selector);if(!box)return
    const message=typeof error==='string'?error:errorMessage(error);box.hidden=!message;box.textContent=message
  }
  protected after(delay:number,run:()=>void){
    const timer=setTimeout(()=>{this.timers.delete(timer);if(this.element.isConnected)run()},delay);this.timers.add(timer);return timer
  }
  protected stopTimer(timer:ReturnType<typeof setTimeout>|undefined){if(timer!==undefined){clearTimeout(timer);this.timers.delete(timer)}}
  protected stopTimers(){for(const timer of this.timers)clearTimeout(timer);this.timers.clear()}
  protected saved(selector:string,delay=2000){const node=this.q<HTMLElement>(selector);node.hidden=false;this.after(delay,()=>node.hidden=true)}
  private async owned<T>(signal:AbortSignal|undefined,run:(signal:AbortSignal)=>Promise<T>):Promise<T>{
    const controller=new AbortController(),abort=()=>controller.abort();this.ownedRequests.add(controller);changeBusy(this.element,1)
    if(signal?.aborted)controller.abort();else signal?.addEventListener('abort',abort,{once:true})
    try{if(controller.signal.aborted||!this.element.isConnected)throw new DOMException('Aborted','AbortError');const result=await run(controller.signal);if(controller.signal.aborted||!this.element.isConnected)throw new DOMException('Aborted','AbortError');return result}
    catch(error){if(controller.signal.aborted||!this.element.isConnected)throw new DOMException('Aborted','AbortError');throw error}
    finally{signal?.removeEventListener('abort',abort);if(this.ownedRequests.delete(controller))changeBusy(this.element,-1)}
  }
  protected request<T>(path:string,options:JSONRequestOptions={},fallback='Request failed'){
    return this.owned(options.signal??undefined,signal=>requestBodyJSON<T>(path,{...options,signal},fallback))
  }
  protected requestForm<T>(path:string,body:FormData,fallback:string){
    return this.owned(undefined,signal=>requestJSON<T>(path,{method:'POST',body,signal},fallback))
  }
  // Explorers inspect non-success HTTP responses too; ownership still covers
  // headers, body reads and stream consumption before committing their result.
  protected response<T>(path:string,options:RequestInit,consume:(response:Response,signal:AbortSignal)=>Promise<T>){
    return this.owned(options.signal??undefined,async signal=>consume(await fetch(path,{credentials:'same-origin',...options,signal}),signal))
  }
  protected text(path:string,fallback:string,options:RequestInit={}){
    return this.readTextWithHeaders(path,fallback,options).then(result=>result.text)
  }
  protected readTextWithHeaders(path:string,fallback:string,options:RequestInit={}){
    return this.owned(options.signal??undefined,async signal=>{
      const response=await fetch(path,{credentials:'same-origin',...options,signal})
      if(!response.ok){const body:unknown=response.headers.get('Content-Type')?.includes('application/json')?await response.json().catch(()=>null):null;throw new BrowserRequestError(response.status===401?'Your session expired. Log in again to continue.':requestErrorMessage(body,fallback),response.status)}
      return {text:await response.text(),headers:response.headers}
    })
  }
  hide(){for(const request of this.ownedRequests)request.abort();const count=this.ownedRequests.size;this.ownedRequests.clear();changeBusy(this.element,-count)}
  disconnect(){this.hide();this.stopTimers()}
}
