import {RequestComponent} from './request-component'
import {isAbortError} from '../shared/json'
import {replaceFragment,replaceFragmentChildren} from '../fragment'
type SectionRead={container:HTMLElement;path:string;selector:string;message:string;loading?:string;errorClass?:string;replaceRoot?:boolean;beforeCommit?:(content:HTMLElement)=>void;loaded?:(content:HTMLElement)=>void}

// Named reads are independent, but each section accepts only its latest body.
export abstract class FragmentSections<K extends string> extends RequestComponent{
 protected abstract retryAttribute:string
 private sections=new Map<K,{request:AbortController;container:HTMLElement}>()
 protected cancelSection(part:K){const current=this.sections.get(part);if(!current)return;current.request.abort();current.container.removeAttribute('aria-busy');this.sections.delete(part)}
 protected sectionPending(part:K){return this.sections.has(part)}
 protected async section(part:K,options:SectionRead):Promise<boolean>{
  this.cancelSection(part);const request=new AbortController(),{container,selector,message}=options;this.sections.set(part,{request,container});container.setAttribute('aria-busy','true');container.querySelector('[data-section-error]')?.remove()
  if(options.loading&&!container.querySelector(selector)&&!container.querySelector('[role="status"]')){const status=document.createElement('div');status.className='war-loading';status.setAttribute('role','status');status.textContent=options.loading;container.replaceChildren(status)}
  const current=()=>this.sections.get(part)?.request===request&&container.isConnected
  try{const text=await this.text(options.path,message,{signal:request.signal,headers:{Accept:'text/html'}}),content=new DOMParser().parseFromString(text,'text/html').querySelector<HTMLElement>(selector);if(!content)throw new Error(message);if(!current())return false;options.beforeCommit?.(content);const next=document.importNode(content,true);if(options.replaceRoot)replaceFragment(container,next);else replaceFragmentChildren(container,next);options.loaded?.(next);return true}
  catch(error){if(!isAbortError(error)&&current()){if(!container.matches(selector)&&!container.querySelector(selector))container.replaceChildren();const panel=document.createElement('div'),retry=document.createElement('button');panel.className=options.errorClass??'war-loading';panel.dataset.sectionError=part;panel.setAttribute('role','alert');panel.textContent=message+' ';retry.type='button';retry.setAttribute(this.retryAttribute,part);retry.textContent='Retry';panel.append(retry);container.append(panel)}return false}
  finally{if(this.sections.get(part)?.request===request){this.sections.delete(part);container.removeAttribute('aria-busy')}}
 }
 hide(){for(const part of [...this.sections.keys()])this.cancelSection(part);super.hide()}
}
